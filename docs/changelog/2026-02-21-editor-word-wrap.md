# Editor Word Wrap for Long Lines

**Date:** 2026-02-21

## Summary

Long lines in the built-in markdown editor are now soft-wrapped at word boundaries instead of being truncated with "...". All text is visible and editable without horizontal scrolling.

## Changes

### Features
- Added soft word wrapping in edit mode — long lines wrap at word boundaries (spaces), with fallback to character-level wrapping for words exceeding the line width
- Continuation lines display a `·` gutter marker instead of a line number, making it easy to distinguish wrapped segments from new logical lines
- Cursor up/down navigation moves through visual (wrapped) lines, preserving horizontal column position
- Mouse click placement works correctly on wrapped lines
- Scroll and viewport logic accounts for wrapped lines consuming multiple display rows

### Removed
- Removed hard truncation ("...") behavior that previously cut off long lines in edit mode

## Files Modified
- `internal/ui/editor.go` — Replaced `renderEditMode()` truncation with word-wrap rendering; added `wrapLine()`, `buildVisualLines()`, `cursorVisualRow()`, `editMaxLineWidth()` helpers; added `moveCursorUpVisual()`/`moveCursorDownVisual()` for visual-line cursor navigation; updated `ensureCursorVisible()`, `ScrollBy()`, and mouse click handler for wrapped lines
- `internal/ui/editor_test.go` — Updated `TestEditor_LongLinesRendering` to verify wrapping (no "..."); added `TestWrapLine` table-driven tests for the wrap helper; added `TestEditor_WrappedCursorNavigation` for visual-line cursor movement

## Rationale
Plan files and markdown documents often contain long prose lines (descriptions, approach evaluations) that were impossible to read or edit because the editor truncated them. Soft word wrap is the standard solution used by virtually every text editor.
