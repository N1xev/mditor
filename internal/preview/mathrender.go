package preview

import (
	"regexp"
	"strings"

	renderer "github.com/N1xev/mditor/internal/latexrender"
)

// hasComplexLatex reports whether the input contains commands that need
// the mathcha renderer (fractions, sqrt, sums, integrals, sub/superscript
// stacking). For simple cases the regex translator in math.go is enough
// and avoids the cost of parsing.
//
// `s` may be the math body AFTER protectMath has doubled backslashes — in
// that case the literal commands look like `\\int`, `\\frac`, etc. We check
// both forms so this works whether the caller has already protected or not.
func hasComplexLatex(s string) bool {
	for _, cmd := range []string{
		`\frac`, `\dfrac`, `\tfrac`, `\cfrac`,
		`\sqrt`, `\sum`, `\prod`, `\int`, `\iint`, `\iiint`, `\oint`,
		`\lim`, `\binom`, `\matrix`, `\pmatrix`, `\bmatrix`, `\vmatrix`, `\begin`,
		`\overline`, `\underline`, `\overbrace`, `\underbrace`,
		`\overset`, `\underset`, `\stackrel`,
		`\left`, `\right`,
		`\hat`, `\bar`, `\vec`, `\tilde`, `\dot`, `\ddot`,
		`\mathcal`, `\mathbb`, `\mathfrak`, `\mathrm`, `\mathbf`,
		`\begin{cases}`,
		`\to`, `\mapsto`,
	} {
		if strings.Contains(s, cmd) {
			return true
		}
	}
	for _, cmd := range []string{
		`\\frac`, `\\dfrac`, `\\tfrac`, `\\cfrac`,
		`\\sqrt`, `\\sum`, `\\prod`, `\\int`, `\\iint`, `\\iiint`, `\\oint`,
		`\\lim`, `\\binom`, `\\matrix`, `\\pmatrix`, `\\bmatrix`, `\\vmatrix`,
		`\\overline`, `\\underline`, `\\overbrace`, `\\underbrace`,
		`\\overset`, `\\underset`, `\\stackrel`,
		`\\hat`, `\\bar`, `\\vec`, `\\tilde`, `\\dot`, `\\ddot`,
		`\\mathcal`, `\\mathbb`, `\\mathfrak`, `\\mathrm`, `\\mathbf`,
	} {
		if strings.Contains(s, cmd) {
			return true
		}
	}
	return false
}

// mathchaUnsupported is the placeholder string mathcha emits when it parses
// an expression but has no implementation for a specific command (e.g.
// `\binom{}`, `\text{}`, etc.). When we see this in the rendered output we
// know mathcha gave up on a piece of the expression and we should fall back
// to the regex translator for a degraded-but-coherent rendering rather than
// showing literal "[unimplemented command container]" to the user.
const mathchaUnsupported = "[unimplemented command container]"

// renderLatexBlock renders a LaTeX expression via the mathcha renderer and
// returns the resulting styled multiline string. Returns the input untouched
// when mathcha panics or returns an empty buffer so the surrounding post-
// processing can still try the regex translator as a fallback. The mathcha
// vendored parser uses panic for unrecoverable errors so we recover here
// rather than letting a single bad expression crash the whole program.
func renderLatexBlock(src string) (result string) {
	defer func() {
		if r := recover(); r != nil {
			// Panic before r.View() ran means we never assigned `result`;
			// the named return above would otherwise be the empty string
			// and downstream code would treat the empty string as the
			// rendered output. Returning src on panic lets the caller's
			// `rendered != src` check fall through to translateMath, which
			// at least gives a degraded unicode form rather than nothing.
			result = src
			_ = r
		}
	}()
	r := renderer.FromFormula(unprotectMath(normalizeForMathcha(src)), true)
	if r == nil {
		return src
	}
	out := r.View()
	if strings.TrimSpace(out) == "" {
		return src
	}
	// Mathcha renders some commands it doesn't actually implement as the
	// literal string "[unimplemented command container]". Treat that the
	// same as a parse failure so the regex translator gets a chance.
	if strings.Contains(out, mathchaUnsupported) {
		return src
	}
	// Mathcha parses `\hat`, `\mathbf`, `\mathbb`, etc. as recognized commands
	// but does not apply the actual accent/font — it just emits the literal
	// `\cmd` text with underline SGR codes around it. Detect this leak so
	// translateMath can produce a proper unicode form (Ĥ, 𝐅, ℝ) instead.
	if mathchaLeaksCommand(out, src) {
		return src
	}
	return out
}

