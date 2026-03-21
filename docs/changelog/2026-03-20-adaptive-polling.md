# Adaptive Polling Based on Tab State

**Date:** 2026-03-20

## Summary

Replaced the fixed 200ms poll interval with adaptive polling that adjusts based on tab state — fast for the active running tab, slow for background tabs, and near-instant after user keystrokes.

## Changes

### Performance
- Active tab + running: 50ms (responsive for typing and live output)
- Active tab + idle/needs_input: 500ms (agent paused, low priority)
- Background tab + running: 500ms (user can't see it, content-change check prevents wasted re-renders)
- Background tab + idle: 1000ms (minimal overhead, just detecting resume)
- Immediate 16ms poll after user keystroke (ensures echo appears within one frame)

### Architecture
- `pollAgentTab` now accepts a `time.Duration` interval parameter
- New `pollIntervalForTab` method computes interval from active tab state and panel status
- `PTYWriteMsg` handler schedules an immediate fast poll for instant keystroke echo
- Existing content-change short-circuit (`PTYPollMsg`) ensures duplicate polls from keystroke+pending are free

## Files Modified
- `internal/app/app.go` — Added poll interval constants, `pollIntervalForTab()` method, parameterized `pollAgentTab()`, added immediate poll on `PTYWriteMsg`
- `internal/app/app_test.go` — Added `TestPollIntervalForTab` with 6 sub-tests covering all tab state combinations
- `internal/app/recovery.go` — Updated `pollAgentTab` call to use adaptive interval

## Rationale

The fixed 200ms interval was a blunt instrument — it made typing feel laggy while still burning CPU on idle background tabs. Adaptive polling gives the best of both worlds: the active tab polls at the original responsive 50ms rate, while background and idle tabs contribute minimal CPU. The immediate poll on keystroke ensures typing echo is never delayed regardless of the current poll cycle.
