# LLM Features Documentation

This document describes the AI-powered features in xelanote, including smart tag suggestions, automatic link suggestions, spell checking, and note summarization.

## Overview

xelanote integrates with cloud LLM providers (**Claude** and **Gemini**) to provide several AI-powered features that enhance the note-taking experience:

1. **Smart Tag Suggestions** — AI analyzes note content and suggests relevant tags
2. **Link Suggestions (Auto-Linking)** — AI identifies terms that could be wikilinks to other notes
3. **Spell Check** — AI-powered spelling and grammar checking with German/English support
4. **Note Summarization** — Automatic generation of concise note summaries

All features require an API key (Claude or Gemini). Users configure their own API keys in Settings → AI.

---

## Setup

### Requirements

- **API Key**: Either a [Claude API key](https://console.anthropic.com/) or a [Gemini API key](https://makersuite.google.com/app/apikey)
- No local installation required (cloud-based)

### Configuration

1. Go to **Settings → AI** in xelanote
2. Enter your Claude API key (starts with `sk-ant-`) or Gemini API key (starts with `AIza`)
3. Save the key (it's stored encrypted on the server)

**Note**: Gemini offers a free tier with reasonable limits for personal use.

### Provider Priority

When multiple providers are configured:
1. **Claude** is preferred (better quality for complex tasks)
2. **Gemini** is used as fallback

### Per-Note AI Control

For each note, you can enable/disable AI features via the `ai_enabled` field:
- When `ai_enabled = true`: AI features are available for that note
- When `ai_enabled = false`: AI features return an error

This allows fine-grained control over which notes send content to external APIs.

---

## Feature 1: Smart Tag Suggestions

### Description

The AI analyzes your note content and suggests relevant tags based on the text. It considers:
- Existing tags in your vault (reuses common tags)
- Content themes and topics
- Suggested new tags with confidence scores

### User Interface

**Location**: Right sidebar in the note editor

**Features**:
- **Collapsible panel**: Click header to expand/collapse
- **Auto-generate on expand**: Suggestions are generated automatically when you first expand the panel
- **Confidence indicators**: Tag opacity reflects confidence (higher opacity = higher confidence)
- **New tag badges**: Suggested tags that don't exist yet are marked with "New" badge
- **One-click add**: Click any suggested tag to add it to the note

**Visual States**:
- **Collapsed**: Shows sparkles icon and title
- **Expanded**: Displays tag suggestions as colored pills
- **Loading**: Spinner indicator during generation
- **Error**: Red error message (e.g., rate limit, no provider configured)
- **No suggestions**: Gray text when no relevant tags found

### Privacy & Encrypted Notes

For end-to-end encrypted notes:
- The frontend decrypts the note content locally
- Sends plaintext content to the server **only during suggestion generation**
- Content is sent to the configured cloud provider (Claude/Gemini)
- Warning shown if encryption is locked (KEK unavailable)

### API Endpoint

**POST** `/api/notes/:id/suggest-tags`

**Request Body** (optional):
```json
{
  "plaintext_content": "..."  // Required for encrypted notes only
}
```

**Response**:
```json
{
  "suggestions": [
    {
      "name": "productivity",
      "score": 0.95,
      "is_new": false
    },
    {
      "name": "workflow",
      "score": 0.78,
      "is_new": true
    }
  ]
}
```

**Error Responses**:
- `412 Precondition Failed`: No API key configured
- `429 Too Many Requests`: Rate limit exceeded

---

## Feature 2: Link Suggestions (Auto-Linking)

### Description

The AI scans your note content and identifies terms that could be wikilinks to other notes in your vault. It:
- Finds concepts and terms that match existing note titles
- Suggests where to insert `[[wikilinks]]`
- Provides confidence scores for each suggestion
- Excludes already-linked terms

### User Interface

**Location**: Right sidebar in the note editor

**Features**:
- **Collapsible panel**: Click header to expand/collapse
- **Auto-generate on expand**: Suggestions are generated automatically on first expand
- **Confidence badges**: Color-coded confidence levels (green = high, yellow = medium, gray = low)
- **Preview**: Shows matched term → target note title
- **One-click insert**: Click "Insert" to replace the term with a wikilink

**Confidence Levels**:
- **Green** (≥80%): High confidence match
- **Yellow** (50-79%): Medium confidence match
- **Gray** (<50%): Low confidence match

### API Endpoint

**POST** `/api/notes/:id/suggest-links`

**Request Body**:
```json
{
  "plaintext_content": "...",      // Required for encrypted notes
  "note_titles": ["Title 1", "Title 2"],  // All note titles in vault
  "existing_links": ["Already Linked"]    // Already linked titles (to exclude)
}
```

**Response**:
```json
{
  "suggestions": [
    {
      "term": "version control",
      "target_title": "Git Workflow",
      "confidence": 0.92
    }
  ]
}
```

**Validation**:
- Max 500 note titles (to prevent payload overflow)
- Returns 413 if limit exceeded

---

## Feature 3: Spell Check

### Description

AI-powered spell checking with support for German and English. The spell checker:
- Detects spelling and grammar errors
- Displays wavy underlines (red for spelling, blue for grammar)
- Shows suggestions in hover tooltips
- Runs automatically after typing stops (2-second debounce)
- Handles UTF-8 to UTF-16 byte offset conversion correctly

### User Interface

**Location**: Editor toolbar (top of note editor)

**Features**:
- **Toggle button**: Click to enable/disable spell checking
- **Language selector**: Dropdown to switch between English (EN) and German (DE)
- **Visual indicators**:
  - Red wavy underline: Spelling errors
  - Blue wavy underline: Grammar errors
- **Hover tooltips**: Shows error message and suggestions
- **One-click fix**: Click suggested word to replace

**States**:
- **Disabled**: Spell check icon (gray)
- **Enabled**: Spell check icon (blue/highlighted)
- **Checking**: Loading spinner (2.5 seconds after typing)

### API Endpoint

**POST** `/api/llm/spell-check`

**Request Body**:
```json
{
  "text": "This is a text with mistkaes.",
  "language": "en"  // "en" or "de"
}
```

**Response**:
```json
{
  "issues": [
    {
      "byte_offset": 22,
      "byte_length": 8,
      "type": "spelling",
      "message": "Possible spelling mistake",
      "suggestions": ["mistakes", "mistake", "mistaken"]
    }
  ]
}
```

**Error Responses**:
- `412 Precondition Failed`: No API key configured - add in Settings → AI
- `413 Request Entity Too Large`: Text exceeds 10,000 bytes

---

## Feature 4: Note Summarization

### Description

Automatic generation of concise note summaries using the configured LLM provider. Summaries are:
- Generated on-demand (click the summary panel)
- Cached based on content hash (regenerated only when content changes)
- Available for both encrypted and unencrypted notes

### User Interface

**Location**: Right sidebar in the note editor

**Features**:
- **Summary panel**: Shows generated summary
- **Regenerate button**: Force regeneration of summary
- **Loading indicator**: Spinner during generation

### API Endpoint

**POST** `/api/notes/:id/summarize`

**Request Body** (optional):
```json
{
  "plaintext_content": "..."  // Required for encrypted notes only
}
```

**Response**:
```json
{
  "summary": "This note discusses..."
}
```

---

## Rate Limiting

All LLM features share a **single rate limiter** to prevent abuse:

- **Limit**: 10 requests per minute per user
- **Applies to**:
  - `/api/notes/:id/summarize`
  - `/api/notes/:id/suggest-tags`
  - `/api/notes/:id/suggest-links`
  - `/api/llm/spell-check`
- **Response**: `429 Too Many Requests` when limit exceeded

---

## Security & Privacy

### Data Flow

1. **User types** in the editor
2. **Frontend** decrypts content (if encrypted) locally in the browser
3. **Frontend** sends plaintext to backend API (HTTPS only)
4. **Backend** forwards to configured cloud provider (Claude or Gemini)
5. **Provider** processes text and returns result
6. **Backend** sends result to frontend
7. **Frontend** displays suggestions/errors

### Privacy Notes

- **Plaintext content** is sent to external cloud APIs during LLM feature usage
- Content is **never persisted** in plaintext on the xelanote server
- For encrypted notes, decryption happens **in the browser** (zero-knowledge locally)
- **API keys** are stored encrypted on the server
- All communication over HTTPS

### Encrypted Notes Workflow

1. User unlocks encryption (enters passphrase)
2. Frontend derives KEK and decrypts note locally
3. User triggers LLM feature (e.g., suggest tags)
4. Frontend sends `plaintext_content` field in API request
5. Backend sends content to cloud provider
6. Backend returns result to frontend
7. Frontend optionally re-encrypts result before storing

---

## Troubleshooting

### Common Issues

**Issue**: "AI provider required - add API key in settings"
- **Cause**: No Claude or Gemini API key configured
- **Solution**: Go to Settings → AI and add an API key

**Issue**: "AI features are disabled for this note"
- **Cause**: Note has `ai_enabled = false`
- **Solution**: Enable AI for the note in note settings

**Issue**: "Rate limit exceeded"
- **Cause**: Too many LLM requests in 1 minute (>10)
- **Solution**: Wait 1 minute before trying again

**Issue**: "Text too large" (Spell check)
- **Cause**: Note content exceeds 10,000 bytes
- **Solution**: Split note into smaller notes

**Issue**: Suggestions not relevant
- **Cause**: Model quality or prompt engineering
- **Solution**: Try a different provider (Claude vs Gemini)

---

## API Reference Summary

| Endpoint | Method | Rate Limit | Purpose |
|----------|--------|------------|---------|
| `/api/notes/:id/summarize` | POST | 10/min | Generate note summary |
| `/api/notes/:id/suggest-tags` | POST | 10/min | Suggest relevant tags |
| `/api/notes/:id/suggest-links` | POST | 10/min | Suggest wikilinks |
| `/api/llm/spell-check` | POST | 10/min | Check spelling/grammar |

All endpoints require authentication (access token in HttpOnly cookie).

---

## Supported Providers

| Provider | Model | Quality | Cost |
|----------|-------|---------|------|
| **Claude** | claude-3-haiku | High | Paid (API credits) |
| **Gemini** | gemini-2.5-flash | Good | Free tier available |

**Recommendation**: Start with Gemini's free tier for evaluation, upgrade to Claude for better quality.

---

## Related Documentation

- [API Reference](./api.md) — Full REST API documentation
- [E2E Encryption](./e2e-encryption.md) — How encryption works
- [Editor Features](./editor-features.md) — Other editor capabilities (wikilinks, tasks, etc.)

---

## Acknowledgments

LLM features powered by:
- [Anthropic Claude](https://www.anthropic.com/) — Advanced AI assistant
- [Google Gemini](https://ai.google.dev/) — Multimodal AI model
- [CodeMirror 6](https://codemirror.net/) — Spell check decorations and tooltips
