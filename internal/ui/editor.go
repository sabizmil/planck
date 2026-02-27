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

	// Selection state (driven by Shift+Arrow keys and Shift+Click)
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
		if msg.Alt {
			// Alt+Left: jump to previous word boundary
			e.clearSelection()
			e.cursorRow, e.cursorCol = e.wordBoundaryLeft(e.cursorRow, e.cursorCol)
		} else {
			e.clearSelection()
			e.moveCursorLeft()
		}

	case tea.KeyRight:
		if msg.Alt {
			// Alt+Right: jump to next word boundary
			e.clearSelection()
			e.cursorRow, e.cursorCol = e.wordBoundaryRight(e.cursorRow, e.cursorCol)
		} else {
			e.clearSelection()
			e.moveCursorRight()
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
		e.moveCursorToLineStart()

	case tea.KeyEnd, tea.KeyCtrlE:
		e.clearSelection()
		e.moveCursorToLineEnd()

	// --- Shift+Arrow: extend selection ---
	case tea.KeyShiftLeft:
		if msg.Alt {
			// Alt+Shift+Left: select to previous word boundary
			fromRow, fromCol := e.cursorRow, e.cursorCol
			e.cursorRow, e.cursorCol = e.wordBoundaryLeft(e.cursorRow, e.cursorCol)
			e.extendSelection(fromRow, fromCol)
		} else {
			fromRow, fromCol := e.cursorRow, e.cursorCol
			e.moveCursorLeft()
			e.extendSelection(fromRow, fromCol)
		}

	case tea.KeyShiftRight:
		if msg.Alt {
			// Alt+Shift+Right: select to next word boundary
			fromRow, fromCol := e.cursorRow, e.cursorCol
			e.cursorRow, e.cursorCol = e.wordBoundaryRight(e.cursorRow, e.cursorCol)
			e.extendSelection(fromRow, fromCol)
		} else {
			fromRow, fromCol := e.cursorRow, e.cursorCol
			e.moveCursorRight()
			e.extendSelection(fromRow, fromCol)
		}

	case tea.KeyShiftUp:
		fromRow, fromCol := e.cursorRow, e.cursorCol
		e.moveCursorUpVisual()
		e.extendSelection(fromRow, fromCol)
		e.ensureCursorVisible()

	case tea.KeyShiftDown:
		fromRow, fromCol := e.cursorRow, e.cursorCol
		e.moveCursorDownVisual()
		e.extendSelection(fromRow, fromCol)
		e.ensureCursorVisible()

	case tea.KeyShiftHome:
		fromRow, fromCol := e.cursorRow, e.cursorCol
		e.moveCursorToLineStart()
		e.extendSelection(fromRow, fromCol)

	case tea.KeyShiftEnd:
		fromRow, fromCol := e.cursorRow, e.cursorCol
		e.moveCursorToLineEnd()
		e.extendSelection(fromRow, fromCol)

	// --- Ctrl+Arrow: word jump (Linux/Windows convention) ---
	case tea.KeyCtrlLeft:
		if msg.Alt {
			// Ctrl+Alt+Left: same as Alt+Left (some terminals send this)
			e.clearSelection()
			e.cursorRow, e.cursorCol = e.wordBoundaryLeft(e.cursorRow, e.cursorCol)
		} else {
			e.clearSelection()
			e.cursorRow, e.cursorCol = e.wordBoundaryLeft(e.cursorRow, e.cursorCol)
		}

	case tea.KeyCtrlRight:
		if msg.Alt {
			e.clearSelection()
			e.cursorRow, e.cursorCol = e.wordBoundaryRight(e.cursorRow, e.cursorCol)
		} else {
			e.clearSelection()
			e.cursorRow, e.cursorCol = e.wordBoundaryRight(e.cursorRow, e.cursorCol)
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

// moveCursorLeft moves the cursor one character to the left, wrapping to the
// end of the previous line if at column 0. Does NOT clear selection.
func (e *Editor) moveCursorLeft() {
	if e.cursorCol > 0 {
		e.cursorCol--
	} else if e.cursorRow > 0 {
		e.cursorRow--
		e.cursorCol = len(e.lines[e.cursorRow])
	}
}

// moveCursorRight moves the cursor one character to the right, wrapping to
// the start of the next line if at end of line. Does NOT clear selection.
func (e *Editor) moveCursorRight() {
	if e.cursorRow < len(e.lines) {
		lineLen := len(e.lines[e.cursorRow])
		if e.cursorCol < lineLen {
			e.cursorCol++
		} else if e.cursorRow < len(e.lines)-1 {
			e.cursorRow++
			e.cursorCol = 0
		}
	}
}

// moveCursorToLineStart moves the cursor to column 0. Does NOT clear selection.
func (e *Editor) moveCursorToLineStart() {
	e.cursorCol = 0
}

// moveCursorToLineEnd moves the cursor to the end of the current line. Does NOT clear selection.
func (e *Editor) moveCursorToLineEnd() {
	if e.cursorRow < len(e.lines) {
		e.cursorCol = len(e.lines[e.cursorRow])
	}
}

// extendSelection extends (or starts) a selection to the current cursor position.
// If no selection exists, the anchor is set to the given (fromRow, fromCol) —
// typically the cursor position BEFORE the move that triggered the extension.
// The selection endpoint is always updated to the current cursor position.
func (e *Editor) extendSelection(fromRow, fromCol int) {
	if !e.hasSelection {
		e.selAnchorRow = fromRow
		e.selAnchorCol = fromCol
	}
	e.selEndRow = e.cursorRow
	e.selEndCol = e.cursorCol
	if e.selEndRow != e.selAnchorRow || e.selEndCol != e.selAnchorCol {
		e.hasSelection = true
	} else {
		e.hasSelection = false
	}
}

// isWordChar returns true for characters considered part of a "word" (letters,
// digits, underscore). Everything else is a boundary.
func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_'
}

// wordBoundaryLeft returns the (row, col) position at the start of the previous
// word from the given position. Skips whitespace/punctuation backward, then skips
// word characters backward. Wraps across line boundaries.
func (e *Editor) wordBoundaryLeft(row, col int) (int, int) {
	if len(e.lines) == 0 {
		return 0, 0
	}

	// If at start of line, wrap to end of previous line
	if col <= 0 {
		if row <= 0 {
			return 0, 0
		}
		row--
		col = len(e.lines[row])
	}

	line := e.lines[row]

	// Skip non-word characters backward (whitespace, punctuation)
	for col > 0 && !isWordChar(line[col-1]) {
		col--
	}

	// If we consumed everything and we're at start of line, try wrapping to previous line
	if col == 0 {
		if row <= 0 {
			return 0, 0
		}
		row--
		col = len(e.lines[row])
		line = e.lines[row]
		// Skip trailing non-word chars on previous line
		for col > 0 && !isWordChar(line[col-1]) {
			col--
		}
	}

	// Skip word characters backward
	for col > 0 && isWordChar(line[col-1]) {
		col--
	}

	return row, col
}

// wordBoundaryRight returns the (row, col) position at the start of the next
// word from the given position. Skips word characters forward, then skips
// whitespace/punctuation forward. Wraps across line boundaries.
func (e *Editor) wordBoundaryRight(row, col int) (int, int) {
	if len(e.lines) == 0 {
		return 0, 0
	}

	line := e.lines[row]

	// If at end of line, wrap to start of next line
	if col >= len(line) {
		if row >= len(e.lines)-1 {
			return row, col
		}
		row++
		col = 0
		line = e.lines[row]
	}

	// Skip word characters forward
	for col < len(line) && isWordChar(line[col]) {
		col++
	}

	// Skip non-word characters forward (whitespace, punctuation)
	for col < len(line) && !isWordChar(line[col]) {
		col++
	}

	// If we reached end of line, that's a valid stop position
	return row, col
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

	return e.theme.DetailPanel.Width(e.width).Height(e.height).MaxHeight(e.height).Render(b.String())
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
			// Selection-aware rendering: batch consecutive same-styled characters
			// into single Render() calls to avoid per-character ANSI escape sequences
			// that can confuse lipgloss's word wrapper (cellbuf.Wrap).
			const (
				runNormal    = 0
				runSelection = 1
				runCursor    = 2
			)
			type styledRun struct {
				text    strings.Builder
				runType int
			}
			var runs []styledRun
			curRunType := -1

			for j := 0; j <= len(vl.text); j++ {
				logCol := vl.colOffset + j
				var ch string
				var rt int

				switch {
				case hasCursor && j == localCursorCol:
					if j >= len(vl.text) {
						ch = " "
					} else {
						ch = string(vl.text[j])
					}
					rt = runCursor
				case j < len(vl.text):
					ch = string(vl.text[j])
					if e.isInSelection(vl.logicalRow, logCol) {
						rt = runSelection
					} else {
						rt = runNormal
					}
				default:
					continue // j == len(vl.text) and no cursor
				}

				if rt != curRunType {
					runs = append(runs, styledRun{runType: rt})
					curRunType = rt
				}
				runs[len(runs)-1].text.WriteString(ch)
			}

			for _, run := range runs {
				switch run.runType {
				case runNormal:
					b.WriteString(e.theme.Normal.Render(run.text.String()))
				case runSelection:
					b.WriteString(selectionStyle.Render(run.text.String()))
				case runCursor:
					b.WriteString(cursorStyle.Render(run.text.String()))
				}
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

// handleMouse processes mouse events for scrolling and click-to-place-cursor.
// Mouse drag selection has been removed — use Shift+Arrow keys or Shift+Click instead.
func (e *Editor) handleMouse(msg tea.MouseMsg) (*Editor, tea.Cmd) {
	me := tea.MouseEvent(msg)

	// Wheel scroll (both modes)
	if me.IsWheel() {
		if msg.Button == tea.MouseButtonWheelUp {
			e.ScrollBy(-3)
		} else if msg.Button == tea.MouseButtonWheelDown {
			e.ScrollBy(3)
		}
		return e, nil
	}

	if me.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		return e.handleMousePress(msg)
	}

	return e, nil
}

// handleMousePress handles left-click: places cursor or extends selection with Shift+Click.
func (e *Editor) handleMousePress(msg tea.MouseMsg) (*Editor, tea.Cmd) {
	me := tea.MouseEvent(msg)

	if e.mode == EditorModeEdit {
		row, col := e.mouseToLogical(msg.X, msg.Y)
		if row < 0 {
			return e, nil
		}

		if me.Shift {
			// Shift+Click: extend selection from current cursor to clicked position
			fromRow, fromCol := e.cursorRow, e.cursorCol
			e.cursorRow = row
			e.cursorCol = col
			e.extendSelection(fromRow, fromCol)
		} else {
			// Plain click: place cursor, clear any selection
			e.cursorRow = row
			e.cursorCol = col
			e.clearSelection()
		}
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
