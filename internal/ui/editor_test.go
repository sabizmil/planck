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

// --- Selection tests ---

func newTestEditor(content string) *Editor {
	theme := DefaultTheme()
	e := NewEditor(theme)
	e.SetContent("test.md", content)
	e.mode = EditorModeEdit
	e.parseLines()
	e.SetSize(80, 40)
	return e
}

func TestEditor_SelectionRange_NormalizesOrder(t *testing.T) {
	e := newTestEditor("hello\nworld")

	tests := []struct {
		name                           string
		anchorRow, anchorCol           int
		endRow, endCol                 int
		wantSR, wantSC, wantER, wantEC int
	}{
		{
			name:      "forward selection",
			anchorRow: 0, anchorCol: 1,
			endRow: 0, endCol: 4,
			wantSR: 0, wantSC: 1, wantER: 0, wantEC: 4,
		},
		{
			name:      "backward selection same line",
			anchorRow: 0, anchorCol: 4,
			endRow: 0, endCol: 1,
			wantSR: 0, wantSC: 1, wantER: 0, wantEC: 4,
		},
		{
			name:      "forward multi-line",
			anchorRow: 0, anchorCol: 2,
			endRow: 1, endCol: 3,
			wantSR: 0, wantSC: 2, wantER: 1, wantEC: 3,
		},
		{
			name:      "backward multi-line",
			anchorRow: 1, anchorCol: 3,
			endRow: 0, endCol: 2,
			wantSR: 0, wantSC: 2, wantER: 1, wantEC: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e.selAnchorRow = tc.anchorRow
			e.selAnchorCol = tc.anchorCol
			e.selEndRow = tc.endRow
			e.selEndCol = tc.endCol
			e.hasSelection = true

			sr, sc, er, ec := e.selectionRange()
			if sr != tc.wantSR || sc != tc.wantSC || er != tc.wantER || ec != tc.wantEC {
				t.Errorf("got (%d,%d)-(%d,%d), want (%d,%d)-(%d,%d)",
					sr, sc, er, ec, tc.wantSR, tc.wantSC, tc.wantER, tc.wantEC)
			}
		})
	}
}

func TestEditor_IsInSelection(t *testing.T) {
	e := newTestEditor("hello\nworld\nfoo")

	// Select from (0,2) to (1,3): "llo\nwor"
	e.selAnchorRow = 0
	e.selAnchorCol = 2
	e.selEndRow = 1
	e.selEndCol = 3
	e.hasSelection = true

	tests := []struct {
		row, col int
		want     bool
	}{
		{0, 0, false}, // before selection
		{0, 1, false}, // just before selection
		{0, 2, true},  // start of selection
		{0, 3, true},  // middle of first line
		{0, 4, true},  // last char of first line
		{1, 0, true},  // start of second line
		{1, 2, true},  // inside second line
		{1, 3, false}, // end of selection (exclusive)
		{1, 4, false}, // after selection
		{2, 0, false}, // different line entirely
	}

	for _, tc := range tests {
		got := e.isInSelection(tc.row, tc.col)
		if got != tc.want {
			t.Errorf("isInSelection(%d, %d) = %v, want %v", tc.row, tc.col, got, tc.want)
		}
	}
}

func TestEditor_IsInSelection_NoSelection(t *testing.T) {
	e := newTestEditor("hello")
	// hasSelection is false by default
	if e.isInSelection(0, 0) {
		t.Error("isInSelection should return false when hasSelection is false")
	}
}

func TestEditor_DeleteSelection_SingleLine(t *testing.T) {
	e := newTestEditor("hello world")

	// Select "llo w" (0,2)-(0,7)
	e.selAnchorRow = 0
	e.selAnchorCol = 2
	e.selEndRow = 0
	e.selEndCol = 7
	e.hasSelection = true

	deleted := e.deleteSelection()
	if !deleted {
		t.Fatal("expected deleteSelection to return true")
	}

	if e.lines[0] != "heorld" {
		t.Errorf("expected 'heorld', got %q", e.lines[0])
	}
	if e.cursorRow != 0 || e.cursorCol != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", e.cursorRow, e.cursorCol)
	}
	if e.hasSelection {
		t.Error("expected hasSelection to be false after delete")
	}
}

