# Line-Based Scrollback for PTY Panels

**Date:** 2026-02-21

## Summary

Implemented real line-based scrollback for PTY panels so users can scroll up through content that has scrolled off the top of the embedded terminal, matching native terminal behavior.

## Changes

### Features
- Line-by-line scrollback in PTY panels (mouse wheel: 3 lines/tick, keyboard: j/k/g/G/PgUp/PgDn)
- Scroll indicator badge ("SCROLL [-N lines]") shown at top-right when scrolled up
- 1000-line ring buffer per session (configurable via `scrollbackLines`)
- Alternate screen filtering: scrollback skips content from vi, less, fzf, etc.
- Sub-region scroll filtering: only captures full-screen scrolls, not status bar updates

### Architecture
- Local fork of `charmbracelet/x/vt` with `ScrollOff` callback hook
- `ScrollbackBuffer` ring buffer in `internal/session/` with independent mutex
- Combined viewport rendering: scrollback lines + live screen lines

## Files Modified
- `internal/vt/` (new) - Local fork of charmbracelet/x/vt with ScrollOff callback
- `internal/vt/callbacks.go` - Added ScrollOff callback field
- `internal/vt/cc.go` - Added scrollUpWithCapture(), hooked into index()
- `internal/vt/handlers.go` - Hooked CSI 'S' (Scroll Up) handler
- `internal/session/scrollback.go` (new) - Thread-safe ring buffer
- `internal/session/pty.go` - Wired ScrollOff callback, added scrollback field and getter
- `internal/ui/pty_panel.go` - Replaced screen-history with line-based scrollback viewport
- `internal/app/app.go` - Wires scrollback buffer from backend to panel on launch
- `go.mod` - Added replace directive for local vt fork

## Rationale

The previous "screen history" approach stored full screen snapshots and only allowed coarse screen-by-screen scrolling. The new approach captures individual lines as they scroll off the terminal (via a callback in the vt emulator), enabling smooth line-by-line scrollback that matches native terminal behavior. A local fork of the vt package was necessary because the screen buffer is private and the library has no scroll-related hooks (it has a TODO for exactly this feature).
