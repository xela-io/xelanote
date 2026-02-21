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
