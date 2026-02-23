package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FolderPickerCanceledMsg is sent when the folder picker is canceled.
type FolderPickerCanceledMsg struct{}

// FolderPicker displays a folder selection dialog
type FolderPicker struct {
	theme         *Theme
	recentFolders []string
	cursor        int
	inputMode     bool
	inputBuffer   string
	inputCursor   int
	width         int
	height        int
	errorMsg      string
	overlay       bool
}

// NewFolderPicker creates a new folder picker
func NewFolderPicker(theme *Theme, recentFolders []string) *FolderPicker {
	return &FolderPicker{
		theme:         theme,
		recentFolders: recentFolders,
	}
}

// Init initializes the folder picker
func (f *FolderPicker) Init() tea.Cmd {
	return nil
}

// FolderSelectedMsg is sent when a folder is selected
type FolderSelectedMsg struct {
	Folder string
}

// Update handles messages
func (f *FolderPicker) Update(msg tea.Msg) (*FolderPicker, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if f.inputMode {
			return f.updateInputMode(msg)
		}
		return f.updateListMode(msg)
	}
	return f, nil
}

func (f *FolderPicker) updateListMode(msg tea.KeyMsg) (*FolderPicker, tea.Cmd) {
	switch msg.String() {
	case "down", "j":
		f.cursor++
		maxCursor := len(f.recentFolders) // Last item is "Browse..."
		if f.cursor > maxCursor {
			f.cursor = maxCursor
		}

	case "up", "k":
		f.cursor--
		if f.cursor < 0 {
			f.cursor = 0
		}

	case "enter":
		// Check if Browse is selected
		if f.cursor == len(f.recentFolders) {
			f.inputMode = true
			f.inputBuffer = ""
			f.inputCursor = 0
			f.errorMsg = ""
			return f, nil
		}

		// Select recent folder
		if f.cursor < len(f.recentFolders) {
			folder := f.recentFolders[f.cursor]
			if err := f.validateFolder(folder); err != nil {
				f.errorMsg = err.Error()
				return f, nil
			}
			return f, func() tea.Msg {
				return FolderSelectedMsg{Folder: folder}
			}
		}

	case "b", "B":
		// Shortcut to browse
		f.inputMode = true
		f.inputBuffer = ""
		f.inputCursor = 0
		f.errorMsg = ""

	case "q", "esc":
		if f.overlay {
			return f, func() tea.Msg { return FolderPickerCanceledMsg{} }
		}
		return f, tea.Quit
	}

	return f, nil
}

func (f *FolderPicker) updateInputMode(msg tea.KeyMsg) (*FolderPicker, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		f.inputMode = false
		f.errorMsg = ""
		return f, nil

	case tea.KeyEnter:
		// Validate and select folder
		folder := f.inputBuffer
		if folder == "" {
			f.errorMsg = "Please enter a folder path"
			return f, nil
		}

		// Expand ~ to home directory
		if strings.HasPrefix(folder, "~") {
			home, err := os.UserHomeDir()
			if err == nil {
				folder = filepath.Join(home, folder[1:])
			}
		}

		// Make absolute
		absPath, err := filepath.Abs(folder)
		if err != nil {
			f.errorMsg = fmt.Sprintf("Invalid path: %v", err)
			return f, nil
		}

		if err := f.validateFolder(absPath); err != nil {
			f.errorMsg = err.Error()
			return f, nil
		}

		return f, func() tea.Msg {
			return FolderSelectedMsg{Folder: absPath}
		}

	case tea.KeyBackspace:
		if f.inputBuffer != "" && f.inputCursor > 0 {
			f.inputBuffer = f.inputBuffer[:f.inputCursor-1] + f.inputBuffer[f.inputCursor:]
			f.inputCursor--
		}
		f.errorMsg = ""

	case tea.KeyDelete:
		if f.inputCursor < len(f.inputBuffer) {
			f.inputBuffer = f.inputBuffer[:f.inputCursor] + f.inputBuffer[f.inputCursor+1:]
		}

	case tea.KeyLeft:
		if f.inputCursor > 0 {
			f.inputCursor--
		}

	case tea.KeyRight:
		if f.inputCursor < len(f.inputBuffer) {
			f.inputCursor++
		}

	case tea.KeyHome, tea.KeyCtrlA:
		f.inputCursor = 0

	case tea.KeyEnd, tea.KeyCtrlE:
		f.inputCursor = len(f.inputBuffer)

	case tea.KeyRunes:
		// Insert characters at cursor
		runes := string(msg.Runes)
		f.inputBuffer = f.inputBuffer[:f.inputCursor] + runes + f.inputBuffer[f.inputCursor:]
		f.inputCursor += len(runes)
		f.errorMsg = ""

	case tea.KeySpace:
		f.inputBuffer = f.inputBuffer[:f.inputCursor] + " " + f.inputBuffer[f.inputCursor:]
		f.inputCursor++
	}

	return f, nil
}

