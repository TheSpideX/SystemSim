package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"` // healthy, unhealthy, degraded
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
}

// HealthMessage represents a WebSocket health message
type HealthMessage struct {
	Type    string      `json:"type"`
	Channel string      `json:"channel,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// HealthManager manages health status subscriptions and broadcasting
type HealthManager struct {
	mu            sync.RWMutex
	subscriptions map[*websocket.Conn]map[string]bool // conn -> channels
	healthStatus  map[string]HealthStatus             // service -> status
}

// NewHealthManager creates a new health manager
func NewHealthManager() *HealthManager {
	return &HealthManager{
		subscriptions: make(map[*websocket.Conn]map[string]bool),
		healthStatus:  make(map[string]HealthStatus),
	}
}

// Subscribe adds a connection to a health channel
func (hm *HealthManager) Subscribe(conn *websocket.Conn, channel string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.subscriptions[conn] == nil {
		hm.subscriptions[conn] = make(map[string]bool)
	}
	hm.subscriptions[conn][channel] = true

	log.Printf("WebSocket subscribed to channel: %s", channel)

	// Send current health status if available
	// Extract service name from channel (e.g., "health:auth" -> "auth")
	if len(channel) > 7 && channel[:7] == "health:" {
		serviceName := channel[7:] // Remove "health:" prefix
		if status, exists := hm.healthStatus[serviceName]; exists {
			hm.sendToConnection(conn, HealthMessage{
				Type:    "health_update",
				Channel: channel,
				Data: map[string]interface{}{
					"service":   serviceName,
					"status":    status.Status,
					"timestamp": status.Timestamp,
					"details":   status.Details,
				},
			})
			log.Printf("Sent current health status to new subscriber: %s = %s", serviceName, status.Status)
		}
	}
}

// Unsubscribe removes a connection from a health channel
func (hm *HealthManager) Unsubscribe(conn *websocket.Conn, channel string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if channels, exists := hm.subscriptions[conn]; exists {
		delete(channels, channel)
		if len(channels) == 0 {
			delete(hm.subscriptions, conn)
		}
	}

	log.Printf("WebSocket unsubscribed from channel: %s", channel)
}

// RemoveConnection removes all subscriptions for a connection
func (hm *HealthManager) RemoveConnection(conn *websocket.Conn) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.subscriptions, conn)
	log.Printf("WebSocket connection removed from health manager")
}

// UpdateHealth updates the health status and broadcasts to subscribers
func (hm *HealthManager) UpdateHealth(service string, status HealthStatus) {
	hm.mu.Lock()
	hm.healthStatus[service] = status
	hm.mu.Unlock()

	// Broadcast to subscribers
	channel := "health:" + service
	hm.BroadcastToChannel(channel, HealthMessage{
		Type:    "health_update",
		Channel: channel,
		Data: map[string]interface{}{
			"service": service,
			"status":  status.Status,
			"timestamp": status.Timestamp,
			"details": status.Details,
		},
	})

	log.Printf("Health status updated for %s: %s", service, status.Status)
}

// BroadcastToChannel sends a message to all subscribers of a channel
func (hm *HealthManager) BroadcastToChannel(channel string, message HealthMessage) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	for conn, channels := range hm.subscriptions {
		if channels[channel] {
			hm.sendToConnection(conn, message)
		}
	}
}

// sendToConnection sends a message to a specific connection
func (hm *HealthManager) sendToConnection(conn *websocket.Conn, message HealthMessage) {
	if err := conn.WriteJSON(message); err != nil {
		log.Printf("Failed to send health message to WebSocket: %v", err)
		// Connection is likely dead, remove it
		hm.RemoveConnection(conn)
	}
}

// GetHealthStatus returns the current health status for a service
func (hm *HealthManager) GetHealthStatus(service string) (HealthStatus, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	status, exists := hm.healthStatus[service]
	return status, exists
}

// HandleWebSocketMessage processes incoming WebSocket messages
func (hm *HealthManager) HandleWebSocketMessage(conn *websocket.Conn, messageData []byte) error {
	var message HealthMessage
	if err := json.Unmarshal(messageData, &message); err != nil {
		return err
	}

	switch message.Type {
	case "subscribe":
		if message.Channel != "" {
			hm.Subscribe(conn, message.Channel)
		}
	case "unsubscribe":
		if message.Channel != "" {
			hm.Unsubscribe(conn, message.Channel)
		}
	}

	return nil
}
