package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines the visual styling
type Theme struct {
	// Base colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color

	// Semantic styles
	Title          lipgloss.Style
	Subtitle       lipgloss.Style
	Normal         lipgloss.Style
	Dimmed         lipgloss.Style
	Selected       lipgloss.Style
	StatusDone     lipgloss.Style
	StatusProgress lipgloss.Style
	StatusPending  lipgloss.Style
	StatusBlocked  lipgloss.Style
	StatusFailed   lipgloss.Style
	Breadcrumb     lipgloss.Style
	KeyHint        lipgloss.Style
	Border         lipgloss.Style

	// Component styles
	Sidebar         lipgloss.Style
	SidebarItem     lipgloss.Style
	SidebarSelected lipgloss.Style
	PlanTree        lipgloss.Style
	TreeItem        lipgloss.Style
	TreeSelected    lipgloss.Style
	DetailPanel     lipgloss.Style
	StatusBar       lipgloss.Style
	Dialog          lipgloss.Style
	DialogTitle     lipgloss.Style

	// Version display
	VersionDev lipgloss.Style

	// Stream event styles
	ThinkingBlock  lipgloss.Style
	ToolUseCard    lipgloss.Style
	ToolUseName    lipgloss.Style
	ToolUseInput   lipgloss.Style
	ToolUseResult  lipgloss.Style
	PermissionCard lipgloss.Style
	SystemMessage  lipgloss.Style
}

// DefaultTheme returns the default dark theme
func DefaultTheme() *Theme {
	// Check for NO_COLOR environment variable
	noColor := os.Getenv("NO_COLOR") != ""

	if noColor {
		return noColorTheme()
	}

	// Colors from spec
	primary := lipgloss.Color("#E0E0E0")
	secondary := lipgloss.Color("#A0A0A0")
	accent := lipgloss.Color("#06B6D4")     // Teal/cyan
	success := lipgloss.Color("#22C55E")    // Green
	warning := lipgloss.Color("#F59E0B")    // Amber
	errorC := lipgloss.Color("#EF4444")     // Red
	dimmed := lipgloss.Color("#6B7280")     // Gray
	selectedBg := lipgloss.Color("#1E3A5F") // Subtle dark blue for selection highlight

	return &Theme{
		Primary:   primary,
		Secondary: secondary,
		Accent:    accent,
		Success:   success,
		Warning:   warning,
		Error:     errorC,

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary),

		Subtitle: lipgloss.NewStyle().
			Foreground(secondary),

		Dimmed: lipgloss.NewStyle().
			Foreground(dimmed),

		Normal: lipgloss.NewStyle().
			Foreground(primary),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Background(selectedBg),

		StatusDone: lipgloss.NewStyle().
			Foreground(success),

		StatusProgress: lipgloss.NewStyle().
			Foreground(warning),

		StatusPending: lipgloss.NewStyle().
			Foreground(dimmed),

		StatusBlocked: lipgloss.NewStyle().
			Foreground(warning),

		StatusFailed: lipgloss.NewStyle().
			Foreground(errorC),

		Breadcrumb: lipgloss.NewStyle().
			Foreground(dimmed),

		KeyHint: lipgloss.NewStyle().
			Foreground(dimmed),

		Border: lipgloss.NewStyle().
			BorderForeground(dimmed),

		Sidebar: lipgloss.NewStyle().
			Width(16).
			BorderStyle(lipgloss.NormalBorder()).
			BorderRight(true).
			BorderForeground(dimmed),

		SidebarItem: lipgloss.NewStyle().
			Foreground(primary).
			PaddingLeft(1),

		SidebarSelected: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			Background(selectedBg).
			PaddingLeft(1),

		PlanTree: lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1),

		TreeItem: lipgloss.NewStyle().
			Foreground(primary),

		TreeSelected: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			Background(selectedBg),

		DetailPanel: lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2),

		StatusBar: lipgloss.NewStyle().
			Foreground(dimmed).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(dimmed),

		VersionDev: lipgloss.NewStyle().
			Foreground(warning).
			Bold(true),

		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 2),

		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),

		// Stream event styles
		ThinkingBlock: lipgloss.NewStyle().
			Foreground(dimmed).
			Italic(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimmed).
			Padding(0, 1),

		ToolUseCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondary).
			Padding(0, 1),

		ToolUseName: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),

		ToolUseInput: lipgloss.NewStyle().
			Foreground(dimmed),

		ToolUseResult: lipgloss.NewStyle().
			Foreground(primary).
			PaddingLeft(2),

		PermissionCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warning).
			Padding(0, 1),

		SystemMessage: lipgloss.NewStyle().
			Foreground(dimmed).
			Italic(true).
			Align(lipgloss.Center),
	}
}

