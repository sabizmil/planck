# Post-Task Documentation Audit

Run a documentation audit on recent changes. This checks whether code changes are reflected in the project's specification and documentation files.

## Step 1: Identify Recent Changes

Determine what changed by examining:
- `git diff --name-only` (unstaged changes)
- `git diff --cached --name-only` (staged changes)
- If no local changes, use `git diff HEAD~1 --name-only` (last commit)
- Read the changed files to understand the nature of each change

## Step 2: Classify Changes

For each changed file, classify it:
- **Feature change**: New capabilities or changed behavior
- **Bug fix**: Corrected behavior that reveals missing or wrong spec
- **API change**: Endpoint, schema, or contract modifications
- **Architecture change**: Structural or pattern changes
- **Config change**: Environment variables, deployment settings
- **Trivial**: Formatting, typos, comments (skip documentation for these)

## Step 3: Check Documentation Impact

For non-trivial changes, check whether these documents need updates:
- `docs/PRD/` — Feature requirements and user stories
- `docs/specs/` — Technical specifications, data models, architecture
- `docs/changelog/` — Whether a changelog entry exists for this work
- `.claude/CLAUDE.md` — Project rules or structure changes

Read the relevant existing docs and compare against the code changes.

## Step 4: Update Documents

For each document that needs updating:
- Edit the document with the new information
- Include a "Last updated" timestamp
- Add a change history entry
- Preserve existing content that's still accurate

## Step 5: Report

Print a summary using this format:

```
Documentation audit complete:
- Updated docs/specs/technical-architecture.md — added new spinner config fields
- Updated docs/PRD/settings.md — added spinner customization requirement
- No changelog needed (already exists for this work)
- .claude/CLAUDE.md — no changes needed
```

If nothing needs updating, say so:
```
Documentation audit complete:
- No documentation updates needed (changes were trivial/already documented)
```

## Rules

- Only update docs for meaningful changes. Don't create noise for formatting or comment-only changes.
- Don't create new doc files unless a clear gap exists. Prefer editing existing files.
- If a changelog entry already covers the recent work, don't duplicate it.
- Be conservative — it's better to skip a marginal update than to add incorrect information.

## User's Input

$ARGUMENTS
