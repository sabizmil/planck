# Planck - Claude Code Project Rules

## Project Overview

Planck is a terminal UI (TUI) application for AI-assisted planning and task management. It orchestrates multiple Claude Code agent sessions via tmux, providing a structured workflow for breaking down complex tasks.

- **Language:** Go 1.24+
- **Module:** `github.com/anthropics/planck`
- **Entry point:** `cmd/planck/main.go`
- **Storage:** SQLite (via `modernc.org/sqlite`)
- **TUI framework:** Bubble Tea (`charmbracelet/bubbletea`)

## Build & Test Commands

```bash
make build          # Build binary to build/planck
make test           # Run all tests with race detector
make test-short     # Run short tests (no integration)
make test-coverage  # Tests with coverage report
make lint           # Run golangci-lint
make fmt            # Format code (go fmt + goimports)
make run            # Build and run
make dev            # Hot reload with air
```

## Plan Lifecycle

Plans are ephemeral working documents. They live outside git and are archived locally when completed.

### Location & Naming

- **Active plans:** `.claude/plans/YYYY-MM-DD-<slug>.md`
- **Archived plans:** `.claude/plans/archive/YYYY-MM-DD-<slug>.md`
- Both directories are gitignored.

### Plan Format

```markdown
# Plan: <Title>

**Date:** YYYY-MM-DD
**Status:** created | in-progress | completed

## Goal
What this plan aims to accomplish.

## Approach
The chosen implementation strategy and rationale.

## Tasks
- [ ] Task 1
- [ ] Task 2

## Notes
Any discoveries, decisions, or context gathered during implementation.
```

### Lifecycle Flow

1. **Create** a plan file in `.claude/plans/` with status `created`
2. **Work** on it, updating status to `in-progress` and checking off tasks
3. **Complete** it when all tasks are done:
   - Set status to `completed`
   - Write a changelog entry to `docs/changelog/`
   - Move the plan file to `.claude/plans/archive/`

## Changelog Workflow

Changelogs are the permanent record of what changed and why. They are tracked in git.

### Location & Naming

- **Directory:** `docs/changelog/`
- **Naming:** `YYYY-MM-DD-<slug>.md` (matching the plan that produced it)

### Changelog Format

```markdown
# <Title>

**Date:** YYYY-MM-DD

## Summary
One or two sentences describing what changed.

## Changes

### <Category> (e.g., Features, Bug Fixes, Refactoring)
- Change 1
- Change 2

## Files Modified
- `path/to/file.go` - description of change

## Rationale
Why these changes were made.
```

### When to Write Changelogs

- When a plan is completed and its changes are committed
- For significant bug fixes or architectural changes (even without a formal plan)
- NOT for minor formatting, typo fixes, or trivial changes

## Development Rules

### Go Conventions

- Follow standard Go idioms and `go fmt` formatting
- Use table-driven tests
- Error handling: return errors, don't panic (except for truly unrecoverable situations)
- Naming: use Go conventions (camelCase for unexported, PascalCase for exported)
- Keep functions focused and small; prefer composition over inheritance

### Testing

- All new features should have tests
- Use `testutil/` helpers for common test setup
- Integration tests go in `e2e/`
- Use `-short` flag to skip slow tests: `if testing.Short() { t.Skip() }`

### Project Structure

```
cmd/planck/        # Application entry point
internal/          # Private application code
  agent/           # Agent session management
  config/          # Configuration
  db/              # Database layer (SQLite)
  model/           # Bubble Tea models (TUI)
  planner/         # Planning logic
  session/         # Session management
  tmux/            # Tmux integration
e2e/               # End-to-end tests
testutil/          # Test utilities
docs/changelog/    # Tracked changelogs (in git)
.claude/plans/     # Working plans (not in git)
```

### Planning Before Coding

Before writing code for any non-trivial task:
1. Consider 3-5 different approaches
2. Evaluate trade-offs (performance, maintainability, complexity)
3. Select the best approach and document rationale
4. Then implement
