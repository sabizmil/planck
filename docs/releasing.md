# Releasing Planck

This document describes how to create a new release of Planck.

## Prerequisites

- Push access to `sabizmil/planck`
- [GoReleaser](https://goreleaser.com/) installed locally (for testing)

## Release Process

### 1. Prepare the release

Ensure all changes are committed and pushed to `main`:

```bash
git status          # should be clean
make test           # all tests pass
make lint           # no lint errors
```

### 2. Test the release locally

```bash
make release-snapshot
```

This builds all platform binaries without publishing. Verify the output in `dist/`.

### 3. Tag the release

Follow [semver](https://semver.org/). The project is pre-1.0, so use `v0.x.y`:

```bash
git tag -a v0.1.0 -m "Description of this release"
git push origin v0.1.0
```

### 4. Automated pipeline

Pushing a tag triggers `.github/workflows/release.yml` which:

1. Runs the full test suite
2. Runs GoReleaser, which:
   - Builds binaries for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`
   - Creates `.tar.gz` archives with README, LICENSE, and docs
   - Generates `checksums.txt` (SHA256)
   - Creates a GitHub Release with auto-generated changelog

### 5. Verify the release

- Check the [GitHub Actions run](https://github.com/sabizmil/planck/actions/workflows/release.yml)
- Check the [Releases page](https://github.com/sabizmil/planck/releases)
- Test the install script: `curl -sSfL https://raw.githubusercontent.com/sabizmil/planck/main/scripts/install.sh | sh`
- Test self-update from a previous version: `planck update`

## Version Injection

GoReleaser injects version information via ldflags:

- `main.version` — the semver tag (e.g., `0.1.0`)
- `main.commit` — the full git commit SHA
- `main.buildTime` — the build timestamp

These are defined in `cmd/planck/main.go` and displayed by `planck --version`.

## Distribution Channels

| Channel | How it works |
|---------|-------------|
| **GitHub Releases** | Automatic via GoReleaser on tag push |
| **Install script** | `scripts/install.sh` downloads from GitHub Releases |
| **Self-update** | `planck update` downloads from GitHub Releases |
| **go install** | Works automatically once repo is public |

## Hotfix Releases

For urgent fixes on a released version:

```bash
git checkout v0.1.0
git checkout -b hotfix/v0.1.1
# ... make fixes ...
git tag -a v0.1.1 -m "Hotfix: description"
git push origin v0.1.1
```
