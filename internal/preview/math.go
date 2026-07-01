package preview

import (
	"regexp"
	"strings"
)

const (
	// Sentinels are 3 chars (guillemet-bracket-guillemet) with a single
	// non-letter rune inside. Glamour's word-wrap treats a 3-char token as
	// atomic when a line break has to be inserted inside it: at narrow widths
	// the body before/after the sentinel gets wrapped, but the sentinel
	// itself stays intact. The previous `«mdtm»` (5-char token with letters)
	// could be split as `«md\ntm»` in narrow cells, which the post-processor
	// regex couldn't match. Single-rune sentinels avoid that path entirely.
	//
	// Three independent symbols (mid-dot, section sign, degree sign) give us
	// distinct open/close pairs without using letters or digits that the
	// user might type in math content.
	MathInlineOpen  = "«·»" // inline math sentinel open
	MathInlineClose = "·»·" // inline math sentinel close
	MathDisplayOpen = "«§»" // display math sentinel open
	MathClose       = "§»§" // display math sentinel close (display blocks share this)
)

var nbsp = " "

var (
	// displayMathRE requires BOTH `$$` markers to be on their own line so it
	// doesn't accidentally treat the trailing `$$` of a one-line display
	// block (e.g. `$$x=1$$`) as the opening of the NEXT block and swallow the
	// prose between them. The opening `$$\n` is anchored to start-of-line so
	// an inline `$$` mid-sentence (like a heading mentioning "double $$")
	// can't trigger a match either.
	displayMathRE = regexp.MustCompile(`(?m)(^|[ \t]*\n)\$\$$[ \t]*\n([\s\S]+?)\n[ \t]*\$\$`)
	inlineMathRE  = regexp.MustCompile(`\$([^$\n]+?)\$`)
)

func convertMath(src string) string {
	src = displayMathRE.ReplaceAllStringFunc(src, func(m string) string {
		sub := displayMathRE.FindStringSubmatch(m)
		// sub[1] = leading whitespace/newline before opening `$$`
		// sub[2] = body content (between opening and closing `$$\n`)
		inner := strings.TrimSpace(sub[2])
		// Pre-flow at `\quad` / `\qquad` boundaries so glamour's word-wrap
		// never has to break a LaTeX command mid-name. With
		// `WithPreservedNewLines()` enabled in the renderer, the inserted
		// newlines survive glamour and each expression lands on its own
		// visual line. The post-processor's `(?s)` regex matches across
		// those newlines so the whole block still translates as one unit.
		inner = flowDisplayMath(inner)
		// Keep the original LaTeX verbatim for display math — mathcha needs
		// the raw `\frac{...}{...}` form, not the regex-translated `(a)/(b)`
		// form. The post-processor will fall back to translateMath for simple
		// expressions that mathcha can't render or for panics it recovers.
		//
		// The brace-clarify pass rewrites `^{x^2}` to `^{x^{2}}` so mathcha's
		// parser sees the nested sup as a distinct child (otherwise the inner
		// `^2` ends up rendered as plain text rather than as a sub-sup).
		return sub[1] + MathDisplayOpen + protectMath(clarifyNestedSupSub(inner)) + MathClose
	})
	src = inlineMathRE.ReplaceAllStringFunc(src, func(m string) string {
		inner := strings.TrimSpace(m[1 : len(m)-1])
		if !isPlausibleMath(inner) {
			return m
		}
		// For simple inline math (no commands needing mathcha's multi-line
		// output), pre-translate directly into unicode and substitute back
		// into the markdown WITHOUT sentinel wrappers. Sentinels let glamour
		// see 11-byte tokens like `«mdtm»α«mdtx»`; glamour's word-wrap then
		// splits those across cell boundaries in narrow table columns,
		// producing artifacts like `«mdtm»A«mdt\nxx»` that the post-processor
		// can't recover. Inline-unicode form (e.g. just `α`) wraps naturally.
		if !hasComplexLatex(inner) {
			return translateMath(inner)
		}
		return MathInlineOpen + protectMath(clarifyNestedSupSub(inner)) + MathInlineClose
	})
	return src
}

