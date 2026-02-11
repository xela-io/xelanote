package api

// listNoteTitles returns a lightweight list of note titles for link suggestions.
// Only returns unencrypted titles to avoid sending encrypted data to LLM.
// Limited to MaxNoteTitlesForSuggestions to prevent memory exhaustion.
const MaxNoteTitlesForSuggestions = 1000
