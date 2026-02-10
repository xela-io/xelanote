package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	WriteWait      = 10 * time.Second
	PongWait       = 60 * time.Second
	PingPeriod     = 50 * time.Second // Must be < pongWait
	MaxMessageSize = 512 * 1024       // 512 KB
)

// Manager handles all WebSocket connections
type Manager struct {
	connections map[int][]*Connection // userID -> connections
	mu          sync.RWMutex
	broadcast   chan BroadcastMessage
	register    chan *Connection
	unregister  chan *Connection
	logger      *slog.Logger
	done        chan struct{}
	stopOnce    sync.Once
}

// Connection represents a WebSocket connection
type Connection struct {
	UserID    int
	Conn      *websocket.Conn
	Send      chan []byte
	closeOnce sync.Once // Prevents double-close race condition
}

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// BroadcastMessage represents a message to broadcast to a user
type BroadcastMessage struct {
	UserID int
	Data   []byte
}

// NewManager creates a new WebSocket manager
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		connections: make(map[int][]*Connection),
		broadcast:   make(chan BroadcastMessage, 256),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		logger:      logger,
		done:        make(chan struct{}),
	}
}

// Run starts the manager's main loop
func (m *Manager) Run() {
	for {
		select {
		case <-m.done:
			return
		case conn := <-m.register:
			m.mu.Lock()
			m.connections[conn.UserID] = append(m.connections[conn.UserID], conn)
			m.mu.Unlock()
			m.logger.Info("WebSocket connected", "userID", conn.UserID)

		case conn := <-m.unregister:
			m.mu.Lock()
			if conns, ok := m.connections[conn.UserID]; ok {
				for i, c := range conns {
					if c == conn {
						m.connections[conn.UserID] = append(conns[:i], conns[i+1:]...)
						// Use closeOnce to prevent double-close race condition
						conn.closeOnce.Do(func() {
							close(conn.Send)
							conn.Conn.Close()
						})
						break
					}
				}
				if len(m.connections[conn.UserID]) == 0 {
					delete(m.connections, conn.UserID)
				}
			}
			m.mu.Unlock()
			m.logger.Info("WebSocket disconnected", "userID", conn.UserID)

		case msg := <-m.broadcast:
			m.mu.RLock()
			for _, conn := range m.connections[msg.UserID] {
				select {
				case conn.Send <- msg.Data:
				default:
					// Channel full → close connection
					go m.Unregister(conn)
				}
			}
			m.mu.RUnlock()
		}
	}
}

// Stop shuts down the manager and closes all connections.
func (m *Manager) Stop() {
	m.stopOnce.Do(func() {
		close(m.done)
		m.mu.Lock()
		for userID, conns := range m.connections {
			for _, conn := range conns {
				conn.closeOnce.Do(func() {
					close(conn.Send)
					conn.Conn.Close()
				})
			}
			delete(m.connections, userID)
		}
		m.mu.Unlock()
	})
}

// Register registers a new connection
func (m *Manager) Register(conn *Connection) {
	select {
	case <-m.done:
		return
	case m.register <- conn:
	}
}

// Unregister unregisters a connection
func (m *Manager) Unregister(conn *Connection) {
	select {
	case <-m.done:
		return
	case m.unregister <- conn:
	}
}

// BroadcastToUser sends a message to all connections of a user
func (m *Manager) BroadcastToUser(userID int, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		m.logger.Error("Failed to marshal WebSocket message", "error", err)
		return
	}
	select {
	case <-m.done:
		return
	case m.broadcast <- BroadcastMessage{UserID: userID, Data: data}:
	}
}

// GetConnectionCount returns the number of active connections
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, conns := range m.connections {
		count += len(conns)
	}
	return count
}
