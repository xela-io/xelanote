# Plan: Claude API Integration für xelanote

## Zusammenfassung

Integration der Claude API als zusätzlicher LLM-Provider neben Ollama. Implementierung eines Freigabe-Systems (per Notiz und Ordner) sowie BYOK (Bring Your Own Key) für Claude API Keys.

## Design-Entscheidungen (FINAL)

| Entscheidung | Ergebnis |
|--------------|----------|
| **ai_enabled Bedeutung** | "Claude erlaubt" - Ollama bleibt IMMER verfügbar |
| **Spell-Check** | Bleibt IMMER lokal (Ollama), kein Claude-Path |
| **Root-Folder Default** | Root-Notizen sind immer `ai_enabled=false` |
| **API-Key Verschlüsselung** | `XELANOTE_API_KEY_SECRET` (wie JWT_SECRET) |
| **Cross-Referenzierung** | Nur über freigegebene Notizen (`ai_enabled=true`) |

## Architektur

```
┌─────────────────────────────────────────────────────────┐
│                      Frontend                            │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐                    │
│  │ Sidebar │ │ Editor  │ │Settings │                    │
│  │ 🤖 Icon │ │ Toggle  │ │ API-Key │                    │
│  └────┬────┘ └────┬────┘ └────┬────┘                    │
└───────┼──────────┼──────────┼───────────────────────────┘
        │          │          │
        ▼          ▼          ▼
┌─────────────────────────────────────────────────────────┐
│                Provider Router                           │
│                                                         │
│  Summarize/Tags/Links:                                  │
│    if (!note.ai_enabled) → Ollama (always)              │
│    elif (user.has_claude_key) → Claude API              │
│    else → Ollama (fallback)                             │
│                                                         │
│  Spell-Check:                                           │
│    → Ollama (always, no Claude path)                    │
└─────────────────────────────────────────────────────────┘
        │                           │
        ▼                           ▼
┌───────────────┐           ┌───────────────┐
│ Ollama (Lokal)│           │ Claude API ☁️ │
│ $OLLAMA_MODEL │           │ $CLAUDE_MODEL │
└───────────────┘           └───────────────┘
```

---

## Phase 1: Foundation - Freigabe-System

### Task 1.1: DB Migration - AI-Enabled Flags + Folder-Default
**Dateien:** `backend/internal/db/migrations/030_ai_enabled_flags.sql`

**SQL:**
```sql
-- Notes: ai_enabled flag
ALTER TABLE notes ADD COLUMN ai_enabled INTEGER DEFAULT 0;
CREATE INDEX idx_notes_ai_enabled ON notes(user_id, ai_enabled) WHERE is_deleted = 0;

-- Folders: default for new notes in this folder
ALTER TABLE folders ADD COLUMN ai_enabled_default INTEGER DEFAULT 0;
```

**Akzeptanzkriterien:**
- [ ] Migration erstellt und läuft ohne Fehler
- [ ] Default ist 0 (safe default)
- [ ] Bestehende Notizen haben `ai_enabled=0`

---

### Task 1.2: DB-Funktionen für Freigabe + Folder-Vererbung
**Dateien:** `backend/internal/db/notes.go`, `backend/internal/db/folders.go`

**Akzeptanzkriterien:**
- [ ] `UpdateNoteAIEnabled(userID int, noteID string, enabled bool) error`
- [ ] `GetNoteAIEnabled(userID int, noteID string) (bool, error)`
- [ ] `UpdateFolderAIEnabledDefault(userID int, folderID int, enabled bool) error`
- [ ] `GetFolderAIEnabledDefault(userID int, folderID int) (bool, error)`
- [ ] `Note` Struct: `AIEnabled bool` hinzufügen
- [ ] `Folder` Struct: `AIEnabledDefault bool` hinzufügen
- [ ] **NEU:** `CreateNote()` setzt `ai_enabled` basierend auf Folder-Default
- [ ] **NEU:** Root-Regel: `if folder_path == "/" → ai_enabled=false` (unabhängig von Folder-Default)
- [ ] **NEU:** `GetNoteTitlesAIEnabled(userID int) ([]string, error)` - für Link-Suggestions mit Claude
- [ ] Unit-Tests für alle Funktionen

---

