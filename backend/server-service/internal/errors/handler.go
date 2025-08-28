package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeValidation         ErrorType = "validation_error"
	ErrorTypeAuthentication     ErrorType = "authentication_error"
	ErrorTypeAuthorization      ErrorType = "authorization_error"
	ErrorTypeNotFound           ErrorType = "not_found_error"
	ErrorTypeConflict           ErrorType = "conflict_error"
	ErrorTypeInternal           ErrorType = "internal_error"
	ErrorTypeServiceUnavailable ErrorType = "service_unavailable_error"
	ErrorTypeTimeout            ErrorType = "timeout_error"
	ErrorTypeRateLimit          ErrorType = "rate_limit_error"
	ErrorTypeCircuitBreaker     ErrorType = "circuit_breaker_error"
)

// APIError represents a structured API error
type APIError struct {
	Type      ErrorType   `json:"type"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Path      string      `json:"path,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *APIError) HTTPStatus() int {
	switch e.Type {
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeAuthentication:
		return http.StatusUnauthorized
	case ErrorTypeAuthorization:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrorTypeTimeout:
		return http.StatusRequestTimeout
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeCircuitBreaker:
		return http.StatusServiceUnavailable
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger *log.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		logger: log.New(log.Writer(), "[ERROR] ", log.LstdFlags|log.Lshortfile),
	}
}

// HandleError handles an error and sends appropriate response
func (eh *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *APIError

	// Convert error to APIError if it's not already
	if ae, ok := err.(*APIError); ok {
		apiErr = ae
	} else {
		apiErr = &APIError{
			Type:      ErrorTypeInternal,
			Code:      "internal_server_error",
			Message:   "An internal server error occurred",
			Timestamp: time.Now(),
			Path:      r.URL.Path,
		}

		// Log the original error for debugging
		eh.logger.Printf("Internal error: %v", err)
	}

	// Set request ID if available
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		apiErr.RequestID = requestID
	}

	// Set path if not already set
	if apiErr.Path == "" {
		apiErr.Path = r.URL.Path
	}

	// Set timestamp if not already set
	if apiErr.Timestamp.IsZero() {
		apiErr.Timestamp = time.Now()
	}

	// Log error
	eh.logError(apiErr, r)

	// Send JSON response
	eh.sendErrorResponse(w, apiErr)
}

// logError logs the error with context
func (eh *ErrorHandler) logError(err *APIError, r *http.Request) {
	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	caller := "unknown"
	if ok {
		caller = fmt.Sprintf("%s:%d", file, line)
	}

	eh.logger.Printf(
		"Error: %s | Code: %s | Message: %s | Path: %s | Method: %s | RequestID: %s | Caller: %s",
		err.Type,
		err.Code,
		err.Message,
		err.Path,
		r.Method,
		err.RequestID,
		caller,
	)
}

// sendErrorResponse sends the error as JSON response
func (eh *ErrorHandler) sendErrorResponse(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus())

	if jsonErr := json.NewEncoder(w).Encode(err); jsonErr != nil {
		eh.logger.Printf("Failed to encode error response: %v", jsonErr)
		// Fallback to plain text
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Internal server error")
	}
}

// Recovery middleware for panic recovery
func (eh *ErrorHandler) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic with stack trace
				stack := make([]byte, 4096)
				length := runtime.Stack(stack, false)
				eh.logger.Printf("Panic recovered: %v\nStack trace:\n%s", err, stack[:length])

				// Create error response
				apiErr := &APIError{
					Type:      ErrorTypeInternal,
					Code:      "panic_recovered",
					Message:   "An unexpected error occurred",
					Timestamp: time.Now(),
					Path:      r.URL.Path,
				}

				eh.sendErrorResponse(w, apiErr)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Timeout middleware for request timeouts
func (eh *ErrorHandler) Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)

			done := make(chan bool, 1)
			go func() {
				next.ServeHTTP(w, r)
				done <- true
			}()

			select {
			case <-done:
				// Request completed normally
			case <-ctx.Done():
				// Request timed out
				if ctx.Err() == context.DeadlineExceeded {
					apiErr := &APIError{
						Type:      ErrorTypeTimeout,
						Code:      "request_timeout",
						Message:   fmt.Sprintf("Request timed out after %v", timeout),
						Timestamp: time.Now(),
						Path:      r.URL.Path,
					}
					eh.sendErrorResponse(w, apiErr)
				}
			}
		})
	}
}

// Helper functions for creating specific error types

// NewValidationError creates a validation error
func NewValidationError(code, message string, details interface{}) *APIError {
	return &APIError{
		Type:      ErrorTypeValidation,
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(code, message string) *APIError {
	return &APIError{
		Type:      ErrorTypeAuthentication,
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewAuthorizationError creates an authorization error
func NewAuthorizationError(code, message string) *APIError {
	return &APIError{
		Type:      ErrorTypeAuthorization,
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(code, message string) *APIError {
	return &APIError{
		Type:      ErrorTypeNotFound,
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewServiceUnavailableError creates a service unavailable error
func NewServiceUnavailableError(code, message string) *APIError {
	return &APIError{
		Type:      ErrorTypeServiceUnavailable,
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewCircuitBreakerError creates a circuit breaker error
func NewCircuitBreakerError(serviceName string) *APIError {
	return &APIError{
		Type:      ErrorTypeCircuitBreaker,
		Code:      "circuit_breaker_open",
		Message:   fmt.Sprintf("Service %s is currently unavailable (circuit breaker open)", serviceName),
		Timestamp: time.Now(),
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(operation string, timeout time.Duration) *APIError {
	return &APIError{
		Type:      ErrorTypeTimeout,
		Code:      "operation_timeout",
		Message:   fmt.Sprintf("Operation %s timed out after %v", operation, timeout),
		Timestamp: time.Now(),
	}
}

// NewInternalError creates an internal server error
func NewInternalError(code, message string) *APIError {
	return &APIError{
		Type:      ErrorTypeInternal,
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}
