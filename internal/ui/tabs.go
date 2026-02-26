package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerTickMsg advances the spinner animation
type SpinnerTickMsg struct{}

// TabInfo describes a single tab in the tab bar
type TabInfo struct {
	Label  string // Display label (e.g., "Planning", "Claude #1")
	Status string // "running", "idle", "needs_input", "completed", "" (agent tabs only)
}

// TabBar displays the tab bar at the top of the screen
type TabBar struct {
	theme           *Theme
	tabs            []TabInfo // tabs[0] is always Planning
	activeIdx       int
	folderPath      string
	width           int
	spinnerFrame    int
	spinnerFrames   []string
	spinnerInterval time.Duration
}

// NewTabBar creates a new tab bar with a Planning tab
func NewTabBar(theme *Theme) *TabBar {
	preset := SpinnerPresetByName(DefaultSpinnerPreset())
	return &TabBar{
		theme: theme,
		tabs: []TabInfo{
			{Label: "Planning"},
		},
		activeIdx:       0,
		spinnerFrames:   preset.Frames,
		spinnerInterval: preset.Interval,
	}
}

// SetSpinner updates the spinner animation to use the given preset.
func (t *TabBar) SetSpinner(preset SpinnerPreset) {
	t.spinnerFrames = preset.Frames
	t.spinnerInterval = preset.Interval
	t.spinnerFrame = 0
}

// Init initializes the tab bar
func (t *TabBar) Init() tea.Cmd {
	return nil
}

// Update handles messages (tab bar is now driven externally by App)
func (t *TabBar) Update(msg tea.Msg) (*TabBar, tea.Cmd) {
	if _, ok := msg.(SpinnerTickMsg); ok {
		t.spinnerFrame = (t.spinnerFrame + 1) % len(t.spinnerFrames)
		if t.HasRunningTabs() {
			cmd := t.Tick()
			return t, cmd
		}
	}
	return t, nil
}

// Tick returns a command that fires a SpinnerTickMsg after the configured interval.
func (t *TabBar) Tick() tea.Cmd {
	interval := t.spinnerInterval
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
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
	// Build tabs
	var tabs strings.Builder

	for i, tab := range t.tabs {
		if i > 0 {
			tabs.WriteString("  ")
		}

		label := tab.Label
		// Add status indicator for agent tabs (always left-aligned before label)
		switch {
		case tab.Status == "needs_input" && i > 0:
			label = IndicatorActive + " " + label
		case (tab.Status == "completed" || tab.Status == "idle") && i > 0:
			label = IndicatorDone + " " + label
		case tab.Status == "running" && i > 0:
			label = t.spinnerFrames[t.spinnerFrame%len(t.spinnerFrames)] + " " + label
		}

		// Add number prefix
		numLabel := fmt.Sprintf("%d:%s", i+1, label)

		// Determine foreground color from tab state
		var fg lipgloss.Color
		switch {
		case tab.Status == "needs_input" && i > 0:
			fg = t.theme.Error
		case (tab.Status == "completed" || tab.Status == "idle") && i > 0:
			fg = t.theme.Success
		case tab.Status == "running" && i > 0:
			fg = t.theme.Accent
		default:
			fg = t.theme.Secondary
		}

		// Active tab: inherit state color, add bold + dark background
		style := lipgloss.NewStyle().Foreground(fg).Padding(0, 2)
		if i == t.activeIdx {
			style = style.Bold(true).Background(lipgloss.Color("#1a1a1a"))
		}

		tabs.WriteString("[")
		tabs.WriteString(style.Render(numLabel))
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

// HitTest returns the tab index at the given X coordinate, or -1 if none.
// This mirrors the layout logic in View() to compute tab positions.
func (t *TabBar) HitTest(x int) int {
	pos := 0
	for i, tab := range t.tabs {
		if i > 0 {
			pos += 2 // gap between tabs
		}

		label := tab.Label
		switch {
		case tab.Status == "needs_input" && i > 0:
			label = IndicatorActive + " " + label
		case (tab.Status == "completed" || tab.Status == "idle") && i > 0:
			label = IndicatorDone + " " + label
		case tab.Status == "running" && i > 0:
			label = t.spinnerFrames[t.spinnerFrame%len(t.spinnerFrames)] + " " + label
		}

		numLabel := fmt.Sprintf("%d:%s", i+1, label)
		// Tab visual width: "[" + 2 padding + content + 2 padding + "]"
		contentWidth := lipgloss.Width(numLabel)
		tabWidth := 1 + 2 + contentWidth + 2 + 1 // brackets + padding + content

		if x >= pos && x < pos+tabWidth {
			return i
		}
		pos += tabWidth
	}
	return -1
}
