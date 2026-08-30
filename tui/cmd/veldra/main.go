// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// veldra — the Veldra TUI. Terminal-native, full-screen, Bubble Tea +
// Lip Gloss. This is the primary user interface of Veldra.

package main

import (
	"fmt"
	"os"

	"veldra/tui/apps"
	"veldra/tui/system"
)

func main() {
	switch {
	case len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v"):
		fmt.Printf("Veldra %s  %s \u2022 %s \u2022 %s\n",
			system.Version, system.Channel, system.BuildStatus, system.ReleaseArch)
		return
	case len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h"):
		fmt.Println("Veldra TUI")
		fmt.Println("  --version   print version, channel, status and arch")
		fmt.Println("  --help      this text")
		return
	}

	if err := apps.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}