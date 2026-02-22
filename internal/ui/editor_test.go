package ui

import (
	"strings"
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
	longLine := "This is a very long line that should be wrapped when the width is small. " +
		"It contains a lot of text to ensure wrapping happens properly."
	editor.SetContent("test.md", longLine)
	editor.SetSize(40, 10)

	// View mode - should not panic
	result := editor.View()
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Edit mode - should not panic and should NOT contain "..."
	editor.mode = EditorModeEdit
	editor.parseLines()
	result = editor.View()
	if result == "" {
		t.Error("Expected non-empty result")
	}
	if strings.Contains(result, "...") {
		t.Error("Edit mode should wrap long lines, not truncate with '...'")
	}
}

func TestWrapLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		maxWidth int
		expected []string
	}{
		{
			name:     "short line no wrap",
			line:     "hello world",
			maxWidth: 20,
			expected: []string{"hello world"},
		},
		{
			name:     "exact width no wrap",
			line:     "hello",
			maxWidth: 5,
			expected: []string{"hello"},
		},
		{
			name:     "wrap at word boundary",
			line:     "hello world foo",
			maxWidth: 11,
			expected: []string{"hello world", "foo"},
		},
		{
			name:     "long word hard wrap",
			line:     "abcdefghijklmnopqrstuvwxyz",
			maxWidth: 10,
			expected: []string{"abcdefghij", "klmnopqrst", "uvwxyz"},
		},
		{
			name:     "empty line",
			line:     "",
			maxWidth: 10,
			expected: []string{""},
		},
		{
			name:     "multiple wraps",
			line:     "one two three four five six",
			maxWidth: 10,
			expected: []string{"one two", "three four", "five six"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := wrapLine(tc.line, tc.maxWidth)
			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d segments, got %d: %v", len(tc.expected), len(result), result)
			}
			for i, seg := range result {
				if seg != tc.expected[i] {
					t.Errorf("segment %d: expected %q, got %q", i, tc.expected[i], seg)
				}
			}
		})
	}
}

func TestEditor_WrappedCursorNavigation(t *testing.T) {
	theme := DefaultTheme()
	editor := NewEditor(theme)

	// A line that will wrap at width 30 (maxLineWidth ~20 after gutter)
	editor.SetContent("test.md", "one two three four five six seven eight nine ten")
	editor.mode = EditorModeEdit
	editor.parseLines()
	editor.SetSize(30, 20)

	// Cursor starts at 0,0
	if editor.cursorRow != 0 || editor.cursorCol != 0 {
		t.Fatalf("expected cursor at 0,0, got %d,%d", editor.cursorRow, editor.cursorCol)
	}

	// Move down should move within the same logical line's wrapped segments
	editor.moveCursorDownVisual()
	if editor.cursorRow != 0 {
		t.Errorf("after down from wrapped line, expected logical row 0, got %d", editor.cursorRow)
	}
	if editor.cursorCol == 0 {
		t.Errorf("after down, cursorCol should have advanced past first visual line")
	}

	// Move back up should return to first segment
	editor.moveCursorUpVisual()
	if editor.cursorCol != 0 {
		t.Errorf("after up, expected cursorCol 0, got %d", editor.cursorCol)
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
