package api

import "github.com/xela-io/xelanote/internal/service"

const errEncryptedNoteAIProcessingDisabled = "server-side AI processing is disabled for encrypted notes"
const errAIPlaintextContentDeprecated = "plaintext_content is no longer accepted for this endpoint"

// --- Summary Endpoints ---

// SummarizeNoteRequest represents the request body for summarizing a note.
type SummarizeNoteRequest struct {
	// PlaintextContent is optional for legacy clients. Encrypted-note processing remains blocked.
	PlaintextContent string `json:"plaintext_content,omitempty"`
	// PlaintextContentHash is the SHA256 hash of plaintext (for E2E notes, computed by frontend)
	PlaintextContentHash string `json:"plaintext_content_hash,omitempty"`
	// EncryptedSummary is the encrypted summary (for E2E notes, encrypted by frontend)
	EncryptedSummary string `json:"encrypted_summary,omitempty"`
}

// SummarizeNoteResponse represents the response from summarizing a note.
type SummarizeNoteResponse struct {
	Summary string `json:"summary"`
}

// ============================================================================
// LLM Feature: Tag Suggestions
// ============================================================================

// SuggestTagsRequest represents the request body for tag suggestions.
type SuggestTagsRequest struct {
	// Deprecated: plaintext_content is intentionally rejected.
	PlaintextContent string `json:"plaintext_content,omitempty"`
}

// SuggestTagsResponse represents the response from tag suggestions.
type SuggestTagsResponse struct {
	Suggestions []service.TagSuggestion `json:"suggestions"`
}

// ============================================================================
// LLM Feature: Link Suggestions
// ============================================================================

// SuggestLinksRequest represents the request body for link suggestions.
type SuggestLinksRequest struct {
	// Deprecated: plaintext_content is intentionally rejected.
	PlaintextContent string   `json:"plaintext_content,omitempty"`
	NoteTitles       []string `json:"note_titles"`
	ExistingLinks    []string `json:"existing_links"`
}

// SuggestLinksResponse represents the response from link suggestions.
type SuggestLinksResponse struct {
	Suggestions []service.LinkSuggestion `json:"suggestions"`
}

// ============================================================================
// AI-Enabled (Claude API Opt-In) Endpoints
// ============================================================================

// UpdateAIEnabledRequest represents the request body for toggling ai_enabled.
type UpdateAIEnabledRequest struct {
	AIEnabled bool `json:"ai_enabled"`
}

// ============================================================================
// LLM Feature: Format Markdown
// ============================================================================

// FormatMarkdownRequest represents the request body for formatting markdown.
type FormatMarkdownRequest struct {
	// Deprecated: plaintext_content is intentionally rejected.
	PlaintextContent string `json:"plaintext_content,omitempty"`
	SelectionOnly    string `json:"selection_only,omitempty"` // When only part is formatted
}

// FormatMarkdownResponse represents the response from formatting markdown.
type FormatMarkdownResponse struct {
	FormattedContent string `json:"formatted_content"`
}

// ============================================================================
// LLM Feature: AI Transform
// ============================================================================

// AITransformRequest represents the request body for AI text transformation.
type AITransformRequest struct {
	Action  string `json:"action"`            // format, summarize, expand, translate_de, translate_en, formal, informal, custom
	Content string `json:"content,omitempty"` // Plain text content
	// Deprecated: plaintext_content is intentionally rejected.
	PlaintextContent string `json:"plaintext_content,omitempty"`
	CustomPrompt     string `json:"custom_prompt,omitempty"` // Only for action="custom"
}

// AITransformResponse represents the response from AI text transformation.
type AITransformResponse struct {
	TransformedContent string `json:"transformed_content"`
}
