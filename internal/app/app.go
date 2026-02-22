package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/anthropics/planck/internal/config"
	"github.com/anthropics/planck/internal/session"
	"github.com/anthropics/planck/internal/store"
	"github.com/anthropics/planck/internal/ui"
	"github.com/anthropics/planck/internal/workspace"
)

// Focus represents which panel is focused
type Focus int

const (
	FocusFileList Focus = iota
	FocusEditor
	FocusAgent
)

const maxAgentTabs = 8

// AgentTab represents a single agent tab with its own PTY session
type AgentTab struct {
	id          string // unique tab ID
	agentKey    string // config key (e.g., "claude-code")
	baseLabel   string // display label from config (e.g., "Claude")
	instanceNum int    // per-type instance number (1, 2, ...)
	session     *session.Session
	panel       *ui.PTYPanel
	agentCfg    config.AgentConfig
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
	sessionBackend *session.PTYBackend

	// UI components
	theme    *ui.Theme
	tabs     *ui.TabBar
	fileList *ui.FileList
	editor   *ui.Editor
	dialog   *ui.Dialog
	help     *ui.Help

	// Overlays
	folderPicker        *ui.FolderPicker
	folderPickerVisible bool

	// Multi-agent tabs
	agentTabs      []*AgentTab    // ordered list of agent tabs
	activeTabIdx   int            // 0 = planning, 1+ = agent tabs
	nextInstanceNum map[string]int // per-agent-type instance counter

	// Current state
	focus         Focus
	width, height int
	message       string
	quitting      bool

	// File watching
	watchChan <-chan struct{}

	// Auto-preview tracking
	prevCursor int
}

