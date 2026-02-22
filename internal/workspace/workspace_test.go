package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetFrontmatterStatus(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		status   string
		expected string
	}{
		{
			name:     "existing frontmatter with status field",
			content:  "---\ntitle: My Plan\nstatus: pending\n---\n# Hello\n",
			status:   "completed",
			expected: "---\ntitle: My Plan\nstatus: completed\n---\n# Hello\n",
		},
		{
			name:     "existing frontmatter without status field",
			content:  "---\ntitle: My Plan\n---\n# Hello\n",
			status:   "completed",
			expected: "---\ntitle: My Plan\nstatus: completed\n---\n# Hello\n",
		},
		{
			name:     "no frontmatter",
			content:  "# Hello\n\nSome content\n",
			status:   "completed",
			expected: "---\nstatus: completed\n---\n# Hello\n\nSome content\n",
		},
		{
			name:     "toggle completed back to pending",
			content:  "---\nstatus: completed\n---\n# Hello\n",
			status:   "pending",
			expected: "---\nstatus: pending\n---\n# Hello\n",
		},
		{
			name:     "in-progress to completed",
			content:  "---\nstatus: in-progress\n---\n# Hello\n",
			status:   "completed",
			expected: "---\nstatus: completed\n---\n# Hello\n",
		},
		{
			name:     "empty file",
			content:  "",
			status:   "completed",
			expected: "---\nstatus: completed\n---\n",
		},
		{
			name:     "frontmatter with multiple fields preserves them",
			content:  "---\ntitle: Plan\ndate: 2026-01-01\nstatus: pending\ntags: feature\n---\n# Content\n",
			status:   "completed",
			expected: "---\ntitle: Plan\ndate: 2026-01-01\nstatus: completed\ntags: feature\n---\n# Content\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := setFrontmatterStatus(tt.content, tt.status)
			if got != tt.expected {
				t.Errorf("setFrontmatterStatus() =\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestToggleFileStatus(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		expectedStatus FileStatus
		expectedInFile string
	}{
		{
			name:           "pending file becomes completed",
			initialContent: "---\nstatus: pending\n---\n# Task\n- [ ] Do something\n",
			expectedStatus: StatusCompleted,
			expectedInFile: "status: completed",
		},
		{
			name:           "completed file becomes pending",
			initialContent: "---\nstatus: completed\n---\n# Task\n- [x] Done thing\n",
			expectedStatus: StatusPending,
			expectedInFile: "status: pending",
		},
		{
			name:           "in-progress file becomes completed",
			initialContent: "---\nstatus: in-progress\n---\n# Task\n",
			expectedStatus: StatusCompleted,
			expectedInFile: "status: completed",
		},
		{
			name:           "file without frontmatter gets completed",
			initialContent: "# Task\n\nSome content\n",
			expectedStatus: StatusCompleted,
			expectedInFile: "status: completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and file
			dir := t.TempDir()
			filePath := filepath.Join(dir, "test.md")
			if err := os.WriteFile(filePath, []byte(tt.initialContent), 0644); err != nil {
				t.Fatalf("write test file: %v", err)
			}

			// Create workspace
			ws, err := New(dir)
			if err != nil {
				t.Fatalf("create workspace: %v", err)
			}

			// Toggle status
			if err := ws.ToggleFileStatus("test.md"); err != nil {
				t.Fatalf("toggle status: %v", err)
			}

			// Verify the file on disk
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			if got := string(content); !contains(got, tt.expectedInFile) {
				t.Errorf("file content missing %q, got:\n%s", tt.expectedInFile, got)
			}

			// Verify the workspace state was refreshed
			f := ws.GetFile("test.md")
			if f == nil {
				t.Fatal("file not found after toggle")
			}
			if f.Status != tt.expectedStatus {
				t.Errorf("status = %q, want %q", f.Status, tt.expectedStatus)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
