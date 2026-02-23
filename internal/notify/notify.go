package notify

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// EventType represents the type of notification event
type EventType int

const (
	EventPlanningComplete EventType = iota
	EventTaskComplete
	EventPhaseComplete
	EventExecutionComplete
	EventError
	EventNeedsInput
)

// Event represents a notification event
type Event struct {
	Type      EventType
	Timestamp time.Time
	PlanID    string
	TaskID    string
	PhaseNum  int
	Message   string
	Error     error
}

// Notifier interface for sending notifications
type Notifier interface {
	// Notify sends a notification
	Notify(event Event)

	// History returns recent notification history
	History() []Event
}

// BellNotifier implements Notifier using terminal bell
type BellNotifier struct {
	enabled    bool
	mu         sync.Mutex
	history    []Event
	maxHistory int
}

// NewBellNotifier creates a new bell notifier
func NewBellNotifier(enabled bool) *BellNotifier {
	return &BellNotifier{
		enabled:    enabled,
		history:    make([]Event, 0),
		maxHistory: 100,
	}
}

// Notify sends a bell notification
func (n *BellNotifier) Notify(event Event) {
	event.Timestamp = time.Now()

	// Add to history
	n.mu.Lock()
	n.history = append(n.history, event)
	if len(n.history) > n.maxHistory {
		n.history = n.history[1:]
	}
	n.mu.Unlock()

	// Ring the bell if enabled
	if n.enabled {
		fmt.Fprint(os.Stdout, "\a") // Bell character
	}
}

// History returns notification history
func (n *BellNotifier) History() []Event {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Return copy
	history := make([]Event, len(n.history))
	copy(history, n.history)
	return history
}

// SetEnabled sets whether notifications are enabled
func (n *BellNotifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = enabled
}

// MultiNotifier sends notifications to multiple backends
type MultiNotifier struct {
	notifiers []Notifier
}

// NewMultiNotifier creates a new multi-notifier
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{
		notifiers: notifiers,
	}
}

// Notify sends to all notifiers
func (m *MultiNotifier) Notify(event Event) {
	for _, n := range m.notifiers {
		n.Notify(event)
	}
}

// History returns history from first notifier
func (m *MultiNotifier) History() []Event {
	if len(m.notifiers) > 0 {
		return m.notifiers[0].History()
	}
	return nil
}

// NullNotifier is a no-op notifier
type NullNotifier struct{}

// Notify does nothing
func (n *NullNotifier) Notify(event Event) {}

// History returns empty history
func (n *NullNotifier) History() []Event {
	return nil
}

// EventString returns a human-readable event description
func EventString(event Event) string {
	switch event.Type {
	case EventPlanningComplete:
		return fmt.Sprintf("Planning complete for %s", event.PlanID)
	case EventTaskComplete:
		return fmt.Sprintf("Task %s complete", event.TaskID)
	case EventPhaseComplete:
		return fmt.Sprintf("Phase %d complete", event.PhaseNum)
	case EventExecutionComplete:
		return fmt.Sprintf("Execution complete for %s", event.PlanID)
	case EventError:
		return fmt.Sprintf("Error: %s", event.Message)
	case EventNeedsInput:
		return fmt.Sprintf("Needs input: %s", event.Message)
	default:
		return event.Message
	}
}
