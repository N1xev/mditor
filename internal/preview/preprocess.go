package preview

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

func Preprocess(src string) string {
	var b strings.Builder
	parts := strings.Split(src, "```")
	for i, part := range parts {
		if i%2 == 0 {
			b.WriteString(transformOutsideFence(part))
		} else {
			b.WriteString(part)
		}
		if i < len(parts)-1 {
			b.WriteString("```")
		}
	}
	return b.String()
}

func transformOutsideFence(src string) string {
	src = stripFrontmatter(src)
	src = convertFootnotes(src)
	src = convertContainers(src)
	src = convertAbbreviations(src)
	src = convertMarkInserted(src)
	// convertMath BEFORE convertSubscriptSuperscript: the pandoc-style `^x^`
	// and `~x~` syntax uses `^` and `~` which appear inside LaTeX math as
	// superscript/sub commands and math accents. If sup/sub conversion runs
	// first, it will eat LaTeX's `e^{-x^2}` as `e{⁻ˣ2}` because supRE matches
	// `^{...x^` patterns. Running convertMath first wraps math bodies in
	// sentinels that convertSubscriptSuperscript leaves alone.
	src = convertMath(src)
	src = convertSubscriptSuperscript(src)
	src = applyTypographer(src)
	src = applySmartQuotes(src)
	src = convertEmoticons(src)
	return src
}

var frontmatterRE = regexp.MustCompile(`^---\n(?s).*?\n---\n`)

func stripFrontmatter(src string) string {
	return frontmatterRE.ReplaceAllString(src, "")
}

var footnoteDefRE = regexp.MustCompile(`(?m)^\[\^([^\]]+)\]:\s+(.*?)$`)

var footnoteRefRE = regexp.MustCompile(`\[\^([^\]]+)\]`)

func convertFootnotes(src string) string {
	defs := footnoteDefRE.FindAllStringSubmatch(src, -1)
	if len(defs) == 0 {
		return src
	}
	order := make([]string, 0, len(defs))
	idToIdx := make(map[string]int, len(defs))
	for _, m := range defs {
		id := m[1]
		if _, ok := idToIdx[id]; ok {
			continue
		}
		idToIdx[id] = len(order)
		order = append(order, id)
	}

	src = footnoteRefRE.ReplaceAllStringFunc(src, func(m string) string {
		sub := footnoteRefRE.FindStringSubmatch(m)
		id := sub[1]
		idx, ok := idToIdx[id]
		if !ok {
			return m
		}
		return supRef(idx + 1)
	})

	src = footnoteDefRE.ReplaceAllStringFunc(src, func(line string) string {
		m := footnoteDefRE.FindStringSubmatch(line)
		id := m[1]
		text := m[2]
		idx, ok := idToIdx[id]
		if !ok {
			return line
		}
		return fmt.Sprintf("  %d. %s — %s", idx+1, id, text)
	})

	return src
}

func supRef(n int) string {
	digits := []rune{'⁰', '¹', '²', '³', '⁴', '⁵', '⁶', '⁷', '⁸', '⁹'}
	if n <= 0 {
		return "⁽⁾"
	}
	var sb strings.Builder
	sb.WriteRune('⁽')
	for _, ch := range fmt.Sprintf("%d", n) {
		if ch >= '0' && ch <= '9' {
			sb.WriteRune(digits[ch-'0'])
		} else {
			sb.WriteRune(ch)
		}
	}
	sb.WriteRune('⁾')
	return sb.String()
}

var containerOpenRE = regexp.MustCompile(`(?m)^::: (\w+)\s*$`)

