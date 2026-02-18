package api

import (
	"errors"
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
)

type shareResource string

const (
	shareRoleViewer = "viewer"
	shareRoleEditor = "editor"

	shareResourceNote       shareResource = "note"
	shareResourceFolder     shareResource = "folder"
	shareResourceCollection shareResource = "collection"
)

func validateShareCreateInput(identifier, role string) string {
	if identifier == "" {
		return "identifier (username or email) required"
	}
	return validateShareRoleInput(role)
}

func validateShareRoleInput(role string) string {
	if role != shareRoleViewer && role != shareRoleEditor {
		return "role must be 'viewer' or 'editor'"
	}
	return ""
}

func mapShareCreateError(resource shareResource, err error) (int, string, bool) {
	switch resource {
	case shareResourceNote:
		switch {
		case errors.Is(err, service.ErrNotFound):
			return http.StatusNotFound, "note not found", true
		case errors.Is(err, service.ErrCannotShareEncrypted):
			return http.StatusBadRequest, err.Error(), true
		case errors.Is(err, service.ErrCannotShareWithSelf):
			return http.StatusBadRequest, err.Error(), true
		case errors.Is(err, service.ErrNotNoteOwner):
			return http.StatusForbidden, "only the note owner can share", true
		case errors.Is(err, service.ErrUserNotFound):
			return http.StatusBadRequest, "unable to share with specified user", true
		}
	case shareResourceFolder:
		switch {
		case errors.Is(err, service.ErrNotFound):
			return http.StatusNotFound, "folder not found", true
		case errors.Is(err, service.ErrCannotShareEncryptedFolder):
			return http.StatusBadRequest, err.Error(), true
		case errors.Is(err, service.ErrFolderHasEncryptedNotes):
			return http.StatusBadRequest, err.Error(), true
		case errors.Is(err, service.ErrCannotShareWithSelf):
			return http.StatusBadRequest, err.Error(), true
		case errors.Is(err, service.ErrNotFolderOwner):
			return http.StatusForbidden, "only the folder owner can share", true
		case errors.Is(err, service.ErrUserNotFound):
			return http.StatusBadRequest, "unable to share with specified user", true
		}
	case shareResourceCollection:
		switch {
		case errors.Is(err, service.ErrNotFound):
			return http.StatusNotFound, "collection not found", true
		case errors.Is(err, service.ErrNotCollectionOwner):
			return http.StatusForbidden, "only the collection owner can share", true
		case errors.Is(err, service.ErrCollectionHasEncryptedRecipes):
			return http.StatusBadRequest, err.Error(), true
		case errors.Is(err, service.ErrCannotShareWithSelf):
			return http.StatusBadRequest, err.Error(), true
		case errors.Is(err, service.ErrCollectionAlreadyShared):
			return http.StatusConflict, err.Error(), true
		case errors.Is(err, service.ErrUserNotFound):
			return http.StatusBadRequest, "unable to share with specified user", true
		}
	}

	return 0, "", false
}

func mapShareAccessError(resource shareResource, err error) (int, string, bool) {
	switch resource {
	case shareResourceNote:
		switch {
		case errors.Is(err, service.ErrNotFound):
			return http.StatusNotFound, "note not found", true
		case errors.Is(err, service.ErrNotNoteOwner):
			return http.StatusForbidden, "only the note owner can view shares", true
		}
	case shareResourceFolder:
		switch {
		case errors.Is(err, service.ErrNotFound):
			return http.StatusNotFound, "folder not found", true
		case errors.Is(err, service.ErrNotFolderOwner):
			return http.StatusForbidden, "only the folder owner can view shares", true
		}
	case shareResourceCollection:
		switch {
		case errors.Is(err, service.ErrNotFound):
			return http.StatusNotFound, "collection not found", true
		case errors.Is(err, service.ErrNotCollectionOwner):
			return http.StatusForbidden, "only the collection owner can view shares", true
		}
	}

	return 0, "", false
}

func mapShareMutateError(resource shareResource, action string, err error) (int, string, bool) {
	switch resource {
	case shareResourceNote:
		switch {
		case errors.Is(err, service.ErrNotFound):
			return http.StatusNotFound, "share not found", true
		case errors.Is(err, service.ErrNotNoteOwner):
			switch action {
			case "update":
				return http.StatusForbidden, "only the note owner can update shares", true
			case "remove":
				return http.StatusForbidden, "only the note owner can remove shares", true
			default:
				return http.StatusForbidden, "only the note owner can manage shares", true
			}
		}
	case shareResourceFolder:
		switch {
		case errors.Is(err, service.ErrNotFound):
			return http.StatusNotFound, "share not found", true
		case errors.Is(err, service.ErrNotFolderOwner):
			switch action {
			case "update":
				return http.StatusForbidden, "only the folder owner can update shares", true
			case "remove":
				return http.StatusForbidden, "only the folder owner can remove shares", true
			default:
				return http.StatusForbidden, "only the folder owner can manage shares", true
			}
		}
	case shareResourceCollection:
		switch {
		case errors.Is(err, service.ErrNotFound):
			return http.StatusNotFound, "share not found", true
		case errors.Is(err, service.ErrNotCollectionOwner):
			return http.StatusForbidden, "only the collection owner can manage shares", true
		}
	}

	return 0, "", false
}
