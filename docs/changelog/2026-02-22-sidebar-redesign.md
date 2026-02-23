# Sidebar Selection Indicator & Width Redesign

**Date:** 2026-02-22

## Summary

Improved sidebar usability with a full-width background highlight for the selected item and user-configurable sidebar width with mouse-drag resizing.

## Changes

### Features
- Selected sidebar item now has a full-width background highlight (`#1E3A5F` dark blue) making the current selection immediately visible at a glance
- Sidebar width is now configurable (16–60 characters, default 28) via the Settings panel under General > Sidebar Width
- Mouse-drag resizing: click and drag the sidebar border to resize interactively; width persists to config on release
- `SidebarWidth` added to `[preferences]` in config.toml

### Improvements
- Selected item prefix (`▸`) is now part of the highlighted row rather than separately styled, eliminating visual fragmentation
- Non-selected items use individually-colored status indicators while selected items use a unified highlight style for clarity
- `noColorTheme` uses `Reverse(true)` for selection highlight (accessible fallback)
- Editor area is guaranteed at least 40 columns; sidebar shrinks automatically on narrow terminals

## Files Modified
- `internal/config/config.go` — Added `SidebarWidth` field to `Preferences` struct with default 28
- `internal/ui/theme.go` — Added `selectedBg` color (`#1E3A5F`), applied `Background()` to `Selected`, `SidebarSelected`, `TreeSelected` styles
- `internal/ui/filelist.go` — Added `padToWidth()` helper, refactored View() to pad selected lines to full width with background highlight
- `internal/ui/settings.go` — Added `SidebarWidth` to `GeneralSettingsChangedMsg`
- `internal/ui/settings_general.go` — Added "Sidebar Width" number field, `adjustNumberField()` method, updated field indices
- `internal/app/app.go` — Added `sidebarWidth` and `draggingSidebar` fields to App, replaced hardcoded width with config value, added drag-resize mouse handling, wired settings persistence

## Rationale

The previous sidebar had two usability issues: the selection indicator (a small `▸` in cyan) was too subtle to spot at a glance, and the fixed 24-character width truncated most filenames. The full-width background highlight follows the universal convention used by VS Code, Finder, and other file trees. The configurable width with drag-resize gives users full control over the sidebar/editor balance.
