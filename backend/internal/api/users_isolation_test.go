//go:build fts5

package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xela-io/xelanote/internal/service"
)

func init() {
	// EncryptAPIKey uses sync.Once — set env before any test runs.
	os.Setenv("XELANOTE_API_KEY_SECRET", "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
}

// usersRouter registers all /api/users/* routes without rate-limiting.
func usersRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/users", func(r chi.Router) {
		r.Get("/preferences", ts.getPreferences)
		r.Put("/preferences", ts.updatePreferences)
		r.Patch("/preferences", ts.patchPreferences)
		r.Put("/preferences/encryption", ts.updateEncryptionPreferences)
		r.Put("/preferences/security", ts.updateSecurityPreferences)
		r.Get("/ai-provider", ts.getAIProviderPreference)
		r.Put("/ai-provider", ts.setAIProviderPreference)
		r.Get("/dietary-preference", ts.getDietaryPreference)
		r.Put("/dietary-preference", ts.setDietaryPreference)
		r.Put("/email", ts.changeEmail)
		r.Post("/recovery-key", ts.setRecoveryKey)
		r.Get("/recovery-key/salt", ts.getRecoveryKeySalt)
		r.Post("/webauthn/credentials", ts.addWebAuthnCredential)
		r.Delete("/webauthn/credentials", ts.deleteWebAuthnCredential)

		// API key endpoints — use a no-op invalidateCache to avoid nil panic
		// (summarizeService is nil in tests).
		claudeKey := apiKeyProvider{
			name:      "claude",
			setKey:    ts.userService.SetClaudeAPIKey,
			deleteKey: ts.userService.DeleteClaudeAPIKey,
			getKeyStatus: func(uid int) (*apiKeyStatusResponse, error) {
				status, err := ts.userService.GetClaudeAPIKeyStatus(uid)
				if err != nil {
					return nil, err
				}
				return mapClaudeAPIKeyStatus(status), nil
			},
			invalidateCache: func(int) {}, // no-op
			validationErr:   service.ErrInvalidClaudeAPIKey,
			invalidKeyMsg:   "invalid Claude API key format (must start with sk-ant-)",
		}
		r.Put("/api-key", ts.handleSetAPIKey(claudeKey))
		r.Delete("/api-key", ts.handleDeleteAPIKey(claudeKey))
		r.Get("/api-key/status", ts.handleGetAPIKeyStatus(claudeKey))
	})
	return r
}

