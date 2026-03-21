package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPTYPanelTabPassthroughInInputMode(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPTYPanel(theme, DefaultKeymap())
	panel.Show("test-task", "Test", "test-session")
	panel.SetSize(80, 24)
	panel.EnterInputMode()

	// Tab key should produce a PTYWriteMsg (forwarded to PTY)
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	_, cmd := panel.Update(tabMsg)

	if cmd == nil {
		t.Fatal("Tab in input mode should produce a command (PTYWriteMsg), got nil")
	}

	// Execute the command to get the message
	msg := cmd()
	writeMsg, ok := msg.(PTYWriteMsg)
	if !ok {
		t.Fatalf("Expected PTYWriteMsg, got %T", msg)
	}
	if len(writeMsg.Data) != 1 || writeMsg.Data[0] != '\t' {
		t.Errorf("Expected tab byte (0x09), got %v", writeMsg.Data)
	}
}

func TestPTYPanelNextTabBlockedInInputMode(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPTYPanel(theme, DefaultKeymap())
	panel.Show("test-task", "Test", "test-session")
	panel.SetSize(80, 24)
	panel.EnterInputMode()

	// | (default next_tab binding) should be blocked (handled at app level)
	pipeMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'|'}}
	_, cmd := panel.Update(pipeMsg)

	if cmd != nil {
		t.Error("| in input mode should be blocked (return nil cmd), got a command")
	}
}

func TestPTYPanelTabNotForwardedInNormalMode(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPTYPanel(theme, DefaultKeymap())
	panel.Show("test-task", "Test", "test-session")
	panel.SetSize(80, 24)
	// Don't enter input mode — stay in normal mode

	// Tab key in normal mode should not produce a command
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	_, cmd := panel.Update(tabMsg)

	if cmd != nil {
		t.Error("Tab in normal mode should not produce a command, got one")
	}
}

func TestPTYPanelAltDigitBlockedInInputMode(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPTYPanel(theme, DefaultKeymap())
	panel.Show("test-task", "Test", "test-session")
	panel.SetSize(80, 24)
	panel.EnterInputMode()

	// Alt+1 through Alt+9 should be blocked (handled at app level for tab switching)
	for _, r := range "123456789" {
		altMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: true}
		_, cmd := panel.Update(altMsg)
		if cmd != nil {
			t.Errorf("Alt+%c in input mode should be blocked, got a command", r)
		}
	}
}

func TestKeyToBytesTab(t *testing.T) {
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	got := keyToBytes(tabMsg)
	if len(got) != 1 || got[0] != '\t' {
		t.Errorf("keyToBytes(Tab) = %v, want [0x09]", got)
	}
}

func TestKeyToBytesShiftTab(t *testing.T) {
	shiftTabMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	got := keyToBytes(shiftTabMsg)
	expected := []byte{0x1b, '[', 'Z'}
	if len(got) != len(expected) {
		t.Fatalf("keyToBytes(ShiftTab) length = %d, want %d", len(got), len(expected))
	}
	for i, b := range expected {
		if got[i] != b {
			t.Errorf("keyToBytes(ShiftTab)[%d] = 0x%02x, want 0x%02x", i, got[i], b)
		}
	}
}

func TestPTYPanelScrollWithNilScrollback(t *testing.T) {
	// Simulates the tmux backend case: no scrollback buffer, but content
	// exceeds the viewport height (scrollback is embedded in content).
	theme := DefaultTheme()
	panel := NewPTYPanel(theme, DefaultKeymap())
	panel.Show("test-task", "Test", "test-session")
	panel.SetSize(80, 24)
	// scrollback is nil (tmux backend default)

	// Set content with 100 lines (exceeds viewport of 24)
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = strings.Repeat("x", 40)
	}
	panel.SetContent(strings.Join(lines, "\n"))

	// Should be able to scroll up
	panel.scrollUp(10)
	if panel.scrollOffset != 10 {
		t.Errorf("scrollOffset after scrollUp(10) = %d, want 10", panel.scrollOffset)
	}

	// Max offset should be 100 - 24 = 76
	expectedMax := 100 - 24
	panel.scrollUp(1000)
	if panel.scrollOffset != expectedMax {
		t.Errorf("scrollOffset after scrollUp(1000) = %d, want %d", panel.scrollOffset, expectedMax)
	}

	// Scroll back down to live view
	panel.scrollDown(1000)
	if panel.scrollOffset != 0 {
		t.Errorf("scrollOffset after scrollDown(1000) = %d, want 0", panel.scrollOffset)
	}
}

func TestPTYPanelScrollToTopNilScrollback(t *testing.T) {
	theme := DefaultTheme()
	panel := NewPTYPanel(theme, DefaultKeymap())
	panel.Show("test-task", "Test", "test-session")
	panel.SetSize(80, 24)

	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "line"
	}
	panel.SetContent(strings.Join(lines, "\n"))

	panel.scrollToTop()
	expectedMax := 50 - 24
	if panel.scrollOffset != expectedMax {
		t.Errorf("scrollToTop() offset = %d, want %d", panel.scrollOffset, expectedMax)
	}
}

func TestPTYPanelScrollNoOverflow(t *testing.T) {
	// When content fits in viewport, scrolling should be a no-op.
	theme := DefaultTheme()
	panel := NewPTYPanel(theme, DefaultKeymap())
	panel.Show("test-task", "Test", "test-session")
	panel.SetSize(80, 24)

	panel.SetContent("short content\nonly two lines")

	panel.scrollUp(10)
	if panel.scrollOffset != 0 {
		t.Errorf("scrollOffset should be 0 when content fits viewport, got %d", panel.scrollOffset)
	}
}
