package notify

import (
	"testing"
	"time"
)

func TestBellNotifierNotify(t *testing.T) {
	// Test with bell disabled to avoid actual bell sounds during tests
	notifier := NewBellNotifier(false)

	event := Event{
		Type:    EventTaskComplete,
		PlanID:  "test-plan",
		TaskID:  "1.1",
		Message: "Task completed",
	}

	notifier.Notify(event)

	history := notifier.History()
	if len(history) != 1 {
		t.Fatalf("History length = %d, want 1", len(history))
	}

	if history[0].TaskID != "1.1" {
		t.Errorf("TaskID = %s, want 1.1", history[0].TaskID)
	}

	if history[0].Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestBellNotifierHistory(t *testing.T) {
	notifier := NewBellNotifier(false)

	// Add multiple events
	for i := 0; i < 5; i++ {
		notifier.Notify(Event{
			Type:    EventTaskComplete,
			TaskID:  string(rune('A' + i)),
			Message: "Task complete",
		})
	}

	history := notifier.History()
	if len(history) != 5 {
		t.Errorf("History length = %d, want 5", len(history))
	}

	// Verify it's a copy
	history[0].TaskID = "modified"
	originalHistory := notifier.History()
	if originalHistory[0].TaskID == "modified" {
		t.Error("History should return a copy")
	}
}

func TestBellNotifierMaxHistory(t *testing.T) {
	notifier := &BellNotifier{
		enabled:    false,
		history:    make([]Event, 0),
		maxHistory: 3,
	}

	// Add more than max
	for i := 0; i < 5; i++ {
		notifier.Notify(Event{
			Type:   EventTaskComplete,
			TaskID: string(rune('A' + i)),
		})
	}

	history := notifier.History()
	if len(history) != 3 {
		t.Errorf("History length = %d, want 3 (max)", len(history))
	}

	// Should have the most recent events
	if history[0].TaskID != "C" {
		t.Errorf("First event TaskID = %s, want C", history[0].TaskID)
	}
}

func TestBellNotifierSetEnabled(t *testing.T) {
	notifier := NewBellNotifier(false)

	if notifier.enabled {
		t.Error("Initially should be disabled")
	}

	notifier.SetEnabled(true)
	if !notifier.enabled {
		t.Error("Should be enabled after SetEnabled(true)")
	}

	notifier.SetEnabled(false)
	if notifier.enabled {
		t.Error("Should be disabled after SetEnabled(false)")
	}
}

func TestMultiNotifier(t *testing.T) {
	notifier1 := NewBellNotifier(false)
	notifier2 := NewBellNotifier(false)

	multi := NewMultiNotifier(notifier1, notifier2)

	event := Event{
		Type:    EventPlanningComplete,
		PlanID:  "test-plan",
		Message: "Planning done",
	}

	multi.Notify(event)

	// Both should have the event
	if len(notifier1.History()) != 1 {
		t.Error("Notifier 1 should have event")
	}
	if len(notifier2.History()) != 1 {
		t.Error("Notifier 2 should have event")
	}
}

func TestMultiNotifierHistory(t *testing.T) {
	notifier1 := NewBellNotifier(false)
	notifier2 := NewBellNotifier(false)

	multi := NewMultiNotifier(notifier1, notifier2)

	multi.Notify(Event{Type: EventTaskComplete, TaskID: "1.1"})

	// History should come from first notifier
	history := multi.History()
	if len(history) != 1 {
		t.Errorf("History length = %d, want 1", len(history))
	}
}

func TestMultiNotifierEmpty(t *testing.T) {
	multi := NewMultiNotifier()

	// Should not panic
	multi.Notify(Event{Type: EventError, Message: "test"})

	history := multi.History()
	if history != nil {
		t.Error("Empty multi-notifier should return nil history")
	}
}

func TestNullNotifier(t *testing.T) {
	notifier := &NullNotifier{}

	// Should not panic
	notifier.Notify(Event{Type: EventError, Message: "test"})

	history := notifier.History()
	if history != nil {
		t.Error("Null notifier should return nil history")
	}
}

func TestEventString(t *testing.T) {
	tests := []struct {
		event    Event
		contains string
	}{
		{
			event:    Event{Type: EventPlanningComplete, PlanID: "test-plan"},
			contains: "Planning complete",
		},
		{
			event:    Event{Type: EventTaskComplete, TaskID: "1.1"},
			contains: "Task 1.1 complete",
		},
		{
			event:    Event{Type: EventPhaseComplete, PhaseNum: 2},
			contains: "Phase 2 complete",
		},
		{
			event:    Event{Type: EventExecutionComplete, PlanID: "test"},
			contains: "Execution complete",
		},
		{
			event:    Event{Type: EventError, Message: "Something went wrong"},
			contains: "Error",
		},
		{
			event:    Event{Type: EventNeedsInput, Message: "Please confirm"},
			contains: "Needs input",
		},
		{
			event:    Event{Type: EventType(99), Message: "Unknown"},
			contains: "Unknown",
		},
	}

	for _, tt := range tests {
		result := EventString(tt.event)
		if result == "" {
			t.Errorf("EventString() returned empty for %v", tt.event.Type)
		}
	}
}

func TestEventTypeValues(t *testing.T) {
	// Verify event types have distinct values
	types := []EventType{
		EventPlanningComplete,
		EventTaskComplete,
		EventPhaseComplete,
		EventExecutionComplete,
		EventError,
		EventNeedsInput,
	}

	seen := make(map[EventType]bool)
	for _, et := range types {
		if seen[et] {
			t.Errorf("Duplicate event type value: %d", et)
		}
		seen[et] = true
	}
}

func TestEventTimestamp(t *testing.T) {
	notifier := NewBellNotifier(false)

	before := time.Now()
	notifier.Notify(Event{Type: EventTaskComplete})
	after := time.Now()

	history := notifier.History()
	ts := history[0].Timestamp

	if ts.Before(before) || ts.After(after) {
		t.Error("Timestamp should be between before and after times")
	}
}
