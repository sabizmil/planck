package ui

import (
	"encoding/json"
	"os"

	"github.com/charmbracelet/glamour"
)

// ThemeName identifies a markdown theme preset.
type ThemeName string

const (
	ThemeNeoBrutalist    ThemeName = "neo-brutalist"
	ThemeTerminalClassic ThemeName = "terminal-classic"
	ThemeMinimalModern   ThemeName = "minimal-modern"
	ThemeRichEditorial   ThemeName = "rich-editorial"
	ThemeSoftPastel      ThemeName = "soft-pastel"
)

// AllThemes returns all available theme names in display order.
func AllThemes() []ThemeName {
	return []ThemeName{
		ThemeNeoBrutalist,
		ThemeTerminalClassic,
		ThemeMinimalModern,
		ThemeRichEditorial,
		ThemeSoftPastel,
	}
}

// ThemeDisplayName returns a human-readable name for a theme.
func ThemeDisplayName(t ThemeName) string {
	switch t {
	case ThemeNeoBrutalist:
		return "Neo-Brutalist"
	case ThemeTerminalClassic:
		return "Terminal Classic"
	case ThemeMinimalModern:
		return "Minimal Modern"
	case ThemeRichEditorial:
		return "Rich Editorial"
	case ThemeSoftPastel:
		return "Soft Pastel"
	default:
		return string(t)
	}
}

// ElementType identifies a markdown element that can be styled independently.
type ElementType string

const (
	ElementH1            ElementType = "h1"
	ElementH2            ElementType = "h2"
	ElementH3            ElementType = "h3"
	ElementH4            ElementType = "h4"
	ElementH5            ElementType = "h5"
	ElementH6            ElementType = "h6"
	ElementBody          ElementType = "body"
	ElementBold          ElementType = "bold"
	ElementItalic        ElementType = "italic"
	ElementInlineCode    ElementType = "inline_code"
	ElementCodeBlock     ElementType = "code_block"
	ElementBlockquote    ElementType = "blockquote"
	ElementLink          ElementType = "link"
	ElementList          ElementType = "list"
	ElementTaskList      ElementType = "task_list"
	ElementTable         ElementType = "table"
	ElementHR            ElementType = "hr"
	ElementImage         ElementType = "image"
	ElementStrikethrough ElementType = "strikethrough"
)

// AllElements returns all configurable element types in display order.
func AllElements() []ElementType {
	return []ElementType{
		ElementH1, ElementH2, ElementH3, ElementH4, ElementH5, ElementH6,
		ElementBody, ElementBold, ElementItalic,
		ElementInlineCode, ElementCodeBlock,
		ElementBlockquote, ElementLink, ElementList, ElementTaskList,
		ElementTable, ElementHR, ElementImage, ElementStrikethrough,
	}
}

// ElementDisplayName returns a human-readable name for an element type.
func ElementDisplayName(e ElementType) string {
	switch e {
	case ElementH1:
		return "Heading 1"
	case ElementH2:
		return "Heading 2"
	case ElementH3:
		return "Heading 3"
	case ElementH4:
		return "Heading 4"
	case ElementH5:
		return "Heading 5"
	case ElementH6:
		return "Heading 6"
	case ElementBody:
		return "Body Text"
	case ElementBold:
		return "Bold"
	case ElementItalic:
		return "Italic"
	case ElementInlineCode:
		return "Inline Code"
	case ElementCodeBlock:
		return "Code Block"
	case ElementBlockquote:
		return "Blockquote"
	case ElementLink:
		return "Links"
	case ElementList:
		return "Lists"
	case ElementTaskList:
		return "Task Lists"
	case ElementTable:
		return "Tables"
	case ElementHR:
		return "Horizontal Rule"
	case ElementImage:
		return "Images"
	case ElementStrikethrough:
		return "Strikethrough"
	default:
		return string(e)
	}
}

// MarkdownStyleConfig represents the user's current style selection.
type MarkdownStyleConfig struct {
	GlobalTheme ThemeName
	Overrides   map[ElementType]ThemeName
}

// elementStyle holds the glamour JSON properties for one element in one theme.
// Each key maps to a glamour style property name, value is the JSON-compatible value.
type elementStyle map[string]interface{}

// StyleRegistry holds all theme definitions and composes final styles.
type StyleRegistry struct {
	// themes[themeName][element] = style properties
	themes map[ThemeName]map[ElementType]elementStyle
}

