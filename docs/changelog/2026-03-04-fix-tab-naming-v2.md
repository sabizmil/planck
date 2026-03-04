# Fix Tab Naming v2 — titleFromInput Actually Set

**Date:** 2026-03-04

## Summary

Fixed the tab naming guard that was supposed to prevent generic OSC titles ("Claude Code") from overwriting user-input-derived titles. The `titleFromInput` flag was never being set to `true`, making the guard always evaluate to `false`. Also fixed session recovery to preserve title protection across Planck restarts.

## Changes

### Bug Fixes
- Added missing `tab.titleFromInput = true` in `trackInputForTitle` — the critical line that enables the OSC title guard
- Set `titleFromInput = true` when recovering sessions with a non-empty `customTitle`, preventing OSC overwrites after Planck restart

### Tests
- Added `TestTitleFromInput_ProtectsAgainstGenericOSC` — full lifecycle test: user input → generic OSC blocked → descriptive OSC allowed
- Added `TestTitleFromInput_InputBufHandling` — verifies character accumulation, backspace, and escape clearing
- Added `TestIsGenericOSCTitle_RealWorldScenario` — tests exact production values: baseLabel="Claude", OSC="Claude Code"

## Files Modified
- `internal/app/app.go` — Added `tab.titleFromInput = true` after `tab.customTitle = title` in `trackInputForTitle`
- `internal/app/recovery.go` — Added `titleFromInput: dbSess.CustomTitle != ""` in recovered `AgentTab` initialization
- `internal/app/app_test.go` — Added 3 new integration/scenario tests

## Rationale

The v1 fix (isGenericOSCTitle + titleFromInput flag) had the correct architecture but the critical `titleFromInput = true` assignment was missing from `trackInputForTitle`. Without it, the guard condition `tab.titleFromInput && isGenericOSCTitle(...)` was always false, so "Claude Code" always overwrote user-typed titles. The recovery gap was also addressed to ensure titles survive Planck restarts.
