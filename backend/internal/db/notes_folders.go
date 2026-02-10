package db

import "fmt"

// GetFolders returns all unique folder paths.
func (db *DB) GetFolders(userID int) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT folder_path
		FROM notes
		WHERE user_id = ? AND is_deleted = 0
		ORDER BY folder_path ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}
	defer rows.Close()

	var folders []string
	for rows.Next() {
		var folder string
		if err := rows.Scan(&folder); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		folders = append(folders, folder)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating folders: %w", err)
	}

	return folders, nil
}

// GetFoldersWithCounts returns all unique folder paths with note counts.
func (db *DB) GetFoldersWithCounts(userID int) ([]FolderInfo, error) {
	rows, err := db.Query(`
		SELECT folder_path, COUNT(*) as note_count
		FROM notes
		WHERE user_id = ? AND is_deleted = 0
		GROUP BY folder_path
		ORDER BY folder_path ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders with counts: %w", err)
	}
	defer rows.Close()

	var folders []FolderInfo
	for rows.Next() {
		var f FolderInfo
		if err := rows.Scan(&f.Path, &f.NoteCount); err != nil {
			return nil, fmt.Errorf("failed to scan folder info: %w", err)
		}
		folders = append(folders, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating folder info: %w", err)
	}

	return folders, nil
}
