package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// agentsFocus tracks focus within the agents page.
type agentsFocus int

const (
	agentsFocusList agentsFocus = iota
	agentsFocusDetail
)

// agentDetailField identifies a field in the detail pane.
type agentDetailField int

const (
	agentFieldCommand agentDetailField = iota
	agentFieldLabel
	agentFieldArgs
	agentFieldDefault
	agentDetailFieldCount
)

// agentsPage implements the Agents settings page.
type agentsPage struct {
	theme *Theme

	// Agent data — ordered keys + configs
	agentKeys []string
	agents    map[string]AgentSettingsConfig

	// Navigation
	focus       agentsFocus
	listIdx     int
	detailIdx   int
	detailField agentDetailField

	// Text editing state
	editing   bool
	editValue string
	editCur   int
}

func newAgentsPage(theme *Theme, agents map[string]AgentSettingsConfig) *agentsPage {
	keys := make([]string, 0, len(agents))
	for k := range agents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// Move default agent to front
	for i, k := range keys {
		if agents[k].Default {
			keys[0], keys[i] = keys[i], keys[0]
			break
		}
	}

	return &agentsPage{
		theme:     theme,
		agentKeys: keys,
		agents:    agents,
		focus:     agentsFocusList,
	}
}

func (p *agentsPage) Title() string { return "Agents" }

func (p *agentsPage) IsEditing() bool { return p.editing }

func (p *agentsPage) FooterHints() string {
	if p.editing {
		return "[Enter] confirm  [Esc] cancel"
	}
	if p.focus == agentsFocusList {
		return "[j/k] navigate  [Enter/\u2192] edit  [Tab] section  [Esc] close"
	}
	return "[j/k] navigate  [Enter] edit  [\u2190/h] back  [Tab] section  [Esc] close"
}

func (p *agentsPage) OnEnter() {
	p.listIdx = 0
	p.detailField = agentFieldCommand
	p.editing = false
}

func (p *agentsPage) OnLeave() tea.Cmd {
	p.editing = false
	agents := make(map[string]AgentSettingsConfig, len(p.agents))
	for k, v := range p.agents {
		agents[k] = v
	}
	return func() tea.Msg {
		return AgentsSettingsChangedMsg{Agents: agents}
	}
}

func (p *agentsPage) selectedAgent() (string, AgentSettingsConfig) {
	if p.listIdx < len(p.agentKeys) {
		key := p.agentKeys[p.listIdx]
		return key, p.agents[key]
	}
	return "", AgentSettingsConfig{}
}

func (p *agentsPage) Update(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if p.editing {
		return p.handleEditKey(key, msg)
		}

	if p.focus == agentsFocusList {
		return p.handleListKey(key)
	}
	return p.handleDetailKey(key, msg)
}

func (p *agentsPage) handleListKey(key string) tea.Cmd {
	switch key {
	case "j", "down":
		if p.listIdx < len(p.agentKeys)-1 {
			p.listIdx++
		}
	case "k", "up":
		if p.listIdx > 0 {
			p.listIdx--
		}
	case "enter", "l", "right":
		p.focus = agentsFocusDetail
		p.detailField = agentFieldCommand
	case "h", "left":
		return nil // signal: go to sidebar
	}
	return nil
}

func (p *agentsPage) handleDetailKey(key string, msg tea.KeyMsg) tea.Cmd {
	switch key {
	case "j", "down":
		if p.detailField < agentDetailFieldCount-1 {
			p.detailField++
		}
	case "k", "up":
		if p.detailField > 0 {
			p.detailField--
		}
	case "h", "left":
		p.focus = agentsFocusList
		// Return a no-op cmd to signal we consumed the key
		return func() tea.Msg { return nil }
	case "enter", "l", "right":
		p.activateDetailField()
	}
	return nil
}

func (p *agentsPage) activateDetailField() {
	agentKey, agent := p.selectedAgent()
	if agentKey == "" {
		return
	}

	switch p.detailField {
	case agentFieldCommand:
		p.editing = true
		p.editValue = agent.Command
		p.editCur = len(p.editValue)
	case agentFieldLabel:
		p.editing = true
		p.editValue = agent.Label
		p.editCur = len(p.editValue)
	case agentFieldArgs:
		p.editing = true
		p.editValue = strings.Join(agent.PlanningArgs, " ")
		p.editCur = len(p.editValue)
	case agentFieldDefault:
		// Toggle default — ensure only one is default
		if !agent.Default {
			// Unset all others
			for k, a := range p.agents {
				a.Default = false
				p.agents[k] = a
			}
			agent.Default = true
			p.agents[agentKey] = agent
		}
	}
}

func (p *agentsPage) handleEditKey(key string, msg tea.KeyMsg) tea.Cmd {
	switch key {
	case "enter":
		p.commitEdit()
		p.editing = false
	case "esc":
		p.editing = false
	case "backspace":
		if p.editCur > 0 {
			p.editValue = p.editValue[:p.editCur-1] + p.editValue[p.editCur:]
			p.editCur--
		}
	case "delete":
		if p.editCur < len(p.editValue) {
			p.editValue = p.editValue[:p.editCur] + p.editValue[p.editCur+1:]
		}
	case "left":
		if p.editCur > 0 {
			p.editCur--
		}
	case "right":
		if p.editCur < len(p.editValue) {
			p.editCur++
		}
	case "home", "ctrl+a":
		p.editCur = 0
	case "end", "ctrl+e":
		p.editCur = len(p.editValue)
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			p.editValue = p.editValue[:p.editCur] + key + p.editValue[p.editCur:]
			p.editCur++
		} else if len(msg.Runes) > 0 {
			ch := string(msg.Runes)
			p.editValue = p.editValue[:p.editCur] + ch + p.editValue[p.editCur:]
			p.editCur += len(ch)
		}
	}
	return nil
}

