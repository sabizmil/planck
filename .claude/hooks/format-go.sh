#!/usr/bin/env bash
# Claude Code PostToolUse hook: auto-format .go files after Edit/Write.
# Runs gofmt and goimports on the edited file to keep formatting clean.

set -euo pipefail

INPUT=$(cat)

# Extract the file path from the tool input
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# Only act on .go files
if [ -z "$FILE_PATH" ] || [[ "$FILE_PATH" != *.go ]]; then
    exit 0
fi

# Only act on files that exist (not deletions)
if [ ! -f "$FILE_PATH" ]; then
    exit 0
fi

# Auto-format with gofmt
if command -v gofmt >/dev/null 2>&1; then
    gofmt -w "$FILE_PATH" 2>/dev/null || true
fi

# Auto-format with goimports
GOBIN="${GOPATH:-$HOME/go}/bin"
if command -v goimports >/dev/null 2>&1; then
    goimports -w "$FILE_PATH" 2>/dev/null || true
elif [ -x "$GOBIN/goimports" ]; then
    "$GOBIN/goimports" -w "$FILE_PATH" 2>/dev/null || true
fi

exit 0
