package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the planck configuration
type Config struct {
	Agents        map[string]AgentConfig `toml:"agents"`
	Preferences   Preferences            `toml:"preferences"`
	Notifications Notifications          `toml:"notifications"`
	Execution     Execution              `toml:"execution"`
	Planning      Planning               `toml:"planning"`
	Session       Session                `toml:"session"`
	MarkdownStyle MarkdownStyle          `toml:"markdown_style"`

	// Runtime fields (not from config file)
	WorkDir    string `toml:"-"`
	ConfigPath string `toml:"-"`
}

// AgentConfig configures an AI agent
type AgentConfig struct {
	Command            string   `toml:"command"`
	Label              string   `toml:"label"`
	PlanningArgs       []string `toml:"planning_args"`
	ImplementationArgs []string `toml:"implementation_args"`
	Default            bool     `toml:"default"`
}

// Preferences holds user preferences
type Preferences struct {
	Editor       string `toml:"editor"`
	TmuxPrefix   string `toml:"tmux_prefix"`
	SpinnerStyle string `toml:"spinner_style"`
	SidebarWidth int    `toml:"sidebar_width"`
}

// MarkdownStyle configures the markdown rendering style
type MarkdownStyle struct {
	Theme     string            `toml:"theme"`     // global theme name
	Overrides map[string]string `toml:"overrides"` // element → theme name
}

// Notifications configures the notification system
type Notifications struct {
	Bell bool `toml:"bell"`
}

// Execution configures autonomous execution
type Execution struct {
	DefaultScope   string `toml:"default_scope"`
	AutoAdvance    bool   `toml:"auto_advance"`
	PermissionMode string `toml:"permission_mode"`
}

// Planning configures planning sessions
type Planning struct {
	Model             string `toml:"model"`
	DefaultApproaches int    `toml:"default_approaches"`
}

// Session configures session backend
type Session struct {
	// Backend specifies the session backend to use: "tmux", "pty", or "auto"
	// "auto" will prefer PTY if available, falling back to tmux
	Backend string `toml:"backend"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Agents: map[string]AgentConfig{
			"claude-code": {
				Command:            "claude",
				Label:              "Claude",
				PlanningArgs:       []string{"-p", "--output-format", "stream-json", "--verbose", "--dangerously-skip-permissions"},
				ImplementationArgs: []string{},
				Default:            true,
			},
			"codex": {
				Command:      "codex",
				Label:        "Codex",
				PlanningArgs: []string{"--full-auto"},
				Default:      false,
			},
		},
		Preferences: Preferences{
			Editor:       "",
			TmuxPrefix:   "planck",
			SpinnerStyle: "claude",
			SidebarWidth: 28,
		},
		Notifications: Notifications{
			Bell: true,
		},
		Execution: Execution{
			DefaultScope:   "phase",
			AutoAdvance:    true,
			PermissionMode: "pre-approve",
		},
		Planning: Planning{
			DefaultApproaches: 3,
		},
		Session: Session{
			Backend: "auto",
		},
		MarkdownStyle: MarkdownStyle{
			Theme:     "neo-brutalist",
			Overrides: map[string]string{},
		},
	}
}

// Load loads configuration from the work directory
func Load(workDir string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.WorkDir = workDir

	configDir := filepath.Join(workDir, ".planck")
	configPath := filepath.Join(configDir, "config.toml")
	cfg.ConfigPath = configPath

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("create default config: %w", err)
		}
		return cfg, nil
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to disk
func (c *Config) Save() error {
	f, err := os.Create(c.ConfigPath)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Agents) == 0 {
		return fmt.Errorf("at least one agent must be configured")
	}

	hasDefault := false
	for name, agent := range c.Agents {
		if agent.Command == "" {
			return fmt.Errorf("agent %q: command is required", name)
		}
		if agent.Default {
			if hasDefault {
				return fmt.Errorf("multiple default agents configured")
			}
			hasDefault = true
		}
	}

	validScopes := map[string]bool{"task": true, "phase": true, "plan": true}
	if !validScopes[c.Execution.DefaultScope] {
		return fmt.Errorf("invalid execution scope: %s", c.Execution.DefaultScope)
	}

	validModes := map[string]bool{"pre-approve": true, "per-phase": true, "verify-at-end": true}
	if !validModes[c.Execution.PermissionMode] {
		return fmt.Errorf("invalid permission mode: %s", c.Execution.PermissionMode)
	}

	validBackends := map[string]bool{"auto": true, "tmux": true, "pty": true}
	if c.Session.Backend != "" && !validBackends[c.Session.Backend] {
		return fmt.Errorf("invalid session backend: %s", c.Session.Backend)
	}

	return nil
}

// GetEditor returns the editor to use
func (c *Config) GetEditor() string {
	if c.Preferences.Editor != "" {
		return c.Preferences.Editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	return "vi"
}

// GetDefaultAgent returns the default agent configuration
func (c *Config) GetDefaultAgent() (string, AgentConfig) {
	for name, agent := range c.Agents {
		if agent.Default {
			return name, agent
		}
	}
	// Return first agent if no default
	for name, agent := range c.Agents {
		return name, agent
	}
	return "", AgentConfig{}
}

// GetAgentLabel returns the display label for an agent, falling back to the config key
func (c *Config) GetAgentLabel(key string) string {
	if agent, ok := c.Agents[key]; ok && agent.Label != "" {
		return agent.Label
	}
	return key
}

// PlansDir returns the plans directory path
func (c *Config) PlansDir() string {
	return filepath.Join(c.WorkDir, "plans")
}

// StateDBPath returns the SQLite database path
func (c *Config) StateDBPath() string {
	return filepath.Join(c.WorkDir, ".planck", "state.db")
}

// SessionsDir returns the sessions directory path
func (c *Config) SessionsDir() string {
	return filepath.Join(c.WorkDir, ".planck", "sessions")
}
