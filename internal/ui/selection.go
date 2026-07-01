package ui

import (
	"strings"

	"charm.land/bubbles/v2/textarea"

	"github.com/N1xev/mditor/internal/edit"
	"github.com/N1xev/mditor/internal/preview"
)

type selRange struct {
	startLine int
	startCol  int
	endLine   int
	endCol    int
}

func (r selRange) toTextRange() edit.TextRange {
	return edit.TextRange{
		StartLine: r.startLine,
		StartCol:  r.startCol,
		EndLine:   r.endLine,
		EndCol:    r.endCol,
	}
}

func (m *Model) clearEditorSelection() {
	m.SelLog = selRange{}
	m.SelDisp = selRange{}
	m.invalidateSelectionText()
}

func (m Model) hasEditorSelection() bool {
	return !m.SelLog.toTextRange().Empty()
}

func (m *Model) clearPreviewSelection() {
	m.PreviewSelStartX, m.PreviewSelStartY = 0, 0
	m.PreviewSelEndX, m.PreviewSelEndY = 0, 0
}

func (m Model) hasPreviewSelection() bool {
	return m.previewSelectedText() != ""
}

func (m *Model) updateSelectionRange() {
	if m.Cur() == nil {
		return
	}
	displayRows := m.getDispMap()
	logicalLines := strings.Split(m.Cur().TA.Value(), "\n")
	if len(logicalLines) == 0 {
		return
	}
	displayCount := len(displayRows)
	if displayCount == 0 {
		return
	}
	offset := m.contentOffsetY()
	editorScrollY := m.Cur().TA.ScrollYOffset()
	textStartX := m.textStartX()
	startX := m.SelAnchorX
	startY := m.SelAnchorY
	endX := m.SelCursorX
	endY := m.SelCursorY
	if startY > endY || (startY == endY && startX > endX) {
		startX, endX = endX, startX
		startY, endY = endY, startY
	}
	startDisplay := startY - offset + editorScrollY
	endDisplay := endY - offset + editorScrollY
	if startDisplay > endDisplay {
		startDisplay, endDisplay = endDisplay, startDisplay
	}
	if startDisplay < 0 {
		startDisplay = 0
	}
	if startDisplay >= displayCount {
		startDisplay = displayCount - 1
	}
	if endDisplay < 0 {
		endDisplay = 0
	}
	if endDisplay >= displayCount {
		endDisplay = displayCount - 1
	}
	startLog := 0
	endLog := 0
	if startDisplay < displayCount {
		startLog = displayRows[startDisplay]
	}
	if endDisplay < displayCount {
		endLog = displayRows[endDisplay]
	}
	startCol := max(0, startX-textStartX)
	endCol := max(0, endX-textStartX)
	if startLog < len(logicalLines) {
		startCol = min(startCol, len(logicalLines[startLog]))
	}
	if endLog < len(logicalLines) {
		endCol = min(endCol, len(logicalLines[endLog]))
	}
	if startLog == endLog && startCol > endCol {
		startCol, endCol = endCol, startCol
	}
	oldLog := m.SelLog
	m.SelLog = selRange{startLine: startLog, startCol: startCol, endLine: endLog, endCol: endCol}
	m.SelDisp = selRange{startLine: startDisplay, endLine: endDisplay}
	if oldLog != m.SelLog {
		m.invalidateSelectionText()
	}
	m.ViewDirty = true
}

func (m *Model) selectAllEditor() {
	if m.Cur() == nil {
		return
	}
	val := m.Cur().TA.Value()
	if val == "" {
		m.clearEditorSelection()
		return
	}
	lines := strings.Split(val, "\n")
	last := len(lines) - 1
	m.SelLog = selRange{startLine: 0, endLine: last, endCol: len(lines[last])}
	m.SelDisp = displayRangeForLogical(m, m.SelLog)
	m.invalidateSelectionText()
	m.ViewDirty = true
}

