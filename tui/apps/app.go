// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Root model for the Veldra terminal desktop shell.
package apps

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "veldra/tui/editor"
    "veldra/tui/files"
    "veldra/tui/system"
    "veldra/tui/taskmanager"
    "veldra/tui/ui"
)

const (
    AppTerminal = iota
    AppFiles
    AppEditor
    AppSettings
    AppTasks
)

var appLabels = []string{"Terminal", "Files", "Editor", "Settings", "Tasks"}
var ansiSequence = regexp.MustCompile("\\x1b\\[[0-9;?]*[ -/]*[@-~]")

type paletteAction struct {
    Label string
    Hint  string
    Key   string
    App   int
}

type terminalResultMsg struct {
    Output   string
    Err      error
    Duration time.Duration
}

type Model struct {
    theme  ui.Theme
    styles ui.Styles

    active int
    width  int
    height int
    ready  bool

    sysInfo  system.Info
    fs       *files.Browser
    buf      *editor.Buffer
    edView   editor.ViewWindow
    tasks    *taskmanager.Collector
    lastTask []taskmanager.Process

    insertMode bool

    cwd             string
    previousCWD     string
    terminalInput   []rune
    terminalCursor  int
    terminalHistory []string
    historyIndex    int
    terminalScratch []rune
    terminalOut     []string
    terminalRunning bool
    terminalStatus  string

    paletteOpen        bool
    paletteInput       []rune
    paletteInputCursor int
    paletteIndex        int

    quitting bool
    errMsg   string
    savedMsg string
}

func NewModel(startDir, startFile string) *Model {
    if startDir == "" {
        if home, err := os.UserHomeDir(); err == nil {
            startDir = home
        } else {
            startDir = "/"
        }
    }
    m := &Model{
        theme:  ui.VeldraDark(),
        active: AppTerminal,
        cwd:    startDir,
        sysInfo: system.Current(),
        terminalOut: []string{
            "Veldra Shell 0.0.1-pre-alpha",
            "Type a command. Ctrl+P opens the command palette.",
        },
    }
    m.styles = ui.NewStyles(m.theme)
    m.fs = files.NewBrowser(startDir)
    m.buf = editor.NewBuffer(startFile)
    if startFile != "" {
        m.active = AppEditor
    }
    m.tasks = taskmanager.NewCollector()
    m.lastTask = m.tasks.Refresh()
    return m
}

func (m *Model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
    return tea.Tick(time.Second, func(time.Time) tea.Msg { return refreshMsg{} })
}

type refreshMsg struct{}

func (m *Model) handleKey(km tea.KeyMsg) tea.Cmd {
    if m.paletteOpen {
        return m.handlePaletteKey(km)
    }
    if m.active == AppTerminal {
        return m.handleTerminalKey(km)
    }
    return m.handleWorkspaceKey(km)
}

func (m *Model) handleWorkspaceKey(km tea.KeyMsg) tea.Cmd {
    switch km.String() {
    case "ctrl+q":
        m.quitting = true
    case "ctrl+p":
        m.openPalette()
    case "ctrl+1":
        m.setActive(AppTerminal)
    case "ctrl+2":
        m.setActive(AppFiles)
    case "ctrl+3":
        m.setActive(AppEditor)
    case "ctrl+4":
        m.setActive(AppSettings)
    case "ctrl+5":
        m.setActive(AppTasks)
    case "tab":
        m.selectNext()
    case "shift+tab":
        m.selectPrev()
    case "r":
        m.refresh()
    case "enter":
        m.activate()
    default:
        m.forwardKey(km)
    }
    return nil
}

