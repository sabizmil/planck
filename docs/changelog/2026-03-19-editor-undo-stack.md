# Undo/Redo Stack for Markdown Editor

**Date:** 2026-03-19

## Summary

Added undo (Ctrl+Z) and redo (Ctrl+Y) support to the custom markdown editor with intelligent grouping — consecutive character typing is grouped into single undo entries, matching VS Code-like behavior.

## Changes

### Features
- Undo via Ctrl+Z restores previous editor state (content + cursor position)
- Redo via Ctrl+Y re-applies undone changes
- Intelligent edit grouping: consecutive typing coalesces into one undo entry
- Group breaks on: edit type change (insert→delete), whitespace/newline, 500ms pause, cursor jump, or paste
- Undo stack capped at 100 entries; oldest entries dropped when exceeded
- Redo stack cleared on any new edit (standard editor behavior)
- Undo/redo history reset when switching files via SetContent

### Architecture
- Snapshot-based approach: captures full `(lines, cursorRow, cursorCol)` before mutations
- No changes to existing edit operations — undo hooks added at the call sites in `updateEditMode`
- New `ContextEditorEdit` keymap context for edit mode bindings (help/settings display)

## Files Modified
- `internal/ui/editor.go` — Added `editorSnapshot`, `editType` types, undo/redo stacks and grouping fields on Editor, `pushUndo()`, `Undo()`, `Redo()`, `resetUndoHistory()` methods, Ctrl+Z/Y handling in updateEditMode, pushUndo calls before all edit operations, stack reset in SetContent
- `internal/ui/editor_test.go` — 16 new tests covering: basic undo/redo, empty stack safety, multiple edits, consecutive insert grouping, group break on type change, group break on timeout, paste always new group, newline undo, stack cap, reset on SetContent, delete backward/selection undo, full round-trip undo/redo
- `internal/ui/keymap.go` — Added `ContextEditorEdit` context, `ActionEditorUndo`/`ActionEditorRedo` actions with Ctrl+Z/Ctrl+Y bindings

## Rationale

The editor had no undo capability — every keystroke was permanent until save. This made editing feel risky, especially for plan documents where accidental deletions could lose work. The snapshot approach was chosen for its simplicity and uniform handling of all edit types, with intelligent grouping to make undo feel natural rather than character-by-character.
