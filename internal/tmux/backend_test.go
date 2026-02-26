package tmux

import (
	"os/exec"
	"testing"
)

func TestTmuxSessionName(t *testing.T) {
	backend := NewTmuxBackend("planck", "/tmp/sessions", "/home/user/project", nil)

	name := backend.tmuxSessionName("abcd1234")
	prefix := backend.tmuxSessionPrefix()

	// Name should start with prefix
	if len(name) <= len(prefix) {
		t.Errorf("session name %q should be longer than prefix %q", name, prefix)
	}
	if name[:len(prefix)] != prefix {
		t.Errorf("session name %q should start with prefix %q", name, prefix)
	}

	// Prefix should include workdir hash
	if prefix == "planck-" {
		t.Error("prefix should include workdir hash, not just 'planck-'")
	}
}

func TestWorkDirIsolation(t *testing.T) {
	b1 := NewTmuxBackend("planck", "/tmp/sessions", "/home/user/project-a", nil)
	b2 := NewTmuxBackend("planck", "/tmp/sessions", "/home/user/project-b", nil)

	if b1.workDirHash == b2.workDirHash {
		t.Error("different work dirs should produce different hashes")
	}

	prefix1 := b1.tmuxSessionPrefix()
	prefix2 := b2.tmuxSessionPrefix()

	if prefix1 == prefix2 {
		t.Errorf("prefixes should differ: %q vs %q", prefix1, prefix2)
	}
}

func TestSameWorkDirSameHash(t *testing.T) {
	b1 := NewTmuxBackend("planck", "/tmp/sessions", "/home/user/project", nil)
	b2 := NewTmuxBackend("planck", "/tmp/sessions", "/home/user/project", nil)

	if b1.workDirHash != b2.workDirHash {
		t.Errorf("same work dir should produce same hash: %q vs %q", b1.workDirHash, b2.workDirHash)
	}
}

func TestIsAvailable(t *testing.T) {
	backend := NewTmuxBackend("planck", "/tmp/sessions", "/tmp", nil)

	// This test checks that IsAvailable doesn't panic.
	// On CI without tmux, it returns false; locally it may return true.
	_ = backend.IsAvailable()
}

func TestName(t *testing.T) {
	backend := NewTmuxBackend("planck", "/tmp/sessions", "/tmp", nil)
	if got := backend.Name(); got != "tmux" {
		t.Errorf("Name() = %q, want %q", got, "tmux")
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		command string
		args    []string
		want    string
	}{
		{"claude", nil, "claude"},
		{"claude", []string{"--verbose"}, "claude --verbose"},
		{"claude", []string{"--system-prompt-file", "/path/to/file.md"}, "claude --system-prompt-file /path/to/file.md"},
		{"claude", []string{"--arg", "value with spaces"}, "claude --arg 'value with spaces'"},
		{"claude", []string{"it's"}, "claude 'it'\"'\"'s'"},
	}

	for _, tt := range tests {
		got := shellQuote(tt.command, tt.args...)
		if got != tt.want {
			t.Errorf("shellQuote(%q, %v) = %q, want %q", tt.command, tt.args, got, tt.want)
		}
	}
}

func TestQuoteArg(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with space", "'with space'"},
		{"with'quote", "'with'\"'\"'quote'"},
		{"--flag=value", "--flag=value"},
		{"$variable", "'$variable'"},
	}

	for _, tt := range tests {
		got := quoteArg(tt.input)
		if got != tt.want {
			t.Errorf("quoteArg(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// Integration tests that require tmux to be installed.
// Skipped in short mode and when tmux is not available.

func tmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func TestLaunchAndKill(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	dir := t.TempDir()
	backend := NewTmuxBackend("planck-test", dir, dir, nil)

	// Launch a simple command
	sess, err := backend.LaunchCommand(t.Context(), dir, "sleep", []string{"30"}, "")
	if err != nil {
		t.Fatalf("LaunchCommand() error = %v", err)
	}

	// Verify session is running
	status, err := backend.Status(sess.BackendHandle)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != "running" {
		t.Errorf("Status() = %v, want running", status)
	}

	// Verify we can capture output
	_, err = backend.Render(sess.BackendHandle)
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	// Verify title works (may be empty for sleep)
	_ = backend.GetTitle(sess.BackendHandle)

	// Verify scrollback returns nil
	if sb := backend.GetScrollback(sess.BackendHandle); sb != nil {
		t.Error("GetScrollback() should return nil for tmux backend")
	}

	// Kill the session
	if err := backend.Kill(sess.BackendHandle); err != nil {
		t.Fatalf("Kill() error = %v", err)
	}

	// Verify session is gone
	status, _ = backend.Status(sess.BackendHandle)
	if status != "completed" {
		t.Errorf("Status() after kill = %v, want completed", status)
	}
}

func TestWriteAndCapture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	dir := t.TempDir()
	backend := NewTmuxBackend("planck-test", dir, dir, nil)

	// Launch bash
	sess, err := backend.LaunchCommand(t.Context(), dir, "bash", nil, "")
	if err != nil {
		t.Fatalf("LaunchCommand() error = %v", err)
	}
	defer backend.Kill(sess.BackendHandle) //nolint:errcheck

	// Write a command
	err = backend.Write(sess.BackendHandle, []byte("echo hello-planck-test\r"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Give it a moment to process
	// tmux needs time to handle the command
	for range 20 {
		output, err := backend.Capture(sess.BackendHandle, 50)
		if err != nil {
			t.Fatalf("Capture() error = %v", err)
		}
		if len(output) > 0 && contains(output, "hello-planck-test") {
			return // success
		}
		// wait a bit and retry
		// We can't use time.Sleep in tests ideally, but tmux needs real time
		exec.Command("sleep", "0.1").Run() //nolint:errcheck
	}

	t.Error("expected to find 'hello-planck-test' in captured output")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestReattachSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	dir := t.TempDir()
	backend := NewTmuxBackend("planck-test", dir, dir, nil)

	// Launch a session
	sess, err := backend.LaunchCommand(t.Context(), dir, "sleep", []string{"30"}, "")
	if err != nil {
		t.Fatalf("LaunchCommand() error = %v", err)
	}

	tmuxName := backend.GetTmuxSessionName(sess.BackendHandle)
	if tmuxName == "" {
		t.Fatal("GetTmuxSessionName() returned empty string")
	}

	// Simulate planck restart: create a new backend and reattach
	backend2 := NewTmuxBackend("planck-test", dir, dir, nil)
	sess2, err := backend2.ReattachSession(tmuxName, sess.ID, dir)
	if err != nil {
		t.Fatalf("ReattachSession() error = %v", err)
	}

	if sess2.Status != "running" {
		t.Errorf("reattached session Status = %v, want running", sess2.Status)
	}

	// Kill via the new backend
	if err := backend2.Kill(sess2.BackendHandle); err != nil {
		t.Fatalf("Kill() error = %v", err)
	}
}
