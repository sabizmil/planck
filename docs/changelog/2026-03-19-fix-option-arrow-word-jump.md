# Fix Option+Arrow Typing Letters Instead of Word-Jumping

**Date:** 2026-03-19

## Summary

Fixed a bug where pressing Option+Left/Right in the editor's edit mode inserted "b" and "f" characters instead of jumping to the previous/next word boundary.

## Changes

### Bug Fixes
- Option+Left (Alt+b) now correctly jumps to the previous word boundary instead of inserting "b"
- Option+Right (Alt+f) now correctly jumps to the next word boundary instead of inserting "f"
- Option+Shift+Left/Right (Alt+B/F) now selects to the previous/next word boundary instead of inserting letters
- All Alt+rune combos are now blocked from text insertion (prevents the entire class of bugs)
- Alt+Space no longer inserts a space

## Files Modified
- `internal/ui/editor.go` — Added Alt+rune guard in `KeyRunes` case of `updateEditMode()`, mapping Alt+b/f to word jump, Alt+B/F to word selection, and dropping all other Alt+rune combos; added Alt guard to `KeySpace` case
- `internal/ui/editor_test.go` — Added 6 tests: Alt+b word-left, Alt+f word-right, Alt+Shift+B select-left, Alt+Shift+F select-right, Alt+other no-insert, Alt+Space no-insert

## Rationale

On macOS, Option+Left/Right sends `ESC b` / `ESC f` (readline word-jump sequences). Bubble Tea parses these as `KeyRunes` with `Alt: true` rather than `KeyLeft`/`KeyRight` with the Alt modifier. The rune handler was unconditionally inserting the character, ignoring the Alt flag. The fix guards all text insertion against Alt and explicitly maps the readline sequences to their correct actions.
