# README.md Comprehensive Audit and Update

**Date:** 2026-02-25

## Summary
Audited every section of README.md against the actual codebase and fixed critical errors, missing features, and outdated information. Also updated the `printHelp()` CLI help text to match.

## Changes

### Bug Fixes (Documentation)
- Fixed Quick Start: `t` key changed to `a` for new agent tab (no `t` binding exists)
- Fixed Quick Start: `planck /path/to/project` changed to `planck -f /path/to/project` (positional args are rejected)
- Fixed Agent Tab keybinding: `Esc` changed to `Ctrl+\` for exiting input mode
- Fixed Global keybinding: removed `Tab` as tab-cycling key (it passes through to PTY for autocomplete); `Shift+Tab` cycles forward only
- Fixed spinner count: 26 → 27 (breathe preset was missing)
- Fixed architecture section: added missing `internal/updater/` package, fixed `cmd/planck/` description

### Features (Documentation)
- Added `Session Persistence` feature section documenting tmux backend and `planck attach`
- Added `Alt+1-9` tab switching to Global keybindings
- Added `Ctrl+X` close tab to Global keybindings
- Added `e` (edit mode), `m` (move/rename), `r` (refresh) to File List keybindings
- Added `d` now works on folders (not just files)
- Added mouse support notes to every keybinding section (click tabs, click files, scroll sidebar, drag-select editor text)
- Split Agent Tab keybindings into Input Mode and Normal Mode tables
- Added `Tab` passthrough for autocomplete in Agent Tab Input Mode
- Added `sidebar_width` to config.toml example
- Added `--check` flag documentation for `planck update` and `planck version`
- Added `-f` short flag to Quick Start examples
- Updated settings page list to match actual code order (Markdown, General, Agents, Keys, Spinner)
- Listed all 27 spinner preset names

### Refactoring
- Updated `printHelp()` in main.go to match new README keybindings (added Shift+Tab, Alt+1-9, removed bare Tab, added m/r/c/h/l keys, added j/k/g/G scrolling for agent normal mode)

## Files Modified
- `README.md` — comprehensive audit and rewrite of all sections
- `cmd/planck/main.go` — updated `printHelp()` keybindings section to match actual behavior

## Rationale
The README had accumulated significant drift from the codebase after 20+ changelogs of new features (2026-02-21 through 2026-02-25). Four critical keybinding errors could confuse new users, and major features like session persistence, mouse support, and interactive move mode were entirely undocumented.