### Task 1.3: Cache-Invalidierung für ai_enabled
**Dateien:** `backend/internal/service/notes.go`

**Problem:** NoteService cached Notes/Folders. Änderungen an `ai_enabled` müssen Cache invalidieren.

**Akzeptanzkriterien:**
- [ ] `UpdateNoteAIEnabled()` invalidiert Note-Cache
- [ ] `UpdateFolderAIEnabledDefault()` invalidiert Folder-Cache
- [ ] Unit-Tests für Cache-Konsistenz

---

### Task 1.4: API-Endpoints für Freigabe
**Dateien:** `backend/internal/api/llm.go`, `backend/internal/api/folders.go`

**Akzeptanzkriterien:**
- [ ] `PUT /api/notes/:id/ai-enabled` (Body: `{"enabled": bool}`)
- [ ] `PUT /api/folders/:id/ai-enabled-default` (Body: `{"enabled": bool}`)
- [ ] Authentifizierung + Ownership-Validierung
- [ ] Integration-Tests

---

### Task 1.5: Frontend API-Client für Freigabe
**Dateien:** `frontend/src/lib/api.ts`, `frontend/src/lib/types.ts`

**Akzeptanzkriterien:**
- [ ] `setNoteAIEnabled(noteId: string, enabled: boolean): Promise<Note>`
- [ ] `setFolderAIEnabledDefault(folderId: number, enabled: boolean): Promise<Folder>`
- [ ] TypeScript-Typen: `Note.ai_enabled`, `Folder.ai_enabled_default`

---

### Task 1.6: Freigabe-Icon in Sidebar
**Dateien:** `frontend/src/lib/components/Sidebar.svelte`, `UnifiedTree.svelte`

**Akzeptanzkriterien:**
- [ ] Icon (Sparkles) rechts neben Notiz-Titel wenn `ai_enabled === true`
- [ ] Tooltip: "Cloud-KI aktiviert"
- [ ] Dezente Färbung (nicht aufdringlich)
- [ ] Responsive auf Mobile

---

### Task 1.7: Freigabe-Toggle im Editor
**Dateien:** `frontend/src/lib/components/Editor.svelte`

**Akzeptanzkriterien:**
- [ ] Toggle-Button in Editor-Toolbar
- [ ] Optimistic Update mit Rollback bei Fehler
- [ ] **Warnung bei E2E-Notizen:** Modal mit Bestätigung
- [ ] Button disabled wenn Note nicht gespeichert (noteId fehlt)

---

### Task 1.8: Tests für Phase 1
**Akzeptanzkriterien:**
- [ ] Unit-Tests für DB-Funktionen (mind. 10 Tests)
- [ ] Integration-Tests für API-Endpoints (mind. 6 Tests)
- [ ] Test: Root-Notizen immer `ai_enabled=false`
- [ ] Test: Folder-Vererbung funktioniert
- [ ] Alle Tests bestehen (`make test`)

---

## Phase 2: BYOK & Provider Abstraction

### Task 2.1: Environment-Variable für API-Key-Secret
**Dateien:** `backend/cmd/server/main.go`, `docs/deployment.md`, `CLAUDE.md`

**Akzeptanzkriterien:**
- [ ] `XELANOTE_API_KEY_SECRET` Env-Variable dokumentiert
- [ ] Validierung: Mindestens 32 Zeichen (wie JWT_SECRET)
- [ ] Server startet ohne Secret (Feature disabled), mit Warning-Log
- [ ] Dokumentation in CLAUDE.md und deployment.md

---

### Task 2.2: DB-Schema für API-Key
**Dateien:** `backend/internal/db/migrations/031_api_key_storage.sql`

**SQL:**
```sql
CREATE TABLE user_api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    provider TEXT NOT NULL,           -- 'claude'
    encrypted_key TEXT NOT NULL,      -- AES-256-GCM encrypted
    key_prefix TEXT NOT NULL,         -- First 12 chars for display: "sk-ant-api3..."
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_user_api_keys ON user_api_keys(user_id, provider);
```

**Akzeptanzkriterien:**
- [ ] Migration erstellt und läuft
- [ ] Unique Constraint auf (user_id, provider)
- [ ] CASCADE DELETE bei User-Löschung

---

### Task 2.3: API-Key Encryption Helper
**Dateien:** `backend/internal/crypto/apikey.go` (NEU)

