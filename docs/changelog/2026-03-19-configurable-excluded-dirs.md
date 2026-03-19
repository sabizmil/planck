# Configurable Excluded Directories

**Date:** 2026-03-19

## Summary

Added a configurable list of excluded directories that controls both the sidebar file listing and the file watcher. Ships with sensible defaults and is editable in the Settings UI.

## Changes

### Features
- New `exclude_dirs` config field in `[preferences]` with defaults: `.git`, `.hg`, `.svn`, `node_modules`, `vendor`, `.next`, `build`, `dist`, `__pycache__`, `.claude`
- "Excluded Dirs" text field in General settings page — comma-separated, editable at runtime
- Changes take effect immediately (workspace re-scans on save)

### Bug Fixes
- `Refresh()` now skips excluded and hidden directories — previously it walked into all directories including `.git` and `node_modules`, showing their files in the sidebar even though the watcher already filtered them

## Files Modified
- `internal/config/config.go` — Added `ExcludeDirs []string` to Preferences struct and `DefaultExcludeDirs` package variable
- `internal/workspace/workspace.go` — Added `excludeDirs` field to Workspace, filtering in `Refresh()` and `Watch()`, `SetExcludeDirs()` method for runtime updates
- `internal/workspace/workspace_test.go` — Added `TestRefreshExcludesDirs` with table-driven tests
- `internal/ui/settings.go` — Added `ExcludeDirs` to `GeneralSettingsChangedMsg`
- `internal/ui/settings_general.go` — Added "Excluded Dirs" text field, `parseExcludeDirs` helper, updated field indices
- `internal/app/app.go` — Pass `ExcludeDirs` to workspace constructor, handle settings changes with `SetExcludeDirs()`

## Rationale

Users reported `node_modules` and other large directories appearing in the sidebar. The watcher already filtered these directories (added in the fd leak fix), but the file listing walk (`Refresh`) had no filtering at all. Making the list configurable lets users adapt it to their project structure.
