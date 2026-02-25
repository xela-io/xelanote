package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

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
// The context is used for cancellation/timeout propagation.
// The caller must call Commit() or Rollback() on the returned Tx.
func (db *DB) BeginTx(ctx context.Context) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &Tx{Tx: tx}, nil
}

// BeginImmediate starts a transaction with BEGIN IMMEDIATE semantics.
// With MaxOpenConns(1), this is functionally equivalent to Begin() since all
// operations are already serialized. This is defense-in-depth for future
// architecture changes (F2-03, F2-07).
func (db *DB) BeginImmediate() (*sql.Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin immediate transaction: %w", err)
	}
	return tx, nil
}

const encryptedDriverName = "sqlite3_xelanote_encrypted"

var registerEncryptedDriverOnce sync.Once

// OpenOptions configures the SQLite connection.
type OpenOptions struct {
	// JournalMode sets the SQLite journal mode.
	// "wal" (default) enables WAL mode with synchronous=NORMAL for better
	// concurrent read performance and lower write latency.
	// "delete" uses DELETE mode with synchronous=FULL for maximum compatibility
	// (e.g. network-mounted or problematic Docker volumes).
	// Configurable via XELANOTE_JOURNAL_MODE environment variable.
	JournalMode string
}

// Open creates a new database connection with configurable journal mode.
// WAL mode is the default for better performance; DELETE mode is available
// as a fallback for environments where WAL is not supported.
// path can be ":memory:" for in-memory database or a file path.
func Open(path, key string, opts ...OpenOptions) (*DB, error) {
	journalMode := "WAL"
	syncMode := "NORMAL"
	if len(opts) > 0 && strings.EqualFold(opts[0].JournalMode, "delete") {
		journalMode = "DELETE"
		syncMode = "FULL"
	}

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
	// SECURITY: MaxOpenConns(1) serializes all DB operations, which mitigates several
	// race conditions (F2-01: admin registration, F2-03: 2FA state transitions).
	// Do NOT increase without adding explicit transaction-level locking to those code paths.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Set pragmas explicitly (connection string params are not always reliable).
	// busy_timeout must be set before journal_mode to avoid SQLITE_BUSY during WAL switch.
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		fmt.Sprintf("PRAGMA journal_mode = %s", journalMode),
		fmt.Sprintf("PRAGMA synchronous = %s", syncMode),
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	// Verify journal mode was set correctly (some filesystems don't support WAL)
	var actualMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&actualMode); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to verify journal_mode: %w", err)
	}
	if !strings.EqualFold(actualMode, journalMode) {
		slog.Warn("journal mode mismatch — filesystem may not support WAL",
			slog.String("requested", journalMode),
			slog.String("actual", actualMode))
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// Optimize runs PRAGMA optimize to update SQLite query planner statistics.
// Should be called at startup (after migrations), periodically, and before shutdown.
func (db *DB) Optimize() {
	if _, err := db.Exec("PRAGMA optimize"); err != nil {
		slog.Warn("PRAGMA optimize failed", slog.String("error", err.Error()))
	}
}

// StartOptimizeScheduler runs PRAGMA optimize and maintenance tasks at the given interval (e.g. 24h).
// Returns a cancel function to stop the scheduler.
func (db *DB) StartOptimizeScheduler(interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				db.Optimize()
				if cleaned, err := db.CleanupExpiredRefreshTokens(); err != nil {
					slog.Warn("scheduled refresh token cleanup failed", slog.String("error", err.Error()))
				} else if cleaned > 0 {
					slog.Info("scheduled cleanup: removed expired/revoked refresh tokens", slog.Int64("count", cleaned))
				}
			}
		}
	}()
	return cancel
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
		"044_refresh_token_reuse_detection.sql",
		"045_registration_secure_default.sql",
		"046_perf_metrics.sql",
		"047_analytics_events.sql",
		"048_canvas_support.sql",
		"049_chatgpt_and_ai_provider.sql",
		"050_ai_model_preferences.sql",
		"051_dietary_preference.sql",
		"052_fix_fts_update_trigger.sql",
		"053_unique_admin_constraint.sql",
		"054_account_lockouts.sql",
		"055_home_dashboard_layout.sql",
		"056_note_user_state.sql",
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

		// Execute migration (transactionally if possible)
		if err := db.executeMigration(migrationFile, string(content)); err != nil {
			return err
		}

		fmt.Printf("Applied migration: %s\n", migrationFile)
	}

	return nil
}

// executeMigration runs a single migration and marks it as applied.
// Self-transactional migrations (containing their own BEGIN TRANSACTION) are
// executed directly. All other migrations are wrapped in a transaction together
// with the "mark as applied" INSERT for atomicity.
func (db *DB) executeMigration(name, content string) error {
	if isSelfTransactional(content) {
		// Migration manages its own transaction — execute unwrapped
		if _, err := db.Exec(content); err != nil {
			return fmt.Errorf("migration %s failed: %w", name, err)
		}
		if _, err := db.Exec("INSERT INTO migrations (name) VALUES (?)", name); err != nil {
			return fmt.Errorf("failed to mark migration %s as applied: %w", name, err)
		}
		return nil
	}

	// Wrap migration + mark-as-applied in a single transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for migration %s: %w", name, err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(content); err != nil {
		return fmt.Errorf("migration %s failed: %w", name, err)
	}
	if _, err := tx.Exec("INSERT INTO migrations (name) VALUES (?)", name); err != nil {
		return fmt.Errorf("failed to mark migration %s as applied: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", name, err)
	}
	return nil
}

// isSelfTransactional returns true if the SQL content contains its own
// BEGIN TRANSACTION / BEGIN statement (not inside a trigger definition).
// This detects migrations like 025_virtual_root.sql that manage their own
// transaction boundaries.
func isSelfTransactional(sql string) bool {
	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(strings.ToUpper(line))
		if trimmed == "BEGIN TRANSACTION;" || trimmed == "BEGIN;" {
			return true
		}
	}
	return false
}

func registerEncryptedDriver(key string) {
	registerEncryptedDriverOnce.Do(func() {
		sql.Register(encryptedDriverName, &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				pragma := "PRAGMA key = '" + escapeSQLCipherKey(key) + "'"
				_, err := conn.Exec(pragma, nil)
				return err
			},
		})
	})
}

func escapeSQLCipherKey(key string) string {
	return strings.ReplaceAll(key, "'", "''")
}