// clarifyNestedSupSub wraps nested un-braced super/subscripts inside an
// existing `^{...}` or `_{...}` group with explicit braces, so the parser
// (mathcha in the complex path) sees each sup/sub level as a distinct child
// rather than glossing over the inner `^x` as text. The pattern matches an
// opening `^{` followed by content that itself contains a bare `^` or `_`,
// then wraps that inner operator's right-hand side in `{}`.
//
// `e^{-x^2}` → `e^{-x^{2}}`
// `x_{i,j}^2` is untouched (the `_{i,j}` is correctly braced already).
// `x^2_i` → unchanged (no `^{...}` group involved).
func clarifyNestedSupSub(s string) string {
	return clarifySupInGroup.ReplaceAllStringFunc(s, func(m string) string {
		sub := clarifySupInGroup.FindStringSubmatch(m)
		body := sub[1]
		// Replace inner un-braced `^X` (and `_X`) inside the body with `^{X}`
		// so mathcha parses each level as its own command. Use a regex that
		// accepts a single token (no spaces) so we don't accidentally
		// re-wrap multi-character expressions that are already correct.
		body = nestedBareSupRE.ReplaceAllString(body, `^{$1}`)
		body = nestedBareSubRE.ReplaceAllString(body, `_{$1}`)
		return `^{` + body + `}`
	})
}

var (
	clarifySupInGroup = regexp.MustCompile(`\^\{([^{}]*[\^_][^{}]*)\}`)
	nestedBareSupRE   = regexp.MustCompile(`\^([A-Za-z0-9+\-=().])`)
	nestedBareSubRE   = regexp.MustCompile(`_([A-Za-z0-9+\-=().])`)
)

// quadSplitRE matches `\quad` or `\qquad` (or the rare `\qquad`) with the
// trailing whitespace following it. Each match becomes a line break in the
// pre-flowed output so glamour's word-wrap never has to split a LaTeX
// command mid-name (e.g. `\sup` / `{n}`).
var quadSplitRE = regexp.MustCompile(`\\[qn]?quad(?:\s+|$)`)

func flowDisplayMath(inner string) string {
	if !strings.Contains(inner, `\quad`) && !strings.Contains(inner, `\qquad`) {
		return inner
	}
	parts := quadSplitRE.Split(inner, -1)
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return strings.Join(parts, "\n")
}

// protectMath prepares a math body for safe passage through glamour's markdown
// renderer. Two transforms are needed:
//
//  1. Double every backslash so CommonMark escaping does not collapse `\\`
//     (LaTeX matrix row separator, line break) down to a single `\` before
//     the post-processor runs. glamour renders `\\` as `\` in regular prose;
//     `\\\\` survives as `\\`.
//
//  2. Replace spaces with U+00A0 so glamour's word-wrap does not split math
//     tokens across lines.
//
// LaTeX's `^`, `_`, `~`, `*`, `[`, `]` are NOT escaped here because glamour
// does not interpret them as markdown syntax on its own. The protection for
// pandoc-style `^x^` / `~x~` is provided by convertSubscriptSuperscript, which
// now skips content inside the math sentinels so LaTeX sub/sup commands inside
// `«§»…§»§` / `«·»…·»·` are left untouched.
func protectMath(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, " ", nbsp)
	return s
}

func isPlausibleMath(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		return false
	}
	if strings.Contains(s, "\\") {
		return true
	}
	if strings.ContainsAny(s, "=+-*/^_<>()[]{}|.,;:") {
		return true
	}
	for _, r := range s {
		if r >= 0x370 && r <= 0x3ff {
			return true
		}
	}
	if isSingleLetterExpr(s) {
		return true
	}
	return false
}

// isSingleLetterExpr reports whether the body is a single math variable —
// either one Latin letter (A–Z, a–z) or a sequence of letters and digits
// like "x1", "AB", "v_n" (the user-facing pattern is `$A$`, `$x$`, etc.).
// This makes `$A$` through `$Z$` render as math in tables while keeping
// bare numerals and punctuation out of the math post-processor.
func isSingleLetterExpr(s string) bool {
	if len(s) == 0 || len(s) > 8 {
		return false
	}
	hasLetter := false
	for i, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			hasLetter = true
		case r >= 'a' && r <= 'z':
			hasLetter = true
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return hasLetter
}

