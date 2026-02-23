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

	// Watch root folder and all non-hidden subdirectories
	err = filepath.WalkDir(w.folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
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

				// Watch newly created directories
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						if !strings.HasPrefix(filepath.Base(event.Name), ".") {
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

// RecentFolders manages the list of recently used folders
type RecentFolders struct {
	Folders []string `json:"folders"`
}

// LoadRecentFolders loads the recent folders from config
func LoadRecentFolders(configDir string) (*RecentFolders, error) {
	path := filepath.Join(configDir, "recent.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RecentFolders{}, nil
		}
		return nil, err
	}

	var recent RecentFolders
	// Simple JSON parsing
	content := string(data)
	content = strings.TrimPrefix(content, `{"folders":[`)
	content = strings.TrimSuffix(content, `]}`)

	if content != "" {
		parts := strings.Split(content, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.Trim(p, `"`)
			if p != "" {
				recent.Folders = append(recent.Folders, p)
			}
		}
	}

	return &recent, nil
}

// Save saves the recent folders to config
func (r *RecentFolders) Save(configDir string) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(configDir, "recent.json")

	// Simple JSON encoding
	var parts []string
	for _, f := range r.Folders {
		parts = append(parts, fmt.Sprintf("%q", f))
	}
	content := fmt.Sprintf(`{"folders":[%s]}`, strings.Join(parts, ","))

	return os.WriteFile(path, []byte(content), 0o644)
}

// Add adds a folder to the recent list
func (r *RecentFolders) Add(folder string) {
	// Remove if already exists
	var newFolders []string
	for _, f := range r.Folders {
		if f != folder {
			newFolders = append(newFolders, f)
		}
	}

	// Add to front
	r.Folders = append([]string{folder}, newFolders...)

	// Keep only last 10
	if len(r.Folders) > 10 {
		r.Folders = r.Folders[:10]
	}
}
