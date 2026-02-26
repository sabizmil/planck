# Fix Tab Key Passthrough in PTY Input Mode

**Date:** 2026-02-25

## Summary

Fixed Tab key being intercepted for tab-switching even when typing into an agent session, preventing autocomplete and path completion. Replaced the tab-switching scheme with Shift+Tab (forward cycle), mouse click on tab bar, and Alt+{1-9} direct tab jumping.

## Changes

### Features
- Tab key now passes through to the PTY in input mode, enabling autocomplete/path completion in Claude Code
- Added mouse click support on the tab bar to switch tabs (works on all tabs, not just planning)
- Added Alt+{1-9} keyboard shortcut for direct tab jumping (works in all modes including input mode)
- Added `HitTest(x int) int` method to `TabBar` for click-to-tab resolution

### Breaking Changes
- Bare Tab key no longer switches tabs (use Shift+Tab instead)
- Shift+Tab now cycles **forward** (was backward)
- Tab/Shift+Tab behavior in input mode: Tab goes to PTY, Shift+Tab cycles tabs

### UI Updates
- Updated status bar hints to reflect new keybindings
- Updated help overlay with new keybinding documentation
- Updated settings keybindings reference page

## Files Modified
- `internal/app/app.go` - Removed bare Tab handler, changed Shift+Tab to cycle forward, added Alt+{1-9} handling, added tab bar mouse click handling, updated status bar hints
- `internal/ui/pty_panel.go` - Removed Tab block in input mode (Tab now passes through to PTY), kept Shift+Tab block (handled at app level)
- `internal/ui/tabs.go` - Added `HitTest()` method for mouse click resolution
- `internal/ui/help.go` - Updated keybinding documentation
- `internal/ui/settings_keybindings.go` - Updated keybinding reference entries
- `internal/ui/tabs_test.go` - New tests for `HitTest()`
- `internal/ui/pty_panel_test.go` - New tests for Tab passthrough and Shift+Tab blocking

## Rationale

The Tab key is essential for autocomplete in CLI tools like Claude Code, but was being intercepted by Planck for tab-switching even when the user was actively typing into an agent session. The new scheme uses Shift+Tab (universally supported in terminals) as the primary tab-cycling key, with mouse clicks and Alt+{1-9} as additional direct-access methods.
