package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/sabizmil/planck/internal/config"
	"github.com/sabizmil/planck/internal/session"
	"github.com/sabizmil/planck/internal/store"
	"github.com/sabizmil/planck/internal/ui"
	"github.com/sabizmil/planck/internal/workspace"
)

// Focus represents which panel is focused
type Focus int

const (
	FocusFileList Focus = iota
	FocusEditor
	FocusAgent
)

const maxAgentTabs = 8

const refreshInterval = 5 * time.Second

// AgentTab represents a single agent tab with its own PTY session
type AgentTab struct {
	id          string // unique tab ID
	agentKey    string // config key (e.g., "claude-code")
	baseLabel   string // display label from config (e.g., "Claude")
	customTitle string // title set by child process via OSC escape sequences
	instanceNum int    // per-type instance number (1, 2, ...)
	session     *session.Session
	panel       *ui.PTYPanel
	agentCfg    config.AgentConfig

	// Idle detection: track when PTY content last changed
	lastContentChange time.Time
	prevContent       string
	lastUserWrite     time.Time // when user last typed into the PTY

	// Input buffer: accumulates user keystrokes to derive a tab title
	// when the child process doesn't send one via OSC.
	inputBuf       []rune
	titleFromInput bool // true if customTitle was set from user input (not OSC)

	// Hook-based state detection (Claude Code agents)
	stateFile string // path to hook state file (empty if not a Claude agent)
}

// App is the main application model
type App struct {
	// Configuration
	config    *config.Config
	configDir string
	folder    string

	// State management
	store          *store.Store
	workspace      *workspace.Workspace
	sessionBackend session.InteractiveBackend

	// UI components
	theme    *ui.Theme
	keymap   *ui.Keymap
	tabs     *ui.TabBar
	fileList *ui.FileList
	editor   *ui.Editor
	dialog   *ui.Dialog
	help     *ui.Help
	settings *ui.Settings

	// Style state
	styleRegistry *ui.StyleRegistry

	// Multi-agent tabs
	agentTabs       []*AgentTab    // ordered list of agent tabs
	activeTabIdx    int            // 0 = planning, 1+ = agent tabs
	nextInstanceNum map[string]int // per-agent-type instance counter

	// Deferred agent launch (set by dialog callback, consumed by Update)
	pendingAgentKey string

	// Spinner tick needs to start (set by syncTabBar, consumed by Update)
	pendingSpinnerStart bool

	// Current state
	focus         Focus
	width, height int
	sidebarWidth  int
	message       string
	quitting      bool

	// Double-tap Ctrl+C quit protection
	ctrlCPressedAt time.Time

	// File watching
	watchChan       <-chan struct{}
	lastRefreshTime time.Time

	// Auto-preview tracking
	prevCursor int

	// Sidebar drag-resize state
	draggingSidebar bool

	// Build version (e.g. "v1.2.3" or "dev" for local builds)
	version string
}

// New creates a new application
func New(cfg *config.Config, configDir, folder string, backend session.InteractiveBackend, version string) (*App, error) {
	// Open store
	st, err := store.Open(cfg.StateDBPath())
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	// Create workspace
	ws, err := workspace.New(folder, cfg.Preferences.ExcludeDirs)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	// Create keymap and apply user overrides
	keymap := ui.DefaultKeymap()
	if cfg.Keybindings != nil {
		keymap.ApplyOverrides(cfg.Keybindings)
	}

	// Create theme from preset (falls back to default)
	theme := ui.ThemeFromPreset(cfg.Preferences.ThemePreset)

	// Build style registry and load persisted config
	registry := ui.NewStyleRegistry()
	styleCfg := loadMarkdownStyleFromConfig(cfg)
	styleJSON := registry.ComposeStyle(styleCfg)

	editor := ui.NewEditor(theme, keymap)
	editor.SetMarkdownStyle(styleJSON)

	// Build settings configs from config
	generalCfg := ui.GeneralSettingsChangedMsg{
		Editor:         cfg.Preferences.Editor,
		Bell:           cfg.Notifications.Bell,
		SidebarWidth:   cfg.Preferences.SidebarWidth,
		ExcludeDirs:    cfg.Preferences.ExcludeDirs,
		Backend:        cfg.Session.Backend,
		DefaultScope:   cfg.Execution.DefaultScope,
		AutoAdvance:    cfg.Execution.AutoAdvance,
		PermissionMode: cfg.Execution.PermissionMode,
	}
	agentsCfg := make(map[string]ui.AgentSettingsConfig, len(cfg.Agents))
	for k, a := range cfg.Agents {
		agentsCfg[k] = ui.AgentSettingsConfig{
			Command:      a.Command,
			Label:        a.Label,
			PlanningArgs: a.PlanningArgs,
			Default:      a.Default,
		}
	}

	sidebarWidth := cfg.Preferences.SidebarWidth
	if sidebarWidth < 16 {
		sidebarWidth = 16
	}
	if sidebarWidth > 60 {
		sidebarWidth = 60
	}

	app := &App{
		config:          cfg,
		configDir:       configDir,
		folder:          folder,
		store:           st,
		workspace:       ws,
		sessionBackend:  backend,
		theme:           theme,
		keymap:          keymap,
		tabs:            ui.NewTabBar(theme),
		fileList:        ui.NewFileList(theme),
		editor:          editor,
		dialog:          ui.NewDialog(theme),
		help:            ui.NewHelp(theme, keymap),
		settings:        ui.NewSettings(theme, keymap, registry, styleCfg, generalCfg, agentsCfg, cfg.Preferences.SpinnerStyle, cfg.Preferences.ThemePreset),
		styleRegistry:   registry,
		activeTabIdx:    0,
		focus:           FocusFileList,
		sidebarWidth:    sidebarWidth,
		nextInstanceNum: make(map[string]int),
		version:         version,
	}

	// Apply configured spinner style to tab bar
	app.tabs.SetSpinner(ui.SpinnerPresetByName(cfg.Preferences.SpinnerStyle))

	// Set folder path in tab bar
	app.tabs.SetFolderPath(folder)

	return app, nil
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	// Load files into list
	a.refreshFiles()

	// Start watching for file changes
	watchChan, err := a.workspace.Watch()
	if err == nil {
		a.watchChan = watchChan
	}

	// Select first file if available
	files := a.workspace.Files()
	if len(files) > 0 {
		a.loadFile(files[0].Name)
		a.prevCursor = a.fileList.Cursor()
	}

	a.fileList.SetFocused(true)

	// Clean up old completed/failed sessions
	a.cleanupOldSessions()

	// Recover sessions from previous run (tmux backend only)
	recoverCmd := a.recoverSessions()

	return tea.Batch(a.watchForChanges(), a.refreshTick(), recoverCmd)
}

// watchForChanges creates a command that watches for file changes
func (a *App) watchForChanges() tea.Cmd {
	if a.watchChan == nil {
		return nil
	}

	return func() tea.Msg {
		<-a.watchChan
		return fileChangedMsg{}
	}
}

type fileChangedMsg struct{}
type refreshTickMsg struct{}
type ctrlCTimeoutMsg struct{}

// refreshTick returns a tea.Cmd that fires a refreshTickMsg after refreshInterval
func (a *App) refreshTick() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

// isOnPlanningTab returns true if the planning tab is active
func (a *App) isOnPlanningTab() bool {
	return a.activeTabIdx == 0
}

// isOnAgentTab returns true if an agent tab is active
func (a *App) isOnAgentTab() bool {
	return a.activeTabIdx > 0
}

