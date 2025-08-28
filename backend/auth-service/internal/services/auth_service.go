package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/systemsim/auth-service/internal/config"
	"github.com/systemsim/auth-service/internal/events"
	"github.com/systemsim/auth-service/internal/metrics"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/repository"
	"github.com/systemsim/auth-service/internal/security"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo       *repository.UserRepository
	sessionRepo    *repository.SessionRepository
	rbacService    *RBACService
	jwtManager     *security.JWTManager
	eventPublisher *events.Publisher
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository, rbacService *RBACService, jwtConfig config.JWTConfig, eventPublisher *events.Publisher) *AuthService {
	jwtManager := security.NewJWTManager(
		jwtConfig.Secret,
		jwtConfig.AccessTokenDuration,
		jwtConfig.RefreshTokenDuration,
		jwtConfig.Issuer,
	)

	return &AuthService{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		rbacService:    rbacService,
		jwtManager:     jwtManager,
		eventPublisher: eventPublisher,
	}
}

// Register registers a new user
func (s *AuthService) Register(req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if user already exists
	if _, err := s.userRepo.GetByEmail(req.Email); err == nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Hash password
	passwordHash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate email verification token
	verificationToken, err := security.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Create user
	user := &models.User{
		ID:                         uuid.New(),
		Email:                      req.Email,
		PasswordHash:               passwordHash,
		FirstName:                  req.FirstName,
		LastName:                   req.LastName,
		Company:                    req.Company,
		EmailVerified:              false,
		EmailVerificationToken:     &verificationToken,
		EmailVerificationExpiresAt: timePtr(time.Now().Add(24 * time.Hour)),
		IsActive:                   true,
		SimulationPreferences:      make(models.JSONMap),
		UIPreferences:              make(models.JSONMap),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Track registration metric
	metrics.IncrementRegistrations()

	// Assign default role to new user
	if err := s.rbacService.EnsureUserHasDefaultRole(user.ID); err != nil {
		// Log error but don't fail registration
		fmt.Printf("Warning: Failed to assign default role to user %s: %v\n", user.ID, err)
	}

	// Publish registration event
	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishUserRegistration(
			user.ID.String(), user.Email, user.FirstName, user.LastName, user.Company, "", // TODO: Add IP address
		); err != nil {
			fmt.Printf("Warning: Failed to publish registration event: %v\n", err)
		}

		// Publish verification email task
		if err := s.eventPublisher.PublishVerificationEmail(
			user.Email, user.FirstName, verificationToken,
		); err != nil {
			fmt.Printf("Warning: Failed to publish verification email: %v\n", err)
		}
	}

	// Create session and generate tokens (no remember me for registration)
	return s.createSessionAndTokens(user, nil, nil, false)
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(req *models.LoginRequest, userAgent, ipAddress *string) (*models.AuthResponse, error) {
	// Track login attempt
	metrics.IncrementLoginAttempts()

	// Get user by email
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		metrics.IncrementLoginFailures()
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user can attempt login
	if !user.CanAttemptLogin() {
		if user.IsLocked() {
			return nil, fmt.Errorf("account is locked due to too many failed attempts")
		}
		return nil, fmt.Errorf("account is not active")
	}

	// Verify password
	if err := security.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		// Increment failed login attempts
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= 5 {
			lockUntil := time.Now().Add(15 * time.Minute)
			user.LockedUntil = &lockUntil
		}
		s.userRepo.Update(user)
		metrics.IncrementLoginFailures()

		// Publish failed login event
		if s.eventPublisher != nil {
			userAgentStr := ""
			ipAddressStr := ""
			if userAgent != nil {
				userAgentStr = *userAgent
			}
			if ipAddress != nil {
				ipAddressStr = *ipAddress
			}

			if err := s.eventPublisher.PublishLoginFailure(
				req.Email, ipAddressStr, userAgentStr, "invalid_credentials",
			); err != nil {
				fmt.Printf("Warning: Failed to publish login failure event: %v\n", err)
			}
		}

		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset failed login attempts on successful login
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = ipAddress

	if err := s.userRepo.Update(user); err != nil {
		metrics.IncrementLoginFailures()
		return nil, fmt.Errorf("failed to update user login info: %w", err)
	}

	// Track successful login
	metrics.IncrementLoginSuccess()

	// Create session and generate tokens with remember me support
	authResponse, err := s.createSessionAndTokens(user, userAgent, ipAddress, req.Remember)
	if err != nil {
		return nil, err
	}

	// Publish successful login event
	if s.eventPublisher != nil {
		userAgentStr := ""
		ipAddressStr := ""
		if userAgent != nil {
			userAgentStr = *userAgent
		}
		if ipAddress != nil {
			ipAddressStr = *ipAddress
		}

		if err := s.eventPublisher.PublishLoginSuccess(
			user.ID.String(), authResponse.SessionID, user.Email, ipAddressStr, userAgentStr,
		); err != nil {
			fmt.Printf("Warning: Failed to publish login event: %v\n", err)
		}
	}

	return authResponse, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *AuthService) RefreshToken(refreshToken string) (*models.TokenResponse, error) {
	// Track token refresh attempt
	metrics.IncrementTokenRefreshes()
	// Validate refresh token
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	// Get session
	tokenHash := security.HashToken(refreshToken)
	session, err := s.sessionRepo.GetByRefreshTokenHash(tokenHash)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	// Check if session can be refreshed
	if !session.CanRefresh() {
		return nil, fmt.Errorf("session cannot be refreshed")
	}

	// Get user
	user, err := s.userRepo.GetByID(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate new token pair
	accessToken, newRefreshToken, err := s.jwtManager.GenerateTokenPair(
		user.ID, user.Email, user.IsAdmin, session.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update session with new tokens
	session.TokenHash = security.HashToken(accessToken)
	newRefreshTokenHash := security.HashToken(newRefreshToken)
	session.RefreshTokenHash = &newRefreshTokenHash
	session.LastUsedAt = time.Now()

	if err := s.sessionRepo.Update(session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.jwtManager.GetAccessTokenDuration().Seconds()),
	}, nil
}

// Logout revokes a user session
func (s *AuthService) Logout(sessionID uuid.UUID) error {
	session, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	session.Revoke("user_logout")
	return s.sessionRepo.Update(session)
}

// ValidateAccessToken validates an access token and returns claims
func (s *AuthService) ValidateAccessToken(tokenString string) (*security.JWTClaims, error) {
	claims, err := s.jwtManager.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims.TokenType != "access" {
		return nil, fmt.Errorf("invalid token type: expected access token")
	}

	return claims, nil
}

// VerifyEmail verifies a user's email address
func (s *AuthService) VerifyEmail(token string) error {
	user, err := s.userRepo.GetByEmailVerificationToken(token)
	if err != nil {
		return fmt.Errorf("invalid verification token")
	}

	if user.EmailVerificationExpiresAt != nil && time.Now().After(*user.EmailVerificationExpiresAt) {
		return fmt.Errorf("verification token has expired")
	}

	user.EmailVerified = true
	user.EmailVerificationToken = nil
	user.EmailVerificationExpiresAt = nil

	return s.userRepo.Update(user)
}

// ResendVerificationEmail resends email verification token
func (s *AuthService) ResendVerificationEmail(email string) (*models.EmailVerificationResponse, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		// Don't reveal if email exists or not for security
		return &models.EmailVerificationResponse{
			Message: "If the email exists and is not verified, a verification token has been sent",
		}, nil
	}

	// Check if email is already verified
	if user.EmailVerified {
		return &models.EmailVerificationResponse{
			Message: "Email is already verified",
		}, nil
	}

	// Check rate limiting - max 3 attempts per hour
	if user.EmailVerificationAttempts >= 3 {
		if user.EmailVerificationExpiresAt != nil && time.Now().Before(*user.EmailVerificationExpiresAt) {
			return nil, fmt.Errorf("too many verification attempts, please try again later")
		}
		// Reset attempts if 24 hours have passed
		user.EmailVerificationAttempts = 0
	}

	// Generate new verification token
	verificationToken, err := security.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	// Set token and expiry (24 hours)
	expiresAt := time.Now().Add(24 * time.Hour)
	user.EmailVerificationToken = &verificationToken
	user.EmailVerificationExpiresAt = &expiresAt
	user.EmailVerificationAttempts++

	// Update user
	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to save verification token: %w", err)
	}

	return &models.EmailVerificationResponse{
		VerificationToken: verificationToken,
		ExpiresIn:         86400, // 24 hours in seconds
		Message:           "Email verification token sent successfully",
	}, nil
}

