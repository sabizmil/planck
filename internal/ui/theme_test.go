package ui

import (
	"os"
	"testing"
)

func TestDefaultTheme(t *testing.T) {
	// Save and restore NO_COLOR
	oldNoColor := os.Getenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", oldNoColor)

	// Test with colors
	os.Setenv("NO_COLOR", "")
	theme := DefaultTheme()

	if theme == nil {
		t.Fatal("DefaultTheme() returned nil")
	}

	// Verify styles are set - at least some styles should be defined
	_ = theme.Title.Value()
	_ = theme.Normal.Value()
}

func TestNoColorMode(t *testing.T) {
	oldNoColor := os.Getenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", oldNoColor)

	os.Setenv("NO_COLOR", "1")
	theme := DefaultTheme()

	if theme == nil {
		t.Fatal("DefaultTheme() returned nil in NO_COLOR mode")
	}
}

func TestStatusIndicatorConstants(t *testing.T) {
	// Verify expected indicators exist and are non-empty
	indicators := map[string]string{
		"done":       IndicatorDone,
		"inProgress": IndicatorInProgress,
		"pending":    IndicatorPending,
		"blocked":    IndicatorBlocked,
		"failed":     IndicatorFailed,
		"skipped":    IndicatorSkipped,
		"active":     IndicatorActive,
		"background": IndicatorBackground,
		"executing":  IndicatorExecuting,
		"paused":     IndicatorPaused,
		"selected":   IndicatorSelected,
	}

	for name, value := range indicators {
		if value == "" {
			t.Errorf("Indicator %s is empty", name)
		}
	}
}

func TestStatusStyle(t *testing.T) {
	theme := DefaultTheme()

	tests := []struct {
		status string
	}{
		{"done"},
		{"completed"},
		{"in-progress"},
		{"running"},
		{"pending"},
		{"ready"},
		{"blocked"},
		{"failed"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			style := theme.StatusStyle(tt.status)
			// Just verify it doesn't panic and returns a style
			_ = style.Render("test")
		})
	}
}

func TestStatusIndicator(t *testing.T) {
	theme := DefaultTheme()

	tests := []struct {
		status   string
		expected string
	}{
		{"done", IndicatorDone},
		{"completed", IndicatorDone},
		{"in-progress", IndicatorInProgress},
		{"running", IndicatorInProgress},
		{"pending", IndicatorPending},
		{"ready", IndicatorPending},
		{"blocked", IndicatorBlocked},
		{"failed", IndicatorFailed},
		{"skipped", IndicatorSkipped},
		{"selected", IndicatorDone},
		{"rejected", IndicatorFailed},
		{"proposed", IndicatorPending},
		{"unknown", IndicatorPending},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			indicator := theme.StatusIndicator(tt.status)
			if indicator != tt.expected {
				t.Errorf("StatusIndicator(%s) = %s, want %s", tt.status, indicator, tt.expected)
			}
		})
	}
}

func TestThemeStyles(t *testing.T) {
	theme := DefaultTheme()

	// Test that all styles can render without panic
	testStyles := []struct {
		name  string
		style func() string
	}{
		{"Title", func() string { return theme.Title.Render("test") }},
		{"Subtitle", func() string { return theme.Subtitle.Render("test") }},
		{"Normal", func() string { return theme.Normal.Render("test") }},
		{"Selected", func() string { return theme.Selected.Render("test") }},
		{"StatusDone", func() string { return theme.StatusDone.Render("test") }},
		{"StatusProgress", func() string { return theme.StatusProgress.Render("test") }},
		{"StatusPending", func() string { return theme.StatusPending.Render("test") }},
		{"StatusBlocked", func() string { return theme.StatusBlocked.Render("test") }},
		{"StatusFailed", func() string { return theme.StatusFailed.Render("test") }},
		{"Breadcrumb", func() string { return theme.Breadcrumb.Render("test") }},
		{"KeyHint", func() string { return theme.KeyHint.Render("test") }},
		{"Sidebar", func() string { return theme.Sidebar.Render("test") }},
		{"SidebarItem", func() string { return theme.SidebarItem.Render("test") }},
		{"SidebarSelected", func() string { return theme.SidebarSelected.Render("test") }},
		{"PlanTree", func() string { return theme.PlanTree.Render("test") }},
		{"TreeItem", func() string { return theme.TreeItem.Render("test") }},
		{"TreeSelected", func() string { return theme.TreeSelected.Render("test") }},
		{"DetailPanel", func() string { return theme.DetailPanel.Render("test") }},
		{"StatusBar", func() string { return theme.StatusBar.Render("test") }},
		{"Dialog", func() string { return theme.Dialog.Render("test") }},
		{"DialogTitle", func() string { return theme.DialogTitle.Render("test") }},
	}

	for _, tt := range testStyles {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.style()
			// Just verify it doesn't panic
			_ = result
		})
	}
}

func TestSafeRepeat(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		count    int
		expected string
	}{
		{"positive count", "a", 3, "aaa"},
		{"zero count", "a", 0, ""},
		{"negative count", "a", -1, ""},
		{"negative count large", "─", -100, ""},
		{"width minus padding negative", "─", 0 - 4, ""}, // simulates width=0, padding=4
		{"empty string positive", "", 5, ""},
		{"unicode char", "─", 3, "───"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeRepeat(tt.s, tt.count)
			if result != tt.expected {
				t.Errorf("safeRepeat(%q, %d) = %q, want %q", tt.s, tt.count, result, tt.expected)
			}
		})
	}
}

func TestSafeRepeat_NoPanic(t *testing.T) {
	// This test ensures safeRepeat never panics with any input
	testCases := []int{-1000, -100, -10, -1, 0, 1, 10, 100}

	for _, count := range testCases {
		t.Run("count_"+string(rune('0'+count)), func(t *testing.T) {
			// Should not panic
			_ = safeRepeat("─", count)
			_ = safeRepeat(" ", count)
			_ = safeRepeat("abc", count)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{"no truncation needed", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate with ellipsis", "hello world", 8, "hello..."},
		{"very short maxLen", "hello", 2, "he"},
		{"maxLen of 3", "hello", 3, "hel"},
		{"maxLen of 4", "hello", 4, "h..."},
		{"empty string", "", 5, ""},
		{"zero maxLen", "hello", 0, ""},
		{"negative maxLen", "hello", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.s, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, result, tt.expected)
			}
		})
	}
}
