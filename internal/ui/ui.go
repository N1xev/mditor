package ui

import (
	"image/color"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/N1xev/mditor/internal/edit"
	"github.com/N1xev/mditor/internal/preview"
	"github.com/N1xev/mditor/internal/uict"
)

type editorMode int

const (
	modeEdit editorMode = iota
	modeMixed
	modeView
)

type focusTarget int

const (
	focusSidebar focusTarget = iota
	focusEditor
	focusPreview
)

func (m editorMode) String() string { return [...]string{"EDIT", "MIXED", "VIEW"}[m] }

type overlay int

const (
	overlayNone   overlay = iota
	overlayHelp
	overlayRename
)

const (
	zoneModeEdit  = "mode-edit"
	zoneModeM     = "mode-mixed"
	zoneModeView  = "mode-view"
	zoneTabPrefix = "tab-"
	zoneTabClose  = "tab-close-"
	zoneSave      = "btn-save"
	zoneNew       = "btn-new"
	zoneHelpBtn   = "btn-help"
	zoneQuit      = "btn-quit"
	zoneHelpClose = "help-close"
	zoneEditor    = "pane-editor"
	zonePreview   = "pane-preview"
	zoneSidebar   = "pane-sidebar"
)

const sidebarWidth = 28

var selHighlightStyle = lipgloss.NewStyle().
	Background(uict.Charple).
	Foreground(uict.Salt).
	Bold(true)

var sidebarActiveTitle = lipgloss.NewStyle().
	Foreground(uict.Salt).
	Bold(true).
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderLeftForeground(uict.Salt).
	PaddingLeft(1).
	MaxWidth(sidebarWidth - 4)
var sidebarActiveDesc = lipgloss.NewStyle().
	Foreground(uict.Salt).
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderLeftForeground(uict.Salt).
	PaddingLeft(1).
	MaxWidth(sidebarWidth - 4)

type Model struct {
	Mode        editorMode
	Focus       focusTarget
	Overlay     overlay
	Keys        keyMap
	St          styles
	Help        help.Model
	HelpSection int
	pre         preActions

	Width  int
	Height int

	Tabs   []edit.Tab
	Active int

	Vp           viewport.Model
	Renderer     *preview.Renderer
	Preview      string
	LastPreviewW int
	PreviewDirty bool
	ViewDirty    bool

	RenameInput textinput.Model

	Notif notification

	FileList    list.Model
	SidebarDele *sidebarDelegate
	ShowSidebar bool
	ShowEmpty   bool

	SelDragging  bool
	SelAnchorX   int
	SelAnchorY   int
	SelCursorX   int
	SelCursorY   int
	LastMotionNS int64
	SelLog       selRange
	SelDisp      selRange

	DispMap        []int
	DispMapValue   string
	DispMapWidth   int
	DispMapLineCnt int

	SelTextCache    string
	SelTextKeyLog   selRange
	SelTextKeyValue string

	PreviewDragging  bool
	PreviewSelStartX int
	PreviewSelStartY int
	PreviewSelEndX   int
	PreviewSelEndY   int

	CodeBlockContents []string
}

