# Minimize OSS Overhead

**Date:** 2026-02-24

## Summary
Removed community-governance boilerplate to match the project's actual scope: a personal tool shared casually with friends, not a formal open-source project.

## Changes

### Removed
- `CODE_OF_CONDUCT.md` — Contributor Covenant (community governance)
- `CONTRIBUTING.md` — Formal contribution guide with commit conventions
- `SECURITY.md` — Vulnerability reporting policy with SLA promises
- `.github/ISSUE_TEMPLATE/bug_report.md` — Structured bug report template
- `.github/ISSUE_TEMPLATE/feature_request.md` — Structured feature request template
- README "Contributing" section and wiki link
- Release badge from README (broken on private repos, unnecessary noise)

### Moved
- `RELEASING.md` → `docs/releasing.md` (internal reference, not contributor-facing)

### Updated
- `.goreleaser.yml` — Stopped bundling `docs/**/*` in release archives; tarballs now contain only the binary, README, and LICENSE
- `README.md` — Simplified footer to just the License section

## Files Modified
- `CODE_OF_CONDUCT.md` — deleted
- `CONTRIBUTING.md` — deleted
- `SECURITY.md` — deleted
- `RELEASING.md` — moved to `docs/releasing.md`
- `.github/ISSUE_TEMPLATE/bug_report.md` — deleted
- `.github/ISSUE_TEMPLATE/feature_request.md` — deleted
- `README.md` — removed Release badge, Contributing section, Support section
- `.goreleaser.yml` — removed `docs/**/*` from archive files

## Rationale
The project had formal OSS scaffolding (Code of Conduct, Contributing guide, Security policy, issue templates) that set expectations of a community-driven project. Since Planck is a personal tool shared informally, these files created false promises and unnecessary overhead. The LICENSE (MIT) and CI workflows remain — they're useful regardless of project formality.
