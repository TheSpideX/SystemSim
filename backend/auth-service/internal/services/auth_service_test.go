package services

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/systemsim/auth-service/internal/config"
	"github.com/systemsim/auth-service/internal/events"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/repository"
	"github.com/systemsim/auth-service/internal/security"
	"github.com/systemsim/auth-service/internal/testutils"
	"golang.org/x/crypto/bcrypt"
)

// Test helper functions to avoid circular dependencies
func createAuthService(t *testing.T, db *sql.DB, redisClient *redis.Client) (*AuthService, *testutils.TestRepositories) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:               "test-secret-key-for-testing-only",
			AccessTokenDuration:  time.Hour,
			RefreshTokenDuration: 24 * time.Hour,
			Issuer:              "auth-service-test",
		},
	}
	repos := testutils.SetupTestRepositories(t, db, redisClient)

	eventPublisher := events.NewPublisher(redisClient)
	rbacService := NewRBACService(repos.RBACRepo, repos.UserRepo)
	authService := NewAuthService(repos.UserRepo, repos.SessionRepo, rbacService, cfg.JWT, eventPublisher)

	return authService, repos
}

func createTestUserWithPassword(t *testing.T, userRepo *repository.UserRepository, email, password string) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		ID:                    uuid.New(),
		Email:                 email,
		PasswordHash:          string(hashedPassword),
		FirstName:             "Test",
		LastName:              "User",
		Company:               "Test Company",
		EmailVerified:         true,
		IsActive:              true,
		SimulationPreferences: make(models.JSONMap),
		UIPreferences:         make(models.JSONMap),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	err = userRepo.Create(user)
	require.NoError(t, err)
	return user
}

func createLockedTestUser(t *testing.T, userRepo *repository.UserRepository, email string) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	lockUntil := time.Now().Add(15 * time.Minute)
	user := &models.User{
		ID:                    uuid.New(),
		Email:                 email,
		PasswordHash:          string(hashedPassword),
		FirstName:             "Test",
		LastName:              "User",
		Company:               "Test Company",
		EmailVerified:         true,
		IsActive:              true,
		FailedLoginAttempts:   5,
		LockedUntil:           &lockUntil,
		SimulationPreferences: make(models.JSONMap),
		UIPreferences:         make(models.JSONMap),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	err = userRepo.Create(user)
	require.NoError(t, err)
	return user
}

func createRegisterRequest(email, password, firstName, lastName, company string) *models.RegisterRequest {
	return &models.RegisterRequest{
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		Company:   company,
	}
}

func createLoginRequest(email, password string, remember bool) *models.LoginRequest {
	return &models.LoginRequest{
		Email:    email,
		Password: password,
		Remember: remember,
	}
}

func createUserWithEmailVerificationToken(t *testing.T, userRepo *repository.UserRepository, email, token string, expiresAt time.Time, verified bool) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		ID:                         uuid.New(),
		Email:                      email,
		PasswordHash:               string(hashedPassword),
		FirstName:                  "Test",
		LastName:                   "User",
		Company:                    "Test Company",
		EmailVerified:              verified,
		EmailVerificationToken:     &token,
		EmailVerificationExpiresAt: &expiresAt,
		IsActive:                   true,
		SimulationPreferences:      make(models.JSONMap),
		UIPreferences:              make(models.JSONMap),
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}

	err = userRepo.Create(user)
	require.NoError(t, err)
	return user
}

func createUserWithPasswordResetToken(t *testing.T, userRepo *repository.UserRepository, email, password, token string, expiresAt time.Time) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		ID:                      uuid.New(),
		Email:                   email,
		PasswordHash:            string(hashedPassword),
		FirstName:               "Test",
		LastName:                "User",
		Company:                 "Test Company",
		EmailVerified:           true,
		IsActive:                true,
		PasswordResetToken:      &token,
		PasswordResetExpiresAt:  &expiresAt,
		PasswordResetAttempts:   1,
		SimulationPreferences:   make(models.JSONMap),
		UIPreferences:           make(models.JSONMap),
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	err = userRepo.Create(user)
	require.NoError(t, err)
	return user
}

func createInactiveUser(t *testing.T, userRepo *repository.UserRepository, email, password string) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		ID:                    uuid.New(),
		Email:                 email,
		PasswordHash:          string(hashedPassword),
		FirstName:             "Test",
		LastName:              "User",
		Company:               "Test Company",
		EmailVerified:         true,
		IsActive:              false, // Inactive user
		SimulationPreferences: make(models.JSONMap),
		UIPreferences:         make(models.JSONMap),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	err = userRepo.Create(user)
	require.NoError(t, err)
	return user
}

