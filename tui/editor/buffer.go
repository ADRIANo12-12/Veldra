// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Package editor implements a real lightweight terminal editor/viewer for
// the Veldra TUI. It operates on the actual filesystem: opens files line
// by line, numbers them, scrolls with the cursor, supports basic insertion
// and deletion, and saves back to disk. It is deliberately small.

package editor

import (
	"os"
	"strings"
)

// Buffer holds a single file's lines.
type Buffer struct {
	Path string
	Lines []string
	ReadOnly bool
	Modified bool
	Err  string
}

// NewBuffer loads a file (or an empty buffer) into memory.
func NewBuffer(path string) *Buffer {
	b := &Buffer{Path: path}
	if path == "" {
		b.Lines = []string{""}
		return b
	}
	data, err := os.ReadFile(path)
	if err != nil {
		b.Err = err.Error()
		b.Lines = []string{""}
		b.ReadOnly = true
		return b
	}
	content := string(data)
	if content == "" {
		b.Lines = []string{""}
	} else {
		b.Lines = strings.Split(content, "\n")
	}
	if st, serr := os.Stat(path); serr == nil {
		if st.Mode().Perm()&0222 == 0 {
			b.ReadOnly = true
		}
	}
	return b
}

// Line returns the line at the given index, clamped.
func (b *Buffer) Line(i int) string {
	if i < 0 || i >= len(b.Lines) {
		return ""
	}
	return b.Lines[i]
}

// Len returns the number of lines.
func (b *Buffer) Len() int { return len(b.Lines) }

// InsertAt inserts text at (row,col).
func (b *Buffer) InsertAt(row, col int, text string) {
	if b.ReadOnly || row < 0 || row >= len(b.Lines) {
		return
	}
	line := b.Lines[row]
	if col < 0 {
		col = 0
	}
	if col > len(line) {
		col = len(line)
	}
	b.Lines[row] = line[:col] + text + line[col:]
	b.Modified = true
}

// DeleteAt removes one character at (row,col).
func (b *Buffer) DeleteAt(row, col int) bool {
	if b.ReadOnly || row < 0 || row >= len(b.Lines) {
		return false
	}
	line := b.Lines[row]
	if col < 0 || col >= len(line) {
		return false // would need join logic; handle backspace at col 0 below
	}
	b.Lines[row] = line[:col] + line[col+1:]
	b.Modified = true
	return true
}

// SplitLine splits the line at the cursor (pressing Enter).
func (b *Buffer) SplitLine(row, col int) bool {
	if b.ReadOnly || row < 0 || row >= len(b.Lines) {
		return false
	}
	line := b.Lines[row]
	if col > len(line) {
		col = len(line)
	}
	if col < 0 {
		col = 0
	}
	head, tail := line[:col], line[col:]
	newLines := make([]string, 0, len(b.Lines)+1)
	newLines = append(newLines, b.Lines[:row]...)
	newLines = append(newLines, head, tail)
	newLines = append(newLines, b.Lines[row+1:]...)
	b.Lines = newLines
	b.Modified = true
	return true
}

// Save writes the buffer back to disk.
func (b *Buffer) Save() error {
	if b.ReadOnly {
		return os.ErrPermission
	}
	if b.Path == "" {
		return nil
	}
	return os.WriteFile(b.Path, []byte(strings.Join(b.Lines, "\n")), 0644)
}

// Find returns the first line matching a substring, or -1.
func (b *Buffer) Find(sub string) int {
	for i, l := range b.Lines {
		if strings.Contains(l, sub) {
			return i
		}
	}
	return -1
}
