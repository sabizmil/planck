# Distribution & Release Automation

**Date:** 2026-02-23

## Summary

Established a complete distribution pipeline for Planck: fixed existing GoReleaser/CI infrastructure, added an install script for frictionless installation, added a self-update command, and updated documentation with release instructions.

## Changes

### Bug Fixes
- Fixed `main.go` variable name mismatch: `date` was never set because ldflags inject `buildTime` — renamed to `buildTime`
- Fixed `ci.yml` coverage step condition referencing Go 1.23 (not in matrix) — updated to 1.24

### Features
- Added `planck update` subcommand — downloads and installs the latest release from GitHub, with SHA256 checksum verification and atomic binary replacement
- Added `planck update --check` — checks for updates without installing
- Added `planck version` subcommand with `--check` flag to check for updates
- Added `scripts/install.sh` — cross-platform install script that detects OS/arch, downloads from GitHub Releases, verifies checksums, and installs to `/usr/local/bin` or `~/.local/bin`
- Detects Homebrew-managed installations and advises `brew upgrade` instead of self-update

### Infrastructure
- Updated `.goreleaser.yml` to use non-deprecated `formats` (list) syntax instead of `format` (string)
- Verified CGO_ENABLED=0 cross-compilation works with modernc.org/sqlite (pure Go)
- Verified GoReleaser snapshot builds all 4 platform targets successfully

### Documentation
- Added release version badge to README
- Expanded README installation section with install script, GitHub Releases, self-update instructions
- Removed Go 1.24 from requirements (not needed for binary installs)
- Created RELEASING.md documenting the full release process

## Files Modified
- `cmd/planck/main.go` — fixed `date` → `buildTime` variable, added `update` and `version` subcommands, updated help text
- `internal/updater/updater.go` — new package: GitHub Releases API client, checksum verification, binary extraction, atomic self-update
- `scripts/install.sh` — new: cross-platform install script
- `.goreleaser.yml` — migrated deprecated `format` → `formats` syntax
- `.github/workflows/ci.yml` — fixed coverage step Go version condition
- `README.md` — expanded installation section, added version badge
- `RELEASING.md` — new: release process documentation

## Rationale

Planck had the scaffolding for releases (GoReleaser, GitHub Actions) but several configuration mismatches and no actual distribution mechanism. This change makes it ready for the first release (v0.1.0) with multiple install paths: install script, go install, GitHub Releases download, and built-in self-update.
