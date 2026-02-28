//go:build fts5

package api

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func validAPIWrappedDEK(seed byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{seed}, 48))
}

func recoveryRouter(ts *testServer) chi.Router {
	r := chi.NewRouter()
	r.Post("/api/auth/recovery/verify", ts.verifyRecoveryKey)
	r.Get("/api/auth/recovery/encrypted-deks", ts.getRecoveryWrappedDEKs)
	r.Post("/api/auth/recovery/reset-password-v2", ts.resetPasswordWithRecoveryToken)
	return r
}

func TestRecoveryResetTokenFlow_EncryptedAccount(t *testing.T) {
	ts := newTestServer(t)
	r := recoveryRouter(ts)

	user := ts.createUser(t, "recoveryv2", "recoveryv2@example.com", "oldpassword1")
	require.NoError(t, ts.db.SetUserEncryptionSalt(user.ID, []byte("0123456789abcdef")))

	recoveryKey := "api-recovery-key"
	hash, err := bcrypt.GenerateFromPassword([]byte(recoveryKey), 12)
	require.NoError(t, err)
	require.NoError(t, ts.db.SetRecoveryKey(user.ID, string(hash), []byte("salt-bytes")))

	_, err = ts.db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
		VALUES ('api-rec-note-1', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, ?, ?, 2)
	`, user.ID, validAPIWrappedDEK(1), validAPIWrappedDEK(2))
	require.NoError(t, err)

	_, err = ts.db.Exec(`
		INSERT INTO note_versions (id, note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
		VALUES (201, 'api-rec-note-1', ?, 1, 'V1', '', datetime('now'), 1, ?, ?, 2)
	`, user.ID, validAPIWrappedDEK(3), validAPIWrappedDEK(4))
	require.NoError(t, err)

	verifyRec := doJSON(t, r, http.MethodPost, "/api/auth/recovery/verify", RecoveryVerifyRequest{
		Email:       "recoveryv2@example.com",
		RecoveryKey: recoveryKey,
	}, "")
	require.Equal(t, http.StatusOK, verifyRec.Code)

	var verifyResp recoveryVerifyResponse
	decodeResponse(t, verifyRec, &verifyResp)
	require.NotEmpty(t, verifyResp.RecoveryResetToken)
	require.NotEmpty(t, verifyResp.EncryptionSalt)

	wrappedRec := doJSONWithHeaders(
		t,
		r,
		http.MethodGet,
		"/api/auth/recovery/encrypted-deks",
		nil,
		"",
		map[string]string{"Authorization": "Bearer " + verifyResp.RecoveryResetToken},
	)
	require.Equal(t, http.StatusOK, wrappedRec.Code)

	var wrappedResp recoveryWrappedDEKsResponse
	decodeResponse(t, wrappedRec, &wrappedResp)
	require.Len(t, wrappedResp.Notes, 1)
	require.Len(t, wrappedResp.Versions, 1)

	finalizeRec := doJSON(t, r, http.MethodPost, "/api/auth/recovery/reset-password-v2", RecoveryResetPasswordWithTokenRequest{
		RecoveryResetToken: verifyResp.RecoveryResetToken,
		NewPassword:        "newpassword1",
		ReWrappedNoteDEKs: map[string]string{
			"api-rec-note-1": validAPIWrappedDEK(9),
		},
		ReWrappedVersionDEKs: map[string]string{
			"201": validAPIWrappedDEK(8),
		},
	}, "")
	require.Equal(t, http.StatusOK, finalizeRec.Code)

	reuseRec := doJSON(t, r, http.MethodPost, "/api/auth/recovery/reset-password-v2", RecoveryResetPasswordWithTokenRequest{
		RecoveryResetToken: verifyResp.RecoveryResetToken,
		NewPassword:        "anotherpassword1",
	}, "")
	require.Equal(t, http.StatusUnauthorized, reuseRec.Code)
}
