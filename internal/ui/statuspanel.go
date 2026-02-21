package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/planck/internal/workspace"
)

// AgentStatus represents the current state of the Claude agent
type AgentStatus string

const (
	AgentIdle    AgentStatus = "idle"
	AgentRunning AgentStatus = "running"
	AgentError   AgentStatus = "error"
)

// StatusPanel displays the always-visible status panel on the right side
type StatusPanel struct {
	theme *Theme

	// Agent state
	agentStatus    AgentStatus
	currentFile    string
	startedAt      time.Time
	elapsedTicker  <-chan time.Time
	stopTicker     chan struct{}

	// File status tracking
	files []*workspace.File

	// Dimensions
	width  int
	height int
}

// NewStatusPanel creates a new status panel
func NewStatusPanel(theme *Theme) *StatusPanel {
	return &StatusPanel{
		theme:       theme,
		agentStatus: AgentIdle,
		width:       20,
	}
}

// Init initializes the panel
func (s *StatusPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (s *StatusPanel) Update(msg tea.Msg) (*StatusPanel, tea.Cmd) {
	// Status panel is read-only, no key handling needed
	return s, nil
}

// View renders the status panel
func (s *StatusPanel) View() string {
	var b strings.Builder

	// Agent status section
	b.WriteString(s.theme.Title.Render("STATUS"))
	b.WriteString("\n")
	b.WriteString(s.theme.Dimmed.Render(safeRepeat("─", s.width-2)))
	b.WriteString("\n\n")

	// Agent status
	var statusLine string
	switch s.agentStatus {
	case AgentRunning:
		statusLine = s.theme.StatusProgress.Render("● Claude: Running")
	case AgentError:
		statusLine = s.theme.StatusFailed.Render("✗ Claude: Error")
	default:
		statusLine = s.theme.Dimmed.Render("○ Claude: Idle")
	}
	b.WriteString(statusLine)
	b.WriteString("\n")

	// Current file if running
	if s.agentStatus == AgentRunning && s.currentFile != "" {
		fileName := truncate(s.currentFile, s.width-4)
		b.WriteString(s.theme.Dimmed.Render("  " + fileName))
		b.WriteString("\n")

		// Elapsed time
		elapsed := time.Since(s.startedAt)
		b.WriteString(s.theme.Dimmed.Render(fmt.Sprintf("  %s elapsed", formatDuration(elapsed))))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Todo section
	b.WriteString(s.theme.Dimmed.Render(safeRepeat("─", s.width-2)))
	b.WriteString("\n")
	b.WriteString(s.theme.Title.Render("TODO"))
	b.WriteString("\n")
	b.WriteString(s.theme.Dimmed.Render(safeRepeat("─", s.width-2)))
	b.WriteString("\n")

	// Count files by status
	pending := 0
	inProgress := 0
	completed := 0
	for _, f := range s.files {
		switch f.Status {
		case workspace.StatusPending:
			pending++
		case workspace.StatusInProgress:
			inProgress++
		case workspace.StatusCompleted:
			completed++
		}
	}

	// Summary
	if len(s.files) > 0 {
		b.WriteString(fmt.Sprintf("(%d files)\n", len(s.files)))
	}

	// Show file list with status
	maxFiles := s.height - 14 // Leave room for header/footer
	if maxFiles < 1 {
		maxFiles = 1
	}

	shown := 0
	for _, f := range s.files {
		if shown >= maxFiles {
			remaining := len(s.files) - shown
			if remaining > 0 {
				b.WriteString(s.theme.Dimmed.Render(fmt.Sprintf("  ... +%d more", remaining)))
				b.WriteString("\n")
			}
			break
		}

		var indicator string
		switch f.Status {
		case workspace.StatusCompleted:
			indicator = s.theme.StatusDone.Render(IndicatorDone)
		case workspace.StatusInProgress:
			indicator = s.theme.StatusProgress.Render(IndicatorInProgress)
		default:
			indicator = s.theme.StatusPending.Render(IndicatorPending)
		}

		name := truncate(f.Name, s.width-4)
		b.WriteString(fmt.Sprintf("%s %s\n", indicator, name))
		shown++
	}

	// Fill remaining space
	contentLines := strings.Count(b.String(), "\n")
	for i := contentLines; i < s.height-2; i++ {
		b.WriteString("\n")
	}

	// Summary line
	b.WriteString(s.theme.Dimmed.Render(safeRepeat("─", s.width-2)))
	b.WriteString("\n")
	summary := fmt.Sprintf("%d/%d done", completed, len(s.files))
	b.WriteString(s.theme.Dimmed.Render(summary))

	// Border style
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(lipgloss.Color("#6B7280")).
		Width(s.width).
		Height(s.height)

	return style.Render(b.String())
}

// SetAgentStatus updates the agent status
func (s *StatusPanel) SetAgentStatus(status AgentStatus) {
	s.agentStatus = status
	if status == AgentRunning {
		s.startedAt = time.Now()
	}
}

// SetCurrentFile sets the file being worked on
func (s *StatusPanel) SetCurrentFile(file string) {
	s.currentFile = file
}

// SetFiles updates the file list for todo tracking
func (s *StatusPanel) SetFiles(files []*workspace.File) {
	s.files = files
}

// SetSize sets the panel dimensions
func (s *StatusPanel) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// formatDuration formats a duration as m:ss
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// TickMsg is sent to update the elapsed time
type TickMsg struct{}

// StartTicker starts the elapsed time ticker
func (s *StatusPanel) StartTicker() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}
