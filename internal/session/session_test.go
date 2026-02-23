package session

import (
	"testing"
)

func TestStatusValues(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusRunning, "running"},
		{StatusPaused, "paused"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusCanceled, "canceled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Status = %s, want %s", tt.status, tt.expected)
		}
	}
}

func TestModeValues(t *testing.T) {
	tests := []struct {
		mode     Mode
		expected string
	}{
		{ModeForeground, "foreground"},
		{ModeBackground, "background"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.expected {
			t.Errorf("Mode = %s, want %s", tt.mode, tt.expected)
		}
	}
}

func TestTypeValues(t *testing.T) {
	tests := []struct {
		typ      Type
		expected string
	}{
		{TypePlanning, "planning"},
		{TypeImplementation, "implementation"},
		{TypeExecution, "execution"},
	}

	for _, tt := range tests {
		if string(tt.typ) != tt.expected {
			t.Errorf("Type = %s, want %s", tt.typ, tt.expected)
		}
	}
}

func TestSessionStructure(t *testing.T) {
	session := &Session{
		ID:             "session-1",
		TaskID:         "1.1",
		PlanID:         "plan-1",
		Type:           TypeImplementation,
		Mode:           ModeForeground,
		Status:         StatusRunning,
		Backend:        "tmux",
		AgentSessionID: "agent-123",
		BackendHandle:  "planck-1.1",
	}

	if session.ID != "session-1" {
		t.Errorf("ID = %s, want session-1", session.ID)
	}
	if session.Type != TypeImplementation {
		t.Errorf("Type = %s, want implementation", session.Type)
	}
	if session.Mode != ModeForeground {
		t.Errorf("Mode = %s, want foreground", session.Mode)
	}
	if session.Status != StatusRunning {
		t.Errorf("Status = %s, want running", session.Status)
	}
}
