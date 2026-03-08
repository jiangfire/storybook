package realtime

import (
	"net/http"
	"sync"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const maxConnPerUser = 5

type Event struct {
	Type      string    `json:"type"`
	Data      any       `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

type Hub struct {
	mu          sync.RWMutex
	connections map[uint]map[*websocket.Conn]struct{}
	upgrader    websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[uint]map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

func (h *Hub) HandleWS(c *gin.Context, userID uint) {
	h.mu.Lock()
	if len(h.connections[userID]) >= maxConnPerUser {
		h.mu.Unlock()
		api.TooManyRequests(c, "WebSocket连接数超过限制")
		return
	}
	h.mu.Unlock()

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	h.addConn(userID, conn)
	defer h.removeConn(userID, conn)

	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-pingTicker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) Broadcast(eventType string, data any) {
	event := Event{Type: eventType, Data: data, Timestamp: time.Now()}

	h.mu.RLock()
	conns := make([]*websocket.Conn, 0)
	for _, userConns := range h.connections {
		for conn := range userConns {
			conns = append(conns, conn)
		}
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteJSON(event); err != nil {
			_ = conn.Close()
		}
	}
}

func (h *Hub) addConn(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.connections[userID]; !ok {
		h.connections[userID] = make(map[*websocket.Conn]struct{})
	}
	h.connections[userID][conn] = struct{}{}
}

func (h *Hub) removeConn(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if userConns, ok := h.connections[userID]; ok {
		delete(userConns, conn)
		if len(userConns) == 0 {
			delete(h.connections, userID)
		}
	}
	_ = conn.Close()
}
