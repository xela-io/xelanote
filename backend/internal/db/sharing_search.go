package db

import "fmt"

// SearchUserByUsernameOrEmail searches for users by username or email prefix.
// Excludes the requesting user and limits results to 5.
// Only searches by username prefix to prevent email enumeration.
func (db *DB) SearchUserByUsernameOrEmail(query string, excludeUserID int) ([]UserSearchResult, error) {
	if len(query) < 3 {
		return nil, fmt.Errorf("query must be at least 3 characters")
	}

	likeQuery := query + "%"
	rows, err := db.Query(`
		SELECT id, username FROM users
		WHERE id != ? AND username LIKE ?
		ORDER BY username ASC
		LIMIT 5
	`, excludeUserID, likeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var results []UserSearchResult
	for rows.Next() {
		var u UserSearchResult
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		results = append(results, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return results, nil
}
