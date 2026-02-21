package agent

import (
	"fmt"
	"strings"
)

// BuildFilePrompt creates a system prompt for working with a markdown file
func BuildFilePrompt(fileName, filePath, content string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`You are helping the user work on a plan described in a markdown file.

Current file: %s
File path: %s

File content:
---
%s
---

Help the user refine, implement, or complete tasks in this plan. You have full access to the filesystem to make changes.

When working on this file:
- Read and understand the plan structure
- Help break down tasks into actionable steps
- Implement code changes as requested
- Update the markdown file to reflect progress
- Follow existing patterns in the codebase
`, fileName, filePath, content))

	return sb.String()
}

// BuildSimplePrompt creates a basic prompt for general assistance
func BuildSimplePrompt(context string) string {
	if context == "" {
		return "Help the user with their development tasks."
	}
	return context
}
