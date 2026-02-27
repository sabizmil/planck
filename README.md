# Planck


[![CI](https://github.com/sabizmil/planck/actions/workflows/ci.yml/badge.svg)](https://github.com/sabizmil/planck/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A terminal UI for AI-assisted planning and task management, orchestrating multiple agent sessions in a tabbed interface.

<img width="1472" height="991" alt="Screenshot 2026-02-24 at 4 46 49 PM" src="https://github.com/user-attachments/assets/4a82d5df-77df-4e1b-86e8-53b4cf14009d" />

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
planck update            # Download and install the latest version
planck update --check    # Check for updates without installing
```

## Requirements

- Claude Code CLI or other AI agent CLI
- tmux (optional — enables session persistence across SSH disconnects)

## Quick Start

```bash
# Start in the current directory
planck

# Or specify a project folder
planck --folder /path/to/project
planck -f /path/to/project
```

1. Planck opens to the **Planning tab** with your workspace files
2. Press `n` to create a new markdown file
3. Press `Enter` to open a file, then `e` to edit
4. Press `a` to open a new agent tab
5. Press `i` to enter input mode, type your prompt, and watch the agent work

## Features

### Multi-Agent Tabs

Run up to 8 concurrent agent sessions, each in its own tab with an independent PTY. Tabs display animated activity spinners while agents are running, and auto-generate titles via OSC escape sequences. Switch tabs with `Shift+Tab`, `Alt+1-9`, `1-9`, or by clicking the tab bar.

### Built-in Editor

A full-featured markdown editor with soft word-wrapping at word boundaries. Visual continuation markers (`·`) show wrapped lines. Click-and-drag to select text with real-time highlighting. Press `Escape` to save, or view rendered markdown in read-only mode.

### Workspace Browser

Recursively scans for markdown files and displays them in a tree view with mouse support (click to select, scroll to navigate, click folders to expand/collapse). Track file status (pending, in-progress, completed) via YAML frontmatter. Toggle completion with `c`. Move or rename files and folders with `m` for an interactive destination picker. File changes are detected automatically via filesystem watching.

### PTY Sessions with Scrollback

Agent sessions run in-process via PTY by default. A 1000-line ring buffer provides smooth scrollback with mouse wheel or keyboard (`j/k`, `g/G`, `PgUp/PgDn`). A scroll indicator badge shows your position. Alternate screen applications (vim, less, fzf) are filtered from the buffer.

### Session Persistence

When tmux is available, planck can run inside a persistent tmux session that survives SSH disconnects and terminal closures. Use `planck attach` to create or reattach to a session. Each project folder gets its own isolated session. Configure the backend in settings or `config.toml`:

```bash
planck attach                       # Current directory
planck attach --folder /path/to    # Specific folder
# Detach with Ctrl+B d, reattach with: planck attach
```

### Markdown Themes

Five built-in rendering themes: **Neo-Brutalist**, **Terminal Classic**, **Minimal Modern**, **Rich Editorial**, and **Soft Pastel**. Switch themes in settings with a live preview. Per-element style overrides are supported.

### Configurable Spinners

Choose from 27 animated spinner presets (claude, dot-pulse, dots, line, star, flip, bounce, box-bounce, arc, circle, circle-half, square-corners, triangle, binary, toggle, arrow, balloon, noise, grow-h, grow-v, layer, moon, hearts, clock, point, meter, breathe). Each has its own tick interval and animation style. Preview them live in the settings panel.

### Settings Panel

A composable, multi-page settings panel accessible with `s`:

- **Markdown** - Theme selection with live preview, per-element style overrides
- **General** - Editor, terminal bell, session backend, sidebar width, execution preferences
- **Agents** - Add/configure agents, set default, customize planning/implementation args
- **Keys** - Reference of all keyboard shortcuts by context
- **Spinner** - Scrollable preset list with animated previews

All changes auto-save to `config.toml`.

### Notification System

Terminal bell notifications on session events (planning complete, task done, errors). Configurable via settings.

## Key Bindings

### Global
| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` (×2) | Quit (double-tap safety) |
| `?` | Toggle help overlay |
| `Shift+Tab` | Next tab |
| `1`–`9` | Jump to tab by number (normal mode) |
| `Alt+1`–`Alt+9` | Jump to tab by number (works in all modes) |
| `a` | Create new agent tab |
| `x` / `Ctrl+X` | Close current agent tab |
| `s` | Open settings |
| `Esc` | Cancel / close dialog |

Mouse: click tab bar to switch tabs.

### File List (Planning Tab)
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate files |
| `Enter` | Open file in editor |
| `e` | Enter edit mode |
| `n` | New file |
| `d` | Delete file or folder |
| `c` | Toggle file completion |
| `m` | Move / rename file or folder |
| `r` | Refresh file list |
| `h/l` or `←/→` | Collapse / expand folders |

Mouse: click to select, scroll to navigate, click folders to toggle.

### Editor
| Key | Action |
|-----|--------|
| `Esc` | Save and close |
| `Ctrl+S` | Save in place |
| Arrow keys | Navigate (respects word wrap) |

Mouse: click to position cursor, drag to select text.

### Agent Tab — Input Mode
| Key | Action |
|-----|--------|
| `Ctrl+\` | Exit to normal mode |
| `Tab` | Sent to agent (autocomplete) |
| `Shift+Tab` | Next tab |
| `Alt+1`–`Alt+9` | Jump to tab |
| `Ctrl+X` | Close tab |

Mouse: scroll wheel to browse output history.

### Agent Tab — Normal Mode
| Key | Action |
|-----|--------|
| `i` / `Enter` | Enter input mode |
| `j/k` | Scroll up/down |
| `g/G` | Jump to top/bottom |
| `PgUp/PgDn` | Page scroll |
| `x` | Close tab |
| `a` | New agent tab |

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
sidebar_width = 28

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
├── cmd/planck/          # Entry point, CLI subcommands, self-update
├── internal/
│   ├── app/             # Main Bubble Tea model, orchestrates all components
│   ├── agent/           # Agent interface and Claude Code integration
│   ├── config/          # TOML configuration management
│   ├── notify/          # Terminal bell notification system
│   ├── session/         # PTY backend for interactive agent sessions
│   ├── store/           # SQLite persistence (WAL mode)
│   ├── tmux/            # Tmux integration (session persistence backend)
│   ├── ui/              # UI components (editor, file list, PTY panel, settings, help)
│   ├── updater/         # Self-update via GitHub Releases
│   ├── vt/              # Local fork of VT emulator with scroll callbacks
│   └── workspace/       # Markdown file discovery and status tracking
└── docs/                # Documentation and changelogs
```

## Development

```bash
make setup          # Set up dev environment (git hooks)
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

Run `make setup` after cloning to install the pre-commit hook, which auto-formats Go files and runs lint before each commit.

## License

MIT License - see [LICENSE](LICENSE) for details.