func (p *agentsPage) commitEdit() {
	agentKey, agent := p.selectedAgent()
	if agentKey == "" {
		return
	}

	switch p.detailField {
	case agentFieldCommand:
		agent.Command = strings.TrimSpace(p.editValue)
	case agentFieldLabel:
		agent.Label = strings.TrimSpace(p.editValue)
	case agentFieldArgs:
		// Parse space-separated args
		raw := strings.TrimSpace(p.editValue)
		if raw == "" {
			agent.PlanningArgs = nil
		} else {
			agent.PlanningArgs = strings.Fields(raw)
		}
	}

	p.agents[agentKey] = agent
}

func (p *agentsPage) View(width, height int, theme *Theme) string {
	listWidth := width / 3
	if listWidth < 18 {
		listWidth = 18
	}
	if listWidth > 24 {
		listWidth = 24
	}
	detailWidth := width - listWidth - 2

	list := p.renderList(listWidth, height)
	detail := p.renderDetail(detailWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, list, detail)
}

func (p *agentsPage) renderList(listWidth, height int) string {
	var sb strings.Builder

	sb.WriteString(p.theme.Title.Render("Agents"))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", listWidth)))
	sb.WriteString("\n")

	for i, key := range p.agentKeys {
		agent := p.agents[key]
		indicator := " "
		if agent.Default {
			indicator = "\u2605" // ★
		}

		isSelected := i == p.listIdx
		if isSelected && p.focus == agentsFocusList {
			sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" \u25B8 %s %s", key, indicator)))
		} else if isSelected {
			sb.WriteString(p.theme.Normal.Render(fmt.Sprintf(" \u25B8 %s %s", key, indicator)))
		} else {
			sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s %s", key, indicator)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(" Edit config.toml\n to add/remove."))
	sb.WriteString("\n")

	// Fill
	contentHeight := height - 4
	rendered := sb.String()
	lineCount := strings.Count(rendered, "\n")
	for i := lineCount; i < contentHeight; i++ {
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(listWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(p.theme.Accent).
		Render(sb.String())
}

func (p *agentsPage) renderDetail(detailWidth, height int) string {
	var sb strings.Builder

	agentKey, agent := p.selectedAgent()
	if agentKey == "" {
		sb.WriteString(p.theme.Dimmed.Render("No agent selected"))
		return lipgloss.NewStyle().Width(detailWidth).PaddingLeft(1).Render(sb.String())
	}

	sb.WriteString(p.theme.Title.Render(agentKey))
	sb.WriteString("\n")
	sb.WriteString(p.theme.Dimmed.Render(safeRepeat("\u2500", detailWidth-2)))
	sb.WriteString("\n")

	type detailRow struct {
		label string
		value string
		field agentDetailField
	}

	rows := []detailRow{
		{"Command", agent.Command, agentFieldCommand},
		{"Label", agent.Label, agentFieldLabel},
		{"Planning Args", strings.Join(agent.PlanningArgs, " "), agentFieldArgs},
	}

	for _, row := range rows {
		sb.WriteString(p.theme.Dimmed.Render(row.label))
		sb.WriteString("\n")

		isSelected := p.focus == agentsFocusDetail && p.detailField == row.field

		if p.editing && isSelected {
			before := p.editValue[:p.editCur]
			after := p.editValue[p.editCur:]
			cursor := p.theme.Selected.Reverse(true).Render(" ")
			if p.editCur < len(p.editValue) {
				cursor = p.theme.Selected.Reverse(true).Render(string(p.editValue[p.editCur]))
				after = after[1:]
			}
			line := "  " + p.theme.Normal.Render(before) + cursor + p.theme.Normal.Render(after)
			sb.WriteString(line)
		} else if isSelected {
			display := row.value
			if display == "" {
				display = "(empty)"
			}
			sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" \u25B8 %s", display)))
		} else {
			display := row.value
			if display == "" {
				display = "(empty)"
			}
			sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s", display)))
		}
		sb.WriteString("\n\n")
	}

	// Default toggle
	sb.WriteString(p.theme.Dimmed.Render("Default Agent"))
	sb.WriteString("\n")

	isSelected := p.focus == agentsFocusDetail && p.detailField == agentFieldDefault
	defaultLabel := "No"
	defaultIndicator := "\u25CB" // ○
	if agent.Default {
		defaultLabel = "Yes"
		defaultIndicator = "\u2605" // ★
	}

	if isSelected {
		sb.WriteString(p.theme.Selected.Render(fmt.Sprintf(" %s %s", defaultIndicator, defaultLabel)))
	} else {
		sb.WriteString(p.theme.Normal.Render(fmt.Sprintf("   %s %s", defaultIndicator, defaultLabel)))
	}
	sb.WriteString("\n\n")

	// Hint
	sb.WriteString(p.theme.Dimmed.Render("\u2508 applies to new tabs \u2508"))
	sb.WriteString("\n")

	// Fill
	contentHeight := height - 4
	rendered := sb.String()
	lineCount := strings.Count(rendered, "\n")
	for i := lineCount; i < contentHeight; i++ {
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().
		Width(detailWidth).
		PaddingLeft(1).
		Render(sb.String())
}
