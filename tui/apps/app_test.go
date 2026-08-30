// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Tests for the root TUI model: keyboard navigation, dock switching,
// application rendering, and mouse targeting.

package apps

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"

	"veldra/tui/editor"
)

func key(s string) tea.KeyMsg {
	if s == "tab" {
		return tea.KeyMsg{Type: tea.KeyTab}
	}
	if s == "right" {
		return tea.KeyMsg{Type: tea.KeyRight}
	}
	if s == "down" {
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	if s == "left" {
		return tea.KeyMsg{Type: tea.KeyLeft}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestInitialModelStartsOnTerminal(t *testing.T) {
	m := NewModel("/", "")
	if m.active != AppTerminal {
		t.Errorf("expected active app %d (Terminal), got %d", AppTerminal, m.active)
	}
	if len(m.fs.Items) == 0 {
		t.Error("file browser should list real files in /")
	}
	if m.sysInfo.Kernel == "" {
		t.Error("system info should hold real kernel data")
	}
}

func TestDockSwitchingKeys(t *testing.T) {
	m := NewModel("/", "")
	size := tea.WindowSizeMsg{Width: 120, Height: 36}
	m.Update(size)

	// right -> Files (1)
	m.Update(key("right"))
	if m.active != AppFiles {
		t.Errorf("right should select Files (1), got %d", m.active)
	}
	// tab -> Editor (2)
	m.Update(key("tab"))
	if m.active != AppEditor {
		t.Errorf("tab should select Editor (2), got %d", m.active)
	}
	// 5 -> Task Manager
	m.Update(key("5"))
	if m.active != AppTasks {
		t.Errorf("5 should select Task Manager (4), got %d", m.active)
	}
	// 1 -> Terminal
	m.Update(key("1"))
	if m.active != AppTerminal {
		t.Errorf("1 should select Terminal (0), got %d", m.active)
	}
}

func TestQuitKeys(t *testing.T) {
	m := NewModel("/", "")
	size := tea.WindowSizeMsg{Width: 120, Height: 36}
	m.Update(size)

	upd, _ := m.Update(key("q"))
	mm := upd.(*Model)
	if !mm.Quitting() {
		t.Error("q should mark the model as quitting")
	}
}

func TestFilesAppNavigation(t *testing.T) {
	m := NewModel("/", "")
	size := tea.WindowSizeMsg{Width: 120, Height: 36}
	m.Update(size)
	m.setActive(AppFiles)
	before := m.fs.Cursor
	m.Update(key("down"))
	if m.fs.Cursor != (before+1)%m.fs.Len() {
		t.Error("down should advance the file cursor")
	}
}

func TestOpenFileInEditor(t *testing.T) {
	m := NewModel("/", "")
	size := tea.WindowSizeMsg{Width: 120, Height: 36}
	m.Update(size)
	// open the first file found in / for editor
	m.setActive(AppFiles)
	target := ""
	for _, e := range m.fs.Items {
		if !e.IsDir {
			target = m.fs.Path + "/" + e.Name
			break
		}
	}
	if target == "" {
		t.Skip("no regular file in /")
	}
	m.buf = editor.NewBuffer(target)
	m.edView = editor.ViewWindow{}
	if m.buf.Len() == 0 {
		t.Error("opened buffer should have content")
	}
}

func TestTaskManagerShowsRealProcesses(t *testing.T) {
	m := NewModel("/", "")
	size := tea.WindowSizeMsg{Width: 120, Height: 36}
	m.Update(size)
	m.setActive(AppTasks)
	out := m.renderTasks()
	if !strings.Contains(out, "PID") {
		t.Error("task view should have a PID header")
	}
	if !strings.Contains(out, "systemd") && !strings.Contains(out, "/") {
		// at minimum there should be rows of processes
		if !strings.Contains(out, "%") {
			t.Error("task view should show CPU percentages")
		}
	}
}

func TestEditorViewWindowClamp(t *testing.T) {
	m := NewModel("/", "")
	size := tea.WindowSizeMsg{Width: 120, Height: 36}
	m.Update(size)
	m.buf = editor.NewBuffer("")
	m.edView.CursorRow = 50
	m.edView.Clamp(m.buf, m.editorHeight())
	if m.edView.CursorRow != m.buf.Len()-1 {
		t.Errorf("cursor should clamp to buffer end, got %d (len %d)", m.edView.CursorRow, m.buf.Len())
	}
}

func TestDockItemAt(t *testing.T) {
	m := NewModel("/", "")
	// [1] Terminal is first at x=0..N
	idx := m.dockItemAt(1)
	if idx != 0 {
		t.Errorf("expected dock item 0 at x=1, got %d", idx)
	}
}

func TestViewRendersWindow(t *testing.T) {
	m := NewModel("/", "")
	size := tea.WindowSizeMsg{Width: 120, Height: 40}
	m.Update(size)
	out := m.View()
	if !strings.Contains(out, "●") {
		t.Error("central window should render the macOS-style bullet dots")
	}
	if !strings.Contains(out, "Terminal") {
		t.Error("dock should render the Terminal item")
	}
	if !strings.Contains(out, "Veldra") {
		t.Error("top bar should render Veldra branding")
	}
}

func TestRefreshKeepsModelAlive(t *testing.T) {
	m := NewModel("/", "")
	size := tea.WindowSizeMsg{Width: 120, Height: 40}
	m.Update(size)
	m.Update(refreshMsg{})
	if m.Quitting() {
		t.Error("a refresh must never quit the model")
	}
}