package workspace

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileStatus represents the status of a markdown file
type FileStatus string

const (
	StatusPending    FileStatus = "pending"
	StatusInProgress FileStatus = "in-progress"
	StatusCompleted  FileStatus = "completed"
)

// File represents a markdown file in the workspace
type File struct {
	Name     string
	Path     string
	ModTime  time.Time
	Status   FileStatus
	Title    string // First heading or filename
	HasTodos bool   // Has unchecked todos
}

// Workspace manages a folder of markdown files
type Workspace struct {
	folder  string
	files   []*File
	watcher *fsnotify.Watcher
}

// New creates a new workspace for the given folder
func New(folder string) (*Workspace, error) {
	absPath, err := filepath.Abs(folder)
	if err != nil {
		return nil, fmt.Errorf("resolve folder path: %w", err)
	}

	// Verify folder exists
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("access folder: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", absPath)
	}

	w := &Workspace{
		folder: absPath,
	}

	// Initial scan
	if err := w.Refresh(); err != nil {
		return nil, err
	}

	return w, nil
}

// Folder returns the workspace folder path
func (w *Workspace) Folder() string {
	return w.folder
}

// Files returns all markdown files in the workspace
func (w *Workspace) Files() []*File {
	return w.files
}

// Refresh rescans the folder recursively for markdown files
func (w *Workspace) Refresh() error {
	var files []*File

	err := filepath.WalkDir(w.folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip entries we can't access
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		rel, err := filepath.Rel(w.folder, path)
		if err != nil {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		file := &File{
			Name:    filepath.ToSlash(rel),
			Path:    path,
			ModTime: info.ModTime(),
		}

		w.parseFile(file)
		files = append(files, file)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk folder: %w", err)
	}

	// Sort by modification time (most recent first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})

	w.files = files
	return nil
}

// parseFile extracts status and title from a markdown file
func (w *Workspace) parseFile(f *File) {
	file, err := os.Open(f.Path)
	if err != nil {
		f.Title = strings.TrimSuffix(filepath.Base(f.Name), ".md")
		f.Status = StatusPending
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inFrontmatter := false
	frontmatterDone := false
	hasUnchecked := false
	hasChecked := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Handle YAML frontmatter
		if trimmed == "---" {
			if !frontmatterDone {
				if !inFrontmatter {
					inFrontmatter = true
					continue
				} else {
					inFrontmatter = false
					frontmatterDone = true
					continue
				}
			}
		}

		// Parse frontmatter status
		if inFrontmatter {
			if strings.HasPrefix(trimmed, "status:") {
				status := strings.TrimSpace(strings.TrimPrefix(trimmed, "status:"))
				status = strings.Trim(status, `"'`)
				switch status {
				case "pending":
					f.Status = StatusPending
				case "in-progress", "in_progress", "inprogress":
					f.Status = StatusInProgress
				case "completed", "done", "complete":
					f.Status = StatusCompleted
				}
			}
			continue
		}

		// Find first heading for title
		if f.Title == "" && strings.HasPrefix(trimmed, "# ") {
			f.Title = strings.TrimPrefix(trimmed, "# ")
		}

		// Check for todo items
		if strings.Contains(line, "- [ ]") || strings.Contains(line, "* [ ]") {
			hasUnchecked = true
		}
		if strings.Contains(line, "- [x]") || strings.Contains(line, "- [X]") ||
			strings.Contains(line, "* [x]") || strings.Contains(line, "* [X]") {
			hasChecked = true
		}
	}

	// Set defaults
	if f.Title == "" {
		f.Title = strings.TrimSuffix(filepath.Base(f.Name), ".md")
	}

	f.HasTodos = hasUnchecked

	// Derive status from checkboxes if not set in frontmatter
	if f.Status == "" {
		switch {
		case hasUnchecked:
			f.Status = StatusPending
		case hasChecked:
			f.Status = StatusCompleted
		default:
			f.Status = StatusPending
		}
	}
}

// GetFile returns a file by name
func (w *Workspace) GetFile(name string) *File {
	for _, f := range w.files {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// GetFileByPath returns a file by its full path
func (w *Workspace) GetFileByPath(path string) *File {
	for _, f := range w.files {
		if f.Path == path {
			return f
		}
	}
	return nil
}

// ReadFile reads the content of a markdown file
func (w *Workspace) ReadFile(name string) (string, error) {
	f := w.GetFile(name)
	if f == nil {
		return "", fmt.Errorf("file not found: %s", name)
	}

	content, err := os.ReadFile(f.Path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return string(content), nil
}

// SaveFile saves content to a markdown file
func (w *Workspace) SaveFile(name, content string) error {
	path := filepath.Join(w.folder, name)

	// Ensure .md extension
	if !strings.HasSuffix(strings.ToLower(name), ".md") {
		path += ".md"
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	// Refresh to update file list
	return w.Refresh()
}

// CreateFile creates a new markdown file
func (w *Workspace) CreateFile(name string) error {
	if !strings.HasSuffix(strings.ToLower(name), ".md") {
		name += ".md"
	}

	path := filepath.Join(w.folder, name)

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", name)
	}

	// Create with basic template
	content := fmt.Sprintf("# %s\n\n", strings.TrimSuffix(name, ".md"))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	return w.Refresh()
}

// ToggleFileStatus toggles a file's status between completed and pending.
// If the file is currently completed, it becomes pending; otherwise it becomes completed.
// The status is persisted by updating the YAML frontmatter in the file.
func (w *Workspace) ToggleFileStatus(name string) error {
	f := w.GetFile(name)
	if f == nil {
		return fmt.Errorf("file not found: %s", name)
	}

	content, err := os.ReadFile(f.Path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	newStatus := "completed"
	if f.Status == StatusCompleted {
		newStatus = "pending"
	}

	updated := setFrontmatterStatus(string(content), newStatus)

	if err := os.WriteFile(f.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return w.Refresh()
}

// setFrontmatterStatus updates or inserts the status field in YAML frontmatter.
func setFrontmatterStatus(content, status string) string {
	lines := strings.Split(content, "\n")

	// Check if file starts with frontmatter
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		// Find closing ---
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				// Found frontmatter block (lines[0] to lines[i])
				// Look for existing status: line
				found := false
				for j := 1; j < i; j++ {
					trimmed := strings.TrimSpace(lines[j])
					if strings.HasPrefix(trimmed, "status:") {
						lines[j] = "status: " + status
						found = true
						break
					}
				}
				if !found {
					// Insert status before closing ---
					newLines := make([]string, 0, len(lines)+1)
					newLines = append(newLines, lines[:i]...)
					newLines = append(newLines, "status: "+status)
					newLines = append(newLines, lines[i:]...)
					lines = newLines
				}
				return strings.Join(lines, "\n")
			}
		}
	}

	// No frontmatter — prepend one
	frontmatter := "---\nstatus: " + status + "\n---\n"
	return frontmatter + content
}

// DeleteFolder recursively deletes a subdirectory and all its contents.
// The name is a relative path within the workspace (e.g. "subdir" or "a/b").
func (w *Workspace) DeleteFolder(name string) error {
	absPath := filepath.Join(w.folder, filepath.FromSlash(name))

	// Safety: ensure the resolved path is inside the workspace
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return fmt.Errorf("resolve folder path: %w", err)
	}
	if !strings.HasPrefix(absPath, w.folder+string(filepath.Separator)) {
		return fmt.Errorf("path is outside workspace: %s", name)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("folder not found: %s", name)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", name)
	}

	if err := os.RemoveAll(absPath); err != nil {
		return fmt.Errorf("delete folder: %w", err)
	}

	return w.Refresh()
}

// MoveFile moves a file to a different directory within the workspace.
// oldName is the file's current relative path (e.g. "subdir/task.md").
// newDir is the destination directory relative path (e.g. "other" or "" for root).
func (w *Workspace) MoveFile(oldName, newDir string) error {
	f := w.GetFile(oldName)
	if f == nil {
		return fmt.Errorf("file not found: %s", oldName)
	}

	baseName := filepath.Base(f.Path)
	destDir := filepath.Join(w.folder, filepath.FromSlash(newDir))
	destPath := filepath.Join(destDir, baseName)

	// Don't move to the same location
	if f.Path == destPath {
		return fmt.Errorf("file is already in this folder")
	}

	// Check for name conflict
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("a file named '%s' already exists in the destination", baseName)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := os.Rename(f.Path, destPath); err != nil {
		return fmt.Errorf("move file: %w", err)
	}

	return w.Refresh()
}

// MoveFolder moves a directory to a different parent directory within the workspace.
// oldPath is the directory's current relative path (e.g. "subdir").
// newDir is the destination parent relative path (e.g. "other" or "" for root).
func (w *Workspace) MoveFolder(oldPath, newDir string) error {
	srcAbs := filepath.Join(w.folder, filepath.FromSlash(oldPath))

	info, err := os.Stat(srcAbs)
	if err != nil {
		return fmt.Errorf("folder not found: %s", oldPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", oldPath)
	}

	baseName := filepath.Base(srcAbs)
	destParent := filepath.Join(w.folder, filepath.FromSlash(newDir))
	destPath := filepath.Join(destParent, baseName)

	// Don't move to the same location
	if srcAbs == destPath {
		return fmt.Errorf("folder is already in this location")
	}

	// Prevent moving a folder into itself or a descendant
	destAbs, err := filepath.Abs(destParent)
	if err != nil {
		return fmt.Errorf("resolve destination path: %w", err)
	}
	srcAbsResolved, err := filepath.Abs(srcAbs)
	if err != nil {
		return fmt.Errorf("resolve source path: %w", err)
	}
	if strings.HasPrefix(destAbs+string(filepath.Separator), srcAbsResolved+string(filepath.Separator)) {
		return fmt.Errorf("cannot move a folder into itself or a subfolder")
	}

	// Check for name conflict
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("a folder named '%s' already exists in the destination", baseName)
	}

	// Ensure destination parent exists
	if err := os.MkdirAll(destParent, 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := os.Rename(srcAbs, destPath); err != nil {
		return fmt.Errorf("move folder: %w", err)
	}

	return w.Refresh()
}

// DeleteFile deletes a markdown file
func (w *Workspace) DeleteFile(name string) error {
	f := w.GetFile(name)
	if f == nil {
		return fmt.Errorf("file not found: %s", name)
	}

	if err := os.Remove(f.Path); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return w.Refresh()
}

// Watch starts watching the folder for changes
func (w *Workspace) Watch() (<-chan struct{}, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	// Directories to skip entirely (heavy or internal — not useful for markdown watching)
	skipDirs := map[string]bool{
		".git": true, ".hg": true, ".svn": true,
		"node_modules": true, "vendor": true, ".next": true,
		"build": true, "dist": true, "__pycache__": true,
	}

	// Watch root folder and non-hidden, non-heavy subdirectories
	err = filepath.WalkDir(w.folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			// Skip hidden dirs (except the root itself) and known heavy dirs
			if path != w.folder && (strings.HasPrefix(name, ".") || skipDirs[name]) {
				return filepath.SkipDir
			}
			_ = watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("watch folder tree: %w", err)
	}

	w.watcher = watcher
	changes := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					close(changes)
					return
				}

				// Watch newly created directories (same filter as initial walk)
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						name := filepath.Base(event.Name)
						if !strings.HasPrefix(name, ".") && !skipDirs[name] {
							_ = watcher.Add(event.Name)
						}
					}
				}

				// Notify for markdown files or directory changes
				if strings.HasSuffix(strings.ToLower(event.Name), ".md") ||
					event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
					// Refresh file list
					_ = w.Refresh()

					// Non-blocking send
					select {
					case changes <- struct{}{}:
					default:
					}
				}

			case _, ok := <-watcher.Errors:
				if !ok {
					close(changes)
					return
				}
			}
		}
	}()

	return changes, nil
}

// StopWatch stops watching for changes
func (w *Workspace) StopWatch() {
	if w.watcher != nil {
		w.watcher.Close()
		w.watcher = nil
	}
}
