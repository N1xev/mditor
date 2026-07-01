package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/fang/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/spf13/cobra"

	"github.com/N1xev/mditor/internal/ui"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:                "mditor [file...]",
	Short:              "A modern terminal markdown editor",
	Long:               "mditor is a terminal markdown editor built with Charmbracelet ecosystem. Open one or more markdown files as tabs, edit in EDIT/MIXED/VIEW modes, and use standard copy/cut/paste.",
	Example:            "# Open the TUI with sidebar and no selected file\nmditor\n\n# Open a single file\nmditor file.md\n\n# Open multiple files as tabs\nmditor a.md b.md c.md",
	Version:            version,
	Args:               cobra.ArbitraryArgs,
	DisableFlagParsing: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		zone.NewGlobal()
		defer zone.Close()

		m := ui.NewModel(args)
		p := tea.NewProgram(m)
		_, err := p.Run()
		return err
	},
}

func main() {
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version),
		fang.WithNotifySignal(os.Interrupt),
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
