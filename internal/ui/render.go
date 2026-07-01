package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/N1xev/mditor/internal/edit"
	"github.com/N1xev/mditor/internal/uict"
)

func (m Model) view() tea.View {
	if m.Width == 0 {
		v := tea.NewView("initializing…")
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		v.BackgroundColor = uict.Pepper
		return v
	}

	var sb strings.Builder
	sb.Grow(m.Width * m.Height)

	sb.WriteString(m.renderHeader())
	sb.WriteRune('\n')

	var sideBordered string
	if m.ShowSidebar {
		var activePath string
		if m.Cur() != nil && m.Cur().Filename != "" {
			if abs, err := filepath.Abs(m.Cur().Filename); err == nil {
				activePath = abs
			} else {
				activePath = m.Cur().Filename
			}
		}
		if m.SidebarDele != nil {
			m.SidebarDele.setActive(activePath, m.Focus == focusSidebar)
		}
		sideView := m.FileList.View()
		sideBorderColor := uict.Char
		if m.Focus == focusSidebar {
			sideBorderColor = uict.Violet
		}
		sideView = sideView + "\n" + renderSidebarFooter(sidebarWidth-2)
		sideBordered = zone.Mark(zoneSidebar, borderedTitle("sidebar", sideView, sidebarWidth, sideBorderColor))
	}

	var main string
	if len(m.Tabs) == 0 && m.ShowEmpty {
		main = renderEmptyState(m.contentWidth(), m.Height-3)
	} else {
		main = m.renderModePane()
	}
	if sideBordered != "" {
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sideBordered, main))
	} else {
		sb.WriteString(main)
	}

	sb.WriteRune('\n')
	sb.WriteString(m.renderStatusBar())

	content := sb.String()
	if m.Overlay != overlayNone {
		content = m.renderOverlay(content)
	}
	content = zone.Scan(content)

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.BackgroundColor = uict.Pepper
	v.WindowTitle = "MDitor"
	if len(m.Tabs) > 0 {
		v.WindowTitle = "mditor – " + m.Cur().DisplayName()
	}
	return v
}

func renderEmptyState(w, availH int) string {
	emptyMsg := lipgloss.NewStyle().
		Foreground(uict.Squid).
		Bold(true).
		Render("No file opened")
	hint := lipgloss.NewStyle().
		Foreground(uict.Iron).
		Render("Select a file from the sidebar or press ctrl+n for a new tab")
	content := lipgloss.JoinVertical(lipgloss.Center, emptyMsg, "", hint)
	contentH := lipgloss.Height(content)
	topPad := max((availH-contentH)/2, 0)
	return lipgloss.NewStyle().
		Width(w).
		PaddingTop(topPad).
		Align(lipgloss.Center).
		Render(content)
}

func (m Model) renderModePane() string {
	switch m.Mode {
	case modeEdit:
		return m.renderEditPane()
	case modeMixed:
		return m.renderMixedPane()
	case modeView:
		return m.renderViewPane()
	default:
		return ""
	}
}

func (m Model) renderHeader() string {
	staticHead := m.St.Brand.Render("MDitor") + m.St.SlashSep.Render("///////////////////")

	var tabsRow strings.Builder
	tabsRow.WriteString(staticHead)
	for i, t := range m.Tabs {
		tabID := fmt.Sprintf("%s%d", zoneTabPrefix, i)
		closeID := fmt.Sprintf("%s%d", zoneTabClose, i)
		name := t.DisplayName()
		tabStyle := m.St.TabInactive
		closeStyle := m.St.TabCloseInactive
		dirtyStyle := m.St.TabDirtyInactive
		if i == m.Active {
			tabStyle = m.St.TabActive
			closeStyle = m.St.TabCloseActive
			dirtyStyle = m.St.TabDirtyActive
		}
		dirtyMark := ""
		if !t.Saved {
			dirtyMark = dirtyStyle.Render("●")
		}
		closeBtn := zone.Mark(closeID, closeStyle.Render(" ✕"))
		label := name + dirtyMark + closeBtn
		rendered := zone.Mark(tabID, tabStyle.Render(label))
		tabsRow.WriteString(rendered)
	}
	tabLine := m.St.HeaderBar.Width(m.Width).Render(tabsRow.String())
	modeRow := m.renderModeBar()
	return tabLine + "\n" + modeRow
}