func displayRangeForLogical(m *Model, log selRange) selRange {
	mapRows := m.getDispMap()
	if len(mapRows) == 0 {
		return selRange{}
	}
	startDisp, endDisp := 0, len(mapRows)-1
	for i, L := range mapRows {
		if L == log.startLine {
			startDisp = i
			break
		}
	}
	for i := len(mapRows) - 1; i >= 0; i-- {
		if mapRows[i] == log.endLine {
			endDisp = i
			break
		}
	}
	return selRange{startLine: startDisp, endLine: endDisp}
}

func (m *Model) currentSelectionText() string {
	if m.Cur() == nil {
		return ""
	}
	val := m.Cur().TA.Value()
	if m.SelTextCache != "" || m.hasEditorSelection() {
		if m.SelTextKeyLog == m.SelLog && m.SelTextKeyValue == val {
			return m.SelTextCache
		}
	}
	text := edit.Extract(val, m.SelLog.toTextRange())
	m.SelTextCache = text
	m.SelTextKeyLog = m.SelLog
	m.SelTextKeyValue = val
	return text
}

func (m *Model) invalidateSelectionText() {
	m.SelTextCache = ""
	m.SelTextKeyLog = selRange{}
	m.SelTextKeyValue = ""
}

func (m *Model) cachedTAView() string {
	t := m.Cur()
	ta := t.TA
	val := ta.Value()
	line := ta.Line()
	col := ta.Column()
	w := ta.Width()
	mh := ta.MaxHeight
	focused := ta.Focused()
	if t.TAViewCache != "" &&
		t.TAViewKeyValue == val &&
		t.TAViewKeyLine == line &&
		t.TAViewKeyCol == col &&
		t.TAViewKeyWidth == w &&
		t.TAViewKeyMaxH == mh &&
		t.TAViewKeyFocused == focused {
		return t.TAViewCache
	}
	view := ta.View()
	t.TAViewCache = view
	t.TAViewKeyValue = val
	t.TAViewKeyLine = line
	t.TAViewKeyCol = col
	t.TAViewKeyWidth = w
	t.TAViewKeyMaxH = mh
	t.TAViewKeyFocused = focused
	return view
}

func (m *Model) invalidateTAView() {
	if m.Cur() == nil {
		return
	}
	t := m.Cur()
	t.TAViewCache = ""
	t.TAViewKeyValue = ""
	t.TAViewKeyLine = -1
	t.TAViewKeyCol = -1
	t.TAViewKeyWidth = 0
	t.TAViewKeyMaxH = 0
	t.TAViewKeyFocused = false
}

func (m *Model) applyDeleteSelection() {
	if m.Cur() == nil || !m.hasEditorSelection() {
		return
	}
	val, line, col := edit.Delete(m.Cur().TA.Value(), m.SelLog.toTextRange())
	m.Cur().TA.SetValue(val)
	m.Cur().RestoreCursor(line, col)
	m.Cur().SyncBaseline()
	m.invalidateSelectionText()
}

func (m Model) previewSelectedText() string { return m.computePreviewSelectedText() }

