package db

import "time"

// AIModelPreferences stores optional model overrides per provider for one user.
// Empty values mean "use server default".
type AIModelPreferences struct {
	ClaudeModel  string `json:"claude_model"`
	GeminiModel  string `json:"gemini_model"`
	ChatGPTModel string `json:"chatgpt_model"`
}

// GetAIModelPreferences returns model preferences for a user.
// If no row exists, empty values are returned (provider defaults apply).
func (db *DB) GetAIModelPreferences(userID int) (*AIModelPreferences, error) {
	prefs, err := db.GetUserPreferences(userID)
	if err == ErrNotFound {
		return &AIModelPreferences{}, nil
	}
	if err != nil {
		return nil, err
	}

	return &AIModelPreferences{
		ClaudeModel:  prefs.ClaudeModel,
		GeminiModel:  prefs.GeminiModel,
		ChatGPTModel: prefs.OpenAIModel,
	}, nil
}

// SetAIModelPreferences sets all model preferences for a user.
// Empty values are allowed and interpreted as "use default model".
func (db *DB) SetAIModelPreferences(userID int, models *AIModelPreferences) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET claude_model = ?, gemini_model = ?, openai_model = ?, updated_at = ?
		WHERE user_id = ?
	`, models.ClaudeModel, models.GeminiModel, models.ChatGPTModel, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		_, err = db.Exec(`
			INSERT INTO user_preferences (
				user_id, theme, editor_mode, claude_model, gemini_model, openai_model, created_at, updated_at
			)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?, ?)
		`, userID, models.ClaudeModel, models.GeminiModel, models.ChatGPTModel, now, now)
		return err
	}

	return nil
}
