package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// ClaudeAgent implements Agent for Claude Code CLI
type ClaudeAgent struct {
	config    Config
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	sessionID string
	mu        sync.Mutex
}

// NewClaudeAgent creates a new Claude agent
func NewClaudeAgent(cfg Config) *ClaudeAgent {
	return &ClaudeAgent{
		config: cfg,
	}
}

// Name returns the agent name
func (c *ClaudeAgent) Name() string {
	return "claude-code"
}

// Run executes a prompt and returns the full output
func (c *ClaudeAgent) Run(ctx context.Context, prompt string) (string, error) {
	events, err := c.Stream(ctx, prompt)
	if err != nil {
		return "", err
	}

	var output strings.Builder
	for event := range events {
		switch event.Type {
		case EventText:
			output.WriteString(event.Text)
		case EventError:
			return output.String(), event.Error
		}
	}

	return output.String(), nil
}

// Stream executes a prompt and streams output
func (c *ClaudeAgent) Stream(ctx context.Context, prompt string) (<-chan StreamEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build command
	args := append([]string{}, c.config.PlanningArgs...)
	args = append(args, prompt)

	c.cmd = exec.CommandContext(ctx, c.config.Command, args...)
	if c.config.WorkDir != "" {
		c.cmd.Dir = c.config.WorkDir
	}

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}
	c.stdin = stdin

	if err := c.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	events := make(chan StreamEvent, 100)

	go func() {
		defer close(events)
		defer func() { _ = c.cmd.Wait() }()

		// Read stderr in background
		go func() {
			errOutput, _ := io.ReadAll(stderr)
			if len(errOutput) > 0 {
				// Check for auth errors
				errStr := string(errOutput)
				if strings.Contains(errStr, "not authenticated") {
					events <- StreamEvent{
						Type:  EventError,
						Error: fmt.Errorf("not authenticated: run 'claude auth' first"),
					}
				}
			}
		}()

		// Parse stream-json output
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			event, err := parseStreamJSON(line)
			if err != nil {
				// Not JSON, treat as plain text
				events <- StreamEvent{
					Type: EventText,
					Text: line + "\n",
				}
				continue
			}

			events <- event
		}

		if err := scanner.Err(); err != nil {
			events <- StreamEvent{
				Type:  EventError,
				Error: fmt.Errorf("read output: %w", err),
			}
		}

		events <- StreamEvent{Type: EventComplete}
	}()

	return events, nil
}

// parseStreamJSON parses a stream-json line
func parseStreamJSON(line string) (StreamEvent, error) {
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return StreamEvent{}, err
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return StreamEvent{}, fmt.Errorf("missing type field")
	}

	switch msgType {
	case "assistant":
		// Assistant message with content
		if content, ok := msg["content"].([]interface{}); ok {
			var text strings.Builder
			for _, c := range content {
				if contentMap, ok := c.(map[string]interface{}); ok {
					if contentMap["type"] == "text" {
						if t, ok := contentMap["text"].(string); ok {
							text.WriteString(t)
						}
					}
				}
			}
			return StreamEvent{
				Type: EventText,
				Text: text.String(),
			}, nil
		}

	case "content_block_start":
		// Check if it's a thinking block
		if contentBlock, ok := msg["content_block"].(map[string]interface{}); ok {
			if contentBlock["type"] == "thinking" {
				if thinking, ok := contentBlock["thinking"].(string); ok {
					return StreamEvent{
						Type: EventThinking,
						Text: thinking,
					}, nil
				}
			}
		}

	case "content_block_delta":
		// Streaming text delta
		if delta, ok := msg["delta"].(map[string]interface{}); ok {
			// Check for thinking delta
			if delta["type"] == "thinking_delta" {
				if thinking, ok := delta["thinking"].(string); ok {
					return StreamEvent{
						Type: EventThinking,
						Text: thinking,
					}, nil
				}
			}
			// Regular text delta
			if text, ok := delta["text"].(string); ok {
				return StreamEvent{
					Type: EventText,
					Text: text,
				}, nil
			}
		}

	case "tool_use":
		// Tool use request
		toolUse := &ToolUse{
			ID:     getString(msg, "id"),
			Name:   getString(msg, "name"),
			Status: ToolUsePending,
		}
		if input, ok := msg["input"].(map[string]interface{}); ok {
			toolUse.Input = input
		}
		return StreamEvent{
			Type:    EventToolUse,
			ToolUse: toolUse,
		}, nil

	case "tool_result":
		// Tool result
		return StreamEvent{
			Type: EventToolResult,
			Text: getString(msg, "content"),
		}, nil

	case "system":
		// System-level messages
		return StreamEvent{
			Type: EventSystemMessage,
			Text: getString(msg, "message"),
		}, nil

	case "permission_request":
		// Permission request from Claude
		perm := &PermissionRequest{
			Tool:        getString(msg, "tool"),
			Description: getString(msg, "description"),
		}
		if opts, ok := msg["options"].([]interface{}); ok {
			for _, opt := range opts {
				if s, ok := opt.(string); ok {
					perm.Options = append(perm.Options, s)
				}
			}
		}
		return StreamEvent{
			Type:       EventPermission,
			Permission: perm,
		}, nil

	case "error":
		// Error message
		errMsg := getString(msg, "error")
		if errMsg == "" {
			if errObj, ok := msg["error"].(map[string]interface{}); ok {
				errMsg = getString(errObj, "message")
			}
		}
		return StreamEvent{
			Type:  EventError,
			Error: fmt.Errorf("%s", errMsg),
		}, nil

	case "message_stop", "message_end":
		return StreamEvent{Type: EventComplete}, nil
	}

	// Unknown type, try to extract text (don't error - return fallback)
	if text, ok := msg["text"].(string); ok {
		return StreamEvent{
			Type: EventText,
			Text: text,
		}, nil
	}

	// Return empty text event for unknown types (don't crash)
	return StreamEvent{
		Type: EventText,
		Text: "",
	}, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// Send sends input to a running session
func (c *ClaudeAgent) Send(input string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdin == nil {
		return fmt.Errorf("no active session")
	}

	_, err := fmt.Fprintln(c.stdin, input)
	return err
}

// Stop stops the current operation
func (c *ClaudeAgent) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdin != nil {
		c.stdin.Close()
		c.stdin = nil
	}

	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// SessionID returns the current session ID
func (c *ClaudeAgent) SessionID() string {
	return c.sessionID
}

// IsAvailable checks if Claude CLI is available
func IsClaudeAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// CheckAuth checks if Claude is authenticated
func CheckClaudeAuth() error {
	cmd := exec.Command("claude", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude CLI not working: %w", err)
	}
	return nil
}
