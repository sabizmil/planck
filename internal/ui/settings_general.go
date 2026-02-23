package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// generalField represents a configurable field on the general page.
type generalField struct {
	Label    string
	Section  string // optional section header before this field
	Kind     string // "toggle", "cycle", "text"
	Options  []string
	HintText string // shown in info pane when selected
}

var generalFields = []generalField{
	{
		Label:    "Editor",
		Kind:     "text",
		HintText: "External editor command.\nLeave blank to use $EDITOR.",
	},
	{
		Label:    "Terminal Bell",
		Kind:     "toggle",
		HintText: "Ring the terminal bell\nwhen agents complete.",
	},
	{
		Label:    "Sidebar Width",
		Kind:     "number",
		HintText: "Width of the file sidebar\nin characters (16–60).\n\nDrag the sidebar border\nwith the mouse to resize\ninteractively.",
	},
	{
		Label:    "Session Backend",
		Kind:     "cycle",
		Options:  []string{"auto", "pty", "tmux"},
		HintText: "How agent sessions run.\nauto: PTY preferred,\nfalls back to tmux.\n\nApplies to new tabs.",
	},
	{
		Label:    "Default Scope",
		Section:  "Execution",
		Kind:     "cycle",
		Options:  []string{"task", "phase", "plan"},
		HintText: "How much an agent can\ndo before needing\napproval.",
	},
	{
		Label:    "Auto Advance",
		Kind:     "toggle",
		HintText: "Automatically advance\nto the next phase when\nthe current one completes.",
	},
	{
		Label:    "Permission Mode",
		Kind:     "cycle",
		Options:  []string{"pre-approve", "per-phase", "verify-at-end"},
		HintText: "When to ask for\npermission.\n\npre-approve: upfront\nper-phase: each phase\nverify-at-end: after all",
	},
}

// generalPage implements the General settings page.
type generalPage struct {
	theme *Theme

	// Values
	editor         string
	bell           bool
	sidebarWidth   int
	backend        string
	defaultScope   string
	autoAdvance    bool
	permissionMode string

	// Navigation
	selectedIdx int

	// Text editing state
	editing   bool
	editValue string
	editCur   int // cursor position within editValue
}

func newGeneralPage(theme *Theme, cfg GeneralSettingsChangedMsg) *generalPage {
	sw := cfg.SidebarWidth
	if sw == 0 {
		sw = 28
	}
	return &generalPage{
		theme:          theme,
		editor:         cfg.Editor,
		bell:           cfg.Bell,
		sidebarWidth:   sw,
		backend:        cfg.Backend,
		defaultScope:   cfg.DefaultScope,
		autoAdvance:    cfg.AutoAdvance,
		permissionMode: cfg.PermissionMode,
	}
}

func (p *generalPage) Title() string { return "General" }

func (p *generalPage) IsEditing() bool { return p.editing }

func (p *generalPage) FooterHints() string {
	if p.editing {
		return "[Enter] confirm  [Esc] cancel"
	}
	return "[j/k] navigate  [Enter/\u2192] change  [Tab] section  [Esc] close"
}

func (p *generalPage) OnEnter() {
	p.selectedIdx = 0
	p.editing = false
}

func (p *generalPage) OnLeave() tea.Cmd {
	p.editing = false
	return func() tea.Msg {
		return GeneralSettingsChangedMsg{
			Editor:         p.editor,
			Bell:           p.bell,
			SidebarWidth:   p.sidebarWidth,
			Backend:        p.backend,
			DefaultScope:   p.defaultScope,
			AutoAdvance:    p.autoAdvance,
			PermissionMode: p.permissionMode,
		}
	}
}

func (p *generalPage) Update(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if p.editing {
		return p.handleEditKey(key, msg)
	}

	switch key {
	case "j", "down":
		if p.selectedIdx < len(generalFields)-1 {
			p.selectedIdx++
		}

	case "k", "up":
		if p.selectedIdx > 0 {
			p.selectedIdx--
		}

	case "enter", "l", "right":
		p.activateField()

	case "h", "left":
		field := generalFields[p.selectedIdx]
		if field.Kind == "number" {
			p.adjustNumberField(-1)
			return nil
		}
		if field.Kind == "cycle" {
			p.cycleField(-1)
			return nil
		}
		// signal to parent: go to sidebar
		return nil
	}

	return nil
}

func (p *generalPage) handleEditKey(key string, msg tea.KeyMsg) tea.Cmd {
	switch key {
	case "enter":
		p.editor = p.editValue
		p.editing = false
	case "esc":
		p.editing = false
	case "backspace":
		if p.editCur > 0 {
			p.editValue = p.editValue[:p.editCur-1] + p.editValue[p.editCur:]
			p.editCur--
		}
	case "delete":
		if p.editCur < len(p.editValue) {
			p.editValue = p.editValue[:p.editCur] + p.editValue[p.editCur+1:]
		}
	case "left":
		if p.editCur > 0 {
			p.editCur--
		}
	case "right":
		if p.editCur < len(p.editValue) {
			p.editCur++
		}
	case "home", "ctrl+a":
		p.editCur = 0
	case "end", "ctrl+e":
		p.editCur = len(p.editValue)
	default:
		// Insert printable characters
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			p.editValue = p.editValue[:p.editCur] + key + p.editValue[p.editCur:]
			p.editCur++
		} else if len(msg.Runes) > 0 {
			ch := string(msg.Runes)
			p.editValue = p.editValue[:p.editCur] + ch + p.editValue[p.editCur:]
			p.editCur += len(ch)
		}
	}
	return nil
}

