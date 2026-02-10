package main

import (
	"log"
	"log/slog"

	"github.com/xela-io/xelanote/internal/websocket"
)

func startWebSocketManager(logger *slog.Logger) *websocket.Manager {
	wsManager := websocket.NewManager(logger)
	go wsManager.Run()
	log.Println("WebSocket manager started")
	return wsManager
}