var greekMap = map[string]string{
	"alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ", "epsilon": "ε",
	"varepsilon": "ε", "zeta": "ζ", "eta": "η", "theta": "θ", "vartheta": "ϑ",
	"iota": "ι", "kappa": "κ", "lambda": "λ", "mu": "μ", "nu": "ν",
	"xi": "ξ", "omicron": "ο", "pi": "π", "varpi": "ϖ", "rho": "ρ",
	"varrho": "ϱ", "sigma": "σ", "varsigma": "ς", "tau": "τ", "upsilon": "υ",
	"phi": "φ", "varphi": "ϕ", "chi": "χ", "psi": "ψ", "omega": "ω",
	"Gamma": "Γ", "Delta": "Δ", "Theta": "Θ", "Lambda": "Λ", "Xi": "Ξ",
	"Pi": "Π", "Sigma": "Σ", "Upsilon": "Υ", "Phi": "Φ", "Psi": "Ψ",
	"Omega": "Ω",
}

var opMap = map[string]string{
	"sum": "∑", "prod": "∏", "int": "∫", "iint": "∬", "iiint": "∭",
	"oint": "∮", "bigcup": "⋃", "bigcap": "⋂", "bigoplus": "⊕",
	"bigotimes": "⊗", "biguplus": "⊎", "bigsqcup": "⊔",
	"pm": "±", "mp": "∓", "times": "×", "div": "÷", "cdot": "⋅",
	"ast": "∗", "star": "⋆", "circ": "∘", "bullet": "•",
	"leq": "≤", "le": "≤", "geq": "≥", "ge": "≥", "neq": "≠", "ne": "≠",
	"equiv": "≡", "approx": "≈", "sim": "∼", "simeq": "≃", "cong": "≅",
	"propto": "∝", "ll": "≪", "gg": "≫",
	"in": "∈", "notin": "∉", "ni": "∋", "subset": "⊂", "supset": "⊃",
	"subseteq": "⊆", "supseteq": "⊇", "cup": "∪", "cap": "∩", "emptyset": "∅",
	"to": "→", "rightarrow": "→", "leftarrow": "←", "leftrightarrow": "↔",
	"Rightarrow": "⇒", "Leftarrow": "⇐", "Leftrightarrow": "⇔",
	"mapsto": "↦", "uparrow": "↑", "downarrow": "↓",
	"forall": "∀", "exists": "∃", "nexists": "∄",
	"nabla": "∇", "partial": "∂", "infty": "∞",
	"hbar": "ℏ", "ell": "ℓ", "Re": "ℜ", "Im": "ℑ",
	"aleph": "ℵ", "wp": "℘",
	"langle": "⟨", "rangle": "⟩", "lceil": "⌈", "rceil": "⌉",
	"lfloor": "⌊", "rfloor": "⌋",
	"ldots": "…", "cdots": "⋯", "vdots": "⋮", "ddots": "⋱",
	"quad": "  ", "qquad": "    ",
	"sqrt": "√",
	// Operator names — translator leaves them as plain text but they look
	// better rendered with a non-italic upright glyph.
	"sin": "sin", "cos": "cos", "tan": "tan", "cot": "cot", "sec": "sec", "csc": "csc",
	"arcsin": "arcsin", "arccos": "arccos", "arctan": "arctan",
	"sinh": "sinh", "cosh": "cosh", "tanh": "tanh",
	"log": "log", "ln": "ln", "exp": "exp", "lg": "lg",
	"min": "min", "max": "max", "sup": "sup", "inf": "inf",
	"lim": "lim", "arg": "arg", "mod": "mod", "det": "det", "gcd": "gcd",
	"Pr": "Pr", "hom": "hom", "ker": "ker",
	// Decorations and accents — the glyph is dropped gracefully when no
	// precomposed form is available (see accentPattern handlers below).
	"hat": "^", "check": "ˇ", "tilde": "~", "acute": "´", "grave": "`",
	"dot": "˙", "ddot": "¨", "breve": "˘", "bar": "¯",
	"vec": "→", "overrightarrow": "→", "overleftarrow": "←",
	"overline": "¯", "underline": "_",
}

