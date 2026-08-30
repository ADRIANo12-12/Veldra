// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
//
// Full-screen terminal desktop shell renderer.
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
    bodyHeight := m.height - lipgloss.Height(top)
    if bodyHeight < 1 { return top }
    body := m.viewWorkspace(bodyHeight)
    if m.paletteOpen {
        body = m.viewPalette(bodyHeight)
    }
    return lipgloss.JoinVertical(lipgloss.Left, top, body)
}

func (m *Model) viewTopBar() string {
    left := m.styles.BarBrand.Render(" ◈ VELDRA ")
    workspaces := ""
    for i, label := range appLabels {
        item := fmt.Sprintf(" %d %s ", i+1, label)
        if i == m.active {
            item = m.styles.BarWorkspaceActive.Render(item)
        } else {
            item = m.styles.BarWorkspace.Render(item)
        }
        workspaces += item
    }
    memUsed := float64(m.sysInfo.MemTotal-m.sysInfo.MemAvailable) / 1024
    right := m.styles.BarRight.Render(fmt.Sprintf(" %s  %s  %dC  MEM %.0fM ", time.Now().Format("15:04:05"), m.sysInfo.Hostname, m.sysInfo.CPUCores, memUsed))
    used := lipgloss.Width(left) + lipgloss.Width(workspaces) + lipgloss.Width(right)
    gap := m.width - used
    if gap < 1 { gap = 1 }
    return lipgloss.JoinHorizontal(lipgloss.Top, left, workspaces, strings.Repeat(" ", gap), right)
}

func (m *Model) viewWorkspace(height int) string {
    sidebarWidth := 22
    mainWidth := m.width - sidebarWidth - 1
    if mainWidth < 25 { mainWidth = 25 }
    side := m.viewSidebar(height, sidebarWidth)
    main := m.viewMainPanel(height, mainWidth)
    return lipgloss.JoinHorizontal(lipgloss.Top, side, m.styles.Divider.Render("│"), main)
}

func (m *Model) viewSidebar(height, width int) string {
    lines := []string{m.styles.SideTitle.Render("WORKSPACES")}
    for i, label := range appLabels {
        prefix := "   "
        if i == m.active { prefix = " ● " }
        line := prefix + fmt.Sprintf("%d  %s", i+1, label)
        if i == m.active { line = m.styles.SideActive.Render(line) } else { line = m.styles.SideItem.Render(line) }
        lines = append(lines, line)
    }
    lines = append(lines, "", m.styles.SideTitle.Render("SYSTEM"))
    lines = append(lines,
        fmt.Sprintf("  Kernel  %s", m.sysInfo.Kernel),
        fmt.Sprintf("  User    %s", m.sysInfo.CurrentUser),
        fmt.Sprintf("  Shell   %s", m.sysInfo.Shell),
        fmt.Sprintf("  Uptime  %s", m.sysInfo.Uptime),
    )
    body := strings.Join(lines, "\n")
    return lipgloss.NewStyle().Width(width).Height(height).Padding(1, 1).Render(body)
}

func (m *Model) viewMainPanel(height, width int) string {
    header := m.styles.PanelHeader.Render(fmt.Sprintf(" %s  %s", appLabels[m.active], m.panelHint()))
    contentHeight := height - lipgloss.Height(header) - 1
    if contentHeight < 2 { contentHeight = 2 }
    content := clipLines(m.viewApp(), contentHeight, width-2)
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
    if maxLines < 1 || maxWidth < 1 { return "" }
    out := make([]string, 0, maxLines)
    for _, line := range strings.Split(s, "\n") {
        if len(out) >= maxLines { break }
        if lipgloss.Width(line) > maxWidth {
            line = lipgloss.NewStyle().Width(maxWidth).MaxWidth(maxWidth).Render(line)
        }
        out = append(out, line)
    }
    return strings.Join(out, "\n")
}

