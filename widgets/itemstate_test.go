package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// backdropWindow is a test host whose window has lost keyboard focus.
type backdropWindow struct{ fakeWindow }

func (*backdropWindow) Active() bool { return false }

// Rows get the whole item state: the current row of a focused view is
// Focused (its focus mark), a view without focus marks its rows Inactive
// (Windows and Mac looks grey the selection), and an inactive window adds
// Backdrop (every look may subdue it).
func TestItemStatesFollowFocus(t *testing.T) {
	tv := NewTableView([]TableColumn{{Title: "A"}}, 5, func(r, c int) string { return fmt.Sprint(r) }, nil)
	host := &fakeWindow{look: style.DarkLook()}
	tv.SetHost(host)
	tv.Arrange(paintengine2d.XYWH(0, 0, 200, 200))
	tv.Selected = 2

	st := tv.rowState(2)
	if !st.Checked() || st.Focused() || !st.Inactive() || st.Backdrop() {
		t.Fatalf("unfocused view: row state %b, want selected+inactive", st)
	}
	host.RequestFocus(tv)
	st = tv.rowState(2)
	if !st.Checked() || !st.Focused() || st.Inactive() {
		t.Fatalf("focused view: current row %b, want selected+focused, not inactive", st)
	}
	if other := tv.rowState(1); other.Focused() || other.Checked() {
		t.Fatalf("focused view: another row %b", other)
	}

	back := &backdropWindow{fakeWindow{look: style.DarkLook()}}
	tv.SetHost(back)
	back.RequestFocus(tv)
	st = tv.rowState(2)
	if !st.Backdrop() || !st.Inactive() || st.Focused() {
		t.Fatalf("inactive window: row state %b, want backdrop+inactive, no focus mark", st)
	}
}

// A modern look keeps a selection bright while only the view lost focus
// (GTK / Qt) and subdues it in the backdrop; a classic one greys it as soon
// as the view is unfocused (Windows).
func TestSelectionDimsPerLookFamily(t *testing.T) {
	pick := func(lk style.LookAndFeel, st style.ControlState) paintengine2d.Color {
		img := paintengine2d.NewImage(40, 20)
		ctx := paintengine2d.NewContext(img)
		lk.DrawListRow(ctx, paintengine2d.XYWH(0, 0, 40, 20), st, "")
		r, g, b, a := img.At(20, 10).RGBA()
		return paintengine2d.RGBA(float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535)
	}
	for _, name := range []string{"fluent", "win95"} {
		p, ok := style.LoadTheme(name)
		if !ok {
			t.Fatalf("pack %s", name)
		}
		lk := p.Look()
		on := pick(lk, style.StateChecked)
		unfocused := pick(lk, style.StateChecked|style.StateInactive)
		backdrop := pick(lk, style.StateChecked|style.StateInactive|style.StateBackdrop)
		classic := name == "win95"
		if (unfocused != on) != classic {
			t.Fatalf("%s: unfocused selection %v vs focused %v (classic=%v)", name, unfocused, on, classic)
		}
		if backdrop == on {
			t.Fatalf("%s: backdrop selection not subdued", name)
		}
	}
}
