# Spec Refinements - 2026-02-20

## Summary

Resolved all open questions in the planck specification and refined the architecture for multi-file plans and future Agent Teams integration.

## Changes

### Resolved Decisions

1. **Session Resume Semantics**: User chooses each time - after rejecting output, prompt "Continue this session or start fresh?"

2. **Multi-file Plans**: Support from start with directory-per-plan structure:
   ```
   plans/
   ├── auth-refactor/
   │   ├── index.md      # Main plan (Goal, Decisions, Approaches)
   │   ├── phase-1.md    # Phase 1 details + tasks
   │   └── phase-2.md    # Phase 2 details + tasks
   ```

3. **Plan Diffing**: Optional diff toggle with `v` key - accept directly by default

4. **Git Integration**: No auto-commit - let users control their own workflow

5. **Agent Teams Integration**: Design for it now (session abstraction layer), implement later

6. **Codex Streaming**: Deferred to Phase 4 - focus on Claude Code first

### Architecture Updates

- Added `internal/session/` package with `Backend` interface for session abstraction
- Simplified context injection to directive prompt style (Claude Code reads plan files directly)
- Added `diff_view.go` UI component for optional diff preview
- Updated SQLite schema: `file_path` → `dir_path` in plan_meta table

### UI Updates

- Session view replaces detail panel (plan tree stays visible)
- Thumbnail tmux preview (5-10 lines) with `x` to expand
- Added `v` keybinding for diff preview before accepting
- Added `r` reject with resume prompt
- Visual animations: pulsing dot for active sessions, blinking cursor for streaming

### MVP Scope Adjustments

- Phase 1: Added multi-file parsing, dark theme only, NO_COLOR support
- Phase 2: Added directive prompt style, diff preview, resume prompt
- Phase 3: Added session management abstraction layer
- Phase 4: Moved Codex support here, added Agent Teams integration

## Files Modified

- `planck-spec.md` - All specification updates

## Rationale

These refinements prioritize:
- Clean architecture that won't require painful migrations
- Flexibility without being opinionated
- MVP simplicity by deferring non-essential features
