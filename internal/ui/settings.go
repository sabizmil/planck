package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MarkdownStyleChangedMsg is emitted when the user changes markdown style settings.
type MarkdownStyleChangedMsg struct {
	Config    MarkdownStyleConfig
	StyleJSON []byte
}

// GeneralSettingsChangedMsg is emitted when the user changes general settings.
type GeneralSettingsChangedMsg struct {
	Editor         string
	Bell           bool
	Backend        string
	DefaultScope   string
	AutoAdvance    bool
	PermissionMode string
	SidebarWidth   int
}

// AgentsSettingsChangedMsg is emitted when the user changes agent settings.
type AgentsSettingsChangedMsg struct {
	Agents map[string]AgentSettingsConfig
}

// AgentSettingsConfig holds the editable agent settings.
type AgentSettingsConfig struct {
	Command      string
	Label        string
	PlanningArgs []string
	Default      bool
}

// SettingsClosedMsg is emitted when the settings panel is closed.
type SettingsClosedMsg struct{}

// settingsCategory represents a settings section in the sidebar.
type settingsCategory struct {
	Name string
	Page settingsPage
}

// settingsPage is the interface for individual settings pages.
type settingsPage interface {
	Title() string
	Update(msg tea.KeyMsg) tea.Cmd
	View(width, height int, theme *Theme) string
	FooterHints() string
	OnEnter()
	OnLeave() tea.Cmd
	IsEditing() bool // true when a text input has focus (suppress nav keys)
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
	sidebarIdx int

	// Categories
	categories []settingsCategory
}

// NewSettings creates a new settings panel.
func NewSettings(theme *Theme, registry *StyleRegistry, mdCfg MarkdownStyleConfig, generalCfg GeneralSettingsChangedMsg, agentsCfg map[string]AgentSettingsConfig, spinnerStyle string) *Settings {
	s := &Settings{
		theme: theme,
		categories: []settingsCategory{
			{Name: "Markdown", Page: newMarkdownPage(theme, registry, mdCfg)},
			{Name: "General", Page: newGeneralPage(theme, generalCfg)},
			{Name: "Agents", Page: newAgentsPage(theme, agentsCfg)},
			{Name: "Keys", Page: newKeybindingsPage(theme)},
			{Name: "Spinner", Page: newSpinnerPage(theme, spinnerStyle)},
		},
	}
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
	case SpinnerTickMsg:
		// Forward spinner ticks to the spinner page for live preview animation
		if sp, ok := s.categories[s.sidebarIdx].Page.(*spinnerPage); ok {
			sp.AdvancePreview()
		}
	}

	return s, nil
}

func (s *Settings) handleKey(msg tea.KeyMsg) (*Settings, tea.Cmd) {
	page := s.categories[s.sidebarIdx].Page

	// If the active page is editing (text input mode), delegate everything to it
	if page.IsEditing() {
		cmd := page.Update(msg)
		return s, cmd
	}

	key := msg.String()

	switch key {
	case "esc", "s":
		s.visible = false
		// Emit close messages from all pages, plus SettingsClosedMsg
		var batch []tea.Cmd
		for _, cat := range s.categories {
			if cmd := cat.Page.OnLeave(); cmd != nil {
				batch = append(batch, cmd)
			}
		}
		batch = append(batch, func() tea.Msg { return SettingsClosedMsg{} })
		return s, tea.Batch(batch...)

	case "tab":
		if s.focus == focusSidebar {
			s.focus = focusOptions
		} else {
			s.focus = focusSidebar
		}

	case "j", "down":
		if s.focus == focusSidebar {
			if s.sidebarIdx < len(s.categories)-1 {
				s.sidebarIdx++
				s.categories[s.sidebarIdx].Page.OnEnter()
			}
		} else {
			page.Update(msg)
		}

	case "k", "up":
		if s.focus == focusSidebar {
			if s.sidebarIdx > 0 {
				s.sidebarIdx--
				s.categories[s.sidebarIdx].Page.OnEnter()
			}
		} else {
			page.Update(msg)
		}

	case "enter", "l", "right":
		if s.focus == focusSidebar {
			s.focus = focusOptions
			page.OnEnter()
		} else {
			page.Update(msg)
		}

	case "h", "left":
		if s.focus == focusOptions {
			// Let page handle it first — if page doesn't consume it, go to sidebar
			cmd := page.Update(msg)
			if cmd != nil {
				return s, cmd
			}
			s.focus = focusSidebar
		}

	default:
		// Delegate to the active page
		if s.focus == focusOptions {
			cmd := page.Update(msg)
			return s, cmd
		}
	}

	return s, nil
}

// View renders the settings panel.
func (s *Settings) View() string {
	if !s.visible {
		return ""
	}

	sidebar := s.renderSidebar()

	page := s.categories[s.sidebarIdx].Page
	pageView := page.View(s.width-16-4, s.height-6, s.theme) // 16=sidebar, 4=chrome

	// Join columns
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, pageView)

	// Title bar
	titleStyle := s.theme.DialogTitle.Bold(true)
	title := titleStyle.Render("Settings")
	closeHint := s.theme.Dimmed.Render("Esc close")
	titlePad := s.width - lipgloss.Width(title) - lipgloss.Width(closeHint) - 4
	if titlePad < 1 {
		titlePad = 1
	}
	titleBar := title + strings.Repeat(" ", titlePad) + closeHint

	// Footer — page-specific hints
	footerHints := s.theme.Dimmed.Render(page.FooterHints())

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
		} else {
			line = s.theme.Normal.Render(fmt.Sprintf("   %s", cat.Name))
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// Fill remaining height
	contentHeight := s.height - 6
	usedLines := 2 + len(s.categories)
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

// Toggle toggles the settings panel visibility.
func (s *Settings) Toggle() {
	s.visible = !s.visible
	if s.visible {
		s.focus = focusOptions
		s.categories[s.sidebarIdx].Page.OnEnter()
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
}

// StyleConfig returns the current markdown style configuration.
func (s *Settings) StyleConfig() MarkdownStyleConfig {
	if mp, ok := s.categories[0].Page.(*markdownPage); ok {
		return mp.styleConfig
	}
	return MarkdownStyleConfig{}
}
