# Open-Source Readiness

**Date:** 2026-02-23

## Summary

Prepare the repository for public release by fixing the module identity, updating the license, adding community files, and fixing CI configuration.

## Changes

### Repository Identity
- Renamed Go module from `github.com/anthropics/planck` to `github.com/sabizmil/planck`
- Updated all internal import paths across 6 Go source files
- Updated `.goreleaser.yml`, `.golangci.yml`, `.claude/CLAUDE.md`, and `README.md` to use correct owner

### License
- Updated copyright from "2024 Anthropic" to "2025 Simon Abizmil"

### Community Files
- Added `CONTRIBUTING.md` with development setup, commit conventions, and PR process
- Added `SECURITY.md` pointing to GitHub Security Advisories
- Added `CODE_OF_CONDUCT.md` (Contributor Covenant v2.1)
- Added `.github/ISSUE_TEMPLATE/bug_report.md` for structured bug reports
- Added `.github/ISSUE_TEMPLATE/feature_request.md` for structured feature requests

### CI & Build Configuration
- Updated Go version from 1.22/1.23 to 1.24 across all GitHub Actions workflows
- Removed Homebrew `brews:` section from `.goreleaser.yml` (deferred until tap is created)

### README
- Added CI status badge and MIT license badge
- Removed nonexistent Homebrew installation section
- Updated contributing reference to link to `CONTRIBUTING.md`

## Files Modified
- `go.mod`, `go.sum` — module path rename
- `cmd/planck/main.go` — import paths and help text URL
- `internal/app/app.go` — import paths
- `internal/session/manager.go` — import path
- `internal/ui/event_renderer.go` — import path
- `internal/ui/filelist.go` — import path
- `internal/ui/pty_panel.go` — import path
- `.goreleaser.yml` — owner, homepage, removed brews section
- `.golangci.yml` — local-prefixes
- `.claude/CLAUDE.md` — module reference
- `.github/workflows/ci.yml` — Go version matrix
- `.github/workflows/release.yml` — Go version
- `LICENSE` — copyright holder and year
- `README.md` — badges, URLs, removed Homebrew section, contributing link
- `CONTRIBUTING.md` — new file
- `SECURITY.md` — new file
- `CODE_OF_CONDUCT.md` — new file
- `.github/ISSUE_TEMPLATE/bug_report.md` — new file
- `.github/ISSUE_TEMPLATE/feature_request.md` — new file
- `docs/changelog/2026-02-23-open-source-readiness.md` — new file

## Rationale

The repository was built with `github.com/anthropics/planck` as the module path but lives at `github.com/sabizmil/planck`. This mismatch would break `go install` for external users once the repo is public. All references have been unified under the correct owner. Community files and CI fixes bring the project to open-source standards.