// NewStyleRegistry creates a registry populated with all built-in themes.
func NewStyleRegistry() *StyleRegistry {
	r := &StyleRegistry{
		themes: make(map[ThemeName]map[ElementType]elementStyle),
	}
	r.themes[ThemeNeoBrutalist] = neoBrutalistElements()
	r.themes[ThemeTerminalClassic] = terminalClassicElements()
	r.themes[ThemeMinimalModern] = minimalModernElements()
	r.themes[ThemeRichEditorial] = richEditorialElements()
	r.themes[ThemeSoftPastel] = softPastelElements()
	return r
}

// ComposeStyle builds a complete glamour JSON style from the given config.
func (r *StyleRegistry) ComposeStyle(cfg MarkdownStyleConfig) []byte {
	noColor := os.Getenv("NO_COLOR") != ""

	style := make(map[string]interface{})

	// Document-level defaults (not element-specific)
	style["document"] = map[string]interface{}{
		"block_prefix": "\n",
		"block_suffix": "\n",
		"color":        "#E0E0E0",
		"margin":       2,
	}
	style["paragraph"] = map[string]interface{}{}
	style["text"] = map[string]interface{}{}
	style["definition_list"] = map[string]interface{}{}

	// For each element, pick the theme (override or global) and pull the style
	for _, elem := range AllElements() {
		theme := cfg.GlobalTheme
		if override, ok := cfg.Overrides[elem]; ok {
			theme = override
		}

		themeMap, ok := r.themes[theme]
		if !ok {
			themeMap = r.themes[ThemeNeoBrutalist]
		}

		if s, ok := themeMap[elem]; ok {
			result := make(elementStyle)
			for k, v := range s {
				result[k] = v
			}
			if noColor {
				result = stripColors(result)
			}
			style[elemToSingleGlamourKey(elem)] = result
		}
	}

	// Always add heading base
	style["heading"] = map[string]interface{}{
		"block_suffix": "\n",
		"bold":         true,
	}

	// List needs level_indent at the list level
	style["list"] = map[string]interface{}{
		"level_indent": 2,
	}

	// Enumeration follows list theme
	listTheme := cfg.GlobalTheme
	if override, ok := cfg.Overrides[ElementList]; ok {
		listTheme = override
	}
	if themeMap, ok := r.themes[listTheme]; ok {
		if _, ok := themeMap[ElementList]; ok {
			style["enumeration"] = map[string]interface{}{
				"block_prefix": ". ",
			}
		}
	}
	if _, ok := style["enumeration"]; !ok {
		style["enumeration"] = map[string]interface{}{
			"block_prefix": ". ",
		}
	}

	// Link text follows link theme
	if linkStyle, ok := style["link"]; ok {
		if ls, ok := linkStyle.(elementStyle); ok {
			linkText := make(elementStyle)
			if c, ok := ls["color"]; ok {
				linkText["color"] = c
			}
			if b, ok := ls["bold"]; ok {
				linkText["bold"] = b
			}
			if noColor {
				linkText = stripColors(linkText)
			}
			style["link_text"] = linkText
		}
	}

	// Image text follows image theme
	imgText := elementStyle{
		"format": "[IMG: {{.text}}]",
	}
	if imgStyle, ok := style["image"]; ok {
		if is, ok := imgStyle.(elementStyle); ok {
			if c, ok := is["color"]; ok {
				imgText["color"] = c
			}
			if b, ok := is["bold"]; ok {
				imgText["bold"] = b
			}
		}
	}
	if noColor {
		imgText = stripColors(imgText)
	}
	style["image_text"] = imgText

	// Definition list elements
	style["definition_term"] = map[string]interface{}{
		"bold": true,
	}
	style["definition_description"] = map[string]interface{}{
		"block_prefix": "  ",
	}

	if noColor {
		if doc, ok := style["document"].(map[string]interface{}); ok {
			delete(doc, "color")
			delete(doc, "background_color")
		}
	}

	data, _ := json.Marshal(style)
	return data
}

// elemToSingleGlamourKey maps an ElementType to its primary glamour JSON key.
func elemToSingleGlamourKey(e ElementType) string {
	switch e {
	case ElementH1:
		return "h1"
	case ElementH2:
		return "h2"
	case ElementH3:
		return "h3"
	case ElementH4:
		return "h4"
	case ElementH5:
		return "h5"
	case ElementH6:
		return "h6"
	case ElementBody:
		return "paragraph"
	case ElementBold:
		return "strong"
	case ElementItalic:
		return "emph"
	case ElementInlineCode:
		return "code"
	case ElementCodeBlock:
		return "code_block"
	case ElementBlockquote:
		return "block_quote"
	case ElementLink:
		return "link"
	case ElementList:
		return "item"
	case ElementTaskList:
		return "task"
	case ElementTable:
		return "table"
	case ElementHR:
		return "hr"
	case ElementImage:
		return "image"
	case ElementStrikethrough:
		return "strikethrough"
	default:
		return string(e)
	}
}

