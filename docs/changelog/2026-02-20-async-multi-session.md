# Async Multi-Session Workflow - 2026-02-20

## Summary

Added comprehensive support for asynchronous, multi-session workflows including fire-and-forget planning, autonomous execution, and session hijacking.

## Changes

### New Features

1. **Session Modes: Foreground vs Background**
   - Sessions can run in foreground (user watches) or background (async)
   - `B` keybinding: Send current session to background
   - `F` keybinding: Bring background session to foreground
   - New status indicator: `◐` for background sessions

2. **Notification System**
   - Terminal bell (`\a`) on session events
   - Events: planning complete, task complete, phase complete, error
   - Configurable via `[notifications]` in config.toml

3. **Autonomous Execution (`X` keybinding)**
   - Scope picker dialog: task / phase / entire plan
   - Permission pre-approval dialog
   - Headless execution with `--dangerously-skip-permissions`
   - Progress tracking: `Phase 2/3 • Task 4/7`
   - Real-time plan tree updates

4. **Session Hijacking**
   - Press `Enter` during autonomous execution to take control
   - Spawns tmux session, attaches to running Claude Code
   - Resume prompt on detach: "Resume autonomous? [Y/n]"

5. **Progress Display**
   - Aggregate progress in status bar
   - Phase/task rollup view during execution
   - New status indicators: `▶` (executing), `⏸` (paused)

### New Config Options

```toml
[notifications]
bell = true

[execution]
default_scope = "phase"
auto_advance = true
permission_mode = "pre-approve"
```

### New Database Tables

- `execution_runs`: Tracks autonomous execution state (scope, progress, status)
- Updated `sessions` table: Added `session_mode` column (foreground/background)

### New Go Packages

- `internal/execution/`: Autonomous execution orchestration
- `internal/notify/`: Notification system (terminal bell)

### New UI Components

- `execution_view.go`: Autonomous execution progress view
- `scope_picker.go`: Scope selection dialog
- `permission_dialog.go`: Permission pre-approval dialog
- `notification_panel.go`: Notification history view

### New Workflows

- **Workflow 6**: Autonomous execution
- **Workflow 7**: Fire-and-forget planning
- **Workflow 8**: Multi-session parallel work

### New Keybindings

| Key | Action |
|-----|--------|
| `X` | Execute plan (shows scope picker) |
| `B` | Background current session |
| `F` | Foreground a background session |
| `N` | Show notification history |

### MVP Scope Update

- Added Phase 3.5: Async multi-session workflow

## Rationale

This update enables a more flexible, async-first workflow where users can:
- Fire off vague planning requests and come back when ready
- Run multiple plans simultaneously
- Execute entire plans hands-off with one approval
- Intervene at any point during autonomous execution

## Files Modified

- `planck-spec.md` - All specification updates

## Documentation Structure Created

Comprehensive phase-based documentation with build/test processes:

```
docs/PRD/
├── README.md                        # PRD index and overview
├── phase-1-plan-viewer.md           # Core TUI, parsing, navigation
├── phase-2-planning-sessions.md     # AI planning, streaming, accept/reject
├── phase-3-implementation-dispatch.md # tmux, sessions, dispatch
├── phase-3.5-async-multi-session.md # Background, execution, notifications
├── phase-4-polish.md                # Codex, templates, command mode
├── phase-5-validation.md            # Testing, performance, release
├── build-process.md                 # Makefile, CI/CD, releases
├── testing-strategy.md              # Unit, integration, E2E testing
└── architecture.md                  # System design, data flow
```
