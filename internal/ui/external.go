package ui

import (
	"github.com/atotto/clipboard"
)

func clipboardWriteAll(text string) error { return clipboard.WriteAll(text) }