// stripColors removes color and background_color from a style, preserving structure.
func stripColors(s elementStyle) elementStyle {
	result := make(elementStyle)
	for k, v := range s {
		switch k {
		case "color", "background_color":
			continue
		case "chroma":
			// Strip colors from chroma sub-map
			if chromaMap, ok := v.(map[string]interface{}); ok {
				stripped := make(map[string]interface{})
				for ck, cv := range chromaMap {
					if subMap, ok := cv.(map[string]interface{}); ok {
						clean := make(map[string]interface{})
						for sk, sv := range subMap {
							if sk != "color" && sk != "background_color" {
								clean[sk] = sv
							}
						}
						if len(clean) > 0 {
							stripped[ck] = clean
						}
					}
				}
				if len(stripped) > 0 {
					result[k] = stripped
				}
			}
		default:
			result[k] = v
		}
	}
	return result
}

// ComposeStyleForRenderer is a convenience that creates a glamour TermRenderer from config.
func (r *StyleRegistry) ComposeStyleForRenderer(cfg MarkdownStyleConfig, width int) (*glamour.TermRenderer, error) {
	styleJSON := r.ComposeStyle(cfg)
	return glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(styleJSON),
		glamour.WithWordWrap(width),
	)
}

// NewMarkdownRenderer creates a glamour renderer with the default Neo-Brutalist style.
// Kept for backward compatibility.
func NewMarkdownRenderer(width int) (*glamour.TermRenderer, error) {
	registry := NewStyleRegistry()
	cfg := MarkdownStyleConfig{
		GlobalTheme: ThemeNeoBrutalist,
		Overrides:   map[ElementType]ThemeName{},
	}
	return registry.ComposeStyleForRenderer(cfg, width)
}

// NewMarkdownRendererWithStyle creates a renderer from a pre-composed style JSON.
func NewMarkdownRendererWithStyle(styleJSON []byte, width int) (*glamour.TermRenderer, error) {
	return glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(styleJSON),
		glamour.WithWordWrap(width),
	)
}

// =============================================================================
// Theme definitions
// =============================================================================

// --- Neo-Brutalist (C): Bold contrasts, heavy box drawing, high visibility ---

func neoBrutalistElements() map[ElementType]elementStyle {
	return map[ElementType]elementStyle{
		ElementH1: {
			"upper":            true,
			"bold":             true,
			"prefix":           " ",
			"suffix":           " ",
			"color":            "#0D0D1A",
			"background_color": "#06B6D4",
			"block_suffix":     "\n",
		},
		ElementH2: {
			"upper":        true,
			"bold":         true,
			"prefix":       "\u258C ",
			"color":        "#E0E0E0",
			"block_suffix": "\n",
		},
		ElementH3: {
			"bold":         true,
			"prefix":       "\u258E ",
			"color":        "#E0E0E0",
			"block_suffix": "\n",
		},
		ElementH4: {
			"bold":   true,
			"prefix": "\u2503 ",
			"color":  "#A0A0A0",
		},
		ElementH5: {
			"bold":   true,
			"prefix": "\u2502 ",
			"color":  "#6B7280",
		},
		ElementH6: {
			"italic": true,
			"prefix": "\u254E ",
			"color":  "#6B7280",
		},
		ElementBody: {},
		ElementBold: {
			"bold":  true,
			"color": "#06B6D4",
		},
		ElementItalic: {
			"italic": true,
		},
		ElementInlineCode: {
			"color":            "#4ADE80",
			"background_color": "#1A1A2E",
		},
		ElementCodeBlock: {
			"color":        "#E0E0E0",
			"margin":       2,
			"indent":       1,
			"indent_token": "\u2503 ",
			"chroma":       neoBrutalistChroma(),
		},
		ElementBlockquote: {
			"indent":       1,
			"indent_token": "\u2503 ",
			"color":        "#F59E0B",
		},
		ElementLink: {
			"color":     "#06B6D4",
			"bold":      true,
			"underline": true,
		},
		ElementList: {
			"block_prefix": "\u25B8 ",
		},
		ElementTaskList: {
			"ticked":   "[\u2713] ",
			"unticked": "[ ] ",
		},
		ElementTable: {
			"center_separator": "\u253C",
			"column_separator": "\u2502",
			"row_separator":    "\u2500",
		},
		ElementHR: {
			"color":  "#A0A0A0",
			"format": "\n\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\n",
		},
		ElementImage: {
			"color": "#A0A0A0",
		},
		ElementStrikethrough: {
			"crossed_out": true,
			"color":       "#EF4444",
		},
	}
}

