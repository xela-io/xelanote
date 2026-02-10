package db

import (
	"database/sql"
	"time"
)

// SystemStats holds basic system statistics
type SystemStats struct {
	TotalUsers    int     `json:"total_users"`
	TotalNotes    int     `json:"total_notes"`
	TotalFolders  int     `json:"total_folders"`
	TotalTags     int     `json:"total_tags"`
	StorageUsedMB float64 `json:"storage_used_mb"`
}

// DetailedStats holds detailed statistics with time series data
type DetailedStats struct {
	Stats        SystemStats  `json:"stats"`
	UserGrowth   []DailyCount `json:"user_growth"`
	NoteGrowth   []DailyCount `json:"note_growth"`
	StorageTrend []DailyFloat `json:"storage_trend"`
}

// DailyCount holds a date and count pair
type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// DailyFloat holds a date and float value pair
type DailyFloat struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// AdminUser represents a user with stats for admin panel
type AdminUser struct {
	ID                 int     `json:"id"`
	Username           string  `json:"username"`
	Email              string  `json:"email"`
	IsAdmin            bool    `json:"is_admin"`
	NoteCount          int     `json:"note_count"`
	StorageMB          float64 `json:"storage_mb"`
	CreatedAt          string  `json:"created_at"`
	TOTPEnabled        bool    `json:"totp_enabled"`
	TOTPVerifiedAt     string  `json:"totp_verified_at"`
	TOTPDisabledAt     string  `json:"totp_disabled_at"`
	TOTPSetupStartedAt string  `json:"totp_setup_started_at"`
}

// GetSystemStats returns basic system statistics
func (db *DB) GetSystemStats() (*SystemStats, error) {
	stats := &SystemStats{}

	// Count users
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}

	// Count notes (active only)
	err = db.QueryRow("SELECT COUNT(*) FROM notes WHERE deleted_at IS NULL").Scan(&stats.TotalNotes)
	if err != nil {
		return nil, err
	}

	// Count folders
	err = db.QueryRow("SELECT COUNT(*) FROM folders").Scan(&stats.TotalFolders)
	if err != nil {
		return nil, err
	}

	// Count tags
	err = db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&stats.TotalTags)
	if err != nil {
		return nil, err
	}

	// Storage is calculated at runtime from uploads directory

	return stats, nil
}

// GetDetailedStats returns detailed statistics with time series data
func (db *DB) GetDetailedStats(days int) (*DetailedStats, error) {
	stats, err := db.GetSystemStats()
	if err != nil {
		return nil, err
	}

	detailed := &DetailedStats{
		Stats:        *stats,
		UserGrowth:   []DailyCount{},
		NoteGrowth:   []DailyCount{},
		StorageTrend: []DailyFloat{},
	}

	// User growth over last N days
	rows, err := db.Query(`
		SELECT date(created_at) as day, COUNT(*) as count
		FROM users
		WHERE created_at >= date('now', '-' || ? || ' days')
		GROUP BY date(created_at)
		ORDER BY day
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dc DailyCount
		if err := rows.Scan(&dc.Date, &dc.Count); err != nil {
			return nil, err
		}
		detailed.UserGrowth = append(detailed.UserGrowth, dc)
	}

	// Note growth over last N days
	rows, err = db.Query(`
		SELECT date(created_at) as day, COUNT(*) as count
		FROM notes
		WHERE created_at >= date('now', '-' || ? || ' days')
		AND deleted_at IS NULL
		GROUP BY date(created_at)
		ORDER BY day
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dc DailyCount
		if err := rows.Scan(&dc.Date, &dc.Count); err != nil {
			return nil, err
		}
		detailed.NoteGrowth = append(detailed.NoteGrowth, dc)
	}

	return detailed, nil
}

