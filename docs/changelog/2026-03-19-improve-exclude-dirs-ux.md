# Improve Excluded Dirs UX

**Date:** 2026-03-19

## Summary

Replaced the comma-separated text field for excluded directories with a proper list editor in Settings, and added a sidebar hotkey to exclude folders directly from the file list.

## Changes

### Features
- **List editor in Settings**: The "Excluded Dirs" field in General settings is now a navigable list. Press Enter to expand, then `j`/`k`/arrows to navigate entries, `n`/Enter to add a new entry with inline text input, `d`/Backspace to remove, Esc to collapse.
- **Sidebar exclude hotkey**: Press `X` (shift+X) on a folder in the file list to exclude it. Shows a confirmation dialog, then adds the folder name to the exclude list and hides it immediately.
- Settings panel syncs with sidebar hotkey — directories added via `X` appear in the Settings list.

### Refactoring
- Removed `parseExcludeDirs()` comma-parsing function — no longer needed with list-based storage
- Changed `excludeDirs` from `string` to `[]string` throughout the general settings page
- Added `AddExcludeDir()` method on Settings for cross-component communication

## Files Modified
- `internal/ui/settings_general.go` — New "list" field kind with sub-list navigation, add/remove, inline text input
- `internal/ui/settings.go` — Added `AddExcludeDir()` delegate method on Settings struct
- `internal/ui/keymap.go` — Added `ActionExcludeDir` action bound to `X` in FileList context
- `internal/app/app.go` — Handle exclude hotkey: confirmation dialog → config update → workspace refresh

## Rationale
A comma-separated text field is error-prone and hard to scan. The list editor gives clear visibility of each entry and simple add/remove operations. The sidebar hotkey provides the fastest workflow — see offending folder, press `X`, done.
