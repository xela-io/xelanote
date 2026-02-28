//go:build fts5

package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func adminRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Use(ts.adminMiddleware)
	r.Route("/api/admin", func(r chi.Router) {
		r.Get("/stats", ts.getAdminStats)
		r.Get("/stats/detailed", ts.getDetailedStats)
		r.Get("/users", ts.listAllUsers)
		r.Get("/users/{id}", ts.getUserDetails)
		r.Put("/users/{id}/admin", ts.toggleUserAdmin)
		r.Put("/users/{id}/storage-limit", ts.setUserStorageLimit)
		r.Delete("/users/{id}", ts.deleteUserAdmin)
		r.Get("/settings", ts.getSettings)
		r.Put("/settings", ts.updateSettings)
	})
	return r
}

func (ts *testServer) makeAdmin(t *testing.T, userID int) {
	t.Helper()
	err := ts.db.SetUserAdmin(userID, true)
	require.NoError(t, err)
}

func TestAdminStats_Success(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodGet, "/api/admin/stats", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp AdminStatsResponse
	decodeResponse(t, rec, &resp)
	assert.Equal(t, 1, resp.TotalUsers)
}

func TestAdminStats_Forbidden(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	// First user is auto-promoted to admin, so create a dummy admin first
	ts.createUser(t, "admin", "admin@example.com", "password123")
	user := ts.createUser(t, "regular", "regular@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/admin/stats", nil, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAdminDetailedStats(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodGet, "/api/admin/stats/detailed", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp DetailedStatsResponse
	decodeResponse(t, rec, &resp)
	assert.Equal(t, 1, resp.Stats.TotalUsers)
}

func TestAdminListUsers(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	ts.createUser(t, "user2", "user2@example.com", "password123")
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodGet, "/api/admin/users", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var users []AdminUserResponse
	decodeResponse(t, rec, &users)
	assert.Len(t, users, 2)
}

func TestAdminGetUserDetails(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	target := ts.createUser(t, "target", "target@example.com", "password123")
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/admin/users/%d", target.User.ID), nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp AdminUserResponse
	decodeResponse(t, rec, &resp)
	assert.Equal(t, "target", resp.Username)
}

func TestAdminGetUserDetails_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodGet, "/api/admin/users/9999", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminToggleAdmin(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	target := ts.createUser(t, "target", "target@example.com", "password123")
	token := ts.getAuthToken(t, admin.User)

	// Promote target to admin (demotes current admin due to single-admin constraint)
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/admin/users/%d/admin", target.User.ID), SetAdminRequest{IsAdmin: true}, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify target is now admin (use target's token since original admin was demoted)
	targetToken := ts.getAuthToken(t, target.User)
	rec = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/admin/users/%d", target.User.ID), nil, targetToken)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp AdminUserResponse
	decodeResponse(t, rec, &resp)
	assert.True(t, resp.IsAdmin)
}

func TestAdminToggleAdmin_SelfDemotion(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/admin/users/%d/admin", admin.User.ID), SetAdminRequest{IsAdmin: false}, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAdminDeleteUser(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	target := ts.createUser(t, "todelete", "delete@example.com", "password123")
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/admin/users/%d", target.User.ID), nil, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// User should no longer exist
	rec = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/admin/users/%d", target.User.ID), nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAdminDeleteUser_Self(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/admin/users/%d", admin.User.ID), nil, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAdminGetSettings(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodGet, "/api/admin/settings", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var settings SettingsResponse
	decodeResponse(t, rec, &settings)
	assert.Equal(t, "true", settings["registration_enabled"])
}

func TestAdminUpdateSettings(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodPut, "/api/admin/settings", UpdateSettingsRequest{
		"maintenance_mode": "true",
	}, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var settings SettingsResponse
	decodeResponse(t, rec, &settings)
	assert.Equal(t, "true", settings["maintenance_mode"])
}

func TestAdminUpdateSettings_EmptyBody(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodPut, "/api/admin/settings", UpdateSettingsRequest{}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSetUserStorageLimit_Success(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	target := ts.createUser(t, "target", "target@example.com", "password123")
	token := ts.getAuthToken(t, admin.User)

	// Set a specific limit
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/admin/users/%d/storage-limit", target.User.ID),
		map[string]interface{}{"storage_limit_mb": 500}, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify it was set
	rec = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/admin/users/%d", target.User.ID), nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp AdminUserResponse
	decodeResponse(t, rec, &resp)
	require.NotNil(t, resp.StorageLimitMB)
	assert.Equal(t, 500, *resp.StorageLimitMB)

	// Clear the limit (set to null)
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/admin/users/%d/storage-limit", target.User.ID),
		map[string]interface{}{"storage_limit_mb": nil}, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify it was cleared
	rec = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/admin/users/%d", target.User.ID), nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	decodeResponse(t, rec, &resp)
	assert.Nil(t, resp.StorageLimitMB)
}

func TestSetUserStorageLimit_Forbidden(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	// First user is auto-promoted to admin, so create a dummy admin first
	ts.createUser(t, "admin", "admin@example.com", "password123")
	user := ts.createUser(t, "regular", "regular@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/admin/users/%d/storage-limit", user.User.ID),
		map[string]interface{}{"storage_limit_mb": 100}, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSetUserStorageLimit_InvalidNegative(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)
	admin := ts.createUser(t, "admin", "admin@example.com", "password123")
	ts.makeAdmin(t, admin.User.ID)
	target := ts.createUser(t, "target", "target@example.com", "password123")
	token := ts.getAuthToken(t, admin.User)

	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/admin/users/%d/storage-limit", target.User.ID),
		map[string]interface{}{"storage_limit_mb": -1}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdmin_Unauthorized(t *testing.T) {
	ts := newTestServer(t)
	r := adminRouter(ts)

	rec := doJSON(t, r, http.MethodGet, "/api/admin/stats", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
