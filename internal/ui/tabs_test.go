package ui

import (
	"testing"
)

func TestTabBarHitTest(t *testing.T) {
	theme := DefaultTheme()
	tb := NewTabBar(theme)
	tb.SetWidth(120)

	// Single tab: [  1:Planning  ]
	// Bracket + 2 pad + "1:Planning" (10 chars) + 2 pad + bracket = 16
	tests := []struct {
		name     string
		tabs     []TabInfo
		x        int
		expected int
	}{
		{"click on first tab", []TabInfo{{Label: "Planning"}}, 5, 0},
		{"click before tabs", []TabInfo{{Label: "Planning"}}, -1, -1},
		{"click past all tabs", []TabInfo{{Label: "Planning"}}, 200, -1},
		{"click on first tab start bracket", []TabInfo{{Label: "Planning"}}, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb.SetTabs(tt.tabs)
			got := tb.HitTest(tt.x)
			if got != tt.expected {
				t.Errorf("HitTest(%d) = %d, want %d", tt.x, got, tt.expected)
			}
		})
	}
}

func TestTabBarHitTestMultipleTabs(t *testing.T) {
	theme := DefaultTheme()
	tb := NewTabBar(theme)
	tb.SetWidth(120)

	tb.SetTabs([]TabInfo{
		{Label: "Planning"},
		{Label: "Claude #1"},
	})

	// Verify first tab is at position 0
	if idx := tb.HitTest(0); idx != 0 {
		t.Errorf("HitTest(0) = %d, want 0", idx)
	}

	// Verify second tab is hittable at a position after the first tab + gap
	// First tab: "[" + 2pad + "1:Planning"(10) + 2pad + "]" = 16
	// Gap: 2
	// Second tab starts at 18: "[" + 2pad + "2:Claude #1"(11) + 2pad + "]" = 17
	if idx := tb.HitTest(18); idx != 1 {
		t.Errorf("HitTest(18) = %d, want 1", idx)
	}

	// Verify gap between tabs returns -1
	// First tab ends at 16, gap is 16-17
	if idx := tb.HitTest(16); idx != -1 {
		t.Errorf("HitTest(16) in gap = %d, want -1", idx)
	}
}

func TestTabBarHitTestReturnsNegativeOneForEmptyArea(t *testing.T) {
	theme := DefaultTheme()
	tb := NewTabBar(theme)
	tb.SetWidth(120)
	tb.SetTabs([]TabInfo{{Label: "Planning"}})

	// Far past the tab area
	if idx := tb.HitTest(100); idx != -1 {
		t.Errorf("HitTest(100) = %d, want -1", idx)
	}
}