func createUserWithFailedAttempts(t *testing.T, userRepo *repository.UserRepository, email, password string, attempts int) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		ID:                    uuid.New(),
		Email:                 email,
		PasswordHash:          string(hashedPassword),
		FirstName:             "Test",
		LastName:              "User",
		Company:               "Test Company",
		EmailVerified:         true,
		IsActive:              true,
		FailedLoginAttempts:   attempts,
		SimulationPreferences: make(models.JSONMap),
		UIPreferences:         make(models.JSONMap),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	err = userRepo.Create(user)
	require.NoError(t, err)
	return user
}

func TestAuthService_Register(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	t.Run("successful_registration", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		req := &models.RegisterRequest{
			Email:     "test@example.com",
			Password:  "StrongP@ssw0rd123",
			FirstName: "John",
			LastName:  "Doe",
			Company:   "Test Corp",
		}

		// Execute
		response, err := authService.Register(req)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify response structure
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.NotEmpty(t, response.SessionID)
		assert.Equal(t, req.Email, response.User.Email)
		assert.Equal(t, req.FirstName, response.User.FirstName)
		assert.Equal(t, req.LastName, response.User.LastName)
		assert.Equal(t, req.Company, response.User.Company)
		assert.False(t, response.User.EmailVerified) // Should be false initially
		assert.True(t, response.User.IsActive)
		assert.False(t, response.RememberMe) // Registration doesn't use remember me

		// Verify user was created in database with proper password hashing
		user, err := repos.UserRepo.GetByEmail(req.Email)
		require.NoError(t, err)
		assert.Equal(t, req.Email, user.Email)
		assert.NotEqual(t, req.Password, user.PasswordHash) // Password should be hashed

		// Verify password was hashed correctly
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
		assert.NoError(t, err)

		// Verify email verification token was generated
		assert.NotNil(t, user.EmailVerificationToken)
		assert.NotNil(t, user.EmailVerificationExpiresAt)
		assert.False(t, user.EmailVerified)

		// Verify session was created
		sessionUUID, err := uuid.Parse(response.SessionID)
		require.NoError(t, err)
		session, err := repos.SessionRepo.GetByID(sessionUUID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, session.UserID)
		assert.True(t, session.IsActive)

		// Verify JWT tokens are valid
		claims, err := security.NewJWTManager("test-secret", time.Hour, 24*time.Hour, "test").ValidateToken(response.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "access", claims.TokenType)
		assert.Equal(t, user.ID.String(), claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
	})

	t.Run("duplicate_email_registration", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		req := &models.RegisterRequest{
			Email:     "duplicate@example.com",
			Password:  "StrongP@ssw0rd123",
			FirstName: "Jane",
			LastName:  "Doe",
			Company:   "Test Corp",
		}

		// Create first user
		_, err := authService.Register(req)
		require.NoError(t, err)

		// Try to register again with same email
		_, err = authService.Register(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")

		// Verify only one user exists in database
		user, err := repos.UserRepo.GetByEmail(req.Email)
		require.NoError(t, err)
		assert.Equal(t, req.Email, user.Email)
	})

	t.Run("weak_password_rejection", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		weakPasswords := []struct {
			password string
			reason   string
		}{
			{"123", "too short"},
			{"password", "no uppercase"},
			{"PASSWORD", "no lowercase"},
			{"Password", "no digit"},
			{"Password123", "no special character"},
			{"Pass1!", "too short"},
			{strings.Repeat("a", 129), "too long"},
		}

		for _, test := range weakPasswords {
			req := &models.RegisterRequest{
				Email:     "weak@example.com",
				Password:  test.password,
				FirstName: "Weak",
				LastName:  "Password",
				Company:   "Test Corp",
			}

			_, err := authService.Register(req)
			assert.Error(t, err, "Password '%s' should be rejected (%s)", test.password, test.reason)
		}
	})

	t.Run("password_security_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Test common weak passwords are rejected
		commonWeakPasswords := []string{
			"Password123!",
			"Admin123!",
			"Qwerty123!",
			"Letmein123!",
		}

		for _, password := range commonWeakPasswords {
			req := createRegisterRequest("weak@example.com", password, "Test", "User", "Test Corp")

			_, err := authService.Register(req)
			assert.Error(t, err, "Common weak password '%s' should be rejected", password)
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	t.Run("successful_login_normal", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create test user
		testUser := createTestUserWithPassword(t, repos.UserRepo, "login@example.com", "StrongP@ssw0rd123")

		req := createLoginRequest("login@example.com", "StrongP@ssw0rd123", false)
		userAgent := "test-user-agent"
		ipAddress := "127.0.0.1"

		// Execute
		response, err := authService.Login(req, &userAgent, &ipAddress)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify response structure
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.NotEmpty(t, response.SessionID)
		assert.Equal(t, req.Email, response.User.Email)
		assert.False(t, response.RememberMe)

		// Verify session was created with correct metadata
		sessionUUID, err := uuid.Parse(response.SessionID)
		require.NoError(t, err)
		session, err := repos.SessionRepo.GetByID(sessionUUID)
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, session.UserID)
		assert.Equal(t, &userAgent, session.UserAgent)
		assert.Equal(t, &ipAddress, session.IPAddress)
		assert.True(t, session.IsActive)

		// Verify user login info was updated
		updatedUser, err := repos.UserRepo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.NotNil(t, updatedUser.LastLoginAt)
		assert.Equal(t, &ipAddress, updatedUser.LastLoginIP)
		assert.Equal(t, 0, updatedUser.FailedLoginAttempts)
		assert.Nil(t, updatedUser.LockedUntil)
	})

	t.Run("successful_login_with_remember_me", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create test user
		_ = createTestUserWithPassword(t, repos.UserRepo, "remember@example.com", "StrongP@ssw0rd123")

		req := createLoginRequest("remember@example.com", "StrongP@ssw0rd123", true)
		userAgent := "test-user-agent"
		ipAddress := "127.0.0.1"

		// Execute
		response, err := authService.Login(req, &userAgent, &ipAddress)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.True(t, response.RememberMe)

		// Verify extended session duration for remember me
		sessionUUID, err := uuid.Parse(response.SessionID)
		require.NoError(t, err)
		session, err := repos.SessionRepo.GetByID(sessionUUID)
		require.NoError(t, err)

		// Remember me should have extended expiry (30 days for access, 90 days for refresh)
		assert.True(t, session.ExpiresAt.After(time.Now().Add(25*24*time.Hour))) // At least 25 days
		if session.RefreshExpiresAt != nil {
			assert.True(t, session.RefreshExpiresAt.After(time.Now().Add(85*24*time.Hour))) // At least 85 days
		}
	})

	t.Run("wrong_password_increments_failed_attempts", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create test user
		testUser := createTestUserWithPassword(t, repos.UserRepo, "wrong@example.com", "StrongP@ssw0rd123")

		req := createLoginRequest("wrong@example.com", "wrongpassword", false)
		userAgent := "test-user-agent"
		ipAddress := "127.0.0.1"

		// Execute
		_, err := authService.Login(req, &userAgent, &ipAddress)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")

		// Verify failed login attempts were incremented
		updatedUser, err := repos.UserRepo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, updatedUser.FailedLoginAttempts)
		assert.Nil(t, updatedUser.LockedUntil)
	})

	t.Run("nonexistent_user_returns_generic_error", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		req := createLoginRequest("nonexistent@example.com", "password", false)
		userAgent := "test-user-agent"
		ipAddress := "127.0.0.1"

		// Execute
		_, err := authService.Login(req, &userAgent, &ipAddress)

		// Assert - should not reveal if user exists
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("inactive_user_cannot_login", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create inactive user
		_ = createInactiveUser(t, repos.UserRepo, "inactive@example.com", "StrongP@ssw0rd123")

		req := createLoginRequest("inactive@example.com", "StrongP@ssw0rd123", false)
		userAgent := "test-user-agent"
		ipAddress := "127.0.0.1"

		// Execute
		_, err := authService.Login(req, &userAgent, &ipAddress)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account is not active")
	})
}

