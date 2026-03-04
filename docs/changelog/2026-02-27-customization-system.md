# Customization System: Keybindings, Themes, and Preferences

**Date:** 2026-02-27

## Summary

Introduced a comprehensive customization system with rebindable keybindings, TUI color theme presets, and interactive settings pages for both. All hardcoded key comparisons have been replaced with a centralized keymap abstraction, and five color theme presets are available.

## Changes

### Features
- **Keymap abstraction layer** — All ~35 bindable keyboard actions across 5 contexts (Global, File List, Editor, Agent Normal, Agent Input) are defined as named constants with a `Keymap` struct providing `Matches()`, `KeysFor()`, `ActionFor()`, and `DisplayKeysFor()` methods
- **Interactive keybinding settings** — The Keys settings page now supports interactive rebinding via key capture, conflict detection with swap-or-cancel UI, per-binding and global reset, and visual distinction (accent color) for customized bindings
- **Keybinding config persistence** — User overrides are stored as sparse `[keybindings]` sections in `config.toml` (only changed bindings are persisted, defaults auto-upgrade)
- **TUI color theme presets** — 5 presets: `default` (cyan/teal), `monokai` (orange/purple), `solarized-dark`, `nord` (blue frost), `dracula` (purple/pink/green)
- **Theme settings page** — New "Theme" page in Settings with preset picker and live color swatch preview
- **Dynamic help and status bar** — Help overlay and status bar key hints now render from the active keymap instead of hardcoded strings

### Refactoring
- Replaced all hardcoded `key == "q"` string comparisons in `app.go`, `pty_panel.go`, and `editor.go` with `keymap.Matches()` calls
- Extracted theme construction into `buildTheme()` factory to eliminate duplication across presets
- Removed static `keybindingContexts` variable; settings page reads directly from the keymap

## Files Modified
- `internal/ui/keymap.go` — **New**: Keymap abstraction with Action/Context types, Binding structs, DefaultKeymap(), Matches/KeysFor/ActionFor/SetBinding/ApplyOverrides/Clone methods
- `internal/ui/keymap_test.go` — **New**: Comprehensive tests for keymap operations
- `internal/ui/theme_presets.go` — **New**: ThemeFromPreset factory, buildTheme helper, 4 additional theme definitions (monokai, solarized-dark, nord, dracula)
- `internal/ui/theme_presets_test.go` — **New**: Tests for all presets and fallback behavior
- `internal/ui/settings_theme.go` — **New**: Theme settings page with preset picker and color preview
- `internal/ui/settings_keybindings.go` — Rewritten: interactive rebinding with key capture, conflict detection, reset support
- `internal/ui/settings.go` — Added keymap and themePreset params to NewSettings; added Theme category
- `internal/ui/help.go` — Dynamic rendering from keymap instead of hardcoded lists
- `internal/ui/statusbar.go` — Dynamic key hints from keymap
- `internal/ui/editor.go` — Added keymap field; editor view mode uses keymap.Matches()
- `internal/ui/pty_panel.go` — Added keymap field; input/normal mode use keymap.Matches()
- `internal/app/app.go` — Keymap creation, override application, refactored handleKeypress/handlePlanningTabKey/renderStatusBar to use keymap; ThemeChangedMsg/KeybindingsChangedMsg handlers
- `internal/app/recovery.go` — Updated NewPTYPanel call signature
- `internal/config/config.go` — Added Keybindings map and ThemePreset field; keybinding validation
- `internal/config/config_test.go` — Tests for keybinding validation and theme preset config
- `internal/ui/editor_test.go` — Updated NewEditor calls for new signature
- `internal/ui/pty_panel_test.go` — Updated NewPTYPanel calls for new signature

## Rationale

Keybindings were hardcoded as string comparisons scattered across multiple files, making them impossible for users to customize and difficult to maintain. The keymap abstraction centralizes all bindings, enables runtime rebinding, and automatically keeps the help overlay and status bar in sync. Color theme presets provide meaningful visual personalization with minimal complexity, following the same pattern as the existing markdown theme system.
