package test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apierrors "server-service/internal/errors"
)

// TestAPIErrorCreation tests API error creation
func TestAPIErrorCreation(t *testing.T) {
	testCases := []struct {
		name           string
		errorFunc      func() *apierrors.APIError
		expectedType   apierrors.ErrorType
		expectedStatus int
	}{
		{
			name: "Validation Error",
			errorFunc: func() *apierrors.APIError {
				return apierrors.NewValidationError("invalid_input", "Input validation failed", map[string]string{"field": "email"})
			},
			expectedType:   apierrors.ErrorTypeValidation,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Authentication Error",
			errorFunc: func() *apierrors.APIError {
				return apierrors.NewAuthenticationError("invalid_token", "Token is invalid")
			},
			expectedType:   apierrors.ErrorTypeAuthentication,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Authorization Error",
			errorFunc: func() *apierrors.APIError {
				return apierrors.NewAuthorizationError("insufficient_permissions", "User lacks required permissions")
			},
			expectedType:   apierrors.ErrorTypeAuthorization,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Not Found Error",
			errorFunc: func() *apierrors.APIError {
				return apierrors.NewNotFoundError("resource_not_found", "Resource does not exist")
			},
			expectedType:   apierrors.ErrorTypeNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service Unavailable Error",
			errorFunc: func() *apierrors.APIError {
				return apierrors.NewServiceUnavailableError("service_down", "Service is temporarily unavailable")
			},
			expectedType:   apierrors.ErrorTypeServiceUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "Circuit Breaker Error",
			errorFunc: func() *apierrors.APIError {
				return apierrors.NewCircuitBreakerError("auth")
			},
			expectedType:   apierrors.ErrorTypeCircuitBreaker,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "Timeout Error",
			errorFunc: func() *apierrors.APIError {
				return apierrors.NewTimeoutError("database_query", 5*time.Second)
			},
			expectedType:   apierrors.ErrorTypeTimeout,
			expectedStatus: http.StatusRequestTimeout,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := tc.errorFunc()

			if apiErr.Type != tc.expectedType {
				t.Errorf("Expected error type %s, got %s", tc.expectedType, apiErr.Type)
			}

			if apiErr.HTTPStatus() != tc.expectedStatus {
				t.Errorf("Expected HTTP status %d, got %d", tc.expectedStatus, apiErr.HTTPStatus())
			}

			if apiErr.Timestamp.IsZero() {
				t.Error("Expected timestamp to be set")
			}

			if apiErr.Code == "" {
				t.Error("Expected error code to be set")
			}

			if apiErr.Message == "" {
				t.Error("Expected error message to be set")
			}
		})
	}
}

// TestErrorHandlerHandleError tests error handling
func TestErrorHandlerHandleError(t *testing.T) {
	handler := apierrors.NewErrorHandler()

	testCases := []struct {
		name           string
		error          error
		expectedStatus int
		expectedType   apierrors.ErrorType
	}{
		{
			name:           "API Error",
			error:          apierrors.NewValidationError("test_code", "test message", nil),
			expectedStatus: http.StatusBadRequest,
			expectedType:   apierrors.ErrorTypeValidation,
		},
		{
			name:           "Generic Error",
			error:          errors.New("generic error"),
			expectedStatus: http.StatusInternalServerError,
			expectedType:   apierrors.ErrorTypeInternal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test request and response recorder
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Request-ID", "test-request-123")
			w := httptest.NewRecorder()

			// Handle the error
			handler.HandleError(w, req, tc.error)

			// Check response status
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected content type 'application/json', got '%s'", contentType)
			}

			// Parse response body
			var responseErr apierrors.APIError
			if err := json.NewDecoder(w.Body).Decode(&responseErr); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			// Check error type
			if responseErr.Type != tc.expectedType {
				t.Errorf("Expected error type %s, got %s", tc.expectedType, responseErr.Type)
			}

			// Check request ID was set
			if responseErr.RequestID != "test-request-123" {
				t.Errorf("Expected request ID 'test-request-123', got '%s'", responseErr.RequestID)
			}

			// Check path was set
			if responseErr.Path != "/test" {
				t.Errorf("Expected path '/test', got '%s'", responseErr.Path)
			}
		})
	}
}

