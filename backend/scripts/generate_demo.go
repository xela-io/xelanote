//go:build ignore

// generate_demo.go - Creates a demo database with realistic sample data for screenshots.
//
// Usage:
//
//	cd backend && CGO_ENABLED=1 go run -tags "fts5" scripts/generate_demo.go
//
// Then run the server with:
//
//	XELANOTE_DB=data/demo.db make dev
//
// Demo user: demo / demo1234
package main

import (
	"crypto/rand"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/db"
	"golang.org/x/crypto/bcrypt"
)

var outputPath = flag.String("output", "data/demo.db", "Output database path")

func main() {
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	os.Remove(*outputPath)

	database, err := db.Open(*outputPath, "")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	d := database.DB
	uid := seedUser(d)
	seedFolders(d, uid)
	noteIDs := seedNotes(d, uid)
	seedTags(d, uid, noteIDs)
	seedLinks(d, noteIDs)
	seedRecipeData(d, uid, noteIDs["Spaghetti Carbonara"])
	seedTemplates(d, uid)
	seedSnippets(d, uid)
	seedVersions(d, uid, noteIDs)
	seedPreferences(d, uid)
	seedFeatures(d, uid)
	seedDueDates(d, uid, noteIDs)

	fmt.Println("\n=== Demo Database Created ===")
	fmt.Printf("Database: %s\n", *outputPath)
	fmt.Printf("User:     demo / demo1234\n")
	fmt.Printf("Notes:    %d\n", len(noteIDs))
	fmt.Printf("\nTo use:\n")
	fmt.Printf("  XELANOTE_DB=%s make dev\n", *outputPath)
}

// --- Helpers ---

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func l(lines ...string) string { return strings.Join(lines, "\n") }

func nilStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func ts(daysAgo int) string {
	return time.Now().UTC().AddDate(0, 0, -daysAgo).Format("2006-01-02 15:04:05")
}

func mustExec(d *sql.DB, query string, args ...interface{}) {
	if _, err := d.Exec(query, args...); err != nil {
		log.Fatalf("SQL error: %v\nQuery: %s", err, query)
	}
}

// --- Seed Functions ---

func seedUser(d *sql.DB) int64 {
	hash, err := bcrypt.GenerateFromPassword([]byte("demo1234"), 12)
	if err != nil {
		log.Fatal(err)
	}
	result, err := d.Exec(
		`INSERT INTO users (username, email, password_hash, is_admin, created_at, updated_at)
		 VALUES (?, ?, ?, 1, ?, ?)`,
		"demo", "demo@example.com", string(hash), ts(30), ts(30))
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	id, _ := result.LastInsertId()
	log.Printf("Created user 'demo' (id=%d)", id)
	return id
}

