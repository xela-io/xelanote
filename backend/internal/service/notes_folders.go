// Package service contains the business logic for xelanote.
package service

import (
	"errors"
	"fmt"

	"github.com/xela-io/xelanote/internal/db"
)

// GetFolders returns all folder paths.
func (s *NoteService) GetFolders(userID int) ([]string, error) {
	return s.db.GetFolders(userID)
}

// GetFoldersWithCounts returns all folder paths with note counts.
func (s *NoteService) GetFoldersWithCounts(userID int) ([]db.FolderInfo, error) {
	return s.db.GetFoldersWithCounts(userID)
}

// GetNotesByFolder returns notes in a folder.
func (s *NoteService) GetNotesByFolder(userID int, path string, fields string) ([]db.Note, error) {
	path = normalizeFolderPath(path)

	// Only use cache for full (non-slim) requests to avoid serving slim data to full-data callers
	if fields == "" {
		key := notesByFolderCacheKey(userID, path)
		if cached, ok := s.cache.Get(key); ok {
			return cached.([]db.Note), nil
		}

		notes, err := s.db.ListNotesByFolder(userID, path, fields)
		if err != nil {
			return nil, err
		}

		s.cache.Set(key, notes)
		return notes, nil
	}

	return s.db.ListNotesByFolder(userID, path, fields)
}

// Folder operations (new folders table)

// GetAllFolders returns all folders with note counts from the folders table.
func (s *NoteService) GetAllFolders(userID int) ([]db.Folder, error) {
	key := fmt.Sprintf("cache:folders:%d:all", userID)

	// Cache hit?
	if cached, ok := s.cache.Get(key); ok {
		return cached.([]db.Folder), nil
	}

	// Cache miss - query database
	folders, err := s.db.GetAllFolders(userID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cache.Set(key, folders)
	return folders, nil
}

// GetFolderByPath retrieves a folder by path.
func (s *NoteService) GetFolderByPath(userID int, path string) (*db.Folder, error) {
	key := fmt.Sprintf("cache:folder:%d:%s", userID, path)

	// Cache hit?
	if cached, ok := s.cache.Get(key); ok {
		return cached.(*db.Folder), nil
	}

	// Cache miss - query database
	folder, err := s.db.GetFolderByPath(userID, path)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cache.Set(key, folder)
	return folder, nil
}

// GetFolderByID retrieves a folder by ID.
func (s *NoteService) GetFolderByID(userID int, id int) (*db.Folder, error) {
	key := fmt.Sprintf("cache:folder_id:%d:%d", userID, id)

	// Cache hit?
	if cached, ok := s.cache.Get(key); ok {
		return cached.(*db.Folder), nil
	}

	// Cache miss - query database
	folder, err := s.db.GetFolderByID(userID, id)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cache.Set(key, folder)
	return folder, nil
}

// invalidateFolderCache clears all folder-related cache entries for a user
func (s *NoteService) invalidateFolderCache(userID int) {
	// Broad invalidation strategy: remove all folder caches for this user
	// This ensures we never serve stale data after mutations
	s.cache.DeleteByPrefix(fmt.Sprintf("cache:folder:%d:", userID))
	s.cache.DeleteByPrefix(fmt.Sprintf("cache:folder_id:%d:", userID))
	s.cache.Delete(fmt.Sprintf("cache:folders:%d:all", userID))
}

// CreateFolder creates a new folder.
func (s *NoteService) CreateFolder(userID int, path string, parentID *int) (*db.Folder, error) {
	folder, err := s.db.CreateFolder(userID, path, parentID)
	if err != nil {
		return nil, err
	}

	// Invalidate folder cache after mutation
	s.invalidateFolderCache(userID)
	return folder, nil
}

// MoveFolder moves a folder to a new parent.
func (s *NoteService) MoveFolder(userID int, folderID int, newParentPath string) error {
	err := s.db.MoveFolder(userID, folderID, newParentPath)
	if err != nil {
		return err
	}

	// Invalidate folder cache after mutation
	s.invalidateFolderCache(userID)
	return nil
}

// DeleteFolder deletes a folder.
func (s *NoteService) DeleteFolder(userID int, folderID int) error {
	err := s.db.DeleteFolder(userID, folderID)
	if err != nil {
		return err
	}

	// Invalidate folder cache after mutation
	s.invalidateFolderCache(userID)
	return nil
}

// RenameFolder renames a folder.
func (s *NoteService) RenameFolder(userID int, folderID int, newName string) error {
	err := s.db.RenameFolder(userID, folderID, newName)
	if err != nil {
		return err
	}

	// Invalidate folder cache after mutation
	s.invalidateFolderCache(userID)
	return nil
}

// ReorderFolders updates the display order of folders.
func (s *NoteService) ReorderFolders(userID int, parentID *int, items []int) error {
	err := s.db.ReorderFolders(userID, parentID, items)
	if err != nil {
		return err
	}

	// Invalidate folder cache after mutation
	s.invalidateFolderCache(userID)
	return nil
}

// ReorderNotes updates the display order of notes within a folder.
func (s *NoteService) ReorderNotes(userID int, folderPath string, noteIDs []string) error {
	folderPath = normalizeFolderPath(folderPath)
	return s.db.ReorderNotes(userID, folderPath, noteIDs)
}

// UpdateFolderColor updates the color of a folder.
func (s *NoteService) UpdateFolderColor(userID int, folderID int, color *string) error {
	err := s.db.UpdateFolderColor(userID, folderID, color)
	if err != nil {
		return err
	}

	// Invalidate folder cache after mutation
	s.invalidateFolderCache(userID)
	return nil
}

// UpdateFolderAIEnabledDefault sets the default AI-enabled flag for a folder.
// New notes created in this folder will inherit this setting.
func (s *NoteService) UpdateFolderAIEnabledDefault(userID int, folderID int, aiEnabled bool) error {
	if err := s.db.UpdateFolderAIEnabledDefault(userID, folderID, aiEnabled); err != nil {
		return err
	}

	// Invalidate folder cache
	s.invalidateFolderCache(userID)

	return nil
}

// GetFolderAIEnabledDefault returns the default ai_enabled setting for a folder.
func (s *NoteService) GetFolderAIEnabledDefault(userID int, folderID int) (bool, error) {
	return s.db.GetFolderAIEnabledDefault(userID, folderID)
}

// UpdateFolderEncryptionDefault sets the encryption_default flag for a folder.
// New notes created in this folder will inherit this setting.
// Cannot enable encryption on a shared folder (must remove all shares first).
func (s *NoteService) UpdateFolderEncryptionDefault(userID int, folderID int, encrypted bool) error {
	// Guard: cannot enable encryption if folder is shared
	if encrypted {
		shares, err := s.db.GetFolderSharesByFolderID(folderID)
		if err != nil {
			return err
		}
		if len(shares) > 0 {
			return errors.New("cannot enable encryption on a shared folder — remove all shares first")
		}
	}

	if err := s.db.UpdateFolderEncryptionDefault(userID, folderID, encrypted); err != nil {
		return err
	}

	// Invalidate folder cache
	s.invalidateFolderCache(userID)

	return nil
}

// GetFolderEncryptionDefault returns the default encryption setting for a folder.
func (s *NoteService) GetFolderEncryptionDefault(userID int, folderID int) (bool, error) {
	return s.db.GetFolderEncryptionDefault(userID, folderID)
}
