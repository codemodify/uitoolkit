package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// playlist is a ListView extended by embedding, which is how the hooks
// are meant to be used.
type playlist struct{ *ListView }

// A list embedded in a wrapper knows it has the focus.
//
// RowItemState was handed the *inner* ListView receiver and compared it
// with the host's focus, which is the outer component Init was called
// with — so the comparison never matched. The rows of a focused list
// drew as unfocused and inactive, and an application extending a list
// had to work the focus out for itself to paint its own rows right.
func TestAnEmbeddedListKnowsItHasTheFocus(t *testing.T) {
	inner := NewListView(3, func(i int) string { return "row" }, nil)
	outer := &playlist{ListView: inner}
	outer.Init(outer)
	h := &host{}
	outer.SetLook(style.DarkLook())
	outer.SetHost(h)
	outer.Arrange(paintengine2d.XYWH(0, 0, 160, 100))
	outer.Selected = 1

	h.focus = outer // the window focuses the wrapper, as it must
	st := outer.rowState(1)
	if !st.Focused() {
		t.Error("the current row of a focused list is not drawn focused")
	}
	if st.Inactive() {
		t.Error("the row of a focused list is drawn inactive")
	}

	// And when the focus is elsewhere it still says so.
	h.focus = nil
	if outer.rowState(1).Focused() {
		t.Error("a row is focused with nothing focused")
	}
}

var _ widget.Component = (*playlist)(nil)
