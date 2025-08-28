package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"server-service/internal/config"
	"server-service/internal/gateway"
	"server-service/internal/websocket"
)

// TestServer represents a test server instance
type TestServer struct {
	gateway *gateway.Gateway
	baseURL string
	client  *http.Client
}

// NewTestServer creates a new test server
func NewTestServer(t *testing.T) *TestServer {
	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:                 "localhost",
			Port:                 "8080",
			TLSEnabled:           false, // Disable TLS for testing
			ReadTimeout:          30 * time.Second,
			WriteTimeout:         30 * time.Second,
			IdleTimeout:          60 * time.Second,
			MaxRequestBodySize:   10 * 1024 * 1024, // 10MB
			MaxConcurrentStreams: 1000,
			MaxFrameSize:         16384,
			HTTP2IdleTimeout:     300 * time.Second,
		},
	}

	// Create mock WebSocket hub
	wsHub := websocket.NewHub(nil)

	// Create gateway configuration
	gatewayConfig := &gateway.Config{
		ServerConfig: &cfg.Server,
		GRPCClients:  nil, // Mock gRPC clients for testing
		WebSocketHub: wsHub,
		RedisClient:  nil, // Mock Redis client for testing
	}

	// Create gateway
	gw := gateway.New(gatewayConfig)

	return &TestServer{
		gateway: gw,
		baseURL: "http://localhost:8080",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start starts the test server
func (ts *TestServer) Start(t *testing.T) {
	go func() {
		if err := ts.gateway.Start(); err != nil {
			t.Logf("Test server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)
}

// Stop stops the test server
func (ts *TestServer) Stop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ts.gateway.Shutdown(ctx); err != nil {
		t.Logf("Error shutting down test server: %v", err)
	}
}

// makeRequest makes an HTTP request to the test server
func (ts *TestServer) makeRequest(method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, ts.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return ts.client.Do(req)
}

// TestHealthEndpoint tests the health endpoint
func TestHealthEndpoint(t *testing.T) {
	server := NewTestServer(t)
	server.Start(t)
	defer server.Stop(t)

	resp, err := server.makeRequest("GET", "/health", nil, nil)
	if err != nil {
		t.Fatalf("Failed to make health request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if _, exists := health["status"]; !exists {
		t.Error("Health response missing 'status' field")
	}

	if _, exists := health["services"]; !exists {
		t.Error("Health response missing 'services' field")
	}
}

// TestMetricsEndpoint tests the metrics endpoint
func TestMetricsEndpoint(t *testing.T) {
	server := NewTestServer(t)
	server.Start(t)
	defer server.Stop(t)

	resp, err := server.makeRequest("GET", "/metrics", nil, nil)
	if err != nil {
		t.Fatalf("Failed to make metrics request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var metrics map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		t.Fatalf("Failed to decode metrics response: %v", err)
	}

	expectedFields := []string{"gateway", "websocket_hub", "server", "circuit_breakers"}
	for _, field := range expectedFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Metrics response missing '%s' field", field)
		}
	}
}

// TestCORSHeaders tests CORS headers
func TestCORSHeaders(t *testing.T) {
	server := NewTestServer(t)
	server.Start(t)
	defer server.Stop(t)

	// Test OPTIONS request
	resp, err := server.makeRequest("OPTIONS", "/health", nil, nil)
	if err != nil {
		t.Fatalf("Failed to make OPTIONS request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", resp.StatusCode)
	}

	// Check CORS headers
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected %s header to be '%s', got '%s'", header, expectedValue, actualValue)
		}
	}
}

// TestAuthEndpoints tests authentication endpoints
func TestAuthEndpoints(t *testing.T) {
	server := NewTestServer(t)
	server.Start(t)
	defer server.Stop(t)

	testCases := []struct {
		name           string
		path           string
		method         string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:           "Validate token without auth header",
			path:           "/api/auth/validate",
			method:         "POST",
			headers:        nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "Validate token with auth header",
			path:   "/api/auth/validate",
			method: "POST",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Get profile with auth header",
			path:   "/api/auth/profile",
			method: "GET",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Get permissions with auth header",
			path:   "/api/auth/permissions",
			method: "GET",
			headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Auth health check",
			path:           "/api/auth/health",
			method:         "GET",
			headers:        nil,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := server.makeRequest(tc.method, tc.path, nil, tc.headers)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Response: %s", tc.expectedStatus, resp.StatusCode, string(body))
			}
		})
	}
}

// TestNotFoundEndpoint tests 404 handling
func TestNotFoundEndpoint(t *testing.T) {
	server := NewTestServer(t)
	server.Start(t)
	defer server.Stop(t)

	resp, err := server.makeRequest("GET", "/nonexistent", nil, nil)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	var errorResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorResp["error"] != "not_found" {
		t.Errorf("Expected error 'not_found', got '%v'", errorResp["error"])
	}
}

// TestWebSocketEndpoint tests WebSocket endpoint
func TestWebSocketEndpoint(t *testing.T) {
	server := NewTestServer(t)
	server.Start(t)
	defer server.Stop(t)

	resp, err := server.makeRequest("GET", "/ws?user_id=test-user", nil, nil)
	if err != nil {
		t.Fatalf("Failed to make WebSocket request: %v", err)
	}
	defer resp.Body.Close()

	// Since WebSocket upgrade is not fully implemented, we expect a JSON response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var wsResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&wsResp); err != nil {
		t.Fatalf("Failed to decode WebSocket response: %v", err)
	}

	if wsResp["message"] != "WebSocket endpoint available" {
		t.Errorf("Unexpected WebSocket response: %v", wsResp)
	}
}