// mathchaLeaksCommand reports whether mathcha returned styled text that still
// contains the literal `\cmd` form of a LaTeX command we asked it to render.
// Mathcha wraps these leaks in underline SGR (`\x1b[4m...\x1b[24m`); a single
// leaked command in the output is enough to fall back to translateMath,
// because the user would otherwise see a partial rendering with `\hat` or
// `\mathbf` literals scattered through an otherwise-rendered equation.
func mathchaLeaksCommand(out, src string) bool {
	srcCmds := countLatexCommands(src)
	if srcCmds == 0 {
		return false
	}
	return strings.Contains(out, "\x1b[4m\\")
}

// countLatexCommands counts `\<letter>` occurrences in s. Double backslashes
// (`\\`) are line breaks in LaTeX, not commands — they're skipped.
func countLatexCommands(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			continue
		}
		if i+1 >= len(s) {
			break
		}
		next := s[i+1]
		if next == '\\' {
			i++ // skip the paired backslash, treat as line break
			continue
		}
		if (next >= 'A' && next <= 'Z') || (next >= 'a' && next <= 'z') {
			n++
		}
	}
	return n
}

// normalizeForMathcha rewrites LaTeX commands mathcha doesn't understand
// into equivalent forms it does, before we hand the expression off to the
// parser. The point is to maximize the chance of getting a properly-stacked
// multi-line output instead of falling back to the regex translator's
// flattened form. Today this is just `\cfrac` → `\dfrac`; mathcha implements
// `\dfrac` (forced displaystyle) but not `\cfrac` (continued fractions), and
// the two are visually identical at the output level.
func normalizeForMathcha(s string) string {
	return strings.ReplaceAll(s, `\cfrac`, `\dfrac`)
}

// renderLatexWithFallback tries mathcha first; if it returns the source
// unchanged (panic recovered, empty buffer, parse failure, or unimplemented
// command sentinel) it falls back to the regex translator so the user at
// least sees the simple unicode form.
func renderLatexWithFallback(src string) string {
	rendered := renderLatexBlock(src)
	if rendered != src {
		return rendered
	}
	return translateMath(src)
}

// renderBodyWithMatrices renders a display-math body that contains one or
// more matrix environments by walking the body, rendering each matrix block
// standalone via mathcha (stacked rows, borders stripped) and translating
// the surrounding text via translateMath. The two pieces are concatenated
// in order. The full-body path (passing the entire body to renderMathMatrix)
// breaks when the body has text AFTER the matrix (the Determinant form:
// `A = \begin{matrix}...\end{matrix} \quad \Rightarrow \quad \det(A) = ad - bc`),
// because mathcha concatenates the suffix onto the first matrix row and only
// the matrix's own second row ends up on its own line — producing output like
//
//	A = a b⇒det (A) = ad - bc
//	    c d
//
// where the matrix row 1 has been merged with the trailing expression and
// the framing `A = ` only appears on row 1. Splitting the body around each
// matrix match lets mathcha see a clean matrix environment (which it
// renders correctly as stacked rows) and lets translateMath handle the
// prefix/suffix with its regex path. Multiple matrices per body are handled
// by walking `matrixPattern.FindAllStringIndex` so they each render in
// place.
//
// A newline separator is inserted between the mathcha matrix output and any
// post-text so the translated suffix lands on its own line below the
// stacked matrix rows. Without the separator, the suffix concatenates onto
// the last matrix row (mathcha's output doesn't end with a newline because
// its renderer uses fixed-width cell layout, not text layout).
func renderBodyWithMatrices(body string) string {
	matches := matrixPattern.FindAllStringIndex(body, -1)
	if len(matches) == 0 {
		return translateMath(body)
	}
	var out strings.Builder
	last := 0
	for _, m := range matches {
		out.WriteString(translateMath(body[last:m[0]]))
		matrixSrc := body[m[0]:m[1]]
		mm := renderMathMatrix(matrixSrc)
		out.WriteString(mm)
		last = m[1]
	}
	post := body[last:]
	if post == "" {
		return out.String()
	}
	if !strings.HasSuffix(out.String(), "\n") {
		out.WriteByte('\n')
	}
	out.WriteString(translateMath(post))
	return out.String()
}