func (m Model) computePreviewSelectedText() string {
	if m.Cur() == nil {
		return ""
	}
	paneTopY := m.contentOffsetY() + 1
	paneX := m.previewPaneStartX()
	yOff := m.Vp.YOffset()
	startRow := max(0, m.PreviewSelStartY-paneTopY) + yOff
	endRow := max(0, m.PreviewSelEndY-paneTopY) + yOff
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}
	startCol := max(0, m.PreviewSelStartX-paneX)
	endCol := max(0, m.PreviewSelEndX-paneX)
	if startCol > endCol {
		startCol, endCol = endCol, startCol
	}

	lines := strings.Split(m.Preview, "\n")
	if startRow >= len(lines) {
		return ""
	}

	if startRow == endRow {
		visible := ansiStrip(lines[startRow])
		runes := []rune(visible)
		rStart := preview.CellToRune(visible, startCol)
		rEnd := preview.CellToRune(visible, endCol)
		if rStart > rEnd {
			rStart, rEnd = rEnd, rStart
		}
		rEnd = min(rEnd, len(runes))
		if rStart >= rEnd {
			return ""
		}
		return string(runes[rStart:rEnd])
	}

	var parts []string
	first := ansiStrip(lines[startRow])
	firstRunes := []rune(first)
	rStart := preview.CellToRune(first, startCol)
	if rStart < len(firstRunes) {
		parts = append(parts, string(firstRunes[rStart:]))
	}
	for i := startRow + 1; i < endRow && i < len(lines); i++ {
		parts = append(parts, ansiStrip(lines[i]))
	}
	if endRow < len(lines) {
		last := ansiStrip(lines[endRow])
		lastRunes := []rune(last)
		rEnd := preview.CellToRune(last, endCol)
		rEnd = min(rEnd, len(lastRunes))
		parts = append(parts, string(lastRunes[:rEnd]))
	}
	return strings.Join(parts, "\n")
}

func buildDisplayLineMap(ta textarea.Model) []int {
	val := ta.Value()
	logical := strings.Split(val, "\n")
	w := ta.Width()
	if w <= 0 {
		out := make([]int, len(logical))
		for i := range out {
			out[i] = i
		}
		return out
	}
	var out []int
	for i, line := range logical {
		wrapped := wrapForCols([]rune(line), w)
		for range wrapped {
			out = append(out, i)
		}
	}
	return out
}

func (m *Model) getDispMap() []int {
	ta := m.Cur().TA
	val := ta.Value()
	w := ta.Width()
	lc := ta.LineCount()
	if m.DispMap != nil && m.DispMapValue == val && m.DispMapWidth == w && m.DispMapLineCnt == lc {
		return m.DispMap
	}
	m.DispMap = buildDisplayLineMap(ta)
	m.DispMapValue = val
	m.DispMapWidth = w
	m.DispMapLineCnt = lc
	return m.DispMap
}

func wrapForCols(runes []rune, width int) [][]rune {
	if len(runes) == 0 {
		return [][]rune{{}}
	}
	var lines [][]rune
	var cur []rune
	col := 0
	word := []rune{}
	flushWord := func() {
		if col+len(word) > width && col > 0 {
			lines = append(lines, cur)
			cur = nil
			col = 0
		}
		cur = append(cur, word...)
		col += len(word)
		word = nil
	}
	for _, r := range runes {
		if r == ' ' || r == '\t' {
			flushWord()
			if col < width {
				cur = append(cur, r)
				col++
			} else if col > 0 {
				lines = append(lines, cur)
				cur = nil
				col = 0
			}
			continue
		}
		word = append(word, r)
		if len(word) >= width {
			flushWord()
		}
	}
	flushWord()
	if cur == nil {
		cur = []rune{}
	}
	lines = append(lines, cur)
	return lines
}

func clampCol(c, hi int) int {
	return max(0, min(c, hi))
}