func convertContainers(src string) string {
	for {
		open := containerOpenRE.FindStringSubmatchIndex(src)
		if open == nil {
			return src
		}
		openStart, openEnd := open[2], open[3]
		name := src[openStart:openEnd]
		bodyStart := open[1]
		closerRE := regexp.MustCompile(`(?m)^:::\s*$`)
		closer := closerRE.FindStringIndex(src[bodyStart:])
		if closer == nil {
			return src
		}
		bodyEnd := bodyStart + closer[0]
		closerEnd := bodyStart + closer[1]
		body := src[bodyStart:bodyEnd]
		header := fmt.Sprintf("> **⚠ %s**", strings.ToUpper(name))
		var prefixBody strings.Builder
		for ln := range strings.SplitSeq(strings.TrimRight(body, "\n"), "\n") {
			prefixBody.WriteString("> ")
			prefixBody.WriteString(ln)
			prefixBody.WriteString("\n")
		}
		replacement := header + "\n>\n" + prefixBody.String()
		src = src[:open[0]] + replacement + src[closerEnd:]
	}
}

var abbrDefRE = regexp.MustCompile(`(?m)^\*\[([^\]]+)\]:\s+.*$\n?`)

func convertAbbreviations(src string) string {
	return abbrDefRE.ReplaceAllString(src, "")
}

var (
	markRE = regexp.MustCompile(`==([^+=].*?)==`)
	insRE  = regexp.MustCompile(`\+\+([^+].*?)\+\+`)
)

const (
	MarkOpen  = "«mdt-mark-open»"
	MarkClose = "«mdt-mark-close»"
)

func convertMarkInserted(src string) string {
	src = markRE.ReplaceAllString(src, MarkOpen+"$1"+MarkClose)
	src = insRE.ReplaceAllString(src, "**$1**")
	return src
}

var (
	subRE = regexp.MustCompile(`(\w)~([^~\s\\ ]{1,8})~(\w)`)
	supRE = regexp.MustCompile(`\^([^\s\\ ]{1,8})\^`)
)

var (
	subTable = map[rune]rune{
		'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄',
		'5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
		'+': '₊', '-': '₋', '=': '₌', '(': '₍', ')': '₎',
		'a': 'ₐ', 'e': 'ₑ', 'h': 'ₕ', 'i': 'ᵢ', 'j': 'ⱼ',
		'k': 'ₖ', 'l': 'ₗ', 'm': 'ₘ', 'n': 'ₙ', 'o': 'ₒ',
		'p': 'ₚ', 'r': 'ᵣ', 's': 'ₛ', 't': 'ₜ', 'u': 'ᵤ',
		'v': 'ᵥ', 'x': 'ₓ',
	}
	supTable = map[rune]rune{
		'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
		'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
		'+': '⁺', '-': '⁻', '=': '⁼', '(': '⁽', ')': '⁾',
		'a': 'ᵃ', 'b': 'ᵇ', 'c': 'ᶜ', 'd': 'ᵈ', 'e': 'ᵉ',
		'f': 'ᶠ', 'g': 'ᵍ', 'h': 'ʰ', 'i': 'ⁱ', 'j': 'ʲ',
		'k': 'ᵏ', 'l': 'ˡ', 'm': 'ᵐ', 'n': 'ⁿ', 'o': 'ᵒ',
		'p': 'ᵖ', 'r': 'ʳ', 's': 'ˢ', 't': 'ᵗ', 'u': 'ᵘ',
		'v': 'ᵛ', 'w': 'ʷ', 'x': 'ˣ', 'y': 'ʸ', 'z': 'ᶻ',
	}
)

