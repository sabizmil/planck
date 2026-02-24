# Click-and-Drag Text Selection in Markdown Editor

**Date:** 2026-02-24

## Summary

Added click-and-drag text selection to the markdown editor's edit mode. Users can now click, hold, and drag to select a range of text with real-time visual highlighting. Selected text integrates with all editing operations — typing, backspace, delete, and enter replace the selection.

## Changes

### Features
- Click-and-drag text selection with real-time highlighting during drag
- Selection-aware editing: typing/backspace/delete/enter replace selected text
- Arrow keys and Home/End clear the selection (standard editor behavior)
- Selection works correctly across word-wrapped visual lines and while scrolled
- Selection is cleared on mode transitions (edit/view) and content changes
- Added `selectedText()` accessor for future copy/paste support

### Internal
- Extracted `mouseToLogical()` helper for translating screen coordinates to buffer positions
- Refactored `handleMouse` into separate `handleMousePress`, `handleMouseMotion`, and `handleMouseRelease` methods
- Added selection state fields to Editor struct (`selecting`, `hasSelection`, anchor/end coordinates)
- Added `clearSelection()`, `selectionRange()`, `isInSelection()`, `deleteSelection()` helpers
- Updated app-level mouse handler to forward motion/release events to editor during drag
- Added `Selecting()` accessor for app-level coordination

### Bug Fixes
- Fixed 7 golangci-lint issues that were failing CI:
  - Removed trailing blank lines (gofmt) in `workspace.go` and `app.go`
  - Added explanations to `//nolint:errcheck` directives in `main.go`
  - Used `http.NoBody` instead of `nil` in `updater.go`
  - Used modern `0o755` octal literal style in `updater.go`
  - Replaced `defer` in loop with explicit `Close()` call in `updater.go`
  - Removed unused `updateFocus()` function from `app.go`
  - Converted `if/else if` chains to `switch` statements in `editor.go`
  - Added named return values to satisfy `unnamedResult` lint rule in `editor.go`

### Tests
- 13 new tests covering selection range normalization, `isInSelection` boundary checks, single/multi-line deletion, backward selection, `selectedText()`, clearing, and rendering with selection

## Files Modified
- `internal/ui/editor.go` — Selection state, mouse state machine, rendering, editing integration, lint fixes
- `internal/ui/editor_test.go` — 13 new selection tests
- `internal/app/app.go` — Forward mouse motion/release to editor during drag selection, removed unused function
- `cmd/planck/main.go` — Added nolint explanations
- `internal/updater/updater.go` — httpNoBody, octal literal, defer-in-loop fixes
- `internal/workspace/workspace.go` — Trailing blank line fix

## Rationale
Click-and-drag selection is a fundamental text editor interaction. The implementation uses a full press/motion/release state machine for real-time visual feedback, reusing the existing visual line infrastructure for coordinate mapping. The selection state is designed to support future keyboard-based selection (Shift+Arrow) and clipboard operations.
