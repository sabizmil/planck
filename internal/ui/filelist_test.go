package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sabizmil/planck/internal/workspace"
)

func newTestFileList(files []*workspace.File) *FileList {
	fl := NewFileList(DefaultTheme())
	fl.SetSize(24, 20)
	fl.SetPosition(2) // tab bar label row + tab bar border row
	fl.SetFocused(true)
	fl.SetFiles(files)
	return fl
}

func testFiles() []*workspace.File {
	return []*workspace.File{
		{Name: "alpha.md", Status: workspace.StatusPending},
		{Name: "beta.md", Status: workspace.StatusInProgress},
		{Name: "subdir/child1.md", Status: workspace.StatusPending},
		{Name: "subdir/child2.md", Status: workspace.StatusCompleted},
		{Name: "gamma.md", Status: workspace.StatusPending},
	}
}

func TestFileList_ScrollBy(t *testing.T) {
	// Create a file list with many files that exceed visible area
	var files []*workspace.File
	for i := 0; i < 30; i++ {
		files = append(files, &workspace.File{
			Name:   "file" + string(rune('a'+i%26)) + ".md",
			Status: workspace.StatusPending,
		})
	}
	fl := newTestFileList(files)
	fl.SetSize(24, 10) // small height = fewer visible lines

	if fl.offset != 0 {
		t.Fatalf("expected initial offset 0, got %d", fl.offset)
	}

	// Scroll down
	fl.ScrollBy(3)
	if fl.offset != 3 {
		t.Errorf("expected offset 3 after ScrollBy(3), got %d", fl.offset)
	}

	// Cursor should have been pushed into view
	if fl.cursor < fl.offset {
		t.Errorf("cursor %d is above offset %d after scroll down", fl.cursor, fl.offset)
	}

	// Scroll back up past top
	fl.ScrollBy(-10)
	if fl.offset != 0 {
		t.Errorf("expected offset clamped to 0, got %d", fl.offset)
	}

	// Scroll way past bottom
	fl.ScrollBy(1000)
	maxOffset := len(fl.visible) - fl.visibleLines()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if fl.offset != maxOffset {
		t.Errorf("expected offset clamped to %d, got %d", maxOffset, fl.offset)
	}
}

func TestFileList_ScrollBy_EmptyList(t *testing.T) {
	fl := newTestFileList(nil)
	// Should not panic on empty list
	fl.ScrollBy(3)
	fl.ScrollBy(-3)
	if fl.offset != 0 {
		t.Errorf("expected offset 0 for empty list, got %d", fl.offset)
	}
}

func TestFileList_HandleMouse_WheelScrollsFileList(t *testing.T) {
	var files []*workspace.File
	for i := 0; i < 30; i++ {
		files = append(files, &workspace.File{
			Name:   "file" + string(rune('a'+i%26)) + ".md",
			Status: workspace.StatusPending,
		})
	}
	fl := newTestFileList(files)
	fl.SetSize(24, 10)

	// Wheel down should scroll the file list
	action := fl.HandleMouse(tea.MouseMsg{
		X:      5,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
	})
	if action != ClickNone {
		t.Errorf("expected ClickNone for wheel, got %d", action)
	}
	if fl.offset == 0 {
		t.Error("expected offset > 0 after wheel down")
	}

	// Wheel up should scroll back
	prevOffset := fl.offset
	fl.HandleMouse(tea.MouseMsg{
		X:      5,
		Y:      5,
		Button: tea.MouseButtonWheelUp,
	})
	if fl.offset >= prevOffset {
		t.Error("expected offset to decrease after wheel up")
	}
}

func TestFileList_HandleMouse_ClickFile(t *testing.T) {
	fl := newTestFileList(testFiles())
	// Files layout (after tree build):
	// visible[0] = subdir (dir, expanded)
	// visible[1] = subdir/child1.md
	// visible[2] = subdir/child2.md
	// visible[3] = alpha.md
	// visible[4] = beta.md
	// visible[5] = gamma.md

	// screenY=2 (tab bar label + border), header=2 lines (FILES + separator)
	// So file at visible index 0 is at screen row 2+2=4
	// File at visible index 3 (alpha.md) is at screen row 2+2+3=7

	action := fl.HandleMouse(tea.MouseMsg{
		X:      5,
		Y:      7, // screenY(2) + header(2) + index(3) = row 7
		Button: tea.MouseButtonLeft,
	})
	if action != ClickFile {
		t.Errorf("expected ClickFile, got %d", action)
	}
	if fl.cursor != 3 {
		t.Errorf("expected cursor at 3, got %d", fl.cursor)
	}
	if fl.SelectedFile() == nil || fl.SelectedFile().Name != "alpha.md" {
		t.Errorf("expected alpha.md selected, got %v", fl.SelectedFile())
	}
}

