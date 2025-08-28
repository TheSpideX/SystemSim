package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fasthttp/websocket"

	"server-service/internal/redis_client"
)

// Hub manages WebSocket connections and message broadcasting
type Hub struct {
	// Connection management
	connections     map[string]*Connection   // connection_id -> connection
	userConnections map[string][]*Connection // user_id -> connections

	// Message channels (high-throughput buffered channels)
	register    chan *Connection
	unregister  chan *Connection
	broadcast   chan *BroadcastMessage
	userMessage chan *UserMessage

	// Redis integration for real-time events
	redisClient *redis_client.Client

	// Health monitoring
	healthManager *HealthManager

	// Performance monitoring
	totalConnections  int64
	activeConnections int64
	messagesProcessed int64
	messagesSent      int64

	// Synchronization
	mutex sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// Connection represents a WebSocket connection
type Connection struct {
	ID     string
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub

	// Connection metadata
	ConnectedAt   time.Time
	LastActivity  time.Time
	Subscriptions map[string]bool // channels this connection subscribes to

	// Performance tracking
	MessagesSent     int64
	MessagesReceived int64

	mutex sync.RWMutex
}

// BroadcastMessage represents a message to broadcast to all connections
type BroadcastMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Channel string      `json:"channel,omitempty"`
}

// UserMessage represents a message to send to specific user(s)
type UserMessage struct {
	UserID  string      `json:"user_id"`
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Channel string      `json:"channel,omitempty"`
}

// NewHub creates a new WebSocket hub with Redis integration
func NewHub(redisClient *redis_client.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		connections:     make(map[string]*Connection),
		userConnections: make(map[string][]*Connection),
		register:        make(chan *Connection, 1000),
		unregister:      make(chan *Connection, 1000),
		broadcast:       make(chan *BroadcastMessage, 10000), // High-throughput buffer
		userMessage:     make(chan *UserMessage, 10000),
		redisClient:     redisClient,
		healthManager:   NewHealthManager(),
		ctx:             ctx,
		cancel:          cancel,
	}

	return hub
}

// GetHealthManager returns the health manager instance
func (h *Hub) GetHealthManager() *HealthManager {
	return h.healthManager
}

// Run starts the WebSocket hub
func (h *Hub) Run() {
	log.Println("Starting WebSocket hub...")

	// Start Redis event processors only if Redis client is available
	if h.redisClient != nil {
		go h.processAuthEvents()
		go h.processProjectEvents()
		go h.processSimulationEvents()
		go h.processSimulationData()
		go h.processNotificationEvents()
		go h.processCollaborationEvents()
	} else {
		log.Println("Redis client not available, real-time events disabled")
	}

	// Start connection cleanup routine
	go h.cleanupConnections()

	// Main hub loop
	for {
		select {
		case conn := <-h.register:
			h.registerConnection(conn)

		case conn := <-h.unregister:
			h.unregisterConnection(conn)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case userMsg := <-h.userMessage:
			h.sendUserMessage(userMsg)

		case <-h.ctx.Done():
			log.Println("WebSocket hub shutting down...")
			return
		}
	}
}

// RegisterConnection registers a new WebSocket connection
func (h *Hub) RegisterConnection(conn *Connection) {
	select {
	case h.register <- conn:
	default:
		log.Printf("Register channel full, dropping connection %s", conn.ID)
		conn.Close()
	}
}

// UnregisterConnection unregisters a WebSocket connection
func (h *Hub) UnregisterConnection(conn *Connection) {
	select {
	case h.unregister <- conn:
	default:
		log.Printf("Unregister channel full for connection %s", conn.ID)
	}
}

// registerConnection handles connection registration
func (h *Hub) registerConnection(conn *Connection) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Add to connections map
	h.connections[conn.ID] = conn

	// Add to user connections
	if conn.UserID != "" {
		h.userConnections[conn.UserID] = append(h.userConnections[conn.UserID], conn)
	}

	atomic.AddInt64(&h.totalConnections, 1)
	atomic.AddInt64(&h.activeConnections, 1)

	log.Printf("Connection registered: %s (user: %s), total: %d",
		conn.ID, conn.UserID, atomic.LoadInt64(&h.activeConnections))
}

// unregisterConnection handles connection unregistration
func (h *Hub) unregisterConnection(conn *Connection) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Remove from connections map
	if _, exists := h.connections[conn.ID]; exists {
		delete(h.connections, conn.ID)
		close(conn.Send)

		// Remove from user connections
		if conn.UserID != "" {
			userConns := h.userConnections[conn.UserID]
			for i, userConn := range userConns {
				if userConn.ID == conn.ID {
					h.userConnections[conn.UserID] = append(userConns[:i], userConns[i+1:]...)
					break
				}
			}

			// Clean up empty user connection slice
			if len(h.userConnections[conn.UserID]) == 0 {
				delete(h.userConnections, conn.UserID)
			}
		}

		atomic.AddInt64(&h.activeConnections, -1)

		log.Printf("Connection unregistered: %s (user: %s), total: %d",
			conn.ID, conn.UserID, atomic.LoadInt64(&h.activeConnections))
	}
}

