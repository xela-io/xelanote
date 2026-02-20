package api

import (
	"net/http"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
)

// getPreferences returns user preferences, creating defaults if not exist
func (s *Server) getPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	prefs, created, err := s.userService.GetOrCreatePreferences(userID)
	if err != nil {
		s.logger().Error("failed to get preferences", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get preferences")
		return
	}

	// Load WebAuthn credentials
	credentials, err := s.userService.GetWebAuthnCredentials(int64(userID))
	if err != nil {
		s.logger().Error("failed to get webauthn credentials", "error", err)
		// Non-fatal, continue with empty list
		credentials = []service.WebAuthnCredential{}
	}

	respondJSON(w, http.StatusOK, preferencesResponse{
		Theme:             prefs.Theme,
		EditorMode:        prefs.EditorMode,
		KeywordsEnabled:   prefs.KeywordsEnabled,
		EncryptTitles:     prefs.EncryptTitles,
		SecurityLevel:     prefs.SecurityLevel,
		AutoLockTimeout:   prefs.AutoLockTimeout,
		ActiveAIProvider:  prefs.ActiveAIProvider,
		DietaryPreference: prefs.DietaryPreference,
		Credentials:       convertWebAuthnCredentials(credentials),
		Created:           created,
	})
}

// updatePreferences updates user preferences
func (s *Server) updatePreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updatePreferencesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	prefs, err := s.userService.UpdatePreferences(userID, req.Theme, req.EditorMode)
	if err != nil {
		switch err {
		case service.ErrInvalidTheme:
			respondError(w, http.StatusBadRequest, "invalid theme")
		case service.ErrInvalidEditorMode:
			respondError(w, http.StatusBadRequest, "invalid editor mode")
		default:
			s.logger().Error("failed to update preferences", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to update preferences")
		}
		return
	}

	// Load WebAuthn credentials
	credentials, err := s.userService.GetWebAuthnCredentials(int64(userID))
	if err != nil {
		s.logger().Warn("failed to load webauthn credentials", "user_id", userID, "error", err)
		credentials = []service.WebAuthnCredential{}
	}

	respondJSON(w, http.StatusOK, preferencesResponse{
		Theme:             prefs.Theme,
		EditorMode:        prefs.EditorMode,
		KeywordsEnabled:   prefs.KeywordsEnabled,
		EncryptTitles:     prefs.EncryptTitles,
		SecurityLevel:     prefs.SecurityLevel,
		AutoLockTimeout:   prefs.AutoLockTimeout,
		ActiveAIProvider:  prefs.ActiveAIProvider,
		DietaryPreference: prefs.DietaryPreference,
		Credentials:       convertWebAuthnCredentials(credentials),
		Created:           false,
	})
}

// getAIProviderPreference returns the currently selected AI provider.
func (s *Server) getAIProviderPreference(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	provider, err := s.userService.GetActiveAIProvider(userID)
	if err != nil {
		s.logger().Error("failed to get AI provider preference", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get AI provider preference")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"provider": provider})
}

// setAIProviderPreference updates the selected AI provider.
func (s *Server) setAIProviderPreference(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateAIProviderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.userService.SetActiveAIProvider(userID, req.Provider); err != nil {
		if err == service.ErrInvalidAIProvider {
			respondError(w, http.StatusBadRequest, "invalid AI provider")
			return
		}
		s.logger().Error("failed to update AI provider preference", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update AI provider preference")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"provider": req.Provider})
}

// getAIModels returns model preferences for all AI providers.
func (s *Server) getAIModels(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	models, err := s.userService.GetAIModelPreferences(userID)
	if err != nil {
		s.logger().Error("failed to get AI model preferences", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get AI model preferences")
		return
	}

	respondJSON(w, http.StatusOK, aiModelsResponse{
		ClaudeModel:  models.ClaudeModel,
		GeminiModel:  models.GeminiModel,
		ChatGPTModel: models.ChatGPTModel,
	})
}

// setAIModels updates model preferences for all AI providers.
func (s *Server) setAIModels(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateAIModelsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := s.userService.SetAIModelPreferences(userID, &db.AIModelPreferences{
		ClaudeModel:  req.ClaudeModel,
		GeminiModel:  req.GeminiModel,
		ChatGPTModel: req.ChatGPTModel,
	})
	if err != nil {
		if err == service.ErrInvalidAIModel {
			respondError(w, http.StatusBadRequest, "invalid AI model")
			return
		}
		s.logger().Error("failed to update AI model preferences", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update AI model preferences")
		return
	}

	s.summarizeService.InvalidateAllAIClients(userID)

	respondJSON(w, http.StatusOK, aiModelsResponse{
		ClaudeModel:  req.ClaudeModel,
		GeminiModel:  req.GeminiModel,
		ChatGPTModel: req.ChatGPTModel,
	})
}

// getDietaryPreference returns the dietary preference for the user.
func (s *Server) getDietaryPreference(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	pref, err := s.userService.GetDietaryPreference(userID)
	if err != nil {
		s.logger().Error("failed to get dietary preference", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get dietary preference")
		return
	}

	respondJSON(w, http.StatusOK, dietaryPreferenceResponse{DietaryPreference: pref})
}

// setDietaryPreference updates the dietary preference for the user.
func (s *Server) setDietaryPreference(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateDietaryPreferenceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.userService.SetDietaryPreference(userID, req.DietaryPreference); err != nil {
		if err == service.ErrInvalidDietaryPreference {
			respondError(w, http.StatusBadRequest, "invalid dietary preference")
			return
		}
		s.logger().Error("failed to update dietary preference", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update dietary preference")
		return
	}

	respondJSON(w, http.StatusOK, dietaryPreferenceResponse{DietaryPreference: req.DietaryPreference})
}

// getAvailableAIModels returns selectable models and estimated pricing metadata.
func (s *Server) getAvailableAIModels(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	claudeConfigured, _ := s.userService.HasClaudeAPIKey(userID)
	geminiConfigured, _ := s.userService.HasGeminiAPIKey(userID)
	chatgptConfigured, _ := s.userService.HasOpenAIAPIKey(userID)

	mapCatalog := func(entries []llm.ModelCatalogEntry) []aiAvailableModelItem {
		items := make([]aiAvailableModelItem, 0, len(entries))
		for _, e := range entries {
			items = append(items, aiAvailableModelItem{
				ID:              e.ID,
				InputCostPer1M:  e.InputCostPer1M,
				OutputCostPer1M: e.OutputCostPer1M,
			})
		}
		return items
	}

	respondJSON(w, http.StatusOK, aiAvailableModelsResponse{
		Currency:          "USD",
		PricingUnit:       "per_1m_tokens",
		CatalogVersion:    llm.CatalogVersion,
		ClaudeConfigured:  claudeConfigured,
		ClaudeModels:      mapCatalog(llm.ClaudeModelCatalog()),
		GeminiConfigured:  geminiConfigured,
		GeminiModels:      mapCatalog(llm.GeminiModelCatalog()),
		ChatGPTConfigured: chatgptConfigured,
		ChatGPTModels:     mapCatalog(llm.ChatGPTModelCatalog()),
	})
}
