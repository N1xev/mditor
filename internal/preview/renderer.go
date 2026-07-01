package preview

import (
	"fmt"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/mattn/go-runewidth"

	"github.com/N1xev/mditor/internal/uict"
)

const CopyZonePrefix = "copy-code-"

func CopyZoneID(blockID int) string {
	return fmt.Sprintf("%s%d", CopyZonePrefix, blockID)
}

type Renderer struct {
	g *glamour.TermRenderer
}

func NewRenderer(width int) (*Renderer, error) {
	if width < 20 {
		width = 20
	}
	merged := mergeDarkWithHeadings(CustomHeadings())
	g, err := glamour.NewTermRenderer(
		glamour.WithStyles(merged),
		glamour.WithWordWrap(width),
		glamour.WithChromaFormatter("terminal256"),
		glamour.WithPreservedNewLines(),
		glamour.WithEmoji(),
	)
	if err != nil {
		return nil, err
	}
	return &Renderer{g: g}, nil
}

func (r *Renderer) Render(md string) (string, error) {
	out, _, err := r.RenderWithBaseDir(md, "", 80, 24)
	return out, err
}

func (r *Renderer) RenderWithBaseDir(md, baseDir string, cellW, cellH int) (string, []ImagePayload, error) {
	src, specs := PreprocessImages(Preprocess(md), baseDir)
	out, err := r.g.Render(src)
	if err != nil {
		return out, nil, err
	}
	rendered, payloads := RenderImages(applyMathPost(applyMarkPost(out)), specs, cellW, cellH)
	return rendered, payloads, nil
}

type CodeBlock struct {
	Code    string
	Lang    string
	BlockID int
}

func (r *Renderer) RenderWithCodeBlocks(raw string, width int) (string, []CodeBlock, []ImagePayload) {
	return r.RenderWithCodeBlocksBase(raw, "", width, width, 24)
}

func (r *Renderer) RenderWithCodeBlocksBase(raw, baseDir string, cellW, cellH, codeW int) (string, []CodeBlock, []ImagePayload) {
	var result strings.Builder
	var blocks []CodeBlock

	src, specs := PreprocessImages(Preprocess(raw), baseDir)
	parts := strings.Split(src, "```")
	for i, part := range parts {
		if i%2 == 0 {
			out, err := r.g.Render(part)
			if err != nil {
				result.WriteString(part)
			} else {
				result.WriteString(applyMathPost(applyMarkPost(out)))
			}
			continue
		}
		lines := strings.Split(part, "\n")
		lang := ""
		codeLines := lines
		if len(lines) > 0 {
			lang = strings.TrimSpace(lines[0])
			codeLines = lines[1:]
		}
		for len(codeLines) > 0 && codeLines[len(codeLines)-1] == "" {
			codeLines = codeLines[:len(codeLines)-1]
		}
		codeContent := strings.Join(codeLines, "\n")
		block := CodeBlock{
			Code:    codeContent,
			Lang:    lang,
			BlockID: len(blocks),
		}
		blocks = append(blocks, block)
		result.WriteString(RenderCodeBlock(block, codeW))
		result.WriteRune('\n')
	}
	rendered, payloads := RenderImages(result.String(), specs, cellW, cellH)
	return rendered, blocks, payloads
}

var (
	copyBtnStyle = lipgloss.NewStyle().
			Foreground(uict.Julep).
			Bold(true)

	badgeStyle = lipgloss.NewStyle().
			Foreground(uict.Pepper).
			Background(uict.Violet).
			Bold(true).
			Padding(0, 1)

	panelWidthStyle = lipgloss.NewStyle()

	panelHeaderRightStyle = lipgloss.NewStyle().Align(lipgloss.Right)
)

const codeBlockLeftMargin = 2

var codeBlockWrap = lipgloss.NewStyle().MarginLeft(codeBlockLeftMargin)

func RenderCodeBlock(b CodeBlock, width int) string {
	lang := b.Lang
	if lang == " " {
		lang = ""
	}
	if width < 8 {
		width = 8
	}
	innerW := width - codeBlockLeftMargin
	headerW := max(1, innerW-4)

	copyBtn := zone.Mark(
		CopyZoneID(b.BlockID),
		copyBtnStyle.Render("[copy]"),
	)

	var header string
	if lang != "" {
		badge := badgeStyle.Render(lang)
		badgeW := lipgloss.Width(badge)
		copyW := lipgloss.Width(copyBtn)
		gap := max(headerW-badgeW-copyW, 1)
		row := badge + strings.Repeat(" ", gap) + copyBtn
		header = panelWidthStyle.Width(innerW).Render(row)
	} else {
		inner := panelHeaderRightStyle.Width(headerW).Render(copyBtn)
		header = panelWidthStyle.Width(innerW).Render(inner)
	}

	if b.Code == "" {
		return codeBlockWrap.Render(header)
	}
	if isMermaidLang(lang) {
		body := RenderMermaid(b.Code, width)
		return codeBlockWrap.Render(header + "\n" + body)
	}
	highlighted := HighlightCode(lang, b.Code)
	return codeBlockWrap.Render(header + "\n" + renderCodeBody(highlighted, headerW))
}

func isMermaidLang(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "mermaid", "mmd":
		return true
	}
	return false
}

func renderCodeBody(highlighted string, innerWidth int) string {
	if innerWidth < 4 {
		innerWidth = 4
	}
	contentWidth := innerWidth - 4

	bgStyle := lipgloss.NewStyle().Background(uict.BBQ)
	leftPad := bgStyle.Width(2).Render("")
	rightPad := bgStyle.Width(2).Render("")
	fullRow := bgStyle.Width(innerWidth).Render("")

	lines := strings.Split(highlighted, "\n")
	var b strings.Builder
	b.Grow(len(highlighted) + innerWidth*4)
	b.WriteString(fullRow)
	b.WriteRune('\n')
	for _, line := range lines {
		if w := ansi.StringWidth(line); w > contentWidth {
			line = ansi.TruncateWc(line, contentWidth, "")
		}
		b.WriteString(leftPad)
		b.WriteString(bgStyle.Width(contentWidth).Render(line))
		b.WriteString(rightPad)
		b.WriteRune('\n')
	}
	b.WriteString(fullRow)
	return b.String()
}

func CellToRune(s string, cell int) int {
	cells := 0
	for i, r := range s {
		if cells == cell {
			return i
		}
		cells += runewidth.RuneWidth(r)
	}
	return len(s)
}
