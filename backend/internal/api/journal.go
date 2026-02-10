package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/xela-io/xelanote/internal/service"
)

// requireJournalFeature checks that the journal feature is enabled for userID.
// Returns true if the handler should return (feature disabled or error).
func (s *Server) requireJournalFeature(w http.ResponseWriter, userID int) bool {
	feature, err := s.noteService.GetDB().GetUserFeature(userID, "journal")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to check feature")
		return true
	}
	if !feature.Enabled {
		respondError(w, http.StatusForbidden, "journal feature not enabled")
		return true
	}
	return false
}

// getJournalLookup checks if a journal exists for a specific date.
// GET /journal?date=YYYY-MM-DD
// Returns: { exists: bool, date: string, note_id: string }
func (s *Server) getJournalLookup(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		respondError(w, http.StatusBadRequest, "date parameter required")
		return
	}

	// Validate date format
	if err := service.ValidateJournalDate(dateStr); err != nil {
		respondError(w, http.StatusBadRequest, "invalid date format, expected YYYY-MM-DD")
		return
	}

	if s.requireJournalFeature(w, userID) {
		return
	}

	// Lookup journal
	exists, noteID, err := s.noteService.GetDB().JournalExistsForDate(userID, dateStr)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to lookup journal")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"exists":  exists,
		"date":    dateStr,
		"note_id": noteID,
	})
}

// listJournalEntries returns all journal entries for the current user.
// GET /journal/entries
// Returns: { entries: [] }
func (s *Server) listJournalEntries(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if s.requireJournalFeature(w, userID) {
		return
	}

	entries, err := s.noteService.GetDB().ListJournalEntries(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list journal entries")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
	})
}

// getJournalCalendar returns all dates with journal entries for a month.
// GET /journal/calendar?year=YYYY&month=MM
// Returns: { year: int, month: int, dates: []string }
func (s *Server) getJournalCalendar(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if s.requireJournalFeature(w, userID) {
		return
	}

	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))

	now := time.Now()
	if year == 0 {
		year = now.Year()
	}
	if month == 0 || month < 1 || month > 12 {
		month = int(now.Month())
	}

	dates, err := s.noteService.GetDB().ListJournalDates(userID, year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get calendar data")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"year":  year,
		"month": month,
		"dates": dates,
	})
}

// getJournalCalendarYear returns all dates with journal entries for a year.
// GET /journal/calendar/year?year=YYYY
// Returns: { year: int, dates: []string }
func (s *Server) getJournalCalendarYear(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if s.requireJournalFeature(w, userID) {
		return
	}

	year, _ := strconv.Atoi(r.URL.Query().Get("year"))

	now := time.Now()
	if year == 0 {
		year = now.Year()
	}

	// Validate year range
	if year < 2000 || year > now.Year()+1 {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("year must be between 2000 and %d", now.Year()+1))
		return
	}

	dates, err := s.noteService.GetDB().ListJournalDatesForYear(userID, year)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get year calendar data")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"year":  year,
		"dates": dates,
	})
}
