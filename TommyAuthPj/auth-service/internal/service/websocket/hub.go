package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

// Hub manages WebSocket connections
type Hub struct {
	connections map[string]*websocket.Conn // code -> connection
	mu          sync.RWMutex
}

// NewHub creates a new hub
func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]*websocket.Conn),
	}
}

// RegisterConnection registers a WebSocket connection for a code
func (h *Hub) RegisterConnection(code string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[code] = conn
}

// UnregisterConnection removes a connection
func (h *Hub) UnregisterConnection(code string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conn, exists := h.connections[code]; exists {
		conn.Close()
		delete(h.connections, code)
	}
}

// SendMessage sends a message to the connection associated with the code
func (h *Hub) SendMessage(code string, message map[string]interface{}) {
	h.mu.RLock()
	conn, exists := h.connections[code]
	h.mu.RUnlock()

	if !exists {
		log.Printf("No connection found for code: %s", code)
		return
	}

	if err := conn.WriteJSON(message); err != nil {
		log.Printf("Error sending message to code %s: %v", code, err)
		h.UnregisterConnection(code)
	}
}

// HandleWebSocket handles WebSocket connections
func (h *Hub) HandleWebSocket(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code parameter required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	h.RegisterConnection(code, conn)

	// Keep the connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Connection closed for code %s: %v", code, err)
			h.UnregisterConnection(code)
			break
		}
	}
}
