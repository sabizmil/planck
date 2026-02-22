# Review Changes Against Specs

Review the current branch's code changes against the project's specification files to catch spec drift before committing.

## Step 1: Gather Changes

Collect all changes on the current branch:
- `git diff --name-only` for unstaged changes
- `git diff --cached --name-only` for staged changes
- `git diff main...HEAD --name-only` for all branch changes vs main
- Read the changed files to understand what was modified

## Step 2: Identify Relevant Specs

Based on the changed files, identify which specification documents are relevant:
- `docs/PRD/*.md` — Feature requirements
- `docs/specs/*.md` — Technical specifications
- `.claude/CLAUDE.md` — Project structure and conventions

Read the relevant spec files.

## Step 3: Compare Code vs Specs

For each relevant spec, check:
- **Missing specs**: Code implements something not described in any spec
- **Contradicted specs**: Code behavior differs from what the spec says
- **Outdated specs**: Spec describes something the code has moved past
- **Naming drift**: Code uses different names/terms than the spec

## Step 4: Report

Print a structured report:

```
Review: <branch-name> vs specifications

Spec Alignment:
  - docs/specs/technical-architecture.md — OK (matches code)
  - docs/PRD/settings.md — DRIFT: spec says X, code does Y

Missing Documentation:
  - New feature Z has no spec coverage
  - Config field `foo_bar` not documented

Suggestions:
  - Update docs/PRD/settings.md to reflect new spinner page
  - Add API documentation for new endpoint
  - No action needed (specs are up to date)
```

## Rules

- Be specific about what drifted — quote the spec and describe the code behavior
- Don't flag trivial differences (e.g., slightly different wording that means the same thing)
- Focus on functional and behavioral drift, not cosmetic differences
- If everything aligns, say so clearly — a clean report is useful information
- Don't make changes — this is a read-only review. Suggest updates but let the user decide.

## User's Input

$ARGUMENTS