func (m *Model) renderWithSelection(content string) string {
	if m.Cur() == nil || !m.hasEditorSelection() {
		return content
	}
	ta := m.Cur().TA
	gutterCells := 1 + m.numDigitsForGutter() + 2

	displayRows := m.getDispMap()
	if len(displayRows) == 0 {
		return content
	}
	yOff := ta.ScrollYOffset()
	lines := strings.Split(content, "\n")
	startLine := m.SelLog.startLine
	endLine := m.SelLog.endLine
	if startLine > endLine {
		startLine, endLine = endLine, startLine
	}
	dispStart, dispEnd := -1, -1
	for vRow := range lines {
		fullRow := vRow + yOff
		if fullRow < 0 || fullRow >= len(displayRows) {
			continue
		}
		logical := displayRows[fullRow]
		if logical < startLine || logical > endLine {
			continue
		}
		if dispStart == -1 {
			dispStart = vRow
		}
		dispEnd = vRow
	}
	if dispStart == -1 {
		return content
	}

	var gutterStyled string
	for i := dispStart; i <= dispEnd && gutterStyled == ""; i++ {
		if i < 0 || i >= len(lines) {
			continue
		}
		if len(ansiStrip(lines[i])) > gutterCells {
			gutterStyled = ansiTruncateWc(lines[i], gutterCells, "")
		}
	}

	for displayIdx := dispStart; displayIdx <= dispEnd; displayIdx++ {
		if displayIdx < 0 || displayIdx >= len(lines) {
			continue
		}
		rowStyled := lines[displayIdx]
		visible := ansiStrip(rowStyled)
		if gutterCells >= len(visible) {
			continue
		}
		if gutterStyled == "" {
			gutterStyled = ansiTruncateWc(rowStyled, gutterCells, "")
		}
		restStyled := ansiTruncateLeftWc(rowStyled, gutterCells, "")

		textLen := len(visible) - gutterCells
		fullRow := displayIdx + yOff
		logical := -1
		if fullRow >= 0 && fullRow < len(displayRows) {
			logical = displayRows[fullRow]
		}

		startCol := 0
		endCol := textLen
		if logical == startLine {
			startCol = clampCol(m.SelLog.startCol, textLen)
		}
		if logical == endLine {
			endCol = clampCol(m.SelLog.endCol, textLen)
		}
		if logical == startLine && logical == endLine && startCol > endCol {
			startCol, endCol = endCol, startCol
		}
		if startCol >= endCol {
			continue
		}
		before := ansiTruncateWc(restStyled, startCol, "")
		middleStyled := ansiTruncateLeftWc(
			ansiTruncateWc(restStyled, endCol, ""),
			startCol, "")
		after := ansiTruncateLeftWc(restStyled, endCol, "")
		middleText := ansiStrip(middleStyled)
		highlighted := selHighlightStyle.Render(middleText)
		lines[displayIdx] = gutterStyled + before + highlighted + after
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderPreviewWithSelection(content string) string {
	if !m.hasPreviewSelection() && !m.PreviewDragging {
		return content
	}
	if m.PreviewSelStartY == m.PreviewSelEndY && m.PreviewSelStartX == m.PreviewSelEndX {
		return content
	}
	paneTopY := m.contentOffsetY() + 1
	yOff := m.Vp.YOffset()
	logicalStart := max(0, m.PreviewSelStartY-paneTopY) + yOff
	logicalEnd := max(0, m.PreviewSelEndY-paneTopY) + yOff
	if logicalStart > logicalEnd {
		logicalStart, logicalEnd = logicalEnd, logicalStart
	}
	startCol := max(0, m.PreviewSelStartX-m.previewPaneStartX())
	endCol := max(0, m.PreviewSelEndX-m.previewPaneStartX())
	visibleStart := logicalStart - yOff
	visibleEnd := logicalEnd - yOff

	lines := strings.Split(content, "\n")
	for i := range lines {
		if i < visibleStart || i > visibleEnd {
			continue
		}
		visible := ansiStrip(lines[i])
		rowStart := 0
		rowEnd := ansiStringWidth(visible)
		if i == visibleStart {
			rowStart = startCol
		}
		if i == visibleEnd {
			rowEnd = endCol
		}
		if i == visibleStart && i == visibleEnd && rowStart > rowEnd {
			rowStart, rowEnd = rowEnd, rowStart
		}
		if rowStart >= rowEnd {
			continue
		}
		before := ansiTruncateWc(lines[i], rowStart, "")
		mid := ansiStrip(ansiTruncateLeftWc(ansiTruncateWc(lines[i], rowEnd, ""), rowStart, ""))
		after := ansiTruncateLeftWc(lines[i], rowEnd, "")
		lines[i] = before + selHighlightStyle.Render(mid) + after
	}
	return strings.Join(lines, "\n")
}
