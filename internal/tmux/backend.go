package tmux

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sabizmil/planck/internal/perf"
	"github.com/sabizmil/planck/internal/session"
)

// TmuxSession represents an active session running inside tmux.
type TmuxSession struct {
	id       string // planck session ID (UUID)
	tmuxName string // tmux session name (e.g., "planck-a3f2b1c4-d9e8f7a6")
	workDir  string

	mu       sync.Mutex
	done     chan struct{}
	exitCode int
	exited   bool
	title    string
}

// TmuxBackend implements session.InteractiveBackend using tmux.
// Each agent runs in its own tmux session, surviving planck restarts.
type TmuxBackend struct {
	prefix      string // naming prefix (e.g., "planck")
	workDirHash string // first 8 hex chars of SHA-256(workDir)
	sessionsDir string
	extraArgs   []string

	mu       sync.Mutex
	sessions map[string]*TmuxSession
}

// NewTmuxBackend creates a new tmux backend.
// prefix is used for naming tmux sessions.
// workDir is hashed to isolate sessions per project folder.
func NewTmuxBackend(prefix, sessionsDir, workDir string, extraArgs []string) *TmuxBackend {
	hash := sha256.Sum256([]byte(workDir))
	return &TmuxBackend{
		prefix:      prefix,
		workDirHash: fmt.Sprintf("%x", hash[:4]), // 8 hex chars
		sessionsDir: sessionsDir,
		extraArgs:   extraArgs,
		sessions:    make(map[string]*TmuxSession),
	}
}

// Name returns the backend name.
func (t *TmuxBackend) Name() string {
	return "tmux"
}

// IsAvailable checks if tmux is installed.
func (t *TmuxBackend) IsAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// tmuxSessionName generates a unique tmux session name for a given session ID.
func (t *TmuxBackend) tmuxSessionName(shortID string) string {
	return fmt.Sprintf("%s-%s-%s", t.prefix, t.workDirHash, shortID)
}

// tmuxSessionPrefix returns the prefix that all sessions for this workdir share.
func (t *TmuxBackend) tmuxSessionPrefix() string {
	return fmt.Sprintf("%s-%s-", t.prefix, t.workDirHash)
}

