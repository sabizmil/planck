# Planck

A terminal UI for iteratively refining plans with AI, then dispatching them to coding agents.

## Overview

Planck is a TUI (Terminal User Interface) that helps you:

1. **Create Plans** - Start with a vague idea, iterate with AI to refine approaches
2. **Structure Work** - Break down plans into phases and tasks
3. **Execute Autonomously** - Dispatch tasks to AI coding agents (Claude Code, etc.)
4. **Track Progress** - Monitor execution, receive notifications, hijack sessions

## Installation

### From Source

```bash
git clone https://github.com/anthropics/planck.git
cd planck
make install
```

### Using Go

```bash
go install github.com/anthropics/planck/cmd/planck@latest
```

### Homebrew (macOS)

```bash
brew tap anthropics/tap
brew install planck
```

## Requirements

- Go 1.22 or later
- tmux (for interactive agent sessions)
- Claude Code CLI (optional, for AI integration)

## Quick Start

1. Navigate to your project directory:
   ```bash
   cd your-project
   ```

2. Start planck:
   ```bash
   planck
   ```

3. Press `n` to create a new plan
4. Describe your idea and let AI refine it
5. Select approaches, review phases and tasks
6. Press `d` to dispatch tasks to agents

## Key Bindings

### Global
| Key | Action |
|-----|--------|
| `q` | Quit |
| `?` | Show help |
| `Tab` | Switch focus |
| `Esc` | Cancel/close dialog |

### Sidebar (Plan List)
| Key | Action |
|-----|--------|
| `n` | New plan |
| `j/k` or `↑/↓` | Navigate |
| `Enter` | Select plan |
| `D` | Delete plan |

### Plan Tree
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate |
| `h/l` | Collapse/Expand |
| `Enter` | View details |
| `Space` | Toggle task status |
| `s` | Start planning session |
| `d` | Dispatch to agent |
| `e` | Edit in $EDITOR |

### Execution
| Key | Action |
|-----|--------|
| `X` | Execute plan (scope picker) |
| `B` | Background session |
| `F` | Foreground session |
| `P` | Pause execution |
| `R` | Resume execution |
| `C` | Cancel execution |

### Session Panel
| Key | Action |
|-----|--------|
| `a` | Accept changes |
| `r` | Reject changes |
| `Enter` | Attach to session |

## Configuration

Planck stores configuration in `.planck/config.toml`:

```toml
[agents]
[[agents]]
name = "claude-code"
command = "claude"
args = ["--dangerously-skip-permissions"]

[preferences]
editor = "vim"
default_agent = "claude-code"

[notifications]
bell = true

[execution]
default_scope = "phase"
auto_advance = true
```

## Architecture

```
planck/
├── cmd/planck/          # Entry point
├── internal/
│   ├── app/             # Main application (Bubble Tea model)
│   ├── config/          # Configuration management
│   ├── store/           # SQLite persistence
│   ├── plan/            # Plan model, parsing, writing
│   ├── agent/           # AI agent integration
│   ├── session/         # Session management (tmux)
│   ├── execution/       # Autonomous execution
│   ├── notify/          # Notification system
│   └── ui/              # UI components
└── docs/                # Documentation
```

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### With Coverage

```bash
make test-coverage
```

### Linting

```bash
make lint
```

### Running

```bash
make run
```

## Workflows

### 1. Interactive Planning

1. Create a new plan with a description
2. AI suggests multiple approaches
3. Select your preferred approach
4. AI breaks down into phases
5. Review and refine phases
6. AI details tasks for each phase
7. Mark tasks as done manually or dispatch to agents

### 2. Autonomous Execution

1. Select a plan with defined tasks
2. Press `X` to start execution
3. Choose scope: task, phase, or entire plan
4. Approve permissions
5. Watch progress as tasks execute
6. Hijack sessions if needed (`Enter`)
7. Receive notifications on completion

### 3. Session Management

- **Foreground**: Watch streaming output, accept/reject
- **Background**: Run async, receive bell notifications
- **Hijack**: Jump into any running session to intervene

## Plan Format

Plans are stored as markdown files in `.planck/plans/`:

```
.planck/plans/
└── auth-refactor/
    ├── index.md      # Overview, approaches
    ├── phase-1.md    # First phase with tasks
    ├── phase-2.md    # Second phase
    └── ...
```

### Example Plan Structure

```markdown
# Authentication Refactor

Refactor authentication system to support OAuth.

## Selected Approach
Approach A: Incremental Migration

## Approaches

### Approach A: Incremental Migration
Migrate one auth method at a time...

### Approach B: Big Bang
Replace everything at once...
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions welcome! Please read our contributing guidelines.

## Support

- [GitHub Issues](https://github.com/anthropics/planck/issues)
- [Documentation](https://github.com/anthropics/planck/wiki)
