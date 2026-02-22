# Complete a Plan

Complete a plan and archive it. Follow these steps in order:

## Step 1: Identify the Plan

- If an argument is given, find the matching plan in `.claude/plans/` (match by slug or partial name)
- If no argument is given, look for plans with `status: in-progress` in `.claude/plans/`
- If multiple plans match, ask the user which one to complete
- If no plans are found, report that and stop

## Step 2: Verify Completion

- Read the plan file
- Check that all tasks are marked `[x]` (completed)
- If any tasks remain unchecked, list them and ask the user whether to proceed anyway or finish them first

## Step 3: Generate Changelog

Create a changelog entry at `docs/changelog/YYYY-MM-DD-<slug>.md` using today's date and the plan's slug.

The changelog should follow this format:

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

To fill this in accurately:
- Use `git diff main...HEAD --name-only` or `git diff --cached --name-only` to find modified files
- Read the plan's tasks and notes for context on what changed and why
- Categorize changes appropriately (Features, Bug Fixes, Refactoring, Technical, etc.)

## Step 4: Archive the Plan

1. Update the plan's frontmatter status to `completed`
2. Move the plan file from `.claude/plans/` to `.claude/plans/archive/`

## Step 5: Report

Print a summary:
- Plan title and path (archived location)
- Changelog path
- Brief summary of what was completed

## User's Input

$ARGUMENTS
