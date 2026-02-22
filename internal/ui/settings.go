package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// MarkdownStyleChangedMsg is emitted when the user changes markdown style settings.
type MarkdownStyleChangedMsg struct {
	Config    MarkdownStyleConfig
	StyleJSON []byte
}

// SettingsClosedMsg is emitted when the settings panel is closed.
type SettingsClosedMsg struct{}

// settingsCategory represents a settings section in the sidebar.
type settingsCategory struct {
	Name    string
	Enabled bool
}

// settingsFocus tracks which column has focus.
type settingsFocus int

const (
	focusSidebar settingsFocus = iota
	focusOptions
)

// Settings displays the settings overlay panel.
type Settings struct {
	theme   *Theme
	visible bool
	width   int
	height  int

	// Navigation
	focus      settingsFocus
	sidebarIdx int // selected category
	optionsIdx int // 0 = theme selector, 1+ = element rows
	optionsOffset int // scroll offset for options list
	previewOffset int // scroll offset for preview

	// Markdown style state
	registry    *StyleRegistry
	styleConfig MarkdownStyleConfig
	renderer    *glamour.TermRenderer
	rendered    string // cached preview render

	// Categories
	categories []settingsCategory
}

// NewSettings creates a new settings panel.
func NewSettings(theme *Theme, registry *StyleRegistry, cfg MarkdownStyleConfig) *Settings {
	s := &Settings{
		theme:       theme,
		registry:    registry,
		styleConfig: cfg,
		categories: []settingsCategory{
			{Name: "Markdown", Enabled: true},
			{Name: "General", Enabled: false},
			{Name: "Agents", Enabled: false},
			{Name: "Keybindings", Enabled: false},
		},
	}
	s.rebuildPreview()
	return s
}

// Init initializes the settings panel.
func (s *Settings) Init() tea.Cmd {
	return nil
}

// Update handles messages for the settings panel.
func (s *Settings) Update(msg tea.Msg) (*Settings, tea.Cmd) {
	if !s.visible {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return s.handleKey(msg)
	}

	return s, nil
}

func (s *Settings) handleKey(msg tea.KeyMsg) (*Settings, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc", "s":
		s.visible = false
		// Emit final style on close
		styleJSON := s.registry.ComposeStyle(s.styleConfig)
		return s, func() tea.Msg {
			return tea.BatchMsg{
				func() tea.Msg {
					return MarkdownStyleChangedMsg{
						Config:    s.styleConfig,
						StyleJSON: styleJSON,
					}
				},
				func() tea.Msg { return SettingsClosedMsg{} },
			}
		}

	case "tab":
		if s.focus == focusSidebar {
			s.focus = focusOptions
		} else {
			s.focus = focusSidebar
		}

	case "j", "down":
		s.moveDown()

	case "k", "up":
		s.moveUp()

	case "enter", "l", "right":
		if s.focus == focusSidebar {
			// Select category, move to options
			if s.categories[s.sidebarIdx].Enabled {
				s.focus = focusOptions
			}
		} else {
			s.cycleForward()
		}

	case "h", "left":
		if s.focus == focusOptions {
			if s.optionsIdx == 0 {
				// On theme selector, go back to sidebar
				s.focus = focusSidebar
			} else {
				s.cyclePrevious()
			}
		}

	case "R":
		// Reset ALL overrides
		if s.focus == focusOptions {
			s.styleConfig.Overrides = map[ElementType]ThemeName{}
			s.rebuildPreview()
		}

	case "r":
		// Reset current element override
		if s.focus == focusOptions && s.optionsIdx > 0 {
			elemIdx := s.optionsIdx - 1
			elements := AllElements()
			if elemIdx < len(elements) {
				delete(s.styleConfig.Overrides, elements[elemIdx])
				s.rebuildPreview()
			}
		}

	case "ctrl+j":
		s.previewOffset++
		s.clampPreviewOffset()

	case "ctrl+k":
		s.previewOffset--
		if s.previewOffset < 0 {
			s.previewOffset = 0
		}
	}

	return s, nil
}

func (s *Settings) moveDown() {
	if s.focus == focusSidebar {
		if s.sidebarIdx < len(s.categories)-1 {
			s.sidebarIdx++
		}
	} else {
		maxIdx := len(AllElements()) // 0=theme, 1..N=elements
		if s.optionsIdx < maxIdx {
			s.optionsIdx++
		}
		s.ensureOptionsVisible()
	}
}

