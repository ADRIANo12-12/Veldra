// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Full-screen terminal desktop shell renderer. Inspired by tiling-WM status
// bars and keyboard-first desktop workflows, while staying entirely TUI.

package apps

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"veldra/tui/files"
	"veldra/tui/ui"
)

func (m *Model) View() string {
	if !m.ready {
		return "Veldra Shell — initializing…"
	}
	if m.width < 60 || m.height < 12 {
		return m.viewCompact()
	}

	top := m.viewTopBar()
	contentHeight := m.height - lipgloss.Height(top)
	if contentHeight < 1 {
		return top
	}
	body := m.viewWorkspace(contentHeight)
	return lipgloss.JoinVertical(lipgloss.Left, top, body)
}

func (m *Model) viewTopBar() string {
	left := m.styles.BarBrand.Render(" ◈ VELDRA ")
	workspaces := ""
	for i, label := range appLabels {
		active := i == m.active
		item := fmt.Sprintf(" %d %s ", i+1, label)
		if active {
			item = m.styles.BarActive.Render(item)
		} else {
			item = m.styles.BarItem.Render(item)
		}
		workspaces += item
	}

	now := time.Now().Format("15:04:05")
	right := m.styles.BarRight.Render(fmt.Sprintf(" %s  %s  CPU %d%%  MEM %.0fM ", now, m.sysInfo.Hostname, m.sysInfo.CPUPercent, float64(m.sysInfo.MemTotal-m.sysInfo.MemAvailable)/1024))
	used := lipgloss.Width(left) + lipgloss.Width(workspaces) + lipgloss.Width(right)
	gap := m.width - used
	if gap < 1 { gap = 1 }
	return lipgloss.JoinHorizontal(lipgloss.Top, left, workspaces, strings.Repeat(" ", gap), right)
}

func (m *Model) viewWorkspace(height int) string {
	mainHeight := height
	if mainHeight < 4 { mainHeight = 4 }

	sidebarWidth := 22
	mainWidth := m.width - sidebarWidth - 1
	if mainWidth < 25 { mainWidth = 25 }

	side := m.viewSidebar(mainHeight, sidebarWidth)
	main := m.viewMainPanel(mainHeight, mainWidth)
	return lipgloss.JoinHorizontal(lipgloss.Top, side, m.styles.Divider.Render("│"), main)
}

