package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/systemsim/auth-service/internal/config"
)

// CORS middleware handles Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Allow specific origins (HTTP/2 prefers HTTPS but allow HTTP for development)
		allowedOrigins := []string{
			"http://localhost:3000",   // Development frontend
			"https://localhost:3000",  // Development frontend with HTTPS
			"http://localhost:5173",   // Vite dev server
			"https://localhost:5173",  // Vite dev server with HTTPS
			"http://localhost:5174",   // Alternative Vite port
			"https://localhost:5174",  // Alternative Vite port with HTTPS
			"https://systemsim.app",   // Production domain
		}
		
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
		
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityHeaders middleware adds security headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';")
		
		// HSTS (only in production with HTTPS)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		c.Next()
	}
}

// RateLimiter middleware implements rate limiting using Redis
func RateLimiter(cfg config.RateLimitConfig) gin.HandlerFunc {
	// Create Redis client for rate limiting
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	return func(c *gin.Context) {
		// Get client identifier (IP address or user ID if authenticated)
		clientID := getClientID(c)
		
		// Create rate limit key
		key := fmt.Sprintf("rate_limit:%s", clientID)
		
		ctx := context.Background()
		
		// Get current count
		current, err := redisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			// If Redis is down, allow the request but log the error
			c.Next()
			return
		}
		
		// Check if limit exceeded
		if current >= cfg.RequestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
				"retry_after": 60,
			})
			c.Abort()
			return
		}
		
		// Increment counter
		pipe := redisClient.Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Minute)
		_, err = pipe.Exec(ctx)
		
		if err != nil {
			// If Redis operation fails, allow the request
			c.Next()
			return
		}
		
		// Add rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(cfg.RequestsPerMinute-current-1))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
		
		c.Next()
	}
}

// LoginRateLimiter implements stricter rate limiting for login attempts
func LoginRateLimiter(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := getClientID(c)
		key := fmt.Sprintf("login_attempts:%s", clientID)
		
		ctx := context.Background()
		
		// Get current attempts
		attempts, err := redisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			c.Next()
			return
		}
		
		// Allow max 5 login attempts per 15 minutes
		if attempts >= 5 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "login_rate_limit_exceeded",
				"message": "Too many login attempts. Please try again in 15 minutes.",
				"retry_after": 900, // 15 minutes
			})
			c.Abort()
			return
		}
		
		c.Next()
		
		// If login failed (check response status), increment counter
		if c.Writer.Status() == http.StatusUnauthorized {
			pipe := redisClient.Pipeline()
			pipe.Incr(ctx, key)
			pipe.Expire(ctx, key, 15*time.Minute)
			pipe.Exec(ctx)
		}
	}
}

// getClientID returns a unique identifier for the client
func getClientID(c *gin.Context) string {
	// Try to get user ID if authenticated
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%v", userID)
	}
	
	// Fall back to IP address
	clientIP := c.ClientIP()
	
	// Consider X-Forwarded-For header for load balancers
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}
	
	return fmt.Sprintf("ip:%s", clientIP)
}

// RequestLogger middleware logs HTTP requests
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}