func neoBrutalistChroma() map[string]interface{} {
	return map[string]interface{}{
		"text":                  map[string]interface{}{"color": "#E0E0E0"},
		"error":                 map[string]interface{}{"color": "#F1F1F1", "background_color": "#EF4444"},
		"comment":               map[string]interface{}{"color": "#6B7280", "italic": true},
		"comment_preproc":       map[string]interface{}{"color": "#F59E0B"},
		"keyword":               map[string]interface{}{"color": "#06B6D4", "bold": true},
		"keyword_reserved":      map[string]interface{}{"color": "#06B6D4", "bold": true},
		"keyword_namespace":     map[string]interface{}{"color": "#06B6D4"},
		"keyword_type":          map[string]interface{}{"color": "#7C3AED"},
		"operator":              map[string]interface{}{"color": "#EF4444"},
		"punctuation":           map[string]interface{}{"color": "#A0A0A0"},
		"name":                  map[string]interface{}{"color": "#E0E0E0"},
		"name_builtin":          map[string]interface{}{"color": "#22C55E"},
		"name_tag":              map[string]interface{}{"color": "#06B6D4"},
		"name_attribute":        map[string]interface{}{"color": "#22C55E"},
		"name_class":            map[string]interface{}{"color": "#F59E0B", "bold": true},
		"name_constant":         map[string]interface{}{"color": "#06B6D4"},
		"name_decorator":        map[string]interface{}{"color": "#F59E0B"},
		"name_exception":        map[string]interface{}{"color": "#EF4444"},
		"name_function":         map[string]interface{}{"color": "#22C55E"},
		"name_other":            map[string]interface{}{"color": "#E0E0E0"},
		"literal":               map[string]interface{}{"color": "#C4B5FD"},
		"literal_number":        map[string]interface{}{"color": "#C4B5FD"},
		"literal_date":          map[string]interface{}{"color": "#C4B5FD"},
		"literal_string":        map[string]interface{}{"color": "#FDE68A"},
		"literal_string_escape": map[string]interface{}{"color": "#F59E0B"},
		"generic_deleted":       map[string]interface{}{"color": "#EF4444"},
		"generic_emph":          map[string]interface{}{"italic": true},
		"generic_inserted":      map[string]interface{}{"color": "#22C55E"},
		"generic_strong":        map[string]interface{}{"bold": true},
		"generic_subheading":    map[string]interface{}{"color": "#06B6D4"},
		"background":            map[string]interface{}{"background_color": "#0D0D1A"},
	}
}

// --- Terminal Classic (E): Traditional terminal look, ASCII-first, dense ---

func terminalClassicElements() map[ElementType]elementStyle {
	return map[ElementType]elementStyle{
		ElementH1: {
			"upper":        true,
			"bold":         true,
			"prefix":       "== ",
			"suffix":       " ==",
			"color":        "#E0E0E0",
			"block_suffix": "\n",
		},
		ElementH2: {
			"bold":         true,
			"prefix":       "-- ",
			"suffix":       " --",
			"color":        "#E0E0E0",
			"block_suffix": "\n",
		},
		ElementH3: {
			"bold":         true,
			"prefix":       "## ",
			"color":        "#A0A0A0",
			"block_suffix": "\n",
		},
		ElementH4: {
			"bold":   true,
			"prefix": "### ",
			"color":  "#A0A0A0",
		},
		ElementH5: {
			"bold":   true,
			"prefix": "#### ",
			"color":  "#6B7280",
		},
		ElementH6: {
			"italic": true,
			"prefix": "##### ",
			"color":  "#6B7280",
		},
		ElementBody: {},
		ElementBold: {
			"bold": true,
		},
		ElementItalic: {
			"italic": true,
		},
		ElementInlineCode: {
			"color": "#22C55E",
		},
		ElementCodeBlock: {
			"color":        "#E0E0E0",
			"margin":       2,
			"indent":       1,
			"indent_token": "| ",
			"chroma":       terminalClassicChroma(),
		},
		ElementBlockquote: {
			"indent":       1,
			"indent_token": "| ",
			"color":        "#A0A0A0",
		},
		ElementLink: {
			"color":     "#22C55E",
			"underline": true,
		},
		ElementList: {
			"block_prefix": "* ",
		},
		ElementTaskList: {
			"ticked":   "[x] ",
			"unticked": "[ ] ",
		},
		ElementTable: {
			"center_separator": "+",
			"column_separator": "|",
			"row_separator":    "-",
		},
		ElementHR: {
			"color":  "#6B7280",
			"format": "\n----------------------------------------\n",
		},
		ElementImage: {
			"color": "#6B7280",
		},
		ElementStrikethrough: {
			"crossed_out": true,
			"color":       "#6B7280",
		},
	}
}

