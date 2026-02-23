package ui

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/planck/internal/workspace"
)

// padToWidth pads a string with spaces so it reaches the given rune width.
func padToWidth(s string, width int) string {
	runeLen := utf8.RuneCountInString(s)
	if runeLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-runeLen)
}

// treeNode represents a node in the file tree (either a directory or a file)
type treeNode struct {
	name     string          // Display name (just segment: "task.md" or "subdir")
	path     string          // Relative path ("subdir/task.md" or "subdir")
	depth    int             // Nesting level (0 = root)
	isDir    bool            // Directory or file
	expanded bool            // For dirs: expanded state
	file     *workspace.File // For files only (nil for dirs)
	children []*treeNode     // Children (for dirs)
}

// FileList displays a list of markdown files as a collapsible tree
type FileList struct {
	theme    *Theme
	files    []*workspace.File // All markdown files (flat)
	root     []*treeNode       // Tree roots
	visible  []*treeNode       // Flattened visible nodes
	dirState map[string]bool   // Persists expand/collapse across refreshes (path → expanded)
	cursor   int
	focused  bool
	height   int
	width    int
	offset   int // scroll offset for long lists
}

// NewFileList creates a new file list
func NewFileList(theme *Theme) *FileList {
	return &FileList{
		theme:    theme,
		width:    24,
		dirState: make(map[string]bool),
	}
}

// Init initializes the file list
func (f *FileList) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (f *FileList) Update(msg tea.Msg) (*FileList, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if !f.focused {
			return f, nil
		}

		switch msg.String() {
		case "down", "j":
			f.cursor++
			if f.cursor >= len(f.visible) {
				f.cursor = len(f.visible) - 1
			}
			if f.cursor < 0 {
				f.cursor = 0
			}
			f.ensureVisible()

		case "up", "k":
			f.cursor--
			if f.cursor < 0 {
				f.cursor = 0
			}
			f.ensureVisible()

		case "home", "g":
			f.cursor = 0
			f.offset = 0

		case "end", "G":
			f.cursor = len(f.visible) - 1
			if f.cursor < 0 {
				f.cursor = 0
			}
			f.ensureVisible()

		case "pgdown", "ctrl+d":
			visibleLines := f.visibleLines()
			f.cursor += visibleLines
			if f.cursor >= len(f.visible) {
				f.cursor = len(f.visible) - 1
			}
			if f.cursor < 0 {
				f.cursor = 0
			}
			f.ensureVisible()

		case "pgup", "ctrl+u":
			visibleLines := f.visibleLines()
			f.cursor -= visibleLines
			if f.cursor < 0 {
				f.cursor = 0
			}
			f.ensureVisible()
		}
	}

	return f, nil
}

// visibleLines returns the number of visible lines
func (f *FileList) visibleLines() int {
	lines := f.height - 5 // header(1) + sep(1) + content + footer sep(1) + padding(2)
	if lines < 1 {
		lines = 1
	}
	return lines
}

// ensureVisible ensures the cursor is visible
func (f *FileList) ensureVisible() {
	visibleLines := f.visibleLines()

	if f.cursor < f.offset {
		f.offset = f.cursor
	}
	if f.cursor >= f.offset+visibleLines {
		f.offset = f.cursor - visibleLines + 1
	}
}

