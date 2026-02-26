# Session Persistence: Survive SSH Disconnects and Laptop Closes

**Date:** 2026-02-25

## Summary

Added tmux-based session persistence so agent sessions survive SSH disconnects, laptop closes, and planck restarts. Sessions are automatically recovered when planck is restarted in the same project folder.

## Changes

### Features
- Tmux backend for agent sessions — each agent runs in its own tmux session, surviving planck exit
- Automatic backend selection: tmux preferred when available, PTY fallback otherwise
- Session state persistence to SQLite — tab configuration, titles, and tmux session mapping
- Automatic session recovery on startup — detects surviving tmux sessions and restores tabs
- `planck attach` command — runs planck inside a persistent tmux session for easy SSH reconnection
- Per-folder session isolation — sessions in `~/project-a` are completely separate from `~/project-b`

### Architecture
- New `InteractiveBackend` interface in `session/session.go` — unifies PTY and tmux backends
- New `internal/tmux/backend.go` — full `TmuxBackend` implementation with Launch, Render, Write, Resize, Kill, Status, and session recovery
- New `internal/app/recovery.go` — session persistence helpers and startup recovery logic
- Updated factory to support `auto`/`tmux`/`pty` backend selection from config

### Database
- Extended `sessions` table with columns: `agent_key`, `agent_label`, `custom_title`, `tmux_session_name`, `backend_type`, `work_dir`, `command`, `args`
- Automatic schema migration — existing databases get new columns without data loss

## Files Modified
- `cmd/planck/main.go` — factory-based backend creation, `planck attach` subcommand, updated help text
- `internal/app/app.go` — `InteractiveBackend` interface usage, persistence hooks, conditional kill behavior
- `internal/app/recovery.go` — new file: session persistence, recovery, and cleanup logic
- `internal/session/session.go` — new `InteractiveBackend` interface definition
- `internal/session/factory.go` — tmux/pty/auto backend selection logic
- `internal/store/store.go` — extended schema, migration, new query methods
- `internal/store/store_test.go` — tests for extended fields, migration, title updates, args encoding
- `internal/tmux/backend.go` — new file: full TmuxBackend implementation
- `internal/tmux/backend_test.go` — new file: unit + integration tests for tmux backend

## Rationale

The core use case is SSH-based workflows: users start agents, disconnect, and reconnect later to see results. Tmux is the natural solution — it's designed for exactly this, is widely available on servers, and the codebase already had scaffolding for it. The hybrid approach (tmux + PTY fallback) ensures the feature works transparently when tmux is available without breaking existing PTY-only usage.
