package realtime

import (
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/metrics"
	"git.neolidy.top/neo/storybook/internal/middleware"
	"git.neolidy.top/neo/storybook/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

const (
	maxConnPerUser = 5
	maxWSSize      = 65536 // 64 KiB max per message
	wsReadBuf      = 4096
	wsWriteBuf     = 4096
)

type Event struct {
	Type      string    `json:"type"`
	Data      any       `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

type Hub struct {
	mu          sync.RWMutex
	db          *gorm.DB
	connections map[uint]map[*websocket.Conn]struct{}
	upgrader    websocket.Upgrader
}

func NewHub(db *gorm.DB) *Hub {
	return &Hub{
		db:          db,
		connections: make(map[uint]map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  wsReadBuf,
			WriteBufferSize: wsWriteBuf,
			CheckOrigin: func(r *http.Request) bool {
				return middleware.IsOriginAllowed(r, strings.TrimSpace(r.Header.Get("Origin")))
			},
		},
	}
}

func (h *Hub) HandleWS(c *gin.Context, userID uint, selectedProtocol string) {
	h.mu.Lock()
	if len(h.connections[userID]) >= maxConnPerUser {
		h.mu.Unlock()
		api.TooManyRequests(c, "WebSocket连接数超过限制")
		return
	}
	h.mu.Unlock()

	headers := http.Header{}
	if selectedProtocol != "" {
		headers.Set("Sec-WebSocket-Protocol", selectedProtocol)
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, headers)
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

	conn.SetReadLimit(maxWSSize)

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// prevent a single malformed message from crashing the hub goroutine
			}
			close(done)
		}()
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

func (h *Hub) BroadcastProject(projectID uint, eventType string, data any) {
	userIDs, err := h.projectRecipientIDs(projectID)
	if err != nil || len(userIDs) == 0 {
		return
	}

	event := Event{Type: eventType, Data: data, Timestamp: time.Now()}

	h.mu.RLock()
	conns := make([]*websocket.Conn, 0)
	for _, userID := range userIDs {
		for conn := range h.connections[userID] {
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

func (h *Hub) projectRecipientIDs(projectID uint) ([]uint, error) {
	if h.db == nil {
		return nil, nil
	}

	ids := make(map[uint]struct{})

	var project model.Project
	if err := h.db.Select("id, owner_id").First(&project, projectID).Error; err != nil {
		return nil, err
	}
	ids[project.OwnerID] = struct{}{}

	var memberIDs []uint
	if err := h.db.Model(&model.ProjectMember{}).Where("project_id = ?", projectID).Pluck("user_id", &memberIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range memberIDs {
		ids[id] = struct{}{}
	}

	var techLeadIDs []uint
	if err := h.db.Model(&model.ProjectTechLead{}).Where("project_id = ?", projectID).Pluck("user_id", &techLeadIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range techLeadIDs {
		ids[id] = struct{}{}
	}

	var adminIDs []uint
	if err := h.db.Model(&model.User{}).Where("role = ?", model.RoleAdmin).Pluck("id", &adminIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range adminIDs {
		ids[id] = struct{}{}
	}

	result := make([]uint, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func (h *Hub) addConn(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.connections[userID]; !ok {
		h.connections[userID] = make(map[*websocket.Conn]struct{})
	}
	h.connections[userID][conn] = struct{}{}
	metrics.WSConnections.Inc()
}

func (h *Hub) removeConn(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if userConns, ok := h.connections[userID]; ok {
		if _, present := userConns[conn]; present {
			delete(userConns, conn)
			metrics.WSConnections.Dec()
		}
		if len(userConns) == 0 {
			delete(h.connections, userID)
		}
	}
	_ = conn.Close()
}
