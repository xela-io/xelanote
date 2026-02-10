package api

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xela-io/xelanote/internal/auth"
	ws "github.com/xela-io/xelanote/internal/websocket"
)

// createUpgrader creates a WebSocket upgrader with origin validation
func (s *Server) createUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")

			// Require Origin header in production (allowedOrigins configured).
			// Allow empty Origin only in development mode.
			if origin == "" {
				return len(s.allowedOrigins) == 0
			}

			// If no allowed origins configured, reject cross-origin requests
			if len(s.allowedOrigins) == 0 {
				return false
			}

			// Check against allowed origins
			for _, allowed := range s.allowedOrigins {
				if origin == allowed {
					return true
				}
			}

			return false
		},
	}
}

// validateToken validates a JWT token and returns the userID
func (s *Server) validateToken(token string) (int, error) {
	claims, err := auth.ValidateAccessToken(token, s.jwtSecret)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

// handleWebSocket upgrades HTTP connection to WebSocket
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get JWT token from HttpOnly cookie only (secure, not logged in URLs or Referer headers)
	// SECURITY: Never accept tokens from URL query parameters - they leak in logs/history
	token := getAccessTokenFromCookie(r)
	if token == "" {
		respondError(w, http.StatusUnauthorized, "missing token")
		return
	}

	// Validate JWT and get userID
	userID, err := s.validateToken(token)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	// Upgrade HTTP connection to WebSocket
	upgrader := s.createUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger().Error("WebSocket upgrade failed", "error", err)
		return
	}

	// Create connection
	connection := &ws.Connection{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	// Register connection
	s.wsManager.Register(connection)

	// Start goroutines
	go s.wsWriter(connection)
	go s.wsReader(connection)
}

// wsReader reads messages from WebSocket (Read pump)
func (s *Server) wsReader(conn *ws.Connection) {
	defer func() {
		s.wsManager.Unregister(conn)
		conn.Conn.Close()
	}()

	conn.Conn.SetReadDeadline(time.Now().Add(ws.PongWait))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(ws.PongWait))
		return nil
	})
	conn.Conn.SetReadLimit(int64(ws.MaxMessageSize))

	for {
		var msg ws.Message
		err := conn.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger().Error("WebSocket read error", "error", err)
			}
			break
		}

		// Handle client messages (ping, subscribe, etc.)
		s.handleWSMessage(conn, msg)
	}
}

// wsWriter writes messages to WebSocket (Write pump)
func (s *Server) wsWriter(conn *ws.Connection) {
	ticker := time.NewTicker(ws.PingPeriod)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(ws.WriteWait))
			if !ok {
				// Channel closed
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := conn.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}

		case <-ticker.C:
			// Send ping every pingPeriod
			conn.Conn.SetWriteDeadline(time.Now().Add(ws.WriteWait))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleWSMessage handles messages from client
func (s *Server) handleWSMessage(conn *ws.Connection, msg ws.Message) {
	// Currently no client messages expected (server pushes updates only)
	// Future: Could handle "ping", "subscribe to specific notes", etc.
	s.logger().Info("WebSocket message received", "type", msg.Type, "userID", conn.UserID)
}
