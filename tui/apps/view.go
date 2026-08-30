// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// View rendering for the Veldra TUI root model: top bar, central window,
// bottom dock and status line, plus each of the five real applications.

package apps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"veldra/tui/ui"
)

// --- Top bar -----------------------------------------------------------------

func (m *Model) viewTopBar() string {
	themeItem := func(label string, active bool) string {
		s := m.styles.MenuItem(active)
		return " " + s.Render(label) + " "
	}
	// Apple-style brand glyph + menu names.
	menus := []struct {
		label string
		app   int
	}{
		{"Plik", AppFiles},
		{"Edycja", AppEditor},
		{"Widok", -1},
		{"Okno", -1},
		{"Pomoc", -1},
	}
	var left strings.Builder
	left.WriteString(m.styles.TopBar.Render(ui.GlyphBrand + "  Veldra  "))
	left.WriteString(themeItem("Plik", m.active == AppFiles))
	left.WriteString(themeItem("Edycja", m.active == AppEditor))
	left.WriteString(themeItem("Widok", false))
	left.WriteString(themeItem("Okno", false))
	left.WriteString(themeItem("Pomoc", false))
	_ = menus

	right := m.styles.TopBar.Render(
		"  " + ui.GlyphBrand + " " + m.sysInfo.Version + "  " +
			strings.ToUpper(m.sysInfo.Channel) + "  x86_64  ")

	spacing := m.width - lipgloss.Width(left.String()) - lipgloss.Width(right)
	if spacing < 1 {
		spacing = 1
	}
	pad := strings.Repeat(" ", spacing)
	return lipgloss.JoinHorizontal(lipgloss.Top, left.String(), m.styles.TopBar.Render(pad), right)
}

// --- Central window ----------------------------------------------------------

func (m *Model) viewCentral() string {
	title := dockLabels[m.active]
	content := m.viewApp()
	win := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Padding(0, 1).
		Background(m.theme.Background).
		Foreground(m.theme.Foreground).
		Width(m.centralWidth()).
		Height(m.centralHeight())

	// header: macOS traffic lights then title
	header := fmt.Sprintf(" %s%s  %s  ", ui.GlyphBullet, ui.GlyphBullet, title)
	body := header + "\n\n" + content
	return win.Render(body)
}

func (m *Model) centralWidth() int {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return w
}

func (m *Model) centralHeight() int {
	h := m.height - 4 - 2 // top bar + dock + status
	if h < 8 {
		h = 8
	}
	return h
}

// --- Dock --------------------------------------------------------------------

func (m *Model) viewDock() string {
	var cells []string
	for i, label := range dockLabels {
		active := i == m.active
		marker := ui.GlyphUnchecked
		if active {
			marker = ui.GlyphChecked
		}
		cell := m.styles.DockItem(active).Render(dockItemCell(i, label) + " " + marker)
		cells = append(cells, cell)
	}
	return lipgloss.JoinHorizontal(lipgloss.Bottom, cells...)
}

// --- Status line -------------------------------------------------------------

func (m *Model) viewStatus() string {
	parts := []string{}
	if m.savedMsg != "" {
		parts = append(parts, m.styles.Value.Render(m.savedMsg))
		m.savedMsg = ""
	}
	if m.errMsg != "" {
		parts = append(parts, m.styles.Error.Render(m.errMsg))
	}
	info := fmt.Sprintf("host: %s   uptime: %s", m.sysInfo.Hostname, m.sysInfo.Uptime)
	right := m.styles.Value.Render(info)
	left := m.styles.Help.Render(strings.Join(parts, "  "))
	spacing := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if spacing < 1 {
		spacing = 1
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", spacing), right)
}

// --- Root view ---------------------------------------------------------------

func (m *Model) View() string {
	if !m.ready {
		return "Veldra — initializing…"
	}
	body := lipgloss.JoinVertical(lipgloss.Left, m.viewTopBar(), m.viewCentral(), m.viewDock(), m.viewStatus())
	if lipgloss.Width(body) > 0 {
		return body
	}
	return body
}