func terminalClassicChroma() map[string]interface{} {
	return map[string]interface{}{
		"text":                  map[string]interface{}{"color": "#E0E0E0"},
		"error":                 map[string]interface{}{"color": "#EF4444"},
		"comment":               map[string]interface{}{"color": "#6B7280", "italic": true},
		"comment_preproc":       map[string]interface{}{"color": "#A0A0A0"},
		"keyword":               map[string]interface{}{"color": "#22C55E", "bold": true},
		"keyword_reserved":      map[string]interface{}{"color": "#22C55E", "bold": true},
		"keyword_namespace":     map[string]interface{}{"color": "#22C55E"},
		"keyword_type":          map[string]interface{}{"color": "#06B6D4"},
		"operator":              map[string]interface{}{"color": "#E0E0E0"},
		"punctuation":           map[string]interface{}{"color": "#A0A0A0"},
		"name":                  map[string]interface{}{"color": "#E0E0E0"},
		"name_builtin":          map[string]interface{}{"color": "#06B6D4"},
		"name_tag":              map[string]interface{}{"color": "#22C55E"},
		"name_attribute":        map[string]interface{}{"color": "#06B6D4"},
		"name_class":            map[string]interface{}{"color": "#E0E0E0", "bold": true},
		"name_constant":         map[string]interface{}{"color": "#06B6D4"},
		"name_decorator":        map[string]interface{}{"color": "#A0A0A0"},
		"name_exception":        map[string]interface{}{"color": "#EF4444"},
		"name_function":         map[string]interface{}{"color": "#E0E0E0"},
		"name_other":            map[string]interface{}{"color": "#E0E0E0"},
		"literal":               map[string]interface{}{"color": "#F59E0B"},
		"literal_number":        map[string]interface{}{"color": "#F59E0B"},
		"literal_date":          map[string]interface{}{"color": "#F59E0B"},
		"literal_string":        map[string]interface{}{"color": "#22C55E"},
		"literal_string_escape": map[string]interface{}{"color": "#F59E0B"},
		"generic_deleted":       map[string]interface{}{"color": "#EF4444"},
		"generic_emph":          map[string]interface{}{"italic": true},
		"generic_inserted":      map[string]interface{}{"color": "#22C55E"},
		"generic_strong":        map[string]interface{}{"bold": true},
		"generic_subheading":    map[string]interface{}{"color": "#A0A0A0"},
		"background":            map[string]interface{}{},
	}
}

// --- Minimal Modern (A): Clean whitespace, subtle accents, light touch ---

func minimalModernElements() map[ElementType]elementStyle {
	return map[ElementType]elementStyle{
		ElementH1: {
			"upper":        true,
			"bold":         true,
			"color":        "#06B6D4",
			"block_suffix": "\n",
		},
		ElementH2: {
			"bold":         true,
			"color":        "#E0E0E0",
			"block_suffix": "\n",
		},
		ElementH3: {
			"bold":         true,
			"color":        "#A0A0A0",
			"block_suffix": "\n",
		},
		ElementH4: {
			"bold":  true,
			"color": "#A0A0A0",
		},
		ElementH5: {
			"italic": true,
			"color":  "#6B7280",
		},
		ElementH6: {
			"italic": true,
			"color":  "#6B7280",
		},
		ElementBody: {},
		ElementBold: {
			"bold": true,
		},
		ElementItalic: {
			"italic": true,
		},
		ElementInlineCode: {
			"color":            "#06B6D4",
			"background_color": "#1A1A2E",
		},
		ElementCodeBlock: {
			"color":        "#E0E0E0",
			"margin":       2,
			"indent":       1,
			"indent_token": "  ",
			"chroma":       minimalModernChroma(),
		},
		ElementBlockquote: {
			"indent":       1,
			"indent_token": "  \u2502 ",
			"color":        "#6B7280",
			"italic":       true,
		},
		ElementLink: {
			"color":     "#06B6D4",
			"underline": true,
		},
		ElementList: {
			"block_prefix": "- ",
		},
		ElementTaskList: {
			"ticked":   "[\u2713] ",
			"unticked": "[ ] ",
		},
		ElementTable: {
			"center_separator": "\u253C",
			"column_separator": "\u2502",
			"row_separator":    "\u2500",
		},
		ElementHR: {
			"color":  "#6B7280",
			"format": "\n\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\n",
		},
		ElementImage: {
			"color": "#6B7280",
		},
		ElementStrikethrough: {
			"crossed_out": true,
			"color":       "#6B7280",
		},
	}
}

