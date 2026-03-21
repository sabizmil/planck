# Fix Idle Detection Broken by Content-Change Short-Circuit

**Date:** 2026-03-20

## Summary

Fixed a bug where agent tabs never transitioned from "running" to "idle" (or "needs_input") after the content-change polling optimization was added. The `PTYPollMsg` short-circuit bypassed all idle detection logic, leaving tabs stuck with a spinner indefinitely.

## Changes

### Bug Fixes
- Extracted idle/needs_input detection into a `checkIdleTransition(tab)` method on App, creating a single source of truth for the running → idle / running → needs_input state machine
- Added `checkIdleTransition` call to the `PTYPollMsg` handler so idle detection runs even when content hasn't changed (the common case after an agent finishes work)
- Replaced inline idle detection in the `PTYRenderMsg` handler with a call to the shared helper

### Tests
- Added `TestCheckIdleTransition` with 7 table-driven subtests covering: no transition from non-running states, user-typing guard, 3-second idle timeout, zero-value lastContentChange guard, hook state detection, hook-state-over-idle priority, and completed-status immutability

## Files Modified
- `internal/app/app.go` — Added `checkIdleTransition` method; replaced inline idle detection in PTYRenderMsg handler; added call in PTYPollMsg handler
- `internal/app/app_test.go` — Added `TestCheckIdleTransition` with 7 subtests

## Rationale

The `PTYPollMsg` content-change optimization (added earlier today) correctly avoided unnecessary View() re-renders by returning a lightweight message when terminal content hadn't changed. However, the `PTYPollMsg` handler only re-polled without checking for state transitions. Since the idle detection logic only lived in the `PTYRenderMsg` handler — which is never sent once content stabilizes — agent tabs could never reach the "idle" state. Extracting the logic into a shared helper fixes the gap while preserving the performance optimization.