// runTmux executes a tmux command and returns its output.
func runTmux(args ...string) (string, error) {
	perf.TmuxCalls.Add(1)
	cmd := exec.Command("tmux", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

// Launch starts a new tmux session running the default "claude" command.
func (t *TmuxBackend) Launch(ctx context.Context, taskID, prompt string) (*session.Session, error) {
	args := append([]string{}, t.extraArgs...)
	return t.LaunchCommand(ctx, taskID, "claude", args, prompt)
}

// LaunchCommand starts a new tmux session with a specific command.
func (t *TmuxBackend) LaunchCommand(ctx context.Context, workDir, command string, args []string, prompt string) (*session.Session, error) {
	id := uuid.New().String()
	shortID := id[:8]
	tmuxName := t.tmuxSessionName(shortID)

	// Ensure sessions directory exists (for prompt files)
	if err := os.MkdirAll(t.sessionsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}

	// Build the command to run inside tmux
	cmdArgs := append([]string{}, args...)

	var promptFile string
	if prompt != "" {
		promptFile = filepath.Join(t.sessionsDir, fmt.Sprintf("%s-prompt.md", id))
		if err := os.WriteFile(promptFile, []byte(prompt), 0o600); err != nil {
			return nil, fmt.Errorf("write prompt file: %w", err)
		}
		cmdArgs = append(cmdArgs, "--system-prompt-file", promptFile)
	}

	// Build the shell command string for tmux.
	// We use exec form to avoid shell interpretation issues.
	shellCmd := shellQuote(command, cmdArgs...)

	// Create a new tmux session running the command.
	// -d: don't attach
	// -s: session name
	// -x/-y: initial size (will be resized by the app)
	tmuxArgs := []string{
		"new-session", "-d",
		"-s", tmuxName,
		"-x", "80", "-y", "24",
	}

	// Set working directory
	if info, err := os.Stat(workDir); err == nil && info.IsDir() {
		tmuxArgs = append(tmuxArgs, "-c", workDir)
	}

	// Set environment and history limit
	tmuxArgs = append(tmuxArgs, "-E") // don't read .tmux.conf

	// The command to run
	tmuxArgs = append(tmuxArgs, shellCmd)

	if _, err := runTmux(tmuxArgs...); err != nil {
		if promptFile != "" {
			os.Remove(promptFile)
		}
		return nil, fmt.Errorf("create tmux session %q: %w", tmuxName, err)
	}

	// Set tmux options for this session
	runTmux("set-option", "-t", tmuxName, "history-limit", "5000")   //nolint:errcheck // best-effort session config
	runTmux("set-option", "-t", tmuxName, "allow-passthrough", "on") //nolint:errcheck // best-effort session config
	runTmux("set-option", "-t", tmuxName, "set-titles", "on")        //nolint:errcheck // best-effort session config
	runTmux("set-option", "-t", tmuxName, "remain-on-exit", "on")    //nolint:errcheck // best-effort session config

	tmuxSess := &TmuxSession{
		id:       id,
		tmuxName: tmuxName,
		workDir:  workDir,
		done:     make(chan struct{}),
	}

	// Start a goroutine to detect when the pane's command exits
	go tmuxSess.watchLoop()

	t.mu.Lock()
	t.sessions[id] = tmuxSess
	t.mu.Unlock()

	return &session.Session{
		ID:            id,
		TaskID:        workDir,
		BackendHandle: id,
		Status:        session.StatusRunning,
		Backend:       "tmux",
		StartedAt:     time.Now(),
	}, nil
}

// watchLoop polls tmux to detect when the pane's process has exited.
func (s *TmuxSession) watchLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			out, err := runTmux("display-message", "-t", s.tmuxName, "-p", "#{pane_dead} #{pane_dead_status}")
			if err != nil {
				// Session no longer exists
				s.mu.Lock()
				if !s.exited {
					s.exited = true
					s.exitCode = 1
					close(s.done)
				}
				s.mu.Unlock()
				return
			}

			parts := strings.SplitN(strings.TrimSpace(out), " ", 2)
			if len(parts) >= 1 && parts[0] == "1" {
				// Pane command has exited
				exitCode := 0
				if len(parts) >= 2 {
					_, _ = fmt.Sscanf(parts[1], "%d", &exitCode)
				}
				s.mu.Lock()
				if !s.exited {
					s.exited = true
					s.exitCode = exitCode
					close(s.done)
				}
				s.mu.Unlock()
				return
			}

			// Also update title while we're polling
			title, err := runTmux("display-message", "-t", s.tmuxName, "-p", "#{pane_title}")
			if err == nil {
				s.mu.Lock()
				s.title = strings.TrimSpace(title)
				s.mu.Unlock()
			}
		}
	}
}

// Attach is a no-op — the TUI renders via Capture/Render.
func (t *TmuxBackend) Attach(handle string) error {
	return nil
}

// AttachCmd returns a command to attach to the tmux session interactively.
func (t *TmuxBackend) AttachCmd(handle string) (*exec.Cmd, error) {
	sess := t.getSession(handle)
	if sess == nil {
		return nil, fmt.Errorf("session not found: %s", handle)
	}
	return exec.Command("tmux", "attach-session", "-t", sess.tmuxName), nil
}

// Detach is a no-op.
func (t *TmuxBackend) Detach(handle string) error {
	return nil
}

// Capture returns the pane content including scrollback.
func (t *TmuxBackend) Capture(handle string, lines int) (string, error) {
	sess := t.getSession(handle)
	if sess == nil {
		return "", fmt.Errorf("session not found: %s", handle)
	}

	startLine := -lines
	if lines <= 0 {
		startLine = -1000
	}

	out, err := runTmux("capture-pane", "-e", "-p",
		"-S", fmt.Sprintf("%d", startLine),
		"-t", sess.tmuxName)
	if err != nil {
		return "", fmt.Errorf("capture pane: %w", err)
	}
	return out, nil
}

