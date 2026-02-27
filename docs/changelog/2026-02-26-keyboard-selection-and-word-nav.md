# Keyboard Selection and Word Navigation

**Date:** 2026-02-26

## Summary

Replaced broken mouse drag selection with reliable keyboard-based selection (Shift+Arrow) and word-level navigation (Alt+Arrow). Mouse drag selection caused persistent rendering corruption due to a four-way interaction between continuous MouseMotion events, per-character ANSI rendering, lipgloss word wrapping, and Bubble Tea's differential renderer. The new system uses discrete keyboard events that produce exactly one re-render per action.

## Changes

### Features
- Shift+Left/Right: character-level selection
- Shift+Up/Down: line-level selection (visual lines with wrapping)
- Shift+Home/End: select to line start/end
- Alt+Left/Right: jump cursor by word boundary
- Alt+Shift+Left/Right: select by word boundary
- Ctrl+Left/Right: word jump (Linux/Windows convention)
- Shift+Click: range selection from current cursor to clicked position
- Status bar and help screen updated with new shortcut hints

### Bug Fixes
- Eliminated mouse drag selection rendering corruption (content duplication, ghost cursor indicators, viewport shifting)
- Added MaxHeight safety nets to editor, file list, and app-level content rendering to prevent lipgloss overflow
- Fixed FileList `visibleLines()` overcounting chrome (was -5, now -2)
- Added `ensureVisible()` call in FileList `SetSize()` after resize

### Refactoring
- Extracted `moveCursorLeft()`, `moveCursorRight()`, `moveCursorToLineStart()`, `moveCursorToLineEnd()` helper methods from inline key handlers
- Added `extendSelection()` method for unified selection extension logic
- Added `wordBoundaryLeft()` and `wordBoundaryRight()` methods for word navigation
- Removed `selecting` field, `Selecting()` method, `handleMouseMotion`, `handleMouseRelease`, and `mouseToLogicalClamped` — all related to mouse drag
- Cleaned up app-level mouse event routing that referenced removed `Selecting()` method
- Kept run-based selection rendering (reduces ANSI output size)

## Files Modified
- `internal/ui/editor.go` - New cursor/selection helpers, word navigation, keyboard selection handlers, removed mouse drag code
- `internal/ui/editor_test.go` - 20+ new tests for word boundaries, shift+arrow selection, alt+arrow jumping, shift+click, cursor movement edge cases
- `internal/app/app.go` - Removed `Selecting()` references, updated status bar hints
- `internal/ui/filelist.go` - MaxHeight addition, visibleLines() fix, ensureVisible() on resize
- `internal/ui/help.go` - Added edit mode keyboard shortcut documentation

## Rationale

Three rounds of attempted fixes for the mouse drag rendering bug failed. The root cause was a fundamental incompatibility between continuous mouse drag events and the lipgloss/Bubble Tea rendering pipeline. Instead of continuing to patch the rendering side, this change eliminates the problematic input source entirely. Keyboard selection produces discrete, predictable re-renders and is the standard editing paradigm developers expect.
