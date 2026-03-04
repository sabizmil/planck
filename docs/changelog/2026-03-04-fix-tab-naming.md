# Fix Tab Naming — OSC Title Overwrites User-Input Title

**Date:** 2026-03-04

## Summary

Fixed a bug where agent tab names always showed "Claude Code" instead of a description of the user's task. The generic OSC program title from Claude Code was overwriting the user-input-derived title.

## Changes

### Bug Fixes
- Added `titleFromInput` flag to `AgentTab` to track whether the custom title came from user input
- Added `isGenericOSCTitle()` helper to detect when an OSC title is just a default program name (case-insensitive containment check against the agent's base label)
- Updated OSC title handling in `Update()` to skip overwriting user-input titles with generic program names
- User-typed prompts now persist as tab titles even when Claude Code emits its default "Claude Code" OSC title
- Truly descriptive OSC titles (e.g., task-specific names) can still override input-derived titles

### Tests
- Added unit tests for `isGenericOSCTitle` covering exact match, substring, case-insensitive, and unrelated titles
- Added unit tests for `sanitizeTabTitle` covering spinner stripping, length validation, and edge cases

## Files Modified
- `internal/app/app.go` — Added `titleFromInput` field, `isGenericOSCTitle()` function, and updated OSC title handling logic
- `internal/app/app_test.go` — New test file with tests for `isGenericOSCTitle` and `sanitizeTabTitle`

## Rationale

Claude Code sets its OSC window title to its program name "Claude Code" (with optional spinner characters), not to a task description. The existing `trackInputForTitle` mechanism correctly captured user prompts as tab titles, but the generic OSC title overwrote them within ~50ms. The fix preserves user-input titles by comparing incoming OSC titles against the agent's base label and skipping generic matches.
