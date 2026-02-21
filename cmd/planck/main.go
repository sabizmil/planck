package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/planck/internal/app"
	"github.com/anthropics/planck/internal/config"
	"github.com/anthropics/planck/internal/session"
	"github.com/anthropics/planck/internal/workspace"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Define flags
	folderFlag := flag.String("folder", "", "Folder containing markdown files")
	versionFlag := flag.Bool("version", false, "Show version information")
	helpFlag := flag.Bool("help", false, "Show help")

	// Short versions
	flag.BoolVar(versionFlag, "v", false, "Show version information")
	flag.BoolVar(helpFlag, "h", false, "Show help")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("planck %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	// Load recent folders
	configDir := getConfigDir()
	recent, err := workspace.LoadRecentFolders(configDir)
	if err != nil {
		recent = &workspace.RecentFolders{}
	}

	// Determine folder to use
	var folder string
	if *folderFlag != "" {
		// Use provided folder
		folder = *folderFlag
		absPath, err := filepath.Abs(folder)
		if err == nil {
			folder = absPath
		}
	} else if len(flag.Args()) > 0 {
		// Use positional argument
		folder = flag.Args()[0]
		absPath, err := filepath.Abs(folder)
		if err == nil {
			folder = absPath
		}
	}

	// If no folder specified, show folder picker
	if folder == "" {
		folder, err = runFolderPicker(recent.Folders)
		if err != nil {
			if err.Error() == "quit" {
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Validate folder
	info, err := os.Stat(folder)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: folder does not exist: %s\n", folder)
		} else {
			fmt.Fprintf(os.Stderr, "Error: cannot access folder: %v\n", err)
		}
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: not a directory: %s\n", folder)
		os.Exit(1)
	}

	// Save to recent folders
	recent.Add(folder)
	if err := recent.Save(configDir); err != nil {
		// Non-fatal, continue
	}

	// Load or create configuration
	cfg, err := config.Load(folder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create session backend
	backend, err := session.NewBackend(session.BackendConfig{
		Backend:     cfg.Session.Backend,
		Prefix:      cfg.Preferences.TmuxPrefix,
		SessionsDir: cfg.SessionsDir(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session backend: %v\n", err)
		os.Exit(1)
	}

	// Initialize and run the app
	application, err := app.New(cfg, configDir, folder, backend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing app: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		application,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running app: %v\n", err)
		os.Exit(1)
	}
}

func getConfigDir() string {
	// Use ~/.config/planck
	home, err := os.UserHomeDir()
	if err != nil {
		return ".planck"
	}
	return filepath.Join(home, ".config", "planck")
}

func runFolderPicker(recentFolders []string) (string, error) {
	picker := app.NewFolderPickerModel(recentFolders)
	p := tea.NewProgram(picker, tea.WithAltScreen())

	model, err := p.Run()
	if err != nil {
		return "", err
	}

	result := model.(*app.FolderPickerModel)
	if result.Quit {
		return "", fmt.Errorf("quit")
	}

	return result.SelectedFolder, nil
}

func printHelp() {
	fmt.Println(`planck - Folder-based markdown editor with embedded Claude agent

Usage:
  planck [options] [folder]

Options:
  -f, --folder PATH  Folder containing markdown files
  -h, --help         Show this help message
  -v, --version      Show version information

Keybindings (Planning Tab):
  ↑/↓, j/k    Navigate files
  Enter       Select file
  e           Edit file
  n           New file
  d           Delete file
  s           Send to Claude agent
  Tab/1/2     Switch tabs

Keybindings (Agent Tab):
  i           Enter input mode
  Ctrl+\      Exit input mode
  Tab/1/2     Switch tabs

Keybindings (Editor):
  ↑/↓/←/→     Move cursor
  Ctrl+S      Save file
  Esc         Exit edit mode

For more information, visit: https://github.com/anthropics/planck`)
}
