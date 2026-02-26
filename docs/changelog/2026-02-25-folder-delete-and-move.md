# Folder Deletion & Interactive Move

**Date:** 2026-02-25

## Summary

Added folder deletion with confirmation dialog and an interactive move mode for relocating files and folders within the workspace.

## Changes

### Features
- **Folder deletion**: Pressing `d` on a folder now shows a confirmation dialog warning that all files inside will be deleted, then recursively removes the folder and its contents
- **Interactive move mode**: Pressing `m` on any file or folder enters move mode — a visual destination picker where you navigate the file tree with `j`/`k`, expand/collapse folders with `l`/`h`, and press `Enter` on a target folder (or `(root)`) to move the item there. Press `Esc` to cancel
- Editor content is automatically cleared when deleting a folder containing the currently-open file
- Editor is updated when moving a file or folder that contains the currently-open file

### Workspace API
- Added `DeleteFolder(name)` with path traversal protection
- Added `MoveFile(oldName, newDir)` with automatic intermediate directory creation
- Added `MoveFolder(oldPath, newDir)` with self-move and descendant-move prevention

## Files Modified
- `internal/ui/filelist.go` — Added move mode state, `MoveConfirmedMsg`/`MoveCancelledMsg` messages, `EnterMoveMode()`/`ExitMoveMode()`/`InMoveMode()` methods, `SelectedDirPath()`/`SelectedPath()`/`SelectPath()` methods, move mode rendering with `(root)` synthetic node
- `internal/workspace/workspace.go` — Added `DeleteFolder()`, `MoveFile()`, `MoveFolder()` methods
- `internal/workspace/workspace_test.go` — Added tests for DeleteFolder, MoveFile, MoveFolder (15 new test cases)
- `internal/app/app.go` — Updated `d` handler for folder deletion, added `m` handler for move mode, added `MoveConfirmedMsg`/`MoveCancelledMsg` handling, added `handleMoveConfirmed()`, suppressed auto-preview during move mode
- `internal/ui/help.go` — Updated `d` description to "Delete file/folder", added `m` keybinding

## Rationale

Users needed the ability to reorganize their workspace — deleting unnecessary folders and moving files/folders between directories. The interactive move mode was chosen over a text input dialog because it provides visual context and prevents typo-related errors.
