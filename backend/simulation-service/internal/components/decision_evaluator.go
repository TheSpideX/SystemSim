package components

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/systemsim/simulation-service/internal/engines"
)

// DecisionEvaluator handles complex decision logic for routing within components
type DecisionEvaluator struct {
	// Available engines for validation
	availableEngines map[engines.EngineType]bool
	
	// Component context for decision making
	componentID   string
	componentType ComponentType
}

// NewDecisionEvaluator creates a new decision evaluator
func NewDecisionEvaluator(componentID string, componentType ComponentType, availableEngines []engines.EngineType) *DecisionEvaluator {
	engineMap := make(map[engines.EngineType]bool)
	for _, engineType := range availableEngines {
		engineMap[engineType] = true
	}

	return &DecisionEvaluator{
		availableEngines: engineMap,
		componentID:      componentID,
		componentType:    componentType,
	}
}

// EvaluateDecisionNode evaluates a decision node and returns the next node ID
func (de *DecisionEvaluator) EvaluateDecisionNode(
	node *DecisionNode,
	operation *engines.Operation,
	previousResult *engines.OperationResult,
) (string, error) {
	
	log.Printf("DecisionEvaluator %s: Evaluating decision node %s", de.componentID, node.ID)

	// If node has routing logic, use enhanced evaluation
	// Note: This would need to be implemented when we have proper routing logic structure
	// For now, fall back to simple condition evaluation

	// Fall back to simple condition evaluation
	return de.evaluateSimpleConditions(node.Conditions, operation, previousResult)
}

// evaluateRoutingLogic evaluates complex routing logic expressions
func (de *DecisionEvaluator) evaluateRoutingLogic(
	routingLogic map[string]interface{},
	operation *engines.Operation,
	result *engines.OperationResult,
) (string, error) {
	
	// Evaluate each condition in priority order
	conditionPriority := []string{
		"cache_hit", "cache_miss", "cache_lookup", "database_query",
		"static_content", "dynamic_content", "error_occurred", 
		"high_priority", "large_data", "small_data", "default",
	}

	for _, conditionName := range conditionPriority {
		if expression, exists := routingLogic[conditionName]; exists {
			if expressionStr, ok := expression.(string); ok {
				if de.evaluateExpression(expressionStr, operation, result) {
					log.Printf("DecisionEvaluator %s: Condition '%s' matched: %s", 
						de.componentID, conditionName, expressionStr)
					return conditionName, nil
				}
			}
		}
	}

	// No conditions matched
	return "default", nil
}

// evaluateExpression evaluates a routing expression
func (de *DecisionEvaluator) evaluateExpression(
	expression string,
	operation *engines.Operation,
	result *engines.OperationResult,
) bool {
	
	// Handle simple boolean expressions
	expression = strings.TrimSpace(expression)
	
	// Split by logical operators
	if strings.Contains(expression, " && ") {
		parts := strings.Split(expression, " && ")
		for _, part := range parts {
			if !de.evaluateSingleCondition(strings.TrimSpace(part), operation, result) {
				return false
			}
		}
		return true
	}
	
	if strings.Contains(expression, " || ") {
		parts := strings.Split(expression, " || ")
		for _, part := range parts {
			if de.evaluateSingleCondition(strings.TrimSpace(part), operation, result) {
				return true
			}
		}
		return false
	}
	
	// Single condition
	return de.evaluateSingleCondition(expression, operation, result)
}

// evaluateSingleCondition evaluates a single condition
func (de *DecisionEvaluator) evaluateSingleCondition(
	condition string,
	operation *engines.Operation,
	result *engines.OperationResult,
) bool {
	
	// Parse condition: "variable operator value"
	parts := de.parseCondition(condition)
	if len(parts) != 3 {
		log.Printf("DecisionEvaluator %s: Invalid condition format: %s", de.componentID, condition)
		return false
	}
	
	variable := parts[0]
	operator := parts[1]
	expectedValue := parts[2]
	
	// Get actual value
	actualValue := de.getVariableValue(variable, operation, result)
	
	// Compare values
	return de.compareValues(actualValue, operator, expectedValue)
}

// parseCondition parses a condition string into [variable, operator, value]
func (de *DecisionEvaluator) parseCondition(condition string) []string {
	// Handle different operators
	operators := []string{"===", "!==", ">=", "<=", ">", "<", "==", "!="}
	
	for _, op := range operators {
		if strings.Contains(condition, " "+op+" ") {
			parts := strings.Split(condition, " "+op+" ")
			if len(parts) == 2 {
				return []string{strings.TrimSpace(parts[0]), op, strings.TrimSpace(parts[1])}
			}
		}
	}
	
	return []string{}
}

