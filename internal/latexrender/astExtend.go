// Some extended parser.Expr that are useful to the editor
package renderer

import (
	parser "github.com/N1xev/mditor/internal/latex"
)

// Cursor represents an editor caret inserted into a parsed LaTeX tree.
// The struct participates in the parser.Expr interface as a regular
// Expr node, so Prerender sees a glyph (the Symbol) rather than a true
// zero-width marker. Keeping the Position fields at zero means downstream
// tree-walks that care about source ranges will treat the cursor as
// spanning no characters, which matches its UI behavior.
type Cursor struct {
	Symbol string // appearance of the cursor
}

type LatexCmdInput struct {
	Prefix string
	Text   *parser.TextStringWrapper
}

func (c *Cursor) Pos() parser.Pos { return parser.Pos(0) }
func (c *Cursor) End() parser.Pos { return parser.Pos(0) }

func (c *Cursor) VisualizeTree() string { return "Cursor" + c.Symbol }
func (c *Cursor) Content() string       { return c.Symbol }
func (c *Cursor) DeepEq(other parser.Expr) bool {
	_, ok := other.(*Cursor)
	return ok
}

func (c *Cursor) DeepEqWith(other parser.Expr, _ parser.DeepEqCfg) bool {
	_, ok := other.(*Cursor)
	return ok
}

func (x *LatexCmdInput) Pos() parser.Pos         { return 0 }
func (x *LatexCmdInput) End() parser.Pos         { return 0 }
func (x *LatexCmdInput) Children() []parser.Expr { return []parser.Expr{x.Text} }
func (x *LatexCmdInput) Parameters() int         { return 1 }
func (x *LatexCmdInput) SetArg(index int, expr parser.Expr) {
	if index > 0 {
		panic("SetArg(): index out of range")
	}
	// The parser always hands us a TextStringWrapper here. If something
	// else shows up it means a caller mixed the Node APIs in an unsupported
	// way; wrap the stray expr rather than crashing the renderer.
	if n, ok := expr.(*parser.TextStringWrapper); ok {
		x.Text = n
	} else {
		x.Text = &parser.TextStringWrapper{Runes: []parser.Expr{expr}}
	}
}
func (x *LatexCmdInput) VisualizeTree() string { return "TextContainer " + x.Text.VisualizeTree() }
func (x *LatexCmdInput) DeepEq(other parser.Expr) bool {
	if o, ok := other.(*LatexCmdInput); ok {
		return x.Text == o.Text
	}
	return false
}

func (x *LatexCmdInput) DeepEqWith(other parser.Expr, _ parser.DeepEqCfg) bool {
	if o, ok := other.(*LatexCmdInput); ok {
		return x.Text == o.Text
	}
	return false
}
