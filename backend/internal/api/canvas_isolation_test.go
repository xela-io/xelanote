//go:build fts5

package api

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xela-io/xelanote/internal/service"
)

// canvasRouter registers canvas routes for isolation tests.
// Canvas service must be created inline since newTestServer leaves it nil.
func canvasRouter(ts *testServer) chi.Router {
	// Create canvas service and attach it to the server
	ts.canvasService = service.NewCanvasService(ts.db, ts.noteService)

	r := ts.testRouter()
	r.Route("/api/canvas", func(r chi.Router) {
		r.Get("/", ts.listCanvasNotes)
		r.Get("/{id}/export", ts.exportCanvas)
	})
	return r
}

func TestCanvas_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := canvasRouter(ts)
	user1 := ts.createUser(t, "canvas1", "canvas1@example.com", "password123")
	user2 := ts.createUser(t, "canvas2", "canvas2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// Enable canvas feature for both users
	err := ts.noteService.SetUserFeature(user1.ID, "canvas", true, nil)
	require.NoError(t, err)
	err = ts.noteService.SetUserFeature(user2.ID, "canvas", true, nil)
	require.NoError(t, err)

	// User1 creates a canvas note
	canvasNote, err := ts.canvasService.CreateCanvasNote(user1.ID, "My Canvas", `{"nodes":[],"edges":[]}`, "/")
	require.NoError(t, err)

	// User1 sees the canvas
	rec := doJSON(t, r, http.MethodGet, "/api/canvas", nil, token1)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp1 map[string]interface{}
	decodeResponse(t, rec, &resp1)
	notes1, ok := resp1["notes"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 1, len(notes1), "User1 should see 1 canvas note")

	// User2 sees empty list
	rec = doJSON(t, r, http.MethodGet, "/api/canvas", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp2 map[string]interface{}
	decodeResponse(t, rec, &resp2)
	notes2, ok := resp2["notes"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 0, len(notes2), "User2 must not see User1's canvas notes")

	// User2 tries to export User1's canvas — 404
	rec = doJSON(t, r, http.MethodGet, "/api/canvas/"+canvasNote.ID+"/export", nil, token2)
	assert.Equal(t, http.StatusNotFound, rec.Code, "User2 must not export User1's canvas")
}
