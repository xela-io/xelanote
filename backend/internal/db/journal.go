package db

import (
	"database/sql"
	"time"
)

// Note-type constants
const (
	NoteTypeNote    = "note"
	NoteTypeJournal = "journal"
	NoteTypeRecipe  = "recipe"
)

// GetJournalByDate retrieves a journal entry for a specific date.
// Returns ErrNotFound if no journal exists for this date.
func (db *DB) GetJournalByDate(userID int, date string) (*Note, error) {
	var note Note
	var createdAt, updatedAt string
	var journalDate sql.NullString
	var noteType sql.NullString

	err := db.QueryRow(`
		SELECT id, user_id, title, folder_path, version, note_type, journal_date,
		       created_at, updated_at, content_encrypted
		FROM notes
		WHERE user_id = ? AND journal_date = ? AND note_type = 'journal' AND is_deleted = 0
	`, userID, date).Scan(
		&note.ID, &note.UserID, &note.Title, &note.FolderPath, &note.Version,
		&noteType, &journalDate, &createdAt, &updatedAt, &note.ContentEncrypted,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if journalDate.Valid {
		note.JournalDate = &journalDate.String
	}
	if noteType.Valid {
		note.NoteType = noteType.String
	}

	return &note, nil
}

// ListJournalDates returns all dates with journal entries for a given month.
func (db *DB) ListJournalDates(userID int, year, month int) ([]string, error) {
	// Calculate correct month boundaries
	firstOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

	startDate := firstOfMonth.Format("2006-01-02")
	endDate := lastOfMonth.Format("2006-01-02")

	rows, err := db.Query(`
		SELECT journal_date
		FROM notes
		WHERE user_id = ? AND note_type = 'journal' AND is_deleted = 0
		  AND journal_date BETWEEN ? AND ?
		ORDER BY journal_date
	`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dates := make([]string, 0)
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}
	return dates, rows.Err()
}

// ListJournalEntries returns all journal entries for a user, ordered by journal_date DESC.
// Only returns fields needed for the list view (no content, encrypted_content, wrapped_dek).
func (db *DB) ListJournalEntries(userID int) ([]Note, error) {
	rows, err := db.Query(`
		SELECT id, title, folder_path, journal_date, note_type, created_at, updated_at, content_encrypted
		FROM notes
		WHERE user_id = ? AND note_type = 'journal' AND is_deleted = 0 AND journal_date IS NOT NULL
		ORDER BY journal_date DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Note, 0)
	for rows.Next() {
		var note Note
		var journalDate sql.NullString
		var noteType sql.NullString
		var createdAt, updatedAt string

		if err := rows.Scan(
			&note.ID, &note.Title, &note.FolderPath,
			&journalDate, &noteType,
			&createdAt, &updatedAt, &note.ContentEncrypted,
		); err != nil {
			return nil, err
		}

		note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if journalDate.Valid {
			note.JournalDate = &journalDate.String
		}
		if noteType.Valid {
			note.NoteType = noteType.String
		}

		entries = append(entries, note)
	}
	return entries, rows.Err()
}

// ListJournalDatesForYear returns all dates with journal entries for a given year.
func (db *DB) ListJournalDatesForYear(userID int, year int) ([]string, error) {
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	endDate := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

	rows, err := db.Query(`
		SELECT DISTINCT journal_date
		FROM notes
		WHERE user_id = ? AND note_type = 'journal' AND is_deleted = 0
		  AND journal_date BETWEEN ? AND ?
		ORDER BY journal_date ASC
	`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dates := make([]string, 0)
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}
	return dates, rows.Err()
}

// JournalExistsForDate checks if a journal exists for the given date and returns its ID.
func (db *DB) JournalExistsForDate(userID int, date string) (exists bool, noteID string, err error) {
	err = db.QueryRow(`
		SELECT id FROM notes
		WHERE user_id = ? AND journal_date = ? AND note_type = 'journal' AND is_deleted = 0
	`, userID, date).Scan(&noteID)

	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, noteID, nil
}
