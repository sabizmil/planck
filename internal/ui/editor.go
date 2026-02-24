package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// EditorMode represents the current mode of the editor
type EditorMode int

const (
	EditorModeView EditorMode = iota
	EditorModeEdit
)

// Editor displays and edits markdown content
type Editor struct {
	theme    *Theme
	renderer *glamour.TermRenderer

	// Style state for rebuilding renderer on resize
	styleJSON []byte

	// Content
	fileName  string
	content   string
	lines     []string
	rendered  string
	lineCount int

	// Mode
	mode EditorMode

	// View mode state
	viewOffset int

	// Edit mode state
	cursorRow int
	cursorCol int
	modified  bool

	// Selection state (for click-and-drag and future shift+arrow)
	selecting    bool // true while a mouse drag is in progress
	hasSelection bool // true when a visible selection exists
	selAnchorRow int  // fixed end of the selection (where drag started)
	selAnchorCol int
	selEndRow    int // moving end of the selection (follows mouse/cursor)
	selEndCol    int

	// Dimensions
	width   int
	height  int
	focused bool

	// Screen position (for mouse coordinate translation)
	screenX int
	screenY int
}

// NewEditor creates a new editor
func NewEditor(theme *Theme) *Editor {
	renderer, _ := NewMarkdownRenderer(80)

	return &Editor{
		theme:    theme,
		renderer: renderer,
		mode:     EditorModeView,
	}
}

// Init initializes the editor
func (e *Editor) Init() tea.Cmd {
	return nil
}

// FileSavedMsg is sent when the file is saved
type FileSavedMsg struct {
	FileName string
	Content  string
}

// Update handles messages
func (e *Editor) Update(msg tea.Msg) (*Editor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Mouse events don't check e.focused — the app controls when to forward them
		return e.handleMouse(msg)
	case tea.KeyMsg:
		if !e.focused {
			return e, nil
		}
		if e.mode == EditorModeView {
			return e.updateViewMode(msg)
		}
		return e.updateEditMode(msg)
	}

	return e, nil
}

func (e *Editor) updateViewMode(msg tea.KeyMsg) (*Editor, tea.Cmd) {
	switch msg.String() {
	case "e":
		// Enter edit mode
		e.mode = EditorModeEdit
		e.cursorRow = 0
		e.cursorCol = 0
		e.parseLines()

	case "down", "j":
		e.viewOffset++
		maxOffset := e.lineCount - e.visibleLines()
		if e.viewOffset > maxOffset {
			e.viewOffset = maxOffset
		}
		if e.viewOffset < 0 {
			e.viewOffset = 0
		}

	case "up", "k":
		e.viewOffset--
		if e.viewOffset < 0 {
			e.viewOffset = 0
		}

	case "pgdown", "ctrl+d":
		e.viewOffset += e.visibleLines() / 2
		maxOffset := e.lineCount - e.visibleLines()
		if e.viewOffset > maxOffset {
			e.viewOffset = maxOffset
		}
		if e.viewOffset < 0 {
			e.viewOffset = 0
		}

	case "pgup", "ctrl+u":
		e.viewOffset -= e.visibleLines() / 2
		if e.viewOffset < 0 {
			e.viewOffset = 0
		}

	case "home", "g":
		e.viewOffset = 0

	case "end", "G":
		e.viewOffset = e.lineCount - e.visibleLines()
		if e.viewOffset < 0 {
			e.viewOffset = 0
		}
	}

	return e, nil
}

