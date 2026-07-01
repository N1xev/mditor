package edit

type UndoState struct {
	Value string
	Line  int
	Col   int
}

func (t *Tab) SyncBaseline() {
	t.LastValue = t.TA.Value()
	t.LastLine = t.TA.Line()
	t.LastCol = t.TA.Column()
	t.SyncStats()
}

func (t *Tab) PushUndo() {
	t.UndoStack = append(t.UndoStack, UndoState{
		Value: t.LastValue,
		Line:  t.LastLine,
		Col:   t.LastCol,
	})
	t.RedoStack = nil
}

func (t *Tab) RestoreCursor(line, col int) {
	t.TA.MoveToBegin()
	for t.TA.Line() < line && t.TA.Line() < t.TA.LineCount()-1 {
		t.TA.CursorDown()
	}
	t.TA.SetCursorColumn(col)
}
