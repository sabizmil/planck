# Fix CLI Argument Handling

**Date:** 2026-02-25

## Summary
Improved CLI argument parsing to handle subcommands correctly even when flags appear before them, reject unknown positional arguments with helpful errors, and register the missing `-f` short flag for `--folder`.

## Changes

### Bug Fixes
- Subcommands (`update`, `version`, `attach`) now correctly detected even when flags precede them (e.g., `planck --folder /path update`)
- Unknown positional arguments now produce a clear error instead of being silently ignored
- Added `-f` short flag for `--folder` to match the documented help text

### Refactoring
- Replaced simple `os.Args[1]` switch with `findSubcommand()` that scans all args, skipping flags and their values
- Extracted `knownSubcommands` map for reuse in both subcommand detection and error messages

## Files Modified
- `cmd/planck/main.go` — rewrote subcommand detection logic, added `-f` flag, added positional arg validation
- `cmd/planck/main_test.go` — new file with table-driven tests for `findSubcommand` edge cases

## Rationale
Users could encounter a confusing "folder does not exist" error when running `planck update` if flags were placed before the subcommand, causing the flag parser to consume the subcommand name as a flag value. The fix makes the CLI robust against all argument orderings.
