package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Help displays the help overlay
type Help struct {
	theme   *Theme
	visible bool
	width   int
	height  int
	offset  int
}

// NewHelp creates a new help overlay
func NewHelp(theme *Theme) *Help {
	return &Help{
		theme: theme,
	}
}

// Init initializes the help
func (h *Help) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (h *Help) Update(msg tea.Msg) (*Help, tea.Cmd) {
	if !h.visible {
		return h, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "?", "esc", "q":
			h.visible = false
		case "j", "down":
			h.offset++
		case "k", "up":
			h.offset--
			if h.offset < 0 {
				h.offset = 0
			}
		}
	}

	return h, nil
}

// View renders the help overlay
func (h *Help) View() string {
	if !h.visible {
		return ""
	}

	content := h.renderHelpContent()

	// Center the help content
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	x := (h.width - contentWidth) / 2
	y := (h.height - contentHeight) / 2

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	var result strings.Builder
	for i := 0; i < y; i++ {
		result.WriteString("\n")
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		result.WriteString(strings.Repeat(" ", x))
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

func (h *Help) renderHelpContent() string {
	var sb strings.Builder

	title := h.theme.DialogTitle.Render("planck Help")
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(h.theme.Dimmed.Render(strings.Repeat("═", 50)))
	sb.WriteString("\n\n")

	// Global
	sb.WriteString(h.theme.Title.Render("Global"))
	sb.WriteString("\n")
	sb.WriteString(h.renderKeySection([][]string{
		{"Shift+Tab", "Next tab"},
		{"Alt+1-9", "Jump to tab by number"},
		{"Click tab", "Switch to tab"},
		{"1-9", "Jump to tab (normal mode)"},
		{"a", "Create new agent tab"},
		{"x / Ctrl+X", "Close current agent tab"},
		{"s", "Settings"},
		{"?", "Toggle this help"},
		{"q / Ctrl+C ×2", "Quit"},
	}))
	sb.WriteString("\n")

	// File List
	sb.WriteString(h.theme.Title.Render("File List (Planning Tab)"))
	sb.WriteString("\n")
	sb.WriteString(h.renderKeySection([][]string{
		{"j / ↓", "Move down"},
		{"k / ↑", "Move up"},
		{"Enter", "Open file in editor"},
		{"→ / l", "Expand folder"},
		{"← / h", "Collapse folder"},
		{"e", "Enter edit mode"},
		{"n", "New file"},
		{"c", "Toggle complete"},
		{"d", "Delete file/folder"},
		{"m", "Move file/folder"},
		{"r", "Refresh file list"},
		{"o", "Switch folder"},
	}))
	sb.WriteString("\n")

	// Edit Mode
	sb.WriteString(h.theme.Title.Render("Edit Mode"))
	sb.WriteString("\n")
	sb.WriteString(h.renderKeySection([][]string{
		{"Esc", "Save & exit edit mode"},
		{"Ctrl+S", "Save without exiting"},
	}))
	sb.WriteString("\n")

	// Agent Tab
	sb.WriteString(h.theme.Title.Render("Agent Tab (Input Mode)"))
	sb.WriteString("\n")
	sb.WriteString(h.renderKeySection([][]string{
		{"Tab", "Sent to agent (autocomplete)"},
		{"Shift+Tab", "Next tab"},
		{"Alt+1-9", "Jump to tab"},
		{"Ctrl+\\", "Exit to normal mode"},
		{"Ctrl+X", "Close tab"},
		{"Scroll", "Browse output history"},
	}))
	sb.WriteString("\n")

	// Agent normal mode
	sb.WriteString(h.theme.Title.Render("Agent Tab (Normal Mode)"))
	sb.WriteString("\n")
	sb.WriteString(h.renderKeySection([][]string{
		{"i / Enter", "Enter input mode"},
		{"x", "Close tab"},
		{"a", "New agent tab"},
	}))
	sb.WriteString("\n")

	sb.WriteString(h.theme.Dimmed.Render("Press ? or Esc to close"))

	return h.theme.Dialog.Width(54).Render(sb.String())
}

func (h *Help) renderKeySection(keys [][]string) string {
	var sb strings.Builder
	for _, kv := range keys {
		key := h.theme.Selected.Width(16).Render(kv[0])
		desc := h.theme.Normal.Render(kv[1])
		sb.WriteString(key)
		sb.WriteString(desc)
		sb.WriteString("\n")
	}
	return sb.String()
}

// Toggle toggles the help visibility
func (h *Help) Toggle() {
	h.visible = !h.visible
	h.offset = 0
}

// IsVisible returns whether help is visible
func (h *Help) IsVisible() bool {
	return h.visible
}

// SetSize sets the container size
func (h *Help) SetSize(width, height int) {
	h.width = width
	h.height = height
}
