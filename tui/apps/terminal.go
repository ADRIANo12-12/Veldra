// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// PTY-backed interactive terminal for the Veldra shell.

package apps

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	termpty "veldra/tui/terminal"
)

type ptyStartedMsg struct {
	Session *termpty.Session
	Err     error
}

type ptyOutputMsg struct {
	Data []byte
	Err  error
}

func startPTY(m *Model) tea.Cmd {
	return func() tea.Msg {
		shell := m.sysInfo.Shell
		if shell == "" { shell = "/bin/bash" }
		s, err := termpty.Start(shell, m.cwd, m.width-2, m.height-5, os.Environ())
		return ptyStartedMsg{Session: s, Err: err}
	}
}

func readPTY(s *termpty.Session) tea.Cmd {
	if s == nil { return nil }
	return func() tea.Msg {
		buf := make([]byte, 8192)
		n, err := s.Read(buf)
		if n > 0 {
			return ptyOutputMsg{Data: append([]byte(nil), buf[:n]...), Err: err}
		}
		return ptyOutputMsg{Err: err}
	}
}

func terminalKeyBytes(km tea.KeyMsg) []byte {
	switch km.String() {
	case "enter": return []byte{13}
	case "backspace": return []byte{127}
	case "delete": return []byte("\x1b[3~")
	case "tab": return []byte{9}
	case "esc": return []byte{27}
	case "up": return []byte("\x1b[A")
	case "down": return []byte("\x1b[B")
	case "right": return []byte("\x1b[C")
	case "left": return []byte("\x1b[D")
	case "home": return []byte("\x1b[H")
	case "end": return []byte("\x1b[F")
	case "pgup": return []byte("\x1b[5~")
	case "pgdown": return []byte("\x1b[6~")
	case "ctrl+c": return []byte{3}
	case "ctrl+d": return []byte{4}
	case "ctrl+z": return []byte{26}
	}
	if km.Type == tea.KeyRunes { return []byte(string(km.Runes)) }
	return nil
}