// getVariableValue gets the value of a variable from operation or result
func (de *DecisionEvaluator) getVariableValue(
	variable string,
	operation *engines.Operation,
	result *engines.OperationResult,
) interface{} {
	
	// Handle nested properties with dot notation
	parts := strings.Split(variable, ".")
	
	switch parts[0] {
	case "operation":
		return de.getOperationValue(parts[1:], operation)
	case "result":
		return de.getResultValue(parts[1:], result)
	case "metrics":
		if result != nil && result.Metrics != nil {
			return de.getNestedValue(parts[1:], result.Metrics)
		}
	}
	
	return nil
}

// getOperationValue gets a value from the operation object
func (de *DecisionEvaluator) getOperationValue(path []string, operation *engines.Operation) interface{} {
	if len(path) == 0 || operation == nil {
		return nil
	}
	
	switch path[0] {
	case "type":
		return operation.Type
	case "data_size":
		return operation.DataSize
	case "priority":
		return operation.Priority
	case "complexity":
		return operation.Complexity
	case "metadata":
		if len(path) > 1 && operation.Metadata != nil {
			return de.getNestedValue(path[1:], operation.Metadata)
		}
		return operation.Metadata
	}
	
	return nil
}

// getResultValue gets a value from the result object
func (de *DecisionEvaluator) getResultValue(path []string, result *engines.OperationResult) interface{} {
	if len(path) == 0 || result == nil {
		return nil
	}
	
	switch path[0] {
	case "success":
		return result.Success
	case "processing_time":
		return result.ProcessingTime.Milliseconds()
	case "metrics":
		if len(path) > 1 && result.Metrics != nil {
			return de.getNestedValue(path[1:], result.Metrics)
		}
		return result.Metrics
	}
	
	return nil
}

// getNestedValue gets a nested value from a map
func (de *DecisionEvaluator) getNestedValue(path []string, data map[string]interface{}) interface{} {
	if len(path) == 0 || data == nil {
		return nil
	}
	
	value, exists := data[path[0]]
	if !exists {
		return nil
	}
	
	if len(path) == 1 {
		return value
	}
	
	// Continue nested lookup
	if nestedMap, ok := value.(map[string]interface{}); ok {
		return de.getNestedValue(path[1:], nestedMap)
	}
	
	return nil
}

// compareValues compares two values using the given operator
func (de *DecisionEvaluator) compareValues(actual interface{}, operator string, expected string) bool {
	if actual == nil {
		return false
	}
	
	switch operator {
	case "===", "==":
		return de.valuesEqual(actual, expected)
	case "!==", "!=":
		return !de.valuesEqual(actual, expected)
	case ">":
		return de.compareNumeric(actual, expected, func(a, b float64) bool { return a > b })
	case ">=":
		return de.compareNumeric(actual, expected, func(a, b float64) bool { return a >= b })
	case "<":
		return de.compareNumeric(actual, expected, func(a, b float64) bool { return a < b })
	case "<=":
		return de.compareNumeric(actual, expected, func(a, b float64) bool { return a <= b })
	}
	
	return false
}

// valuesEqual checks if two values are equal
func (de *DecisionEvaluator) valuesEqual(actual interface{}, expected string) bool {
	// Handle different types
	switch v := actual.(type) {
	case string:
		return v == strings.Trim(expected, "\"'")
	case bool:
		expectedBool, err := strconv.ParseBool(expected)
		return err == nil && v == expectedBool
	case int, int64:
		expectedInt, err := strconv.ParseInt(expected, 10, 64)
		if err != nil {
			return false
		}
		if intVal, ok := actual.(int); ok {
			return int64(intVal) == expectedInt
		}
		if int64Val, ok := actual.(int64); ok {
			return int64Val == expectedInt
		}
	case float64:
		expectedFloat, err := strconv.ParseFloat(expected, 64)
		return err == nil && v == expectedFloat
	}
	
	return fmt.Sprintf("%v", actual) == expected
}

// compareNumeric compares numeric values
func (de *DecisionEvaluator) compareNumeric(actual interface{}, expected string, compareFn func(float64, float64) bool) bool {
	expectedFloat, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return false
	}
	
	var actualFloat float64
	switch v := actual.(type) {
	case int:
		actualFloat = float64(v)
	case int64:
		actualFloat = float64(v)
	case float64:
		actualFloat = v
	default:
		return false
	}
	
	return compareFn(actualFloat, expectedFloat)
}

// evaluateSimpleConditions evaluates simple string-based conditions (fallback)
func (de *DecisionEvaluator) evaluateSimpleConditions(
	conditions map[string]string,
	operation *engines.Operation,
	result *engines.OperationResult,
) (string, error) {
	
	// Check operation type
	if nextNode, exists := conditions[operation.Type]; exists {
		return nextNode, nil
	}
	
	// Check result success/failure
	if result != nil {
		if result.Success {
			if nextNode, exists := conditions["success"]; exists {
				return nextNode, nil
			}
		} else {
			if nextNode, exists := conditions["failure"]; exists {
				return nextNode, nil
			}
		}
	}
	
	// Check default
	if nextNode, exists := conditions["default"]; exists {
		return nextNode, nil
	}
	
	return "", fmt.Errorf("no matching condition found")
}
