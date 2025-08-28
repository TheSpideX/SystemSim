package gateway

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"server-service/internal/circuit"
	"server-service/internal/config"
	"server-service/internal/errors"
	"server-service/internal/grpc_clients"
	"server-service/internal/http2"
	"server-service/internal/middleware"
	"server-service/internal/redis_client"
	"server-service/internal/router"
	"server-service/internal/services"
	websocketpkg "server-service/internal/websocket"
)

// Gateway represents the API Gateway
type Gateway struct {
	config      *Config
	server      *http2.Server
	grpcClients *grpc_clients.ClientPool
	wsHub       *websocketpkg.Hub
	redisClient *redis_client.Client

	// Service clients
	authService *services.AuthService

	// Middleware
	authMiddleware *middleware.AuthMiddleware

	// Routing
	serviceRouter *router.ServiceRouter

	// Error handling and resilience
	errorHandler   *errors.ErrorHandler
	circuitManager *circuit.Manager

	// Performance monitoring
	requestsProcessed int64
	requestsPerSecond int64
	avgResponseTime   time.Duration
}

// Config holds gateway configuration
type Config struct {
	ServerConfig *config.ServerConfig
	GRPCClients  *grpc_clients.ClientPool
	WebSocketHub *websocketpkg.Hub
	RedisClient  *redis_client.Client
}

// New creates a new API Gateway with HTTP/2 support
func New(cfg *Config) *Gateway {
	gateway := &Gateway{
		config:      cfg,
		grpcClients: cfg.GRPCClients,
		wsHub:       cfg.WebSocketHub,
		redisClient: cfg.RedisClient,
	}

	// Initialize service clients
	if cfg.GRPCClients != nil {
		gateway.authService = services.NewAuthService(cfg.GRPCClients.GetAuthPool())
	} else {
		// Use mock auth service when gRPC clients are not available
		gateway.authService = services.NewAuthService(nil)
	}

	// Always initialize auth middleware
	gateway.authMiddleware = middleware.NewAuthMiddleware(gateway.authService)

	// Initialize routing
	gateway.serviceRouter = router.NewServiceRouter()

	// Initialize error handling and resilience
	gateway.errorHandler = errors.NewErrorHandler()
	gateway.circuitManager = circuit.NewManager()

	// Create HTTP/2 server with request handler
	server, err := http2.NewServer(cfg.ServerConfig, gateway)
	if err != nil {
		log.Fatalf("Failed to create HTTP/2 server: %v", err)
	}
	gateway.server = server

	// Apply performance optimizations
	http2.OptimizeForThroughput(server)
	log.Println("API Gateway initialized with HTTP/2 + TLS")

	return gateway
}

// ServeHTTP implements the http.Handler interface
func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Apply middleware chain
	handler := gw.getRouteHandler()

	// Apply error handling middleware
	handler = gw.errorHandler.Recovery(handler)
	handler = gw.errorHandler.Timeout(30 * time.Second)(handler)

	// Apply CORS middleware
	if gw.authMiddleware != nil {
		handler = gw.authMiddleware.CORS(handler)
		handler = gw.authMiddleware.Logging(handler)
	}

	handler.ServeHTTP(w, r)
}

// getRouteHandler returns the main routing handler
func (gw *Gateway) getRouteHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		defer func() {
			// Record response time
			responseTime := time.Since(startTime)
			gw.avgResponseTime = (gw.avgResponseTime + responseTime) / 2
			gw.requestsProcessed++
		}()

		path := r.URL.Path
		method := r.Method

		// Route requests based on path using service router
		switch {
		case path == "/ws":
			gw.handleWebSocketGeneric(w, r)
		case path == "/ws/health":
			gw.handleWebSocketHealth(w, r)
		case strings.HasPrefix(path, "/ws/"):
			gw.handleWebSocketSpecific(w, r)
		case path == "/health":
			gw.handleHealthCheck(w, r)
		case path == "/health/auth":
			gw.handleAuthHealthCheck(w, r)
		case path == "/health/project":
			gw.handleProjectHealthCheck(w, r)
		case path == "/health/simulation":
			gw.handleSimulationHealthCheck(w, r)
		case path == "/grpc/stats":
			gw.handleGRPCStats(w, r)
		case path == "/metrics":
			gw.handleMetrics(w, r)
		case strings.HasPrefix(path, "/api/"):
			gw.handleAPIRequest(w, r, method, path)
		default:
			gw.handleNotFound(w, r)
		}
	})
}

// handleAPIRequest routes API requests to appropriate services
func (gw *Gateway) handleAPIRequest(w http.ResponseWriter, r *http.Request, method, path string) {
	// Determine which service should handle this request
	service := gw.serviceRouter.GetServiceForPath(path)

	// Check if route requires authentication
	requiresAuth := gw.serviceRouter.IsProtectedRoute(path)

	// Create handler based on service
	var handler http.HandlerFunc
	switch service {
	case "auth":
		handler = func(w http.ResponseWriter, r *http.Request) {
			gw.handleAuthRequest(w, r, method, path)
		}
	case "project":
		handler = func(w http.ResponseWriter, r *http.Request) {
			gw.handleProjectRequest(w, r, method, path)
		}
	case "simulation":
		handler = func(w http.ResponseWriter, r *http.Request) {
			gw.handleSimulationRequest(w, r, method, path)
		}
	default:
		gw.handleNotFound(w, r)
		return
	}

	// Apply authentication middleware if required
	if requiresAuth && gw.authMiddleware != nil {
		gw.authMiddleware.RequireAuth(handler).ServeHTTP(w, r)
	} else {
		handler(w, r)
	}
}

