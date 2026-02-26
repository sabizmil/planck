# File Browser Mouse Scroll and Click Support

**Date:** 2026-02-25

## Summary
Added mouse interaction support to the planning tab's file browser sidebar, including wheel scrolling and click-to-select for files and directories.

## Changes

### Features
- Mouse wheel scrolling in the file browser sidebar (scrolls the file list independently of the markdown viewer)
- Click on files to select and auto-preview them in the editor
- Click on directories to toggle expand/collapse state
- Mouse wheel events are now routed based on cursor position: over sidebar scrolls the file list, over editor scrolls the markdown

### Architecture
- Added `HandleMouse()` method to `FileList` with `ClickAction` return type for clean separation of concerns
- Added `ScrollBy()` method to `FileList` for programmatic scroll control with cursor clamping
- Added `SetPosition()` to `FileList` for screen coordinate translation
- Updated `handlePlanningMouse()` in App to route events based on X position relative to sidebar border

## Files Modified
- `internal/ui/filelist.go` - Added `ClickAction` type, `screenY` field, `SetPosition()`, `ScrollBy()`, and `HandleMouse()` methods
- `internal/ui/filelist_test.go` - New test file with 8 tests covering scroll, click, directory toggle, edge cases, and move mode
- `internal/app/app.go` - Updated `handlePlanningMouse()` for position-based routing, added `SetPosition()` call in `updateSizes()`

## Rationale
The file browser previously only supported keyboard navigation, which was inconsistent with the markdown viewer's mouse scroll support. Users expect to be able to scroll and click in sidebar panels. The implementation routes events based on mouse X position to avoid any regression in the existing markdown scroll behavior.
