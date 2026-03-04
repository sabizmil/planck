package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Check agents
	if len(cfg.Agents) == 0 {
		t.Error("No agents in default config")
	}

	// Check claude agent exists
	if agent, ok := cfg.Agents["claude-code"]; !ok {
		t.Error("claude-code agent not found in default config")
	} else {
		if agent.Command != "claude" {
			t.Errorf("Expected claude command, got %s", agent.Command)
		}
		if !agent.Default {
			t.Error("claude-code should be default agent")
		}
	}

	// Check notifications
	if !cfg.Notifications.Bell {
		t.Error("Default bell notification should be true")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "no agents",
			cfg: &Config{
				Agents:      map[string]AgentConfig{},
				Preferences: DefaultConfig().Preferences,
				Execution:   DefaultConfig().Execution,
			},
			wantErr: true,
		},
		{
			name: "agent without command",
			cfg: &Config{
				Agents: map[string]AgentConfig{
					"test": {Command: ""},
				},
				Preferences: DefaultConfig().Preferences,
				Execution:   DefaultConfig().Execution,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidate_Keybindings(t *testing.T) {
	tests := []struct {
		name    string
		kb      map[string]map[string]string
		wantErr bool
	}{
		{
			name:    "nil keybindings is valid",
			kb:      nil,
			wantErr: false,
		},
		{
			name: "valid overrides",
			kb: map[string]map[string]string{
				"global": {"quit": "Q"},
			},
			wantErr: false,
		},
		{
			name: "empty context name",
			kb: map[string]map[string]string{
				"": {"quit": "Q"},
			},
			wantErr: true,
		},
		{
			name: "empty action name",
			kb: map[string]map[string]string{
				"global": {"": "Q"},
			},
			wantErr: true,
		},
		{
			name: "empty key binding",
			kb: map[string]map[string]string{
				"global": {"quit": ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Keybindings = tt.kb
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigThemePreset(t *testing.T) {
	cfg := DefaultConfig()
	// Default should be empty (which means "default" theme)
	if cfg.Preferences.ThemePreset != "" {
		t.Errorf("Default ThemePreset should be empty, got %q", cfg.Preferences.ThemePreset)
	}

	// Setting it should work
	cfg.Preferences.ThemePreset = "nord"
	if cfg.Preferences.ThemePreset != "nord" {
		t.Errorf("ThemePreset should be 'nord', got %q", cfg.Preferences.ThemePreset)
	}
}

func TestGetEditor(t *testing.T) {
	tests := []struct {
		name          string
		cfgEditor     string
		envEditor     string
		envVisual     string
		expectedStart string
	}{
		{
			name:          "config editor",
			cfgEditor:     "nano",
			envEditor:     "",
			envVisual:     "",
			expectedStart: "nano",
		},
		{
			name:          "env EDITOR fallback",
			cfgEditor:     "",
			envEditor:     "emacs",
			envVisual:     "",
			expectedStart: "emacs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env
			oldEditor := os.Getenv("EDITOR")
			oldVisual := os.Getenv("VISUAL")
			defer func() {
				os.Setenv("EDITOR", oldEditor)
				os.Setenv("VISUAL", oldVisual)
			}()

			os.Setenv("EDITOR", tt.envEditor)
			os.Setenv("VISUAL", tt.envVisual)

			cfg := &Config{
				Preferences: Preferences{
					Editor: tt.cfgEditor,
				},
			}

			editor := cfg.GetEditor()
			if editor != tt.expectedStart {
				t.Errorf("GetEditor() = %s, want %s", editor, tt.expectedStart)
			}
		})
	}
}

func TestGetDefaultAgent(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"first":  {Command: "first", Default: false},
			"second": {Command: "second", Default: true},
		},
	}

	// Should return the one marked as default
	name, agent := cfg.GetDefaultAgent()
	if name != "second" {
		t.Errorf("GetDefaultAgent() name = %s, want second", name)
	}
	if agent.Command != "second" {
		t.Errorf("GetDefaultAgent() command = %s, want second", agent.Command)
	}
}

func TestPlansDir(t *testing.T) {
	cfg := &Config{
		WorkDir: "/project",
	}

	result := cfg.PlansDir()
	expected := "/project/plans"
	if result != expected {
		t.Errorf("PlansDir() = %s, want %s", result, expected)
	}
}

func TestStateDBPath(t *testing.T) {
	cfg := &Config{
		WorkDir: "/project",
	}
	result := cfg.StateDBPath()
	expected := "/project/.planck/state.db"
	if result != expected {
		t.Errorf("StateDBPath() = %s, want %s", result, expected)
	}
}

func TestLoadSaveConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "planck-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Load should create default config
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Modify and save
	cfg.Preferences.Editor = "custom-editor"
	err = cfg.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ".planck", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load again and verify
	cfg2, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() second time error = %v", err)
	}
	if cfg2.Preferences.Editor != "custom-editor" {
		t.Errorf("Loaded editor = %s, want custom-editor", cfg2.Preferences.Editor)
	}
}
