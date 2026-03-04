package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KeybindingsChangedMsg is emitted when keybindings are changed.
type KeybindingsChangedMsg struct {
	Keymap *Keymap
}

// keybindingsPage implements the Keybindings settings page with interactive rebinding.
type keybindingsPage struct {
	theme  *Theme
	keymap *Keymap

	// Navigation
	contextIdx int
	bindingIdx int

	// Key capture state
	capturing bool
	conflict  *keybindingConflict // non-nil when a conflict is detected

	// Dirty flag
	dirty bool
}

// keybindingConflict represents a detected keybinding conflict.
type keybindingConflict struct {
	NewKey         string
	ConflictAction Action
	ConflictDesc   string
}

func newKeybindingsPage(theme *Theme, keymap *Keymap) *keybindingsPage {
	return &keybindingsPage{
		theme:  theme,
		keymap: keymap,
	}
}

func (p *keybindingsPage) Title() string { return "Keybindings" }

func (p *keybindingsPage) IsEditing() bool { return p.capturing || p.conflict != nil }

func (p *keybindingsPage) FooterHints() string {
	if p.conflict != nil {
		return "[Enter] swap bindings  [Esc] cancel"
	}
	if p.capturing {
		return "Press new key...  [Esc] cancel"
	}
	return "[j/k] navigate  [Enter] rebind  [r] reset  [R] reset all  [Tab] section  [Esc] close"
}

func (p *keybindingsPage) OnEnter() {
	p.contextIdx = 0
	p.bindingIdx = 0
	p.capturing = false
	p.conflict = nil
}

func (p *keybindingsPage) OnLeave() tea.Cmd {
	if !p.dirty {
		return nil
	}
	p.dirty = false
	km := p.keymap
	return func() tea.Msg {
		return KeybindingsChangedMsg{Keymap: km}
	}
}

func (p *keybindingsPage) Update(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// Handle conflict resolution
	if p.conflict != nil {
		return p.handleConflict(key)
	}

	// Handle key capture mode
	if p.capturing {
		return p.handleCapture(key, msg)
	}

	// Normal navigation
	switch key {
	case "j", "down":
		ctx := p.keymap.Contexts[p.contextIdx]
		if p.bindingIdx < len(ctx.Bindings)-1 {
			p.bindingIdx++
		}
	case "k", "up":
		if p.bindingIdx > 0 {
			p.bindingIdx--
		}
	case "h", "left":
		return nil // signal: go to sidebar

	case "enter", "l", "right":
		// Enter capture mode for selected binding
		p.capturing = true

	case "r":
		// Reset selected binding to default
		p.resetBinding()

	case "R":
		// Reset all bindings to defaults
		p.resetAll()
	}

	return nil
}

func (p *keybindingsPage) handleCapture(key string, msg tea.KeyMsg) tea.Cmd {
	// Esc cancels capture
	if msg.Type == tea.KeyEscape {
		p.capturing = false
		return nil
	}

	// Don't allow binding tab or shift+tab (used for navigation)
	if msg.Type == tea.KeyTab || msg.Type == tea.KeyShiftTab {
		return nil
	}

	ctx := p.keymap.Contexts[p.contextIdx]
	binding := ctx.Bindings[p.bindingIdx]

	// Check for conflicts in the same context
	conflictAction := p.keymap.ActionFor(ctx.Context, key)
	if conflictAction != "" && conflictAction != binding.Action {
		p.capturing = false
		p.conflict = &keybindingConflict{
			NewKey:         key,
			ConflictAction: conflictAction,
			ConflictDesc:   p.keymap.DescFor(ctx.Context, conflictAction),
		}
		return nil
	}

	// Apply the binding
	p.keymap.SetBinding(ctx.Context, binding.Action, []string{key})
	p.capturing = false
	p.dirty = true

	return nil
}

func (p *keybindingsPage) handleConflict(key string) tea.Cmd {
	switch key {
	case "enter":
		// Swap bindings
		ctx := p.keymap.Contexts[p.contextIdx]
		currentBinding := ctx.Bindings[p.bindingIdx]

		// Get the current key of the binding being changed
		oldKeys := p.keymap.KeysFor(ctx.Context, currentBinding.Action)

		// Set the new key on the selected binding
		p.keymap.SetBinding(ctx.Context, currentBinding.Action, []string{p.conflict.NewKey})

		// Give the conflicting action the old keys
		if len(oldKeys) > 0 {
			p.keymap.SetBinding(ctx.Context, p.conflict.ConflictAction, oldKeys)
		}

		p.conflict = nil
		p.dirty = true

	case "esc":
		p.conflict = nil
	}

	return nil
}

func (p *keybindingsPage) resetBinding() {
	ctx := p.keymap.Contexts[p.contextIdx]
	binding := ctx.Bindings[p.bindingIdx]

	defaults := DefaultKeymap()
	defaultKeys := defaults.KeysFor(ctx.Context, binding.Action)
	if len(defaultKeys) > 0 {
		p.keymap.SetBinding(ctx.Context, binding.Action, defaultKeys)
		p.dirty = true
	}
}

func (p *keybindingsPage) resetAll() {
	defaults := DefaultKeymap()
	for _, cb := range defaults.Contexts {
		for _, b := range cb.Bindings {
			keys := make([]string, len(b.Keys))
			copy(keys, b.Keys)
			p.keymap.SetBinding(cb.Context, b.Action, keys)
		}
	}
	p.dirty = true
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

	for i, ctx := range p.keymap.Contexts {
		if i == p.contextIdx {
			sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" \u25B8 %s", ctx.Label)))
		} else {
			sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s", ctx.Label)))
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

	if p.contextIdx >= len(p.keymap.Contexts) {
		return ""
	}

	ctx := p.keymap.Contexts[p.contextIdx]

	sb.WriteString(p.theme.Title.Render(ctx.Label))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", shortcutsWidth-2)))
	sb.WriteString("\n\n")

	keyColWidth := 16
	descColWidth := shortcutsWidth - keyColWidth - 4

	for i, b := range ctx.Bindings {
		keyDisplay := formatKeys(b.Keys)
		isSelected := i == p.bindingIdx
		isCustomized := p.keymap.IsCustomized(ctx.Context, b.Action)

		var keyStr string
		var descStr string

		switch {
		case isSelected && p.capturing:
			// Show capture prompt
			keyStr = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFCC00")).
				Width(keyColWidth).
				Render("...")
			descStr = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFCC00")).
				Width(descColWidth).
				Render("Press new key")
		case isSelected && p.conflict != nil:
			// Show conflict warning
			keyStr = lipgloss.NewStyle().
				Bold(true).
				Foreground(p.theme.Error).
				Width(keyColWidth).
				Render(displayKey(p.conflict.NewKey))
			descStr = lipgloss.NewStyle().
				Foreground(p.theme.Error).
				Width(descColWidth).
				Render(fmt.Sprintf("Conflicts with: %s", p.conflict.ConflictDesc))
		case isSelected:
			keyStr = p.theme.Selected.Width(keyColWidth).Render(keyDisplay)
			descStr = p.theme.Selected.Width(descColWidth).Render(b.Desc)
		case isCustomized:
			keyStr = lipgloss.NewStyle().
				Bold(true).
				Foreground(p.theme.Accent).
				Width(keyColWidth).
				Render(keyDisplay)
			descStr = p.theme.Normal.Width(descColWidth).Render(b.Desc)
		default:
			keyStr = p.theme.Normal.Width(keyColWidth).Render(keyDisplay)
			descStr = p.theme.Dimmed.Width(descColWidth).Render(b.Desc)
		}

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
