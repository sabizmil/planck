package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Braille spinner frames for running tabs
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// SpinnerTickMsg advances the spinner animation
type SpinnerTickMsg struct{}

// TabInfo describes a single tab in the tab bar
type TabInfo struct {
	Label  string // Display label (e.g., "Planning", "Claude #1")
	Status string // "running", "completed", "" (only for agent tabs)
}

// TabBar displays the tab bar at the top of the screen
type TabBar struct {
	theme        *Theme
	tabs         []TabInfo // tabs[0] is always Planning
	activeIdx    int
	folderPath   string
	width        int
	spinnerFrame int
}

// NewTabBar creates a new tab bar with a Planning tab
func NewTabBar(theme *Theme) *TabBar {
	return &TabBar{
		theme: theme,
		tabs: []TabInfo{
			{Label: "Planning"},
		},
		activeIdx: 0,
	}
}

// Init initializes the tab bar
func (t *TabBar) Init() tea.Cmd {
	return nil
}

// Update handles messages (tab bar is now driven externally by App)
func (t *TabBar) Update(msg tea.Msg) (*TabBar, tea.Cmd) {
	if _, ok := msg.(SpinnerTickMsg); ok {
		t.spinnerFrame = (t.spinnerFrame + 1) % len(spinnerFrames)
		if t.HasRunningTabs() {
			return t, t.Tick()
		}
	}
	return t, nil
}

// Tick returns a command that fires a SpinnerTickMsg after 80ms
func (t *TabBar) Tick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return SpinnerTickMsg{}
	})
}

// HasRunningTabs returns true if any tab has status "running"
func (t *TabBar) HasRunningTabs() bool {
	for _, tab := range t.tabs {
		if tab.Status == "running" {
			return true
		}
	}
	return false
}

// View renders the tab bar
func (t *TabBar) View() string {
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.theme.Accent).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 2)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(t.theme.Secondary).
		Padding(0, 2)

	completedStyle := lipgloss.NewStyle().
		Foreground(t.theme.Success).
		Padding(0, 2)

	runningStyle := lipgloss.NewStyle().
		Foreground(t.theme.Accent).
		Padding(0, 2)

	// Build tabs
	var tabs strings.Builder

	for i, tab := range t.tabs {
		if i > 0 {
			tabs.WriteString("  ")
		}

		label := tab.Label
		// Add status indicator for agent tabs
		if tab.Status == "completed" && i > 0 {
			label += " " + IndicatorDone
		} else if tab.Status == "running" && i > 0 {
			label = spinnerFrames[t.spinnerFrame] + " " + label
		}

		// Add number prefix
		numLabel := fmt.Sprintf("%d:%s", i+1, label)

		tabs.WriteString("[")
		if i == t.activeIdx {
			tabs.WriteString(activeStyle.Render(numLabel))
		} else if tab.Status == "completed" && i > 0 {
			tabs.WriteString(completedStyle.Render(numLabel))
		} else if tab.Status == "running" && i > 0 {
			tabs.WriteString(runningStyle.Render(numLabel))
		} else {
			tabs.WriteString(inactiveStyle.Render(numLabel))
		}
		tabs.WriteString("]")
	}

	// Add folder path on the right
	tabsStr := tabs.String()
	tabsWidth := lipgloss.Width(tabsStr)

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

// SetActiveIdx sets the active tab by index
func (t *TabBar) SetActiveIdx(idx int) {
	if idx >= 0 && idx < len(t.tabs) {
		t.activeIdx = idx
	}
}

// ActiveIdx returns the currently active tab index
func (t *TabBar) ActiveIdx() int {
	return t.activeIdx
}

// TabCount returns the number of tabs
func (t *TabBar) TabCount() int {
	return len(t.tabs)
}

// SetTabs replaces the tab list (Planning tab is always preserved as tabs[0])
func (t *TabBar) SetTabs(tabs []TabInfo) {
	if len(tabs) == 0 {
		t.tabs = []TabInfo{{Label: "Planning"}}
	} else {
		t.tabs = tabs
	}
	// Clamp active index
	if t.activeIdx >= len(t.tabs) {
		t.activeIdx = len(t.tabs) - 1
	}
}

// UpdateTabStatus updates the status of a specific tab
func (t *TabBar) UpdateTabStatus(idx int, status string) {
	if idx >= 0 && idx < len(t.tabs) {
		t.tabs[idx].Status = status
	}
}

// SetFolderPath sets the folder path to display
func (t *TabBar) SetFolderPath(path string) {
	t.folderPath = path
}

// SetWidth sets the width of the tab bar
func (t *TabBar) SetWidth(width int) {
	t.width = width
}