func TranslateMath(s string) string {
	return translateMath(s)
}

func translateMath(s string) string {
	s = translateGreek(s)
	// Run translateCommands (which handles \cmd → unicode via opMap,
	// \frac{...}{...}, \sqrt, etc.) BEFORE translateLatexSubSup so that:
	//   1. Command names like `\infty`, `\in`, `\sum` are converted to their
	//      single-codepoint unicode form BEFORE we try to apply sup/sub to
	//      a brace-group body. Otherwise toSub("x \in S") would map `i` →
	//      `ᵢ` and `n` → `ₙ`, fragmenting the `\in` into `\ᵢₙ` literal text.
	//   2. `\frac{x^2}{y}` becomes `(x^2)/(y)` before we translate `^2` →
	//      `²` inside the captured `x^2` arg.
	s = translateCommands(s)
	s = translateLatexSubSup(s)
	s = collapseBraces(s)
	return s
}

var cmdPattern = regexp.MustCompile(`\\([A-Za-z]+|.)`)

// braceBody matches the body of a `{...}` group allowing up to two levels of
// nested braces, so commands like `\frac{a^{2}}{b}` and `\sqrt{b^{2}-4ac}`
// (where the body contains a `^{...}` group with its own braces) can be
// translated by the regex below. The previous `[^{}]*` body would not match
// at all when a single nested group was present, leaving the literal `\frac`
// in the output (visible to the user as un-translated LaTeX). RE2 has no
// recursion, so we hand-roll two levels with `(?:\{[^{}]*\})?` inside the
// `[^{}]*` run; this covers the common mathcha/post-process cases without
// exploding the pattern.
const braceBody = `(?:[^{}]|\{[^{}]*(?:\{[^{}]*\})?[^{}]*\})*`

