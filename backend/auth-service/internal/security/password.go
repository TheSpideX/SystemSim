package security

import (
	"fmt"
	"regexp"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum required password length
	MinPasswordLength = 8
	// MaxPasswordLength is the maximum allowed password length
	MaxPasswordLength = 128
	// BcryptCost is the cost factor for bcrypt hashing
	BcryptCost = 12
)

// PasswordValidator handles password validation and security
type PasswordValidator struct {
	minLength      int
	maxLength      int
	requireUpper   bool
	requireLower   bool
	requireDigit   bool
	requireSpecial bool
}

// NewPasswordValidator creates a new password validator with default rules
func NewPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		minLength:      MinPasswordLength,
		maxLength:      MaxPasswordLength,
		requireUpper:   true,
		requireLower:   true,
		requireDigit:   true,
		requireSpecial: true,
	}
}

// ValidatePassword validates a password against security rules
func (pv *PasswordValidator) ValidatePassword(password string) error {
	if len(password) < pv.minLength {
		return fmt.Errorf("password must be at least %d characters long", pv.minLength)
	}

	if len(password) > pv.maxLength {
		return fmt.Errorf("password must be no more than %d characters long", pv.maxLength)
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasDigit   = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if pv.requireUpper && !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if pv.requireLower && !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if pv.requireDigit && !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}

	if pv.requireSpecial && !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	// Check for common weak patterns
	if err := pv.checkWeakPatterns(password); err != nil {
		return err
	}

	return nil
}

// checkWeakPatterns checks for common weak password patterns
func (pv *PasswordValidator) checkWeakPatterns(password string) error {
	// Check for common weak passwords
	weakPasswords := []string{
		"password", "123456", "qwerty", "abc123", "password123",
		"admin", "letmein", "welcome", "monkey", "dragon",
	}

	for _, weak := range weakPasswords {
		if matched, _ := regexp.MatchString("(?i)"+weak, password); matched {
			return fmt.Errorf("password contains common weak patterns")
		}
	}

	// Check for repeated characters (more than 3 in a row)
	if matched, _ := regexp.MatchString(`(.)\1{3,}`, password); matched {
		return fmt.Errorf("password cannot contain more than 3 repeated characters")
	}

	// Check for sequential characters
	sequential := []string{
		"123", "234", "345", "456", "567", "678", "789",
		"abc", "bcd", "cde", "def", "efg", "fgh", "ghi",
		"qwe", "wer", "ert", "rty", "tyu", "yui", "uio",
	}

	for _, seq := range sequential {
		if matched, _ := regexp.MatchString("(?i)"+seq, password); matched {
			return fmt.Errorf("password cannot contain sequential characters")
		}
	}

	return nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	// Validate password before hashing
	validator := NewPasswordValidator()
	if err := validator.ValidatePassword(password); err != nil {
		return "", fmt.Errorf("password validation failed: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return fmt.Errorf("invalid password")
	}
	return nil
}

// IsPasswordCompromised checks if a password has been compromised (placeholder)
// In production, this could integrate with HaveIBeenPwned API
func IsPasswordCompromised(password string) (bool, error) {
	// Placeholder implementation
	// In production, you would check against a compromised password database
	// or use an API like HaveIBeenPwned
	
	// For now, just check against a small list of known compromised passwords
	compromised := []string{
		"password", "123456", "password123", "admin", "qwerty123",
	}

	for _, comp := range compromised {
		if password == comp {
			return true, nil
		}
	}

	return false, nil
}

// GeneratePasswordStrengthScore calculates a password strength score (0-100)
func GeneratePasswordStrengthScore(password string) int {
	score := 0
	
	// Length score (up to 25 points)
	if len(password) >= 8 {
		score += 10
	}
	if len(password) >= 12 {
		score += 10
	}
	if len(password) >= 16 {
		score += 5
	}

	// Character variety (up to 40 points)
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if hasUpper {
		score += 10
	}
	if hasLower {
		score += 10
	}
	if hasDigit {
		score += 10
	}
	if hasSpecial {
		score += 10
	}

	// Uniqueness (up to 35 points)
	unique := make(map[rune]bool)
	for _, char := range password {
		unique[char] = true
	}
	
	uniqueRatio := float64(len(unique)) / float64(len(password))
	score += int(uniqueRatio * 35)

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}
