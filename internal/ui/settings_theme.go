package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ThemeChangedMsg is emitted when the user changes the TUI theme.
type ThemeChangedMsg struct {
	PresetName string
	Theme      *Theme
}

// themePage implements the Theme settings page.
type themePage struct {
	theme   *Theme
	presets []ThemePresetInfo
	current int // index of currently selected preset
	dirty   bool
}

func newThemePage(theme *Theme, currentPreset string) *themePage {
	presets := AllThemePresets()
	current := 0
	for i, p := range presets {
		if p.Name == currentPreset {
			current = i
			break
		}
	}
	return &themePage{
		theme:   theme,
		presets: presets,
		current: current,
	}
}

func (p *themePage) Title() string { return "Theme" }

func (p *themePage) IsEditing() bool { return false }

func (p *themePage) FooterHints() string {
	return "[j/k] navigate  [Enter/→] select  [Tab] section  [Esc] close"
}

func (p *themePage) OnEnter() {}

func (p *themePage) OnLeave() tea.Cmd {
	if !p.dirty {
		return nil
	}
	p.dirty = false
	preset := p.presets[p.current]
	theme := ThemeFromPreset(preset.Name)
	return func() tea.Msg {
		return ThemeChangedMsg{
			PresetName: preset.Name,
			Theme:      theme,
		}
	}
}

func (p *themePage) Update(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "j", "down":
		if p.current < len(p.presets)-1 {
			p.current++
			p.dirty = true
		}
	case "k", "up":
		if p.current > 0 {
			p.current--
			p.dirty = true
		}
	case "enter", "l", "right":
		p.dirty = true
	case "h", "left":
		return nil // signal: go to sidebar
	}
	return nil
}

func (p *themePage) View(width, height int, _ *Theme) string {
	var sb strings.Builder

	sb.WriteString(p.theme.Title.Render("TUI Color Theme"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", width-4)))
	sb.WriteString("\n\n")

	for i, preset := range p.presets {
		isSelected := i == p.current

		// Indicator
		indicator := "  "
		if isSelected {
			indicator = "\u25B8 "
		}

		// Name + description
		nameStyle := p.theme.Normal
		descStyle := p.theme.Dimmed
		if isSelected {
			nameStyle = p.theme.Selected
			descStyle = lipgloss.NewStyle().
				Foreground(p.theme.Accent)
		}

		name := nameStyle.Render(indicator + preset.Name)
		desc := descStyle.Render("  " + preset.Description)

		sb.WriteString(name)
		sb.WriteString(desc)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Show a preview swatch of the selected theme's colors
	selectedPreset := p.presets[p.current]
	previewTheme := ThemeFromPreset(selectedPreset.Name)
	sb.WriteString(p.renderPreview(previewTheme, width-4))

	// Fill remaining height
	contentHeight := height - 4
	rendered := sb.String()
	lineCount := strings.Count(rendered, "\n")
	for i := lineCount; i < contentHeight; i++ {
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(width).
		PaddingLeft(1).
		Render(sb.String())
}

func (p *themePage) renderPreview(t *Theme, width int) string {
	var sb strings.Builder

	sb.WriteString(p.theme.Title.Render("Preview"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", width)))
	sb.WriteString("\n\n")

	// Color swatches
	swatchWidth := 3
	colors := []struct {
		label string
		color lipgloss.Color
	}{
		{"Primary", t.Primary},
		{"Secondary", t.Secondary},
		{"Accent", t.Accent},
		{"Success", t.Success},
		{"Warning", t.Warning},
		{"Error", t.Error},
	}

	for _, c := range colors {
		swatch := lipgloss.NewStyle().
			Background(c.color).
			Width(swatchWidth).
			Render("   ")
		label := t.Normal.Render(fmt.Sprintf(" %s", c.label))
		sb.WriteString(swatch)
		sb.WriteString(label)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Style previews
	sb.WriteString(t.Title.Render("Title Style"))
	sb.WriteString("  ")
	sb.WriteString(t.Dimmed.Render("Dimmed text"))
	sb.WriteString("  ")
	sb.WriteString(t.Selected.Render(" Selected "))
	sb.WriteString("\n")

	sb.WriteString(t.StatusDone.Render(IndicatorDone + " Done"))
	sb.WriteString("  ")
	sb.WriteString(t.StatusProgress.Render(IndicatorInProgress + " Running"))
	sb.WriteString("  ")
	sb.WriteString(t.StatusFailed.Render(IndicatorFailed + " Failed"))
	sb.WriteString("\n")

	return sb.String()
}
