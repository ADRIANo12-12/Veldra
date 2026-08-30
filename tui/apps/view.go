// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Full-screen renderer for the Veldra terminal desktop shell.
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
    if !m.ready || m.width < 1 || m.height < 1 {
        return ""
    }
    topH, titleH, bottomH := 1, 1, 1
    bodyH := m.height - topH - titleH - bottomH
    if bodyH < 1 { bodyH = 1 }

    screen := lipgloss.JoinVertical(lipgloss.Left,
        m.viewTopBar(),
        m.viewTitleBar(),
        m.viewBody(bodyH),
        m.viewBottomBar(),
    )
    return fitScreen(screen, m.width, m.height, m.styles.Window)
}

func (m *Model) viewTopBar() string {
    left := m.styles.BarBrand.Render(" ◈ VELDRA ")
    center := ""
    for i, cell := range m.topBarCells() {
        if i == m.active {
            center += m.styles.BarWorkspaceActive.Render(cell)
        } else {
            center += m.styles.BarWorkspace.Render(cell)
        }
    }
    mem := float64(0)
    if m.sysInfo.MemTotal >= m.sysInfo.MemAvailable {
        mem = float64(m.sysInfo.MemTotal-m.sysInfo.MemAvailable) / 1024
    }
    right := m.styles.BarRight.Render(fmt.Sprintf(" %s  %s  MEM %.0fM ", time.Now().Format("15:04"), m.sysInfo.Hostname, mem))
    gap := m.width - lipgloss.Width(left) - lipgloss.Width(center) - lipgloss.Width(right)
    if gap < 1 { gap = 1 }
    return m.styles.TopBar.Width(m.width).Render(left + center + strings.Repeat(" ", gap) + right)
}

func (m *Model) viewTitleBar() string {
    traffic := strings.Join([]string{
        m.styles.MacRed.Render("●"),
        m.styles.MacYellow.Render("●"),
        m.styles.MacGreen.Render("●"),
    }, " ")
    title := fmt.Sprintf("   %s   ·   %s", appLabels[m.active], m.panelHint())
    right := m.styles.AppMode.Render(m.editorStatus())
    gap := m.width - lipgloss.Width(traffic) - lipgloss.Width(title) - lipgloss.Width(right)
    if gap < 1 { gap = 1 }
    return m.styles.TitleBar.Width(m.width).Render(traffic + title + strings.Repeat(" ", gap) + right)
}

func (m *Model) panelHint() string {
    switch m.active {
    case AppTerminal: return "command workspace"
    case AppFiles: return "filesystem"
    case AppEditor: return "editor"
    case AppSettings: return "control center"
    case AppTasks: return "process monitor"
    default: return ""
    }
}

func (m *Model) viewBody(height int) string {
    if m.paletteOpen { return m.viewPalette(height) }
    content := clipLines(m.viewApp(), height-2, m.width-4)
    return lipgloss.NewStyle().Width(m.width).Height(height).Padding(1, 2).Render(content)
}

func (m *Model) viewBottomBar() string {
    status := "ready"
    if m.terminalRunning { status = "running…" }
    left := m.styles.Status.Render(" " + status)
    center := m.styles.Help.Render("Ctrl+P commands   Ctrl+1…5 workspaces   Tab next")
    right := m.styles.Help.Render("Ctrl+Q quit ")
    gap := m.width - lipgloss.Width(left) - lipgloss.Width(center) - lipgloss.Width(right)
    if gap < 1 { gap = 1 }
    return m.styles.BottomBar.Width(m.width).Render(left + strings.Repeat(" ", gap/2) + center + strings.Repeat(" ", gap-gap/2) + right)
}

func (m *Model) viewPalette(height int) string {
    items := m.filteredPaletteActions()
    width := m.width - 12
    if width > 82 { width = 82 }
    if width < 36 { width = m.width - 4 }
    if width < 10 { width = 10 }

    lines := []string{m.styles.PaletteTitle.Render("COMMAND PALETTE")}
    lines = append(lines, m.styles.PaletteInput.Render("❯ "+string(m.paletteInput[:m.paletteCursor])+"▌"+string(m.paletteInput[m.paletteCursor:])), "")
    for i, a := range items {
        line := fmt.Sprintf(" %-22s  %-34s  %s", a.Label, a.Hint, a.Key)
        if i == m.paletteCursor { lines = append(lines, m.styles.PaletteActive.Render(line)) } else { lines = append(lines, m.styles.PaletteItem.Render(line)) }
    }
    if len(items) == 0 { lines = append(lines, m.styles.Help.Render("No commands match your query.")) }
    lines = append(lines, "", m.styles.PaletteHint.Render("↑/↓ select   Enter run   Esc close"))

    panel := m.styles.Palette.Width(width).Render(strings.Join(lines, "\n"))
    pad := (height - lipgloss.Height(panel)) / 2
    if pad < 0 { pad = 0 }
    return strings.Repeat("\n", pad) + lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(panel)
}

func (m *Model) viewApp() string {
    switch m.active {
    case AppTerminal: return m.renderTerminal()
    case AppFiles: return m.renderFiles()
    case AppEditor: return m.renderEditor()
    case AppSettings: return m.renderSettings()
    case AppTasks: return m.renderTasks()
    default: return "unknown workspace"
    }
}

