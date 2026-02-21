package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

// App is the main application model
type App struct {
	// Configuration
	config    *config.Config
	configDir string
	folder    string

	// State management
	store          *store.Store
	workspace      *workspace.Workspace
	sessionBackend session.Backend

	// UI components
	theme       *ui.Theme
	tabs        *ui.TabBar
	fileList    *ui.FileList
	editor      *ui.Editor
	ptyPanel    *ui.PTYPanel
	statusPanel *ui.StatusPanel
	dialog      *ui.Dialog
	help        *ui.Help

	// Overlays
	folderPicker        *ui.FolderPicker
	folderPickerVisible bool

	// Current state
	activeTab      ui.Tab
	focus          Focus
	width, height  int
	message        string
	quitting       bool
	currentSession *session.Session

	// File watching
	watchChan <-chan struct{}

	// Auto-preview tracking
	prevCursor int
}

// New creates a new application
func New(cfg *config.Config, configDir, folder string, backend session.Backend) (*App, error) {
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
		config:         cfg,
		configDir:      configDir,
		folder:         folder,
		store:          st,
		workspace:      ws,
		sessionBackend: backend,
		theme:          theme,
		tabs:           ui.NewTabBar(theme),
		fileList:       ui.NewFileList(theme),
		editor:         ui.NewEditor(theme),
		ptyPanel:       ui.NewPTYPanel(theme),
		statusPanel:    ui.NewStatusPanel(theme),
		dialog:         ui.NewDialog(theme),
		help:           ui.NewHelp(theme),
		activeTab:      ui.TabPlanning,
		focus:          FocusFileList,
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

	// Handle tick for status panel
	if _, ok := msg.(ui.TickMsg); ok {
		// Update elapsed time display
		if a.statusPanel != nil {
			cmds = append(cmds, a.statusPanel.StartTicker())
		}
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

	// Handle PTY messages
	if msg, ok := msg.(ui.PTYWriteMsg); ok {
		if a.currentSession != nil {
			if ptyBackend, ok := a.sessionBackend.(*session.PTYBackend); ok {
				ptyBackend.Write(a.currentSession.BackendHandle, msg.Data)
			}
		}
	}

	if msg, ok := msg.(ui.PTYRenderMsg); ok {
		a.ptyPanel.SetContent(msg.Content)
		cmds = append(cmds, a.pollPTY())
	}

	if msg, ok := msg.(ui.PTYExitedMsg); ok {
		a.ptyPanel.SetStatus("completed")
		a.statusPanel.SetAgentStatus(ui.AgentIdle)
		a.message = fmt.Sprintf("Session completed (exit code: %d)", msg.ExitCode)
	}

	// Handle mouse events on planning tab
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		if a.activeTab == ui.TabPlanning {
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
	if a.activeTab == ui.TabPlanning {
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
	} else {
		// Agent tab
		if a.ptyPanel.IsVisible() {
			panel, cmd := a.ptyPanel.Update(msg)
			a.ptyPanel = panel
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
		if a.ptyPanel.IsInputMode() {
			return nil
		}
		a.help.Toggle()
		return nil

	case "q":
		if a.editor.Mode() == ui.EditorModeEdit {
			// In edit mode, q is just text
			return nil
		}
		if a.ptyPanel.IsInputMode() {
			// In PTY input mode, q is just text
			return nil
		}
		a.quitting = true
		return tea.Quit

	case "tab":
		// Handle tab switching (works even in PTY input mode)
		if a.editor.Mode() == ui.EditorModeEdit {
			return nil
		}
		if a.activeTab == ui.TabPlanning {
			a.activeTab = ui.TabAgent
			a.tabs.SetActiveTab(ui.TabAgent)
			a.focus = FocusAgent
			a.updateFocus()
			return a.startAgentSession()
		} else {
			a.ptyPanel.ExitInputMode()
			a.activeTab = ui.TabPlanning
			a.tabs.SetActiveTab(ui.TabPlanning)
			a.focus = FocusFileList
			a.fileList.SetFocused(true)
			a.updateFocus()
		}
		return nil

	case "1":
		if a.editor.Mode() == ui.EditorModeEdit {
			return nil
		}
		if a.activeTab == ui.TabAgent {
			a.ptyPanel.ExitInputMode()
		}
		a.activeTab = ui.TabPlanning
		a.tabs.SetActiveTab(ui.TabPlanning)
		a.focus = FocusFileList
		a.fileList.SetFocused(true)
		a.updateFocus()
		return nil

	case "2":
		if a.editor.Mode() == ui.EditorModeEdit {
			return nil
		}
		a.activeTab = ui.TabAgent
		a.tabs.SetActiveTab(ui.TabAgent)
		a.focus = FocusAgent
		a.updateFocus()
		return a.startAgentSession()
	}

	// Tab-specific keys
	if a.activeTab == ui.TabPlanning {
		return a.handlePlanningTabKey(key, msg)
	}

	return nil
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

func (a *App) startAgentSession() tea.Cmd {
	// Check if already running
	if a.currentSession != nil {
		// Session already exists, just show the panel
		a.ptyPanel.Show("claude", "Claude Code", a.currentSession.ID)
		a.ptyPanel.EnterInputMode() // Auto-enter input mode on re-entry
		return a.pollPTY()
	}

	// Check if Claude is available
	if !a.sessionBackend.IsAvailable() {
		a.message = "Claude CLI not available"
		return nil
	}

	// Launch interactive Claude session in the workspace folder
	ctx := context.Background()
	sess, err := a.sessionBackend.Launch(ctx, a.folder, "")
	if err != nil {
		a.message = fmt.Sprintf("Error launching Claude: %v", err)
		return nil
	}

	a.currentSession = sess

	// Resize the PTY to match current panel dimensions
	if ptyBackend, ok := a.sessionBackend.(*session.PTYBackend); ok {
		rows, cols := a.ptyPanel.TerminalSize()
		ptyBackend.Resize(sess.BackendHandle, uint16(rows), uint16(cols))
	}

	// Show PTY panel
	a.ptyPanel.Show("claude", "Claude Code", sess.ID)
	a.ptyPanel.EnterInputMode() // Start in input mode for interactive use

	// Update status panel
	a.statusPanel.SetAgentStatus(ui.AgentRunning)

	a.message = "Claude session started"

	// Return the first poll command to kick off the polling loop
	return a.pollPTY()
}

// pollPTY returns a tea.Cmd that polls the PTY for rendered output
func (a *App) pollPTY() tea.Cmd {
	sess := a.currentSession
	if sess == nil {
		return nil
	}

	ptyBackend, ok := a.sessionBackend.(*session.PTYBackend)
	if !ok {
		return nil
	}

	handle := sess.BackendHandle
	sessionID := sess.ID

	return func() tea.Msg {
		time.Sleep(50 * time.Millisecond)

		// Check if session has exited
		status, err := ptyBackend.Status(handle)
		if err != nil || status != session.StatusRunning {
			exitCode, _ := ptyBackend.GetExitCode(handle)
			return ui.PTYExitedMsg{SessionID: sessionID, ExitCode: exitCode}
		}

		// Get rendered terminal output
		content, err := ptyBackend.Render(handle)
		if err != nil {
			return ui.PTYRenderMsg{SessionID: sessionID, Content: ""}
		}

		return ui.PTYRenderMsg{SessionID: sessionID, Content: content}
	}
}

func (a *App) updateFocus() {
	a.fileList.SetFocused(a.focus == FocusFileList)
	a.editor.SetFocused(a.focus == FocusEditor)
	a.ptyPanel.SetFocused(a.focus == FocusAgent)
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
	a.statusPanel.SetFiles(files)
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
	if a.activeTab == ui.TabPlanning {
		// Planning tab: file list + editor + status panel
		fileListView := a.fileList.View()
		editorView := a.editor.View()
		statusView := a.statusPanel.View()

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, fileListView, editorView, statusView)
	} else {
		// Agent tab: PTY panel + status panel
		ptyView := a.ptyPanel.View()
		statusView := a.statusPanel.View()

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, ptyView, statusView)
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
		leftContent = a.theme.KeyHint.Render("[↑↓] navigate  [Enter] select  [b] browse  [Esc] cancel")
	} else {
		// Show context hints
		if a.activeTab == ui.TabPlanning {
			if a.editor.Mode() == ui.EditorModeEdit {
				leftContent = a.theme.KeyHint.Render("[Esc] save & exit  [Ctrl+S] save")
			} else {
				leftContent = a.theme.KeyHint.Render("[↑↓] navigate  [Enter] open  [e] edit  [←→] collapse/expand  [n] new  [d] delete  [o] folder  [Tab] agent")
			}
		} else {
			if a.ptyPanel.IsInputMode() {
				leftContent = a.theme.KeyHint.Render("[Tab] planning  [Ctrl+\\] scrollback mode")
			} else if a.ptyPanel.IsScrollback() {
				leftContent = a.theme.KeyHint.Render("[j/k] scroll  [g/G] top/bottom  [i] interact  [Tab] planning")
			} else {
				leftContent = a.theme.KeyHint.Render("[i] interact  [s] scrollback  [Tab] planning")
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
	statusPanelWidth := 22
	editorWidth := a.width - fileListWidth - statusPanelWidth
	if editorWidth < 40 {
		editorWidth = 40
	}

	contentHeight := a.height - 4 // Tab bar + status bar

	// Set component sizes
	a.tabs.SetWidth(a.width)
	a.fileList.SetSize(fileListWidth, contentHeight)
	a.editor.SetSize(editorWidth, contentHeight)
	a.editor.SetPosition(fileListWidth+1, 2) // +1 for border, 2 for tab bar
	a.statusPanel.SetSize(statusPanelWidth, contentHeight)
	a.ptyPanel.SetSize(a.width-statusPanelWidth, contentHeight)
	a.dialog.SetSize(a.width, a.height)
	a.help.SetSize(a.width, a.height)
	if a.folderPicker != nil {
		a.folderPicker.SetSize(a.width, a.height)
	}

	// Resize PTY to match panel dimensions
	if a.currentSession != nil {
		if ptyBackend, ok := a.sessionBackend.(*session.PTYBackend); ok {
			rows, cols := a.ptyPanel.TerminalSize()
			ptyBackend.Resize(a.currentSession.BackendHandle, uint16(rows), uint16(cols))
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

	// Kill active PTY session if any
	if a.currentSession != nil {
		if ptyBackend, ok := a.sessionBackend.(*session.PTYBackend); ok {
			_ = ptyBackend.Kill(a.currentSession.BackendHandle)
		}
		a.currentSession = nil
		a.ptyPanel.Hide()
	}

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

	// Update UI
	a.tabs.SetFolderPath(newFolder)
	a.refreshFiles()
	a.editor.SetContent("", "")

	// Select first file if available
	files := a.workspace.Files()
	if len(files) > 0 {
		a.loadFile(files[0].Name)
		a.prevCursor = a.fileList.Cursor()
	}

	// Switch to planning tab
	a.activeTab = ui.TabPlanning
	a.tabs.SetActiveTab(ui.TabPlanning)
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
