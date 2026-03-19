# Switch Next-Tab Default from Shift+Tab to | (pipe)

**Date:** 2026-03-04

## Summary
Changed the default tab-cycling keybinding from `Shift+Tab` to `|` (pipe / Shift+\) to avoid conflicting with Claude Code's own `Shift+Tab` feature. The PTY panel now uses the keymap system instead of a hardcoded key check, so user-configured bindings are properly respected. `Shift+Tab` is now passed through to the PTY session.

## Changes

### Bug Fixes
- Resolved conflict between Planck's `Shift+Tab` tab cycling and Claude Code's `Shift+Tab` cycle feature
- `Shift+Tab` now passes through to the PTY so Claude Code can react to it

### Improvements
- PTY panel input-mode intercept is now keymap-aware instead of hardcoded to `tea.KeyShiftTab`
- Settings keybinding capture no longer blocks `Shift+Tab` from being bound (only `Tab` is blocked for widget navigation)
- Users can customize the next-tab binding via `[keybindings.global]` → `next_tab` in config

## Files Modified
- `internal/ui/keymap.go` - Changed default `next_tab` binding from `"shift+tab"` to `"|"`
- `internal/ui/pty_panel.go` - Replaced hardcoded `tea.KeyShiftTab` check with keymap lookup
- `internal/ui/settings_keybindings.go` - Removed `shift+tab` from the capture block list
- `internal/ui/pty_panel_test.go` - Renamed and updated test to use `|`
- `internal/ui/keymap_test.go` - Updated expected default binding in test
- `cmd/planck/main.go` - Updated help text
- `README.md` - Updated keybinding documentation

## Rationale
`Shift+Tab` is used by Claude Code for its own tab-cycling feature, causing the key to be intercepted before reaching Planck's PTY sessions. `Alt+Tab` was considered but is intercepted by macOS at the OS level. `|` (pipe / Shift+\) is a rarely-used character in normal agent input that provides a quick, accessible binding. Making the PTY panel keymap-aware ensures that any future rebinding via settings works correctly end-to-end.
