package websocket

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestManagerRegisterBroadcastUnregister(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager := NewManager(logger)
	go manager.Run()
	t.Cleanup(manager.Stop)

	wsConn, cleanup := newTestWebSocketConn(t)
	t.Cleanup(cleanup)

	conn := &Connection{
		UserID: 42,
		Conn:   wsConn,
		Send:   make(chan []byte, 1),
	}

	manager.Register(conn)
	waitForConnectionCount(t, manager, 1)

	msg := Message{Type: "ping", Payload: json.RawMessage(`{"ok":true}`)}
	manager.BroadcastToUser(42, msg)

	select {
	case data := <-conn.Send:
		var got Message
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal broadcast: %v", err)
		}
		if got.Type != "ping" {
			t.Fatalf("expected type %q, got %q", "ping", got.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for broadcast")
	}

	manager.Unregister(conn)
	waitForConnectionCount(t, manager, 0)
	waitForChannelClosed(t, conn.Send)
}

func TestManagerStopClosesConnections(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager := NewManager(logger)
	go manager.Run()

	wsConn, cleanup := newTestWebSocketConn(t)
	t.Cleanup(cleanup)

	conn := &Connection{
		UserID: 7,
		Conn:   wsConn,
		Send:   make(chan []byte, 1),
	}

	manager.Register(conn)
	waitForConnectionCount(t, manager, 1)

	manager.Stop()
	waitForConnectionCount(t, manager, 0)
	waitForChannelClosed(t, conn.Send)
}

func newTestWebSocketConn(t *testing.T) (*websocket.Conn, func()) {
	t.Helper()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	var serverConn *websocket.Conn
	connected := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade error: %v", err)
			return
		}
		serverConn = conn
		close(connected)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		server.Close()
		t.Fatalf("dial error: %v", err)
	}

	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		clientConn.Close()
		server.Close()
		t.Fatal("timed out waiting for server connection")
	}

	cleanup := func() {
		clientConn.Close()
		if serverConn != nil {
			serverConn.Close()
		}
		server.Close()
	}

	return clientConn, cleanup
}

func waitForConnectionCount(t *testing.T, manager *Manager, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if manager.GetConnectionCount() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected connection count %d, got %d", want, manager.GetConnectionCount())
}

func waitForChannelClosed(t *testing.T, ch <-chan []byte) {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatal("timed out waiting for channel to close")
		}
	}
}
