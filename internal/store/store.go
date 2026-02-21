package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages persistent state in SQLite
type Store struct {
	db   *sql.DB
	path string
}

// Session represents a session record in the database
type Session struct {
	ID            string
	FilePath      string
	Status        string // running | completed | failed | cancelled
	StartedAt     time.Time
	EndedAt       *time.Time
	ExitCode      *int
}

// Open opens or creates the store database
func Open(path string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	store := &Store{db: db, path: path}

	// Run migrations
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate runs database migrations
func (s *Store) migrate() error {
	// Check if we have an old schema that needs migration
	if err := s.migrateFromOldSchema(); err != nil {
		return fmt.Errorf("migrate from old schema: %w", err)
	}

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			file_path TEXT NOT NULL,
			status TEXT DEFAULT 'running',
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			ended_at DATETIME,
			exit_code INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_file_path ON sessions(file_path)`,
	}

	for _, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			return fmt.Errorf("execute migration: %w", err)
		}
	}

	return nil
}

// migrateFromOldSchema handles migration from the old planck schema
func (s *Store) migrateFromOldSchema() error {
	// Check if sessions table exists with old schema (has plan_id column)
	hasOldSchema := false
	rows, err := s.db.Query("PRAGMA table_info(sessions)")
	if err != nil {
		// Table doesn't exist, that's fine
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			continue
		}
		if name == "plan_id" || name == "node_path" {
			hasOldSchema = true
			break
		}
	}

	if !hasOldSchema {
		return nil
	}

	// Drop old tables and let the new schema be created
	oldTables := []string{
		"DROP TABLE IF EXISTS execution_runs",
		"DROP TABLE IF EXISTS decisions",
		"DROP TABLE IF EXISTS sessions",
		"DROP TABLE IF EXISTS plan_meta",
	}

	for _, stmt := range oldTables {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("drop old table: %w", err)
		}
	}

	return nil
}

// Session operations

// SaveSession saves or updates a session
func (s *Store) SaveSession(session *Session) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (id, file_path, status, started_at, ended_at, exit_code)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			ended_at = excluded.ended_at,
			exit_code = excluded.exit_code
	`, session.ID, session.FilePath, session.Status, session.StartedAt, session.EndedAt, session.ExitCode)
	return err
}

// GetSession retrieves a session by ID
func (s *Store) GetSession(id string) (*Session, error) {
	var session Session
	err := s.db.QueryRow(`
		SELECT id, file_path, status, started_at, ended_at, exit_code
		FROM sessions WHERE id = ?
	`, id).Scan(&session.ID, &session.FilePath, &session.Status, &session.StartedAt, &session.EndedAt, &session.ExitCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListSessions returns sessions, optionally filtered by status
func (s *Store) ListSessions(status string) ([]*Session, error) {
	query := "SELECT id, file_path, status, started_at, ended_at, exit_code FROM sessions WHERE 1=1"
	args := []interface{}{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY started_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		if err := rows.Scan(&session.ID, &session.FilePath, &session.Status, &session.StartedAt, &session.EndedAt, &session.ExitCode); err != nil {
			return nil, err
		}
		sessions = append(sessions, &session)
	}
	return sessions, rows.Err()
}

// ListActiveSessions returns all running sessions
func (s *Store) ListActiveSessions() ([]*Session, error) {
	return s.ListSessions("running")
}

// UpdateSessionStatus updates a session's status
func (s *Store) UpdateSessionStatus(id, status string, exitCode *int) error {
	var endedAt interface{}
	if status == "completed" || status == "failed" || status == "cancelled" {
		endedAt = time.Now()
	}
	_, err := s.db.Exec(`
		UPDATE sessions SET status = ?, ended_at = ?, exit_code = ? WHERE id = ?
	`, status, endedAt, exitCode, id)
	return err
}

// DeleteSession deletes a session
func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// CleanupOldSessions removes sessions older than the given duration
func (s *Store) CleanupOldSessions(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := s.db.Exec(`
		DELETE FROM sessions WHERE status != 'running' AND started_at < ?
	`, cutoff)
	return err
}