// Start starts the API Gateway server
func (gw *Gateway) Start() error {
	log.Printf("Starting API Gateway on port %s with HTTP/2", gw.config.ServerConfig.Port)

	// Subscribe to Redis events for real-time broadcasting (if Redis is available)
	if gw.redisClient != nil {
		if err := gw.redisClient.SubscribeToEvents(); err != nil {
			log.Printf("Warning: Failed to subscribe to Redis events: %v", err)
			log.Println("Continuing without real-time events")
		}
	}

	// Start performance monitoring
	go gw.monitorPerformance()

	// Start HTTP/2 server
	return gw.server.Start()
}

// StartWebSocketServer starts a dedicated HTTP/1.1 server for WebSocket connections
func (gw *Gateway) StartWebSocketServer(port int) error {
	mux := http.NewServeMux()

	// WebSocket health endpoint
	mux.HandleFunc("/ws/health", gw.handleWebSocketHealthDedicated)

	// Generic WebSocket endpoint
	mux.HandleFunc("/ws", gw.handleWebSocketGenericDedicated)

	// Specific WebSocket endpoints
	mux.HandleFunc("/ws/", gw.handleWebSocketSpecificDedicated)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	log.Printf("WebSocket server starting on port %d with TLS", port)
	return server.ListenAndServeTLS("certs/server.crt", "certs/server.key")
}

// Shutdown gracefully shuts down the gateway
func (gw *Gateway) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API Gateway...")

	// Close WebSocket hub
	if gw.wsHub != nil {
		gw.wsHub.Close()
	}

	// Close gRPC clients
	if gw.grpcClients != nil {
		if err := gw.grpcClients.Close(); err != nil {
			log.Printf("Error closing gRPC clients: %v", err)
		}
	}

	// Close Redis client
	if gw.redisClient != nil {
		if err := gw.redisClient.Close(); err != nil {
			log.Printf("Error closing Redis client: %v", err)
		}
	}

	// Shutdown HTTP server
	if err := gw.server.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v", err)
		return err
	}

	log.Println("API Gateway stopped")
	return nil
}

// setCORSHeaders sets CORS headers for cross-origin requests
func (gw *Gateway) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// handleWebSocketGeneric handles generic WebSocket upgrade requests (/ws)
func (gw *Gateway) handleWebSocketGeneric(w http.ResponseWriter, r *http.Request) {
	gw.upgradeWebSocket(w, r, "general", "")
}