func (m *Model) handleTerminalKey(km tea.KeyMsg) tea.Cmd {
    switch km.String() {
    case "ctrl+q":
        m.quitting = true
        return nil
    case "ctrl+p":
        m.openPalette()
        return nil
    case "ctrl+l":
        m.terminalOut = nil
        return nil
    case "ctrl+r":
        m.refresh()
        return nil
    case "ctrl+1":
        m.setActive(AppTerminal)
        return nil
    case "ctrl+2":
        m.setActive(AppFiles)
        return nil
    case "ctrl+3":
        m.setActive(AppEditor)
        return nil
    case "ctrl+4":
        m.setActive(AppSettings)
        return nil
    case "ctrl+5":
        m.setActive(AppTasks)
        return nil
    case "enter":
        return m.acceptTerminalInput()
    case "backspace":
        m.deleteTerminalRune(-1)
        return nil
    case "delete":
        m.deleteTerminalRune(1)
        return nil
    case "left":
        if m.terminalCursor > 0 { m.terminalCursor-- }
        return nil
    case "right":
        if m.terminalCursor < len(m.terminalInput) { m.terminalCursor++ }
        return nil
    case "home":
        m.terminalCursor = 0
        return nil
    case "end":
        m.terminalCursor = len(m.terminalInput)
        return nil
    case "up":
        m.historyUp()
        return nil
    case "down":
        m.historyDown()
        return nil
    case "esc":
        m.terminalInput = nil
        m.terminalCursor = 0
        m.historyIndex = 0
        return nil
    }
    if km.Type == tea.KeyRunes && len(km.Runes) > 0 {
        m.insertTerminalRunes(km.Runes)
    }
    return nil
}

func (m *Model) insertTerminalRunes(rs []rune) {
    next := make([]rune, 0, len(m.terminalInput)+len(rs))
    next = append(next, m.terminalInput[:m.terminalCursor]...)
    next = append(next, rs...)
    next = append(next, m.terminalInput[m.terminalCursor:]...)
    m.terminalInput = next
    m.terminalCursor += len(rs)
    m.historyIndex = 0
}

func (m *Model) deleteTerminalRune(direction int) {
    if direction < 0 {
        if m.terminalCursor == 0 { return }
        i := m.terminalCursor - 1
        m.terminalInput = append(m.terminalInput[:i], m.terminalInput[m.terminalCursor:]...)
        m.terminalCursor--
        return
    }
    if m.terminalCursor >= len(m.terminalInput) { return }
    m.terminalInput = append(m.terminalInput[:m.terminalCursor], m.terminalInput[m.terminalCursor+1:]...)
}

func (m *Model) acceptTerminalInput() tea.Cmd {
    command := strings.TrimSpace(string(m.terminalInput))
    m.terminalInput = nil
    m.terminalCursor = 0
    m.historyIndex = 0
    m.terminalScratch = nil
    if command == "" || m.terminalRunning { return nil }

    m.terminalHistory = append(m.terminalHistory, command)
    m.terminalOut = append(m.terminalOut, m.prompt()+" "+command)
    m.trimTerminalOutput()
    if handled, cmd := m.handleBuiltin(command); handled { return cmd }

    m.terminalRunning = true
    m.terminalStatus = "running"
    return runTerminalCommand(m.cwd, command)
}

func (m *Model) handleBuiltin(command string) (bool, tea.Cmd) {
    fields := strings.Fields(command)
    if len(fields) == 0 { return true, nil }
    switch fields[0] {
    case "cd":
        target := "~"
        if len(fields) > 1 { target = strings.Join(fields[1:], " ") }
        return true, m.changeDirectory(target)
    case "pwd":
        m.terminalOut = append(m.terminalOut, m.cwd)
        m.trimTerminalOutput()
        return true, nil
    case "clear":
        m.terminalOut = nil
        return true, nil
    case "help":
        m.terminalOut = append(m.terminalOut,
            "Builtins: cd <dir>, pwd, clear, help, exit",
            "Navigation: Ctrl+1..5 workspaces · Tab cycles workspaces",
            "Ctrl+P command palette · Ctrl+L clear output · Ctrl+Q quit",
        )
        m.trimTerminalOutput()
        return true, nil
    case "exit":
        m.quitting = true
        return true, nil
    default:
        return false, nil
    }
}

