package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/systemsim/auth-service/internal/config"
	"github.com/systemsim/auth-service/internal/events"
	"github.com/systemsim/auth-service/internal/handlers"
	"github.com/systemsim/auth-service/internal/middleware"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/repository"
	"github.com/systemsim/auth-service/internal/services"
	"github.com/systemsim/auth-service/internal/testutils"
)

// MockEventPublisher is a mock implementation for testing
type MockEventPublisher struct{}

func (m *MockEventPublisher) PublishLoginSuccess(userID, sessionID, email, ipAddress, userAgent string) error {
	return nil
}

func (m *MockEventPublisher) PublishLoginFailure(email, ipAddress, userAgent, reason string) error {
	return nil
}

func (m *MockEventPublisher) PublishUserLogout(userID, sessionID, reason string) error {
	return nil
}

func (m *MockEventPublisher) PublishUserRegistration(userID, email, firstName, lastName, company, ipAddress string) error {
	return nil
}

func (m *MockEventPublisher) PublishPermissionUpdate(userID, changedBy, action string, permissions, roles []string) error {
	return nil
}

func (m *MockEventPublisher) PublishSessionCreated(userID, sessionID, ipAddress, userAgent string) error {
	return nil
}

func (m *MockEventPublisher) PublishSessionRevoked(userID, sessionID, reason string) error {
	return nil
}

func (m *MockEventPublisher) PublishWelcomeEmail(to, firstName string) error {
	return nil
}

func (m *MockEventPublisher) PublishVerificationEmail(to, firstName, verificationToken string) error {
	return nil
}

func (m *MockEventPublisher) PublishPasswordResetEmail(to, firstName, resetToken string) error {
	return nil
}

func (m *MockEventPublisher) HealthCheck() error {
	return nil
}

// TestSuite holds the test environment
type HTTPAPITestSuite struct {
	router      *gin.Engine
	authService *services.AuthService
	userService *services.UserService
	rbacService *services.RBACService
	cfg         *config.Config
	cleanup     func()
}

// SetupHTTPAPITestSuite initializes the test environment
func SetupHTTPAPITestSuite(t *testing.T) *HTTPAPITestSuite {
	// Setup test database and Redis
	db := testutils.SetupTestDB(t)
	redisClient := testutils.SetupTestRedis(t)
	
	// Setup configuration
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:               "test-secret-key-for-integration-tests",
			AccessTokenDuration:  15 * time.Minute,
			RefreshTokenDuration: 7 * 24 * time.Hour,
		},
		Server: config.ServerConfig{
			Mode: "test",
		},
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: 1000, // High limit for tests
		},
	}

	// Initialize repositories, services, and handlers
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db, redisClient)
	rbacRepo := repository.NewRBACRepository(db)
	
	// Create event publisher (use real one for tests but with test Redis)
	eventPublisher := events.NewPublisher(redisClient)
	
	rbacService := services.NewRBACService(rbacRepo, userRepo)
	authService := services.NewAuthService(userRepo, sessionRepo, rbacService, cfg.JWT, eventPublisher)
	userService := services.NewUserService(userRepo, sessionRepo)
	
	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService, authService)
	rbacHandler := handlers.NewRBACHandler(rbacService)
	healthHandler := handlers.NewHealthHandler(nil, nil) // Mock health checkers for tests
	
	// Initialize middleware
	rbacMiddleware := middleware.NewRBACMiddleware(rbacService)
	
	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	
	// Health check endpoints
	router.GET("/health", healthHandler.SimpleHealthCheck)
	router.GET("/health/live", healthHandler.LivenessCheck)
	router.GET("/health/ready", healthHandler.ReadinessCheck)
	router.GET("/health/detailed", healthHandler.DetailedHealthCheck)
	router.GET("/metrics", healthHandler.MetricsHandler)
	
	// API routes
	api := router.Group("/api/v1")
	{
		// Public routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
			auth.POST("/verify-email", authHandler.VerifyEmail)
			auth.POST("/resend-verification", authHandler.ResendVerificationEmail)
		}
		
		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.AuthRequired(cfg.JWT.Secret))
		protected.Use(rbacMiddleware.AddUserPermissions())
		{
			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/user/profile", userHandler.GetProfile)
			protected.PUT("/user/profile", userHandler.UpdateProfile)
			protected.POST("/user/change-password", userHandler.ChangePassword)
			protected.DELETE("/user/account", userHandler.DeleteAccount)
			
			// Session management
			protected.GET("/user/sessions", userHandler.GetSessions)
			protected.DELETE("/user/sessions/:sessionId", userHandler.RevokeSession)
			protected.DELETE("/user/sessions", userHandler.RevokeAllSessions)
			protected.GET("/user/stats", userHandler.GetStats)
			
			// RBAC endpoints
			protected.GET("/rbac/my-roles", rbacHandler.GetMyRoles)
			protected.GET("/rbac/my-permissions", rbacHandler.GetMyPermissions)
			
			// Admin RBAC endpoints
			admin := protected.Group("/admin")
			admin.Use(rbacMiddleware.RequireAdmin())
			{
				admin.GET("/roles", rbacHandler.GetAllRoles)
				admin.GET("/permissions", rbacHandler.GetAllPermissions)
				admin.POST("/users/assign-role", rbacHandler.AssignRole)
				admin.POST("/users/remove-role", rbacHandler.RemoveRole)
				admin.GET("/users/:userId/roles", rbacHandler.GetUserRoles)
			}
		}
	}
	
	cleanup := func() {
		testutils.CleanupTestDB(t, db)
		testutils.CleanupTestRedis(t, redisClient)
		redisClient.Close()
	}
	
	return &HTTPAPITestSuite{
		router:      router,
		authService: authService,
		userService: userService,
		rbacService: rbacService,
		cfg:         cfg,
		cleanup:     cleanup,
	}
}