func NewModel(filenames []string) Model {
	km := defaultKeyMap()
	st := newStyles()
	hlp := help.New()

	var tabs []edit.Tab
	if len(filenames) > 0 {
		for _, f := range filenames {
			tabs = append(tabs, edit.NewTab(f))
		}
	}

	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3
	vp.FillHeight = true

	dlg := &sidebarDelegate{inner: list.NewDefaultDelegate()}
	dlg.inner.ShowDescription = true
	dlg.inner.SetHeight(2)
	dlg.inner.SetSpacing(1)
	maxItemW := sidebarWidth - 4
	dlg.inner.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(uict.Steam).
		PaddingLeft(2).
		MaxWidth(maxItemW)
	dlg.inner.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(uict.Squid).
		PaddingLeft(2).
		MaxWidth(maxItemW)
	dlg.inner.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(uict.Violet).
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderLeftForeground(uict.Violet).
		PaddingLeft(1).
		MaxWidth(maxItemW)
	dlg.inner.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(uict.Violet).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderLeftForeground(uict.Violet).
		PaddingLeft(1).
		MaxWidth(maxItemW)

	fl := list.New(nil, dlg, sidebarWidth, 20)
	fl.SetShowTitle(false)
	fl.SetShowStatusBar(false)
	fl.SetShowPagination(false)
	fl.Styles.ActivePaginationDot = lipgloss.NewStyle().
		Foreground(uict.Violet).
		Bold(true)
	fl.Styles.InactivePaginationDot = lipgloss.NewStyle().
		Foreground(uict.Squid)
	fl.Styles.PaginationStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(uict.Squid)
	fl.Styles.Filter.Focused.Prompt = lipgloss.NewStyle().Foreground(uict.Violet).Bold(true)
	fl.Styles.Filter.Focused.Text = lipgloss.NewStyle().Foreground(uict.Salt)
	fl.Styles.Filter.Blurred.Prompt = lipgloss.NewStyle().Foreground(uict.Violet)
	fl.Styles.Filter.Blurred.Text = lipgloss.NewStyle().Foreground(uict.Steam)
	fl.Styles.Filter.Cursor.Color = uict.Violet
	fl.FilterInput.Prompt = "/ "
	fl.FilterInput.Placeholder = "filter…"
	fl.FilterInput.CharLimit = 64
	fl.SetShowHelp(false)
	fl.SetShowFilter(true)
	fl.SetFilteringEnabled(true)

	ri := textinput.New()
	ri.Placeholder = "filename.md"
	ri.Prompt = ""
	ri.CharLimit = 128
	ri.SetValue("")

	m := Model{
		Mode:         modeEdit,
		Keys:         km,
		St:           st,
		Help:         hlp,
		Tabs:         tabs,
		Active:       0,
		Vp:           vp,
		RenameInput:  ri,
		PreviewDirty: true,

		FileList:    fl,
		SidebarDele: dlg,
		ShowSidebar: true,
		ShowEmpty:   len(filenames) == 0,
		pre: preActions{
			New:  zone.Mark(zoneNew, preActionNewRaw),
			Help: zone.Mark(zoneHelpBtn, preActionHelpRaw),
			Quit: zone.Mark(zoneQuit, preActionQuitRaw),
		},
	}

	return m
}

// scanSidebarMsg carries the result of an async sidebar scan.
type scanSidebarMsg struct {
	items []list.Item
	err   error
}

// scanSidebarAsync returns a tea.Cmd that walks the project tree off the
// main goroutine so large projects don't block init with "initializing…".
func scanSidebarAsync(descCells int) tea.Cmd {
	return func() tea.Msg {
		items, err := edit.ScanMarkdownFiles(descCells)
		return scanSidebarMsg{items: items, err: err}
	}
}

func notifFromScanErr(err error) notification {
	if err == nil {
		return notification{}
	}
	return newNotif("sidebar scan: "+err.Error(), true)
}

