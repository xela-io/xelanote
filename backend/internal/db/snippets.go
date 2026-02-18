package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/constraints"
)

// Snippet represents a reusable text snippet.
type Snippet struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	NameNorm    string    `json:"-"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Shortcut    string    `json:"shortcut"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// normalizeSnippetName normalizes a snippet name for case-insensitive matching.
func normalizeSnippetName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// GetAllSnippets returns all snippets for a user.
func (d *DB) GetAllSnippets(userID int) ([]Snippet, error) {
	query := `
		SELECT id, user_id, name, name_norm, description, content, shortcut, created_at, updated_at
		FROM snippets
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`

	rows, err := d.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query snippets: %w", err)
	}
	defer rows.Close()

	var snippets []Snippet
	for rows.Next() {
		var s Snippet
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.NameNorm, &s.Description, &s.Content, &s.Shortcut, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan snippet: %w", err)
		}
		snippets = append(snippets, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snippets: %w", err)
	}

	return snippets, nil
}

// GetSnippet returns a single snippet by ID with ACL check.
func (d *DB) GetSnippet(userID, snippetID int) (*Snippet, error) {
	query := `
		SELECT id, user_id, name, name_norm, description, content, shortcut, created_at, updated_at
		FROM snippets
		WHERE id = ? AND user_id = ?
	`

	var s Snippet
	err := d.QueryRow(query, snippetID, userID).Scan(
		&s.ID, &s.UserID, &s.Name, &s.NameNorm, &s.Description, &s.Content, &s.Shortcut, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query snippet: %w", err)
	}

	return &s, nil
}

// CreateSnippet creates a new snippet for a user.
func (d *DB) CreateSnippet(userID int, name, description, content, shortcut string) (*Snippet, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("snippet name cannot be empty")
	}
	if len(name) > constraints.MaxSnippetNameSize {
		return nil, fmt.Errorf("snippet name must not exceed %d characters", constraints.MaxSnippetNameSize)
	}

	if len(content) > constraints.MaxSnippetContentSize {
		return nil, fmt.Errorf("snippet content must not exceed %d bytes", constraints.MaxSnippetContentSize)
	}

	nameNorm := normalizeSnippetName(name)

	result, err := d.Exec(`
		INSERT INTO snippets (user_id, name, name_norm, description, content, shortcut)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, name, nameNorm, description, content, shortcut)
	if err != nil {
		return nil, fmt.Errorf("failed to insert snippet: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get snippet id: %w", err)
	}
	snippetID, err := validateLastInsertID(id, "snippet id")
	if err != nil {
		return nil, err
	}

	return d.GetSnippet(userID, snippetID)
}

// UpdateSnippet updates an existing snippet with ACL check.
func (d *DB) UpdateSnippet(userID, snippetID int, name, description, content, shortcut string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("snippet name cannot be empty")
	}
	if len(name) > constraints.MaxSnippetNameSize {
		return fmt.Errorf("snippet name must not exceed %d characters", constraints.MaxSnippetNameSize)
	}

	if len(content) > constraints.MaxSnippetContentSize {
		return fmt.Errorf("snippet content must not exceed %d bytes", constraints.MaxSnippetContentSize)
	}

	nameNorm := normalizeSnippetName(name)

	result, err := d.Exec(`
		UPDATE snippets
		SET name = ?, name_norm = ?, description = ?, content = ?, shortcut = ?
		WHERE id = ? AND user_id = ?
	`, name, nameNorm, description, content, shortcut, snippetID, userID)
	if err != nil {
		return fmt.Errorf("failed to update snippet: %w", err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return err
	}

	return nil
}

// DeleteSnippet deletes a snippet with ACL check.
func (d *DB) DeleteSnippet(userID, snippetID int) error {
	result, err := d.Exec(`
		DELETE FROM snippets
		WHERE id = ? AND user_id = ?
	`, snippetID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete snippet: %w", err)
	}
	if err := ensureRowsAffected(result); err != nil {
		return err
	}

	return nil
}