func TestPreferences_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "prefs1", "prefs1@example.com", "password123")
	user2 := ts.createUser(t, "prefs2", "prefs2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 sets theme to nord-dark (default is default-dark)
	rec := doJSON(t, r, http.MethodPut, "/api/users/preferences", updatePreferencesRequest{
		Theme:      "nord-dark",
		EditorMode: "edit",
	}, token1)
	require.Equal(t, http.StatusOK, rec.Code)

	// User2 reads preferences — should NOT see "nord-dark"
	rec = doJSON(t, r, http.MethodGet, "/api/users/preferences", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp2 map[string]interface{}
	decodeResponse(t, rec, &resp2)
	assert.NotEqual(t, "nord-dark", resp2["theme"], "User2 must not see User1's theme")

	// User1 still sees "nord-dark"
	rec = doJSON(t, r, http.MethodGet, "/api/users/preferences", nil, token1)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp1 map[string]interface{}
	decodeResponse(t, rec, &resp1)
	assert.Equal(t, "nord-dark", resp1["theme"])
}

func TestPatchPreferences_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "patch1", "patch1@example.com", "password123")
	user2 := ts.createUser(t, "patch2", "patch2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 sets home_dashboard_layout (must match expected schema)
	rec := doJSON(t, r, http.MethodPatch, "/api/users/preferences", patchPreferencesRequest{
		HomeDashboardLayout: json.RawMessage(`{"version":1,"collapsed_sections":{"hero":false,"recent":false,"activity":false,"created":false,"all":false},"right_section_order":["recent","activity","created","all"]}`),
	}, token1)
	require.Equal(t, http.StatusOK, rec.Code)

	// User2 reads preferences — home_dashboard_layout should be null
	rec = doJSON(t, r, http.MethodGet, "/api/users/preferences", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp2 map[string]interface{}
	decodeResponse(t, rec, &resp2)
	assert.Nil(t, resp2["home_dashboard_layout"], "User2 must not see User1's dashboard layout")
}

func TestAIProvider_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "ai1", "ai1@example.com", "password123")
	user2 := ts.createUser(t, "ai2", "ai2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 sets provider to gemini
	rec := doJSON(t, r, http.MethodPut, "/api/users/ai-provider", updateAIProviderRequest{
		Provider: "gemini",
	}, token1)
	require.Equal(t, http.StatusOK, rec.Code)

	// User2 reads AI provider — should get default (not "gemini")
	rec = doJSON(t, r, http.MethodGet, "/api/users/ai-provider", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp2 map[string]interface{}
	decodeResponse(t, rec, &resp2)
	assert.NotEqual(t, "gemini", resp2["provider"], "User2 must not see User1's AI provider")
}

func TestDietaryPreference_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "diet1", "diet1@example.com", "password123")
	user2 := ts.createUser(t, "diet2", "diet2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 sets dietary preference to vegan
	rec := doJSON(t, r, http.MethodPut, "/api/users/dietary-preference", updateDietaryPreferenceRequest{
		DietaryPreference: "vegan",
	}, token1)
	require.Equal(t, http.StatusOK, rec.Code)

	// User2 reads dietary preference — should see "none" (default)
	rec = doJSON(t, r, http.MethodGet, "/api/users/dietary-preference", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp2 map[string]interface{}
	decodeResponse(t, rec, &resp2)
	assert.NotEqual(t, "vegan", resp2["dietary_preference"], "User2 must not see User1's dietary preference")
}

func TestAPIKeyStatus_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "key1", "key1@example.com", "password123")
	user2 := ts.createUser(t, "key2", "key2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 stores a Claude API key
	rec := doJSON(t, r, http.MethodPut, "/api/users/api-key", map[string]string{
		"api_key": "sk-ant-api03-test1234567890abcdef",
	}, token1)
	require.Equal(t, http.StatusOK, rec.Code)

	// User1 should see has_key=true
	rec = doJSON(t, r, http.MethodGet, "/api/users/api-key/status", nil, token1)
	require.Equal(t, http.StatusOK, rec.Code)
	var status1 map[string]interface{}
	decodeResponse(t, rec, &status1)
	assert.Equal(t, true, status1["has_key"])

	// User2 should see has_key=false
	rec = doJSON(t, r, http.MethodGet, "/api/users/api-key/status", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	var status2 map[string]interface{}
	decodeResponse(t, rec, &status2)
	assert.Equal(t, false, status2["has_key"], "User2 must not see User1's API key")
}

func TestSecurityPreferences_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "sec1", "sec1@example.com", "password123")
	user2 := ts.createUser(t, "sec2", "sec2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 sets security_level to paranoid
	paranoid := "paranoid"
	rec := doJSON(t, r, http.MethodPut, "/api/users/preferences/security", updateSecurityPreferencesRequest{
		SecurityLevel: &paranoid,
	}, token1)
	require.Equal(t, http.StatusOK, rec.Code)

	// User2 reads preferences — should see default security_level ("balanced")
	rec = doJSON(t, r, http.MethodGet, "/api/users/preferences", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp2 map[string]interface{}
	decodeResponse(t, rec, &resp2)
	assert.Equal(t, "balanced", resp2["security_level"], "User2 must not see User1's security level")
}

func TestEncryptionPreferences_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "enc1", "enc1@example.com", "password123")
	user2 := ts.createUser(t, "enc2", "enc2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 tries to enable keywords indexing (must remain disabled)
	rec := doJSON(t, r, http.MethodPut, "/api/users/preferences/encryption", updateEncryptionPreferencesRequest{
		KeywordsEnabled: true,
	}, token1)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doJSON(t, r, http.MethodGet, "/api/users/preferences", nil, token1)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp1 map[string]interface{}
	decodeResponse(t, rec, &resp1)
	assert.Equal(t, false, resp1["keywords_enabled"], "keywords_enabled must stay disabled")

	// User2 reads preferences — keywords_enabled should be false (default)
	rec = doJSON(t, r, http.MethodGet, "/api/users/preferences", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp2 map[string]interface{}
	decodeResponse(t, rec, &resp2)
	assert.Equal(t, false, resp2["keywords_enabled"], "User2 must not see User1's encryption preferences")
}

func TestRecoveryKey_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "rec1", "rec1@example.com", "password123")
	user2 := ts.createUser(t, "rec2", "rec2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 sets a recovery key
	salt := base64.StdEncoding.EncodeToString([]byte("random-salt-bytes123"))
	rec := doJSON(t, r, http.MethodPost, "/api/users/recovery-key", setRecoveryKeyRequest{
		RecoveryKeyHash: "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012",
		Salt:            salt,
	}, token1)
	require.Equal(t, http.StatusOK, rec.Code)

	// User1 can retrieve salt
	rec = doJSON(t, r, http.MethodGet, "/api/users/recovery-key/salt", nil, token1)
	assert.Equal(t, http.StatusOK, rec.Code)

	// User2 gets 404 — no recovery key set
	rec = doJSON(t, r, http.MethodGet, "/api/users/recovery-key/salt", nil, token2)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWebAuthnCredentials_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "wa1", "wa1@example.com", "password123")
	user2 := ts.createUser(t, "wa2", "wa2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 adds a WebAuthn credential
	rec := doJSON(t, r, http.MethodPost, "/api/users/webauthn/credentials", addWebAuthnCredentialRequest{
		CredentialID: "cred-abc-123",
		DeviceName:   "User1 YubiKey",
	}, token1)
	require.Equal(t, http.StatusCreated, rec.Code)

	// User2 tries to delete User1's credential — returns 200 but is a no-op
	rec = doJSON(t, r, http.MethodDelete, "/api/users/webauthn/credentials?credential_id=cred-abc-123", nil, token2)
	assert.Equal(t, http.StatusOK, rec.Code)

	// User1's credential should still exist
	rec = doJSON(t, r, http.MethodGet, "/api/users/preferences", nil, token1)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp1 map[string]interface{}
	decodeResponse(t, rec, &resp1)
	creds, ok := resp1["webauthn_credentials"].([]interface{})
	require.True(t, ok, "webauthn_credentials should be a slice")
	assert.Equal(t, 1, len(creds), "User1's credential must still exist after User2's delete attempt")
}

func TestChangeEmail_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := usersRouter(ts)
	user1 := ts.createUser(t, "email1", "email1@example.com", "alpha123!")
	user2 := ts.createUser(t, "email2", "email2@example.com", "beta456!")
	_ = ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User2 tries to change email using User1's password — should fail (401)
	rec := doJSON(t, r, http.MethodPut, "/api/users/email", changeEmailRequest{
		NewEmail:        "hacked@example.com",
		CurrentPassword: "alpha123!",
	}, token2)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "User2 must not be able to use User1's password")
}