func minimalModernChroma() map[string]interface{} {
	return map[string]interface{}{
		"text":                  map[string]interface{}{"color": "#E0E0E0"},
		"error":                 map[string]interface{}{"color": "#EF4444"},
		"comment":               map[string]interface{}{"color": "#6B7280", "italic": true},
		"comment_preproc":       map[string]interface{}{"color": "#A0A0A0"},
		"keyword":               map[string]interface{}{"color": "#06B6D4"},
		"keyword_reserved":      map[string]interface{}{"color": "#06B6D4"},
		"keyword_namespace":     map[string]interface{}{"color": "#06B6D4"},
		"keyword_type":          map[string]interface{}{"color": "#7C3AED"},
		"operator":              map[string]interface{}{"color": "#A0A0A0"},
		"punctuation":           map[string]interface{}{"color": "#6B7280"},
		"name":                  map[string]interface{}{"color": "#E0E0E0"},
		"name_builtin":          map[string]interface{}{"color": "#06B6D4"},
		"name_tag":              map[string]interface{}{"color": "#06B6D4"},
		"name_attribute":        map[string]interface{}{"color": "#22C55E"},
		"name_class":            map[string]interface{}{"color": "#E0E0E0", "bold": true},
		"name_constant":         map[string]interface{}{"color": "#06B6D4"},
		"name_decorator":        map[string]interface{}{"color": "#F59E0B"},
		"name_exception":        map[string]interface{}{"color": "#EF4444"},
		"name_function":         map[string]interface{}{"color": "#22C55E"},
		"name_other":            map[string]interface{}{"color": "#E0E0E0"},
		"literal":               map[string]interface{}{"color": "#C4B5FD"},
		"literal_number":        map[string]interface{}{"color": "#C4B5FD"},
		"literal_date":          map[string]interface{}{"color": "#C4B5FD"},
		"literal_string":        map[string]interface{}{"color": "#FDE68A"},
		"literal_string_escape": map[string]interface{}{"color": "#F59E0B"},
		"generic_deleted":       map[string]interface{}{"color": "#EF4444"},
		"generic_emph":          map[string]interface{}{"italic": true},
		"generic_inserted":      map[string]interface{}{"color": "#22C55E"},
		"generic_strong":        map[string]interface{}{"bold": true},
		"generic_subheading":    map[string]interface{}{"color": "#06B6D4"},
		"background":            map[string]interface{}{"background_color": "#0D0D1A"},
	}
}

// --- Rich Editorial (B): Strong typographic hierarchy, borders, magazine feel ---

func richEditorialElements() map[ElementType]elementStyle {
	return map[ElementType]elementStyle{
		ElementH1: {
			"bold":         true,
			"prefix":       "\u2503 ",
			"color":        "#E0E0E0",
			"block_suffix": "\n",
		},
		ElementH2: {
			"bold":         true,
			"prefix":       "\u2503 ",
			"color":        "#06B6D4",
			"block_suffix": "\n",
		},
		ElementH3: {
			"bold":         true,
			"prefix":       "\u2502 ",
			"color":        "#A0A0A0",
			"block_suffix": "\n",
		},
		ElementH4: {
			"bold":   true,
			"prefix": "\u2502 ",
			"color":  "#6B7280",
		},
		ElementH5: {
			"italic": true,
			"prefix": "  ",
			"color":  "#6B7280",
		},
		ElementH6: {
			"italic": true,
			"prefix": "  ",
			"color":  "#6B7280",
		},
		ElementBody: {},
		ElementBold: {
			"bold":  true,
			"color": "#E0E0E0",
		},
		ElementItalic: {
			"italic": true,
			"color":  "#A0A0A0",
		},
		ElementInlineCode: {
			"color":            "#22C55E",
			"background_color": "#1A1A2E",
		},
		ElementCodeBlock: {
			"color":        "#E0E0E0",
			"margin":       2,
			"indent":       1,
			"indent_token": "\u2502 ",
			"chroma":       richEditorialChroma(),
		},
		ElementBlockquote: {
			"indent":       1,
			"indent_token": "\u2503 ",
			"color":        "#A0A0A0",
			"italic":       true,
		},
		ElementLink: {
			"color":     "#06B6D4",
			"bold":      true,
			"underline": true,
		},
		ElementList: {
			"block_prefix": "\u2022 ",
		},
		ElementTaskList: {
			"ticked":   "[\u2713] ",
			"unticked": "[\u2007] ",
		},
		ElementTable: {
			"center_separator": "\u253C",
			"column_separator": "\u2502",
			"row_separator":    "\u2500",
		},
		ElementHR: {
			"color":  "#06B6D4",
			"format": "\n\u2500\u2500\u2500 \u25C6 \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500 \u25C6 \u2500\u2500\u2500\n",
		},
		ElementImage: {
			"color": "#A0A0A0",
			"bold":  true,
		},
		ElementStrikethrough: {
			"crossed_out": true,
			"color":       "#6B7280",
		},
	}
}

