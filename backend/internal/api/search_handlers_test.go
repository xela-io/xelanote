//go:build fts5

package api

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func searchRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api", func(r chi.Router) {
		r.Get("/search", ts.search)
		r.Get("/quick-search", ts.quickSearch)
		r.Get("/folders-legacy", ts.getFolders)
		r.Route("/notes", func(r chi.Router) {
			r.Post("/", ts.createNote)
			r.Get("/titles", ts.listNoteTitles)
			r.Get("/{id}/backlinks", ts.getBacklinks)
		})
	})
	return r
}

func TestSearch_EmptyQuery(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "searchuser", "search@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/search", nil, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSearch_NoResults(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "searchuser", "search@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/search?q=nonexistent", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	results := resp["results"].([]interface{})
	assert.Empty(t, results)
	assert.Equal(t, "nonexistent", resp["query"])
}

func TestSearch_FindsNote(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "searchuser", "search@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create a note with searchable content
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Quantum Computing Basics",
		Content: "Quantum computing uses qubits instead of classical bits",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Search for it
	rec = doJSON(t, r, http.MethodGet, "/api/search?q=quantum", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	results := resp["results"].([]interface{})
	assert.NotEmpty(t, results)
}

func TestSearch_CrossUserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)

	user1 := ts.createUser(t, "owner", "owner@example.com", "password123")
	user2 := ts.createUser(t, "other", "other@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 creates a note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Secret Research",
		Content: "Confidential material about plutonium",
	}, token1)
	require.Equal(t, http.StatusCreated, rec.Code)

	// User2 cannot find it
	rec = doJSON(t, r, http.MethodGet, "/api/search?q=plutonium", nil, token2)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	results := resp["results"].([]interface{})
	assert.Empty(t, results)
}

func TestSearch_WithLimit(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "searchuser", "search@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create multiple notes
	for i := 0; i < 5; i++ {
		doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
			Title:   "Searchable Note",
			Content: "This note contains a unique keyword xyzzy",
		}, token)
	}

	rec := doJSON(t, r, http.MethodGet, "/api/search?q=xyzzy&limit=2", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	results := resp["results"].([]interface{})
	assert.LessOrEqual(t, len(results), 2)
}

func TestQuickSearch_EmptyQuery(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "qsuser", "qs@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Empty query should return recent notes
	rec := doJSON(t, r, http.MethodGet, "/api/quick-search?q=", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.NotNil(t, resp["notes"])
}

func TestQuickSearch_WithFolderFilter(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "qsuser", "qs@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// FilteredSearch matches on title_norm (not content), so use a searchable title keyword
	doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Root Omega Note", Content: "filterable content", FolderPath: "/",
	}, token)
	doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Project Omega Note", Content: "filterable content", FolderPath: "/projects",
	}, token)

	rec := doJSON(t, r, http.MethodGet, "/api/quick-search?q=omega&folders=/projects", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	notes := resp["notes"].([]interface{})
	assert.Len(t, notes, 1)
}

func TestGetFolders_DefaultRoot(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "flduser", "fld@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/folders-legacy", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	folders := resp["folders"].([]interface{})
	assert.NotEmpty(t, folders)
}

func TestGetFolders_WithNotes(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "flduser", "fld@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Note in Work", Content: "c", FolderPath: "/work",
	}, token)
	doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Note in Personal", Content: "c", FolderPath: "/personal",
	}, token)

	rec := doJSON(t, r, http.MethodGet, "/api/folders-legacy", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	folders := resp["folders"].([]interface{})
	assert.GreaterOrEqual(t, len(folders), 2)
}

func TestNoteTitles_Success(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "titleuser", "title@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "First Note", Content: "content",
	}, token)
	doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Second Note", Content: "content",
	}, token)

	rec := doJSON(t, r, http.MethodGet, "/api/notes/titles", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Titles []NoteTitleInfo `json:"titles"`
	}
	decodeResponse(t, rec, &resp)
	assert.Len(t, resp.Titles, 2)
}

func TestBacklinks_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "bluser", "bl@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create a note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Target Note", Content: "target content",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Get backlinks (should be empty)
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+noteID+"/backlinks", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	backlinks := resp["backlinks"].([]interface{})
	assert.Empty(t, backlinks)
}

func TestBacklinks_WithLink(t *testing.T) {
	ts := newTestServer(t)
	r := searchRouter(ts)
	user := ts.createUser(t, "bluser", "bl@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create target note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Target Note", Content: "I am the target",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var target map[string]interface{}
	decodeResponse(t, rec, &target)
	targetID := target["id"].(string)

	// Create source note that links to target via wikilink in content
	rec = doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Source Note",
		Content: "This links to [[Target Note]]",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Get backlinks for target
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+targetID+"/backlinks", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	backlinks := resp["backlinks"].([]interface{})
	assert.Len(t, backlinks, 1)
}