// TestErrorHandlerRecovery tests panic recovery middleware
func TestErrorHandlerRecovery(t *testing.T) {
	handler := apierrors.NewErrorHandler()

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	recoveryHandler := handler.Recovery(panicHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic
	recoveryHandler.ServeHTTP(w, req)

	// Check that error response was sent
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var responseErr apierrors.APIError
	if err := json.NewDecoder(w.Body).Decode(&responseErr); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if responseErr.Type != apierrors.ErrorTypeInternal {
		t.Errorf("Expected error type %s, got %s", apierrors.ErrorTypeInternal, responseErr.Type)
	}

	if responseErr.Code != "panic_recovered" {
		t.Errorf("Expected error code 'panic_recovered', got '%s'", responseErr.Code)
	}
}

// TestErrorHandlerTimeout tests timeout middleware
func TestErrorHandlerTimeout(t *testing.T) {
	handler := apierrors.NewErrorHandler()

	// Create a slow handler
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with timeout middleware (100ms timeout)
	timeoutHandler := handler.Timeout(100 * time.Millisecond)(slowHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should timeout
	timeoutHandler.ServeHTTP(w, req)

	// Check that timeout error response was sent
	if w.Code != http.StatusRequestTimeout {
		t.Errorf("Expected status 408, got %d", w.Code)
	}

	var responseErr apierrors.APIError
	if err := json.NewDecoder(w.Body).Decode(&responseErr); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if responseErr.Type != apierrors.ErrorTypeTimeout {
		t.Errorf("Expected error type %s, got %s", apierrors.ErrorTypeTimeout, responseErr.Type)
	}

	if responseErr.Code != "request_timeout" {
		t.Errorf("Expected error code 'request_timeout', got '%s'", responseErr.Code)
	}
}

// TestErrorHandlerTimeoutSuccess tests that fast requests don't timeout
func TestErrorHandlerTimeoutSuccess(t *testing.T) {
	handler := apierrors.NewErrorHandler()

	// Create a fast handler
	fastHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with timeout middleware (100ms timeout)
	timeoutHandler := handler.Timeout(100 * time.Millisecond)(fastHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should succeed
	timeoutHandler.ServeHTTP(w, req)

	// Check that success response was sent
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "success" {
		t.Errorf("Expected body 'success', got '%s'", body)
	}
}

// TestAPIErrorJSONSerialization tests JSON serialization of API errors
func TestAPIErrorJSONSerialization(t *testing.T) {
	apiErr := &apierrors.APIError{
		Type:      apierrors.ErrorTypeValidation,
		Code:      "test_code",
		Message:   "test message",
		Details:   map[string]string{"field": "email"},
		RequestID: "test-request-123",
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Path:      "/test",
	}

	jsonData, err := json.Marshal(apiErr)
	if err != nil {
		t.Fatalf("Failed to marshal API error: %v", err)
	}

	var unmarshaled apierrors.APIError
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal API error: %v", err)
	}

	if unmarshaled.Type != apiErr.Type {
		t.Errorf("Expected type %s, got %s", apiErr.Type, unmarshaled.Type)
	}

	if unmarshaled.Code != apiErr.Code {
		t.Errorf("Expected code %s, got %s", apiErr.Code, unmarshaled.Code)
	}

	if unmarshaled.Message != apiErr.Message {
		t.Errorf("Expected message %s, got %s", apiErr.Message, unmarshaled.Message)
	}

	if unmarshaled.RequestID != apiErr.RequestID {
		t.Errorf("Expected request ID %s, got %s", apiErr.RequestID, unmarshaled.RequestID)
	}

	if unmarshaled.Path != apiErr.Path {
		t.Errorf("Expected path %s, got %s", apiErr.Path, unmarshaled.Path)
	}
}