func (m *Model) changeDirectory(target string) tea.Cmd {
    expanded := target
    if expanded == "~" || strings.HasPrefix(expanded, "~/") {
        if home, err := os.UserHomeDir(); err == nil {
            expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
        }
    } else if expanded == "-" {
        if m.previousCWD == "" {
            m.terminalOut = append(m.terminalOut, "cd: OLDPWD not set")
            return nil
        }
        expanded = m.previousCWD
    } else if !filepath.IsAbs(expanded) {
        expanded = filepath.Join(m.cwd, expanded)
    }
    clean, err := filepath.Abs(filepath.Clean(expanded))
    if err != nil {
        m.terminalOut = append(m.terminalOut, "cd: "+err.Error())
        m.trimTerminalOutput()
        return nil
    }
    info, err := os.Stat(clean)
    if err != nil || !info.IsDir() {
        if err != nil { m.terminalOut = append(m.terminalOut, "cd: "+err.Error()) } else { m.terminalOut = append(m.terminalOut, "cd: not a directory: "+clean) }
        m.trimTerminalOutput()
        return nil
    }
    m.previousCWD = m.cwd
    m.cwd = clean
    m.fs.Path = clean
    m.fs.Cursor = 0
    m.fs.Reload()
    return nil
}

func runTerminalCommand(dir, command string) tea.Cmd {
    return func() tea.Msg {
        start := time.Now()
        cmd := exec.Command("bash", "-lc", command)
        cmd.Dir = dir
        cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor", "PAGER=cat", "GIT_PAGER=cat", "MANPAGER=cat")
        out, err := cmd.CombinedOutput()
        return terminalResultMsg{Output: string(out), Err: err, Duration: time.Since(start)}
    }
}

func (m *Model) historyUp() {
    if len(m.terminalHistory) == 0 || m.terminalRunning { return }
    if m.historyIndex == 0 { m.terminalScratch = append([]rune(nil), m.terminalInput...) }
    if m.historyIndex < len(m.terminalHistory) { m.historyIndex++ }
    m.terminalInput = []rune(m.terminalHistory[len(m.terminalHistory)-m.historyIndex])
    m.terminalCursor = len(m.terminalInput)
}

func (m *Model) historyDown() {
    if m.historyIndex == 0 { return }
    m.historyIndex--
    if m.historyIndex == 0 { m.terminalInput = append([]rune(nil), m.terminalScratch...) } else { m.terminalInput = []rune(m.terminalHistory[len(m.terminalHistory)-m.historyIndex]) }
    m.terminalCursor = len(m.terminalInput)
}

func (m *Model) trimTerminalOutput() {
    const maxLines = 600
    if len(m.terminalOut) > maxLines { m.terminalOut = m.terminalOut[len(m.terminalOut)-maxLines:] }
}

func cleanOutput(s string) string { return ansiSequence.ReplaceAllString(s, "") }

func (m *Model) handlePaletteKey(km tea.KeyMsg) tea.Cmd {
    switch km.String() {
    case "esc", "ctrl+p":
        m.paletteOpen = false
        m.paletteInput = nil
        m.paletteInputCursor = 0
        m.paletteIndex = 0
        return nil
    case "up":
        m.movePalette(-1)
        return nil
    case "down":
        m.movePalette(1)
        return nil
    case "enter":
        return m.runPaletteSelection()
    case "backspace":
        if m.paletteInputCursor > 0 {
            i := m.paletteInputCursor - 1
            m.paletteInput = append(m.paletteInput[:i], m.paletteInput[m.paletteInputCursor:]...)
            m.paletteInputCursor--
            m.paletteIndex = 0
        }
        return nil
    case "delete":
        if m.paletteInputCursor < len(m.paletteInput) { m.paletteInput = append(m.paletteInput[:m.paletteInputCursor], m.paletteInput[m.paletteInputCursor+1:]...) }
        return nil
    case "left":
        if m.paletteInputCursor > 0 { m.paletteInputCursor-- }
        return nil
    case "right":
        if m.paletteInputCursor < len(m.paletteInput) { m.paletteInputCursor++ }
        return nil
    }
    if km.Type == tea.KeyRunes && len(km.Runes) > 0 {
        next := make([]rune, 0, len(m.paletteInput)+len(km.Runes))
        next = append(next, m.paletteInput[:m.paletteInputCursor]...)
        next = append(next, km.Runes...)
        next = append(next, m.paletteInput[m.paletteInputCursor:]...)
        m.paletteInput = next
        m.paletteInputCursor += len(km.Runes)
        m.paletteIndex = 0
    }
    return nil
}

