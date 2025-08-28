package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS middleware for handling Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Allow specific origins or all origins in development
		allowedOrigins := map[string]bool{
			"http://localhost:3000":  true, // React dev server
			"http://localhost:3001":  true, // Alternative React port
			"http://localhost:8080":  true, // Vue dev server
			"http://127.0.0.1:3000":  true,
			"http://127.0.0.1:3001":  true,
			"http://127.0.0.1:8080":  true,
		}

		// In production, you should restrict this to your actual domain
		if allowedOrigins[origin] || origin == "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