func richEditorialChroma() map[string]interface{} {
	return map[string]interface{}{
		"text":                  map[string]interface{}{"color": "#E0E0E0"},
		"error":                 map[string]interface{}{"color": "#EF4444"},
		"comment":               map[string]interface{}{"color": "#6B7280", "italic": true},
		"comment_preproc":       map[string]interface{}{"color": "#F59E0B"},
		"keyword":               map[string]interface{}{"color": "#06B6D4", "bold": true},
		"keyword_reserved":      map[string]interface{}{"color": "#06B6D4", "bold": true},
		"keyword_namespace":     map[string]interface{}{"color": "#06B6D4"},
		"keyword_type":          map[string]interface{}{"color": "#7C3AED"},
		"operator":              map[string]interface{}{"color": "#A0A0A0"},
		"punctuation":           map[string]interface{}{"color": "#6B7280"},
		"name":                  map[string]interface{}{"color": "#E0E0E0"},
		"name_builtin":          map[string]interface{}{"color": "#22C55E"},
		"name_tag":              map[string]interface{}{"color": "#06B6D4"},
		"name_attribute":        map[string]interface{}{"color": "#22C55E"},
		"name_class":            map[string]interface{}{"color": "#F59E0B", "bold": true},
		"name_constant":         map[string]interface{}{"color": "#06B6D4"},
		"name_decorator":        map[string]interface{}{"color": "#F59E0B"},
		"name_exception":        map[string]interface{}{"color": "#EF4444"},
		"name_function":         map[string]interface{}{"color": "#22C55E"},
		"name_other":            map[string]interface{}{"color": "#E0E0E0"},
		"literal":               map[string]interface{}{"color": "#C4B5FD"},
		"literal_number":        map[string]interface{}{"color": "#C4B5FD"},
		"literal_date":          map[string]interface{}{"color": "#C4B5FD"},
		"literal_string":        map[string]interface{}{"color": "#FDE68A"},
		"literal_string_escape": map[string]interface{}{"color": "#F59E0B"},
		"generic_deleted":       map[string]interface{}{"color": "#EF4444"},
		"generic_emph":          map[string]interface{}{"italic": true},
		"generic_inserted":      map[string]interface{}{"color": "#22C55E"},
		"generic_strong":        map[string]interface{}{"bold": true},
		"generic_subheading":    map[string]interface{}{"color": "#06B6D4"},
		"background":            map[string]interface{}{"background_color": "#0D0D1A"},
	}
}

// --- Soft Pastel (D): Gentle background tints, rounded feel, warm tones ---

