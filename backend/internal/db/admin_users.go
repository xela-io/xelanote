package db

import "database/sql"

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
