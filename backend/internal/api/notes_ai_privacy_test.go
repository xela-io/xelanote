//go:build fts5

package api

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func aiPrivacyRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/notes", func(r chi.Router) {
		r.Post("/", ts.createNote)
		r.Post("/{id}/suggest-tags", ts.suggestTags)
		r.Post("/{id}/suggest-links", ts.suggestLinks)
		r.Post("/{id}/format-markdown", ts.formatMarkdown)
		r.Post("/{id}/ai-transform", ts.aiTransform)
	})
	return r
}

func createEncryptedNoteForAIPrivacyTest(t *testing.T, r chi.Router, token string) string {
	t.Helper()

	encContent, wrappedDEK, metadata := makeEncryptedPayload(t)
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:              "Encrypted AI Note",
		EncryptedContent:   encContent,
		WrappedDEK:         wrappedDEK,
		EncryptionMetadata: metadata,
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created map[string]interface{}
	decodeResponse(t, rec, &created)

	noteID, ok := created["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, noteID)
	return noteID
}

func createPlaintextNoteForAIPrivacyTest(t *testing.T, r chi.Router, token string) string {
	t.Helper()

	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:   "Plaintext AI Note",
		Content: "This is plaintext content for handler validation.",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created map[string]interface{}
	decodeResponse(t, rec, &created)

	noteID, ok := created["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, noteID)
	return noteID
}

func TestAIEndpoints_BlockEncryptedNoteServerProcessing(t *testing.T) {
	ts := newTestServer(t)
	r := aiPrivacyRouter(ts)
	user := ts.createUser(t, "ai-privacy", "ai-privacy@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	noteID := createEncryptedNoteForAIPrivacyTest(t, r, token)

	testCases := []struct {
		name string
		path string
		body interface{}
	}{
		{
			name: "suggest tags",
			path: "/api/notes/" + noteID + "/suggest-tags",
			body: SuggestTagsRequest{
				PlaintextContent: "secret plaintext that must never reach AI provider",
			},
		},
		{
			name: "suggest links",
			path: "/api/notes/" + noteID + "/suggest-links",
			body: SuggestLinksRequest{
				PlaintextContent: "secret plaintext that must never reach AI provider",
				NoteTitles:       []string{"Target Note"},
				ExistingLinks:    []string{},
			},
		},
		{
			name: "format markdown",
			path: "/api/notes/" + noteID + "/format-markdown",
			body: FormatMarkdownRequest{
				PlaintextContent: "secret plaintext that must never reach AI provider",
			},
		},
		{
			name: "ai transform",
			path: "/api/notes/" + noteID + "/ai-transform",
			body: AITransformRequest{
				Action:           "format",
				Content:          "secret plaintext that must never reach AI provider",
				PlaintextContent: "secret plaintext that must never reach AI provider",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, r, http.MethodPost, tc.path, tc.body, token)
			require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

			var resp map[string]string
			decodeResponse(t, rec, &resp)
			assert.Equal(t, errEncryptedNoteAIProcessingDisabled, resp["error"])
		})
	}
}

func TestAIEndpoints_RejectLegacyPlaintextContentField(t *testing.T) {
	ts := newTestServer(t)
	r := aiPrivacyRouter(ts)
	user := ts.createUser(t, "ai-plaintext-field", "ai-plaintext-field@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	noteID := createPlaintextNoteForAIPrivacyTest(t, r, token)

	testCases := []struct {
		name string
		path string
		body interface{}
	}{
		{
			name: "suggest tags",
			path: "/api/notes/" + noteID + "/suggest-tags",
			body: SuggestTagsRequest{
				PlaintextContent: "legacy plaintext payload",
			},
		},
		{
			name: "suggest links",
			path: "/api/notes/" + noteID + "/suggest-links",
			body: SuggestLinksRequest{
				PlaintextContent: "legacy plaintext payload",
				NoteTitles:       []string{"Target Note"},
				ExistingLinks:    []string{},
			},
		},
		{
			name: "format markdown",
			path: "/api/notes/" + noteID + "/format-markdown",
			body: FormatMarkdownRequest{
				PlaintextContent: "legacy plaintext payload",
				SelectionOnly:    "selection",
			},
		},
		{
			name: "ai transform",
			path: "/api/notes/" + noteID + "/ai-transform",
			body: AITransformRequest{
				Action:           "format",
				Content:          "selection",
				PlaintextContent: "legacy plaintext payload",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, r, http.MethodPost, tc.path, tc.body, token)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

			var resp map[string]string
			decodeResponse(t, rec, &resp)
			assert.Equal(t, errAIPlaintextContentDeprecated, resp["error"])
		})
	}
}
