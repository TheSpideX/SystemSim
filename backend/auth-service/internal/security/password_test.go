package security

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_strong_password",
			password:    "V@lidP4ss!",
			expectError: false,
		},
		{
			name:        "valid_complex_password",
			password:    "MyStr0ng!P@ssw0rd",
			expectError: false,
		},
		{
			name:        "short_password",
			password:    "Short1!",
			expectError: true,
			errorMsg:    "password must be at least 8 characters",
		},
		{
			name:        "empty_password",
			password:    "",
			expectError: true,
			errorMsg:    "password must be at least 8 characters",
		},
		{
			name:        "too_long_password",
			password:    strings.Repeat("A1a!", 50), // 200 characters
			expectError: true,
			errorMsg:    "password must be no more than 128 characters",
		},
		{
			name:        "missing_uppercase",
			password:    "lowercase123!",
			expectError: true,
			errorMsg:    "password must contain at least one uppercase letter",
		},
		{
			name:        "missing_lowercase",
			password:    "UPPERCASE123!",
			expectError: true,
			errorMsg:    "password must contain at least one lowercase letter",
		},
		{
			name:        "missing_digit",
			password:    "NoDigits!",
			expectError: true,
			errorMsg:    "password must contain at least one digit",
		},
		{
			name:        "missing_special_char",
			password:    "NoSpecial123",
			expectError: true,
			errorMsg:    "password must contain at least one special character",
		},
		{
			name:        "weak_password_pattern",
			password:    "Password123!",
			expectError: true,
			errorMsg:    "password contains common weak patterns",
		},
		{
			name:        "sequential_characters",
			password:    "Abcd1234!",
			expectError: true,
			errorMsg:    "sequential characters",
		},
		// Skip repeated characters test as it's difficult to create a password
		// that triggers only the repeated characters validation without triggering
		// other validations first
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			hashedPassword, err := HashPassword(tt.password)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Empty(t, hashedPassword)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hashedPassword)

				// Verify it's a valid bcrypt hash
				assert.True(t, strings.HasPrefix(hashedPassword, "$2a$") ||
					strings.HasPrefix(hashedPassword, "$2b$") ||
					strings.HasPrefix(hashedPassword, "$2y$"))

				// Verify hash is different from original password
				assert.NotEqual(t, tt.password, hashedPassword)

				// Verify we can verify the password with the hash
				err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(tt.password))
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordValidator(t *testing.T) {
	validator := NewPasswordValidator()

	tests := []struct {
		name        string
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_strong_password",
			password:    "V@lidP4ss!",
			expectError: false,
		},
		{
			name:        "too_short",
			password:    "Short1!",
			expectError: true,
			errorMsg:    "password must be at least 8 characters long",
		},
		{
			name:        "too_long",
			password:    strings.Repeat("A1a!", 50), // 200 characters
			expectError: true,
			errorMsg:    "password must be no more than 128 characters long",
		},
		{
			name:        "missing_uppercase",
			password:    "lowercase123!",
			expectError: true,
			errorMsg:    "password must contain at least one uppercase letter",
		},
		{
			name:        "missing_lowercase",
			password:    "UPPERCASE123!",
			expectError: true,
			errorMsg:    "password must contain at least one lowercase letter",
		},
		{
			name:        "missing_digit",
			password:    "NoDigits!",
			expectError: true,
			errorMsg:    "password must contain at least one digit",
		},
		{
			name:        "missing_special_char",
			password:    "NoSpecial123",
			expectError: true,
			errorMsg:    "password must contain at least one special character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			err := validator.ValidatePassword(tt.password)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	// Setup - create a valid hash
	validPassword := "V@lidP4ss!"
	validHash, err := HashPassword(validPassword)
	require.NoError(t, err)

	tests := []struct {
		name        string
		password    string
		hash        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "correct_password",
			password:    validPassword,
			hash:        validHash,
			expectError: false,
		},
		{
			name:        "wrong_password",
			password:    "Wr0ngP4ss!",
			hash:        validHash,
			expectError: true,
			errorMsg:    "invalid password",
		},
		{
			name:        "empty_password",
			password:    "",
			hash:        validHash,
			expectError: true,
			errorMsg:    "invalid password",
		},
		{
			name:        "empty_hash",
			password:    validPassword,
			hash:        "",
			expectError: true,
			errorMsg:    "invalid password",
		},
		{
			name:        "invalid_hash",
			password:    validPassword,
			hash:        "invalid-hash",
			expectError: true,
			errorMsg:    "invalid password",
		},
		{
			name:        "case_sensitive",
			password:    "validpass123!",
			hash:        validHash,
			expectError: true,
			errorMsg:    "invalid password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			err := VerifyPassword(tt.password, tt.hash)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordHashing_Consistency(t *testing.T) {
	password := "C0ns!stency#T3st"

	// Hash the same password multiple times
	hash1, err := HashPassword(password)
	require.NoError(t, err)

	hash2, err := HashPassword(password)
	require.NoError(t, err)

	hash3, err := HashPassword(password)
	require.NoError(t, err)

	// Hashes should be different (due to salt)
	assert.NotEqual(t, hash1, hash2)
	assert.NotEqual(t, hash2, hash3)
	assert.NotEqual(t, hash1, hash3)

	// But all should verify correctly
	assert.NoError(t, VerifyPassword(password, hash1))
	assert.NoError(t, VerifyPassword(password, hash2))
	assert.NoError(t, VerifyPassword(password, hash3))
}

func TestPasswordHashing_Performance(t *testing.T) {
	password := "P3rf0rm@nc3#T3st"

	// Test that hashing doesn't take too long (should be under 1 second)
	start := time.Now()
	_, err := HashPassword(password)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, duration, time.Second, "Password hashing took too long")
}

func TestPasswordValidation(t *testing.T) {
	validator := NewPasswordValidator()

	tests := []struct {
		name        string
		password    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_minimum_length",
			password:    "MinLen1!", // Exactly 8 characters with all requirements
			expectError: false,
		},
		{
			name:        "too_short",
			password:    "Short1!", // 7 characters
			expectError: true,
			errorMsg:    "password must be at least 8 characters",
		},
		{
			name:        "empty_password",
			password:    "",
			expectError: true,
			errorMsg:    "password must be at least 8 characters",
		},
		{
			name:        "whitespace_only",
			password:    "        ", // 8 spaces
			expectError: true,
			errorMsg:    "password must contain at least one uppercase letter",
		},
		{
			name:        "very_long_password",
			password:    strings.Repeat("A1a!", 50), // 200 characters
			expectError: true,
			errorMsg:    "password must be no more than 128 characters",
		},
		{
			name:        "exactly_max_length",
			password:    strings.Repeat("A1a!", 32), // Exactly 128 characters
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			err := validator.ValidatePassword(tt.password)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBcryptCostFactor(t *testing.T) {
	password := "C0st#F4ct0r!T3st"

	// Hash password
	hashedPassword, err := HashPassword(password)
	require.NoError(t, err)

	// Extract cost factor from hash
	cost, err := bcrypt.Cost([]byte(hashedPassword))
	require.NoError(t, err)

	// Verify cost factor is reasonable (should be at least 10 for security)
	assert.GreaterOrEqual(t, cost, 10, "Bcrypt cost factor should be at least 10")
	assert.LessOrEqual(t, cost, 15, "Bcrypt cost factor should not be too high for performance")

	// Verify it matches our constant
	assert.Equal(t, BcryptCost, cost, "Cost factor should match our constant")
}

func TestPasswordSecurity(t *testing.T) {
	// Test that weak passwords are properly rejected
	weakPasswords := []struct {
		password string
		reason   string
	}{
		{"Password123!", "contains common weak patterns"},
		{"Admin123!", "contains common weak patterns"},
		{"Qwerty123!", "contains common weak patterns"},
		{"Letmein123!", "contains common weak patterns"},
	}

	for _, test := range weakPasswords {
		t.Run("weak_password_"+test.password, func(t *testing.T) {
			// Weak passwords should be rejected
			hashedPassword, err := HashPassword(test.password)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.reason)
			assert.Empty(t, hashedPassword)
		})
	}
}

func TestIsPasswordCompromised(t *testing.T) {
	tests := []struct {
		name         string
		password     string
		compromised  bool
	}{
		{
			name:        "compromised_password",
			password:    "password",
			compromised: true,
		},
		{
			name:        "compromised_123456",
			password:    "123456",
			compromised: true,
		},
		{
			name:        "safe_password",
			password:    "S@f3#P4ssw0rd!",
			compromised: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			compromised, err := IsPasswordCompromised(tt.password)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.compromised, compromised)
		})
	}
}

func TestGeneratePasswordStrengthScore(t *testing.T) {
	tests := []struct {
		name     string
		password string
		minScore int
		maxScore int
	}{
		{
			name:     "very_strong_password",
			password: "V3ry#Str0ng!P@ssw0rd",
			minScore: 80,
			maxScore: 100,
		},
		{
			name:     "medium_password",
			password: "M3d!um#P4ss",
			minScore: 50,
			maxScore: 90,
		},
		{
			name:     "weak_password",
			password: "weak",
			minScore: 0,
			maxScore: 50,
		},
		{
			name:     "empty_password",
			password: "",
			minScore: 0,
			maxScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			score := GeneratePasswordStrengthScore(tt.password)

			// Assert
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
			assert.GreaterOrEqual(t, score, 0)
			assert.LessOrEqual(t, score, 100)
		})
	}
}
