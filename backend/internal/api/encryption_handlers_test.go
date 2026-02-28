//go:build fts5

package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encryptionRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/notes", func(r chi.Router) {
		r.Post("/", ts.createNote)
		r.Get("/{id}", ts.getNote)
		r.Put("/{id}", ts.updateNote)
		r.Post("/{id}/decrypt", ts.decryptNote)
		r.Post("/batch-reencrypt-deks", ts.batchReencryptDEKs)
	})
	return r
}

// makeEncryptedPayload creates a valid encrypted note payload for testing.
func makeEncryptedPayload(t *testing.T) (encryptedContent, wrappedDEK, metadata string) {
	t.Helper()

	// Generate a fake AES key (32 bytes for AES-256)
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	// Encrypt some content with AES-GCM
	block, err := aes.NewCipher(dek)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)
	ciphertext := gcm.Seal(nonce, nonce, []byte("encrypted content here"), nil)

	encryptedContent = base64.StdEncoding.EncodeToString(ciphertext)

	// Wrap the DEK (in test, just base64 encode it — min 32 bytes)
	wrappedDEK = base64.StdEncoding.EncodeToString(dek)

	// Encryption metadata (valid JSON)
	meta := map[string]interface{}{
		"algorithm": "AES-256-GCM",
		"version":   1,
	}
	metaBytes, _ := json.Marshal(meta)
	metadata = string(metaBytes)

	return
}

// makeEncryptedTitlePayload creates a valid encrypted_title JSON payload that matches frontend format.
func makeEncryptedTitlePayload(t *testing.T, version int) string {
	t.Helper()

	ciphertext := make([]byte, 48)
	_, err := rand.Read(ciphertext)
	require.NoError(t, err)

	wrappedDEK := make([]byte, 32)
	_, err = rand.Read(wrappedDEK)
	require.NoError(t, err)

	payload := map[string]interface{}{
		"ciphertext": base64.StdEncoding.EncodeToString(ciphertext),
		"metadata": map[string]interface{}{
			"version":      version,
			"algorithm":    "XChaCha20-Poly1305",
			"kdf":          "Argon2id",
			"kdf_strength": "interactive",
			"nonce_bytes":  24,
			"wrapped_dek":  base64.StdEncoding.EncodeToString(wrappedDEK),
		},
	}

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)

	return string(encoded)
}

func countRowsForNote(t *testing.T, ts *testServer, query string, noteID string) int {
	t.Helper()

	var count int
	err := ts.db.QueryRow(query, noteID).Scan(&count)
	require.NoError(t, err)
	return count
}

func TestCreateEncryptedNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "encuser", "enc@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, wrappedDEK, meta := makeEncryptedPayload(t)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:              "Encrypted Note",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
	}, token)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var note map[string]interface{}
	decodeResponse(t, rec, &note)
	assert.Equal(t, "Encrypted Note", note["title"])
	assert.Equal(t, true, note["content_encrypted"])
	assert.NotEmpty(t, note["wrapped_dek"])
}

func TestCreateEncryptedNote_WithEncryptedTitle_Success(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "encuser-title", "enc-title@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, wrappedDEK, meta := makeEncryptedPayload(t)
	encryptedTitle := makeEncryptedTitlePayload(t, 2)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:              "",
		EncryptedTitle:     &encryptedTitle,
		TitleEncrypted:     true,
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
	}, token)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var note map[string]interface{}
	decodeResponse(t, rec, &note)
	assert.Equal(t, true, note["title_encrypted"])
	assert.Equal(t, encryptedTitle, note["encrypted_title"])
}

func TestCreateEncryptedNote_IgnoresClientLinksAndDueDates(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "encuser-meta", "enc-meta@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, wrappedDEK, meta := makeEncryptedPayload(t)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:              "Encrypted Note",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
		Links: []ClientLink{
			{TargetTitle: "ShouldNotBeStored"},
		},
		DueDates: []ClientDueDate{
			{
				DueDate:     "2026-03-01",
				LineText:    "- [ ] hidden metadata",
				LineIndex:   0,
				IsTaskItem:  true,
				IsCompleted: false,
			},
		},
		Keywords: []string{"secret", "metadata"},
	}, token)

	require.Equal(t, http.StatusCreated, rec.Code)

	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	linksCount := countRowsForNote(t, ts, "SELECT COUNT(*) FROM links WHERE source_id = ?", noteID)
	unresolvedCount := countRowsForNote(t, ts, "SELECT COUNT(*) FROM unresolved_links WHERE source_id = ?", noteID)
	dueDatesCount := countRowsForNote(t, ts, "SELECT COUNT(*) FROM note_due_dates WHERE note_id = ?", noteID)
	keywordsCount := countRowsForNote(t, ts, "SELECT COUNT(*) FROM note_keywords WHERE note_id = ?", noteID)

	assert.Equal(t, 0, linksCount)
	assert.Equal(t, 0, unresolvedCount)
	assert.Equal(t, 0, dueDatesCount)
	assert.Equal(t, 0, keywordsCount)
}