var (
	preStyleSpacer = lipgloss.NewStyle().UnsetBackground().Render(" ")

	preActionNewRaw  = lipgloss.NewStyle().Render(" NEW ")
	preActionHelpRaw = lipgloss.NewStyle().Render(" HELP ")
	preActionQuitRaw = lipgloss.NewStyle().Render(" QUIT ")
)

type preActions struct {
	New  string
	Help string
	Quit string
}

func (m Model) renderModeBar() string {
	modes := []struct {
		id     string
		label  string
		mode   editorMode
		active lipgloss.Style
	}{
		{zoneModeEdit, " EDIT ", modeEdit, m.St.ModeBtnActiveEdit},
		{zoneModeM, " MIXED ", modeMixed, m.St.ModeBtnActiveMixed},
		{zoneModeView, " VIEW ", modeView, m.St.ModeBtnActiveView},
	}

	var modeStr strings.Builder
	for _, b := range modes {
		var s lipgloss.Style
		if m.Mode == b.mode {
			s = b.active
		} else {
			s = m.St.ModeBtnInactive
		}
		modeStr.WriteString(zone.Mark(b.id, s.Render(b.label)))
	}

	saveLabel := "SAVE"
	if m.Cur() != nil && !m.Cur().Saved {
		saveLabel = "● SAVE"
	}
	saveBtn := zone.Mark(zoneSave, m.St.ActionBtnHot.Render(" "+saveLabel+" "))
	actions := saveBtn + preStyleSpacer + m.pre.New + preStyleSpacer + m.pre.Help + preStyleSpacer + m.pre.Quit

	notifStr := ""
	if m.Notif.alive() {
		if m.Notif.IsErr {
			notifStr = m.St.NotifErr.Render(m.Notif.Text)
		} else {
			notifStr = m.St.NotifOk.Render(m.Notif.Text)
		}
	}

	modesW := lipgloss.Width(modeStr.String())
	actionsW := lipgloss.Width(actions)
	notifW := lipgloss.Width(notifStr)
	midGapW := m.Width - modesW - actionsW - notifW
	if midGapW < 0 {
		keep := max(m.Width-modesW-actionsW, 1)
		notifStr = ansiTruncateWc(notifStr, keep, "…")
		notifW = lipgloss.Width(notifStr)
		midGapW = max(m.Width-modesW-actionsW-notifW, 0)
	}

	full := modeStr.String() + strings.Repeat(" ", midGapW) + notifStr + actions
	return lipgloss.NewStyle().Width(m.Width).Render(full)
}