// activeAgentTab returns the currently active agent tab, or nil
func (a *App) activeAgentTab() *AgentTab {
	idx := a.activeTabIdx - 1
	if idx < 0 || idx >= len(a.agentTabs) {
		return nil
	}
	return a.agentTabs[idx]
}

// activePanel returns the PTY panel of the active agent tab, or nil
func (a *App) activePanel() *ui.PTYPanel {
	if tab := a.activeAgentTab(); tab != nil {
		return tab.panel
	}
	return nil
}

// isInPTYInputMode returns true if the active agent panel is in input mode
func (a *App) isInPTYInputMode() bool {
	if panel := a.activePanel(); panel != nil {
		return panel.IsInputMode()
	}
	return false
}

// findAgentTabBySessionID finds the agent tab that owns a given session ID
func (a *App) findAgentTabBySessionID(sessionID string) *AgentTab {
	for _, tab := range a.agentTabs {
		if tab.session != nil && tab.session.ID == sessionID {
			return tab
		}
	}
	return nil
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle window size
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		a.width = msg.Width
		a.height = msg.Height
		a.updateSizes()
		return a, nil
	}

	// Handle quit: double-tap Ctrl+C to exit
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "ctrl+c" {
			if !a.ctrlCPressedAt.IsZero() && time.Since(a.ctrlCPressedAt) < 2*time.Second {
				// Second press within window — quit
				a.killAllSessions()
				a.quitting = true
				return a, tea.Quit
			}
			// First press — start timeout window
			a.ctrlCPressedAt = time.Now()
			return a, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return ctrlCTimeoutMsg{}
			})
		}
	}

	// Handle Ctrl+C timeout: clear warning from status bar
	if _, ok := msg.(ctrlCTimeoutMsg); ok {
		a.ctrlCPressedAt = time.Time{}
		return a, nil
	}

	// Handle file changes
	if _, ok := msg.(fileChangedMsg); ok {
		a.refreshFiles()
		cmds = append(cmds, a.watchForChanges())
		return a, tea.Batch(cmds...)
	}

	// Handle periodic refresh tick
	if _, ok := msg.(refreshTickMsg); ok {
		// Always re-chain the next tick
		cmds = append(cmds, a.refreshTick())
		// Debounce: skip if refreshed recently (e.g. fsnotify already fired)
		if time.Since(a.lastRefreshTime) >= 2*time.Second {
			a.refreshFiles()
		}
		return a, tea.Batch(cmds...)
	}

	// Handle file saved
	if msg, ok := msg.(ui.FileSavedMsg); ok {
		if err := a.workspace.SaveFile(msg.FileName, msg.Content); err != nil {
			a.message = fmt.Sprintf("Error saving: %v", err)
		} else {
			a.message = "File saved"
			a.editor.ClearModified()
		}
		return a, nil
	}

	// Handle markdown style changes from settings panel
	if msg, ok := msg.(ui.MarkdownStyleChangedMsg); ok {
		a.editor.SetMarkdownStyle(msg.StyleJSON)
		a.config.MarkdownStyle.Theme = string(msg.Config.GlobalTheme)
		a.config.MarkdownStyle.Overrides = make(map[string]string)
		for elem, theme := range msg.Config.Overrides {
			a.config.MarkdownStyle.Overrides[string(elem)] = string(theme)
		}
		_ = a.config.Save()
		return a, nil
	}

	// Handle general settings changes from settings panel
	if msg, ok := msg.(ui.GeneralSettingsChangedMsg); ok {
		a.config.Preferences.Editor = msg.Editor
		a.config.Notifications.Bell = msg.Bell
		a.config.Session.Backend = msg.Backend
		a.config.Execution.DefaultScope = msg.DefaultScope
		a.config.Execution.AutoAdvance = msg.AutoAdvance
		a.config.Execution.PermissionMode = msg.PermissionMode
		if msg.SidebarWidth >= 16 && msg.SidebarWidth <= 60 {
			a.config.Preferences.SidebarWidth = msg.SidebarWidth
			a.sidebarWidth = msg.SidebarWidth
			a.updateSizes()
		}
		if msg.ExcludeDirs != nil {
			a.config.Preferences.ExcludeDirs = msg.ExcludeDirs
			a.workspace.SetExcludeDirs(msg.ExcludeDirs)
			a.refreshFiles()
		}
		_ = a.config.Save()
		return a, nil
	}

	// Handle agent settings changes from settings panel
	if msg, ok := msg.(ui.AgentsSettingsChangedMsg); ok {
		for key, agentUI := range msg.Agents {
			agent := a.config.Agents[key]
			agent.Command = agentUI.Command
			agent.Label = agentUI.Label
			agent.PlanningArgs = agentUI.PlanningArgs
			agent.Default = agentUI.Default
			a.config.Agents[key] = agent
		}
		_ = a.config.Save()
		// Update base labels on existing tabs so syncTabBar picks them up
		for _, tab := range a.agentTabs {
			tab.baseLabel = a.config.GetAgentLabel(tab.agentKey)
		}
		a.syncTabBar()
		return a, nil
	}

	// Handle spinner settings changes from settings panel
	if msg, ok := msg.(ui.SpinnerSettingsChangedMsg); ok {
		a.config.Preferences.SpinnerStyle = msg.Style
		_ = a.config.Save()
		a.tabs.SetSpinner(ui.SpinnerPresetByName(msg.Style))
		// Restart spinner tick if tabs are running so new interval takes effect
		if a.tabs.HasRunningTabs() {
			cmds = append(cmds, a.tabs.Tick())
		}
		return a, tea.Batch(cmds...)
	}

	// Handle theme changes from settings panel
	if msg, ok := msg.(ui.ThemeChangedMsg); ok {
		a.theme = msg.Theme
		a.config.Preferences.ThemePreset = msg.PresetName
		_ = a.config.Save()
		return a, nil
	}

	// Handle keybinding changes from settings panel
	if msg, ok := msg.(ui.KeybindingsChangedMsg); ok {
		a.keymap = msg.Keymap
		// Persist overrides to config
		a.config.Keybindings = a.keymapOverrides()
		_ = a.config.Save()
		return a, nil
	}

	// Handle PTY messages BEFORE overlay checks — polling must never be
	// interrupted by dialogs, help, or folder picker, otherwise the polling
	// chain breaks and the tab goes permanently silent.
	if msg, ok := msg.(ui.PTYWriteMsg); ok {
		if tab := a.findAgentTabBySessionID(msg.SessionID); tab != nil {
			_ = a.sessionBackend.Write(tab.session.BackendHandle, msg.Data)
			tab.lastUserWrite = time.Now()

			// Accumulate user keystrokes to derive a tab title.
			a.trackInputForTitle(tab, msg.Data)

			// Clear needs_input on user interaction (they responded to the prompt)
			if tab.panel.GetStatus() == "needs_input" {
				tab.panel.SetStatus("idle")
				a.clearHookState(tab)
				a.syncTabBar()
			}
		}
	}

	if msg, ok := msg.(ui.PTYRenderMsg); ok {
		if tab := a.findAgentTabBySessionID(msg.SessionID); tab != nil {
			// Idle detection: track content changes
			contentChanged := msg.Content != tab.prevContent
			tab.prevContent = msg.Content

			// Don't attribute content changes to the agent if the user just typed.
			// Terminal echo from keystrokes causes content changes that aren't agent output.
			userRecentlyTyped := !tab.lastUserWrite.IsZero() && time.Since(tab.lastUserWrite) < 1*time.Second

			if contentChanged {
				tab.lastContentChange = time.Now()
				// Resume from idle/needs_input only if agent is producing output
				// (not just the terminal echoing the user's keystrokes)
				if !userRecentlyTyped {
					if status := tab.panel.GetStatus(); status == "idle" || status == "needs_input" {
						tab.panel.SetStatus("running")
						a.clearHookState(tab)
						a.syncTabBar()
					}
				}
			} else if tab.panel.GetStatus() == "running" && !userRecentlyTyped {
				// Content unchanged — check for state transitions

				// Hook state is checked immediately (no timeout) so permission
				// prompts get a red dot within one poll cycle (~50ms).
				if state := a.readHookState(tab); state == "needs_input" {
					tab.panel.SetStatus("needs_input")
					a.syncTabBar()
				} else if !tab.lastContentChange.IsZero() &&
					time.Since(tab.lastContentChange) > 3*time.Second {
					// No output for 3s and no hook state — agent is idle
					tab.panel.SetStatus("idle")
					a.syncTabBar()
				}
			}

			tab.panel.SetContent(msg.Content)
			// Update tab title if the child process set a meaningful one via OSC.
			// Skip generic program names (e.g. "Claude Code") when we already
			// have a descriptive title derived from user input.
			if title := sanitizeTabTitle(msg.Title); title != "" && title != tab.customTitle {
				if tab.titleFromInput && isGenericOSCTitle(title, tab.baseLabel) {
					// Don't overwrite a user-input title with a generic program name.
				} else {
					tab.customTitle = title
					tab.titleFromInput = false // OSC-derived, not from input
					a.syncTabBar()
					// Persist title for recovery
					if tab.session != nil {
						_ = a.store.UpdateSessionTitle(tab.session.ID, title)
					}
				}
			}
			cmds = append(cmds, a.pollAgentTab(tab))
		}
	}

	if msg, ok := msg.(ui.PTYExitedMsg); ok {
		if tab := a.findAgentTabBySessionID(msg.SessionID); tab != nil {
			tab.panel.SetStatus("completed")
			a.cleanupHookState(tab)
			a.syncTabBar()
			a.markSessionCompleted(msg.SessionID, msg.ExitCode)
			a.message = fmt.Sprintf("%s completed (exit code: %d)", tab.baseLabel, msg.ExitCode)
		}
	}

	// Handle move mode messages from FileList
	if msg, ok := msg.(ui.MoveConfirmedMsg); ok {
		a.handleMoveConfirmed(msg)
		a.fileList.ExitMoveMode()
		return a, tea.Batch(cmds...)
	}
	if _, ok := msg.(ui.MoveCanceledMsg); ok {
		a.fileList.ExitMoveMode()
		a.message = "Move canceled"
		return a, tea.Batch(cmds...)
	}

	// Handle spinner tick — forward to tab bar and settings (for live preview)
	if _, ok := msg.(ui.SpinnerTickMsg); ok {
		_, cmd := a.tabs.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if a.settings.IsVisible() {
			a.settings.Update(msg)
			// Keep tick running for settings preview even without running tabs
			if cmd == nil {
				cmds = append(cmds, a.tabs.Tick())
			}
		}
		return a, tea.Batch(cmds...)
	}

	// Dialog takes priority
	if a.dialog.IsVisible() {
		dialog, cmd := a.dialog.Update(msg)
		a.dialog = dialog
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// If dialog just closed and set a pending agent, launch it now
		if !a.dialog.IsVisible() && a.pendingAgentKey != "" {
			key := a.pendingAgentKey
			a.pendingAgentKey = ""
			if cmd := a.launchAgentTab(key); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return a, tea.Batch(cmds...)
	}

	// Launch pending agent (safety net for non-dialog code paths)
	if a.pendingAgentKey != "" {
		key := a.pendingAgentKey
		a.pendingAgentKey = ""
		if cmd := a.launchAgentTab(key); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Start spinner tick if syncTabBar detected new running tabs
	if a.pendingSpinnerStart {
		a.pendingSpinnerStart = false
		cmds = append(cmds, a.tabs.Tick())
	}

	// Help takes priority
	if a.help.IsVisible() {
		help, cmd := a.help.Update(msg)
		a.help = help
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)
	}

	// Settings takes priority
	if a.settings.IsVisible() {
		settings, cmd := a.settings.Update(msg)
		a.settings = settings
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)
	}

	// Handle mouse events
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		me := tea.MouseEvent(mouseMsg)
		// Tab bar click: left click on row 0 switches tabs (works on all tabs)
		if mouseMsg.Y == 0 && mouseMsg.Button == tea.MouseButtonLeft && me.Action == tea.MouseActionPress {
			if idx := a.tabs.HitTest(mouseMsg.X); idx >= 0 {
				cmd := a.switchToTab(idx)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return a, tea.Batch(cmds...)
			}
		}
		// Planning tab mouse events (sidebar drag, editor clicks)
		if a.isOnPlanningTab() || a.draggingSidebar {
			a.handlePlanningMouse(mouseMsg)
			return a, tea.Batch(cmds...)
		}
	}

	// Handle key messages
	if msg, ok := msg.(tea.KeyMsg); ok {
		cmd := a.handleKeypress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update focused component based on active tab
	if a.isOnPlanningTab() {
		switch a.focus {
		case FocusFileList:
			fileList, cmd := a.fileList.Update(msg)
			a.fileList = fileList
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Auto-preview: load file when cursor moves (only in view mode, not in move mode)
			if a.fileList.Cursor() != a.prevCursor && a.editor.Mode() == ui.EditorModeView && !a.fileList.InMoveMode() {
				a.prevCursor = a.fileList.Cursor()
				if file := a.fileList.SelectedFile(); file != nil {
					a.loadFile(file.Name)
				}
			}

		case FocusEditor:
			editor, cmd := a.editor.Update(msg)
			a.editor = editor
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Auto-switch focus back to file list when editor exits edit mode
			if a.editor.Mode() == ui.EditorModeView {
				a.focus = FocusFileList
				a.fileList.SetFocused(true)
				a.editor.SetFocused(false)
			}
		}
	} else if panel := a.activePanel(); panel != nil {
		// Agent tab: update the active panel
		if panel.IsVisible() {
			updated, cmd := panel.Update(msg)
			// PTYPanel.Update returns *PTYPanel, reassign
			if tab := a.activeAgentTab(); tab != nil {
				tab.panel = updated
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) handleKeypress(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	km := a.keymap
	inEdit := a.editor.Mode() == ui.EditorModeEdit
	inPTY := a.isInPTYInputMode()

	// Global keys that work even in input mode
	if km.Matches(ui.ContextGlobal, ui.ActionNextTab, key) {
		return a.cycleTab(1)
	}
	if km.Matches(ui.ContextGlobal, ui.ActionCloseTab, key) {
		if a.isOnAgentTab() {
			if panel := a.activePanel(); panel != nil {
				panel.ExitInputMode()
			}
			return a.closeCurrentAgentTab()
		}
		return nil
	}

	// Alt+1-9 for tab switching — works in all modes (including input mode)
	if key >= "alt+1" && key <= "alt+9" {
		idx := int(key[len(key)-1]-'0') - 1
		return a.switchToTab(idx)
	}

	// Keys that are suppressed in edit mode or PTY input mode
	if inEdit || inPTY {
		// Number keys 1-9 for tab switching (normal mode only)
		if key >= "1" && key <= "9" {
			return nil
		}
		// Tab-specific keys
		if a.isOnPlanningTab() {
			return a.handlePlanningTabKey(key, msg)
		}
		return nil
	}

	// Normal mode global keys
	if km.Matches(ui.ContextGlobal, ui.ActionToggleHelp, key) {
		a.help.Toggle()
		return nil
	}
	if km.Matches(ui.ContextGlobal, ui.ActionQuit, key) {
		a.killAllSessions()
		a.quitting = true
		return tea.Quit
	}
	if km.Matches(ui.ContextGlobal, ui.ActionSettings, key) {
		a.settings.Toggle()
		if a.settings.IsVisible() && !a.tabs.HasRunningTabs() {
			return a.tabs.Tick()
		}
		return nil
	}
	if km.Matches(ui.ContextGlobal, ui.ActionCreateAgent, key) {
		return a.createAgentTab()
	}
	if km.Matches(ui.ContextGlobal, ui.ActionCloseAgent, key) {
		return a.closeCurrentAgentTab()
	}

	// Number keys 1-9 for tab switching (normal mode only)
	if key >= "1" && key <= "9" {
		idx := int(key[0]-'0') - 1
		return a.switchToTab(idx)
	}

	// Tab-specific keys
	if a.isOnPlanningTab() {
		return a.handlePlanningTabKey(key, msg)
	}

	return nil
}

// cycleTab cycles to the next (+1) or previous (-1) tab
func (a *App) cycleTab(direction int) tea.Cmd {
	totalTabs := 1 + len(a.agentTabs) // planning + agents
	if totalTabs <= 1 {
		return nil // Only planning tab, nowhere to cycle
	}

	// Exit input mode on current agent tab
	if panel := a.activePanel(); panel != nil {
		panel.ExitInputMode()
	}

	next := (a.activeTabIdx + direction + totalTabs) % totalTabs
	return a.switchToTab(next)
}

// switchToTab switches to a tab by index (0 = planning, 1+ = agent tabs)
func (a *App) switchToTab(idx int) tea.Cmd {
	totalTabs := 1 + len(a.agentTabs)
	if idx < 0 || idx >= totalTabs {
		return nil
	}

	// Exit input mode on current agent tab
	if panel := a.activePanel(); panel != nil {
		panel.ExitInputMode()
	}

	a.activeTabIdx = idx
	a.tabs.SetActiveIdx(idx)

	if a.isOnPlanningTab() {
		a.focus = FocusFileList
		a.fileList.SetFocused(true)
		a.editor.SetFocused(false)
	} else {
		a.focus = FocusAgent
		a.fileList.SetFocused(false)
		a.editor.SetFocused(false)
		// Auto-enter input mode if session is running
		if tab := a.activeAgentTab(); tab != nil && tab.session != nil {
			status, _ := a.sessionBackend.Status(tab.session.BackendHandle)
			if status == session.StatusRunning {
				tab.panel.EnterInputMode()
			}
		}
	}

	return nil
}

// createAgentTab handles the "a" key to create a new agent tab
func (a *App) createAgentTab() tea.Cmd {
	if len(a.agentTabs) >= maxAgentTabs {
		a.message = "Maximum agent tabs reached (8)"
		return nil
	}

	// Get sorted agent keys for consistent ordering
	agentKeys := a.sortedAgentKeys()

	if len(agentKeys) == 0 {
		a.message = "No agents configured"
		return nil
	}

	if len(agentKeys) == 1 {
		// Single agent type → create immediately
		return a.launchAgentTab(agentKeys[0])
	}

	// Multiple agent types → show selection dialog
	var options []ui.DialogOption
	for _, key := range agentKeys {
		agent := a.config.Agents[key]
		label := a.config.GetAgentLabel(key)
		options = append(options, ui.DialogOption{
			Label:       label,
			Description: agent.Command,
		})
	}

	a.dialog.ShowSelect("New Agent", options, func(result ui.DialogResult) {
		if result.Confirmed && result.Selected >= 0 && result.Selected < len(agentKeys) {
			a.pendingAgentKey = agentKeys[result.Selected]
		}
	})

	return nil
}

// launchAgentTab creates and launches a new agent tab for the given config key
func (a *App) launchAgentTab(agentKey string) tea.Cmd {
	agentCfg, ok := a.config.Agents[agentKey]
	if !ok {
		a.message = fmt.Sprintf("Agent %q not found in config", agentKey)
		return nil
	}

	// Check if command is available
	if !session.IsCommandAvailable(agentCfg.Command) {
		a.message = fmt.Sprintf("%s not found in PATH", agentCfg.Command)
		return nil
	}

	// Assign instance number
	a.nextInstanceNum[agentKey]++
	instanceNum := a.nextInstanceNum[agentKey]

	baseLabel := a.config.GetAgentLabel(agentKey)

	// Create PTY panel
	panel := ui.NewPTYPanel(a.theme, a.keymap)
	contentHeight := a.height - 4
	panel.SetSize(a.width, contentHeight)

	tab := &AgentTab{
		id:          uuid.New().String(),
		agentKey:    agentKey,
		baseLabel:   baseLabel,
		instanceNum: instanceNum,
		panel:       panel,
		agentCfg:    agentCfg,
	}

	// Add to tabs
	a.agentTabs = append(a.agentTabs, tab)
	a.syncTabBar()

	// Switch to the new tab
	newIdx := len(a.agentTabs) // 1-indexed (planning is 0)
	a.activeTabIdx = newIdx
	a.tabs.SetActiveIdx(newIdx)
	a.focus = FocusAgent
	a.fileList.SetFocused(false)
	a.editor.SetFocused(false)

	// Build args for launch — filter interactive-safe args from PlanningArgs
	var launchArgs []string
	hasSkipPerms := false
	for _, arg := range agentCfg.PlanningArgs {
		switch arg {
		case "--dangerously-skip-permissions", "--full-auto":
			launchArgs = append(launchArgs, arg)
			hasSkipPerms = true
		}
	}

	// For Claude Code agents: inject hooks to detect permission prompts.
	// Only useful when permissions are NOT skipped (otherwise there are no prompts).
	if agentCfg.Command == "claude" && !hasSkipPerms {
		stateFile := filepath.Join(os.TempDir(), fmt.Sprintf("planck-%s-state", tab.id))
		tab.stateFile = stateFile
		launchArgs = append(launchArgs, "--settings", hookSettingsJSON(stateFile))
	}

	// Launch the session
	ctx := context.Background()
	sess, err := a.sessionBackend.LaunchCommand(ctx, a.folder, agentCfg.Command, launchArgs, "")
	if err != nil {
		a.message = fmt.Sprintf("Error launching %s: %v", baseLabel, err)
		// Remove the tab we just added
		a.agentTabs = a.agentTabs[:len(a.agentTabs)-1]
		a.syncTabBar()
		a.activeTabIdx = 0
		a.tabs.SetActiveIdx(0)
		a.focus = FocusFileList
		a.fileList.SetFocused(true)
		return nil
	}

	tab.session = sess

	// Resize the PTY to match panel dimensions
	rows, cols := tab.panel.TerminalSize()
	_ = a.sessionBackend.Resize(sess.BackendHandle, uint16(rows), uint16(cols))

	// Wire scrollback buffer from backend to panel
	if sb := a.sessionBackend.GetScrollback(sess.BackendHandle); sb != nil {
		tab.panel.SetScrollback(sb)
	}

	// Persist session to SQLite for recovery
	a.persistSession(tab)

	// Show PTY panel and enter input mode
	tab.panel.Show(agentKey, baseLabel, sess.ID)
	tab.panel.EnterInputMode()

	a.message = fmt.Sprintf("%s session started", a.computeTabLabel(tab))

	// Start polling
	return a.pollAgentTab(tab)
}

// closeCurrentAgentTab handles the "x" key to close the current agent tab
func (a *App) closeCurrentAgentTab() tea.Cmd {
	if a.isOnPlanningTab() {
		return nil // Can't close planning tab
	}

	tab := a.activeAgentTab()
	if tab == nil {
		return nil
	}

	// Check if session is still running
	isRunning := false
	if tab.session != nil {
		status, _ := a.sessionBackend.Status(tab.session.BackendHandle)
		isRunning = status == session.StatusRunning
	}

	if isRunning {
		// Show confirmation dialog
		a.dialog.ShowConfirm(
			"Close Agent?",
			fmt.Sprintf("Kill the running session and close %s?", a.computeTabLabel(tab)),
			func(result ui.DialogResult) {
				if result.Confirmed {
					a.removeAgentTab(tab)
				}
			},
		)
		return nil
	}

	// Session completed or no session — close immediately
	a.removeAgentTab(tab)
	return nil
}

// removeAgentTab kills the session and removes the tab
func (a *App) removeAgentTab(tab *AgentTab) {
	// Kill session if running
	if tab.session != nil {
		_ = a.sessionBackend.Kill(tab.session.BackendHandle)
		a.markSessionCanceled(tab.session.ID)
	}

	// Clean up hook state files
	a.cleanupHookState(tab)

	// Find and remove from slice
	for i, t := range a.agentTabs {
		if t.id == tab.id {
			a.agentTabs = append(a.agentTabs[:i], a.agentTabs[i+1:]...)
			break
		}
	}

	a.syncTabBar()

	// Move focus to previous tab or planning
	if a.activeTabIdx > len(a.agentTabs) {
		a.activeTabIdx = len(a.agentTabs) // Last agent tab, or 0 if none
	}
	if a.activeTabIdx == 0 {
		a.focus = FocusFileList
		a.fileList.SetFocused(true)
	} else {
		a.focus = FocusAgent
	}
	a.tabs.SetActiveIdx(a.activeTabIdx)
}

// maxTabTitleLen is the maximum rune length for tab titles
const maxTabTitleLen = 30

// computeTabLabel returns the display label for an agent tab.
// It prefers the custom title set by the child process via OSC escape
// sequences, falling back to the base label with instance number.
func (a *App) computeTabLabel(tab *AgentTab) string {
	if tab.customTitle != "" {
		title := tab.customTitle
		if utf8.RuneCountInString(title) > maxTabTitleLen {
			runes := []rune(title)
			title = string(runes[:maxTabTitleLen-1]) + "..."
		}
		return title
	}
	count := 0
	for _, t := range a.agentTabs {
		if t.agentKey == tab.agentKey {
			count++
		}
	}
	if count > 1 {
		return fmt.Sprintf("%s #%d", tab.baseLabel, tab.instanceNum)
	}
	return tab.baseLabel
}

// trackInputForTitle accumulates user keystrokes and, on Enter, derives a
// tab title from the submitted line. This ensures slash commands (which
// don't trigger an OSC title from the child process) still produce a
// meaningful tab label. The OSC title, when it arrives, will overwrite
// this input-derived title naturally.
func (a *App) trackInputForTitle(tab *AgentTab, data []byte) {
	for _, b := range data {
		switch {
		case b == '\r' || b == '\n':
			if len(tab.inputBuf) > 0 {
				line := strings.TrimSpace(string(tab.inputBuf))
				// Strip leading slash so "/plan fix auth" → "plan fix auth"
				line = strings.TrimLeft(line, "/")
				if title := sanitizeTabTitle(line); title != "" {
					if len([]rune(title)) > maxTabTitleLen {
						title = string([]rune(title)[:maxTabTitleLen-1]) + "..."
					}
					tab.customTitle = title
					tab.titleFromInput = true
					a.syncTabBar()
				}
				tab.inputBuf = nil
			}
		case b == 0x7f: // backspace
			if len(tab.inputBuf) > 0 {
				tab.inputBuf = tab.inputBuf[:len(tab.inputBuf)-1]
			}
		case b == 0x03 || b == 0x1b: // Ctrl+C or Escape — discard
			tab.inputBuf = nil
		case b == 0x15: // Ctrl+U — clear line
			tab.inputBuf = nil
		case b == 0x17: // Ctrl+W — delete word
			// Trim trailing spaces, then trim to last space
			for len(tab.inputBuf) > 0 && tab.inputBuf[len(tab.inputBuf)-1] == ' ' {
				tab.inputBuf = tab.inputBuf[:len(tab.inputBuf)-1]
			}
			for len(tab.inputBuf) > 0 && tab.inputBuf[len(tab.inputBuf)-1] != ' ' {
				tab.inputBuf = tab.inputBuf[:len(tab.inputBuf)-1]
			}
		case b >= 0x20 && b < 0x7f: // printable ASCII
			tab.inputBuf = append(tab.inputBuf, rune(b))
		}
	}
}

// minTabTitleLen is the minimum rune length for a title to be considered useful.
// Shorter values (e.g. single characters, emoji, program names like "~") are
// treated as resets and ignored.
const minTabTitleLen = 3

// sanitizeTabTitle cleans an OSC-provided title, stripping non-printable and
// zero-width characters and returning "" for values that aren't useful as tab
// labels.
func sanitizeTabTitle(raw string) string {
	if raw == "" {
		return ""
	}

	// Strip non-printable, control, zero-width / format characters, braille
	// pattern characters (U+2800–U+28FF), and dingbats (U+2700–U+27BF).
	// Both braille dots and dingbat asterisks (✢ ✳ ✶ ✻ ✽) leak from Claude
	// Code's OSC window title as spinner frames; we strip them so the spinner
	// is solely Planck's.
	var sb strings.Builder
	for _, r := range raw {
		if r >= 0x2700 && r <= 0x28FF {
			continue
		}
		if unicode.IsPrint(r) && !unicode.Is(unicode.Cf, r) {
			sb.WriteRune(r)
		}
	}
	title := strings.TrimSpace(sb.String())

	// Reject if too short to be useful
	if utf8.RuneCountInString(title) < minTabTitleLen {
		return ""
	}

	return title
}

// isGenericOSCTitle reports whether oscTitle is a generic/default program title
// that shouldn't overwrite a more descriptive user-input-derived title.
// It returns true when the OSC title is a case-insensitive match or substring
// of baseLabel (or vice versa), e.g. "Claude Code" vs "Claude Code" or "Claude".
func isGenericOSCTitle(oscTitle, baseLabel string) bool {
	osc := strings.ToLower(strings.TrimSpace(oscTitle))
	base := strings.ToLower(strings.TrimSpace(baseLabel))
	if osc == "" || base == "" {
		return false
	}
	return strings.Contains(osc, base) || strings.Contains(base, osc)
}

// syncTabBar rebuilds the tab bar to match current agent tabs
func (a *App) syncTabBar() {
	hadRunning := a.tabs.HasRunningTabs()

	tabs := []ui.TabInfo{{Label: "Planning"}}
	for _, tab := range a.agentTabs {
		status := ""
		if tab.panel != nil && tab.session != nil {
			s, _ := a.sessionBackend.Status(tab.session.BackendHandle)
			switch s {
			case session.StatusCompleted:
				status = "completed"
			case session.StatusRunning:
				// Respect panel-level status for idle/needs_input states
				switch tab.panel.GetStatus() {
				case "idle":
					status = "idle"
				case "needs_input":
					status = "needs_input"
				default:
					status = "running"
				}
			}
		}
		tabs = append(tabs, ui.TabInfo{
			Label:  a.computeTabLabel(tab),
			Status: status,
		})
	}
	a.tabs.SetTabs(tabs)

	// Start spinner tick if we just gained running tabs
	if !hadRunning && a.tabs.HasRunningTabs() {
		a.pendingSpinnerStart = true
	}
}

// sortedAgentKeys returns agent config keys in deterministic order
func (a *App) sortedAgentKeys() []string {
	keys := make([]string, 0, len(a.config.Agents))
	for k := range a.config.Agents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// Move default agent to front
	for i, k := range keys {
		if a.config.Agents[k].Default {
			keys[0], keys[i] = keys[i], keys[0]
			break
		}
	}
	return keys
}

// pollAgentTab returns a tea.Cmd that polls a specific agent tab's PTY
func (a *App) pollAgentTab(tab *AgentTab) tea.Cmd {
	if tab.session == nil {
		return nil
	}

	handle := tab.session.BackendHandle
	sessionID := tab.session.ID
	backend := a.sessionBackend

	return func() tea.Msg {
		time.Sleep(50 * time.Millisecond)

		// Check if session has exited
		status, err := backend.Status(handle)
		if err != nil || status != session.StatusRunning {
			exitCode, _ := backend.GetExitCode(handle)
			return ui.PTYExitedMsg{SessionID: sessionID, ExitCode: exitCode}
		}

		// Get rendered terminal output
		content, err := backend.Render(handle)
		if err != nil {
			return ui.PTYRenderMsg{SessionID: sessionID, Content: ""}
		}

		// Get window title (set by child process via OSC 0/2 escape sequences)
		title := backend.GetTitle(handle)

		return ui.PTYRenderMsg{SessionID: sessionID, Content: content, Title: title}
	}
}

// killAllSessions kills all running agent sessions.
// For tmux backend, sessions are left alive for recovery on next startup.
// For PTY backend, sessions are killed since they can't survive without planck.
func (a *App) killAllSessions() {
	if a.isTmuxBackend() {
		// Tmux sessions persist independently — just clean up UI state
		for _, tab := range a.agentTabs {
			a.cleanupHookState(tab)
		}
		return
	}

	// PTY backend: kill all sessions
	for _, tab := range a.agentTabs {
		if tab.session != nil {
			_ = a.sessionBackend.Kill(tab.session.BackendHandle)
			a.markSessionCanceled(tab.session.ID)
		}
		a.cleanupHookState(tab)
	}
}

// Close releases resources held by the app (store, file watcher).
// Call after tea.Program.Run() returns.
func (a *App) Close() {
	a.workspace.StopWatch()
	a.store.Close()
}

func (a *App) handlePlanningTabKey(key string, _ tea.KeyMsg) tea.Cmd {
	// Skip if in edit mode - let editor handle
	if a.editor.Mode() == ui.EditorModeEdit {
		return nil
	}

	// In move mode, let FileList handle all keys
	if a.fileList.InMoveMode() {
		return nil
	}

	km := a.keymap

	if a.focus == FocusFileList {
		switch {
		case km.Matches(ui.ContextFileList, ui.ActionOpenFile, key):
			if file := a.fileList.SelectedFile(); file != nil {
				a.loadFile(file.Name)
				a.prevCursor = a.fileList.Cursor()
			}

		case km.Matches(ui.ContextFileList, ui.ActionEditMode, key):
			if a.editor.FileName() != "" {
				a.editor.EnterEditMode()
				a.focus = FocusEditor
				a.editor.SetFocused(true)
				a.fileList.SetFocused(false)
			}

		case km.Matches(ui.ContextFileList, ui.ActionNewFile, key):
			a.dialog.ShowInput("New File", "Enter file name:", func(result ui.DialogResult) {
				if result.Confirmed && result.Input != "" {
					name := result.Input
					if err := a.workspace.CreateFile(name); err != nil {
						a.message = fmt.Sprintf("Error: %v", err)
					} else {
						a.refreshFiles()
						a.fileList.SelectFile(name + ".md")
						a.loadFile(name + ".md")
						a.message = fmt.Sprintf("Created: %s.md", name)
					}
				}
			})

		case km.Matches(ui.ContextFileList, ui.ActionDeleteFile, key):
			if file := a.fileList.SelectedFile(); file != nil {
				a.dialog.ShowConfirm(
					"Delete File?",
					fmt.Sprintf("Delete '%s'?", file.Name),
					func(result ui.DialogResult) {
						if result.Confirmed {
							if err := a.workspace.DeleteFile(file.Name); err != nil {
								a.message = fmt.Sprintf("Error: %v", err)
							} else {
								a.refreshFiles()
								a.editor.SetContent("", "")
								a.message = "File deleted"
							}
						}
					},
				)
			} else if dirPath := a.fileList.SelectedDirPath(); dirPath != "" {
				a.dialog.ShowConfirm(
					"Delete Folder?",
					fmt.Sprintf("Delete '%s' and all files inside?", dirPath),
					func(result ui.DialogResult) {
						if result.Confirmed {
							editorFile := a.editor.FileName()
							if editorFile != "" && strings.HasPrefix(editorFile, dirPath+"/") {
								a.editor.SetContent("", "")
							}
							if err := a.workspace.DeleteFolder(dirPath); err != nil {
								a.message = fmt.Sprintf("Error: %v", err)
							} else {
								a.refreshFiles()
								a.message = "Folder deleted"
							}
						}
					},
				)
			}

		case km.Matches(ui.ContextFileList, ui.ActionExpandFolder, key):
			if a.fileList.IsSelectedDir() {
				a.fileList.ExpandSelected()
			}

		case km.Matches(ui.ContextFileList, ui.ActionCollapseDir, key):
			if a.fileList.IsSelectedDir() {
				a.fileList.CollapseSelected()
			}

		case km.Matches(ui.ContextFileList, ui.ActionToggleComplete, key):
			if file := a.fileList.SelectedFile(); file != nil {
				if err := a.workspace.ToggleFileStatus(file.Name); err != nil {
					a.message = fmt.Sprintf("Error: %v", err)
				} else {
					a.refreshFiles()
					if a.editor.FileName() == file.Name {
						a.loadFile(file.Name)
					}
				}
			}

		case km.Matches(ui.ContextFileList, ui.ActionMoveFile, key):
			if a.fileList.SelectedPath() != "" {
				a.fileList.EnterMoveMode()
			}

		case km.Matches(ui.ContextFileList, ui.ActionRefresh, key):
			a.refreshFiles()
			a.message = "Files refreshed"

		case km.Matches(ui.ContextFileList, ui.ActionExcludeDir, key):
			if dirPath := a.fileList.SelectedDirPath(); dirPath != "" {
				dirName := filepath.Base(dirPath)
				a.dialog.ShowConfirm(
					"Exclude Folder?",
					fmt.Sprintf("Hide '%s' from sidebar?", dirName),
					func(result ui.DialogResult) {
						if result.Confirmed {
							a.config.Preferences.ExcludeDirs = append(a.config.Preferences.ExcludeDirs, dirName)
							a.workspace.SetExcludeDirs(a.config.Preferences.ExcludeDirs)
							a.settings.AddExcludeDir(dirName)
							_ = a.config.Save()
							a.refreshFiles()
							a.message = fmt.Sprintf("Excluded: %s", dirName)
						}
					},
				)
			}
		}
	}

	return nil
}

func (a *App) handleMoveConfirmed(msg ui.MoveConfirmedMsg) {
	var err error
	if msg.IsDir {
		err = a.workspace.MoveFolder(msg.SourcePath, msg.DestDir)
	} else {
		err = a.workspace.MoveFile(msg.SourcePath, msg.DestDir)
	}

	if err != nil {
		a.message = fmt.Sprintf("Move error: %v", err)
		return
	}

	a.refreshFiles()

	// Compute the new relative path of the moved item
	baseName := msg.SourcePath
	if idx := strings.LastIndex(msg.SourcePath, "/"); idx >= 0 {
		baseName = msg.SourcePath[idx+1:]
	}
	var newPath string
	if msg.DestDir == "" {
		newPath = baseName
	} else {
		newPath = msg.DestDir + "/" + baseName
	}

	// Re-select the moved item and update editor if needed
	a.fileList.SelectPath(newPath)
	if !msg.IsDir && a.editor.FileName() == msg.SourcePath {
		a.loadFile(newPath)
	}
	// If we moved a folder containing the open file, update the editor
	if msg.IsDir && a.editor.FileName() != "" && strings.HasPrefix(a.editor.FileName(), msg.SourcePath+"/") {
		newFileName := newPath + a.editor.FileName()[len(msg.SourcePath):]
		a.loadFile(newFileName)
	}

	a.message = fmt.Sprintf("Moved to %s", newPath)
}

func (a *App) handlePlanningMouse(msg tea.MouseMsg) {
	me := tea.MouseEvent(msg)

	borderCol := a.sidebarWidth // the border sits at this column

	// Wheel events: route to file list or editor based on cursor X position.
	// Suppress editor wheel events while a drag selection is in progress —
	// trackpads on macOS commonly intersperse scroll events during a click-drag,
	// and scrolling the viewport mid-selection causes the same screen position
	// to map to different text rows, producing visual jumping.
	if me.IsWheel() {
		if msg.X < borderCol {
			// Mouse is over the file list — scroll the file list
			a.fileList.HandleMouse(msg)
		} else {
			// Mouse is over the editor — scroll the markdown viewer
			a.editor.Update(msg)
		}
		return
	}

	// Start drag: mouse down on or near the sidebar border (±1 col tolerance)
	if msg.Button == tea.MouseButtonLeft && me.Action == tea.MouseActionPress {
		if msg.X >= borderCol-1 && msg.X <= borderCol+1 {
			a.draggingSidebar = true
			return
		}
	}

	// During drag: update sidebar width as mouse moves
	if a.draggingSidebar {
		if me.Action == tea.MouseActionMotion {
			newWidth := msg.X
			if newWidth < 16 {
				newWidth = 16
			}
			if newWidth > 60 {
				newWidth = 60
			}
			// Ensure editor gets at least 40 columns
			if a.width-newWidth < 40 {
				newWidth = a.width - 40
				if newWidth < 16 {
					newWidth = 16
				}
			}
			if newWidth != a.sidebarWidth {
				a.sidebarWidth = newWidth
				a.updateSizes()
			}
			return
		}
		// End drag: mouse release — persist width to config
		if me.Action == tea.MouseActionRelease {
			a.draggingSidebar = false
			a.config.Preferences.SidebarWidth = a.sidebarWidth
			_ = a.config.Save()
			return
		}
	}

	// Left click on file list area: switch focus and handle the click
	if msg.Button == tea.MouseButtonLeft && me.Action == tea.MouseActionPress && msg.X < borderCol-1 {
		action := a.fileList.HandleMouse(msg)
		if action != ui.ClickNone {
			// Switch focus to file list
			if a.focus != FocusFileList {
				a.focus = FocusFileList
				a.fileList.SetFocused(true)
				a.editor.SetFocused(false)
			}
			// Auto-preview: load clicked file in editor (only in view mode)
			if action == ui.ClickFile && a.editor.Mode() == ui.EditorModeView {
				if file := a.fileList.SelectedFile(); file != nil {
					a.loadFile(file.Name)
				}
			}
			a.prevCursor = a.fileList.Cursor()
		}
		return
	}

	// Left click on editor area: switch focus and place cursor
	if msg.Button == tea.MouseButtonLeft && me.Action == tea.MouseActionPress && msg.X >= a.sidebarWidth+1 {
		if a.focus != FocusEditor {
			a.focus = FocusEditor
			a.editor.SetFocused(true)
			a.fileList.SetFocused(false)
		}
		a.editor.Update(msg)
		a.prevCursor = a.fileList.Cursor()
	}
}

func (a *App) loadFile(name string) {
	content, err := a.workspace.ReadFile(name)
	if err != nil {
		a.message = fmt.Sprintf("Error loading file: %v", err)
		return
	}

	a.editor.SetContent(name, content)
}

func (a *App) refreshFiles() {
	a.lastRefreshTime = time.Now()

	if err := a.workspace.Refresh(); err != nil {
		a.message = fmt.Sprintf("Error refreshing: %v", err)
		return
	}

	files := a.workspace.Files()
	a.fileList.SetFiles(files)
	a.prevCursor = a.fileList.Cursor()

	// Stale editor detection: check if the displayed file still exists
	editorFile := a.editor.FileName()
	if editorFile != "" {
		found := false
		for _, f := range files {
			if f.Name == editorFile {
				found = true
				break
			}
		}
		if !found {
			if a.editor.Mode() == ui.EditorModeEdit {
				// Preserve content in edit mode, just warn
				a.message = "Warning: file no longer exists on disk"
			} else {
				a.editor.SetContent("", "")
				a.message = "File no longer exists"
			}
		}
	}
}

// View renders the application
func (a *App) View() string {
	if a.quitting {
		return "Goodbye!\n"
	}

	var b strings.Builder

	// Tab bar at top
	tabBar := a.tabs.View()
	b.WriteString(tabBar)
	b.WriteString("\n")

	// Main content area
	contentHeight := a.height - 4 // Tab bar + status bar

	var mainContent string
	if a.isOnPlanningTab() {
		// Planning tab: file list + editor
		fileListView := a.fileList.View()
		editorView := a.editor.View()

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, fileListView, editorView)
	} else if panel := a.activePanel(); panel != nil {
		// Agent tab: PTY panel
		mainContent = panel.View()
	}

	// Ensure content fits height (MaxHeight truncates overflow; Height only pads)
	mainContent = lipgloss.NewStyle().Height(contentHeight).MaxHeight(contentHeight).Render(mainContent)
	b.WriteString(mainContent)

	// Status bar at bottom
	statusBar := a.renderStatusBar()
	b.WriteString("\n")
	b.WriteString(statusBar)

	view := b.String()

	// Overlay dialog if visible
	if a.dialog.IsVisible() {
		view = a.dialog.View()
	}

	// Overlay help if visible
	if a.help.IsVisible() {
		view = a.help.View()
	}

	// Overlay settings if visible
	if a.settings.IsVisible() {
		view = a.settings.View()
	}

	return view
}

// keymapOverrides computes sparse overrides from the current keymap
// by comparing against defaults (only changed bindings are persisted).
func (a *App) keymapOverrides() map[string]map[string]string {
	defaults := ui.DefaultKeymap()
	overrides := make(map[string]map[string]string)

	for _, cb := range a.keymap.Contexts {
		for _, b := range cb.Bindings {
			defaultKeys := defaults.KeysFor(cb.Context, b.Action)
			currentKeys := a.keymap.KeysFor(cb.Context, b.Action)

			// Check if different from default
			if !sameKeys(currentKeys, defaultKeys) {
				if overrides[string(cb.Context)] == nil {
					overrides[string(cb.Context)] = make(map[string]string)
				}
				overrides[string(cb.Context)][string(b.Action)] = strings.Join(currentKeys, ",")
			}
		}
	}

	if len(overrides) == 0 {
		return nil
	}
	return overrides
}

// sameKeys returns true if two key slices contain the same keys (order-insensitive).
func sameKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]bool, len(a))
	for _, k := range a {
		set[k] = true
	}
	for _, k := range b {
		if !set[k] {
			return false
		}
	}
	return true
}

