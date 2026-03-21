package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewDialog(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)

	if dialog == nil {
		t.Fatal("NewDialog() returned nil")
	}

	if dialog.IsVisible() {
		t.Error("Dialog should not be visible initially")
	}
}

func TestDialogShowConfirm(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	var result *DialogResult
	dialog.ShowConfirm("Test Title", "Are you sure?", func(r DialogResult) {
		result = &r
	})

	if !dialog.IsVisible() {
		t.Error("Dialog should be visible after ShowConfirm")
	}

	// View should not be empty
	view := dialog.View()
	if view == "" {
		t.Error("View() should not be empty when visible")
	}

	// Test Y key confirms
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if result == nil {
		t.Fatal("Callback was not called")
	}
	if !result.Confirmed {
		t.Error("Y key should confirm")
	}
	if dialog.IsVisible() {
		t.Error("Dialog should be hidden after confirmation")
	}
}

func TestDialogShowConfirmCancel(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	var result *DialogResult
	dialog.ShowConfirm("Test", "Message", func(r DialogResult) {
		result = &r
	})

	// Test N key cancels
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if result == nil {
		t.Fatal("Callback was not called")
	}
	if result.Confirmed {
		t.Error("N key should cancel")
	}
}

func TestDialogShowInput(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	var result *DialogResult
	dialog.ShowInput("Enter Name", "Name:", func(r DialogResult) {
		result = &r
	})

	if !dialog.IsVisible() {
		t.Error("Dialog should be visible after ShowInput")
	}

	// Type some characters
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	// Press enter
	dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if result == nil {
		t.Fatal("Callback was not called")
	}
	if !result.Confirmed {
		t.Error("Enter should confirm")
	}
	if result.Input != "hi" {
		t.Errorf("Input = %s, want 'hi'", result.Input)
	}
}

func TestDialogShowInputBackspace(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	var result *DialogResult
	dialog.ShowInput("Test", "Prompt:", func(r DialogResult) {
		result = &r
	})

	// Type and backspace
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	dialog.Update(tea.KeyMsg{Type: tea.KeyBackspace})

	dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if result.Input != "a" {
		t.Errorf("Input after backspace = %s, want 'a'", result.Input)
	}
}

func TestDialogShowSelect(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	options := []DialogOption{
		{Label: "Option A", Description: "First option"},
		{Label: "Option B", Description: "Second option"},
		{Label: "Option C", Description: "Third option"},
	}

	var result *DialogResult
	dialog.ShowSelect("Choose", options, func(r DialogResult) {
		result = &r
	})

	if !dialog.IsVisible() {
		t.Error("Dialog should be visible after ShowSelect")
	}

	// Navigate down
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if result == nil {
		t.Fatal("Callback was not called")
	}
	if result.Selected != 1 {
		t.Errorf("Selected = %d, want 1", result.Selected)
	}
}

func TestDialogSelectNavigation(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	options := []DialogOption{
		{Label: "A"},
		{Label: "B"},
		{Label: "C"},
	}

	var result *DialogResult
	dialog.ShowSelect("Test", options, func(r DialogResult) {
		result = &r
	})

	// Move down past end (should stay at last)
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if result.Selected != 2 {
		t.Errorf("Selected = %d, want 2 (last)", result.Selected)
	}

	// Test up navigation
	dialog.ShowSelect("Test", options, func(r DialogResult) {
		result = &r
	})

	// Move up from start (should stay at 0)
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if result.Selected != 0 {
		t.Errorf("Selected after up = %d, want 0", result.Selected)
	}
}

func TestDialogShowScopePicker(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	var result *DialogResult
	dialog.ShowScopePicker(3, 2, 7, func(r DialogResult) {
		result = &r
	})

	if !dialog.IsVisible() {
		t.Error("Dialog should be visible after ShowScopePicker")
	}

	// Select second option (phase)
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dialog.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if result.Selected != 1 {
		t.Errorf("Selected = %d, want 1", result.Selected)
	}
}

func TestDialogShowPermission(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	var result *DialogResult
	dialog.ShowPermissionDialog(func(r DialogResult) {
		result = &r
	})

	if !dialog.IsVisible() {
		t.Error("Dialog should be visible after ShowPermissionDialog")
	}

	view := dialog.View()
	if view == "" {
		t.Error("View should not be empty")
	}

	// Approve with Y
	dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})

	if !result.Confirmed {
		t.Error("Y should approve")
	}
}

func TestDialogHide(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	dialog.ShowConfirm("Test", "Message", func(r DialogResult) {})

	if !dialog.IsVisible() {
		t.Error("Should be visible after show")
	}

	dialog.Hide()

	if dialog.IsVisible() {
		t.Error("Should be hidden after Hide()")
	}
}

func TestDialogViewWhenHidden(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	view := dialog.View()
	if view != "" {
		t.Error("View() should be empty when hidden")
	}
}

func TestDialogUpdateWhenHidden(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)

	// Update when hidden should do nothing
	_, cmd := dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if cmd != nil {
		t.Error("Update when hidden should return nil cmd")
	}
}

func TestDialogEscCancel(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	var result *DialogResult
	dialog.ShowSelect("Test", []DialogOption{{Label: "A"}}, func(r DialogResult) {
		result = &r
	})

	dialog.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if result == nil {
		t.Fatal("Callback was not called")
	}
	if result.Confirmed {
		t.Error("Escape should cancel")
	}
}

func TestDialog_ShowSelect_ClearsStaleMessage(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	// First show a confirm dialog (sets message)
	dialog.ShowConfirm("Delete File?", "Delete 'test.md'?", func(r DialogResult) {})
	dialog.Update(tea.KeyMsg{Type: tea.KeyEscape}) // dismiss

	// Now show a select dialog
	dialog.ShowSelect("New Agent", []DialogOption{
		{Label: "Claude", Description: "claude"},
	}, func(r DialogResult) {})

	if dialog.message != "" {
		t.Errorf("ShowSelect should clear stale message, got %q", dialog.message)
	}

	// Verify the rendered view does not contain the old message
	view := dialog.View()
	if contains(view, "Delete") {
		t.Error("Select dialog should not show stale 'Delete' message from previous confirm dialog")
	}
}

func TestDialog_ShowScopePicker_ClearsStaleMessage(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	dialog.ShowConfirm("Delete?", "Delete 'foo.md'?", func(r DialogResult) {})
	dialog.Update(tea.KeyMsg{Type: tea.KeyEscape})

	dialog.ShowScopePicker(3, 1, 5, func(r DialogResult) {})

	if dialog.message != "" {
		t.Errorf("ShowScopePicker should clear stale message, got %q", dialog.message)
	}
}

func TestDialog_ShowPermission_ClearsStaleMessage(t *testing.T) {
	theme := DefaultTheme()
	dialog := NewDialog(theme)
	dialog.SetSize(80, 24)

	dialog.ShowConfirm("Delete?", "Delete 'foo.md'?", func(r DialogResult) {})
	dialog.Update(tea.KeyMsg{Type: tea.KeyEscape})

	dialog.ShowPermissionDialog(func(r DialogResult) {})

	if dialog.message != "" {
		t.Errorf("ShowPermissionDialog should clear stale message, got %q", dialog.message)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
