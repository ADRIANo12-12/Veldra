// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Low-level PTY session wrapper used by the Veldra TUI.

package terminal

import (
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

type Session struct {
	cmd  *exec.Cmd
	file *os.File
	mu   sync.Mutex
}

func Start(shell string, cols, rows int, env []string) (*Session, error) {
	if shell == "" { shell = "/bin/bash" }
	cmd := exec.Command(shell, "-i")
	cmd.Env = append([]string{}, env...)
	cmd.Env = append(cmd.Env,
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"VELDRA_TUI_STARTED=1",
		"TERM_PROGRAM=Veldra",
	)
	file, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(max(cols, 40)), Rows: uint16(max(rows, 12))})
	if err != nil { return nil, err }
	return &Session{cmd: cmd, file: file}, nil
}

func (s *Session) Read(p []byte) (int, error) { return s.file.Read(p) }

func (s *Session) Write(p []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil { return os.ErrClosed }
	_, err := s.file.Write(p)
	return err
}

func (s *Session) Resize(cols, rows int) error {
	if s.file == nil { return os.ErrClosed }
	return pty.Setsize(s.file, &pty.Winsize{Cols: uint16(max(cols, 40)), Rows: uint16(max(rows, 12))})
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil { return nil }
	if s.cmd.Process != nil { _ = s.cmd.Process.Signal(os.Interrupt) }
	_ = s.file.Close()
	s.file = nil
	return nil
}

func max(v, fallback int) int {
	if v < fallback { return fallback }
	return v
}
