# Automate Lint Fixes Before Commits

**Date:** 2026-02-26

## Summary

Added automated lint enforcement via git pre-commit hook and Claude Code hooks, so lint errors are caught and fixed before they reach CI.

## Changes

### Features
- Git pre-commit hook that auto-fixes gofmt/goimports on staged .go files and runs golangci-lint, blocking commits on failure
- Claude Code PostToolUse hook that auto-formats .go files immediately after Edit/Write tool calls
- Claude Code PreToolUse hook that intercepts `git commit`/`git push` and runs golangci-lint, blocking with actionable feedback on failure
- `make setup` target to install git hooks via `core.hooksPath`

### Documentation
- Updated README.md with `make setup` in the development commands
- Updated .claude/CLAUDE.md with lint automation section explaining both hook systems

## Files Modified
- `.githooks/pre-commit` - New git pre-commit hook (auto-format + lint gate)
- `.claude/hooks/format-go.sh` - New Claude Code hook for auto-formatting .go files
- `.claude/hooks/pre-commit-lint.sh` - New Claude Code hook for pre-commit lint gate
- `.claude/settings.json` - New Claude Code hook configuration (PostToolUse + PreToolUse)
- `Makefile` - Added `make setup` target and help entry
- `README.md` - Added `make setup` to development commands with explanation
- `.claude/CLAUDE.md` - Added Lint Automation section

## Rationale

Lint errors (gofmt formatting, gocritic issues like dupBranchBody/unnamedResult) were repeatedly slipping through to CI. The combined approach ensures: (1) the git hook catches everything regardless of editor, (2) the Claude Code hooks provide real-time formatting and feed lint errors back to Claude for self-correction before committing.
