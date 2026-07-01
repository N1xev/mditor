package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"github.com/N1xev/mditor/internal/uict"
)

var (
	helpCornerStyle = lipgloss.NewStyle().Foreground(uict.Charple)
	helpTitleStyle  = lipgloss.NewStyle().Foreground(uict.Charple).Bold(true)
	helpTitleHead   = helpTitleStyle.Render("─" + " help ")
	helpConnector   = helpCornerStyle.Render(lipgloss.RoundedBorder().Top)

	helpSectionActive = lipgloss.NewStyle().
				Background(uict.Charple).
				Foreground(uict.Salt).
				Padding(0, 1).
				Bold(true)
	helpSectionInactive = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(uict.Charple)

	helpRowBaseStyle = lipgloss.NewStyle()
	helpHintStyle    = lipgloss.NewStyle().Foreground(uict.Iron)
	helpPadStyle     = lipgloss.NewStyle().Padding(1, 1)

	renameHintStyle = lipgloss.NewStyle().Foreground(uict.Squid)
)

func (m Model) renderHelpPanel() string {
	groups := []struct {
		heading  string
		bindings []key.Binding
	}{
		{"Navigation", []key.Binding{
			m.Keys.CycleFocus,
			m.Keys.CycleFocusBack,
			m.Keys.CycleMode,
			m.Keys.NextTab,
			m.Keys.PrevTab,
			m.Keys.ScrollUp,
			m.Keys.ScrollDown,
		}},
		{"Files", []key.Binding{
			m.Keys.ToggleSidebar,
			m.Keys.New,
			m.Keys.Save,
			m.Keys.Rename,
			m.Keys.Close,
		}},
		{"Edit", []key.Binding{
			m.Keys.Undo,
			m.Keys.Redo,
			m.Keys.Copy,
			m.Keys.Cut,
			m.Keys.Paste,
			m.Keys.SelectAll,
			m.Keys.Duplicate,
			m.Keys.CopyAll,
		}},
		{"App", []key.Binding{
			m.Keys.ToggleHelp,
			m.Keys.EscOverlay,
			m.Keys.Quit,
			key.NewBinding(key.WithKeys("left-click tab"), key.WithHelp("left-click tab", "switch to tab")),
			key.NewBinding(key.WithKeys("right-click tab"), key.WithHelp("right-click tab", "rename tab")),
			key.NewBinding(key.WithKeys("middle-click tab"), key.WithHelp("middle-click tab", "close tab")),
			key.NewBinding(key.WithKeys("click ✕"), key.WithHelp("click ✕", "close tab")),
			key.NewBinding(key.WithKeys("drag select"), key.WithHelp("drag select", "select text")),
		}},
	}

	boxW := max(20, min(60, m.Width-8))
	b := lipgloss.RoundedBorder()
	innerW := boxW - 2

	var sectionTabs strings.Builder
	sectionTabs.Grow(64)
	for i, g := range groups {
		var s lipgloss.Style
		if i == m.HelpSection {
			s = helpSectionActive
		} else {
			s = helpSectionInactive
		}
		if i > 0 {
			sectionTabs.WriteString(helpConnector)
		}
		sectionTabs.WriteString(s.Render(g.heading))
	}
	tabBar := sectionTabs.String()

	usedW := 1 +
		lipgloss.Width(helpTitleHead) +
		lipgloss.Width(helpConnector) +
		lipgloss.Width(tabBar) +
		lipgloss.Width(helpConnector) +
		1
	fillW := max(0, innerW-(usedW-2))
	fill := helpCornerStyle.Render(strings.Repeat(b.Top, fillW))

	topLine := helpCornerStyle.Render(b.TopLeft) +
		helpTitleHead +
		helpConnector +
		tabBar +
		helpConnector +
		fill +
		helpCornerStyle.Render(b.TopRight)

	bodyW := max(1, innerW-2)
	g := groups[m.HelpSection]
	contentLines := make([]string, 0, len(g.bindings)+2)
	for _, b := range g.bindings {
		keyStr := m.St.HelpKeyBind.Render(b.Help().Key)
		descStr := m.St.HelpKeyDesc.Render(b.Help().Desc)
		contentLines = append(contentLines, helpRowBaseStyle.Width(bodyW).Render(keyStr+descStr))
	}
	contentLines = append(contentLines, "")
	hint := "←/→ to switch · esc to close"
	contentLines = append(contentLines, helpHintStyle.Width(bodyW).Render(hint))

	body := strings.Join(contentLines, "\n")
	paddedBody := helpPadStyle.Width(max(1, innerW-2)).Render(body)
	panel := borderedTitle("", paddedBody, boxW, uict.Charple)
	panelLines := strings.SplitN(panel, "\n", 2)
	if len(panelLines) == 2 {
		panel = topLine + "\n" + panelLines[1]
	} else {
		panel = topLine + "\n" + panel
	}
	return panel
}

func (m Model) renderRenamePanel() string {
	prompt := m.St.RenamePrompt.Render("Rename file: ")
	input := m.RenameInput.View()
	hint := renameHintStyle.Render("enter to confirm · esc to cancel")
	content := prompt + input + "\n" + hint
	panelW := min(40, m.Width-4)
	return borderedTitle("rename", content, panelW, uict.Hazy)
}