func (e *Editor) updateEditMode(msg tea.KeyMsg) (*Editor, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		// Exit edit mode, auto-save if modified
		e.clearSelection()
		e.rebuildContent()
		e.mode = EditorModeView
		if e.modified {
			content := e.content
			fileName := e.fileName
			return e, func() tea.Msg {
				return FileSavedMsg{FileName: fileName, Content: content}
			}
		}
		return e, nil

	case tea.KeyCtrlS:
		// Save
		e.rebuildContent()
		content := e.content
		fileName := e.fileName
		return e, func() tea.Msg {
			return FileSavedMsg{FileName: fileName, Content: content}
		}

	case tea.KeyLeft:
		e.clearSelection()
		if e.cursorCol > 0 {
			e.cursorCol--
		} else if e.cursorRow > 0 {
			// Move to end of previous line
			e.cursorRow--
			e.cursorCol = len(e.lines[e.cursorRow])
		}

	case tea.KeyRight:
		e.clearSelection()
		if e.cursorRow < len(e.lines) {
			lineLen := len(e.lines[e.cursorRow])
			if e.cursorCol < lineLen {
				e.cursorCol++
			} else if e.cursorRow < len(e.lines)-1 {
				// Move to start of next line
				e.cursorRow++
				e.cursorCol = 0
			}
		}

	case tea.KeyUp:
		e.clearSelection()
		e.moveCursorUpVisual()
		e.ensureCursorVisible()

	case tea.KeyDown:
		e.clearSelection()
		e.moveCursorDownVisual()
		e.ensureCursorVisible()

	case tea.KeyHome, tea.KeyCtrlA:
		e.clearSelection()
		e.cursorCol = 0

	case tea.KeyEnd, tea.KeyCtrlE:
		e.clearSelection()
		if e.cursorRow < len(e.lines) {
			e.cursorCol = len(e.lines[e.cursorRow])
		}

	case tea.KeyEnter:
		if e.hasSelection {
			e.deleteSelection()
		}
		e.insertNewline()
		e.modified = true

	case tea.KeyBackspace:
		if e.hasSelection {
			e.deleteSelection()
			e.modified = true
		} else {
			e.deleteBackward()
			e.modified = true
		}

	case tea.KeyDelete:
		if e.hasSelection {
			e.deleteSelection()
			e.modified = true
		} else {
			e.deleteForward()
			e.modified = true
		}

	case tea.KeyTab:
		if e.hasSelection {
			e.deleteSelection()
		}
		e.insertText("    ")
		e.modified = true

	case tea.KeyRunes:
		if e.hasSelection {
			e.deleteSelection()
		}
		e.insertText(string(msg.Runes))
		e.modified = true

	case tea.KeySpace:
		if e.hasSelection {
			e.deleteSelection()
		}
		e.insertText(" ")
		e.modified = true
	}

	return e, nil
}

func (e *Editor) parseLines() {
	e.lines = strings.Split(e.content, "\n")
	if len(e.lines) == 0 {
		e.lines = []string{""}
	}
}

func (e *Editor) rebuildContent() {
	e.content = strings.Join(e.lines, "\n")
	e.renderContent()
}

func (e *Editor) insertText(text string) {
	if e.cursorRow >= len(e.lines) {
		e.lines = append(e.lines, "")
	}

	line := e.lines[e.cursorRow]
	if e.cursorCol > len(line) {
		e.cursorCol = len(line)
	}

	newLine := line[:e.cursorCol] + text + line[e.cursorCol:]
	e.lines[e.cursorRow] = newLine
	e.cursorCol += len(text)
}

func (e *Editor) insertNewline() {
	if e.cursorRow >= len(e.lines) {
		e.lines = append(e.lines, "")
		e.cursorRow = len(e.lines) - 1
		e.cursorCol = 0
		return
	}

	line := e.lines[e.cursorRow]
	if e.cursorCol > len(line) {
		e.cursorCol = len(line)
	}

	// Split line at cursor
	before := line[:e.cursorCol]
	after := line[e.cursorCol:]

	// Insert new line
	newLines := make([]string, len(e.lines)+1)
	copy(newLines, e.lines[:e.cursorRow+1])
	newLines[e.cursorRow] = before
	newLines[e.cursorRow+1] = after
	copy(newLines[e.cursorRow+2:], e.lines[e.cursorRow+1:])

	e.lines = newLines
	e.cursorRow++
	e.cursorCol = 0
	e.ensureCursorVisible()
}

func (e *Editor) deleteBackward() {
	if e.cursorCol > 0 {
		// Delete character before cursor
		line := e.lines[e.cursorRow]
		if e.cursorCol <= len(line) {
			e.lines[e.cursorRow] = line[:e.cursorCol-1] + line[e.cursorCol:]
			e.cursorCol--
		}
	} else if e.cursorRow > 0 {
		// Join with previous line
		prevLine := e.lines[e.cursorRow-1]
		currLine := e.lines[e.cursorRow]

		e.lines[e.cursorRow-1] = prevLine + currLine

		// Remove current line
		newLines := make([]string, len(e.lines)-1)
		copy(newLines, e.lines[:e.cursorRow])
		copy(newLines[e.cursorRow:], e.lines[e.cursorRow+1:])
		e.lines = newLines

		e.cursorRow--
		e.cursorCol = len(prevLine)
	}
}