// handleWebSocketSpecific handles specific WebSocket endpoints
func (gw *Gateway) handleWebSocketSpecific(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Parse WebSocket endpoint type and resource ID
	wsType, resourceID := gw.parseWebSocketEndpoint(path)

	log.Printf("WebSocket specific connection: type=%s, resource=%s", wsType, resourceID)

	// Validate resource access based on type
	userID, err := gw.authenticateWebSocketUser(r)
	if err != nil {
		log.Printf("WebSocket authentication failed: %v", err)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Validate resource access
	if !gw.validateResourceAccess(userID, wsType, resourceID) {
		log.Printf("WebSocket access denied: user=%s, type=%s, resource=%s", userID, wsType, resourceID)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Upgrade WebSocket with specific context
	gw.upgradeWebSocket(w, r, wsType, resourceID)
}

// upgradeWebSocket performs the actual WebSocket upgrade
func (gw *Gateway) upgradeWebSocket(w http.ResponseWriter, r *http.Request, wsType, resourceID string) {
	// Authenticate user from JWT token
	userID, err := gw.authenticateWebSocketUser(r)
	if err != nil {
		log.Printf("WebSocket authentication failed: %v", err)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check if WebSocket hub is available
	if gw.wsHub == nil {
		log.Printf("WebSocket hub not available")
		http.Error(w, "WebSocket service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Create WebSocket context
	wsContext := &websocketpkg.WebSocketContext{
		UserID:     userID,
		Type:       wsType,
		ResourceID: resourceID,
		Path:       r.URL.Path,
	}

	// Use the existing WebSocket upgrade function with context
	err = websocketpkg.UpgradeStandardHTTPWithContext(w, r, wsContext, gw.wsHub)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}

	log.Printf("WebSocket connection established: user=%s, type=%s, resource=%s", userID, wsType, resourceID)
}

// handleWebSocketHealth handles WebSocket connections for health updates
func (gw *Gateway) handleWebSocketHealth(w http.ResponseWriter, r *http.Request) {
	// Check if WebSocket hub is available
	if gw.wsHub == nil {
		log.Printf("WebSocket hub not available")
		http.Error(w, "WebSocket service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Get health manager from hub
	healthManager := gw.wsHub.GetHealthManager()
	if healthManager == nil {
		log.Printf("Health manager not available")
		http.Error(w, "Health monitoring service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Create health WebSocket handler
	healthHandler := websocketpkg.NewHealthWebSocketHandler(healthManager)

	// Handle the WebSocket connection
	healthHandler.HandleWebSocket(w, r)
}

// handleWebSocketHealthDedicated handles WebSocket health connections on dedicated server
func (gw *Gateway) handleWebSocketHealthDedicated(w http.ResponseWriter, r *http.Request) {
	log.Printf("WebSocket health connection attempt from %s", r.RemoteAddr)

	// Check if WebSocket hub is available
	if gw.wsHub == nil {
		log.Printf("WebSocket hub not available")
		http.Error(w, "WebSocket service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Get health manager from hub
	healthManager := gw.wsHub.GetHealthManager()
	if healthManager == nil {
		log.Printf("Health manager not available")
		http.Error(w, "Health monitoring service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Create health WebSocket handler
	healthHandler := websocketpkg.NewHealthWebSocketHandler(healthManager)

	// Handle the WebSocket connection
	healthHandler.HandleWebSocket(w, r)
}

// handleWebSocketGenericDedicated handles generic WebSocket connections on dedicated server
func (gw *Gateway) handleWebSocketGenericDedicated(w http.ResponseWriter, r *http.Request) {
	log.Printf("Generic WebSocket connection attempt from %s", r.RemoteAddr)
	gw.upgradeWebSocketDedicated(w, r, "general", "")
}

// handleWebSocketSpecificDedicated handles specific WebSocket connections on dedicated server
func (gw *Gateway) handleWebSocketSpecificDedicated(w http.ResponseWriter, r *http.Request) {
	log.Printf("Specific WebSocket connection attempt from %s to %s", r.RemoteAddr, r.URL.Path)

	path := r.URL.Path
	wsType, resourceID := gw.parseWebSocketEndpoint(path)

	// Authenticate user
	userID, err := gw.authenticateWebSocketUser(r)
	if err != nil {
		log.Printf("WebSocket authentication failed: %v", err)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Validate resource access
	if !gw.validateResourceAccess(userID, wsType, resourceID) {
		log.Printf("WebSocket access denied: user=%s, type=%s, resource=%s", userID, wsType, resourceID)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Upgrade WebSocket with specific context
	gw.upgradeWebSocketDedicated(w, r, wsType, resourceID)
}

// upgradeWebSocketDedicated performs WebSocket upgrade on dedicated HTTP/1.1 server
func (gw *Gateway) upgradeWebSocketDedicated(w http.ResponseWriter, r *http.Request, wsType, resourceID string) {
	// Authenticate user from JWT token
	userID, err := gw.authenticateWebSocketUser(r)
	if err != nil {
		log.Printf("WebSocket authentication failed: %v", err)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check if WebSocket hub is available
	if gw.wsHub == nil {
		log.Printf("WebSocket hub not available")
		http.Error(w, "WebSocket service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Create WebSocket context
	wsContext := &websocketpkg.WebSocketContext{
		UserID:     userID,
		Type:       wsType,
		ResourceID: resourceID,
		Path:       r.URL.Path,
	}

	// Use the existing WebSocket upgrade function with context
	err = websocketpkg.UpgradeStandardHTTPWithContext(w, r, wsContext, gw.wsHub)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}

	log.Printf("WebSocket connection established: user=%s, type=%s, resource=%s", userID, wsType, resourceID)
}

// parseWebSocketEndpoint parses WebSocket endpoint to extract type and resource ID
func (gw *Gateway) parseWebSocketEndpoint(path string) (wsType, resourceID string) {
	// Remove /ws/ prefix
	path = strings.TrimPrefix(path, "/ws/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 {
		return "general", ""
	}

	switch parts[0] {
	case "notifications":
		// /ws/notifications
		return "notifications", ""

	case "simulation":
		// /ws/simulation/{id}
		if len(parts) >= 2 {
			return "simulation", parts[1]
		}
		return "simulation", ""

	case "collaboration":
		// /ws/collaboration/{project_id}
		if len(parts) >= 2 {
			return "collaboration", parts[1]
		}
		return "collaboration", ""

	default:
		return "unknown", ""
	}
}

// validateResourceAccess validates if user has access to the WebSocket resource
func (gw *Gateway) validateResourceAccess(userID, wsType, resourceID string) bool {
	switch wsType {
	case "general", "notifications":
		// General and notifications are always accessible to authenticated users
		return true

	case "simulation":
		// TODO: Validate simulation access via simulation service
		// For now, allow all authenticated users
		if resourceID == "" {
			return false // Simulation ID is required
		}
		log.Printf("Validating simulation access: user=%s, simulation=%s", userID, resourceID)
		return true

	case "collaboration":
		// TODO: Validate project access via project service
		// For now, allow all authenticated users
		if resourceID == "" {
			return false // Project ID is required
		}
		log.Printf("Validating collaboration access: user=%s, project=%s", userID, resourceID)
		return true

	default:
		return false
	}
}

// parseSSEEventType extracts event type from SSE path
func (gw *Gateway) parseSSEEventType(path string) string {
	// /events/notifications -> "notifications"
	// /events/simulation/123/data -> "simulation"
	// /events/system/health -> "system"

	parts := strings.Split(strings.TrimPrefix(path, "/events/"), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// extractSimulationID extracts simulation ID from path
func (gw *Gateway) extractSimulationID(path string) string {
	// /events/simulation/123/data -> "123"
	parts := strings.Split(path, "/")
	if len(parts) >= 4 && parts[2] == "simulation" {
		return parts[3]
	}
	return ""
}

// handleSSENotifications handles notification events via SSE
func (gw *Gateway) handleSSENotifications(w http.ResponseWriter, flusher http.Flusher, ctx context.Context, userID string) {
	ticker := time.NewTicker(30 * time.Second) // Heartbeat
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("SSE notifications connection closed for user: %s", userID)
			return

		case <-ticker.C:
			// Send heartbeat
			fmt.Fprintf(w, "data: {\"type\":\"heartbeat\",\"timestamp\":%d}\n\n", time.Now().Unix())
			flusher.Flush()

		default:
			// TODO: Listen to Redis pub/sub for user notifications
			// For now, send periodic test notifications
			time.Sleep(5 * time.Second)

			notification := map[string]interface{}{
				"type":      "notification",
				"user_id":   userID,
				"title":     "Test Notification",
				"message":   "This is a test notification via SSE",
				"timestamp": time.Now().Unix(),
			}

			data, _ := json.Marshal(notification)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleSSESimulation handles simulation data events via SSE
func (gw *Gateway) handleSSESimulation(w http.ResponseWriter, flusher http.Flusher, ctx context.Context, userID, simulationID string) {
	ticker := time.NewTicker(1 * time.Second) // High frequency for simulation data
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("SSE simulation connection closed for user: %s, simulation: %s", userID, simulationID)
			return

		case <-ticker.C:
			// TODO: Get real simulation data from Redis streams
			// For now, send mock simulation metrics

			simulationData := map[string]interface{}{
				"type":          "simulation_data",
				"simulation_id": simulationID,
				"metrics": map[string]interface{}{
					"cpu_usage":    float64(time.Now().Unix()%100) / 100.0,
					"memory_usage": float64(time.Now().Unix()%80) / 100.0,
					"requests_per_sec": time.Now().Unix() % 1000,
				},
				"timestamp": time.Now().Unix(),
			}

			data, _ := json.Marshal(simulationData)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleSSESystem handles system events via SSE
func (gw *Gateway) handleSSESystem(w http.ResponseWriter, flusher http.Flusher, ctx context.Context, userID string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("SSE system connection closed for user: %s", userID)
			return

		case <-ticker.C:
			// Send system health updates
			systemData := map[string]interface{}{
				"type": "system_health",
				"services": map[string]interface{}{
					"auth_service":       true,
					"project_service":    true,
					"simulation_service": true,
				},
				"timestamp": time.Now().Unix(),
			}

			data, _ := json.Marshal(systemData)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleSSEGeneric handles generic SSE events
func (gw *Gateway) handleSSEGeneric(w http.ResponseWriter, flusher http.Flusher, ctx context.Context, userID, eventType string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("SSE generic connection closed for user: %s, type: %s", userID, eventType)
			return

		case <-ticker.C:
			// Send generic heartbeat
			genericData := map[string]interface{}{
				"type":       "generic_event",
				"event_type": eventType,
				"user_id":    userID,
				"message":    "Generic SSE event",
				"timestamp":  time.Now().Unix(),
			}

			data, _ := json.Marshal(genericData)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// authenticateWebSocketUser authenticates a WebSocket user from JWT token
func (gw *Gateway) authenticateWebSocketUser(r *http.Request) (string, error) {
	// Try to get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		token := services.ExtractTokenFromHeader(authHeader)
		if token != "" {
			// Validate token with auth service
			requestID := services.GenerateRequestID()
			resp, err := gw.authService.ValidateToken(token, "api-gateway-ws", requestID)
			if err == nil && resp.Valid {
				return resp.UserId, nil
			}
		}
	}

	// Try to get token from query parameter (for WebSocket connections)
	token := r.URL.Query().Get("token")
	if token != "" {
		requestID := services.GenerateRequestID()
		resp, err := gw.authService.ValidateToken(token, "api-gateway-ws", requestID)
		if err == nil && resp.Valid {
			return resp.UserId, nil
		}
	}

	// Try to get user_id from query parameter (for development/testing)
	userID := r.URL.Query().Get("user_id")
	if userID != "" {
		log.Printf("WebSocket: Using user_id from query parameter (development mode): %s", userID)
		return userID, nil
	}

	return "", fmt.Errorf("no valid authentication found")
}

// generateConnectionID generates a unique connection ID
func (gw *Gateway) generateConnectionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("conn-%d", time.Now().UnixNano())
	}
	return "conn-" + hex.EncodeToString(bytes)
}

// handleAuthRequest handles authentication service requests
func (gw *Gateway) handleAuthRequest(w http.ResponseWriter, r *http.Request, method, path string) {
	// Check if auth service is available
	if gw.authService == nil {
		gw.handleServiceUnavailable(w, r, "auth")
		return
	}

	// Route based on path and method
	switch {
	// Public auth routes
	case path == "/api/auth/register" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/auth/register")
	case path == "/api/auth/login" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/auth/login")
	case path == "/api/auth/refresh" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/auth/refresh")
	case path == "/api/auth/forgot-password" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/auth/forgot-password")
	case path == "/api/auth/reset-password" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/auth/reset-password")
	case path == "/api/auth/verify-email" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/auth/verify-email")
	case path == "/api/auth/resend-verification" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/auth/resend-verification")
	case path == "/api/auth/logout" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/auth/logout")
	case path == "/api/auth/health" && method == "GET":
		gw.handleAuthHealth(w, r)

	// User management routes
	case path == "/api/user/profile" && method == "GET":
		gw.forwardToAuthService(w, r, "/api/v1/user/profile")
	case path == "/api/user/profile" && method == "PUT":
		gw.forwardToAuthService(w, r, "/api/v1/user/profile")
	case path == "/api/user/change-password" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/user/change-password")
	case path == "/api/user/account" && method == "DELETE":
		gw.forwardToAuthService(w, r, "/api/v1/user/account")
	case path == "/api/user/sessions" && method == "GET":
		gw.forwardToAuthService(w, r, "/api/v1/user/sessions")
	case path == "/api/user/sessions" && method == "DELETE":
		gw.forwardToAuthService(w, r, "/api/v1/user/sessions")
	case path == "/api/user/stats" && method == "GET":
		gw.forwardToAuthService(w, r, "/api/v1/user/stats")

	// RBAC routes
	case path == "/api/rbac/my-roles" && method == "GET":
		gw.forwardToAuthService(w, r, "/api/v1/rbac/my-roles")
	case path == "/api/rbac/my-permissions" && method == "GET":
		gw.forwardToAuthService(w, r, "/api/v1/rbac/my-permissions")

	// Admin routes
	case path == "/api/admin/roles" && method == "GET":
		gw.forwardToAuthService(w, r, "/api/v1/admin/roles")
	case path == "/api/admin/permissions" && method == "GET":
		gw.forwardToAuthService(w, r, "/api/v1/admin/permissions")
	case path == "/api/admin/users/assign-role" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/admin/users/assign-role")
	case path == "/api/admin/users/remove-role" && method == "POST":
		gw.forwardToAuthService(w, r, "/api/v1/admin/users/remove-role")

	// Gateway-specific auth routes (using gRPC)
	case path == "/api/auth/validate" && method == "POST":
		gw.handleValidateToken(w, r)
	case path == "/api/auth/profile" && method == "GET":
		gw.handleGetProfile(w, r)
	case path == "/api/auth/permissions" && method == "GET":
		gw.handleGetPermissions(w, r)

	default:
		// Handle dynamic routes (like /api/user/sessions/:sessionId and /api/admin/users/:userId/roles)
		if gw.handleDynamicAuthRoutes(w, r, method, path) {
			return
		}
		gw.handleNotFound(w, r)
	}
}

// handleValidateToken handles token validation requests
func (gw *Gateway) handleValidateToken(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	token := services.ExtractTokenFromHeader(authHeader)

	if token == "" {
		err := errors.NewAuthenticationError("missing_token", "Authorization header with Bearer token required")
		gw.errorHandler.HandleError(w, r, err)
		return
	}

	// Use circuit breaker for auth service call
	breaker := gw.circuitManager.GetBreaker("auth")
	requestID := services.GenerateRequestID()

	result, err := breaker.Execute(func() (interface{}, error) {
		return gw.authService.ValidateToken(token, "api-gateway", requestID)
	})

	if err != nil {
		log.Printf("Token validation failed: %v", err)
		if err == circuit.ErrOpenState {
			err = errors.NewCircuitBreakerError("auth")
		} else {
			err = errors.NewAuthenticationError("token_validation_failed", "Token validation failed")
		}
		gw.errorHandler.HandleError(w, r, err)
		return
	}

	resp := result.(*services.ValidateTokenResponse)

	// Return validation result
	gw.sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"valid":      resp.Valid,
		"user_id":    resp.UserId,
		"expires_at": resp.ExpiresAt,
		"request_id": requestID,
	})
}

// sendJSONResponse sends a JSON response
func (gw *Gateway) sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v", err)
		http.Error(w, `{"error":"internal_error","message":"Failed to encode response"}`, http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

// handleGetProfile handles user profile requests
func (gw *Gateway) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := services.ExtractTokenFromHeader(authHeader)

	if token == "" {
		gw.sendJSONResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"error":   "missing_token",
			"message": "Authorization header required",
		})
		return
	}

	requestID := services.GenerateRequestID()
	tokenResp, err := gw.authService.ValidateToken(token, "api-gateway", requestID)
	if err != nil || !tokenResp.Valid {
		gw.sendJSONResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"error":   "invalid_token",
			"message": "Token validation failed",
		})
		return
	}

	userResp, err := gw.authService.GetUserContext(tokenResp.UserId, "api-gateway", requestID)
	if err != nil {
		log.Printf("Failed to get user context: %v", err)
		gw.sendJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"error":   "internal_error",
			"message": "Failed to retrieve user profile",
		})
		return
	}

	gw.sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"user_id":     userResp.UserId,
		"email":       userResp.Email,
		"roles":       userResp.Roles,
		"permissions": userResp.Permissions,
		"request_id":  requestID,
	})
}

// handleGetPermissions handles user permissions requests
func (gw *Gateway) handleGetPermissions(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := services.ExtractTokenFromHeader(authHeader)

	if token == "" {
		gw.sendJSONResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"error":   "missing_token",
			"message": "Authorization header required",
		})
		return
	}

	requestID := services.GenerateRequestID()
	tokenResp, err := gw.authService.ValidateToken(token, "api-gateway", requestID)
	if err != nil || !tokenResp.Valid {
		gw.sendJSONResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"error":   "invalid_token",
			"message": "Token validation failed",
		})
		return
	}

	permResp, err := gw.authService.GetUserPermissions(tokenResp.UserId, "api-gateway", requestID)
	if err != nil {
		log.Printf("Failed to get user permissions: %v", err)
		gw.sendJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"error":   "internal_error",
			"message": "Failed to retrieve user permissions",
		})
		return
	}

	gw.sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"user_id":     tokenResp.UserId,
		"permissions": permResp.Permissions,
		"roles":       permResp.Roles,
		"request_id":  requestID,
	})
}

// handleAuthHealth handles auth service health check
func (gw *Gateway) handleAuthHealth(w http.ResponseWriter, r *http.Request) {
	requestID := services.GenerateRequestID()
	resp, err := gw.authService.HealthCheck("api-gateway", requestID)
	if err != nil {
		gw.sendJSONResponse(w, http.StatusServiceUnavailable, map[string]interface{}{
			"error":   "service_unavailable",
			"message": "Auth service health check failed",
			"details": err.Error(),
		})
		return
	}

	gw.sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":     resp.Status,
		"service":    "auth-service",
		"request_id": requestID,
	})
}