func (m *Model) viewSidebar(height, width int) string {
	lines := []string{m.styles.SideTitle.Render("WORKSPACES")}
	for i, label := range appLabels {
		prefix := "   "
		if i == m.active { prefix = " ● " }
		line := prefix + fmt.Sprintf("%d  %s", i+1, label)
		if i == m.active {
			line = m.styles.SideActive.Render(line)
		} else {
			line = m.styles.SideItem.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", m.styles.SideTitle.Render("SYSTEM"))
	lines = append(lines,
		fmt.Sprintf("  Kernel  %s", m.sysInfo.Kernel),
		fmt.Sprintf("  User    %s", m.sysInfo.CurrentUser),
		fmt.Sprintf("  Shell   %s", m.sysInfo.Shell),
	)
	body := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Padding(1, 1).Render(body)
}

func (m *Model) viewMainPanel(height, width int) string {
	header := m.styles.PanelHeader.Render(fmt.Sprintf(" %s  %s", appLabels[m.active], m.panelHint()))
	contentHeight := height - lipgloss.Height(header) - 1
	if contentHeight < 2 { contentHeight = 2 }
	content := m.viewApp()
	content = clipLines(content, contentHeight, width-2)
	panel := lipgloss.NewStyle().Width(width-1).Height(contentHeight).Padding(1, 1).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, header, panel)
}

func (m *Model) panelHint() string {
	switch m.active {
	case AppTerminal: return "interactive shell"
	case AppFiles: return "file manager"
	case AppEditor: return "text editor"
	case AppSettings: return "system control center"
	case AppTasks: return "live process monitor"
	default: return ""
	}
}

func (m *Model) viewCompact() string {
	head := fmt.Sprintf(" VELDRA  •  %s  •  %s ", appLabels[m.active], time.Now().Format("15:04:05"))
	body := clipLines(m.viewApp(), m.height-2, m.width)
	return lipgloss.JoinVertical(lipgloss.Left, m.styles.BarBrand.Render(head), body)
}

func clipLines(s string, maxLines, maxWidth int) string {
	if maxLines < 1 { return "" }
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if len(out) >= maxLines { break }
		if lipgloss.Width(line) > maxWidth {
			line = lipgloss.NewStyle().Width(maxWidth).MaxWidth(maxWidth).Render(line)
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// --- Application renderers --------------------------------------------------

func (m *Model) viewApp() string {
	switch m.active {
	case AppTerminal:
		return m.renderTerminal()
	case AppFiles:
		return m.renderFiles()
	case AppEditor:
		return m.renderEditor()
	case AppSettings:
		return m.renderSettings()
	case AppTasks:
		return m.renderTasks()
	default:
		return "unknown application"
	}
}

func (m *Model) renderTerminal() string {
	var b strings.Builder
	for _, line := range m.terminalOut {
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.Prompt.Render(m.prompt()+" ") + m.styles.TerminalInput.Render(m.terminalInput+"▌"))
	return b.String()
}

func (m *Model) renderFiles() string {
	var b strings.Builder
	b.WriteString(m.styles.Path.Render("▸ "+m.fs.Path) + "\n\n")
	for i, e := range m.fs.Items {
		marker := "  "
		name := e.Name
		if e.IsDir { marker = "▸ "; name += "/" }
		suffix := ""
		if !e.IsDir { suffix = files.HumanSize(e.Size) }
		line := fmt.Sprintf("%s%-44s %10s", marker, name, suffix)
		if i == m.fs.Cursor { line = m.styles.Selected.Render("▸ "+name) }
		b.WriteString(line + "\n")
	}
	b.WriteString("\n"+m.styles.Help.Render("↑/↓ navigate  Enter open  ← parent  h home  / root  1-5 workspace"))
	return b.String()
}

func (m *Model) renderEditor() string {
	var b strings.Builder
	title := m.buf.Path
	if title == "" { title = "(untitled)" }
	mode := "VIEW"
	if m.insertMode { mode = "INSERT" }
	b.WriteString(m.styles.Path.Render(title)+m.styles.Mode.Render("  ["+mode+"]")+"\n\n")
	if m.buf.Err != "" { b.WriteString(m.styles.Error.Render(m.buf.Err)+"\n") }
	height := m.editorHeight()
	start := m.edView.Top
	for i := start; i < start+height && i < m.buf.Len(); i++ {
		lineNo := fmt.Sprintf("%4d ", i+1)
		if i == m.edView.CursorRow {
			b.WriteString(m.styles.Selected.Render(lineNo+m.buf.Line(i))+"\n")
		} else {
			b.WriteString(m.styles.LineNo.Render(lineNo)+m.styles.File.Render(m.buf.Line(i))+"\n")
		}
	}
	b.WriteString("\n"+m.styles.Help.Render("i insert  Ctrl-S save  arrows move  Esc view  3 editor workspace"))
	return b.String()
}

func (m *Model) renderSettings() string {
	info := m.sysInfo
	var b strings.Builder
	b.WriteString(m.styles.Section.Render("SYSTEM")+"\n")
	b.WriteString(row("OS", "Veldra OS"))
	b.WriteString(row("Version", info.Version))
	b.WriteString(row("Channel", info.Channel))
	b.WriteString(row("Kernel", info.KernelFull))
	b.WriteString(row("Hostname", info.Hostname))
	b.WriteString(row("User", info.CurrentUser))
	b.WriteString(row("Shell", info.Shell))
	b.WriteString("\n"+m.styles.Section.Render("NETWORK")+"\n")
	if len(info.Interfaces) == 0 { b.WriteString(row("Interfaces", "none")) }
	for _, n := range info.Interfaces {
		b.WriteString(row(n.Name, fmt.Sprintf("%s (%s)", n.State, strings.Join(n.Addresses, ", "))))
	}
	return b.String()
}

func row(label, value string) string { return fmt.Sprintf("  %-16s  %s\n", label, value) }

func (m *Model) renderTasks() string {
	var b strings.Builder
	b.WriteString(m.styles.TableHeader.Render(fmt.Sprintf("%-7s %-7s %-8s %-10s %s", "PID", "PPID", "CPU", "RAM", "PROCESS"))+"\n")
	b.WriteString(m.styles.Divider.Render(strings.Repeat("─", 72))+"\n")
	for _, p := range m.lastTask {
		b.WriteString(fmt.Sprintf("%-7d %-7d %-8s %-10s %s", p.PID, p.PPID, fmt.Sprintf("%.1f%%", p.CPU), safef(p.RSS), p.Name)+"\n")
	}
	return b.String()
}

func safef(kib uint64) string { return fmt.Sprintf("%.1fM", float64(kib)/1024) }

var _ = ui.GlyphBullet
