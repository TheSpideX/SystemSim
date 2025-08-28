package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fasthttp/websocket"
	gorillaWS "github.com/gorilla/websocket"
	"github.com/valyala/fasthttp"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

// WebSocket upgrader with optimized settings
var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		// Allow all origins for development (restrict in production)
		return true
	},
	EnableCompression: true, // Enable per-message compression
}

// NewConnection creates a new WebSocket connection (FastHTTP)
func NewConnection(conn *websocket.Conn, userID, connectionID string, hub *Hub) *Connection {
	return &Connection{
		ID:            connectionID,
		UserID:        userID,
		Conn:          conn,
		Send:          make(chan []byte, 256), // Buffered send channel
		Hub:           hub,
		ConnectedAt:   time.Now(),
		LastActivity:  time.Now(),
		Subscriptions: make(map[string]bool),
	}
}

// WebSocketContext contains context information for WebSocket connections
type WebSocketContext struct {
	UserID     string
	Type       string // "general", "notifications", "simulation", "collaboration"
	ResourceID string // simulation_id, project_id, etc.
	Path       string // original request path
}

// GorillaConnection wraps a Gorilla WebSocket connection to work with our Hub
type GorillaConnection struct {
	ID            string
	UserID        string
	Conn          *gorillaWS.Conn
	Send          chan []byte
	Hub           *Hub
	ConnectedAt   time.Time
	LastActivity  time.Time
	Subscriptions map[string]bool
	MessagesSent     int64
	MessagesReceived int64
	mutex         sync.RWMutex

	// WebSocket context
	Context *WebSocketContext
}

// NewGorillaConnection creates a new Gorilla WebSocket connection
func NewGorillaConnection(conn *gorillaWS.Conn, userID, connectionID string, hub *Hub) *GorillaConnection {
	return &GorillaConnection{
		ID:            connectionID,
		UserID:        userID,
		Conn:          conn,
		Send:          make(chan []byte, 256), // Buffered send channel
		Hub:           hub,
		ConnectedAt:   time.Now(),
		LastActivity:  time.Now(),
		Subscriptions: make(map[string]bool),
		Context:       &WebSocketContext{UserID: userID, Type: "general"},
	}
}

// NewGorillaConnectionWithContext creates a new Gorilla WebSocket connection with context
func NewGorillaConnectionWithContext(conn *gorillaWS.Conn, connectionID string, context *WebSocketContext, hub *Hub) *GorillaConnection {
	return &GorillaConnection{
		ID:            connectionID,
		UserID:        context.UserID,
		Conn:          conn,
		Send:          make(chan []byte, 256), // Buffered send channel
		Hub:           hub,
		ConnectedAt:   time.Now(),
		LastActivity:  time.Now(),
		Subscriptions: make(map[string]bool),
		Context:       context,
	}
}

// SendMessage sends a message to the Gorilla WebSocket connection
func (c *GorillaConnection) SendMessage(data []byte) {
	select {
	case c.Send <- data:
		atomic.AddInt64(&c.MessagesSent, 1)
		c.updateLastActivity()
	default:
		// Channel full, close connection
		log.Printf("Send channel full for connection %s, closing", c.ID)
		c.Close()
	}
}

// Close closes the Gorilla WebSocket connection
func (c *GorillaConnection) Close() {
	c.Conn.Close()
}

// updateLastActivity updates the last activity timestamp
func (c *GorillaConnection) updateLastActivity() {
	c.mutex.Lock()
	c.LastActivity = time.Now()
	c.mutex.Unlock()
}

// HandleConnection handles the Gorilla WebSocket connection lifecycle
func (c *GorillaConnection) HandleConnection() {
	defer func() {
		// Broadcast presence left for collaboration
		if c.Context != nil && c.Context.Type == "collaboration" {
			c.broadcastPresence("left")
		}

		c.Hub.UnregisterConnection(c.toConnection())
		c.Close()
	}()

	// Register connection with hub
	c.Hub.RegisterConnection(c.toConnection())

	// Start goroutines for reading and writing
	go c.writePump()
	go c.readPump()

	// Keep connection alive until context is done
	<-c.Hub.ctx.Done()
}

