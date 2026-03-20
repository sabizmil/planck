package ui

import "strings"

// Action identifies a bindable keyboard action.
type Action string

// Context identifies a keybinding context (scope where keys have specific meaning).
type Context string

// Keybinding contexts.
const (
	ContextGlobal      Context = "global"
	ContextFileList    Context = "file_list"
	ContextEditor      Context = "editor"
	ContextEditorEdit  Context = "editor_edit"
	ContextAgentNormal Context = "agent_normal"
	ContextAgentInput  Context = "agent_input"
)

// AllContexts returns all valid keybinding contexts.
func AllContexts() []Context {
	return []Context{
		ContextGlobal,
		ContextFileList,
		ContextEditor,
		ContextEditorEdit,
		ContextAgentNormal,
		ContextAgentInput,
	}
}

// --- Global actions ---

const (
	ActionQuit        Action = "quit"
	ActionToggleHelp  Action = "toggle_help"
	ActionNextTab     Action = "next_tab"
	ActionCloseTab    Action = "close_tab"
	ActionSettings    Action = "settings"
	ActionCreateAgent Action = "create_agent"
	ActionCloseAgent  Action = "close_agent"
	// Alt+N and N tab switching are handled structurally, not via keymap.
)

// --- File list actions ---

const (
	ActionMoveDown       Action = "move_down"
	ActionMoveUp         Action = "move_up"
	ActionOpenFile       Action = "open_file"
	ActionExpandFolder   Action = "expand_folder"
	ActionCollapseDir    Action = "collapse_dir"
	ActionEditMode       Action = "edit_mode"
	ActionNewFile        Action = "new_file"
	ActionDeleteFile     Action = "delete_file"
	ActionToggleComplete Action = "toggle_complete"
	ActionMoveFile       Action = "move_file"
	ActionRefresh        Action = "refresh"
	ActionExcludeDir     Action = "exclude_dir"
)

// --- Editor view mode actions ---

const (
	ActionEditorDown     Action = "editor_down"
	ActionEditorUp       Action = "editor_up"
	ActionEditorPageDown Action = "editor_page_down"
	ActionEditorPageUp   Action = "editor_page_up"
	ActionEditorTop      Action = "editor_top"
	ActionEditorBottom   Action = "editor_bottom"
	ActionEditorEdit     Action = "editor_edit"
)

// --- Editor edit mode actions ---

const (
	ActionEditorUndo Action = "editor_undo"
	ActionEditorRedo Action = "editor_redo"
)

// --- Agent normal mode actions ---

const (
	ActionEnterInput     Action = "enter_input"
	ActionAgentClose     Action = "agent_close"
	ActionAgentNew       Action = "agent_new"
	ActionScrollUp       Action = "scroll_up"
	ActionScrollDown     Action = "scroll_down"
	ActionScrollTop      Action = "scroll_top"
	ActionScrollBottom   Action = "scroll_bottom"
	ActionScrollPageUp   Action = "scroll_page_up"
	ActionScrollPageDown Action = "scroll_page_down"
)

// --- Agent input mode actions ---

const (
	ActionExitInput Action = "exit_input"
)

// Binding maps an action to one or more key strings.
type Binding struct {
	Action Action
	Keys   []string
	Desc   string // human-readable description for help/settings
}

// ContextBindings holds all bindings for a single context.
type ContextBindings struct {
	Context  Context
	Label    string // human-readable context name
	Bindings []Binding
}

// Keymap holds all keybindings organized by context.
type Keymap struct {
	Contexts []ContextBindings

	// Precomputed lookup: context → action → set of keys
	lookup map[Context]map[Action]map[string]bool
	// Reverse lookup: context → key → action
	reverse map[Context]map[string]Action
}

