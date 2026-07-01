package preview

import (
	"strings"

	mermaidcmd "github.com/AlexanderGrooff/mermaid-ascii/cmd"
	"github.com/AlexanderGrooff/mermaid-ascii/pkg/diagram"
	"github.com/charmbracelet/x/ansi"

	"github.com/N1xev/mditor/internal/uict"
	"charm.land/lipgloss/v2"
)

func RenderMermaid(code string, width int) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	d, err := mermaidcmd.DiagramFactory(code)
	if err != nil {
		return mermaidError(code, err.Error(), width)
	}
	if err := d.Parse(code); err != nil {
		return mermaidError(code, err.Error(), width)
	}
	cfg := diagram.DefaultConfig()
	cfg.StyleType = "cli"
	cfg.UseAscii = false
	cfg.BoxBorderPadding = 1
	cfg.PaddingBetweenX = 4
	cfg.PaddingBetweenY = 3
	out, err := d.Render(cfg)
	if err != nil {
		return mermaidError(code, err.Error(), width)
	}
	return indentMermaid(out, width)
}

func mermaidError(code, reason string, width int) string {
	if width < 8 {
		width = 8
	}
	header := lipgloss.NewStyle().
		Foreground(uict.Salt).
		Background(uict.Coral).
		Bold(true).
		Padding(0, 1).
		Render("mermaid parse error")
	reasonLine := lipgloss.NewStyle().
		Foreground(uict.Coral).
		Render(reason)
	var body strings.Builder
	body.WriteString(header)
	body.WriteRune('\n')
	body.WriteString(reasonLine)
	body.WriteRune('\n')
	for ln := range strings.SplitSeq(code, "\n") {
		body.WriteString(lipgloss.NewStyle().Foreground(uict.Squid).Render(ln))
		body.WriteRune('\n')
	}
	return body.String()
}

func indentMermaid(s string, width int) string {
	if width < 4 {
		width = 4
	}
	innerW := width - codeBlockLeftMargin
	pad := strings.Repeat(" ", codeBlockLeftMargin)
	lines := strings.Split(s, "\n")
	var b strings.Builder
	b.Grow(len(s) + len(pad)*len(lines))
	for _, ln := range lines {
		w := lipgloss.Width(ln)
		if w > innerW {
			ln = ansi.TruncateWc(ln, innerW, "")
		}
		b.WriteString(pad)
		b.WriteString(ln)
		b.WriteRune('\n')
	}
	return b.String()
}
