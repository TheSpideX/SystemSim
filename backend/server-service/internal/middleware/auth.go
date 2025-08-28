package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"server-service/internal/services"
)

// AuthMiddleware provides JWT token validation middleware
type AuthMiddleware struct {
	authService *services.AuthService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *services.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth middleware that requires valid JWT token
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			am.sendUnauthorized(w, "missing_authorization_header", "Authorization header is required")
			return
		}

		token := services.ExtractTokenFromHeader(authHeader)
		if token == "" {
			am.sendUnauthorized(w, "invalid_authorization_format", "Authorization header must be in format 'Bearer <token>'")
			return
		}

		// Validate token with auth service
		requestID := services.GenerateRequestID()
		resp, err := am.authService.ValidateToken(token, "api-gateway", requestID)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			am.sendUnauthorized(w, "token_validation_failed", "Failed to validate token")
			return
		}

		if !resp.Valid {
			am.sendUnauthorized(w, "invalid_token", "Token is invalid or expired")
			return
		}

		// Add user context to request
		ctx := context.WithValue(r.Context(), "user_id", resp.UserId)
		ctx = context.WithValue(ctx, "request_id", requestID)

		// Continue to next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission middleware that requires specific permission
func (am *AuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// First validate token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				am.sendUnauthorized(w, "missing_authorization_header", "Authorization header is required")
				return
			}

			token := services.ExtractTokenFromHeader(authHeader)
			if token == "" {
				am.sendUnauthorized(w, "invalid_authorization_format", "Authorization header must be in format 'Bearer <token>'")
				return
			}

			// Validate token
			requestID := services.GenerateRequestID()
			tokenResp, err := am.authService.ValidateToken(token, "api-gateway", requestID)
			if err != nil || !tokenResp.Valid {
				am.sendUnauthorized(w, "invalid_token", "Token validation failed")
				return
			}

			// Check permission
			permResp, err := am.authService.CheckPermission(tokenResp.UserId, permission, "", "api-gateway", requestID)
			if err != nil {
				log.Printf("Permission check failed: %v", err)
				am.sendForbidden(w, "permission_check_failed", "Failed to check permissions")
				return
			}

			if !permResp.Allowed {
				am.sendForbidden(w, "insufficient_permissions", "User does not have required permission: "+permission)
				return
			}

			// Add user context to request
			ctx := context.WithValue(r.Context(), "user_id", tokenResp.UserId)
			ctx = context.WithValue(ctx, "request_id", requestID)

			// Continue to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth middleware that validates token if present but doesn't require it
func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No token provided, continue without user context
			next.ServeHTTP(w, r)
			return
		}

		token := services.ExtractTokenFromHeader(authHeader)
		if token == "" {
			// Invalid format, continue without user context
			next.ServeHTTP(w, r)
			return
		}

		// Try to validate token
		requestID := services.GenerateRequestID()
		resp, err := am.authService.ValidateToken(token, "api-gateway", requestID)
		if err != nil || !resp.Valid {
			// Invalid token, continue without user context
			next.ServeHTTP(w, r)
			return
		}

		// Add user context to request
		ctx := context.WithValue(r.Context(), "user_id", resp.UserId)
		ctx = context.WithValue(ctx, "request_id", requestID)

		// Continue to next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CORS middleware for handling cross-origin requests
func (am *AuthMiddleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimit middleware for basic rate limiting (placeholder)
func (am *AuthMiddleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement rate limiting logic
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// Logging middleware for request logging
func (am *AuthMiddleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("Completed %s %s with status %d in %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Helper methods for sending error responses
func (am *AuthMiddleware) sendUnauthorized(w http.ResponseWriter, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + errorCode + `","message":"` + message + `"}`))
}

func (am *AuthMiddleware) sendForbidden(w http.ResponseWriter, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error":"` + errorCode + `","message":"` + message + `"}`))
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) string {
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	return ""
}

// GetRequestID extracts request ID from request context
func GetRequestID(r *http.Request) string {
	if requestID, ok := r.Context().Value("request_id").(string); ok {
		return requestID
	}
	return ""
}
