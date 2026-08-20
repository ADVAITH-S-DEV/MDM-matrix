package hub

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

type Hub struct {
	Mu    sync.RWMutex
	Conns map[string]*websocket.Conn
}

func NewHub() *Hub {
	return &Hub{
		Conns: make(map[string]*websocket.Conn),
	}
}

func (h *Hub) Add(deviceID string, conn *websocket.Conn) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	h.Conns[deviceID] = conn
}

func (h *Hub) Remove(deviceID string) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	delete(h.Conns, deviceID)
}

// AdminHub manages connected React dashboard clients
type AdminHub struct {
	Mu    sync.RWMutex
	Conns map[*websocket.Conn]bool
}

func NewAdminHub() *AdminHub {
	return &AdminHub{
		Conns: make(map[*websocket.Conn]bool),
	}
}

func (h *AdminHub) Add(conn *websocket.Conn) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	h.Conns[conn] = true
}

func (h *AdminHub) Remove(conn *websocket.Conn) {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	delete(h.Conns, conn)
}

// Broadcast sends a JSON message to all connected dashboards in O(K) time
func (h *AdminHub) Broadcast(message interface{}) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	for conn := range h.Conns {
		// If a write fails, we ignore it here; the read loop will clean it up
		conn.WriteJSON(message) 
	}
}