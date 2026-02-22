# Settings Panel, Markdown Themes, and UX Improvements

**Date:** 2026-02-21

## Summary

Added a settings panel with customizable markdown rendering themes, reverse tab navigation, automatic tab naming via OSC escape sequences, file completion toggling, and a bug fix for agent tab creation after dialog selection.

## Changes

### Features

- **Settings panel** (`s` key): Full-screen overlay for customizing markdown rendering with live preview. Supports 5 themes (Neo-Brutalist, Terminal Classic, Minimal Modern, Rich Editorial, Soft Pastel) and per-element overrides. Style choices persist to `config.toml`.
- **Markdown style engine**: Element-level style registry that composes glamour JSON from per-element theme selections. Supports mix-and-match across themes with `NO_COLOR` compatibility.
- **Reverse tab navigation** (`Shift+Tab`): Cycle backwards through tabs. Parameterized `cycleTab(direction int)` to handle both directions.
- **Auto tab naming**: Agent tabs automatically pick up titles from child process OSC 0/2 escape sequences. Titles are sanitized and truncated to 30 characters.
- **File completion toggle** (`c` key): Toggle plan file status between completed/pending directly from the file list. Updates YAML frontmatter in-place. Completed files render with dimmed text.
- **Codex agent config**: Added `codex` as a built-in agent with `--full-auto` flag support.

### Bug Fixes

- **Agent tab refresh after dialog**: Fixed execution order bug where selecting an agent from the dialog didn't immediately render the new tab. The `pendingAgentKey` check now runs inside the dialog block after `Dialog.Update()` returns.
- **PTY polling continuity**: Moved PTY message handling before overlay checks so polling chains aren't interrupted by dialogs, help, or folder picker.
- **Overlay command batching**: All overlay handlers (dialog, help, settings, folder picker) now batch commands instead of discarding accumulated PTY cmds on early return.

## Files Modified

- `internal/app/app.go` - Integrated settings panel, reverse tab nav, auto tab naming, file completion toggle, agent refresh fix, PTY message reordering
- `internal/config/config.go` - Added `MarkdownStyle` config section, codex agent default
- `internal/session/pty.go` - OSC title callback in VTE, `GetTitle()` method
- `internal/ui/editor.go` - `SetMarkdownStyle()` for hot-swapping renderer, style-aware resize
- `internal/ui/filelist.go` - Dimmed style for completed file names
- `internal/ui/help.go` - Updated keybinding docs for Shift+Tab, settings, file completion
- `internal/ui/pty_panel.go` - Shift+Tab passthrough filter, `Title` field in `PTYRenderMsg`
- `internal/ui/markdown_style.go` - **New**: Style registry, 5 theme definitions, composition engine
- `internal/ui/settings.go` - **New**: Settings panel overlay with three-column layout and live preview
- `internal/workspace/workspace.go` - `ToggleFileStatus()` and `setFrontmatterStatus()` for frontmatter manipulation
- `internal/workspace/workspace_test.go` - **New**: Table-driven tests for frontmatter status toggle

## Rationale

These changes significantly improve the daily-use ergonomics of planck: settings are now discoverable and interactive instead of requiring manual config editing, tab navigation scales to many tabs, and plan files can be marked complete without opening the editor. The auto tab naming leverages the existing VTE infrastructure with zero external dependencies.
