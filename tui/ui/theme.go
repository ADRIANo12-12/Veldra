// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Shared visual system for the Veldra terminal desktop shell.

package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Background     lipgloss.Color
	Foreground     lipgloss.Color
	Accent         lipgloss.Color
	AccentAlt      lipgloss.Color
	Muted          lipgloss.Color
	Border         lipgloss.Color
	HighlightBg    lipgloss.Color
	BarBg          lipgloss.Color
	DockActiveBg   lipgloss.Color
	DockActiveFg   lipgloss.Color
	DockInactiveFg lipgloss.Color
	Error          lipgloss.Color
	WinDir         lipgloss.Color
	WinFile        lipgloss.Color
}

func VeldraDark() Theme {
	return Theme{
		Background:     lipgloss.Color("#0b0f14"),
		Foreground:     lipgloss.Color("#d7dee8"),
		Accent:         lipgloss.Color("#8be9fd"),
		AccentAlt:      lipgloss.Color("#7fffd4"),
		Muted:          lipgloss.Color("#6f7d8c"),
		Border:         lipgloss.Color("#25303c"),
		HighlightBg:    lipgloss.Color("#17212b"),
		BarBg:          lipgloss.Color("#111820"),
		DockActiveBg:   lipgloss.Color("#17212b"),
		DockActiveFg:   lipgloss.Color("#8be9fd"),
		DockInactiveFg: lipgloss.Color("#718091"),
		Error:          lipgloss.Color("#ff6b6b"),
		WinDir:         lipgloss.Color("#8be9fd"),
		WinFile:        lipgloss.Color("#d7dee8"),
	}
}

type Styles struct {
	Window        lipgloss.Style
	TopBar        lipgloss.Style
	BarBrand      lipgloss.Style
	BarActive     lipgloss.Style
	BarItem       lipgloss.Style
	BarRight      lipgloss.Style
	Divider       lipgloss.Style
	SideTitle     lipgloss.Style
	SideActive    lipgloss.Style
	SideItem      lipgloss.Style
	PanelHeader   lipgloss.Style
	Prompt        lipgloss.Style
	TerminalInput lipgloss.Style
	Path          lipgloss.Style
	Mode          lipgloss.Style
	LineNo        lipgloss.Style
	TableHeader   lipgloss.Style
	DockItem      func(active bool) lipgloss.Style
	MenuItem      func(active bool) lipgloss.Style
	Help          lipgloss.Style
	Section       lipgloss.Style
	Value         lipgloss.Style
	Label         lipgloss.Style
	Error         lipgloss.Style
	Dir           lipgloss.Style
	File          lipgloss.Style
	Selected      lipgloss.Style
}

func NewStyles(t Theme) Styles {
	return Styles{
		Window: lipgloss.NewStyle().Background(t.Background).Foreground(t.Foreground),
		TopBar: lipgloss.NewStyle().Background(t.BarBg).Foreground(t.Foreground),
		BarBrand: lipgloss.NewStyle().Background(t.Accent).Foreground(t.Background).Bold(true),
		BarActive: lipgloss.NewStyle().Background(t.HighlightBg).Foreground(t.Accent).Bold(true),
		BarItem: lipgloss.NewStyle().Background(t.BarBg).Foreground(t.Muted),
		BarRight: lipgloss.NewStyle().Background(t.BarBg).Foreground(t.AccentAlt),
		Divider: lipgloss.NewStyle().Foreground(t.Border),
		SideTitle: lipgloss.NewStyle().Foreground(t.Muted).Bold(true),
		SideActive: lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
		SideItem: lipgloss.NewStyle().Foreground(t.Foreground),
		PanelHeader: lipgloss.NewStyle().Background(t.HighlightBg).Foreground(t.Accent).Bold(true).Padding(0, 1),
		Prompt: lipgloss.NewStyle().Foreground(t.AccentAlt).Bold(true),
		TerminalInput: lipgloss.NewStyle().Foreground(t.Foreground),
		Path: lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
		Mode: lipgloss.NewStyle().Foreground(t.AccentAlt).Bold(true),
		LineNo: lipgloss.NewStyle().Foreground(t.Muted),
		TableHeader: lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
		DockItem: func(active bool) lipgloss.Style {
			if active { return lipgloss.NewStyle().Background(t.DockActiveBg).Foreground(t.DockActiveFg).Bold(true).Padding(0, 1) }
			return lipgloss.NewStyle().Background(t.BarBg).Foreground(t.DockInactiveFg).Padding(0, 1)
		},
		MenuItem: func(active bool) lipgloss.Style {
			if active { return lipgloss.NewStyle().Background(t.HighlightBg).Foreground(t.Accent).Bold(true) }
			return lipgloss.NewStyle().Foreground(t.Foreground).Faint(true)
		},
		Help: lipgloss.NewStyle().Foreground(t.Muted),
		Section: lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
		Value: lipgloss.NewStyle().Foreground(t.AccentAlt),
		Label: lipgloss.NewStyle().Foreground(t.Muted),
		Error: lipgloss.NewStyle().Foreground(t.Error),
		Dir: lipgloss.NewStyle().Foreground(t.WinDir).Bold(true),
		File: lipgloss.NewStyle().Foreground(t.WinFile),
		Selected: lipgloss.NewStyle().Background(t.HighlightBg).Foreground(t.Accent).Bold(true),
	}
}

const (
	GlyphBrand     = "◈"
	GlyphBullet    = "●"
	GlyphDot       = "•"
	GlyphFolder    = "▸"
	GlyphArrow     = "→"
	GlyphChecked   = "●"
	GlyphUnchecked = "○"
)