**Akzeptanzkriterien:**
- [ ] `EncryptAPIKey(plainKey string, secret []byte) (string, error)` - AES-256-GCM
- [ ] `DecryptAPIKey(encryptedKey string, secret []byte) (string, error)`
- [ ] `GetKeyPrefix(key string) string` - Returns first 12 chars + "..." (z.B. "sk-ant-api3...")
- [ ] Secret aus `XELANOTE_API_KEY_SECRET`
- [ ] Unit-Tests (mind. 6 Tests)

---

### Task 2.4: DB-Funktionen für API-Key
**Dateien:** `backend/internal/db/api_keys.go` (NEU)

**Akzeptanzkriterien:**
- [ ] `StoreAPIKey(userID int, provider, encryptedKey, keyPrefix string) error`
- [ ] `GetAPIKey(userID int, provider string) (encryptedKey string, err error)`
- [ ] `DeleteAPIKey(userID int, provider string) error`
- [ ] `HasAPIKey(userID int, provider string) (bool, error)`
- [ ] `GetAPIKeyInfo(userID int, provider string) (prefix string, createdAt time.Time, err error)`
- [ ] UPSERT für Duplikate
- [ ] Unit-Tests

---

### Task 2.5: Provider Interface MIT Streaming
**Dateien:** `backend/internal/llm/provider.go` (NEU)

**Interface:**
```go
type Provider interface {
    // Non-streaming
    Generate(ctx context.Context, prompt string, maxTokens int) (string, error)

    // Streaming (for summaries)
    GenerateStream(ctx context.Context, prompt string, maxTokens int, onToken func(string)) (string, error)

    // Meta
    IsAvailable(ctx context.Context) bool
    Model() string
    SupportsStreaming() bool
}
```

**Akzeptanzkriterien:**
- [ ] Interface definiert mit Streaming-Support
- [ ] `OllamaClient` implementiert Interface (Refactoring)
- [ ] Bestehende `Summarize()` / `SummarizeStream()` nutzen `Generate()` / `GenerateStream()`
- [ ] Bestehende Tests laufen weiterhin (keine Regression)

---

### Task 2.6: Claude API Client
**Dateien:** `backend/internal/llm/claude.go` (NEU)

**Akzeptanzkriterien:**
- [ ] `ClaudeClient` implementiert `Provider` Interface
- [ ] HTTP-Client für `https://api.anthropic.com/v1/messages`
- [ ] Streaming via SSE (`stream: true`)
- [ ] Header: `x-api-key`, `anthropic-version: 2023-06-01`
- [ ] Model aus `CLAUDE_MODEL` Env (Default: `claude-sonnet-4-5-20250929`)
- [ ] Error-Handling: 429 (Rate Limit), 401 (Invalid Key), 5xx
- [ ] Timeout: 2 Minuten
- [ ] `SupportsStreaming() = true`
- [ ] Unit-Tests mit Mock-Server (mind. 8 Tests)

---

### Task 2.7: Provider-Router
**Dateien:** `backend/internal/service/provider_router.go` (NEU)

**Routing-Logik:**
```go
func (r *ProviderRouter) SelectProvider(ctx context.Context, userID int, noteID string) (Provider, error) {
    // 1. Get note's ai_enabled status
    aiEnabled, err := r.db.GetNoteAIEnabled(userID, noteID)
    if err != nil || !aiEnabled {
        return r.ollama, nil  // Always available
    }

    // 2. Check if user has Claude API key
    hasKey, _ := r.db.HasAPIKey(userID, "claude")
    if !hasKey {
        return r.ollama, nil  // Fallback
    }

    // 3. Get and decrypt API key
    encryptedKey, _ := r.db.GetAPIKey(userID, "claude")
    apiKey, _ := crypto.DecryptAPIKey(encryptedKey, r.secret)

    // 4. Return Claude client with user's key
    return llm.NewClaudeClient(apiKey, r.claudeModel), nil
}
```

**Akzeptanzkriterien:**
- [ ] `SelectProvider(ctx, userID, noteID) (Provider, error)` - Haupt-Funktion
- [ ] `SelectProviderForUser(ctx, userID) (Provider, error)` - für Settings-Test-Call ("API-Key testen")
- [ ] Cache für API-Keys (5 Min TTL) um DB-Lookups zu reduzieren
- [ ] Logging: Welcher Provider gewählt wurde
- [ ] Fallback zu Ollama bei Claude-Fehler (mit Warning-Log)
- [ ] Unit-Tests für alle Routing-Szenarien (mind. 12 Tests)