func (p *generalPage) activateField() {
	field := generalFields[p.selectedIdx]
	switch field.Kind {
	case "toggle":
		p.toggleField()
	case "cycle":
		p.cycleField(1)
	case "number":
		p.adjustNumberField(1)
	case "text":
		p.editing = true
		p.editValue = p.getFieldValue(p.selectedIdx)
		p.editCur = len(p.editValue)
	}
}

func (p *generalPage) toggleField() {
	switch p.selectedIdx {
	case 1: // Bell
		p.bell = !p.bell
	case 5: // Auto Advance
		p.autoAdvance = !p.autoAdvance
	}
}

func (p *generalPage) cycleField(dir int) {
	field := generalFields[p.selectedIdx]
	current := p.getFieldValue(p.selectedIdx)

	idx := 0
	for i, opt := range field.Options {
		if opt == current {
			idx = i
			break
		}
	}
	next := (idx + dir + len(field.Options)) % len(field.Options)

	switch p.selectedIdx {
	case 3: // Backend
		p.backend = field.Options[next]
	case 4: // Default Scope
		p.defaultScope = field.Options[next]
	case 6: // Permission Mode
		p.permissionMode = field.Options[next]
	}
}

func (p *generalPage) adjustNumberField(dir int) {
	switch p.selectedIdx {
	case 2: // Sidebar Width
		p.sidebarWidth += dir * 2
		if p.sidebarWidth < 16 {
			p.sidebarWidth = 16
		}
		if p.sidebarWidth > 60 {
			p.sidebarWidth = 60
		}
	}
}

func (p *generalPage) getFieldValue(idx int) string {
	switch idx {
	case 0:
		return p.editor
	case 1:
		if p.bell {
			return "On"
		}
		return "Off"
	case 2:
		return strconv.Itoa(p.sidebarWidth)
	case 3:
		return p.backend
	case 4:
		return p.defaultScope
	case 5:
		if p.autoAdvance {
			return "On"
		}
		return "Off"
	case 6:
		return p.permissionMode
	}
	return ""
}

func (p *generalPage) View(width, height int, theme *Theme) string {
	optionsWidth := width * 2 / 3
	if optionsWidth > 36 {
		optionsWidth = 36
	}
	infoWidth := width - optionsWidth - 3

	options := p.renderFields(optionsWidth, height)
	info := p.renderInfo(infoWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, options, info)
}

func (p *generalPage) renderFields(optionsWidth, height int) string {
	var sb strings.Builder

	sb.WriteString(p.theme.Title.Render("General"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", optionsWidth)))
	sb.WriteString("\n")

	for i, field := range generalFields {
		// Section header
		if field.Section != "" {
			sb.WriteString("\n")
			sb.WriteString(p.theme.Dimmed.Render(fmt.Sprintf("\u2500\u2500 %s \u2500\u2500", field.Section)))
			sb.WriteString("\n")
		}

		// Label
		sb.WriteString(p.theme.Dimmed.Render(field.Label))
		sb.WriteString("\n")

		// Value
		value := p.getFieldValue(i)
		isSelected := i == p.selectedIdx

		if p.editing && isSelected {
			// Show text input
			before := p.editValue[:p.editCur]
			after := p.editValue[p.editCur:]
			cursor := p.theme.Selected.Reverse(true).Render(" ")
			if p.editCur < len(p.editValue) {
				cursor = p.theme.Selected.Reverse(true).Render(string(p.editValue[p.editCur]))
				after = after[1:]
			}
			line := "  " + p.theme.Normal.Render(before) + cursor + p.theme.Normal.Render(after)
			sb.WriteString(line)
		} else if isSelected {
			switch field.Kind {
			case "cycle", "number":
				sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" \u25C0 %s \u25B6", value)))
			case "toggle":
				indicator := "\u25CF"
				sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" %s %s", indicator, value)))
			case "text":
				display := value
				if display == "" {
					display = "(default)"
				}
				sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" \u25B8 %s", display)))
			}
		} else {
			switch field.Kind {
			case "toggle":
				indicator := "\u25CF"
				sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s %s", indicator, value)))
			case "text":
				display := value
				if display == "" {
					display = "(default)"
				}
				sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s", display)))
			default:
				sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s", value)))
			}
		}
		sb.WriteString("\n")
	}

	// Fill remaining
	contentHeight := height - 4
	rendered := sb.String()
	lineCount := strings.Count(rendered, "\n")
	for i := lineCount; i < contentHeight; i++ {
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(optionsWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(p.theme.Accent).
		Render(sb.String())
}

func (p *generalPage) renderInfo(infoWidth, height int) string {
	if infoWidth < 10 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(p.theme.Title.Render("Info"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", infoWidth-2)))
	sb.WriteString("\n\n")

	// Show hint for selected field
	if p.selectedIdx < len(generalFields) {
		field := generalFields[p.selectedIdx]
		sb.WriteString(p.theme.Normal.Render(field.HintText))
		sb.WriteString("\n")

		// For editor field, show resolved path
		if p.selectedIdx == 0 {
			sb.WriteString("\n")
			resolved := p.resolveEditor()
			sb.WriteString(p.theme.Dimmed.Render("Resolves to:"))
			sb.WriteString("\n")
			sb.WriteString(p.theme.Normal.Render("  " + resolved))
			sb.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(infoWidth).
		PaddingLeft(1).
		Render(sb.String())
}

func (p *generalPage) resolveEditor() string {
	if p.editor != "" {
		path, err := exec.LookPath(p.editor)
		if err == nil {
			return path
		}
		return p.editor + " (not found)"
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		path, err := exec.LookPath(editor)
		if err == nil {
			return path + " ($EDITOR)"
		}
		return editor + " ($EDITOR)"
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor + " ($VISUAL)"
	}
	return "vi (fallback)"
}
