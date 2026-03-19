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
			ws, err := New(dir, nil)
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

func TestDeleteFolder(t *testing.T) {
	t.Run("delete folder with nested content", func(t *testing.T) {
		dir := t.TempDir()
		// Create nested structure
		sub := filepath.Join(dir, "plans")
		os.MkdirAll(filepath.Join(sub, "deep"), 0o755)
		os.WriteFile(filepath.Join(sub, "a.md"), []byte("# A\n"), 0o644)
		os.WriteFile(filepath.Join(sub, "deep", "b.md"), []byte("# B\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}
		if len(ws.Files()) != 2 {
			t.Fatalf("expected 2 files, got %d", len(ws.Files()))
		}

		if err := ws.DeleteFolder("plans"); err != nil {
			t.Fatalf("delete folder: %v", err)
		}

		// Folder should be gone
		if _, err := os.Stat(sub); !os.IsNotExist(err) {
			t.Error("folder still exists after deletion")
		}
		// Workspace should have no files
		if len(ws.Files()) != 0 {
			t.Errorf("expected 0 files after deletion, got %d", len(ws.Files()))
		}
	})

	t.Run("delete non-existent folder", func(t *testing.T) {
		dir := t.TempDir()
		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		err = ws.DeleteFolder("nonexistent")
		if err == nil {
			t.Error("expected error deleting non-existent folder")
		}
	})

	t.Run("delete rejects path outside workspace", func(t *testing.T) {
		dir := t.TempDir()
		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		err = ws.DeleteFolder("../../etc")
		if err == nil {
			t.Error("expected error for path traversal")
		}
	})

	t.Run("delete rejects file path", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "file.md"), []byte("# File\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		err = ws.DeleteFolder("file.md")
		if err == nil {
			t.Error("expected error when deleting a file as folder")
		}
	})
}

func TestMoveFile(t *testing.T) {
	t.Run("move file to subfolder", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "task.md"), []byte("# Task\n"), 0o644)
		os.MkdirAll(filepath.Join(dir, "archive"), 0o755)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		if err := ws.MoveFile("task.md", "archive"); err != nil {
			t.Fatalf("move file: %v", err)
		}

		// Old path should be gone
		if _, err := os.Stat(filepath.Join(dir, "task.md")); !os.IsNotExist(err) {
			t.Error("file still exists at old path")
		}
		// New path should exist
		if _, err := os.Stat(filepath.Join(dir, "archive", "task.md")); err != nil {
			t.Error("file not found at new path")
		}
		// Workspace should reflect the move
		if f := ws.GetFile("archive/task.md"); f == nil {
			t.Error("workspace doesn't have file at new path")
		}
	})

	t.Run("move file to root", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
		os.WriteFile(filepath.Join(dir, "sub", "task.md"), []byte("# Task\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		if err := ws.MoveFile("sub/task.md", ""); err != nil {
			t.Fatalf("move file: %v", err)
		}

		if f := ws.GetFile("task.md"); f == nil {
			t.Error("file not found at root after move")
		}
	})

	t.Run("move file creates intermediate dirs", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "task.md"), []byte("# Task\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		if err := ws.MoveFile("task.md", "a/b/c"); err != nil {
			t.Fatalf("move file: %v", err)
		}

		if f := ws.GetFile("a/b/c/task.md"); f == nil {
			t.Error("file not found at new nested path")
		}
	})

	t.Run("move file rejects same location", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
		os.WriteFile(filepath.Join(dir, "sub", "task.md"), []byte("# Task\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		err = ws.MoveFile("sub/task.md", "sub")
		if err == nil {
			t.Error("expected error moving to same location")
		}
	})

	t.Run("move file rejects name conflict", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
		os.WriteFile(filepath.Join(dir, "task.md"), []byte("# Task 1\n"), 0o644)
		os.WriteFile(filepath.Join(dir, "sub", "task.md"), []byte("# Task 2\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		err = ws.MoveFile("task.md", "sub")
		if err == nil {
			t.Error("expected error for name conflict")
		}
	})
}

func TestMoveFolder(t *testing.T) {
	t.Run("move folder to another folder", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "src"), 0o755)
		os.MkdirAll(filepath.Join(dir, "dest"), 0o755)
		os.WriteFile(filepath.Join(dir, "src", "task.md"), []byte("# Task\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		if err := ws.MoveFolder("src", "dest"); err != nil {
			t.Fatalf("move folder: %v", err)
		}

		if _, err := os.Stat(filepath.Join(dir, "dest", "src", "task.md")); err != nil {
			t.Error("file not found at new path after folder move")
		}
		if f := ws.GetFile("dest/src/task.md"); f == nil {
			t.Error("workspace doesn't reflect moved file")
		}
	})

	t.Run("move folder to root", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "parent", "child"), 0o755)
		os.WriteFile(filepath.Join(dir, "parent", "child", "task.md"), []byte("# Task\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		if err := ws.MoveFolder("parent/child", ""); err != nil {
			t.Fatalf("move folder: %v", err)
		}

		if f := ws.GetFile("child/task.md"); f == nil {
			t.Error("file not found at root/child after move")
		}
	})

	t.Run("move folder into itself is rejected", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "src", "sub"), 0o755)
		os.WriteFile(filepath.Join(dir, "src", "task.md"), []byte("# Task\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		err = ws.MoveFolder("src", "src/sub")
		if err == nil {
			t.Error("expected error moving folder into itself")
		}
	})

	t.Run("move folder same location is rejected", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "src"), 0o755)
		os.WriteFile(filepath.Join(dir, "src", "task.md"), []byte("# Task\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		err = ws.MoveFolder("src", "")
		if err == nil {
			t.Error("expected error moving to same location")
		}
	})

	t.Run("move folder rejects name conflict", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "src"), 0o755)
		os.MkdirAll(filepath.Join(dir, "dest", "src"), 0o755)
		os.WriteFile(filepath.Join(dir, "src", "task.md"), []byte("# Task\n"), 0o644)

		ws, err := New(dir, nil)
		if err != nil {
			t.Fatalf("create workspace: %v", err)
		}

		err = ws.MoveFolder("src", "dest")
		if err == nil {
			t.Error("expected error for name conflict")
		}
	})
}

