package ui

import (
	"github.com/charmbracelet/x/ansi"
)

func ansiTruncateWc(s string, width int, tail string) string    { return ansi.TruncateWc(s, width, tail) }
func ansiTruncateLeftWc(s string, width int, tail string) string { return ansi.TruncateLeftWc(s, width, tail) }
func ansiStrip(s string) string                                  { return ansi.Strip(s) }
func ansiStringWidth(s string) int                               { return ansi.StringWidth(s) }
