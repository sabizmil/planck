# Planck

[![CI](https://github.com/sabizmil/planck/actions/workflows/ci.yml/badge.svg)](https://github.com/sabizmil/planck/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A terminal UI for AI-assisted planning and task management, orchestrating multiple agent sessions in a tabbed interface.

<img width="1472" height="991" alt="Screenshot 2026-02-24 at 4 46 49 PM" src="https://github.com/user-attachments/assets/4a82d5df-77df-4e1b-86e8-53b4cf14009d" />

## Overview

Planck is a TUI (Terminal User Interface) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) that helps you:

1. **Manage Workspace Files** - Browse, create, and edit markdown files with status tracking
2. **Run Multiple Agents** - Launch up to 8 concurrent AI agent tabs (Claude Code, Codex, etc.)
3. **Plan & Execute** - Break down complex tasks and dispatch them to coding agents
4. **Customize Everything** - Themes, spinners, keybindings, and per-agent configuration

## Installation

### Quick Install (macOS / Linux)

```bash
curl -sSfL https://raw.githubusercontent.com/sabizmil/planck/main/scripts/install.sh | sh
```

### Using Go

```bash
go install github.com/sabizmil/planck/cmd/planck@latest
```

### From GitHub Releases

Download a pre-built binary from the [latest release](https://github.com/sabizmil/planck/releases/latest) for your platform.

### From Source

```bash
git clone https://github.com/sabizmil/planck.git
cd planck
make install
```

### Updating

```bash
planck update
```

## Requirements

- tmux (optional, for tmux-based sessions)
- Claude Code CLI or other AI agent CLI

## Quick Start

```bash
# Start in the current directory
planck

# Or specify a project folder
planck /path/to/project
```

1. Planck opens to the **Planning tab** with your workspace files
2. Press `n` to create a new markdown file
3. Press `Enter` to open a file in the editor
4. Press `t` to open a new agent tab and start an AI session
5. Type your prompt and watch the agent work

## Features

### Multi-Agent Tabs

Run up to 8 concurrent agent sessions, each in its own tab with an independent PTY. Tabs display animated activity spinners while agents are running, and auto-generate titles from your input.

### Built-in Editor

A full-featured markdown editor with soft word-wrapping at word boundaries. Visual continuation markers show wrapped lines. Press `Escape` to save, or view rendered markdown in read-only mode.

### Workspace Browser

Recursively scans for markdown files and displays them in a tree view. Track file status (pending, in-progress, completed) via YAML frontmatter. Toggle completion with `c`. File changes are detected automatically via filesystem watching.

### PTY Sessions with Scrollback

Agent sessions run in-process via PTY (no tmux overhead by default). A 1000-line ring buffer provides smooth scrollback with mouse wheel or keyboard navigation. Alternate screen applications (vim, less, fzf) are filtered from the buffer.

### Markdown Themes

Five built-in rendering themes: **Neo-Brutalist**, **Terminal Classic**, **Minimal Modern**, **Rich Editorial**, and **Soft Pastel**. Switch themes in settings with a live preview. Per-element style overrides are supported.

### Configurable Spinners

Choose from 26 animated spinner presets (claude, dots, line, star, flip, bounce, arc, circle, moon, hearts, clock, and more). Each has its own tick interval and animation style. Preview them live in the settings panel.

### Settings Panel

A composable, multi-page settings panel accessible with `s`:

- **General** - Editor, terminal bell, session backend, execution preferences
- **Agents** - Add/configure agents, set default, customize command args
- **Markdown** - Theme selection with live preview, per-element overrides
- **Spinner** - Scrollable preset list with animated previews
- **Keybindings** - Reference of all keyboard shortcuts by context

All changes auto-save to `config.toml`.

### Notification System

Terminal bell notifications on session events (planning complete, task done, errors). Configurable via settings.

## Key Bindings

### Global
| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` (×2) | Quit (double-tap safety) |
| `?` | Toggle help overlay |
| `Tab` / `Shift+Tab` | Cycle tabs forward/backward |
| `1`–`9` | Jump to tab by number |
| `s` | Open settings |
| `Esc` | Cancel / close dialog |

### File List (Planning Tab)
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate files |
| `Enter` | Open file in editor |
| `n` | New file |
| `d` | Delete file |
| `c` | Toggle file completion |
| `t` | New agent tab |
| `h/l` | Collapse/expand folders |

### Editor
| Key | Action |
|-----|--------|
| `Esc` | Save and close |
| `Ctrl+S` | Save in place |
| Arrow keys | Navigate (respects word wrap) |

### Agent Tab
| Key | Action |
|-----|--------|
| `i` | Enter input mode |
| `Esc` | Exit input mode |
| `j/k` | Scroll up/down |
| `g/G` | Jump to top/bottom |
| `PgUp/PgDn` | Page scroll |

## Configuration

Planck stores configuration in `.planck/config.toml`:

```toml
[[agents]]
command = "claude"
label = "Claude"
planning_args = ["--verbose"]
implementation_args = ["--dangerously-skip-permissions"]
default = true

[preferences]
editor = "vim"
spinner_style = "claude"

[notifications]
bell = true

[execution]
default_scope = "phase"
auto_advance = true
permission_mode = "pre-approve"

[session]
backend = "auto"  # auto | tmux | pty

[markdown_style]
theme = "neo-brutalist"
```

## Architecture

```
planck/
├── cmd/planck/          # Entry point, folder picker, version info
├── internal/
│   ├── app/             # Main Bubble Tea model, orchestrates all components
│   ├── agent/           # Agent interface and Claude Code integration
│   ├── config/          # TOML configuration management
│   ├── notify/          # Terminal bell notification system
│   ├── session/         # PTY backend for interactive agent sessions
│   ├── store/           # SQLite persistence (WAL mode)
│   ├── tmux/            # Tmux integration (alternative backend)
│   ├── ui/              # UI components (editor, file list, PTY panel, settings, help)
│   ├── vt/              # Local fork of VT emulator with scroll callbacks
│   └── workspace/       # Markdown file discovery and status tracking
└── docs/                # Documentation and changelogs
```

## Development

```bash
make build          # Build binary to build/planck
make test           # Run all tests with race detector
make test-short     # Run short tests (skip integration)
make test-coverage  # Tests with coverage report (outputs coverage.html)
make lint           # Run golangci-lint
make fmt            # Format code (go fmt + goimports)
make run            # Build and run
make dev            # Hot reload with air
make build-all      # Cross-platform binaries (darwin/linux, amd64/arm64)
```

## License

MIT License - see [LICENSE](LICENSE) for details.
