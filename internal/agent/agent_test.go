package agent

import (
	"context"
	"testing"
)

// MockAgent for testing
type MockAgent struct {
	name      string
	sessionID string
	output    string
	err       error
}

func (m *MockAgent) Name() string {
	return m.name
}

func (m *MockAgent) Run(ctx context.Context, prompt string) (string, error) {
	return m.output, m.err
}

func (m *MockAgent) Stream(ctx context.Context, prompt string) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent, 2)
	go func() {
		defer close(ch)
		ch <- StreamEvent{Type: EventText, Text: m.output}
		ch <- StreamEvent{Type: EventComplete}
	}()
	return ch, m.err
}

func (m *MockAgent) Send(input string) error {
	return nil
}

func (m *MockAgent) Stop() error {
	return nil
}

func (m *MockAgent) SessionID() string {
	return m.sessionID
}

func TestRegistryRegisterGet(t *testing.T) {
	registry := NewRegistry()

	mockAgent := &MockAgent{name: "test-agent"}

	registry.Register("test", mockAgent)

	agent := registry.Get("test")
	if agent == nil {
		t.Fatal("Get() returned nil")
	}
	if agent.Name() != "test-agent" {
		t.Errorf("Agent name = %s, want test-agent", agent.Name())
	}
}

func TestRegistryGetNonExistent(t *testing.T) {
	registry := NewRegistry()

	agent := registry.Get("nonexistent")
	if agent != nil {
		t.Error("Get() should return nil for non-existent agent")
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	registry.Register("agent1", &MockAgent{name: "agent1"})
	registry.Register("agent2", &MockAgent{name: "agent2"})

	names := registry.List()
	if len(names) != 2 {
		t.Errorf("List() count = %d, want 2", len(names))
	}
}

func TestStreamEventTypes(t *testing.T) {
	tests := []struct {
		eventType StreamEventType
		expected  int
	}{
		{EventText, 0},
		{EventToolUse, 1},
		{EventToolResult, 2},
		{EventComplete, 3},
		{EventError, 4},
	}

	for _, tt := range tests {
		if int(tt.eventType) != tt.expected {
			t.Errorf("StreamEventType = %d, want %d", tt.eventType, tt.expected)
		}
	}
}

func TestMockAgentRun(t *testing.T) {
	agent := &MockAgent{
		name:   "test",
		output: "test output",
	}

	output, err := agent.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if output != "test output" {
		t.Errorf("Output = %s, want 'test output'", output)
	}
}

func TestMockAgentStream(t *testing.T) {
	agent := &MockAgent{
		name:   "test",
		output: "streamed output",
	}

	ch, err := agent.Stream(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var events []StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
	if events[0].Type != EventText {
		t.Errorf("First event type = %v, want EventText", events[0].Type)
	}
	if events[1].Type != EventComplete {
		t.Errorf("Second event type = %v, want EventComplete", events[1].Type)
	}
}