func (s *Settings) moveUp() {
	if s.focus == focusSidebar {
		if s.sidebarIdx > 0 {
			s.sidebarIdx--
		}
	} else {
		if s.optionsIdx > 0 {
			s.optionsIdx--
		}
		s.ensureOptionsVisible()
	}
}

func (s *Settings) cycleForward() {
	themes := AllThemes()
	if s.optionsIdx == 0 {
		// Cycle global theme
		current := s.styleConfig.GlobalTheme
		idx := 0
		for i, t := range themes {
			if t == current {
				idx = i
				break
			}
		}
		s.styleConfig.GlobalTheme = themes[(idx+1)%len(themes)]
		s.rebuildPreview()
	} else {
		// Cycle element override
		elemIdx := s.optionsIdx - 1
		elements := AllElements()
		if elemIdx >= len(elements) {
			return
		}
		elem := elements[elemIdx]
		current, hasOverride := s.styleConfig.Overrides[elem]
		if !hasOverride {
			current = s.styleConfig.GlobalTheme
		}
		idx := 0
		for i, t := range themes {
			if t == current {
				idx = i
				break
			}
		}
		next := themes[(idx+1)%len(themes)]
		if next == s.styleConfig.GlobalTheme && hasOverride {
			// Cycling back to global = remove override
			delete(s.styleConfig.Overrides, elem)
		} else if next != s.styleConfig.GlobalTheme {
			if s.styleConfig.Overrides == nil {
				s.styleConfig.Overrides = map[ElementType]ThemeName{}
			}
			s.styleConfig.Overrides[elem] = next
		} else {
			// Already on global, cycle to next
			if s.styleConfig.Overrides == nil {
				s.styleConfig.Overrides = map[ElementType]ThemeName{}
			}
			s.styleConfig.Overrides[elem] = next
		}
		s.rebuildPreview()
	}
}

func (s *Settings) cyclePrevious() {
	themes := AllThemes()
	if s.optionsIdx == 0 {
		current := s.styleConfig.GlobalTheme
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
		s.styleConfig.GlobalTheme = themes[newIdx]
		s.rebuildPreview()
	} else {
		elemIdx := s.optionsIdx - 1
		elements := AllElements()
		if elemIdx >= len(elements) {
			return
		}
		elem := elements[elemIdx]
		current, hasOverride := s.styleConfig.Overrides[elem]
		if !hasOverride {
			current = s.styleConfig.GlobalTheme
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
		if next == s.styleConfig.GlobalTheme {
			delete(s.styleConfig.Overrides, elem)
		} else {
			if s.styleConfig.Overrides == nil {
				s.styleConfig.Overrides = map[ElementType]ThemeName{}
			}
			s.styleConfig.Overrides[elem] = next
		}
		s.rebuildPreview()
	}
}

func (s *Settings) ensureOptionsVisible() {
	visibleRows := s.optionsVisibleRows()
	if s.optionsIdx < s.optionsOffset {
		s.optionsOffset = s.optionsIdx
	}
	if s.optionsIdx >= s.optionsOffset+visibleRows {
		s.optionsOffset = s.optionsIdx - visibleRows + 1
	}
}

func (s *Settings) optionsVisibleRows() int {
	// header(1) + separator(1) + "Theme" label(1) + theme box(1) + blank(1) + "Element Overrides" label(1) + separator(1) = 7 lines of chrome
	rows := s.height - 7 - 4 // 4 for outer chrome (title, separator, footer, border)
	if rows < 5 {
		rows = 5
	}
	return rows
}

func (s *Settings) clampPreviewOffset() {
	previewLines := strings.Count(s.rendered, "\n") + 1
	maxOffset := previewLines - (s.height - 6)
	if maxOffset < 0 {
		maxOffset = 0
	}
	if s.previewOffset > maxOffset {
		s.previewOffset = maxOffset
	}
}

func (s *Settings) rebuildPreview() {
	styleJSON := s.registry.ComposeStyle(s.styleConfig)
	previewWidth := s.previewWidth()
	if previewWidth < 20 {
		previewWidth = 40
	}
	renderer, err := NewMarkdownRendererWithStyle(styleJSON, previewWidth-4)
	if err != nil {
		s.rendered = "Error rendering preview"
		return
	}
	s.renderer = renderer
	rendered, err := renderer.Render(PreviewMarkdown)
	if err != nil {
		s.rendered = "Error: " + err.Error()
		return
	}
	s.rendered = strings.TrimSpace(rendered)
}

func (s *Settings) previewWidth() int {
	sidebarW := 16
	optionsW := 32
	return s.width - sidebarW - optionsW - 6 // borders and padding
}

// View renders the settings panel.
func (s *Settings) View() string {
	if !s.visible {
		return ""
	}

	sidebar := s.renderSidebar()
	options := s.renderOptions()
	preview := s.renderPreview()

	// Join columns
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, options, preview)

	// Title bar
	titleStyle := s.theme.DialogTitle.Bold(true)
	title := titleStyle.Render("Settings")
	closeHint := s.theme.Dimmed.Render("Esc close")
	titlePad := s.width - lipgloss.Width(title) - lipgloss.Width(closeHint) - 4
	if titlePad < 1 {
		titlePad = 1
	}
	titleBar := title + strings.Repeat(" ", titlePad) + closeHint

	// Footer
	footerHints := s.theme.Dimmed.Render("[j/k] navigate  [Enter/\u2192] change  [\u2190] prev  [Tab] section  [r] reset  [R] reset all  [Esc] close")

	var b strings.Builder
	b.WriteString(titleBar)
	b.WriteString("\n")
	b.WriteString(s.theme.Dimmed.Render(safeRepeat("\u2500", s.width-4)))
	b.WriteString("\n")
	b.WriteString(content)
	b.WriteString("\n")
	b.WriteString(s.theme.Dimmed.Render(safeRepeat("\u2500", s.width-4)))
	b.WriteString("\n")
	b.WriteString(footerHints)

	// Wrap in dialog style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.theme.Accent).
		Width(s.width - 2).
		Height(s.height - 2).
		Padding(0, 1)

	return boxStyle.Render(b.String())
}