// toConnection converts GorillaConnection to Connection for hub compatibility
func (c *GorillaConnection) toConnection() *Connection {
	return &Connection{
		ID:               c.ID,
		UserID:           c.UserID,
		Conn:             nil, // This will be nil for Gorilla connections
		Send:             c.Send,
		Hub:              c.Hub,
		ConnectedAt:      c.ConnectedAt,
		LastActivity:     c.LastActivity,
		Subscriptions:    c.Subscriptions,
		MessagesSent:     c.MessagesSent,
		MessagesReceived: c.MessagesReceived,
	}
}

// readPump handles reading messages from the Gorilla WebSocket connection
func (c *GorillaConnection) readPump() {
	defer c.Close()

	// Set read limits and deadline
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		c.updateLastActivity()
		return nil
	})

	for {
		// Read message from WebSocket
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if gorillaWS.IsUnexpectedCloseError(err, gorillaWS.CloseGoingAway, gorillaWS.CloseAbnormalClosure) {
				log.Printf("WebSocket error for connection %s: %v", c.ID, err)
			}
			break
		}

		// Update activity and stats
		atomic.AddInt64(&c.MessagesReceived, 1)
		c.updateLastActivity()

		// Process incoming message
		c.handleIncomingMessage(message)
	}
}

// writePump handles writing messages to the Gorilla WebSocket connection
func (c *GorillaConnection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(gorillaWS.CloseMessage, []byte{})
				return
			}

			// Send message
			if err := c.Conn.WriteMessage(gorillaWS.TextMessage, message); err != nil {
				log.Printf("Write error for connection %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(gorillaWS.PingMessage, nil); err != nil {
				return
			}

		case <-c.Hub.ctx.Done():
			return
		}
	}
}

// handleIncomingMessage processes messages received from the Gorilla WebSocket client
func (c *GorillaConnection) handleIncomingMessage(data []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to unmarshal message from connection %s: %v", c.ID, err)
		return
	}

	switch msg.Type {
	case "subscribe":
		c.handleSubscribe(msg)
	case "unsubscribe":
		c.handleUnsubscribe(msg)
	case "ping":
		c.handlePing()
	default:
		log.Printf("Unknown message type from connection %s: %s", c.ID, msg.Type)
	}
}

// handleSubscribe handles subscription requests for Gorilla WebSocket
func (c *GorillaConnection) handleSubscribe(msg IncomingMessage) {
	if msg.Channel == "" {
		return
	}

	c.mutex.Lock()
	c.Subscriptions[msg.Channel] = true
	c.mutex.Unlock()

	// Send confirmation
	response := map[string]interface{}{
		"type":    "subscribed",
		"channel": msg.Channel,
		"status":  "success",
	}

	if data, err := json.Marshal(response); err == nil {
		c.SendMessage(data)
	}

	log.Printf("Connection %s subscribed to channel: %s", c.ID, msg.Channel)
}

// handleUnsubscribe handles unsubscription requests for Gorilla WebSocket
func (c *GorillaConnection) handleUnsubscribe(msg IncomingMessage) {
	if msg.Channel == "" {
		return
	}

	c.mutex.Lock()
	delete(c.Subscriptions, msg.Channel)
	c.mutex.Unlock()

	// Send confirmation
	response := map[string]interface{}{
		"type":    "unsubscribed",
		"channel": msg.Channel,
		"status":  "success",
	}

	if data, err := json.Marshal(response); err == nil {
		c.SendMessage(data)
	}

	log.Printf("Connection %s unsubscribed from channel: %s", c.ID, msg.Channel)
}

// handlePing handles ping messages for Gorilla WebSocket
func (c *GorillaConnection) handlePing() {
	response := map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now().Unix(),
	}

	if data, err := json.Marshal(response); err == nil {
		c.SendMessage(data)
	}
}