func (a *App) renderStatusBar() string {
	// Build status bar content
	var leftContent string
	switch {
	case !a.ctrlCPressedAt.IsZero() && time.Since(a.ctrlCPressedAt) < 2*time.Second:
		leftContent = a.theme.StatusFailed.Render("Press Ctrl+C again to quit")
	case a.message != "":
		leftContent = a.theme.Normal.Render(a.message)
		a.message = "" // Clear after showing
	default:
		// Show context hints
		km := a.keymap
		if a.isOnPlanningTab() {
			if a.editor.Mode() == ui.EditorModeEdit {
				leftContent = a.theme.KeyHint.Render("[Esc] save & exit  [Ctrl+S] save  [Shift+Arrow] select  [Alt+Arrow] word jump")
			} else {
				leftContent = a.theme.KeyHint.Render(
					"[" + km.DisplayKeysFor(ui.ContextFileList, ui.ActionMoveDown) + "] navigate  " +
						"[" + km.DisplayKeysFor(ui.ContextFileList, ui.ActionEditMode) + "] edit  " +
						"[" + km.DisplayKeysFor(ui.ContextFileList, ui.ActionNewFile) + "] new  " +
						"[" + km.DisplayKeysFor(ui.ContextFileList, ui.ActionDeleteFile) + "] delete  " +
						"[" + km.DisplayKeysFor(ui.ContextFileList, ui.ActionRefresh) + "] refresh  " +
						"[" + km.DisplayKeysFor(ui.ContextGlobal, ui.ActionCreateAgent) + "] agent  " +
						"[" + km.DisplayKeysFor(ui.ContextGlobal, ui.ActionSettings) + "] settings  " +
						"[" + km.DisplayKeysFor(ui.ContextGlobal, ui.ActionNextTab) + "] next tab")
			}
		} else {
			panel := a.activePanel()
			if panel != nil && panel.IsInputMode() {
				leftContent = a.theme.KeyHint.Render(
					"[" + km.DisplayKeysFor(ui.ContextAgentInput, ui.ActionExitInput) + "] normal  " +
						"[" + km.DisplayKeysFor(ui.ContextGlobal, ui.ActionCloseTab) + "] close  " +
						"[" + km.DisplayKeysFor(ui.ContextGlobal, ui.ActionNextTab) + "] next tab  " +
						"[scroll] scrollback")
			} else {
				leftContent = a.theme.KeyHint.Render(
					"[" + km.DisplayKeysFor(ui.ContextAgentNormal, ui.ActionEnterInput) + "] interact  " +
						"[" + km.DisplayKeysFor(ui.ContextAgentNormal, ui.ActionAgentNew) + "] new agent  " +
						"[" + km.DisplayKeysFor(ui.ContextAgentNormal, ui.ActionAgentClose) + "] close  " +
						"[" + km.DisplayKeysFor(ui.ContextGlobal, ui.ActionNextTab) + "] next tab")
			}
		}
	}

	// Right side: help hint + version
	km := a.keymap
	var versionLabel string
	if a.version == "dev" {
		versionLabel = a.theme.VersionDev.Render("dev")
	} else {
		versionLabel = a.theme.Dimmed.Render(a.version)
	}
	rightContent := a.theme.Dimmed.Render(
		"["+km.DisplayKeysFor(ui.ContextGlobal, ui.ActionToggleHelp)+"] help  "+
			"["+km.DisplayKeysFor(ui.ContextGlobal, ui.ActionQuit)+"] quit") + "  " + versionLabel

	// Calculate spacing
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	spacing := a.width - leftWidth - rightWidth
	if spacing < 1 {
		spacing = 1
	}

	statusBar := leftContent + lipgloss.NewStyle().Width(spacing).Render("") + rightContent

	return a.theme.StatusBar.Width(a.width).Render(statusBar)
}

