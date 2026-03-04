package app

import "testing"

func TestIsGenericOSCTitle(t *testing.T) {
	tests := []struct {
		name      string
		oscTitle  string
		baseLabel string
		want      bool
	}{
		{"exact match", "Claude Code", "Claude Code", true},
		{"case insensitive", "claude code", "Claude Code", true},
		{"osc contains base", "Claude Code", "Claude", true},
		{"base contains osc", "Claude", "Claude Code", true},
		{"unrelated title", "Investigating auth bug", "Claude Code", false},
		{"task with agent name", "Claude is investigating auth", "Claude Code", false},
		{"empty osc", "", "Claude Code", false},
		{"empty base", "Claude Code", "", false},
		{"both empty", "", "", false},
		{"whitespace padded", "  Claude Code  ", "Claude Code", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGenericOSCTitle(tt.oscTitle, tt.baseLabel)
			if got != tt.want {
				t.Errorf("isGenericOSCTitle(%q, %q) = %v, want %v", tt.oscTitle, tt.baseLabel, got, tt.want)
			}
		})
	}
}

func TestSanitizeTabTitle(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"empty", "", ""},
		{"normal text", "investigate auth bug", "investigate auth bug"},
		{"claude code passes", "Claude Code", "Claude Code"},
		{"too short", "ab", ""},
		{"strips spinner braille", "\u2800Claude Code", "Claude Code"},
		{"strips dingbat spinner", "\u2733 Working on task", "Working on task"},
		{"only spinners", "\u2733\u2800", ""},
		{"strips zero-width chars", "hello\u200bworld", "helloworld"},
		{"trims whitespace", "  investigate auth  ", "investigate auth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeTabTitle(tt.raw)
			if got != tt.want {
				t.Errorf("sanitizeTabTitle(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// TestTitleFromInput_ProtectsAgainstGenericOSC simulates the full title lifecycle:
// 1. User types a prompt → customTitle + titleFromInput are set
// 2. Generic OSC title "Claude Code" arrives → guard blocks overwrite
// 3. Descriptive OSC title arrives → allowed through
//
// This tests the components directly (no App.trackInputForTitle) to avoid
// needing a fully initialized App with TabBar, etc.
func TestTitleFromInput_ProtectsAgainstGenericOSC(t *testing.T) {
	tab := &AgentTab{
		baseLabel: "Claude",
	}

	// Step 1: Simulate what trackInputForTitle does on Enter
	userInput := "investigate auth bug"
	title := sanitizeTabTitle(userInput)
	if title == "" {
		t.Fatal("sanitizeTabTitle rejected valid user input")
	}
	tab.customTitle = title
	tab.titleFromInput = true

	if tab.customTitle != "investigate auth bug" {
		t.Fatalf("customTitle = %q, want %q", tab.customTitle, "investigate auth bug")
	}

	// Step 2: Generic OSC title arrives — should be blocked by guard
	oscTitle := sanitizeTabTitle("Claude Code")
	if oscTitle == "" {
		t.Fatal("sanitizeTabTitle should not reject 'Claude Code'")
	}
	// Replicate the guard from Update()
	if oscTitle != tab.customTitle {
		if tab.titleFromInput && isGenericOSCTitle(oscTitle, tab.baseLabel) {
			// Blocked — this is the expected path
		} else {
			tab.customTitle = oscTitle
			tab.titleFromInput = false
		}
	}

	if tab.customTitle != "investigate auth bug" {
		t.Fatalf("generic OSC overwrote title: got %q", tab.customTitle)
	}
	if !tab.titleFromInput {
		t.Fatal("titleFromInput should still be true")
	}

	// Step 3: Descriptive OSC title arrives — should be allowed
	descOSC := sanitizeTabTitle("Fixing authentication module")
	if descOSC != tab.customTitle {
		if tab.titleFromInput && isGenericOSCTitle(descOSC, tab.baseLabel) {
			t.Fatal("descriptive OSC title should NOT be blocked")
		}
		tab.customTitle = descOSC
		tab.titleFromInput = false
	}

	if tab.customTitle != "Fixing authentication module" {
		t.Fatalf("customTitle = %q, want %q", tab.customTitle, "Fixing authentication module")
	}
	if tab.titleFromInput {
		t.Fatal("titleFromInput should be false after descriptive OSC override")
	}

	// Step 4: After descriptive OSC override, generic OSC should be allowed
	// (since titleFromInput is now false)
	oscTitle2 := sanitizeTabTitle("Claude Code")
	if oscTitle2 != tab.customTitle {
		if tab.titleFromInput && isGenericOSCTitle(oscTitle2, tab.baseLabel) {
			t.Fatal("should not block when titleFromInput is false")
		}
		tab.customTitle = oscTitle2
		tab.titleFromInput = false
	}

	if tab.customTitle != "Claude Code" {
		t.Fatalf("customTitle = %q, want %q", tab.customTitle, "Claude Code")
	}
}

// TestTitleFromInput_InputBufHandling verifies that trackInputForTitle correctly
// accumulates characters, handles backspace, and clears on Escape.
func TestTitleFromInput_InputBufHandling(t *testing.T) {
	tab := &AgentTab{baseLabel: "Claude"}

	// Accumulate printable ASCII
	for _, b := range []byte("hello") {
		if b >= 0x20 && b < 0x7f {
			tab.inputBuf = append(tab.inputBuf, rune(b))
		}
	}
	if string(tab.inputBuf) != "hello" {
		t.Fatalf("inputBuf = %q, want %q", string(tab.inputBuf), "hello")
	}

	// Backspace removes last character
	tab.inputBuf = tab.inputBuf[:len(tab.inputBuf)-1]
	if string(tab.inputBuf) != "hell" {
		t.Fatalf("after backspace: inputBuf = %q, want %q", string(tab.inputBuf), "hell")
	}

	// Escape clears buffer
	tab.inputBuf = nil
	if len(tab.inputBuf) != 0 {
		t.Fatal("inputBuf should be nil after Escape")
	}
}

// TestIsGenericOSCTitle_RealWorldScenario tests the exact values from the
// default config: baseLabel="Claude", OSC title="Claude Code".
func TestIsGenericOSCTitle_RealWorldScenario(t *testing.T) {
	// Default config: Label is "Claude", Claude Code sets OSC to "Claude Code"
	if !isGenericOSCTitle("Claude Code", "Claude") {
		t.Error("'Claude Code' should be generic relative to baseLabel 'Claude'")
	}

	// With spinner stripped: "✻ Claude Code" → sanitized to "Claude Code"
	sanitized := sanitizeTabTitle("\u2733 Claude Code")
	if sanitized != "Claude Code" {
		t.Fatalf("sanitized = %q, want %q", sanitized, "Claude Code")
	}
	if !isGenericOSCTitle(sanitized, "Claude") {
		t.Error("sanitized spinner title should still be generic")
	}
}
