package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// markdownPage implements the Markdown settings page.
type markdownPage struct {
	theme *Theme

	// Markdown style state
	registry    *StyleRegistry
	styleConfig MarkdownStyleConfig
	renderer    *glamour.TermRenderer
	rendered    string // cached preview render

	// Navigation
	optionsIdx    int
	optionsOffset int
	previewOffset int

	// Cached dimensions
	lastWidth  int
	lastHeight int
}

func newMarkdownPage(theme *Theme, registry *StyleRegistry, cfg MarkdownStyleConfig) *markdownPage {
	p := &markdownPage{
		theme:       theme,
		registry:    registry,
		styleConfig: cfg,
	}
	return p
}

func (p *markdownPage) Title() string { return "Markdown Formatting" }

func (p *markdownPage) IsEditing() bool { return false }

func (p *markdownPage) FooterHints() string {
	return "[j/k] navigate  [Enter/\u2192] change  [\u2190] prev  [Tab] section  [r] reset  [R] reset all  [Esc] close"
}

func (p *markdownPage) OnEnter() {
	p.optionsIdx = 0
	p.optionsOffset = 0
	p.previewOffset = 0
	p.rebuildPreview()
}

func (p *markdownPage) OnLeave() tea.Cmd {
	styleJSON := p.registry.ComposeStyle(p.styleConfig)
	return func() tea.Msg {
		return MarkdownStyleChangedMsg{
			Config:    p.styleConfig,
			StyleJSON: styleJSON,
		}
	}
}

func (p *markdownPage) Update(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "j", "down":
		maxIdx := len(AllElements())
		if p.optionsIdx < maxIdx {
			p.optionsIdx++
		}
		p.ensureOptionsVisible()

	case "k", "up":
		if p.optionsIdx > 0 {
			p.optionsIdx--
		}
		p.ensureOptionsVisible()

	case "enter", "l", "right":
		p.cycleForward()

	case "h", "left":
		if p.optionsIdx == 0 {
			return nil // signal: didn't consume, go to sidebar
		}
		p.cyclePrevious()
		return func() tea.Msg { return nil } // consumed

	case "R":
		p.styleConfig.Overrides = map[ElementType]ThemeName{}
		p.rebuildPreview()

	case "r":
		if p.optionsIdx > 0 {
			elemIdx := p.optionsIdx - 1
			elements := AllElements()
			if elemIdx < len(elements) {
				delete(p.styleConfig.Overrides, elements[elemIdx])
				p.rebuildPreview()
			}
		}

	case "ctrl+j":
		p.previewOffset++
		p.clampPreviewOffset()

	case "ctrl+k":
		p.previewOffset--
		if p.previewOffset < 0 {
			p.previewOffset = 0
		}
	}

	return nil
}

func (p *markdownPage) View(width, height int, theme *Theme) string {
	p.lastWidth = width
	p.lastHeight = height

	optionsWidth := 30
	previewWidth := width - optionsWidth - 3 // border + padding

	// Rebuild preview if dimensions changed
	if p.rendered == "" {
		p.rebuildPreview()
	}

	options := p.renderOptions(optionsWidth, height)
	preview := p.renderPreview(previewWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, options, preview)
}