// GetAllUsersWithStats returns all users with their note counts
func (db *DB) GetAllUsersWithStats() ([]AdminUser, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, u.email, u.is_admin, u.created_at,
			   COALESCE(n.note_count, 0) as note_count,
			   COALESCE(u.totp_enabled, 0) as totp_enabled,
			   COALESCE(u.totp_verified_at, '') as totp_verified_at,
			   COALESCE(u.totp_disabled_at, '') as totp_disabled_at,
			   COALESCE(u.totp_setup_started_at, '') as totp_setup_started_at
		FROM users u
		LEFT JOIN (
			SELECT user_id, COUNT(*) as note_count
			FROM notes
			WHERE deleted_at IS NULL
			GROUP BY user_id
		) n ON u.id = n.user_id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin, &u.CreatedAt, &u.NoteCount,
			&u.TOTPEnabled, &u.TOTPVerifiedAt, &u.TOTPDisabledAt, &u.TOTPSetupStartedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

// GetUserStats returns detailed stats for a single user
func (db *DB) GetUserStats(userID int) (*AdminUser, error) {
	var u AdminUser
	err := db.QueryRow(`
		SELECT u.id, u.username, u.email, u.is_admin, u.created_at,
			   COALESCE(n.note_count, 0) as note_count,
			   COALESCE(u.totp_enabled, 0) as totp_enabled,
			   COALESCE(u.totp_verified_at, '') as totp_verified_at,
			   COALESCE(u.totp_disabled_at, '') as totp_disabled_at,
			   COALESCE(u.totp_setup_started_at, '') as totp_setup_started_at
		FROM users u
		LEFT JOIN (
			SELECT user_id, COUNT(*) as note_count
			FROM notes
			WHERE deleted_at IS NULL AND user_id = ?
			GROUP BY user_id
		) n ON u.id = n.user_id
		WHERE u.id = ?
	`, userID, userID).Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin, &u.CreatedAt, &u.NoteCount,
		&u.TOTPEnabled, &u.TOTPVerifiedAt, &u.TOTPDisabledAt, &u.TOTPSetupStartedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// SetUserAdmin sets the admin status of a user
func (db *DB) SetUserAdmin(userID int, isAdmin bool) error {
	result, err := db.Exec(`
		UPDATE users SET is_admin = ?, updated_at = datetime('now')
		WHERE id = ?
	`, isAdmin, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteUserByAdmin deletes a user and all their data
func (db *DB) DeleteUserByAdmin(userID int) error {
	// Start a transaction for atomic deletion
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete user's refresh tokens
	if _, err := tx.Exec("DELETE FROM refresh_tokens WHERE user_id = ?", userID); err != nil {
		return err
	}

	// Delete user's note versions
	if _, err := tx.Exec(`
		DELETE FROM note_versions WHERE note_id IN (SELECT id FROM notes WHERE user_id = ?)
	`, userID); err != nil {
		return err
	}

	// Delete user's links
	if _, err := tx.Exec(`
		DELETE FROM links WHERE source_id IN (SELECT id FROM notes WHERE user_id = ?)
	`, userID); err != nil {
		return err
	}

	// Delete user's unresolved links
	if _, err := tx.Exec(`
		DELETE FROM unresolved_links WHERE source_id IN (SELECT id FROM notes WHERE user_id = ?)
	`, userID); err != nil {
		return err
	}

	// Delete note-tag associations
	if _, err := tx.Exec(`
		DELETE FROM note_tags WHERE note_id IN (SELECT id FROM notes WHERE user_id = ?)
	`, userID); err != nil {
		return err
	}

	// Delete user's tags
	if _, err := tx.Exec("DELETE FROM tags WHERE user_id = ?", userID); err != nil {
		return err
	}

	// Delete from FTS table
	if _, err := tx.Exec(`
		DELETE FROM notes_fts WHERE rowid IN (SELECT rowid FROM notes WHERE user_id = ?)
	`, userID); err != nil {
		return err
	}

	// Delete user's notes
	if _, err := tx.Exec("DELETE FROM notes WHERE user_id = ?", userID); err != nil {
		return err
	}

	// Delete user's folders
	if _, err := tx.Exec("DELETE FROM folders WHERE user_id = ?", userID); err != nil {
		return err
	}

	// Delete user's templates
	if _, err := tx.Exec("DELETE FROM templates WHERE user_id = ?", userID); err != nil {
		return err
	}

	// Delete user's snippets
	if _, err := tx.Exec("DELETE FROM snippets WHERE user_id = ?", userID); err != nil {
		return err
	}

	// Delete user's preferences
	if _, err := tx.Exec("DELETE FROM user_preferences WHERE user_id = ?", userID); err != nil {
		return err
	}

	// Finally, delete the user
	result, err := tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return tx.Commit()
}

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

// GetRecentUsers returns recently created users
func (db *DB) GetRecentUsers(limit int) ([]AdminUser, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, u.email, u.is_admin, u.created_at,
			   COALESCE(n.note_count, 0) as note_count,
			   COALESCE(u.totp_enabled, 0) as totp_enabled,
			   COALESCE(u.totp_verified_at, '') as totp_verified_at,
			   COALESCE(u.totp_disabled_at, '') as totp_disabled_at,
			   COALESCE(u.totp_setup_started_at, '') as totp_setup_started_at
		FROM users u
		LEFT JOIN (
			SELECT user_id, COUNT(*) as note_count
			FROM notes
			WHERE deleted_at IS NULL
			GROUP BY user_id
		) n ON u.id = n.user_id
		ORDER BY u.created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin, &u.CreatedAt, &u.NoteCount,
			&u.TOTPEnabled, &u.TOTPVerifiedAt, &u.TOTPDisabledAt, &u.TOTPSetupStartedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

// IsUserAdmin checks if a user is an admin
func (db *DB) IsUserAdmin(userID int) (bool, error) {
	var isAdmin bool
	err := db.QueryRow("SELECT is_admin FROM users WHERE id = ?", userID).Scan(&isAdmin)
	if err == sql.ErrNoRows {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}
	return isAdmin, nil
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