// buildTree builds the tree structure from the flat file list
func (f *FileList) buildTree() {
	// Map of dir path → dir node for reuse
	dirNodes := make(map[string]*treeNode)
	var roots []*treeNode

	for _, file := range f.files {
		parts := strings.Split(file.Name, "/")

		if len(parts) == 1 {
			// Root-level file
			node := &treeNode{
				name:  parts[0],
				path:  file.Name,
				depth: 0,
				isDir: false,
				file:  file,
			}
			roots = append(roots, node)
			continue
		}

		// Ensure all parent directories exist
		var parent *[]*treeNode = &roots
		for i := 0; i < len(parts)-1; i++ {
			dirPath := strings.Join(parts[:i+1], "/")
			dirNode, exists := dirNodes[dirPath]
			if !exists {
				expanded := true
				if state, ok := f.dirState[dirPath]; ok {
					expanded = state
				}
				dirNode = &treeNode{
					name:     parts[i],
					path:     dirPath,
					depth:    i,
					isDir:    true,
					expanded: expanded,
				}
				dirNodes[dirPath] = dirNode
				*parent = append(*parent, dirNode)
			}
			parent = &dirNode.children
		}

		// Add the file as a leaf
		fileNode := &treeNode{
			name:  parts[len(parts)-1],
			path:  file.Name,
			depth: len(parts) - 1,
			isDir: false,
			file:  file,
		}
		*parent = append(*parent, fileNode)
	}

	// Sort children at each level: dirs first alphabetically, then files alphabetically
	sortChildren(roots)

	f.root = roots
}

// sortChildren recursively sorts children: dirs first (alphabetical), then files (alphabetical)
func sortChildren(nodes []*treeNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].isDir != nodes[j].isDir {
			return nodes[i].isDir // dirs first
		}
		return strings.ToLower(nodes[i].name) < strings.ToLower(nodes[j].name)
	})
	for _, n := range nodes {
		if n.isDir && len(n.children) > 0 {
			sortChildren(n.children)
		}
	}
}

// rebuildVisible flattens the tree respecting collapsed directories
func (f *FileList) rebuildVisible() {
	f.visible = nil
	f.flattenNodes(f.root)
}

func (f *FileList) flattenNodes(nodes []*treeNode) {
	for _, n := range nodes {
		f.visible = append(f.visible, n)
		if n.isDir && n.expanded {
			f.flattenNodes(n.children)
		}
	}
}

// View renders the file list
func (f *FileList) View() string {
	var b strings.Builder

	// Header
	header := f.theme.Title.Render("FILES")
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(f.theme.Dimmed.Render(safeRepeat("─", f.width-2)))
	b.WriteString("\n")

	// File list
	visibleLines := f.visibleLines()

	if len(f.visible) == 0 {
		b.WriteString(f.theme.Dimmed.Render("  No markdown files"))
		b.WriteString("\n")
		visibleLines--
	} else {
		endIdx := f.offset + visibleLines
		if endIdx > len(f.visible) {
			endIdx = len(f.visible)
		}

		for i := f.offset; i < endIdx; i++ {
			node := f.visible[i]
			isSelected := f.focused && i == f.cursor
			indent := safeRepeat("  ", node.depth)

			// Available content width inside the sidebar (excluding the right border)
			contentWidth := f.width - 2

			if node.isDir {
				// Directory node
				arrow := IndicatorFolderOpen
				if !node.expanded {
					arrow = IndicatorFolderClosed
				}

				maxLen := f.width - 6 - (node.depth * 2)
				dirName := truncate(node.name, maxLen)

				var line string
				if isSelected {
					raw := fmt.Sprintf("%s%s%s %s", IndicatorSelected, indent, arrow, dirName)
					line = f.theme.TreeSelected.Render(padToWidth(raw, contentWidth))
				} else {
					line = f.theme.TreeItem.Render(fmt.Sprintf(" %s%s %s", indent, arrow, dirName))
				}

				b.WriteString(line)
				b.WriteString("\n")
			} else {
				// File node
				file := node.file

				// Status indicator character
				var indicatorChar string
				switch file.Status {
				case workspace.StatusCompleted:
					indicatorChar = IndicatorDone
				case workspace.StatusInProgress:
					indicatorChar = IndicatorInProgress
				default:
					indicatorChar = IndicatorPending
				}

				// File name (truncate if needed)
				maxLen := f.width - 6 - (node.depth * 2)
				name := truncate(node.name, maxLen)
				var line string
				if isSelected {
					raw := fmt.Sprintf("%s%s%s %s", IndicatorSelected, indent, indicatorChar, name)
					line = f.theme.SidebarSelected.Render(padToWidth(raw, contentWidth))
				} else {
					// Non-selected: use individually-styled status indicator
					var indicator string
					switch file.Status {
					case workspace.StatusCompleted:
						indicator = f.theme.StatusDone.Render(indicatorChar)
					case workspace.StatusInProgress:
						indicator = f.theme.StatusProgress.Render(indicatorChar)
					default:
						indicator = f.theme.StatusPending.Render(indicatorChar)
					}
					if file.Status == workspace.StatusCompleted {
						line = f.theme.Dimmed.PaddingLeft(1).Render(fmt.Sprintf(" %s%s %s", indent, indicator, name))
					} else {
						line = f.theme.SidebarItem.Render(fmt.Sprintf(" %s%s %s", indent, indicator, name))
					}
				}

				b.WriteString(line)
				b.WriteString("\n")
			}
		}
	}

	// Fill remaining space
	contentLines := len(f.visible)
	if contentLines > visibleLines {
		contentLines = visibleLines
	}
	if len(f.visible) == 0 {
		contentLines = 1
	}
	for i := contentLines; i < visibleLines; i++ {
		b.WriteString("\n")
	}

	return f.theme.Sidebar.Width(f.width).Height(f.height).Render(b.String())
}

