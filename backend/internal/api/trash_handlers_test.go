//go:build fts5

package api

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func trashRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/notes", func(r chi.Router) {
		r.Post("/", ts.createNote)
		r.Delete("/{id}", ts.deleteNote)
		r.Post("/{id}/restore", ts.restoreNote)
		r.Delete("/{id}/permanent", ts.permanentlyDeleteNote)
	})
	r.Get("/api/trash", ts.listTrash)
	r.Get("/api/trash/count", ts.getTrashCount)
	r.Delete("/api/trash", ts.emptyTrash)
	return r
}

// createAndTrashNote is a test helper that creates a note and soft-deletes it.
func createAndTrashNote(t *testing.T, r chi.Router, token, title string) string {
	t.Helper()
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: title, Content: "content",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	rec = doJSON(t, r, http.MethodDelete, "/api/notes/"+noteID, nil, token)
	require.Equal(t, http.StatusNoContent, rec.Code)
	return noteID
}

func TestListTrash_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := trashRouter(ts)
	user := ts.createUser(t, "trashempty", "trashempty@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/trash", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp NoteListResponse
	decodeResponse(t, rec, &resp)
	assert.Empty(t, resp.Notes)
}

func TestListTrash_WithDeletedNotes(t *testing.T) {
	ts := newTestServer(t)
	r := trashRouter(ts)
	user := ts.createUser(t, "trashlist", "trashlist@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	createAndTrashNote(t, r, token, "Trashed Note 1")
	createAndTrashNote(t, r, token, "Trashed Note 2")

	rec := doJSON(t, r, http.MethodGet, "/api/trash", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp NoteListResponse
	decodeResponse(t, rec, &resp)
	assert.Len(t, resp.Notes, 2)
}

func TestGetTrashCount(t *testing.T) {
	ts := newTestServer(t)
	r := trashRouter(ts)
	user := ts.createUser(t, "trashcount", "trashcount@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	createAndTrashNote(t, r, token, "Count Me")

	rec := doJSON(t, r, http.MethodGet, "/api/trash/count", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.Equal(t, float64(1), resp["count"])
}

func TestRestoreNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := trashRouter(ts)
	user := ts.createUser(t, "restorer", "restore@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	noteID := createAndTrashNote(t, r, token, "Restore Me")

	// Restore
	rec := doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/restore", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var restored map[string]interface{}
	decodeResponse(t, rec, &restored)
	assert.Equal(t, "Restore Me", restored["title"])

	// Trash should be empty now
	rec = doJSON(t, r, http.MethodGet, "/api/trash/count", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var count map[string]interface{}
	decodeResponse(t, rec, &count)
	assert.Equal(t, float64(0), count["count"])
}

func TestRestoreNote_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := trashRouter(ts)
	user := ts.createUser(t, "restorenf", "restorenf@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes/nonexistent/restore", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPermanentlyDeleteNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := trashRouter(ts)
	user := ts.createUser(t, "permadel", "permadel@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	noteID := createAndTrashNote(t, r, token, "Permanent Delete")

	// Permanently delete
	rec := doJSON(t, r, http.MethodDelete, "/api/notes/"+noteID+"/permanent", nil, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Trash should be empty
	rec = doJSON(t, r, http.MethodGet, "/api/trash/count", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var count map[string]interface{}
	decodeResponse(t, rec, &count)
	assert.Equal(t, float64(0), count["count"])
}

func TestEmptyTrash_Success(t *testing.T) {
	ts := newTestServer(t)
	r := trashRouter(ts)
	user := ts.createUser(t, "emptytrash", "emptytrash@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	createAndTrashNote(t, r, token, "Trash 1")
	createAndTrashNote(t, r, token, "Trash 2")
	createAndTrashNote(t, r, token, "Trash 3")

	rec := doJSON(t, r, http.MethodDelete, "/api/trash", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.Equal(t, float64(3), resp["deleted_count"])

	// Verify trash is empty
	rec = doJSON(t, r, http.MethodGet, "/api/trash/count", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var count map[string]interface{}
	decodeResponse(t, rec, &count)
	assert.Equal(t, float64(0), count["count"])
}

func TestTrash_CrossUserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := trashRouter(ts)

	user1 := ts.createUser(t, "trashown", "trashown@example.com", "password123")
	user2 := ts.createUser(t, "trashother", "trashother@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	noteID := createAndTrashNote(t, r, token1, "User1's Trash")

	// User2 cannot restore User1's trashed note
	rec := doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/restore", nil, token2)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// User2's trash should be empty
	rec = doJSON(t, r, http.MethodGet, "/api/trash", nil, token2)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp NoteListResponse
	decodeResponse(t, rec, &resp)
	assert.Empty(t, resp.Notes)
}
