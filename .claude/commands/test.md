# Run Tests and Summarize

Run the project's test suite and provide a clear summary of results with actionable suggestions for any failures.

## Step 1: Determine Test Scope

- If an argument specifies a package or file, test only that (e.g., `go test ./internal/ui/...`)
- If the argument says "short", run `make test-short`
- If the argument says "coverage", run `make test-coverage`
- Otherwise, run `make test` (full suite with race detector)

## Step 2: Run Tests

Execute the appropriate test command and capture output.

## Step 3: Analyze Results

Parse the test output and categorize:
- **Passed**: Total count
- **Failed**: List each with the test name, file, and failure message
- **Skipped**: Count and reason (e.g., `-short` flag)
- **Build errors**: Any compilation failures

## Step 4: For Failures — Diagnose

For each failing test:
1. Read the test file to understand what the test expects
2. Read the relevant source code that the test exercises
3. Identify the likely root cause
4. Suggest a specific fix

## Step 5: Report

Print a summary:

```
Test Results: X passed, Y failed, Z skipped

Failures:
  1. TestFooBar (internal/ui/tabs_test.go:42)
     Expected: "hello"
     Got: "world"
     Likely cause: Recent change to FormatGreeting() changed the default
     Suggested fix: Update the expected value or revert the format change

  2. TestBazQux (internal/config/config_test.go:88)
     Error: nil pointer dereference
     Likely cause: New SpinnerStyle field not initialized in test fixture
     Suggested fix: Add SpinnerStyle: "claude" to the test config

All passing — no issues found.
```

## Rules

- Always run the tests — don't just read test files and guess
- For large test suites, focus the detailed diagnosis on failures only
- If there are build errors, address those first (tests can't run if code doesn't compile)
- Don't fix the code automatically — report and suggest, let the user decide
- If all tests pass, keep the report brief

## User's Input

$ARGUMENTS