// fracPattern matches \frac, \dfrac, and \tfrac — the [cdt]? prefix is
// optional, and `[cdt]` keeps the regex anchored to the LaTeX prefix letters
// rather than accidentally accepting any sequence like `\xfrac`. Body
// expression allows nested braces so `\frac{f^{(n)}(a)}{n!}` matches.
var fracPattern = regexp.MustCompile(`\\[cdt]?frac\{(` + braceBody + `)\}\{(` + braceBody + `)\}`)
var cfracPattern = regexp.MustCompile(`\\cfrac\{(` + braceBody + `)\}\{(` + braceBody + `)\}`)
var binomPattern = regexp.MustCompile(`\\(?:[td]?binom)\{(` + braceBody + `)\}\{(` + braceBody + `)\}`)
var textPattern = regexp.MustCompile(`\\text(?:rm)?\{(` + braceBody + `)\}`)
var sqrtPattern = regexp.MustCompile(`\\sqrt(?:\[([^\]]*)\])?\{(` + braceBody + `)\}`)
// Accent commands that take a single argument and decorate it. Without
// argument-aware handling, opMap would replace `\vec` with `→` and drop the
// `x` in `\vec{x}`.
var (
	vecPattern     = regexp.MustCompile(`\\vec\{([^{}]*)\}`)
	hatPattern     = regexp.MustCompile(`\\hat\{([^{}]*)\}`)
	tildePattern   = regexp.MustCompile(`\\tilde\{([^{}]*)\}`)
	barPattern     = regexp.MustCompile(`\\bar\{([^{}]*)\}`)
	dotPattern     = regexp.MustCompile(`\\dot\{([^{}]*)\}`)
	ddotPattern    = regexp.MustCompile(`\\ddot\{([^{}]*)\}`)
	widehatPattern = regexp.MustCompile(`\\widehat\{([^{}]*)\}`)
	checkPattern   = regexp.MustCompile(`\\check\{([^{}]*)\}`)
	acutePattern   = regexp.MustCompile(`\\acute\{([^{}]*)\}`)
	gravePattern   = regexp.MustCompile(`\\grave\{([^{}]*)\}`)
	brevePattern   = regexp.MustCompile(`\\breve\{([^{}]*)\}`)

	overrightarrowPattern = regexp.MustCompile(`\\overrightarrow\{([^{}]*)\}`)
	overleftarrowPattern = regexp.MustCompile(`\\overleftarrow\{([^{}]*)\}`)
	underlinePattern     = regexp.MustCompile(`\\underline\{([^{}]*)\}`)
	overlinePattern      = regexp.MustCompile(`\\overline\{([^{}]*)\}`)

	oversetPattern  = regexp.MustCompile(`\\overset\{([^{}]*)\}\{([^{}]*)\}`)
	undersetPattern = regexp.MustCompile(`\\underset\{([^{}]*)\}\{([^{}]*)\}`)
	stackrelPattern = regexp.MustCompile(`\\stackrel\{([^{}]*)\}\{([^{}]*)\}`)

	notPattern = regexp.MustCompile(`\\not\s*(=|<|>|\\le|\\ge|\\in|\\equiv|\\sim|\\approx)`)

	operatornamePattern = regexp.MustCompile(`\\operatorname\{([^{}]*)\}`)

	// Font commands — translate single-letter arguments to mathematical
	// alphanumeric symbols so they render distinctly in the terminal. The
	// unicode ranges: bold U+1D400..U+1D433, double-struck U+1D538..U+1D56B,
	// script U+1D49C..U+1D4CF, fraktur U+1D504..U+1D537.
	mathbfPattern   = regexp.MustCompile(`\\mathbf\{([^{}]*)\}`)
	mathbbPattern   = regexp.MustCompile(`\\mathbb\{([^{}]*)\}`)
	mathcalPattern  = regexp.MustCompile(`\\mathcal\{([^{}]*)\}`)
	mathfrakPattern = regexp.MustCompile(`\\mathfrak\{([^{}]*)\}`)

	// Pre-sup/sub notation used in nuclear physics (¹⁴₆C) and the
	// hypergeometric function ({}_2F_1). The empty group `{}` followed by
	// explicit super/subscripts collapses to a bare `^`/`_` after our brace
	// collapse pass; we recognize and translate it to unicode sup/sub glyphs
	// before that pass drops the structure entirely. Subscripts and
	// superscripts accept both `{n}` and bare `n` forms (LaTeX allows both).
	preSupSubPattern = regexp.MustCompile(`(?:\{\})?\s*\^\{([^}]+)\}\s*_\{?([^}]+?)\}?(?:\{([^{}]+)\}|([A-Za-z]))`)
	preSubSupPattern = regexp.MustCompile(`(?:\{\})?\s*_\{?([^}]+?)\}?\s*\^\{([^}]+)\}(?:\{([^{}]+)\}|([A-Za-z]))`)

	// LaTeX `^{...}` and `_{...}` brace-groups. The body must contain no
	// nested braces (the regex stops at the first `}`). Nested braces like
	// `^{\frac{1}{x}}` are handled by re-running the pattern after the
	// contents of the inner groups have already been translated by an earlier
	// iteration (see translateLatexSubSup).
	latexSupGroupRE = regexp.MustCompile(`\^\{([^{}]*)\}`)
	latexSubGroupRE = regexp.MustCompile(`_\{([^{}]*)\}`)

	// `\left` and `\right` delimiters in LaTeX — `\left( ... \right)` is
	// equivalent to `( ... )` for our single-line output. Strip the
	// keyword, keeping the actual delimiter character so `\left(` becomes
	// `(` and `\right|` becomes `|`. Without this strip the post-processor
	// sees an "unresolved" `\left` and falls back to mathcha, which then
	// emits its bordered matrix rendering for the wrapped matrix.
	leftRightRE = regexp.MustCompile(`\\(left|right)`)
	// Bare (un-braced) `^x` and `_x` for a single token. Single token so
	// `x^2y` correctly produces `x²y` rather than `x²` consuming `y`. The
	// un-braced form only matters after the brace-group form has been
	// translated, since the brace form wraps multi-token expressions.
	simpleSupRE = regexp.MustCompile(`\^([A-Za-z0-9+\-=().])`)
	simpleSubRE = regexp.MustCompile(`_([A-Za-z0-9+\-=().])`)
)

