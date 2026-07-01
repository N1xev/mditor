package uict

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Light/dark variants for every color in the palette. Lipgloss picks Dark
// when the terminal reports a dark background and Light otherwise. The
// vibrant colors (Charple, Jelly, ...) read well on dark; on a light
// background we darken them slightly for contrast. The grayscale ramp inverts:
// Pepper (almost black) stays readable as text on both themes, while
// backgrounds that were near-black on dark theme become near-white on light.
func a(dark, light string) compat.AdaptiveColor {
	return compat.AdaptiveColor{Light: lipgloss.Color(light), Dark: lipgloss.Color(dark)}
}

var (
	Charple = a("#6B50FF", "#4A30D9") // primary accent
	Hazy    = a("#8B75FF", "#5B40EC")
	Jelly   = a("#4A30D9", "#2E1FA8")
	Darple  = a("#5B40EC", "#3D26B6")
	Larple  = a("#7B62FF", "#4A30D9")
	Violet  = a("#C259FF", "#8E2ECC")
	Orchid  = a("#AD6EFF", "#7B47BF")

	Malibu  = a("#00A4FF", "#006BB3")
	Anchovy = a("#719AFC", "#3D6BD9")
	Sardine = a("#4FBEFE", "#1F8FCC")

	Turtle = a("#0ADCD9", "#068A88")
	Lichen = a("#5CDFEA", "#1FA8B0")

	Julep = a("#00FFB2", "#00A475")
	Bok   = a("#68FFD6", "#1FB893")
	Guac  = a("#12C78F", "#068558")

	Coral    = a("#FF577D", "#CC1F4A")
	Cherry   = a("#FF388B", "#B81F5F")
	Sriracha = a("#EB4268", "#B81F3D")
	Tuna     = a("#FF6DAA", "#CC3973")

	Tang    = a("#FF985A", "#CC5F1F")
	Yam     = a("#FFB587", "#CC7A47")
	Mustard = a("#F5EF34", "#A89F0F")
	Zest    = a("#E8FE96", "#8BA83D")
	Butter  = a("#FFFAF1", "#E8D9BF")

	Cumin = a("#BF976F", "#7B5F3D")
	Uni   = a("#FF937D", "#CC5F47")
)

// Grayscale ramp. Names are kept (Salt, Pepper, ...) but Light/Dark values
// are mirrored so the same name reads as "the near-white" on dark themes
// and "the near-black" on light themes. Squid is a true mid-gray that
// stays put across themes.
var (
	Pepper = a("#201F26", "#201F26") // near-black text on light, dark accent
	BBQ    = a("#2D2C36", "#EAE8F0") // subtle background
	Char   = a("#3A3943", "#D6D3DC") // border / divider
	Iron   = a("#4D4C57", "#BFBCC8") // muted text
	Oyster = a("#605F6B", "#A2A0AD") // secondary text
	Squid  = a("#858392", "#858392") // mid-gray, theme-neutral
	Steam  = a("#A2A0AD", "#605F6B") // bright text on dark, dim on light
	Smoke  = a("#BFBCC8", "#4D4C57")
	Steep  = a("#D6D3DC", "#3A3943")
	Salt   = a("#F7F6FB", "#1A1922") // primary foreground: light on dark, near-black on light
)

// Secondary palette (used for inline marks, callouts, selected rows).
var (
	Spinach = a("#1C3634", "#D6EDE9")
	Gator   = a("#18463D", "#BFDFD8")
	Pickle  = a("#00A475", "#008058")
	Toast   = a("#412130", "#F5DDE3")
	Steak   = a("#582238", "#F2D2DD")
	Pom     = a("#AB2454", "#8B1F45")
)

func C(name string) color.Color { return lipgloss.Color(name) }

func Hex(c color.Color) string {
	r, g, b, _ := color.RGBAModel.Convert(c).RGBA()
	return fmt.Sprintf("#%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

func SP(c color.Color) *string {
	if c == nil {
		return nil
	}
	s := Hex(c)
	return &s
}