func TestAuthService_AccountLockout(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	t.Run("progressive_lockout_after_5_failed_attempts", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create test user
		testUser := createTestUserWithPassword(t, repos.UserRepo, "lockout@example.com", "StrongP@ssw0rd123")

		wrongPasswordRequest := createLoginRequest("lockout@example.com", "wrongpassword", false)
		userAgent := "test-user-agent"
		ipAddress := "127.0.0.1"

		// First 4 attempts should fail but not lock account
		for i := 0; i < 4; i++ {
			_, err := authService.Login(wrongPasswordRequest, &userAgent, &ipAddress)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid credentials")

			// Check user is not locked yet
			user, err := repos.UserRepo.GetByID(testUser.ID)
			require.NoError(t, err)
			assert.Equal(t, i+1, user.FailedLoginAttempts)
			assert.Nil(t, user.LockedUntil)
		}

		// 5th attempt should lock the account
		_, err := authService.Login(wrongPasswordRequest, &userAgent, &ipAddress)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")

		// Check user is now locked
		user, err := repos.UserRepo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, user.FailedLoginAttempts)
		assert.NotNil(t, user.LockedUntil)
		assert.True(t, user.LockedUntil.After(time.Now()))
		assert.True(t, user.LockedUntil.Before(time.Now().Add(16*time.Minute))) // Should be ~15 minutes

		// Correct password should also fail when account is locked
		correctPasswordRequest := createLoginRequest("lockout@example.com", "StrongP@ssw0rd123", false)

		_, err = authService.Login(correctPasswordRequest, &userAgent, &ipAddress)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account is locked")
	})

	t.Run("successful_login_resets_failed_attempts", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create test user with some failed attempts
		testUser := createUserWithFailedAttempts(t, repos.UserRepo, "reset@example.com", "StrongP@ssw0rd123", 3)

		correctRequest := createLoginRequest("reset@example.com", "StrongP@ssw0rd123", false)
		userAgent := "test-user-agent"
		ipAddress := "127.0.0.1"

		// Execute successful login
		response, err := authService.Login(correctRequest, &userAgent, &ipAddress)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify failed attempts were reset
		updatedUser, err := repos.UserRepo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, updatedUser.FailedLoginAttempts)
		assert.Nil(t, updatedUser.LockedUntil)
		assert.NotNil(t, updatedUser.LastLoginAt)
	})

	t.Run("locked_user_cannot_login_even_with_correct_password", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create locked user
		_ = createLockedTestUser(t, repos.UserRepo, "locked@example.com")

		correctRequest := createLoginRequest("locked@example.com", "testpassword123", false)
		userAgent := "test-user-agent"
		ipAddress := "127.0.0.1"

		// Execute
		_, err := authService.Login(correctRequest, &userAgent, &ipAddress)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account is locked")
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	// Create test user and login
	_ = createTestUserWithPassword(t, repos.UserRepo, "refresh@example.com", "testpassword123")
	
	loginRequest := createLoginRequest("refresh@example.com", "testpassword123", false)

	userAgent := "test-user-agent"
	ipAddress := "127.0.0.1"
	loginResponse, err := authService.Login(loginRequest, &userAgent, &ipAddress)
	require.NoError(t, err)

	tests := []struct {
		name         string
		refreshToken string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "successful_refresh",
			refreshToken: loginResponse.RefreshToken,
			expectError:  false,
		},
		{
			name:         "invalid_refresh_token",
			refreshToken: "invalid-token",
			expectError:  true,
			errorMsg:     "invalid refresh token",
		},
		{
			name:         "empty_refresh_token",
			refreshToken: "",
			expectError:  true,
			errorMsg:     "refresh token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			response, err := authService.RefreshToken(tt.refreshToken)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)

				// New tokens should be different from original
				assert.NotEqual(t, loginResponse.AccessToken, response.AccessToken)
				assert.NotEqual(t, loginResponse.RefreshToken, response.RefreshToken)
			}
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	// Create test user and login
	_ = createTestUserWithPassword(t, repos.UserRepo, "logout@example.com", "testpassword123")

	loginRequest := createLoginRequest("logout@example.com", "testpassword123", false)

	userAgent := "test-user-agent"
	ipAddress := "127.0.0.1"
	loginResponse, err := authService.Login(loginRequest, &userAgent, &ipAddress)
	require.NoError(t, err)

	tests := []struct {
		name        string
		sessionID   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successful_logout",
			sessionID:   loginResponse.SessionID,
			expectError: false,
		},
		{
			name:        "invalid_session_id",
			sessionID:   "invalid-session-id",
			expectError: true,
			errorMsg:    "session not found",
		},
		{
			name:        "empty_session_id",
			sessionID:   "",
			expectError: true,
			errorMsg:    "session ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			sessionUUID, _ := uuid.Parse(tt.sessionID)
			err := authService.Logout(sessionUUID)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify session was revoked
				sessionUUID, _ := uuid.Parse(tt.sessionID)
				session, err := repos.SessionRepo.GetByID(sessionUUID)
				if err == nil {
					assert.False(t, session.IsActive) // Should be revoked
				}
			}
		})
	}
}