// BroadcastMessage broadcasts a message to all connections
func (h *Hub) BroadcastMessage(msgType string, data interface{}, channel string) {
	message := &BroadcastMessage{
		Type:    msgType,
		Data:    data,
		Channel: channel,
	}

	select {
	case h.broadcast <- message:
	default:
		log.Printf("Broadcast channel full, dropping message type: %s", msgType)
	}
}

// SendUserMessage sends a message to a specific user
func (h *Hub) SendUserMessage(userID, msgType string, data interface{}, channel string) {
	message := &UserMessage{
		UserID:  userID,
		Type:    msgType,
		Data:    data,
		Channel: channel,
	}

	select {
	case h.userMessage <- message:
	default:
		log.Printf("User message channel full, dropping message for user: %s", userID)
	}
}

// BroadcastToChannel broadcasts a message to all connections subscribed to a specific channel
func (h *Hub) BroadcastToChannel(channel, msgType string, data interface{}) {
	h.mutex.RLock()
	subscribedConnections := make([]*Connection, 0)

	for _, conn := range h.connections {
		if conn.Subscriptions != nil && conn.Subscriptions[channel] {
			subscribedConnections = append(subscribedConnections, conn)
		}
	}
	h.mutex.RUnlock()

	if len(subscribedConnections) == 0 {
		log.Printf("No connections subscribed to channel: %s", channel)
		return
	}

	// Create message
	message := map[string]interface{}{
		"type":    msgType,
		"channel": channel,
		"data":    data,
	}

	// Serialize message once
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal channel broadcast message: %v", err)
		return
	}

	// Send to all subscribed connections
	for _, conn := range subscribedConnections {
		select {
		case conn.Send <- jsonData:
		default:
			// Connection's send channel is full, close it
			log.Printf("Send channel full for connection %s on channel %s, closing", conn.ID, channel)
			close(conn.Send)
		}
	}

	log.Printf("Broadcasted to channel %s: %d connections", channel, len(subscribedConnections))
}

// broadcastMessage handles broadcasting to all connections
func (h *Hub) broadcastMessage(message *BroadcastMessage) {
	h.mutex.RLock()
	connections := make([]*Connection, 0, len(h.connections))
	for _, conn := range h.connections {
		connections = append(connections, conn)
	}
	h.mutex.RUnlock()

	// Serialize message once
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	// Send to all connections in parallel
	var wg sync.WaitGroup
	for _, conn := range connections {
		wg.Add(1)
		go func(c *Connection) {
			defer wg.Done()
			c.SendMessage(jsonData)
		}(conn)
	}
	wg.Wait()

	atomic.AddInt64(&h.messagesProcessed, 1)
	atomic.AddInt64(&h.messagesSent, int64(len(connections)))
}

// sendUserMessage handles sending message to specific user
func (h *Hub) sendUserMessage(userMsg *UserMessage) {
	h.mutex.RLock()
	userConnections := make([]*Connection, len(h.userConnections[userMsg.UserID]))
	copy(userConnections, h.userConnections[userMsg.UserID])
	h.mutex.RUnlock()

	if len(userConnections) == 0 {
		return // User not connected
	}

	// Serialize message once
	jsonData, err := json.Marshal(userMsg)
	if err != nil {
		log.Printf("Failed to marshal user message: %v", err)
		return
	}

	// Send to all user connections in parallel
	var wg sync.WaitGroup
	for _, conn := range userConnections {
		wg.Add(1)
		go func(c *Connection) {
			defer wg.Done()
			c.SendMessage(jsonData)
		}(conn)
	}
	wg.Wait()

	atomic.AddInt64(&h.messagesProcessed, 1)
	atomic.AddInt64(&h.messagesSent, int64(len(userConnections)))
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mutex.RLock()
	totalUsers := len(h.userConnections)
	h.mutex.RUnlock()

	return map[string]interface{}{
		"total_connections":   atomic.LoadInt64(&h.totalConnections),
		"active_connections":  atomic.LoadInt64(&h.activeConnections),
		"connected_users":     totalUsers,
		"messages_processed":  atomic.LoadInt64(&h.messagesProcessed),
		"messages_sent":       atomic.LoadInt64(&h.messagesSent),
		"register_buffer":     len(h.register),
		"unregister_buffer":   len(h.unregister),
		"broadcast_buffer":    len(h.broadcast),
		"user_message_buffer": len(h.userMessage),
	}
}

// GetConnectionStats returns detailed connection statistics
func (h *Hub) GetConnectionStats() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	connectionStats := make(map[string]interface{})
	for connID, conn := range h.connections {
		conn.mutex.RLock()
		connectionStats[connID] = map[string]interface{}{
			"user_id":           conn.UserID,
			"connected_at":      conn.ConnectedAt,
			"last_activity":     conn.LastActivity,
			"messages_sent":     atomic.LoadInt64(&conn.MessagesSent),
			"messages_received": atomic.LoadInt64(&conn.MessagesReceived),
			"subscriptions":     len(conn.Subscriptions),
		}
		conn.mutex.RUnlock()
	}

	return connectionStats
}