// SetFiles sets the list of files and rebuilds the tree
func (f *FileList) SetFiles(files []*workspace.File) {
	// Remember selected file path before rebuild
	var selectedPath string
	if f.cursor >= 0 && f.cursor < len(f.visible) {
		selectedPath = f.visible[f.cursor].path
	}

	f.files = files
	f.buildTree()
	f.rebuildVisible()

	// Try to re-select same node
	if selectedPath != "" {
		for i, n := range f.visible {
			if n.path == selectedPath {
				f.cursor = i
				f.ensureVisible()
				return
			}
		}
	}

	// Clamp cursor
	if f.cursor >= len(f.visible) {
		f.cursor = len(f.visible) - 1
		if f.cursor < 0 {
			f.cursor = 0
		}
	}
	f.ensureVisible()
}

// SetFocused sets the focused state
func (f *FileList) SetFocused(focused bool) {
	f.focused = focused
}

// SetSize sets the file list dimensions
func (f *FileList) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// SelectedFile returns the currently selected file (nil for dirs)
func (f *FileList) SelectedFile() *workspace.File {
	if f.cursor >= 0 && f.cursor < len(f.visible) {
		return f.visible[f.cursor].file
	}
	return nil
}

// IsSelectedDir returns true if the cursor is on a directory node
func (f *FileList) IsSelectedDir() bool {
	if f.cursor >= 0 && f.cursor < len(f.visible) {
		return f.visible[f.cursor].isDir
	}
	return false
}

// ExpandSelected expands the selected directory node
func (f *FileList) ExpandSelected() {
	if f.cursor >= 0 && f.cursor < len(f.visible) {
		node := f.visible[f.cursor]
		if node.isDir && !node.expanded {
			node.expanded = true
			f.dirState[node.path] = true
			f.rebuildVisible()
		}
	}
}

// CollapseSelected collapses the selected directory node
func (f *FileList) CollapseSelected() {
	if f.cursor >= 0 && f.cursor < len(f.visible) {
		node := f.visible[f.cursor]
		if node.isDir && node.expanded {
			node.expanded = false
			f.dirState[node.path] = false
			f.rebuildVisible()
		}
	}
}

// Cursor returns the current cursor position
func (f *FileList) Cursor() int {
	return f.cursor
}

// SetCursor sets the cursor position
func (f *FileList) SetCursor(cursor int) {
	f.cursor = cursor
	if f.cursor < 0 {
		f.cursor = 0
	}
	if f.cursor >= len(f.visible) {
		f.cursor = len(f.visible) - 1
	}
	f.ensureVisible()
}

// SelectFile selects a file by name
func (f *FileList) SelectFile(name string) {
	for i, node := range f.visible {
		if !node.isDir && node.file != nil && node.file.Name == name {
			f.cursor = i
			f.ensureVisible()
			return
		}
	}
}

// FileCount returns the number of markdown files (not tree nodes)
func (f *FileList) FileCount() int {
	return len(f.files)
}