**Hinweis:** `SelectProviderForUser` wird nur für Settings-UI gebraucht (Test-Call ob API-Key funktioniert). Alle Feature-Calls nutzen `SelectProvider` mit noteID.

---

### Task 2.8: API-Endpoints für API-Key Management
**Dateien:** `backend/internal/api/users.go`, `backend/internal/api/server.go`

**Rate-Limiting:**
```go
// In Server struct (api/server.go)
apiKeyLimiter *rate.Limiter  // 5 req/min pro User

// Oder: Reuse llmLimiter wenn vorhanden
```

**Akzeptanzkriterien:**
- [ ] `PUT /api/users/me/api-keys/claude` - Body: `{"api_key": "sk-ant-..."}`
- [ ] `DELETE /api/users/me/api-keys/claude`
- [ ] `GET /api/users/me/api-keys` - Response: `{"claude": {"has_key": true, "prefix": "sk-ant-api3...", "created_at": "..."}}`
- [ ] **Security:** API-Key wird NIEMALS zurückgegeben
- [ ] Validierung: Key startet mit `sk-ant-`
- [ ] **Rate-Limiting:** Neuer `apiKeyLimiter` in Server struct (5 req/min)
- [ ] Integration-Tests

---

### Task 2.9: Settings-UI für API-Key
**Dateien:** `frontend/src/routes/settings/+page.svelte`, `frontend/src/lib/api.ts`

**Akzeptanzkriterien:**
- [ ] Neue Sektion "KI-Features" in Settings
- [ ] Input-Field für API-Key (type="password", autocomplete="off")
- [ ] Buttons: "Speichern", "Löschen" (nur wenn Key vorhanden)
- [ ] Status: "Claude API: Aktiviert ✓ (sk-ant-a...)" oder "Nicht konfiguriert"
- [ ] Warnung: "Du bist für API-Kosten verantwortlich"
- [ ] Link: "API-Key erstellen" → https://console.anthropic.com/settings/keys
- [ ] Toast-Benachrichtigungen bei Erfolg/Fehler

---

### Task 2.10: Tests für Phase 2
**Akzeptanzkriterien:**
- [ ] Unit-Tests für Crypto (mind. 6 Tests)
- [ ] Unit-Tests für Claude Client (mind. 8 Tests)
- [ ] Unit-Tests für Provider-Router (mind. 12 Tests)
- [ ] Integration-Tests für API-Key-Endpoints (mind. 6 Tests)
- [ ] Security-Test: API-Key nicht über API extrahierbar
- [ ] Alle Tests bestehen

---

## Phase 3: Features auf Provider-Router migrieren

### Task 3.1: Signatur-Änderung: noteID für SuggestTags/SuggestLinks
**Dateien:** `backend/internal/service/summarize.go`, `backend/internal/api/notes.go`, `frontend/src/lib/api.ts`

**Problem:** `SuggestTags(ctx, userID, content)` hat keine noteID → Router kann nicht entscheiden.

**Änderungen:**
```go
// Alt
SuggestTags(ctx context.Context, userID int, content string) ([]TagSuggestion, error)
SuggestLinks(ctx context.Context, userID int, content string, noteTitles []string) ([]LinkSuggestion, error)

// Neu
SuggestTags(ctx context.Context, userID int, noteID string, content string) ([]TagSuggestion, error)
SuggestLinks(ctx context.Context, userID int, noteID string, content string) ([]LinkSuggestion, error)
```

**Akzeptanzkriterien:**
- [ ] Service-Signaturen geändert
- [ ] API-Handler nutzen noteID aus URL (bereits vorhanden: `/api/notes/:id/suggest-tags`)
- [ ] Frontend-API-Calls unverändert (noteID ist bereits im URL)
- [ ] Bestehende Tests angepasst

---

### Task 3.2: Prompts-Refactoring + Semantische Tag-Erkennung
**Dateien:** `backend/internal/llm/prompts.go`

