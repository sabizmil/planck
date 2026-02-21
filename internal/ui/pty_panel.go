package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PTYPanel displays an embedded PTY session
type PTYPanel struct {
	theme     *Theme
	visible   bool
	focused   bool
	inputMode bool // true = keystrokes go to PTY
	scrollback bool // true = scrolling through history

	// Session info
	taskID    string
	taskTitle string
	sessionID string
	status    string

	// Terminal state
	content string

	// Dimensions
	width  int
	height int

	// Scrollback
	scrollOffset int

	// Escape key sequence
	escapeKey string
}

// NewPTYPanel creates a new PTY panel
func NewPTYPanel(theme *Theme) *PTYPanel {
	return &PTYPanel{
		theme:     theme,
		escapeKey: `ctrl+\`,
		status:    "idle",
	}
}

// Init initializes the panel
func (p *PTYPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (p *PTYPanel) Update(msg tea.Msg) (*PTYPanel, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.inputMode {
			// Tab is for tab-switching, not forwarded to PTY
			if msg.Type == tea.KeyTab {
				return p, nil
			}

			// Check escape hatch (Ctrl+\)
			if msg.Type == tea.KeyCtrlBackslash {
				p.inputMode = false
				return p, nil
			}

			// Forward raw key to PTY
			raw := keyToBytes(msg)
			if len(raw) > 0 {
				return p, func() tea.Msg {
					return PTYWriteMsg{SessionID: p.sessionID, Data: raw}
				}
			}
			return p, nil
		}

		// Non-input mode keybindings
		switch msg.String() {
		case "i", "enter":
			p.inputMode = true
			p.scrollback = false
		case "s":
			p.scrollback = !p.scrollback
			if !p.scrollback {
				p.scrollOffset = 0
			}
		case "j", "down":
			if p.scrollback {
				p.scrollOffset++
				lines := strings.Count(p.content, "\n") + 1
				maxOffset := lines - (p.height - 5)
				if p.scrollOffset > maxOffset {
					p.scrollOffset = maxOffset
				}
				if p.scrollOffset < 0 {
					p.scrollOffset = 0
				}
			}
		case "k", "up":
			if p.scrollback {
				p.scrollOffset--
				if p.scrollOffset < 0 {
					p.scrollOffset = 0
				}
			}
		case "G":
			// Jump to bottom (live mode)
			p.scrollback = false
			p.scrollOffset = 0
		case "g":
			if p.scrollback {
				p.scrollOffset = 0
			}
		}
	}

	return p, nil
}

// PTYWriteMsg carries data to write to the PTY
type PTYWriteMsg struct {
	SessionID string
	Data      []byte
}

// PTYRenderMsg carries the rendered terminal output to the UI
type PTYRenderMsg struct {
	SessionID string
	Content   string
}

// PTYExitedMsg signals the PTY process has exited
type PTYExitedMsg struct {
	SessionID string
	ExitCode  int
}

// keyToBytes converts a tea.KeyMsg to raw bytes for PTY
func keyToBytes(msg tea.KeyMsg) []byte {
	switch msg.Type {
	case tea.KeyEnter:
		return []byte{'\r'}
	case tea.KeyTab:
		return []byte{'\t'}
	case tea.KeyBackspace:
		return []byte{0x7f}
	case tea.KeyEscape:
		return []byte{0x1b}
	case tea.KeyCtrlC:
		return []byte{0x03}
	case tea.KeyCtrlD:
		return []byte{0x04}
	case tea.KeyCtrlZ:
		return []byte{0x1a}
	case tea.KeyCtrlL:
		return []byte{0x0c}
	case tea.KeyCtrlA:
		return []byte{0x01}
	case tea.KeyCtrlE:
		return []byte{0x05}
	case tea.KeyCtrlK:
		return []byte{0x0b}
	case tea.KeyCtrlU:
		return []byte{0x15}
	case tea.KeyCtrlW:
		return []byte{0x17}
	case tea.KeyUp:
		return []byte{0x1b, '[', 'A'}
	case tea.KeyDown:
		return []byte{0x1b, '[', 'B'}
	case tea.KeyRight:
		return []byte{0x1b, '[', 'C'}
	case tea.KeyLeft:
		return []byte{0x1b, '[', 'D'}
	case tea.KeyHome:
		return []byte{0x1b, '[', 'H'}
	case tea.KeyEnd:
		return []byte{0x1b, '[', 'F'}
	case tea.KeyPgUp:
		return []byte{0x1b, '[', '5', '~'}
	case tea.KeyPgDown:
		return []byte{0x1b, '[', '6', '~'}
	case tea.KeyDelete:
		return []byte{0x1b, '[', '3', '~'}
	case tea.KeySpace:
		return []byte{' '}
	// Shifted keys
	case tea.KeyShiftTab:
		return []byte{0x1b, '[', 'Z'}
	case tea.KeyShiftUp:
		return []byte{0x1b, '[', '1', ';', '2', 'A'}
	case tea.KeyShiftDown:
		return []byte{0x1b, '[', '1', ';', '2', 'B'}
	case tea.KeyShiftRight:
		return []byte{0x1b, '[', '1', ';', '2', 'C'}
	case tea.KeyShiftLeft:
		return []byte{0x1b, '[', '1', ';', '2', 'D'}
	case tea.KeyShiftHome:
		return []byte{0x1b, '[', '1', ';', '2', 'H'}
	case tea.KeyShiftEnd:
		return []byte{0x1b, '[', '1', ';', '2', 'F'}
	// Ctrl+arrow keys
	case tea.KeyCtrlUp:
		return []byte{0x1b, '[', '1', ';', '5', 'A'}
	case tea.KeyCtrlDown:
		return []byte{0x1b, '[', '1', ';', '5', 'B'}
	case tea.KeyCtrlRight:
		return []byte{0x1b, '[', '1', ';', '5', 'C'}
	case tea.KeyCtrlLeft:
		return []byte{0x1b, '[', '1', ';', '5', 'D'}
	case tea.KeyCtrlHome:
		return []byte{0x1b, '[', '1', ';', '5', 'H'}
	case tea.KeyCtrlEnd:
		return []byte{0x1b, '[', '1', ';', '5', 'F'}
	case tea.KeyRunes:
		return []byte(string(msg.Runes))
	default:
		return nil
	}
}

// View renders the PTY panel
func (p *PTYPanel) View() string {
	if !p.visible {
		return ""
	}

	var sb strings.Builder

	// Header
	var statusIcon string
	switch p.status {
	case "running":
		statusIcon = p.theme.StatusProgress.Render("● Running")
	case "completed":
		statusIcon = p.theme.StatusDone.Render("✓ Completed")
	case "failed":
		statusIcon = p.theme.StatusFailed.Render("✗ Failed")
	default:
		statusIcon = p.theme.Dimmed.Render("○ Idle")
	}

	header := fmt.Sprintf("Claude  %s", statusIcon)
	if p.inputMode {
		header += lipgloss.NewStyle().Foreground(p.theme.Accent).Render("  [INTERACTIVE]")
	} else if p.scrollback {
		header += p.theme.Dimmed.Render("  [SCROLLBACK - press i to interact]")
	}

	sb.WriteString(p.theme.Title.Render(header))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("─", p.width-4)))
	sb.WriteString("\n")

	// Terminal content
	termHeight := p.height - 5 // Account for header(1) + sep(1) + footer(newline+sep=2) + padding
	if termHeight < 1 {
		termHeight = 1
	}

	if p.content != "" {
		lines := strings.Split(p.content, "\n")
		for i := 0; i < termHeight; i++ {
			if i < len(lines) {
				sb.WriteString(lines[i])
			}
			sb.WriteString("\n")
		}
	} else {
		// Empty terminal
		for i := 0; i < termHeight; i++ {
			sb.WriteString("\n")
		}
	}

	// Footer separator
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("─", p.width-4)))

	return p.theme.DetailPanel.Width(p.width).Height(p.height).Render(sb.String())
}

// Show shows the PTY panel
func (p *PTYPanel) Show(taskID, taskTitle, sessionID string) {
	p.visible = true
	p.taskID = taskID
	p.taskTitle = taskTitle
	p.sessionID = sessionID
	p.status = "running"
	p.inputMode = false
	p.scrollback = false
	p.scrollOffset = 0
	p.content = ""
}

// Hide hides the PTY panel
func (p *PTYPanel) Hide() {
	p.visible = false
	p.inputMode = false
}

// IsVisible returns whether the panel is visible
func (p *PTYPanel) IsVisible() bool {
	return p.visible
}

// IsInputMode returns whether the panel is in input mode
func (p *PTYPanel) IsInputMode() bool {
	return p.inputMode
}

// SetSize sets the panel dimensions
func (p *PTYPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetContent updates the terminal content with pre-rendered ANSI string
func (p *PTYPanel) SetContent(content string) {
	// vt.Emulator.Render() uses \r\n line endings. The \r must be stripped
	// because lipgloss pads lines after content — if \r remains, the padding
	// spaces overwrite the visible text (cursor returns to column 0 first).
	p.content = strings.ReplaceAll(content, "\r", "")
}

// SetStatus updates the session status
func (p *PTYPanel) SetStatus(status string) {
	p.status = status
}

// GetSessionID returns the current session ID
func (p *PTYPanel) GetSessionID() string {
	return p.sessionID
}

// SetFocused sets whether the panel is focused
func (p *PTYPanel) SetFocused(focused bool) {
	p.focused = focused
}

// EnterInputMode enters input mode
func (p *PTYPanel) EnterInputMode() {
	p.inputMode = true
	p.scrollback = false
}

// ExitInputMode exits input mode
func (p *PTYPanel) ExitInputMode() {
	p.inputMode = false
}

// IsScrollback returns whether the panel is in scrollback mode
func (p *PTYPanel) IsScrollback() bool {
	return p.scrollback
}

// TerminalSize returns the size for the PTY terminal
func (p *PTYPanel) TerminalSize() (rows, cols int) {
	rows = p.height - 5 // Account for header(1) + sep(1) + footer(newline+sep=2) + padding
	cols = p.width - 4  // Account for padding
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	return rows, cols
}
