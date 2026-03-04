package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// ThemePresetInfo describes an available theme preset.
type ThemePresetInfo struct {
	Name        string
	Description string
}

// AllThemePresets returns the list of available TUI theme presets.
func AllThemePresets() []ThemePresetInfo {
	return []ThemePresetInfo{
		{"default", "Cyan/teal dark theme"},
		{"monokai", "Warm orange and purple"},
		{"solarized-dark", "Classic solarized palette"},
		{"nord", "Blue frost tones"},
		{"dracula", "Purple, pink, and green"},
	}
}

// ThemeFromPreset returns a Theme for the given preset name.
// Falls back to DefaultTheme if the preset is unknown or NO_COLOR is set.
func ThemeFromPreset(name string) *Theme {
	if os.Getenv("NO_COLOR") != "" {
		return noColorTheme()
	}

	switch name {
	case "", "default":
		return DefaultTheme()
	case "monokai":
		return monokaiTheme()
	case "solarized-dark":
		return solarizedDarkTheme()
	case "nord":
		return nordTheme()
	case "dracula":
		return draculaTheme()
	default:
		return DefaultTheme()
	}
}

// buildTheme constructs a full Theme from a color palette.
func buildTheme(primary, secondary, accent, success, warning, errorC, dimmed, selectedBg lipgloss.Color) *Theme {
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

func monokaiTheme() *Theme {
	return buildTheme(
		lipgloss.Color("#F8F8F2"), // primary (light)
		lipgloss.Color("#75715E"), // secondary (comment brown)
		lipgloss.Color("#FD971F"), // accent (orange)
		lipgloss.Color("#A6E22E"), // success (green)
		lipgloss.Color("#E6DB74"), // warning (yellow)
		lipgloss.Color("#F92672"), // error (pink/red)
		lipgloss.Color("#75715E"), // dimmed (comment)
		lipgloss.Color("#3E3D32"), // selectedBg
	)
}

func solarizedDarkTheme() *Theme {
	return buildTheme(
		lipgloss.Color("#839496"), // primary (base0)
		lipgloss.Color("#657B83"), // secondary (base00)
		lipgloss.Color("#268BD2"), // accent (blue)
		lipgloss.Color("#859900"), // success (green)
		lipgloss.Color("#B58900"), // warning (yellow)
		lipgloss.Color("#DC322F"), // error (red)
		lipgloss.Color("#586E75"), // dimmed (base01)
		lipgloss.Color("#073642"), // selectedBg (base02)
	)
}

func nordTheme() *Theme {
	return buildTheme(
		lipgloss.Color("#ECEFF4"), // primary (snow storm 3)
		lipgloss.Color("#D8DEE9"), // secondary (snow storm 1)
		lipgloss.Color("#88C0D0"), // accent (frost 3)
		lipgloss.Color("#A3BE8C"), // success (aurora green)
		lipgloss.Color("#EBCB8B"), // warning (aurora yellow)
		lipgloss.Color("#BF616A"), // error (aurora red)
		lipgloss.Color("#4C566A"), // dimmed (polar night 4)
		lipgloss.Color("#3B4252"), // selectedBg (polar night 2)
	)
}

func draculaTheme() *Theme {
	return buildTheme(
		lipgloss.Color("#F8F8F2"), // primary (foreground)
		lipgloss.Color("#6272A4"), // secondary (comment)
		lipgloss.Color("#BD93F9"), // accent (purple)
		lipgloss.Color("#50FA7B"), // success (green)
		lipgloss.Color("#F1FA8C"), // warning (yellow)
		lipgloss.Color("#FF5555"), // error (red)
		lipgloss.Color("#6272A4"), // dimmed (comment)
		lipgloss.Color("#44475A"), // selectedBg (current line)
	)
}
