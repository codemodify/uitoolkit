package widgets

import (
	"testing"

	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A stack shows one page at a time, so there is nothing for the probe to
// read: it narrows a component and watches for it to grow taller, and a
// stack is as tall as its tallest page whatever its width. It has to answer
// for itself, with the widest of its pages' floors — every page, showing or
// not, so that any of them can be shown without the window having to grow.
func TestAStacksMinimumIsItsWidestPagesFloor(t *testing.T) {
	look := style.DarkLook()
	// The *wider* page is the one hidden below, so that counting only what
	// is showing would give a different, smaller answer.
	a, b := NewTextField("", "", nil), NewTextField("", "", nil)
	a.PreferredWidth, b.PreferredWidth = 260, 296
	st := NewStack(a, b)
	st.SetLook(look)
	st.SetHost(&host{})
	a.SetLook(look)
	b.SetLook(look)

	want := style.Dip(look, 296)
	if got := st.MinWidth(); got != want {
		t.Errorf("stack floor %g, want its widest page's %g", got, want)
	}
	// The page that is not showing still counts — and it is the wide one.
	b.SetVisible(false)
	if got := st.MinWidth(); got != want {
		t.Errorf("with a page hidden the floor is %g, want %g", got, want)
	}
	// And it is what MinWidthOf answers, rather than the probe's guess.
	if got := widget.MinWidthOf(st); got != want {
		t.Errorf("MinWidthOf(stack) is %g, want %g", got, want)
	}
}

// A splitter side by side is both panes' floors and the bar between them; a
// window sized from content made of splitters used to get the probe's guess.
func TestASplittersMinimumIsBothPanesAndItsBar(t *testing.T) {
	look := style.DarkLook()
	a, b := NewTextField("", "", nil), NewTextField("", "", nil)
	a.PreferredWidth, b.PreferredWidth = 300, 200
	a.SetLook(look)
	b.SetLook(look)

	sp := NewSplitter(SplitColumns, a, b)
	sp.SetLook(look)
	sp.SetHost(&host{})
	want := style.Dip(look, 300) + style.Dip(look, 200) + max(sp.bar(), 1)
	if got := sp.MinWidth(); got != want {
		t.Errorf("columns: %g, want both panes and the bar, %g", got, want)
	}

	// Split in rows the panes share the whole width, so the floor is the
	// wider of the two and the bar does not come into it.
	rows := NewSplitter(SplitRows, a, b)
	rows.SetLook(look)
	rows.SetHost(&host{})
	if got, w := rows.MinWidth(), style.Dip(look, 300); got != w {
		t.Errorf("rows: %g, want the wider pane's %g", got, w)
	}
}
