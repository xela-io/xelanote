//go:build fts5

package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func journalRouter(ts *testServer) chi.Router {
	r := ts.testRouter()
	r.Route("/api/notes", func(r chi.Router) {
		r.Post("/", ts.createNote)
	})
	r.Route("/api/journal", func(r chi.Router) {
		r.Get("/", ts.getJournalLookup)
		r.Get("/calendar", ts.getJournalCalendar)
		r.Get("/calendar/year", ts.getJournalCalendarYear)
		r.Get("/entries", ts.listJournalEntries)
	})
	return r
}

func TestJournalLookup_NoEntry(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal?date=2026-01-15", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.Equal(t, false, resp["exists"])
	assert.Equal(t, "2026-01-15", resp["date"])
}

func TestJournalLookup_WithEntry(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	today := time.Now().Format("2006-01-02")

	// Create a journal note via API
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:       "Journal Entry",
		Content:     "Today's notes",
		NoteType:    "journal",
		JournalDate: &today,
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Look it up
	rec = doJSON(t, r, http.MethodGet, "/api/journal?date="+today, nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.Equal(t, true, resp["exists"])
	assert.NotEmpty(t, resp["note_id"])
}

func TestJournalLookup_MissingDate(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal", nil, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestJournalLookup_InvalidDate(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal?date=not-a-date", nil, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestJournalLookup_FeatureDisabled(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	// Explicitly disable journal (it's enabled by default)
	err := ts.Server.noteService.SetUserFeature(user.User.ID, "journal", false, nil)
	require.NoError(t, err)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal?date=2026-01-15", nil, token)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestJournalEntries_Empty(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal/entries", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.NotNil(t, resp["entries"])
}

func TestJournalCalendar_Defaults(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal/calendar", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	now := time.Now()
	assert.Equal(t, float64(now.Year()), resp["year"])
	assert.Equal(t, float64(now.Month()), resp["month"])
}

func TestJournalCalendar_SpecificMonth(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal/calendar?year=2025&month=6", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.Equal(t, float64(2025), resp["year"])
	assert.Equal(t, float64(6), resp["month"])
}

func TestJournalCalendar_InvalidMonth(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal/calendar?month=13", nil, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestJournalCalendar_InvalidYear(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal/calendar?year=1990", nil, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestJournalCalendarYear_Success(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal/calendar/year?year=2025", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	decodeResponse(t, rec, &resp)
	assert.Equal(t, float64(2025), resp["year"])
}

func TestJournalCalendarYear_InvalidYear(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/journal/calendar/year?year=1990", nil, token)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestJournalCreate_DuplicateDate(t *testing.T) {
	ts := newTestServer(t)
	r := journalRouter(ts)
	user := ts.createUser(t, "jrnluser", "jrnl@example.com", "password123")
	ts.enableJournal(t, user.User.ID)
	token := ts.getAuthToken(t, user.User)

	date := "2026-03-15"

	// First journal note
	rec := doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:       "Journal 1",
		Content:     "First entry",
		NoteType:    "journal",
		JournalDate: &date,
	}, token)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Second journal note for same date should fail
	rec = doJSON(t, r, http.MethodPost, "/api/notes", NoteRequest{
		Title:       "Journal 2",
		Content:     "Duplicate",
		NoteType:    "journal",
		JournalDate: &date,
	}, token)
	assert.NotEqual(t, http.StatusCreated, rec.Code)
}