func (m *Model) paletteActions() []paletteAction {
    return []paletteAction{
        {Label: "Terminal", Hint: "open terminal workspace", Key: "Ctrl+1", App: AppTerminal},
        {Label: "Files", Hint: "open filesystem browser", Key: "Ctrl+2", App: AppFiles},
        {Label: "Editor", Hint: "open text editor", Key: "Ctrl+3", App: AppEditor},
        {Label: "Settings", Hint: "system control center", Key: "Ctrl+4", App: AppSettings},
        {Label: "Tasks", Hint: "live process monitor", Key: "Ctrl+5", App: AppTasks},
        {Label: "Refresh system", Hint: "refresh hardware/network data", Key: "Ctrl+R", App: -1},
        {Label: "Clear terminal", Hint: "clear terminal output", Key: "Ctrl+L", App: -1},
    }
}

func (m *Model) filteredPaletteActions() []paletteAction {
    q := strings.ToLower(strings.TrimSpace(string(m.paletteInput)))
    var out []paletteAction
    for _, a := range m.paletteActions() {
        if q == "" || strings.Contains(strings.ToLower(a.Label), q) || strings.Contains(strings.ToLower(a.Hint), q) { out = append(out, a) }
    }
    return out
}

func (m *Model) movePalette(delta int) {
    items := m.filteredPaletteActions()
    if len(items) == 0 { m.paletteIndex = 0; return }
    m.paletteIndex = (m.paletteIndex + delta + len(items)) % len(items)
}

func (m *Model) runPaletteSelection() tea.Cmd {
    items := m.filteredPaletteActions()
    if len(items) == 0 { return nil }
    a := items[m.paletteIndex]
    m.paletteOpen = false
    m.paletteInput = nil
    m.paletteInputCursor = 0
    m.paletteIndex = 0
    switch a.Label {
    case "Refresh system": m.refresh()
    case "Clear terminal": m.terminalOut = nil
    default:
        if a.App >= 0 { m.setActive(a.App) }
    }
    return nil
}

func (m *Model) openPalette() {
    m.paletteOpen = true
    m.paletteInput = nil
    m.paletteInputCursor = 0
    m.paletteIndex = 0
}

func (m *Model) setActive(i int) {
    if i < 0 || i >= len(appLabels) { return }
    m.active = i
    if i == AppFiles {
        m.fs.Path = m.cwd
        m.fs.Cursor = 0
        m.fs.Reload()
    }
    if i == AppTasks { m.lastTask = m.tasks.Refresh() }
}

func (m *Model) selectPrev() { m.setActive((m.active + len(appLabels) - 1) % len(appLabels)) }
func (m *Model) selectNext() { m.setActive((m.active + 1) % len(appLabels)) }
func (m *Model) activate() {}

func (m *Model) refresh() {
    m.sysInfo = system.Current()
    if m.active == AppTasks { m.lastTask = m.tasks.Refresh() }
}

func (m *Model) forwardKey(km tea.KeyMsg) {
    switch m.active {
    case AppFiles: m.filesKey(km)
    case AppEditor: m.editorKey(km)
    case AppTasks:
    }
}

func (m *Model) filesKey(km tea.KeyMsg) {
    switch km.String() {
    case "up": m.fs.Up()
    case "down": m.fs.Down()
    case "left": m.fs.GoUp()
    case "enter":
        if target := m.fs.Open(); target != "" {
            m.buf = editor.NewBuffer(target)
            m.edView = editor.ViewWindow{}
            m.active = AppEditor
        }
    case "h": m.fs.Home()
    case "/": m.fs.Root()
    }
    if m.active == AppFiles { m.cwd = m.fs.Path }
}

func (m *Model) editorKey(km tea.KeyMsg) {
    if m.insertMode { m.editorEditKey(km); return }
    switch km.String() {
    case "up": m.edView.CursorRow--
    case "down": m.edView.CursorRow++
    case "left": if m.edView.CursorCol > 0 { m.edView.CursorCol-- }
    case "right": m.edView.CursorCol++
    case "pgup": m.edView.CursorRow -= 10
    case "pgdown": m.edView.CursorRow += 10
    case "home": m.edView.CursorCol = 0
    case "end": if m.buf.Len() > 0 { m.edView.CursorCol = len([]rune(m.buf.Line(m.edView.CursorRow))) }
    case "i": m.insertMode = true
    case "ctrl+s": m.saveEditor()
    }
}

