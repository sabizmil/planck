package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
	"github.com/google/uuid"
)

// PTYSession represents an active PTY session
type PTYSession struct {
	id         string
	taskID     string
	master     *os.File     // PTY master fd (read/write)
	cmd        *exec.Cmd    // Child process
	terminal   *vt.Emulator // Virtual terminal state
	scrollback *ScrollbackBuffer
	mu         sync.Mutex
	done       chan struct{}
	exitCode   int
	exited     bool
	promptFile string // Temporary prompt file to clean up
	title      string // Window title set by child via OSC 0/2
}

// PTYBackend implements Backend using in-process PTY
type PTYBackend struct {
	prefix          string
	sessionsDir     string
	escapeKey       string
	scrollbackLines int
	extraArgs       []string
	mu              sync.Mutex
	sessions        map[string]*PTYSession
}

// NewPTYBackend creates a new PTY backend
func NewPTYBackend(prefix, sessionsDir string, extraArgs []string) *PTYBackend {
	return &PTYBackend{
		prefix:          prefix,
		sessionsDir:     sessionsDir,
		escapeKey:       `ctrl+\`,
		scrollbackLines: 1000,
		extraArgs:       extraArgs,
		sessions:        make(map[string]*PTYSession),
	}
}

// Name returns the backend name
func (p *PTYBackend) Name() string {
	return "pty"
}

// IsAvailable returns whether PTY backend is available (checks default "claude" command)
func (p *PTYBackend) IsAvailable() bool {
	return IsCommandAvailable("claude")
}

// IsCommandAvailable checks if a command exists in PATH
func IsCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// Launch starts a new PTY session using the default "claude" command.
// If prompt is empty, launches an interactive Claude session without a system prompt.
// taskID is used as the working directory for the session.
func (p *PTYBackend) Launch(ctx context.Context, taskID, prompt string) (*Session, error) {
	args := append([]string{}, p.extraArgs...)
	return p.LaunchCommand(ctx, taskID, "claude", args, prompt)
}

// LaunchCommand starts a new PTY session with a specified command and arguments.
// workDir is the working directory for the session.
// If prompt is non-empty, it's written to a temp file and passed via --system-prompt-file.
func (p *PTYBackend) LaunchCommand(ctx context.Context, workDir, command string, args []string, prompt string) (*Session, error) {
	id := uuid.New().String()

	// Ensure sessions directory exists
	if err := os.MkdirAll(p.sessionsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}

	var promptFile string

	cmdArgs := append([]string{}, args...)
	if prompt != "" {
		// Write prompt to temp file
		promptFile = filepath.Join(p.sessionsDir, fmt.Sprintf("%s-prompt.md", id))
		if err := os.WriteFile(promptFile, []byte(prompt), 0o600); err != nil {
			return nil, fmt.Errorf("write prompt file: %w", err)
		}
		cmdArgs = append(cmdArgs, "--system-prompt-file", promptFile)
	}
	cmd := exec.CommandContext(ctx, command, cmdArgs...)

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Set working directory
	if info, err := os.Stat(workDir); err == nil && info.IsDir() {
		cmd.Dir = workDir
	}

	// Start in PTY
	master, err := pty.Start(cmd)
	if err != nil {
		if promptFile != "" {
			os.Remove(promptFile)
		}
		return nil, fmt.Errorf("start PTY: %w", err)
	}

	// Set initial size (non-fatal if it fails)
	_ = pty.Setsize(master, &pty.Winsize{Rows: 24, Cols: 80})

	// Create virtual terminal emulator (default 80x24)
	terminal := vt.NewEmulator(80, 24)

	scrollback := NewScrollbackBuffer(p.scrollbackLines)

	ptySess := &PTYSession{
		id:         id,
		taskID:     workDir,
		master:     master,
		cmd:        cmd,
		terminal:   terminal,
		scrollback: scrollback,
		done:       make(chan struct{}),
		promptFile: promptFile,
	}

	terminal.SetCallbacks(vt.Callbacks{
		ScrollOff: func(lines []string, altScreen bool) {
			if !altScreen {
				scrollback.Push(lines)
			}
		},
		Title: func(title string) {
			// Called from terminal.Write() which is already under ptySess.mu
			ptySess.title = title
		},
	})

	// Background reader: PTY master → vt.Terminal
	go ptySess.readLoop()

	// Background writer: vt.Terminal responses → PTY master
	// This forwards mouse events, device attribute responses, etc.
	go ptySess.responseLoop()

	// Background waiter: detect child exit
	go ptySess.waitLoop()

	p.mu.Lock()
	p.sessions[id] = ptySess
	p.mu.Unlock()

	return &Session{
		ID:            id,
		TaskID:        workDir,
		BackendHandle: id,
		Status:        StatusRunning,
		Backend:       "pty",
		StartedAt:     time.Now(),
	}, nil
}

// readLoop reads from PTY master and writes to terminal
func (s *PTYSession) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.master.Read(buf)
		if n > 0 {
			s.mu.Lock()
			_, _ = s.terminal.Write(buf[:n])
			s.mu.Unlock()
		}
		if err != nil {
			// Close the emulator pipe to unblock responseLoop
			s.terminal.Close()
			return
		}
	}
}

// responseLoop reads responses from the virtual terminal (device attribute
// responses, etc.) and forwards them to the child PTY.
func (s *PTYSession) responseLoop() {
	buf := make([]byte, 256)
	for {
		n, err := s.terminal.Read(buf)
		if n > 0 {
			if _, werr := s.master.Write(buf[:n]); werr != nil {
				return // master closed
			}
		}
		if err != nil {
			return // emulator pipe closed
		}
	}
}

// waitLoop waits for the child process to exit
func (s *PTYSession) waitLoop() {
	err := s.cmd.Wait()
	s.mu.Lock()
	s.exited = true
	if err != nil {
		// Process exited with error
		if exitErr, ok := err.(*exec.ExitError); ok {
			s.exitCode = exitErr.ExitCode()
		} else {
			s.exitCode = 1
		}
	}
	s.mu.Unlock()
	close(s.done)

	// Clean up prompt file
	if s.promptFile != "" {
		os.Remove(s.promptFile)
	}
}

// getSession returns a PTY session by handle
func (p *PTYBackend) getSession(handle string) *PTYSession {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessions[handle]
}

// removeSession removes a session from the map
func (p *PTYBackend) removeSession(handle string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.sessions, handle)
}

// Attach is a no-op for PTY - the panel handles rendering
func (p *PTYBackend) Attach(handle string) error {
	return nil
}

// AttachCmd returns an error - PTY doesn't use external attach
func (p *PTYBackend) AttachCmd(handle string) (*exec.Cmd, error) {
	return nil, fmt.Errorf("PTY backend does not use external attach — use the embedded panel")
}

// Detach is a no-op for PTY
func (p *PTYBackend) Detach(handle string) error {
	return nil
}

// Render returns the terminal output with ANSI styles preserved
func (p *PTYBackend) Render(handle string) (string, error) {
	sess := p.getSession(handle)
	if sess == nil {
		return "", fmt.Errorf("session not found: %s", handle)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	return sess.terminal.Render(), nil
}

// Capture captures output from a PTY session
func (p *PTYBackend) Capture(handle string, lines int) (string, error) {
	sess := p.getSession(handle)
	if sess == nil {
		return "", fmt.Errorf("session not found: %s", handle)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	// Get current screen content
	return sess.terminal.String(), nil
}

// Kill kills a PTY session
func (p *PTYBackend) Kill(handle string) error {
	sess := p.getSession(handle)
	if sess == nil {
		return fmt.Errorf("session not found: %s", handle)
	}

	// Send SIGTERM, then SIGKILL after timeout
	if sess.cmd.Process != nil {
		_ = sess.cmd.Process.Signal(syscall.SIGTERM)
	}

	select {
	case <-sess.done:
		// Process exited cleanly
	case <-time.After(5 * time.Second):
		// Force kill
		if sess.cmd.Process != nil {
			_ = sess.cmd.Process.Kill()
		}
	}

	// Close master fd and emulator
	sess.master.Close()
	sess.terminal.Close() // unblock responseLoop if still running

	// Clean up prompt file
	if sess.promptFile != "" {
		os.Remove(sess.promptFile)
	}

	p.removeSession(handle)
	return nil
}

// List lists all PTY sessions
func (p *PTYBackend) List() ([]*Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var sessions []*Session
	for _, s := range p.sessions {
		status := StatusRunning
		s.mu.Lock()
		if s.exited {
			status = StatusCompleted
		}
		s.mu.Unlock()

		sessions = append(sessions, &Session{
			ID:            s.id,
			TaskID:        s.taskID,
			BackendHandle: s.id,
			Status:        status,
			Backend:       "pty",
		})
	}
	return sessions, nil
}

// Status returns the status of a PTY session
func (p *PTYBackend) Status(handle string) (Status, error) {
	sess := p.getSession(handle)
	if sess == nil {
		// Session not found means it already finished
		return StatusCompleted, nil
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.exited {
		return StatusCompleted, nil
	}
	return StatusRunning, nil
}

// Resize resizes a PTY session
func (p *PTYBackend) Resize(handle string, rows, cols uint16) error {
	sess := p.getSession(handle)
	if sess == nil {
		return fmt.Errorf("session not found: %s", handle)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	// Resize PTY
	if err := pty.Setsize(sess.master, &pty.Winsize{Rows: rows, Cols: cols}); err != nil {
		return fmt.Errorf("set PTY size: %w", err)
	}

	// Resize virtual terminal
	sess.terminal.Resize(int(cols), int(rows))

	// Signal child process
	if sess.cmd.Process != nil {
		_ = sess.cmd.Process.Signal(syscall.SIGWINCH)
	}

	return nil
}

// Write writes data to the PTY master
func (p *PTYBackend) Write(handle string, data []byte) error {
	sess := p.getSession(handle)
	if sess == nil {
		return fmt.Errorf("session not found: %s", handle)
	}

	_, err := sess.master.Write(data)
	return err
}

// GetTitle returns the window title set by the child process via OSC 0/2
func (p *PTYBackend) GetTitle(handle string) string {
	sess := p.getSession(handle)
	if sess == nil {
		return ""
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	return sess.title
}

// GetTerminal returns the terminal state for rendering
func (p *PTYBackend) GetTerminal(handle string) (*vt.Emulator, error) {
	sess := p.getSession(handle)
	if sess == nil {
		return nil, fmt.Errorf("session not found: %s", handle)
	}
	return sess.terminal, nil
}

// GetScrollback returns the scrollback buffer for a session
func (p *PTYBackend) GetScrollback(handle string) *ScrollbackBuffer {
	sess := p.getSession(handle)
	if sess == nil {
		return nil
	}
	return sess.scrollback
}

// GetDoneChannel returns the done channel for a session
func (p *PTYBackend) GetDoneChannel(handle string) (<-chan struct{}, error) {
	sess := p.getSession(handle)
	if sess == nil {
		return nil, fmt.Errorf("session not found: %s", handle)
	}
	return sess.done, nil
}

// GetExitCode returns the exit code of a completed session
func (p *PTYBackend) GetExitCode(handle string) (int, error) {
	sess := p.getSession(handle)
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
