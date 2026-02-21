package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a tab in the tab bar
type Tab int

const (
	TabPlanning Tab = iota
	TabAgent
)

// String returns the display name of the tab
func (t Tab) String() string {
	switch t {
	case TabPlanning:
		return "Planning"
	case TabAgent:
		return "Agent"
	default:
		return "Unknown"
	}
}

// TabBar displays the tab bar at the top of the screen
type TabBar struct {
	theme      *Theme
	activeTab  Tab
	folderPath string
	width      int
}

// NewTabBar creates a new tab bar
func NewTabBar(theme *Theme) *TabBar {
	return &TabBar{
		theme:     theme,
		activeTab: TabPlanning,
	}
}

// Init initializes the tab bar
func (t *TabBar) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (t *TabBar) Update(msg tea.Msg) (*TabBar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// Cycle through tabs
			if t.activeTab == TabPlanning {
				t.activeTab = TabAgent
			} else {
				t.activeTab = TabPlanning
			}
		case "1":
			t.activeTab = TabPlanning
		case "2":
			t.activeTab = TabAgent
		}
	}
	return t, nil
}

// View renders the tab bar
func (t *TabBar) View() string {
	// Tab styles
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.theme.Accent).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 2)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(t.theme.Secondary).
		Padding(0, 2)

	// Build tabs
	var tabs strings.Builder

	tabs.WriteString("[")
	if t.activeTab == TabPlanning {
		tabs.WriteString(activeStyle.Render("Planning"))
	} else {
		tabs.WriteString(inactiveStyle.Render("Planning"))
	}
	tabs.WriteString("]  [")
	if t.activeTab == TabAgent {
		tabs.WriteString(activeStyle.Render("Agent"))
	} else {
		tabs.WriteString(inactiveStyle.Render("Agent"))
	}
	tabs.WriteString("]")

	// Add folder path on the right
	tabsStr := tabs.String()
	tabsWidth := lipgloss.Width(tabsStr)

	// Folder path
	folderStyle := t.theme.Dimmed
	folder := t.folderPath
	if folder == "" {
		folder = "No folder selected"
	}

	// Truncate folder if needed
	maxFolderWidth := t.width - tabsWidth - 4
	if maxFolderWidth > 0 && len(folder) > maxFolderWidth {
		folder = "..." + folder[len(folder)-maxFolderWidth+3:]
	}

	// Calculate spacing
	spacing := t.width - tabsWidth - len(folder)
	if spacing < 2 {
		spacing = 2
	}

	result := tabsStr + safeRepeat(" ", spacing) + folderStyle.Render(folder)

	// Add border at bottom
	border := t.theme.Dimmed.Render(safeRepeat("─", t.width))

	return result + "\n" + border
}

// SetActiveTab sets the active tab
func (t *TabBar) SetActiveTab(tab Tab) {
	t.activeTab = tab
}

// ActiveTab returns the currently active tab
func (t *TabBar) ActiveTab() Tab {
	return t.activeTab
}

// SetFolderPath sets the folder path to display
func (t *TabBar) SetFolderPath(path string) {
	t.folderPath = path
}

// SetWidth sets the width of the tab bar
func (t *TabBar) SetWidth(width int) {
	t.width = width
}