func TestAuthService_EmailVerification(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	t.Run("successful_email_verification", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create unverified user with verification token
		token := "valid_verification_token_123"
		expiresAt := time.Now().Add(24 * time.Hour)
		testUser := createUserWithEmailVerificationToken(t, repos.UserRepo, "verify@example.com", token, expiresAt, false)

		// Execute
		err := authService.VerifyEmail(token)

		// Assert
		require.NoError(t, err)

		// Verify user is now verified and token is cleared
		updatedUser, err := repos.UserRepo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.True(t, updatedUser.EmailVerified)
		assert.Nil(t, updatedUser.EmailVerificationToken)
		assert.Nil(t, updatedUser.EmailVerificationExpiresAt)
	})

	t.Run("invalid_verification_token", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Execute with invalid token
		err := authService.VerifyEmail("invalid_token")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid verification token")
	})

	t.Run("expired_verification_token", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create user with expired verification token
		token := "expired_token_123"
		expiresAt := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
		testUser := createUserWithEmailVerificationToken(t, repos.UserRepo, "expired@example.com", token, expiresAt, false)

		// Execute
		err := authService.VerifyEmail(token)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "verification token has expired")

		// Verify user is still unverified
		updatedUser, err := repos.UserRepo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.False(t, updatedUser.EmailVerified)
	})
}

