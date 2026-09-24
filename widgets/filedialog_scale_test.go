package widgets

import (
	"testing"

	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// The dialog asks for a card big enough to hold a path field, a table and
// a row of buttons. Every one of those is laid out in device pixels, so
// the card has to be asked for in device pixels too. It was not: it asked
// for a flat 520x420 whatever the scale, and on a 1.75 desktop the title
// was clipped, the table ran past the right edge and the buttons crowded
// the bottom. Reported from a KDE session, 2026-09-24.
func TestFileDialogAsksForACardThatScales(t *testing.T) {
	at := func(scale float32) (w, h float32) {
		fd := NewFileDialog(FileDialogOptions{Mode: FileOpen, Path: t.TempDir()})
		fd.SetHost(&scaledHost{look: style.WithScale(style.DarkLook(), scale)})
		sz := fd.Measure(layout.Unbounded())
		return sz.X, sz.Y
	}
	w1, h1 := at(1)
	w2, h2 := at(2)
	if w1 <= 0 || h1 <= 0 {
		t.Fatalf("no size at all: %vx%v", w1, h1)
	}
	if w2 < w1*1.9 || h2 < h1*1.9 {
		t.Errorf("the card does not grow with the scale: %vx%v at 1, %vx%v at 2", w1, h1, w2, h2)
	}
}