// handleDynamicAuthRoutes handles auth routes with dynamic parameters
func (gw *Gateway) handleDynamicAuthRoutes(w http.ResponseWriter, r *http.Request, method, path string) bool {
	// Handle /api/user/sessions/:sessionId
	if method == "DELETE" && strings.HasPrefix(path, "/api/user/sessions/") && path != "/api/user/sessions/" {
		sessionID := strings.TrimPrefix(path, "/api/user/sessions/")
		if sessionID != "" {
			targetPath := "/api/v1/user/sessions/" + sessionID
			gw.forwardToAuthService(w, r, targetPath)
			return true
		}
	}

	// Handle /api/admin/users/:userId/roles
	if method == "GET" && strings.HasPrefix(path, "/api/admin/users/") && strings.HasSuffix(path, "/roles") {
		// Extract userId from path like /api/admin/users/123/roles
		parts := strings.Split(path, "/")
		if len(parts) == 6 && parts[1] == "api" && parts[2] == "admin" && parts[3] == "users" && parts[5] == "roles" {
			userID := parts[4]
			if userID != "" {
				targetPath := "/api/v1/admin/users/" + userID + "/roles"
				gw.forwardToAuthService(w, r, targetPath)
				return true
			}
		}
	}

	return false
}

// forwardToAuthService forwards HTTP requests to the auth service
func (gw *Gateway) forwardToAuthService(w http.ResponseWriter, r *http.Request, targetPath string) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		gw.errorHandler.HandleError(w, r, errors.NewValidationError("invalid_request", "Failed to read request body", nil))
		return
	}
	defer r.Body.Close()

	// Create request to auth service
	authServiceURL := "https://localhost:9001" + targetPath
	req, err := http.NewRequest(r.Method, authServiceURL, bytes.NewBuffer(body))
	if err != nil {
		gw.errorHandler.HandleError(w, r, errors.NewInternalError("request_creation_failed", "Failed to create auth service request"))
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Create HTTP/2 client with TLS skip verification for development
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			ForceAttemptHTTP2: true, // Force HTTP/2
		},
	}

	// Make the request
	log.Printf("Making HTTP/2 request to auth service: %s %s", req.Method, req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Auth service request failed: %v", err)
		gw.errorHandler.HandleError(w, r, errors.NewServiceUnavailableError("auth_service_unavailable", "Auth service is not available"))
		return
	}
	log.Printf("Auth service response: %d %s", resp.StatusCode, resp.Status)
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		gw.errorHandler.HandleError(w, r, errors.NewInternalError("response_read_failed", "Failed to read auth service response"))
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code and write response
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// handleProjectRequest handles project service requests
func (gw *Gateway) handleProjectRequest(w http.ResponseWriter, r *http.Request, method, path string) {
	// Provide mock responses when gRPC clients are not available
	if gw.grpcClients == nil {
		gw.handleProjectRequestMock(w, r, method, path)
		return
	}

	// TODO: Implement actual gRPC calls when services are available
	response := map[string]interface{}{
		"service": "project",
		"method":  method,
		"path":    path,
		"status":  "not_implemented",
		"message": "Project service integration pending",
	}

	gw.sendJSONResponse(w, http.StatusOK, response)
}

