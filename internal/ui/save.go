package ui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"

	"github.com/N1xev/mditor/internal/edit"
)

func (m *Model) save() tea.Cmd {
	t := m.Cur()
	if t == nil {
		return nil
	}
	fname := t.Filename
	if fname == "" {
		fname = "untitled.md"
		t.Filename = fname
	}
	base, err := edit.SaveFile(fname, t.TA.Value())
	if err != nil {
		return m.setErrNotif("✗ " + err.Error())
	}
	t.Saved = true
	return m.setNotif("✓ saved → "+base, false)
}

func (m *Model) copy() tea.Cmd {
	if m.Cur() == nil {
		return nil
	}
	if m.hasEditorSelection() {
		return m.clipboardWrite(m.currentSelectionText(), "copied")
	}
	return m.copyLine()
}

func (m *Model) copyPreview() tea.Cmd {
	text := m.previewSelectedText()
	if text == "" {
		return nil
	}
	return m.clipboardWrite(text, "copied")
}

func (m *Model) copyLine() tea.Cmd {
	if m.Cur() == nil {
		return nil
	}
	lines := strings.Split(m.Cur().TA.Value(), "\n")
	line := m.Cur().TA.Line()
	if line < len(lines) {
		return m.clipboardWrite(lines[line], "copied line")
	}
	return nil
}

func (m *Model) copyAll() tea.Cmd {
	if m.Cur() == nil {
		return nil
	}
	return m.clipboardWrite(m.Cur().TA.Value(), "copied all")
}

func (m *Model) cut() tea.Cmd {
	if m.Cur() == nil || m.Mode == modeView || !m.hasEditorSelection() {
		return nil
	}
	text := m.currentSelectionText()
	m.recordUndoBeforeEdit()
	m.applyDeleteSelection()
	m.clearEditorSelection()
	m.Cur().Saved = false
	m.ViewDirty = true
	m.PreviewDirty = true
	return m.clipboardWrite(text, "cut")
}

func (m *Model) paste() tea.Cmd {
	if m.Cur() == nil || m.Mode == modeView {
		return nil
	}
	if m.hasEditorSelection() {
		m.applyDeleteSelection()
		m.clearEditorSelection()
	}
	m.recordUndoBeforeEdit()
	return textarea.Paste
}

func (m *Model) duplicate() tea.Cmd {
	if m.Cur() == nil || m.Mode == modeView {
		return nil
	}
	t := m.Cur()
	m.recordUndoBeforeEdit()
	var val string
	var line, col int
	if m.hasEditorSelection() {
		val, line, col = edit.DuplicateRange(t.TA.Value(), m.SelLog.toTextRange())
	} else {
		line = t.TA.Line()
		val = edit.DuplicateLine(t.TA.Value(), line)
		col = t.TA.Column()
		line++
	}
	t.TA.SetValue(val)
	t.RestoreCursor(line, col)
	t.Saved = false
	t.SyncBaseline()
	m.clearEditorSelection()
	m.ViewDirty = true
	m.PreviewDirty = true
	return m.setNotif("duplicated", false)
}

func (m *Model) clipboardWrite(text, label string) tea.Cmd {
	if text == "" {
		return nil
	}
	if err := clipboardWriteAll(text); err == nil {
		return m.setNotif(label, false)
	}
	return nil
}

func (m *Model) recordUndoBeforeEdit() {
	if t := m.Cur(); t != nil {
		t.PushUndo()
	}
}

func (m *Model) markEditedAfterPaste() {
	if t := m.Cur(); t != nil {
		t.Saved = false
		t.SyncBaseline()
	}
	m.invalidateSelectionText()
	m.ViewDirty = true
	m.PreviewDirty = true
}

func (m *Model) moveCursorToClick(x, y int) {
	if m.Cur() == nil || m.Mode == modeView {
		return
	}
	ta := &m.Cur().TA
	visibleRow := max(0, y-m.contentOffsetY())
	dispRow := ta.ScrollYOffset() + visibleRow
	logicalLine := logicalLineAtDisplayRow(*ta, dispRow)
	totalLines := ta.LineCount()
	if logicalLine >= totalLines {
		logicalLine = totalLines - 1
	}
	if logicalLine < 0 {
		logicalLine = 0
	}
	col := max(0, x-m.textStartX())
	if val := ta.Value(); val != "" {
		lines := strings.Split(val, "\n")
		if logicalLine < len(lines) {
			col = min(col, len(lines[logicalLine]))
		}
	}
	startLine := ta.Line()
	delta := logicalLine - startLine
	if delta > 0 {
		for i := 0; i < delta && ta.Line() < ta.LineCount()-1; i++ {
			ta.CursorDown()
		}
	} else if delta < 0 {
		for i := 0; i < -delta && ta.Line() > 0; i++ {
			ta.CursorUp()
		}
	}
	ta.SetCursorColumn(col)
}

func logicalLineAtDisplayRow(ta textarea.Model, row int) int {
	if row <= 0 {
		return 0
	}
	val := ta.Value()
	if val == "" {
		return 0
	}
	w := ta.Width()
	if w <= 0 {
		logical := strings.Split(val, "\n")
		if row >= len(logical) {
			return len(logical) - 1
		}
		return row
	}
	logical := strings.Split(val, "\n")
	dispRow := 0
	for i, line := range logical {
		wrapped := wrapForCols([]rune(line), w)
		h := len(wrapped)
		if dispRow+h > row {
			return i
		}
		dispRow += h
	}
	return len(logical) - 1
}
