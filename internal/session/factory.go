package session

import (
	"fmt"
)

// BackendConfig holds configuration for creating a backend
type BackendConfig struct {
	// Backend specifies which backend to use: "auto" or "pty"
	Backend string
	// Prefix is used for naming sessions (e.g., "planck")
	Prefix string
	// SessionsDir is the directory for session-related files
	SessionsDir string
	// ExtraArgs are additional CLI arguments passed to claude
	ExtraArgs []string
}

// NewBackend creates a session backend based on configuration
func NewBackend(cfg BackendConfig) (Backend, error) {
	switch cfg.Backend {
	case "pty", "auto", "":
		// Always use PTY backend
		backend := NewPTYBackend(cfg.Prefix, cfg.SessionsDir, cfg.ExtraArgs)
		if !backend.IsAvailable() {
			return nil, fmt.Errorf("PTY backend not available: claude CLI not found")
		}
		return backend, nil

	default:
		return nil, fmt.Errorf("unknown session backend: %s (only 'pty' is supported)", cfg.Backend)
	}
}
