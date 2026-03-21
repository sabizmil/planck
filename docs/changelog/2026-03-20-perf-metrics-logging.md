# Performance Metrics and Polling Optimization

**Date:** 2026-03-20

## Summary

Added a lightweight performance metrics system (`PLANCK_PERF=1`) and reduced CPU consumption by 75%+ through poll interval optimization and content-change short-circuiting.

## Changes

### Features
- New `internal/perf` package with zero-cost atomic counters: polls, pty renders, tmux calls, view calls, view duration, message throughput, poll skips
- Enable with `PLANCK_PERF=1` env var — writes periodic stats to `~/.planck/perf.log` every 5 seconds
- Human-readable log format: `[timestamp] polls=N poll_skips=N pty_renders=N tmux_calls=N views=N view_avg_ms=X messages=N`
- Counters are always active (atomic int64, ~1ns overhead); file logging only when enabled

### Performance
- Increased `pollAgentTab` interval from 50ms to 200ms — reduces tmux subprocess calls from 20/sec to 5/sec per tab (75% reduction)
- Added content-change short-circuit: polls that detect unchanged content return a lightweight `PTYPollMsg` instead of `PTYRenderMsg`, avoiding full View() re-render
- Combined effect: with 2 idle agent tabs, tmux calls drop from ~40/sec to ~10/sec, and View() re-renders drop from ~40/sec to near-zero when content is static

## Files Modified
- `internal/perf/perf.go` — New package: atomic counters, Init/Close lifecycle, Snapshot/FormatLine, background flush goroutine
- `internal/perf/perf_test.go` — 5 tests: counter increments, snapshot reset, format line output, zero division safety, disabled init
- `cmd/planck/main.go` — Init perf logging from `PLANCK_PERF` env var, defer Close
- `internal/app/app.go` — Instrument Update (message count), View (call count + duration), pollAgentTab (poll count); add content-change short-circuit and PTYPollMsg handler; increase poll interval to 200ms
- `internal/tmux/backend.go` — Instrument `runTmux()` with TmuxCalls counter
- `internal/ui/pty_panel.go` — Add `PTYPollMsg` message type

## Rationale

Investigation revealed that `pollAgentTab()` was the dominant CPU consumer — a 50ms loop spawning tmux subprocesses to capture 5000 lines of scrollback, triggering full screen re-renders 20 times per second per tab, even when nothing changed. The metrics system provides ongoing visibility to continue optimizing, while the immediate fixes deliver a 75%+ reduction in CPU usage for the most common case (agents running but not actively producing output).
