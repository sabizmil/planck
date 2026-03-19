package store

import (
	"database/sql"
	"encoding/json"
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
	ID        string
	FilePath  string
	Status    string // running | completed | failed | canceled
	StartedAt time.Time
	EndedAt   *time.Time
	ExitCode  *int

	// Extended fields for session recovery
	AgentKey        string // config key (e.g., "claude-code")
	AgentLabel      string // display label (e.g., "Claude")
	CustomTitle     string // title set by child process or user input
	TmuxSessionName string // tmux session name for recovery
	BackendType     string // "tmux" or "pty"
	WorkDir         string // working directory
	Command         string // command that was launched
	Args            string // JSON-encoded arguments
}

// Open opens or creates the store database
func Open(path string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
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

	// UI state key-value table for persisting ephemeral UI state (folder state, etc.)
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS ui_state (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create ui_state table: %w", err)
	}

	// Add new columns for session recovery (idempotent via IF NOT EXISTS check)
	newColumns := []struct {
		name string
		def  string
	}{
		{"agent_key", "TEXT DEFAULT ''"},
		{"agent_label", "TEXT DEFAULT ''"},
		{"custom_title", "TEXT DEFAULT ''"},
		{"tmux_session_name", "TEXT DEFAULT ''"},
		{"backend_type", "TEXT DEFAULT ''"},
		{"work_dir", "TEXT DEFAULT ''"},
		{"command", "TEXT DEFAULT ''"},
		{"args", "TEXT DEFAULT '[]'"},
	}

	for _, col := range newColumns {
		if !s.hasColumn("sessions", col.name) {
			if _, err := s.db.Exec(fmt.Sprintf("ALTER TABLE sessions ADD COLUMN %s %s", col.name, col.def)); err != nil {
				return fmt.Errorf("add column %s: %w", col.name, err)
			}
		}
	}

	return nil
}

// hasColumn checks if a table has a specific column
func (s *Store) hasColumn(table, column string) bool {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
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
		if name == column {
			return true
		}
	}
	return false
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
		INSERT INTO sessions (id, file_path, status, started_at, ended_at, exit_code,
			agent_key, agent_label, custom_title, tmux_session_name, backend_type, work_dir, command, args)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			ended_at = excluded.ended_at,
			exit_code = excluded.exit_code,
			custom_title = excluded.custom_title,
			tmux_session_name = excluded.tmux_session_name
	`, session.ID, session.FilePath, session.Status, session.StartedAt, session.EndedAt, session.ExitCode,
		session.AgentKey, session.AgentLabel, session.CustomTitle, session.TmuxSessionName,
		session.BackendType, session.WorkDir, session.Command, session.Args)
	return err
}

// GetSession retrieves a session by ID
func (s *Store) GetSession(id string) (*Session, error) {
	var session Session
	err := s.db.QueryRow(`
		SELECT id, file_path, status, started_at, ended_at, exit_code,
			COALESCE(agent_key, ''), COALESCE(agent_label, ''),
			COALESCE(custom_title, ''), COALESCE(tmux_session_name, ''),
			COALESCE(backend_type, ''), COALESCE(work_dir, ''),
			COALESCE(command, ''), COALESCE(args, '[]')
		FROM sessions WHERE id = ?
	`, id).Scan(&session.ID, &session.FilePath, &session.Status, &session.StartedAt, &session.EndedAt, &session.ExitCode,
		&session.AgentKey, &session.AgentLabel, &session.CustomTitle, &session.TmuxSessionName,
		&session.BackendType, &session.WorkDir, &session.Command, &session.Args)
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
	query := `SELECT id, file_path, status, started_at, ended_at, exit_code,
		COALESCE(agent_key, ''), COALESCE(agent_label, ''),
		COALESCE(custom_title, ''), COALESCE(tmux_session_name, ''),
		COALESCE(backend_type, ''), COALESCE(work_dir, ''),
		COALESCE(command, ''), COALESCE(args, '[]')
		FROM sessions WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY started_at ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		if err := rows.Scan(&session.ID, &session.FilePath, &session.Status, &session.StartedAt, &session.EndedAt, &session.ExitCode,
			&session.AgentKey, &session.AgentLabel, &session.CustomTitle, &session.TmuxSessionName,
			&session.BackendType, &session.WorkDir, &session.Command, &session.Args); err != nil {
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
	if status == "completed" || status == "failed" || status == "canceled" {
		endedAt = time.Now()
	}
	_, err := s.db.Exec(`
		UPDATE sessions SET status = ?, ended_at = ?, exit_code = ? WHERE id = ?
	`, status, endedAt, exitCode, id)
	return err
}

// UpdateSessionTitle updates just the custom title of a session
func (s *Store) UpdateSessionTitle(id, title string) error {
	_, err := s.db.Exec(`UPDATE sessions SET custom_title = ? WHERE id = ?`, title, id)
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

// EncodeArgs serializes arguments to JSON for storage.
func EncodeArgs(args []string) string {
	data, err := json.Marshal(args)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// DecodeArgs deserializes JSON arguments from storage.
func DecodeArgs(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var args []string
	if err := json.Unmarshal([]byte(s), &args); err != nil {
		return nil
	}
	return args
}

// UI State operations

// GetUIState retrieves a UI state value by key. Returns "" if not found.
func (s *Store) GetUIState(key string) string {
	var value string
	err := s.db.QueryRow("SELECT value FROM ui_state WHERE key = ?", key).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}

// SetUIState saves a UI state value by key.
func (s *Store) SetUIState(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO ui_state (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}