func TestRefreshExcludesDirs(t *testing.T) {
	dir := t.TempDir()

	// Create files in root, an included subdir, and excluded dirs
	os.WriteFile(filepath.Join(dir, "root.md"), []byte("# Root\n"), 0o644)

	included := filepath.Join(dir, "docs")
	os.MkdirAll(included, 0o755)
	os.WriteFile(filepath.Join(included, "doc.md"), []byte("# Doc\n"), 0o644)

	excluded := filepath.Join(dir, "node_modules")
	os.MkdirAll(excluded, 0o755)
	os.WriteFile(filepath.Join(excluded, "pkg.md"), []byte("# Pkg\n"), 0o644)

	hidden := filepath.Join(dir, ".git")
	os.MkdirAll(hidden, 0o755)
	os.WriteFile(filepath.Join(hidden, "internal.md"), []byte("# Git\n"), 0o644)

	tests := []struct {
		name        string
		excludeDirs []string
		wantFiles   int
		wantNames   []string
	}{
		{
			name:        "excludes node_modules and .git",
			excludeDirs: []string{"node_modules", ".git"},
			wantFiles:   2,
			wantNames:   []string{"root.md", "docs/doc.md"},
		},
		{
			name:        "nil exclude list shows all dirs including hidden",
			excludeDirs: nil,
			wantFiles:   4, // root.md, docs/doc.md, node_modules/pkg.md, .git/internal.md
			wantNames:   []string{"root.md", "docs/doc.md", "node_modules/pkg.md", ".git/internal.md"},
		},
		{
			name:        "exclude multiple dirs",
			excludeDirs: []string{"node_modules", "docs", ".git"},
			wantFiles:   1,
			wantNames:   []string{"root.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, err := New(dir, tt.excludeDirs)
			if err != nil {
				t.Fatalf("create workspace: %v", err)
			}
			if got := len(ws.Files()); got != tt.wantFiles {
				names := make([]string, len(ws.Files()))
				for i, f := range ws.Files() {
					names[i] = f.Name
				}
				t.Fatalf("got %d files %v, want %d files %v", got, names, tt.wantFiles, tt.wantNames)
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