func TestFileList_HandleMouse_ClickDirToggles(t *testing.T) {
	fl := newTestFileList(testFiles())
	// visible[0] should be "subdir" (expanded dir)
	if !fl.visible[0].isDir {
		t.Fatal("expected visible[0] to be a directory")
	}
	if !fl.visible[0].expanded {
		t.Fatal("expected subdir to be expanded initially")
	}
	initialVisibleCount := len(fl.visible)

	// Click on the directory (index 0 → screen row 2+2+0=4)
	action := fl.HandleMouse(tea.MouseMsg{
		X:      5,
		Y:      4,
		Button: tea.MouseButtonLeft,
	})
	if action != ClickDirToggle {
		t.Errorf("expected ClickDirToggle, got %d", action)
	}
	// Directory should now be collapsed, hiding its children
	if fl.visible[0].expanded {
		t.Error("expected subdir to be collapsed after click")
	}
	if len(fl.visible) >= initialVisibleCount {
		t.Errorf("expected fewer visible nodes after collapse, got %d (was %d)",
			len(fl.visible), initialVisibleCount)
	}

	// Click again to expand
	action = fl.HandleMouse(tea.MouseMsg{
		X:      5,
		Y:      4,
		Button: tea.MouseButtonLeft,
	})
	if action != ClickDirToggle {
		t.Errorf("expected ClickDirToggle on re-expand, got %d", action)
	}
	if !fl.visible[0].expanded {
		t.Error("expected subdir to be expanded after second click")
	}
	if len(fl.visible) != initialVisibleCount {
		t.Errorf("expected %d visible nodes after re-expand, got %d",
			initialVisibleCount, len(fl.visible))
	}
}

func TestFileList_HandleMouse_ClickOutOfRange(t *testing.T) {
	fl := newTestFileList(testFiles())

	// Click above the content area (in the header)
	action := fl.HandleMouse(tea.MouseMsg{
		X:      5,
		Y:      2, // screenY(2) + 0 = in header area
		Button: tea.MouseButtonLeft,
	})
	if action != ClickNone {
		t.Errorf("expected ClickNone for header click, got %d", action)
	}

	// Click below all items
	action = fl.HandleMouse(tea.MouseMsg{
		X:      5,
		Y:      50, // way below
		Button: tea.MouseButtonLeft,
	})
	if action != ClickNone {
		t.Errorf("expected ClickNone for click below items, got %d", action)
	}
}

func TestFileList_HandleMouse_IgnoredInMoveMode(t *testing.T) {
	fl := newTestFileList(testFiles())
	fl.SetCursor(3) // select a file
	fl.EnterMoveMode()

	action := fl.HandleMouse(tea.MouseMsg{
		X:      5,
		Y:      5,
		Button: tea.MouseButtonLeft,
	})
	if action != ClickNone {
		t.Errorf("expected ClickNone in move mode, got %d", action)
	}
}

func TestFileList_HandleMouse_ClickSwitchesFocus(t *testing.T) {
	// This tests the integration concern: clicking a file should cause
	// the app to switch focus. We verify here that HandleMouse returns
	// ClickFile so the app layer can react.
	fl := newTestFileList(testFiles())
	fl.SetFocused(false) // simulate editor having focus

	// Click on a file
	action := fl.HandleMouse(tea.MouseMsg{
		X:      5,
		Y:      7, // alpha.md at index 3
		Button: tea.MouseButtonLeft,
	})
	if action != ClickFile {
		t.Errorf("expected ClickFile even when unfocused, got %d", action)
	}
	// The FileList itself doesn't manage app-level focus, but cursor should still move
	if fl.cursor != 3 {
		t.Errorf("expected cursor at 3, got %d", fl.cursor)
	}
}
