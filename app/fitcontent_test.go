package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// tallBy builds a column of n labels, so a test can ask for a content
// height it knows the shape of.
func tallBy(n int) *widgets.FlexBox {
	col := widgets.NewColumn().WithGap(4)
	for i := 0; i < n; i++ {
		col.Add(widgets.NewLabel("a line of text"))
	}
	return col
}

// A window opened with FitContent is as tall as what is in it, at every
// scale, with no clipped last row and no empty band.
func TestFitContentAtEveryScale(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		a := New(Options{Look: style.DarkLook(), Headless: true, Scale: scale})
		w, err := a.NewWindow(platform.WindowOptions{
			Width: 320, Height: 600, Headless: true, FitContent: true,
			Sizing: platform.SizingFixed,
		})
		if err != nil {
			t.Fatal(err)
		}
		content := tallBy(3)
		w.SetContent(content)
		a.PumpOnce()

		// What the content wanted at the width it was given.
		box := content.Bounds()
		want := content.Measure(layout.Constraints{MinW: box.Dx(), MaxW: box.Dx(), MaxH: -1})
		if box.Dy() < want.Y-0.5 {
			t.Fatalf("%.1fx: the content has %v of the %v it wants (clipped)", scale, box.Dy(), want.Y)
		}
		if box.Dy() > want.Y+2 {
			t.Fatalf("%.1fx: the content has %v for the %v it wants (empty band)", scale, box.Dy(), want.Y)
		}
		// And much shorter than the 600 it was opened at.
		if _, h := w.Size(); h > 400 {
			t.Fatalf("%.1fx: the window is %d logical pixels tall, want the content's", scale, h)
		}
	}
}

// FitToContent does the same afterwards, and follows the content when it
// changes.
func TestFitToContentFollowsTheContent(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 320, Height: 600, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(tallBy(2))
	a.PumpOnce()
	if !w.FitToContent() {
		t.Fatal("FitToContent refused")
	}
	a.PumpOnce()
	_, short := w.Size()

	w.SetContent(tallBy(8))
	a.PumpOnce()
	if !w.FitToContent() {
		t.Fatal("FitToContent refused the taller content")
	}
	a.PumpOnce()
	_, tall := w.Size()
	if tall <= short {
		t.Fatalf("eight lines is %d tall, two lines was %d", tall, short)
	}
}

// It includes the caption where the toolkit draws the frame, which is
// the half a caller cannot compute: the caption's height is the look's.
func TestFitContentIncludesAToolkitFrame(t *testing.T) {
	sizeWith := func(deco platform.Decorations) int {
		a := New(Options{Look: style.DarkLook(), Headless: true})
		w, err := a.NewWindow(platform.WindowOptions{
			Width: 320, Height: 600, Headless: true, Decorations: deco,
		})
		if err != nil {
			t.Fatal(err)
		}
		w.SetContent(tallBy(3))
		a.PumpOnce()
		w.FitToContent()
		a.PumpOnce()
		_, h := w.Size()
		return h
	}
	server := sizeWith(platform.DecorationsServer)
	client := sizeWith(platform.DecorationsClient)
	if client <= server {
		t.Fatalf("with the toolkit's frame the window is %d tall, with the desktop's %d — the caption was not counted", client, server)
	}
}

// A window with no content has nothing to fit to, and says so.
func TestFitToContentWithoutContent(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	if w.FitToContent() {
		t.Fatal("a window with no content fitted to it")
	}
	w.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()
	w.Close()
	if w.FitToContent() {
		t.Fatal("a closed window fitted")
	}
}
