package ledger

import (
	"database/sql"
	"embed"
	"errors"
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
	if errors.Is(err, sql.ErrNoRows) {
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

	for _, entry := range entries {
		name := entry.Name()
		// Parse migration number from leading 4-digit prefix (e.g. "0001_initial.sql" → 1).
		if len(name) < 4 {
			return fmt.Errorf("migration filename %q does not start with a 4-digit number", name)
		}
		var migrationNum int
		for _, ch := range name[:4] {
			if ch < '0' || ch > '9' {
				return fmt.Errorf("migration filename %q does not start with a 4-digit number", name)
			}
			migrationNum = migrationNum*10 + int(ch-'0')
		}
		if migrationNum <= currentVersion {
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		// Apply migration and update schema_version atomically.
		tx, err := l.db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration tx %s: %w", name, err)
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			tx.Rollback()
			return fmt.Errorf("run migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`UPDATE metadata SET value = ? WHERE key = 'schema_version'`, migrationNum); err != nil {
			tx.Rollback()
			return fmt.Errorf("update schema_version after %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}