func (m *Model) Cur() *edit.Tab {
	if len(m.Tabs) == 0 {
		return nil
	}
	return &m.Tabs[m.Active]
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.notifTickOnce(),
		scanSidebarAsync(sidebarWidth - 6),
	}
	if m.Cur() != nil {
		cmds = append(cmds, m.Cur().TA.Focus())
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.relayout()
		m.refreshRenderer()
		m.ViewDirty = true
		m.PreviewDirty = true

	case notifExpiredMsg:
		if m.Notif.alive() {
			cmds = append(cmds, notifExpireCmd(remainingNotifTTL(m.Notif)))
		} else {
			m.Notif = notification{}
			cmds = append(cmds, notifExpireCmd(remainingNotifTTL(m.Notif)))
		}

	case tea.MouseClickMsg:
		cmd := m.handleMouseClick(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if msg.Button == tea.MouseLeft && m.Cur() != nil {
			if m.inPreviewPane(msg.X, msg.Y) {
				m.clearPreviewSelection()
				m.PreviewDragging = true
				m.PreviewSelStartX = msg.X
				m.PreviewSelStartY = msg.Y
				m.PreviewSelEndX = msg.X
				m.PreviewSelEndY = msg.Y
			} else if m.Mode != modeView {
				m.SelDragging = true
				m.SelAnchorX = msg.X
				m.SelAnchorY = msg.Y
				m.SelCursorX = msg.X
				m.SelCursorY = msg.Y
			}
		}

	case tea.MouseMotionMsg:
		if m.PreviewDragging {
			m.PreviewSelEndX = msg.X
			m.PreviewSelEndY = msg.Y
			now := time.Now().UnixNano()
			if now-m.LastMotionNS >= 16_000_000 {
				m.LastMotionNS = now
				m.ViewDirty = true
			}
			break
		}
		if m.SelDragging && m.Mode != modeView && m.Cur() != nil {
			m.SelCursorX = msg.X
			m.SelCursorY = msg.Y
			now := time.Now().UnixNano()
			if now-m.LastMotionNS >= 16_000_000 {
				m.LastMotionNS = now
				m.updateSelectionRange()
			}
		}

	case tea.MouseReleaseMsg:
		if m.PreviewDragging {
			m.PreviewDragging = false
			if !m.hasPreviewSelection() {
				m.clearPreviewSelection()
			}
			m.ViewDirty = true
			break
		}
		if m.SelDragging {
			m.SelDragging = false
			m.updateSelectionRange()
			if !m.hasEditorSelection() {
				m.clearEditorSelection()
			}
			m.ViewDirty = true
		}

	case tea.PasteMsg:
		if m.Cur() != nil && m.Focus == focusEditor && m.Mode != modeView {
			var taCmd tea.Cmd
			m.Cur().TA, taCmd = m.Cur().TA.Update(msg)
			m.markEditedAfterPaste()
			if m.Mode == modeMixed {
				m.syncEditorToPreview()
			}
			cmds = append(cmds, taCmd)
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			switch m.Focus {
			case focusSidebar:
				if m.ShowSidebar {
					m.FileList.CursorUp()
				}
			case focusEditor:
				if m.Cur() != nil && m.Mode != modeView {
					m.Cur().TA.PageUp()
				}
			default:
				m.Vp.ScrollUp(3)
			}
		case tea.MouseWheelDown:
			switch m.Focus {
			case focusSidebar:
				if m.ShowSidebar {
					m.FileList.CursorDown()
				}
			case focusEditor:
				if m.Cur() != nil && m.Mode != modeView {
					m.Cur().TA.PageDown()
				}
			default:
				m.Vp.ScrollDown(3)
			}
		}

	case tea.KeyPressMsg:
		cmd := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.KeyReleaseMsg:
		// Suppress key release events to prevent delayed repeat after
		// the user releases a held key. Without this, the terminal may
		// flush buffered repeat events after the release, causing a
		// visible scroll "tail" after the user's finger is up.
		return m, nil

	case list.FilterMatchesMsg:
		var listCmd tea.Cmd
		m.FileList, listCmd = m.FileList.Update(msg)
		if listCmd != nil {
			cmds = append(cmds, listCmd)
		}

	case scanSidebarMsg:
		m.Notif = notifFromScanErr(msg.err)
		if len(msg.items) == 0 {
			break
		}
		if listCmd := m.FileList.SetItems(msg.items); listCmd != nil {
			cmds = append(cmds, listCmd)
		}
		m.syncSidebarSelection()
		m.ViewDirty = true

	default:
		switch m.Focus {
		case focusEditor:
			if m.Cur() != nil && m.Mode != modeView {
				var taCmd tea.Cmd
				m.Cur().TA, taCmd = m.Cur().TA.Update(msg)
				if taCmd != nil {
					cmds = append(cmds, taCmd)
				}
			}
		case focusPreview:
			var vpCmd tea.Cmd
			m.Vp, vpCmd = m.Vp.Update(msg)
			if vpCmd != nil {
				cmds = append(cmds, vpCmd)
			}
		}
	}

	if m.PreviewDirty {
		if imgData, _ := m.refreshPreview(); len(imgData) > 0 {
			for _, d := range imgData {
				cmds = append(cmds, tea.Raw(d))
			}
		}
		m.PreviewDirty = false
	}
	m.ViewDirty = false

	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View { return m.view() }

func (m *Model) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	switch m.Overlay {
	case overlayHelp:
		switch {
		case key.Matches(msg, m.Keys.EscOverlay), key.Matches(msg, m.Keys.ToggleHelp):
			m.Overlay = overlayNone
		case msg.String() == "right", msg.String() == "l":
			m.HelpSection = (m.HelpSection + 1) % 4
		case msg.String() == "left", msg.String() == "h":
			m.HelpSection = (m.HelpSection + 3) % 4
		}
		return nil

	case overlayRename:
		switch {
		case key.Matches(msg, m.Keys.EscOverlay):
			m.Overlay = overlayNone
			m.RenameInput.Blur()
		case msg.String() == "enter":
			return m.commitRename()
		default:
			var cmd tea.Cmd
			m.RenameInput, cmd = m.RenameInput.Update(msg)
			return cmd
		}
	}

	switch {
	case key.Matches(msg, m.Keys.Quit):
		return tea.Quit
	case key.Matches(msg, m.Keys.CycleMode):
		m.cycleMode()
		return nil
	case key.Matches(msg, m.Keys.Save):
		return m.save()
	case key.Matches(msg, m.Keys.New):
		return m.openNewTab()
	case key.Matches(msg, m.Keys.Close):
		return m.closeActiveTab()
	case key.Matches(msg, m.Keys.NextTab):
		m.switchTab(m.Active + 1)
		return nil
	case key.Matches(msg, m.Keys.PrevTab):
		m.switchTab(m.Active - 1)
		return nil
	case key.Matches(msg, m.Keys.ToggleHelp):
		m.toggleHelp()
		return nil
	case key.Matches(msg, m.Keys.Rename):
		m.startRename()
		return nil
	case key.Matches(msg, m.Keys.ScrollUp):
		m.Vp.ScrollUp(3)
		if m.Mode == modeMixed {
			m.syncPreviewToEditor()
		}
		return nil
	case key.Matches(msg, m.Keys.ScrollDown):
		m.Vp.ScrollDown(3)
		if m.Mode == modeMixed {
			m.syncPreviewToEditor()
		}
		return nil
	case key.Matches(msg, m.Keys.ToggleSidebar):
		return m.toggleSidebar()
	case key.Matches(msg, m.Keys.Undo):
		m.undo()
		return nil
	case key.Matches(msg, m.Keys.Redo):
		m.redo()
		return nil
	case key.Matches(msg, m.Keys.CycleFocus):
		return m.cycleFocusCmd(1)
	case key.Matches(msg, m.Keys.CycleFocusBack):
		return m.cycleFocusCmd(-1)
	}

	return m.handleFocusedPaneKey(msg)
}

