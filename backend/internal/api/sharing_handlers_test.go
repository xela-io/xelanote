//go:build fts5

package api

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sharingRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api", func(r chi.Router) {
		r.Route("/notes", func(r chi.Router) {
			r.Post("/", ts.createNote)
			r.Post("/{id}/shares", ts.shareNote)
			r.Get("/{id}/shares", ts.getNoteShares)
			r.Put("/{id}/shares/{userId}", ts.updateShareRole)
			r.Delete("/{id}/shares/{userId}", ts.removeShare)
		})
		r.Get("/shared", ts.getSharedNotes)
		r.Get("/shared/{id}", ts.getSharedNote)
		r.Put("/shared/{id}", ts.updateSharedNote)
		r.Get("/users/search", ts.searchUsers)
	})
	return r
}

func TestShareNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	recipient := ts.createUser(t, "viewer", "viewer@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)

	// Create a note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Shared Note", Content: "Share this",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Share with recipient
	rec = doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: recipient.Username,
		Role:       "viewer",
	}, ownerToken)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestShareNote_InvalidRole(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	ts.createUser(t, "other", "other@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Note", Content: "c",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	rec = doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: "other",
		Role:       "admin", // invalid
	}, ownerToken)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestShareNote_SelfShare(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Note", Content: "c",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	rec = doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: "owner",
		Role:       "viewer",
	}, ownerToken)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestShareNote_NonOwner(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	other := ts.createUser(t, "other", "other@example.com", "password123")
	ts.createUser(t, "third", "third@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)
	otherToken := ts.getAuthToken(t, other.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Note", Content: "c",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Non-owner tries to share
	rec = doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: "third",
		Role:       "viewer",
	}, otherToken)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGetNoteShares_Success(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	ts.createUser(t, "viewer", "viewer@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Note", Content: "c",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Share it
	doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: "viewer", Role: "viewer",
	}, ownerToken)

	// Get shares
	rec = doJSON(t, r, http.MethodGet, "/api/notes/"+noteID+"/shares", nil, ownerToken)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	shares := resp["shares"].([]interface{})
	assert.Len(t, shares, 1)
}

func TestGetSharedNotes_Recipient(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	recipient := ts.createUser(t, "viewer", "viewer@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)
	recipientToken := ts.getAuthToken(t, recipient.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Shared Note", Content: "shared content",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Share with recipient
	doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: "viewer", Role: "viewer",
	}, ownerToken)

	// Recipient sees it in shared notes
	rec = doJSON(t, r, http.MethodGet, "/api/shared", nil, recipientToken)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	notes := resp["notes"].([]interface{})
	assert.Len(t, notes, 1)
}

func TestGetSharedNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	recipient := ts.createUser(t, "viewer", "viewer@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)
	recipientToken := ts.getAuthToken(t, recipient.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "View This", Content: "readable content",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: "viewer", Role: "viewer",
	}, ownerToken)

	// Recipient reads shared note
	rec = doJSON(t, r, http.MethodGet, "/api/shared/"+noteID, nil, recipientToken)
	assert.Equal(t, http.StatusOK, rec.Code)
	var note map[string]interface{}
	decodeResponse(t, rec, &note)
	assert.Equal(t, "View This", note["title"])
}

func TestGetSharedNote_NotShared(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	other := ts.createUser(t, "other", "other@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)
	otherToken := ts.getAuthToken(t, other.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Private", Content: "not shared",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Other user cannot read it
	rec = doJSON(t, r, http.MethodGet, "/api/shared/"+noteID, nil, otherToken)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRemoveShare_Success(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	recipient := ts.createUser(t, "viewer", "viewer@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)
	recipientToken := ts.getAuthToken(t, recipient.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Unshare Me", Content: "c",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Share
	doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: "viewer", Role: "viewer",
	}, ownerToken)

	// Remove share
	rec = doJSON(t, r, http.MethodDelete, "/api/notes/"+noteID+"/shares/"+itoa(recipient.User.ID), nil, ownerToken)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Recipient no longer has access
	rec = doJSON(t, r, http.MethodGet, "/api/shared/"+noteID, nil, recipientToken)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateShareRole_Success(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	recipient := ts.createUser(t, "viewer", "viewer@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title: "Role Update", Content: "c",
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Share as viewer
	doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: "viewer", Role: "viewer",
	}, ownerToken)

	// Upgrade to editor
	rec = doJSON(t, r, http.MethodPut, "/api/notes/"+noteID+"/shares/"+itoa(recipient.User.ID), UpdateShareRoleRequest{
		Role: "editor",
	}, ownerToken)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestSearchUsers_Success(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	ts.createUser(t, "alice", "alice@example.com", "password123")
	searcher := ts.createUser(t, "bob", "bob@example.com", "password123")
	searcherToken := ts.getAuthToken(t, searcher.User)

	rec := doJSON(t, r, http.MethodGet, "/api/users/search?q=alice", nil, searcherToken)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	users := resp["users"].([]interface{})
	assert.NotEmpty(t, users)
}

func TestSearchUsers_TooShortQuery(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	user := ts.createUser(t, "bob", "bob@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/users/search?q=ab", nil, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSearchUsers_ExcludesSelf(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	user := ts.createUser(t, "searchme", "searchme@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/users/search?q=searchme", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	users := resp["users"].([]interface{})
	assert.Empty(t, users)
}

func TestGetSharedNotes_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	user := ts.createUser(t, "lonely", "lonely@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/shared", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	notes := resp["notes"].([]interface{})
	assert.Empty(t, notes)
}

func TestShareEncryptedNote_Blocked(t *testing.T) {
	ts := newTestServer(t)
	r := sharingRouter(ts)

	owner := ts.createUser(t, "owner", "owner@example.com", "password123")
	ts.createUser(t, "viewer", "viewer@example.com", "password123")
	ownerToken := ts.getAuthToken(t, owner.User)

	encContent, wrappedDEK, meta := makeEncryptedPayload(t)

	// Create encrypted note
	encRouter := encryptionRouter(ts)
	rec := doJSON(t, encRouter, http.MethodPost, "/api/notes", NoteRequest{
		ID:                 testNoteID(31),
		Title:              "Secret Note",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
	}, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Sharing encrypted notes should be blocked
	rec = doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/shares", ShareNoteRequest{
		Identifier: "viewer",
		Role:       "viewer",
	}, ownerToken)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