func TestEditor_DeleteSelection_MultiLine(t *testing.T) {
	e := newTestEditor("hello\nworld\nfoo bar")

	// Select from (0,3) to (2,4): "lo\nworld\nfoo "
	e.selAnchorRow = 0
	e.selAnchorCol = 3
	e.selEndRow = 2
	e.selEndCol = 4
	e.hasSelection = true

	deleted := e.deleteSelection()
	if !deleted {
		t.Fatal("expected deleteSelection to return true")
	}

	if len(e.lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(e.lines), e.lines)
	}
	if e.lines[0] != "helbar" {
		t.Errorf("expected 'helbar', got %q", e.lines[0])
	}
	if e.cursorRow != 0 || e.cursorCol != 3 {
		t.Errorf("expected cursor at (0,3), got (%d,%d)", e.cursorRow, e.cursorCol)
	}
}

func TestEditor_DeleteSelection_BackwardSelection(t *testing.T) {
	e := newTestEditor("abcdef")

	// Backward selection: anchor at 5, end at 2 → should delete "cde"
	e.selAnchorRow = 0
	e.selAnchorCol = 5
	e.selEndRow = 0
	e.selEndCol = 2
	e.hasSelection = true

	e.deleteSelection()

	if e.lines[0] != "abf" {
		t.Errorf("expected 'abf', got %q", e.lines[0])
	}
	if e.cursorCol != 2 {
		t.Errorf("expected cursorCol 2, got %d", e.cursorCol)
	}
}

func TestEditor_DeleteSelection_NoSelection(t *testing.T) {
	e := newTestEditor("hello")
	deleted := e.deleteSelection()
	if deleted {
		t.Error("expected deleteSelection to return false when no selection")
	}
}

func TestEditor_SelectedText_SingleLine(t *testing.T) {
	e := newTestEditor("hello world")

	e.selAnchorRow = 0
	e.selAnchorCol = 6
	e.selEndRow = 0
	e.selEndCol = 11
	e.hasSelection = true

	got := e.selectedText()
	if got != "world" {
		t.Errorf("expected 'world', got %q", got)
	}
}

func TestEditor_SelectedText_MultiLine(t *testing.T) {
	e := newTestEditor("hello\nworld\nfoo")

	e.selAnchorRow = 0
	e.selAnchorCol = 3
	e.selEndRow = 2
	e.selEndCol = 2
	e.hasSelection = true

	got := e.selectedText()
	if got != "lo\nworld\nfo" {
		t.Errorf("expected 'lo\\nworld\\nfo', got %q", got)
	}
}

func TestEditor_ClearSelection(t *testing.T) {
	e := newTestEditor("hello")
	e.hasSelection = true
	e.selecting = true
	e.selAnchorRow = 0
	e.selAnchorCol = 1

	e.clearSelection()

	if e.hasSelection {
		t.Error("expected hasSelection to be false")
	}
	if e.selecting {
		t.Error("expected selecting to be false")
	}
}

func TestEditor_RenderWithSelection_NoPanic(t *testing.T) {
	e := newTestEditor("hello\nworld\nfoo bar baz")
	e.SetSize(80, 20)

	// Set up a selection
	e.selAnchorRow = 0
	e.selAnchorCol = 2
	e.selEndRow = 1
	e.selEndCol = 3
	e.hasSelection = true

	// Should render without panicking
	result := e.View()
	if result == "" {
		t.Error("expected non-empty rendering")
	}
}

func TestEditor_RenderWithSelection_WrappedLines(t *testing.T) {
	e := newTestEditor("one two three four five six seven eight nine ten")
	e.SetSize(30, 20)

	// Select across wrapped visual lines
	e.selAnchorRow = 0
	e.selAnchorCol = 4
	e.selEndRow = 0
	e.selEndCol = 20
	e.hasSelection = true

	// Should render without panicking
	result := e.View()
	if result == "" {
		t.Error("expected non-empty rendering")
	}
}

func TestEditor_SetContent_ClearsSelection(t *testing.T) {
	e := newTestEditor("hello")
	e.hasSelection = true
	e.selecting = true

	e.SetContent("new.md", "new content")

	if e.hasSelection {
		t.Error("SetContent should clear selection")
	}
	if e.selecting {
		t.Error("SetContent should clear selecting state")
	}
}
