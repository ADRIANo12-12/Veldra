// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Shared visual system for the Veldra terminal desktop shell.
package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
    Background  lipgloss.Color
    Surface     lipgloss.Color
    SurfaceAlt  lipgloss.Color
    Foreground  lipgloss.Color
    Strong      lipgloss.Color
    Accent      lipgloss.Color
    AccentAlt   lipgloss.Color
    Muted       lipgloss.Color
    Border      lipgloss.Color
    FocusBg     lipgloss.Color
    BarBg       lipgloss.Color
    BottomBg    lipgloss.Color
    Error       lipgloss.Color
    Warning     lipgloss.Color
    Success     lipgloss.Color
    MacRed      lipgloss.Color
    MacYellow   lipgloss.Color
    MacGreen    lipgloss.Color
    WinDir      lipgloss.Color
    WinFile     lipgloss.Color
}

func VeldraDark() Theme {
    return Theme{
        Background: lipgloss.Color("#090d12"),
        Surface:    lipgloss.Color("#0f151c"),
        SurfaceAlt: lipgloss.Color("#141c25"),
        Foreground: lipgloss.Color("#d8dee8"),
        Strong:     lipgloss.Color("#f4f7fb"),
        Accent:     lipgloss.Color("#8be9fd"),
        AccentAlt:  lipgloss.Color("#9ef0d0"),
        Muted:      lipgloss.Color("#6f7d8c"),
        Border:     lipgloss.Color("#202a35"),
        FocusBg:    lipgloss.Color("#182532"),
        BarBg:      lipgloss.Color("#0c1218"),
        BottomBg:   lipgloss.Color("#0a0f15"),
        Error:      lipgloss.Color("#ff6b6b"),
        Warning:    lipgloss.Color("#f4c96b"),
        Success:    lipgloss.Color("#7ee2a8"),
        MacRed:     lipgloss.Color("#ff5f57"),
        MacYellow:  lipgloss.Color("#febc2e"),
        MacGreen:   lipgloss.Color("#28c840"),
        WinDir:     lipgloss.Color("#8be9fd"),
        WinFile:    lipgloss.Color("#d8dee8"),
    }
}

type Styles struct {
    Window              lipgloss.Style
    TopBar              lipgloss.Style
    BarBrand            lipgloss.Style
    BarWorkspaceActive  lipgloss.Style
    BarWorkspace        lipgloss.Style
    BarRight            lipgloss.Style
    TitleBar            lipgloss.Style
    AppMode             lipgloss.Style
    BottomBar            lipgloss.Style
    MacRed              lipgloss.Style
    MacYellow           lipgloss.Style
    MacGreen            lipgloss.Style
    Prompt              lipgloss.Style
    TerminalInput       lipgloss.Style
    Status              lipgloss.Style
    Path                lipgloss.Style
    Mode                lipgloss.Style
    LineNo               lipgloss.Style
    TableHeader          lipgloss.Style
    Palette              lipgloss.Style
    PaletteTitle        lipgloss.Style
    PaletteInput        lipgloss.Style
    PaletteItem         lipgloss.Style
    PaletteActive       lipgloss.Style
    PaletteHint         lipgloss.Style
    Divider              lipgloss.Style
    Help                 lipgloss.Style
    Section              lipgloss.Style
    Value                lipgloss.Style
    Label                lipgloss.Style
    Error                lipgloss.Style
    Success              lipgloss.Style
    Dir                  lipgloss.Style
    File                 lipgloss.Style
    Selected             lipgloss.Style
    WindowPanel          lipgloss.Style
    SideTitle            lipgloss.Style
    SideActive           lipgloss.Style
    SideItem             lipgloss.Style
    PanelHeader          lipgloss.Style
    DockItem             func(active bool) lipgloss.Style
    MenuItem             func(active bool) lipgloss.Style
}

