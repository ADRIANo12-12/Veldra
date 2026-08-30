// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// veldra — alternate entry point for the Veldra TUI. Equivalent to the
// canonical tui/cmd/veldra binary; provided so `go build ./cmd/veldra`
// and `go build ./tui/cmd/veldra` both work.

package main

import (
	"fmt"
	"os"

	"veldra/tui/apps"
)

func main() {
	if err := apps.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
