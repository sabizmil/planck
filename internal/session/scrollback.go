package session

import "sync"

// ScrollbackBuffer is a thread-safe ring buffer that stores lines that have
// scrolled off the top of a terminal screen.
type ScrollbackBuffer struct {
	mu    sync.Mutex
	lines []string
	head  int // next write position
	count int // number of valid lines
	cap   int // max capacity
}

// NewScrollbackBuffer creates a new scrollback buffer with the given capacity.
func NewScrollbackBuffer(capacity int) *ScrollbackBuffer {
	if capacity <= 0 {
		capacity = 1000
	}
	return &ScrollbackBuffer{
		lines: make([]string, capacity),
		cap:   capacity,
	}
}

// Push appends one or more lines to the ring buffer.
func (s *ScrollbackBuffer) Push(lines []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, line := range lines {
		s.lines[s.head] = line
		s.head = (s.head + 1) % s.cap
		if s.count < s.cap {
			s.count++
		}
	}
}

// Len returns the number of stored lines.
func (s *ScrollbackBuffer) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

// Line returns the line at index i (0 = oldest). Returns "" if out of range.
func (s *ScrollbackBuffer) Line(i int) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if i < 0 || i >= s.count {
		return ""
	}
	// oldest line starts at (head - count) mod cap
	idx := (s.head - s.count + i + s.cap) % s.cap
	return s.lines[idx]
}

// Lines returns a slice of lines from offset with the given count,
// suitable for rendering a viewport.
func (s *ScrollbackBuffer) Lines(offset, count int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if offset < 0 {
		offset = 0
	}
	if offset >= s.count {
		return nil
	}
	if offset+count > s.count {
		count = s.count - offset
	}

	result := make([]string, count)
	start := (s.head - s.count + offset + s.cap) % s.cap
	for i := 0; i < count; i++ {
		result[i] = s.lines[(start+i)%s.cap]
	}
	return result
}
