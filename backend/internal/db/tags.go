package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// Tag represents a tag that can be attached to notes.
type Tag struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	NameNorm string `json:"-"` // Normalized name for case-insensitive matching
	UserID   int    `json:"user_id"`
}

// normalizeTagName normalizes a tag name for case-insensitive matching.
func normalizeTagName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// GetAllTags returns all tags for a user.
func (d *DB) GetAllTags(userID int) ([]Tag, error) {
	query := `
		SELECT id, name, name_norm, user_id
		FROM tags
		WHERE user_id = ?
		ORDER BY name_norm
	`

	rows, err := d.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.NameNorm, &tag.UserID); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

// GetOrCreateTag retrieves or creates a tag by name for a user.
func (d *DB) GetOrCreateTag(userID int, name string) (*Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}

	nameNorm := normalizeTagName(name)

	// Try to get existing tag
	var tag Tag
	err := d.QueryRow(`
		SELECT id, name, name_norm, user_id
		FROM tags
		WHERE user_id = ? AND name_norm = ?
	`, userID, nameNorm).Scan(&tag.ID, &tag.Name, &tag.NameNorm, &tag.UserID)

	if err == nil {
		return &tag, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query tag: %w", err)
	}

	// Tag doesn't exist, create it
	result, err := d.Exec(`
		INSERT INTO tags (name, name_norm, user_id)
		VALUES (?, ?, ?)
	`, name, nameNorm, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert tag: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get tag id: %w", err)
	}
	tagID, err := validateLastInsertID(id, "tag id")
	if err != nil {
		return nil, err
	}

	tag = Tag{
		ID:       tagID,
		Name:     name,
		NameNorm: nameNorm,
		UserID:   userID,
	}

	return &tag, nil
}

// GetNoteTags returns all tags for a specific note.
func (d *DB) GetNoteTags(noteID string) ([]Tag, error) {
	query := `
		SELECT t.id, t.name, t.name_norm, t.user_id
		FROM tags t
		INNER JOIN note_tags nt ON nt.tag_id = t.id
		WHERE nt.note_id = ?
		ORDER BY t.name_norm
	`

	rows, err := d.Query(query, noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to query note tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.NameNorm, &tag.UserID); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating note tags: %w", err)
	}

	return tags, nil
}

// SetNoteTags sets the tags for a note, replacing any existing tags.
func (d *DB) SetNoteTags(noteID string, userID int, tagNames []string) error {
	// Start transaction
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Remove all existing tags for this note
	_, err = tx.Exec(`DELETE FROM note_tags WHERE note_id = ?`, noteID)
	if err != nil {
		return fmt.Errorf("failed to delete existing tags: %w", err)
	}

	// If no new tags, we're done
	if len(tagNames) == 0 {
		return tx.Commit()
	}

	// Get or create tags and link them to the note
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		nameNorm := normalizeTagName(name)

		// Get or create tag
		var tagID int
		err := tx.QueryRow(`
			SELECT id FROM tags WHERE user_id = ? AND name_norm = ?
		`, userID, nameNorm).Scan(&tagID)

		if err == sql.ErrNoRows {
			// Create new tag
			result, err := tx.Exec(`
				INSERT INTO tags (name, name_norm, user_id)
				VALUES (?, ?, ?)
			`, name, nameNorm, userID)
			if err != nil {
				return fmt.Errorf("failed to insert tag: %w", err)
			}

			id, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get tag id: %w", err)
			}
			tagID, err = validateLastInsertID(id, "tag id")
			if err != nil {
				return err
			}
		} else if err != nil {
			return fmt.Errorf("failed to query tag: %w", err)
		}

		// Link tag to note
		_, err = tx.Exec(`
			INSERT INTO note_tags (note_id, tag_id)
			VALUES (?, ?)
		`, noteID, tagID)
		if err != nil {
			return fmt.Errorf("failed to link tag to note: %w", err)
		}
	}

	return tx.Commit()
}

// DeleteTag deletes a tag and all its associations.
func (d *DB) DeleteTag(userID int, tagID int) error {
	// note_tags will be cascade deleted due to foreign key
	result, err := d.Exec(`
		DELETE FROM tags
		WHERE id = ? AND user_id = ?
	`, tagID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}
