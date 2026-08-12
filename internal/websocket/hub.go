package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for simplicity, can be restricted in production
	},
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	clientsMux sync.Mutex
}

var GlobalHub = &Hub{
	clients: make(map[*websocket.Conn]bool),
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket Upgrade Error:", err)
		return
	}

	h.clientsMux.Lock()
	h.clients[conn] = true
	h.clientsMux.Unlock()

	defer func() {
		h.clientsMux.Lock()
		delete(h.clients, conn)
		h.clientsMux.Unlock()
		conn.Close()
	}()

	// Keep alive loop, read messages if any (not strictly needed since we just push to client)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *Hub) Broadcast(msgType string, payload interface{}) {
	msg := map[string]interface{}{
		"type":    msgType,
		"payload": payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("WS Marshal Error:", err)
		return
	}

	h.clientsMux.Lock()
	defer h.clientsMux.Unlock()

	for conn := range h.clients {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Println("WS Write Error:", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