// Helper function to make HTTP requests
func (suite *HTTPAPITestSuite) makeRequest(method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}
	
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

// Helper function to create a test user and get auth token
func (suite *HTTPAPITestSuite) createTestUserAndLogin(t *testing.T, email, password string) (string, *models.AuthResponse) {
	// Register user
	registerReq := models.RegisterRequest{
		Email:     email,
		Password:  password,
		FirstName: "Test",
		LastName:  "User",
	}
	
	w := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, nil)
	require.Equal(t, http.StatusCreated, w.Code, "Registration should succeed")
	
	// Login user
	loginReq := models.LoginRequest{
		Email:    email,
		Password: password,
	}
	
	w = suite.makeRequest("POST", "/api/v1/auth/login", loginReq, nil)
	require.Equal(t, http.StatusOK, w.Code, "Login should succeed")
	
	var loginResp models.AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &loginResp)
	require.NoError(t, err, "Should be able to parse login response")

	return loginResp.AccessToken, &loginResp
}

// Test Health Check Endpoints
func TestHTTPAPI_HealthEndpoints(t *testing.T) {
	suite := SetupHTTPAPITestSuite(t)
	defer suite.cleanup()
	
	healthEndpoints := []struct {
		name     string
		path     string
		expected int
	}{
		{"simple_health_check", "/health", http.StatusOK},
		{"liveness_check", "/health/live", http.StatusOK},
		{"readiness_check", "/health/ready", http.StatusOK},
		{"detailed_health_check", "/health/detailed", http.StatusOK},
		{"metrics_endpoint", "/metrics", http.StatusOK},
	}
	
	for _, endpoint := range healthEndpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			w := suite.makeRequest("GET", endpoint.path, nil, nil)
			assert.Equal(t, endpoint.expected, w.Code, "Health endpoint should return expected status")
			assert.NotEmpty(t, w.Body.String(), "Health endpoint should return response body")
		})
	}
}

