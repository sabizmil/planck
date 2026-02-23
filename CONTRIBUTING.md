# Contributing to Planck

Thanks for your interest in contributing to Planck! This document explains how to get involved.

## Reporting Bugs

Open a [GitHub Issue](https://github.com/sabizmil/planck/issues/new?template=bug_report.md) with:

- A clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Your environment (OS, Go version, planck version, tmux version)

## Suggesting Features

Open a [GitHub Issue](https://github.com/sabizmil/planck/issues/new?template=feature_request.md) describing:

- The problem you're trying to solve
- Your proposed solution
- Any alternatives you've considered

## Development Setup

```bash
# Clone the repo
git clone https://github.com/sabizmil/planck.git
cd planck

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

### Requirements

- Go 1.24+
- tmux (optional, for tmux-based sessions)
- golangci-lint (for linting)

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b my-feature`)
3. Make your changes
4. Run `make fmt && make lint && make test` to verify
5. Commit with a descriptive message (see commit style below)
6. Push to your fork and open a Pull Request

### Commit Messages

We use conventional-style commit prefixes for changelog generation:

- `feat:` — new feature
- `fix:` — bug fix
- `perf:` — performance improvement
- `docs:` — documentation only
- `test:` — adding or updating tests
- `chore:` — maintenance, CI, dependencies
- `refactor:` — code changes that neither fix a bug nor add a feature

### Code Style

- Follow standard Go conventions (`go fmt`, `goimports`)
- Use table-driven tests
- Return errors instead of panicking
- Keep functions small and focused

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Questions?

Open a GitHub Issue or start a Discussion — happy to help.