func seedFolders(d *sql.DB, uid int64) {
	type folder struct {
		id       int
		path     string
		parentID interface{}
		name     string
		color    string
		order    int
	}

	folders := []folder{
		{1, "/Projects", nil, "Projects", "#458588", 0},
		{2, "/Projects/xelanote", 1, "xelanote", "#689d6a", 0},
		{3, "/Projects/Website Redesign", 1, "Website Redesign", "#d65d0e", 1},
		{4, "/Knowledge Base", nil, "Knowledge Base", "#98971a", 1},
		{5, "/Knowledge Base/Programming", 4, "Programming", "", 0},
		{6, "/Knowledge Base/DevOps", 4, "DevOps", "", 1},
		{7, "/Personal", nil, "Personal", "#b16286", 2},
		{8, "/Personal/Travel", 7, "Travel", "", 0},
		{9, "/Recipes", nil, "Recipes", "#cc241d", 3},
	}

	for _, f := range folders {
		mustExec(d,
			`INSERT INTO folders (id, path, parent_id, name, user_id, display_order, color, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			f.id, f.path, f.parentID, f.name, uid, f.order, nilStr(f.color), ts(30), ts(30))
	}
	log.Printf("Created %d folders", len(folders))
}

func seedNotes(d *sql.DB, uid int64) map[string]string {
	type note struct {
		title       string
		folder      string
		content     string
		noteType    string
		journalDate string
		color       string
		daysAgo     int
		updateAgo   int
		isDeleted   bool
	}

	notes := []note{
		// --- Root ---
		{
			title: "Welcome to xelanote", folder: "/", noteType: "note", daysAgo: 30, updateAgo: 5,
			content: l(
				"# Welcome to xelanote",
				"",
				"Your personal, self-hosted knowledge base for notes, ideas, and projects.",
				"",
				"## Getting Started",
				"",
				"xelanote is a modern note-taking app with powerful features:",
				"",
				"- **Markdown Editor** with live preview and syntax highlighting",
				"- **Folder Organization** with drag & drop and color coding",
				"- **Full-Text Search** across all your notes (powered by SQLite FTS5)",
				"- **Internal Links** between notes using [[Wiki Links]]",
				"- **Tags** for flexible, cross-folder categorization",
				"- **Version History** to track changes over time",
				"- **Journal** for daily reflections and logging",
				"- **Recipes** with ingredient management and scaling",
				"- **Templates & Snippets** for reusable content",
				"- **Sharing** notes and folders with other users",
				"- **Two-Factor Auth** with TOTP and FIDO2/WebAuthn",
				"- **End-to-End Encryption** for sensitive notes",
				"",
				"## Quick Tips",
				"",
				"> Press `Ctrl+K` to quickly search all your notes.",
				"",
				"> Double-click a folder in the sidebar to change its color.",
				"",
				"> Use `[[Note Title]]` to create links between notes.",
				"",
				"## Explore",
				"",
				"Check out the [[Architecture Overview]] to see how xelanote is built,",
				"or visit the [[Project Roadmap]] to see what's planned next.",
				"",
				"Browse the [[Docker Cheatsheet]] and [[Git Workflow Guide]] in the Knowledge Base",
				"for useful technical references.",
				"",
				"---",
				"",
				"*Happy note-taking!*",
			),
		},
		{
			title: "Quick Notes", folder: "/", noteType: "note", daysAgo: 28, updateAgo: 2,
			content: l(
				"Remember to update the SSL certificate before March 15.",
				"",
				"---",
				"",
				"Interesting talk by Mitchell Hashimoto about terminal emulators:",
				"",
				"- GPU-accelerated rendering",
				"- Custom font shaping pipeline",
				"- Cross-platform challenges with different OS text APIs",
				"",
				"---",
				"",
				"Meeting room changed to B-204 for Friday standups.",
				"",
				"---",
				"",
				"Good article on SQLite performance tuning:",
				"- `PRAGMA journal_mode = WAL` for concurrent reads",
				"- `PRAGMA synchronous = NORMAL` for better write speed",
				"- Use `EXPLAIN QUERY PLAN` to check index usage",
			),
		},

		// --- Projects/xelanote ---
		{
			title: "Project Roadmap", folder: "/Projects/xelanote", noteType: "note",
			color: "#689d6a", daysAgo: 25, updateAgo: 3,
			content: l(
				"# xelanote Roadmap",
				"",
				"## Phase 1: Core Features",
				"",
				"- [x] Markdown editor with live preview",
				"- [x] Folder-based organization with drag & drop",
				"- [x] Full-text search (SQLite FTS5)",
				"- [x] Tag system",
				"- [x] Dark and light themes (Gruvbox)",
				"- [x] User authentication with JWT",
				"- [x] Two-factor authentication (TOTP + FIDO2)",
				"",
				"## Phase 2: Collaboration",
				"",
				"- [x] Note sharing (viewer/editor roles)",
				"- [x] Folder sharing",
				"- [ ] Real-time collaborative editing",
				"- [ ] Comments on shared notes",
				"",
				"## Phase 3: Advanced Features",
				"",
				"- [x] Journal mode with calendar view",
				"- [x] Recipe management with ingredient scaling",
				"- [x] Note templates and snippets",
				"- [ ] Multi-tab editing",
				"- [ ] Split view for side-by-side editing",
				"- [ ] Mobile app (Progressive Web App)",
				"- [ ] Plugin system",
				"",
				"## Phase 4: Enterprise",
				"",
				"- [ ] LDAP/SSO integration",
				"- [ ] Audit log export",
				"- [ ] S3 storage backend",
				"- [ ] Team workspaces",
				"",
				"See [[Architecture Overview]] for technical details.",
			),
		},
		{
			title: "Architecture Overview", folder: "/Projects/xelanote", noteType: "note",
			daysAgo: 25, updateAgo: 7,
			content: l(
				"# Architecture Overview",
				"",
				"## Tech Stack",
				"",
				"| Component | Technology |",
				"|-----------|-----------|",
				"| Frontend | SvelteKit + Tailwind CSS v4 |",
				"| Backend | Go 1.24 + Chi Router |",
				"| Database | SQLite + FTS5 |",
				"| Auth | JWT + TOTP + FIDO2/WebAuthn |",
				"| Container | Docker (multi-stage build) |",
				"",
				"## Project Structure",
				"",
				"```",
				"xelanote/",
				"+-- frontend/          # SvelteKit application",
				"|   +-- src/",
				"|   |   +-- lib/       # Shared components & stores",
				"|   |   +-- routes/    # Page routes",
				"|   +-- tests/         # Vitest + Playwright",
				"+-- backend/           # Go API server",
				"|   +-- cmd/server/    # Entry point",
				"|   +-- internal/      # Business logic",
				"|       +-- api/       # HTTP handlers",
				"|       +-- db/        # SQLite data layer",
				"|       +-- service/   # Core services",
				"+-- docs/              # Documentation",
				"```",
				"",
				"## API Design",
				"",
				"The REST API follows standard conventions with JWT authentication:",
				"",
				"```go",
				"func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {",
				"    noteID := chi.URLParam(r, \"id\")",
				"    note, err := h.service.GetNote(r.Context(), noteID, userID)",
				"    if err != nil {",
				"        respondError(w, err)",
				"        return",
				"    }",
				"    respondJSON(w, http.StatusOK, note)",
				"}",
				"```",
				"",
				"## Database Design",
				"",
				"SQLite with FTS5 for full-text search and WAL mode for concurrent reads:",
				"",
				"```sql",
				"-- Core note storage",
				"CREATE TABLE notes (",
				"    note_rowid INTEGER PRIMARY KEY,",
				"    id TEXT UNIQUE NOT NULL,      -- UUID for API",
				"    title TEXT NOT NULL,",
				"    content TEXT NOT NULL,         -- Markdown",
				"    folder_path TEXT DEFAULT '/',",
				"    user_id INTEGER REFERENCES users(id)",
				");",
				"",
				"-- Full-text search index",
				"CREATE VIRTUAL TABLE notes_fts USING fts5(",
				"    title, content,",
				"    tokenize='unicode61 remove_diacritics 2'",
				");",
				"```",
				"",
				"## Security",
				"",
				"- Passwords hashed with **bcrypt** (cost 12)",
				"- Refresh tokens stored as hashed values",
				"- Optional **E2E encryption** (AES-256-GCM + Argon2id KDF)",
				"- CORS protection with configurable origins",
				"- Rate limiting on authentication endpoints",
				"",
				"See also: [[REST API Reference]], [[Deployment Guide]]",
			),
		},
		{
			title: "Sprint Review Notes", folder: "/Projects/xelanote", noteType: "note",
			daysAgo: 5, updateAgo: 5,
			content: l(
				"# Sprint Review \u2014 February 3, 2026",
				"",
				"## Attendees",
				"",
				"- Alex (Lead Dev)",
				"- Maria (Frontend)",
				"- Tom (Backend)",
				"",
				"## Completed This Sprint",
				"",
				"### Features",
				"",
				"- [x] Trash page for deleted notes",
				"- [x] Table of contents on mobile",
				"- [x] Docker Node.js 22 upgrade",
				"",
				"### Bug Fixes",
				"",
				"- [x] Fixed notes not displaying in trash view",
				"- [x] Fixed TOC transparent background",
				"- [x] Resolved nested scroll issue in editor",
				"",
				"## Demo Highlights",
				"",
				"Maria showed the new **trash management** feature:",
				"",
				"- Users can now view and restore deleted notes",
				"- Auto-cleanup after 30 days",
				"- Bulk restore coming next sprint",
				"",
				"Tom presented the **Docker upgrade**:",
				"",
				"> Moving from Node 20 to 22 improves build times by ~15%",
				"> and gives us native fetch support in SSR.",
				"",
				"## Action Items",
				"",
				"- [ ] @Alex: Review PR for multi-tab editing \u2014 **due Feb 10**",
				"- [ ] @Maria: Fix Prettier formatting in trash page",
				"- [ ] @Tom: Update deployment docs for Node 22",
				"- [ ] All: Write unit tests for new trash endpoints",
				"",
				"## Next Sprint Goals",
				"",
				"1. Multi-tab editing (Phase 1)",
				"2. Split view prototype",
				"3. Performance optimization for large notes",
			),
		},
		{
			title: "REST API Reference", folder: "/Projects/xelanote", noteType: "note",
			daysAgo: 20, updateAgo: 8,
			content: l(
				"# REST API Reference",
				"",
				"## Authentication",
				"",
				"All endpoints except `/api/auth/login` and `/api/auth/register` require:",
				"",
				"```",
				"Authorization: Bearer <jwt-token>",
				"```",
				"",
				"## Endpoints",
				"",
				"### Notes",
				"",
				"| Method | Endpoint | Description |",
				"|--------|----------|-------------|",
				"| GET | `/api/notes` | List all notes |",
				"| POST | `/api/notes` | Create a note |",
				"| GET | `/api/notes/:id` | Get a single note |",
				"| PUT | `/api/notes/:id` | Update a note |",
				"| DELETE | `/api/notes/:id` | Soft-delete a note |",
				"",
				"### Folders",
				"",
				"| Method | Endpoint | Description |",
				"|--------|----------|-------------|",
				"| GET | `/api/folders` | List folder tree |",
				"| POST | `/api/folders` | Create a folder |",
				"| PUT | `/api/folders/:id` | Rename or move |",
				"| DELETE | `/api/folders/:id` | Delete with contents |",
				"",
				"### Tags",
				"",
				"| Method | Endpoint | Description |",
				"|--------|----------|-------------|",
				"| GET | `/api/tags` | List all tags |",
				"| POST | `/api/notes/:id/tags` | Add tag to note |",
				"| DELETE | `/api/notes/:id/tags/:tagId` | Remove tag |",
				"",
				"### Search & Graph",
				"",
				"| Method | Endpoint | Description |",
				"|--------|----------|-------------|",
				"| GET | `/api/search?q=term` | Full-text search |",
				"| GET | `/api/graph` | Note link graph |",
				"",
				"## Example: Create a Note",
				"",
				"```bash",
				"curl -X POST http://localhost:8080/api/notes \\",
				"  -H \"Authorization: Bearer $TOKEN\" \\",
				"  -H \"Content-Type: application/json\" \\",
				"  -d '{",
				"    \"title\": \"My Note\",",
				"    \"content\": \"# Hello\\n\\nMarkdown content.\",",
				"    \"folder_path\": \"/Projects\"",
				"  }'",
				"```",
				"",
				"## Response Format",
				"",
				"```json",
				"{",
				"  \"id\": \"a1b2c3d4-e5f6-7890-abcd-ef1234567890\",",
				"  \"title\": \"My Note\",",
				"  \"folder_path\": \"/Projects\",",
				"  \"version\": 1,",
				"  \"created_at\": \"2026-02-09T10:30:00Z\"",
				"}",
				"```",
				"",
				"See also: [[Architecture Overview]]",
			),
		},
		{
			title: "Ideas Backlog", folder: "/Projects/xelanote", noteType: "note",
			color: "#d79921", daysAgo: 15, updateAgo: 2,
			content: l(
				"# Feature Ideas",
				"",
				"## High Priority",
				"",
				"- [ ] **Vim keybindings** \u2014 Many power users request this",
				"- [ ] **PDF export** \u2014 Export notes as styled PDFs",
				"- [ ] **Offline mode** \u2014 Service worker for offline access",
				"",
				"## Medium Priority",
				"",
				"- [ ] Mermaid diagram rendering in preview",
				"- [ ] Note pinning (pin to top of folder)",
				"- [ ] Kanban view for task-based notes",
				"- [ ] Import from Obsidian / Notion / Markdown files",
				"- [ ] Keyboard shortcuts customization",
				"",
				"## Low Priority / Someday",
				"",
				"- [ ] Voice notes with transcription",
				"- [ ] AI-powered note summarization",
				"- [ ] Calendar integration for journal",
				"- [ ] Browser extension for web clipping",
				"- [ ] Graph visualization of note links",
				"",
				"## Rejected",
				"",
				"- ~~Real-time collaboration via CRDT~~ \u2014 Too complex for SQLite backend",
				"- ~~Electron desktop app~~ \u2014 PWA is sufficient",
				"- ~~Built-in drawing tool~~ \u2014 Better to integrate Excalidraw",
			),
		},

		// --- Projects/Website Redesign ---
		{
			title: "Website Redesign Brief", folder: "/Projects/Website Redesign", noteType: "note",
			daysAgo: 12, updateAgo: 6,
			content: l(
				"# Website Redesign Brief",
				"",
				"## Objective",
				"",
				"Modernize the company website to improve conversion rates and user experience.",
				"",
				"## Timeline",
				"",
				"| Phase | Duration | Status |",
				"|-------|----------|--------|",
				"| Research & Discovery | 2 weeks | Complete |",
				"| Wireframing | 1 week | Complete |",
				"| Visual Design | 2 weeks | **In Progress** |",
				"| Development | 4 weeks | Upcoming |",
				"| Testing & Launch | 1 week | Upcoming |",
				"",
				"## Design Principles",
				"",
				"1. **Mobile-first** responsive approach",
				"2. **Accessibility** compliance (WCAG 2.1 AA)",
				"3. **Performance** targets (Core Web Vitals)",
				"4. Clean, **minimalist** aesthetic with Gruvbox palette",
				"",
				"## Action Items",
				"",
				"- [x] Conduct user interviews (12 participants)",
				"- [x] Analyze competitor websites",
				"- [x] Create wireframes for key pages",
				"- [ ] Finalize color palette and typography",
				"- [ ] Build component library in Storybook",
				"- [ ] Implement responsive layouts",
				"- [ ] Set up CI/CD pipeline",
				"- [ ] Performance testing with Lighthouse",
				"",
				"## Budget",
				"",
				"| Item | Cost |",
				"|------|------|",
				"| Design Tools | $200/mo |",
				"| Stock Photos | $500 |",
				"| Hosting (CDN) | $50/mo |",
				"| **Total (Year 1)** | **$3,500** |",
			),
		},

		// --- Knowledge Base/DevOps ---
		{
			title: "Docker Cheatsheet", folder: "/Knowledge Base/DevOps", noteType: "note",
			daysAgo: 22, updateAgo: 10,
			content: l(
				"# Docker Cheatsheet",
				"",
				"## Container Management",
				"",
				"```bash",
				"# Run a container in background",
				"docker run -d --name myapp -p 8080:80 nginx:latest",
				"",
				"# List running containers",
				"docker ps",
				"",
				"# Stop and remove",
				"docker stop myapp && docker rm myapp",
				"",
				"# View logs (follow, last 100 lines)",
				"docker logs -f --tail 100 myapp",
				"",
				"# Execute command in running container",
				"docker exec -it myapp /bin/sh",
				"```",
				"",
				"## Image Management",
				"",
				"```bash",
				"# Build from Dockerfile",
				"docker build -t myapp:latest .",
				"",
				"# Multi-stage build (smaller images)",
				"docker build --target production -t myapp:prod .",
				"",
				"# Tag and push to registry",
				"docker tag myapp:latest registry.example.com/myapp:v1.0",
				"docker push registry.example.com/myapp:v1.0",
				"",
				"# Clean up unused images",
				"docker image prune -a",
				"```",
				"",
				"## Docker Compose",
				"",
				"```yaml",
				"services:",
				"  app:",
				"    build: .",
				"    ports:",
				"      - \"8080:8080\"",
				"    environment:",
				"      - DATABASE_URL=sqlite:///data/app.db",
				"    volumes:",
				"      - app-data:/data",
				"    restart: unless-stopped",
				"",
				"volumes:",
				"  app-data:",
				"```",
				"",
				"## Useful Commands",
				"",
				"| Command | Description |",
				"|---------|-------------|",
				"| `docker stats` | Live resource usage |",
				"| `docker system df` | Disk usage summary |",
				"| `docker system prune` | Clean everything |",
				"| `docker inspect <id>` | Container details (JSON) |",
				"| `docker network ls` | List networks |",
				"",
				"> **Tip:** Use `docker compose` (V2, no hyphen) instead of `docker-compose`.",
				"",
				"See also: [[Deployment Guide]]",
			),
		},
		{
			title: "Deployment Guide", folder: "/Knowledge Base/DevOps", noteType: "note",
			daysAgo: 18, updateAgo: 5,
			content: l(
				"# Deployment Guide",
				"",
				"## Prerequisites",
				"",
				"- Docker 24+ installed",
				"- Domain with DNS configured",
				"- TLS certificate (Let's Encrypt recommended)",
				"",
				"## Quick Start",
				"",
				"### 1. Clone and Build",
				"",
				"```bash",
				"git clone https://git.example.com/xelanote.git",
				"cd xelanote",
				"make docker",
				"```",
				"",
				"### 2. Configure Environment",
				"",
				"```bash",
				"cp .env.example .env.production",
				"",
				"# Required settings:",
				"# JWT_SECRET=<at-least-64-random-characters>",
				"# CORS_ALLOWED_ORIGINS=https://notes.example.com",
				"```",
				"",
				"### 3. Run with Docker",
				"",
				"```bash",
				"docker run -d \\",
				"  --name xelanote \\",
				"  --restart unless-stopped \\",
				"  -p 8080:8080 \\",
				"  -v /data/xelanote:/app/data \\",
				"  --env-file .env.production \\",
				"  xelanote:latest",
				"```",
				"",
				"### 4. Verify",
				"",
				"```bash",
				"curl -s http://localhost:8080/health",
				"# {\"status\":\"ok\",\"version\":\"v1.2.3\"}",
				"```",
				"",
				"## Reverse Proxy (Caddy)",
				"",
				"```",
				"notes.example.com {",
				"    reverse_proxy localhost:8080",
				"}",
				"```",
				"",
				"## Backup",
				"",
				"```bash",
				"# Safe online backup (no downtime needed)",
				"sqlite3 /data/xelanote/xelanote.db \".backup /backups/xelanote-$(date +%F).db\"",
				"```",
				"",
				"## Monitoring",
				"",
				"- Health endpoint: `GET /health`",
				"- Activity logs: Admin panel > Activity tab",
				"- Database size: `docker exec xelanote du -sh /app/data/`",
				"",
				"> **Important:** Test your backup restoration process regularly!",
				"",
				"See also: [[Docker Cheatsheet]], [[Architecture Overview]]",
			),
		},

		// --- Knowledge Base/Programming ---
		{
			title: "Git Workflow Guide", folder: "/Knowledge Base/Programming", noteType: "note",
			daysAgo: 20, updateAgo: 12,
			content: l(
				"# Git Workflow Guide",
				"",
				"## Branch Strategy",
				"",
				"We use a simplified Git Flow:",
				"",
				"- `main` \u2014 production-ready code",
				"- `feature/*` \u2014 new features",
				"- `fix/*` \u2014 bug fixes",
				"- `release/*` \u2014 release preparation",
				"",
				"## Daily Workflow",
				"",
				"### 1. Start a Feature",
				"",
				"```bash",
				"git checkout main",
				"git pull origin main",
				"git checkout -b feature/my-feature",
				"```",
				"",
				"### 2. Commit Changes",
				"",
				"Use [Conventional Commits](https://conventionalcommits.org):",
				"",
				"```bash",
				"git add -p                    # Stage interactively",
				"git commit -m \"feat: add user authentication\"",
				"git commit -m \"fix: resolve login redirect loop\"",
				"git commit -m \"docs: update API documentation\"",
				"```",
				"",
				"### 3. Push and Create PR",
				"",
				"```bash",
				"git push -u origin feature/my-feature",
				"gh pr create --title \"Add user authentication\"",
				"```",
				"",
				"### 4. After Merge",
				"",
				"```bash",
				"git checkout main",
				"git pull origin main",
				"git branch -d feature/my-feature",
				"```",
				"",
				"## Useful Aliases",
				"",
				"```bash",
				"# Add to ~/.gitconfig",
				"[alias]",
				"    st = status -sb",
				"    lg = log --oneline --graph --decorate -20",
				"    undo = reset --soft HEAD~1",
				"    amend = commit --amend --no-edit",
				"```",
				"",
				"## Golden Rules",
				"",
				"1. **Never** force-push to `main`",
				"2. **Always** pull before pushing",
				"3. **Keep commits atomic** \u2014 one logical change per commit",
				"4. **Write meaningful** commit messages that explain *why*",
				"",
				"See also: [[Docker Cheatsheet]], [[Deployment Guide]]",
			),
		},
		{
			title: "Go Error Handling Patterns", folder: "/Knowledge Base/Programming", noteType: "note",
			daysAgo: 18, updateAgo: 18,
			content: l(
				"# Go Error Handling Patterns",
				"",
				"## The Basics",
				"",
				"Go uses explicit error returns instead of exceptions:",
				"",
				"```go",
				"func readFile(path string) ([]byte, error) {",
				"    data, err := os.ReadFile(path)",
				"    if err != nil {",
				"        return nil, fmt.Errorf(\"reading %s: %w\", path, err)",
				"    }",
				"    return data, nil",
				"}",
				"```",
				"",
				"## Always Wrap Errors",
				"",
				"Add context when propagating errors:",
				"",
				"```go",
				"// Bad - loses context",
				"if err != nil {",
				"    return err",
				"}",
				"",
				"// Good - adds context with %w for wrapping",
				"if err != nil {",
				"    return fmt.Errorf(\"creating user %q: %w\", username, err)",
				"}",
				"```",
				"",
				"## Custom Error Types",
				"",
				"```go",
				"type NotFoundError struct {",
				"    Resource string",
				"    ID       string",
				"}",
				"",
				"func (e *NotFoundError) Error() string {",
				"    return fmt.Sprintf(\"%s %q not found\", e.Resource, e.ID)",
				"}",
				"",
				"// Check with errors.As",
				"var target *NotFoundError",
				"if errors.As(err, &target) {",
				"    http.Error(w, target.Error(), 404)",
				"}",
				"```",
				"",
				"## Sentinel Errors",
				"",
				"```go",
				"var (",
				"    ErrNotFound     = errors.New(\"not found\")",
				"    ErrUnauthorized = errors.New(\"unauthorized\")",
				")",
				"",
				"if errors.Is(err, ErrNotFound) {",
				"    // handle not found",
				"}",
				"```",
				"",
				"## Error Groups",
				"",
				"For concurrent operations:",
				"",
				"```go",
				"g, ctx := errgroup.WithContext(ctx)",
				"",
				"g.Go(func() error { return fetchUsers(ctx) })",
				"g.Go(func() error { return fetchOrders(ctx) })",
				"",
				"if err := g.Wait(); err != nil {",
				"    log.Printf(\"operation failed: %v\", err)",
				"}",
				"```",
				"",
				"> **Rule of thumb:** Handle errors at the level where you have",
				"> enough context to make a meaningful decision.",
			),
		},

		// --- Personal ---
		{
			title: "Japan Travel Plan 2026", folder: "/Personal/Travel", noteType: "note",
			daysAgo: 10, updateAgo: 3,
			content: l(
				"# Japan Trip \u2014 October 2026",
				"",
				"## Itinerary",
				"",
				"### Week 1: Tokyo",
				"",
				"- [ ] Book flights (ANA direct, ~\u20ac800 pp)",
				"- [ ] Reserve hotel in Shinjuku",
				"- [ ] Visit Meiji Shrine & Harajuku",
				"- [ ] Explore Akihabara & Shibuya",
				"- [ ] Day trip to Kamakura (Great Buddha)",
				"",
				"### Week 2: Kyoto & Osaka",
				"",
				"- [ ] Shinkansen tickets (get JR Pass)",
				"- [ ] Fushimi Inari (early morning!)",
				"- [ ] Kinkaku-ji & Arashiyama bamboo grove",
				"- [ ] Street food tour in Dotonbori, Osaka",
				"- [ ] Nara deer park day trip",
				"",
				"## Budget",
				"",
				"| Category | Estimated |",
				"|----------|-----------|",
				"| Flights (2 pax) | \u20ac1,600 |",
				"| Hotels (14 nights) | \u20ac2,100 |",
				"| JR Pass (14 days) | \u20ac400 |",
				"| Food & Drinks | \u20ac700 |",
				"| Activities & Entrance | \u20ac300 |",
				"| Shopping & Souvenirs | \u20ac500 |",
				"| **Total** | **\u20ac5,600** |",
				"",
				"## Packing Checklist",
				"",
				"- [x] Passport (valid until 2028)",
				"- [ ] Travel adapter (Type A/B)",
				"- [ ] Pocket WiFi reservation",
				"- [ ] Comfortable walking shoes",
				"- [ ] Rain jacket (October = typhoon season)",
				"- [ ] Power bank for phone",
				"",
				"## Useful Phrases",
				"",
				"| Japanese | English |",
				"|----------|---------|",
				"| \u3059\u307f\u307e\u305b\u3093 | Excuse me |",
				"| \u3042\u308a\u304c\u3068\u3046\u3054\u3056\u3044\u307e\u3059 | Thank you very much |",
				"| \u3044\u304f\u3089\u3067\u3059\u304b | How much is this? |",
				"| \u82f1\u8a9e\u3092\u8a71\u305b\u307e\u3059\u304b | Do you speak English? |",
				"",
				"> **Tip:** Get a Suica/Pasmo card at the airport \u2014 works for trains, buses, and convenience stores!",
			),
		},
		{
			title: "Reading List", folder: "/Personal", noteType: "note",
			daysAgo: 20, updateAgo: 1,
			content: l(
				"# Reading List 2026",
				"",
				"## Currently Reading",
				"",
				"- [ ] **Designing Data-Intensive Applications** \u2014 Martin Kleppmann",
				"  *Chapter 7: Transactions*",
				"",
				"## To Read",
				"",
				"### Technical",
				"",
				"- [ ] \"The Pragmatic Programmer\" \u2014 Hunt & Thomas",
				"- [ ] \"Clean Architecture\" \u2014 Robert C. Martin",
				"- [ ] \"Database Internals\" \u2014 Alex Petrov",
				"",
				"### Non-Fiction",
				"",
				"- [ ] \"Thinking, Fast and Slow\" \u2014 Daniel Kahneman",
				"- [ ] \"Deep Work\" \u2014 Cal Newport",
				"- [ ] \"Atomic Habits\" \u2014 James Clear",
				"",
				"## Completed",
				"",
				"### 2026",
				"",
				"- [x] \"The Go Programming Language\" \u2014 Donovan & Kernighan \u2b50\u2b50\u2b50\u2b50\u2b50",
				"- [x] \"System Design Interview\" \u2014 Alex Xu \u2b50\u2b50\u2b50\u2b50",
				"",
				"### 2025",
				"",
				"- [x] \"Crafting Interpreters\" \u2014 Robert Nystrom \u2b50\u2b50\u2b50\u2b50\u2b50",
				"- [x] \"Staff Engineer\" \u2014 Will Larson \u2b50\u2b50\u2b50\u2b50",
				"- [x] \"Project Hail Mary\" \u2014 Andy Weir \u2b50\u2b50\u2b50\u2b50\u2b50",
			),
		},

		// --- Recipe ---
		{
			title: "Spaghetti Carbonara", folder: "/Recipes", noteType: "recipe",
			daysAgo: 8, updateAgo: 8,
			content: l(
				"# Spaghetti Carbonara",
				"",
				"The authentic Roman recipe \u2014 no cream needed!",
				"",
				"## Instructions",
				"",
				"1. Bring a large pot of salted water to a rolling boil",
				"2. Cook spaghetti until **al dente** (1 min less than package directions)",
				"3. While pasta cooks, cut guanciale into small strips",
				"4. Fry guanciale in a cold pan on medium heat until crispy (~8 min)",
				"5. In a bowl, whisk together egg yolks, whole eggs, and grated pecorino",
				"6. Add generous fresh-cracked black pepper to the egg mixture",
				"7. When pasta is done, **reserve 1 cup of pasta water** before draining",
				"8. Add drained pasta to the guanciale pan (**heat OFF!**)",
				"9. Toss quickly, then add egg mixture while stirring vigorously",
				"10. Add pasta water a splash at a time for a silky, creamy consistency",
				"11. Serve immediately with extra pecorino and black pepper on top",
				"",
				"> **Key tip:** Never add the egg mixture to a hot pan \u2014 it will scramble!",
				"> Take the pan off the heat first and toss quickly.",
				"",
				"## Notes",
				"",
				"- Guanciale > Pancetta > Bacon (in order of authenticity)",
				"- Pecorino Romano is essential \u2014 Parmesan changes the flavor",
				"- Fresh pasta works, but dried spaghetti is the traditional choice",
				"- The residual heat from pasta and pan is enough to cook the eggs",
				"- Total time: about 25 minutes from start to plate",
			),
		},

		// --- Journals ---
		{
			title: "February 9, 2026", folder: "/", noteType: "journal",
			journalDate: "2026-02-09", daysAgo: 0, updateAgo: 0,
			content: l(
				"# Sunday, February 9",
				"",
				"Quiet Sunday. Spent the morning working on xelanote:",
				"",
				"- Created demo data for repository screenshots",
				"- Reviewed the multi-tab editing plan",
				"- Fixed a small CSS issue in the sidebar hover state",
				"",
				"**Afternoon:** Went for a walk in the park. The weather is surprisingly mild for February.",
				"",
				"**Reading:** Continued with \"Designing Data-Intensive Applications\" \u2014 the chapter on",
				"transactions is fascinating. The comparison between serializable snapshot isolation and",
				"two-phase locking really clicked today.",
				"",
				"## Tomorrow's Priorities",
				"",
				"- [ ] Review PR from Maria",
				"- [ ] Start implementing tab store",
				"- [ ] Update deployment documentation",
			),
		},
		{
			title: "February 8, 2026", folder: "/", noteType: "journal",
			journalDate: "2026-02-08", daysAgo: 1, updateAgo: 1,
			content: l(
				"# Saturday, February 8",
				"",
				"## Sprint Planning",
				"",
				"Good session today. We agreed on the scope for the next two weeks:",
				"",
				"1. Multi-tab editing (MVP)",
				"2. Performance improvements for search",
				"3. Bug fixes from user feedback",
				"",
				"The team is excited about multi-tab. Maria had great ideas for the UX.",
				"",
				"## Evening",
				"",
				"Tried a new [[Spaghetti Carbonara]] recipe \u2014 turned out great!",
				"Need to remember: no cream, just eggs and pecorino.",
				"The key is tempering the eggs slowly off the heat.",
			),
		},
		{
			title: "February 6, 2026", folder: "/", noteType: "journal",
			journalDate: "2026-02-06", daysAgo: 3, updateAgo: 3,
			content: l(
				"# Thursday, February 6",
				"",
				"Long debugging session today. The production deployment was failing",
				"intermittently after container restarts.",
				"",
				"**Root cause:** The Docker volume was configured as a named volume",
				"instead of a bind mount, which created an empty database on each restart.",
				"",
				"**Fix:** Changed the `docker run` command to use",
				"`-v /data/xelanote:/app/data` instead of the named volume.",
				"",
				"**Lesson learned:** Always verify mount types when containers behave",
				"differently between restarts. Named volumes persist data, but if the",
				"initial run doesn't copy data in, you get an empty volume.",
				"",
				"Also fixed the Prettier formatting issue in the trash page component.",
				"Small but annoying \u2014 the pre-commit hook kept failing.",
			),
		},

		// --- Deleted note ---
		{
			title: "Old Meeting Notes", folder: "/Projects/xelanote", noteType: "note",
			daysAgo: 25, updateAgo: 25, isDeleted: true,
			content: l(
				"# Team Sync \u2014 January 15, 2026",
				"",
				"This meeting has been superseded by the Sprint Review format.",
				"",
				"## Notes",
				"",
				"- Discussed initial project setup",
				"- Agreed on tech stack (Go + SvelteKit)",
				"- Set up CI/CD pipeline",
				"",
				"See [[Sprint Review Notes]] for the latest updates.",
			),
		},
	}

	noteIDs := make(map[string]string)

	for i, n := range notes {
		id := newUUID()
		noteIDs[n.title] = id

		created := ts(n.daysAgo)
		updated := ts(n.updateAgo)

		isDeleted := 0
		var deletedAt interface{} = nil
		if n.isDeleted {
			isDeleted = 1
			deletedAt = ts(5)
		}

		_, err := d.Exec(
			`INSERT INTO notes (id, title, title_norm, content, folder_path, user_id,
			 note_type, journal_date, color, display_order, is_deleted, deleted_at,
			 created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, n.title, strings.ToLower(strings.TrimSpace(n.title)), n.content,
			n.folder, uid, n.noteType, nilStr(n.journalDate), nilStr(n.color),
			i, isDeleted, deletedAt, created, updated)
		if err != nil {
			log.Fatalf("Failed to insert note %q: %v", n.title, err)
		}
	}

	log.Printf("Created %d notes (%d active, %d deleted)",
		len(notes),
		len(notes)-1, // all except deleted
		1)

	return noteIDs
}

func seedTags(d *sql.DB, uid int64, noteIDs map[string]string) {
	tagNames := []string{
		"project", "important", "reference", "todo", "programming",
		"devops", "personal", "meeting", "guide", "documentation",
	}

	tagIDs := make(map[string]int64)
	for i, name := range tagNames {
		id := int64(i + 1)
		mustExec(d, `INSERT INTO tags (id, name, name_norm, user_id) VALUES (?, ?, ?, ?)`,
			id, name, strings.ToLower(name), uid)
		tagIDs[name] = id
	}

	assignments := map[string][]string{
		"Welcome to xelanote":       {"guide"},
		"Project Roadmap":           {"project", "todo"},
		"Architecture Overview":     {"project", "reference", "documentation"},
		"Sprint Review Notes":       {"meeting", "project"},
		"REST API Reference":        {"reference", "documentation", "project"},
		"Ideas Backlog":             {"project", "todo"},
		"Website Redesign Brief":    {"project", "todo"},
		"Docker Cheatsheet":         {"reference", "devops"},
		"Deployment Guide":          {"guide", "devops"},
		"Git Workflow Guide":        {"guide", "programming"},
		"Go Error Handling Patterns": {"programming", "reference"},
		"Japan Travel Plan 2026":    {"personal", "todo"},
		"Reading List":              {"personal"},
	}

	count := 0
	for title, tags := range assignments {
		noteID, ok := noteIDs[title]
		if !ok {
			continue
		}
		for _, tag := range tags {
			tagID := tagIDs[tag]
			mustExec(d, `INSERT INTO note_tags (note_id, tag_id) VALUES (?, ?)`, noteID, tagID)
			count++
		}
	}
	log.Printf("Created %d tags, %d assignments", len(tagNames), count)
}

func seedLinks(d *sql.DB, noteIDs map[string]string) {
	// Resolved links (both notes exist)
	links := [][2]string{
		{"Welcome to xelanote", "Architecture Overview"},
		{"Welcome to xelanote", "Project Roadmap"},
		{"Welcome to xelanote", "Docker Cheatsheet"},
		{"Welcome to xelanote", "Git Workflow Guide"},
		{"Project Roadmap", "Architecture Overview"},
		{"Architecture Overview", "REST API Reference"},
		{"Architecture Overview", "Deployment Guide"},
		{"Docker Cheatsheet", "Deployment Guide"},
		{"Git Workflow Guide", "Docker Cheatsheet"},
		{"Git Workflow Guide", "Deployment Guide"},
		{"Deployment Guide", "Docker Cheatsheet"},
		{"Deployment Guide", "Architecture Overview"},
		{"REST API Reference", "Architecture Overview"},
		{"February 8, 2026", "Spaghetti Carbonara"},
		{"Old Meeting Notes", "Sprint Review Notes"},
	}

	for _, link := range links {
		src, ok1 := noteIDs[link[0]]
		tgt, ok2 := noteIDs[link[1]]
		if ok1 && ok2 {
			d.Exec(`INSERT OR IGNORE INTO links (source_id, target_id) VALUES (?, ?)`, src, tgt)
		}
	}

	// Unresolved links (target doesn't exist yet)
	unresolved := [][2]string{
		{"Ideas Backlog", "Vim Keybindings"},
		{"Ideas Backlog", "PDF Export"},
		{"Project Roadmap", "Mobile App"},
		{"Website Redesign Brief", "Component Library"},
	}

	for _, link := range unresolved {
		src, ok := noteIDs[link[0]]
		if ok {
			d.Exec(`INSERT OR IGNORE INTO unresolved_links (source_id, target_ref, target_ref_norm)
				VALUES (?, ?, ?)`, src, link[1], strings.ToLower(link[1]))
		}
	}

	log.Printf("Created %d resolved links, %d unresolved links", len(links), len(unresolved))
}

func seedRecipeData(d *sql.DB, uid int64, noteID string) {
	if noteID == "" {
		log.Println("Skipping recipe data: note ID not found")
		return
	}

	mustExec(d,
		`INSERT INTO recipe_metadata (note_id, user_id, servings, prep_time_minutes, cook_time_minutes, difficulty)
		 VALUES (?, ?, 4, 10, 15, 'easy')`, noteID, uid)

	ingredients := []struct {
		amount     float64
		amountText string
		unit       string
		name       string
		scalable   int
	}{
		{400, "400", "g", "Spaghetti", 1},
		{200, "200", "g", "Guanciale", 1},
		{4, "4", "", "Egg yolks", 1},
		{2, "2", "", "Whole eggs", 1},
		{100, "100", "g", "Pecorino Romano (finely grated)", 1},
		{0, "", "", "Black pepper (freshly cracked)", 0},
		{0, "", "", "Salt (for pasta water)", 0},
	}

	for i, ing := range ingredients {
		mustExec(d,
			`INSERT INTO recipe_ingredients (note_id, user_id, amount, amount_text, unit, name, display_order, scalable)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			noteID, uid, ing.amount, ing.amountText, ing.unit, ing.name, i, ing.scalable)
	}
	log.Printf("Created recipe metadata + %d ingredients", len(ingredients))
}

