# Use Current Working Directory on Startup

**Date:** 2026-02-23

## Summary

Changed Planck's startup behavior to always open in the current working directory instead of reopening the last-used folder. Removed folder switching functionality and all related code.

## Changes

### Features
- Planck now opens in the current working directory by default (no more reopening last-used folder)
- The `-folder` flag remains as an explicit override

### Removed
- Removed folder picker overlay and `o` keybinding for switching folders at runtime
- Removed `FolderPicker` UI component (`internal/ui/folderpicker.go`)
- Removed `RecentFolders` struct and `recent.json` persistence (`internal/workspace/workspace.go`)
- Removed standalone `FolderPickerModel` from app package
- Removed `runFolderPicker` function from main

## Files Modified
- `cmd/planck/main.go` - Replaced recent-folder startup logic with `os.Getwd()`, removed `runFolderPicker`, removed `workspace` import, updated help text
- `internal/app/app.go` - Removed `folderPicker`/`folderPickerVisible` fields, folder picker overlay handling, `showFolderPicker()`, `switchFolder()`, `FolderPickerModel`, `NewFolderPickerModel`, `o` keybinding, and related status bar hints
- `internal/ui/folderpicker.go` - Deleted entirely
- `internal/workspace/workspace.go` - Removed `RecentFolders` struct, `LoadRecentFolders`, `Save`, and `Add` methods

## Rationale

The previous behavior of reopening the last-used folder was confusing — running `planck` from any directory would open a different folder. The new behavior follows the principle of least surprise: the app opens where you are. This simplifies both the codebase and the mental model.
