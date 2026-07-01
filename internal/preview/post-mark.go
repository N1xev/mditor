package preview

import (
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/N1xev/mditor/internal/uict"
)

var markPostRE = regexp.MustCompile(`(?s)` + regexp.QuoteMeta(MarkOpen) + `(.*?)` + regexp.QuoteMeta(MarkClose))

var (
	markOpenEsc  = "\033[3m\033[38;2;" + rgbTriplet(uict.Tang) + "m\033[48;2;" + rgbTriplet(uict.Squid) + "m"
	markCloseEsc = "\033[0m"
)

func rgbTriplet(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return strconv.Itoa(int(r>>8)) + ";" + strconv.Itoa(int(g>>8)) + ";" + strconv.Itoa(int(b>>8))
}

var (
	// mathDisplayRE / mathInlineRE consume the leading whitespace BEFORE the
	// open sentinel and the trailing whitespace AFTER the close sentinel so
	// the post-processor can recover the glamour line's indent and drop the
	// glamour line's wrap-width trailing pad. Without consuming the trailing
	// pad, only the last line of a multi-line mathcha output inherits the
	// pad (rendered as invisible padding past the math content).
	mathInlineRE  = regexp.MustCompile(`(?s)([ \t]*)` + regexp.QuoteMeta(MathInlineOpen) + `(.*?)` + regexp.QuoteMeta(MathInlineClose) + `([ \t]*)`)
	mathDisplayRE = regexp.MustCompile(`(?s)([ \t]*)` + regexp.QuoteMeta(MathDisplayOpen) + `(.*?)` + regexp.QuoteMeta(MathClose) + `([ \t]*)`)
)

var (
	mathInlineStyle = lipgloss.NewStyle().
			Foreground(uict.Julep).
			Italic(true)

	mathDisplayStyle = lipgloss.NewStyle().
			Foreground(uict.Bok).
			Italic(true).
			Bold(true)
)

func applyMathPost(s string) string {
	if !strings.Contains(s, MathInlineOpen) && !strings.Contains(s, MathDisplayOpen) {
		return s
	}
	// Rejoin any glamour-induced close-sentinel split BEFORE mathDisplayRE
	// runs. The regex `§»§` is a literal 3-char sequence; glamour can wrap
	// at narrow pane widths and break it as `§»\n  §`, leaving no complete
	// `§»§` for mathDisplayRE to anchor on — so the math block leaks through
	// as raw LaTeX. Fix the split here so the regex sees a contiguous
	// sentinel.
	s = sentinelCloseSplitRE.ReplaceAllString(s, MathClose)
	s = mathDisplayRE.ReplaceAllStringFunc(s, func(match string) string {
		sub := mathDisplayRE.FindStringSubmatch(match)
		indent := sub[1]
		rawBody := sub[2]
		rawStripped := stripANSI(rawBody)
		body := strings.ReplaceAll(rawStripped, nbsp, " ")
		// Glamour wraps the sentinel body at narrow pane widths, splitting
		// it across multiple lines even when the body is NBSP-protected. The
		// split breaks the matrix body (and any other multi-token LaTeX
		// expression) so that translateMath's regexes no longer match. Rejoin
		// the wrapped fragments by trimming each line and joining with a
		// single space — `WithPreservedNewLines()` keeps the intentional
		// `\quad`/`\qquad` splits from flowDisplayMath in place because they
		// carry no leading indent, while glamour's word-wrap adds a 2-space
		// indent (the sentinel's `$$` indent) that we strip here.
		body = rejoinWrappedBody(body)
		// protectMath doubled every `\` before glamour, so `\\` in the source
		// LaTeX arrives here as `\\` and single `\` stays as `\` — no unescape
		// needed.
		var rendered string
		matched := matrixPattern.MatchString(body)
		if matched {
			// Matrices: pass the whole body to mathcha when it's a clean
			// matrix (possibly wrapped in `\left` / `\right` delimiters) —
			// mathcha renders outer ⎛⎞⎝⎠ brackets correctly for the wrapped
			// form, and renders plain `R(θ) = matrix` prefixes correctly.
			// For bodies with significant LaTeX text AFTER the matrix
			// (the Determinant form:
			// `A = \begin{matrix}...\end{matrix} \quad \Rightarrow \quad
			// \det(A) = ad - bc`), mathcha concatenates the suffix onto
			// the matrix's first row. Splitting the body around each matrix
			// match lets mathcha see a clean matrix and translateMath handle
			// the surrounding text with its regex path.
			if hasMatrixSplitSuffix(body) {
				rendered = mathDisplayStyle.Render(renderBodyWithMatrices(body))
			} else {
				mm := renderMathMatrix(body)
				rendered = mathDisplayStyle.Render(mm)
			}
		} else if hasComplexLatex(body) {
			// Prefer translateMath for display math — mathcha renders inline-
			// style subscripts (a_n, x_2) by stacking the subscript BELOW
			// the base glyph instead of below-right, which produces visually
			// broken multi-line output for the majority of natural LaTeX.
			// Only fall back to mathcha when translateMath leaves a literal
			// `\cmd` unresolved (regex miss) — that signals an expression
			// mathcha might handle, e.g. `\begin{cases}` or `\overbrace`.
			translated := translateMath(body)
			if containsUnresolvedLatexCommand(translated) {
				rendered = mathDisplayStyle.Render(renderLatexWithFallback(body))
			} else {
				rendered = mathDisplayStyle.Render(translated)
			}
		} else {
			rendered = mathDisplayStyle.Render(translateMath(body))
		}
		return reindentMathBlock(indent, rendered)
	})
	s = mathInlineRE.ReplaceAllStringFunc(s, func(match string) string {
		sub := mathInlineRE.FindStringSubmatch(match)
		body := strings.ReplaceAll(stripANSI(sub[2]), nbsp, " ")
		if hasComplexLatex(body) {
			rendered := renderLatexBlock(body)
			// Inline math MUST fit on one line — mathcha renders fractions,
			// sums, integrals etc. as multi-line stacks, which would split a
			// table row across multiple lines and break cell alignment. Fall
			// back to the regex translator (which produces a single-line form
			// like `(a)/(b)` or `∑_{i=1}^{n} i`) whenever mathcha returns a
			// multi-line output.
			if rendered != body && !strings.Contains(rendered, "\n") {
				return mathInlineStyle.Render(rendered)
			}
			return mathInlineStyle.Render(translateMath(body))
		}
		return mathInlineStyle.Render(body)
	})
	return s
}