func (e *Editor) deleteForward() {
	if e.cursorRow >= len(e.lines) {
		return
	}

	line := e.lines[e.cursorRow]
	if e.cursorCol < len(line) {
		// Delete character at cursor
		e.lines[e.cursorRow] = line[:e.cursorCol] + line[e.cursorCol+1:]
	} else if e.cursorRow < len(e.lines)-1 {
		// Join with next line
		nextLine := e.lines[e.cursorRow+1]
		e.lines[e.cursorRow] = line + nextLine

		// Remove next line
		newLines := make([]string, len(e.lines)-1)
		copy(newLines, e.lines[:e.cursorRow+1])
		copy(newLines[e.cursorRow+1:], e.lines[e.cursorRow+2:])
		e.lines = newLines
	}
}

// moveCursorUpVisual moves the cursor up by one visual (wrapped) line.
func (e *Editor) moveCursorUpVisual() {
	maxLineWidth := e.editMaxLineWidth()
	vlines := e.buildVisualLines(maxLineWidth)
	curVRow := cursorVisualRow(vlines, e.cursorRow, e.cursorCol)

	if curVRow <= 0 {
		return // already at top
	}

	prev := vlines[curVRow-1]
	e.cursorRow = prev.logicalRow
	// Try to keep same visual column position
	localCol := e.cursorCol - vlines[curVRow].colOffset
	newCol := prev.colOffset + localCol
	if newCol > prev.colOffset+len(prev.text) {
		newCol = prev.colOffset + len(prev.text)
	}
	e.cursorCol = newCol
}

// moveCursorDownVisual moves the cursor down by one visual (wrapped) line.
func (e *Editor) moveCursorDownVisual() {
	maxLineWidth := e.editMaxLineWidth()
	vlines := e.buildVisualLines(maxLineWidth)
	curVRow := cursorVisualRow(vlines, e.cursorRow, e.cursorCol)

	if curVRow >= len(vlines)-1 {
		return // already at bottom
	}

	next := vlines[curVRow+1]
	e.cursorRow = next.logicalRow
	// Try to keep same visual column position
	localCol := e.cursorCol - vlines[curVRow].colOffset
	newCol := next.colOffset + localCol
	if newCol > next.colOffset+len(next.text) {
		newCol = next.colOffset + len(next.text)
	}
	e.cursorCol = newCol
}

func (e *Editor) ensureCursorVisible() {
	visibleLines := e.visibleLines()
	maxLineWidth := e.editMaxLineWidth()
	vlines := e.buildVisualLines(maxLineWidth)
	curVRow := cursorVisualRow(vlines, e.cursorRow, e.cursorCol)

	if curVRow < e.viewOffset {
		e.viewOffset = curVRow
	}
	if curVRow >= e.viewOffset+visibleLines {
		e.viewOffset = curVRow - visibleLines + 1
	}
}

func (e *Editor) visibleLines() int {
	chrome := 3 // header(1) + sep(1) + footer sep(1)
	if e.mode == EditorModeEdit {
		chrome = 4 // + cursor pos line
	}
	lines := e.height - chrome
	if lines < 1 {
		lines = 1
	}
	return lines
}

func (e *Editor) renderContent() {
	if e.renderer == nil || e.content == "" {
		e.rendered = ""
		e.lineCount = 0
		return
	}

	rendered, err := e.renderer.Render(e.content)
	if err != nil {
		e.rendered = e.content
	} else {
		e.rendered = strings.TrimSpace(rendered)
	}

	e.lineCount = strings.Count(e.rendered, "\n") + 1
}

// View renders the editor
func (e *Editor) View() string {
	var b strings.Builder

	// Header
	var modeIndicator string
	if e.mode == EditorModeEdit {
		modeIndicator = e.theme.StatusProgress.Render("[EDIT]")
		if e.modified {
			modeIndicator += e.theme.StatusProgress.Render(" *")
		}
	} else {
		modeIndicator = e.theme.Dimmed.Render("[VIEW]")
	}

	header := fmt.Sprintf("%s  %s", e.theme.Title.Render(e.fileName), modeIndicator)
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(e.theme.Dimmed.Render(safeRepeat("─", e.width-4)))
	b.WriteString("\n")

	visibleLines := e.visibleLines()

	if e.mode == EditorModeEdit {
		// Edit mode: show raw text with cursor
		b.WriteString(e.renderEditMode(visibleLines))
	} else {
		// View mode: show rendered markdown
		b.WriteString(e.renderViewMode(visibleLines))
	}

	if e.mode == EditorModeEdit {
		b.WriteString("\n")
		b.WriteString(e.theme.Dimmed.Render(fmt.Sprintf("Ln %d, Col %d", e.cursorRow+1, e.cursorCol+1)))
	}

	return e.theme.DetailPanel.Width(e.width).Height(e.height).Render(b.String())
}