func TestCreateEncryptedNote_InvalidDEK(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "encuser", "enc@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, _, meta := makeEncryptedPayload(t)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:              "Bad DEK Note",
		EncryptedContent:   encContent,
		WrappedDEK:         "not-valid-base64!!!",
		EncryptionMetadata: meta,
	}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateEncryptedNote_ClientProvidedID(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "encuser", "enc@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, wrappedDEK, _ := makeEncryptedPayload(t)
	providedID := "550e8400-e29b-41d4-a716-446655440000"
	meta := `{"algorithm":"XChaCha20-Poly1305","version":3}`

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		ID:                 providedID,
		Title:              "Encrypted Note",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
	}, token)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var note map[string]interface{}
	decodeResponse(t, rec, &note)
	assert.Equal(t, providedID, note["id"])
}

func TestCreateEncryptedNote_V3MissingIDRejected(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "encuser", "enc@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, wrappedDEK, _ := makeEncryptedPayload(t)
	meta := `{"algorithm":"XChaCha20-Poly1305","version":3}`

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:              "Encrypted Note",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
	}, token)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing id")
}

func TestUpdateEncryptedNote_WithEncryptedTitle_Success(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "encuser-update-title", "enc-update-title@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, wrappedDEK, meta := makeEncryptedPayload(t)

	// Create encrypted note first
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:              "Encrypted Note",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)
	etag := rec.Header().Get("ETag")
	require.NotEmpty(t, etag)

	updatedEncryptedContent, updatedWrappedDEK, updatedMeta := makeEncryptedPayload(t)
	updatedEncryptedTitle := makeEncryptedTitlePayload(t, 3)

	rec = doJSONWithHeaders(t, r, http.MethodPut, "/api/notes/"+noteID, NoteRequest{
		Title:              "",
		EncryptedTitle:     &updatedEncryptedTitle,
		TitleEncrypted:     true,
		EncryptedContent:   updatedEncryptedContent,
		WrappedDEK:         updatedWrappedDEK,
		EncryptionMetadata: updatedMeta,
	}, token, map[string]string{"If-Match": etag})

	assert.Equal(t, http.StatusOK, rec.Code)
	var updated map[string]interface{}
	decodeResponse(t, rec, &updated)
	assert.Equal(t, true, updated["title_encrypted"])
	assert.Equal(t, updatedEncryptedTitle, updated["encrypted_title"])
}

func TestUpdateEncryptedNote_ClearsExistingLinksAndDueDates(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "encuser-clear-meta", "enc-clear-meta@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Target note for a resolved link.
	targetRec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Target Note",
		Content: "Plain target",
	}, token)
	require.Equal(t, http.StatusCreated, targetRec.Code)

	// Create plaintext source note with explicit client links + due dates metadata.
	sourceRec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Source Note",
		Content: "Plain source",
		Links: []ClientLink{
			{TargetTitle: "Target Note"},
		},
		DueDates: []ClientDueDate{
			{
				DueDate:     "2026-03-02",
				LineText:    "- [ ] plaintext task",
				LineIndex:   0,
				IsTaskItem:  true,
				IsCompleted: false,
			},
		},
	}, token)
	require.Equal(t, http.StatusCreated, sourceRec.Code)

	var source map[string]interface{}
	decodeResponse(t, sourceRec, &source)
	sourceID := source["id"].(string)
	etag := sourceRec.Header().Get("ETag")
	require.NotEmpty(t, etag)

	// Verify metadata exists before encrypted update.
	linksBefore := countRowsForNote(t, ts, "SELECT COUNT(*) FROM links WHERE source_id = ?", sourceID)
	dueDatesBefore := countRowsForNote(t, ts, "SELECT COUNT(*) FROM note_due_dates WHERE note_id = ?", sourceID)
	_, err := ts.db.Exec(`INSERT INTO note_keywords (note_id, keyword) VALUES (?, ?)`, sourceID, "legacy")
	require.NoError(t, err)
	keywordsBefore := countRowsForNote(t, ts, "SELECT COUNT(*) FROM note_keywords WHERE note_id = ?", sourceID)
	assert.Greater(t, linksBefore, 0)
	assert.Greater(t, dueDatesBefore, 0)
	assert.Greater(t, keywordsBefore, 0)

	encContent, wrappedDEK, meta := makeEncryptedPayload(t)
	rec := doJSONWithHeaders(t, r, http.MethodPut, "/api/notes/"+sourceID, NoteRequest{
		Title:              "Encrypted Source",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
		// These should be ignored for encrypted updates.
		Links: []ClientLink{
			{TargetTitle: "ShouldAlsoBeIgnored"},
		},
		DueDates: []ClientDueDate{
			{
				DueDate:     "2026-04-01",
				LineText:    "- [ ] should not persist",
				LineIndex:   1,
				IsTaskItem:  true,
				IsCompleted: false,
			},
		},
	}, token, map[string]string{"If-Match": etag})

	require.Equal(t, http.StatusOK, rec.Code)

	linksAfter := countRowsForNote(t, ts, "SELECT COUNT(*) FROM links WHERE source_id = ?", sourceID)
	unresolvedAfter := countRowsForNote(t, ts, "SELECT COUNT(*) FROM unresolved_links WHERE source_id = ?", sourceID)
	dueDatesAfter := countRowsForNote(t, ts, "SELECT COUNT(*) FROM note_due_dates WHERE note_id = ?", sourceID)
	keywordsAfter := countRowsForNote(t, ts, "SELECT COUNT(*) FROM note_keywords WHERE note_id = ?", sourceID)
	assert.Equal(t, 0, linksAfter)
	assert.Equal(t, 0, unresolvedAfter)
	assert.Equal(t, 0, dueDatesAfter)
	assert.Equal(t, 0, keywordsAfter)
}

