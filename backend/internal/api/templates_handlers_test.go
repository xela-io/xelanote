//go:build fts5

package api

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func templatesRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/templates", func(r chi.Router) {
		r.Get("/", ts.getAllTemplates)
		r.Get("/{id}", ts.getTemplate)
		r.Post("/", ts.createTemplate)
		r.Put("/{id}", ts.updateTemplate)
		r.Delete("/{id}", ts.deleteTemplate)
	})
	return r
}

func snippetsRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/snippets", func(r chi.Router) {
		r.Get("/", ts.getAllSnippets)
		r.Get("/{id}", ts.getSnippet)
		r.Post("/", ts.createSnippet)
		r.Put("/{id}", ts.updateSnippet)
		r.Delete("/{id}", ts.deleteSnippet)
	})
	return r
}

// --- Templates ---

func TestCreateTemplate_Success(t *testing.T) {
	ts := newTestServer(t)
	r := templatesRouter(ts)
	user := ts.createUser(t, "tmpluser", "tmpl@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/templates", CreateTemplateRequest{
		Name:    "Meeting Notes",
		Title:   "Meeting: {{date}}",
		Content: "## Attendees\n\n## Agenda\n\n## Notes",
	}, token)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var tmpl map[string]interface{}
	decodeResponse(t, rec, &tmpl)
	assert.Equal(t, "Meeting Notes", tmpl["name"])
	assert.Equal(t, "Meeting: {{date}}", tmpl["title"])
}