// Close gracefully shuts down the hub
func (h *Hub) Close() error {
	log.Println("Closing WebSocket hub...")

	// Cancel context to stop all goroutines
	h.cancel()

	// Close all connections
	h.mutex.Lock()
	for _, conn := range h.connections {
		conn.Close()
	}
	h.mutex.Unlock()

	log.Printf("WebSocket hub closed. Total connections served: %d",
		atomic.LoadInt64(&h.totalConnections))

	return nil
}

// processAuthEvents processes auth service events from Redis
func (h *Hub) processAuthEvents() {
	if h.redisClient == nil {
		return
	}

	for {
		select {
		case event := <-h.redisClient.GetAuthEvents():
			if event == nil {
				continue
			}

			// Broadcast auth events to all connected clients
			h.BroadcastMessage("auth_event", map[string]interface{}{
				"channel":   event.Channel,
				"payload":   event.Payload,
				"timestamp": event.Timestamp,
			}, event.Channel)

		case <-h.ctx.Done():
			return
		}
	}
}

// processProjectEvents processes project service events from Redis
func (h *Hub) processProjectEvents() {
	if h.redisClient == nil {
		return
	}

	for {
		select {
		case event := <-h.redisClient.GetProjectEvents():
			if event == nil {
				continue
			}

			// Broadcast project events to all connected clients
			h.BroadcastMessage("project_event", map[string]interface{}{
				"channel":   event.Channel,
				"payload":   event.Payload,
				"timestamp": event.Timestamp,
			}, event.Channel)

		case <-h.ctx.Done():
			return
		}
	}
}

// processSimulationEvents processes simulation service events from Redis
func (h *Hub) processSimulationEvents() {
	if h.redisClient == nil {
		return
	}

	for {
		select {
		case event := <-h.redisClient.GetSimulationEvents():
			if event == nil {
				continue
			}

			// Broadcast simulation events to subscribed clients only
			h.BroadcastToChannel(event.Channel, "simulation_event", map[string]interface{}{
				"payload":   event.Payload,
				"timestamp": event.Timestamp,
			})

		case <-h.ctx.Done():
			return
		}
	}
}

// processSimulationData processes high-frequency simulation data from Redis
func (h *Hub) processSimulationData() {
	if h.redisClient == nil {
		return
	}

	for {
		select {
		case event := <-h.redisClient.GetSimulationData():
			if event == nil {
				continue
			}

			// Broadcast simulation data to subscribed clients only
			h.BroadcastToChannel(event.Channel, "simulation_data", map[string]interface{}{
				"payload":   event.Payload,
				"timestamp": event.Timestamp,
			})

		case <-h.ctx.Done():
			return
		}
	}
}

// cleanupConnections periodically cleans up stale connections
func (h *Hub) cleanupConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.performConnectionCleanup()
		case <-h.ctx.Done():
			return
		}
	}
}

// performConnectionCleanup removes stale connections
func (h *Hub) performConnectionCleanup() {
	h.mutex.RLock()
	staleConnections := make([]*Connection, 0)
	cutoff := time.Now().Add(-5 * time.Minute) // 5 minutes of inactivity

	for _, conn := range h.connections {
		conn.mutex.RLock()
		if conn.LastActivity.Before(cutoff) {
			staleConnections = append(staleConnections, conn)
		}
		conn.mutex.RUnlock()
	}
	h.mutex.RUnlock()

	// Close stale connections
	for _, conn := range staleConnections {
		log.Printf("Closing stale connection: %s (last activity: %v)",
			conn.ID, conn.LastActivity)
		h.UnregisterConnection(conn)
	}

	if len(staleConnections) > 0 {
		log.Printf("Cleaned up %d stale connections", len(staleConnections))
	}
}

// processNotificationEvents processes user notification events from Redis
func (h *Hub) processNotificationEvents() {
	if h.redisClient == nil {
		return
	}

	// TODO: Implement Redis notification channel listener
	// For now, simulate periodic notifications
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Simulate notification events
			// In production, this would listen to Redis pub/sub channels like:
			// "notifications:user:*"

			// Example: Send test notification to all users
			h.BroadcastMessage("notification", map[string]interface{}{
				"title":   "System Notification",
				"message": "This is a test notification from WebSocket hub",
				"type":    "info",
			}, "system:announcements")

		case <-h.ctx.Done():
			return
		}
	}
}

// processCollaborationEvents processes real-time collaboration events from Redis
func (h *Hub) processCollaborationEvents() {
	if h.redisClient == nil {
		return
	}

	// TODO: Implement Redis collaboration channel listener
	// For now, simulate collaboration events
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Simulate collaboration events
			// In production, this would listen to Redis pub/sub channels like:
			// "project:collaboration:*"
			// "project:presence:*"

			// Example: Broadcast collaboration activity
			h.BroadcastToChannel("project:collaboration:demo", "collaboration_event", map[string]interface{}{
				"type":       "cursor_move",
				"user_id":    "demo_user",
				"project_id": "demo",
				"position":   map[string]int{"x": 100, "y": 200},
				"timestamp":  time.Now().Unix(),
			})

		case <-h.ctx.Done():
			return
		}
	}
}