func (s *Settings) renderSidebar() string {
	sidebarWidth := 14
	var sb strings.Builder

	sb.WriteString(s.theme.Title.Render("Categories"))
	sb.WriteString("\n")
	sb.WriteString(s.theme.Dimmed.Render(safeRepeat("\u2500", sidebarWidth)))
	sb.WriteString("\n")

	for i, cat := range s.categories {
		var line string
		if i == s.sidebarIdx && s.focus == focusSidebar {
			line = s.theme.Selected.Render(fmt.Sprintf(" \u25B8 %s", cat.Name))
		} else if i == s.sidebarIdx {
			line = s.theme.Normal.Render(fmt.Sprintf(" \u25B8 %s", cat.Name))
		} else if !cat.Enabled {
			line = s.theme.Dimmed.Render(fmt.Sprintf("   %s", cat.Name))
		} else {
			line = s.theme.Normal.Render(fmt.Sprintf("   %s", cat.Name))
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// Fill remaining height
	contentHeight := s.height - 6
	usedLines := 2 + len(s.categories) // header + separator + categories
	for i := usedLines; i < contentHeight; i++ {
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(sidebarWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(s.theme.Accent).
		Render(sb.String())
}

func (s *Settings) renderOptions() string {
	optionsWidth := 30
	var sb strings.Builder

	sb.WriteString(s.theme.Title.Render("Markdown Formatting"))
	sb.WriteString("\n")
	sb.WriteString(s.theme.Dimmed.Render(safeRepeat("\u2500", optionsWidth)))
	sb.WriteString("\n")

	// Theme selector row
	themeLabel := s.theme.Dimmed.Render("Theme")
	sb.WriteString(themeLabel)
	sb.WriteString("\n")

	themeName := ThemeDisplayName(s.styleConfig.GlobalTheme)
	if s.optionsIdx == 0 && s.focus == focusOptions {
		sb.WriteString(s.theme.Selected.Render(fmt.Sprintf(" \u25C0 %s \u25B6", themeName)))
	} else {
		sb.WriteString(s.theme.Normal.Render(fmt.Sprintf("   %s", themeName)))
	}
	sb.WriteString("\n\n")

	// Element overrides header
	sb.WriteString(s.theme.Dimmed.Render("Element Overrides"))
	sb.WriteString("\n")
	sb.WriteString(s.theme.Dimmed.Render(safeRepeat("\u2500", optionsWidth)))
	sb.WriteString("\n")

	elements := AllElements()
	visibleRows := s.optionsVisibleRows()
	start := s.optionsOffset
	end := s.optionsOffset + visibleRows
	if end > len(elements) {
		end = len(elements)
	}
	if start > len(elements) {
		start = len(elements)
	}

	for i := start; i < end; i++ {
		elem := elements[i]
		displayName := ElementDisplayName(elem)
		override, hasOverride := s.styleConfig.Overrides[elem]

		var themeName string
		var indicator string
		if hasOverride {
			themeName = ThemeDisplayName(override)
			indicator = "\u25D0" // ◐ overridden
		} else {
			themeName = ThemeDisplayName(s.styleConfig.GlobalTheme)
			indicator = "\u25CF" // ● synced
		}

		rowIdx := i + 1 // +1 because 0 is the theme selector
		isSelected := rowIdx == s.optionsIdx && s.focus == focusOptions

		// Truncate display name if needed
		maxNameLen := 13
		if len(displayName) > maxNameLen {
			displayName = displayName[:maxNameLen]
		}

		// Format: name + padding + theme + indicator
		nameStr := fmt.Sprintf("%-*s", maxNameLen, displayName)
		maxThemeLen := optionsWidth - maxNameLen - 4
		if len(themeName) > maxThemeLen {
			themeName = themeName[:maxThemeLen]
		}

		if isSelected {
			line := fmt.Sprintf("\u25B8%s %s %s", nameStr, themeName, indicator)
			sb.WriteString(s.theme.Selected.Render(line))
		} else {
			nameRendered := s.theme.Normal.Render(" " + nameStr)
			var themeRendered string
			if hasOverride {
				themeRendered = s.theme.StatusProgress.Render(themeName + " " + indicator)
			} else {
				themeRendered = s.theme.Dimmed.Render(themeName + " " + indicator)
			}
			sb.WriteString(nameRendered + " " + themeRendered)
		}
		sb.WriteString("\n")
	}

	// Scroll indicators
	if s.optionsOffset > 0 {
		sb.WriteString(s.theme.Dimmed.Render("  \u25B2 more above"))
		sb.WriteString("\n")
	}
	if end < len(elements) {
		sb.WriteString(s.theme.Dimmed.Render("  \u25BC more below"))
		sb.WriteString("\n")
	}

	// Fill remaining height
	contentHeight := s.height - 6
	rendered := sb.String()
	lineCount := strings.Count(rendered, "\n")
	for i := lineCount; i < contentHeight; i++ {
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(optionsWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(s.theme.Accent).
		Render(sb.String())
}

func (s *Settings) renderPreview() string {
	previewW := s.previewWidth()
	if previewW < 10 {
		previewW = 10
	}

	var sb strings.Builder
	sb.WriteString(s.theme.Title.Render("Preview"))
	sb.WriteString("\n")
	sb.WriteString(s.theme.Dimmed.Render(safeRepeat("\u2500", previewW-2)))
	sb.WriteString("\n")

	if s.rendered == "" {
		sb.WriteString(s.theme.Dimmed.Render("No preview available"))
	} else {
		lines := strings.Split(s.rendered, "\n")
		contentHeight := s.height - 8
		if contentHeight < 5 {
			contentHeight = 5
		}

		start := s.previewOffset
		end := s.previewOffset + contentHeight
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
			// Truncate wide lines to prevent overflow
			if lipgloss.Width(line) > previewW-2 {
				line = truncate(line, previewW-4)
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}

		// Fill remaining
		for i := end - start; i < contentHeight; i++ {
			sb.WriteString("\n")
		}
	}

	return lipgloss.NewStyle().
		Width(previewW).
		PaddingLeft(1).
		Render(sb.String())
}

// Toggle toggles the settings panel visibility.
func (s *Settings) Toggle() {
	s.visible = !s.visible
	if s.visible {
		s.focus = focusOptions
		s.optionsIdx = 0
		s.optionsOffset = 0
		s.previewOffset = 0
		s.rebuildPreview()
	}
}

// IsVisible returns whether the settings panel is visible.
func (s *Settings) IsVisible() bool {
	return s.visible
}

// SetSize sets the container size.
func (s *Settings) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.rebuildPreview()
}

// StyleConfig returns the current style configuration.
func (s *Settings) StyleConfig() MarkdownStyleConfig {
	return s.styleConfig
}
