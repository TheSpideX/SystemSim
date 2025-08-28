package testfactory

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// UserBuilder helps build test users with fluent interface
type UserBuilder struct {
	user *models.User
}

// NewUserBuilder creates a new user builder with defaults
func NewUserBuilder() *UserBuilder {
	return &UserBuilder{
		user: &models.User{
			ID:                    uuid.New(),
			Email:                 "test@example.com",
			PasswordHash:          "$2a$12$dummy.hash.for.testing",
			FirstName:             "Test",
			LastName:              "User",
			Company:               "Test Company",
			EmailVerified:         true,
			IsActive:              true,
			FailedLoginAttempts:   0,
			SimulationPreferences: make(models.JSONMap),
			UIPreferences:         make(models.JSONMap),
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		},
	}
}

// WithEmail sets the email
func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.user.Email = email
	return b
}

// WithPassword sets the password (will be hashed)
func (b *UserBuilder) WithPassword(password string) *UserBuilder {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	b.user.PasswordHash = string(hashedPassword)
	return b
}

// WithName sets first and last name
func (b *UserBuilder) WithName(firstName, lastName string) *UserBuilder {
	b.user.FirstName = firstName
	b.user.LastName = lastName
	return b
}

// WithCompany sets the company
func (b *UserBuilder) WithCompany(company string) *UserBuilder {
	b.user.Company = company
	return b
}

// WithEmailVerified sets email verification status
func (b *UserBuilder) WithEmailVerified(verified bool) *UserBuilder {
	b.user.EmailVerified = verified
	return b
}

// WithEmailVerificationToken sets verification token and expiry
func (b *UserBuilder) WithEmailVerificationToken(token string, expiresAt time.Time) *UserBuilder {
	b.user.EmailVerificationToken = &token
	b.user.EmailVerificationExpiresAt = &expiresAt
	return b
}

// WithPasswordResetToken sets password reset token and expiry
func (b *UserBuilder) WithPasswordResetToken(token string, expiresAt time.Time) *UserBuilder {
	b.user.PasswordResetToken = &token
	b.user.PasswordResetExpiresAt = &expiresAt
	return b
}

// WithFailedLoginAttempts sets failed login attempts
func (b *UserBuilder) WithFailedLoginAttempts(attempts int) *UserBuilder {
	b.user.FailedLoginAttempts = attempts
	return b
}

// WithLockedUntil sets account lock expiry
func (b *UserBuilder) WithLockedUntil(lockedUntil time.Time) *UserBuilder {
	b.user.LockedUntil = &lockedUntil
	return b
}

// WithInactive sets user as inactive
func (b *UserBuilder) WithInactive() *UserBuilder {
	b.user.IsActive = false
	return b
}

// WithAdmin sets user as admin
func (b *UserBuilder) WithAdmin() *UserBuilder {
	b.user.IsAdmin = true
	return b
}

// Build returns the built user
func (b *UserBuilder) Build() *models.User {
	return b.user
}

// CreateInDB creates the user in the database
func (b *UserBuilder) CreateInDB(t *testing.T, userRepo *repository.UserRepository) *models.User {
	err := userRepo.Create(b.user)
	require.NoError(t, err)
	return b.user
}

// SessionBuilder helps build test sessions
type SessionBuilder struct {
	session *models.Session
}

// NewSessionBuilder creates a new session builder with defaults
func NewSessionBuilder(userID uuid.UUID) *SessionBuilder {
	return &SessionBuilder{
		session: &models.Session{
			ID:               uuid.New(),
			UserID:           userID,
			TokenHash:        "dummy_token_hash",
			DeviceInfo:       make(models.JSONMap),
			ExpiresAt:        time.Now().Add(15 * time.Minute),
			LastUsedAt:       time.Now(),
			IsActive:         true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}
}

// WithTokenHash sets the token hash
func (b *SessionBuilder) WithTokenHash(tokenHash string) *SessionBuilder {
	b.session.TokenHash = tokenHash
	return b
}

// WithRefreshToken sets the refresh token hash and expiry
func (b *SessionBuilder) WithRefreshToken(refreshTokenHash string, expiresAt time.Time) *SessionBuilder {
	b.session.RefreshTokenHash = &refreshTokenHash
	b.session.RefreshExpiresAt = &expiresAt
	return b
}

// WithUserAgent sets the user agent
func (b *SessionBuilder) WithUserAgent(userAgent string) *SessionBuilder {
	b.session.UserAgent = &userAgent
	return b
}

// WithIPAddress sets the IP address
func (b *SessionBuilder) WithIPAddress(ipAddress string) *SessionBuilder {
	b.session.IPAddress = &ipAddress
	return b
}

// WithExpiry sets the session expiry
func (b *SessionBuilder) WithExpiry(expiresAt time.Time) *SessionBuilder {
	b.session.ExpiresAt = expiresAt
	return b
}

// WithRevoked sets the session as revoked
func (b *SessionBuilder) WithRevoked(reason string) *SessionBuilder {
	now := time.Now()
	b.session.RevokedAt = &now
	b.session.RevokedReason = &reason
	b.session.IsActive = false
	return b
}

// Build returns the built session
func (b *SessionBuilder) Build() *models.Session {
	return b.session
}

// CreateInDB creates the session in the database
func (b *SessionBuilder) CreateInDB(t *testing.T, sessionRepo *repository.SessionRepository) *models.Session {
	err := sessionRepo.Create(b.session)
	require.NoError(t, err)
	return b.session
}

// RequestBuilder helps build test requests
type RequestBuilder struct{}

// NewRequestBuilder creates a new request builder
func NewRequestBuilder() *RequestBuilder {
	return &RequestBuilder{}
}

// RegisterRequest creates a register request
func (b *RequestBuilder) RegisterRequest(email, password, firstName, lastName, company string) *models.RegisterRequest {
	return &models.RegisterRequest{
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		Company:   company,
	}
}

// LoginRequest creates a login request
func (b *RequestBuilder) LoginRequest(email, password string, remember bool) *models.LoginRequest {
	return &models.LoginRequest{
		Email:    email,
		Password: password,
		Remember: remember,
	}
}

// Common test data patterns
func CreateTestUserWithPassword(t *testing.T, userRepo *repository.UserRepository, email, password string) *models.User {
	return NewUserBuilder().
		WithEmail(email).
		WithPassword(password).
		CreateInDB(t, userRepo)
}

func CreateLockedTestUser(t *testing.T, userRepo *repository.UserRepository, email string) *models.User {
	return NewUserBuilder().
		WithEmail(email).
		WithFailedLoginAttempts(5).
		WithLockedUntil(time.Now().Add(15 * time.Minute)).
		CreateInDB(t, userRepo)
}

func CreateUnverifiedTestUser(t *testing.T, userRepo *repository.UserRepository, email string) *models.User {
	token := "verification_token_123"
	expiresAt := time.Now().Add(24 * time.Hour)
	return NewUserBuilder().
		WithEmail(email).
		WithEmailVerified(false).
		WithEmailVerificationToken(token, expiresAt).
		CreateInDB(t, userRepo)
}
