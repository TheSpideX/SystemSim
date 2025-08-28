package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/systemsim/auth-service/internal/security"
)

// AuthRequired middleware validates JWT tokens
func AuthRequired(jwtSecret string) gin.HandlerFunc {
	jwtManager := security.NewJWTManager(jwtSecret, 0, 0, "systemsim-auth")
	
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		token, err := security.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Ensure it's an access token
		if claims.TokenType != "access" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid token type",
			})
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("session_id", claims.SessionID)

		c.Next()
	}
}

// AdminRequired middleware ensures the user is an admin
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Admin access required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware validates JWT tokens but doesn't require them
func OptionalAuth(jwtSecret string) gin.HandlerFunc {
	jwtManager := security.NewJWTManager(jwtSecret, 0, 0, "systemsim-auth")
	
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token, err := security.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.Next()
			return
		}

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		if claims.TokenType == "access" {
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("is_admin", claims.IsAdmin)
			c.Set("session_id", claims.SessionID)
		}

		c.Next()
	}
}
