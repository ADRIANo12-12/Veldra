// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Tests for the real terminal file browser.

package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserListsRealDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	b := NewBrowser(root)
	if b.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", b.Len())
	}
	// dirs sort first
	if !b.Items[0].IsDir {
		t.Error("directories should sort before files")
	}
	// sizes are real
	fileEntry := b.Items[1]
	if fileEntry.Size != 1 {
		t.Errorf("expected real size 1, got %d", fileEntry.Size)
	}
}

func TestNavigateAndOpen(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "f.txt")
	if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	b := NewBrowser(root)
	b.Down() // move to sub
	if b.Current() == nil || !b.Current().IsDir {
		t.Fatalf("expected sub dir at cursor")
	}
	if target := b.Open(); target != "" {
		t.Errorf("opening a dir should not return an editor target, got %s", target)
	}
	if b.Path != sub {
		t.Errorf("expected path %s, got %s", sub, b.Path)
	}
	if b.Len() != 1 {
		t.Fatalf("expected 1 entry in sub, got %d", b.Len())
	}
	// open the file -> editor target
	if target := b.Open(); target != file {
		t.Errorf("expected editor target %s, got %s", file, target)
	}
}

func TestGoUp(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	b := NewBrowser(sub)
	b.GoUp()
	if b.Path != root {
		t.Errorf("expected up to %s, got %s", root, b.Path)
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		512:     "512B",
		2048:    "2.0K",
		1048576: "1.0M",
	}
	for in, want := range cases {
		if got := HumanSize(in); got != want {
			t.Errorf("HumanSize(%d) = %q, want %q", in, got, want)
		}
	}
}