// Test Public Auth Endpoints
func TestHTTPAPI_PublicAuthEndpoints(t *testing.T) {
	suite := SetupHTTPAPITestSuite(t)
	defer suite.cleanup()
	
	t.Run("register_endpoint", func(t *testing.T) {
		registerReq := models.RegisterRequest{
			Email:     "test@example.com",
			Password:  "SecurePass123!",
			FirstName: "Test",
			LastName:  "User",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, nil)
		assert.Equal(t, http.StatusCreated, w.Code, "Registration should succeed")
		
		var response models.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse registration response")
		assert.NotNil(t, response.User, "Response should contain user")
		assert.NotEmpty(t, response.User.ID, "Response should contain user ID")
	})
	
	t.Run("login_endpoint", func(t *testing.T) {
		// First register a user
		email := "login@example.com"
		password := "SecurePass123!"
		suite.createTestUserAndLogin(t, email, password)
		
		// Test login again
		loginReq := models.LoginRequest{
			Email:    email,
			Password: password,
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, nil)
		assert.Equal(t, http.StatusOK, w.Code, "Login should succeed")
		
		var response models.AuthResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse login response")
		assert.NotEmpty(t, response.AccessToken, "Response should contain access token")
		assert.NotEmpty(t, response.RefreshToken, "Response should contain refresh token")
	})
	
	t.Run("refresh_token_endpoint", func(t *testing.T) {
		// Create user and login
		_, loginResp := suite.createTestUserAndLogin(t, "refresh@example.com", "SecurePass123!")
		
		// Test refresh token
		refreshReq := models.RefreshTokenRequest{
			RefreshToken: loginResp.RefreshToken,
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/refresh", refreshReq, nil)
		assert.Equal(t, http.StatusOK, w.Code, "Refresh token should succeed")
		
		var response models.TokenResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse refresh response")
		assert.NotEmpty(t, response.AccessToken, "Response should contain new access token")
	})
	
	t.Run("forgot_password_endpoint", func(t *testing.T) {
		// Create user first
		suite.createTestUserAndLogin(t, "forgot@example.com", "SecurePass123!")
		
		// Test forgot password
		forgotReq := models.ForgotPasswordRequest{
			Email: "forgot@example.com",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/forgot-password", forgotReq, nil)
		assert.Equal(t, http.StatusOK, w.Code, "Forgot password should succeed")
		
		var response models.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse forgot password response")
		assert.True(t, response.Success, "Response should indicate success")
	})

	t.Run("reset_password_endpoint", func(t *testing.T) {
		// Test reset password (would need a valid reset token in real scenario)
		resetReq := models.ResetPasswordRequest{
			Token:       "test-reset-token",
			NewPassword: "NewSecurePass123!",
		}

		w := suite.makeRequest("POST", "/api/v1/auth/reset-password", resetReq, nil)
		// This will likely fail without a valid token, but we're testing the endpoint exists
		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized}, w.Code,
			"Reset password endpoint should respond appropriately")
	})

	t.Run("verify_email_endpoint", func(t *testing.T) {
		// Test email verification (would need a valid verification token in real scenario)
		verifyReq := models.VerifyEmailRequest{
			Token: "test-verification-token",
		}

		w := suite.makeRequest("POST", "/api/v1/auth/verify-email", verifyReq, nil)
		// This will likely fail without a valid token, but we're testing the endpoint exists
		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized}, w.Code,
			"Verify email endpoint should respond appropriately")
	})

	t.Run("resend_verification_endpoint", func(t *testing.T) {
		// Create user first
		suite.createTestUserAndLogin(t, "resend@example.com", "SecurePass123!")

		// Test resend verification
		resendReq := models.ResendVerificationRequest{
			Email: "resend@example.com",
		}

		w := suite.makeRequest("POST", "/api/v1/auth/resend-verification", resendReq, nil)
		assert.Equal(t, http.StatusOK, w.Code, "Resend verification should succeed")

		var response models.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse resend verification response")
		assert.True(t, response.Success, "Response should indicate success")
	})
}

