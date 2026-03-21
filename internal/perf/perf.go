// Package perf provides lightweight performance counters for Planck.
//
// Enable with PLANCK_PERF=1 to write periodic stats to ~/.planck/perf.log.
// When disabled, counter operations are atomic increments with no I/O.
package perf

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// flushInterval controls how often stats are written to the log file.
const flushInterval = 5 * time.Second

// Counters — always incremented (atomic, ~1ns overhead), only logged when enabled.
var (
	Polls          atomic.Int64
	PtyRenders     atomic.Int64
	TmuxCalls      atomic.Int64
	ViewCalls      atomic.Int64
	ViewDurationNs atomic.Int64
	MessagesTotal  atomic.Int64
	PollSkips      atomic.Int64 // polls where content was unchanged (short-circuited)
)

var (
	enabled bool
	logFile *os.File
	done    chan struct{}
)

// Init starts the perf logging system. If enable is false, counters still
// work but nothing is written to disk.
func Init(enable bool) {
	enabled = enable
	if !enabled {
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	dir := filepath.Join(home, ".planck")
	_ = os.MkdirAll(dir, 0o755)

	f, err := os.OpenFile(
		filepath.Join(dir, "perf.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return
	}
	logFile = f
	done = make(chan struct{})

	go flushLoop()
}

// Close stops the flush goroutine and writes final stats.
func Close() {
	if !enabled || done == nil {
		return
	}
	close(done)
	// Final flush
	writeStats()
	if logFile != nil {
		logFile.Close()
	}
}

// Snapshot returns the current counter values and resets them to zero.
func Snapshot() Stats {
	return Stats{
		Polls:          Polls.Swap(0),
		PtyRenders:     PtyRenders.Swap(0),
		TmuxCalls:      TmuxCalls.Swap(0),
		ViewCalls:      ViewCalls.Swap(0),
		ViewDurationNs: ViewDurationNs.Swap(0),
		MessagesTotal:  MessagesTotal.Swap(0),
		PollSkips:      PollSkips.Swap(0),
	}
}

// Stats holds a point-in-time snapshot of all counters.
type Stats struct {
	Polls          int64
	PtyRenders     int64
	TmuxCalls      int64
	ViewCalls      int64
	ViewDurationNs int64
	MessagesTotal  int64
	PollSkips      int64
}

// FormatLine returns a human-readable one-line summary.
func (s Stats) FormatLine() string {
	var viewAvgMs float64
	if s.ViewCalls > 0 {
		viewAvgMs = float64(s.ViewDurationNs) / float64(s.ViewCalls) / 1e6
	}
	return fmt.Sprintf(
		"polls=%d poll_skips=%d pty_renders=%d tmux_calls=%d views=%d view_avg_ms=%.1f messages=%d",
		s.Polls, s.PollSkips, s.PtyRenders, s.TmuxCalls, s.ViewCalls, viewAvgMs, s.MessagesTotal,
	)
}

func flushLoop() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			writeStats()
		}
	}
}

func writeStats() {
	if logFile == nil {
		return
	}
	s := Snapshot()
	// Skip writing if nothing happened
	if s.MessagesTotal == 0 && s.Polls == 0 && s.ViewCalls == 0 {
		return
	}
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), s.FormatLine())
	_, _ = logFile.WriteString(line)
}