func (e *Editor) renderViewMode(visibleLines int) string {
	var b strings.Builder

	if e.content == "" {
		b.WriteString(e.theme.Dimmed.Render("No content"))
		b.WriteString("\n")
		for i := 1; i < visibleLines; i++ {
			b.WriteString("\n")
		}
		return b.String()
	}

	lines := strings.Split(e.rendered, "\n")
	e.lineCount = len(lines)

	start := e.viewOffset
	end := e.viewOffset + visibleLines

	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	for i := start; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}

	// Fill remaining space
	for i := end - start; i < visibleLines; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

// wrapLine splits a logical line into visual lines at word boundaries.
// If a word is longer than maxWidth, it falls back to character-level wrapping.
func wrapLine(line string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 10
	}
	if len(line) <= maxWidth {
		return []string{line}
	}

	var result []string
	remaining := line
	for len(remaining) > maxWidth {
		// Find the last space within maxWidth
		splitAt := -1
		for i := maxWidth; i >= 0; i-- {
			if i < len(remaining) && remaining[i] == ' ' {
				splitAt = i
				break
			}
		}

		if splitAt <= 0 {
			// No space found — hard wrap at maxWidth
			result = append(result, remaining[:maxWidth])
			remaining = remaining[maxWidth:]
		} else {
			result = append(result, remaining[:splitAt])
			remaining = remaining[splitAt+1:] // skip the space
		}
	}
	result = append(result, remaining)
	return result
}

// visualLine represents a single display row in the wrapped editor view.
type visualLine struct {
	logicalRow int    // index into e.lines
	wrapIndex  int    // which wrapped segment (0 = first)
	text       string // the text content of this visual line
	colOffset  int    // character offset in the logical line where this segment starts
}

// buildVisualLines builds the full list of visual (wrapped) lines for the editor.
func (e *Editor) buildVisualLines(maxLineWidth int) []visualLine {
	var vlines []visualLine
	for i, line := range e.lines {
		wrapped := wrapLine(line, maxLineWidth)
		offset := 0
		for j, seg := range wrapped {
			vlines = append(vlines, visualLine{
				logicalRow: i,
				wrapIndex:  j,
				text:       seg,
				colOffset:  offset,
			})
			offset += len(seg)
			if j < len(wrapped)-1 {
				offset++ // account for the space consumed by word wrap split
			}
		}
	}
	if len(vlines) == 0 {
		vlines = []visualLine{{logicalRow: 0, wrapIndex: 0, text: "", colOffset: 0}}
	}
	return vlines
}

// cursorVisualRow returns the visual row index where the cursor currently sits.
func cursorVisualRow(vlines []visualLine, logicalRow, logicalCol int) int {
	for i, vl := range vlines {
		if vl.logicalRow != logicalRow {
			continue
		}
		// Check if cursor falls within this visual line segment
		segEnd := vl.colOffset + len(vl.text)
		if logicalCol >= vl.colOffset && logicalCol <= segEnd {
			return i
		}
	}
	// Fallback: return last visual line of this logical row
	last := 0
	for i, vl := range vlines {
		if vl.logicalRow == logicalRow {
			last = i
		}
	}
	return last
}

func (e *Editor) editMaxLineWidth() int {
	lineNumWidth := len(fmt.Sprintf("%d", len(e.lines)))
	if lineNumWidth < 2 {
		lineNumWidth = 2
	}
	maxLineWidth := e.width - lineNumWidth - 8
	if maxLineWidth < 10 {
		maxLineWidth = 10
	}
	return maxLineWidth
}

