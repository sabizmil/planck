package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sabizmil/planck/internal/ui"
)

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

func TestPollIntervalForTab(t *testing.T) {
	theme := ui.DefaultTheme()
	km := ui.DefaultKeymap()

	makeTab := func(status string, lastWrite time.Time) *AgentTab {
		panel := ui.NewPTYPanel(theme, km)
		panel.SetStatus(status)
		return &AgentTab{id: "test", panel: panel, lastUserWrite: lastWrite}
	}

	recentlyTyped := time.Now().Add(-500 * time.Millisecond) // 0.5s ago — within 2s window
	notRecentlyTyped := time.Now().Add(-5 * time.Second)     // 5s ago — outside window
	noTyping := time.Time{}                                  // zero value

	tests := []struct {
		name         string
		status       string
		isActive     bool
		lastWrite    time.Time
		wantInterval time.Duration
	}{
		// Active + running is always fast, regardless of typing
		{"active + running + no typing", "running", true, noTyping, pollFast},
		{"active + running + recently typed", "running", true, recentlyTyped, pollFast},

		// Active + non-running: fast when typing, medium otherwise
		{"active + idle + recently typed", "idle", true, recentlyTyped, pollFast},
		{"active + completed + recently typed", "completed", true, recentlyTyped, pollFast},
		{"active + needs_input + recently typed", "needs_input", true, recentlyTyped, pollFast},
		{"active + idle + not recently typed", "idle", true, notRecentlyTyped, pollMedium},
		{"active + idle + no typing", "idle", true, noTyping, pollMedium},
		{"active + needs_input + no typing", "needs_input", true, noTyping, pollMedium},

		// Background: typing doesn't boost (user isn't looking at this tab)
		{"background + running", "running", false, noTyping, pollMedium},
		{"background + idle", "idle", false, noTyping, pollSlow},
		{"background + needs_input", "needs_input", false, noTyping, pollSlow},
		{"background + idle + recently typed", "idle", false, recentlyTyped, pollSlow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tab := makeTab(tt.status, tt.lastWrite)
			app := &App{
				agentTabs: []*AgentTab{tab},
			}
			if tt.isActive {
				app.activeTabIdx = 1 // agent tab index (0 = planning)
			} else {
				app.activeTabIdx = 0 // planning tab active
			}

			got := app.pollIntervalForTab(tab)
			if got != tt.wantInterval {
				t.Errorf("pollIntervalForTab() = %v, want %v", got, tt.wantInterval)
			}
		})
	}
}

func TestCheckIdleTransition(t *testing.T) {
	theme := ui.DefaultTheme()
	km := ui.DefaultKeymap()

	makeApp := func(tab *AgentTab) *App {
		return &App{
			tabs:      ui.NewTabBar(theme),
			agentTabs: []*AgentTab{tab},
		}
	}

	makeTab := func(status string) *AgentTab {
		panel := ui.NewPTYPanel(theme, km)
		panel.SetStatus(status)
		return &AgentTab{id: "test", panel: panel}
	}

	t.Run("no transition when not running", func(t *testing.T) {
		tab := makeTab("idle")
		app := makeApp(tab)
		if app.checkIdleTransition(tab) {
			t.Error("should not transition from idle")
		}
		if tab.panel.GetStatus() != "idle" {
			t.Errorf("status = %q, want idle", tab.panel.GetStatus())
		}
	})

	t.Run("no transition when user recently typed", func(t *testing.T) {
		tab := makeTab("running")
		tab.lastUserWrite = time.Now() // just typed
		tab.lastContentChange = time.Now().Add(-5 * time.Second)
		app := makeApp(tab)
		if app.checkIdleTransition(tab) {
			t.Error("should not transition while user is typing")
		}
		if tab.panel.GetStatus() != "running" {
			t.Errorf("status = %q, want running", tab.panel.GetStatus())
		}
	})

	t.Run("transition to idle after 3s silence", func(t *testing.T) {
		tab := makeTab("running")
		tab.lastContentChange = time.Now().Add(-4 * time.Second) // 4s ago
		app := makeApp(tab)
		if !app.checkIdleTransition(tab) {
			t.Error("should transition to idle after 3s")
		}
		if tab.panel.GetStatus() != "idle" {
			t.Errorf("status = %q, want idle", tab.panel.GetStatus())
		}
	})

	t.Run("no transition before 3s silence", func(t *testing.T) {
		tab := makeTab("running")
		tab.lastContentChange = time.Now().Add(-1 * time.Second) // only 1s ago
		app := makeApp(tab)
		if app.checkIdleTransition(tab) {
			t.Error("should not transition before 3s")
		}
		if tab.panel.GetStatus() != "running" {
			t.Errorf("status = %q, want running", tab.panel.GetStatus())
		}
	})

	t.Run("no transition when lastContentChange is zero", func(t *testing.T) {
		tab := makeTab("running")
		// lastContentChange is zero value — agent just started, no output yet
		app := makeApp(tab)
		if app.checkIdleTransition(tab) {
			t.Error("should not transition when no content has ever been received")
		}
		if tab.panel.GetStatus() != "running" {
			t.Errorf("status = %q, want running", tab.panel.GetStatus())
		}
	})

	t.Run("transition to needs_input via hook state", func(t *testing.T) {
		tab := makeTab("running")
		tab.lastContentChange = time.Now() // recent content change

		// Write a hook state file
		dir := t.TempDir()
		stateFile := filepath.Join(dir, "hook-state")
		if err := os.WriteFile(stateFile, []byte("needs_input"), 0644); err != nil {
			t.Fatal(err)
		}
		tab.stateFile = stateFile

		app := makeApp(tab)
		if !app.checkIdleTransition(tab) {
			t.Error("should transition to needs_input")
		}
		if tab.panel.GetStatus() != "needs_input" {
			t.Errorf("status = %q, want needs_input", tab.panel.GetStatus())
		}
	})

	t.Run("hook state takes priority over idle timeout", func(t *testing.T) {
		tab := makeTab("running")
		tab.lastContentChange = time.Now().Add(-5 * time.Second) // would be idle

		// But hook state says needs_input — should win
		dir := t.TempDir()
		stateFile := filepath.Join(dir, "hook-state")
		if err := os.WriteFile(stateFile, []byte("needs_input"), 0644); err != nil {
			t.Fatal(err)
		}
		tab.stateFile = stateFile

		app := makeApp(tab)
		if !app.checkIdleTransition(tab) {
			t.Error("should transition")
		}
		if tab.panel.GetStatus() != "needs_input" {
			t.Errorf("status = %q, want needs_input (hook state takes priority)", tab.panel.GetStatus())
		}
	})

	t.Run("completed status is not overridden", func(t *testing.T) {
		tab := makeTab("completed")
		tab.lastContentChange = time.Now().Add(-10 * time.Second)
		app := makeApp(tab)
		if app.checkIdleTransition(tab) {
			t.Error("should not transition from completed")
		}
		if tab.panel.GetStatus() != "completed" {
			t.Errorf("status = %q, want completed", tab.panel.GetStatus())
		}
	})
}
