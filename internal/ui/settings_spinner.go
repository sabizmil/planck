package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerSettingsChangedMsg is emitted when the user changes the spinner style.
type SpinnerSettingsChangedMsg struct {
	Style string
}

// spinnerPage implements the Spinner settings page with live preview.
type spinnerPage struct {
	theme *Theme

	// All available presets
	presets []SpinnerPreset

	// Current selection
	selectedIdx int
	style       string // preset name that will be persisted

	// Live preview animation
	previewFrame int

	// Scroll offset for the preset list (for long lists)
	scrollOffset int
}

func newSpinnerPage(theme *Theme, currentStyle string) *spinnerPage {
	presets := SpinnerPresets()
	selectedIdx := 0
	for i, p := range presets {
		if p.Name == currentStyle {
			selectedIdx = i
			break
		}
	}
	return &spinnerPage{
		theme:       theme,
		presets:     presets,
		selectedIdx: selectedIdx,
		style:       currentStyle,
	}
}

func (p *spinnerPage) Title() string { return "Spinner" }

func (p *spinnerPage) IsEditing() bool { return false }

func (p *spinnerPage) FooterHints() string {
	return "[j/k] navigate  [Enter/\u2192] select  [Tab] section  [Esc] close"
}

func (p *spinnerPage) OnEnter() {
	p.previewFrame = 0
}

func (p *spinnerPage) OnLeave() tea.Cmd {
	return func() tea.Msg {
		return SpinnerSettingsChangedMsg{
			Style: p.style,
		}
	}
}

func (p *spinnerPage) Update(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "j", "down":
		if p.selectedIdx < len(p.presets)-1 {
			p.selectedIdx++
			p.previewFrame = 0
		}

	case "k", "up":
		if p.selectedIdx > 0 {
			p.selectedIdx--
			p.previewFrame = 0
		}

	case "enter", "l", "right":
		p.style = p.presets[p.selectedIdx].Name

	case "h", "left":
		return nil // signal to parent: go to sidebar
	}

	return nil
}

// AdvancePreview advances the live preview frame. Called externally on SpinnerTickMsg.
func (p *spinnerPage) AdvancePreview() {
	preset := p.presets[p.selectedIdx]
	if len(preset.Frames) > 0 {
		p.previewFrame = (p.previewFrame + 1) % len(preset.Frames)
	}
}

func (p *spinnerPage) View(width, height int, theme *Theme) string {
	listWidth := width * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	if listWidth > 30 {
		listWidth = 30
	}
	previewWidth := width - listWidth - 3

	list := p.renderList(listWidth, height)
	preview := p.renderPreview(previewWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, list, preview)
}

func (p *spinnerPage) renderList(listWidth, height int) string {
	var sb strings.Builder

	sb.WriteString(p.theme.Title.Render("Spinner Style"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", listWidth)))
	sb.WriteString("\n")

	contentHeight := height - 4
	visibleItems := contentHeight - 2 // account for header lines

	// Adjust scroll offset to keep selected item visible
	if p.selectedIdx < p.scrollOffset {
		p.scrollOffset = p.selectedIdx
	}
	if p.selectedIdx >= p.scrollOffset+visibleItems {
		p.scrollOffset = p.selectedIdx - visibleItems + 1
	}

	end := p.scrollOffset + visibleItems
	if end > len(p.presets) {
		end = len(p.presets)
	}

	for i := p.scrollOffset; i < end; i++ {
		preset := p.presets[i]
		isSelected := i == p.selectedIdx
		isActive := preset.Name == p.style

		name := preset.Name
		if isActive {
			name += " \u2713"
		}

		if isSelected {
			sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" \u25B8 %s", name)))
		} else if isActive {
			sb.WriteString(lipgloss.NewStyle().
				Foreground(p.theme.Success).
				Render(fmt.Sprintf("   %s", name)))
		} else {
			sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s", name)))
		}
		sb.WriteString("\n")
	}

	// Scroll indicators
	if p.scrollOffset > 0 {
		sb.WriteString(p.theme.Dimmed.Render("   \u25B2 more"))
		sb.WriteString("\n")
	}
	if end < len(p.presets) {
		sb.WriteString(p.theme.Dimmed.Render("   \u25BC more"))
		sb.WriteString("\n")
	}

	// Fill remaining
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

func (p *spinnerPage) renderPreview(previewWidth, height int) string {
	if previewWidth < 10 {
		return ""
	}

	preset := p.presets[p.selectedIdx]
	var sb strings.Builder

	sb.WriteString(p.theme.Title.Render("Preview"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", previewWidth-2)))
	sb.WriteString("\n\n")

	// Live preview: show the current frame large and prominent
	if len(preset.Frames) > 0 {
		frame := preset.Frames[p.previewFrame%len(preset.Frames)]
		previewStyle := lipgloss.NewStyle().
			Foreground(p.theme.Accent).
			Bold(true)
		sb.WriteString("  ")
		sb.WriteString(previewStyle.Render(frame))
		sb.WriteString("  ")
		sb.WriteString(p.theme.Dimmed.Render(preset.Name))
		sb.WriteString("\n\n")

		// Simulated tab bar preview
		sb.WriteString(p.theme.Dimmed.Render("Tab preview:"))
		sb.WriteString("\n")
		tabPreview := fmt.Sprintf("  [1:Planning]  [2:%s Agent]", frame)
		sb.WriteString(lipgloss.NewStyle().
			Foreground(p.theme.Accent).
			Render(tabPreview))
		sb.WriteString("\n\n")
	}

	// Info section
	sb.WriteString(p.theme.Dimmed.Render("Interval:"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("  %s", preset.Interval)))
	sb.WriteString("\n\n")

	sb.WriteString(p.theme.Dimmed.Render("Frames:"))
	sb.WriteString("\n")
	// Show all frames with current one highlighted
	var frameDisplay strings.Builder
	frameDisplay.WriteString("  ")
	for i, f := range preset.Frames {
		if i > 0 {
			frameDisplay.WriteString(" ")
		}
		if i == p.previewFrame%len(preset.Frames) {
			frameDisplay.WriteString(lipgloss.NewStyle().
				Foreground(p.theme.Accent).
				Bold(true).
				Render(f))
		} else {
			frameDisplay.WriteString(p.theme.Dimmed.Render(f))
		}
		// Wrap long frame sequences
		if (i+1)%12 == 0 && i < len(preset.Frames)-1 {
			frameDisplay.WriteString("\n  ")
		}
	}
	sb.WriteString(frameDisplay.String())
	sb.WriteString("\n")

	return lipgloss.NewStyle().
		Width(previewWidth).
		PaddingLeft(1).
		Render(sb.String())
}
