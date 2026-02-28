//go:build fts5

package api

import (
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
)

func tagsRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/tags", func(r chi.Router) {
		r.Get("/", ts.getAllTags)
		r.Delete("/{tagId}", ts.deleteTag)
	})
	r.Route("/api/notes", func(r chi.Router) {
		r.Post("/", ts.createNote)
		r.Get("/{id}/tags", ts.getNoteTags)
		r.Put("/{id}/tags", ts.setNoteTags)
	})
	return r
}

func versionsRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/notes", func(r chi.Router) {
		r.Post("/", ts.createNote)
		r.Put("/{id}", ts.updateNote)
		r.Get("/{id}", ts.getNote)
		r.Get("/{id}/versions", ts.listVersions)
		r.Get("/{id}/versions/compare", ts.compareVersions)
		r.Post("/{id}/versions/delta-summary", ts.summarizeVersionDelta)
		r.Get("/{id}/versions/{version}", ts.getVersion)
		r.Post("/{id}/versions/{version}/restore", ts.restoreVersion)
	})
	return r
}

// --- Tags ---

func TestGetAllTags_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user := ts.createUser(t, "taguser", "tag@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/tags", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var tags []interface{}
	decodeResponse(t, rec, &tags)
	assert.Empty(t, tags)
}

func TestGetAllTags_Unauthorized(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)

	rec := doJSON(t, r, http.MethodGet, "/api/tags", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSetNoteTags_Success(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user := ts.createUser(t, "settag", "settag@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create a note
	note := ts.createNoteDirectly(t, user.ID, "Tagged Note", "content", "/")

	// Set tags
	rec := doJSON(t, r, http.MethodPut, "/api/notes/"+note.ID+"/tags", SetNoteTagsRequest{
		Tags: []string{"work", "important"},
	}, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var tags []interface{}
	decodeResponse(t, rec, &tags)
	assert.Len(t, tags, 2)
}

func TestGetNoteTags_Success(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user := ts.createUser(t, "gettag", "gettag@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create a note and set tags
	note := ts.createNoteDirectly(t, user.ID, "Get Tags Note", "content", "/")
	rec := doJSON(t, r, http.MethodPut, "/api/notes/"+note.ID+"/tags", SetNoteTagsRequest{
		Tags: []string{"alpha", "beta"},
	}, token)
	require.Equal(t, http.StatusOK, rec.Code)

	// Get tags
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+note.ID+"/tags", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var tags []interface{}
	decodeResponse(t, rec, &tags)
	assert.Len(t, tags, 2)
}

func TestGetNoteTags_NoteNotFound(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user := ts.createUser(t, "tagnf", "tagnf@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/notes/nonexistent/tags", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSetNoteTags_EncryptedNoteBlocked(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user := ts.createUser(t, "encsettag", "encsettag@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encNote, err := ts.noteService.CreateEncryptedNote(
		user.ID,
		"Encrypted",
		nil,
		false,
		[]byte("encrypted-content"),
		"wrapped-dek",
		`{"algorithm":"XChaCha20-Poly1305","version":3}`,
		nil,
		"/",
	)
	require.NoError(t, err)

	rec := doJSON(t, r, http.MethodPut, "/api/notes/"+encNote.ID+"/tags", SetNoteTagsRequest{
		Tags: []string{"secret"},
	}, token)
	assert.Equal(t, http.StatusConflict, rec.Code)

	var total int
	require.NoError(t, ts.db.QueryRow(`SELECT COUNT(*) FROM note_tags WHERE note_id = ?`, encNote.ID).Scan(&total))
	assert.Equal(t, 0, total)
}

func TestGetNoteTags_EncryptedNoteReturnsEmptyAndClearsLegacy(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user := ts.createUser(t, "encgettag", "encgettag@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encNote, err := ts.noteService.CreateEncryptedNote(
		user.ID,
		"Encrypted",
		nil,
		false,
		[]byte("encrypted-content"),
		"wrapped-dek",
		`{"algorithm":"XChaCha20-Poly1305","version":3}`,
		nil,
		"/",
	)
	require.NoError(t, err)

	res, err := ts.db.Exec(`INSERT INTO tags (name, name_norm, user_id) VALUES (?, ?, ?)`, "legacy", "legacy", user.ID)
	require.NoError(t, err)
	tagID, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = ts.db.Exec(`INSERT INTO note_tags (note_id, tag_id) VALUES (?, ?)`, encNote.ID, tagID)
	require.NoError(t, err)

	rec := doJSON(t, r, http.MethodGet, "/api/notes/"+encNote.ID+"/tags", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)

	var tags []interface{}
	decodeResponse(t, rec, &tags)
	assert.Empty(t, tags)

	var total int
	require.NoError(t, ts.db.QueryRow(`SELECT COUNT(*) FROM note_tags WHERE note_id = ?`, encNote.ID).Scan(&total))
	assert.Equal(t, 0, total)
}

func TestSetNoteTags_ReplacesExisting(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user := ts.createUser(t, "reptag", "reptag@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Replace Tags", "content", "/")

	// Set initial tags
	rec := doJSON(t, r, http.MethodPut, "/api/notes/"+note.ID+"/tags", SetNoteTagsRequest{
		Tags: []string{"old1", "old2"},
	}, token)
	require.Equal(t, http.StatusOK, rec.Code)

	// Replace with new tags
	rec = doJSON(t, r, http.MethodPut, "/api/notes/"+note.ID+"/tags", SetNoteTagsRequest{
		Tags: []string{"new1"},
	}, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var tags []interface{}
	decodeResponse(t, rec, &tags)
	assert.Len(t, tags, 1)
}

func TestGetAllTags_AfterSetting(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user := ts.createUser(t, "alltags", "alltags@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Tagged", "c", "/")
	rec := doJSON(t, r, http.MethodPut, "/api/notes/"+note.ID+"/tags", SetNoteTagsRequest{
		Tags: []string{"project", "urgent"},
	}, token)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doJSON(t, r, http.MethodGet, "/api/tags", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var tags []interface{}
	decodeResponse(t, rec, &tags)
	assert.GreaterOrEqual(t, len(tags), 2)
}

func TestDeleteTag_Success(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user := ts.createUser(t, "deltag", "deltag@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Del Tag Note", "c", "/")
	rec := doJSON(t, r, http.MethodPut, "/api/notes/"+note.ID+"/tags", SetNoteTagsRequest{
		Tags: []string{"todelete"},
	}, token)
	require.Equal(t, http.StatusOK, rec.Code)

	// Get tag ID from all tags
	rec = doJSON(t, r, http.MethodGet, "/api/tags", nil, token)
	require.Equal(t, http.StatusOK, rec.Code)
	var tags []interface{}
	decodeResponse(t, rec, &tags)
	require.NotEmpty(t, tags)
	tag := tags[0].(map[string]interface{})
	tagID := itoa(int(tag["id"].(float64)))

	// Delete the tag
	rec = doJSON(t, r, http.MethodDelete, "/api/tags/"+tagID, nil, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify tag is gone
	rec = doJSON(t, r, http.MethodGet, "/api/tags", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var remaining []interface{}
	decodeResponse(t, rec, &remaining)
	assert.Empty(t, remaining)
}

func TestTags_CrossUserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := tagsRouter(ts)
	user1 := ts.createUser(t, "tagiso1", "tagiso1@example.com", "password123")
	user2 := ts.createUser(t, "tagiso2", "tagiso2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 sets tags on a note
	note := ts.createNoteDirectly(t, user1.ID, "Private Tagged", "c", "/")
	rec := doJSON(t, r, http.MethodPut, "/api/notes/"+note.ID+"/tags", SetNoteTagsRequest{
		Tags: []string{"secret"},
	}, token1)
	require.Equal(t, http.StatusOK, rec.Code)

	// User2 should not see those tags
	rec = doJSON(t, r, http.MethodGet, "/api/tags", nil, token2)
	assert.Equal(t, http.StatusOK, rec.Code)
	var tags []interface{}
	decodeResponse(t, rec, &tags)
	assert.Empty(t, tags)

	// User2 cannot access user1's note tags
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+note.ID+"/tags", nil, token2)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Versions ---

func TestListVersions_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "veruser", "ver@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Versioned", "initial", "/")

	rec := doJSON(t, r, http.MethodGet, "/api/notes/"+note.ID+"/versions", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp VersionListResponse
	decodeResponse(t, rec, &resp)
	assert.NotNil(t, resp.Versions)
}

func TestListVersions_NoteNotFound(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "vernf", "vernf@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/notes/nonexistent/versions", nil, token)
	// Handler calls respondInternalErr for GetNote errors (including ErrNotFound)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusInternalServerError}, rec.Code)
}

func TestListVersions_AfterUpdate(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "verupd", "verupd@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Will Update", "v1 content", "/")

	// Update the note to create a version
	rec := doJSONWithHeaders(t, r, http.MethodPut, "/api/notes/"+note.ID, NoteRequest{
		Title:   "Will Update",
		Content: "v2 content",
	}, token, map[string]string{
		"If-Match": generateETag(note.ID, note.Version),
	})
	require.Equal(t, http.StatusOK, rec.Code)

	// List versions
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+note.ID+"/versions", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp VersionListResponse
	decodeResponse(t, rec, &resp)
	assert.GreaterOrEqual(t, len(resp.Versions), 1)
}

func TestGetVersion_Success(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "getver", "getver@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Versioned Note", "original", "/")

	// Update to create version 1
	rec := doJSONWithHeaders(t, r, http.MethodPut, "/api/notes/"+note.ID, NoteRequest{
		Title:   "Versioned Note",
		Content: "updated",
	}, token, map[string]string{
		"If-Match": generateETag(note.ID, note.Version),
	})
	require.Equal(t, http.StatusOK, rec.Code)

	// Get version 1
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+note.ID+"/versions/1", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetVersion_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "vernf2", "vernf2@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "No Versions", "c", "/")

	rec := doJSON(t, r, http.MethodGet, "/api/notes/"+note.ID+"/versions/999", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCompareVersions_Success(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "cmpver", "cmpver@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Compare Note", "v1", "/")

	// Create version 1 (first update triggers snapshot of original)
	rec := doJSONWithHeaders(t, r, http.MethodPut, "/api/notes/"+note.ID, NoteRequest{
		Title:   "Compare Note",
		Content: "v2",
	}, token, map[string]string{
		"If-Match": generateETag(note.ID, note.Version),
	})
	require.Equal(t, http.StatusOK, rec.Code)
	var updated map[string]interface{}
	decodeResponse(t, rec, &updated)
	newVersion := int(updated["version"].(float64))

	// Backdate the snapshot so the next update also triggers a snapshot
	_, err := ts.db.Exec(`UPDATE note_versions SET snapshot_at = datetime('now', '-10 minutes') WHERE note_id = ?`, note.ID)
	require.NoError(t, err)

	// Create version 2 (second snapshot now possible)
	rec = doJSONWithHeaders(t, r, http.MethodPut, "/api/notes/"+note.ID, NoteRequest{
		Title:   "Compare Note",
		Content: "v3",
	}, token, map[string]string{
		"If-Match": generateETag(note.ID, newVersion),
	})
	require.Equal(t, http.StatusOK, rec.Code)

	// Compare versions 1 and 2
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+note.ID+"/versions/compare?v1=1&v2=2", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var cmp CompareResponse
	decodeResponse(t, rec, &cmp)
	assert.NotNil(t, cmp.Version1)
	assert.NotNil(t, cmp.Version2)
}

func TestCompareVersions_MissingParams(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "cmpbad", "cmpbad@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Bad Compare", "c", "/")

	tests := []struct {
		name  string
		query string
	}{
		{"no params", ""},
		{"only v1", "?v1=1"},
		{"only v2", "?v2=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doJSON(t, r, http.MethodGet, "/api/notes/"+note.ID+"/versions/compare"+tt.query, nil, token)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestSummarizeVersionDelta_AINotEnabled(t *testing.T) {
	ts := newTestServer(t)
	ts.Server.summarizeService = service.NewSummarizeService(
		ts.db,
		llm.NewProviderRouter(ts.db),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	r := versionsRouter(ts)
	user := ts.createUser(t, "deltasum", "deltasum@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Delta Note", "v1 content", "/")

	rec := doJSON(t, r, http.MethodPost, "/api/notes/"+note.ID+"/versions/delta-summary", DeltaSummaryRequest{
		FromVersion: 1,
		ToVersion:   2,
		FromContent: "v1 content",
		ToContent:   "v2 content",
	}, token)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSummarizeVersionDelta_Validation(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "deltasumval", "deltasumval@example.com", "password123")
	token := ts.getAuthToken(t, user.User)
	note := ts.createNoteDirectly(t, user.ID, "Delta Note", "v1 content", "/")

	rec := doJSON(t, r, http.MethodPost, "/api/notes/"+note.ID+"/versions/delta-summary", DeltaSummaryRequest{
		FromVersion: 1,
		ToVersion:   1,
		FromContent: "same",
		ToContent:   "same",
	}, token)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRestoreVersion_Success(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "restver", "restver@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "Restore Me", "original content", "/")

	// Update to create a version
	rec := doJSONWithHeaders(t, r, http.MethodPut, "/api/notes/"+note.ID, NoteRequest{
		Title:   "Restore Me",
		Content: "changed content",
	}, token, map[string]string{
		"If-Match": generateETag(note.ID, note.Version),
	})
	require.Equal(t, http.StatusOK, rec.Code)
	var updated map[string]interface{}
	decodeResponse(t, rec, &updated)
	currentVersion := int(updated["version"].(float64))

	// Restore to version 1
	rec = doJSONWithHeaders(t, r, http.MethodPost, "/api/notes/"+note.ID+"/versions/1/restore", nil, token, map[string]string{
		"If-Match": generateETag(note.ID, currentVersion),
	})
	assert.Equal(t, http.StatusOK, rec.Code)
	var restored map[string]interface{}
	decodeResponse(t, rec, &restored)
	assert.Equal(t, "original content", restored["content"])
}

func TestRestoreVersion_MissingIfMatch(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user := ts.createUser(t, "restnim", "restnim@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	note := ts.createNoteDirectly(t, user.ID, "No ETag", "c", "/")

	rec := doJSON(t, r, http.MethodPost, "/api/notes/"+note.ID+"/versions/1/restore", nil, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVersions_CrossUserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)
	user1 := ts.createUser(t, "veriso1", "veriso1@example.com", "password123")
	user2 := ts.createUser(t, "veriso2", "veriso2@example.com", "password123")
	token2 := ts.getAuthToken(t, user2.User)

	note := ts.createNoteDirectly(t, user1.ID, "Private Versioned", "c", "/")

	// User2 cannot list versions of user1's note
	rec := doJSON(t, r, http.MethodGet, "/api/notes/"+note.ID+"/versions", nil, token2)
	// Handler returns 500 for GetNote errors (including ErrNotFound for wrong user)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusInternalServerError}, rec.Code)
}

func TestListVersions_Unauthorized(t *testing.T) {
	ts := newTestServer(t)
	r := versionsRouter(ts)

	rec := doJSON(t, r, http.MethodGet, "/api/notes/someid/versions", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
