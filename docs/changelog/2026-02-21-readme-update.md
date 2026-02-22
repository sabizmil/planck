# README Update

**Date:** 2026-02-21

## Summary
Updated the top-level README.md to accurately reflect the current state of the project after recent feature additions.

## Changes

### Documentation
- Rewrote project overview to highlight the tabbed multi-agent TUI approach
- Added comprehensive Features section covering: multi-agent tabs, built-in editor with word wrap, workspace browser, PTY scrollback, markdown themes, configurable spinners, composable settings panel, and notifications
- Updated key bindings to match actual UI contexts (global, file list, editor, agent tab)
- Corrected architecture diagram to reflect real package structure (app, workspace, ui, vt, session, store, etc.)
- Updated configuration example to show current TOML format with agents, spinner, session backend, and markdown theme settings
- Fixed Go version requirement from 1.22 to 1.24
- Added CLI usage examples with folder argument
- Added all development commands including build-all and dev
- Removed outdated sections referencing non-existent packages and workflows

## Files Modified
- `README.md` - Full rewrite to align with current features and architecture

## Rationale
The README had fallen significantly out of date as features were added (multi-agent tabs, PTY sessions, settings panel, markdown themes, spinners, editor word wrap, etc.). This update ensures new users and contributors see an accurate picture of the project.