func (p *markdownPage) renderOptions(optionsWidth, height int) string {
	var sb strings.Builder

	sb.WriteString(p.theme.Title.Render("Markdown Formatting"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", optionsWidth)))
	sb.WriteString("\n")

	// Theme selector row
	sb.WriteString(p.theme.Dimmed.Render("Theme"))
	sb.WriteString("\n")

	themeName := ThemeDisplayName(p.styleConfig.GlobalTheme)
	if p.optionsIdx == 0 {
		sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" \u25C0 %s \u25B6", themeName)))
	} else {
		sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s", themeName)))
	}
	sb.WriteString("\n\n")

	// Element overrides header
	sb.WriteString(p.theme.Dimmed.Render("Element Overrides"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", optionsWidth)))
	sb.WriteString("\n")

	elements := AllElements()
	visibleRows := p.optionsVisibleRows(height)
	start := p.optionsOffset
	end := p.optionsOffset + visibleRows
	if end > len(elements) {
		end = len(elements)
	}
	if start > len(elements) {
		start = len(elements)
	}

	for i := start; i < end; i++ {
		elem := elements[i]
		displayName := ElementDisplayName(elem)
		override, hasOverride := p.styleConfig.Overrides[elem]

		var tn string
		var indicator string
		if hasOverride {
			tn = ThemeDisplayName(override)
			indicator = "\u25D0"
		} else {
			tn = ThemeDisplayName(p.styleConfig.GlobalTheme)
			indicator = "\u25CF"
		}

		rowIdx := i + 1
		isSelected := rowIdx == p.optionsIdx

		maxNameLen := 13
		if len(displayName) > maxNameLen {
			displayName = displayName[:maxNameLen]
		}

		nameStr := fmt.Sprintf("%-*s", maxNameLen, displayName)
		maxThemeLen := optionsWidth - maxNameLen - 4
		if len(tn) > maxThemeLen {
			tn = tn[:maxThemeLen]
		}

		if isSelected {
			line := fmt.Sprintf("\u25B8%s %s %s", nameStr, tn, indicator)
			sb.WriteString(p.theme.Selected.Render(line))
		} else {
			nameRendered := p.theme.Normal.Render(" " + nameStr)
			var themeRendered string
			if hasOverride {
				themeRendered = p.theme.StatusProgress.Render(tn + " " + indicator)
			} else {
				themeRendered = p.theme.Dimmed.Render(tn + " " + indicator)
			}
			sb.WriteString(nameRendered + " " + themeRendered)
		}
		sb.WriteString("\n")
	}

	if p.optionsOffset > 0 {
		sb.WriteString(p.theme.Dimmed.Render("  \u25B2 more above"))
		sb.WriteString("\n")
	}
	if end < len(elements) {
		sb.WriteString(p.theme.Dimmed.Render("  \u25BC more below"))
		sb.WriteString("\n")
	}

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

func (p *markdownPage) renderPreview(previewW, height int) string {
	if previewW < 10 {
		previewW = 10
	}

	var sb strings.Builder
	sb.WriteString(p.theme.Title.Render("Preview"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", previewW-2)))
	sb.WriteString("\n")

	if p.rendered == "" {
		sb.WriteString(p.theme.Dimmed.Render("No preview available"))
	} else {
		lines := strings.Split(p.rendered, "\n")
		contentHeight := height - 6
		if contentHeight < 5 {
			contentHeight = 5
		}

		start := p.previewOffset
		end := p.previewOffset + contentHeight
		if start >= len(lines) {
			start = len(lines) - 1
		}
		if start < 0 {
			start = 0
		}
		if end > len(lines) {
			end = len(lines)
		}

		for i := start; i < end; i++ {
			line := lines[i]
			if lipgloss.Width(line) > previewW-2 {
				line = truncate(line, previewW-4)
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}

		for i := end - start; i < contentHeight; i++ {
			sb.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(previewW).
		PaddingLeft(1).
		Render(sb.String())
}

func (p *markdownPage) cycleForward() {
	themes := AllThemes()
	if p.optionsIdx == 0 {
		current := p.styleConfig.GlobalTheme
		idx := 0
		for i, t := range themes {
			if t == current {
				idx = i
				break
			}
		}
		p.styleConfig.GlobalTheme = themes[(idx+1)%len(themes)]
		p.rebuildPreview()
	} else {
		elemIdx := p.optionsIdx - 1
		elements := AllElements()
		if elemIdx >= len(elements) {
			return
		}
		elem := elements[elemIdx]
		current, hasOverride := p.styleConfig.Overrides[elem]
		if !hasOverride {
			current = p.styleConfig.GlobalTheme
		}
		idx := 0
		for i, t := range themes {
			if t == current {
				idx = i
				break
			}
		}
		next := themes[(idx+1)%len(themes)]
		if next == p.styleConfig.GlobalTheme && hasOverride {
			delete(p.styleConfig.Overrides, elem)
		} else if next != p.styleConfig.GlobalTheme {
			if p.styleConfig.Overrides == nil {
				p.styleConfig.Overrides = map[ElementType]ThemeName{}
			}
			p.styleConfig.Overrides[elem] = next
		} else {
			if p.styleConfig.Overrides == nil {
				p.styleConfig.Overrides = map[ElementType]ThemeName{}
			}
			p.styleConfig.Overrides[elem] = next
		}
		p.rebuildPreview()
	}
}

func (p *markdownPage) cyclePrevious() {
	themes := AllThemes()
	if p.optionsIdx == 0 {
		current := p.styleConfig.GlobalTheme
		idx := 0
		for i, t := range themes {
			if t == current {
				idx = i
				break
			}
		}
		newIdx := idx - 1
		if newIdx < 0 {
			newIdx = len(themes) - 1
		}
		p.styleConfig.GlobalTheme = themes[newIdx]
		p.rebuildPreview()
	} else {
		elemIdx := p.optionsIdx - 1
		elements := AllElements()
		if elemIdx >= len(elements) {
			return
		}
		elem := elements[elemIdx]
		current, hasOverride := p.styleConfig.Overrides[elem]
		if !hasOverride {
			current = p.styleConfig.GlobalTheme
		}
		idx := 0
		for i, t := range themes {
			if t == current {
				idx = i
				break
			}
		}
		newIdx := idx - 1
		if newIdx < 0 {
			newIdx = len(themes) - 1
		}
		next := themes[newIdx]
		if next == p.styleConfig.GlobalTheme {
			delete(p.styleConfig.Overrides, elem)
		} else {
			if p.styleConfig.Overrides == nil {
				p.styleConfig.Overrides = map[ElementType]ThemeName{}
			}
			p.styleConfig.Overrides[elem] = next
		}
		p.rebuildPreview()
	}
}

func (p *markdownPage) ensureOptionsVisible() {
	visibleRows := p.optionsVisibleRows(p.lastHeight)
	if p.optionsIdx < p.optionsOffset {
		p.optionsOffset = p.optionsIdx
	}
	if p.optionsIdx >= p.optionsOffset+visibleRows {
		p.optionsOffset = p.optionsIdx - visibleRows + 1
	}
}

func (p *markdownPage) optionsVisibleRows(height int) int {
	rows := height - 7 - 4
	if rows < 5 {
		rows = 5
	}
	return rows
}

func (p *markdownPage) clampPreviewOffset() {
	previewLines := strings.Count(p.rendered, "\n") + 1
	maxOffset := previewLines - (p.lastHeight - 6)
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.previewOffset > maxOffset {
		p.previewOffset = maxOffset
	}
}

func (p *markdownPage) rebuildPreview() {
	styleJSON := p.registry.ComposeStyle(p.styleConfig)
	previewWidth := p.lastWidth - 30 - 6
	if previewWidth < 20 {
		previewWidth = 40
	}
	renderer, err := NewMarkdownRendererWithStyle(styleJSON, previewWidth-4)
	if err != nil {
		p.rendered = "Error rendering preview"
		return
	}
	p.renderer = renderer
	rendered, err := renderer.Render(PreviewMarkdown)
	if err != nil {
		p.rendered = "Error: " + err.Error()
		return
	}
	p.rendered = strings.TrimSpace(rendered)
}