func (a *App) updateSizes() {
	// Layout calculations
	fileListWidth := a.sidebarWidth
	// Ensure editor gets at least 40 columns
	if a.width-fileListWidth < 40 {
		fileListWidth = a.width - 40
		if fileListWidth < 16 {
			fileListWidth = 16
		}
	}
	editorWidth := a.width - fileListWidth

	contentHeight := a.height - 4 // Tab bar + status bar

	// Set component sizes
	a.tabs.SetWidth(a.width)
	a.fileList.SetSize(fileListWidth, contentHeight)
	a.fileList.SetPosition(2) // tab bar label row + tab bar border row
	a.editor.SetSize(editorWidth, contentHeight)
	a.editor.SetPosition(fileListWidth+1, 2) // +1 for border, 2 for tab bar
	a.dialog.SetSize(a.width, a.height)
	a.help.SetSize(a.width, a.height)
	a.settings.SetSize(a.width, a.height)
	// Resize all agent PTY panels
	for _, tab := range a.agentTabs {
		tab.panel.SetSize(a.width, contentHeight)
		if tab.session != nil {
			rows, cols := tab.panel.TerminalSize()
			_ = a.sessionBackend.Resize(tab.session.BackendHandle, uint16(rows), uint16(cols))
		}
	}
}

// hookSettingsJSON returns a --settings JSON string that configures Claude Code
// hooks to write state changes to the given file path. This enables Planck to
// detect when the agent needs human input (permission prompts).
func hookSettingsJSON(stateFile string) string {
	// The hook command writes "needs_input" to the state file when Claude
	// shows a permission prompt. The command is a simple shell one-liner
	// so it runs instantly with no dependencies.
	cmd := fmt.Sprintf("echo needs_input > %s", stateFile)
	return fmt.Sprintf(`{"hooks":{"Notification":[{"matcher":"permission_prompt","hooks":[{"type":"command","command":%q}]}]}}`, cmd)
}

