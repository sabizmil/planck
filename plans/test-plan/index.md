# Embed Claude Code in TUI

> Audit the existing app and clean up the Claude Code experience so it can be embedded within the TUI while retaining the sidebar with notification updates.

Status: active
Created: 2026-02-20
Updated: 2026-02-20

## Goal

Refactor Planck's Claude Code integration so that interactive Claude sessions run **inside** the Bubble Tea TUI rather than being dispatched to external tmux sessions. The sidebar with plan/session lists and notification updates must remain visible and functional at all times, giving users a unified, single-pane-of-glass experience.

## Current State

- **Planning sessions** run headless via `claude -p --output-format stream-json` with streaming JSON parsed in `session_panel.go`
- **Implementation sessions** spawn interactive tmux sessions (`tmux new-session -d -s planck-{id} "claude ..."`) — the user leaves the TUI to interact
- **Autonomous execution** runs headless with `--dangerously-skip-permissions`
- Sidebar (`sidebar.go`) shows plans + session status but can't show live Claude output inline
- Notification system (`notify/`) fires terminal bells and tracks event history

## Key Challenges

1. Claude Code is a full interactive CLI — embedding it requires PTY management or protocol-level integration
2. The TUI must handle raw terminal I/O from Claude alongside Bubble Tea's event loop
3. Sidebar and notification updates must remain responsive during embedded sessions
4. Session hijacking, background/foreground switching must still work

## Approaches

See below for 3 distinct approaches to achieving this goal.