// hasMatrixSplitSuffix reports whether the body has significant LaTeX text
// AFTER the last matrix environment that mathcha can't reliably render
// alongside the matrix. Mathcha handles a body that is just a matrix
// (possibly wrapped in `\left...\right` delimiters) correctly, but it
// concatenates the suffix onto the first matrix row when the body has
// additional LaTeX commands like `\quad \Rightarrow \quad \det(A) = ad - bc`
// trailing the matrix. Returning true here tells the caller to split the
// body around the matrix and translate the suffix via translateMath instead.
//
// We strip trailing `\left`/`\right<delim>` (which mathcha handles correctly
// as delimiters) from the suffix and check whether anything else remains.
// The Augmented form (`\right)` only) returns false (no split needed); the
// Determinant form returns true.
func hasMatrixSplitSuffix(body string) bool {
	matches := matrixPattern.FindAllStringIndex(body, -1)
	if len(matches) == 0 {
		return false
	}
	suffix := strings.TrimLeft(body[matches[len(matches)-1][1]:], " \t")
	for {
		stripped := strings.TrimSpace(leftRightStripRE.ReplaceAllString(suffix, ""))
		if stripped == suffix {
			break
		}
		suffix = stripped
	}
	return strings.Contains(suffix, `\`)
}

var leftRightStripRE = regexp.MustCompile(`\\(?:left|right)[ \t]*[([{|\\.\\/=+*<>!?,;:\]'\"~^_]?`)

// renderMathMatrix renders a LaTeX matrix via mathcha and returns the multi-
// line stacked layout with the box-drawing border characters (⎡ ⎤ ⎢ ⎥ ⎣ ⎦)
// stripped from every line. The user wants the vertically-stacked rows that
// mathcha produces — each row on its own line — but without the visible
// surrounding bracket frame, which they find visually intrusive inside a
// flowing math paragraph.
//
// Falls back to translateMath's collapsed `(a b; c d)` form when mathcha
// panics, returns the source unchanged, or produces an empty buffer. This
// keeps a degraded-but-coherent rendering visible instead of leaving the
// raw `\begin{matrix}` text on screen.
func renderMathMatrix(src string) string {
	rendered := renderLatexBlock(src)
	if rendered == src || strings.TrimSpace(rendered) == "" {
		return translateMath(src)
	}
	return stripMatrixBorders(rendered)
}

// matrixBorderChars are the box-drawing characters mathcha wraps around
// matrix rows. Top row uses ⎡ (U+23A1) / ⎤ (U+23A4), middle rows use
// ⎢ (U+23A2) / ⎥ (U+23A5), bottom row uses ⎣ (U+23A3) / ⎦ (U+23A6). The
// set is treated as a single string and matched via strings.ContainsRune
// per line so we don't have to enumerate every codepoint individually.
var matrixBorderChars = []rune{
	'⎡', '⎤', '⎢', '⎥', '⎣', '⎦',
}

// stripMatrixBorders removes the box-drawing border characters (⎡ ⎤ ⎢ ⎥ ⎣ ⎦)
// from each line of mathcha's matrix output. Mathcha wraps the inner cell
// text in italic SGR codes (`\x1b[3m...\x1b[23m`) which sit BETWEEN the
// border characters and the cell content, so a naïve per-line trim only
// catches borders at the line's start/end. We instead strip ANSI first to
// see the visible cell content, drop pure-border lines (e.g. a lone ⎢ row
// with only SGR noise), and then strip every border character from the
// surviving lines so the inner ⎡⎤ in `A = ⎡a b⎤` expressions disappears too.
//
// Outer non-border brackets from `\left(\begin{matrix}` (⎛⎞) are preserved
// because they aren't in the border set — those form the user's chosen
// matrix delimiter, which the user wants to see.
func stripMatrixBorders(rendered string) string {
	lines := strings.Split(rendered, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		visible := stripANSI(line)
		visibleNoBorders := strings.Map(func(r rune) rune {
			if isMatrixBorder(r) {
				return -1
			}
			return r
		}, visible)
		// Pure-border line (e.g. a lone ⎢ with only SGR noise around it) —
		// drop it entirely.
		if strings.TrimSpace(visibleNoBorders) == "" {
			continue
		}
		// Strip the border characters from the styled line too so any
		// SGR codes interleaved with borders are simplified.
		stripped := strings.Map(func(r rune) rune {
			if isMatrixBorder(r) {
				return -1
			}
			return r
		}, line)
		// Collapse runs of spaces that result from border removal (e.g.
		// " = " from " = ⎡a b" becomes " = a b" which is fine, but
		// "\x1b[3m⎡a\x1b[23m" becomes "\x1b[3ma\x1b[23m" cleanly).
		out = append(out, stripped)
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n")
}

func isMatrixBorder(r rune) bool {
	for _, b := range matrixBorderChars {
		if r == b {
			return true
		}
	}
	return false
}

func unprotectMath(s string) string {
	return strings.ReplaceAll(s, nbsp, " ")
}
