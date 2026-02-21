package session

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/anthropics/planck/internal/store"
)

// Manager manages all sessions
type Manager struct {
	backend Backend
	store   *store.Store

	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewManager creates a new session manager
func NewManager(backend Backend, st *store.Store) *Manager {
	return &Manager{
		backend:  backend,
		store:    st,
		sessions: make(map[string]*Session),
	}
}

// Launch starts a new session
func (m *Manager) Launch(ctx context.Context, filePath, prompt string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Launch via backend
	session, err := m.backend.Launch(ctx, filePath, prompt)
	if err != nil {
		return nil, fmt.Errorf("launch session: %w", err)
	}

	// Fill in session details
	session.ID = uuid.New().String()
	session.TaskID = filePath
	session.Mode = ModeForeground
	session.Status = StatusRunning
	session.Backend = m.backend.Name()
	session.StartedAt = time.Now()

	// Store in memory
	m.sessions[session.ID] = session

	// Persist to database
	dbSession := &store.Session{
		ID:        session.ID,
		FilePath:  filePath,
		Status:    string(session.Status),
		StartedAt: session.StartedAt,
	}
	if err := m.store.SaveSession(dbSession); err != nil {
		// Log error but don't fail
	}

	return session, nil
}

// Attach attaches to a session
func (m *Manager) Attach(sessionID string) error {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return m.backend.Attach(session.BackendHandle)
}

// Detach detaches from a session
func (m *Manager) Detach(sessionID string) error {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return m.backend.Detach(session.BackendHandle)
}

// Capture returns recent output from a session
func (m *Manager) Capture(sessionID string, lines int) (string, error) {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}

	return m.backend.Capture(session.BackendHandle, lines)
}

// Kill terminates a session
func (m *Manager) Kill(sessionID string) error {
	m.mu.Lock()
	session, exists := m.sessions[sessionID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}

	delete(m.sessions, sessionID)
	m.mu.Unlock()

	// Kill via backend
	if err := m.backend.Kill(session.BackendHandle); err != nil {
		return err
	}

	// Update database
	return m.store.UpdateSessionStatus(sessionID, string(StatusCancelled), nil)
}

// Get returns a session by ID
func (m *Manager) Get(sessionID string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionID]
}

// List returns all active sessions
func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// UpdateStatus updates a session's status
func (m *Manager) UpdateStatus(sessionID string, status Status) error {
	m.mu.Lock()
	session, exists := m.sessions[sessionID]
	if exists {
		session.Status = status
		if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
			now := time.Now()
			session.EndedAt = &now
		}
	}
	m.mu.Unlock()

	return m.store.UpdateSessionStatus(sessionID, string(status), nil)
}

// Status returns the backend status of a session
func (m *Manager) Status(sessionID string) (Status, error) {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()
	if !exists {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}
	return m.backend.Status(session.BackendHandle)
}

// AttachCmd returns an *exec.Cmd for interactive attachment
func (m *Manager) AttachCmd(sessionID string) (*exec.Cmd, error) {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return m.backend.AttachCmd(session.BackendHandle)
}

// IsAvailable returns whether the backend is available
func (m *Manager) IsAvailable() bool {
	return m.backend.IsAvailable()
}
