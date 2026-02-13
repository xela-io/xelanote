package service

import "github.com/xela-io/xelanote/internal/db"

// JournalExistsForDate checks if a journal entry exists for a given date.
func (s *NoteService) JournalExistsForDate(userID int, date string) (bool, string, error) {
	return s.db.JournalExistsForDate(userID, date)
}

// ListJournalEntries returns all journal entries for a user.
func (s *NoteService) ListJournalEntries(userID int) ([]db.Note, error) {
	return s.db.ListJournalEntries(userID)
}

// ListJournalDates returns all dates with journal entries for a given month.
func (s *NoteService) ListJournalDates(userID int, year, month int) ([]string, error) {
	return s.db.ListJournalDates(userID, year, month)
}

// ListJournalDatesForYear returns all dates with journal entries for a given year.
func (s *NoteService) ListJournalDatesForYear(userID int, year int) ([]string, error) {
	return s.db.ListJournalDatesForYear(userID, year)
}
