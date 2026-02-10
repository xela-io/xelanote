//go:build ignore
// +build ignore

// generate_fixture.go - Creates a deterministic test database for performance benchmarks
//
// Usage:
//   go run -tags "fts5" backend/scripts/generate_fixture.go -output data/xelanote_fixture.db -seed 42
//
// Requirements:
//   - CGO_ENABLED=1
//   - Build tag: fts5
//
// Output:
//   - 500 notes (20% small, 60% medium, 20% large, 10% encrypted)
//   - 50 folders (5 root, 15 level-1, 30 level-2)
//   - 1000 links (70% resolved, 30% unresolved, 5 hub nodes)
//   - 100 tags with 500 assignments

package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math"
	mrand "math/rand"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var (
	outputPath = flag.String("output", "data/xelanote_fixture.db", "Output database path")
	seed       = flag.Int64("seed", 42, "Random seed for reproducibility")
)

// Constants for fixture generation
const (
	NumNotes            = 500
	NumFolders          = 50
	NumLinks            = 1000
	NumTags             = 100
	NumTagAssignments   = 500
	NumRootFolders      = 5
	NumLevel1Folders    = 15
	NumLevel2Folders    = 30
	EncryptedPercentage = 10
	ResolvedLinksPct    = 70
	NumHubNodes         = 5
	HubBacklinks        = 50
)

type Note struct {
	ID        string
	Title     string
	TitleNorm string
	Content   string
	FolderID  int
	Encrypted bool
}

type Folder struct {
	ID       int
	Path     string
	ParentID *int
	Name     string
	Level    int
}

func main() {
	flag.Parse()

	// Initialize deterministic random with seed
	rng := mrand.New(mrand.NewSource(*seed))

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Remove existing file
	if _, err := os.Stat(*outputPath); err == nil {
		if err := os.Remove(*outputPath); err != nil {
			log.Fatalf("Failed to remove existing database: %v", err)
		}
	}

	// Open database with FTS5 support
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_fk=1", *outputPath))
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Enable WAL mode
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Fatalf("Failed to enable WAL: %v", err)
	}

	log.Println("Creating schema...")
	if err := createSchema(db); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	log.Println("Creating test user...")
	userID, err := createTestUser(db)
	if err != nil {
		log.Fatalf("Failed to create test user: %v", err)
	}

	log.Println("Generating folders...")
	folders, err := generateFolders(db, rng, userID)
	if err != nil {
		log.Fatalf("Failed to generate folders: %v", err)
	}
	log.Printf("  Created %d folders", len(folders))

	log.Println("Generating notes...")
	notes, err := generateNotes(db, rng, userID, folders)
	if err != nil {
		log.Fatalf("Failed to generate notes: %v", err)
	}
	log.Printf("  Created %d notes", len(notes))

	log.Println("Generating links...")
	resolvedLinks, unresolvedLinks, err := generateLinks(db, rng, notes)
	if err != nil {
		log.Fatalf("Failed to generate links: %v", err)
	}
	log.Printf("  Created %d resolved, %d unresolved links", resolvedLinks, unresolvedLinks)

	log.Println("Generating tags...")
	tagsCreated, tagsAssigned, err := generateTags(db, rng, userID, notes)
	if err != nil {
		log.Fatalf("Failed to generate tags: %v", err)
	}
	log.Printf("  Created %d tags, %d assignments", tagsCreated, tagsAssigned)

	// Output first note ID for benchmarking
	fmt.Println()
	fmt.Println("=== Fixture Generation Complete ===")
	fmt.Printf("Database: %s\n", *outputPath)
	fmt.Printf("Notes: %d | Folders: %d | Links: %d resolved, %d unresolved | Tags: %d\n",
		len(notes), len(folders), resolvedLinks, unresolvedLinks, tagsCreated)
	fmt.Printf("\nFirst note ID for benchmarks: %s\n", notes[0].ID)
	fmt.Println("\nTo use:")
	fmt.Printf("  XELANOTE_DB=%s make run-backend\n", *outputPath)
}

