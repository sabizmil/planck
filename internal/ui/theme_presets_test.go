package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestAllThemePresets_NonEmpty(t *testing.T) {
	presets := AllThemePresets()
	if len(presets) == 0 {
		t.Fatal("AllThemePresets should return at least one preset")
	}
	for _, p := range presets {
		if p.Name == "" {
			t.Error("Preset name should not be empty")
		}
		if p.Description == "" {
			t.Error("Preset description should not be empty")
		}
	}
}

func TestThemeFromPreset_AllPresets(t *testing.T) {
	presets := AllThemePresets()
	for _, p := range presets {
		t.Run(p.Name, func(t *testing.T) {
			theme := ThemeFromPreset(p.Name)
			if theme == nil {
				t.Fatal("ThemeFromPreset should not return nil")
			}
			// Check that key fields are populated
			if theme.Primary == lipgloss.Color("") {
				t.Error("Theme Primary should be set")
			}
			if theme.Accent == lipgloss.Color("") {
				t.Error("Theme Accent should be set")
			}
		})
	}
}

func TestThemeFromPreset_DefaultFallback(t *testing.T) {
	// Unknown preset should fall back to default
	theme := ThemeFromPreset("nonexistent-theme")
	if theme == nil {
		t.Fatal("ThemeFromPreset should not return nil for unknown preset")
	}
	defaultTheme := DefaultTheme()
	if theme.Accent != defaultTheme.Accent {
		t.Error("Unknown preset should fall back to default theme")
	}
}

func TestThemeFromPreset_EmptyString(t *testing.T) {
	// Empty string should return default
	theme := ThemeFromPreset("")
	if theme == nil {
		t.Fatal("ThemeFromPreset should not return nil for empty string")
	}
	defaultTheme := DefaultTheme()
	if theme.Accent != defaultTheme.Accent {
		t.Error("Empty preset name should fall back to default theme")
	}
}

func TestBuildTheme_PopulatesAllFields(t *testing.T) {
	theme := buildTheme(
		lipgloss.Color("#FFFFFF"),
		lipgloss.Color("#AAAAAA"),
		lipgloss.Color("#00FF00"),
		lipgloss.Color("#00FF00"),
		lipgloss.Color("#FFFF00"),
		lipgloss.Color("#FF0000"),
		lipgloss.Color("#666666"),
		lipgloss.Color("#333333"),
	)

	if theme.Primary != lipgloss.Color("#FFFFFF") {
		t.Error("Primary color not set correctly")
	}
	if theme.Accent != lipgloss.Color("#00FF00") {
		t.Error("Accent color not set correctly")
	}
}