func TestAuthService_PasswordReset(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	t.Run("successful_forgot_password", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create test user
		testUser := createTestUserWithPassword(t, repos.UserRepo, "forgot@example.com", "OldP@ssw0rd123")

		// Execute
		response, err := authService.ForgotPassword("forgot@example.com")

		// Assert
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.NotEmpty(t, response.ResetToken)
		assert.Equal(t, int64(3600), response.ExpiresIn) // 1 hour
		assert.Contains(t, response.Message, "generated successfully")

		// Verify user has reset token and expiry
		updatedUser, err := repos.UserRepo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.NotNil(t, updatedUser.PasswordResetToken)
		assert.NotNil(t, updatedUser.PasswordResetExpiresAt)
		assert.Equal(t, 1, updatedUser.PasswordResetAttempts)
		assert.True(t, updatedUser.PasswordResetExpiresAt.After(time.Now()))
	})

	t.Run("forgot_password_nonexistent_user_no_error", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Execute with nonexistent email
		response, err := authService.ForgotPassword("nonexistent@example.com")

		// Assert - should not reveal if email exists
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Contains(t, response.Message, "If the email exists")
	})

	t.Run("successful_password_reset", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create user with reset token
		resetToken := "valid_reset_token_123"
		expiresAt := time.Now().Add(1 * time.Hour)
		testUser := createUserWithPasswordResetToken(t, repos.UserRepo, "reset@example.com", "OldP@ssw0rd123", resetToken, expiresAt)

		oldPasswordHash := testUser.PasswordHash
		newPassword := "NewStr0ngP@ssw0rd!"

		// Execute
		err := authService.ResetPassword(resetToken, newPassword)

		// Assert
		require.NoError(t, err)

		// Verify password was changed and token cleared
		updatedUser, err := repos.UserRepo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.NotEqual(t, oldPasswordHash, updatedUser.PasswordHash)
		assert.Nil(t, updatedUser.PasswordResetToken)
		assert.Nil(t, updatedUser.PasswordResetExpiresAt)
		assert.Equal(t, 0, updatedUser.PasswordResetAttempts)

		// Verify new password works
		err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte(newPassword))
		assert.NoError(t, err)

		// Verify old password no longer works
		err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte("OldP@ssw0rd123"))
		assert.Error(t, err)
	})

	t.Run("invalid_reset_token", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Execute with invalid token
		err := authService.ResetPassword("invalid_token", "NewP@ssw0rd123")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid reset token")
	})

	t.Run("expired_reset_token", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create user with expired reset token
		resetToken := "expired_reset_token_123"
		expiresAt := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
		testUser := createUserWithPasswordResetToken(t, repos.UserRepo, "expired@example.com", "OldP@ssw0rd123", resetToken, expiresAt)

		oldPasswordHash := testUser.PasswordHash

		// Execute
		err := authService.ResetPassword(resetToken, "NewP@ssw0rd123")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reset token has expired")

		// Verify password was not changed
		updatedUser, err := repos.UserRepo.GetByID(testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, oldPasswordHash, updatedUser.PasswordHash)
	})

	t.Run("weak_password_in_reset_rejected", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create user with reset token
		resetToken := "valid_reset_token_123"
		expiresAt := time.Now().Add(1 * time.Hour)
		_ = createUserWithPasswordResetToken(t, repos.UserRepo, "weak@example.com", "OldP@ssw0rd123", resetToken, expiresAt)

		// Execute with weak password
		err := authService.ResetPassword(resetToken, "weak")

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password")
	})
}

