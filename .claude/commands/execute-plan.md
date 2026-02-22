# Execute Plan

Find the active plan and execute it end-to-end: work through all tasks, run tests, and complete the plan on success.

## Step 1: Find the Active Plan

- If an argument is given, match it against plan files in `.claude/plans/` (by slug or partial name)
- If no argument is given, look for plans with `status: created` or `status: in-progress` in `.claude/plans/`
- If multiple plans match, ask the user which one to execute
- If no plans are found, report that and stop

## Step 2: Read and Understand the Plan

- Read the full plan file
- Understand the goal, chosen approach, and task list
- Read any referenced source files to understand the current codebase state
- Update the plan status to `in-progress` if it's still `created`

## Step 3: Execute Tasks

Work through each unchecked task in order:

1. **Before each task**: Read the relevant code, understand the context
2. **Implement the task**: Write the code changes needed
3. **Mark it done**: Update the plan file, changing `- [ ]` to `- [x]` for the completed task
4. **Add notes**: If you discover anything important during implementation, add it to the Notes section

If a task is blocked or unclear:
- Add a note explaining the blocker
- Ask the user for guidance before proceeding
- Do NOT skip tasks silently

## Step 4: Run Tests

After all tasks are marked complete:

1. Run `make build` to verify the code compiles
2. Run `make test` to run the full test suite with race detector
3. If tests fail:
   - Diagnose the failure
   - Fix the issue
   - Re-run tests
   - Repeat until tests pass (max 3 attempts)
   - If still failing after 3 attempts, report the failures and ask the user for guidance

## Step 5: Complete the Plan

Once all tasks are done AND tests pass:

1. **Generate changelog**: Create `docs/changelog/YYYY-MM-DD-<slug>.md` with:
   - Summary of changes
   - Categorized list of changes (Features, Bug Fixes, Refactoring, etc.)
   - Files modified with descriptions
   - Rationale for the changes
   - Use `git diff` to accurately identify all modified files

2. **Archive the plan**:
   - Update the plan frontmatter to `status: completed`
   - Move the plan from `.claude/plans/` to `.claude/plans/archive/`

3. **Report completion**:
   ```
   Plan completed: <Title>
   - All N tasks done
   - Tests passing
   - Changelog: docs/changelog/YYYY-MM-DD-<slug>.md
   - Archived: .claude/plans/archive/YYYY-MM-DD-<slug>.md
   ```

## Rules

- Always work through tasks in the order they appear in the plan
- Mark each task done in the plan file as you complete it (not all at once at the end)
- Do not skip the test step — tests must pass before completion
- If the plan has already-checked tasks, skip those and continue from the first unchecked task
- Keep the plan's Notes section updated with any discoveries or decisions made during execution

## User's Input

$ARGUMENTS
