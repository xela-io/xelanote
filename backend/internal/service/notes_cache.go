// Package service contains the business logic for xelanote.
package service

import "fmt"

func (s *NoteService) invalidateNoteCache(userID int, noteID string) {
	s.cache.Delete(noteCacheKey(userID, noteID))
}

func (s *NoteService) invalidateNotesByFolderCache(userID int, path string) {
	if path == "" {
		return
	}
	s.cache.Delete(notesByFolderCacheKey(userID, normalizeFolderPath(path)))
}

func (s *NoteService) invalidateQuickSearchCache(userID int) {
	s.cache.DeleteByPrefix(fmt.Sprintf("cache:notes:quick:%d:", userID))
}
