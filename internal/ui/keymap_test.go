package ui

import (
	"testing"
)

func TestDefaultKeymap_AllContextsPresent(t *testing.T) {
	km := DefaultKeymap()

	expectedContexts := AllContexts()
	for _, ctx := range expectedContexts {
		found := false
		for _, cb := range km.Contexts {
			if cb.Context == ctx {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultKeymap missing context %q", ctx)
		}
	}
}

func TestDefaultKeymap_Matches(t *testing.T) {
	km := DefaultKeymap()

	tests := []struct {
		ctx    Context
		action Action
		key    string
		want   bool
	}{
		{ContextGlobal, ActionQuit, "q", true},
		{ContextGlobal, ActionQuit, "x", false},
		{ContextGlobal, ActionToggleHelp, "?", true},
		{ContextGlobal, ActionNextTab, "shift+tab", true},
		{ContextGlobal, ActionCloseTab, "ctrl+x", true},
		{ContextGlobal, ActionSettings, "s", true},
		{ContextGlobal, ActionCreateAgent, "a", true},
		{ContextGlobal, ActionCloseAgent, "x", true},
		{ContextFileList, ActionMoveDown, "j", true},
		{ContextFileList, ActionMoveDown, "down", true},
		{ContextFileList, ActionMoveDown, "x", false},
		{ContextFileList, ActionOpenFile, "enter", true},
		{ContextEditor, ActionEditorEdit, "e", true},
		{ContextEditor, ActionEditorDown, "j", true},
		{ContextEditor, ActionEditorDown, "down", true},
		{ContextAgentNormal, ActionEnterInput, "i", true},
		{ContextAgentNormal, ActionEnterInput, "enter", true},
		{ContextAgentInput, ActionExitInput, `ctrl+\`, true},
	}

	for _, tc := range tests {
		got := km.Matches(tc.ctx, tc.action, tc.key)
		if got != tc.want {
			t.Errorf("Matches(%s, %s, %q) = %v, want %v", tc.ctx, tc.action, tc.key, got, tc.want)
		}
	}
}

func TestDefaultKeymap_KeysFor(t *testing.T) {
	km := DefaultKeymap()

	keys := km.KeysFor(ContextGlobal, ActionQuit)
	if len(keys) != 1 || keys[0] != "q" {
		t.Errorf("KeysFor(Global, Quit) = %v, want [q]", keys)
	}

	keys = km.KeysFor(ContextFileList, ActionMoveDown)
	if len(keys) != 2 {
		t.Errorf("KeysFor(FileList, MoveDown) = %v, want 2 keys", keys)
	}
}

func TestDefaultKeymap_ActionFor(t *testing.T) {
	km := DefaultKeymap()

	action := km.ActionFor(ContextGlobal, "q")
	if action != ActionQuit {
		t.Errorf("ActionFor(Global, q) = %q, want %q", action, ActionQuit)
	}

	action = km.ActionFor(ContextGlobal, "nonexistent")
	if action != "" {
		t.Errorf("ActionFor(Global, nonexistent) = %q, want empty", action)
	}
}

func TestKeymap_SetBinding(t *testing.T) {
	km := DefaultKeymap()

	ok := km.SetBinding(ContextGlobal, ActionQuit, []string{"Q"})
	if !ok {
		t.Fatal("SetBinding should return true")
	}

	if !km.Matches(ContextGlobal, ActionQuit, "Q") {
		t.Error("After SetBinding, Q should match Quit")
	}
	if km.Matches(ContextGlobal, ActionQuit, "q") {
		t.Error("After SetBinding, q should no longer match Quit")
	}
}

func TestKeymap_ApplyOverrides(t *testing.T) {
	km := DefaultKeymap()

	overrides := map[string]map[string]string{
		"global": {
			"quit": "Q",
		},
		"file_list": {
			"move_down": "J,arrow_down",
		},
	}

	km.ApplyOverrides(overrides)

	if !km.Matches(ContextGlobal, ActionQuit, "Q") {
		t.Error("After override, Q should match Quit")
	}
	if km.Matches(ContextGlobal, ActionQuit, "q") {
		t.Error("After override, q should no longer match Quit")
	}
	if !km.Matches(ContextFileList, ActionMoveDown, "J") {
		t.Error("After override, J should match MoveDown")
	}
}

func TestKeymap_ApplyOverrides_UnknownIgnored(t *testing.T) {
	km := DefaultKeymap()

	overrides := map[string]map[string]string{
		"nonexistent_context": {
			"quit": "Q",
		},
		"global": {
			"nonexistent_action": "Q",
		},
	}

	// Should not panic
	km.ApplyOverrides(overrides)

	// Original binding should be unchanged
	if !km.Matches(ContextGlobal, ActionQuit, "q") {
		t.Error("Original binding should be preserved")
	}
}

func TestKeymap_IsCustomized(t *testing.T) {
	km := DefaultKeymap()

	if km.IsCustomized(ContextGlobal, ActionQuit) {
		t.Error("Default keymap should not be customized")
	}

	km.SetBinding(ContextGlobal, ActionQuit, []string{"Q"})

	if !km.IsCustomized(ContextGlobal, ActionQuit) {
		t.Error("After SetBinding, should be customized")
	}
}

func TestKeymap_Clone(t *testing.T) {
	km := DefaultKeymap()
	clone := km.Clone()

	// Modify clone
	clone.SetBinding(ContextGlobal, ActionQuit, []string{"Q"})

	// Original should be unchanged
	if !km.Matches(ContextGlobal, ActionQuit, "q") {
		t.Error("Original should be unchanged after cloning and modifying")
	}
	if km.Matches(ContextGlobal, ActionQuit, "Q") {
		t.Error("Original should not have the cloned modification")
	}
}

func TestKeymap_DisplayKeysFor(t *testing.T) {
	km := DefaultKeymap()

	display := km.DisplayKeysFor(ContextFileList, ActionMoveDown)
	if display == "" {
		t.Error("DisplayKeysFor should return non-empty string")
	}
	// Should contain arrow character
	if display != "j / ↓" {
		t.Errorf("DisplayKeysFor(FileList, MoveDown) = %q, want %q", display, "j / ↓")
	}
}

func TestKeymap_ValidAction(t *testing.T) {
	km := DefaultKeymap()

	if !km.ValidAction(ContextGlobal, ActionQuit) {
		t.Error("ActionQuit should be valid in Global context")
	}
	if km.ValidAction(ContextGlobal, "nonexistent") {
		t.Error("nonexistent should not be valid")
	}
}

func TestKeymap_ValidContext(t *testing.T) {
	km := DefaultKeymap()

	if !km.ValidContext(ContextGlobal) {
		t.Error("Global should be a valid context")
	}
	if km.ValidContext("nonexistent") {
		t.Error("nonexistent should not be a valid context")
	}
}

func TestFormatKeys(t *testing.T) {
	tests := []struct {
		keys []string
		want string
	}{
		{[]string{"j", "down"}, "j / ↓"},
		{[]string{"enter"}, "Enter"},
		{[]string{"ctrl+x"}, "Ctrl+x"},
		{[]string{`ctrl+\`}, `Ctrl+\`},
	}

	for _, tc := range tests {
		got := formatKeys(tc.keys)
		if got != tc.want {
			t.Errorf("formatKeys(%v) = %q, want %q", tc.keys, got, tc.want)
		}
	}
}

func TestSplitKeys(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"j,down", []string{"j", "down"}},
		{"enter", []string{"enter"}},
		{"  q , x  ", []string{"q", "x"}},
		{"", nil},
	}

	for _, tc := range tests {
		got := splitKeys(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("splitKeys(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i, g := range got {
			if g != tc.want[i] {
				t.Errorf("splitKeys(%q)[%d] = %q, want %q", tc.input, i, g, tc.want[i])
			}
		}
	}
}
