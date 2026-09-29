package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// A click is aimed at something the user can see. A button below a
// ScrollView's fold is at coordinates outside the view, so clicking where
// its box says it is pressed whatever happened to be there — and an
// application's own test read that as a button that did not work.
func TestClickComponentRevealsBelowTheFold(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 160, Headless: true})
	if err != nil {
		t.Fatal(err)
	}

	col := widgets.NewColumn().WithGap(8)
	for i := 0; i < 20; i++ {
		col.Add(widgets.NewLabel("filler"))
	}
	clicked := 0
	target := widgets.NewButton("Apply", func() { clicked++ })
	col.Add(target)

	sv := widgets.NewScrollView(col)
	w.SetContent(sv)
	a.PumpOnce()

	if !w.ClickComponent(target) {
		t.Fatal("ClickComponent refused")
	}
	if clicked != 1 {
		t.Errorf("the button below the fold was pressed %d times, want 1", clicked)
	}
}

// Something already in view is clicked without the scroll position
// moving, so revealing does not disturb a test that did not need it.
func TestClickComponentLeavesAVisibleTargetAlone(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 400, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	clicked := 0
	target := widgets.NewButton("Apply", func() { clicked++ })
	col := widgets.NewColumn(target)
	sv := widgets.NewScrollView(col)
	w.SetContent(sv)
	a.PumpOnce()

	before := sv.OffsetY
	if !w.ClickComponent(target) {
		t.Fatal("ClickComponent refused")
	}
	if clicked != 1 {
		t.Errorf("clicked %d times, want 1", clicked)
	}
	if sv.OffsetY != before {
		t.Errorf("the view scrolled to %v from %v for a target already in it", sv.OffsetY, before)
	}
}
