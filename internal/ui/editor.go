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
	fileName    string
	content     string
	lines       []string
	rendered    string
	lineCount   int

	// Mode
	mode EditorMode

	// View mode state
	viewOffset int

	// Edit mode state
	cursorRow int
	cursorCol int
	modified  bool

	// Dimensions
	width  int
	height int
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
		if e.cursorCol > 0 {
			e.cursorCol--
		} else if e.cursorRow > 0 {
			// Move to end of previous line
			e.cursorRow--
			e.cursorCol = len(e.lines[e.cursorRow])
		}

	case tea.KeyRight:
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
		if e.cursorRow > 0 {
			e.cursorRow--
			// Adjust column if line is shorter
			if e.cursorCol > len(e.lines[e.cursorRow]) {
				e.cursorCol = len(e.lines[e.cursorRow])
			}
		}
		e.ensureCursorVisible()

	case tea.KeyDown:
		if e.cursorRow < len(e.lines)-1 {
			e.cursorRow++
			// Adjust column if line is shorter
			if e.cursorCol > len(e.lines[e.cursorRow]) {
				e.cursorCol = len(e.lines[e.cursorRow])
			}
		}
		e.ensureCursorVisible()

	case tea.KeyHome, tea.KeyCtrlA:
		e.cursorCol = 0

	case tea.KeyEnd, tea.KeyCtrlE:
		if e.cursorRow < len(e.lines) {
			e.cursorCol = len(e.lines[e.cursorRow])
		}

	case tea.KeyEnter:
		e.insertNewline()
		e.modified = true

	case tea.KeyBackspace:
		e.deleteBackward()
		e.modified = true

	case tea.KeyDelete:
		e.deleteForward()
		e.modified = true

	case tea.KeyTab:
		// Insert spaces for tab
		e.insertText("    ")
		e.modified = true

	case tea.KeyRunes:
		e.insertText(string(msg.Runes))
		e.modified = true

	case tea.KeySpace:
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