// handleProjectRequestMock provides mock responses for project service
func (gw *Gateway) handleProjectRequestMock(w http.ResponseWriter, r *http.Request, method, path string) {
	switch {
	case path == "/api/projects" && method == "GET":
		// Mock project list
		response := map[string]interface{}{
			"projects": []map[string]interface{}{
				{
					"id":          "project-123",
					"name":        "Mock E-commerce System",
					"description": "A mock e-commerce platform design",
					"owner_id":    "mock-user-123",
					"created_at":  "2023-12-01T12:00:00Z",
					"updated_at":  "2023-12-01T15:30:00Z",
					"permissions": []string{"read", "write", "share"},
				},
			},
			"total":  1,
			"limit":  20,
			"offset": 0,
		}
		gw.sendJSONResponse(w, http.StatusOK, response)

	case path == "/api/projects" && method == "POST":
		// Mock project creation
		response := map[string]interface{}{
			"id":          "project-456",
			"name":        "New Mock Project",
			"description": "A newly created mock project",
			"owner_id":    "mock-user-123",
			"created_at":  time.Now().Format(time.RFC3339),
			"updated_at":  time.Now().Format(time.RFC3339),
		}
		gw.sendJSONResponse(w, http.StatusCreated, response)

	default:
		// Default mock response
		response := map[string]interface{}{
			"service": "project",
			"method":  method,
			"path":    path,
			"status":  "mock_response",
			"message": "Mock project service response",
			"data":    map[string]interface{}{"mock": true},
		}
		gw.sendJSONResponse(w, http.StatusOK, response)
	}
}

