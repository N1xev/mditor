package ui

import (
	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	CycleMode      key.Binding
	Save           key.Binding
	New            key.Binding
	Close          key.Binding
	NextTab        key.Binding
	PrevTab        key.Binding
	Quit           key.Binding
	ToggleHelp     key.Binding
	ScrollUp       key.Binding
	ScrollDown     key.Binding
	Rename         key.Binding
	EscOverlay     key.Binding
	ToggleSidebar  key.Binding
	Undo           key.Binding
	Redo           key.Binding
	Copy           key.Binding
	Cut            key.Binding
	Paste          key.Binding
	CopyAll        key.Binding
	SelectAll      key.Binding
	Duplicate      key.Binding
	CycleFocus     key.Binding
	CycleFocusBack key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		CycleMode: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl + w", "cycle mode edit→mixed→view"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl + s", "save file"),
		),
		New: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl + n", "new tab"),
		),
		Close: key.NewBinding(
			key.WithKeys("ctrl+shift+x"),
			key.WithHelp("ctrl + shift + x", "close tab"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("alt+right"),
			key.WithHelp("alt + →", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("alt+left"),
			key.WithHelp("alt + ←", "prev tab"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+q"),
			key.WithHelp("ctrl + q", "quit"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("f1", "toggle help"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("ctrl+up"),
			key.WithHelp("ctrl + ↑", "scroll preview up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("ctrl+down"),
			key.WithHelp("ctrl + ↓", "scroll preview down"),
		),
		Rename: key.NewBinding(
			key.WithKeys("f2"),
			key.WithHelp("f2", "rename file"),
		),
		EscOverlay: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close overlay"),
		),
		ToggleSidebar: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl + b", "toggle sidebar"),
		),
		Undo: key.NewBinding(
			key.WithKeys("ctrl+z"),
			key.WithHelp("ctrl + z", "undo"),
		),
		Redo: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("ctrl + y", "redo"),
		),
		Copy: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl + c", "copy"),
		),
		Cut: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl + x", "cut"),
		),
		Paste: key.NewBinding(
			key.WithKeys("ctrl+v"),
			key.WithHelp("ctrl + v", "paste"),
		),
		CopyAll: key.NewBinding(
			key.WithKeys("ctrl+shift+a"),
			key.WithHelp("ctrl+ shift + a", "copy all"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl + a", "select all"),
		),
		Duplicate: key.NewBinding(
			key.WithKeys("ctrl+d", "ctrl+shift+d", "shift+alt+down"),
			key.WithHelp("ctrl+d/ctrl+⇧d", "duplicate line/selection"),
		),
		CycleFocus: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "cycle focus"),
		),
		CycleFocusBack: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("⇧tab", "cycle focus back"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.CycleMode, k.Save, k.New, k.ToggleSidebar, k.ToggleHelp, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.CycleMode, k.Save, k.New, k.Close},
		{k.NextTab, k.PrevTab, k.CycleFocus, k.CycleFocusBack, k.Rename, k.EscOverlay},
		{k.Copy, k.Cut, k.Paste, k.SelectAll, k.Duplicate},
		{k.CopyAll, k.Undo, k.Redo, k.ScrollUp, k.ScrollDown},
		{k.ToggleHelp, k.Quit},
	}
}
