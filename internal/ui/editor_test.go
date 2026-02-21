package ui

import (
	"testing"
)

func TestEditor_RenderViewMode_SmallWidth(t *testing.T) {
	theme := DefaultTheme()
	editor := NewEditor(theme)

	// Set content
	editor.SetContent("test.md", "This is a test line that is fairly long")

	// Test with various small widths - should not panic
	testWidths := []int{0, 1, 2, 3, 4, 5, 6, 7, 10, 20}

	for _, width := range testWidths {
		t.Run("width_"+string(rune('0'+width)), func(t *testing.T) {
			editor.SetSize(width, 10)
			// Should not panic
			result := editor.View()
			_ = result
		})
	}
}

func TestEditor_RenderEditMode_SmallWidth(t *testing.T) {
	theme := DefaultTheme()
	editor := NewEditor(theme)

	// Set content and enter edit mode
	editor.SetContent("test.md", "This is a test line that is fairly long")
	editor.mode = EditorModeEdit
	editor.parseLines()

	// Test with various small widths - should not panic
	testWidths := []int{0, 1, 2, 3, 4, 5, 6, 7, 10, 20}

	for _, width := range testWidths {
		t.Run("width_"+string(rune('0'+width)), func(t *testing.T) {
			editor.SetSize(width, 10)
			// Should not panic
			result := editor.View()
			_ = result
		})
	}
}

func TestEditor_RenderLineWithCursor_SmallWidth(t *testing.T) {
	theme := DefaultTheme()
	editor := NewEditor(theme)

	// Set up editor state
	editor.SetContent("test.md", "This is a test line")
	editor.mode = EditorModeEdit
	editor.parseLines()

	// Test various cursor positions and small widths
	testCases := []struct {
		width     int
		cursorCol int
	}{
		{0, 0},
		{1, 0},
		{5, 3},
		{10, 5},
		{10, 15}, // cursor past visible area
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			editor.cursorCol = tc.cursorCol
			// renderLineWithCursor is private, test via View()
			editor.SetSize(tc.width, 10)
			// Should not panic
			result := editor.View()
			_ = result
		})
	}
}

func TestEditor_ZeroWidthNoPanic(t *testing.T) {
	theme := DefaultTheme()
	editor := NewEditor(theme)

	// Simulate the state before any WindowSizeMsg is received
	// width and height are 0 by default

	editor.SetContent("test.md", "# Test\n\nSome content here")

	// View mode with zero size - should not panic
	result := editor.View()
	_ = result

	// Edit mode with zero size - should not panic
	editor.mode = EditorModeEdit
	editor.parseLines()
	result = editor.View()
	_ = result
}

func TestEditor_LongLinesRendering(t *testing.T) {
	theme := DefaultTheme()
	editor := NewEditor(theme)

	// Create content with very long lines
	longLine := "This is a very long line that should be truncated when the width is small. " +
		"It contains a lot of text to ensure truncation happens properly."
	editor.SetContent("test.md", longLine)
	editor.SetSize(40, 10)

	// View mode - should not panic
	result := editor.View()
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Edit mode - should not panic
	editor.mode = EditorModeEdit
	editor.parseLines()
	result = editor.View()
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestEditor_SetSize(t *testing.T) {
	theme := DefaultTheme()
	editor := NewEditor(theme)

	tests := []struct {
		width  int
		height int
	}{
		{0, 0},
		{-1, -1},
		{100, 50},
		{1, 1},
	}

	for _, tc := range tests {
		editor.SetSize(tc.width, tc.height)
		// Should not panic when rendering
		_ = editor.View()
	}
}
