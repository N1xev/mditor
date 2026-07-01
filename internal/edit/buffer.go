package edit

import "strings"

type TextRange struct {
	StartLine, StartCol int
	EndLine, EndCol     int
}

func (r TextRange) Normalize() TextRange {
	sl, sc, el, ec := r.StartLine, r.StartCol, r.EndLine, r.EndCol
	if sl > el || (sl == el && sc > ec) {
		sl, sc, el, ec = el, ec, sl, sc
	}
	return TextRange{StartLine: sl, StartCol: sc, EndLine: el, EndCol: ec}
}

func (r TextRange) Empty() bool {
	r = r.Normalize()
	return r.StartLine == r.EndLine && r.StartCol == r.EndCol
}

func Extract(value string, r TextRange) string {
	r = r.Normalize()
	if r.Empty() {
		return ""
	}
	lines := strings.Split(value, "\n")
	if r.StartLine >= len(lines) {
		return ""
	}
	if r.StartLine == r.EndLine {
		line := lines[r.StartLine]
		end := min(r.EndCol, len(line))
		start := min(r.StartCol, end)
		if start >= end {
			return ""
		}
		return line[start:end]
	}
	var parts []string
	start := min(r.StartCol, len(lines[r.StartLine]))
	parts = append(parts, lines[r.StartLine][start:])
	for i := r.StartLine + 1; i < r.EndLine && i < len(lines); i++ {
		parts = append(parts, lines[i])
	}
	if r.EndLine < len(lines) {
		end := min(r.EndCol, len(lines[r.EndLine]))
		parts = append(parts, lines[r.EndLine][:end])
	}
	return strings.Join(parts, "\n")
}

func Delete(value string, r TextRange) (newValue string, line, col int) {
	r = r.Normalize()
	if r.Empty() {
		return value, r.StartLine, r.StartCol
	}
	lines := strings.Split(value, "\n")
	if r.StartLine >= len(lines) {
		return value, 0, 0
	}
	if r.StartLine == r.EndLine {
		line := lines[r.StartLine]
		end := min(r.EndCol, len(line))
		start := min(r.StartCol, end)
		lines[r.StartLine] = line[:start] + line[end:]
		return strings.Join(lines, "\n"), r.StartLine, start
	}
	first := lines[r.StartLine][:min(r.StartCol, len(lines[r.StartLine]))]
	last := ""
	if r.EndLine < len(lines) {
		last = lines[r.EndLine][min(r.EndCol, len(lines[r.EndLine])):]
	}
	merged := first + last
	out := append(lines[:r.StartLine], merged)
	if r.EndLine+1 < len(lines) {
		out = append(out, lines[r.EndLine+1:]...)
	}
	return strings.Join(out, "\n"), r.StartLine, min(r.StartCol, len(merged))
}

func DuplicateLine(value string, line int) string {
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		return value
	}
	if line < 0 {
		line = 0
	}
	if line >= len(lines) {
		line = len(lines) - 1
	}
	dup := lines[line]
	out := append([]string{}, lines[:line+1]...)
	out = append(out, dup)
	out = append(out, lines[line+1:]...)
	return strings.Join(out, "\n")
}

func DuplicateRange(value string, r TextRange) (newValue string, line, col int) {
	r = r.Normalize()
	text := Extract(value, r)
	if text == "" {
		return value, r.EndLine, r.EndCol
	}
	lines := strings.Split(value, "\n")
	endLine := r.EndLine
	endCol := min(r.EndCol, len(lines[endLine]))
	prefix := lines[endLine][:endCol]
	suffix := lines[endLine][endCol:]
	lines[endLine] = prefix + text + suffix
	return strings.Join(lines, "\n"), endLine, endCol + len(text)
}
