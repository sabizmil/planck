# Configurable spinner with settings UI picker

**Date:** 2026-02-22

## Summary

Added a spinner preset registry with 26 styles and a new Spinner settings page with live preview. The spinner style is configurable at runtime and persists in `planck.toml`.

## Changes

### Features
- Spinner preset registry with 26 styles: claude (default), dot-pulse, dots, line, star, flip, bounce, box-bounce, arc, circle, circle-half, square-corners, triangle, binary, toggle, arrow, balloon, noise, grow-h, grow-v, layer, moon, hearts, clock, point, meter, breathe
- `reverseMirror` support expands frames into forward+backward sequences for smooth breathing animations (used by claude, noise, grow-h, grow-v, layer presets)
- Each preset defines its own tick interval (70ms–250ms) instead of using a hardcoded 250ms
- New "Spinner" category in the Settings panel (5th sidebar entry)
- Live animated preview of the selected spinner in the settings page, including a simulated tab bar preview, frame display, and interval info
- Scrollable preset list with checkmark on the active selection
- Spinner style persists via `spinner_style` in `[preferences]` section of `planck.toml`
- Default spinner changed from dot-pulse to "claude" (asterisk breathing: `· ✢ ✳ ✶ ✻ ✽` at 120ms)

### Technical
- TabBar spinner state moved from package-level var to struct fields (`spinnerFrames`, `spinnerInterval`)
- `SetSpinner(preset)` method on TabBar allows runtime spinner changes
- Spinner tick keeps running while Settings panel is visible (even without running tabs) to power the live preview
- Settings panel forwards `SpinnerTickMsg` to the spinner page for preview animation

## Files Modified
- `internal/ui/spinners.go` — New: SpinnerPreset struct, 26 presets with reverseMirror expansion, SpinnerPresets(), SpinnerPresetByName(), DefaultSpinnerPreset()
- `internal/ui/settings_spinner.go` — New: spinnerPage implementing settingsPage interface with scrollable list and live preview
- `internal/ui/tabs.go` — Moved spinner state to TabBar struct, added SetSpinner(), configurable tick interval
- `internal/ui/settings.go` — Added Spinner category, updated NewSettings signature, forward SpinnerTickMsg to spinner page
- `internal/config/config.go` — Added SpinnerStyle to Preferences, default "claude"
- `internal/app/app.go` — Apply spinner from config on startup, handle SpinnerSettingsChangedMsg, start tick for settings preview

## Rationale

The hardcoded dot-pulse spinner was functional but offered no customization. A preset registry with a settings UI picker makes spinner selection discoverable and instant, while persisting the choice avoids re-selection on restart. The claude-style asterisk breathing animation is a better default that matches the tool's identity.