// Render returns the current terminal content with ANSI styles, including
// scrollback history. For tmux, this captures the pane content plus up to
// 5000 lines of scrollback so the UI can scroll through past output.
func (t *TmuxBackend) Render(handle string) (string, error) {
	sess := t.getSession(handle)
	if sess == nil {
		return "", fmt.Errorf("session not found: %s", handle)
	}

	out, err := runTmux("capture-pane", "-e", "-p", "-S", "-5000", "-t", sess.tmuxName)
	if err != nil {
		return "", fmt.Errorf("render pane: %w", err)
	}

	// Strip trailing empty lines that tmux emits when scrollback is shorter
	// than the requested range. Without this, the panel shows excess blank
	// space at the top.
	out = strings.TrimRight(out, "\n")

	return out, nil
}

// Write sends raw bytes to the tmux pane using hex-encoded send-keys.
func (t *TmuxBackend) Write(handle string, data []byte) error {
	sess := t.getSession(handle)
	if sess == nil {
		return fmt.Errorf("session not found: %s", handle)
	}

	if len(data) == 0 {
		return nil
	}

	// Convert bytes to hex pairs for tmux send-keys -H
	args := []string{"send-keys", "-H", "-t", sess.tmuxName}
	for _, b := range data {
		args = append(args, fmt.Sprintf("%02x", b))
	}

	_, err := runTmux(args...)
	return err
}

// Resize changes the tmux pane dimensions.
func (t *TmuxBackend) Resize(handle string, rows, cols uint16) error {
	sess := t.getSession(handle)
	if sess == nil {
		return fmt.Errorf("session not found: %s", handle)
	}

	// Resize the tmux window (which resizes all panes in it)
	_, err := runTmux("resize-window", "-t", sess.tmuxName,
		"-x", fmt.Sprintf("%d", cols),
		"-y", fmt.Sprintf("%d", rows))
	return err
}

// Kill terminates the tmux session.
func (t *TmuxBackend) Kill(handle string) error {
	sess := t.getSession(handle)
	if sess == nil {
		return fmt.Errorf("session not found: %s", handle)
	}

	// Kill the tmux session
	_, _ = runTmux("kill-session", "-t", sess.tmuxName)

	// Mark as exited and close done channel
	sess.mu.Lock()
	if !sess.exited {
		sess.exited = true
		sess.exitCode = -1
		close(sess.done)
	}
	sess.mu.Unlock()

	t.removeSession(handle)
	return nil
}

// List returns all active sessions managed by this backend.
func (t *TmuxBackend) List() ([]*session.Session, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var sessions []*session.Session
	for _, s := range t.sessions {
		status := session.StatusRunning
		s.mu.Lock()
		if s.exited {
			status = session.StatusCompleted
		}
		s.mu.Unlock()

		sessions = append(sessions, &session.Session{
			ID:            s.id,
			TaskID:        s.workDir,
			BackendHandle: s.id,
			Status:        status,
			Backend:       "tmux",
		})
	}
	return sessions, nil
}

