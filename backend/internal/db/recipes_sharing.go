package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// GetCollectionOwnerUserID returns the owner user_id for a collection.
func (db *DB) GetCollectionOwnerUserID(collectionID int) (int, error) {
	var userID int
	err := db.QueryRow(`SELECT user_id FROM recipe_collections WHERE id = ?`, collectionID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get collection owner: %w", err)
	}
	return userID, nil
}

// CreateCollectionShare creates a new collection sharing record.
func (db *DB) CreateCollectionShare(ownerUserID, collectionID, sharedWithUserID int, role string) (*CollectionShare, error) {
	if role != "viewer" && role != "editor" {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	result, err := db.Exec(`
		INSERT INTO recipe_collection_shares (collection_id, owner_user_id, shared_with_user_id, role)
		VALUES (?, ?, ?, ?)
	`, collectionID, ownerUserID, sharedWithUserID, role)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to create collection share: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get collection share id: %w", err)
	}
	shareID, err := validateLastInsertID(id, "collection share id")
	if err != nil {
		return nil, err
	}
	return db.getCollectionShareByID(shareID)
}

// DeleteCollectionShare removes a collection sharing record.
func (db *DB) DeleteCollectionShare(ownerUserID, collectionID, sharedWithUserID int) error {
	result, err := db.Exec(`
		DELETE FROM recipe_collection_shares
		WHERE collection_id = ? AND owner_user_id = ? AND shared_with_user_id = ?
	`, collectionID, ownerUserID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to delete collection share: %w", err)
	}
	return ensureRowsAffectedWithContext(result, "failed to check rows affected")
}

