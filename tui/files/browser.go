// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Package files implements a real terminal file browser for the Veldra TUI.
// It operates directly on the live filesystem: directory listing, sizes,
// permissions, and navigation all come from os.ReadDir / os.Stat.

package files

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is a single listed directory entry.
type Entry struct {
	Name    string
	IsDir   bool
	Size    int64
	Mode    string
	LinkTo  string
	IsLink  bool
}

// Browser navigates a real directory tree.
type Browser struct {
	Path   string
	Items  []Entry
	Cursor int
	Err    string
}

// NewBrowser starts at the given path (or $HOME).
func NewBrowser(start string) *Browser {
	if start == "" {
		if h, err := os.UserHomeDir(); err == nil && h != "" {
			start = h
		} else {
			start = "/"
		}
	}
	b := &Browser{Path: start}
	b.Reload()
	return b
}

// Reload re-reads the current directory into Items.
func (b *Browser) Reload() {
	b.Items = nil
	b.Err = ""
	if b.Cursor > b.Len()-1 {
		b.Cursor = 0
	}
	entries, err := os.ReadDir(b.Path)
	if err != nil {
		b.Err = err.Error()
		return
	}
	for _, e := range entries {
		info, ierr := e.Info()
		name := e.Name()
		ent := Entry{Name: name}
		if ierr == nil {
			ent.Size = info.Size()
			ent.Mode = info.Mode().String()
			if info.Mode()&os.ModeSymlink != 0 {
				ent.IsLink = true
				if t, terr := os.Readlink(filepath.Join(b.Path, name)); terr == nil {
					ent.LinkTo = t
				}
			}
		}
		if e.IsDir() {
			ent.IsDir = true
		} else if !ent.IsLink {
			if st, serr := os.Stat(filepath.Join(b.Path, name)); serr == nil && st.IsDir() {
				ent.IsDir = true
			}
		}
		b.Items = append(b.Items, ent)
	}
	sort.Slice(b.Items, func(i, j int) bool {
		if b.Items[i].IsDir != b.Items[j].IsDir {
			return b.Items[i].IsDir
		}
		return strings.ToLower(b.Items[i].Name) < strings.ToLower(b.Items[j].Name)
	})
}

// Len returns the number of entries.
func (b *Browser) Len() int { return len(b.Items) }

// Down moves the cursor down.
func (b *Browser) Down() {
	if b.Len() == 0 {
		return
	}
	b.Cursor = (b.Cursor + 1) % b.Len()
}

// Up moves the cursor up.
func (b *Browser) Up() {
	if b.Len() == 0 {
		return
	}
	b.Cursor = (b.Cursor - 1 + b.Len()) % b.Len()
}

// Current returns the entry under the cursor.
func (b *Browser) Current() *Entry {
	if b.Len() == 0 {
		return nil
	}
	if b.Cursor < 0 {
		b.Cursor = 0
	}
	if b.Cursor >= b.Len() {
		b.Cursor = b.Len() - 1
	}
	return &b.Items[b.Cursor]
}

// Open enters the selected directory, or returns a target path to open in
// the editor when the selection is a regular file.
func (b *Browser) Open() (editorTarget string) {
	e := b.Current()
	if e == nil {
		return ""
	}
	full := filepath.Join(b.Path, e.Name)
	if e.IsDir {
		b.Path = full
		b.Cursor = 0
		b.Reload()
		return ""
	}
	return full
}

// GoUp moves to the parent directory.
func (b *Browser) GoUp() {
	parent := filepath.Dir(b.Path)
	if parent != b.Path {
		base := filepath.Base(b.Path)
		b.Path = parent
		b.Reload()
		for i, it := range b.Items {
			if it.Name == base {
				b.Cursor = i
				break
			}
		}
	}
}

// Home jumps to the user home directory.
func (b *Browser) Home() {
	if h, err := os.UserHomeDir(); err == nil {
		b.Path = h
		b.Reload()
	}
}

// Root jumps to the filesystem root.
func (b *Browser) Root() {
	b.Path = "/"
	b.Reload()
}

// HumanSize renders a file size in human-readable units.
func HumanSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	units := []string{"K", "M", "G", "T"}
	v := float64(n)
	u := ""
	for _, unit := range units {
		v /= 1024
		u = unit
		if v < 1024 {
			break
		}
	}
	return fmt.Sprintf("%.1f%s", v, u)
}