// DefaultKeymap returns the keymap with all default bindings matching the
// current hardcoded values throughout the codebase.
func DefaultKeymap() *Keymap {
	km := &Keymap{
		Contexts: []ContextBindings{
			{
				Context: ContextGlobal,
				Label:   "Global",
				Bindings: []Binding{
					{ActionToggleHelp, []string{"?"}, "Toggle help"},
					{ActionQuit, []string{"q"}, "Quit"},
					{ActionNextTab, []string{"|"}, "Next tab"},
					{ActionCloseTab, []string{"ctrl+x"}, "Close agent tab"},
					{ActionSettings, []string{"s"}, "Settings"},
					{ActionCreateAgent, []string{"a"}, "Create agent tab"},
					{ActionCloseAgent, []string{"x"}, "Close agent tab"},
				},
			},
			{
				Context: ContextFileList,
				Label:   "File Browser",
				Bindings: []Binding{
					{ActionMoveDown, []string{"j", "down"}, "Move down"},
					{ActionMoveUp, []string{"k", "up"}, "Move up"},
					{ActionOpenFile, []string{"enter"}, "Open file"},
					{ActionExpandFolder, []string{"l", "right"}, "Expand folder"},
					{ActionCollapseDir, []string{"h", "left"}, "Collapse folder"},
					{ActionEditMode, []string{"e"}, "Edit mode"},
					{ActionNewFile, []string{"n"}, "New file"},
					{ActionDeleteFile, []string{"d", "D"}, "Delete file"},
					{ActionToggleComplete, []string{"c"}, "Toggle complete"},
					{ActionMoveFile, []string{"m"}, "Move file/folder"},
					{ActionRefresh, []string{"r"}, "Refresh file list"},
					{ActionExcludeDir, []string{"X"}, "Exclude folder"},
				},
			},
			{
				Context: ContextEditor,
				Label:   "Editor (View)",
				Bindings: []Binding{
					{ActionEditorEdit, []string{"e"}, "Enter edit mode"},
					{ActionEditorDown, []string{"down", "j"}, "Scroll down"},
					{ActionEditorUp, []string{"up", "k"}, "Scroll up"},
					{ActionEditorPageDown, []string{"pgdown", "ctrl+d"}, "Page down"},
					{ActionEditorPageUp, []string{"pgup", "ctrl+u"}, "Page up"},
					{ActionEditorTop, []string{"home", "g"}, "Go to top"},
					{ActionEditorBottom, []string{"end", "G"}, "Go to bottom"},
				},
			},
			{
				Context: ContextEditorEdit,
				Label:   "Editor (Edit)",
				Bindings: []Binding{
					{ActionEditorUndo, []string{"ctrl+z"}, "Undo"},
					{ActionEditorRedo, []string{"ctrl+y"}, "Redo"},
				},
			},
			{
				Context: ContextAgentNormal,
				Label:   "Agent (Normal)",
				Bindings: []Binding{
					{ActionEnterInput, []string{"i", "enter"}, "Enter input mode"},
					{ActionScrollUp, []string{"k", "up"}, "Scroll up"},
					{ActionScrollDown, []string{"j", "down"}, "Scroll down"},
					{ActionScrollTop, []string{"g"}, "Scroll to top"},
					{ActionScrollBottom, []string{"G"}, "Scroll to bottom"},
					{ActionScrollPageUp, []string{"pgup"}, "Page up"},
					{ActionScrollPageDown, []string{"pgdown"}, "Page down"},
					{ActionAgentClose, []string{"x"}, "Close tab"},
					{ActionAgentNew, []string{"a"}, "New agent tab"},
				},
			},
			{
				Context: ContextAgentInput,
				Label:   "Agent (Input)",
				Bindings: []Binding{
					{ActionExitInput, []string{`ctrl+\`}, "Exit to normal mode"},
				},
			},
		},
	}
	km.rebuild()
	return km
}

// Matches returns true if the given key matches any binding for the action
// in the specified context.
func (km *Keymap) Matches(ctx Context, action Action, key string) bool {
	if ctxMap, ok := km.lookup[ctx]; ok {
		if keys, ok := ctxMap[action]; ok {
			return keys[key]
		}
	}
	return false
}

// KeysFor returns the key strings bound to an action in a context.
func (km *Keymap) KeysFor(ctx Context, action Action) []string {
	if ctxMap, ok := km.lookup[ctx]; ok {
		if keys, ok := ctxMap[action]; ok {
			result := make([]string, 0, len(keys))
			for k := range keys {
				result = append(result, k)
			}
			return result
		}
	}
	return nil
}

// ActionFor returns the action bound to a key in a context, or "" if none.
func (km *Keymap) ActionFor(ctx Context, key string) Action {
	if ctxMap, ok := km.reverse[ctx]; ok {
		return ctxMap[key]
	}
	return ""
}

// DisplayKeysFor returns a human-readable string of keys for an action
// (e.g., "j / ↓").
func (km *Keymap) DisplayKeysFor(ctx Context, action Action) string {
	for _, cb := range km.Contexts {
		if cb.Context != ctx {
			continue
		}
		for _, b := range cb.Bindings {
			if b.Action == action {
				return formatKeys(b.Keys)
			}
		}
	}
	return ""
}

// DescFor returns the description for an action in a context.
func (km *Keymap) DescFor(ctx Context, action Action) string {
	for _, cb := range km.Contexts {
		if cb.Context != ctx {
			continue
		}
		for _, b := range cb.Bindings {
			if b.Action == action {
				return b.Desc
			}
		}
	}
	return ""
}

// SetBinding replaces the keys for an action in a context.
// Returns false if the context or action was not found.
func (km *Keymap) SetBinding(ctx Context, action Action, keys []string) bool {
	for i, cb := range km.Contexts {
		if cb.Context != ctx {
			continue
		}
		for j, b := range cb.Bindings {
			if b.Action == action {
				km.Contexts[i].Bindings[j].Keys = keys
				km.rebuild()
				return true
			}
		}
	}
	return false
}

// ApplyOverrides merges user overrides onto the keymap.
// overrides is context → action → comma-separated keys.
// Unknown contexts/actions are silently ignored (validation happens at config level).
func (km *Keymap) ApplyOverrides(overrides map[string]map[string]string) {
	for ctxStr, actions := range overrides {
		ctx := Context(ctxStr)
		for actionStr, keysStr := range actions {
			action := Action(actionStr)
			keys := splitKeys(keysStr)
			if len(keys) > 0 {
				km.SetBinding(ctx, action, keys)
			}
		}
	}
}

// IsCustomized returns true if the binding for an action differs from defaults.
func (km *Keymap) IsCustomized(ctx Context, action Action) bool {
	defaults := DefaultKeymap()
	currentKeys := km.KeysFor(ctx, action)
	defaultKeys := defaults.KeysFor(ctx, action)
	if len(currentKeys) != len(defaultKeys) {
		return true
	}
	currentSet := make(map[string]bool, len(currentKeys))
	for _, k := range currentKeys {
		currentSet[k] = true
	}
	for _, k := range defaultKeys {
		if !currentSet[k] {
			return true
		}
	}
	return false
}

// ValidAction returns true if the action exists in the given context.
func (km *Keymap) ValidAction(ctx Context, action Action) bool {
	for _, cb := range km.Contexts {
		if cb.Context != ctx {
			continue
		}
		for _, b := range cb.Bindings {
			if b.Action == action {
				return true
			}
		}
	}
	return false
}

// ValidContext returns true if the context exists.
func (km *Keymap) ValidContext(ctx Context) bool {
	for _, cb := range km.Contexts {
		if cb.Context == ctx {
			return true
		}
	}
	return false
}

// Clone returns a deep copy of the keymap.
func (km *Keymap) Clone() *Keymap {
	clone := &Keymap{
		Contexts: make([]ContextBindings, len(km.Contexts)),
	}
	for i, cb := range km.Contexts {
		clone.Contexts[i] = ContextBindings{
			Context:  cb.Context,
			Label:    cb.Label,
			Bindings: make([]Binding, len(cb.Bindings)),
		}
		for j, b := range cb.Bindings {
			keys := make([]string, len(b.Keys))
			copy(keys, b.Keys)
			clone.Contexts[i].Bindings[j] = Binding{
				Action: b.Action,
				Keys:   keys,
				Desc:   b.Desc,
			}
		}
	}
	clone.rebuild()
	return clone
}

// rebuild regenerates the lookup and reverse maps from the Contexts slice.
func (km *Keymap) rebuild() {
	km.lookup = make(map[Context]map[Action]map[string]bool)
	km.reverse = make(map[Context]map[string]Action)

	for _, cb := range km.Contexts {
		if km.lookup[cb.Context] == nil {
			km.lookup[cb.Context] = make(map[Action]map[string]bool)
		}
		if km.reverse[cb.Context] == nil {
			km.reverse[cb.Context] = make(map[string]Action)
		}
		for _, b := range cb.Bindings {
			if km.lookup[cb.Context][b.Action] == nil {
				km.lookup[cb.Context][b.Action] = make(map[string]bool)
			}
			for _, k := range b.Keys {
				km.lookup[cb.Context][b.Action][k] = true
				km.reverse[cb.Context][k] = b.Action
			}
		}
	}
}

// splitKeys splits a comma-separated key string into individual keys.
func splitKeys(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// formatKeys returns a display string for a list of keys.
func formatKeys(keys []string) string {
	display := make([]string, len(keys))
	for i, k := range keys {
		display[i] = displayKey(k)
	}
	return strings.Join(display, " / ")
}

// displayKey converts a key string to a human-readable form.
func displayKey(key string) string {
	replacer := strings.NewReplacer(
		"up", "↑",
		"down", "↓",
		"left", "←",
		"right", "→",
		"enter", "Enter",
		"pgup", "PgUp",
		"pgdown", "PgDn",
		"home", "Home",
		"end", "End",
		"ctrl+", "Ctrl+",
		"alt+", "Alt+",
		"shift+", "Shift+",
	)
	return replacer.Replace(key)
}
