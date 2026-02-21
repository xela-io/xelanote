//go:build fts5

package api

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func foldersRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/folders", func(r chi.Router) {
		r.Get("/", ts.getAllFolders)
		r.Post("/", ts.createFolder)
		r.Post("/reorder", ts.reorderFolders)
		r.Put("/{id}/move", ts.moveFolder)
		r.Put("/{id}/rename", ts.renameFolder)
		r.Put("/{id}/color", ts.updateFolderColor)
		r.Delete("/{id}", ts.deleteFolder)
	})
	return r
}

func TestCreateFolder_Success(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "folderuser", "folder@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{
		Path: "/Projects",
	}, token)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.Equal(t, "/Projects", resp["path"])
}

func TestCreateFolder_Nested(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "nestuser", "nest@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create parent folder first
	rec := doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{
		Path: "/Parent",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Create nested folder
	rec = doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{
		Path: "/Parent/Child",
	}, token)
	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.Equal(t, "/Parent/Child", resp["path"])
}

func TestCreateFolder_InvalidPath(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "badpath", "bad@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{
		Path: "",
	}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateFolder_Unauthorized(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)

	rec := doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{
		Path: "/Test",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetAllFolders_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "listuser", "list@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/folders", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	// folders key exists (value may be null when no folders exist)
	_, hasFolders := resp["folders"]
	assert.True(t, hasFolders, "response should have 'folders' key")
}

func TestGetAllFolders_WithFolders(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "listuser2", "list2@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create folders
	doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/A"}, token)
	doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/B"}, token)

	rec := doJSON(t, r, http.MethodGet, "/api/folders", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	folders := resp["folders"].([]interface{})
	assert.GreaterOrEqual(t, len(folders), 2)
}

func TestGetAllFolders_CrossUserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user1 := ts.createUser(t, "iso1", "iso1@example.com", "password123")
	user2 := ts.createUser(t, "iso2", "iso2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 creates a folder
	doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/Private"}, token1)

	// User2 should not see it (beyond default folders)
	rec := doJSON(t, r, http.MethodGet, "/api/folders", nil, token2)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	if folders, ok := resp["folders"].([]interface{}); ok {
		for _, f := range folders {
			folder := f.(map[string]interface{})
			assert.NotEqual(t, "/Private", folder["path"])
		}
	}
}

func TestRenameFolder_Success(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "renuser", "ren@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create folder
	rec := doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/OldName"}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	folderID := itoa(int(created["id"].(float64)))

	// Rename it
	rec = doJSON(t, r, http.MethodPut, "/api/folders/"+folderID+"/rename", RenameFolderRequest{
		NewName: "NewName",
	}, token)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRenameFolder_EmptyName(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "renblank", "renblank@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/ToRename"}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	folderID := itoa(int(created["id"].(float64)))

	rec = doJSON(t, r, http.MethodPut, "/api/folders/"+folderID+"/rename", RenameFolderRequest{
		NewName: "",
	}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteFolder_Success(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "deluser", "del@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create folder
	rec := doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/ToDelete"}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	folderID := itoa(int(created["id"].(float64)))

	// Delete it
	rec = doJSON(t, r, http.MethodDelete, "/api/folders/"+folderID, nil, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestMoveFolder_Success(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "moveuser", "move@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create two folders
	doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/Source"}, token)
	doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/Target"}, token)

	// Get source folder ID
	rec := doJSON(t, r, http.MethodGet, "/api/folders", nil, token)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	folders := resp["folders"].([]interface{})
	var sourceID string
	for _, f := range folders {
		folder := f.(map[string]interface{})
		if folder["path"] == "/Source" {
			sourceID = itoa(int(folder["id"].(float64)))
			break
		}
	}
	require.NotEmpty(t, sourceID)

	// Move /Source under /Target
	rec = doJSON(t, r, http.MethodPut, "/api/folders/"+sourceID+"/move", MoveFolderRequest{
		NewParentPath: "/Target",
	}, token)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMoveFolder_ToRoot(t *testing.T) {
	ts := newTestServer(t)
	r := foldersRouter(ts)
	user := ts.createUser(t, "rootmove", "rootmove@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create parent and child
	doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/Parent"}, token)
	rec := doJSON(t, r, http.MethodPost, "/api/folders", CreateFolderRequest{Path: "/Parent/Child"}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	childID := itoa(int(created["id"].(float64)))

	// Move child to root
	rec = doJSON(t, r, http.MethodPut, "/api/folders/"+childID+"/move", MoveFolderRequest{
		NewParentPath: "/",
	}, token)
	assert.Equal(t, http.StatusOK, rec.Code)
}
