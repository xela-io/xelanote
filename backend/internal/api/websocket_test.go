package api

import (
	"net/http"
	"testing"
)

func TestCheckOrigin_EmptyOriginProduction(t *testing.T) {
	s := &Server{
		allowedOrigins: []string{"https://xelanote.com"},
	}
	upgrader := s.createUpgrader()

	r, _ := http.NewRequest("GET", "/ws", nil)
	// No Origin header = empty origin

	if upgrader.CheckOrigin(r) {
		t.Error("empty Origin should be rejected when allowedOrigins is configured")
	}
}

func TestCheckOrigin_EmptyOriginDev(t *testing.T) {
	s := &Server{
		allowedOrigins: nil,
	}
	upgrader := s.createUpgrader()

	r, _ := http.NewRequest("GET", "/ws", nil)
	// No Origin header

	if !upgrader.CheckOrigin(r) {
		t.Error("empty Origin should be accepted in dev mode (no allowedOrigins)")
	}
}

func TestCheckOrigin_ValidOriginProduction(t *testing.T) {
	s := &Server{
		allowedOrigins: []string{"https://xelanote.com", "https://staging.example.com"},
	}
	upgrader := s.createUpgrader()

	r, _ := http.NewRequest("GET", "/ws", nil)
	r.Header.Set("Origin", "https://xelanote.com")

	if !upgrader.CheckOrigin(r) {
		t.Error("valid Origin should be accepted")
	}
}

func TestCheckOrigin_InvalidOriginProduction(t *testing.T) {
	s := &Server{
		allowedOrigins: []string{"https://xelanote.com"},
	}
	upgrader := s.createUpgrader()

	r, _ := http.NewRequest("GET", "/ws", nil)
	r.Header.Set("Origin", "https://evil.com")

	if upgrader.CheckOrigin(r) {
		t.Error("invalid Origin should be rejected")
	}
}