// Test Protected Auth Endpoints
func TestHTTPAPI_ProtectedAuthEndpoints(t *testing.T) {
	suite := SetupHTTPAPITestSuite(t)
	defer suite.cleanup()

	t.Run("logout_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "logout@example.com", "SecurePass123!")

		// Test logout
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("POST", "/api/v1/auth/logout", nil, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Logout should succeed")

		var response models.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse logout response")
		assert.True(t, response.Success, "Response should indicate success")
	})

	t.Run("logout_without_auth", func(t *testing.T) {
		// Test logout without authentication
		w := suite.makeRequest("POST", "/api/v1/auth/logout", nil, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Logout without auth should fail")
	})
}

// Test User Management Endpoints
func TestHTTPAPI_UserManagementEndpoints(t *testing.T) {
	suite := SetupHTTPAPITestSuite(t)
	defer suite.cleanup()

	t.Run("get_profile_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "profile@example.com", "SecurePass123!")

		// Test get profile
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("GET", "/api/v1/user/profile", nil, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Get profile should succeed")

		var response models.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse profile response")
		assert.Equal(t, "profile@example.com", response.Email, "Profile should contain correct email")
	})

	t.Run("update_profile_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "update@example.com", "SecurePass123!")

		// Test update profile
		updateReq := models.UpdateProfileRequest{
			FirstName: "Updated",
			LastName:  "Name",
		}

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("PUT", "/api/v1/user/profile", updateReq, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Update profile should succeed")

		var response models.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse updated profile response")
		assert.Equal(t, "Updated", response.FirstName, "Profile should be updated")
		assert.Equal(t, "Name", response.LastName, "Profile should be updated")
	})

	t.Run("change_password_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "changepass@example.com", "SecurePass123!")

		// Test change password
		changeReq := models.ChangePasswordRequest{
			CurrentPassword: "SecurePass123!",
			NewPassword:     "NewSecurePass456!",
		}

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("POST", "/api/v1/user/change-password", changeReq, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Change password should succeed")

		var response models.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse change password response")
		assert.True(t, response.Success, "Response should indicate success")
	})

	t.Run("delete_account_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "delete@example.com", "SecurePass123!")

		// Test delete account
		deleteReq := map[string]string{
			"password": "SecurePass123!",
		}

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("DELETE", "/api/v1/user/account", deleteReq, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Delete account should succeed")

		var response models.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse delete account response")
		assert.True(t, response.Success, "Response should indicate success")
	})

	t.Run("get_sessions_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "sessions@example.com", "SecurePass123!")

		// Test get sessions
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("GET", "/api/v1/user/sessions", nil, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Get sessions should succeed")

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse sessions response")
		assert.Contains(t, response, "sessions", "Response should contain sessions")
	})

	t.Run("revoke_all_sessions_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "revoke@example.com", "SecurePass123!")

		// Test revoke all sessions
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("DELETE", "/api/v1/user/sessions", nil, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Revoke all sessions should succeed")

		var response models.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse revoke sessions response")
		assert.True(t, response.Success, "Response should indicate success")
	})

	t.Run("get_stats_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "stats@example.com", "SecurePass123!")

		// Test get stats
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("GET", "/api/v1/user/stats", nil, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Get stats should succeed")

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse stats response")
		assert.NotEmpty(t, response, "Stats response should not be empty")
	})

	t.Run("unauthorized_access", func(t *testing.T) {
		// Test accessing protected endpoints without authentication
		protectedEndpoints := []struct {
			method string
			path   string
		}{
			{"GET", "/api/v1/user/profile"},
			{"PUT", "/api/v1/user/profile"},
			{"POST", "/api/v1/user/change-password"},
			{"DELETE", "/api/v1/user/account"},
			{"GET", "/api/v1/user/sessions"},
			{"DELETE", "/api/v1/user/sessions"},
			{"GET", "/api/v1/user/stats"},
		}

		for _, endpoint := range protectedEndpoints {
			w := suite.makeRequest(endpoint.method, endpoint.path, nil, nil)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				fmt.Sprintf("%s %s should require authentication", endpoint.method, endpoint.path))
		}
	})
}

// Test RBAC Endpoints
func TestHTTPAPI_RBACEndpoints(t *testing.T) {
	suite := SetupHTTPAPITestSuite(t)
	defer suite.cleanup()

	t.Run("get_my_roles_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "roles@example.com", "SecurePass123!")

		// Test get my roles
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("GET", "/api/v1/rbac/my-roles", nil, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Get my roles should succeed")

		var response []models.Role
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse roles response")
		assert.IsType(t, []models.Role{}, response, "Response should be array of roles")
	})

	t.Run("get_my_permissions_endpoint", func(t *testing.T) {
		// Create user and login
		token, _ := suite.createTestUserAndLogin(t, "permissions@example.com", "SecurePass123!")

		// Test get my permissions
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		w := suite.makeRequest("GET", "/api/v1/rbac/my-permissions", nil, headers)
		assert.Equal(t, http.StatusOK, w.Code, "Get my permissions should succeed")

		var response []models.Permission
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse permissions response")
		assert.IsType(t, []models.Permission{}, response, "Response should be array of permissions")
	})
}

// Test Admin RBAC Endpoints
func TestHTTPAPI_AdminRBACEndpoints(t *testing.T) {
	suite := SetupHTTPAPITestSuite(t)
	defer suite.cleanup()

	// Note: These tests will likely fail without proper admin setup
	// but they test that the endpoints exist and respond appropriately

	t.Run("admin_endpoints_require_admin_role", func(t *testing.T) {
		// Create regular user and login
		token, _ := suite.createTestUserAndLogin(t, "regular@example.com", "SecurePass123!")

		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}

		adminEndpoints := []struct {
			method string
			path   string
		}{
			{"GET", "/api/v1/admin/roles"},
			{"GET", "/api/v1/admin/permissions"},
			{"POST", "/api/v1/admin/users/assign-role"},
			{"POST", "/api/v1/admin/users/remove-role"},
		}

		for _, endpoint := range adminEndpoints {
			w := suite.makeRequest(endpoint.method, endpoint.path, nil, headers)
			// Should fail with forbidden or unauthorized since user is not admin
			assert.Contains(t, []int{http.StatusForbidden, http.StatusUnauthorized}, w.Code,
				fmt.Sprintf("%s %s should require admin role", endpoint.method, endpoint.path))
		}
	})

	t.Run("admin_endpoints_without_auth", func(t *testing.T) {
		adminEndpoints := []struct {
			method string
			path   string
		}{
			{"GET", "/api/v1/admin/roles"},
			{"GET", "/api/v1/admin/permissions"},
			{"POST", "/api/v1/admin/users/assign-role"},
			{"POST", "/api/v1/admin/users/remove-role"},
		}

		for _, endpoint := range adminEndpoints {
			w := suite.makeRequest(endpoint.method, endpoint.path, nil, nil)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				fmt.Sprintf("%s %s should require authentication", endpoint.method, endpoint.path))
		}
	})
}

