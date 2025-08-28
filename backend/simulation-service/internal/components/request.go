package components

import (
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

// Request represents a request flowing through the system with shared references
type Request struct {
	// Core identification
	ID string `json:"id"`

	// Shared data (pointer for automatic sharing across flows)
	Data *RequestData `json:"data"`

	// Flow chaining (much simpler than complex sub-flows)
	FlowChain *FlowChain `json:"flow_chain"`

	// Optional tracking (configurable per request)
	TrackHistory bool                  `json:"track_history"`
	History      []RequestHistoryEntry `json:"history,omitempty"`

	// Simple counters (lightweight when tracking disabled)
	ComponentCount int       `json:"component_count"`
	EngineCount    int       `json:"engine_count"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Status         RequestStatus `json:"status"`

	// Current position tracking
	CurrentComponent string `json:"current_component"`
	CurrentEngine    string `json:"current_engine"`
	CurrentNode      string `json:"current_node"`
}

// RequestData contains the actual request data shared across flows
type RequestData struct {
	// Core request data
	UserID    string      `json:"user_id"`
	ProductID string      `json:"product_id"`
	Operation string      `json:"operation"`
	Payload   interface{} `json:"payload"`

	// Flow results (automatically shared via pointer)
	AuthResult      *AuthResult      `json:"auth_result,omitempty"`
	InventoryResult *InventoryResult `json:"inventory_result,omitempty"`
	PaymentResult   *PaymentResult   `json:"payment_result,omitempty"`
}

// FlowChain manages the sequence of flows for a request
type FlowChain struct {
	Flows        []string               `json:"flows"`         // ["auth_flow", "purchase_flow"]
	CurrentIndex int                    `json:"current_index"` // 0, 1, 2...
	Results      map[string]interface{} `json:"results"`       // Shared results
}

// RequestHistoryEntry tracks request journey for educational purposes
type RequestHistoryEntry struct {
	ComponentID string      `json:"component_id"`
	EngineType  string      `json:"engine_type"`
	Operation   string      `json:"operation"`
	Timestamp   time.Time   `json:"timestamp"`
	Result      interface{} `json:"result"`
}

// RequestStatus represents the current status of a request
type RequestStatus int

const (
	RequestStatusActive RequestStatus = iota
	RequestStatusWaitingForSubFlow
	RequestStatusCompleted
	RequestStatusFailed
)

// Flow result types
type AuthResult struct {
	IsAuthenticated bool   `json:"is_authenticated"`
	UserID          string `json:"user_id"`
	Token           string `json:"token"`
	Permissions     []string `json:"permissions"`
}

type InventoryResult struct {
	InStock   bool    `json:"in_stock"`
	Quantity  int     `json:"quantity"`
	Reserved  bool    `json:"reserved"`
	Price     float64 `json:"price"`
}

type PaymentResult struct {
	Processed     bool   `json:"processed"`
	TransactionID string `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Status        string `json:"status"`
}

// NewRequest creates a new request with shared references
func NewRequest(id, userID, operation string, trackHistory bool) *Request {
	return &Request{
		ID: id,
		Data: &RequestData{
			UserID:    userID,
			Operation: operation,
		},
		FlowChain: &FlowChain{
			Flows:        []string{},
			CurrentIndex: 0,
			Results:      make(map[string]interface{}),
		},
		TrackHistory:   trackHistory,
		History:        []RequestHistoryEntry{},
		ComponentCount: 0,
		EngineCount:    0,
		StartTime:      time.Now(),
		Status:         RequestStatusActive,
	}
}

// AddToHistory adds an entry to the request history (only if tracking enabled)
func (r *Request) AddToHistory(componentID, engineType, operation string, result interface{}) {
	if !r.TrackHistory {
		return
	}

	r.History = append(r.History, RequestHistoryEntry{
		ComponentID: componentID,
		EngineType:  engineType,
		Operation:   operation,
		Timestamp:   time.Now(),
		Result:      result,
	})
}

// IncrementEngineCount increments the engine counter
func (r *Request) IncrementEngineCount() {
	r.EngineCount++
}

// IncrementComponentCount increments the component counter
func (r *Request) IncrementComponentCount() {
	r.ComponentCount++
}

// SetCurrentPosition sets the current position in the system
func (r *Request) SetCurrentPosition(component, engine, node string) {
	r.CurrentComponent = component
	r.CurrentEngine = engine
	r.CurrentNode = node
}

// GetCurrentFlow returns the current flow in the chain
func (r *Request) GetCurrentFlow() string {
	if r.FlowChain.CurrentIndex >= len(r.FlowChain.Flows) {
		return ""
	}
	return r.FlowChain.Flows[r.FlowChain.CurrentIndex]
}

// MoveToNextFlow moves to the next flow in the chain
func (r *Request) MoveToNextFlow() bool {
	r.FlowChain.CurrentIndex++
	return r.FlowChain.CurrentIndex < len(r.FlowChain.Flows)
}

// IsFlowComplete checks if all flows are complete
func (r *Request) IsFlowComplete() bool {
	return r.FlowChain.CurrentIndex >= len(r.FlowChain.Flows)
}

// SetFlowResult sets a result for a specific flow
func (r *Request) SetFlowResult(flowName string, result interface{}) {
	r.FlowChain.Results[flowName] = result
}

// GetFlowResult gets a result for a specific flow
func (r *Request) GetFlowResult(flowName string) interface{} {
	return r.FlowChain.Results[flowName]
}

// MarkComplete marks the request as complete
func (r *Request) MarkComplete() {
	r.Status = RequestStatusCompleted
	r.EndTime = time.Now()
}

// MarkFailed marks the request as failed
func (r *Request) MarkFailed() {
	r.Status = RequestStatusFailed
	r.EndTime = time.Now()
}

// GetTotalLatency returns the total request latency
func (r *Request) GetTotalLatency() time.Duration {
	if r.EndTime.IsZero() {
		return time.Since(r.StartTime)
	}
	return r.EndTime.Sub(r.StartTime)
}

// NewRequestWithFlowChain creates a new request with predefined flow chain
func NewRequestWithFlowChain(id, userID, operation string, flows []string, trackHistory bool) *Request {
	return &Request{
		ID: id,
		Data: &RequestData{
			UserID:    userID,
			Operation: operation,
		},
		FlowChain: &FlowChain{
			Flows:        flows,
			CurrentIndex: 0,
			Results:      make(map[string]interface{}),
		},
		TrackHistory:   trackHistory,
		History:        []RequestHistoryEntry{},
		ComponentCount: 0,
		EngineCount:    0,
		StartTime:      time.Now(),
		Status:         RequestStatusActive,
	}
}

// Clone creates a deep copy of the request (for testing purposes)
func (r *Request) Clone() *Request {
	clone := &Request{
		ID:               r.ID,
		TrackHistory:     r.TrackHistory,
		ComponentCount:   r.ComponentCount,
		EngineCount:      r.EngineCount,
		StartTime:        r.StartTime,
		EndTime:          r.EndTime,
		Status:           r.Status,
		CurrentComponent: r.CurrentComponent,
		CurrentEngine:    r.CurrentEngine,
		CurrentNode:      r.CurrentNode,
	}

	// Deep copy RequestData
	if r.Data != nil {
		clone.Data = &RequestData{
			UserID:    r.Data.UserID,
			ProductID: r.Data.ProductID,
			Operation: r.Data.Operation,
			Payload:   r.Data.Payload,
		}
		// Note: Flow results are shared references, so we keep the same pointers
		clone.Data.AuthResult = r.Data.AuthResult
		clone.Data.InventoryResult = r.Data.InventoryResult
		clone.Data.PaymentResult = r.Data.PaymentResult
	}

	// Deep copy FlowChain
	if r.FlowChain != nil {
		clone.FlowChain = &FlowChain{
			CurrentIndex: r.FlowChain.CurrentIndex,
			Flows:        make([]string, len(r.FlowChain.Flows)),
			Results:      make(map[string]interface{}),
		}
		copy(clone.FlowChain.Flows, r.FlowChain.Flows)
		for k, v := range r.FlowChain.Results {
			clone.FlowChain.Results[k] = v
		}
	}

	// Deep copy History
	if r.History != nil {
		clone.History = make([]RequestHistoryEntry, len(r.History))
		copy(clone.History, r.History)
	}

	return clone
}