**Verbesserung für Tag-Suggestions:**
```
NEW RULES for semantic matching:
- "meetings" → use existing "meeting" (plural/singular)
- "Budgetplanung" → use existing "budget" (compound words)
- "Teamarbeit" → use existing "team" (compound words)
- "projects" → use existing "projekt" (language variants)
- Prefer existing tags even for partial semantic matches
- Only create new tag if NO existing tag covers the concept
```

**Akzeptanzkriterien:**
- [ ] Prompts funktionieren mit Ollama (keine Regression)
- [ ] Prompts funktionieren mit Claude
- [ ] Semantische Ähnlichkeit explizit im Prompt
- [ ] Unit-Tests für Prompt-Generation

---

### Task 3.3: Summarize mit Provider-Router + Streaming
**Dateien:** `backend/internal/service/summarize.go`

**Streaming-Fallback-Logik:**
```go
// 1. Versuche Claude-Stream zu starten
stream, err := claudeProvider.GenerateStream(ctx, prompt, maxTokens, onToken)

// 2. Fallback-Regeln:
//    - Fehler VOR erstem Token → Fallback zu Ollama, neuer Stream
//    - Fehler NACH erstem Token → Fehler zurückgeben, KEIN Fallback
//      (Client hat bereits Partial-Response, Doppel-Stream wäre inkonsistent)
```

**Akzeptanzkriterien:**
- [ ] `SummarizeNote()` ruft `router.SelectProvider()` auf
- [ ] `SummarizeNoteStream()` nutzt `provider.GenerateStream()`
- [ ] Streaming funktioniert mit beiden Providern
- [ ] **Fallback-Regel:** Vor erstem Token → Ollama-Fallback erlaubt
- [ ] **Fallback-Regel:** Nach erstem Token → Fehler, kein Fallback
- [ ] Bestehende Tests laufen
- [ ] Neue Tests für Claude-Path + Fallback-Szenarien

---

### Task 3.4: Tag-Suggestions mit Provider-Router
**Dateien:** `backend/internal/service/summarize.go`

**Akzeptanzkriterien:**
- [ ] `SuggestTags()` ruft `router.SelectProvider(ctx, userID, noteID)` auf
- [ ] Tests für beide Provider

---

### Task 3.5: Link-Suggestions mit Provider-Router + ai_enabled Filter
**Dateien:** `backend/internal/service/summarize.go`, `backend/internal/api/notes.go`

**Problem:** `listNoteTitles()` liefert ALLE Titel. Bei Claude sollten nur `ai_enabled=true` Notizen im Kontext sein.

**Lösung:** Neue DB-Funktion (aus Task 1.2) auf DB-Ebene filtern, nicht nachträglich:
```go
if provider == "claude" {
    titles, _ = db.GetNoteTitlesAIEnabled(userID)  // Nur ai_enabled=true
} else {
    titles, _ = db.GetNoteTitles(userID)  // Alle (wie bisher)
}
```

**Akzeptanzkriterien:**
- [ ] `SuggestLinks()` ruft `router.SelectProvider()` auf
- [ ] **NEU:** Wenn Provider=Claude: `db.GetNoteTitlesAIEnabled()` (DB-Level Filter)
- [ ] Wenn Provider=Ollama: `db.GetNoteTitles()` (alle, wie bisher)
- [ ] Keine nachträgliche Filterung (Performance)
- [ ] Tests für beide Szenarien

---

### Task 3.6: Spell-Check bleibt Ollama-only
**Dateien:** `backend/internal/service/summarize.go`, `backend/internal/api/llm.go`

**Entscheidung:** Spell-Check nutzt IMMER Ollama (lokal), keinen Provider-Router.

**Akzeptanzkriterien:**
- [ ] Keine Änderung an Spell-Check (bleibt wie ist)
- [ ] Dokumentation: "Spell-Check ist immer lokal"

---

### Task 3.7: Provider-Indicator im Frontend
**Dateien:**
- `backend/internal/api/notes.go` (Response-Structs)
- `frontend/src/lib/api.ts` (TypeScript Types)
- `frontend/src/lib/components/SummaryPanel.svelte`