func TestAuthService_RateLimiting(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	t.Run("password_reset_rate_limiting", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create test user
		_ = createTestUserWithPassword(t, repos.UserRepo, "ratelimit@example.com", "TestP@ssw0rd123")

		// First 3 attempts should succeed
		for i := 0; i < 3; i++ {
			_, err := authService.ForgotPassword("ratelimit@example.com")
			assert.NoError(t, err, "Attempt %d should succeed", i+1)
		}

		// 4th attempt should be rate limited
		_, err := authService.ForgotPassword("ratelimit@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit")
	})

	t.Run("email_verification_rate_limiting", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create unverified user
		token := "rate_limit_token"
		expiresAt := time.Now().Add(24 * time.Hour)
		user := createUserWithEmailVerificationToken(t, repos.UserRepo, "verify_rate@example.com", token, expiresAt, false)

		// Set high verification attempts to test rate limiting
		user.EmailVerificationAttempts = 5
		err := repos.UserRepo.Update(user)
		require.NoError(t, err)

		// Should be rate limited
		err = authService.VerifyEmail(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many attempts")
	})
}

func TestAuthService_SecurityFeatures(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	t.Run("password_security_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Test various insecure passwords
		insecurePasswords := []struct {
			password string
			reason   string
		}{
			{"password", "common password"},
			{"123456789", "sequential numbers"},
			{"qwerty123", "keyboard pattern"},
			{"admin123", "common admin password"},
			{"Password1", "too predictable"},
			{"test@test.com", "contains email pattern"},
		}

		for _, test := range insecurePasswords {
			req := createRegisterRequest("security@example.com", test.password, "Security", "Test", "Test Corp")

			_, err := authService.Register(req)
			assert.Error(t, err, "Password '%s' should be rejected (%s)", test.password, test.reason)
		}
	})

	t.Run("token_security_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Test invalid token formats
		invalidTokens := []string{
			"",
			"short",
			"invalid-token-format",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid",
			strings.Repeat("a", 1000), // Very long token
		}

		for _, token := range invalidTokens {
			err := authService.VerifyEmail(token)
			assert.Error(t, err, "Invalid token '%s' should be rejected", token)

			err = authService.ResetPassword(token, "NewP@ssw0rd123")
			assert.Error(t, err, "Invalid reset token '%s' should be rejected", token)
		}
	})

	t.Run("session_security", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create user and login
		user := createTestUserWithPassword(t, repos.UserRepo, "session@example.com", "SessionTest123!")

		loginReq := createLoginRequest("session@example.com", "SessionTest123!", false)
		userAgent := "test-agent"
		ipAddress := "192.168.1.100"

		response, err := authService.Login(loginReq, &userAgent, &ipAddress)
		require.NoError(t, err)

		// Verify session has security metadata
		sessionUUID, err := uuid.Parse(response.SessionID)
		require.NoError(t, err)

		session, err := repos.SessionRepo.GetByID(sessionUUID)
		require.NoError(t, err)

		assert.Equal(t, user.ID, session.UserID)
		assert.Equal(t, &userAgent, session.UserAgent)
		assert.Equal(t, &ipAddress, session.IPAddress)
		assert.True(t, session.IsActive)
		assert.NotEmpty(t, session.TokenHash)

		// Test session expiry
		assert.True(t, session.ExpiresAt.After(time.Now()))
		assert.True(t, session.ExpiresAt.Before(time.Now().Add(2*time.Hour)))
	})
}