// mathAlphaToBold maps a single ASCII letter or digit to its mathematical
// bold counterpart (mathematical alphanumeric symbols block).
func mathAlphaToBold(r rune) rune {
	switch {
	case r >= 'A' && r <= 'Z':
		return 0x1D400 + (r - 'A')
	case r >= 'a' && r <= 'z':
		return 0x1D41A + (r - 'a')
	case r >= '0' && r <= '9':
		return 0x1D7CE + (r - '0')
	}
	return 0
}

// mathAlphaToDoubleStruck maps a single ASCII letter to its double-struck
// counterpart (blackboard bold). Only A..Z and a..z have assigned codepoints
// in this block; digits fall back to the bold form.
func mathAlphaToDoubleStruck(r rune) rune {
	switch {
	case r >= 'A' && r <= 'Z':
		return 0x1D538 + (r - 'A')
	case r >= 'a' && r <= 'z':
		return 0x1D552 + (r - 'a')
	}
	return 0
}

// mathAlphaToScript maps a single ASCII uppercase letter to its script
// counterpart. The script block only assigns A..Z; lowercase letters fall
// back to plain text.
func mathAlphaToScript(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return 0x1D49C + (r - 'A')
	}
	return 0
}

// mathAlphaToFraktur maps a single ASCII letter to its fraktur counterpart.
func mathAlphaToFraktur(r rune) rune {
	switch {
	case r >= 'A' && r <= 'Z':
		return 0x1D504 + (r - 'A')
	case r >= 'a' && r <= 'z':
		return 0x1D51E + (r - 'a')
	}
	return 0
}

