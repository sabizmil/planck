# Fix Editor UTF-8 Rendering Bug

**Date:** 2026-03-20

## Summary

Fixed rendering corruption in the editor's edit mode where multi-byte UTF-8 characters (emoji, checkboxes, accented letters) caused garbled output and doubled lines. The entire editor was operating on byte offsets instead of rune offsets.

## Changes

### Bug Fixes
- Fixed `renderEditMode()` to iterate over runes instead of bytes, eliminating garbled character rendering (e.g., "â◆◆" from split 3-byte UTF-8 sequences)
- Fixed cursor splitting in `renderEditMode()` to slice at rune boundaries instead of byte boundaries
- Fixed `wrapLine()` to use rune count for width measurement, preventing premature line wrapping with multi-byte content
- Fixed `buildVisualLines()` to track `colOffset` in rune count instead of byte count
- Fixed `cursorVisualRow()` to use rune-based segment length comparisons
- Fixed `moveCursorLeft/Right()` to advance by one rune instead of one byte
- Fixed `moveCursorToLineEnd()` to use rune count instead of byte length
- Fixed `deleteBackward()` to delete full runes using `utf8.DecodeLastRuneInString`
- Fixed `deleteForward()` to delete full runes using `utf8.DecodeRuneInString`
- Fixed `insertText()` to use rune-based cursor positioning
- Fixed `insertNewline()` to split at rune boundary
- Fixed `wordBoundaryLeft/Right()` to operate on runes (renamed `isWordChar` → `isWordRune`)
- Fixed `mouseToLogical()` to clamp click position using rune count
- Fixed `deleteSelection()` and `selectedText()` to use rune-based byte offset conversion

### New Code
- Added `runeLen()`, `runeToByteOffset()`, `byteToRuneOffset()` helper functions for rune/byte conversion

### Tests
- Added 10 new UTF-8 test cases covering cursor movement, insert, delete, newline, wrap, word boundary, selection, and helper functions with multi-byte characters

## Files Modified
- `internal/ui/editor.go` — Converted all editor operations from byte-based to rune-based indexing
- `internal/ui/editor_test.go` — Added UTF-8 test cases, updated `isWordChar` test to `isWordRune`

## Rationale

The editor stored cursor positions as byte offsets and used Go's byte-level string indexing (`s[i]`, `len(s)`, `s[:n]`) throughout. Multi-byte UTF-8 characters (3 bytes for ☐, 2 bytes for é, etc.) were split into individual garbled bytes during rendering. The frequent redraws from agent polling amplified this into visible ghosting/doubled lines via Bubble Tea's terminal diff algorithm.