// New creates a new application
func New(cfg *config.Config, configDir, folder string, backend *session.PTYBackend) (*App, error) {
	// Open store
	st, err := store.Open(cfg.StateDBPath())
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	// Create workspace
	ws, err := workspace.New(folder)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	// Create theme and UI components
	theme := ui.DefaultTheme()

	app := &App{
		config:          cfg,
		configDir:       configDir,
		folder:          folder,
		store:           st,
		workspace:       ws,
		sessionBackend:  backend,
		theme:           theme,
		tabs:            ui.NewTabBar(theme),
		fileList:        ui.NewFileList(theme),
		editor:          ui.NewEditor(theme),
		dialog:          ui.NewDialog(theme),
		help:            ui.NewHelp(theme),
		activeTabIdx:    0,
		focus:           FocusFileList,
		nextInstanceNum: make(map[string]int),
	}

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

	return a.watchForChanges()
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

	// Handle quit
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "ctrl+c" {
			a.killAllSessions()
			a.quitting = true
			return a, tea.Quit
		}
	}

	// Handle file changes
	if _, ok := msg.(fileChangedMsg); ok {
		a.refreshFiles()
		cmds = append(cmds, a.watchForChanges())
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

	// Dialog takes priority
	if a.dialog.IsVisible() {
		dialog, cmd := a.dialog.Update(msg)
		a.dialog = dialog
		return a, cmd
	}

	// Help takes priority
	if a.help.IsVisible() {
		help, cmd := a.help.Update(msg)
		a.help = help
		return a, cmd
	}

	// Folder picker overlay takes priority
	if a.folderPickerVisible {
		if _, ok := msg.(ui.FolderSelectedMsg); ok {
			a.folderPickerVisible = false
			return a, a.switchFolder(msg.(ui.FolderSelectedMsg).Folder)
		}
		if _, ok := msg.(ui.FolderPickerCancelledMsg); ok {
			a.folderPickerVisible = false
			return a, nil
		}
		picker, cmd := a.folderPicker.Update(msg)
		a.folderPicker = picker
		return a, cmd
	}

	// Handle PTY messages — route to the correct agent tab by session ID
	if msg, ok := msg.(ui.PTYWriteMsg); ok {
		if tab := a.findAgentTabBySessionID(msg.SessionID); tab != nil {
			a.sessionBackend.Write(tab.session.BackendHandle, msg.Data)
		}
	}

	if msg, ok := msg.(ui.PTYRenderMsg); ok {
		if tab := a.findAgentTabBySessionID(msg.SessionID); tab != nil {
			tab.panel.SetContent(msg.Content)
			cmds = append(cmds, a.pollAgentTab(tab))
		}
	}

	if msg, ok := msg.(ui.PTYExitedMsg); ok {
		if tab := a.findAgentTabBySessionID(msg.SessionID); tab != nil {
			tab.panel.SetStatus("completed")
			a.syncTabBar()
			a.message = fmt.Sprintf("%s completed (exit code: %d)", tab.baseLabel, msg.ExitCode)
		}
	}

	// Handle mouse events on planning tab
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		if a.isOnPlanningTab() {
			cmd := a.handlePlanningMouse(mouseMsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
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
			// Auto-preview: load file when cursor moves (only in view mode)
			if a.fileList.Cursor() != a.prevCursor && a.editor.Mode() == ui.EditorModeView {
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

	// Global keys
	switch key {
	case "?":
		if a.editor.Mode() == ui.EditorModeEdit {
			return nil
		}
		if a.isInPTYInputMode() {
			return nil
		}
		a.help.Toggle()
		return nil

	case "q":
		if a.editor.Mode() == ui.EditorModeEdit {
			return nil
		}
		if a.isInPTYInputMode() {
			return nil
		}
		a.killAllSessions()
		a.quitting = true
		return tea.Quit

	case "tab":
		if a.editor.Mode() == ui.EditorModeEdit {
			return nil
		}
		return a.cycleTab()

	case "ctrl+x":
		// Close agent tab — works even in input mode
		if a.isOnAgentTab() {
			if panel := a.activePanel(); panel != nil {
				panel.ExitInputMode()
			}
			return a.closeCurrentAgentTab()
		}
		return nil

	case "a":
		if a.editor.Mode() == ui.EditorModeEdit {
			return nil
		}
		if a.isInPTYInputMode() {
			return nil
		}
		return a.createAgentTab()

	case "x":
		if a.editor.Mode() == ui.EditorModeEdit {
			return nil
		}
		if a.isInPTYInputMode() {
			return nil
		}
		return a.closeCurrentAgentTab()
	}

	// Number keys 1-9 for tab switching
	if key >= "1" && key <= "9" {
		if a.editor.Mode() == ui.EditorModeEdit {
			return nil
		}
		if a.isInPTYInputMode() {
			return nil
		}
		idx := int(key[0] - '0') - 1 // "1" → 0, "2" → 1, etc.
		return a.switchToTab(idx)
	}

	// Tab-specific keys
	if a.isOnPlanningTab() {
		return a.handlePlanningTabKey(key, msg)
	}

	return nil
}

// cycleTab cycles to the next tab
func (a *App) cycleTab() tea.Cmd {
	totalTabs := 1 + len(a.agentTabs) // planning + agents
	if totalTabs <= 1 {
		return nil // Only planning tab, nowhere to cycle
	}

	// Exit input mode on current agent tab
	if panel := a.activePanel(); panel != nil {
		panel.ExitInputMode()
	}

	next := (a.activeTabIdx + 1) % totalTabs
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
			a.launchAgentTab(agentKeys[result.Selected])
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
	panel := ui.NewPTYPanel(a.theme)
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
	for _, arg := range agentCfg.PlanningArgs {
		switch arg {
		case "--dangerously-skip-permissions":
			launchArgs = append(launchArgs, arg)
		}
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
	a.sessionBackend.Resize(sess.BackendHandle, uint16(rows), uint16(cols))

	// Wire scrollback buffer from backend to panel
	if sb := a.sessionBackend.GetScrollback(sess.BackendHandle); sb != nil {
		tab.panel.SetScrollback(sb)
	}

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
	}

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

// computeTabLabel returns the display label for an agent tab,
// including instance number if there are multiple tabs of the same type
func (a *App) computeTabLabel(tab *AgentTab) string {
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

// syncTabBar rebuilds the tab bar to match current agent tabs
func (a *App) syncTabBar() {
	tabs := []ui.TabInfo{{Label: "Planning"}}
	for _, tab := range a.agentTabs {
		status := ""
		if tab.panel != nil {
			// Check if panel shows completed status
			if tab.session != nil {
				s, _ := a.sessionBackend.Status(tab.session.BackendHandle)
				if s == session.StatusCompleted {
					status = "completed"
				}
			}
		}
		tabs = append(tabs, ui.TabInfo{
			Label:  a.computeTabLabel(tab),
			Status: status,
		})
	}
	a.tabs.SetTabs(tabs)
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

		return ui.PTYRenderMsg{SessionID: sessionID, Content: content}
	}
}

// killAllSessions kills all running agent sessions
func (a *App) killAllSessions() {
	for _, tab := range a.agentTabs {
		if tab.session != nil {
			_ = a.sessionBackend.Kill(tab.session.BackendHandle)
		}
	}
}

func (a *App) handlePlanningTabKey(key string, msg tea.KeyMsg) tea.Cmd {
	// Skip if in edit mode - let editor handle
	if a.editor.Mode() == ui.EditorModeEdit {
		return nil
	}

	switch a.focus {
	case FocusFileList:
		switch key {
		case "enter":
			// Load selected file (no-op on directories)
			if file := a.fileList.SelectedFile(); file != nil {
				a.loadFile(file.Name)
				a.prevCursor = a.fileList.Cursor()
			}

		case "e":
			// Enter edit mode on current file
			if a.editor.FileName() != "" {
				a.editor.EnterEditMode()
				a.focus = FocusEditor
				a.editor.SetFocused(true)
				a.fileList.SetFocused(false)
			}

		case "n":
			// New file
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

		case "d", "D":
			// Delete file (no-op on directories)
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
			}

		case "l", "right":
			if a.fileList.IsSelectedDir() {
				a.fileList.ExpandSelected()
			}

		case "h", "left":
			if a.fileList.IsSelectedDir() {
				a.fileList.CollapseSelected()
			}

		case "o":
			// Open folder picker
			a.showFolderPicker()
		}
	}

	return nil
}

func (a *App) handlePlanningMouse(msg tea.MouseMsg) tea.Cmd {
	me := tea.MouseEvent(msg)

	// Wheel events: forward to editor regardless of focus
	if me.IsWheel() {
		a.editor.Update(msg)
		return nil
	}

	// Left click on editor area: switch focus and place cursor
	fileListWidth := 24
	if msg.Button == tea.MouseButtonLeft && me.Action == tea.MouseActionPress && msg.X >= fileListWidth+1 {
		if a.focus != FocusEditor {
			a.focus = FocusEditor
			a.editor.SetFocused(true)
			a.fileList.SetFocused(false)
		}
		a.editor.Update(msg)
		a.prevCursor = a.fileList.Cursor()
		return nil
	}

	return nil
}

func (a *App) updateFocus() {
	a.fileList.SetFocused(a.focus == FocusFileList)
	a.editor.SetFocused(a.focus == FocusEditor)
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
	if err := a.workspace.Refresh(); err != nil {
		a.message = fmt.Sprintf("Error refreshing: %v", err)
		return
	}

	files := a.workspace.Files()
	a.fileList.SetFiles(files)
	a.prevCursor = a.fileList.Cursor()
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

	// Ensure content fits height
	mainContent = lipgloss.NewStyle().Height(contentHeight).Render(mainContent)
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

	// Overlay folder picker if visible
	if a.folderPickerVisible && a.folderPicker != nil {
		view = a.folderPicker.View()
	}

	return view
}

func (a *App) renderStatusBar() string {
	// Build status bar content
	var leftContent string
	if a.message != "" {
		leftContent = a.theme.Normal.Render(a.message)
		a.message = "" // Clear after showing
	} else if a.folderPickerVisible {
		leftContent = a.theme.KeyHint.Render("[^|v] navigate  [Enter] select  [b] browse  [Esc] cancel")
	} else {
		// Show context hints
		if a.isOnPlanningTab() {
			if a.editor.Mode() == ui.EditorModeEdit {
				leftContent = a.theme.KeyHint.Render("[Esc] save & exit  [Ctrl+S] save")
			} else {
				leftContent = a.theme.KeyHint.Render("[^|v] navigate  [e] edit  [n] new  [d] delete  [o] folder  [a] agent  [Tab] cycle")
			}
		} else {
			panel := a.activePanel()
			if panel != nil && panel.IsInputMode() {
				leftContent = a.theme.KeyHint.Render("[Ctrl+\\] normal  [Ctrl+X] close  [Tab] cycle  [scroll] scrollback")
			} else {
				leftContent = a.theme.KeyHint.Render("[i] interact  [a] new agent  [x] close  [Tab] cycle")
			}
		}
	}

	// Right side: help hint
	rightContent := a.theme.Dimmed.Render("[?] help  [q] quit")

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
	fileListWidth := 24
	editorWidth := a.width - fileListWidth
	if editorWidth < 40 {
		editorWidth = 40
	}

	contentHeight := a.height - 4 // Tab bar + status bar

	// Set component sizes
	a.tabs.SetWidth(a.width)
	a.fileList.SetSize(fileListWidth, contentHeight)
	a.editor.SetSize(editorWidth, contentHeight)
	a.editor.SetPosition(fileListWidth+1, 2) // +1 for border, 2 for tab bar
	a.dialog.SetSize(a.width, a.height)
	a.help.SetSize(a.width, a.height)
	if a.folderPicker != nil {
		a.folderPicker.SetSize(a.width, a.height)
	}

	// Resize all agent PTY panels
	for _, tab := range a.agentTabs {
		tab.panel.SetSize(a.width, contentHeight)
		if tab.session != nil {
			rows, cols := tab.panel.TerminalSize()
			a.sessionBackend.Resize(tab.session.BackendHandle, uint16(rows), uint16(cols))
		}
	}
}

func (a *App) showFolderPicker() {
	// Load fresh recent folders
	recent, err := workspace.LoadRecentFolders(a.configDir)
	if err != nil {
		recent = &workspace.RecentFolders{}
	}

	a.folderPicker = ui.NewFolderPicker(a.theme, recent.Folders)
	a.folderPicker.SetOverlayMode(true)
	a.folderPicker.SetSize(a.width, a.height)
	a.folderPickerVisible = true
}

func (a *App) switchFolder(newFolder string) tea.Cmd {
	// Auto-save editor if modified
	if a.editor.IsModified() {
		content := a.editor.GetContent()
		fileName := a.editor.FileName()
		if fileName != "" {
			_ = a.workspace.SaveFile(fileName, content)
		}
		a.editor.ClearModified()
	}

	// Kill all agent sessions and remove all agent tabs
	a.killAllSessions()
	a.agentTabs = nil
	a.nextInstanceNum = make(map[string]int)

	// Stop old workspace watcher
	a.workspace.StopWatch()

	// Create new workspace
	ws, err := workspace.New(newFolder)
	if err != nil {
		a.message = fmt.Sprintf("Error switching folder: %v", err)
		return nil
	}
	a.workspace = ws
	a.folder = newFolder

	// Start new file watcher
	watchChan, err := a.workspace.Watch()
	if err == nil {
		a.watchChan = watchChan
	}

	// Reload config
	cfg, err := config.Load(newFolder)
	if err == nil {
		a.config = cfg
	}

	// Reopen store
	a.store.Close()
	st, err := store.Open(a.config.StateDBPath())
	if err == nil {
		a.store = st
	}

	// Update UI — back to just Planning tab
	a.tabs.SetFolderPath(newFolder)
	a.syncTabBar()
	a.refreshFiles()
	a.editor.SetContent("", "")

	// Select first file if available
	files := a.workspace.Files()
	if len(files) > 0 {
		a.loadFile(files[0].Name)
		a.prevCursor = a.fileList.Cursor()
	}

	// Switch to planning tab
	a.activeTabIdx = 0
	a.tabs.SetActiveIdx(0)
	a.focus = FocusFileList
	a.fileList.SetFocused(true)
	a.updateFocus()

	// Update recent folders
	recent, err := workspace.LoadRecentFolders(a.configDir)
	if err != nil {
		recent = &workspace.RecentFolders{}
	}
	recent.Add(newFolder)
	_ = recent.Save(a.configDir)

	a.message = fmt.Sprintf("Switched to %s", newFolder)

	return a.watchForChanges()
}

// FolderPickerModel is a standalone model for the folder picker
type FolderPickerModel struct {
	picker         *ui.FolderPicker
	theme          *ui.Theme
	SelectedFolder string
	Quit           bool
	width, height  int
}

// NewFolderPickerModel creates a new folder picker model
func NewFolderPickerModel(recentFolders []string) *FolderPickerModel {
	theme := ui.DefaultTheme()
	return &FolderPickerModel{
		picker: ui.NewFolderPicker(theme, recentFolders),
		theme:  theme,
	}
}

// Init initializes the folder picker
func (m *FolderPickerModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *FolderPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.picker.SetSize(msg.Width-4, msg.Height-4)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.Quit = true
			return m, tea.Quit
		}

	case ui.FolderSelectedMsg:
		m.SelectedFolder = msg.Folder
		return m, tea.Quit
	}

	picker, cmd := m.picker.Update(msg)
	m.picker = picker
	return m, cmd
}

// View renders the folder picker
func (m *FolderPickerModel) View() string {
	if m.width == 0 {
		m.width = 80
		m.height = 24
		m.picker.SetSize(m.width-4, m.height-4)
	}
	return m.picker.View()
}
