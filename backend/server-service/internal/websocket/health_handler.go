package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// HealthWebSocketHandler handles WebSocket connections for health updates
type HealthWebSocketHandler struct {
	healthManager *HealthManager
	upgrader      websocket.Upgrader
}

// NewHealthWebSocketHandler creates a new health WebSocket handler
func NewHealthWebSocketHandler(healthManager *HealthManager) *HealthWebSocketHandler {
	return &HealthWebSocketHandler{
		healthManager: healthManager,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development (restrict in production)
				return true
			},
		},
	}
}

// HandleWebSocket handles WebSocket connections for health updates
func (h *HealthWebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("New WebSocket connection established for health updates")

	// Handle incoming messages
	for {
		_, messageData, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process health subscription messages
		if err := h.healthManager.HandleWebSocketMessage(conn, messageData); err != nil {
			log.Printf("Failed to handle WebSocket message: %v", err)
		}
	}

	// Clean up subscriptions when connection closes
	h.healthManager.RemoveConnection(conn)
	log.Printf("WebSocket connection closed for health updates")
}
