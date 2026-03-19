# Fix "too many open files" Error

**Date:** 2026-03-19

## Summary

Fixed file descriptor exhaustion that caused `bubbletea: error creating cancel reader: create kqueue: too many open files` on startup, particularly in projects with large directory trees (`.git`, `node_modules`).

## Changes

### Bug Fixes
- **Workspace watcher directory filtering**: The `Watch()` function was adding *every* directory (including `.git`, `node_modules`, `vendor`, etc.) to the fsnotify watcher, consuming one kqueue fd per directory. Added filtering to skip hidden directories and known heavy directories (`node_modules`, `vendor`, `build`, `dist`, `.next`, `__pycache__`). Same filter applied to runtime directory creation events.
- **PTY master fd leak on natural exit**: When a child process exited naturally, the PTY master file descriptor was never closed (only `Kill()` closed it). Added `master.Close()` in `waitLoop()` so readLoop/responseLoop goroutines unblock and exit.
- **ClaudeAgent pipe leak on error paths**: In `Stream()`, if stderr/stdin pipe creation or `cmd.Start()` failed after stdout pipe was already created, the earlier pipes leaked. Added explicit cleanup of previously-opened pipes on each error path.
- **App resource cleanup on exit**: The SQLite store and workspace file watcher were never closed when the app exited. Added `app.Close()` method and call it from `main.go` after `tea.Program.Run()` returns.

## Files Modified
- `internal/workspace/workspace.go` - Filter hidden/heavy dirs from watcher walk; apply same filter to runtime dir creation
- `internal/app/app.go` - Add `Close()` method to release store and watcher resources
- `cmd/planck/main.go` - Call `application.Close()` on both normal and error exit paths
- `internal/session/pty.go` - Close PTY master fd in `waitLoop()` after child process exits
- `internal/agent/claude.go` - Close pipes on error paths in `Stream()`

## Rationale

The macOS default soft fd limit is 256. A typical project with `.git` (50+ subdirs) and `node_modules` (500+ subdirs) would exhaust this limit during workspace watcher initialization, before Bubble Tea could create its kqueue-based input reader. The remaining fixes prevent fd accumulation over longer sessions.