func toSub(s string) string {
	var b strings.Builder
	for _, r := range s {
		if sub, ok := subTable[r]; ok {
			b.WriteRune(sub)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func toSup(s string) string {
	var b strings.Builder
	for _, r := range s {
		if sup, ok := supTable[r]; ok {
			b.WriteRune(sup)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func convertSubscriptSuperscript(src string) string {
	// Skip pandoc-style `^x^` / `~x~` conversion inside math sentinels so the
	// LaTeX commands `\frac{x^2}{y}` and `e^{-x^2}` inside `«§»…§»§` /
	// `«·»…·»·` blocks don't get eaten as superscript markup. We split on the
	// sentinels, convert each non-math chunk, and rejoin.
	src = convertSubscriptSuperscriptOutsideSentinels(src)
	return src
}

var mathSentinelSplitRE = regexp.MustCompile(
	regexp.QuoteMeta(MathDisplayOpen) + ".*?" + regexp.QuoteMeta(MathClose) +
		"|" + regexp.QuoteMeta(MathInlineOpen) + ".*?" + regexp.QuoteMeta(MathInlineClose),
)

func convertSubscriptSuperscriptOutsideSentinels(src string) string {
	var b strings.Builder
	last := 0
	for _, m := range mathSentinelSplitRE.FindAllStringIndex(src, -1) {
		b.WriteString(convertSubscriptSuperscriptInner(src[last:m[0]]))
		b.WriteString(src[m[0]:m[1]])
		last = m[1]
	}
	b.WriteString(convertSubscriptSuperscriptInner(src[last:]))
	return b.String()
}

func convertSubscriptSuperscriptInner(src string) string {
	src = supRE.ReplaceAllStringFunc(src, func(m string) string {
		parts := supRE.FindStringSubmatch(m)
		return toSup(parts[1])
	})
	src = subRE.ReplaceAllStringFunc(src, func(m string) string {
		parts := subRE.FindStringSubmatch(m)
		return parts[1] + toSub(parts[2]) + parts[3]
	})
	return src
}

var typographerReplacements = []struct{ from, to string }{
	{` --- `, ` — `},
	{` -- `, ` — `},
	{`...`, `…`},
	{`+-`, `±`},
	{`(c)`, `©`},
	{`(C)`, `©`},
	{`(r)`, `®`},
	{`(R)`, `®`},
	{`(tm)`, `™`},
	{`(TM)`, `™`},
	{`(p)`, `§`},
	{`(P)`, `§`},
}

var typographerSorted = sortByLenDesc(typographerReplacements)

func sortByLenDesc(pairs []struct{ from, to string }) []struct{ from, to string } {
	out := make([]struct{ from, to string }, len(pairs))
	copy(out, pairs)
	sort.SliceStable(out, func(i, j int) bool {
		return len(out[i].from) > len(out[j].from)
	})
	return out
}

var (
	openDoubleRE = regexp.MustCompile(`(^|[\s\(\[\{])"`)
	openSingleRE = regexp.MustCompile(`(^|[\s\(\[\{])'`)
)

func applySmartQuotes(src string) string {
	src = openDoubleRE.ReplaceAllString(src, "$1“")
	src = strings.ReplaceAll(src, `"`, "”")
	src = openSingleRE.ReplaceAllString(src, "$1‘")
	src = strings.ReplaceAll(src, `'`, "’")
	return src
}

func applyTypographer(src string) string {
	for _, p := range typographerSorted {
		src = strings.ReplaceAll(src, p.from, p.to)
	}
	return src
}

var emoticonReplacements = []struct{ from, to string }{
	{` 8-) `, ` 😎 `},
	{` B-) `, ` 😎 `},
	{` :-D `, ` 😀 `},
	{` :-( `, ` ☹ `},
	{` ;-) `, ` 😉 `},
	{` :-p `, ` 😛 `},
	{` :-P `, ` 😛 `},
	{` :-| `, ` 😐 `},
	{` :-/ `, ` 😕 `},
	{` :-) `, ` ☺ `},
	{` :) `, ` ☺ `},
	{` :( `, ` ☹ `},
	{` :D `, ` 😀 `},
	{` :p `, ` 😛 `},
	{` :P `, ` 😛 `},
	{` ;) `, ` 😉 `},
	{` <3 `, ` ❤ `},
	{` </3 `, ` 💔 `},
}

var emoticonSorted = sortByLenDesc(emoticonReplacements)

func convertEmoticons(src string) string {
	for _, p := range emoticonSorted {
		src = strings.ReplaceAll(src, p.from, p.to)
	}
	return src
}