// handleSimulationRequest handles simulation service requests
func (gw *Gateway) handleSimulationRequest(w http.ResponseWriter, r *http.Request, method, path string) {
	// Provide mock responses when gRPC clients are not available
	if gw.grpcClients == nil {
		gw.handleSimulationRequestMock(w, r, method, path)
		return
	}

	// TODO: Implement actual gRPC calls when services are available
	response := map[string]interface{}{
		"service": "simulation",
		"method":  method,
		"path":    path,
		"status":  "not_implemented",
		"message": "Simulation service integration pending",
	}

	gw.sendJSONResponse(w, http.StatusOK, response)
}

// handleSimulationRequestMock provides mock responses for simulation service
func (gw *Gateway) handleSimulationRequestMock(w http.ResponseWriter, r *http.Request, method, path string) {
	switch {
	case path == "/api/simulations" && method == "GET":
		// Mock simulation list
		response := map[string]interface{}{
			"simulations": []map[string]interface{}{
				{
					"id":         "sim-123",
					"project_id": "project-123",
					"name":       "Mock Load Test Simulation",
					"status":     "completed",
					"created_at": "2023-12-01T14:00:00Z",
					"started_at": "2023-12-01T14:05:00Z",
					"progress":   100,
				},
			},
			"total": 1,
		}
		gw.sendJSONResponse(w, http.StatusOK, response)

	case path == "/api/simulations" && method == "POST":
		// Mock simulation creation
		response := map[string]interface{}{
			"id":         "sim-456",
			"project_id": "project-123",
			"name":       "New Mock Simulation",
			"status":     "created",
			"created_at": time.Now().Format(time.RFC3339),
		}
		gw.sendJSONResponse(w, http.StatusCreated, response)

	default:
		// Default mock response
		response := map[string]interface{}{
			"service": "simulation",
			"method":  method,
			"path":    path,
			"status":  "mock_response",
			"message": "Mock simulation service response",
			"data":    map[string]interface{}{"mock": true},
		}
		gw.sendJSONResponse(w, http.StatusOK, response)
	}
}

// handleHealthCheck handles comprehensive health check requests with service aggregation
func (gw *Gateway) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Collect health from all services
	healthData := gw.aggregateServiceHealth()

	// Calculate response time
	responseTimeMs := time.Since(start).Milliseconds()
	healthData["response_time_ms"] = responseTimeMs
	healthData["timestamp"] = time.Now().Unix()

	// Determine HTTP status code based on overall health
	statusCode := http.StatusOK
	if healthData["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	} else if healthData["status"] == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	gw.sendJSONResponse(w, statusCode, healthData)
}

