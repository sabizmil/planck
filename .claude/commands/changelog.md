# Generate a Changelog Entry

Create a standalone changelog entry for recent work that doesn't have a formal plan.

## Step 1: Understand What Changed

- If an argument describes the change, use that as context
- Examine recent changes: `git diff --name-only`, `git diff --cached --name-only`, or `git log --oneline -5`
- Read the changed files to understand the nature and scope of the changes

## Step 2: Generate the Changelog

Create a file at `docs/changelog/YYYY-MM-DD-<slug>.md` using today's date and a descriptive slug.

Follow this format:

```markdown
# <Title>

**Date:** YYYY-MM-DD

## Summary
One or two sentences describing what changed.

## Changes

### <Category> (e.g., Features, Bug Fixes, Refactoring)
- Change 1
- Change 2

## Files Modified
- `path/to/file.go` - description of change

## Rationale
Why these changes were made.
```

## Rules

- Check `docs/changelog/` first to avoid duplicating an existing entry for the same work
- If a changelog already covers this work, report that and stop (don't create a duplicate)
- The slug should be concise and descriptive (e.g., "fix-tab-crash", "add-dark-mode")
- Categorize changes accurately: Features, Bug Fixes, Refactoring, Technical, Performance, etc.
- List all modified files with brief descriptions of what changed in each
- The rationale should explain *why*, not just restate *what*
- If changes span multiple categories, use multiple subsections

## User's Input

$ARGUMENTS
