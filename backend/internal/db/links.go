package db

import (
	"fmt"
)

// Link represents a resolved link between two notes.
type Link struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
}

// UnresolvedLink represents a link to a note that doesn't exist yet.
type UnresolvedLink struct {
	SourceID      string `json:"source_id"`
	TargetRef     string `json:"target_ref"`
	TargetRefNorm string `json:"target_ref_norm"`
}

// Backlink represents a note that links to another note.
type Backlink struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SetLinks replaces all links for a source note.
// resolvedTargetIDs are IDs of existing notes.
// unresolvedRefs are titles of notes that don't exist yet.
func (db *DB) SetLinks(sourceID string, resolvedTargetIDs []string, unresolvedRefs []string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing links
	_, err = tx.Exec("DELETE FROM links WHERE source_id = ?", sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete existing links: %w", err)
	}

	_, err = tx.Exec("DELETE FROM unresolved_links WHERE source_id = ?", sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete existing unresolved links: %w", err)
	}

	// Insert resolved links
	if len(resolvedTargetIDs) > 0 {
		stmt, err := tx.Prepare("INSERT OR IGNORE INTO links (source_id, target_id) VALUES (?, ?)")
		if err != nil {
			return fmt.Errorf("failed to prepare links insert: %w", err)
		}
		defer stmt.Close()

		for _, targetID := range resolvedTargetIDs {
			if _, err := stmt.Exec(sourceID, targetID); err != nil {
				return fmt.Errorf("failed to insert link: %w", err)
			}
		}
	}

	// Insert unresolved links
	if len(unresolvedRefs) > 0 {
		stmt, err := tx.Prepare("INSERT OR IGNORE INTO unresolved_links (source_id, target_ref, target_ref_norm) VALUES (?, ?, ?)")
		if err != nil {
			return fmt.Errorf("failed to prepare unresolved links insert: %w", err)
		}
		defer stmt.Close()

		for _, ref := range unresolvedRefs {
			refNorm := NormalizeTitle(ref)
			if _, err := stmt.Exec(sourceID, ref, refNorm); err != nil {
				return fmt.Errorf("failed to insert unresolved link: %w", err)
			}
		}
	}

	return tx.Commit()
}

// GetBacklinks returns all notes that link to the given note ID.
func (db *DB) GetBacklinks(userID int, noteID string) ([]Backlink, error) {
	rows, err := db.Query(`
		SELECT n.id, n.title
		FROM notes n
		JOIN links l ON l.source_id = n.id
		JOIN notes t ON t.id = l.target_id
		WHERE l.target_id = ?
		  AND n.user_id = ?
		  AND t.user_id = ?
		  AND n.is_deleted = 0
		  AND t.is_deleted = 0
		ORDER BY n.title ASC
	`, noteID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backlinks: %w", err)
	}
	defer rows.Close()

	var backlinks []Backlink
	for rows.Next() {
		var bl Backlink
		if err := rows.Scan(&bl.ID, &bl.Title); err != nil {
			return nil, fmt.Errorf("failed to scan backlink: %w", err)
		}
		backlinks = append(backlinks, bl)
	}

	return backlinks, nil
}

// GetUnresolvedBacklinks returns notes that have unresolved links to the given title.
// This is useful when a new note is created to resolve pending links.
// Only returns notes owned by the specified user.
func (db *DB) GetUnresolvedBacklinks(userID int, title string) ([]Backlink, error) {
	titleNorm := NormalizeTitle(title)

	rows, err := db.Query(`
		SELECT n.id, n.title
		FROM notes n
		JOIN unresolved_links ul ON ul.source_id = n.id
		WHERE ul.target_ref_norm = ? AND n.user_id = ? AND n.is_deleted = 0
		ORDER BY n.title ASC
	`, titleNorm, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unresolved backlinks: %w", err)
	}
	defer rows.Close()

	var backlinks []Backlink
	for rows.Next() {
		var bl Backlink
		if err := rows.Scan(&bl.ID, &bl.Title); err != nil {
			return nil, fmt.Errorf("failed to scan backlink: %w", err)
		}
		backlinks = append(backlinks, bl)
	}

	return backlinks, nil
}

// GetOutgoingLinks returns all notes that the given note links to.
// Only returns target notes owned by the specified user.
func (db *DB) GetOutgoingLinks(userID int, noteID string) ([]Backlink, error) {
	rows, err := db.Query(`
		SELECT n.id, n.title
		FROM notes n
		JOIN links l ON l.target_id = n.id
		JOIN notes src ON src.id = l.source_id
		WHERE l.source_id = ? AND n.user_id = ? AND src.user_id = ? AND n.is_deleted = 0
		ORDER BY n.title ASC
	`, noteID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing links: %w", err)
	}
	defer rows.Close()

	var links []Backlink
	for rows.Next() {
		var link Backlink
		if err := rows.Scan(&link.ID, &link.Title); err != nil {
			return nil, fmt.Errorf("failed to scan link: %w", err)
		}
		links = append(links, link)
	}

	return links, nil
}

// GetUnresolvedOutgoingLinks returns unresolved link targets for a note.
// Only returns links from notes owned by the specified user.
func (db *DB) GetUnresolvedOutgoingLinks(userID int, noteID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT ul.target_ref
		FROM unresolved_links ul
		JOIN notes n ON n.id = ul.source_id
		WHERE ul.source_id = ? AND n.user_id = ?
		ORDER BY ul.target_ref ASC
	`, noteID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unresolved outgoing links: %w", err)
	}
	defer rows.Close()

	var refs []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			return nil, fmt.Errorf("failed to scan ref: %w", err)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

// ResolveUnresolvedLink converts a single unresolved link to a resolved link.
// This is used when a new note is created that matches an existing unresolved link,
// especially for encrypted notes where we can't re-parse the source content.
func (db *DB) ResolveUnresolvedLink(sourceID string, targetTitle string, targetNoteID string) error {
	titleNorm := NormalizeTitle(targetTitle)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete the unresolved link
	_, err = tx.Exec(`
		DELETE FROM unresolved_links
		WHERE source_id = ? AND target_ref_norm = ?
	`, sourceID, titleNorm)
	if err != nil {
		return fmt.Errorf("failed to delete unresolved link: %w", err)
	}

	// Insert the resolved link (ignore if already exists)
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO links (source_id, target_id)
		VALUES (?, ?)
	`, sourceID, targetNoteID)
	if err != nil {
		return fmt.Errorf("failed to insert resolved link: %w", err)
	}

	return tx.Commit()
}

// GetNotesLinkingTo returns source IDs of notes linking to the given note.
// Includes both resolved and unresolved links (by title).
// Only returns notes owned by the specified user.
func (db *DB) GetNotesLinkingTo(userID int, noteID string, noteTitle string) ([]string, error) {
	titleNorm := NormalizeTitle(noteTitle)

	rows, err := db.Query(`
		SELECT DISTINCT l.source_id
		FROM links l
		JOIN notes n ON n.id = l.source_id
		WHERE l.target_id = ? AND n.user_id = ? AND n.is_deleted = 0
		UNION
		SELECT DISTINCT ul.source_id
		FROM unresolved_links ul
		JOIN notes n ON n.id = ul.source_id
		WHERE ul.target_ref_norm = ? AND n.user_id = ? AND n.is_deleted = 0
	`, noteID, userID, titleNorm, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes linking to: %w", err)
	}
	defer rows.Close()

	var sourceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan source id: %w", err)
		}
		sourceIDs = append(sourceIDs, id)
	}

	return sourceIDs, nil
}
