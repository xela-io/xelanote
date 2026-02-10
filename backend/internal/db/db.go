package db

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"sync"

	sqlite3 "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the SQLite database connection with xelanote-specific methods.
type DB struct {
	*sql.DB
}

// Tx wraps a database transaction with xelanote-specific methods.
// It provides the same interface as DB for transaction-aware operations.
type Tx struct {
	*sql.Tx
}

// BeginTx starts a new transaction and returns a Tx wrapper.
// The caller must call Commit() or Rollback() on the returned Tx.
func (db *DB) BeginTx() (*Tx, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &Tx{Tx: tx}, nil
}

const encryptedDriverName = "sqlite3_xelanote_encrypted"

var registerEncryptedDriverOnce sync.Once

// Open creates a new database connection with DELETE journal mode for Docker compatibility.
// path can be ":memory:" for in-memory database or a file path.
func Open(path, key string) (*DB, error) {
	// Enable foreign keys and DELETE mode via connection string
	// DELETE mode is more stable in Docker environments vs WAL mode
	driverName := "sqlite3"
	if key != "" {
		registerEncryptedDriver(key)
		driverName = encryptedDriverName
	}

	db, err := sql.Open(driverName, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	// SQLite performs best with a single connection and keeps per-connection pragmas consistent.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Explicitly enable foreign keys and set pragmas
	// Connection string params are not always reliable, so set them explicitly
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = DELETE",
		"PRAGMA synchronous = FULL",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// Migrate applies the database schema and runs migrations.
func (db *DB) Migrate() error {
	// Check if this is a new database by looking for the notes table
	var tableCount int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='notes'").Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("failed to check notes table: %w", err)
	}

	isNewDB := tableCount == 0

	if isNewDB {
		// New database: apply base schema (notes table with FTS5)
		schema, err := schemaFS.ReadFile("schema.sql")
		if err != nil {
			return fmt.Errorf("failed to read schema: %w", err)
		}

		_, err = db.Exec(string(schema))
		if err != nil {
			return fmt.Errorf("failed to apply schema: %w", err)
		}
	}

	// Run migrations for both new and existing databases
	// For new databases, this builds on top of schema.sql
	// For existing databases, this applies only pending migrations
	if err := db.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// initMigrations creates the migrations tracking table.
func (db *DB) initMigrations() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			applied_at TEXT DEFAULT (datetime('now'))
		)
	`)
	return err
}

// runMigrations executes all pending migrations.
func (db *DB) runMigrations() error {
	if err := db.initMigrations(); err != nil {
		return fmt.Errorf("failed to init migrations table: %w", err)
	}

	migrations := []string{
		"002_folders_table.sql",
		"003_add_order_fields.sql",
		"004_add_users.sql",
		"005_add_user_ownership.sql",
		"006_refresh_token_hash.sql",
		"007_add_deleted_at.sql",
		"008_fix_deleted_at_null.sql",
		"009_fix_fts_delete_trigger.sql",
		"010_composite_unique_indexes.sql",
		"011_note_versions.sql",
		"012_add_user_to_tags.sql",
		"013_create_templates.sql",
		"014_create_snippets.sql",
		"015_user_preferences.sql",
		"016_add_admin_flag.sql",
		"017_activity_logs.sql",
		"018_system_settings.sql",
		"019_add_two_factor_auth.sql",
		"020_e2e_encryption.sql",
		"021_encryption_preferences.sql",
		"022_kek_persistence.sql",
		"023_add_color_field.sql",
		"024_folder_scoped_unique_title.sql",
		"025_virtual_root.sql",
		"026_notes_order_index.sql",
		"027_graph_indexes.sql",
		"028_note_summaries.sql",
		"029_note_types_and_features.sql",
		"030_ai_enabled_flags.sql",
		"031_claude_api_key.sql",
		"032_gemini_api_key.sql",
		"033_fido2_credentials.sql",
		"034_note_sharing.sql",
		"035_encryption_toggle.sql",
		"036_shared_note_placements.sql",
		"037_recipes.sql",
		"038_recipe_collection_shares.sql",
		"039_recipe_images.sql",
		"040_task_events.sql",
		"041_fix_keyword_fts_triggers.sql",
		"042_note_due_dates.sql",
		"043_enable_journal_recipe_default.sql",
	}

	for _, migrationFile := range migrations {
		// Check if already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", migrationFile).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", migrationFile, err)
		}

		if count > 0 {
			// Already applied, skip
			continue
		}

		// Read migration content
		content, err := migrationsFS.ReadFile("migrations/" + migrationFile)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migrationFile, err)
		}

		// Execute migration
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("migration %s failed: %w", migrationFile, err)
		}

		// Mark as applied
		_, err = db.Exec("INSERT INTO migrations (name) VALUES (?)", migrationFile)
		if err != nil {
			return fmt.Errorf("failed to mark migration %s as applied: %w", migrationFile, err)
		}

		fmt.Printf("Applied migration: %s\n", migrationFile)
	}

	return nil
}

func registerEncryptedDriver(key string) {
	registerEncryptedDriverOnce.Do(func() {
		sql.Register(encryptedDriverName, &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				pragma := fmt.Sprintf("PRAGMA key = '%s'", escapeSQLCipherKey(key))
				_, err := conn.Exec(pragma, nil)
				return err
			},
		})
	})
}

func escapeSQLCipherKey(key string) string {
	return strings.ReplaceAll(key, "'", "''")
}