// ForgotPassword initiates password reset process
func (s *AuthService) ForgotPassword(email string) (*models.PasswordResetResponse, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		// Don't reveal if email exists or not for security
		return &models.PasswordResetResponse{
			Message: "If the email exists, a reset token has been generated",
		}, nil
	}

	// Check rate limiting - max 3 attempts per hour
	if user.PasswordResetAttempts >= 3 {
		if user.PasswordResetExpiresAt != nil && time.Now().Before(*user.PasswordResetExpiresAt) {
			return nil, fmt.Errorf("too many reset attempts, please try again later")
		}
		// Reset attempts if hour has passed
		user.PasswordResetAttempts = 0
	}

	// Generate reset token
	resetToken, err := security.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate reset token: %w", err)
	}

	user.PasswordResetToken = &resetToken
	expiresAt := time.Now().Add(1 * time.Hour)
	user.PasswordResetExpiresAt = &expiresAt
	user.PasswordResetAttempts++

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to save reset token: %w", err)
	}

	return &models.PasswordResetResponse{
		ResetToken: resetToken,
		ExpiresIn:  3600, // 1 hour in seconds
		Message:    "Password reset token generated successfully",
	}, nil
}

// ResetPassword resets a user's password using a reset token
func (s *AuthService) ResetPassword(token, newPassword string) error {
	user, err := s.userRepo.GetByPasswordResetToken(token)
	if err != nil {
		return fmt.Errorf("invalid reset token")
	}

	if user.PasswordResetExpiresAt != nil && time.Now().After(*user.PasswordResetExpiresAt) {
		return fmt.Errorf("reset token has expired")
	}

	// Hash new password
	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = passwordHash
	user.PasswordResetToken = nil
	user.PasswordResetExpiresAt = nil
	user.PasswordResetAttempts = 0

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke all existing sessions for security
	if err := s.sessionRepo.RevokeAllUserSessions(user.ID, "password_reset"); err != nil {
		// Log error but don't fail the password reset
		fmt.Printf("Warning: Failed to revoke sessions after password reset for user %s: %v\n", user.ID, err)
	}

	return nil
}

