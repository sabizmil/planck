# Settings refactor, activity spinner, and UI cleanup

**Date:** 2026-02-21

## Summary

Refactored the settings panel into a composable page-based architecture with full General, Agents, and Keybindings pages. Added a braille spinner animation for running agent tabs. Removed decorative bottom separators from the Files and Editor panes.

## Changes

### Features
- Braille dot spinner (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) animates on running agent tabs at 80ms intervals
- Running tabs are styled with the accent color in the tab bar
- Braille pattern characters (U+2800–U+28FF) are stripped from agent OSC window titles to prevent Claude Code's own spinner from leaking through

### Settings Panel — Composable Page Architecture
- Introduced `settingsPage` interface: `Title()`, `Update()`, `View()`, `FooterHints()`, `OnEnter()`, `OnLeave()`, `IsEditing()`
- Extracted existing markdown style settings into `settings_markdown.go`
- All four sidebar categories are now navigable (previously General, Agents, Keys were disabled stubs)
- Footer hints update per-page instead of being hardcoded
- On close, all pages emit their save messages via `OnLeave()`

### Settings — General Page (`settings_general.go`)
- Editor (text input), Terminal Bell (toggle), Session Backend (cycle selector)
- Default Scope, Auto Advance, Permission Mode — execution settings
- Changes saved via `GeneralSettingsChangedMsg` → `config.Save()`

### Settings — Agents Page (`settings_agents.go`)
- Agent list with detail view for the selected agent
- Text inputs for Command and Label, editable Planning Args
- Default agent toggle (ensures single default)
- Changes saved via `AgentsSettingsChangedMsg` → `config.Save()`
- Tab bar labels update immediately when agent labels change

### Settings — Keybindings Page (`settings_keybindings.go`)
- Read-only reference of all keyboard shortcuts organized by context
- Contexts: Global, File Browser, Editor, Agent Tab, Settings, Dialog

### UI Cleanup
- Removed bottom `───` separator from the Files pane (`filelist.go`)
- Removed bottom `───` separator from the Editor pane (`editor.go`)

## Files Modified
- `internal/app/app.go` — Settings init with page configs, GeneralSettingsChangedMsg/AgentsSettingsChangedMsg handlers, spinner tick forwarding, running tab status tracking, braille character filtering in sanitizeTabTitle
- `internal/ui/tabs.go` — SpinnerTickMsg, braille spinner frames, HasRunningTabs(), Tick(), running tab styling
- `internal/ui/settings.go` — Refactored to settingsPage interface, sidebar delegates to pages, removed inline markdown rendering
- `internal/ui/settings_markdown.go` — New: extracted markdown style settings page
- `internal/ui/settings_general.go` — New: general settings page
- `internal/ui/settings_agents.go` — New: agents settings page
- `internal/ui/settings_keybindings.go` — New: keybindings reference page
- `internal/ui/editor.go` — Removed bottom separator
- `internal/ui/filelist.go` — Removed bottom separator

## Rationale

The monolithic settings panel couldn't scale to multiple pages. The page interface pattern keeps each settings domain self-contained while sharing sidebar navigation. The spinner gives immediate visual feedback that an agent is actively working. Removing bottom separators reclaims vertical space in a TUI where every line counts.