func NewStyles(t Theme) Styles {
    return Styles{
        Window:             lipgloss.NewStyle().Background(t.Background).Foreground(t.Foreground),
        TopBar:             lipgloss.NewStyle().Background(t.BarBg).Foreground(t.Foreground),
        BarBrand:           lipgloss.NewStyle().Background(t.Accent).Foreground(t.Background).Bold(true),
        BarWorkspaceActive: lipgloss.NewStyle().Background(t.FocusBg).Foreground(t.Accent).Bold(true).Padding(0, 1),
        BarWorkspace:       lipgloss.NewStyle().Foreground(t.Muted).Padding(0, 1),
        BarRight:           lipgloss.NewStyle().Foreground(t.AccentAlt),
        TitleBar:           lipgloss.NewStyle().Background(t.Surface).Foreground(t.Foreground),
        AppMode:            lipgloss.NewStyle().Foreground(t.Muted),
        BottomBar:          lipgloss.NewStyle().Background(t.BottomBg).Foreground(t.Muted),
        MacRed:             lipgloss.NewStyle().Foreground(t.MacRed),
        MacYellow:          lipgloss.NewStyle().Foreground(t.MacYellow),
        MacGreen:           lipgloss.NewStyle().Foreground(t.MacGreen),
        Prompt:             lipgloss.NewStyle().Foreground(t.AccentAlt).Bold(true),
        TerminalInput:      lipgloss.NewStyle().Foreground(t.Strong),
        Status:             lipgloss.NewStyle().Foreground(t.Accent),
        Path:               lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
        Mode:               lipgloss.NewStyle().Foreground(t.AccentAlt).Bold(true),
        LineNo:             lipgloss.NewStyle().Foreground(t.Muted),
        TableHeader:        lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
        Palette:            lipgloss.NewStyle().Background(t.SurfaceAlt).Foreground(t.Foreground).Border(lipgloss.RoundedBorder()).BorderForeground(t.Border).Padding(1, 2),
        PaletteTitle:       lipgloss.NewStyle().Foreground(t.Strong).Bold(true),
        PaletteInput:       lipgloss.NewStyle().Background(t.FocusBg).Foreground(t.Strong).Padding(0, 1),
        PaletteItem:        lipgloss.NewStyle().Foreground(t.Foreground).Padding(0, 1),
        PaletteActive:      lipgloss.NewStyle().Background(t.FocusBg).Foreground(t.Accent).Bold(true).Padding(0, 1),
        PaletteHint:        lipgloss.NewStyle().Foreground(t.Muted),
        Divider:            lipgloss.NewStyle().Foreground(t.Border),
        Help:               lipgloss.NewStyle().Foreground(t.Muted),
        Section:            lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
        Value:              lipgloss.NewStyle().Foreground(t.AccentAlt),
        Label:              lipgloss.NewStyle().Foreground(t.Muted),
        Error:              lipgloss.NewStyle().Foreground(t.Error),
        Success:            lipgloss.NewStyle().Foreground(t.Success),
        Dir:                lipgloss.NewStyle().Foreground(t.WinDir).Bold(true),
        File:               lipgloss.NewStyle().Foreground(t.WinFile),
        Selected:           lipgloss.NewStyle().Background(t.FocusBg).Foreground(t.Accent).Bold(true),
        WindowPanel:        lipgloss.NewStyle().Background(t.Surface).Foreground(t.Foreground),
        SideTitle:          lipgloss.NewStyle().Foreground(t.Muted).Bold(true),
        SideActive:         lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
        SideItem:           lipgloss.NewStyle().Foreground(t.Foreground),
        PanelHeader:        lipgloss.NewStyle().Background(t.FocusBg).Foreground(t.Accent).Bold(true).Padding(0, 1),
        DockItem: func(active bool) lipgloss.Style {
            if active { return lipgloss.NewStyle().Background(t.FocusBg).Foreground(t.Accent).Bold(true).Padding(0, 1) }
            return lipgloss.NewStyle().Foreground(t.Muted).Padding(0, 1)
        },
        MenuItem: func(active bool) lipgloss.Style {
            if active { return lipgloss.NewStyle().Background(t.FocusBg).Foreground(t.Accent).Bold(true) }
            return lipgloss.NewStyle().Foreground(t.Foreground)
        },
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