func (f *FolderPicker) validateFolder(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("folder does not exist")
		}
		return fmt.Errorf("cannot access folder: %v", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}

	return nil
}

// View renders the folder picker
func (f *FolderPicker) View() string {
	var b strings.Builder

	// Title
	title := f.theme.Title.Render("Select Folder")
	b.WriteString(title)
	b.WriteString("\n\n")

	if f.inputMode {
		// Input mode
		b.WriteString(f.theme.Normal.Render("Enter folder path:"))
		b.WriteString("\n\n")

		// Input field with cursor
		inputStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(f.theme.Accent).
			Padding(0, 1).
			Width(f.width - 8)

		// Show cursor
		displayText := f.inputBuffer
		if f.inputCursor <= len(displayText) {
			before := displayText[:f.inputCursor]
			after := displayText[f.inputCursor:]
			cursor := lipgloss.NewStyle().Reverse(true).Render("_")
			displayText = before + cursor + after
		}
		if displayText == "" {
			displayText = lipgloss.NewStyle().Reverse(true).Render("_")
		}

		b.WriteString(inputStyle.Render(displayText))
		b.WriteString("\n\n")

		// Error message
		if f.errorMsg != "" {
			b.WriteString(f.theme.StatusFailed.Render("Error: " + f.errorMsg))
			b.WriteString("\n\n")
		}

		// Help
		b.WriteString(f.theme.KeyHint.Render("[Enter] confirm  [Esc] back"))
	} else {
		// List mode
		if len(f.recentFolders) > 0 {
			b.WriteString(f.theme.Subtitle.Render("Recent folders:"))
			b.WriteString("\n\n")

			for i, folder := range f.recentFolders {
				isSelected := i == f.cursor

				prefix := "  "
				if isSelected {
					prefix = f.theme.Selected.Render(IndicatorSelected + " ")
				}

				// Truncate long paths
				displayPath := folder
				maxWidth := f.width - 8
				if len(displayPath) > maxWidth {
					displayPath = "..." + displayPath[len(displayPath)-maxWidth+3:]
				}

				var line string
				if isSelected {
					line = f.theme.SidebarSelected.Render(prefix + displayPath)
				} else {
					line = f.theme.SidebarItem.Render(prefix + displayPath)
				}
				b.WriteString(line)
				b.WriteString("\n")
			}

			b.WriteString("\n")
		}

		// Browse option
		browseSelected := f.cursor == len(f.recentFolders)
		prefix := "  "
		if browseSelected {
			prefix = f.theme.Selected.Render(IndicatorSelected + " ")
		}

		browseText := "Browse..."
		if browseSelected {
			b.WriteString(f.theme.SidebarSelected.Render(prefix + browseText))
		} else {
			b.WriteString(f.theme.SidebarItem.Render(prefix + browseText))
		}
		b.WriteString("\n\n")

		// Error message
		if f.errorMsg != "" {
			b.WriteString(f.theme.StatusFailed.Render("Error: " + f.errorMsg))
			b.WriteString("\n\n")
		}

		// Help
		b.WriteString(f.theme.KeyHint.Render("[Enter] select  [b] browse  [q] quit"))
	}

	// Center the dialog
	dialogStyle := f.theme.Dialog.
		Width(f.width).
		Padding(2, 4)

	return lipgloss.Place(
		f.width,
		f.height,
		lipgloss.Center,
		lipgloss.Center,
		dialogStyle.Render(b.String()),
	)
}

// SetSize sets the picker dimensions
func (f *FolderPicker) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// SetRecentFolders updates the recent folders list
func (f *FolderPicker) SetRecentFolders(folders []string) {
	f.recentFolders = folders
}

// SetOverlayMode sets whether the picker is used as an in-app overlay
func (f *FolderPicker) SetOverlayMode(overlay bool) {
	f.overlay = overlay
}
