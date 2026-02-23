# Fix all golangci-lint errors for CI

**Date:** 2026-02-22

## Summary
Fixed ~60 golangci-lint errors across 27 files to pass CI. No behavioral changes.

## Changes

### Lint Fixes
- **errcheck**: Added explicit error handling or `_ =` discards for unchecked return values (12 instances)
- **gofmt**: Auto-formatted 10 files with `gofmt -w`
- **gocritic/octalLiteral**: Modernized octal literals to `0o` prefix style (9 instances)
- **gocritic/ifElseChain**: Rewrote if-else chains to switch statements (~10 instances)
- **gocritic/singleCaseSwitch**: Rewrote single-case type switches to if statements (8 instances)
- **gocritic/emptyStringTest**: Replaced `len(s) > 0` with `s != ""` (2 instances)
- **gocritic/sprintfQuotedString**: Replaced `"%s"` format with `%q` (2 instances)
- **gocritic/other**: Fixed commentedOutCode, dupBranchBody, evalOrder warnings
- **unparam**: Renamed unused parameters to `_` or removed always-nil return values (7 instances)
- **misspell**: Renamed `StatusCancelled` to `StatusCanceled` and updated all references (5 instances)
- **staticcheck/SA9003**: Fixed empty if-branches with explicit discards or removed them (5 instances)
- **unused**: Removed unused `detailIdx` field from `agentsPage`
- **ineffassign**: Removed ineffectual `maxWidth` assignment in dialog

## Files Modified
- `cmd/planck/main.go` - errcheck, SA9003
- `internal/agent/claude.go` - errcheck, SA9003
- `internal/app/app.go` - errcheck, ifElseChain, singleCaseSwitch, sprintfQuotedString, unparam
- `internal/config/config.go` - octalLiteral
- `internal/notify/notify.go` - gofmt
- `internal/session/manager.go` - SA9003, canceled rename
- `internal/session/pty.go` - errcheck, octalLiteral, paramTypeCombine, SA9003, commentedOutCode, gofmt
- `internal/session/session.go` - misspell (canceled rename)
- `internal/session/session_test.go` - misspell (canceled rename)
- `internal/store/store.go` - octalLiteral, misspell, gofmt
- `internal/ui/dialog.go` - singleCaseSwitch, emptyStringTest, ineffassign
- `internal/ui/editor.go` - gofmt
- `internal/ui/event_renderer.go` - unparam
- `internal/ui/filelist.go` - singleCaseSwitch, gofmt
- `internal/ui/folderpicker.go` - singleCaseSwitch, emptyStringTest, misspell (canceled rename)
- `internal/ui/help.go` - singleCaseSwitch
- `internal/ui/markdown_style.go` - errcheck, gofmt
- `internal/ui/settings.go` - ifElseChain, gofmt
- `internal/ui/settings_agents.go` - unused field, unparam, ifElseChain, gofmt
- `internal/ui/settings_general.go` - unparam, ifElseChain, singleCaseSwitch
- `internal/ui/settings_keybindings.go` - gofmt
- `internal/ui/settings_markdown.go` - ifElseChain, dupBranchBody
- `internal/ui/settings_spinner.go` - unparam, ifElseChain
- `internal/ui/tabs.go` - evalOrder, ifElseChain
- `internal/ui/theme.go` - gofmt
- `internal/ui/theme_test.go` - SA9003
- `internal/workspace/workspace.go` - errcheck, octalLiteral, sprintfQuotedString, ifElseChain

## Rationale
CI was failing due to golangci-lint v1.64.8 reporting errors across the codebase. All fixes are mechanical/stylistic with no behavioral changes.