func (e *Editor) ensureCursorVisible() {
	visibleLines := e.visibleLines()

	if e.cursorRow < e.viewOffset {
		e.viewOffset = e.cursorRow
	}
	if e.cursorRow >= e.viewOffset+visibleLines {
		e.viewOffset = e.cursorRow - visibleLines + 1
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

func (e *Editor) renderEditMode(visibleLines int) string {
	var b strings.Builder

	if len(e.lines) == 0 {
		e.lines = []string{""}
	}

	start := e.viewOffset
	end := e.viewOffset + visibleLines

	if start >= len(e.lines) {
		start = len(e.lines) - 1
	}
	if start < 0 {
		start = 0
	}
	if end > len(e.lines) {
		end = len(e.lines)
	}

	// Line number width
	lineNumWidth := len(fmt.Sprintf("%d", len(e.lines)))
	if lineNumWidth < 2 {
		lineNumWidth = 2
	}

	for i := start; i < end; i++ {
		line := e.lines[i]

		// Line number
		lineNum := fmt.Sprintf("%*d", lineNumWidth, i+1)
		b.WriteString(e.theme.Dimmed.Render(lineNum + " │ "))

		// Line content with cursor
		maxLineWidth := e.width - lineNumWidth - 8
		if maxLineWidth < 10 {
			maxLineWidth = 10
		}

		if i == e.cursorRow {
			// Show cursor on this line
			displayLine := e.renderLineWithCursor(line, maxLineWidth)
			b.WriteString(displayLine)
		} else {
			// Regular line
			if len(line) > maxLineWidth {
				line = line[:maxLineWidth-3] + "..."
			}
			b.WriteString(e.theme.Normal.Render(line))
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

func (e *Editor) renderLineWithCursor(line string, maxWidth int) string {
	cursorStyle := lipgloss.NewStyle().Reverse(true)

	col := e.cursorCol
	if col > len(line) {
		col = len(line)
	}

	// Handle cursor at end of line
	if col >= len(line) {
		displayLine := line
		if len(displayLine) > maxWidth-1 {
			displayLine = displayLine[:maxWidth-4] + "..."
		}
		return e.theme.Normal.Render(displayLine) + cursorStyle.Render(" ")
	}

	// Cursor in middle of line
	before := line[:col]
	cursor := string(line[col])
	after := line[col+1:]

	// Truncate if needed
	if len(line) > maxWidth {
		// Keep cursor visible
		if col < maxWidth-3 {
			afterLen := maxWidth - col - 4
			if afterLen < 0 {
				afterLen = 0
			}
			if afterLen < len(after) {
				after = after[:afterLen] + "..."
			}
		} else {
			// Scroll line to show cursor
			start := col - maxWidth/2
			if start < 0 {
				start = 0
			}
			before = "..." + line[start:col]
			if col+maxWidth/2 < len(line) {
				after = line[col+1:col+maxWidth/2] + "..."
			} else {
				after = line[col+1:]
			}
		}
	}

	return e.theme.Normal.Render(before) + cursorStyle.Render(cursor) + e.theme.Normal.Render(after)
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
	e.parseLines()
}

// ExitEditMode exits edit mode
func (e *Editor) ExitEditMode() {
	e.mode = EditorModeView
	e.rebuildContent()
}

// ClearModified clears the modified flag
func (e *Editor) ClearModified() {
	e.modified = false
}

// ScrollBy scrolls the view by the given number of lines. Works in both view and edit modes.
func (e *Editor) ScrollBy(delta int) {
	e.viewOffset += delta

	var maxOffset int
	if e.mode == EditorModeEdit {
		maxOffset = len(e.lines) - e.visibleLines()
	} else {
		maxOffset = e.lineCount - e.visibleLines()
	}
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

// SetPosition sets the screen position of the editor panel (for mouse coordinate translation).
func (e *Editor) SetPosition(x, y int) {
	e.screenX = x
	e.screenY = y
}

// handleMouse processes mouse events for scrolling and click-to-place-cursor.
func (e *Editor) handleMouse(msg tea.MouseMsg) (*Editor, tea.Cmd) {
	me := tea.MouseEvent(msg)

	// Only handle press events (ignore release/motion)
	if me.Action != tea.MouseActionPress {
		return e, nil
	}

	// Wheel scroll (both modes)
	if me.IsWheel() {
		if msg.Button == tea.MouseButtonWheelUp {
			e.ScrollBy(-3)
		} else if msg.Button == tea.MouseButtonWheelDown {
			e.ScrollBy(3)
		}
		return e, nil
	}

	// Left click only
	if msg.Button != tea.MouseButtonLeft {
		return e, nil
	}

	// Translate screen coordinates to editor-relative coordinates
	// screenY+2 accounts for header row and separator row
	relY := msg.Y - e.screenY - 2
	// screenX+2 accounts for panel padding left
	relX := msg.X - e.screenX - 2

	if relY < 0 || relX < 0 {
		return e, nil
	}

	if e.mode == EditorModeEdit {
		// Edit mode: precise cursor placement
		row := relY + e.viewOffset

		// Calculate line number gutter width
		lineNumWidth := len(fmt.Sprintf("%d", len(e.lines)))
		if lineNumWidth < 2 {
			lineNumWidth = 2
		}
		col := relX - lineNumWidth - 3 // line number + " │ "

		// Clamp row
		if row < 0 {
			row = 0
		}
		if row >= len(e.lines) {
			row = len(e.lines) - 1
		}

		// Clamp col
		if col < 0 {
			col = 0
		}
		if row >= 0 && row < len(e.lines) && col > len(e.lines[row]) {
			col = len(e.lines[row])
		}

		e.cursorRow = row
		e.cursorCol = col
		e.ensureCursorVisible()
	} else {
		// View mode: enter edit mode with approximate cursor position
		renderedLine := relY + e.viewOffset
		e.mode = EditorModeEdit
		e.parseLines()

		// Map rendered line to raw line using ratio
		lineCount := e.lineCount
		if lineCount < 1 {
			lineCount = 1
		}
		rawLine := renderedLine * len(e.lines) / lineCount

		// Clamp
		if rawLine < 0 {
			rawLine = 0
		}
		if rawLine >= len(e.lines) {
			rawLine = len(e.lines) - 1
		}

		e.cursorRow = rawLine
		e.cursorCol = 0
		e.ensureCursorVisible()
	}

	return e, nil
}
