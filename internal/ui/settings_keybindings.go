package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// keybindingEntry represents a single key binding.
type keybindingEntry struct {
	Key  string
	Desc string
}

// keybindingContext groups keybindings by context.
type keybindingContext struct {
	Name     string
	Bindings []keybindingEntry
}

var keybindingContexts = []keybindingContext{
	{
		Name: "Global",
		Bindings: []keybindingEntry{
			{"Tab / Shift+Tab", "Cycle through tabs"},
			{"1-9", "Jump to tab by number"},
			{"a", "Create new agent tab"},
			{"x / Ctrl+X", "Close agent tab"},
			{"s", "Settings"},
			{"?", "Toggle help"},
			{"q / Ctrl+c", "Quit"},
		},
	},
	{
		Name: "File Browser",
		Bindings: []keybindingEntry{
			{"j / \u2193", "Move down"},
			{"k / \u2191", "Move up"},
			{"Enter", "Open file"},
			{"\u2192 / l", "Expand folder"},
			{"\u2190 / h", "Collapse folder"},
			{"e", "Edit mode"},
			{"n", "New file"},
			{"c", "Toggle complete"},
			{"d", "Delete file"},
			{"o", "Switch folder"},
		},
	},
	{
		Name: "Edit Mode",
		Bindings: []keybindingEntry{
			{"Esc", "Save & exit"},
			{"Ctrl+S", "Save"},
		},
	},
	{
		Name: "Agent (Input)",
		Bindings: []keybindingEntry{
			{"Ctrl+\\", "Normal mode"},
			{"Ctrl+X", "Close tab"},
			{"Scroll", "Browse scrollback"},
			{"Tab / Shift+Tab", "Cycle tabs"},
		},
	},
	{
		Name: "Agent (Normal)",
		Bindings: []keybindingEntry{
			{"i / Enter", "Input mode"},
			{"x", "Close tab"},
			{"a", "New agent tab"},
		},
	},
	{
		Name: "Settings",
		Bindings: []keybindingEntry{
			{"j / k", "Navigate"},
			{"Enter / \u2192", "Change value"},
			{"\u2190 / h", "Previous / back"},
			{"Tab", "Switch section"},
			{"r", "Reset (markdown)"},
			{"R", "Reset all (markdown)"},
			{"Esc / s", "Close"},
		},
	},
}

// keybindingsPage implements the Keybindings reference page.
type keybindingsPage struct {
	theme *Theme

	// Navigation
	contextIdx int
	scrollOffset int
}

func newKeybindingsPage(theme *Theme) *keybindingsPage {
	return &keybindingsPage{
		theme: theme,
	}
}

func (p *keybindingsPage) Title() string { return "Keybindings" }

func (p *keybindingsPage) IsEditing() bool { return false }

func (p *keybindingsPage) FooterHints() string {
	return "[j/k] navigate  [Tab] section  [Esc] close"
}

func (p *keybindingsPage) OnEnter() {
	p.contextIdx = 0
	p.scrollOffset = 0
}

func (p *keybindingsPage) OnLeave() tea.Cmd {
	return nil // read-only, no save
}

func (p *keybindingsPage) Update(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "j", "down":
		if p.contextIdx < len(keybindingContexts)-1 {
			p.contextIdx++
			p.scrollOffset = 0
		}
	case "k", "up":
		if p.contextIdx > 0 {
			p.contextIdx--
			p.scrollOffset = 0
		}
	case "h", "left":
		return nil // signal: go to sidebar
	}
	return nil
}

func (p *keybindingsPage) View(width, height int, theme *Theme) string {
	contextListWidth := width / 3
	if contextListWidth < 18 {
		contextListWidth = 18
	}
	if contextListWidth > 22 {
		contextListWidth = 22
	}
	shortcutsWidth := width - contextListWidth - 2

	list := p.renderContextList(contextListWidth, height)
	shortcuts := p.renderShortcuts(shortcutsWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, list, shortcuts)
}

func (p *keybindingsPage) renderContextList(listWidth, height int) string {
	var sb strings.Builder

	sb.WriteString(p.theme.Title.Render("Contexts"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", listWidth)))
	sb.WriteString("\n")

	for i, ctx := range keybindingContexts {
		if i == p.contextIdx {
			sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" \u25B8 %s", ctx.Name)))
		} else {
			sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s", ctx.Name)))
		}
		sb.WriteString("\n")
	}

	// Fill
	contentHeight := height - 4
	rendered := sb.String()
	lineCount := strings.Count(rendered, "\n")
	for i := lineCount; i < contentHeight; i++ {
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(listWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(p.theme.Accent).
		Render(sb.String())
}

func (p *keybindingsPage) renderShortcuts(shortcutsWidth, height int) string {
	var sb strings.Builder

	if p.contextIdx >= len(keybindingContexts) {
		return ""
	}

	ctx := keybindingContexts[p.contextIdx]

	sb.WriteString(p.theme.Title.Render(ctx.Name))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", shortcutsWidth-2)))
	sb.WriteString("\n\n")

	keyColWidth := 16
	for _, b := range ctx.Bindings {
		keyStr := p.theme.Selected.Width(keyColWidth).Render(b.Key)
		descStr := p.theme.Normal.Render(b.Desc)
		sb.WriteString(keyStr)
		sb.WriteString(descStr)
		sb.WriteString("\n")
	}

	// Fill
	contentHeight := height - 4
	rendered := sb.String()
	lineCount := strings.Count(rendered, "\n")
	for i := lineCount; i < contentHeight; i++ {
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(shortcutsWidth).
		PaddingLeft(1).
		Render(sb.String())
}