// readHookState reads the hook state file for a tab. Returns the state string
// (e.g., "needs_input") or empty string if no state is set.
func (a *App) readHookState(tab *AgentTab) string {
	if tab.stateFile == "" {
		return ""
	}
	data, err := os.ReadFile(tab.stateFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// clearHookState removes the hook state file so the tab returns to normal state.
func (a *App) clearHookState(tab *AgentTab) {
	if tab.stateFile != "" {
		os.Remove(tab.stateFile)
	}
}

// cleanupHookState removes hook state files when a tab is done.
func (a *App) cleanupHookState(tab *AgentTab) {
	a.clearHookState(tab)
}

// loadMarkdownStyleFromConfig converts config.MarkdownStyle to ui.MarkdownStyleConfig
func loadMarkdownStyleFromConfig(cfg *config.Config) ui.MarkdownStyleConfig {
	result := ui.MarkdownStyleConfig{
		GlobalTheme: ui.ThemeName(cfg.MarkdownStyle.Theme),
		Overrides:   make(map[ui.ElementType]ui.ThemeName),
	}
	if result.GlobalTheme == "" {
		result.GlobalTheme = ui.ThemeNeoBrutalist
	}
	for elem, theme := range cfg.MarkdownStyle.Overrides {
		result.Overrides[ui.ElementType(elem)] = ui.ThemeName(theme)
	}
	return result
}
