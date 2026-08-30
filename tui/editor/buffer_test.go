// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Tests for the real terminal editor buffer.

package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewBufferReadsRealFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sample.txt")
	content := "line1\nline2\nline3"
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	b := NewBuffer(p)
	if b.Len() != 3 {
		t.Fatalf("expected 3 lines, got %d", b.Len())
	}
	if b.Line(1) != "line2" {
		t.Errorf("expected line2 at index 1, got %q", b.Line(1))
	}
	if b.Err != "" {
		t.Errorf("unexpected error: %v", b.Err)
	}
}

func TestSaveWritesRealFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "out.txt")
	b := NewBuffer("")
	b.Lines = []string{"a", "b"}
	b.Path = p
	if err := b.Save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a\nb" {
		t.Errorf("unexpected saved content: %q", string(data))
	}
}

func TestInsertAtAndDocumentModified(t *testing.T) {
	b := NewBuffer("")
	b.Lines = []string{"hello"}
	b.InsertAt(0, 1, "X")
	if b.Line(0) != "hXello" {
		t.Errorf("expected hXello, got %q", b.Line(0))
	}
	if !b.Modified {
		t.Error("insert should mark the buffer modified")
	}
}

func TestSplitLine(t *testing.T) {
	b := NewBuffer("")
	b.Lines = []string{"ab", "cd"}
	ok := b.SplitLine(0, 1)
	if !ok {
		t.Fatal("split should succeed")
	}
	if b.Line(0) != "a" || b.Line(1) != "b" || b.Line(2) != "cd" {
		t.Errorf("unexpected split result: %v", b.Lines)
	}
}

func TestFind(t *testing.T) {
	b := NewBuffer("")
	b.Lines = []string{"foo", "package main", "bar"}
	if i := b.Find("package"); i != 1 {
		t.Errorf("expected find at index 1, got %d", i)
	}
	if i := b.Find("zzz"); i != -1 {
		t.Errorf("expected -1, got %d", i)
	}
}

func TestViewWindowClamp(t *testing.T) {
	b := NewBuffer("")
	b.Lines = []string{"a", "b", "c", "d", "e"}
	w := &ViewWindow{CursorRow: 4}
	w.Clamp(b, 3)
	if w.Top != 2 {
		t.Errorf("expected Top=2, got %d", w.Top)
	}
}