// GetCollectionShares returns all shares for a specific collection (owner view).
func (db *DB) GetCollectionShares(ownerUserID, collectionID int) ([]CollectionShare, error) {
	rows, err := db.Query(`
		SELECT rcs.id, rcs.collection_id, rc.name,
		       rcs.owner_user_id, ou.username,
		       rcs.shared_with_user_id, su.username, rcs.role,
		       rcs.created_at, rcs.updated_at
		FROM recipe_collection_shares rcs
		JOIN recipe_collections rc ON rc.id = rcs.collection_id
		JOIN users ou ON ou.id = rcs.owner_user_id
		JOIN users su ON su.id = rcs.shared_with_user_id
		WHERE rcs.collection_id = ? AND rcs.owner_user_id = ?
		ORDER BY rcs.created_at DESC
	`, collectionID, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection shares: %w", err)
	}
	defer rows.Close()

	var shares []CollectionShare
	for rows.Next() {
		var s CollectionShare
		if err := rows.Scan(
			&s.ID, &s.CollectionID, &s.CollectionName,
			&s.OwnerUserID, &s.OwnerUsername,
			&s.SharedWithUserID, &s.SharedWithUsername, &s.Role,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan collection share: %w", err)
		}
		shares = append(shares, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection shares: %w", err)
	}

	return shares, nil
}

// UpdateCollectionShareRole updates the role for a collection share record.
func (db *DB) UpdateCollectionShareRole(ownerUserID, collectionID, sharedWithUserID int, role string) error {
	if role != "viewer" && role != "editor" {
		return fmt.Errorf("invalid role: %s", role)
	}

	result, err := db.Exec(`
		UPDATE recipe_collection_shares
		SET role = ?, updated_at = datetime('now')
		WHERE collection_id = ? AND owner_user_id = ? AND shared_with_user_id = ?
	`, role, collectionID, ownerUserID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to update collection share role: %w", err)
	}
	return ensureRowsAffectedWithContext(result, "failed to check rows affected")
}

// GetSharedCollectionsForUser returns all collections shared with a user.
func (db *DB) GetSharedCollectionsForUser(userID int) ([]SharedCollection, error) {
	rows, err := db.Query(`
		SELECT rc.id, rc.name, rc.description, rc.color,
		       (SELECT COUNT(*) FROM recipe_collection_items ci
		        JOIN notes n ON n.id = ci.note_id
		        WHERE ci.collection_id = rc.id AND n.is_deleted = 0 AND n.note_type = 'recipe') as recipe_count,
		       ou.username, rcs.role, rcs.id,
		       rc.created_at, rc.updated_at
		FROM recipe_collection_shares rcs
		JOIN recipe_collections rc ON rc.id = rcs.collection_id
		JOIN users ou ON ou.id = rcs.owner_user_id
		WHERE rcs.shared_with_user_id = ?
		ORDER BY ou.username, rc.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared collections: %w", err)
	}
	defer rows.Close()

	var collections []SharedCollection
	for rows.Next() {
		var sc SharedCollection
		var description, color sql.NullString
		if err := rows.Scan(
			&sc.ID, &sc.Name, &description, &color,
			&sc.RecipeCount,
			&sc.SharedBy, &sc.ShareRole, &sc.ShareID,
			&sc.CreatedAt, &sc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shared collection: %w", err)
		}
		if description.Valid {
			sc.Description = &description.String
		}
		if color.Valid {
			sc.Color = &color.String
		}
		collections = append(collections, sc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared collections: %w", err)
	}

	return collections, nil
}

// GetCollectionSharePermission returns the role for a user on a collection,
// or empty string if no share exists.
func (db *DB) GetCollectionSharePermission(userID, collectionID int) (string, error) {
	var role string
	err := db.QueryRow(`
		SELECT role FROM recipe_collection_shares
		WHERE shared_with_user_id = ? AND collection_id = ?
	`, userID, collectionID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get collection share permission: %w", err)
	}
	return role, nil
}

// ListRecipesInSharedCollection returns all recipe notes in a shared collection.
// SECURITY: No user_id filter — must only be called after permission check in service layer.
func (db *DB) ListRecipesInSharedCollection(collectionID int) ([]Note, error) {
	rows, err := db.Query(`
		SELECT n.id, n.title, n.content, n.folder_path, n.version, n.display_order,
		       n.color, n.created_at, n.updated_at,
		       n.encrypted_content, n.content_encrypted, n.encrypted_title, n.title_encrypted,
		       n.wrapped_dek, n.encryption_version, n.encryption_metadata,
		       n.note_type, n.journal_date, n.ai_enabled, n.user_id
		FROM notes n
		JOIN recipe_collection_items ci ON ci.note_id = n.id
		WHERE ci.collection_id = ? AND n.is_deleted = 0 AND n.note_type = 'recipe'
		ORDER BY ci.display_order, n.title
	`, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipes in shared collection: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		var content, encryptedTitle, wrappedDEK, encryptionMetadata sql.NullString
		var encryptedContent []byte
		var noteType, journalDate sql.NullString

		if err := rows.Scan(
			&note.ID, &note.Title, &content, &note.FolderPath, &note.Version,
			&note.DisplayOrder, &note.Color, &createdAt, &updatedAt,
			&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
			&wrappedDEK, &note.EncryptionVersion, &encryptionMetadata,
			&noteType, &journalDate, &note.AIEnabled, &note.UserID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan recipe note: %w", err)
		}

		note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		note.Content = content.String
		note.EncryptedContent = encryptedContent
		if encryptedTitle.Valid {
			note.EncryptedTitle = &encryptedTitle.String
		}
		note.WrappedDEK = wrappedDEK.String
		note.EncryptionMetadata = encryptionMetadata.String
		if noteType.Valid {
			note.NoteType = noteType.String
		}
		if journalDate.Valid {
			note.JournalDate = &journalDate.String
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipes: %w", err)
	}

	if notes == nil {
		notes = []Note{}
	}
	return notes, nil
}

// CollectionHasEncryptedRecipes returns true if any recipe in the collection is encrypted.
func (db *DB) CollectionHasEncryptedRecipes(collectionID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM recipe_collection_items ci
		JOIN notes n ON n.id = ci.note_id
		WHERE ci.collection_id = ? AND n.is_deleted = 0 AND n.content_encrypted = 1
	`, collectionID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check collection encrypted recipes: %w", err)
	}
	return count > 0, nil
}

// getCollectionShareByID retrieves a collection share by its ID (internal helper).
func (db *DB) getCollectionShareByID(id int) (*CollectionShare, error) {
	var s CollectionShare
	err := db.QueryRow(`
		SELECT rcs.id, rcs.collection_id, rc.name,
		       rcs.owner_user_id, ou.username,
		       rcs.shared_with_user_id, su.username, rcs.role,
		       rcs.created_at, rcs.updated_at
		FROM recipe_collection_shares rcs
		JOIN recipe_collections rc ON rc.id = rcs.collection_id
		JOIN users ou ON ou.id = rcs.owner_user_id
		JOIN users su ON su.id = rcs.shared_with_user_id
		WHERE rcs.id = ?
	`, id).Scan(
		&s.ID, &s.CollectionID, &s.CollectionName,
		&s.OwnerUserID, &s.OwnerUsername,
		&s.SharedWithUserID, &s.SharedWithUsername, &s.Role,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get collection share: %w", err)
	}
	return &s, nil
}
