package ui

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sabizmil/planck/internal/workspace"
)

// padToWidth pads a string with spaces so it reaches the given rune width.
func padToWidth(s string, width int) string {
	runeLen := utf8.RuneCountInString(s)
	if runeLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-runeLen)
}

// MoveConfirmedMsg is sent when the user confirms a move destination.
type MoveConfirmedMsg struct {
	SourcePath string // relative path of the item being moved
	DestDir    string // relative path of the destination directory ("" = root)
	IsDir      bool   // whether the source is a directory
}

// MoveCanceledMsg is sent when the user cancels move mode.
type MoveCanceledMsg struct{}

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

// ClickAction describes what the user clicked on in the file list.
type ClickAction int

const (
	ClickNone      ClickAction = iota // click didn't hit any item
	ClickFile                         // clicked a file node
	ClickDirToggle                    // clicked a directory (toggled expand/collapse)
)

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
	screenY  int // vertical offset of the file list on screen (for mouse coordinate translation)

	// Move mode state
	moveMode      bool   // whether the file list is in move-destination-picker mode
	moveSource    string // relative path of the item being moved
	moveSourceDir bool   // whether the source is a directory
	moveCursor    int    // separate cursor for move mode navigation
	moveOffset    int    // separate scroll offset for move mode
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

		if f.moveMode {
			return f.updateMoveMode(msg)
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

// updateMoveMode handles key events while in move mode.
func (f *FileList) updateMoveMode(msg tea.KeyMsg) (*FileList, tea.Cmd) {
	// In move mode, visible list has a synthetic (root) node at index 0,
	// followed by the normal tree. We use moveCursor/moveOffset.
	moveVisible := f.moveVisibleNodes()
	maxIdx := len(moveVisible) - 1

	switch msg.String() {
	case "down", "j":
		// Skip non-directory nodes — only folders are valid move targets
		for next := f.moveCursor + 1; next <= maxIdx; next++ {
			if moveVisible[next].isDir {
				f.moveCursor = next
				break
			}
		}
		f.ensureMoveVisible()

	case "up", "k":
		// Skip non-directory nodes — only folders are valid move targets
		for prev := f.moveCursor - 1; prev >= 0; prev-- {
			if moveVisible[prev].isDir {
				f.moveCursor = prev
				break
			}
		}
		f.ensureMoveVisible()

	case "l", "right":
		node := moveVisible[f.moveCursor]
		if node.isDir && !node.expanded {
			node.expanded = true
			f.dirState[node.path] = true
			f.rebuildVisible()
		}

	case "h", "left":
		node := moveVisible[f.moveCursor]
		if node.isDir && node.expanded {
			node.expanded = false
			f.dirState[node.path] = false
			f.rebuildVisible()
		}

	case "enter":
		node := moveVisible[f.moveCursor]
		// Only allow selecting directories or the (root) node
		if node.isDir {
			destDir := node.path // "" for root node
			source := f.moveSource
			isDir := f.moveSourceDir
			f.exitMoveMode()
			return f, func() tea.Msg {
				return MoveConfirmedMsg{
					SourcePath: source,
					DestDir:    destDir,
					IsDir:      isDir,
				}
			}
		}

	case "esc":
		f.exitMoveMode()
		return f, func() tea.Msg { return MoveCanceledMsg{} }
	}

	return f, nil
}

// moveVisibleNodes returns the visible node list for move mode:
// a synthetic (root) node followed by all normal visible nodes.
func (f *FileList) moveVisibleNodes() []*treeNode {
	rootNode := &treeNode{
		name:  "(root)",
		path:  "",
		depth: 0,
		isDir: true,
	}
	nodes := make([]*treeNode, 0, len(f.visible)+1)
	nodes = append(nodes, rootNode)
	nodes = append(nodes, f.visible...)
	return nodes
}

func (f *FileList) ensureMoveVisible() {
	visibleLines := f.moveVisibleLines()
	if f.moveCursor < f.moveOffset {
		f.moveOffset = f.moveCursor
	}
	if f.moveCursor >= f.moveOffset+visibleLines {
		f.moveOffset = f.moveCursor - visibleLines + 1
	}
}

// visibleLines returns the number of content lines visible in normal mode.
// Chrome: header(1) + separator(1) = 2 lines.
func (f *FileList) visibleLines() int {
	lines := f.height - 2 // header(1) + sep(1)
	if lines < 1 {
		lines = 1
	}
	return lines
}

// moveVisibleLines returns the number of content lines visible in move mode.
// Move mode has extra footer chrome: footer separator(1) + footer hint(1).
func (f *FileList) moveVisibleLines() int {
	lines := f.height - 4 // header(1) + sep(1) + footer sep(1) + footer hint(1)
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
	if f.moveMode {
		return f.viewMoveMode()
	}

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

	return f.theme.Sidebar.Width(f.width).Height(f.height).MaxHeight(f.height).Render(b.String())
}

// viewMoveMode renders the file list in move mode.
func (f *FileList) viewMoveMode() string {
	var b strings.Builder

	// Header — indicates move mode
	header := f.theme.Title.Render("MOVE: pick destination")
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(f.theme.Dimmed.Render(safeRepeat("─", f.width-2)))
	b.WriteString("\n")

	moveNodes := f.moveVisibleNodes()
	visibleLines := f.moveVisibleLines()
	contentWidth := f.width - 2

	endIdx := f.moveOffset + visibleLines
	if endIdx > len(moveNodes) {
		endIdx = len(moveNodes)
	}

	for i := f.moveOffset; i < endIdx; i++ {
		node := moveNodes[i]
		isSelected := i == f.moveCursor
		isMoveSource := node.path == f.moveSource && node.path != ""

		indent := safeRepeat("  ", node.depth)

		if node.isDir {
			arrow := IndicatorFolderOpen
			if !node.expanded && node.path != "" { // root has no expand state
				arrow = IndicatorFolderClosed
			}
			if node.path == "" {
				// (root) node — special rendering
				arrow = "◇"
			}

			maxLen := f.width - 6 - (node.depth * 2)
			dirName := truncate(node.name, maxLen)

			var line string
			switch {
			case isMoveSource:
				// Source item is dimmed
				line = f.theme.Dimmed.PaddingLeft(1).Render(fmt.Sprintf(" %s%s %s", indent, arrow, dirName))
			case isSelected:
				raw := fmt.Sprintf("%s%s%s %s", IndicatorSelected, indent, arrow, dirName)
				line = f.theme.TreeSelected.Render(padToWidth(raw, contentWidth))
			default:
				line = f.theme.TreeItem.Render(fmt.Sprintf(" %s%s %s", indent, arrow, dirName))
			}

			b.WriteString(line)
			b.WriteString("\n")
		} else {
			// Files are shown dimmed (not valid targets) in move mode
			maxLen := f.width - 6 - (node.depth * 2)
			name := truncate(node.name, maxLen)

			var line string
			if isMoveSource {
				line = f.theme.Dimmed.PaddingLeft(1).Render(fmt.Sprintf(" %s~ %s", indent, name))
			} else {
				line = f.theme.Dimmed.PaddingLeft(1).Render(fmt.Sprintf(" %s  %s", indent, name))
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Fill remaining space
	contentLines := len(moveNodes)
	if contentLines > visibleLines {
		contentLines = visibleLines
	}
	for i := contentLines; i < visibleLines; i++ {
		b.WriteString("\n")
	}

	// Footer hint
	b.WriteString(f.theme.Dimmed.Render(safeRepeat("─", f.width-2)))
	b.WriteString("\n")
	b.WriteString(f.theme.Dimmed.Render(" Enter=move  Esc=cancel"))

	return f.theme.Sidebar.Width(f.width).Height(f.height).MaxHeight(f.height).Render(b.String())
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

// SetSize sets the file list dimensions and ensures the cursor remains visible.
func (f *FileList) SetSize(width, height int) {
	f.width = width
	f.height = height
	f.ensureVisible()
}

// SetPosition sets the screen Y offset for mouse coordinate translation.
func (f *FileList) SetPosition(screenY int) {
	f.screenY = screenY
}

// ScrollBy adjusts the scroll offset by delta lines, clamping to valid range.
// If the cursor goes out of view, it is moved to stay visible.
func (f *FileList) ScrollBy(delta int) {
	if len(f.visible) == 0 {
		return
	}

	visibleLines := f.visibleLines()
	maxOffset := len(f.visible) - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}

	f.offset += delta
	if f.offset < 0 {
		f.offset = 0
	}
	if f.offset > maxOffset {
		f.offset = maxOffset
	}

	// Keep cursor within the visible window
	if f.cursor < f.offset {
		f.cursor = f.offset
	}
	if f.cursor >= f.offset+visibleLines {
		f.cursor = f.offset + visibleLines - 1
	}
}

// HandleMouse processes a mouse event and returns the action taken.
// The caller should use the returned ClickAction to decide follow-up behavior
// (e.g., loading a file preview, switching focus).
func (f *FileList) HandleMouse(msg tea.MouseMsg) ClickAction {
	if f.moveMode {
		return ClickNone
	}

	me := tea.MouseEvent(msg)

	// Wheel events: scroll the file list
	if me.IsWheel() {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			f.ScrollBy(-3)
		case tea.MouseButtonWheelDown:
			f.ScrollBy(3)
		}
		return ClickNone
	}

	// Only handle left-click press
	if msg.Button != tea.MouseButtonLeft || me.Action != tea.MouseActionPress {
		return ClickNone
	}

	// Translate screen Y to item index.
	// Layout: screenY + header(1) + separator(1) + content rows
	headerLines := 2 // "FILES" header + "───" separator
	relY := msg.Y - f.screenY - headerLines
	if relY < 0 {
		return ClickNone
	}

	idx := f.offset + relY
	if idx < 0 || idx >= len(f.visible) {
		return ClickNone
	}

	node := f.visible[idx]
	f.cursor = idx
	f.ensureVisible()

	if node.isDir {
		// Toggle expand/collapse
		node.expanded = !node.expanded
		f.dirState[node.path] = node.expanded
		f.rebuildVisible()
		// Clamp cursor after rebuild (collapsing may remove children)
		if f.cursor >= len(f.visible) {
			f.cursor = len(f.visible) - 1
		}
		return ClickDirToggle
	}

	return ClickFile
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

// SelectedDirPath returns the relative path of the selected directory node.
// Returns empty string if the cursor is not on a directory.
func (f *FileList) SelectedDirPath() string {
	if f.cursor >= 0 && f.cursor < len(f.visible) && f.visible[f.cursor].isDir {
		return f.visible[f.cursor].path
	}
	return ""
}

// SelectedPath returns the relative path of the currently selected node (file or dir).
// Returns empty string if nothing is selected.
func (f *FileList) SelectedPath() string {
	if f.cursor >= 0 && f.cursor < len(f.visible) {
		return f.visible[f.cursor].path
	}
	return ""
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

// SelectPath selects a node by its relative path (works for both files and dirs).
func (f *FileList) SelectPath(path string) {
	for i, node := range f.visible {
		if node.path == path {
			f.cursor = i
			f.ensureVisible()
			return
		}
	}
}

// EnterMoveMode enters move mode for the currently selected item.
func (f *FileList) EnterMoveMode() {
	if f.cursor < 0 || f.cursor >= len(f.visible) {
		return
	}
	node := f.visible[f.cursor]
	f.moveMode = true
	f.moveSource = node.path
	f.moveSourceDir = node.isDir
	f.moveCursor = 0 // start on (root)
	f.moveOffset = 0
}

// exitMoveMode clears move state but preserves moveSource/moveSourceDir
// so the confirmed message can still reference them.
func (f *FileList) exitMoveMode() {
	f.moveMode = false
	f.moveCursor = 0
	f.moveOffset = 0
}

// ExitMoveMode exits move mode and clears all move state.
func (f *FileList) ExitMoveMode() {
	f.moveMode = false
	f.moveSource = ""
	f.moveSourceDir = false
	f.moveCursor = 0
	f.moveOffset = 0
}

// InMoveMode returns whether the file list is in move mode.
func (f *FileList) InMoveMode() bool {
	return f.moveMode
}

// FileCount returns the number of markdown files (not tree nodes)
func (f *FileList) FileCount() int {
	return len(f.files)
}
