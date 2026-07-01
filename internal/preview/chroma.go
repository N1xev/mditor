package preview

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/N1xev/mditor/internal/uict"
)

var chromaStyle = chroma.MustNewStyle("mditor-charmtone", chroma.StyleEntries{
	chroma.Text:  uict.Hex(uict.Steam),
	chroma.Error: uict.Hex(uict.Coral),
	chroma.Other: uict.Hex(uict.Steam),

	chroma.Comment:            uict.Hex(uict.Squid) + " italic",
	chroma.CommentHashbang:    uict.Hex(uict.Lichen),
	chroma.CommentMultiline:   uict.Hex(uict.Squid) + " italic",
	chroma.CommentSingle:      uict.Hex(uict.Squid) + " italic",
	chroma.CommentSpecial:     uict.Hex(uict.Lichen),
	chroma.CommentPreproc:     uict.Hex(uict.Lichen),
	chroma.CommentPreprocFile: uict.Hex(uict.Lichen),

	chroma.Keyword:            uict.Hex(uict.Violet),
	chroma.KeywordConstant:    uict.Hex(uict.Orchid),
	chroma.KeywordDeclaration: uict.Hex(uict.Violet),
	chroma.KeywordNamespace:   uict.Hex(uict.Charple),
	chroma.KeywordPseudo:      uict.Hex(uict.Orchid) + " italic",
	chroma.KeywordReserved:    uict.Hex(uict.Violet),
	chroma.KeywordType:        uict.Hex(uict.Malibu),

	chroma.Name:                  uict.Hex(uict.Steam),
	chroma.NameAttribute:         uict.Hex(uict.Mustard),
	chroma.NameBuiltin:           uict.Hex(uict.Malibu),
	chroma.NameBuiltinPseudo:     uict.Hex(uict.Smoke) + " italic",
	chroma.NameClass:             uict.Hex(uict.Mustard),
	chroma.NameConstant:          uict.Hex(uict.Orchid),
	chroma.NameDecorator:         uict.Hex(uict.Tang),
	chroma.NameEntity:            uict.Hex(uict.Mustard),
	chroma.NameException:         uict.Hex(uict.Coral),
	chroma.NameFunction:          uict.Hex(uict.Julep),
	chroma.NameFunctionMagic:     uict.Hex(uict.Lichen) + " italic",
	chroma.NameKeyword:           uict.Hex(uict.Smoke),
	chroma.NameLabel:             uict.Hex(uict.Tang),
	chroma.NameNamespace:         uict.Hex(uict.Charple),
	chroma.NameOperator:          uict.Hex(uict.Smoke),
	chroma.NameOther:             uict.Hex(uict.Steam),
	chroma.NamePseudo:            uict.Hex(uict.Smoke) + " italic",
	chroma.NameProperty:          uict.Hex(uict.Malibu),
	chroma.NameTag:               uict.Hex(uict.Tang),
	chroma.NameVariable:          uict.Hex(uict.Steam),
	chroma.NameVariableAnonymous: uict.Hex(uict.Smoke) + " italic",
	chroma.NameVariableClass:     uict.Hex(uict.Mustard),
	chroma.NameVariableGlobal:    uict.Hex(uict.Mustard),
	chroma.NameVariableInstance:  uict.Hex(uict.Steam),
	chroma.NameVariableMagic:     uict.Hex(uict.Lichen) + " italic",

	chroma.Literal:     uict.Hex(uict.Yam),
	chroma.LiteralDate: uict.Hex(uict.Tang),

	chroma.LiteralString:          uict.Hex(uict.Julep),
	chroma.LiteralStringAffix:     uict.Hex(uict.Lichen),
	chroma.LiteralStringAtom:      uict.Hex(uict.Bok),
	chroma.LiteralStringBacktick:  uict.Hex(uict.Bok),
	chroma.LiteralStringBoolean:   uict.Hex(uict.Orchid),
	chroma.LiteralStringChar:      uict.Hex(uict.Bok),
	chroma.LiteralStringDelimiter: uict.Hex(uict.Lichen),
	chroma.LiteralStringDoc:       uict.Hex(uict.Squid) + " italic",
	chroma.LiteralStringDouble:    uict.Hex(uict.Julep),
	chroma.LiteralStringEscape:    uict.Hex(uict.Lichen),
	chroma.LiteralStringHeredoc:   uict.Hex(uict.Julep),
	chroma.LiteralStringInterpol:  uict.Hex(uict.Lichen),
	chroma.LiteralStringName:      uict.Hex(uict.Bok),
	chroma.LiteralStringOther:     uict.Hex(uict.Julep),
	chroma.LiteralStringRegex:     uict.Hex(uict.Tang),
	chroma.LiteralStringSingle:    uict.Hex(uict.Julep),
	chroma.LiteralStringSymbol:    uict.Hex(uict.Bok),

	chroma.LiteralNumber:            uict.Hex(uict.Malibu),
	chroma.LiteralNumberBin:         uict.Hex(uict.Turtle),
	chroma.LiteralNumberFloat:       uict.Hex(uict.Malibu),
	chroma.LiteralNumberHex:         uict.Hex(uict.Sardine),
	chroma.LiteralNumberInteger:     uict.Hex(uict.Malibu),
	chroma.LiteralNumberIntegerLong: uict.Hex(uict.Malibu),
	chroma.LiteralNumberOct:         uict.Hex(uict.Turtle),

	chroma.Operator:     uict.Hex(uict.Smoke),
	chroma.OperatorWord: uict.Hex(uict.Violet),
	chroma.Punctuation:  uict.Hex(uict.Steam),

	chroma.Generic:           uict.Hex(uict.Steam),
	chroma.GenericDeleted:    uict.Hex(uict.Coral),
	chroma.GenericEmph:       uict.Hex(uict.Salt) + " italic",
	chroma.GenericError:      uict.Hex(uict.Coral),
	chroma.GenericHeading:    uict.Hex(uict.Salt) + " bold",
	chroma.GenericInserted:   uict.Hex(uict.Julep),
	chroma.GenericOutput:     uict.Hex(uict.Squid),
	chroma.GenericPrompt:     uict.Hex(uict.Malibu),
	chroma.GenericStrong:     uict.Hex(uict.Salt) + " bold",
	chroma.GenericSubheading: uict.Hex(uict.Violet),
	chroma.GenericTraceback:  uict.Hex(uict.Coral),
	chroma.GenericUnderline:  uict.Hex(uict.Salt) + " underline",
})

func HighlightCode(lang, code string) string {
	lexer := selectLexer(lang, code)
	if lexer == nil {
		return code
	}
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}
	var sb strings.Builder
	sb.Grow(len(code) * 2)
	for _, token := range iterator.Tokens() {
		sb.WriteString(renderToken(token))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderToken(token chroma.Token) string {
	entry := chromaStyle.Get(token.Type)
	style := lipgloss.NewStyle().Background(uict.BBQ)
	if entry.Colour.IsSet() {
		style = style.Foreground(lipgloss.Color(entry.Colour.String()))
	}
	if entry.Bold == chroma.Yes {
		style = style.Bold(true)
	}
	if entry.Italic == chroma.Yes {
		style = style.Italic(true)
	}
	if entry.Underline == chroma.Yes {
		style = style.Underline(true)
	}
	return style.Render(token.Value)
}

func selectLexer(lang, code string) chroma.Lexer {
	if lang != "" {
		if l := lexers.Get(lang); l != nil {
			return l
		}
	}
	if l := lexers.Analyse(code); l != nil {
		return l
	}
	return lexers.Fallback
}
