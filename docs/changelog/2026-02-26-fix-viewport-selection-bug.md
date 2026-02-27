# Fix Viewport/Selection Bug

**Date:** 2026-02-26

## Summary

Fixed a bug where the TUI viewport would shift down during text selection in edit mode, causing the tab bar to disappear, cursor indicators to duplicate, and content lines to render out of order.

## Changes

### Bug Fixes
- Added `MaxHeight()` constraints to editor, file list, and app-level content rendering to prevent overflow when lipgloss `Height()` fails to truncate content
- Reduced `editMaxLineWidth()` by 1 character to create a safety margin from the lipgloss word-wrap boundary, avoiding a boundary condition where per-character ANSI styling during selection could trigger incorrect line wrapping
- Fixed FileList `visibleLines()` calculation from `f.height - 5` to `f.height - 2` (was overcounting chrome by 3 lines)
- Added separate `moveVisibleLines()` for move mode which correctly accounts for footer chrome (`f.height - 4`)
- Added `ensureVisible()` call in FileList `SetSize()` to keep cursor in view after terminal resize

## Files Modified
- `internal/ui/editor.go` - Added `.MaxHeight(e.height)` to View() render; reduced editMaxLineWidth margin from -8 to -9
- `internal/ui/filelist.go` - Added `.MaxHeight(f.height)` to both View() and viewMoveMode(); fixed visibleLines(); added moveVisibleLines(); added ensureVisible() to SetSize()
- `internal/app/app.go` - Added `.MaxHeight(contentHeight)` to content render

## Rationale

The root cause was that lipgloss `Height()` does NOT truncate content that exceeds the specified height — it only pads shorter content. During text selection, per-character ANSI escape codes create extreme ANSI density that can cause lipgloss's word wrapper to miscalculate visual widths, adding extra lines. Without truncation, this overflow cascades through the rendering pipeline, causing the terminal's alt screen to scroll and Bubble Tea's incremental renderer to leave ghost artifacts from previous frames. The fix uses `MaxHeight()` as a hard safety net alongside `Height()` for padding, with margin adjustments to avoid the wrapping boundary condition.
