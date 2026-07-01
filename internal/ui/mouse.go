package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/N1xev/mditor/internal/edit"
	"github.com/N1xev/mditor/internal/preview"
)

func (m *Model) handleMouseClick(msg tea.MouseClickMsg) tea.Cmd {
	zoneHit := func(id string) bool { return zone.Get(id).InBounds(msg) }

	switch {
	case zoneHit(zoneModeEdit):
		m.setMode(modeEdit)
	case zoneHit(zoneModeM):
		m.setMode(modeMixed)
	case zoneHit(zoneModeView):
		m.setMode(modeView)

	case zoneHit(zoneSave):
		return m.save()
	case zoneHit(zoneNew):
		return m.openNewTab()
	case zoneHit(zoneHelpBtn):
		if m.Overlay == overlayHelp {
			m.Overlay = overlayNone
		} else {
			m.HelpSection = 0
			m.Overlay = overlayHelp
		}
	case zoneHit(zoneQuit):
		return tea.Quit

	case m.Overlay == overlayHelp && zoneHit(zoneHelpClose):
		m.Overlay = overlayNone

	case msg.Button == tea.MouseLeft && zoneHit(zoneEditor):
		if m.Cur() != nil {
			m.Focus = focusEditor
			if m.Mode != modeView {
				m.clearEditorSelection()
				m.moveCursorToClick(msg.X, msg.Y)
				return m.Cur().TA.Focus()
			}
		}
	case msg.Button == tea.MouseLeft && zoneHit(zonePreview):
		m.Focus = focusPreview
		if m.Cur() != nil && m.Mode != modeView {
			m.Cur().TA.Blur()
		}
	case msg.Button == tea.MouseLeft && m.ShowSidebar && zoneHit(zoneSidebar):
		m.Focus = focusSidebar
		if m.Cur() != nil && m.Mode != modeView {
			m.Cur().TA.Blur()
		}
		if items := m.FileList.VisibleItems(); len(items) > 0 {
			perItem := 2 + 1
			if perItem <= 0 {
				perItem = 3
			}
			rowInBody := msg.Y - m.contentOffsetY() - 1
			if rowInBody >= 0 {
				visibleIdx := rowInBody / perItem
				if visibleIdx < len(items) {
					globalIdx := m.FileList.Paginator.Page*m.FileList.Paginator.PerPage + visibleIdx
					items := m.FileList.Items()
					if globalIdx >= 0 && globalIdx < len(items) {
						if sf, ok := items[globalIdx].(edit.SideFile); ok {
							return m.openFileInTab(sf.Path)
						}
					}
				}
			}
		}

	default:
		for blockID := range m.CodeBlockContents {
			copyID := preview.CopyZoneID(blockID)
			if zoneHit(copyID) {
				if err := clipboardWriteAll(m.CodeBlockContents[blockID]); err == nil {
					return m.setNotif("copied code block", false)
				}
				return nil
			}
		}

		for i := range m.Tabs {
			tabID := fmt.Sprintf("%s%d", zoneTabPrefix, i)
			closeID := fmt.Sprintf("%s%d", zoneTabClose, i)

			if zoneHit(closeID) {
				m.Active = i
				return m.closeActiveTab()
			}
			if zoneHit(tabID) {
				switch msg.Button {
				case tea.MouseLeft:
					return m.switchTab(i)
				case tea.MouseRight:
					m.switchTab(i)
					m.startRename()
					return nil
				case tea.MouseMiddle:
					m.Active = i
					return m.closeActiveTab()
				}
			}
		}
	}
	return nil
}
