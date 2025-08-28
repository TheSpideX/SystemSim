package components

import (
	"testing"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func TestCentralizedOutputManager_EvaluateCondition(t *testing.T) {
	// Create a centralized output manager for testing
	com := &CentralizedOutputManager{
		InstanceID:    "test-instance",
		ComponentID:   "test-component",
		DefaultRouting: "default-target",
	}

	tests := []struct {
		name          string
		conditionName string
		result        *engines.OperationResult
		expected      bool
	}{
		{
			name:          "Success condition - true",
			conditionName: "success",
			result: &engines.OperationResult{
				Success: true,
			},
			expected: true,
		},
		{
			name:          "Success condition - false",
			conditionName: "success",
			result: &engines.OperationResult{
				Success: false,
			},
			expected: false,
		},
		{
			name:          "Failure condition - true",
			conditionName: "failure",
			result: &engines.OperationResult{
				Success: false,
			},
			expected: true,
		},
		{
			name:          "Cache hit condition - true",
			conditionName: "cache_hit",
			result: &engines.OperationResult{
				Success: true,
				Metrics: map[string]interface{}{
					"cache_hit": true,
				},
			},
			expected: true,
		},
		{
			name:          "Cache hit condition - false",
			conditionName: "cache_hit",
			result: &engines.OperationResult{
				Success: true,
				Metrics: map[string]interface{}{
					"cache_hit": false,
				},
			},
			expected: false,
		},
		{
			name:          "Cache miss condition - true",
			conditionName: "cache_miss",
			result: &engines.OperationResult{
				Success: true,
				Metrics: map[string]interface{}{
					"cache_hit": false,
				},
			},
			expected: true,
		},
		{
			name:          "High priority condition - true",
			conditionName: "high_priority",
			result: &engines.OperationResult{
				Success: true,
				Metrics: map[string]interface{}{
					"priority": 8,
				},
			},
			expected: true,
		},
		{
			name:          "High priority condition - false",
			conditionName: "high_priority",
			result: &engines.OperationResult{
				Success: true,
				Metrics: map[string]interface{}{
					"priority": 5,
				},
			},
			expected: false,
		},
		{
			name:          "Large data condition - true",
			conditionName: "large_data",
			result: &engines.OperationResult{
				Success: true,
				Metrics: map[string]interface{}{
					"data_size": int64(2000000), // 2MB
				},
			},
			expected: true,
		},
		{
			name:          "Small data condition - true",
			conditionName: "small_data",
			result: &engines.OperationResult{
				Success: true,
				Metrics: map[string]interface{}{
					"data_size": int64(32000), // 32KB
				},
			},
			expected: true,
		},
		{
			name:          "Timeout condition - true",
			conditionName: "timeout",
			result: &engines.OperationResult{
				Success:        true,
				ProcessingTime: time.Second * 2, // 2 seconds
			},
			expected: true,
		},
		{
			name:          "Timeout condition - false",
			conditionName: "timeout",
			result: &engines.OperationResult{
				Success:        true,
				ProcessingTime: time.Millisecond * 500, // 0.5 seconds
			},
			expected: false,
		},
		{
			name:          "Database query condition - true",
			conditionName: "database_query",
			result: &engines.OperationResult{
				Success:       true,
				OperationType: "read_request",
			},
			expected: true,
		},
		{
			name:          "Unknown condition - false",
			conditionName: "unknown_condition",
			result: &engines.OperationResult{
				Success: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := com.evaluateCondition(tt.conditionName, tt.result)
			if result != tt.expected {
				t.Errorf("evaluateCondition(%s) = %v, expected %v", tt.conditionName, result, tt.expected)
			}
		})
	}
}

func TestCentralizedOutputManager_EvaluateStepConditions(t *testing.T) {
	// Create a centralized output manager for testing
	com := &CentralizedOutputManager{
		InstanceID:    "test-instance",
		ComponentID:   "test-component",
		DefaultRouting: "default-target",
	}

	tests := []struct {
		name     string
		step     *UserFlowStep
		result   *engines.OperationResult
		expected string
	}{
		{
			name: "Success condition matches",
			step: &UserFlowStep{
				ComponentID: "test-component",
				Operation:   "test_operation",
				Conditions: map[string]string{
					"success": "success-target",
					"failure": "failure-target",
					"default": "default-target",
				},
			},
			result: &engines.OperationResult{
				Success:       true,
				OperationType: "test_operation",
			},
			expected: "success-target",
		},
		{
			name: "Failure condition matches",
			step: &UserFlowStep{
				ComponentID: "test-component",
				Operation:   "test_operation",
				Conditions: map[string]string{
					"success": "success-target",
					"failure": "failure-target",
					"default": "default-target",
				},
			},
			result: &engines.OperationResult{
				Success:       false,
				OperationType: "test_operation",
			},
			expected: "failure-target",
		},
		{
			name: "Cache hit condition matches",
			step: &UserFlowStep{
				ComponentID: "test-component",
				Operation:   "cache_operation",
				Conditions: map[string]string{
					"cache_hit":  "cache-target",
					"cache_miss": "database-target",
					"default":    "default-target",
				},
			},
			result: &engines.OperationResult{
				Success:       true,
				OperationType: "cache_operation",
				Metrics: map[string]interface{}{
					"cache_hit": true,
				},
			},
			expected: "cache-target",
		},
		{
			name: "Operation type matches",
			step: &UserFlowStep{
				ComponentID: "test-component",
				Operation:   "test_operation",
				Conditions: map[string]string{
					"read_request": "read-target",
					"default":      "default-target",
				},
			},
			result: &engines.OperationResult{
				Success:       true,
				OperationType: "read_request",
			},
			expected: "read-target",
		},
		{
			name: "Default condition used",
			step: &UserFlowStep{
				ComponentID: "test-component",
				Operation:   "test_operation",
				Conditions: map[string]string{
					"cache_hit": "cache-target",
					"default":   "default-target",
				},
			},
			result: &engines.OperationResult{
				Success:       true,
				OperationType: "test_operation",
				Metrics: map[string]interface{}{
					"cache_hit": false,
				},
			},
			expected: "default-target",
		},
		{
			name: "No matching condition",
			step: &UserFlowStep{
				ComponentID: "test-component",
				Operation:   "test_operation",
				Conditions: map[string]string{
					"cache_hit": "cache-target",
				},
			},
			result: &engines.OperationResult{
				Success:       true,
				OperationType: "test_operation",
				Metrics: map[string]interface{}{
					"cache_hit": false,
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := com.evaluateStepConditions(tt.step, tt.result)
			if result != tt.expected {
				t.Errorf("evaluateStepConditions() = %s, expected %s", result, tt.expected)
			}
		})
	}
}