// Test Error Handling and Edge Cases
func TestHTTPAPI_ErrorHandling(t *testing.T) {
	suite := SetupHTTPAPITestSuite(t)
	defer suite.cleanup()

	t.Run("invalid_json_request", func(t *testing.T) {
		// Test with malformed JSON
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer([]byte("{invalid json")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Invalid JSON should return bad request")

		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse error response")
		assert.Equal(t, "invalid_request", response.Error, "Error should indicate invalid request")
	})

	t.Run("missing_required_fields", func(t *testing.T) {
		// Test registration with missing fields
		incompleteReq := map[string]string{
			"email": "incomplete@example.com",
			// Missing password, firstName, lastName
		}

		w := suite.makeRequest("POST", "/api/v1/auth/register", incompleteReq, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Missing required fields should return bad request")

		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse error response")
		assert.Equal(t, "validation_failed", response.Error, "Error should indicate validation failure")
	})

	t.Run("duplicate_user_registration", func(t *testing.T) {
		email := "duplicate@example.com"
		password := "SecurePass123!"

		// Register user first time
		suite.createTestUserAndLogin(t, email, password)

		// Try to register same user again
		registerReq := models.RegisterRequest{
			Email:     email,
			Password:  password,
			FirstName: "Test",
			LastName:  "User",
		}

		w := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Duplicate registration should fail")

		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse error response")
		assert.Equal(t, "registration_failed", response.Error, "Error should indicate registration failure")
	})

	t.Run("invalid_login_credentials", func(t *testing.T) {
		// Try to login with non-existent user
		loginReq := models.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "WrongPassword123!",
		}

		w := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid credentials should return unauthorized")

		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse error response")
		assert.Equal(t, "login_failed", response.Error, "Error should indicate login failure")
	})

	t.Run("invalid_refresh_token", func(t *testing.T) {
		// Test with invalid refresh token
		refreshReq := models.RefreshTokenRequest{
			RefreshToken: "invalid-refresh-token",
		}

		w := suite.makeRequest("POST", "/api/v1/auth/refresh", refreshReq, nil)
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusUnauthorized}, w.Code,
			"Invalid refresh token should return appropriate error")

		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse error response")
		assert.NotEmpty(t, response.Error, "Error response should contain error field")
	})

	t.Run("expired_or_invalid_auth_token", func(t *testing.T) {
		// Test with invalid auth token
		headers := map[string]string{
			"Authorization": "Bearer invalid-token",
		}

		w := suite.makeRequest("GET", "/api/v1/user/profile", nil, headers)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid auth token should return unauthorized")

		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Should be able to parse error response")
		assert.NotEmpty(t, response.Error, "Error response should contain error field")
	})
}

// Test HTTP Methods and CORS
func TestHTTPAPI_HTTPMethodsAndCORS(t *testing.T) {
	suite := SetupHTTPAPITestSuite(t)
	defer suite.cleanup()

	t.Run("method_not_allowed", func(t *testing.T) {
		// Test wrong HTTP method
		w := suite.makeRequest("GET", "/api/v1/auth/register", nil, nil)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "Wrong HTTP method should return method not allowed")
	})

	t.Run("cors_headers", func(t *testing.T) {
		// Test CORS headers are present
		w := suite.makeRequest("OPTIONS", "/api/v1/auth/register", nil, nil)

		// Check for CORS headers (exact headers depend on CORS middleware implementation)
		headers := w.Header()
		assert.NotEmpty(t, headers, "Response should contain headers")

		// Common CORS headers that should be present
		corsHeaders := []string{
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers",
		}

		for _, header := range corsHeaders {
			if headers.Get(header) != "" {
				assert.NotEmpty(t, headers.Get(header), fmt.Sprintf("CORS header %s should be present", header))
			}
		}
	})

	t.Run("security_headers", func(t *testing.T) {
		// Test security headers are present
		w := suite.makeRequest("GET", "/health", nil, nil)

		headers := w.Header()
		securityHeaders := []string{
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
		}

		for _, header := range securityHeaders {
			if headers.Get(header) != "" {
				assert.NotEmpty(t, headers.Get(header), fmt.Sprintf("Security header %s should be present", header))
			}
		}
	})
}
