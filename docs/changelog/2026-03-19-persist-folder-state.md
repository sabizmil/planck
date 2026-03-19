# Persist Sidebar Folder Expand/Collapse State

**Date:** 2026-03-19

## Summary

Sidebar folder expand/collapse state is now persisted between app launches via SQLite. The tree looks the same when you reopen planck.

## Changes

### Features
- Folder expand/collapse state saved on app exit and restored on startup
- New `ui_state` key-value table in SQLite for persisting ephemeral UI state
- Zero runtime overhead — state is only written to DB during `app.Close()`, not on every toggle

### Infrastructure
- Added `GetUIState(key)` / `SetUIState(key, value)` methods to Store
- Added `GetDirState()` / `SetDirState()` methods to FileList for state import/export
- `ui_state` table is generic (key/value) and can store other UI state in the future

## Files Modified
- `internal/store/store.go` — Added `ui_state` table migration and Get/Set methods
- `internal/store/store_test.go` — Added `TestUIState` round-trip test
- `internal/ui/filelist.go` — Added `GetDirState()` and `SetDirState()` export/import methods
- `internal/app/app.go` — Save dirState in `Close()`, restore in `Init()` before first refresh

## Rationale
Folder state resets on every launch, making users re-expand their preferred tree layout each time. Persisting via SQLite (write-on-quit only) is the simplest approach with zero runtime overhead. If the app crashes, only changes since the last clean exit are lost — acceptable for a convenience feature.