func (e *Editor) renderEditMode(visibleLines int) string {
	var b strings.Builder

	if len(e.lines) == 0 {
		e.lines = []string{""}
	}

	lineNumWidth := len(fmt.Sprintf("%d", len(e.lines)))
	if lineNumWidth < 2 {
		lineNumWidth = 2
	}

	maxLineWidth := e.editMaxLineWidth()
	vlines := e.buildVisualLines(maxLineWidth)

	// Find cursor visual position
	cursorVRow := cursorVisualRow(vlines, e.cursorRow, e.cursorCol)

	start := e.viewOffset
	end := e.viewOffset + visibleLines

	if start >= len(vlines) {
		start = len(vlines) - 1
	}
	if start < 0 {
		start = 0
	}
	if end > len(vlines) {
		end = len(vlines)
	}

	cursorStyle := lipgloss.NewStyle().Reverse(true)
	selectionStyle := lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))

	for i := start; i < end; i++ {
		vl := vlines[i]

		// Line number gutter: show number only on first visual line of a logical line
		if vl.wrapIndex == 0 {
			lineNum := fmt.Sprintf("%*d", lineNumWidth, vl.logicalRow+1)
			b.WriteString(e.theme.Dimmed.Render(lineNum + " │ "))
		} else {
			blank := fmt.Sprintf("%*s", lineNumWidth, "")
			b.WriteString(e.theme.Dimmed.Render(blank + " · "))
		}

		// Determine cursor position on this visual line (if any)
		hasCursor := i == cursorVRow
		localCursorCol := -1
		if hasCursor {
			localCursorCol = e.cursorCol - vl.colOffset
			if localCursorCol < 0 {
				localCursorCol = 0
			}
			if localCursorCol > len(vl.text) {
				localCursorCol = len(vl.text)
			}
		}

		// Render each character with the appropriate style
		switch {
		case e.hasSelection:
			// Selection-aware rendering: build runs of same-styled characters
			for j := 0; j <= len(vl.text); j++ {
				logCol := vl.colOffset + j

				switch {
				case hasCursor && j == localCursorCol:
					// Cursor position
					if j >= len(vl.text) {
						b.WriteString(cursorStyle.Render(" "))
					} else {
						b.WriteString(cursorStyle.Render(string(vl.text[j])))
					}
				case j < len(vl.text):
					if e.isInSelection(vl.logicalRow, logCol) {
						b.WriteString(selectionStyle.Render(string(vl.text[j])))
					} else {
						b.WriteString(e.theme.Normal.Render(string(vl.text[j])))
					}
				}
				// j == len(vl.text) and no cursor: nothing to render
			}
		case hasCursor:
			// No selection, but cursor is on this line
			if localCursorCol >= len(vl.text) {
				b.WriteString(e.theme.Normal.Render(vl.text))
				b.WriteString(cursorStyle.Render(" "))
			} else {
				before := vl.text[:localCursorCol]
				cursor := string(vl.text[localCursorCol])
				after := vl.text[localCursorCol+1:]
				b.WriteString(e.theme.Normal.Render(before))
				b.WriteString(cursorStyle.Render(cursor))
				b.WriteString(e.theme.Normal.Render(after))
			}
		default:
			b.WriteString(e.theme.Normal.Render(vl.text))
		}

		b.WriteString("\n")
	}

	// Fill remaining space
	for i := end - start; i < visibleLines; i++ {
		lineNum := fmt.Sprintf("%*s", lineNumWidth, "~")
		b.WriteString(e.theme.Dimmed.Render(lineNum + " │ "))
		b.WriteString("\n")
	}

	return b.String()
}

// SetContent sets the editor content
func (e *Editor) SetContent(fileName, content string) {
	e.fileName = fileName
	e.content = content
	e.mode = EditorModeView
	e.viewOffset = 0
	e.cursorRow = 0
	e.cursorCol = 0
	e.modified = false
	e.clearSelection()
	e.parseLines()
	e.renderContent()
}

// GetContent returns the current content
func (e *Editor) GetContent() string {
	if e.mode == EditorModeEdit {
		return strings.Join(e.lines, "\n")
	}
	return e.content
}

// SetFocused sets the focused state
func (e *Editor) SetFocused(focused bool) {
	e.focused = focused
}

// SetSize sets the editor dimensions
func (e *Editor) SetSize(width, height int) {
	e.width = width
	e.height = height

	// Update renderer word wrap
	if e.renderer != nil {
		if e.styleJSON != nil {
			e.renderer, _ = NewMarkdownRendererWithStyle(e.styleJSON, width-8)
		} else {
			e.renderer, _ = NewMarkdownRenderer(width - 8)
		}
		if e.mode == EditorModeView {
			e.renderContent()
		}
	}
}

