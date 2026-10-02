package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// A window's floor can be stated from its content, which is the thing
// WindowOptions.MinWidth cannot do: the options are given as the window is
// made, before the widgets that decide the floor exist, so a program that
// wanted one had to hard-code a constant it found by experiment.
func TestAWindowsFloorCanBeStatedFromItsContent(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Width: 800, Height: 600, MinWidth: 100, MinHeight: 80, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Two fields side by side in a splitter: a floor nothing could have
	// known when the window was made.
	left, right := widgets.NewTextField("", "", nil), widgets.NewTextField("", "", nil)
	left.PreferredWidth, right.PreferredWidth = 300, 200
	content := widgets.NewSplitter(widgets.SplitColumns, left, right)
	w.SetContent(content)
	a.PumpOnce()

	floor := widget.MinWidthOf(content)
	if floor <= 0 {
		t.Fatal("the content has no floor to state")
	}
	logical := floor / w.Scale()
	if !w.SetMinSize(logical, 0) {
		t.Fatal("the surface did not take a minimum size")
	}
	gotW, gotH := w.MinSize()
	if int(gotW) != int(logical+0.5) {
		t.Errorf("the window's floor is %g, want the content's %g", gotW, logical)
	}
	// The height it was made with is untouched by a width-only call.
	if gotH != 80 {
		t.Errorf("stating a width changed the height floor to %g, want 80", gotH)
	}
}