func seedTemplates(d *sql.DB, uid int64) {
	templates := []struct {
		name, desc, title, content string
	}{
		{
			name: "Meeting Notes",
			desc: "Template for recurring meetings with agenda, notes, and action items",
			title: "Meeting: [Topic]",
			content: l(
				"# Meeting: [Topic]",
				"",
				"**Date:** [Date]",
				"**Attendees:** [Names]",
				"",
				"## Agenda",
				"",
				"1. ",
				"2. ",
				"3. ",
				"",
				"## Discussion Notes",
				"",
				"",
				"",
				"## Action Items",
				"",
				"- [ ] @[Name]: [Task] \u2014 due [Date]",
				"- [ ] @[Name]: [Task] \u2014 due [Date]",
				"",
				"## Next Meeting",
				"",
				"- Date: [Date]",
				"- Topics: ",
			),
		},
		{
			name: "Project Brief",
			desc: "Structured project kickoff document with goals, timeline, and risks",
			title: "Project: [Name]",
			content: l(
				"# Project: [Name]",
				"",
				"## Overview",
				"",
				"[Brief description of the project and its goals]",
				"",
				"## Goals",
				"",
				"1. ",
				"2. ",
				"3. ",
				"",
				"## Timeline",
				"",
				"| Phase | Duration | Status |",
				"|-------|----------|--------|",
				"| Planning | | |",
				"| Development | | |",
				"| Testing | | |",
				"| Launch | | |",
				"",
				"## Team",
				"",
				"- **Lead:** [Name]",
				"- **Frontend:** [Name]",
				"- **Backend:** [Name]",
				"",
				"## Risks & Mitigations",
				"",
				"| Risk | Impact | Mitigation |",
				"|------|--------|------------|",
				"| | | |",
			),
		},
	}

	for _, t := range templates {
		mustExec(d,
			`INSERT INTO templates (user_id, name, name_norm, description, title, content, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			uid, t.name, strings.ToLower(t.name), t.desc, t.title, t.content, ts(20), ts(20))
	}
	log.Printf("Created %d templates", len(templates))
}

func seedSnippets(d *sql.DB, uid int64) {
	snippets := []struct {
		name, desc, content string
	}{
		{
			name: "Go HTTP Handler",
			desc: "Boilerplate for a Chi HTTP handler with JSON response",
			content: l(
				"func (h *Handler) Handle[Name](w http.ResponseWriter, r *http.Request) {",
				"\tid := chi.URLParam(r, \"id\")",
				"",
				"\tresult, err := h.service.GetByID(r.Context(), id)",
				"\tif err != nil {",
				"\t\trespondError(w, err)",
				"\t\treturn",
				"\t}",
				"",
				"\trespondJSON(w, http.StatusOK, result)",
				"}",
			),
		},
		{
			name: "Docker Compose Service",
			desc: "Template for a Docker Compose service definition",
			content: l(
				"  service-name:",
				"    image: ${IMAGE}:${TAG:-latest}",
				"    restart: unless-stopped",
				"    ports:",
				"      - \"${PORT:-8080}:8080\"",
				"    environment:",
				"      - DATABASE_URL=${DATABASE_URL}",
				"    volumes:",
				"      - data:/app/data",
				"    healthcheck:",
				"      test: [\"CMD\", \"curl\", \"-f\", \"http://localhost:8080/health\"]",
				"      interval: 30s",
				"      timeout: 5s",
				"      retries: 3",
			),
		},
	}

	for _, s := range snippets {
		mustExec(d,
			`INSERT INTO snippets (user_id, name, name_norm, description, content, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			uid, s.name, strings.ToLower(s.name), s.desc, s.content, ts(20), ts(20))
	}
	log.Printf("Created %d snippets", len(snippets))
}

func seedVersions(d *sql.DB, uid int64, noteIDs map[string]string) {
	archID, ok := noteIDs["Architecture Overview"]
	if !ok {
		return
	}

	// Old version of Architecture Overview (simpler, earlier draft)
	oldContent := l(
		"# Architecture Overview",
		"",
		"## Tech Stack",
		"",
		"- Frontend: SvelteKit",
		"- Backend: Go",
		"- Database: SQLite",
		"",
		"## Project Structure",
		"",
		"Work in progress \u2014 see README.md for current details.",
		"",
		"## Notes",
		"",
		"- Using Chi router for HTTP",
		"- FTS5 for full-text search",
		"- JWT for authentication",
	)

	mustExec(d,
		`INSERT INTO note_versions (note_id, user_id, version, title, content, snapshot_at)
		 VALUES (?, ?, 1, 'Architecture Overview', ?, ?)`,
		archID, uid, oldContent, ts(25))

	// Update current note to version 2
	mustExec(d, `UPDATE notes SET version = 2 WHERE id = ?`, archID)

	// Also add a version for Project Roadmap
	roadmapID, ok := noteIDs["Project Roadmap"]
	if ok {
		oldRoadmap := l(
			"# xelanote Roadmap",
			"",
			"## Phase 1: Core Features",
			"",
			"- [x] Markdown editor",
			"- [x] Folder organization",
			"- [ ] Full-text search",
			"- [ ] Tag system",
			"- [ ] Authentication",
			"",
			"## Phase 2: TBD",
			"",
			"To be planned after Phase 1 is complete.",
		)
		mustExec(d,
			`INSERT INTO note_versions (note_id, user_id, version, title, content, snapshot_at)
			 VALUES (?, ?, 1, 'Project Roadmap', ?, ?)`,
			roadmapID, uid, oldRoadmap, ts(25))
		mustExec(d, `UPDATE notes SET version = 2 WHERE id = ?`, roadmapID)
	}

	log.Println("Created 2 version history entries")
}

func seedPreferences(d *sql.DB, uid int64) {
	mustExec(d,
		`INSERT INTO user_preferences (user_id, theme, editor_mode, created_at, updated_at)
		 VALUES (?, 'gruvbox-dark', 'split', ?, ?)`,
		uid, ts(30), ts(30))
	log.Println("Set user preferences (gruvbox-dark theme, split editor)")
}

func seedFeatures(d *sql.DB, uid int64) {
	features := []string{"journal", "recipes"}
	for _, f := range features {
		mustExec(d,
			`INSERT INTO user_features (user_id, feature, enabled, created_at, updated_at)
			 VALUES (?, ?, 1, ?, ?)`,
			uid, f, ts(30), ts(30))
	}
	log.Printf("Enabled features: %s", strings.Join(features, ", "))
}

func seedDueDates(d *sql.DB, uid int64, noteIDs map[string]string) {
	sprintID, ok := noteIDs["Sprint Review Notes"]
	if !ok {
		return
	}

	mustExec(d,
		`INSERT INTO note_due_dates (note_id, user_id, line_text, line_index, due_date, is_task_item, is_completed)
		 VALUES (?, ?, ?, ?, ?, 1, 0)`,
		sprintID, uid, "@Alex: Review PR for multi-tab editing", 0, "2026-02-10")

	mustExec(d,
		`INSERT INTO note_due_dates (note_id, user_id, line_text, line_index, due_date, is_task_item, is_completed)
		 VALUES (?, ?, ?, ?, ?, 1, 0)`,
		sprintID, uid, "@Maria: Fix Prettier formatting in trash page", 1, "2026-02-12")

	mustExec(d,
		`INSERT INTO note_due_dates (note_id, user_id, line_text, line_index, due_date, is_task_item, is_completed)
		 VALUES (?, ?, ?, ?, ?, 1, 0)`,
		sprintID, uid, "@Tom: Update deployment docs for Node 22", 2, "2026-02-12")

	log.Println("Created 3 due date entries")
}
