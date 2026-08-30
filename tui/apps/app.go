// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Package apps is the Veldra TUI program itself: a full-screen Bubble Tea
// application with a macOS-inspired top bar, central window, bottom dock and
// status information. It hosts five real applications: Terminal, Files,
// Editor, Settings and Task Manager. Everything shown is gathered from the
// live system at runtime.

package apps

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"veldra/tui/editor"
	"veldra/tui/files"
	"veldra/tui/system"
	"veldra/tui/taskmanager"
	"veldra/tui/ui"
)

// Application identifiers.
const (
	AppTerminal = 0
	AppFiles    = 1
	AppEditor   = 2
	AppSettings = 3
	AppTasks    = 4
)

// dockLabels are the five dock applications.
var dockLabels = []string{"Terminal", "Pliki", "Edytor", "Ustawienia", "Menedżer Zadań"}

// EditorTargetOverride, when set, tells the Editor app to open a specific
// real file (used by `veldra-tui --editor <file>`).
var EditorTargetOverride string

// Model is the root Bubble Tea model.
type Model struct {
	theme   ui.Theme
	styles  ui.Styles
	active  int
	width   int
	height  int
	ready   bool

	// application states
	term     string // terminal panel text (rendered lazily)
	sysInfo  system.Info
	fs       *files.Browser
	buf      *editor.Buffer
	edView   editor.ViewWindow
	settings string
	tasks    *taskmanager.Collector
	lastTask []taskmanager.Process

	// editor sub-mode (true = typing when selected)
	insertMode bool
	edTop      int

	// status
	quitting bool
	errMsg   string
	savedMsg string
	deadline time.Time
}

// NewModel constructs the root model with live system data.
func NewModel(startDir, startFile string) *Model {
	m := &Model{
		theme:  ui.VeldraDark(),
		active: AppTerminal,
		sysInfo: system.Current(),
	}
	m.styles = ui.NewStyles(m.theme)
	if startDir == "" {
		if h, err := os.UserHomeDir(); err == nil {
			startDir = h
		}
	}
	m.fs = files.NewBrowser(startDir)
	if startFile != "" {
		m.buf = editor.NewBuffer(startFile)
		m.active = AppEditor
	} else {
		m.buf = editor.NewBuffer("")
		m.sysInfo = system.Current()
	}
	m.tasks = taskmanager.NewCollector()
	m.lastTask = m.tasks.Refresh()
	m.deadline = time.Time{}
	return m
}

// Init starts any async commands.
func (m *Model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return refreshMsg{} })
}

type refreshMsg struct{}

// --- View-mode keys --------------------------------------------------------

func (m *Model) handleKey(km tea.KeyMsg) {
	switch km.String() {
	case "q", "ctrl+c":
		m.quitting = true
	case "left":
		m.selectPrev()
	case "right":
		m.selectNext()
	case "tab":
		m.selectNext()
	case "enter":
		m.activate()
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
	default:
		// pass keys to the active application
		m.forwardKey(km)
	}
}

func (m *Model) selectPrev() {
	m.active = (m.active - 1 + len(dockLabels)) % len(dockLabels)
}

func (m *Model) selectNext() {
	m.active = (m.active + 1) % len(dockLabels)
}

func (m *Model) setActive(i int) {
	m.active = i
	if i == AppTasks {
		m.lastTask = m.tasks.Refresh()
	}
}

func (m *Model) activate() {
	m.setActive(m.active)
}

// forwardKey sends a key to the focused application.
func (m *Model) forwardKey(km tea.KeyMsg) {
	switch m.active {
	case AppFiles:
		m.filesKey(km)
	case AppEditor:
		m.editorKey(km)
	case AppTasks:
		m.taskKey(km)
	case AppSettings:
		// no inside-app keys for settings yet
	}
}

