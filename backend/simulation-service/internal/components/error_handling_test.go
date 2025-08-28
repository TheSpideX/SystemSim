package components

import (
	"context"
	"errors"
	"testing"
)

func TestComponentError_Creation(t *testing.T) {
	compErr := CreateComponentError(
		ErrorCategoryTransient,
		ErrorSeverityMedium,
		"test_error",
		"This is a test error",
		"test-component",
	)

	if compErr.Category != ErrorCategoryTransient {
		t.Errorf("Expected category %s, got %s", ErrorCategoryTransient, compErr.Category)
	}

	if compErr.Severity != ErrorSeverityMedium {
		t.Errorf("Expected severity %s, got %s", ErrorSeverityMedium, compErr.Severity)
	}

	if compErr.Code != "test_error" {
		t.Errorf("Expected code 'test_error', got %s", compErr.Code)
	}

	if compErr.Message != "This is a test error" {
		t.Errorf("Expected message 'This is a test error', got %s", compErr.Message)
	}

	if compErr.ComponentID != "test-component" {
		t.Errorf("Expected component ID 'test-component', got %s", compErr.ComponentID)
	}

	if compErr.ID == "" {
		t.Error("Expected non-empty error ID")
	}

	if compErr.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestComponentError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		category ErrorCategory
		expected bool
	}{
		{"Transient errors are retryable", ErrorCategoryTransient, true},
		{"Network errors are retryable", ErrorCategoryNetwork, true},
		{"Timeout errors are retryable", ErrorCategoryTimeout, true},
		{"Permanent errors are not retryable", ErrorCategoryPermanent, false},
		{"Configuration errors are not retryable", ErrorCategoryConfiguration, false},
		{"Validation errors are not retryable", ErrorCategoryValidation, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compErr := CreateComponentError(
				tt.category,
				ErrorSeverityMedium,
				"test_code",
				"test message",
				"test-component",
			)

			if compErr.IsRetryable() != tt.expected {
				t.Errorf("Expected IsRetryable() to return %v for category %s", tt.expected, tt.category)
			}
		})
	}
}

func TestComponentError_ShouldCircuitBreak(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		category ErrorCategory
		expected bool
	}{
		{"High severity should circuit break", ErrorSeverityHigh, ErrorCategoryInternal, true},
		{"Critical severity should circuit break", ErrorSeverityCritical, ErrorCategoryInternal, true},
		{"Resource category should circuit break", ErrorSeverityMedium, ErrorCategoryResource, true},
		{"Low severity should not circuit break", ErrorSeverityLow, ErrorCategoryInternal, false},
		{"Medium severity non-resource should not circuit break", ErrorSeverityMedium, ErrorCategoryValidation, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compErr := CreateComponentError(
				tt.category,
				tt.severity,
				"test_code",
				"test message",
				"test-component",
			)

			if compErr.ShouldCircuitBreak() != tt.expected {
				t.Errorf("Expected ShouldCircuitBreak() to return %v for severity %s and category %s", 
					tt.expected, tt.severity, tt.category)
			}
		})
	}
}

func TestErrorHandler_CategorizeError(t *testing.T) {
	eh := NewErrorHandler()

	tests := []struct {
		name             string
		errorMessage     string
		expectedCategory ErrorCategory
		expectedSeverity ErrorSeverity
		expectedCode     string
	}{
		{
			name:             "Timeout error",
			errorMessage:     "operation timeout after 5s",
			expectedCategory: ErrorCategoryTimeout,
			expectedSeverity: ErrorSeverityMedium,
			expectedCode:     "timeout_error",
		},
		{
			name:             "Network error",
			errorMessage:     "connection refused",
			expectedCategory: ErrorCategoryNetwork,
			expectedSeverity: ErrorSeverityMedium,
			expectedCode:     "network_error",
		},
		{
			name:             "Resource error",
			errorMessage:     "system overloaded",
			expectedCategory: ErrorCategoryResource,
			expectedSeverity: ErrorSeverityHigh,
			expectedCode:     "resource_error",
		},
		{
			name:             "Configuration error",
			errorMessage:     "invalid configuration",
			expectedCategory: ErrorCategoryConfiguration,
			expectedSeverity: ErrorSeverityHigh,
			expectedCode:     "configuration_error",
		},
		{
			name:             "Validation error",
			errorMessage:     "validation failed",
			expectedCategory: ErrorCategoryValidation,
			expectedSeverity: ErrorSeverityLow,
			expectedCode:     "validation_error",
		},
		{
			name:             "Critical error",
			errorMessage:     "panic occurred",
			expectedCategory: ErrorCategoryInternal,
			expectedSeverity: ErrorSeverityCritical,
			expectedCode:     "critical_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMessage)
			compErr := eh.categorizeError(err, "test-component")

			if compErr.Category != tt.expectedCategory {
				t.Errorf("Expected category %s, got %s", tt.expectedCategory, compErr.Category)
			}

			if compErr.Severity != tt.expectedSeverity {
				t.Errorf("Expected severity %s, got %s", tt.expectedSeverity, compErr.Severity)
			}

			if compErr.Code != tt.expectedCode {
				t.Errorf("Expected code %s, got %s", tt.expectedCode, compErr.Code)
			}

			if compErr.ComponentID != "test-component" {
				t.Errorf("Expected component ID 'test-component', got %s", compErr.ComponentID)
			}

			if compErr.OriginalError != err {
				t.Error("Expected original error to be preserved")
			}
		})
	}
}