// autoSubscribeBasedOnContext automatically subscribes to relevant channels based on WebSocket context
func (c *GorillaConnection) autoSubscribeBasedOnContext() {
	if c.Context == nil {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch c.Context.Type {
	case "notifications":
		// Subscribe to user-specific notifications
		channel := fmt.Sprintf("notifications:user:%s", c.Context.UserID)
		c.Subscriptions[channel] = true
		log.Printf("Auto-subscribed connection %s to notifications: %s", c.ID, channel)

	case "simulation":
		if c.Context.ResourceID != "" {
			// Subscribe to simulation-specific events and data
			eventChannel := fmt.Sprintf("simulation:events:%s", c.Context.ResourceID)
			dataChannel := fmt.Sprintf("simulation:data:%s", c.Context.ResourceID)

			c.Subscriptions[eventChannel] = true
			c.Subscriptions[dataChannel] = true

			log.Printf("Auto-subscribed connection %s to simulation channels: %s, %s",
				c.ID, eventChannel, dataChannel)
		}

	case "collaboration":
		if c.Context.ResourceID != "" {
			// Subscribe to project collaboration events
			collabChannel := fmt.Sprintf("project:collaboration:%s", c.Context.ResourceID)
			presenceChannel := fmt.Sprintf("project:presence:%s", c.Context.ResourceID)

			c.Subscriptions[collabChannel] = true
			c.Subscriptions[presenceChannel] = true

			log.Printf("Auto-subscribed connection %s to collaboration channels: %s, %s",
				c.ID, collabChannel, presenceChannel)

			// Send presence notification
			c.broadcastPresence("joined")
		}

	case "general":
		// Subscribe to general system notifications
		c.Subscriptions["system:announcements"] = true
		log.Printf("Auto-subscribed connection %s to general system announcements", c.ID)
	}
}

// broadcastPresence broadcasts user presence changes for collaboration
func (c *GorillaConnection) broadcastPresence(action string) {
	if c.Context == nil || c.Context.Type != "collaboration" || c.Context.ResourceID == "" {
		return
	}

	presenceData := map[string]interface{}{
		"type":       "presence",
		"action":     action, // "joined", "left", "active"
		"user_id":    c.Context.UserID,
		"project_id": c.Context.ResourceID,
		"timestamp":  time.Now().Unix(),
	}

	// Broadcast to all users in the same project
	c.Hub.BroadcastToChannel(
		fmt.Sprintf("project:presence:%s", c.Context.ResourceID),
		"presence_update",
		presenceData,
	)
}

// SendMessage sends a message to the WebSocket connection
func (c *Connection) SendMessage(data []byte) {
	select {
	case c.Send <- data:
		atomic.AddInt64(&c.MessagesSent, 1)
		c.updateLastActivity()
	default:
		// Channel full, close connection
		log.Printf("Send channel full for connection %s, closing", c.ID)
		c.Close()
	}
}

// Close closes the WebSocket connection
func (c *Connection) Close() {
	c.Conn.Close()
}

// updateLastActivity updates the last activity timestamp
func (c *Connection) updateLastActivity() {
	c.mutex.Lock()
	c.LastActivity = time.Now()
	c.mutex.Unlock()
}

// HandleConnection handles the WebSocket connection lifecycle
func (c *Connection) HandleConnection() {
	defer func() {
		c.Hub.UnregisterConnection(c)
		c.Close()
	}()

	// Register connection with hub
	c.Hub.RegisterConnection(c)

	// Start goroutines for reading and writing
	go c.writePump()
	go c.readPump()

	// Keep connection alive until context is done
	<-c.Hub.ctx.Done()
}

// readPump handles reading messages from the WebSocket connection
func (c *Connection) readPump() {
	defer c.Close()

	// Set read limits and deadline
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		c.updateLastActivity()
		return nil
	})

	for {
		// Read message from WebSocket
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for connection %s: %v", c.ID, err)
			}
			break
		}

		// Update activity and stats
		atomic.AddInt64(&c.MessagesReceived, 1)
		c.updateLastActivity()

		// Process incoming message
		c.handleIncomingMessage(message)
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send message
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error for connection %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.Hub.ctx.Done():
			return
		}
	}
}

// IncomingMessage represents a message received from the client
type IncomingMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data,omitempty"`
	Channel string      `json:"channel,omitempty"`
}

// handleIncomingMessage processes messages received from the client
func (c *Connection) handleIncomingMessage(data []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Failed to unmarshal message from connection %s: %v", c.ID, err)
		return
	}

	switch msg.Type {
	case "subscribe":
		c.handleSubscribe(msg)
	case "unsubscribe":
		c.handleUnsubscribe(msg)
	case "ping":
		c.handlePing()
	default:
		log.Printf("Unknown message type from connection %s: %s", c.ID, msg.Type)
	}
}

