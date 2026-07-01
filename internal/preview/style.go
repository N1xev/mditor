package preview

import (
	"charm.land/glamour/v2/ansi"
	glamstyles "charm.land/glamour/v2/styles"

	"github.com/N1xev/mditor/internal/uict"
)

func boolPtr(b bool) *bool { return &b }
func uintPtr(u uint) *uint  { return &u }

func mergeDarkWithHeadings(headings ansi.StyleConfig) ansi.StyleConfig {
	base := glamstyles.DarkStyleConfig
	base.H1 = headings.H1
	base.H2 = headings.H2
	base.H3 = headings.H3
	base.H4 = headings.H4
	base.H5 = headings.H5
	base.H6 = headings.H6
	base.Heading = headings.Heading
	base.Document = headings.Document
	return base
}

func CustomHeadings() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
			},
			Margin: uintPtr(2),
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Bold:        boolPtr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           uict.SP(uict.Salt),
				BackgroundColor: uict.SP(uict.Violet),
				Bold:            boolPtr(true),
				Upper:           boolPtr(true),
				Prefix:          " ",
				Suffix:          " ",
				BlockPrefix:     "\n",
				BlockSuffix:     "\n\n",
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           uict.SP(uict.Pepper),
				BackgroundColor: uict.SP(uict.Malibu),
				Bold:            boolPtr(true),
				Prefix:          " ",
				Suffix:          " ",
				BlockPrefix:     "\n",
				BlockSuffix:     "\n",
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           uict.SP(uict.Pepper),
				BackgroundColor: uict.SP(uict.Julep),
				Bold:            boolPtr(true),
				Prefix:          " ",
				Suffix:          " ",
				BlockSuffix:     "\n",
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           uict.SP(uict.Salt),
				BackgroundColor: uict.SP(uict.Charple),
				Bold:            boolPtr(true),
				Prefix:          " ",
				Suffix:          " ",
				BlockSuffix:     "\n",
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           uict.SP(uict.Steam),
				BackgroundColor: uict.SP(uict.Char),
				Bold:            boolPtr(false),
				Prefix:          " ",
				Suffix:          " ",
				BlockSuffix:     "\n",
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           uict.SP(uict.Squid),
				BackgroundColor: uict.SP(uict.BBQ),
				Bold:            boolPtr(false),
				Faint:           boolPtr(true),
				Prefix:          " ",
				Suffix:          " ",
				BlockSuffix:     "\n",
			},
		},
	}
}