**API-Response-Erweiterung:**
```go
// backend/internal/api/notes.go
type SummarizeNoteResponse struct {
    Summary  string `json:"summary"`
    Provider string `json:"provider"`  // NEU: "claude" | "ollama"
}

type SuggestTagsResponse struct {
    Suggestions []TagSuggestion `json:"suggestions"`
    Provider    string          `json:"provider"`  // NEU
}

type SuggestLinksResponse struct {
    Suggestions []LinkSuggestion `json:"suggestions"`
    Provider    string           `json:"provider"`  // NEU
}
```

**Akzeptanzkriterien:**
- [ ] Backend: Response-Structs um `provider` Feld erweitert
- [ ] Frontend: TypeScript-Typen aktualisiert
- [ ] UI zeigt "Generiert mit Claude ☁️" oder "Generiert mit Ollama (lokal)"
- [ ] Dezenter Indicator (nicht aufdringlich)

---

### Task 3.8: Tests für Phase 3
**Akzeptanzkriterien:**
- [ ] Integration-Tests: Summarize mit beiden Providern
- [ ] Test: Tags mit Claude + Ollama
- [ ] Test: Links mit ai_enabled Filter
- [ ] Test: Streaming funktioniert mit Claude
- [ ] Alle Tests bestehen

---

## Phase 4: Neue Features

### Task 4.1: Redundanz-Erkennung - Backend
**Dateien:** `backend/internal/service/summarize.go`, `backend/internal/llm/prompts.go`, `backend/internal/api/notes.go`

**Akzeptanzkriterien:**
- [ ] `DetectRedundancy(ctx, userID, noteID string) ([]RedundancyMatch, error)`
- [ ] Lädt nur `ai_enabled=true` Notizen des Users
- [ ] Max 20 Notizen pro Request (Token-Limit)
- [ ] Response: `{TargetNoteID, TargetTitle, Similarity float64, Snippet string}`
- [ ] Endpoint: `POST /api/notes/:id/detect-redundancy`
- [ ] Nutzt Provider-Router (Claude wenn freigegeben)
- [ ] Unit-Tests

---

### Task 4.2: Redundanz-Erkennung - Frontend
**Dateien:** `frontend/src/lib/components/RedundancyPanel.svelte` (NEU)

**Akzeptanzkriterien:**
- [ ] Panel in rechter Sidebar (wie SummaryPanel)
- [ ] Button "Redundanzen prüfen" (nur aktiv wenn `ai_enabled`)
- [ ] Liste mit Links zu ähnlichen Notizen
- [ ] Similarity-Score: Farb-Coding (grün >80%, gelb 50-80%, grau <50%)
- [ ] Snippet-Preview
- [ ] Warnung wenn Note nicht freigegeben

---

### Task 4.3: Schreibstil-Verbesserung - Backend
**Dateien:** `backend/internal/service/summarize.go`, `backend/internal/llm/prompts.go`

**Akzeptanzkriterien:**
- [ ] `ImproveWriting(ctx, userID int, noteID string, content string) (string, error)`
- [ ] Prompt: Grammatik, Klarheit, Ton verbessern, Bedeutung erhalten
- [ ] Max 5000 Zeichen pro Request
- [ ] Endpoint: `POST /api/notes/:id/improve-writing`
- [ ] Nutzt Provider-Router
- [ ] Unit-Tests

---

### Task 4.4: Schreibstil-Verbesserung - Frontend
**Dateien:** `frontend/src/lib/components/Editor.svelte`

**Akzeptanzkriterien:**
- [ ] Button "Stil verbessern" (Wand-Icon) in Toolbar
- [ ] Button nur aktiv wenn `ai_enabled`
- [ ] Modal: Original vs. Verbessert (side-by-side oder tabs)
- [ ] Buttons: "Übernehmen" / "Verwerfen"
- [ ] Loading-State während Verbesserung

---

### Task 4.5: Markdown-Formatierung - Backend
**Dateien:** `backend/internal/service/summarize.go`, `backend/internal/llm/prompts.go`

**Akzeptanzkriterien:**
- [ ] `FormatMarkdown(ctx, userID int, noteID string, content string) (string, error)`
- [ ] Prompt: Überschriften, Listen, Code-Blöcke hinzufügen
- [ ] Bewahrt bestehende Markdown-Syntax
- [ ] Endpoint: `POST /api/notes/:id/format-markdown`
- [ ] Unit-Tests

---

### Task 4.6: Markdown-Formatierung - Frontend
**Dateien:** `frontend/src/lib/components/Editor.svelte`

