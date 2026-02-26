package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sabizmil/planck/internal/app"
	"github.com/sabizmil/planck/internal/config"
	"github.com/sabizmil/planck/internal/session"
	"github.com/sabizmil/planck/internal/tmux"
	"github.com/sabizmil/planck/internal/updater"
)

var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

// knownSubcommands lists all valid subcommand names.
var knownSubcommands = map[string]bool{
	"update":  true,
	"version": true,
	"attach":  true,
}

// findSubcommand scans os.Args for a subcommand, skipping flags and their values.
// Returns the subcommand name, the remaining args after it, and whether one was found.
func findSubcommand(args []string) (cmd string, rest []string, found bool) {
	// Flags that take a value argument (must skip the next arg too)
	valuedFlags := map[string]bool{
		"-folder": true, "--folder": true,
		"-f": true,
	}

	for i := 1; i < len(args); i++ {
		arg := args[i]

		// Skip flags
		if strings.HasPrefix(arg, "-") {
			// If this flag takes a value, skip the next arg too
			if valuedFlags[arg] {
				i++ // skip the value
			} else if strings.Contains(arg, "=") {
				// --flag=value form, already consumed
				continue
			}
			continue
		}

		// First non-flag argument — check if it's a subcommand
		if knownSubcommands[arg] {
			// Only pass args that appear after the subcommand
			rest = args[i+1:]
			return arg, rest, true
		}

		// Non-flag, non-subcommand argument — stop scanning
		break
	}
	return "", nil, false
}