func (m *Model) viewPalette(height int) string {
    items := m.filteredPaletteActions()
    width := m.width - 12
    if width > 82 { width = 82 }
    if width < 36 { width = m.width - 4 }
    if width < 10 { width = 10 }

    lines := []string{m.styles.PaletteTitle.Render("COMMAND PALETTE")}
    cursor := m.paletteInputCursor
    if cursor < 0 { cursor = 0 }
    if cursor > len(m.paletteInput) { cursor = len(m.paletteInput) }
    lines = append(lines,
        m.styles.PaletteInput.Render("❯ "+string(m.paletteInput[:cursor])+"▌"+string(m.paletteInput[cursor:])),
        "",
    )
    for i, a := range items {
        line := fmt.Sprintf(" %-22s  %-34s  %s", a.Label, a.Hint, a.Key)
        if i == m.paletteIndex { lines = append(lines, m.styles.PaletteActive.Render(line)) } else { lines = append(lines, m.styles.PaletteItem.Render(line)) }
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
    default: return "unknown application"
    }
}

func (m *Model) renderTerminal() string {
    var b strings.Builder
    for _, line := range m.terminalOut { b.WriteString(cleanOutput(line)+"\n") }
    b.WriteString("\n")
    cursor := m.terminalCursor
    if cursor < 0 { cursor = 0 }
    if cursor > len(m.terminalInput) { cursor = len(m.terminalInput) }
    input := string(m.terminalInput)
    prompt := m.styles.Prompt.Render(m.prompt()+" ")
    b.WriteString(prompt + m.styles.TerminalInput.Render(input[:cursor]+"▌"+input[cursor:]))
    if m.terminalRunning { b.WriteString("  "+m.styles.Help.Render("running…")) }
    return b.String()
}

func (m *Model) renderFiles() string {
    var b strings.Builder
    b.WriteString(m.styles.Path.Render("▸ "+m.fs.Path)+"\n\n")
    for i, e := range m.fs.Items {
        marker := "  "
        name := e.Name
        if e.IsDir { marker = "▸ "; name += "/" }
        suffix := ""
        if !e.IsDir { suffix = files.HumanSize(e.Size) }
        line := fmt.Sprintf("%s%-44s %10s", marker, name, suffix)
        if i == m.fs.Cursor { line = m.styles.Selected.Render("▸ "+name) }
        b.WriteString(line+"\n")
    }
    b.WriteString("\n"+m.styles.Help.Render("↑/↓ navigate  Enter open  ← parent  h home  / root  Ctrl+1…5 workspaces"))
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
        if i == m.edView.CursorRow { b.WriteString(m.styles.Selected.Render(lineNo+m.buf.Line(i))+"\n") } else { b.WriteString(m.styles.LineNo.Render(lineNo)+m.styles.File.Render(m.buf.Line(i))+"\n") }
    }
    b.WriteString("\n"+m.styles.Help.Render("i insert  Ctrl-S save  arrows move  Esc view  Ctrl+3 editor"))
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
    for _, n := range info.Interfaces { b.WriteString(row(n.Name, fmt.Sprintf("%s (%s)", n.State, strings.Join(n.Addresses, ", ")))) }
    return b.String()
}

func row(label, value string) string { return fmt.Sprintf("  %-16s  %s\n", label, value) }

func (m *Model) renderTasks() string {
    var b strings.Builder
    b.WriteString(m.styles.TableHeader.Render(fmt.Sprintf("%-7s %-7s %-8s %-10s %s", "PID", "PPID", "CPU", "RAM", "PROCESS"))+"\n")
    b.WriteString(m.styles.Divider.Render(strings.Repeat("─", 72))+"\n")
    for _, p := range m.lastTask { b.WriteString(fmt.Sprintf("%-7d %-7d %-8s %-10s %s", p.PID, p.PPID, fmt.Sprintf("%.1f%%", p.CPU), safef(p.RSS), p.Name)+"\n") }
    b.WriteString("\n"+m.styles.Help.Render("Processes are read from /proc and refreshed every second. Ctrl+5 focuses Tasks."))
    return b.String()
}

func safef(kib uint64) string { return fmt.Sprintf("%.1fM", float64(kib)/1024) }

var _ = ui.GlyphBullet
