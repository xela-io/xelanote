package db

import (
	"database/sql"
	"fmt"
)

// GetSharedRecipesForUser returns all recipe notes shared with the given user.
// 3-way UNION: note_shares + folder_shares + collection_shares.
// Dedup via NOT EXISTS: highest priority source wins (R2, R3).
func (db *DB) GetSharedRecipesForUser(userID int) ([]SharedNote, error) {
	rows, err := db.Query(`
		-- 1. Direct note shares (highest priority)
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, ns.role, ns.id
		FROM note_shares ns
		JOIN notes n ON n.id = ns.note_id
		JOIN users ou ON ou.id = ns.owner_user_id
		WHERE ns.shared_with_user_id = ? AND n.is_deleted = 0 AND n.note_type = 'recipe'

		UNION ALL

		-- 2. Folder shares (dedup against note_shares)
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, fs.role, fs.id
		FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		JOIN notes n ON n.folder_path = f.path AND n.is_deleted = 0 AND n.content_encrypted = 0
		JOIN users ou ON ou.id = fs.owner_user_id
		WHERE fs.shared_with_user_id = ? AND n.note_type = 'recipe'
		  AND NOT EXISTS (
		    SELECT 1 FROM note_shares ns2
		    WHERE ns2.note_id = n.id AND ns2.shared_with_user_id = ?
		  )

		UNION ALL

		-- 3. Collection shares (dedup against note_shares + folder_shares)
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, rcs.role, rcs.id
		FROM recipe_collection_shares rcs
		JOIN recipe_collection_items rci ON rci.collection_id = rcs.collection_id
		JOIN notes n ON n.id = rci.note_id AND n.is_deleted = 0 AND n.note_type = 'recipe'
		JOIN users ou ON ou.id = rcs.owner_user_id
		WHERE rcs.shared_with_user_id = ?
		  AND NOT EXISTS (
		    SELECT 1 FROM note_shares ns3
		    WHERE ns3.note_id = n.id AND ns3.shared_with_user_id = ?
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM folder_shares fs2
		    JOIN folders f2 ON f2.id = fs2.folder_id
		    WHERE fs2.shared_with_user_id = ? AND f2.path = n.folder_path
		  )

		ORDER BY 7 DESC
	`, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared recipes: %w", err)
	}
	defer rows.Close()

	var notes []SharedNote
	for rows.Next() {
		var sn SharedNote
		var createdAt, updatedAt string
		var content sql.NullString
		if err := rows.Scan(
			&sn.ID, &sn.Title, &content, &sn.FolderPath, &sn.Version,
			&createdAt, &updatedAt,
			&sn.NoteType,
			&sn.SharedBy, &sn.ShareRole, &sn.ShareID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shared recipe: %w", err)
		}
		sn.CreatedAt = parseTime(createdAt)
		sn.UpdatedAt = parseTime(updatedAt)
		if content.Valid {
			sn.Content = content.String
		}
		notes = append(notes, sn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared recipes: %w", err)
	}

	return notes, nil
}