// SetMarkdownStyle replaces the renderer with a new style and re-renders.
func (e *Editor) SetMarkdownStyle(styleJSON []byte) {
	e.styleJSON = styleJSON
	width := e.width - 8
	if width < 20 {
		width = 80
	}
	e.renderer, _ = NewMarkdownRendererWithStyle(styleJSON, width)
	if e.mode == EditorModeView {
		e.renderContent()
	}
}

// Mode returns the current editor mode
func (e *Editor) Mode() EditorMode {
	return e.mode
}

// IsModified returns whether the content has been modified
func (e *Editor) IsModified() bool {
	return e.modified
}

// FileName returns the current file name
func (e *Editor) FileName() string {
	return e.fileName
}

// EnterEditMode enters edit mode
func (e *Editor) EnterEditMode() {
	e.mode = EditorModeEdit
	e.cursorRow = 0
	e.cursorCol = 0
	e.clearSelection()
	e.parseLines()
}

// ExitEditMode exits edit mode
func (e *Editor) ExitEditMode() {
	e.mode = EditorModeView
	e.clearSelection()
	e.rebuildContent()
}

// ClearModified clears the modified flag
func (e *Editor) ClearModified() {
	e.modified = false
}

// Selecting returns true while a mouse drag selection is in progress.
func (e *Editor) Selecting() bool {
	return e.selecting
}

// ScrollBy scrolls the view by the given number of lines. Works in both view and edit modes.
func (e *Editor) ScrollBy(delta int) {
	e.viewOffset += delta

	var totalLines int
	if e.mode == EditorModeEdit {
		maxLineWidth := e.editMaxLineWidth()
		vlines := e.buildVisualLines(maxLineWidth)
		totalLines = len(vlines)
	} else {
		totalLines = e.lineCount
	}
	maxOffset := totalLines - e.visibleLines()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if e.viewOffset > maxOffset {
		e.viewOffset = maxOffset
	}
	if e.viewOffset < 0 {
		e.viewOffset = 0
	}
}

// clearSelection removes any active selection.
func (e *Editor) clearSelection() {
	e.hasSelection = false
	e.selecting = false
}

// selectionRange returns the selection bounds in document order (start <= end).
// Returns (startRow, startCol, endRow, endCol).
func (e *Editor) selectionRange() (startRow, startCol, endRow, endCol int) {
	ar, ac := e.selAnchorRow, e.selAnchorCol
	er, ec := e.selEndRow, e.selEndCol
	if ar > er || (ar == er && ac > ec) {
		return er, ec, ar, ac
	}
	return ar, ac, er, ec
}

// isInSelection returns whether the given logical (row, col) position falls
// within the current selection. The selection is inclusive of the start
// position and exclusive of the end position (like a half-open range on
// the linearized text).
func (e *Editor) isInSelection(row, col int) bool {
	if !e.hasSelection {
		return false
	}
	sr, sc, er, ec := e.selectionRange()
	if row < sr || row > er {
		return false
	}
	if row == sr && col < sc {
		return false
	}
	if row == er && col >= ec {
		return false
	}
	return true
}

// deleteSelection removes all text in the current selection and positions the
// cursor at the start of the former selection. Returns true if a selection was
// deleted, false if there was no selection.
func (e *Editor) deleteSelection() bool {
	if !e.hasSelection {
		return false
	}

	sr, sc, er, ec := e.selectionRange()

	if sr == er {
		// Single-line selection: remove characters [sc, ec) from that line
		line := e.lines[sr]
		if sc > len(line) {
			sc = len(line)
		}
		if ec > len(line) {
			ec = len(line)
		}
		e.lines[sr] = line[:sc] + line[ec:]
	} else {
		// Multi-line selection: keep prefix of start line + suffix of end line,
		// remove everything in between.
		startLine := e.lines[sr]
		endLine := e.lines[er]

		if sc > len(startLine) {
			sc = len(startLine)
		}
		if ec > len(endLine) {
			ec = len(endLine)
		}

		merged := startLine[:sc] + endLine[ec:]
		newLines := make([]string, 0, len(e.lines)-(er-sr))
		newLines = append(newLines, e.lines[:sr]...)
		newLines = append(newLines, merged)
		newLines = append(newLines, e.lines[er+1:]...)
		e.lines = newLines
	}

	e.cursorRow = sr
	e.cursorCol = sc
	e.clearSelection()
	return true
}

