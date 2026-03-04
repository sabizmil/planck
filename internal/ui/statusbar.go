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
	keymap     *Keymap
	mode       Mode
	breadcrumb []string
	message    string
	width      int
}

// NewStatusBar creates a new status bar
func NewStatusBar(theme *Theme, keymap *Keymap) *StatusBar {
	return &StatusBar{
		theme:  theme,
		keymap: keymap,
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
	km := s.keymap

	type hint struct {
		key  string
		desc string
	}

	var hints []hint
	switch s.mode {
	case ModeNormal:
		hints = []hint{
			{km.DisplayKeysFor(ContextFileList, ActionMoveDown), "navigate"},
			{km.DisplayKeysFor(ContextFileList, ActionOpenFile), "select"},
			{km.DisplayKeysFor(ContextFileList, ActionEditMode), "edit"},
			{km.DisplayKeysFor(ContextGlobal, ActionSettings), "settings"},
		}
	case ModeSession:
		hints = []hint{
			{km.DisplayKeysFor(ContextAgentNormal, ActionEnterInput), "input"},
			{km.DisplayKeysFor(ContextAgentInput, ActionExitInput), "exit"},
			{km.DisplayKeysFor(ContextGlobal, ActionNextTab), "switch"},
		}
	}

	var parts []string
	for _, h := range hints {
		if h.key == "" {
			continue
		}
		key := s.theme.Selected.Render(fmt.Sprintf("[%s]", h.key))
		action := s.theme.Dimmed.Render(h.desc)
		parts = append(parts, fmt.Sprintf("%s%s", key, action))
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
