package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
)

// resolveOwnerID determines the owner user_id for a recipe note (I1).
// If callerUserID is the owner, returns callerUserID.
// If callerUserID has share access (editor), returns the actual owner.
func (s *RecipeService) resolveOwnerID(noteID string, callerUserID int) (int, string, error) {
	// Try as owner first
	note, err := s.db.GetNote(callerUserID, noteID)
	if err == nil && note != nil {
		return note.UserID, "", nil
	}

	// Not found as owner — check share permission
	perm, err := s.db.GetSharePermission(callerUserID, noteID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to check share permission: %w", err)
	}
	if perm == "" {
		return 0, "", ErrForbidden
	}

	ownerID, err := s.db.GetNoteOwnerUserID(noteID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get note owner: %w", err)
	}

	return ownerID, perm, nil
}

// checkRecipeWriteAccess verifies ownership/editor permission and encryption guard.
// Returns the ownerID for subsequent DB operations.
func (s *RecipeService) checkRecipeWriteAccess(callerUserID int, noteID string) (int, error) {
	ownerID, perm, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return 0, err
	}
	if callerUserID != ownerID && perm != "editor" {
		return 0, ErrForbidden
	}
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return 0, err
	}
	if encrypted {
		return 0, ErrRecipeEncrypted
	}
	return ownerID, nil
}

// checkRecipeReadAccess verifies read access and encryption guard.
// Returns the ownerID for subsequent DB operations.
func (s *RecipeService) checkRecipeReadAccess(callerUserID int, noteID string) (int, error) {
	ownerID, _, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return 0, err
	}
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return 0, err
	}
	if encrypted {
		return 0, ErrRecipeEncrypted
	}
	return ownerID, nil
}

// checkFeatureEnabled verifies the recipe feature is enabled for the user.
func (s *RecipeService) checkFeatureEnabled(userID int) error {
	feature, err := s.db.GetUserFeature(userID, "recipe")
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return ErrRecipeFeatureNotEnabled
	}
	return nil
}

// --- Validation helpers ---

func validateRecipeMetadata(m *db.RecipeMetadata) error {
	if m.Servings < 1 || m.Servings > 999 {
		return fmt.Errorf("servings must be between 1 and 999")
	}
	if m.PrepTimeMinutes != nil && (*m.PrepTimeMinutes < 0 || *m.PrepTimeMinutes > 99999) {
		return fmt.Errorf("prep_time_minutes must be between 0 and 99999")
	}
	if m.CookTimeMinutes != nil && (*m.CookTimeMinutes < 0 || *m.CookTimeMinutes > 99999) {
		return fmt.Errorf("cook_time_minutes must be between 0 and 99999")
	}
	if m.Difficulty != nil {
		d := *m.Difficulty
		if d != "easy" && d != "medium" && d != "hard" {
			return fmt.Errorf("difficulty must be 'easy', 'medium', or 'hard'")
		}
	}
	if m.SourceURL != nil {
		u := *m.SourceURL
		if len(u) > 2048 {
			return fmt.Errorf("source_url too long (max 2048)")
		}
		if u != "" {
			parsed, err := url.Parse(u)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return fmt.Errorf("source_url must be a valid http/https URL")
			}
		}
	}
	return nil
}

func validateIngredients(ingredients []db.RecipeIngredient) error {
	for i, ing := range ingredients {
		if strings.TrimSpace(ing.Name) == "" {
			return fmt.Errorf("ingredient %d: name is required", i)
		}
		if len(ing.Name) > 200 {
			return fmt.Errorf("ingredient %d: name too long (max 200)", i)
		}
		if ing.Unit != nil && len(*ing.Unit) > 50 {
			return fmt.Errorf("ingredient %d: unit too long (max 50)", i)
		}
		if ing.GroupName != nil && len(*ing.GroupName) > 100 {
			return fmt.Errorf("ingredient %d: group_name too long (max 100)", i)
		}
		if ing.AmountText != nil && len(*ing.AmountText) > 100 {
			return fmt.Errorf("ingredient %d: amount_text too long (max 100)", i)
		}
	}
	return nil
}