func TestCreateTemplate_ValidationErrors(t *testing.T) {
	ts := newTestServer(t)
	r := templatesRouter(ts)
	user := ts.createUser(t, "tmplval", "tmplval@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	tests := []struct {
		name string
		req  CreateTemplateRequest
	}{
		{"empty name", CreateTemplateRequest{Name: "", Title: "T", Content: "C"}},
		{"empty title", CreateTemplateRequest{Name: "N", Title: "", Content: "C"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doJSON(t, r, http.MethodPost, "/api/templates", tt.req, token)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestCreateTemplate_Unauthorized(t *testing.T) {
	ts := newTestServer(t)
	r := templatesRouter(ts)

	rec := doJSON(t, r, http.MethodPost, "/api/templates", CreateTemplateRequest{
		Name: "X", Title: "Y", Content: "Z",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetAllTemplates_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := templatesRouter(ts)
	user := ts.createUser(t, "listtmpl", "listtmpl@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/templates", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	_, hasTemplates := resp["templates"]
	assert.True(t, hasTemplates, "response should have 'templates' key")
}

func TestGetTemplate_Success(t *testing.T) {
	ts := newTestServer(t)
	r := templatesRouter(ts)
	user := ts.createUser(t, "gettmpl", "gettmpl@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create
	rec := doJSON(t, r, http.MethodPost, "/api/templates", CreateTemplateRequest{
		Name: "My Template", Title: "Title", Content: "Body",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	id := itoa(int(created["id"].(float64)))

	// Get
	rec = doJSON(t, r, http.MethodGet, "/api/templates/"+id, nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var tmpl map[string]interface{}
	decodeResponse(t, rec, &tmpl)
	assert.Equal(t, "My Template", tmpl["name"])
}

func TestGetTemplate_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := templatesRouter(ts)
	user := ts.createUser(t, "tmplnf", "tmplnf@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/templates/99999", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateTemplate_Success(t *testing.T) {
	ts := newTestServer(t)
	r := templatesRouter(ts)
	user := ts.createUser(t, "updtmpl", "updtmpl@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create
	rec := doJSON(t, r, http.MethodPost, "/api/templates", CreateTemplateRequest{
		Name: "Old Name", Title: "Old Title", Content: "Old",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	id := itoa(int(created["id"].(float64)))

	// Update
	rec = doJSON(t, r, http.MethodPut, "/api/templates/"+id, UpdateTemplateRequest{
		Name: "New Name", Title: "New Title", Content: "New",
	}, token)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify
	rec = doJSON(t, r, http.MethodGet, "/api/templates/"+id, nil, token)
	var tmpl map[string]interface{}
	decodeResponse(t, rec, &tmpl)
	assert.Equal(t, "New Name", tmpl["name"])
	assert.Equal(t, "New Title", tmpl["title"])
}

func TestDeleteTemplate_Success(t *testing.T) {
	ts := newTestServer(t)
	r := templatesRouter(ts)
	user := ts.createUser(t, "deltmpl", "deltmpl@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create
	rec := doJSON(t, r, http.MethodPost, "/api/templates", CreateTemplateRequest{
		Name: "To Delete", Title: "T", Content: "C",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	id := itoa(int(created["id"].(float64)))

	// Delete
	rec = doJSON(t, r, http.MethodDelete, "/api/templates/"+id, nil, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify gone
	rec = doJSON(t, r, http.MethodGet, "/api/templates/"+id, nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTemplates_CrossUserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := templatesRouter(ts)
	user1 := ts.createUser(t, "tmpliso1", "tmpliso1@example.com", "password123")
	user2 := ts.createUser(t, "tmpliso2", "tmpliso2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 creates template
	rec := doJSON(t, r, http.MethodPost, "/api/templates", CreateTemplateRequest{
		Name: "Private Template", Title: "T", Content: "C",
	}, token1)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	id := itoa(int(created["id"].(float64)))

	// User2 cannot access it
	rec = doJSON(t, r, http.MethodGet, "/api/templates/"+id, nil, token2)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Snippets ---

func TestCreateSnippet_Success(t *testing.T) {
	ts := newTestServer(t)
	r := snippetsRouter(ts)
	user := ts.createUser(t, "snipuser", "snip@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/snippets", CreateSnippetRequest{
		Name:     "Code Block",
		Content:  "```go\nfmt.Println(\"hello\")\n```",
		Shortcut: "cb",
	}, token)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var snip map[string]interface{}
	decodeResponse(t, rec, &snip)
	assert.Equal(t, "Code Block", snip["name"])
	assert.Equal(t, "cb", snip["shortcut"])
}

func TestCreateSnippet_ValidationErrors(t *testing.T) {
	ts := newTestServer(t)
	r := snippetsRouter(ts)
	user := ts.createUser(t, "snipval", "snipval@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodPost, "/api/snippets", CreateSnippetRequest{
		Name:    "",
		Content: "content",
	}, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetAllSnippets_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := snippetsRouter(ts)
	user := ts.createUser(t, "listsnip", "listsnip@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/snippets", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	_, hasSnippets := resp["snippets"]
	assert.True(t, hasSnippets, "response should have 'snippets' key")
}

func TestGetSnippet_Success(t *testing.T) {
	ts := newTestServer(t)
	r := snippetsRouter(ts)
	user := ts.createUser(t, "getsnip", "getsnip@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create
	rec := doJSON(t, r, http.MethodPost, "/api/snippets", CreateSnippetRequest{
		Name: "My Snippet", Content: "text",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	id := itoa(int(created["id"].(float64)))

	// Get
	rec = doJSON(t, r, http.MethodGet, "/api/snippets/"+id, nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var snip map[string]interface{}
	decodeResponse(t, rec, &snip)
	assert.Equal(t, "My Snippet", snip["name"])
}

func TestGetSnippet_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := snippetsRouter(ts)
	user := ts.createUser(t, "snipnf", "snipnf@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/snippets/99999", nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateSnippet_Success(t *testing.T) {
	ts := newTestServer(t)
	r := snippetsRouter(ts)
	user := ts.createUser(t, "updsnip", "updsnip@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create
	rec := doJSON(t, r, http.MethodPost, "/api/snippets", CreateSnippetRequest{
		Name: "Old", Content: "old",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	id := itoa(int(created["id"].(float64)))

	// Update
	rec = doJSON(t, r, http.MethodPut, "/api/snippets/"+id, UpdateSnippetRequest{
		Name: "Updated", Content: "new content", Shortcut: "u",
	}, token)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify
	rec = doJSON(t, r, http.MethodGet, "/api/snippets/"+id, nil, token)
	var snip map[string]interface{}
	decodeResponse(t, rec, &snip)
	assert.Equal(t, "Updated", snip["name"])
}

func TestDeleteSnippet_Success(t *testing.T) {
	ts := newTestServer(t)
	r := snippetsRouter(ts)
	user := ts.createUser(t, "delsnip", "delsnip@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	// Create
	rec := doJSON(t, r, http.MethodPost, "/api/snippets", CreateSnippetRequest{
		Name: "To Delete", Content: "c",
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	id := itoa(int(created["id"].(float64)))

	// Delete
	rec = doJSON(t, r, http.MethodDelete, "/api/snippets/"+id, nil, token)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify gone
	rec = doJSON(t, r, http.MethodGet, "/api/snippets/"+id, nil, token)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSnippets_CrossUserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := snippetsRouter(ts)
	user1 := ts.createUser(t, "snipiso1", "snipiso1@example.com", "password123")
	user2 := ts.createUser(t, "snipiso2", "snipiso2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 creates snippet
	rec := doJSON(t, r, http.MethodPost, "/api/snippets", CreateSnippetRequest{
		Name: "Private", Content: "secret",
	}, token1)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]interface{}
	decodeResponse(t, rec, &created)
	id := itoa(int(created["id"].(float64)))

	// User2 cannot access it
	rec = doJSON(t, r, http.MethodGet, "/api/snippets/"+id, nil, token2)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