// Status returns the status of a tmux session.
func (t *TmuxBackend) Status(handle string) (session.Status, error) {
	sess := t.getSession(handle)
	if sess == nil {
		return session.StatusCompleted, nil
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.exited {
		return session.StatusCompleted, nil
	}
	return session.StatusRunning, nil
}

// GetTitle returns the pane title.
func (t *TmuxBackend) GetTitle(handle string) string {
	sess := t.getSession(handle)
	if sess == nil {
		return ""
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.title
}

// GetScrollback returns nil — tmux manages its own scrollback and it's
// included in the Render() output when needed via Capture().
func (t *TmuxBackend) GetScrollback(handle string) *session.ScrollbackBuffer {
	return nil
}

// GetExitCode returns the exit code of a completed session.
func (t *TmuxBackend) GetExitCode(handle string) (int, error) {
	sess := t.getSession(handle)
	if sess == nil {
		return 0, fmt.Errorf("session not found: %s", handle)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if !sess.exited {
		return 0, fmt.Errorf("session still running")
	}
	return sess.exitCode, nil
}

// GetDoneChannel returns a channel that closes when the session exits.
func (t *TmuxBackend) GetDoneChannel(handle string) (<-chan struct{}, error) {
	sess := t.getSession(handle)
	if sess == nil {
		return nil, fmt.Errorf("session not found: %s", handle)
	}
	return sess.done, nil
}

// ListTmuxSessions returns all tmux sessions matching this backend's prefix.
// This is used for session recovery — finding sessions that survived a restart.
func (t *TmuxBackend) ListTmuxSessions() ([]string, error) {
	out, err := runTmux("list-sessions", "-F", "#{session_name}")
	if err != nil {
		// No tmux server running or no sessions
		return nil, nil
	}

	prefix := t.tmuxSessionPrefix()
	var matching []string
	for _, name := range strings.Split(out, "\n") {
		name = strings.TrimSpace(name)
		if strings.HasPrefix(name, prefix) {
			matching = append(matching, name)
		}
	}
	return matching, nil
}

// ReattachSession creates a TmuxSession for an existing tmux session.
// Used during session recovery to reconnect to surviving sessions.
func (t *TmuxBackend) ReattachSession(tmuxName, sessionID, workDir string) (*session.Session, error) {
	// Verify the tmux session actually exists
	_, err := runTmux("has-session", "-t", tmuxName)
	if err != nil {
		return nil, fmt.Errorf("tmux session %q does not exist", tmuxName)
	}

	tmuxSess := &TmuxSession{
		id:       sessionID,
		tmuxName: tmuxName,
		workDir:  workDir,
		done:     make(chan struct{}),
	}

	// Check if already exited
	out, _ := runTmux("display-message", "-t", tmuxName, "-p", "#{pane_dead} #{pane_dead_status}")
	parts := strings.SplitN(strings.TrimSpace(out), " ", 2)

	status := session.StatusRunning
	if len(parts) >= 1 && parts[0] == "1" {
		tmuxSess.exited = true
		if len(parts) >= 2 {
			_, _ = fmt.Sscanf(parts[1], "%d", &tmuxSess.exitCode)
		}
		close(tmuxSess.done)
		status = session.StatusCompleted
	} else {
		// Start watching for exit
		go tmuxSess.watchLoop()
	}

	// Get current title
	title, _ := runTmux("display-message", "-t", tmuxName, "-p", "#{pane_title}")
	tmuxSess.title = strings.TrimSpace(title)

	t.mu.Lock()
	t.sessions[sessionID] = tmuxSess
	t.mu.Unlock()

	return &session.Session{
		ID:            sessionID,
		TaskID:        workDir,
		BackendHandle: sessionID,
		Status:        status,
		Backend:       "tmux",
		StartedAt:     time.Now(),
	}, nil
}

// GetTmuxSessionName returns the tmux session name for a given session handle.
func (t *TmuxBackend) GetTmuxSessionName(handle string) string {
	sess := t.getSession(handle)
	if sess == nil {
		return ""
	}
	return sess.tmuxName
}

// getSession returns a session by handle.
func (t *TmuxBackend) getSession(handle string) *TmuxSession {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.sessions[handle]
}

// removeSession removes a session from the map.
func (t *TmuxBackend) removeSession(handle string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.sessions, handle)
}

// shellQuote builds a shell command string safe for tmux new-session.
func shellQuote(command string, args ...string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, quoteArg(command))
	for _, arg := range args {
		parts = append(parts, quoteArg(arg))
	}
	return strings.Join(parts, " ")
}

// quoteArg wraps an argument in single quotes for shell safety.
func quoteArg(s string) string {
	// If no special chars, return as-is
	if !strings.ContainsAny(s, " \t\n'\"\\$`!#&|;(){}[]<>?*~") {
		return s
	}
	// Single-quote the string, escaping embedded single quotes
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
