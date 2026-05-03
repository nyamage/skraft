package ledger

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Ledger wraps the SQLite database.
type Ledger struct {
	db *sql.DB
}

// UploadState records the last upload to a target environment.
type UploadState struct {
	SkillName   string
	Target      string // "claudeai" | "claude_code"
	Version     string
	ContentHash string
	UploadedAt  time.Time
}

// Open opens (or creates) the SQLite ledger at path and runs pending migrations.
// Use ":memory:" for tests.
func Open(path string) (*Ledger, error) {
	dsn := path
	if path != ":memory:" {
		dsn = "file:" + path
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open ledger %s: %w", path, err)
	}
	// Enable WAL for concurrent access safety (no-op for :memory:)
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	l := &Ledger{db: db}
	if err := l.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return l, nil
}

// Close closes the database connection.
func (l *Ledger) Close() error {
	return l.db.Close()
}

// GetUploadState returns the last recorded upload state for a skill+target pair.
// Returns nil (not an error) if no record exists.
func (l *Ledger) GetUploadState(skillName, target string) (*UploadState, error) {
	row := l.db.QueryRow(`
		SELECT skill_name, target, version, content_hash, uploaded_at
		FROM upload_state
		WHERE skill_name = ? AND target = ?`, skillName, target)

	var s UploadState
	var uploadedAt string
	err := row.Scan(&s.SkillName, &s.Target, &s.Version, &s.ContentHash, &uploadedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.UploadedAt, err = time.Parse(time.RFC3339, uploadedAt)
	if err != nil {
		return nil, fmt.Errorf("parse uploaded_at: %w", err)
	}
	return &s, nil
}

// SetUploadState inserts or replaces the upload state for a skill+target pair.
func (l *Ledger) SetUploadState(state UploadState) error {
	_, err := l.db.Exec(`
		INSERT OR REPLACE INTO upload_state (skill_name, target, version, content_hash, uploaded_at)
		VALUES (?, ?, ?, ?, ?)`,
		state.SkillName,
		state.Target,
		state.Version,
		state.ContentHash,
		state.UploadedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// migrate runs any SQL migration files not yet applied.
func (l *Ledger) migrate() error {
	// Ensure metadata table exists (bootstrap for schema_version)
	if _, err := l.db.Exec(`CREATE TABLE IF NOT EXISTS metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create metadata table: %w", err)
	}
	if _, err := l.db.Exec(`INSERT OR IGNORE INTO metadata (key, value) VALUES ('schema_version', '0')`); err != nil {
		return fmt.Errorf("init schema_version: %w", err)
	}

	var currentVersion int
	if err := l.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM metadata WHERE key = 'schema_version'`).Scan(&currentVersion); err != nil {
		return fmt.Errorf("read schema_version: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	for i, entry := range entries {
		migrationNum := i + 1
		if migrationNum <= currentVersion {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if _, err := l.db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("run migration %s: %w", entry.Name(), err)
		}
		if _, err := l.db.Exec(`UPDATE metadata SET value = ? WHERE key = 'schema_version'`, migrationNum); err != nil {
			return fmt.Errorf("update schema_version: %w", err)
		}
	}
	return nil
}
