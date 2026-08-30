// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Package ui provides the Veldra dark theme and shared Lip Gloss styles.
// A macOS-inspired, terminal-native aesthetic. Everything stays in the
// terminal: no GUI, no X11, no Wayland.

package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme holds the full set of colors and pre-built styles for the Veldra
// TUI. The active accent is cyan/turquoise, per the Veldra design.
type Theme struct {
	// Palette
	Background lipgloss.Color
	Foreground lipgloss.Color
	Accent     lipgloss.Color
	AccentAlt  lipgloss.Color
	Muted      lipgloss.Color
	Border     lipgloss.Color
	HighlightBg lipgloss.Color
	BarBg      lipgloss.Color
	DockActiveBg lipgloss.Color
	DockActiveFg lipgloss.Color
	DockInactiveFg lipgloss.Color
	Error      lipgloss.Color
	WinDir     lipgloss.Color
	WinFile    lipgloss.Color
}

// VeldraDark returns the default dark theme.
func VeldraDark() Theme {
	return Theme{
		Background:    lipgloss.Color("#101826"),
		Foreground:    lipgloss.Color("#c7d0dd"),
		Accent:        lipgloss.Color("#00d4ff"),
		AccentAlt:     lipgloss.Color("#00ffcc"),
		Muted:         lipgloss.Color("#5c6a7a"),
		Border:        lipgloss.Color("#2a3b52"),
		HighlightBg:   lipgloss.Color("#0f2333"),
		BarBg:         lipgloss.Color("#0c1624"),
		DockActiveBg:  lipgloss.Color("#10283a"),
		DockActiveFg:  lipgloss.Color("#00d4ff"),
		DockInactiveFg: lipgloss.Color("#6f7d8c"),
		Error:         lipgloss.Color("#ff5f5f"),
		WinDir:        lipgloss.Color("#00d4ff"),
		WinFile:       lipgloss.Color("#c7d0dd"),
	}
}

// Styles bundles ready-made rendered styles for convenience.
type Styles struct {
	Window    lipgloss.Style // rounded-border central window
	TopBar    lipgloss.Style
	DockItem  func(active bool) lipgloss.Style
	MenuItem  func(active bool) lipgloss.Style
	Help      lipgloss.Style
	Section   lipgloss.Style
	Value     lipgloss.Style
	Label     lipgloss.Style
	Error     lipgloss.Style
	Dir       lipgloss.Style
	File      lipgloss.Style
	Selected  lipgloss.Style
}

// NewStyles builds all styles from a theme.
func NewStyles(t Theme) Styles {
	return Styles{
		Window: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(1, 2).
			Background(t.Background).
			Foreground(t.Foreground),
		TopBar: lipgloss.NewStyle().
			Background(t.BarBg).
			Foreground(t.Foreground),
		DockItem: func(active bool) lipgloss.Style {
			if active {
				return lipgloss.NewStyle().
					Background(t.DockActiveBg).
					Foreground(t.DockActiveFg).
					Bold(true).
					Padding(0, 1)
			}
			return lipgloss.NewStyle().
				Background(t.BarBg).
				Foreground(t.DockInactiveFg).
				Padding(0, 1)
		},
		MenuItem: func(active bool) lipgloss.Style {
			if active {
				return lipgloss.NewStyle().
					Background(t.HighlightBg).
					Foreground(t.Accent).
					Bold(true)
			}
			return lipgloss.NewStyle().
				Background(t.Background).
				Foreground(t.Foreground).
				Faint(true)
		},
		Help: lipgloss.NewStyle().
			Foreground(t.Muted),
		Section: lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true),
		Value: lipgloss.NewStyle().
			Foreground(t.AccentAlt),
		Label: lipgloss.NewStyle().
			Foreground(t.Muted),
		Error: lipgloss.NewStyle().
			Foreground(t.Error),
		Dir: lipgloss.NewStyle().
			Foreground(t.WinDir).
			Bold(true),
		File: lipgloss.NewStyle().
			Foreground(t.WinFile),
		Selected: lipgloss.NewStyle().
			Background(t.HighlightBg).
			Foreground(t.Accent).
			Bold(true),
	}
}

// Unicode glyphs used by the UI.
const (
	GlyphBrand   = "V"
	GlyphBullet  = "●"
	GlyphDot     = "•"
	GlyphFolder  = "▸"
	GlyphArrow   = "→"
	GlyphChecked = "[•]"
	GlyphUnchecked = "[ ]"
)