// handleSubscribe handles subscription requests
func (c *Connection) handleSubscribe(msg IncomingMessage) {
	if msg.Channel == "" {
		return
	}

	c.mutex.Lock()
	c.Subscriptions[msg.Channel] = true
	c.mutex.Unlock()

	// Handle health channel subscriptions
	if len(msg.Channel) > 7 && msg.Channel[:7] == "health:" {
		// Convert to Gorilla WebSocket for health manager
		gorillaConn := &gorillaWS.Conn{}
		// Note: This is a simplified conversion - in production you'd need proper conversion
		c.Hub.healthManager.Subscribe(gorillaConn, msg.Channel)
	}

	// Send confirmation
	response := map[string]interface{}{
		"type":    "subscribed",
		"channel": msg.Channel,
		"status":  "success",
	}

	if data, err := json.Marshal(response); err == nil {
		c.SendMessage(data)
	}

	log.Printf("Connection %s subscribed to channel: %s", c.ID, msg.Channel)
}

// handleUnsubscribe handles unsubscription requests
func (c *Connection) handleUnsubscribe(msg IncomingMessage) {
	if msg.Channel == "" {
		return
	}

	c.mutex.Lock()
	delete(c.Subscriptions, msg.Channel)
	c.mutex.Unlock()

	// Send confirmation
	response := map[string]interface{}{
		"type":    "unsubscribed",
		"channel": msg.Channel,
		"status":  "success",
	}

	if data, err := json.Marshal(response); err == nil {
		c.SendMessage(data)
	}

	log.Printf("Connection %s unsubscribed from channel: %s", c.ID, msg.Channel)
}

// handlePing handles ping messages
func (c *Connection) handlePing() {
	response := map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now().Unix(),
	}

	if data, err := json.Marshal(response); err == nil {
		c.SendMessage(data)
	}
}

// IsSubscribedTo checks if the connection is subscribed to a channel
func (c *Connection) IsSubscribedTo(channel string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Subscriptions[channel]
}

// GetSubscriptions returns all subscriptions for this connection
func (c *Connection) GetSubscriptions() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	subscriptions := make([]string, 0, len(c.Subscriptions))
	for channel := range c.Subscriptions {
		subscriptions = append(subscriptions, channel)
	}
	return subscriptions
}

// GetConnectionInfo returns connection information
func (c *Connection) GetConnectionInfo() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"id":                c.ID,
		"user_id":           c.UserID,
		"connected_at":      c.ConnectedAt,
		"last_activity":     c.LastActivity,
		"messages_sent":     atomic.LoadInt64(&c.MessagesSent),
		"messages_received": atomic.LoadInt64(&c.MessagesReceived),
		"subscriptions":     c.GetSubscriptions(),
		"send_buffer_size":  len(c.Send),
		"send_buffer_cap":   cap(c.Send),
	}
}

// UpgradeHTTP upgrades a FastHTTP connection to WebSocket
func UpgradeHTTP(ctx *fasthttp.RequestCtx, userID string, hub *Hub) error {
	return upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		// Generate connection ID
		connectionID := generateConnectionID()

		// Create connection
		wsConn := NewConnection(conn, userID, connectionID, hub)

		// Handle connection (blocking)
		wsConn.HandleConnection()
	})
}

// Standard HTTP WebSocket upgrader for Gorilla WebSocket
var gorillaUpgrader = gorillaWS.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development (restrict in production)
		return true
	},
	EnableCompression: true,
}

// UpgradeStandardHTTP upgrades a standard HTTP connection to WebSocket
func UpgradeStandardHTTP(w http.ResponseWriter, r *http.Request, userID string, hub *Hub) error {
	conn, err := gorillaUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	// Generate connection ID
	connectionID := generateConnectionID()

	// Create a bridge connection that wraps Gorilla WebSocket
	wsConn := NewGorillaConnection(conn, userID, connectionID, hub)

	// Handle connection (blocking)
	wsConn.HandleConnection()

	return nil
}

// UpgradeStandardHTTPWithContext upgrades a standard HTTP connection to WebSocket with context
func UpgradeStandardHTTPWithContext(w http.ResponseWriter, r *http.Request, context *WebSocketContext, hub *Hub) error {
	conn, err := gorillaUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	// Generate connection ID
	connectionID := generateConnectionID()

	// Create a bridge connection with context
	wsConn := NewGorillaConnectionWithContext(conn, connectionID, context, hub)

	// Auto-subscribe to relevant channels based on context
	wsConn.autoSubscribeBasedOnContext()

	// Handle connection (blocking)
	wsConn.HandleConnection()

	return nil
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	// Simple timestamp-based ID for now
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
