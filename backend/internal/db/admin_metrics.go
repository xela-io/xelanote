package db

import "time"

// CountUsers returns the total number of users
func (db *DB) CountUsers() (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// GetNoteCountForUser returns the number of notes for a user
func (db *DB) GetNoteCountForUser(userID int) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM notes WHERE user_id = ? AND deleted_at IS NULL", userID).Scan(&count)
	return count, err
}

// GetActiveUsersLast30Days returns count of users with activity in last 30 days
func (db *DB) GetActiveUsersLast30Days() (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM activity_logs
		WHERE created_at >= datetime('now', '-30 days')
	`).Scan(&count)
	return count, err
}

// GetNotesCreatedToday returns count of notes created today
func (db *DB) GetNotesCreatedToday() (int, error) {
	today := time.Now().Format("2006-01-02")
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM notes
		WHERE date(created_at) = ? AND deleted_at IS NULL
	`, today).Scan(&count)
	return count, err
}
