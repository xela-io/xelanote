package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	MaxTemplateContentSize = 102400 // 100KB
	MaxTemplateTitleSize   = 200
	MaxTemplateNameSize    = 100
)

// Template represents a reusable note template.
type Template struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	NameNorm    string    `json:"-"`
	Description string    `json:"description"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// normalizeTemplateName normalizes a template name for case-insensitive matching.
func normalizeTemplateName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// GetAllTemplates returns all templates for a user.
func (d *DB) GetAllTemplates(userID int) ([]Template, error) {
	query := `
		SELECT id, user_id, name, name_norm, description, title, content, created_at, updated_at
		FROM templates
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`

	rows, err := d.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.NameNorm, &t.Description, &t.Title, &t.Content, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating templates: %w", err)
	}

	return templates, nil
}

// GetTemplate returns a single template by ID with ACL check.
func (d *DB) GetTemplate(userID, templateID int) (*Template, error) {
	query := `
		SELECT id, user_id, name, name_norm, description, title, content, created_at, updated_at
		FROM templates
		WHERE id = ? AND user_id = ?
	`

	var t Template
	err := d.QueryRow(query, templateID, userID).Scan(
		&t.ID, &t.UserID, &t.Name, &t.NameNorm, &t.Description, &t.Title, &t.Content, &t.CreatedAt, &t.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query template: %w", err)
	}

	return &t, nil
}

// CreateTemplate creates a new template for a user.
func (d *DB) CreateTemplate(userID int, name, description, title, content string) (*Template, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("template name cannot be empty")
	}
	if len(name) > MaxTemplateNameSize {
		return nil, fmt.Errorf("template name must not exceed %d characters", MaxTemplateNameSize)
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("template title cannot be empty")
	}
	if len(title) > MaxTemplateTitleSize {
		return nil, fmt.Errorf("template title must not exceed %d characters", MaxTemplateTitleSize)
	}

	if len(content) > MaxTemplateContentSize {
		return nil, fmt.Errorf("template content must not exceed %d bytes", MaxTemplateContentSize)
	}

	nameNorm := normalizeTemplateName(name)

	result, err := d.Exec(`
		INSERT INTO templates (user_id, name, name_norm, description, title, content)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, name, nameNorm, description, title, content)
	if err != nil {
		return nil, fmt.Errorf("failed to insert template: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get template id: %w", err)
	}
	templateID, err := validateLastInsertID(id, "template id")
	if err != nil {
		return nil, err
	}

	return d.GetTemplate(userID, templateID)
}

// UpdateTemplate updates an existing template with ACL check.
func (d *DB) UpdateTemplate(userID, templateID int, name, description, title, content string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}
	if len(name) > MaxTemplateNameSize {
		return fmt.Errorf("template name must not exceed %d characters", MaxTemplateNameSize)
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("template title cannot be empty")
	}
	if len(title) > MaxTemplateTitleSize {
		return fmt.Errorf("template title must not exceed %d characters", MaxTemplateTitleSize)
	}

	if len(content) > MaxTemplateContentSize {
		return fmt.Errorf("template content must not exceed %d bytes", MaxTemplateContentSize)
	}

	nameNorm := normalizeTemplateName(name)

	result, err := d.Exec(`
		UPDATE templates
		SET name = ?, name_norm = ?, description = ?, title = ?, content = ?
		WHERE id = ? AND user_id = ?
	`, name, nameNorm, description, title, content, templateID, userID)
	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
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

// DeleteTemplate deletes a template with ACL check.
func (d *DB) DeleteTemplate(userID, templateID int) error {
	result, err := d.Exec(`
		DELETE FROM templates
		WHERE id = ? AND user_id = ?
	`, templateID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
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
