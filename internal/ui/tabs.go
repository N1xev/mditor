package ui

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/N1xev/mditor/internal/edit"
)

func (m *Model) setMode(md editorMode) {
	m.Mode = md
	m.clearEditorSelection()
	m.relayout()
	if m.Cur() != nil {
		if md == modeView {
			m.Cur().TA.Blur()
		} else {
			m.ViewDirty = true
		}
	}
	m.refreshRenderer()
	m.ViewDirty = true
	m.PreviewDirty = true
}

func (m *Model) syncSidebarSelection() {
	if !m.ShowSidebar || m.Cur() == nil {
		return
	}
	activePath := m.Cur().Filename
	if activePath == "" {
		return
	}
	activeBase := strings.TrimSuffix(filepath.Base(activePath), filepath.Ext(activePath))
	for i, item := range m.FileList.Items() {
		sf, ok := item.(edit.SideFile)
		if !ok {
			continue
		}
		if sf.Path == activePath || sf.Path == filepath.Clean(activePath) {
			m.FileList.Select(i)
			return
		}
		if sf.Name == activePath || strings.TrimSuffix(filepath.Base(sf.Path), filepath.Ext(sf.Path)) == activeBase {
			m.FileList.Select(i)
			return
		}
	}
}

func (m *Model) switchTab(idx int) tea.Cmd {
	if len(m.Tabs) == 0 {
		return nil
	}
	if idx < 0 {
		idx = len(m.Tabs) - 1
	}
	if idx >= len(m.Tabs) {
		idx = 0
	}
	if m.Cur() != nil {
		m.Cur().TA.Blur()
	}
	m.Active = idx
	m.relayout()
	if c := m.Cur(); c != nil {
		m.clearEditorSelection()
		c.SyncBaseline()
	}
	m.syncSidebarSelection()
	switch m.Mode {
	case modeView:
		m.Focus = focusPreview
	default:
		m.Focus = focusEditor
	}
	m.ViewDirty = true
	m.PreviewDirty = true
	if m.Mode != modeView && m.Cur() != nil {
		return m.Cur().TA.Focus()
	}
	return nil
}

func (m *Model) openNewTab() tea.Cmd {
	if m.Cur() != nil {
		m.Cur().TA.Blur()
	}
	m.Tabs = append(m.Tabs, edit.NewTab(""))
	m.Active = len(m.Tabs) - 1
	m.ShowEmpty = false
	m.relayout()
	if c := m.Cur(); c != nil {
		c.SyncBaseline()
	}
	m.ViewDirty = true
	m.PreviewDirty = true
	return m.Cur().TA.Focus()
}

func (m *Model) closeActiveTab() tea.Cmd {
	if len(m.Tabs) <= 1 {
		m.Tabs = nil
		m.Active = 0
		m.ShowSidebar = true
		m.ShowEmpty = true
		m.clearEditorSelection()
		m.relayout()
		return nil
	}
	m.Tabs = append(m.Tabs[:m.Active], m.Tabs[m.Active+1:]...)
	if m.Active >= len(m.Tabs) {
		m.Active = len(m.Tabs) - 1
	}
	m.relayout()
	if c := m.Cur(); c != nil {
		m.clearEditorSelection()
		c.SyncBaseline()
	}
	m.syncSidebarSelection()
	m.ViewDirty = true
	m.PreviewDirty = true
	return m.Cur().TA.Focus()
}

func (m *Model) startRename() {
	if m.Cur() == nil {
		return
	}
	m.RenameInput.SetValue(m.Cur().DisplayName())
	m.RenameInput.CursorEnd()
	m.RenameInput.Focus()
	m.Overlay = overlayRename
}

func (m *Model) cycleFocus(dir int) {
	var order []focusTarget
	switch m.Mode {
	case modeEdit:
		order = []focusTarget{focusSidebar, focusEditor}
	case modeMixed:
		order = []focusTarget{focusSidebar, focusEditor, focusPreview}
	case modeView:
		order = []focusTarget{focusSidebar, focusPreview}
	}
	if len(order) == 0 {
		return
	}
	idx := 0
	for i, f := range order {
		if f == m.Focus {
			idx = i
			break
		}
	}
	m.Focus = order[(idx+dir+len(order))%len(order)]
}

func (m *Model) refreshSidebarFiles() tea.Cmd {
	items, err := edit.ScanMarkdownFiles(sidebarWidth - 6)
	m.FileList.SetItems(items)
	if err != nil {
		return m.setErrNotif("sidebar scan: " + err.Error())
	}
	return nil
}

func (m *Model) openSidebarSelected() tea.Cmd {
	selected := m.FileList.SelectedItem()
	if selected == nil {
		return nil
	}
	if sf, ok := selected.(edit.SideFile); ok {
		return m.openFileInTab(sf.Path)
	}
	return nil
}

func (m *Model) openFileInTab(path string) tea.Cmd {
	for i := range m.Tabs {
		if m.Tabs[i].Filename == path {
			return m.switchTab(i)
		}
	}
	if m.Cur() != nil {
		m.Cur().TA.Blur()
	}
	m.Tabs = append(m.Tabs, edit.NewTab(path))
	m.Active = len(m.Tabs) - 1
	m.ShowEmpty = false
	m.relayout()
	if c := m.Cur(); c != nil {
		c.SyncBaseline()
	}
	m.syncSidebarSelection()
	m.ViewDirty = true
	m.PreviewDirty = true
	return m.Cur().TA.Focus()
}
