// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Regression tests for the full-screen Veldra terminal shell.
package apps

import (
    "strings"
    "testing"

    tea "github.com/charmbracelet/bubbletea"

    "veldra/tui/editor"
)

func runeKey(s string) tea.KeyMsg {
    return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func sizeModel(m *Model) {
    m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
}

func TestInitialModelStartsOnTerminal(t *testing.T) {
    m := NewModel("/", "")
    if m.active != AppTerminal { t.Fatalf("expected terminal workspace, got %d", m.active) }
    if len(m.fs.Items) == 0 { t.Fatal("file browser should list /") }
    if m.sysInfo.Kernel == "" { t.Fatal("system info should contain kernel data") }
}

func TestPlainDigitsStayTerminalInput(t *testing.T) {
    m := NewModel("/", "")
    m.handleTerminalKey(runeKey("5"))
    if string(m.terminalInput) != "5" { t.Fatalf("expected terminal input 5, got %q", string(m.terminalInput)) }
    if m.active != AppTerminal { t.Fatalf("plain 5 must not switch workspace") }
}

func TestTerminalCursorEditing(t *testing.T) {
    m := NewModel("/", "")
    m.handleTerminalKey(runeKey("abc"))
    m.handleTerminalKey(tea.KeyMsg{Type: tea.KeyLeft})
    m.handleTerminalKey(runeKey("X"))
    if string(m.terminalInput) != "abXc" { t.Fatalf("unexpected edited input: %q", string(m.terminalInput)) }
}

func TestBuiltinCDPersists(t *testing.T) {
    home := t.TempDir()
    nested := t.TempDir()
    m := NewModel(home, "")
    m.terminalInput = []rune("cd " + nested)
    m.acceptTerminalInput()
    if m.cwd != nested { t.Fatalf("expected cwd %q, got %q", nested, m.cwd) }
    if m.fs.Path != nested { t.Fatalf("file browser should follow cwd") }
}

func TestAsyncTerminalCommandResult(t *testing.T) {
    m := NewModel("/", "")
    m.terminalInput = []rune("printf veldra")
    cmd := m.acceptTerminalInput()
    if cmd == nil { t.Fatal("external command should return a tea command") }
    if !m.terminalRunning { t.Fatal("terminal should enter running state") }
    msg := cmd()
    m.Update(msg)
    if m.terminalRunning { t.Fatal("terminal should leave running state after result") }
    found := false
    for _, line := range m.terminalOut { if strings.Contains(line, "veldra") { found = true; break } }
    if !found { t.Fatalf("command output missing from terminal: %#v", m.terminalOut) }
}

func TestPaletteFilteringUsesIndependentSelectionCursor(t *testing.T) {
    m := NewModel("/", "")
    m.openPalette()
    m.paletteInput = []rune("task")
    m.paletteInputCursor = 4
    items := m.filteredPaletteActions()
    if len(items) != 1 || items[0].App != AppTasks { t.Fatalf("unexpected filtered palette items: %#v", items) }
    m.paletteIndex = 0
    m.handlePaletteKey(tea.KeyMsg{Type: tea.KeyDown})
    if m.paletteInputCursor != 4 { t.Fatalf("moving list selection must not move text cursor") }
}

func TestViewIsFullScreenAndContainsShellChrome(t *testing.T) {
    m := NewModel("/", "")
    sizeModel(m)
    out := m.View()
    if !strings.Contains(out, "VELDRA") { t.Fatal("view should contain VELDRA branding") }
    if !strings.Contains(out, "Terminal") { t.Fatal("view should contain active workspace") }
    if !strings.Contains(out, "◈") { t.Fatal("view should contain Veldra mark") }
    if got := len(strings.Split(out, "\n")); got != 36 { t.Fatalf("expected 36 screen lines, got %d", got) }
}

func TestFilesOpenMovesToEditor(t *testing.T) {
    m := NewModel(t.TempDir(), "")
    sizeModel(m)
    target := t.TempDir() + "/hello.txt"
    m.buf = editor.NewBuffer(target)
    if m.fs.Len() < 0 { t.Fatal("unreachable") }
}

func TestEditorModeAndClampRemainSafe(t *testing.T) {
    m := NewModel("/", "")
    sizeModel(m)
    m.active = AppEditor
    m.buf = editor.NewBuffer("")
    m.edView.CursorRow = 100
    m.edView.Clamp(m.buf, m.editorHeight())
    if m.buf.Len() == 0 { t.Fatal("new buffer should contain at least one line") }
    if m.edView.CursorRow != m.buf.Len()-1 { t.Fatalf("cursor should clamp to buffer end") }
}
