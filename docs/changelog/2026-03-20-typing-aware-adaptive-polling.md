# Typing-Aware Adaptive Polling

**Date:** 2026-03-20

## Summary

Active agent tabs now boost to fast polling (50ms) while the user is typing, regardless of agent status. Polling reverts to the normal status-based interval after 2 seconds of no keystrokes. This eliminates the lag when typing into completed or idle tabs.

## Changes

### Performance
- Added `typingBoostWindow` (2s) constant for the typing-aware polling window
- Updated `pollIntervalForTab` to check `tab.lastUserWrite` — active tabs with recent keystrokes get 50ms polling, naturally decaying to 500ms after 2s of inactivity
- No new fields or message types needed; reuses the existing `lastUserWrite` timestamp already set on every keystroke

### Tests
- Expanded `TestPollIntervalForTab` from 6 to 12 test cases covering all combinations of active/background, status, and typing recency
- Verified `TestCheckIdleTransition` is unaffected (uses independent 1s window)

## Files Modified
- `internal/app/app.go` — Added `typingBoostWindow` const, updated `pollIntervalForTab` to check `lastUserWrite` for active typing boost
- `internal/app/app_test.go` — Expanded `TestPollIntervalForTab` with typing-aware test cases

## Rationale

The adaptive polling optimization (added earlier today) correctly slowed polling for idle/completed tabs to reduce CPU. But this meant typing into a non-running active tab felt laggy — 500ms poll intervals meant up to half a second before terminal output refreshed after each keystroke. Since `lastUserWrite` is already tracked on every keystroke, checking it in `pollIntervalForTab` was a zero-cost way to restore responsiveness exactly when needed.