// --- Sub-application key handlers -------------------------------------------

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
	case "up":
		m.edView.CursorRow--
	case "down":
		m.edView.CursorRow++
	case "pgup":
		m.edView.CursorRow -= 10
	case "pgdown":
		m.edView.CursorRow += 10
	case "left":
		if m.edView.CursorCol > 0 {
			m.edView.CursorCol--
		}
	case "right":
		m.edView.CursorCol++
	case "home":
		m.edView.CursorCol = 0
	case "i":
		m.insertMode = true
	case "ctrl+s":
		if err := m.buf.Save(); err != nil {
			m.errMsg = "save failed: " + err.Error()
		} else {
			m.savedMsg = "saved " + m.buf.Path
		}
	case "enter":
		if m.edView.CursorRow < m.buf.Len() && m.buf.SplitLine(m.edView.CursorRow, m.edView.CursorCol) {
			m.edView.CursorRow++
			m.edView.CursorCol = 0
		}
	case "f":
		// quick find: jump to "package " or first non-empty line
		if i := m.buf.Find("package"); i >= 0 {
			m.edView.CursorRow = i
		}
	}
}

func (m *Model) editorEditKey(km tea.KeyMsg) {
	switch km.String() {
	case "escape", "ctrl+c":
		m.insertMode = false
	case "enter":
		if m.buf.SplitLine(m.edView.CursorRow, m.edView.CursorCol) {
			m.edView.CursorRow++
			m.edView.CursorCol = 0
		}
	case "backspace":
		if m.edView.CursorCol > 0 {
			m.buf.DeleteAt(m.edView.CursorRow, m.edView.CursorCol-1)
			m.edView.CursorCol--
		}
	case "ctrl+s":
		if err := m.buf.Save(); err != nil {
			m.errMsg = "save failed: " + err.Error()
		} else {
			m.savedMsg = "saved " + m.buf.Path
		}
	default:
		if len(km.String()) == 1 {
			m.buf.InsertAt(m.edView.CursorRow, m.edView.CursorCol, km.String())
			m.edView.CursorCol++
		}
	}
}

func (m *Model) taskKey(km tea.KeyMsg) {
	// no custom task-manager keys for now; view refreshes on a timer
}

// --- Mouse ------------------------------------------------------------------

func (m *Model) handleMouse(me tea.MouseMsg) {
	if me.Action != tea.MouseActionPress {
		return
	}
	x, y := me.X, me.Y
	// Dock is the bottom row(s). If the click y is within the dock band,
	// activate the dock item under it.
	if y >= m.height-2 && y <= m.height-1 {
		idx := m.dockItemAt(x)
		if idx >= 0 {
			m.setActive(idx)
		}
	}
}

// dockItemAt maps an x coordinate to a dock item, mirroring the layout math
// in View().
func (m *Model) dockItemAt(x int) int {
	var cursor int
	for i, label := range dockLabels {
		cell := dockItemCell(i, label)
		width := lipgloss.Width(cell)
		end := cursor + width
		if x >= cursor && x < end+1 {
			return i
		}
		cursor = end + 2 // item + spacing
	}
	return -1
}

func dockItemCell(i int, label string) string {
	return fmt.Sprintf("[%d] %s", i+1, label)
}

// --- Update -----------------------------------------------------------------

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	case tea.KeyMsg:
		m.handleKey(msg)
		if m.quitting {
			return m, tea.Quit
		}
	case tea.MouseMsg:
		m.handleMouse(msg)
	case refreshMsg:
		if m.active == AppTasks {
			m.lastTask = m.tasks.Refresh()
		}
		if m.active == AppTerminal {
			m.sysInfo = system.Current()
		}
		if m.active == AppSettings {
			m.settings = ""
		}
		return m, tick()
	}
	m.edView.Clamp(m.buf, m.editorHeight())
	return m, nil
}

func (m *Model) editorHeight() int {
	h := m.height - 2 - 4 // top bar, dock, borders, status
	if h < 3 {
		h = 3
	}
	return h - 3
}

// --- Quit -------------------------------------------------------------------

func (m *Model) Quitting() bool { return m.quitting }

func strconvToInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

var _ = strings.TrimSpace
