package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Mode represents the current app mode
type Mode int

const (
	ModeNormal Mode = iota
	ModeSession
	ModeDialog
)

// StatusBar displays keybindings and breadcrumb
type StatusBar struct {
	theme      *Theme
	mode       Mode
	breadcrumb []string
	message    string
	width      int
}

// NewStatusBar creates a new status bar
func NewStatusBar(theme *Theme) *StatusBar {
	return &StatusBar{
		theme: theme,
	}
}

// View renders the status bar
func (s *StatusBar) View() string {
	leftContent := s.renderKeybindings()
	rightContent := s.renderBreadcrumb()

	// If there's a message, show it
	if s.message != "" {
		leftContent = s.theme.Selected.Render(s.message)
	}

	// Calculate spacing
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	spacer := s.width - leftWidth - rightWidth - 4

	if spacer < 1 {
		spacer = 1
	}

	content := fmt.Sprintf(" %s%s%s ",
		leftContent,
		safeRepeat(" ", spacer),
		rightContent,
	)

	return s.theme.StatusBar.Width(s.width).Render(content)
}

func (s *StatusBar) renderKeybindings() string {
	var keys []string

	switch s.mode {
	case ModeNormal:
		keys = []string{"↑/↓", "navigate", "enter", "select", "e", "edit", "s", "agent"}
	case ModeSession:
		keys = []string{"i", "input", "Ctrl+\\", "exit", "Tab", "switch"}
	}

	var parts []string
	for i := 0; i < len(keys); i += 2 {
		if i+1 < len(keys) {
			key := s.theme.Selected.Render(fmt.Sprintf("[%s]", keys[i]))
			action := s.theme.Dimmed.Render(keys[i+1])
			parts = append(parts, fmt.Sprintf("%s%s", key, action))
		}
	}

	return strings.Join(parts, " ")
}

func (s *StatusBar) renderBreadcrumb() string {
	if len(s.breadcrumb) == 0 {
		return ""
	}

	parts := make([]string, len(s.breadcrumb))
	for i, part := range s.breadcrumb {
		if i == len(s.breadcrumb)-1 {
			parts[i] = s.theme.Normal.Render(part)
		} else {
			parts[i] = s.theme.Dimmed.Render(part)
		}
	}

	return strings.Join(parts, s.theme.Dimmed.Render(" > "))
}

// SetMode sets the current mode
func (s *StatusBar) SetMode(mode Mode) {
	s.mode = mode
}

// SetBreadcrumb sets the breadcrumb path
func (s *StatusBar) SetBreadcrumb(parts ...string) {
	s.breadcrumb = parts
}

// SetMessage sets a temporary message
func (s *StatusBar) SetMessage(msg string) {
	s.message = msg
}

// ClearMessage clears the message
func (s *StatusBar) ClearMessage() {
	s.message = ""
}

// SetWidth sets the status bar width
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}