func softPastelElements() map[ElementType]elementStyle {
	return map[ElementType]elementStyle{
		ElementH1: {
			"bold":             true,
			"prefix":           " ",
			"suffix":           " ",
			"color":            "#1A1A2E",
			"background_color": "#C4B5FD",
			"block_suffix":     "\n",
		},
		ElementH2: {
			"bold":         true,
			"prefix":       "\u2502 ",
			"color":        "#C4B5FD",
			"block_suffix": "\n",
		},
		ElementH3: {
			"bold":         true,
			"prefix":       "\u2502 ",
			"color":        "#FCA5A5",
			"block_suffix": "\n",
		},
		ElementH4: {
			"bold":  true,
			"color": "#93C5FD",
		},
		ElementH5: {
			"italic": true,
			"color":  "#A0A0A0",
		},
		ElementH6: {
			"italic": true,
			"color":  "#6B7280",
		},
		ElementBody: {},
		ElementBold: {
			"bold":  true,
			"color": "#FCA5A5",
		},
		ElementItalic: {
			"italic": true,
			"color":  "#93C5FD",
		},
		ElementInlineCode: {
			"color":            "#BBF7D0",
			"background_color": "#1A1A2E",
		},
		ElementCodeBlock: {
			"color":        "#E0E0E0",
			"margin":       2,
			"indent":       1,
			"indent_token": "\u2502 ",
			"chroma":       softPastelChroma(),
		},
		ElementBlockquote: {
			"indent":       1,
			"indent_token": "\u2502 ",
			"color":        "#93C5FD",
			"italic":       true,
		},
		ElementLink: {
			"color":     "#C4B5FD",
			"underline": true,
		},
		ElementList: {
			"block_prefix": "\u25E6 ",
		},
		ElementTaskList: {
			"ticked":   "[\u2713] ",
			"unticked": "[\u2007] ",
		},
		ElementTable: {
			"center_separator": "\u253C",
			"column_separator": "\u2502",
			"row_separator":    "\u2500",
		},
		ElementHR: {
			"color":  "#C4B5FD",
			"format": "\n\u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500 \u2500\n",
		},
		ElementImage: {
			"color": "#93C5FD",
		},
		ElementStrikethrough: {
			"crossed_out": true,
			"color":       "#A0A0A0",
		},
	}
}

func softPastelChroma() map[string]interface{} {
	return map[string]interface{}{
		"text":                  map[string]interface{}{"color": "#E0E0E0"},
		"error":                 map[string]interface{}{"color": "#FCA5A5"},
		"comment":               map[string]interface{}{"color": "#6B7280", "italic": true},
		"comment_preproc":       map[string]interface{}{"color": "#A0A0A0"},
		"keyword":               map[string]interface{}{"color": "#C4B5FD"},
		"keyword_reserved":      map[string]interface{}{"color": "#C4B5FD"},
		"keyword_namespace":     map[string]interface{}{"color": "#C4B5FD"},
		"keyword_type":          map[string]interface{}{"color": "#93C5FD"},
		"operator":              map[string]interface{}{"color": "#A0A0A0"},
		"punctuation":           map[string]interface{}{"color": "#6B7280"},
		"name":                  map[string]interface{}{"color": "#E0E0E0"},
		"name_builtin":          map[string]interface{}{"color": "#BBF7D0"},
		"name_tag":              map[string]interface{}{"color": "#C4B5FD"},
		"name_attribute":        map[string]interface{}{"color": "#BBF7D0"},
		"name_class":            map[string]interface{}{"color": "#FCA5A5", "bold": true},
		"name_constant":         map[string]interface{}{"color": "#93C5FD"},
		"name_decorator":        map[string]interface{}{"color": "#FCA5A5"},
		"name_exception":        map[string]interface{}{"color": "#FCA5A5"},
		"name_function":         map[string]interface{}{"color": "#BBF7D0"},
		"name_other":            map[string]interface{}{"color": "#E0E0E0"},
		"literal":               map[string]interface{}{"color": "#FDE68A"},
		"literal_number":        map[string]interface{}{"color": "#FDE68A"},
		"literal_date":          map[string]interface{}{"color": "#FDE68A"},
		"literal_string":        map[string]interface{}{"color": "#BBF7D0"},
		"literal_string_escape": map[string]interface{}{"color": "#FCA5A5"},
		"generic_deleted":       map[string]interface{}{"color": "#FCA5A5"},
		"generic_emph":          map[string]interface{}{"italic": true},
		"generic_inserted":      map[string]interface{}{"color": "#BBF7D0"},
		"generic_strong":        map[string]interface{}{"bold": true},
		"generic_subheading":    map[string]interface{}{"color": "#C4B5FD"},
		"background":            map[string]interface{}{"background_color": "#0D0D1A"},
	}
}

// PreviewMarkdown is a sample markdown string used in the settings live preview.
const PreviewMarkdown = `# Main Heading

## Section Title

### Subsection

#### Fourth Level

This is body text with **bold text**, *italic text*, and ***bold italic***.
You can also use ~~strikethrough~~ for deleted content.

Use ` + "`" + `inline code` + "`" + ` for short references like ` + "`" + `fmt.Println()` + "`" + `.

> Wisdom is the reward you get for a lifetime of listening
> when you would have preferred to talk.

` + "```" + `go
func main() {
    fmt.Println("Hello, world!")
}
` + "```" + `

- First item
- Second item
  - Nested item

1. Ordered first
2. Ordered second

- [x] Completed task
- [ ] Pending task

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Cell A   | Cell B   | Cell C   |

---

Visit [Example Link](https://example.com) for details.
![Screenshot](preview.png)
`