func TestAuthService_ErrorScenarios(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	t.Run("database_connection_errors", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Close database connection to simulate error
		db.Close()

		req := createRegisterRequest("db_error@example.com", "TestP@ssw0rd123", "DB", "Error", "Test Corp")

		_, err := authService.Register(req)
		assert.Error(t, err, "Should fail when database is unavailable")
	})

	t.Run("redis_connection_errors", func(t *testing.T) {
		// This test would require mocking Redis failures
		// For now, we'll test with Redis unavailable scenarios in integration tests
		t.Skip("Redis connection error testing requires more complex setup")
	})

	t.Run("malformed_input_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Test malformed email addresses
		malformedEmails := []string{
			"",
			"invalid",
			"@example.com",
			"user@",
			"user@.com",
			"user..user@example.com",
			strings.Repeat("a", 250) + "@example.com", // Very long email
		}

		for _, email := range malformedEmails {
			req := createRegisterRequest(email, "ValidP@ssw0rd123", "Test", "User", "Test Corp")

			_, err := authService.Register(req)
			assert.Error(t, err, "Malformed email '%s' should be rejected", email)
		}

		// Test malformed names
		malformedNames := []struct {
			firstName, lastName string
		}{
			{"", "LastName"},
			{"FirstName", ""},
			{strings.Repeat("a", 101), "LastName"}, // Too long
			{"First<script>", "LastName"},          // XSS attempt
			{"First\x00Name", "LastName"},          // Null byte
		}

		for _, names := range malformedNames {
			req := createRegisterRequest("malformed@example.com", "ValidP@ssw0rd123", names.firstName, names.lastName, "Test Corp")

			_, err := authService.Register(req)
			assert.Error(t, err, "Malformed names '%s %s' should be rejected", names.firstName, names.lastName)
		}
	})

	t.Run("concurrent_session_management", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create user
		user := createTestUserWithPassword(t, repos.UserRepo, "concurrent@example.com", "ConcurrentTest123!")

		// Test concurrent logins
		const numConcurrentLogins = 5
		results := make(chan error, numConcurrentLogins)
		sessionIDs := make(chan string, numConcurrentLogins)

		for i := 0; i < numConcurrentLogins; i++ {
			go func(index int) {
				loginReq := createLoginRequest("concurrent@example.com", "ConcurrentTest123!", false)
				userAgent := fmt.Sprintf("test-agent-%d", index)
				ipAddress := fmt.Sprintf("192.168.1.%d", 100+index)

				response, err := authService.Login(loginReq, &userAgent, &ipAddress)
				results <- err
				if err == nil {
					sessionIDs <- response.SessionID
				}
			}(i)
		}

		// Collect results
		var errors []error
		var sessions []string
		for i := 0; i < numConcurrentLogins; i++ {
			if err := <-results; err != nil {
				errors = append(errors, err)
			} else {
				sessions = append(sessions, <-sessionIDs)
			}
		}

		assert.Empty(t, errors, "No errors should occur during concurrent logins")
		assert.Len(t, sessions, numConcurrentLogins, "All concurrent logins should succeed")

		// Verify all sessions are unique
		sessionSet := make(map[string]bool)
		for _, sessionID := range sessions {
			assert.False(t, sessionSet[sessionID], "Session IDs should be unique")
			sessionSet[sessionID] = true
		}

		// Verify user login info was updated correctly
		updatedUser, err := repos.UserRepo.GetByID(user.ID)
		require.NoError(t, err)
		assert.NotNil(t, updatedUser.LastLoginAt)
		assert.Equal(t, 0, updatedUser.FailedLoginAttempts)
	})

	t.Run("edge_case_token_expiry", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create user with token that expires in 1 second
		token := "expiring_token_123"
		expiresAt := time.Now().Add(1 * time.Second)
		user := createUserWithEmailVerificationToken(t, repos.UserRepo, "expiry@example.com", token, expiresAt, false)

		// Wait for token to expire
		time.Sleep(2 * time.Second)

		// Should fail due to expiry
		err := authService.VerifyEmail(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")

		// Verify user is still unverified
		updatedUser, err := repos.UserRepo.GetByID(user.ID)
		require.NoError(t, err)
		assert.False(t, updatedUser.EmailVerified)
	})
}

