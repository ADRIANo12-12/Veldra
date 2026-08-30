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
	"strconv"
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

// EditorTargetOverride, when set, tells the Editor app to open a specific file.
var EditorTargetOverride string

type Model struct {
	theme  ui.Theme
	styles ui.Styles
	active int
	width  int
	height int
	ready  bool

	sysInfo system.Info
	fs      *files.Browser
	buf     *editor.Buffer
	edView  editor.ViewWindow
	tasks   *taskmanager.Collector
	lastTask []taskmanager.Process

	insertMode bool

	terminalInput   string
	terminalHistory []string
	terminalPos     int
	terminalOut     []string

	quitting bool
	errMsg   string
	savedMsg string
}

func NewModel(startDir, startFile string) *Model {
	m := &Model{
		theme:   ui.VeldraDark(),
		active:  AppTerminal,
		sysInfo: system.Current(),
		terminalOut: []string{
			"Veldra Shell 0.0.1-pre-alpha",
			"Terminal ready. Type commands below.",
		},
	}
	m.styles = ui.NewStyles(m.theme)

	if startDir == "" {
		if h, err := os.UserHomeDir(); err == nil {
			startDir = h
		}
	}
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

func (m *Model) handleKey(km tea.KeyMsg) {
	if m.active == AppTerminal {
		if m.terminalHandleKey(km) {
			return
		}
	}

	switch km.String() {
	case "ctrl+q", "ctrl+c":
		m.quitting = true
	case "tab", "right":
		m.selectNext()
	case "shift+tab", "left":
		m.selectPrev()
	case "1":
		m.setActive(AppTerminal)
	case "2":
		m.setActive(AppFiles)
	case "3":
		m.setActive(AppEditor)
	case "4":
		m.setActive(AppSettings)
	case "5":
		m.setActive(AppTasks)
	case "r":
		m.refresh()
	case "enter":
		m.activate()
	default:
		m.forwardKey(km)
	}
}

func (m *Model) terminalHandleKey(km tea.KeyMsg) bool {
	s := km.String()
	switch s {
	case "ctrl+q":
		m.quitting = true
		return true
	case "enter":
		m.runTerminalCommand()
		return true
	case "backspace":
		if len(m.terminalInput) > 0 {
			m.terminalInput = m.terminalInput[:len(m.terminalInput)-1]
		}
		return true
	case "up":
		if len(m.terminalHistory) > 0 {
			if m.terminalPos < len(m.terminalHistory) {
				m.terminalPos++
			}
			m.terminalInput = historyAt(m.terminalHistory, m.terminalPos)
		}
		return true
	case "down":
		if m.terminalPos > 0 {
			m.terminalPos--
			m.terminalInput = historyAt(m.terminalHistory, m.terminalPos)
		} else {
			m.terminalInput = ""
		}
		return true
	case "esc":
		m.terminalInput = ""
		return true
	}
	if s == "q" && m.terminalInput == "" {
		m.quitting = true
		return true
	}
	if len(km.Runes) == 1 && km.Type == tea.KeyRunes {
		m.terminalInput += string(km.Runes)
		return true
	}
	return false
}

func historyAt(history []string, pos int) string {
	if len(history) == 0 || pos <= 0 || pos > len(history) {
		return ""
	}
	return history[len(history)-pos]
}

func (m *Model) runTerminalCommand() {
	command := strings.TrimSpace(m.terminalInput)
	if command == "" {
		m.terminalOut = append(m.terminalOut, m.prompt()+" ")
		m.terminalInput = ""
		return
	}
	m.terminalHistory = append(m.terminalHistory, command)
	m.terminalPos = 0
	m.terminalOut = append(m.terminalOut, m.prompt()+" "+command)

	cmd := exec.Command("bash", "-lc", command)
	cmd.Dir = m.terminalDir()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
			m.terminalOut = append(m.terminalOut, line)
		}
	}
	if err != nil {
		m.terminalOut = append(m.terminalOut, m.styles.Error.Render(err.Error()))
	}
	if len(m.terminalOut) > 500 {
		m.terminalOut = m.terminalOut[len(m.terminalOut)-500:]
	}
	m.terminalInput = ""
}

func (m *Model) terminalDir() string {
	if m.fs != nil && m.fs.Path != "" {
		return m.fs.Path
	}
	if dir, err := os.UserHomeDir(); err == nil {
		return dir
	}
	return "/"
}

func (m *Model) prompt() string {
	user := m.sysInfo.CurrentUser
	if user == "" {
		user = "veldra"
	}
	return fmt.Sprintf("%s@%s:%s$", user, m.sysInfo.Hostname, m.terminalDir())
}

