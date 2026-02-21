package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestOpen_NewDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	// Verify we can use the store
	session := &Session{
		ID:        "test-1",
		FilePath:  "/path/to/file.md",
		Status:    "running",
		StartedAt: time.Now(),
	}

	if err := store.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	got, err := store.GetSession("test-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetSession() returned nil")
	}
	if got.FilePath != session.FilePath {
		t.Errorf("FilePath = %v, want %v", got.FilePath, session.FilePath)
	}
}

func TestOpen_MigrateFromOldSchema(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create a database with the OLD schema
	if err := createOldSchemaDatabase(dbPath); err != nil {
		t.Fatalf("createOldSchemaDatabase() error = %v", err)
	}

	// Now open with the new store - should migrate successfully
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil (should migrate from old schema)", err)
	}
	defer store.Close()

	// Verify we can use the store with new schema
	session := &Session{
		ID:        "test-1",
		FilePath:  "/path/to/file.md",
		Status:    "running",
		StartedAt: time.Now(),
	}

	if err := store.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() after migration error = %v", err)
	}

	got, err := store.GetSession("test-1")
	if err != nil {
		t.Fatalf("GetSession() after migration error = %v", err)
	}
	if got == nil {
		t.Fatal("GetSession() returned nil after migration")
	}
	if got.FilePath != session.FilePath {
		t.Errorf("FilePath = %v, want %v", got.FilePath, session.FilePath)
	}
}

func TestOpen_MigrateFromOldSchemaWithData(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create a database with the OLD schema and some data
	if err := createOldSchemaDatabaseWithData(dbPath); err != nil {
		t.Fatalf("createOldSchemaDatabaseWithData() error = %v", err)
	}

	// Now open with the new store - should migrate successfully
	// Old data will be lost, but that's expected for this major schema change
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v, want nil (should migrate from old schema)", err)
	}
	defer store.Close()

	// Verify the new schema works
	sessions, err := store.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	// Old sessions should be gone after migration
	if len(sessions) != 0 {
		t.Errorf("ListSessions() returned %d sessions, want 0 (old data should be cleared)", len(sessions))
	}
}

func TestSessionOperations(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	// Test SaveSession
	session := &Session{
		ID:        "session-1",
		FilePath:  "/path/to/plan.md",
		Status:    "running",
		StartedAt: time.Now(),
	}

	if err := store.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Test GetSession
	got, err := store.GetSession("session-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.ID != session.ID {
		t.Errorf("ID = %v, want %v", got.ID, session.ID)
	}

	// Test ListSessions
	sessions, err := store.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("ListSessions() returned %d sessions, want 1", len(sessions))
	}

	// Test ListActiveSessions
	active, err := store.ListActiveSessions()
	if err != nil {
		t.Fatalf("ListActiveSessions() error = %v", err)
	}
	if len(active) != 1 {
		t.Errorf("ListActiveSessions() returned %d sessions, want 1", len(active))
	}

	// Test UpdateSessionStatus
	exitCode := 0
	if err := store.UpdateSessionStatus("session-1", "completed", &exitCode); err != nil {
		t.Fatalf("UpdateSessionStatus() error = %v", err)
	}

	got, _ = store.GetSession("session-1")
	if got.Status != "completed" {
		t.Errorf("Status = %v, want completed", got.Status)
	}

	// Active sessions should now be empty
	active, _ = store.ListActiveSessions()
	if len(active) != 0 {
		t.Errorf("ListActiveSessions() returned %d sessions, want 0", len(active))
	}

	// Test DeleteSession
	if err := store.DeleteSession("session-1"); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	got, _ = store.GetSession("session-1")
	if got != nil {
		t.Error("GetSession() returned session after delete, want nil")
	}
}

// createOldSchemaDatabase creates a database with the old planck schema
func createOldSchemaDatabase(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create old schema tables
	oldSchema := []string{
		`CREATE TABLE plan_meta (
			id TEXT PRIMARY KEY,
			dir_path TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			plan_id TEXT NOT NULL,
			node_path TEXT NOT NULL,
			agent TEXT NOT NULL,
			session_type TEXT NOT NULL,
			session_mode TEXT DEFAULT 'foreground',
			agent_session_id TEXT,
			backend_handle TEXT,
			status TEXT DEFAULT 'running',
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			ended_at DATETIME
		)`,
		`CREATE TABLE decisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_id TEXT NOT NULL,
			node_path TEXT NOT NULL,
			decision TEXT NOT NULL,
			reasoning TEXT,
			decided_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE execution_runs (
			id TEXT PRIMARY KEY,
			plan_id TEXT NOT NULL,
			scope TEXT NOT NULL,
			scope_path TEXT NOT NULL,
			total_tasks INTEGER NOT NULL,
			completed_tasks INTEGER DEFAULT 0,
			current_phase INTEGER,
			total_phases INTEGER,
			status TEXT DEFAULT 'running',
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			ended_at DATETIME
		)`,
	}

	for _, stmt := range oldSchema {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

// createOldSchemaDatabaseWithData creates a database with old schema and sample data
func createOldSchemaDatabaseWithData(path string) error {
	if err := createOldSchemaDatabase(path); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	// Insert some old data
	_, err = db.Exec(`
		INSERT INTO plan_meta (id, dir_path) VALUES ('plan-1', '/old/path')
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO sessions (id, plan_id, node_path, agent, session_type, status)
		VALUES ('old-session-1', 'plan-1', 'task-1', 'claude', 'planning', 'completed')
	`)
	if err != nil {
		return err
	}

	return nil
}
