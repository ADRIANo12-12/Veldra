// Veldra
// Copyright (c) 2026 Adrian Sikora
// All rights reserved.
// Proprietary and confidential.
//
// Editor presentation helpers: scrolling window over the buffer and its
// numbered-line rendering.

package editor

// ViewWindow describes which slice of the buffer is visible.
type ViewWindow struct {
	Top      int
	CursorRow int
	CursorCol int
}

// ClampWindow ensures Top is positioned relative to the cursor row.
func (w *ViewWindow) Clamp(b *Buffer, height int) {
	if height <= 0 {
		height = 1
	}
	if w.CursorRow < 0 {
		w.CursorRow = 0
	}
	if w.CursorRow >= b.Len() {
		w.CursorRow = b.Len() - 1
		if w.CursorRow < 0 {
			w.CursorRow = 0
		}
	}
	// keep the cursor visible within [Top, Top+height)
	if w.CursorRow < w.Top {
		w.Top = w.CursorRow
	}
	if w.CursorRow >= w.Top+height {
		w.Top = w.CursorRow - height + 1
	}
	if w.Top < 0 {
		w.Top = 0
	}
	line := b.Line(w.CursorRow)
	if w.CursorCol < 0 {
		w.CursorCol = 0
	}
	if w.CursorCol > len(line) {
		w.CursorCol = len(line)
	}
}
