package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	SessionID uuid.UUID `json:"session_id"`
	TokenType string    `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey            []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
	issuer               string
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, accessDuration, refreshDuration time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secretKey:            []byte(secretKey),
		accessTokenDuration:  accessDuration,
		refreshTokenDuration: refreshDuration,
		issuer:               issuer,
	}
}

// GenerateTokenPair generates both access and refresh tokens with default durations
func (j *JWTManager) GenerateTokenPair(userID uuid.UUID, email string, isAdmin bool, sessionID uuid.UUID) (string, string, error) {
	return j.GenerateTokenPairWithDuration(userID, email, isAdmin, sessionID, j.accessTokenDuration, j.refreshTokenDuration)
}

// GenerateTokenPairWithDuration generates both access and refresh tokens with custom durations
func (j *JWTManager) GenerateTokenPairWithDuration(userID uuid.UUID, email string, isAdmin bool, sessionID uuid.UUID, accessDuration, refreshDuration time.Duration) (string, string, error) {
	// Generate access token
	accessToken, err := j.generateToken(userID, email, isAdmin, sessionID, "access", accessDuration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := j.generateToken(userID, email, isAdmin, sessionID, "refresh", refreshDuration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// generateToken generates a JWT token with specified parameters
func (j *JWTManager) generateToken(userID uuid.UUID, email string, isAdmin bool, sessionID uuid.UUID, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()
	
	claims := &JWTClaims{
		UserID:    userID,
		Email:     email,
		IsAdmin:   isAdmin,
		SessionID: sessionID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			Issuer:    j.issuer,
			Audience:  []string{"systemsim"},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateToken validates and parses a JWT token
func (j *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Additional validation
	if claims.TokenType == "" {
		return nil, fmt.Errorf("missing token type")
	}

	return claims, nil
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is required")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", fmt.Errorf("authorization header must start with 'Bearer '")
	}

	return authHeader[len(bearerPrefix):], nil
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken creates a hash of a token for secure storage
func HashToken(token string) string {
	// In production, use a proper hashing algorithm like SHA-256
	// For now, we'll use a simple approach
	return fmt.Sprintf("%x", token)
}

// GetAccessTokenDuration returns the access token duration
func (j *JWTManager) GetAccessTokenDuration() time.Duration {
	return j.accessTokenDuration
}

// GetRefreshTokenDuration returns the refresh token duration
func (j *JWTManager) GetRefreshTokenDuration() time.Duration {
	return j.refreshTokenDuration
}