// aggregateServiceHealth collects health status from all microservices and dependencies
func (gw *Gateway) aggregateServiceHealth() map[string]interface{} {
	var errors []string
	services := make(map[string]interface{})

	// Check gRPC services health with detailed information
	grpcServices := gw.checkGRPCServicesHealth()
	services["grpc_services"] = grpcServices["services"]
	if grpcServices["errors"] != nil {
		if grpcErrors, ok := grpcServices["errors"].([]string); ok {
			errors = append(errors, grpcErrors...)
		}
	}

	// Check Redis health
	redisHealth := gw.checkRedisHealth()
	services["redis"] = redisHealth["status"]
	if redisHealth["error"] != nil {
		errors = append(errors, fmt.Sprintf("redis: %v", redisHealth["error"]))
	}

	// Check WebSocket hub health
	wsHealth := gw.checkWebSocketHubHealth()
	services["websocket_hub"] = wsHealth

	// Check API Gateway internal health
	gatewayHealth := gw.checkGatewayHealth()
	services["api_gateway"] = gatewayHealth

	// Determine overall status
	overallStatus := "healthy"
	if len(errors) > 0 {
		// Check if any critical services are down
		if grpcServices["critical_down"].(bool) {
			overallStatus = "unhealthy"
		} else {
			overallStatus = "degraded"
		}
	}

	result := map[string]interface{}{
		"status":   overallStatus,
		"services": services,
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	return result
}

// checkGRPCServicesHealth performs detailed health checks on all gRPC services
func (gw *Gateway) checkGRPCServicesHealth() map[string]interface{} {
	services := make(map[string]interface{})
	var errors []string
	criticalDown := false

	if gw.grpcClients != nil {
		// Check Auth Service (critical)
		authHealth := gw.checkAuthServiceHealth()
		services["auth_service"] = authHealth
		if !authHealth["healthy"].(bool) {
			criticalDown = true
			errors = append(errors, fmt.Sprintf("auth_service: %v", authHealth["error"]))
		}

		// Check Project Service (TODO: implement when service is ready)
		projectHealth := gw.checkProjectServiceHealth()
		services["project_service"] = projectHealth
		if !projectHealth["healthy"].(bool) {
			errors = append(errors, fmt.Sprintf("project_service: %v", projectHealth["error"]))
		}

		// Check Simulation Service (TODO: implement when service is ready)
		simulationHealth := gw.checkSimulationServiceHealth()
		services["simulation_service"] = simulationHealth
		if !simulationHealth["healthy"].(bool) {
			errors = append(errors, fmt.Sprintf("simulation_service: %v", simulationHealth["error"]))
		}
	} else {
		// Mock implementations when gRPC clients are not available
		services["auth_service"] = map[string]interface{}{
			"healthy": true,
			"status":  "mock",
			"message": "Using mock implementation",
		}
		services["project_service"] = map[string]interface{}{
			"healthy": true,
			"status":  "mock",
			"message": "Using mock implementation",
		}
		services["simulation_service"] = map[string]interface{}{
			"healthy": true,
			"status":  "mock",
			"message": "Using mock implementation",
		}
	}

	return map[string]interface{}{
		"services":      services,
		"errors":        errors,
		"critical_down": criticalDown,
	}
}

// checkAuthServiceHealth performs detailed health check on auth service
func (gw *Gateway) checkAuthServiceHealth() map[string]interface{} {
	if gw.authService == nil {
		return map[string]interface{}{
			"healthy": false,
			"status":  "unavailable",
			"error":   "auth service client not initialized",
		}
	}

	start := time.Now()
	requestID := services.GenerateRequestID()

	resp, err := gw.authService.HealthCheck("api-gateway", requestID)
	responseTime := time.Since(start).Milliseconds()

	if err != nil {
		return map[string]interface{}{
			"healthy":         false,
			"status":          "unhealthy",
			"error":           err.Error(),
			"response_time_ms": responseTime,
		}
	}

	healthy := resp.Status == "healthy"
	return map[string]interface{}{
		"healthy":          healthy,
		"status":           resp.Status,
		"response_time_ms": responseTime,
		"request_id":       requestID,
	}
}

// checkProjectServiceHealth performs health check on project service
func (gw *Gateway) checkProjectServiceHealth() map[string]interface{} {
	// TODO: Implement when project service is ready
	// For now, check if gRPC connection pool exists and is healthy
	if gw.grpcClients != nil {
		projectPool := gw.grpcClients.GetProjectPool()
		if projectPool != nil && projectPool.HealthCheck() {
			return map[string]interface{}{
				"healthy": true,
				"status":  "healthy",
				"message": "gRPC connection pool healthy",
			}
		}
		return map[string]interface{}{
			"healthy": false,
			"status":  "unhealthy",
			"error":   "gRPC connection pool unhealthy",
		}
	}

	return map[string]interface{}{
		"healthy": false,
		"status":  "not_implemented",
		"error":   "project service not implemented yet",
	}
}

// checkSimulationServiceHealth performs health check on simulation service
func (gw *Gateway) checkSimulationServiceHealth() map[string]interface{} {
	// TODO: Implement when simulation service is ready
	// For now, check if gRPC connection pool exists and is healthy
	if gw.grpcClients != nil {
		simulationPool := gw.grpcClients.GetSimulationPool()
		if simulationPool != nil && simulationPool.HealthCheck() {
			return map[string]interface{}{
				"healthy": true,
				"status":  "healthy",
				"message": "gRPC connection pool healthy",
			}
		}
		return map[string]interface{}{
			"healthy": false,
			"status":  "unhealthy",
			"error":   "gRPC connection pool unhealthy",
		}
	}

	return map[string]interface{}{
		"healthy": false,
		"status":  "not_implemented",
		"error":   "simulation service not implemented yet",
	}
}

// checkRedisHealth performs detailed health check on Redis
func (gw *Gateway) checkRedisHealth() map[string]interface{} {
	if gw.redisClient == nil {
		return map[string]interface{}{
			"status": false,
			"error":  "redis client not initialized",
		}
	}

	start := time.Now()
	healthy := gw.redisClient.HealthCheck()
	responseTime := time.Since(start).Milliseconds()

	if healthy {
		return map[string]interface{}{
			"status":           true,
			"response_time_ms": responseTime,
		}
	}

	return map[string]interface{}{
		"status":           false,
		"error":            "redis health check failed",
		"response_time_ms": responseTime,
	}
}

// checkWebSocketHubHealth gets WebSocket hub statistics and health
func (gw *Gateway) checkWebSocketHubHealth() map[string]interface{} {
	if gw.wsHub == nil {
		return map[string]interface{}{
			"healthy": false,
			"error":   "websocket hub not initialized",
		}
	}

	stats := gw.wsHub.GetStats()

	// Add health status based on hub statistics
	hubHealth := map[string]interface{}{
		"healthy":             true,
		"active_connections":  stats["active_connections"],
		"total_messages":      stats["total_messages"],
		"messages_processed":  stats["messages_processed"],
	}

	// Add messages per second if available
	if messagesPerSecond, exists := stats["messages_per_second"]; exists {
		hubHealth["messages_per_second"] = messagesPerSecond
	}

	return hubHealth
}

// checkGatewayHealth performs internal API Gateway health checks
func (gw *Gateway) checkGatewayHealth() map[string]interface{} {
	return map[string]interface{}{
		"healthy":            true,
		"requests_processed": gw.requestsProcessed,
		"requests_per_second": gw.requestsPerSecond,
		"avg_response_time_ms": gw.avgResponseTime.Milliseconds(),
		"uptime_seconds":     time.Since(time.Now().Add(-time.Hour)).Seconds(), // TODO: Track actual start time
	}
}

// handleAuthHealthCheck handles individual auth service health check
func (gw *Gateway) handleAuthHealthCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	authHealth := gw.checkAuthServiceHealth()
	responseTime := time.Since(start).Milliseconds()

	// Add response time to the health data
	authHealth["response_time_ms"] = responseTime
	authHealth["service"] = "auth-service"
	authHealth["timestamp"] = time.Now().Unix()

	statusCode := http.StatusOK
	if !authHealth["healthy"].(bool) {
		statusCode = http.StatusServiceUnavailable
	}

	gw.sendJSONResponse(w, statusCode, authHealth)
}