func (m *Model) editorEditKey(km tea.KeyMsg) {
    switch km.String() {
    case "escape": m.insertMode = false
    case "enter":
        if m.buf.SplitLine(m.edView.CursorRow, m.edView.CursorCol) { m.edView.CursorRow++; m.edView.CursorCol = 0 }
    case "backspace":
        if m.edView.CursorCol > 0 { m.buf.DeleteAt(m.edView.CursorRow, m.edView.CursorCol-1); m.edView.CursorCol-- }
    case "delete":
        m.buf.DeleteAt(m.edView.CursorRow, m.edView.CursorCol)
    case "ctrl+s":
        m.saveEditor()
    default:
        if km.Type == tea.KeyRunes && len(km.Runes) > 0 {
            s := string(km.Runes)
            m.buf.InsertAt(m.edView.CursorRow, m.edView.CursorCol, s)
            m.edView.CursorCol += len([]rune(s))
        }
    }
}

func (m *Model) saveEditor() {
    if err := m.buf.Save(); err != nil { m.errMsg = "save failed: "+err.Error(); return }
    m.savedMsg = "saved "+m.buf.Path
}

func (m *Model) handleMouse(me tea.MouseMsg) {
    if me.Action != tea.MouseActionPress || me.Y != 0 || m.paletteOpen { return }
    if idx := m.topBarAppAt(me.X); idx >= 0 { m.setActive(idx) }
}

func (m *Model) topBarAppAt(x int) int {
    pos := lipgloss.Width(" ◈ VELDRA ")
    for i, cell := range m.topBarCells() {
        w := lipgloss.Width(cell)
        if x >= pos && x < pos+w { return i }
        pos += w
    }
    return -1
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        m.ready = true
    case tea.KeyMsg:
        cmd = m.handleKey(msg)
        if m.quitting { return m, tea.Quit }
    case tea.MouseMsg:
        m.handleMouse(msg)
    case terminalResultMsg:
        m.terminalRunning = false
        m.terminalStatus = "ready"
        text := cleanOutput(msg.Output)
        if text != "" { m.terminalOut = append(m.terminalOut, strings.Split(strings.TrimRight(text, "\n"), "\n")...) }
        if msg.Err != nil { m.terminalOut = append(m.terminalOut, m.styles.Error.Render(msg.Err.Error())) }
        m.terminalOut = append(m.terminalOut, fmt.Sprintf("[%s]", msg.Duration.Round(time.Millisecond)))
        m.trimTerminalOutput()
    case refreshMsg:
        m.refresh()
        cmd = tick()
    }
    if m.buf != nil { m.edView.Clamp(m.buf, m.editorHeight()) }
    return m, cmd
}

func (m *Model) editorHeight() int {
    h := m.height - 5
    if h < 3 { h = 3 }
    return h
}

func (m *Model) Quitting() bool { return m.quitting }

func (m *Model) prompt() string {
    user := m.sysInfo.CurrentUser
    if user == "" { user = "veldra" }
    host := m.sysInfo.Hostname
    if host == "" { host = "veldra" }
    return fmt.Sprintf("%s@%s:%s$", user, host, m.displayCWD())
}

func (m *Model) displayCWD() string {
    if home, err := os.UserHomeDir(); err == nil {
        if m.cwd == home { return "~" }
        if strings.HasPrefix(m.cwd, home+string(os.PathSeparator)) { return "~"+strings.TrimPrefix(m.cwd, home) }
    }
    return m.cwd
}

func (m *Model) editorStatus() string {
    mode := "NORMAL"
    if m.insertMode { mode = "INSERT" }
    return mode+"  "+m.displayCWD()
}

func (m *Model) topBarCells() []string {
    out := make([]string, len(appLabels))
    for i, label := range appLabels { out[i] = fmt.Sprintf(" %d %s ", i+1, label) }
    return out
}

// Compatibility helper retained for existing tests/tools.
func (m *Model) dockItemAt(x int) int { return m.topBarAppAt(x) }
