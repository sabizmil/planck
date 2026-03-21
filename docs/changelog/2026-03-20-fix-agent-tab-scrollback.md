# Fix Agent Tab Mouse/Trackpad Scrollback

**Date:** 2026-03-20

## Summary

Restored mouse/trackpad scrollback in agent tabs. Scrolling broke when the default backend switched from PTY to tmux in commit 667502b — the tmux backend had no scrollback integration with the panel's scroll mechanism.

## Changes

### Bug Fixes
- Fixed `PTYPanel.scrollUp()` to compute max scroll offset from total content lines (scrollback + screen lines - viewport height) instead of just scrollback buffer length
- Added `maxScrollOffset()` helper used by `scrollUp()` and `scrollToTop()`
- Changed tmux `Render()` to capture pane content including up to 5000 lines of scrollback history (`-S -5000`)
- Strip trailing empty lines from tmux capture output to avoid excess blank space when scrollback is short

### Documentation
- Fixed contradictory doc comment in `config.go` (said "auto will prefer PTY" but code prefers tmux)

### Tests
- Added `TestPTYPanelScrollWithNilScrollback` — verifies scroll works with nil scrollback buffer and content exceeding viewport
- Added `TestPTYPanelScrollToTopNilScrollback` — verifies scrollToTop with nil scrollback
- Added `TestPTYPanelScrollNoOverflow` — verifies scrolling is a no-op when content fits viewport

## Files Modified
- `internal/ui/pty_panel.go` — Added `maxScrollOffset()`, updated `scrollUp()` and `scrollToTop()` to use it
- `internal/ui/pty_panel_test.go` — Added three scroll test cases
- `internal/tmux/backend.go` — Updated `Render()` to include scrollback via `-S -5000`
- `internal/config/config.go` — Fixed doc comment for auto backend preference

## Rationale

The root cause was a two-part issue: (1) the tmux backend's `GetScrollback()` returns nil because tmux manages scrollback internally, and (2) `scrollUp()` clamped the scroll offset to `scrollbackLen()` which was 0 for nil scrollback. The fix captures tmux scrollback in the render output and computes the max scroll offset from all available content, making both backends scroll correctly.