// createSessionAndTokens creates a session and generates tokens
func (s *AuthService) createSessionAndTokens(user *models.User, userAgent, ipAddress *string, remember bool) (*models.AuthResponse, error) {
	sessionID := uuid.New()

	// Determine session duration based on remember me
	var accessDuration, refreshDuration time.Duration
	if remember {
		// Extended duration for remember me: 30 days for access, 90 days for refresh
		accessDuration = 30 * 24 * time.Hour  // 30 days
		refreshDuration = 90 * 24 * time.Hour // 90 days
	} else {
		// Normal duration
		accessDuration = s.jwtManager.GetAccessTokenDuration()
		refreshDuration = s.jwtManager.GetRefreshTokenDuration()
	}

	// Generate token pair with custom duration
	accessToken, refreshToken, err := s.jwtManager.GenerateTokenPairWithDuration(
		user.ID, user.Email, user.IsAdmin, sessionID, accessDuration, refreshDuration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	session := &models.Session{
		ID:                 sessionID,
		UserID:             user.ID,
		TokenHash:          security.HashToken(accessToken),
		RefreshTokenHash:   stringPtr(security.HashToken(refreshToken)),
		DeviceInfo:         make(models.JSONMap),
		UserAgent:          userAgent,
		IPAddress:          ipAddress,
		ExpiresAt:          time.Now().Add(accessDuration),
		RefreshExpiresAt:   timePtr(time.Now().Add(refreshDuration)),
		LastUsedAt:         time.Now(),
		IsActive:           true,
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Track session creation
	metrics.IncrementSessionsCreated()

	return &models.AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessDuration.Seconds()),
		RememberMe:   remember,
		SessionID:    sessionID.String(),
	}, nil
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}