func (m *Model) handleFocusedPaneKey(msg tea.KeyPressMsg) tea.Cmd {
	if m.ShowSidebar && m.Focus == focusSidebar {
		var listCmd tea.Cmd
		m.FileList, listCmd = m.FileList.Update(msg)
		if msg.String() == "enter" {
			cmd := m.openSidebarSelected()
			if m.Mode == modeView {
				m.Focus = focusPreview
			} else {
				m.Focus = focusEditor
			}
			return cmd
		}
		return listCmd
	}
	if m.Cur() == nil {
		return nil
	}
	switch {
	case key.Matches(msg, m.Keys.Copy):
		if m.Focus == focusPreview && (m.Mode == modeMixed || m.Mode == modeView) {
			return m.copyPreview()
		}
		return m.copy()
	case key.Matches(msg, m.Keys.Cut):
		return m.cut()
	case key.Matches(msg, m.Keys.Paste):
		return m.paste()
	case key.Matches(msg, m.Keys.CopyAll):
		return m.copyAll()
	case key.Matches(msg, m.Keys.SelectAll):
		if m.Focus == focusEditor && m.Mode != modeView {
			m.selectAllEditor()
		}
		return nil
	case key.Matches(msg, m.Keys.Duplicate):
		return m.duplicate()
	}
	if m.Focus == focusPreview && (m.Mode == modeMixed || m.Mode == modeView) {
		var vpCmd tea.Cmd
		m.Vp, vpCmd = m.Vp.Update(msg)
		if m.Mode == modeMixed {
			m.syncPreviewToEditor()
		}
		return vpCmd
	}
	if m.Mode != modeView && m.Focus == focusEditor {
		return m.handleEditorKey(msg)
	}
	var vpCmd tea.Cmd
	m.Vp, vpCmd = m.Vp.Update(msg)
	return vpCmd
}

func (m *Model) commitRename() tea.Cmd {
	name := strings.TrimSpace(m.RenameInput.Value())
	m.Overlay = overlayNone
	m.RenameInput.Blur()
	if name == "" || m.Cur() == nil {
		return nil
	}
	if name == m.Cur().DisplayName() {
		return nil
	}
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}
	m.Cur().Filename = name
	m.Cur().Saved = false
	return m.setNotif("renamed → "+name, false)
}

func borderedTitle(title string, content string, width int, fg color.Color) string {
	if width < 6 {
		width = 6
	}
	b := lipgloss.RoundedBorder()
	borderStyle := lipgloss.NewStyle().Foreground(fg)
	innerW := width - 2

	titleStyle := lipgloss.NewStyle().Foreground(fg).Bold(true)
	titleText := titleStyle.Render("── " + title + " ")
	fillW := max(0, innerW-lipgloss.Width(titleText))
	fill := strings.Repeat(borderStyle.Render(b.Top), fillW)
	topLine := borderStyle.Render(b.TopLeft) +
		titleText +
		fill +
		borderStyle.Render(b.TopRight)

	contentLines := strings.Split(content, "\n")
	var body strings.Builder
	body.Grow((innerW + 2) * (len(contentLines) + 2))
	left := borderStyle.Render(b.Left)
	right := borderStyle.Render(b.Right)
	for _, line := range contentLines {
		clipped := ansiTruncateWc(line, innerW, "")
		pad := max(innerW-lipgloss.Width(clipped), 0)
		body.WriteString(left)
		body.WriteString(clipped)
		if pad > 0 {
			body.WriteString(strings.Repeat(" ", pad))
		}
		body.WriteString(right)
		body.WriteRune('\n')
	}

	bottomLine := borderStyle.Render(b.BottomLeft) +
		strings.Repeat(borderStyle.Render(b.Bottom), innerW) +
		borderStyle.Render(b.BottomRight)

	return topLine + "\n" + body.String() + bottomLine
}
