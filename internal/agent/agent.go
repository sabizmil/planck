package agent

import (
	"context"
	"time"
)

// Agent represents an AI agent interface
type Agent interface {
	// Name returns the agent name
	Name() string

	// Run executes a prompt and returns the full output
	Run(ctx context.Context, prompt string) (string, error)

	// Stream executes a prompt and streams output
	Stream(ctx context.Context, prompt string) (<-chan StreamEvent, error)

	// Send sends input to a running session (for interactive responses)
	Send(input string) error

	// Stop stops the current operation
	Stop() error

	// SessionID returns the current session ID for resuming
	SessionID() string
}

// StreamEvent represents a streaming output event
type StreamEvent struct {
	Type       StreamEventType
	Text       string
	ToolUse    *ToolUse
	Permission *PermissionRequest
	Error      error
}

// StreamEventType represents the type of stream event
type StreamEventType int

const (
	EventText StreamEventType = iota
	EventToolUse
	EventToolResult
	EventComplete
	EventError
	EventThinking      // Claude's thinking/reasoning blocks
	EventPermission    // Permission request from Claude
	EventSystemMessage // System-level messages (auth, rate limit, etc.)
)

// ToolUseStatus represents the status of a tool use operation
type ToolUseStatus int

const (
	ToolUsePending ToolUseStatus = iota
	ToolUseRunning
	ToolUseComplete
	ToolUseFailed
)

// ToolUse represents a tool use request
type ToolUse struct {
	ID       string
	Name     string
	Input    map[string]interface{}
	Result   string
	Status   ToolUseStatus
	Duration time.Duration
}

// PermissionRequest represents a permission prompt from Claude
type PermissionRequest struct {
	Tool        string   // Which tool needs permission
	Description string   // Human-readable description
	Options     []string // e.g., ["Allow", "Deny", "Allow All"]
}

// Config holds agent configuration
type Config struct {
	Command            string
	PlanningArgs       []string
	ImplementationArgs []string
	WorkDir            string
}

// Registry holds registered agents
type Registry struct {
	agents map[string]Agent
}

// NewRegistry creates a new agent registry
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]Agent),
	}
}

// Register registers an agent
func (r *Registry) Register(name string, agent Agent) {
	r.agents[name] = agent
}

// Get returns an agent by name
func (r *Registry) Get(name string) Agent {
	return r.agents[name]
}

// List returns all registered agent names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}