func TestErrorHandler_DetermineRecoveryStrategy(t *testing.T) {
	eh := NewErrorHandler()

	tests := []struct {
		name             string
		category         ErrorCategory
		severity         ErrorSeverity
		retryCount       int
		maxRetries       int
		expectedStrategy RecoveryStrategy
	}{
		{
			name:             "Transient error should retry",
			category:         ErrorCategoryTransient,
			severity:         ErrorSeverityMedium,
			retryCount:       0,
			maxRetries:       3,
			expectedStrategy: RecoveryStrategyRetry,
		},
		{
			name:             "Transient error at max retries should fallback",
			category:         ErrorCategoryTransient,
			severity:         ErrorSeverityMedium,
			retryCount:       3,
			maxRetries:       3,
			expectedStrategy: RecoveryStrategyFallback,
		},
		{
			name:             "Critical error should circuit break",
			category:         ErrorCategoryInternal,
			severity:         ErrorSeverityCritical,
			retryCount:       0,
			maxRetries:       3,
			expectedStrategy: RecoveryStrategyCircuit,
		},
		{
			name:             "Permanent error should fail",
			category:         ErrorCategoryPermanent,
			severity:         ErrorSeverityMedium,
			retryCount:       0,
			maxRetries:       3,
			expectedStrategy: RecoveryStrategyFail,
		},
		{
			name:             "Resource error should circuit break",
			category:         ErrorCategoryResource,
			severity:         ErrorSeverityMedium,
			retryCount:       0,
			maxRetries:       3,
			expectedStrategy: RecoveryStrategyCircuit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compErr := CreateComponentError(
				tt.category,
				tt.severity,
				"test_code",
				"test message",
				"test-component",
			)
			compErr.RetryCount = tt.retryCount
			compErr.MaxRetries = tt.maxRetries

			strategy := eh.determineRecoveryStrategy(compErr)

			if strategy != tt.expectedStrategy {
				t.Errorf("Expected strategy %s, got %s", tt.expectedStrategy, strategy)
			}
		})
	}
}

func TestErrorHandler_HandleError(t *testing.T) {
	eh := NewErrorHandler()
	ctx := context.Background()

	// Test error callback
	var callbackCalled bool
	var callbackError *ComponentError
	eh.SetErrorCallback(func(ce *ComponentError) {
		callbackCalled = true
		callbackError = ce
	})

	// Handle a test error
	originalErr := errors.New("test timeout error")
	compErr := eh.HandleError(ctx, originalErr, "test-component")

	// Verify error was processed correctly
	if compErr == nil {
		t.Fatal("Expected ComponentError to be returned")
	}

	if compErr.Category != ErrorCategoryTimeout {
		t.Errorf("Expected category %s, got %s", ErrorCategoryTimeout, compErr.Category)
	}

	if compErr.ComponentID != "test-component" {
		t.Errorf("Expected component ID 'test-component', got %s", compErr.ComponentID)
	}

	if compErr.StackTrace == "" {
		t.Error("Expected stack trace to be populated")
	}

	// Verify callback was called
	if !callbackCalled {
		t.Error("Expected error callback to be called")
	}

	if callbackError != compErr {
		t.Error("Expected callback to receive the same error")
	}

	// Verify error tracking
	stats := eh.GetErrorStats()
	if stats["total_errors"].(int) != 1 {
		t.Errorf("Expected 1 total error, got %d", stats["total_errors"].(int))
	}
}

func TestWrapError(t *testing.T) {
	// Test wrapping nil error
	wrapped := WrapError(nil, "test-component", "test-op")
	if wrapped != nil {
		t.Error("Expected nil when wrapping nil error")
	}

	// Test wrapping generic error
	originalErr := errors.New("generic error")
	wrapped = WrapError(originalErr, "test-component", "test-op")

	if wrapped == nil {
		t.Fatal("Expected ComponentError to be returned")
	}

	if wrapped.ComponentID != "test-component" {
		t.Errorf("Expected component ID 'test-component', got %s", wrapped.ComponentID)
	}

	if wrapped.OperationID != "test-op" {
		t.Errorf("Expected operation ID 'test-op', got %s", wrapped.OperationID)
	}

	if wrapped.OriginalError != originalErr {
		t.Error("Expected original error to be preserved")
	}

	// Test wrapping existing ComponentError
	existingErr := CreateComponentError(
		ErrorCategoryTransient,
		ErrorSeverityLow,
		"existing_error",
		"existing message",
		"",
	)

	wrapped = WrapError(existingErr, "test-component", "test-op")

	if wrapped != existingErr {
		t.Error("Expected same ComponentError instance to be returned")
	}

	if wrapped.ComponentID != "test-component" {
		t.Errorf("Expected component ID to be updated to 'test-component', got %s", wrapped.ComponentID)
	}

	if wrapped.OperationID != "test-op" {
		t.Errorf("Expected operation ID to be updated to 'test-op', got %s", wrapped.OperationID)
	}
}
