package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/N1xev/mditor/internal/uict"
)

type styles struct {
	HeaderBar   lipgloss.Style
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	TabDivider  lipgloss.Style

	TabDirtyActive   lipgloss.Style
	TabDirtyInactive lipgloss.Style
	TabCloseActive   lipgloss.Style
	TabCloseInactive lipgloss.Style

	Brand    lipgloss.Style
	BrandPad lipgloss.Style
	SlashSep lipgloss.Style

	ModeBtnInactive    lipgloss.Style
	ModeBtnActiveEdit   lipgloss.Style
	ModeBtnActiveMixed  lipgloss.Style
	ModeBtnActiveView   lipgloss.Style

	ActionBtnHot lipgloss.Style

	PaneDivider lipgloss.Style

	StatusBar lipgloss.Style
	StatKey   lipgloss.Style
	StatVal   lipgloss.Style
	StatSep   lipgloss.Style
	StatDirty lipgloss.Style
	StatFile  lipgloss.Style
	StatSel   lipgloss.Style
	StatHint  lipgloss.Style
	StatNoFile lipgloss.Style

	NotifOk  lipgloss.Style
	NotifErr lipgloss.Style

	HelpSection lipgloss.Style
	HelpKeyBind lipgloss.Style
	HelpKeyDesc lipgloss.Style
	HelpSep     lipgloss.Style

	RenameInput  lipgloss.Style
	RenamePrompt lipgloss.Style
}

func newStyles() styles {
	return styles{
		HeaderBar: lipgloss.NewStyle().
			Background(uict.BBQ),

		TabActive: lipgloss.NewStyle().
			Background(uict.Charple).
			Foreground(uict.Salt).
			Padding(0, 1).
			Bold(true),

		TabInactive: lipgloss.NewStyle().
			Background(uict.Char).
			Foreground(uict.Smoke).
			Padding(0, 1),

		TabDivider: lipgloss.NewStyle().
			Foreground(uict.Iron).
			Background(uict.BBQ),

		TabDirtyActive: lipgloss.NewStyle().
			Foreground(uict.Coral).
			Background(uict.Charple),

		TabDirtyInactive: lipgloss.NewStyle().
			Foreground(uict.Coral).
			Background(uict.Char),

		TabCloseActive: lipgloss.NewStyle().
			Foreground(uict.Coral).
			Bold(true).
			Background(uict.Charple),

		TabCloseInactive: lipgloss.NewStyle().
			Foreground(uict.Coral).
			Bold(true).
			Background(uict.Char),

		Brand: lipgloss.NewStyle().
			Foreground(uict.Violet).
			Bold(true).
			Background(uict.BBQ).
			Padding(0, 1),

		SlashSep: lipgloss.NewStyle().
			Foreground(uict.Violet).
			Background(uict.BBQ).
			Padding(0, 1, 0, 0),

		ModeBtnActiveEdit: lipgloss.NewStyle().
			Bold(true).
			Foreground(uict.Salt).
			Background(uict.Jelly).
			Padding(0, 1),

		ModeBtnActiveMixed: lipgloss.NewStyle().
			Bold(true).
			Foreground(uict.Gator).
			Background(uict.Julep).
			Padding(0, 1),

		ModeBtnActiveView: lipgloss.NewStyle().
			Bold(true).
			Foreground(uict.Pepper).
			Background(uict.Malibu).
			Padding(0, 1),

		ModeBtnInactive: lipgloss.NewStyle().
			Foreground(uict.Squid).
			UnsetBackground().
			Padding(0, 1),

		ActionBtnHot: lipgloss.NewStyle().
			Foreground(uict.Julep).
			Bold(true).
			UnsetBackground().
			Padding(0, 1),

		PaneDivider: lipgloss.NewStyle().
			Foreground(uict.Iron),

		StatusBar: lipgloss.NewStyle().
			Background(uict.Pepper).
			Foreground(uict.Squid).
			Padding(0, 2),

		StatKey: lipgloss.NewStyle().
			Foreground(uict.Iron).
			Background(uict.Pepper),

		StatVal: lipgloss.NewStyle().
			Foreground(uict.Anchovy).
			Background(uict.Pepper).
			Bold(true),

		StatSep: lipgloss.NewStyle().
			Foreground(uict.Char).
			Background(uict.Pepper),

		StatDirty: lipgloss.NewStyle().
			Foreground(uict.Coral),

		StatFile: lipgloss.NewStyle().
			Foreground(uict.Anchovy),

		StatSel: lipgloss.NewStyle().
			Foreground(uict.Violet).
			Bold(true),

		StatHint: lipgloss.NewStyle().
			Foreground(uict.Iron),

		StatNoFile: lipgloss.NewStyle().
			Foreground(uict.Iron),

		NotifOk: lipgloss.NewStyle().
			Background(uict.Gator).
			Foreground(uict.Julep).
			Padding(0, 1).
			Bold(true),

		NotifErr: lipgloss.NewStyle().
			Background(uict.Steak).
			Foreground(uict.Coral).
			Padding(0, 1).
			Bold(true),

		HelpSection: lipgloss.NewStyle().
			Foreground(uict.Hazy).
			Bold(true).
			MarginTop(1),

		HelpKeyBind: lipgloss.NewStyle().
			Foreground(uict.Charple).
			Bold(true).
			Width(18),

		HelpKeyDesc: lipgloss.NewStyle().
			Foreground(uict.Steam),

		HelpSep: lipgloss.NewStyle().
			Foreground(uict.Iron),

		RenameInput: lipgloss.NewStyle().
			Background(uict.Char).
			Foreground(uict.Salt).
			Padding(0, 1),

		RenamePrompt: lipgloss.NewStyle().
			Foreground(uict.Hazy).
			Bold(true),
	}
}
