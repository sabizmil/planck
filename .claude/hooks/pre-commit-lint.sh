#!/usr/bin/env bash
# Claude Code PreToolUse hook: run golangci-lint before git commit/push.
# Exits 2 to block the tool call and feed errors back to Claude for self-correction.

set -euo pipefail

INPUT=$(cat)

# Extract the bash command
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Only intercept git commit and git push commands
if ! echo "$COMMAND" | grep -qE '^\s*git\s+(commit|push)'; then
    exit 0
fi

# Resolve golangci-lint
LINT_CMD=""
if command -v golangci-lint >/dev/null 2>&1; then
    LINT_CMD="golangci-lint"
else
    GOBIN="${GOPATH:-$HOME/go}/bin"
    if [ -x "$GOBIN/golangci-lint" ]; then
        LINT_CMD="$GOBIN/golangci-lint"
    fi
fi

if [ -z "$LINT_CMD" ]; then
    echo "warning: golangci-lint not found, skipping pre-commit lint" >&2
    exit 0
fi

# Run lint on the whole project
LINT_OUTPUT=$($LINT_CMD run ./... 2>&1) || {
    echo "Lint errors found — fix these before committing:" >&2
    echo "" >&2
    echo "$LINT_OUTPUT" >&2
    exit 2
}

exit 0
