package edit

import (
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"

	"github.com/N1xev/mditor/internal/preview"
	"github.com/N1xev/mditor/internal/uict"
)

type Tab struct {
	Filename string
	Content  string
	Saved    bool
	TA       textarea.Model

	UndoStack []UndoState
	RedoStack []UndoState
	LastValue string
	LastLine  int
	LastCol   int

	Words int
	Chars int

	PreviewCache       string
	PreviewCacheSrc    string
	PreviewCacheW      int
	PreviewCodeBlocks  []string
	PreviewImageData   []preview.ImagePayload
	EmittedImagePaths  map[string]bool

	TAViewCache      string
	TAViewKeyValue   string
	TAViewKeyLine    int
	TAViewKeyCol     int
	TAViewKeyWidth   int
	TAViewKeyMaxH    int
	TAViewKeyFocused bool
}

func (t *Tab) SyncStats() {
	t.Chars = len([]rune(t.TA.Value()))
	t.Words = len(strings.Fields(t.TA.Value()))
}

func textareaStyles() textarea.Styles {
	s := textarea.DefaultStyles(true)
	s.Focused.Base = lipgloss.NewStyle().Foreground(uict.Steam)
	s.Focused.CursorLine = lipgloss.NewStyle().Background(uict.Char)
	s.Focused.LineNumber = lipgloss.NewStyle().Foreground(uict.Iron)
	s.Focused.CursorLineNumber = lipgloss.NewStyle().Foreground(uict.Squid).Background(uict.Char)
	s.Blurred.Base = lipgloss.NewStyle().Foreground(uict.Oyster)
	s.Blurred.LineNumber = lipgloss.NewStyle().Foreground(uict.Char)
	return s
}

func NewTab(filename string) Tab {
	if filename != "" {
		if abs, err := filepath.Abs(filename); err == nil {
			filename = abs
		}
	}
	ta := textarea.New()
	ta.Placeholder = "Start writing markdown…"
	ta.ShowLineNumbers = true
	ta.Prompt = " "
	km := textarea.DefaultKeyMap()
	km.WordForward = key.NewBinding(
		key.WithKeys("ctrl+right"),
		key.WithHelp("ctrl+→", "word forward"),
	)
	km.WordBackward = key.NewBinding(
		key.WithKeys("ctrl+left"),
		key.WithHelp("ctrl+←", "word backward"),
	)
	ta.KeyMap = km
	ta.MaxWidth = 0
	s := textareaStyles()
	s.Focused.Base = s.Focused.Base.Background(uict.Pepper)
	s.Blurred.Base = s.Blurred.Base.Background(uict.Pepper)
	ta.SetStyles(s)
	ta.MoveToBegin()
	ta.SetCursorColumn(0)
	t := Tab{
		Filename: filename,
		Content:  "",
		Saved:    true,
		TA:       ta,
	}
	if filename != "" {
		if data, err := LoadFile(filename); err == nil {
			t.Content = data
			t.TA.SetValue(t.Content)
			t.TA.MoveToBegin()
		}
	}
	t.SyncBaseline()
	t.SyncStats()
	return t
}

func (t Tab) DisplayName() string {
	if t.Filename == "" {
		return "untitled.md"
	}
	return filepath.Base(t.Filename)
}