// handleProjectHealthCheck handles individual project service health check
func (gw *Gateway) handleProjectHealthCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	projectHealth := gw.checkProjectServiceHealth()
	responseTime := time.Since(start).Milliseconds()

	// Add response time to the health data
	projectHealth["response_time_ms"] = responseTime
	projectHealth["service"] = "project-service"
	projectHealth["timestamp"] = time.Now().Unix()

	statusCode := http.StatusOK
	if !projectHealth["healthy"].(bool) {
		statusCode = http.StatusServiceUnavailable
	}

	gw.sendJSONResponse(w, statusCode, projectHealth)
}

// handleSimulationHealthCheck handles individual simulation service health check
func (gw *Gateway) handleSimulationHealthCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	simulationHealth := gw.checkSimulationServiceHealth()
	responseTime := time.Since(start).Milliseconds()

	// Add response time to the health data
	simulationHealth["response_time_ms"] = responseTime
	simulationHealth["service"] = "simulation-service"
	simulationHealth["timestamp"] = time.Now().Unix()

	statusCode := http.StatusOK
	if !simulationHealth["healthy"].(bool) {
		statusCode = http.StatusServiceUnavailable
	}

	gw.sendJSONResponse(w, statusCode, simulationHealth)
}

// handleGRPCStats handles gRPC pool statistics requests
func (gw *Gateway) handleGRPCStats(w http.ResponseWriter, r *http.Request) {
	if gw.grpcClients == nil {
		gw.sendJSONResponse(w, http.StatusServiceUnavailable, map[string]interface{}{
			"error": "gRPC clients not available",
			"stats": nil,
		})
		return
	}

	stats := gw.grpcClients.GetStats()
	gw.sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
		"grpc_pools": stats,
	})
}

// handleMetrics handles metrics requests
func (gw *Gateway) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"gateway": map[string]interface{}{
			"requests_processed":   gw.requestsProcessed,
			"requests_per_second":  gw.requestsPerSecond,
			"avg_response_time_ms": gw.avgResponseTime.Milliseconds(),
		},
		"websocket_hub":    gw.wsHub.GetStats(),
		"server":           gw.server.GetServerStats(),
		"circuit_breakers": gw.circuitManager.GetStats(),
	}

	if gw.grpcClients != nil {
		metrics["grpc_clients"] = gw.grpcClients.GetStats()
	} else {
		metrics["grpc_clients"] = map[string]interface{}{"status": "unavailable"}
	}

	if gw.redisClient != nil {
		metrics["redis_client"] = gw.redisClient.GetStats()
	} else {
		metrics["redis_client"] = map[string]interface{}{"status": "unavailable"}
	}

	gw.sendJSONResponse(w, http.StatusOK, metrics)
}

// handleServiceUnavailable handles service unavailable responses
func (gw *Gateway) handleServiceUnavailable(w http.ResponseWriter, r *http.Request, serviceName string) {
	response := map[string]interface{}{
		"error":   "service_unavailable",
		"service": serviceName,
		"message": fmt.Sprintf("%s service is currently unavailable", serviceName),
	}

	gw.sendJSONResponse(w, http.StatusServiceUnavailable, response)
}

// handleNotFound handles 404 responses
func (gw *Gateway) handleNotFound(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"error":   "not_found",
		"path":    r.URL.Path,
		"message": "Endpoint not found",
	}

	gw.sendJSONResponse(w, http.StatusNotFound, response)
}

// monitorPerformance monitors gateway performance metrics
func (gw *Gateway) monitorPerformance() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastProcessed int64
	for range ticker.C {
		current := gw.requestsProcessed
		gw.requestsPerSecond = current - lastProcessed
		lastProcessed = current
	}
}
