package perf

import (
	"strings"
	"testing"
)

func TestCounters_Increment(t *testing.T) {
	// Reset counters
	Polls.Store(0)
	PtyRenders.Store(0)
	TmuxCalls.Store(0)
	ViewCalls.Store(0)
	ViewDurationNs.Store(0)
	MessagesTotal.Store(0)
	PollSkips.Store(0)

	Polls.Add(5)
	PtyRenders.Add(3)
	TmuxCalls.Add(10)
	ViewCalls.Add(7)
	ViewDurationNs.Add(14_000_000) // 14ms total
	MessagesTotal.Add(20)
	PollSkips.Add(2)

	s := Snapshot()

	if s.Polls != 5 {
		t.Errorf("Polls = %d, want 5", s.Polls)
	}
	if s.PtyRenders != 3 {
		t.Errorf("PtyRenders = %d, want 3", s.PtyRenders)
	}
	if s.TmuxCalls != 10 {
		t.Errorf("TmuxCalls = %d, want 10", s.TmuxCalls)
	}
	if s.ViewCalls != 7 {
		t.Errorf("ViewCalls = %d, want 7", s.ViewCalls)
	}
	if s.ViewDurationNs != 14_000_000 {
		t.Errorf("ViewDurationNs = %d, want 14000000", s.ViewDurationNs)
	}
	if s.MessagesTotal != 20 {
		t.Errorf("MessagesTotal = %d, want 20", s.MessagesTotal)
	}
	if s.PollSkips != 2 {
		t.Errorf("PollSkips = %d, want 2", s.PollSkips)
	}
}

func TestSnapshot_ResetsCounters(t *testing.T) {
	Polls.Store(0)
	Polls.Add(10)
	_ = Snapshot()

	// After snapshot, counters should be zero
	if v := Polls.Load(); v != 0 {
		t.Errorf("Polls after Snapshot = %d, want 0", v)
	}
}

func TestFormatLine(t *testing.T) {
	s := Stats{
		Polls:          42,
		PollSkips:      30,
		PtyRenders:     12,
		TmuxCalls:      84,
		ViewCalls:      12,
		ViewDurationNs: 24_000_000, // 24ms total → 2.0ms avg
		MessagesTotal:  128,
	}

	line := s.FormatLine()

	expected := []string{
		"polls=42",
		"poll_skips=30",
		"pty_renders=12",
		"tmux_calls=84",
		"views=12",
		"view_avg_ms=2.0",
		"messages=128",
	}

	for _, want := range expected {
		if !strings.Contains(line, want) {
			t.Errorf("FormatLine() = %q, missing %q", line, want)
		}
	}
}

func TestFormatLine_ZeroViewCalls(t *testing.T) {
	s := Stats{ViewCalls: 0, ViewDurationNs: 0}
	line := s.FormatLine()

	if !strings.Contains(line, "view_avg_ms=0.0") {
		t.Errorf("FormatLine() with zero views = %q, want view_avg_ms=0.0", line)
	}
}

func TestInit_Disabled(t *testing.T) {
	// Init with false should not panic and not create files
	Init(false)
	// Close should be safe even when disabled
	Close()
}
