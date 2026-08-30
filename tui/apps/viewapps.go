// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Rendering for each of the five real Veldra applications.

package apps

import (
	"fmt"
	"strings"

	"veldra/tui/files"
	"veldra/tui/system"
	"veldra/tui/ui"
)

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
	}
	return "unknown application"
}

// --- Terminal (real system info) --------------------------------------------

func (m *Model) renderTerminal() string {
	info := m.sysInfo
	var b strings.Builder
	b.WriteString(m.styles.Value.Render("VELDRA CORE") + "\n")
	b.WriteString(m.styles.Label.Render("Version:  ") + m.styles.Value.Render(info.Version) + "\n")
	b.WriteString(m.styles.Label.Render("Status:   ") + m.styles.Value.Render(info.Status+" / "+info.Channel) + "\n\n")

	b.WriteString(m.styles.Section.Render("Kernel") + "\n")
	b.WriteString("  " + info.KernelFull + "\n\n")

	b.WriteString(m.styles.Section.Render("System") + "\n")
	line := m.styles.Label.Render("Hostname:  ") + m.styles.Value.Render(info.Hostname) + "\n"
	line += m.styles.Label.Render("Uptime:    ") + m.styles.Value.Render(info.Uptime) + "\n"
	line += m.styles.Label.Render("User:      ") + m.styles.Value.Render(info.CurrentUser) + "\n"
	line += m.styles.Label.Render("Shell:     ") + m.styles.Value.Render(info.Shell) + "\n"
	line += m.styles.Label.Render("OS:        ") + m.styles.Value.Render(info.OSPretty) + "\n"
	b.WriteString("  " + line + "\n")

	b.WriteString(m.styles.Section.Render("Hardware") + "\n")
	hw := m.styles.Label.Render("CPU:   ") + m.styles.Value.Render(fmt.Sprintf("%s (%d core)", info.CPUModel, info.CPUCores)) + "\n"
	hw += m.styles.Label.Render("RAM:   ") + m.styles.Value.Render(fmt.Sprintf("%.0f MiB total / %.0f MiB available",
		float64(info.MemTotal)/1024, float64(info.MemAvailable)/1024)) + "\n"
	b.WriteString("  " + hw + "\n")

	b.WriteString(m.styles.Help.Render("Press [5] to view real processes. This data comes from /proc and uname at runtime."))
	return b.String()
}

// --- Files -------------------------------------------------------------------

func (m *Model) renderFiles() string {
	var b strings.Builder
	b.WriteString(m.styles.Value.Render("▸ " + m.fs.Path) + "\n")
	if m.fs.Err != "" {
		b.WriteString(m.styles.Error.Render(m.fs.Err) + "\n")
	}
	for i, e := range m.fs.Items {
		marker := " "
		if e.IsDir {
			marker = ui.GlyphFolder
		}
		size := ""
		if !e.IsDir {
			size = files.HumanSize(e.Size)
		}
		name := e.Name
		extra := ""
		if e.IsLink {
			extra = m.styles.Label.Render(" -> " + e.LinkTo)
		}
		var cell string
		if i == m.fs.Cursor {
			cell = m.styles.Selected.Render(" " + marker + " " + name + "  " + size + " ")
			_ = extra
		} else {
			if e.IsDir {
				cell = m.styles.Dir.Render(" " + marker + " " + name + " ")
			} else {
				cell = m.styles.File.Render(" " + marker + " " + name + "  " + size + " ")
			}
		}
		b.WriteString(cell + extra + "\n")
	}
	b.WriteString("\n" + m.styles.Help.Render("[up/down] navigate  [enter] open  [left] up  [h] home  [ / ] root"))
	return b.String()
}

// --- Editor ------------------------------------------------------------------

func (m *Model) renderEditor() string {
	var b strings.Builder
	title := m.buf.Path
	if title == "" {
		title = "(untitled)"
	}
	mode := "VIEW"
	if m.insertMode {
		mode = "INSERT"
	}
	b.WriteString(m.styles.Value.Render(title+"   ["+mode+"]") + "\n")
	if m.buf.Err != "" {
		b.WriteString(m.styles.Error.Render(m.buf.Err) + "\n")
	}
	height := m.editorHeight()
	if height < 5 {
		height = 5
	}
	start := m.edView.Top
	for i := start; i < start+height && i < m.buf.Len(); i++ {
		lineNo := fmt.Sprintf("%4d ", i+1)
		var cell string
		if i == m.edView.CursorRow {
			cell = m.styles.Selected.Render(lineNo + m.buf.Line(i))
		} else {
			cell = m.styles.Label.Render(lineNo) + m.styles.File.Render(m.buf.Line(i))
		}
		b.WriteString(cell + "\n")
	}
	b.WriteString("\n" + m.styles.Help.Render("[i] insert  [ctrl+s] save  [arrow]/[pgup]/[pgdn] move  [esc/q] back"))
	return b.String()
}

// --- Settings ----------------------------------------------------------------

func (m *Model) renderSettings() string {
	info := m.sysInfo
	var b strings.Builder
	b.WriteString(m.styles.Section.Render("System") + "\n")
	b.WriteString(row("System", "Veldra OS"))
	b.WriteString(row("Version", info.Version))
	b.WriteString(row("Theme", "Veldra Dark"))
	b.WriteString(row("Kernel", info.KernelFull))
	b.WriteString(row("Hostname", info.Hostname))
	b.WriteString(row("Shell", info.Shell))
	b.WriteString(row("User", info.CurrentUser))
	b.WriteString("\n" + m.styles.Section.Render("Network") + "\n")
	if len(info.Interfaces) == 0 {
		b.WriteString(row("Interfaces", "none"))
	} else {
		for _, n := range info.Interfaces {
			state := fmt.Sprintf("%s (%s)", n.State, strings.Join(n.Addresses, ", "))
			b.WriteString(row(n.Name, state))
		}
	}
	return b.String()
}

func row(label, value string) string {
	return "  " + fmt.Sprintf("%-14s", label) + "  " + value + "\n"
}

// --- Task Manager ------------------------------------------------------------

func (m *Model) renderTasks() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%-7s %-6s %-8s %-9s  %s\n", "PID", "PPID", "CPU%", "RAM(MiB)", "PROCESS"))
	b.WriteString(strings.Repeat("─", 60) + "\n")
	for _, p := range m.lastTask {
		cpu := fmt.Sprintf("%.1f%%", p.CPU)
		line := fmt.Sprintf("%-7d %-6d %-8s %-9s  %s",
			p.PID, p.PPID, cpu, safef(p.RSS), p.Name)
		if p.User != "?" {
			line += "  (" + p.User + ")"
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + m.styles.Help.Render("Real processes from /proc — refreshes every second. Press [1] for system info."))
	return b.String()
}

func safef(kib uint64) string {
	return fmt.Sprintf("%.1f", float64(kib)/1024)
}

var _ = system.Info{}