// selectedText returns the currently selected text, or empty string if no selection.
func (e *Editor) selectedText() string {
	if !e.hasSelection {
		return ""
	}

	sr, sc, er, ec := e.selectionRange()

	if sr == er {
		line := e.lines[sr]
		if sc > len(line) {
			sc = len(line)
		}
		if ec > len(line) {
			ec = len(line)
		}
		return line[sc:ec]
	}

	var b strings.Builder
	// First line: from sc to end
	firstLine := e.lines[sr]
	if sc > len(firstLine) {
		sc = len(firstLine)
	}
	b.WriteString(firstLine[sc:])

	// Middle lines: entire lines
	for i := sr + 1; i < er; i++ {
		b.WriteString("\n")
		b.WriteString(e.lines[i])
	}

	// Last line: from start to ec
	b.WriteString("\n")
	lastLine := e.lines[er]
	if ec > len(lastLine) {
		ec = len(lastLine)
	}
	b.WriteString(lastLine[:ec])

	return b.String()
}

// SetPosition sets the screen position of the editor panel (for mouse coordinate translation).
func (e *Editor) SetPosition(x, y int) {
	e.screenX = x
	e.screenY = y
}

// mouseToLogical translates raw screen coordinates from a mouse event into
// logical (row, col) positions in the text buffer. Returns (-1, -1) if the
// coordinates fall outside the editable area. Only valid in edit mode.
func (e *Editor) mouseToLogical(screenX, screenY int) (logRow, logCol int) {
	relY := screenY - e.screenY - 2 // header + separator
	relX := screenX - e.screenX - 2 // panel padding

	if relY < 0 || relX < 0 {
		return -1, -1
	}

	vrow := relY + e.viewOffset

	lineNumWidth := len(fmt.Sprintf("%d", len(e.lines)))
	if lineNumWidth < 2 {
		lineNumWidth = 2
	}
	col := relX - lineNumWidth - 3 // line number + " │ "

	maxLineWidth := e.editMaxLineWidth()
	vlines := e.buildVisualLines(maxLineWidth)

	if vrow < 0 {
		vrow = 0
	}
	if vrow >= len(vlines) {
		vrow = len(vlines) - 1
	}

	vl := vlines[vrow]

	if col < 0 {
		col = 0
	}
	if col > len(vl.text) {
		col = len(vl.text)
	}

	return vl.logicalRow, vl.colOffset + col
}

// handleMouse processes mouse events for scrolling, click-to-place-cursor,
// and click-and-drag text selection.
func (e *Editor) handleMouse(msg tea.MouseMsg) (*Editor, tea.Cmd) {
	me := tea.MouseEvent(msg)

	// Wheel scroll (both modes) — but suppress while a drag selection is active,
	// since trackpad scroll events during a click-drag would shift the viewport
	// and cause the selection to jump erratically.
	if me.IsWheel() {
		if !e.selecting {
			if msg.Button == tea.MouseButtonWheelUp {
				e.ScrollBy(-3)
			} else if msg.Button == tea.MouseButtonWheelDown {
				e.ScrollBy(3)
			}
		}
		return e, nil
	}

	switch me.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return e, nil
		}
		return e.handleMousePress(msg)

	case tea.MouseActionMotion:
		if !e.selecting {
			return e, nil
		}
		return e.handleMouseMotion(msg)

	case tea.MouseActionRelease:
		if !e.selecting {
			return e, nil
		}
		return e.handleMouseRelease(msg)
	}

	return e, nil
}

