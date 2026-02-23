package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sabizmil/planck/internal/agent"
)

// Tool use status icons
const (
	IconToolPending  = "○"
	IconToolRunning  = "◉"
	IconToolComplete = "✓"
	IconToolFailed   = "✗"
	IconThinking     = "💭"
	IconPermission   = "🔑"
	IconError        = "⚠"
	IconSystem       = "ℹ"
)

// RenderStreamEvent renders a single event as styled text
func RenderStreamEvent(theme *Theme, event agent.StreamEvent, width int, collapsed bool) string {
	switch event.Type {
	case agent.EventText:
		return renderTextEvent(theme, event, width)
	case agent.EventThinking:
		return renderThinkingEvent(theme, event, width, collapsed)
	case agent.EventToolUse:
		return renderToolUseEvent(theme, event, width, collapsed)
	case agent.EventToolResult:
		return renderToolResultEvent(theme, event, width)
	case agent.EventPermission:
		return renderPermissionEvent(theme, event, width)
	case agent.EventError:
		return renderErrorEvent(theme, event, width)
	case agent.EventSystemMessage:
		return renderSystemMessage(theme, event, width)
	case agent.EventComplete:
		return renderCompleteEvent(theme, width)
	default:
		return ""
	}
}

// RenderEventHistory renders a slice of events with collapsible sections
func RenderEventHistory(theme *Theme, events []agent.StreamEvent, width int, collapsedSet map[int]bool) string {
	var sb strings.Builder

	for i, event := range events {
		collapsed := false
		if collapsedSet != nil {
			collapsed = collapsedSet[i]
		}
		rendered := RenderStreamEvent(theme, event, width, collapsed)
		if rendered != "" {
			sb.WriteString(rendered)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func renderTextEvent(_ *Theme, event agent.StreamEvent, width int) string {
	if event.Text == "" {
		return ""
	}
	// Wrap text at width
	return wrapText(event.Text, width)
}

func renderThinkingEvent(theme *Theme, event agent.StreamEvent, width int, collapsed bool) string {
	if collapsed {
		// Show collapsed indicator
		header := fmt.Sprintf("%s Thinking...", IconThinking)
		return theme.Dimmed.Render(header)
	}

	// Wrap the thinking content
	content := wrapText(event.Text, width-4) // Account for border padding
	return theme.ThinkingBlock.Width(width).Render(
		fmt.Sprintf("%s Thinking\n%s", IconThinking, content),
	)
}

func renderToolUseEvent(theme *Theme, event agent.StreamEvent, width int, collapsed bool) string {
	if event.ToolUse == nil {
		return ""
	}

	tu := event.ToolUse

	// Status icon
	var statusIcon string
	switch tu.Status {
	case agent.ToolUsePending:
		statusIcon = IconToolPending
	case agent.ToolUseRunning:
		statusIcon = IconToolRunning
	case agent.ToolUseComplete:
		statusIcon = IconToolComplete
	case agent.ToolUseFailed:
		statusIcon = IconToolFailed
	default:
		statusIcon = IconToolPending
	}

	// Tool name header
	header := fmt.Sprintf("%s %s", statusIcon, theme.ToolUseName.Render(tu.Name))

	if collapsed {
		return header
	}

	// Format input as truncated JSON
	var inputStr string
	if tu.Input != nil {
		inputBytes, _ := json.Marshal(tu.Input)
		inputStr = string(inputBytes)
		// Truncate if too long
		maxLen := width - 6
		if len(inputStr) > maxLen && maxLen > 3 {
			inputStr = inputStr[:maxLen-3] + "..."
		}
	}

	var content strings.Builder
	content.WriteString(header)
	if inputStr != "" {
		content.WriteString("\n")
		content.WriteString(theme.ToolUseInput.Render(inputStr))
	}

	return theme.ToolUseCard.Width(width).Render(content.String())
}

func renderToolResultEvent(theme *Theme, event agent.StreamEvent, width int) string {
	if event.Text == "" {
		return ""
	}

	// Truncate result if too long
	result := event.Text
	maxLen := 200
	if len(result) > maxLen {
		result = result[:maxLen] + "..."
	}

	return theme.ToolUseResult.Render(wrapText(result, width-4))
}

func renderPermissionEvent(theme *Theme, event agent.StreamEvent, width int) string {
	if event.Permission == nil {
		return ""
	}

	perm := event.Permission

	var content strings.Builder
	content.WriteString(fmt.Sprintf("%s Permission Required\n", IconPermission))
	content.WriteString(fmt.Sprintf("Tool: %s\n", perm.Tool))
	if perm.Description != "" {
		content.WriteString(perm.Description)
		content.WriteString("\n")
	}
	if len(perm.Options) > 0 {
		content.WriteString("\nOptions: ")
		content.WriteString(strings.Join(perm.Options, " | "))
	}

	return theme.PermissionCard.Width(width).Render(content.String())
}

func renderErrorEvent(theme *Theme, event agent.StreamEvent, _ int) string {
	errMsg := "Unknown error"
	if event.Error != nil {
		errMsg = event.Error.Error()
	}

	return lipgloss.NewStyle().
		Foreground(theme.Error).
		Render(fmt.Sprintf("%s Error: %s", IconError, errMsg))
}

func renderSystemMessage(theme *Theme, event agent.StreamEvent, width int) string {
	if event.Text == "" {
		return ""
	}
	return theme.SystemMessage.Width(width).Render(
		fmt.Sprintf("%s %s", IconSystem, event.Text),
	)
}

func renderCompleteEvent(theme *Theme, width int) string {
	return lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render("✓ Session complete")
}

// wrapText wraps text at the given width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Simple word wrapping
		words := strings.Fields(line)
		lineLen := 0

		for j, word := range words {
			wordLen := len(word)

			if lineLen+wordLen+1 > width && lineLen > 0 {
				result.WriteString("\n")
				lineLen = 0
			}

			if lineLen > 0 {
				result.WriteString(" ")
				lineLen++
			}

			result.WriteString(word)
			lineLen += wordLen

			// Handle very long words
			if j == 0 && wordLen > width {
				break
			}
		}
	}

	return result.String()
}
