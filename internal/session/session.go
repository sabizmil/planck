package session

import (
	"context"
	"os/exec"
	"time"
)

// Status represents session status
type Status string

const (
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Mode represents session mode
type Mode string

const (
	ModeForeground Mode = "foreground"
	ModeBackground Mode = "background"
)

// Type represents session type
type Type string

const (
	TypePlanning       Type = "planning"
	TypeImplementation Type = "implementation"
	TypeExecution      Type = "execution"
)

// Session represents an active session
type Session struct {
	ID             string
	TaskID         string
	PlanID         string
	Type           Type
	Mode           Mode
	Status         Status
	Backend        string // "tmux" | "pty" | "headless"
	AgentSessionID string
	BackendHandle  string // Opaque identifier for the backend (tmux session name, PTY ID, etc.)
	StartedAt      time.Time
	EndedAt        *time.Time
	Output         string
}

// Backend represents a session backend interface
type Backend interface {
	// Name returns the backend name
	Name() string

	// IsAvailable returns whether this backend can be used
	IsAvailable() bool

	// Launch starts a new session with the given prompt
	Launch(ctx context.Context, taskID string, prompt string) (*Session, error)

	// Attach gives the user full interactive control
	Attach(handle string) error

	// AttachCmd returns an *exec.Cmd for interactive attachment (used with tea.ExecProcess)
	AttachCmd(handle string) (*exec.Cmd, error)

	// Detach returns control from an attached session
	Detach(handle string) error

	// Capture returns recent output for preview
	Capture(handle string, lines int) (string, error)

	// Kill terminates the session
	Kill(handle string) error

	// List returns all active sessions
	List() ([]*Session, error)

	// Status returns the current status of a session
	Status(handle string) (Status, error)
}
