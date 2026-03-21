package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DialogType represents the type of dialog
type DialogType int

const (
	DialogConfirm DialogType = iota
	DialogInput
	DialogSelect
	DialogScopePicker
	DialogPermission
)

// DialogResult represents the result of a dialog
type DialogResult struct {
	Confirmed bool
	Input     string
	Selected  int
}

// DialogOption represents a selectable option
type DialogOption struct {
	Label       string
	Description string
}

// Dialog displays modal dialogs
type Dialog struct {
	theme   *Theme
	visible bool
	dtype   DialogType
	title   string
	message string
	options []DialogOption
	cursor  int
	input   string
	width   int
	height  int

	onClose func(DialogResult)
}

// NewDialog creates a new dialog
func NewDialog(theme *Theme) *Dialog {
	return &Dialog{
		theme: theme,
	}
}

// Init initializes the dialog
func (d *Dialog) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (d *Dialog) Update(msg tea.Msg) (*Dialog, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch d.dtype {
		case DialogConfirm:
			switch msg.String() {
			case "y", "Y", "enter":
				d.close(DialogResult{Confirmed: true})
			case "n", "N", "esc":
				d.close(DialogResult{Confirmed: false})
			}

		case DialogInput:
			switch msg.String() {
			case "enter":
				d.close(DialogResult{Confirmed: true, Input: d.input})
			case "esc":
				d.close(DialogResult{Confirmed: false})
			case "backspace":
				if d.input != "" {
					d.input = d.input[:len(d.input)-1]
				}
			default:
				if len(msg.String()) == 1 {
					d.input += msg.String()
				}
			}

		case DialogSelect, DialogScopePicker:
			switch msg.String() {
			case "j", "down":
				d.cursor++
				if d.cursor >= len(d.options) {
					d.cursor = len(d.options) - 1
				}
			case "k", "up":
				d.cursor--
				if d.cursor < 0 {
					d.cursor = 0
				}
			case "enter":
				d.close(DialogResult{Confirmed: true, Selected: d.cursor})
			case "esc":
				d.close(DialogResult{Confirmed: false})
			}

		case DialogPermission:
			switch msg.String() {
			case "y", "Y":
				d.close(DialogResult{Confirmed: true})
			case "n", "N", "esc":
				d.close(DialogResult{Confirmed: false})
			}
		}
	}

	return d, nil
}

// View renders the dialog
func (d *Dialog) View() string {
	if !d.visible {
		return ""
	}

	var content strings.Builder

	// Title
	content.WriteString(d.theme.DialogTitle.Render(d.title))
	content.WriteString("\n")
	content.WriteString(d.theme.Dimmed.Render(strings.Repeat("─", d.dialogWidth()-4)))
	content.WriteString("\n\n")

	// Message
	if d.message != "" {
		content.WriteString(d.theme.Normal.Render(d.message))
		content.WriteString("\n\n")
	}

	// Type-specific content
	switch d.dtype {
	case DialogConfirm:
		content.WriteString(d.theme.KeyHint.Render("[Y] Yes  [N] No"))

	case DialogInput:
		content.WriteString(d.theme.Normal.Render(d.input))
		content.WriteString(d.theme.Selected.Render("▍"))
		content.WriteString("\n\n")
		content.WriteString(d.theme.KeyHint.Render("[Enter] Confirm  [Esc] Cancel"))

	case DialogSelect, DialogScopePicker:
		for i, opt := range d.options {
			var prefix string
			if i == d.cursor {
				prefix = d.theme.Selected.Render("● ")
			} else {
				prefix = d.theme.Dimmed.Render("○ ")
			}

			var label string
			if i == d.cursor {
				label = d.theme.Selected.Render(opt.Label)
			} else {
				label = d.theme.Normal.Render(opt.Label)
			}

			content.WriteString(prefix)
			content.WriteString(label)
			content.WriteString("\n")

			if opt.Description != "" {
				content.WriteString("  ")
				content.WriteString(d.theme.Dimmed.Render(opt.Description))
				content.WriteString("\n")
			}
		}
		content.WriteString("\n")
		content.WriteString(d.theme.KeyHint.Render("[Enter] Select  [Esc] Cancel"))

	case DialogPermission:
		content.WriteString(d.theme.Normal.Render("This will run autonomously.\n"))
		content.WriteString(d.theme.Normal.Render("Claude Code will be allowed to:\n\n"))
		content.WriteString(d.theme.StatusDone.Render("  ✓ Read/write files in project\n"))
		content.WriteString(d.theme.StatusDone.Render("  ✓ Run shell commands\n"))
		content.WriteString(d.theme.StatusDone.Render("  ✓ Make git commits\n\n"))
		content.WriteString(d.theme.Normal.Render("You can hijack the session anytime\n"))
		content.WriteString(d.theme.Normal.Render("with [Enter] to take control.\n\n"))
		content.WriteString(d.theme.KeyHint.Render("[Y] Approve & Start  [N] Cancel"))
	}

	return d.centerDialog(d.theme.Dialog.Render(content.String()))
}

func (d *Dialog) dialogWidth() int {
	return 40
}

func (d *Dialog) centerDialog(content string) string {
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	// Calculate position
	x := (d.width - contentWidth) / 2
	y := (d.height - contentHeight) / 2

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// Build centered output
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

func (d *Dialog) close(result DialogResult) {
	d.visible = false
	if d.onClose != nil {
		d.onClose(result)
	}
}

// Show methods for different dialog types

// ShowConfirm shows a confirmation dialog
func (d *Dialog) ShowConfirm(title, message string, onClose func(DialogResult)) {
	d.dtype = DialogConfirm
	d.title = title
	d.message = message
	d.visible = true
	d.onClose = onClose
}

// ShowInput shows an input dialog
func (d *Dialog) ShowInput(title, message string, onClose func(DialogResult)) {
	d.dtype = DialogInput
	d.title = title
	d.message = message
	d.input = ""
	d.visible = true
	d.onClose = onClose
}

// ShowSelect shows a selection dialog
func (d *Dialog) ShowSelect(title string, options []DialogOption, onClose func(DialogResult)) {
	d.dtype = DialogSelect
	d.title = title
	d.message = ""
	d.options = options
	d.cursor = 0
	d.visible = true
	d.onClose = onClose
}

// ShowScopePicker shows the execution scope picker
func (d *Dialog) ShowScopePicker(taskCount, phaseCount, totalTasks int, onClose func(DialogResult)) {
	d.dtype = DialogScopePicker
	d.title = "Execute Plan"
	d.message = ""
	d.options = []DialogOption{
		{Label: "This task only", Description: "Run just the selected task"},
		{Label: fmt.Sprintf("Current phase (%d tasks)", taskCount), Description: "Complete all tasks in this phase"},
		{Label: fmt.Sprintf("Entire plan (%d phases, %d tasks)", phaseCount, totalTasks), Description: "Execute the complete plan"},
	}
	d.cursor = 0
	d.visible = true
	d.onClose = onClose
}

// ShowPermissionDialog shows the permission pre-approval dialog
func (d *Dialog) ShowPermissionDialog(onClose func(DialogResult)) {
	d.dtype = DialogPermission
	d.title = "Pre-approve Permissions"
	d.message = ""
	d.visible = true
	d.onClose = onClose
}

// IsVisible returns whether the dialog is visible
func (d *Dialog) IsVisible() bool {
	return d.visible
}

// Hide hides the dialog
func (d *Dialog) Hide() {
	d.visible = false
}

// SetSize sets the dialog container size
func (d *Dialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}
