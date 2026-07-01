package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/N1xev/mditor/internal/edit"
)

func (m *Model) cycleMode() {
	if m.Cur() == nil {
		return
	}
	m.clearEditorSelection()
	m.Mode = (m.Mode + 1) % 3
	if m.Mode == modeView {
		m.Focus = focusPreview
		m.Cur().TA.Blur()
	} else {
		m.Focus = focusEditor
	}
	m.relayout()
	m.refreshRenderer()
	m.ViewDirty = true
	m.PreviewDirty = true
}

func (m *Model) toggleHelp() {
	if m.Overlay == overlayHelp {
		m.Overlay = overlayNone
	} else {
		m.HelpSection = 0
		m.Overlay = overlayHelp
	}
}

func (m *Model) toggleSidebar() tea.Cmd {
	m.ShowSidebar = !m.ShowSidebar
	var cmd tea.Cmd
	if m.ShowSidebar {
		cmd = m.refreshSidebarFiles()
	}
	m.relayout()
	m.refreshRenderer()
	m.ViewDirty = true
	m.PreviewDirty = true
	return cmd
}

func (m *Model) cycleFocusCmd(delta int) tea.Cmd {
	prev := m.Focus
	m.cycleFocus(delta)
	if prev == focusSidebar && m.Focus != focusSidebar {
		m.syncSidebarSelection()
	}
	if m.Focus == focusEditor && m.Mode != modeView && m.Cur() != nil {
		return m.Cur().TA.Focus()
	}
	if m.Cur() != nil && m.Focus != focusEditor {
		m.Cur().TA.Blur()
	}
	return nil
}

func (m *Model) undo() {
	t := m.Cur()
	if t == nil || len(t.UndoStack) == 0 {
		return
	}
	top := t.UndoStack[len(t.UndoStack)-1]
	t.UndoStack = t.UndoStack[:len(t.UndoStack)-1]
	t.RedoStack = append(t.RedoStack, edit.UndoState{
		Value: t.TA.Value(),
		Line:  t.TA.Line(),
		Col:   t.TA.Column(),
	})
	applySnap(t, top)
	m.clearEditorSelection()
	m.ViewDirty = true
	m.PreviewDirty = true
}

func (m *Model) redo() {
	t := m.Cur()
	if t == nil || len(t.RedoStack) == 0 {
		return
	}
	top := t.RedoStack[len(t.RedoStack)-1]
	t.RedoStack = t.RedoStack[:len(t.RedoStack)-1]
	t.UndoStack = append(t.UndoStack, edit.UndoState{
		Value: t.TA.Value(),
		Line:  t.TA.Line(),
		Col:   t.TA.Column(),
	})
	applySnap(t, top)
	m.clearEditorSelection()
	m.ViewDirty = true
	m.PreviewDirty = true
}

func applySnap(t *edit.Tab, s edit.UndoState) {
	t.TA.SetValue(s.Value)
	t.RestoreCursor(s.Line, s.Col)
	t.SyncBaseline()
}

func (m *Model) handleEditorKey(msg tea.KeyPressMsg) tea.Cmd {
	t := m.Cur()
	if t == nil || m.Mode == modeView {
		return nil
	}
	if m.hasEditorSelection() {
		switch {
		case keyReplacesSelection(msg):
			m.applyDeleteSelection()
		case keyClearsSelection(msg):
			m.clearEditorSelection()
		}
	}
	preVal, preLine, preCol := t.TA.Value(), t.TA.Line(), t.TA.Column()
	var taCmd tea.Cmd
	t.TA, taCmd = t.TA.Update(msg)
	if t.TA.Value() != preVal {
		t.UndoStack = append(t.UndoStack, edit.UndoState{Value: preVal, Line: preLine, Col: preCol})
		t.RedoStack = nil
		t.Saved = false
		m.invalidateSelectionText()
		m.ViewDirty = true
		m.PreviewDirty = true
	}
	t.SyncBaseline()
	if m.Mode == modeMixed {
		m.syncEditorToPreview()
	}
	return taCmd
}

func keyReplacesSelection(msg tea.KeyPressMsg) bool {
	if msg.String() == "enter" || msg.String() == "tab" {
		return true
	}
	return len(msg.Text) == 1
}

func keyClearsSelection(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "up", "down", "left", "right", "home", "end", "pgup", "pgdown",
		"ctrl+up", "ctrl+down", "ctrl+left", "ctrl+right":
		return true
	}
	return false
}
