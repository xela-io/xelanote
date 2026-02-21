//go:build fts5

package api

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func notesRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/notes", func(r chi.Router) {
		r.Get("/", ts.listNotes)
		r.Post("/", ts.createNote)
		r.Get("/{id}", ts.getNote)
		r.Put("/{id}", ts.updateNote)
		r.Delete("/{id}", ts.deleteNote)
	})
	return r
}

func TestCreateNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "noteuser", "note@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Test Note",
		Content: "Hello world",
	}, token)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var note map[string]interface{}
	decodeResponse(t, rec, &note)
	assert.Equal(t, "Test Note", note["title"])
	assert.NotEmpty(t, note["id"])
	assert.NotEmpty(t, rec.Header().Get("ETag"))
}

func TestCreateNote_WithFolder(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "folderuser", "folder@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:      "Folder Note",
		Content:    "In a folder",
		FolderPath: "/projects/test",
	}, token)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var note map[string]interface{}
	decodeResponse(t, rec, &note)
	assert.Equal(t, "/projects/test", note["folder_path"])
}

func TestCreateNote_Unauthorized(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Should Fail",
		Content: "No auth",
	}, "")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "getuser", "get@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create a note first
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Fetch Me",
		Content: "Fetch this note",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Fetch it
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+noteID, nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var fetched map[string]interface{}
	decodeResponse(t, rec, &fetched)
	assert.Equal(t, "Fetch Me", fetched["title"])
	assert.Equal(t, noteID, fetched["id"])
}

func TestGetNote_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "nfuser", "nf@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/notes/nonexistent-id", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetNote_CrossUserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)

	user1 := ts.createUser(t, "owner", "owner@example.com", "password123")
	user2 := ts.createUser(t, "other", "other@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User 1 creates a note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Private Note",
		Content: "Secret",
	}, token1)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// User 2 cannot access it
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+noteID, nil, token2)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestListNotes_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "emptyuser", "empty@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/notes", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp NoteListResponse
	decodeResponse(t, rec, &resp)
	assert.Empty(t, resp.Notes)
}

func TestListNotes_WithNotes(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "listuser", "list@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create 3 notes
	for i := 0; i < 3; i++ {
		rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
			Title:   "Note " + string(rune('A'+i)),
			Content: "Content",
		}, token)
		require.Equal(t, http.StatusCreated, rec.Code)
	}

	rec := doJSON(t, r, http.MethodGet, "/api/notes", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp NoteListResponse
	decodeResponse(t, rec, &resp)
	assert.Len(t, resp.Notes, 3)
}

func TestListNotes_FolderFilter(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "filtuser", "filt@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create notes in different folders
	doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Root Note", Content: "c", FolderPath: "/",
	}, token)
	doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Project Note", Content: "c", FolderPath: "/projects",
	}, token)

	rec := doJSON(t, r, http.MethodGet, "/api/notes?folder=/projects", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp NoteListResponse
	decodeResponse(t, rec, &resp)
	assert.Len(t, resp.Notes, 1)
	assert.Equal(t, "Project Note", resp.Notes[0].Title)
}

func TestUpdateNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "updateuser", "update@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Original Title",
		Content: "Original content",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)
	etag := rec.Header().Get("ETag")
	require.NotEmpty(t, etag)

	// Update note with If-Match
	rec = doJSONWithHeaders(t, r, http.MethodPut, "/api/notes/"+noteID, NoteRequest{
		Title:   "Updated Title",
		Content: "Updated content",
	}, token, map[string]string{"If-Match": etag})
	assert.Equal(t, http.StatusOK, rec.Code)
	var updated map[string]interface{}
	decodeResponse(t, rec, &updated)
	assert.Equal(t, "Updated Title", updated["title"])
}

func TestUpdateNote_MissingIfMatch(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "noetag", "noetag@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Title", Content: "Content",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Update without If-Match
	rec = doJSON(t, r, http.MethodPut, "/api/notes/"+noteID, NoteRequest{
		Title: "New Title", Content: "New content",
	}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "deluser", "del@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Delete Me", Content: "Trash bound",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Soft-delete
	rec = doJSON(t, r, http.MethodDelete, "/api/notes/"+noteID, nil, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Note no longer in list
	rec = doJSON(t, r, http.MethodGet, "/api/notes", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp NoteListResponse
	decodeResponse(t, rec, &resp)
	assert.Empty(t, resp.Notes)
}

func TestDeleteNote_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := notesRouter(ts)
	user := ts.createUser(t, "delnf", "delnf@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodDelete, "/api/notes/nonexistent", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