func (m *Model) selectPrev() { m.active = (m.active + len(appLabels) - 1) % len(appLabels) }
func (m *Model) selectNext() { m.active = (m.active + 1) % len(appLabels) }

func (m *Model) setActive(i int) {
	m.active = i
	if i == AppTasks {
		m.lastTask = m.tasks.Refresh()
	}
}

func (m *Model) activate() { m.setActive(m.active) }

func (m *Model) refresh() {
	m.sysInfo = system.Current()
	if m.active == AppTasks {
		m.lastTask = m.tasks.Refresh()
	}
}

func (m *Model) forwardKey(km tea.KeyMsg) {
	switch m.active {
	case AppFiles:
		m.filesKey(km)
	case AppEditor:
		m.editorKey(km)
	case AppTasks:
		m.taskKey(km)
	}
}

func (m *Model) filesKey(km tea.KeyMsg) {
	switch km.String() {
	case "up":
		m.fs.Up()
	case "down":
		m.fs.Down()
	case "left":
		m.fs.GoUp()
	case "enter":
		if target := m.fs.Open(); target != "" {
			m.buf = editor.NewBuffer(target)
			m.edView = editor.ViewWindow{}
			m.active = AppEditor
		}
	case "h":
		m.fs.Home()
	case "/":
		m.fs.Root()
	}
}

func (m *Model) editorKey(km tea.KeyMsg) {
	if m.insertMode {
		m.editorEditKey(km)
		return
	}
	switch km.String() {
	case "up": m.edView.CursorRow--
	case "down": m.edView.CursorRow++
	case "pgup": m.edView.CursorRow -= 10
	case "pgdown": m.edView.CursorRow += 10
	case "left": if m.edView.CursorCol > 0 { m.edView.CursorCol-- }
	case "right": m.edView.CursorCol++
	case "home": m.edView.CursorCol = 0
	case "i": m.insertMode = true
	case "ctrl+s": m.saveEditor()
	case "enter":
		if m.edView.CursorRow < m.buf.Len() && m.buf.SplitLine(m.edView.CursorRow, m.edView.CursorCol) {
			m.edView.CursorRow++
			m.edView.CursorCol = 0
		}
	}
}

func (m *Model) editorEditKey(km tea.KeyMsg) {
	switch km.String() {
	case "escape", "ctrl+c": m.insertMode = false
	case "enter":
		if m.buf.SplitLine(m.edView.CursorRow, m.edView.CursorCol) {
			m.edView.CursorRow++
			m.edView.CursorCol = 0
		}
	case "backspace":
		if m.edView.CursorCol > 0 { m.buf.DeleteAt(m.edView.CursorRow, m.edView.CursorCol-1); m.edView.CursorCol-- }
	case "ctrl+s": m.saveEditor()
	default:
		if len(km.Runes) == 1 && km.Type == tea.KeyRunes {
			m.buf.InsertAt(m.edView.CursorRow, m.edView.CursorCol, string(km.Runes[0]))
			m.edView.CursorCol++
		}
	}
}

func (m *Model) saveEditor() {
	if err := m.buf.Save(); err != nil { m.errMsg = "save failed: " + err.Error(); return }
	m.savedMsg = "saved " + m.buf.Path
}

func (m *Model) taskKey(km tea.KeyMsg) {}

func (m *Model) handleMouse(me tea.MouseMsg) {
	if me.Action != tea.MouseActionPress { return }
	if me.Y == 0 {
		idx := m.topBarAppAt(me.X)
		if idx >= 0 { m.setActive(idx) }
	}
}

func (m *Model) topBarAppAt(x int) int {
	cursor := 1
	for i := range appLabels {
		cell := fmt.Sprintf(" %d:%s ", i+1, appLabels[i])
		w := lipgloss.Width(cell)
		if x >= cursor && x < cursor+w { return i }
		cursor += w + 1
	}
	return -1
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
	case tea.KeyMsg:
		m.handleKey(msg)
		if m.quitting { return m, tea.Quit }
	case tea.MouseMsg:
		m.handleMouse(msg)
	case refreshMsg:
		m.refresh()
		return m, tick()
	}
	if m.buf != nil {
		m.edView.Clamp(m.buf, m.editorHeight())
	}
	return m, nil
}

func (m *Model) editorHeight() int {
	h := m.height - 6
	if h < 3 { h = 3 }
	return h
}

func (m *Model) Quitting() bool { return m.quitting }

func strconvToInt(s string) int { n, _ := strconv.Atoi(s); return n }

var _ = strings.TrimSpace
