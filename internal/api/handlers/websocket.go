package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex
	upgrader   websocket.Upgrader
}

type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type MLStatusUpdate struct {
	Enabled     bool      `json:"enabled"`
	LastUpdate  time.Time `json:"last_update"`
	ModelVersion string   `json:"model_version"`
}

type RequestCompleted struct {
	RequestID    string    `json:"request_id"`
	Domain       string    `json:"domain"`
	Success      bool      `json:"success"`
	Latency      int64     `json:"latency"`
	Techniques   []string  `json:"techniques"`
	DPIType      string    `json:"dpi_type"`
	Confidence   float64   `json:"confidence"`
	CompletedAt  time.Time `json:"completed_at"`
}

type TechniqueEffectivenessChanged struct {
	Technique     string    `json:"technique"`
	Domain        string    `json:"domain"`
	Effectiveness float64   `json:"effectiveness"`
	SampleCount   int       `json:"sample_count"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				// Allow connections from localhost development servers
				return origin == "http://localhost:3000" || 
					   origin == "http://localhost:5173" ||
					   origin == "http://127.0.0.1:3000" ||
					   origin == "http://127.0.0.1:5173" ||
					   origin == ""
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Add client
	h.clientsMux.Lock()
	h.clients[conn] = true
	h.clientsMux.Unlock()

	log.Printf("WebSocket client connected. Total clients: %d", len(h.clients))

	// Send initial status
	h.sendInitialStatus(conn)

	// Handle messages from client
	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle client messages if needed
		log.Printf("Received WebSocket message: %s", msg.Type)
	}

	// Remove client
	h.clientsMux.Lock()
	delete(h.clients, conn)
	h.clientsMux.Unlock()

	log.Printf("WebSocket client disconnected. Total clients: %d", len(h.clients))
}

func (h *WebSocketHandler) sendInitialStatus(conn *websocket.Conn) {
	status := MLStatusUpdate{
		Enabled:     true,
		LastUpdate:  time.Now(),
		ModelVersion: "1.0",
	}

	msg := WSMessage{
		Type:      "ml.status_update",
		Data:      status,
		Timestamp: time.Now(),
	}

	conn.WriteJSON(msg)
}

func (h *WebSocketHandler) BroadcastMLStatus(status MLStatusUpdate) {
	msg := WSMessage{
		Type:      "ml.status_update",
		Data:      status,
		Timestamp: time.Now(),
	}

	h.broadcast(msg)
}

func (h *WebSocketHandler) BroadcastRequestCompleted(req RequestCompleted) {
	msg := WSMessage{
		Type:      "request.completed",
		Data:      req,
		Timestamp: time.Now(),
	}

	h.broadcast(msg)
}

func (h *WebSocketHandler) BroadcastTechniqueEffectiveness(tech TechniqueEffectivenessChanged) {
	msg := WSMessage{
		Type:      "technique.effectiveness_changed",
		Data:      tech,
		Timestamp: time.Now(),
	}

	h.broadcast(msg)
}

func (h *WebSocketHandler) BroadcastCustom(eventType string, data interface{}) {
	msg := WSMessage{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	h.broadcast(msg)
}

func (h *WebSocketHandler) broadcast(msg WSMessage) {
	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal WebSocket message: %v", err)
		return
	}

	for conn := range h.clients {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Failed to send WebSocket message: %v", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

func (h *WebSocketHandler) GetClientCount() int {
	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()
	return len(h.clients)
}