// reindentMathBlock fixes two alignment artifacts that appear when a multi-
// line mathcha output is substituted into a glamour-rendered line:
//
//  1. The glamour line was padded with leading whitespace (typically a 2-cell
//     block indent) BEFORE the math sentinel and trailing whitespace to the
//     wrap width AFTER the sentinel. When we replace the sentinel-wrapped
//     body with mathcha's multi-line output, only the FIRST line keeps the
//     leading indent and only the LAST line keeps the trailing pad — so the
//     middle/bottom rows of a fraction render shifted left and the bottom
//     row drags invisible padding to the right edge.
//
//  2. The mathcha output itself starts with SGR codes that consume any
//     leading-whitespace detection in the result, so we capture the indent
//     from the OUTER match (which sits in front of `«mdtM»`) and re-apply it
//     to every line. The trailing pad is consumed by the regex itself and
//     dropped entirely.
//
// Trailing whitespace is stripped from every line (the outer pad only ever
// lived on the last line but stripping is idempotent).
func reindentMathBlock(indent, rendered string) string {
	if rendered == "" {
		return rendered
	}
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
		if indent != "" && !strings.HasPrefix(lines[i], indent) {
			lines[i] = indent + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

// rejoinWrappedBody collapses glamour's word-wrap newlines inside the sentinel
// body back into a single logical line. Glamour wraps at hyphens and other
// natural break points even when the body is NBSP-protected, so a matrix like
// `(cosθ   -sinθ; sinθ   cosθ)` can arrive as two lines with the `-sinθ` half
// dangling on line 2. Splitting the body across lines breaks the regexes in
// translateMath (matrixPattern, fracPattern, etc.), which then leave the
// expression un-translated and force a mathcha fallback that produces the
// bordered matrix rendering the user complained about.
//
// The rejoin drops every leading-whitespace newline pair and joins the lines
// with a single space. Intentional `\quad` splits from flowDisplayMath are
// preserved: they appear as bare newlines (no leading whitespace) in the
// sentinel body because glamour doesn't pad them, while word-wrap lines always
// carry glamour's 2-cell leading indent (the original `$$` indent).
//
// CRITICAL: glamour can also word-wrap the sentinel text itself, splitting
// `§»§` into `§\n  »§` across a line boundary. After the line-based rejoin
// above, the body becomes `\end{matrix}§»§ Identity 3×3: «§»\begin{matrix}…`
// — the close sentinel of the previous matrix, the intervening section text,
// and the open sentinel of the next matrix are all glued into this body
// because mathDisplayRE's `.*?` ran forward past the broken sentinel looking
// for the next intact `§»§`. To recover the original matrix body, truncate
// at the first surviving `§»§` — anything past it is leaked content from the
// next math block, not part of this one. Without this truncation, mathcha
// receives a body containing two matrices concatenated and renders them as a
// single overlapping stack with rows interleaved.
func rejoinWrappedBody(body string) string {
	if !strings.Contains(body, "\n") {
		return body
	}
	// Repair glamour word-wraps that split LaTeX syntax across a line
	// boundary BEFORE the line-based rejoin runs — otherwise the
	// whitespace-prepend step below inserts a space between the broken
	// fragments (e.g., `\cfra` + `c{…}` becomes `\cfra c{…}` with a
	// space, which translateMath then can't match). Six regex passes,
	// each targeting a different break shape:
	//
	//  1. brokenEnvNameRE — `\\begin{<partial>{<rest>}` style. Glamour
	//     wraps inside the env name like `\end{matri` + `x}`. The two
	//     fragments are rejoined to restore `\end{matrix}`.
	//
	//  2. brokenCommandRE — `\<partial>` + `<rest>{…` style. Glamour
	//     wraps inside a backslash command like `\fra` + `c{1}{3}`.
	//     Rejoined to restore `\frac{1}{3}`.
	//
	//  3. brokenCommandCloseRE — `\<partial>` + `)` / `]` / `}` style.
	//     Glamour wraps right before a closing delimiter like `\right`
	//     + `)`. Rejoined to restore `\right)`.
	//
	//  4. orphanBackslashRE — orphan `\` at end of line followed by
	//     `begin` / `end` on the next line. Reconstructs `\begin{` /
	//     `\end{` (the existing `{` on the next line is left in place).
	//
	//  5. midCommandNameRE — `\<letters>` + `<letters>` style. Glamour
	//     wraps inside a multi-letter command name like `\t` + `heta`
	//     (from `\theta`) or `\righ` + `t` (from `\right`). Greedy on
	//     both letter groups so `\R` + `ightarrow` (from
	//     `\Rightarrow`) rejoins in one match.
	//
	//  6. subSupSplitRE — `^` / `_` followed by newline + `{...}`.
	//     Glamour wraps between the operator and its brace group, e.g.
	//     `^{n}` becomes `^` + NL + `{n}`. The `^{` is rejoined so
	//     translateLatexSubSup can match the complete group.
	body = brokenEnvNameRE.ReplaceAllString(body, `\$1{$2$3}`)
	body = brokenCommandRE.ReplaceAllString(body, "\\$1$2$3")
	body = brokenCommandCloseRE.ReplaceAllString(body, "\\$1$2")
	body = orphanBackslashRE.ReplaceAllString(body, "\\$1")
	body = midCommandNameRE.ReplaceAllString(body, "\\$1$2")
	body = subSupSplitRE.ReplaceAllString(body, "$1{")
	lines := strings.Split(body, "\n")
	var out strings.Builder
	prev := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if prev == "" {
			out.WriteString(trimmed)
			prev = trimmed
			continue
		}
		switch {
		case strings.HasSuffix(prev, "\\"):
			// Glamour wrapped inside a LaTeX escape sequence that the
			// regex passes above missed. Drop the newline so the
			// original escape is restored.
			out.WriteString(trimmed)
		case strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t"):
			// Glamour word-wrap at whitespace: join with a single space
			// (glamour's 2-cell indent is the sentinel's `$$` indent that
			// we want to drop here).
			out.WriteByte(' ')
			out.WriteString(trimmed)
		default:
			// Bare newline (no leading whitespace from glamour) signals
			// an intentional split from flowDisplayMath; keep it.
			out.WriteByte('\n')
			out.WriteString(trimmed)
		}
		prev = trimmed
	}
	joined := out.String()
	// Collapse sentinel fragments that glamour's wrap split. We strip any
	// whitespace (space or tab) that ended up between the sentinel runes so
	// the contiguous form is restored for the post-processor regexes.
	joined = rejoinSentinelRE.ReplaceAllString(joined, "§»§")
	joined = rejoinSentinelOpenRE.ReplaceAllString(joined, "«§»")
	// If glamour wrapped the close sentinel `§»§` itself across a line
	// boundary, the mathDisplayRE regex's `.*?` ran forward past the broken
	// sentinel and captured content from the next math block into this
	// body's `.*?`. After the rejoin above, the leaked close sentinel sits
	// in the middle of the body followed by the intervening section text
	// and the next block's open sentinel. Drop everything from the first
	// surviving `§»§` onward — anything past it is leaked, not part of this
	// matrix block.
	if idx := strings.Index(joined, "§»§"); idx >= 0 {
		joined = joined[:idx]
	}
	// Glamour interprets the LaTeX row separator `\\` as a CommonMark hard
	// line break and replaces it with a literal `\n` in the wrapped output.
	// The line-based rejoin above keeps that `\n` (it has no leading
	// whitespace, so it looks like an intentional split from
	// flowDisplayMath) but mathcha cannot parse a matrix body that contains
	// a bare newline — it renders the rows as a single collapsed line
	// instead of the stacked layout the user wants. Convert any internal
	// `\n` inside `\begin{matrix|pmatrix|...}...\end{...}` back to `\\`
	// so mathcha sees the proper row separator.
	return fixMatrixRowSeparators(joined)
}

// brokenEnvNameRE matches a glamour wrap inside a `\begin{…}` or
// `\end{…}` environment name. Glamour can split the env name in four
// shapes at narrow pane widths:
//
//  1. `\end{matri` + `x}` → `\end{matrix}` (name split across lines)
//  2. `\end{` + `matrix}` (wrap right after `{`, name on line 2)
//  3. `{matrix` + `}` (wrap right before `}`, name on line 1)
//  4. `\end` + `{matrix}` (wrap between `\end` and `{`, brace on line 2)
//
// The leading `\\(begin|end)` keeps the regex focused on actual LaTeX
// environments so we don't accidentally merge two unrelated word
// fragments that happen to sit on either side of a wrap inside arbitrary
// `{...}` brace groups. Both letter groups are allowed to be empty to
// cover cases 2 and 3. Case 4 is handled by allowing the opening brace
// to appear on either side of the newline.
var brokenEnvNameRE = regexp.MustCompile(`\\(begin|end)\{?([A-Za-z]{0,15})\n[ \t]*\{?([A-Za-z]{0,15})([}])`)

// brokenCommandRE matches a glamour wrap inside a single-backslash
// LaTeX command whose next char is `{` — e.g., `\fra` + `c{1}{3}` →
// `\frac{1}{3}`. Captured groups: tail letters on prev, head letters on
// cur, the opening `{` (or end-of-input anchor) that follows.
var brokenCommandRE = regexp.MustCompile(`\\([A-Za-z]{1,15})\n[ \t]*([A-Za-z]{1,15})([{]|$)`)

// brokenCommandCloseRE matches a glamour wrap that split a LaTeX
// command from its closing delimiter — e.g., `\right` + `)` → `\right)`.
// Captured groups: tail letters on prev, the closing delim on cur.
var brokenCommandCloseRE = regexp.MustCompile(`\\([A-Za-z]{1,15})\n[ \t]*([)}\]])`)

// orphanBackslashRE matches an orphan `\` at end of line followed by
// `begin` or `end` on the next line. Glamour wraps inside LaTeX
// commands when the body is too wide — for the augmented-matrix body
// at width 60 the wrap lands between the `\` and the `end` of `\end`,
// producing:
//
//	…f \
//	  end{matrix}\right)
//
// where the `\` is the backslash of `\end` and `end{matrix}` is on the
// next line. Rejoining `\\\n[ \t]*end` to `\end` restores the original
// command. The `{` (if present on the same line) is NOT consumed by the
// regex so it stays in place — replacement is just `\begin` / `\end`.
//
// This replaces the more aggressive brokenBackslashFollowedByLettersRE
// (`\\+\n[ \t]*[A-Za-z]+` → empty) which ate BOTH the orphan `\` AND the
// `end` text, breaking `\end{matrix}` into `{matrix}`.
var orphanBackslashRE = regexp.MustCompile(`\\\n[ \t]*(begin|end)`)

// midCommandNameRE matches a LaTeX command name broken across a line by
// glamour's word-wrap. At width 40 the rotation matrix body has the
// `θ` glyph rendered as `\theta` (6 chars), which glamour splits as
// `\t\n  heta`. The Augmented form at width 40 splits `\right` (6 chars)
// as `\righ\n  t`. Each fragment is rejoined to restore the original
// command name.
//
// The regex is greedy on both letter groups so `\R\n  ightarrow` (from a
// wrap inside `\Rightarrow`) becomes `\Rightarrow` in one match. Adjacent
// commands in the source (e.g. `\cos\theta` with no space) don't have
// NBSP between them, so glamour doesn't insert a wrap there — the regex
// only fires when the wrap genuinely fell inside a single command name.
var midCommandNameRE = regexp.MustCompile(`\\([A-Za-z]+)\n[ \t]*([A-Za-z]+)`)

// subSupSplitRE matches a LaTeX sub/sup operator (`^` or `_`) broken
// across a line from its `{...}` group. The `\sum_{i=1}^{n}` form at
// width 30 has glamour wrap between `^` and `{n}`:
//
//	\sum_{i=1}^
//	  {n} i
//
// producing `^\n  {n}` in the body. translateLatexSubSup's regex needs
// `^{n}` on a single line to translate it to `ⁿ`, so the bare newline
// breaks the translation and the user sees a literal `^ {n}`. Rejoin
// `^\n  {` (and `_\n  {`) to `^{` / `_{` so the translator sees a
// complete group.
var subSupSplitRE = regexp.MustCompile(`([\^_])\n[ \t]*\{`)

// sentinelCloseSplitRE matches the display-math close sentinel `§»§`
// split across a line boundary. Glamour wraps at narrow pane widths and
// can break the 3-char sentinel as `§»\n  §` (line 1 ends with `§»` and
// line 2 starts with `§`, often followed by trailing space-padding to
// fill the wrap width). Without rejoining, mathDisplayRE can't anchor on
// a contiguous `§»§` and the entire math block leaks through as raw
// LaTeX. Replace with the contiguous sentinel so mathDisplayRE matches
// the block normally. Optional whitespace on both sides of the newline
// is consumed.
//
// Runs BEFORE mathDisplayRE in applyMathPost — the line-based rejoin
// inside rejoinWrappedBody can't help here because it never executes if
// mathDisplayRE failed to anchor on the math block in the first place.
var sentinelCloseSplitRE = regexp.MustCompile(`§[ \t]*»\n[ \t]*§`)

// fixMatrixRowSeparators replaces any literal `\n` that survives inside a
// `\begin{matrix|pmatrix|...}...\end{...}` environment with `\\`. Glamour
// treats the LaTeX row separator `\\` as a CommonMark hard line break, so
// in narrow panes it ends up as a bare newline in the wrapped output. The
// line-based rejoin keeps bare newlines (they look like intentional
// flowDisplayMath splits), but mathcha can't parse a matrix body with a
// literal newline — it renders the matrix as a single collapsed line. The
// `matrixPattern` body group `(?:[^{}]|\{[^{}]*\})*` allows `\n`, so the
// match succeeds even with a newline inside; we then substitute `\\` for
// every internal `\n` to restore the row separator mathcha expects.
var fixMatrixRowSepRE = regexp.MustCompile(`\\begin\{(matrix|pmatrix|bmatrix|vmatrix|Vmatrix)\}((?:[^{}]|\{[^{}]*\})*)\\end\{(matrix|pmatrix|bmatrix|vmatrix|Vmatrix)\}`)

func fixMatrixRowSeparators(body string) string {
	return fixMatrixRowSepRE.ReplaceAllStringFunc(body, func(m string) string {
		sub := fixMatrixRowSepRE.FindStringSubmatch(m)
		fixed := strings.ReplaceAll(sub[2], "\n", `\\`)
		return `\begin{` + sub[1] + `}` + fixed + `\end{` + sub[3] + `}`
	})
}

// rejoinSentinelRE matches `§` followed by optional whitespace and then `»§`.
// Glamour's word-wrap can split the display-math close sentinel `§»§` across
// a line boundary (e.g. `…\end{matrix}§\n  »§`), which after a naive rejoin
// becomes `…\end{matrix}§ »§` — a single space inside the sentinel. Strip the
// inserted whitespace to restore the contiguous sentinel.
var rejoinSentinelRE = regexp.MustCompile(`§[ \t]+»§`)

// rejoinSentinelOpenRE matches `«§»` and is the open-sentinel counterpart.
// In practice glamour doesn't split `«§»` because it's the first thing in
// the wrapped line and glamour doesn't insert leading whitespace before
// it, but the regex is here for symmetry in case that ever changes.
var rejoinSentinelOpenRE = regexp.MustCompile(`«[ \t]+§[ \t]+»`)

func applyMarkPost(s string) string {
	if !strings.Contains(s, MarkOpen) {
		return s
	}
	return markPostRE.ReplaceAllStringFunc(s, func(match string) string {
		sub := markPostRE.FindStringSubmatch(match)
		inner := stripANSI(sub[1])
		if inner == "" {
			return match
		}
		return markOpenEsc + inner + markCloseEsc
	})
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

// containsUnresolvedLatexCommand reports whether s still contains a LaTeX
// command of the form `\letter…` that translateMath didn't recognize. Used
// after a translateMath pass to decide whether the result is acceptable or
// if mathcha should get another try.
var unresolvedLatexCmdRE = regexp.MustCompile(`\\[A-Za-z]+`)

func containsUnresolvedLatexCommand(s string) bool {
	return unresolvedLatexCmdRE.MatchString(s)
}