func renderSidebarFooter(innerW int) string {
	if innerW < 1 {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(uict.Violet).
		Bold(true).
		Render(strings.Repeat("/", innerW))
}

func (m *Model) renderEditPane() string {
	if m.Cur() == nil {
		return ""
	}
	taView := m.cachedTAView()
	if m.hasEditorSelection() {
		taView = m.renderWithSelection(taView)
	}
	editBorderColor := uict.Char
	if m.Focus == focusEditor {
		editBorderColor = uict.Violet
	}
	return zone.Mark(zoneEditor, borderedTitle("editor", taView, m.contentWidth(), editBorderColor))
}

func (m Model) renderViewPane() string {
	preBorderColor := uict.Char
	if m.Focus == focusPreview {
		preBorderColor = uict.Violet
	}
	view := m.Vp.View()
	view = m.renderPreviewWithSelection(view)
	return zone.Mark(zonePreview, borderedTitle("preview", view, m.contentWidth(), preBorderColor))
}

func (m Model) renderMixedPane() string {
	if m.Cur() == nil {
		return ""
	}
	half := m.contentWidth() / 2
	editBorderColor := uict.Char
	if m.Focus == focusEditor {
		editBorderColor = uict.Violet
	}
	preBorderColor := uict.Char
	if m.Focus == focusPreview {
		preBorderColor = uict.Violet
	}
	editView := m.renderWithSelection(m.cachedTAView())
	previewView := m.renderPreviewWithSelection(m.Vp.View())
	left := zone.Mark(zoneEditor, borderedTitle("editor", editView, half, editBorderColor))
	right := zone.Mark(zonePreview, borderedTitle("preview", previewView, half, preBorderColor))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) cwdPill() string {
	dir := edit.FindProjectRoot()
	base := dir
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	if base == "" || base == "." {
		base = "CWD"
	}
	return lipgloss.NewStyle().
		Background(uict.Charple).
		Foreground(uict.Salt).
		Bold(true).
		Padding(0, 1).
		Render(base)
}

var (
	statusHint    string
	statusNoFile  string
	statusNoFileW int
	statusHintW   int
)

func init() {
	statusHint = "f1 help  ctrl+q quit"
	statusNoFile = "no file open"
	statusHintW = len(statusHint)
	statusNoFileW = len(statusNoFile)
}

func (m Model) renderStatusBar() string {
	sk := m.St.StatKey
	sv := m.St.StatVal
	sep := m.St.StatSep.Render(" · ")
	hint := m.St.StatHint.Render(statusHint)
	if m.Cur() == nil {
		left := sk.Render(statusNoFile)
		gapW := max(m.Width-statusNoFileW-statusHintW-4, 0)
		return m.St.StatusBar.Width(m.Width).
			Render(left + strings.Repeat(" ", gapW) + hint)
	}
	t := m.Cur()
	line := t.TA.Line() + 1
	col := t.TA.Column() + 1
	words := t.Words
	chars := t.Chars
	pos := sk.Render("ln ") + sv.Render(fmt.Sprintf("%d", line)) + sk.Render(":") + sv.Render(fmt.Sprintf("%d", col))
	wc := sk.Render("words ") + sv.Render(fmt.Sprintf("%d", words))
	cc := sk.Render("chars ") + sv.Render(fmt.Sprintf("%d", chars))
	fname := t.DisplayName()
	if !t.Saved {
		fname += m.St.StatDirty.Render(" ●")
	}
	file := sk.Render("file ") + m.St.StatFile.Render(fname)
	var selectInfo string
	if m.hasEditorSelection() {
		if sel := m.currentSelectionText(); sel != "" {
			selLen := len([]rune(sel))
			selectInfo = sep + m.St.StatSel.Render(fmt.Sprintf("sel %d chars", selLen))
		}
	}
	cwd := m.cwdPill()
	left := cwd + sep + pos + sep + wc + sep + cc + sep + file + selectInfo
	gapW := max(m.Width-lipgloss.Width(left)-lipgloss.Width(hint)-4, 0)
	return m.St.StatusBar.Width(m.Width).
		Render(left + strings.Repeat(" ", gapW) + hint)
}

func (m Model) renderOverlay(base string) string {
	var panel string
	switch m.Overlay {
	case overlayHelp:
		panel = m.renderHelpPanel()
	case overlayRename:
		panel = m.renderRenamePanel()
	default:
		return base
	}
	panelW := lipgloss.Width(panel)
	panelLines := strings.Split(panel, "\n")
	panelH := len(panelLines)
	x := (m.Width - panelW) / 2
	y := (m.Height - panelH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	baseLayer := lipgloss.NewLayer(base).Z(0)
	overlayLayer := lipgloss.NewLayer(panel).X(x).Y(y).Z(1)
	comp := lipgloss.NewCompositor(baseLayer, overlayLayer)
	return comp.Render()
}