func (m *Model) renderTerminal() string {
    max := m.height - 6
    if max < 1 { max = 1 }
    out := m.terminalOut
    if len(out) > max-2 { out = out[len(out)-(max-2):] }
    var b strings.Builder
    for _, line := range out { b.WriteString(cleanOutput(line)); b.WriteByte('\n') }
    b.WriteString("\n")
    prompt := m.styles.Prompt.Render(m.prompt()+" ")
    input := string(m.terminalInput[:m.terminalCursor]) + "▌" + string(m.terminalInput[m.terminalCursor:])
    b.WriteString(prompt + m.styles.TerminalInput.Render(input))
    return b.String()
}

func (m *Model) renderFiles() string {
    var b strings.Builder
    b.WriteString(m.styles.Path.Render("▸ "+m.fs.Path)+"\n\n")
    for i, e := range m.fs.Items {
        name := e.Name
        marker := "  "
        if e.IsDir { marker = "▸ "; name += "/" }
        if i == m.fs.Cursor {
            b.WriteString(m.styles.Selected.Render("▸ "+name)+"\n")
        } else {
            size := ""
            if !e.IsDir { size = files.HumanSize(e.Size) }
            b.WriteString(m.styles.File.Render(fmt.Sprintf("%s%-52s %8s", marker, name, size))+"\n")
        }
    }
    b.WriteString("\n"+m.styles.Help.Render("↑/↓ navigate   Enter open   ← parent   h home   / root"))
    return b.String()
}

func (m *Model) renderEditor() string {
    var b strings.Builder
    title := m.buf.Path
    if title == "" { title = "(untitled)" }
    mode := "NORMAL"
    if m.insertMode { mode = "INSERT" }
    b.WriteString(m.styles.Path.Render(title)+"  "+m.styles.Mode.Render(mode)+"\n\n")
    if m.buf.Err != "" { b.WriteString(m.styles.Error.Render(m.buf.Err)+"\n") }
    for i := m.edView.Top; i < m.edView.Top+m.editorHeight() && i < m.buf.Len(); i++ {
        no := fmt.Sprintf("%4d ", i+1)
        line := m.buf.Line(i)
        if i == m.edView.CursorRow {
            runes := []rune(line)
            col := m.edView.CursorCol
            if col > len(runes) { col = len(runes) }
            line = string(runes[:col])+"▌"+string(runes[col:])
            b.WriteString(m.styles.Selected.Render(no+line)+"\n")
        } else {
            b.WriteString(m.styles.LineNo.Render(no)+m.styles.File.Render(line)+"\n")
        }
    }
    if m.savedMsg != "" { b.WriteString(m.styles.Success.Render(m.savedMsg)+"\n") }
    if m.errMsg != "" { b.WriteString(m.styles.Error.Render(m.errMsg)+"\n") }
    b.WriteString("\n"+m.styles.Help.Render("i insert   Ctrl-S save   arrows move   Esc normal"))
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
    b.WriteString(row("Architecture", info.Arch))
    b.WriteString(row("Hostname", info.Hostname))
    b.WriteString(row("User", info.CurrentUser))
    b.WriteString(row("Shell", info.Shell))
    b.WriteString("\n"+m.styles.Section.Render("NETWORK")+"\n")
    if len(info.Interfaces) == 0 { b.WriteString(row("Interfaces", "none")) }
    for _, n := range info.Interfaces { b.WriteString(row(n.Name, fmt.Sprintf("%s (%s)", n.State, strings.Join(n.Addresses, ", ")))) }
    return b.String()
}

func row(label, value string) string { return fmt.Sprintf("  %-16s  %s\n", label, value) }

func (m *Model) renderTasks() string {
    var b strings.Builder
    b.WriteString(m.styles.TableHeader.Render(fmt.Sprintf("%-7s %-7s %-8s %-10s %s", "PID", "PPID", "CPU", "RAM", "PROCESS"))+"\n")
    b.WriteString(m.styles.Divider.Render(strings.Repeat("─", 72))+"\n")
    for _, p := range m.lastTask { b.WriteString(fmt.Sprintf("%-7d %-7d %-8s %-10s %s\n", p.PID, p.PPID, fmt.Sprintf("%.1f%%", p.CPU), safef(p.RSS), p.Name)) }
    b.WriteString("\n"+m.styles.Help.Render("Live /proc process data · refreshes every second"))
    return b.String()
}

func safef(kib uint64) string { return fmt.Sprintf("%.1fM", float64(kib)/1024) }

func fitScreen(s string, width, height int, style lipgloss.Style) string {
    lines := strings.Split(s, "\n")
    if len(lines) > height { lines = lines[:height] }
    for len(lines) < height { lines = append(lines, "") }
    for i, line := range lines {
        w := lipgloss.Width(line)
        if w > width { lines[i] = lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line) } else if w < width { lines[i] += strings.Repeat(" ", width-w) }
    }
    return style.Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func clipLines(s string, maxLines, maxWidth int) string {
    if maxLines < 1 || maxWidth < 1 { return "" }
    lines := strings.Split(s, "\n")
    if len(lines) > maxLines { lines = lines[len(lines)-maxLines:] }
    for i, line := range lines {
        if lipgloss.Width(line) > maxWidth { lines[i] = lipgloss.NewStyle().Width(maxWidth).MaxWidth(maxWidth).Render(line) }
    }
    return strings.Join(lines, "\n")
}

var _ = ui.GlyphBullet