func createSchema(db *sql.DB) error {
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		is_admin INTEGER DEFAULT 0,
		encryption_salt BLOB,
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now'))
	);

	-- Folders table
	CREATE TABLE IF NOT EXISTS folders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL,
		parent_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now')),
		user_id INTEGER,
		display_order INTEGER NOT NULL DEFAULT 0,
		color TEXT DEFAULT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_folders_path ON folders(path);
	CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id);
	CREATE INDEX IF NOT EXISTS idx_folders_user_id ON folders(user_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_folders_user_path ON folders(user_id, path);

	-- Notes table
	CREATE TABLE IF NOT EXISTS notes (
		note_rowid INTEGER PRIMARY KEY,
		id TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		title_norm TEXT NOT NULL,
		content TEXT NOT NULL,
		folder_path TEXT DEFAULT '/',
		version INTEGER NOT NULL DEFAULT 1,
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now')),
		is_deleted INTEGER DEFAULT 0,
		deleted_at TEXT,
		user_id INTEGER,
		encrypted_content BLOB,
		content_encrypted INTEGER DEFAULT 0,
		encrypted_title TEXT,
		title_encrypted INTEGER DEFAULT 0,
		wrapped_dek TEXT,
		encryption_version INTEGER DEFAULT 0,
		encryption_metadata TEXT,
		color TEXT DEFAULT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_notes_folder ON notes(folder_path);
	CREATE INDEX IF NOT EXISTS idx_notes_id ON notes(id);
	CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);
	CREATE INDEX IF NOT EXISTS idx_notes_deleted ON notes(is_deleted, deleted_at);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_notes_user_title_norm ON notes(user_id, title_norm) WHERE is_deleted = 0;

	-- FTS5 for notes
	CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
		title, content,
		content='notes',
		content_rowid='note_rowid',
		tokenize='unicode61 remove_diacritics 2'
	);

	-- FTS Triggers
	CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes WHEN NEW.is_deleted = 0 BEGIN
		INSERT INTO notes_fts(rowid, title, content)
		VALUES (NEW.note_rowid, NEW.title, NEW.content);
	END;

	CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN
		INSERT INTO notes_fts(notes_fts, rowid, title, content)
		VALUES('delete', OLD.note_rowid, OLD.title, OLD.content);
		INSERT INTO notes_fts(rowid, title, content)
		SELECT NEW.note_rowid, NEW.title, NEW.content WHERE NEW.is_deleted = 0;
	END;

	CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN
		INSERT INTO notes_fts(notes_fts, rowid, title, content)
		VALUES('delete', OLD.note_rowid, OLD.title, OLD.content);
	END;

	-- Links table
	CREATE TABLE IF NOT EXISTS links (
		source_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
		target_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
		PRIMARY KEY (source_id, target_id)
	);
	CREATE INDEX IF NOT EXISTS idx_links_target ON links(target_id);

	-- Unresolved Links table
	CREATE TABLE IF NOT EXISTS unresolved_links (
		source_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
		target_ref TEXT NOT NULL,
		target_ref_norm TEXT NOT NULL,
		PRIMARY KEY (source_id, target_ref)
	);
	CREATE INDEX IF NOT EXISTS idx_unresolved_norm ON unresolved_links(target_ref_norm);

	-- Tags tables
	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		name_norm TEXT UNIQUE NOT NULL,
		user_id INTEGER
	);

	CREATE TABLE IF NOT EXISTS note_tags (
		note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (note_id, tag_id)
	);
	CREATE INDEX IF NOT EXISTS idx_note_tags_tag ON note_tags(tag_id);
	`
	_, err := db.Exec(schema)
	return err
}

func createTestUser(db *sql.DB) (int64, error) {
	// Generate a hashed password for "testpassword123"
	hash, err := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	result, err := db.Exec(`
		INSERT INTO users (username, email, password_hash, is_admin)
		VALUES (?, ?, ?, ?)
	`, "testuser", "test@example.com", string(hash), 0)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func generateFolders(db *sql.DB, rng *mrand.Rand, userID int64) ([]Folder, error) {
	folders := make([]Folder, 0, NumFolders)
	folderID := 1

	// Generate root-level folders
	for i := 0; i < NumRootFolders; i++ {
		name := fmt.Sprintf("Category-%d", i+1)
		path := "/" + name
		folders = append(folders, Folder{
			ID:       folderID,
			Path:     path,
			ParentID: nil,
			Name:     name,
			Level:    0,
		})
		folderID++
	}

	// Generate level-1 folders (children of root)
	level1Start := len(folders)
	for i := 0; i < NumLevel1Folders; i++ {
		parentIdx := rng.Intn(NumRootFolders)
		parent := folders[parentIdx]
		name := fmt.Sprintf("Subcategory-%d-%d", parentIdx+1, i+1)
		path := parent.Path + "/" + name
		parentID := parent.ID
		folders = append(folders, Folder{
			ID:       folderID,
			Path:     path,
			ParentID: &parentID,
			Name:     name,
			Level:    1,
		})
		folderID++
	}

	// Generate level-2 folders (children of level-1)
	for i := 0; i < NumLevel2Folders; i++ {
		parentIdx := level1Start + rng.Intn(NumLevel1Folders)
		parent := folders[parentIdx]
		name := fmt.Sprintf("Project-%d", i+1)
		path := parent.Path + "/" + name
		parentID := parent.ID
		folders = append(folders, Folder{
			ID:       folderID,
			Path:     path,
			ParentID: &parentID,
			Name:     name,
			Level:    2,
		})
		folderID++
	}

	// Insert folders into database
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO folders (id, path, parent_id, name, user_id, display_order, color)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	colors := []string{"#3498DB", "#E74C3C", "#2ECC71", "#9B59B6", "#F39C12", "#1ABC9C"}
	for i, f := range folders {
		var color interface{} = nil
		if rng.Float32() < 0.3 { // 30% have colors
			color = colors[rng.Intn(len(colors))]
		}
		_, err := stmt.Exec(f.ID, f.Path, f.ParentID, f.Name, userID, i, color)
		if err != nil {
			return nil, fmt.Errorf("insert folder %s: %w", f.Path, err)
		}
	}

	return folders, tx.Commit()
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateDeterministicUUID(rng *mrand.Rand) string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(rng.Intn(256))
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateContent(rng *mrand.Rand, size string) string {
	var minLen, maxLen int
	switch size {
	case "small":
		minLen, maxLen = 100, 500
	case "medium":
		minLen, maxLen = 500, 2000
	case "large":
		minLen, maxLen = 2000, 5000
	default:
		minLen, maxLen = 500, 2000
	}

	length := minLen + rng.Intn(maxLen-minLen)

	words := []string{
		"the", "a", "is", "are", "was", "were", "has", "have", "had", "be", "been", "being",
		"do", "does", "did", "will", "would", "could", "should", "may", "might", "must",
		"note", "document", "content", "text", "paragraph", "section", "chapter", "article",
		"idea", "concept", "thought", "theory", "principle", "method", "approach", "technique",
		"project", "task", "work", "goal", "objective", "target", "milestone", "deadline",
		"meeting", "discussion", "conversation", "presentation", "review", "feedback", "comment",
		"important", "critical", "essential", "significant", "relevant", "useful", "helpful",
		"development", "implementation", "testing", "deployment", "maintenance", "optimization",
		"performance", "efficiency", "quality", "reliability", "security", "scalability",
		"database", "server", "client", "API", "interface", "protocol", "network", "system",
		"user", "customer", "team", "manager", "developer", "designer", "analyst", "engineer",
	}

	var sb strings.Builder
	currentLen := 0
	sentenceLen := 0

	for currentLen < length {
		word := words[rng.Intn(len(words))]
		if sentenceLen == 0 {
			word = strings.ToUpper(word[:1]) + word[1:]
		}
		sb.WriteString(word)
		currentLen += len(word)
		sentenceLen++

		if sentenceLen > 5+rng.Intn(10) {
			sb.WriteString(". ")
			sentenceLen = 0
			if rng.Float32() < 0.2 {
				sb.WriteString("\n\n")
			}
		} else {
			sb.WriteString(" ")
		}
		currentLen++
	}

	if sentenceLen > 0 {
		sb.WriteString(".")
	}

	return sb.String()
}

func generateNotes(db *sql.DB, rng *mrand.Rand, userID int64, folders []Folder) ([]Note, error) {
	notes := make([]Note, 0, NumNotes)

	// Size distribution: 20% small, 60% medium, 20% large
	smallCount := NumNotes * 20 / 100
	mediumCount := NumNotes * 60 / 100
	// largeCount is implicitly NumNotes - smallCount - mediumCount

	// Encrypted count (10%)
	encryptedCount := NumNotes * EncryptedPercentage / 100

	// Topic prefixes for realistic titles
	topics := []string{
		"Meeting Notes:", "Project Plan:", "Research:", "Ideas:", "TODO:",
		"Documentation:", "Analysis:", "Review:", "Summary:", "Guide:",
		"Tutorial:", "Reference:", "Notes on", "Thoughts about", "Plan for",
		"Draft:", "Outline:", "Discussion:", "Memo:", "Report:",
	}

	subjects := []string{
		"Architecture", "Performance", "Security", "Testing", "Deployment",
		"Database", "API Design", "Frontend", "Backend", "Infrastructure",
		"User Experience", "Authentication", "Authorization", "Caching", "Logging",
		"Monitoring", "Scaling", "Optimization", "Refactoring", "Migration",
		"Integration", "Configuration", "Documentation", "Code Review", "Sprint",
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id,
			content_encrypted, encrypted_content, encryption_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	// Power-law distribution for folder assignment (few folders get many notes)
	folderWeights := make([]float64, len(folders))
	for i := range folderWeights {
		folderWeights[i] = math.Pow(float64(len(folders)-i), 1.5)
	}
	totalWeight := 0.0
	for _, w := range folderWeights {
		totalWeight += w
	}

	pickFolder := func() *Folder {
		r := rng.Float64() * totalWeight
		cumulative := 0.0
		for i, w := range folderWeights {
			cumulative += w
			if r <= cumulative {
				return &folders[i]
			}
		}
		return &folders[len(folders)-1]
	}

	for i := 0; i < NumNotes; i++ {
		id := generateDeterministicUUID(rng)
		topic := topics[rng.Intn(len(topics))]
		subject := subjects[rng.Intn(len(subjects))]
		title := fmt.Sprintf("%s %s %d", topic, subject, i+1)
		titleNorm := strings.ToLower(strings.TrimSpace(title))

		var size string
		if i < smallCount {
			size = "small"
		} else if i < smallCount+mediumCount {
			size = "medium"
		} else {
			size = "large"
		}

		content := generateContent(rng, size)
		folder := pickFolder()
		encrypted := i < encryptedCount

		var encryptedContent interface{} = nil
		var contentEncrypted int = 0
		var encryptionVersion int = 0

		if encrypted {
			// Simulate encrypted content with random bytes
			encBytes := make([]byte, len(content))
			for j := range encBytes {
				encBytes[j] = byte(rng.Intn(256))
			}
			encryptedContent = encBytes
			contentEncrypted = 1
			encryptionVersion = 1
			content = "[encrypted]" // Placeholder for encrypted notes
		}

		_, err := stmt.Exec(id, title, titleNorm, content, folder.Path, userID,
			contentEncrypted, encryptedContent, encryptionVersion)
		if err != nil {
			return nil, fmt.Errorf("insert note %d: %w", i, err)
		}

		notes = append(notes, Note{
			ID:        id,
			Title:     title,
			TitleNorm: titleNorm,
			Content:   content,
			FolderID:  folder.ID,
			Encrypted: encrypted,
		})
	}

	return notes, tx.Commit()
}

func generateLinks(db *sql.DB, rng *mrand.Rand, notes []Note) (int, int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	resolvedStmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO links (source_id, target_id) VALUES (?, ?)
	`)
	if err != nil {
		return 0, 0, err
	}
	defer resolvedStmt.Close()

	unresolvedStmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO unresolved_links (source_id, target_ref, target_ref_norm) VALUES (?, ?, ?)
	`)
	if err != nil {
		return 0, 0, err
	}
	defer unresolvedStmt.Close()

	resolvedCount := NumLinks * ResolvedLinksPct / 100
	unresolvedCount := NumLinks - resolvedCount

	// Track existing links to avoid duplicates
	existingLinks := make(map[string]bool)

	// Select hub nodes (5 notes that will have many backlinks)
	hubIndices := make(map[int]bool)
	for len(hubIndices) < NumHubNodes {
		hubIndices[rng.Intn(len(notes))] = true
	}

	resolvedInserted := 0
	unresolvedInserted := 0

	// Generate hub backlinks first
	for hubIdx := range hubIndices {
		hubNote := notes[hubIdx]
		for j := 0; j < HubBacklinks && resolvedInserted < resolvedCount; j++ {
			sourceIdx := rng.Intn(len(notes))
			if sourceIdx == hubIdx {
				continue
			}
			sourceNote := notes[sourceIdx]
			key := sourceNote.ID + "->" + hubNote.ID
			if existingLinks[key] {
				continue
			}
			existingLinks[key] = true
			if _, err := resolvedStmt.Exec(sourceNote.ID, hubNote.ID); err == nil {
				resolvedInserted++
			}
		}
	}

	// Generate remaining resolved links
	for resolvedInserted < resolvedCount {
		sourceIdx := rng.Intn(len(notes))
		targetIdx := rng.Intn(len(notes))
		if sourceIdx == targetIdx {
			continue
		}
		sourceNote := notes[sourceIdx]
		targetNote := notes[targetIdx]
		key := sourceNote.ID + "->" + targetNote.ID
		if existingLinks[key] {
			continue
		}
		existingLinks[key] = true
		if _, err := resolvedStmt.Exec(sourceNote.ID, targetNote.ID); err == nil {
			resolvedInserted++
		}
	}

	// Generate cyclic links (A->B->C->A) - about 5% of resolved links
	cyclicCount := resolvedCount * 5 / 100
	for i := 0; i < cyclicCount/3; i++ {
		// Pick 3 random notes for a cycle
		a := rng.Intn(len(notes))
		b := rng.Intn(len(notes))
		c := rng.Intn(len(notes))
		if a == b || b == c || a == c {
			continue
		}

		// Insert cycle links (if not already exists)
		links := [][2]int{{a, b}, {b, c}, {c, a}}
		for _, link := range links {
			key := notes[link[0]].ID + "->" + notes[link[1]].ID
			if !existingLinks[key] {
				existingLinks[key] = true
				resolvedStmt.Exec(notes[link[0]].ID, notes[link[1]].ID)
			}
		}
	}

	// Generate unresolved links
	unresolvedRefs := []string{
		"Future Project", "Upcoming Feature", "Planned Integration",
		"TODO Implementation", "Draft Document", "Pending Review",
		"Work in Progress", "Next Steps", "Follow-up Required",
	}

	for unresolvedInserted < unresolvedCount {
		sourceIdx := rng.Intn(len(notes))
		sourceNote := notes[sourceIdx]
		targetRef := fmt.Sprintf("%s %d", unresolvedRefs[rng.Intn(len(unresolvedRefs))], rng.Intn(100))
		targetRefNorm := strings.ToLower(strings.TrimSpace(targetRef))

		key := sourceNote.ID + "->" + targetRef
		if existingLinks[key] {
			continue
		}
		existingLinks[key] = true
		if _, err := unresolvedStmt.Exec(sourceNote.ID, targetRef, targetRefNorm); err == nil {
			unresolvedInserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return resolvedInserted, unresolvedInserted, nil
}

func generateTags(db *sql.DB, rng *mrand.Rand, userID int64, notes []Note) (int, int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	tagStmt, err := tx.Prepare(`
		INSERT INTO tags (id, name, name_norm, user_id) VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return 0, 0, err
	}
	defer tagStmt.Close()

	assignStmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)
	`)
	if err != nil {
		return 0, 0, err
	}
	defer assignStmt.Close()

	// Generate tags
	tagPrefixes := []string{
		"project", "status", "priority", "type", "category",
		"team", "sprint", "release", "feature", "bug",
	}
	tagSuffixes := []string{
		"active", "pending", "completed", "review", "urgent",
		"low", "medium", "high", "critical", "blocked",
	}

	tags := make([]int, NumTags)
	for i := 0; i < NumTags; i++ {
		name := fmt.Sprintf("%s-%s-%d",
			tagPrefixes[rng.Intn(len(tagPrefixes))],
			tagSuffixes[rng.Intn(len(tagSuffixes))],
			i+1)
		nameNorm := strings.ToLower(name)
		if _, err := tagStmt.Exec(i+1, name, nameNorm, userID); err != nil {
			return 0, 0, fmt.Errorf("insert tag %d: %w", i, err)
		}
		tags[i] = i + 1
	}

	// Assign tags to notes (0-5 tags per note)
	assigned := 0
	for assigned < NumTagAssignments {
		noteIdx := rng.Intn(len(notes))
		tagID := tags[rng.Intn(len(tags))]
		if _, err := assignStmt.Exec(notes[noteIdx].ID, tagID); err == nil {
			assigned++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return NumTags, assigned, nil
}

func randomHexString(rng *mrand.Rand, length int) string {
	b := make([]byte, length/2)
	for i := range b {
		b[i] = byte(rng.Intn(256))
	}
	return hex.EncodeToString(b)
}
