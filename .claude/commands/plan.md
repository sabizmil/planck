# Create a New Plan

Create a new plan file for the given task. Follow these steps:

1. **Generate a slug** from the task description (lowercase, hyphenated, concise — e.g., "add-dark-mode" or "fix-tab-crash")
2. **Create the plan file** at `.claude/plans/YYYY-MM-DD-<slug>.md` using today's date
3. **Fill in the template** below, expanding the user's description into a proper goal, considering 3-5 approaches, evaluating trade-offs, and selecting the best one

## Plan Template

```markdown
---
status: created
---
# <Title>

**Date:** YYYY-MM-DD

---

## Goal
What this plan aims to accomplish. Expand on the user's request with enough context that someone reading this later understands the full scope.

## Approaches Considered

### 1. <Approach Name>
- **Description:** Brief explanation
- **Pros:** What's good about it
- **Cons:** What's bad about it

### 2. <Approach Name>
- **Description:** Brief explanation
- **Pros:** What's good about it
- **Cons:** What's bad about it

### 3. <Approach Name>
- **Description:** Brief explanation
- **Pros:** What's good about it
- **Cons:** What's bad about it

## Chosen Approach
Which approach was selected and why it's the best option.

## Tasks
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

## Notes
Any discoveries, decisions, or open questions.
```

## Rules

- Before writing the plan, **explore the relevant code** to understand the current state and inform the approach evaluation
- Always consider at least 3 approaches before selecting one
- Tasks should be concrete and checkable — not vague ("improve X") but specific ("add Y method to Z struct")
- If the task is ambiguous, ask clarifying questions before creating the plan
- After creating the file, print the full path and a summary of the chosen approach
- Remind the user of the next steps: review/iterate on the plan, then run `/execute-plan` to execute all tasks, run tests, and auto-complete

## User's Task

$ARGUMENTS