**Akzeptanzkriterien:**
- [ ] Button "Formatieren" (AlignLeft-Icon) in Toolbar
- [ ] Button nur aktiv wenn `ai_enabled`
- [ ] Modal mit Preview (Markdown gerendert)
- [ ] Buttons: "Übernehmen" / "Verwerfen"

---

### Task 4.7: Tests für Phase 4
**Akzeptanzkriterien:**
- [ ] Unit-Tests für alle 3 neuen Features
- [ ] Integration-Tests für Endpoints
- [ ] E2E-Test: Feature-Workflow (optional)
- [ ] Alle Tests bestehen

---

## Phase 5: Polish & Bulk-Operations

### Task 5.1: Bulk-Freigabe für Notizen
**Dateien:** `frontend/src/lib/components/Sidebar.svelte`, `backend/internal/api/notes.go`

**Akzeptanzkriterien:**
- [ ] Multi-Select in Sidebar (Checkboxen)
- [ ] Context-Menu: "Cloud-KI aktivieren/deaktivieren"
- [ ] `PUT /api/notes/bulk/ai-enabled` (Body: `{"note_ids": [...], "enabled": bool}`)
- [ ] Progress-Indicator bei >10 Notizen

---

### Task 5.2: Ordner-Freigabe UI
**Dateien:** `frontend/src/lib/components/FolderDialog.svelte`

**Akzeptanzkriterien:**
- [ ] Checkbox "Cloud-KI für neue Notizen aktivieren" im Folder-Dialog
- [ ] Tooltip erklärt Vererbung
- [ ] Bestehende Notizen bleiben unverändert

---

### Task 5.3: Usage-Tracking (optional)
**Akzeptanzkriterien:**
- [ ] Tabelle `llm_usage_stats` (user_id, provider, feature, count, last_used)
- [ ] Settings zeigt Statistiken
- [ ] Keine Inhalte loggen (Privacy)

---

### Task 5.4: Dokumentation
**Dateien:** `docs/claude-api-integration.md` (NEU)

**Akzeptanzkriterien:**
- [ ] User-Guide: API-Key Einrichtung
- [ ] Erklärung: Freigabe-System (per Notiz/Ordner)
- [ ] Security-Hinweise: API-Key-Verschlüsselung
- [ ] Privacy-Hinweise: E2E + Freigabe = Plaintext zu Anthropic
- [ ] Troubleshooting: Invalid Key, Rate Limits

---

### Task 5.5: Final Testing & Security-Audit
**Akzeptanzkriterien:**
- [ ] E2E-Tests auf Staging
- [ ] Security-Audit: API-Keys nicht extrahierbar
- [ ] Performance-Check: Provider-Auswahl < 50ms
- [ ] Code-Review
- [ ] Alle Tests bestehen

---

## Zeitschätzung

| Phase | Fokus | Tasks | Geschätzt |
|-------|-------|-------|-----------|
| 1 | Freigabe-System + Folder-Vererbung | 8 | ~6h |
| 2 | BYOK & Provider mit Streaming | 10 | ~12h |
| 3 | Feature-Migration + Filter | 8 | ~6h |
| 4 | Neue Features | 7 | ~8h |
| 5 | Polish & Docs | 5 | ~5h |
| **Gesamt** | | **38** | **~37h** |

---

## Environment-Variablen (NEU)

| Variable | Beschreibung | Required |
|----------|--------------|----------|
| `XELANOTE_API_KEY_SECRET` | Secret für API-Key-Verschlüsselung (min 32 chars) | Nein (Feature disabled ohne) |
| `CLAUDE_MODEL` | Claude-Modell (Default: `claude-sonnet-4-5-20250929`) | Nein |
| `OLLAMA_MODEL` | Ollama-Modell (Default: `qwen2.5:3b`) | Nein |
| `OLLAMA_URL` | Ollama-URL (Default: `http://localhost:11434`) | Nein |

---

## Checkliste vor Start

- [x] Design-Entscheidungen final (A, B, C, D)
- [ ] Bestehende Tests laufen (`make test`)
- [ ] Backup der Datenbank
- [ ] Claude API-Key für Testing verfügbar
- [ ] `XELANOTE_API_KEY_SECRET` generiert (`openssl rand -hex 32`)