func noColorTheme() *Theme {
	return &Theme{
		Title:           lipgloss.NewStyle().Bold(true),
		Subtitle:        lipgloss.NewStyle(),
		Normal:          lipgloss.NewStyle(),
		Dimmed:          lipgloss.NewStyle(),
		Selected:        lipgloss.NewStyle().Bold(true).Reverse(true),
		StatusDone:      lipgloss.NewStyle(),
		StatusProgress:  lipgloss.NewStyle(),
		StatusPending:   lipgloss.NewStyle(),
		StatusBlocked:   lipgloss.NewStyle(),
		StatusFailed:    lipgloss.NewStyle(),
		Breadcrumb:      lipgloss.NewStyle(),
		KeyHint:         lipgloss.NewStyle(),
		Border:          lipgloss.NewStyle(),
		Sidebar:         lipgloss.NewStyle().Width(16),
		SidebarItem:     lipgloss.NewStyle().PaddingLeft(1),
		SidebarSelected: lipgloss.NewStyle().Bold(true).Reverse(true).PaddingLeft(1),
		PlanTree:        lipgloss.NewStyle().PaddingLeft(1),
		TreeItem:        lipgloss.NewStyle(),
		TreeSelected:    lipgloss.NewStyle().Bold(true).Reverse(true),
		DetailPanel:     lipgloss.NewStyle().PaddingLeft(2),
		StatusBar:       lipgloss.NewStyle(),
		VersionDev:      lipgloss.NewStyle().Bold(true),
		Dialog:          lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2),
		DialogTitle:     lipgloss.NewStyle().Bold(true),
		ThinkingBlock:   lipgloss.NewStyle().Italic(true).Border(lipgloss.RoundedBorder()).Padding(0, 1),
		ToolUseCard:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		ToolUseName:     lipgloss.NewStyle().Bold(true),
		ToolUseInput:    lipgloss.NewStyle(),
		ToolUseResult:   lipgloss.NewStyle().PaddingLeft(2),
		PermissionCard:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		SystemMessage:   lipgloss.NewStyle().Italic(true).Align(lipgloss.Center),
	}
}

// Status indicators
const (
	IndicatorDone         = "✓"
	IndicatorInProgress   = "◉"
	IndicatorPending      = "○"
	IndicatorBlocked      = "⊘"
	IndicatorFailed       = "✗"
	IndicatorSkipped      = "—"
	IndicatorActive       = "●"
	IndicatorBackground   = "◐"
	IndicatorExecuting    = "▶"
	IndicatorPaused       = "⏸"
	IndicatorSelected     = "▸"
	IndicatorFolderOpen   = "▾"
	IndicatorFolderClosed = "▸"
)

// StatusStyle returns the appropriate style for a status
func (t *Theme) StatusStyle(status string) lipgloss.Style {
	switch status {
	case "done", "completed":
		return t.StatusDone
	case "in-progress", "running":
		return t.StatusProgress
	case "pending", "ready":
		return t.StatusPending
	case "blocked":
		return t.StatusBlocked
	case "failed":
		return t.StatusFailed
	default:
		return t.Normal
	}
}

// StatusIndicator returns the indicator character for a status
func (t *Theme) StatusIndicator(status string) string {
	switch status {
	case "done", "completed":
		return IndicatorDone
	case "in-progress", "running":
		return IndicatorInProgress
	case "pending", "ready":
		return IndicatorPending
	case "blocked":
		return IndicatorBlocked
	case "failed":
		return IndicatorFailed
	case "skipped":
		return IndicatorSkipped
	case "selected":
		return IndicatorDone
	case "rejected":
		return IndicatorFailed
	case "proposed":
		return IndicatorPending
	default:
		return IndicatorPending
	}
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// safeRepeat safely repeats a string, returning empty string if count <= 0
func safeRepeat(s string, count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(s, count)
}
