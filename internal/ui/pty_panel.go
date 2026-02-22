package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/planck/internal/session"
)

// PTYPanel displays an embedded PTY session
type PTYPanel struct {
	theme     *Theme
	visible   bool
	focused   bool
	inputMode bool // true = keystrokes go to PTY

	// Session info
	taskID    string
	taskTitle string
	sessionID string
	status    string

	// Terminal state
	content string

	// Line-based scrollback
	scrollback   *session.ScrollbackBuffer
	scrollOffset int // 0 = live view, >0 = lines scrolled up from bottom

	// Dimensions
	width  int
	height int

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
	case tea.MouseMsg:
		me := tea.MouseEvent(msg)
		if me.IsWheel() {
			if msg.Button == tea.MouseButtonWheelUp {
				p.scrollUp(3)
			} else if msg.Button == tea.MouseButtonWheelDown {
				p.scrollDown(3)
			}
			return p, nil
		}

	case tea.KeyMsg:
		if p.inputMode {
			// Any keypress in input mode snaps to live view
			if p.scrollOffset > 0 {
				p.scrollOffset = 0
			}

			// Tab/Shift+Tab is for tab-switching, not forwarded to PTY
			if msg.Type == tea.KeyTab || msg.Type == tea.KeyShiftTab {
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
			p.scrollOffset = 0
		case "k", "up":
			p.scrollUp(1)
		case "j", "down":
			p.scrollDown(1)
		case "g":
			p.scrollToTop()
		case "G":
			p.scrollOffset = 0
		case "pgup":
			p.scrollUp(p.height / 2)
		case "pgdown":
			p.scrollDown(p.height / 2)
		}
	}

	return p, nil
}

// scrollUp scrolls up by n lines, clamped to scrollback length.
func (p *PTYPanel) scrollUp(n int) {
	p.scrollOffset += n
	maxOffset := p.scrollbackLen()
	if p.scrollOffset > maxOffset {
		p.scrollOffset = maxOffset
	}
}

// scrollDown scrolls down by n lines, clamped to 0 (live view).
func (p *PTYPanel) scrollDown(n int) {
	p.scrollOffset -= n
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
}

// scrollToTop scrolls to the top of the scrollback buffer.
func (p *PTYPanel) scrollToTop() {
	p.scrollOffset = p.scrollbackLen()
}

// scrollbackLen returns the number of lines in the scrollback buffer, or 0.
func (p *PTYPanel) scrollbackLen() int {
	if p.scrollback == nil {
		return 0
	}
	return p.scrollback.Len()
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
	Title     string // Window title set by child process via OSC escape sequences
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

// View renders the PTY panel — full-height, no chrome
func (p *PTYPanel) View() string {
	if !p.visible {
		return ""
	}

	// Split live screen content into lines
	var screenLines []string
	if p.content != "" {
		screenLines = strings.Split(p.content, "\n")
	}
	screenHeight := len(screenLines)

	sbLen := p.scrollbackLen()
	totalLines := sbLen + screenHeight

	// The viewport is p.height lines tall.
	// viewportBottom is the last virtual line index (exclusive) visible.
	// When scrollOffset==0 the viewport shows the live screen.
	viewportBottom := totalLines - p.scrollOffset
	viewportTop := viewportBottom - p.height
	if viewportTop < 0 {
		viewportTop = 0
	}

	var sb strings.Builder
	for row := viewportTop; row < viewportTop+p.height; row++ {
		if row < sbLen {
			// Scrollback region
			if p.scrollback != nil {
				sb.WriteString(p.scrollback.Line(row))
			}
		} else if row < totalLines {
			// Live screen region
			screenIdx := row - sbLen
			if screenIdx < screenHeight {
				sb.WriteString(screenLines[screenIdx])
			}
		}
		// else: beyond content, leave blank

		if row < viewportTop+p.height-1 {
			sb.WriteString("\n")
		}
	}

	rendered := p.theme.DetailPanel.Width(p.width).Height(p.height).Render(sb.String())

	// Overlay scroll indicator when scrolled up
	if p.scrollOffset > 0 {
		indicator := fmt.Sprintf(" SCROLL [-%d lines] ", p.scrollOffset)
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFCC00"))
		badge := style.Render(indicator)

		// Place indicator at top-right of the rendered output
		renderedLines := strings.Split(rendered, "\n")
		if len(renderedLines) > 0 {
			badgeWidth := lipgloss.Width(badge)
			firstLine := renderedLines[0]
			lineWidth := lipgloss.Width(firstLine)
			if badgeWidth < lineWidth {
				pos := lineWidth - badgeWidth - 1
				if pos < 0 {
					pos = 0
				}
				renderedLines[0] = firstLine[:pos] + badge
				rendered = strings.Join(renderedLines, "\n")
			}
		}
	}

	return rendered
}

// Show shows the PTY panel
func (p *PTYPanel) Show(taskID, taskTitle, sessionID string) {
	p.visible = true
	p.taskID = taskID
	p.taskTitle = taskTitle
	p.sessionID = sessionID
	p.status = "running"
	p.inputMode = false
	p.content = ""
	p.scrollOffset = 0
}

// SetScrollback sets the scrollback buffer reference for this panel.
func (p *PTYPanel) SetScrollback(sb *session.ScrollbackBuffer) {
	p.scrollback = sb
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

// GetStatus returns the current session status
func (p *PTYPanel) GetStatus() string {
	return p.status
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
}

// ExitInputMode exits input mode
func (p *PTYPanel) ExitInputMode() {
	p.inputMode = false
}

// TerminalSize returns the size for the PTY terminal (full panel, minus padding)
func (p *PTYPanel) TerminalSize() (rows, cols int) {
	rows = p.height
	cols = p.width - 4 // Account for DetailPanel horizontal padding (2+2)
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	return rows, cols
}
