package session

import (
	"fmt"
)

// BackendConfig holds configuration for creating a backend
type BackendConfig struct {
	// Backend specifies which backend to use: "auto", "pty", or "tmux"
	Backend string
	// Prefix is used for naming sessions (e.g., "planck")
	Prefix string
	// SessionsDir is the directory for session-related files
	SessionsDir string
	// WorkDir is the working directory (used for tmux session isolation)
	WorkDir string
	// ExtraArgs are additional CLI arguments passed to the agent command
	ExtraArgs []string
	// TmuxFactory creates a tmux backend (injected to avoid import cycle)
	TmuxFactory func(prefix, sessionsDir, workDir string, extraArgs []string) InteractiveBackend
}

// NewBackend creates a session backend based on configuration.
// Returns an InteractiveBackend that supports PTY-like interactive sessions.
func NewBackend(cfg BackendConfig) (InteractiveBackend, error) {
	switch cfg.Backend {
	case "tmux":
		if cfg.TmuxFactory == nil {
			return nil, fmt.Errorf("tmux backend requested but no factory provided")
		}
		backend := cfg.TmuxFactory(cfg.Prefix, cfg.SessionsDir, cfg.WorkDir, cfg.ExtraArgs)
		if !backend.IsAvailable() {
			return nil, fmt.Errorf("tmux backend not available: tmux not found in PATH")
		}
		return backend, nil

	case "pty":
		backend := NewPTYBackend(cfg.Prefix, cfg.SessionsDir, cfg.ExtraArgs)
		if !backend.IsAvailable() {
			return nil, fmt.Errorf("PTY backend not available: claude CLI not found")
		}
		return backend, nil

	case "auto", "":
		// Prefer tmux if available, fall back to PTY
		if cfg.TmuxFactory != nil {
			tmuxBackend := cfg.TmuxFactory(cfg.Prefix, cfg.SessionsDir, cfg.WorkDir, cfg.ExtraArgs)
			if tmuxBackend.IsAvailable() {
				return tmuxBackend, nil
			}
		}
		backend := NewPTYBackend(cfg.Prefix, cfg.SessionsDir, cfg.ExtraArgs)
		if !backend.IsAvailable() {
			return nil, fmt.Errorf("no session backend available: neither tmux nor claude CLI found")
		}
		return backend, nil

	default:
		return nil, fmt.Errorf("unknown session backend: %s (supported: auto, pty, tmux)", cfg.Backend)
	}
}