func main() {
	// Handle subcommands — scan all args, not just os.Args[1]
	if cmd, rest, found := findSubcommand(os.Args); found {
		switch cmd {
		case "update":
			runUpdate(rest)
		case "version":
			runVersion(rest)
		case "attach":
			runAttach(rest)
		}
		return
	}

	// Define flags
	folderFlag := flag.String("folder", "", "Folder containing markdown files")
	versionFlag := flag.Bool("version", false, "Show version information")
	helpFlag := flag.Bool("help", false, "Show help")

	// Short versions
	flag.StringVar(folderFlag, "f", "", "Folder containing markdown files")
	flag.BoolVar(versionFlag, "v", false, "Show version information")
	flag.BoolVar(helpFlag, "h", false, "Show help")

	flag.Parse()

	// Reject unexpected positional arguments
	if flag.NArg() > 0 {
		arg := flag.Arg(0)
		if knownSubcommands[arg] {
			fmt.Fprintf(os.Stderr, "Error: unknown position for command %q. Usage: planck %s [options]\n", arg, arg)
		} else {
			fmt.Fprintf(os.Stderr, "Error: unexpected argument %q\n", arg)
			fmt.Fprintln(os.Stderr, "Run 'planck --help' for usage information.")
		}
		os.Exit(1)
	}

	if *versionFlag {
		fmt.Printf("planck %s (commit: %s, built: %s)\n", version, commit, buildTime)
		os.Exit(0)
	}

	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	configDir := getConfigDir()

	// Determine folder to use: -folder flag, or current working directory
	var folder string
	if *folderFlag != "" {
		folder = *folderFlag
		absPath, err := filepath.Abs(folder)
		if err == nil {
			folder = absPath
		}
	} else {
		var err error
		folder, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot determine current directory: %v\n", err)
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

	// Load or create configuration
	cfg, err := config.Load(folder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create session backend based on configuration
	backend, err := session.NewBackend(session.BackendConfig{
		Backend:     cfg.Session.Backend,
		Prefix:      cfg.Preferences.TmuxPrefix,
		SessionsDir: cfg.SessionsDir(),
		WorkDir:     folder,
		ExtraArgs:   nil, // per-agent args passed at launch
		TmuxFactory: func(prefix, sessionsDir, workDir string, extraArgs []string) session.InteractiveBackend {
			return tmux.NewTmuxBackend(prefix, sessionsDir, workDir, extraArgs)
		},
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

func runUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	checkOnly := fs.Bool("check", false, "Only check for updates, don't install")
	fs.Parse(args) //nolint:errcheck // flag.ExitOnError handles parse errors

	fmt.Printf("Current version: %s\n", version)

	if version == "dev" {
		fmt.Println("Development build — cannot check for updates.")
		fmt.Println("Install a release build to use self-update.")
		os.Exit(0)
	}

	fmt.Println("Checking for updates...")
	result, err := updater.Check(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if !result.UpdateAvail {
		fmt.Printf("Already up to date (v%s).\n", result.CurrentVersion)
		return
	}

	fmt.Printf("Update available: v%s -> v%s\n", result.CurrentVersion, result.LatestVersion)

	if *checkOnly {
		fmt.Println("\nRun 'planck update' to install the update.")
		return
	}

	fmt.Println("Downloading and installing...")
	if err := updater.Update(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated to v%s!\n", result.LatestVersion)
}

func runVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	checkUpdate := fs.Bool("check", false, "Check if a newer version is available")
	fs.Parse(args) //nolint:errcheck // flag.ExitOnError handles parse errors

	fmt.Printf("planck %s (commit: %s, built: %s)\n", version, commit, buildTime)

	if *checkUpdate && version != "dev" {
		fmt.Println()
		result, err := updater.Check(version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not check for updates: %v\n", err)
			return
		}
		if result.UpdateAvail {
			fmt.Printf("Update available: v%s -> v%s\n", result.CurrentVersion, result.LatestVersion)
			fmt.Println("Run 'planck update' to install.")
		} else {
			fmt.Println("Up to date.")
		}
	}
}

func runAttach(args []string) {
	fs := flag.NewFlagSet("attach", flag.ExitOnError)
	folderFlag := fs.String("folder", "", "Working directory (default: current directory)")
	fs.Parse(args) //nolint:errcheck // flag.ExitOnError handles parse errors

	// Check tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: tmux is required for 'planck attach' but was not found in PATH.")
		fmt.Fprintln(os.Stderr, "Install tmux: brew install tmux (macOS) or apt install tmux (Linux)")
		os.Exit(1)
	}

	// Determine the working directory
	folder := *folderFlag
	if folder == "" {
		var err error
		folder, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot determine current directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		absPath, err := filepath.Abs(folder)
		if err == nil {
			folder = absPath
		}
	}

	// Generate a tmux session name based on the folder path
	// This ensures each project gets its own persistent session
	h := sha256Short(folder)
	tmuxSession := fmt.Sprintf("planck-%s", h)

	// Check if session already exists
	checkCmd := exec.Command("tmux", "has-session", "-t", tmuxSession)
	if err := checkCmd.Run(); err == nil {
		// Session exists — attach to it
		fmt.Printf("Reattaching to existing planck session for %s\n", folder)
		attachCmd := exec.Command("tmux", "attach-session", "-t", tmuxSession)
		attachCmd.Stdin = os.Stdin
		attachCmd.Stdout = os.Stdout
		attachCmd.Stderr = os.Stderr
		if err := attachCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error attaching to tmux session: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Session doesn't exist — create it with planck running inside
	planckBin, err := os.Executable()
	if err != nil {
		planckBin = "planck"
	}
	planckCmd := fmt.Sprintf("%s --folder %s", planckBin, quoteShellArg(folder))

	fmt.Printf("Starting planck in persistent tmux session for %s\n", folder)
	fmt.Printf("Session name: %s\n", tmuxSession)
	fmt.Println("Detach with Ctrl+B d, reattach with: planck attach")

	newCmd := exec.Command("tmux", "new-session", "-s", tmuxSession, "-c", folder, planckCmd)
	newCmd.Stdin = os.Stdin
	newCmd.Stdout = os.Stdout
	newCmd.Stderr = os.Stderr
	if err := newCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating tmux session: %v\n", err)
		os.Exit(1)
	}
}

func sha256Short(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:4])
}

func quoteShellArg(s string) string {
	if !strings.ContainsAny(s, " \t\n'\"\\$`!#&|;(){}[]<>?*~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func getConfigDir() string {
	// Use ~/.config/planck
	home, err := os.UserHomeDir()
	if err != nil {
		return ".planck"
	}
	return filepath.Join(home, ".config", "planck")
}

func printHelp() {
	fmt.Println(`planck - Folder-based markdown editor with multi-agent support

Usage:
  planck [options]
  planck attach [--folder PATH]
  planck update [--check]
  planck version [--check]

Options:
  -f, --folder PATH  Folder containing markdown files (default: current directory)
  -h, --help         Show this help message
  -v, --version      Show version information

Commands:
  attach             Run planck inside a persistent tmux session (survives SSH disconnects)
  attach --folder    Specify working directory for the attached session
  update             Download and install the latest version
  update --check     Check for updates without installing
  version            Show version information
  version --check    Show version and check for updates

Keybindings (Global):
  Shift+Tab    Next tab
  Alt+1-9      Jump to tab by number (all modes)
  1-9          Jump to tab by number (normal mode)
  a            Create new agent tab
  x / Ctrl+X   Close current agent tab
  s            Settings
  ?            Toggle help
  q            Quit

Keybindings (Planning Tab):
  ↑/↓, j/k    Navigate files
  Enter        Open file in editor
  e            Enter edit mode
  n            New file
  d            Delete file/folder
  c            Toggle completion
  m            Move/rename file or folder
  r            Refresh file list
  h/l          Collapse/expand folders

Keybindings (Agent Tab - Input Mode):
  Ctrl+\       Exit to normal mode
  Tab          Sent to agent (autocomplete)
  Ctrl+X       Close tab
  Scroll       Browse output history

Keybindings (Agent Tab - Normal Mode):
  i / Enter    Enter input mode
  j/k          Scroll up/down
  g/G          Jump to top/bottom
  x            Close tab
  a            New agent tab

For more information, visit: https://github.com/sabizmil/planck`)
}
