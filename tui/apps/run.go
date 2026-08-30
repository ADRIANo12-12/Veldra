// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Run builds and starts the Bubble Tea program for the Veldra TUI.

package apps

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Start launches the Veldra TUI in full-screen AltScreen mode with mouse
// motion support. It returns an error only when the program fails to run.
func Start() error {
	startDir := ""
	startFile := ""
	fs := flag.NewFlagSet("veldra", flag.ContinueOnError)
	fs.StringVar(&startDir, "dir", "", "initial directory for the Files app")
	fs.StringVar(&startFile, "editor", "", "open this file in the Editor app")
	fs.Parse(os.Args[1:])

	m := NewModel(startDir, startFile)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("Veldra TUI error: %w", err)
	}
	return nil
}
