package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sabizmil/planck/internal/store"
	"github.com/sabizmil/planck/internal/tmux"
	"github.com/sabizmil/planck/internal/ui"
)

// persistSession saves the current state of an agent tab to SQLite.
func (a *App) persistSession(tab *AgentTab) {
	if tab.session == nil {
		return
	}

	tmuxName := ""
	if tb, ok := a.sessionBackend.(*tmux.TmuxBackend); ok {
		tmuxName = tb.GetTmuxSessionName(tab.session.BackendHandle)
	}

	dbSession := &store.Session{
		ID:              tab.session.ID,
		FilePath:        tab.session.TaskID,
		Status:          string(tab.session.Status),
		StartedAt:       tab.session.StartedAt,
		AgentKey:        tab.agentKey,
		AgentLabel:      tab.baseLabel,
		CustomTitle:     tab.customTitle,
		TmuxSessionName: tmuxName,
		BackendType:     a.sessionBackend.Name(),
		WorkDir:         a.folder,
		Command:         tab.agentCfg.Command,
		Args:            store.EncodeArgs(tab.agentCfg.PlanningArgs),
	}

	_ = a.store.SaveSession(dbSession)
}

// markSessionCompleted updates a session's status to completed in SQLite.
func (a *App) markSessionCompleted(sessionID string, exitCode int) {
	ec := exitCode
	_ = a.store.UpdateSessionStatus(sessionID, "completed", &ec)
}

// markSessionCanceled updates a session's status to canceled in SQLite.
func (a *App) markSessionCanceled(sessionID string) {
	_ = a.store.UpdateSessionStatus(sessionID, "canceled", nil)
}

// isTmuxBackend returns true if the session backend is tmux.
func (a *App) isTmuxBackend() bool {
	return a.sessionBackend.Name() == "tmux"
}

// recoverSessions attempts to recover agent tabs from a previous planck session.
// It queries SQLite for "running" sessions and cross-references with live tmux sessions.
// Returns tea.Cmd to start polling recovered tabs.
func (a *App) recoverSessions() tea.Cmd {
	tb, isTmux := a.sessionBackend.(*tmux.TmuxBackend)
	if !isTmux {
		// PTY sessions can't survive a restart — mark any stale "running" sessions as failed
		a.cleanupStaleSessions()
		return nil
	}

	// Get sessions that were "running" when planck last exited
	activeSessions, err := a.store.ListActiveSessions()
	if err != nil || len(activeSessions) == 0 {
		return nil
	}

	var cmds []tea.Cmd

	for _, dbSess := range activeSessions {
		// Skip if no tmux session name recorded
		if dbSess.TmuxSessionName == "" {
			_ = a.store.UpdateSessionStatus(dbSess.ID, "failed", nil)
			continue
		}

		// Skip if agent config no longer exists
		agentCfg, ok := a.config.Agents[dbSess.AgentKey]
		if !ok {
			_ = a.store.UpdateSessionStatus(dbSess.ID, "failed", nil)
			continue
		}

		// Try to reattach to the tmux session
		sess, err := tb.ReattachSession(dbSess.TmuxSessionName, dbSess.ID, a.folder)
		if err != nil {
			// Tmux session no longer exists
			_ = a.store.UpdateSessionStatus(dbSess.ID, "failed", nil)
			continue
		}

		// Assign instance number
		a.nextInstanceNum[dbSess.AgentKey]++
		instanceNum := a.nextInstanceNum[dbSess.AgentKey]

		baseLabel := dbSess.AgentLabel
		if baseLabel == "" {
			baseLabel = a.config.GetAgentLabel(dbSess.AgentKey)
		}

		// Create panel
		panel := ui.NewPTYPanel(a.theme, a.keymap)
		contentHeight := a.height - 4
		if contentHeight < 1 {
			contentHeight = 24 // default before first resize
		}
		panel.SetSize(a.width, contentHeight)

		tab := &AgentTab{
			id:             dbSess.ID,
			agentKey:       dbSess.AgentKey,
			baseLabel:      baseLabel,
			customTitle:    dbSess.CustomTitle,
			titleFromInput: dbSess.CustomTitle != "",
			instanceNum:    instanceNum,
			session:        sess,
			panel:          panel,
			agentCfg:       agentCfg,
		}

		// Show panel
		panel.Show(dbSess.AgentKey, baseLabel, sess.ID)

		// Set status based on whether tmux reports it as still running
		if sess.Status == "completed" {
			panel.SetStatus("completed")
		} else {
			panel.SetStatus("running")
			// Resize to match current terminal
			rows, cols := panel.TerminalSize()
			_ = a.sessionBackend.Resize(sess.BackendHandle, uint16(rows), uint16(cols))
		}

		// Wire scrollback (nil for tmux, which is fine)
		if sb := a.sessionBackend.GetScrollback(sess.BackendHandle); sb != nil {
			panel.SetScrollback(sb)
		}

		a.agentTabs = append(a.agentTabs, tab)
		cmds = append(cmds, a.pollAgentTab(tab))
	}

	if len(a.agentTabs) > 0 {
		a.syncTabBar()
	}

	return tea.Batch(cmds...)
}

// cleanupStaleSessions marks any "running" sessions as failed.
// Called on startup when using PTY backend (sessions can't survive).
func (a *App) cleanupStaleSessions() {
	sessions, err := a.store.ListActiveSessions()
	if err != nil {
		return
	}
	for _, s := range sessions {
		_ = a.store.UpdateSessionStatus(s.ID, "failed", nil)
	}
}

// cleanupOldSessions removes completed/failed sessions older than 7 days.
func (a *App) cleanupOldSessions() {
	_ = a.store.CleanupOldSessions(7 * 24 * time.Hour)
}