func TestAuthService_ComprehensiveValidation(t *testing.T) {
	// Setup
	db := testutils.SetupTestDB(t)
	defer db.Close()

	redis := testutils.SetupTestRedis(t)
	defer redis.Close()

	authService, repos := createAuthService(t, db, redis)
	defer testutils.CleanupTestDB(t, db)
	defer testutils.CleanupTestRedis(t, redis)

	t.Run("register_comprehensive_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Test nil request
		_, err := authService.Register(nil)
		assert.Error(t, err)

		// Test empty request
		_, err = authService.Register(&models.RegisterRequest{})
		assert.Error(t, err)

		// Test SQL injection attempts
		sqlInjectionAttempts := []string{
			"'; DROP TABLE users; --",
			"admin@example.com'; INSERT INTO users",
			"test@example.com' OR '1'='1",
		}

		for _, maliciousEmail := range sqlInjectionAttempts {
			req := createRegisterRequest(maliciousEmail, "ValidP@ssw0rd123", "Test", "User", "Test Corp")

			_, err := authService.Register(req)
			assert.Error(t, err, "SQL injection attempt should be rejected: %s", maliciousEmail)
		}

		// Test XSS attempts in all fields
		xssPayloads := []string{
			"<script>alert('xss')</script>",
			"javascript:alert('xss')",
			"<img src=x onerror=alert('xss')>",
			"';alert('xss');//",
		}

		for _, payload := range xssPayloads {
			req := createRegisterRequest("xss@example.com", "ValidP@ssw0rd123", payload, payload, payload)

			_, err := authService.Register(req)
			assert.Error(t, err, "XSS payload should be rejected: %s", payload)
		}
	})

	t.Run("login_comprehensive_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Create valid user for testing
		_ = createTestUserWithPassword(t, repos.UserRepo, "valid@example.com", "ValidP@ssw0rd123")

		// Test nil request
		_, err := authService.Login(nil, nil, nil)
		assert.Error(t, err)

		// Test empty request
		_, err = authService.Login(&models.LoginRequest{}, nil, nil)
		assert.Error(t, err)

		// Test with nil user agent and IP (should still work)
		loginReq := createLoginRequest("valid@example.com", "ValidP@ssw0rd123", false)
		_, err = authService.Login(loginReq, nil, nil)
		assert.NoError(t, err, "Login should work with nil user agent and IP")

		// Test with very long user agent
		longUserAgent := strings.Repeat("a", 2000)
		_, err = authService.Login(loginReq, &longUserAgent, nil)
		assert.NoError(t, err, "Should handle long user agent gracefully")

		// Test with invalid IP formats
		invalidIPs := []string{
			"999.999.999.999",
			"not.an.ip.address",
			"192.168.1",
			"::ffff:192.168.1.999",
		}

		for _, invalidIP := range invalidIPs {
			userAgent := "test-agent"
			_, err = authService.Login(loginReq, &userAgent, &invalidIP)
			// Should not fail due to invalid IP format, but should handle gracefully
			// The service should sanitize or handle invalid IPs appropriately
		}
	})

	t.Run("token_refresh_comprehensive_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Test empty token
		_, err := authService.RefreshToken("")
		assert.Error(t, err)

		// Test malformed tokens
		malformedTokens := []string{
			"invalid.token.format",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid",
			strings.Repeat("a", 5000), // Very long token
			"null",
			"undefined",
		}

		for _, token := range malformedTokens {
			_, err := authService.RefreshToken(token)
			assert.Error(t, err, "Malformed token should be rejected: %s", token)
		}
	})

	t.Run("logout_comprehensive_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Test with nil UUID
		err := authService.Logout(uuid.Nil)
		assert.Error(t, err)

		// Test with random UUID (non-existent session)
		randomUUID := uuid.New()
		err = authService.Logout(randomUUID)
		assert.Error(t, err, "Should fail for non-existent session")
	})

	t.Run("email_verification_comprehensive_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Test empty token
		err := authService.VerifyEmail("")
		assert.Error(t, err)

		// Test tokens with special characters
		specialTokens := []string{
			"token with spaces",
			"token\nwith\nnewlines",
			"token\x00with\x00nulls",
			"token;with;semicolons",
			"token'with'quotes",
		}

		for _, token := range specialTokens {
			err := authService.VerifyEmail(token)
			assert.Error(t, err, "Special character token should be rejected: %s", token)
		}
	})

	t.Run("password_reset_comprehensive_validation", func(t *testing.T) {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redis)

		// Test forgot password with empty email
		_, err := authService.ForgotPassword("")
		assert.Error(t, err)

		// Test reset password with empty token
		err = authService.ResetPassword("", "NewP@ssw0rd123")
		assert.Error(t, err)

		// Test reset password with empty password
		err = authService.ResetPassword("valid_token", "")
		assert.Error(t, err)

		// Test reset password with same password patterns
		samePasswordPatterns := []string{
			"password",
			"Password",
			"PASSWORD",
			"p@ssword",
			"P@SSWORD",
		}

		for _, password := range samePasswordPatterns {
			err = authService.ResetPassword("valid_token", password)
			assert.Error(t, err, "Weak password pattern should be rejected: %s", password)
		}
	})
}
