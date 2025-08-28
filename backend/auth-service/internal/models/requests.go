package models

// Authentication request/response models

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email,max=255"`
	Password  string `json:"password" validate:"required,min=8,max=128"`
	FirstName string `json:"first_name" validate:"max=100"`
	LastName  string `json:"last_name" validate:"max=100"`
	Company   string `json:"company" validate:"max=200"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Remember bool   `json:"remember"` // For extended session duration
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ForgotPasswordRequest represents a forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

// VerifyEmailRequest represents an email verification request
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FirstName             string  `json:"first_name" validate:"max=100"`
	LastName              string  `json:"last_name" validate:"max=100"`
	Company               string  `json:"company" validate:"max=200"`
	SimulationPreferences JSONMap `json:"simulation_preferences"`
	UIPreferences         JSONMap `json:"ui_preferences"`
}

// AuthResponse represents authentication response with tokens
type AuthResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    int64         `json:"expires_in"` // seconds
	RememberMe   bool          `json:"remember_me"` // indicates if this is a remember me session
	SessionID    string        `json:"session_id"`  // session identifier for event tracking
}

// TokenResponse represents a token refresh response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// PasswordResetResponse represents password reset response
type PasswordResetResponse struct {
	ResetToken string `json:"reset_token,omitempty"`
	ExpiresIn  int64  `json:"expires_in,omitempty"`
	Message    string `json:"message"`
}

// EmailVerificationResponse represents email verification response
type EmailVerificationResponse struct {
	VerificationToken string `json:"verification_token,omitempty"`
	ExpiresIn         int64  `json:"expires_in,omitempty"`
	Message           string `json:"message"`
}

// ResendVerificationRequest represents resend verification request
type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
