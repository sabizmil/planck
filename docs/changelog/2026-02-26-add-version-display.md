# Add Version Display to Status Bar

**Date:** 2026-02-26

## Summary

Added a version indicator to the bottom-right of the TUI status bar, next to the help/quit hotkey labels. Production builds show the semantic version in dimmed text; local dev builds (compiled without ldflags) show "dev" in bold yellow for clear differentiation.

## Changes

### Features
- Version label displayed in the status bar right side, after `[?] help  [q] quit`
- Production builds show the version string (e.g., `v0.2.3`) in dimmed gray
- Local dev builds (`go build ./cmd/planck` without ldflags) show `dev` in bold yellow
- Added `VersionDev` theme style for the yellow dev-build indicator
- Supports both color and no-color terminal modes

## Files Modified
- `internal/app/app.go` - Added `version` field to App struct, updated constructor to accept version param, updated `renderStatusBar()` to display version
- `cmd/planck/main.go` - Pass `version` variable to `app.New()` constructor
- `internal/ui/theme.go` - Added `VersionDev` style (bold yellow) to both default and no-color themes

## Rationale

When developing locally, it's important to know whether you're running a production install or a local test build. The existing ldflags-based version injection already defaults to "dev" for bare `go build` commands, so the detection mechanism was free—we just needed to surface it in the UI.
