//go:build fts5

package api

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xela-io/xelanote/internal/parser"
)

// dueDatesExportRouter registers notes CRUD + due-dates + export + import + task-events.
func dueDatesExportRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	// Notes CRUD (needed for task-events ownership check)
	r.Route("/api/notes", func(r chi.Router) {
		r.Post("/", ts.createNote)
		r.Get("/{id}", ts.getNote)
		r.Post("/{id}/task-events", ts.recordTaskEvent)
	})
	r.Get("/api/due-dates", ts.getDueDates)
	r.Get("/api/export/markdown", ts.exportMarkdown)
	r.Post("/api/import/markdown", ts.importMarkdown)
	return r
}

func TestDueDates_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := dueDatesExportRouter(ts)
	user1 := ts.createUser(t, "due1", "due1@example.com", "password123")
	user2 := ts.createUser(t, "due2", "due2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 creates a note and sets due dates via service layer
	note := ts.createNoteDirectly(t, user1.ID, "Task Note", "Buy milk @due(2026-03-15)", "/")
	err := ts.noteService.SetNoteDueDates(note.ID, user1.ID, []parser.DueDate{
		{
			Date:       "2026-03-15",
			LineText:   "Buy milk",
			LineIndex:  0,
			IsTaskItem: false,
		},
	})
	require.NoError(t, err)

	// User1 sees the due date
	rec := doJSON(t, r, http.MethodGet, "/api/due-dates", nil, token1)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp1 map[string]interface{}
	decodeResponse(t, rec, &resp1)
	dueDates1, ok := resp1["due_dates"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 1, len(dueDates1), "User1 should see 1 due date")

	// User2 sees empty list
	rec = doJSON(t, r, http.MethodGet, "/api/due-dates", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp2 map[string]interface{}
	decodeResponse(t, rec, &resp2)
	if dueDates2, ok := resp2["due_dates"].([]interface{}); ok {
		assert.Equal(t, 0, len(dueDates2), "User2 must not see User1's due dates")
	}
	// nil is also acceptable (no due dates)
}

func TestExport_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := dueDatesExportRouter(ts)
	user1 := ts.createUser(t, "exp1", "exp1@example.com", "password123")
	user2 := ts.createUser(t, "exp2", "exp2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 creates 2 notes
	ts.createNoteDirectly(t, user1.ID, "User1 Note A", "Content A", "/")
	ts.createNoteDirectly(t, user1.ID, "User1 Note B", "Content B", "/")

	// User2 creates 1 note
	ts.createNoteDirectly(t, user2.ID, "User2 Note", "Content C", "/")

	// User1 export should contain exactly 2 entries
	rec := doJSON(t, r, http.MethodGet, "/api/export/markdown", nil, token1)
	require.Equal(t, http.StatusOK, rec.Code)
	count1 := countZipEntries(t, rec.Body.Bytes())
	assert.Equal(t, 2, count1, "User1 export must contain exactly 2 notes")

	// User2 export should contain exactly 1 entry
	rec = doJSON(t, r, http.MethodGet, "/api/export/markdown", nil, token2)
	require.Equal(t, http.StatusOK, rec.Code)
	count2 := countZipEntries(t, rec.Body.Bytes())
	assert.Equal(t, 1, count2, "User2 export must contain exactly 1 note")
}

func TestTaskEvents_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	r := dueDatesExportRouter(ts)
	user1 := ts.createUser(t, "task1", "task1@example.com", "password123")
	user2 := ts.createUser(t, "task2", "task2@example.com", "password123")
	token1 := ts.getAuthToken(t, user1.User)
	token2 := ts.getAuthToken(t, user2.User)

	// User1 creates a note
	note := ts.createNoteDirectly(t, user1.ID, "Checklist", "- [ ] Item 1", "/")

	// User1 can record a task event
	taskText := "Item 1"
	rec := doJSON(t, r, http.MethodPost, "/api/notes/"+note.ID+"/task-events", map[string]interface{}{
		"task_text":  taskText,
		"task_index": 0,
		"event_type": "completed",
	}, token1)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// User2 tries to record a task event on User1's note — 404
	rec = doJSON(t, r, http.MethodPost, "/api/notes/"+note.ID+"/task-events", map[string]interface{}{
		"task_text":  taskText,
		"task_index": 0,
		"event_type": "completed",
	}, token2)
	assert.Equal(t, http.StatusNotFound, rec.Code, "User2 must not access User1's note for task events")
}

// countZipEntries reads ZIP data and returns the number of files inside.
func countZipEntries(t *testing.T, data []byte) int {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		// If the body is empty or not a valid ZIP, try reading it for a better error message
		t.Fatalf("failed to open ZIP: %v (body length: %d, first bytes: %q)", err, len(data), truncate(data, 100))
	}
	count := 0
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("failed to open ZIP entry %q: %v", f.Name, err)
		}
		_, _ = io.Copy(io.Discard, rc)
		_ = rc.Close()
		count++
	}
	return count
}

func truncate(b []byte, max int) []byte {
	if len(b) <= max {
		return b
	}
	return b[:max]
}