// mouseToLogicalClamped is like mouseToLogical but clamps the screen Y to
// the visible content area. This prevents mapping to off-screen visual rows
// when the mouse is dragged past the top or bottom edge of the editor,
// avoiding feedback loops with viewport scrolling.
func (e *Editor) mouseToLogicalClamped(screenX, screenY int) (logRow, logCol int) {
	relY := screenY - e.screenY - 2 // header + separator
	relX := screenX - e.screenX - 2 // panel padding

	if relX < 0 {
		relX = 0
	}

	// Clamp relY to the visible content area
	if relY < 0 {
		relY = 0
	}
	maxRelY := e.visibleLines() - 1
	if maxRelY < 0 {
		maxRelY = 0
	}
	if relY > maxRelY {
		relY = maxRelY
	}

	vrow := relY + e.viewOffset

	lineNumWidth := len(fmt.Sprintf("%d", len(e.lines)))
	if lineNumWidth < 2 {
		lineNumWidth = 2
	}
	col := relX - lineNumWidth - 3 // line number + " │ "

	maxLineWidth := e.editMaxLineWidth()
	vlines := e.buildVisualLines(maxLineWidth)

	if vrow >= len(vlines) {
		vrow = len(vlines) - 1
	}
	if vrow < 0 {
		vrow = 0
	}

	vl := vlines[vrow]

	if col < 0 {
		col = 0
	}
	if col > len(vl.text) {
		col = len(vl.text)
	}

	return vl.logicalRow, vl.colOffset + col
}

// handleMousePress handles left-click: places cursor and begins a potential drag.
func (e *Editor) handleMousePress(msg tea.MouseMsg) (*Editor, tea.Cmd) {
	if e.mode == EditorModeEdit {
		row, col := e.mouseToLogicalClamped(msg.X, msg.Y)
		if row < 0 {
			return e, nil
		}

		// Place cursor and start a potential drag
		e.cursorRow = row
		e.cursorCol = col
		e.selAnchorRow = row
		e.selAnchorCol = col
		e.selEndRow = row
		e.selEndCol = col
		e.selecting = true
		e.hasSelection = false // no visible selection until the mouse moves
		// NOTE: no ensureCursorVisible — mouseToLogicalClamped guarantees the
		// click maps to a visible row, so scrolling is unnecessary and would
		// shift the viewport out from under the mouse.
	} else {
		// View mode: enter edit mode with approximate cursor position
		relY := msg.Y - e.screenY - 2
		if relY < 0 {
			relY = 0
		}
		renderedLine := relY + e.viewOffset
		e.mode = EditorModeEdit
		e.parseLines()

		lineCount := e.lineCount
		if lineCount < 1 {
			lineCount = 1
		}
		rawLine := renderedLine * len(e.lines) / lineCount

		if rawLine < 0 {
			rawLine = 0
		}
		if rawLine >= len(e.lines) {
			rawLine = len(e.lines) - 1
		}

		e.cursorRow = rawLine
		e.cursorCol = 0
		e.clearSelection()
		e.ensureCursorVisible()
	}

	return e, nil
}

// handleMouseMotion updates the selection end point as the user drags.
func (e *Editor) handleMouseMotion(msg tea.MouseMsg) (*Editor, tea.Cmd) {
	if e.mode != EditorModeEdit {
		return e, nil
	}

	row, col := e.mouseToLogicalClamped(msg.X, msg.Y)
	if row < 0 {
		return e, nil
	}

	e.selEndRow = row
	e.selEndCol = col
	e.cursorRow = row
	e.cursorCol = col

	// Mark selection as visible if anchor and end differ
	if e.selEndRow != e.selAnchorRow || e.selEndCol != e.selAnchorCol {
		e.hasSelection = true
	} else {
		e.hasSelection = false
	}

	// NOTE: Do NOT call ensureCursorVisible() here. During a drag, the mouse
	// is pointing at a visible screen position so the cursor is already visible.
	// Calling ensureCursorVisible creates a feedback loop: the viewport scrolls,
	// which changes what row the same screen Y maps to, which triggers more
	// scrolling — causing visual jumping and glitching.
	return e, nil
}

// handleMouseRelease finalizes the selection when the user releases the mouse.
func (e *Editor) handleMouseRelease(msg tea.MouseMsg) (*Editor, tea.Cmd) {
	if e.mode != EditorModeEdit {
		e.selecting = false
		return e, nil
	}

	row, col := e.mouseToLogical(msg.X, msg.Y)
	if row >= 0 {
		e.selEndRow = row
		e.selEndCol = col
		e.cursorRow = row
		e.cursorCol = col
	}

	e.selecting = false

	// If anchor == end, it was just a click, not a drag
	if e.selEndRow == e.selAnchorRow && e.selEndCol == e.selAnchorCol {
		e.hasSelection = false
	} else {
		e.hasSelection = true
	}

	e.ensureCursorVisible()
	return e, nil
}