func TestDecryptNote_Success(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "decuser", "dec@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, wrappedDEK, meta := makeEncryptedPayload(t)

	// Create encrypted note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:              "Decrypt Me",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)
	etag := rec.Header().Get("ETag")
	require.NotEmpty(t, etag)

	// Decrypt it
	rec = doJSONWithHeaders(t, r, http.MethodPost, "/api/notes/"+noteID+"/decrypt", DecryptNoteRequest{
		Title:   "Decrypted Title",
		Content: "Decrypted content in plaintext",
	}, token, map[string]string{"If-Match": etag})
	assert.Equal(t, http.StatusOK, rec.Code)
	var decrypted map[string]interface{}
	decodeResponse(t, rec, &decrypted)
	assert.Equal(t, "Decrypted Title", decrypted["title"])
	assert.Equal(t, false, decrypted["content_encrypted"])
}

func TestDecryptNote_MissingIfMatch(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "decuser", "dec@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, wrappedDEK, meta := makeEncryptedPayload(t)

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:              "Decrypt Me",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: meta,
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	noteID := created["id"].(string)

	// Try to decrypt without If-Match
	rec = doJSON(t, r, http.MethodPost, "/api/notes/"+noteID+"/decrypt", DecryptNoteRequest{
		Title:   "Decrypted",
		Content: "Content",
	}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDecryptNote_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "decuser", "dec@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSONWithHeaders(t, r, http.MethodPost, "/api/notes/nonexistent/decrypt", DecryptNoteRequest{
		Title:   "Test",
		Content: "Content",
	}, token, map[string]string{"If-Match": `"someetag"`})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestBatchReencryptDEKs_Success(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "batchuser", "batch@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	encContent, wrappedDEK, meta := makeEncryptedPayload(t)

	// Create two encrypted notes with unique titles
	var noteIDs []string
	titles := []string{"Batch Note Alpha", "Batch Note Beta"}
	for i := 0; i < 2; i++ {
		rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
			Title:              titles[i],
			EncryptedContent:   encContent,
			WrappedDEK:         wrappedDEK,
			EncryptionMetadata: meta,
		}, token)
		require.Equal(t, http.StatusCreated, rec.Code)
		var created map[string]interface{}
		decodeResponse(t, rec, &created)
		noteIDs = append(noteIDs, created["id"].(string))
	}

	// Generate new wrapped DEKs
	newDEK := make([]byte, 32)
	_, err := rand.Read(newDEK)
	require.NoError(t, err)
	newWrappedDEK := base64.StdEncoding.EncodeToString(newDEK)

	// Batch re-encrypt
	rec := doJSON(t, r, http.MethodPost, "/api/notes/batch-reencrypt-deks", BatchReencryptDEKsRequest{
		Updates: []struct {
			NoteID     string `json:"note_id"`
			WrappedDEK string `json:"wrapped_dek"`
		}{
			{NoteID: noteIDs[0], WrappedDEK: newWrappedDEK},
			{NoteID: noteIDs[1], WrappedDEK: newWrappedDEK},
		},
	}, token)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.Equal(t, float64(2), resp["updated_count"])
}

func TestBatchReencryptDEKs_EmptyUpdates(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "batchuser", "batch@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes/batch-reencrypt-deks", BatchReencryptDEKsRequest{
		Updates: nil,
	}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBatchReencryptDEKs_InvalidDEK(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "batchuser", "batch@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/notes/batch-reencrypt-deks", BatchReencryptDEKsRequest{
		Updates: []struct {
			NoteID     string `json:"note_id"`
			WrappedDEK string `json:"wrapped_dek"`
		}{
			{NoteID: "some-id", WrappedDEK: "not-valid-base64!!!"},
		},
	}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBatchReencryptDEKs_MissingNoteID(t *testing.T) {
	ts := newTestServer(t)
	r := encryptionRouter(ts)
	user := ts.createUser(t, "batchuser", "batch@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	newDEK := make([]byte, 32)
	rand.Read(newDEK)

	rec := doJSON(t, r, http.MethodPost, "/api/notes/batch-reencrypt-deks", BatchReencryptDEKsRequest{
		Updates: []struct {
			NoteID     string `json:"note_id"`
			WrappedDEK string `json:"wrapped_dek"`
		}{
			{NoteID: "", WrappedDEK: base64.StdEncoding.EncodeToString(newDEK)},
		},
	}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