// translateAlphaMap converts each character in s using mapper, falling back
// to the original rune if no codepoint is assigned (so e.g. `\mathbf{+}`
// keeps the plus sign instead of dropping it).
func translateAlphaMap(s string, mapper func(rune) rune) string {
	var b strings.Builder
	for _, r := range s {
		if mapped := mapper(r); mapped != 0 {
			b.WriteRune(mapped)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// translateLatexSubSup converts LaTeX brace-grouped super/subscripts to
// unicode sup/sub glyphs. LaTeX uses `^{x^2}` for stacked superscripts and
// `_{i,j}` for multi-token subscripts, which the regex-based pattern
// replace-all tools can't handle in one pass (the inner `^2` would be left
// literal). The trick is to iterate: each pass peels off the outermost
// `^{...}` / `_{...}` whose body has no nested braces; after translation,
// the body may itself contain `^{...}` / `_{...}` groups that get peeled
// in the next pass. After brace-groups are gone, simple `^x` / `_x` are
// converted in a final pass.
func translateLatexSubSup(s string) string {
	for changed := true; changed; {
		changed = false
		s2 := latexSupGroupRE.ReplaceAllStringFunc(s, func(m string) string {
			sub := latexSupGroupRE.FindStringSubmatch(m)
			return toSup(translateLatexSubSup(sub[1]))
		})
		if s2 != s {
			s = s2
			changed = true
			continue
		}
		s2 = latexSubGroupRE.ReplaceAllStringFunc(s, func(m string) string {
			sub := latexSubGroupRE.FindStringSubmatch(m)
			return toSub(translateLatexSubSup(sub[1]))
		})
		if s2 != s {
			s = s2
			changed = true
		}
	}
	s = simpleSupRE.ReplaceAllStringFunc(s, func(m string) string {
		return toSup(m[1:])
	})
	s = simpleSubRE.ReplaceAllStringFunc(s, func(m string) string {
		return toSub(m[1:])
	})
	return s
}

// translateSqrtNested peels `\sqrt{...}` calls from the outside in, one
// layer per pass. The single-pass replacement above only catches the
// outermost call — for nested forms like `\sqrt{1 + \sqrt{1 + ...}}` the
// inner `\sqrt` survives verbatim and the subsequent `cmdPattern` pass
// matches it as the bare `sqrt` op (substituting `√`) without touching
// the trailing `{...}` braces, producing the visual artifact
// `√(1 + √{1 + ...})`. Iterating until the regex finds no more matches
// handles 3, 4, and deeper levels of nesting identically.
func translateSqrtNested(s string) string {
	for {
		next := sqrtPattern.ReplaceAllStringFunc(s, func(m string) string {
			sub := sqrtPattern.FindStringSubmatch(m)
			body := sub[2]
			if sub[1] != "" {
				return sub[1] + "√(" + body + ")"
			}
			return "√(" + body + ")"
		})
		if next == s {
			return s
		}
		s = next
	}
}

// matrixPattern matches `\begin{matrix|pmatrix|bmatrix|...}...\end{matrix}`.
// RE2 doesn't support backrefs, so we match any `\begin{type}...\end{type}`
// and verify the types match inside the ReplaceAllStringFunc callback.
// Nested braces inside the matrix body are matched with a non-greedy
// `(?:\{[^{}]*\})*` so cells like `{1 + {2}}` collapse before the matrix sees them.
var matrixPattern = regexp.MustCompile(`\\begin\{(matrix|pmatrix|bmatrix|vmatrix|Vmatrix)\}((?:[^{}]|\{[^{}]*\})*)\\end\{(matrix|pmatrix|bmatrix|vmatrix|Vmatrix)\}`)

func translateCommands(s string) string {
	// Strip inline matrices to a compact row-major form before any other pass
	// runs. Mathcha only does multi-line output for matrices, which would
	// break an inline cell — so for the single-line regex path we collapse
	// the matrix into `(a b; c d)` notation. Cells separated by `&` join with
	// spaces, rows separated by `\\` join with `;`.
	s = matrixPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := matrixPattern.FindStringSubmatch(m)
		env := sub[1]
		endEnv := sub[3]
		if env != endEnv {
			// \begin and \end types differ — leave the original so mathcha can
			// surface a parse error rather than us silently collapsing it.
			return m
		}
		body := sub[2]
		left, right := "(", ")"
		switch env {
		case "pmatrix":
			left, right = "(", ")"
		case "bmatrix":
			left, right = "[", "]"
		case "vmatrix":
			left, right = "|", "|"
		case "Vmatrix":
			left, right = "‖", "‖"
		}
		rows := strings.Split(body, `\\`)
		for i, row := range rows {
			rows[i] = strings.TrimSpace(strings.ReplaceAll(row, "&", " "))
		}
		return left + strings.Join(rows, "; ") + right
	})
	// Strip `\left` and `\right` keywords, keeping the actual delimiter
	// character. The wrapping matrix / cases / etc. environments handle
	// their own delimiters, so the leftover `\left` / `\right` words
	// would otherwise look like unresolved commands to the fallback
	// detector and force a mathcha round-trip for bordered rendering.
	s = leftRightRE.ReplaceAllString(s, "")
	s = binomPattern.ReplaceAllString(s, "C($1,$2)")
	s = translateSqrtNested(s)
	s = cfracPattern.ReplaceAllString(s, "($1)/($2)")
	s = fracPattern.ReplaceAllString(s, "($1)/($2)")
	s = textPattern.ReplaceAllString(s, "$1")
	s = operatornamePattern.ReplaceAllString(s, "$1")
	s = oversetPattern.ReplaceAllString(s, "$2^$1")
	s = undersetPattern.ReplaceAllString(s, "$2_$1")
	s = stackrelPattern.ReplaceAllString(s, "$2=$1")
	// Accents — `\vec{x}` → `x→`, `\hat{x}` → `x̂`, etc. The precomposed
	// unicode form is used where one exists (e.g. Ĥ, x̄); otherwise we fall
	// back to a combining mark after the letter so the glyph is at least
	// distinct in the terminal.
	s = vecPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := vecPattern.FindStringSubmatch(m)
		return sub[1] + "⃗"
	})
	s = hatPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := hatPattern.FindStringSubmatch(m)
		return sub[1] + "̂"
	})
	s = tildePattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := tildePattern.FindStringSubmatch(m)
		return sub[1] + "̃"
	})
	s = barPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := barPattern.FindStringSubmatch(m)
		return sub[1] + "̄"
	})
	s = dotPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := dotPattern.FindStringSubmatch(m)
		return sub[1] + "̇"
	})
	s = ddotPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := ddotPattern.FindStringSubmatch(m)
		return sub[1] + "̈"
	})
	s = widehatPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := widehatPattern.FindStringSubmatch(m)
		return sub[1] + "̂"
	})
	s = checkPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := checkPattern.FindStringSubmatch(m)
		return sub[1] + "̌"
	})
	s = acutePattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := acutePattern.FindStringSubmatch(m)
		return sub[1] + "́"
	})
	s = gravePattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := gravePattern.FindStringSubmatch(m)
		return sub[1] + "̀"
	})
	s = brevePattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := brevePattern.FindStringSubmatch(m)
		return sub[1] + "̆"
	})
	s = overrightarrowPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := overrightarrowPattern.FindStringSubmatch(m)
		return sub[1] + "⃗"
	})
	s = overleftarrowPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := overleftarrowPattern.FindStringSubmatch(m)
		return sub[1] + "⃖"
	})
	s = underlinePattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := underlinePattern.FindStringSubmatch(m)
		return sub[1] + "̲"
	})
	s = overlinePattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := overlinePattern.FindStringSubmatch(m)
		return sub[1] + "̅"
	})
	// Font commands — translate each char in the argument through the
	// unicode mathematical alphanumeric block so bold/blackboard/script/
	// fraktur letters render distinctly. Multi-character args translate
	// char-by-char; anything outside the mapped range falls back to plain.
	s = mathbfPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := mathbfPattern.FindStringSubmatch(m)
		return translateAlphaMap(sub[1], mathAlphaToBold)
	})
	s = mathbbPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := mathbbPattern.FindStringSubmatch(m)
		return translateAlphaMap(sub[1], mathAlphaToDoubleStruck)
	})
	s = mathcalPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := mathcalPattern.FindStringSubmatch(m)
		return translateAlphaMap(sub[1], mathAlphaToScript)
	})
	s = mathfrakPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := mathfrakPattern.FindStringSubmatch(m)
		return translateAlphaMap(sub[1], mathAlphaToFraktur)
	})
	// Pre-sup-sub notation — `{}^{14}_6C` and `{}_2F_1(a,b;c;z)`. The body
	// inside the trailing `{...}` becomes the symbol; the two preceding groups
	// become superscript and subscript. Each digit is mapped through the
	// unicode sup/sub table so the layout is visible without mathcha.
	s = preSupSubPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := preSupSubPattern.FindStringSubmatch(m)
		body := sub[3]
		if body == "" {
			body = sub[4]
		}
		return toSup(sub[1]) + toSub(sub[2]) + body
	})
	s = preSubSupPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := preSubSupPattern.FindStringSubmatch(m)
		body := sub[3]
		if body == "" {
			body = sub[4]
		}
		return toSub(sub[1]) + toSup(sub[2]) + body
	})
	// \not= → ≠, \not< → ≮, etc. before cmdPattern would otherwise pass them
	// through unchanged.
	s = notPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := notPattern.FindStringSubmatch(m)
		switch sub[1] {
		case "=":
			return "≠"
		case "<", "\\le":
			return "≮"
		case ">", "\\ge":
			return "≯"
		case "\\in":
			return "∉"
		case "\\equiv":
			return "≢"
		case "\\sim":
			return "≁"
		case "\\approx":
			return "≉"
		}
		return m
	})
	return cmdPattern.ReplaceAllStringFunc(s, func(m string) string {
		name := m[1:]
		if r, ok := opMap[name]; ok {
			return r
		}
		if len(name) == 1 {
			switch name {
			case "^", "_":
				return name
			case "\\":
				return ""
			case "%":
				return ""
			case ",":
				return " "
			case ";":
				return " "
			}
			return m
		}
		return m
	})
}

func translateGreek(s string) string {
	for k, v := range greekMap {
		s = strings.ReplaceAll(s, "\\"+k, v)
	}
	return s
}

func collapseBraces(s string) string {
	for strings.Contains(s, "{}") {
		s = strings.ReplaceAll(s, "{}", "")
	}
	return s
}
