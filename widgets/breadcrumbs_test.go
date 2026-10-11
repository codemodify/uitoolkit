package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// crumbRig is a path of three steps, laid out at the width it asked for.
func crumbRig(t *testing.T, arrows bool, picked *[]int) *Breadcrumbs {
	t.Helper()
	b := NewBreadcrumbs(
		Crumb{Text: "Vault", Icon: style.IconSafe},
		Crumb{Text: "personal", Icon: style.IconFolder},
		Crumb{Text: "Logins", Icon: style.IconKey},
	)
	b.Arrows = arrows
	b.OnPick = func(i int) { *picked = append(*picked, i) }
	b.SetLook(style.DarkLook())
	b.SetHost(&host{})
	sz := b.Measure(layout.Loose(900, 60))
	b.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	return b
}

// click presses and releases at p.
func clickCrumb(b *Breadcrumbs, p paintengine2d.Point) {
	b.MousePress(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
	b.MouseRelease(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
}

// mid is the middle of a step's box.
func mid(r paintengine2d.Rect) paintengine2d.Point {
	return paintengine2d.Pt(r.Min.X+r.Dx()*0.5, r.Min.Y+r.Dy()*0.5)
}

// **A path of steps, each of which can be pressed.** There was no
// breadcrumb bar, no stepper and no row whose children overlap, so an
// application that wanted to say where its window is drew its own — the
// arrow's outline, the overlapping row and a hit test by shape.
func TestEachCrumbCanBePressed(t *testing.T) {
	for _, arrows := range []bool{false, true} {
		var picked []int
		b := crumbRig(t, arrows, &picked)
		for i := range b.Crumbs {
			r := b.CrumbRect(i)
			if r.Empty() {
				t.Fatalf("arrows=%v: step %d has no box", arrows, i)
			}
			clickCrumb(b, mid(r))
		}
		if len(picked) != 3 || picked[0] != 0 || picked[1] != 1 || picked[2] != 2 {
			t.Errorf("arrows=%v: picked %v, want each step once in order", arrows, picked)
		}
	}
}

// The steps run across the row in order and do not overlap in the
// chevron shape.
func TestTheStepsRunInOrder(t *testing.T) {
	var picked []int
	b := crumbRig(t, false, &picked)
	prev := b.CrumbRect(0)
	for i := 1; i < len(b.Crumbs); i++ {
		r := b.CrumbRect(i)
		if r.Min.X < prev.Max.X {
			t.Errorf("step %d starts at %g, before step %d ends at %g", i, r.Min.X, i-1, prev.Max.X)
		}
		prev = r
	}
}

// **A press in the notch belongs to the step before.** In the slab shape
// each step's point lies over the next one's notch, and a hit test by
// box alone gives the notch to the wrong step — which is the one thing
// the shape makes obvious and a rectangle cannot say.
func TestAPressInTheNotchGoesToTheStepBefore(t *testing.T) {
	var picked []int
	b := crumbRig(t, true, &picked)
	second := b.CrumbRect(1)
	// Just inside the second slab's box, at the middle of its leading
	// edge: that is where its notch is deepest, and what is there is the
	// first slab's point showing through.
	at := paintengine2d.Pt(second.Min.X+2, (second.Min.Y+second.Max.Y)*0.5)
	if !second.Contains(at) {
		t.Fatalf("%v is not inside the second slab's box %v", at, second)
	}
	clickCrumb(b, at)
	if len(picked) != 1 || picked[0] != 0 {
		t.Errorf("a press in the notch picked %v, want the step before it", picked)
	}
	// And the middle of the slab is still its own.
	picked = nil
	clickCrumb(b, mid(second))
	if len(picked) != 1 || picked[0] != 1 {
		t.Errorf("a press in the middle picked %v, want step 1", picked)
	}
}

// A step that offers alternatives opens them instead of going anywhere.
func TestACrumbOpensWhatItCanChangeTo(t *testing.T) {
	var picked []int
	b := crumbRig(t, false, &picked)
	opened := 0
	chosen := -1
	b.Crumbs[1].Menu = func() []*MenuItem {
		opened++
		return []*MenuItem{
			{Text: "personal", OnClick: func() { chosen = 0 }},
			{Text: "work", OnClick: func() { chosen = 1 }},
		}
	}
	h := &fakeWindow{}
	b.SetHost(h)
	clickCrumb(b, mid(b.CrumbRect(1)))
	if opened != 1 {
		t.Fatalf("the step's menu was asked for %d times", opened)
	}
	if h.popup == nil {
		t.Fatal("no drop-down under the step")
	}
	if len(picked) != 0 {
		t.Errorf("OnPick ran as well as the menu: %v", picked)
	}
	// And the item does what it says.
	pop, ok := h.popup.(*PopupMenu)
	if !ok {
		t.Fatalf("the popup is %T", h.popup)
	}
	pop.Items[1].OnClick()
	if chosen != 1 {
		t.Errorf("the chosen item did nothing")
	}
}

// **A path too long for its row keeps its end**, and the steps there was
// no room for are still reachable: the last step is where the window is,
// and a path read from the start is useless.
func TestALongPathKeepsItsEnd(t *testing.T) {
	var picked []int
	b := crumbRig(t, false, &picked)
	full := b.Measure(layout.Loose(900, 60))
	// Half the room it wanted.
	b.Arrange(paintengine2d.XYWH(0, 0, full.X*0.5, full.Y))

	if r := b.CrumbRect(len(b.Crumbs) - 1); r.Empty() {
		t.Error("the last step was dropped")
	}
	if r := b.CrumbRect(0); !r.Empty() {
		t.Errorf("the first step is still shown at %v in half the room", r)
	}
	if !b.overflow || len(b.hidden) == 0 {
		t.Fatalf("overflow=%v hidden=%v", b.overflow, b.hidden)
	}
	// The ellipsis stands in for them, and opens them.
	h := &fakeWindow{}
	b.SetHost(h)
	ell := b.CrumbRect(crumbOverflow)
	if ell.Empty() {
		t.Fatal("no ellipsis step")
	}
	clickCrumb(b, mid(ell))
	pop, ok := h.popup.(*PopupMenu)
	if !ok {
		t.Fatalf("the ellipsis opened %T", h.popup)
	}
	if len(pop.Items) != len(b.hidden) {
		t.Errorf("the drop-down has %d items for %d hidden steps", len(pop.Items), len(b.hidden))
	}
	pop.Items[0].OnClick()
	if len(picked) != 1 || picked[0] != b.hidden[0] {
		t.Errorf("picking a hidden step gave %v, want %v", picked, b.hidden[:1])
	}
	// A row that can show nothing else can always show "… › here".
	if min := b.MinWidth(); min <= 0 || min > full.X {
		t.Errorf("MinWidth is %g of a natural %g", min, full.X)
	}
}

// The keyboard walks the path: one tab stop for the row, the arrows
// inside it, Enter to go.
func TestTheKeyboardWalksThePath(t *testing.T) {
	var picked []int
	b := crumbRig(t, false, &picked)
	if got := b.FocusIndex(); got != 0 {
		t.Fatalf("the keyboard starts on step %d", got)
	}
	key := func(k platform.Key) bool { return b.KeyPress(widget.KeyEvent{Key: k}) }
	if !key(platform.KeyRight) || b.FocusIndex() != 1 {
		t.Errorf("Right left the keyboard on %d", b.FocusIndex())
	}
	if !key(platform.KeyEnd) || b.FocusIndex() != 2 {
		t.Errorf("End left the keyboard on %d", b.FocusIndex())
	}
	// It stops at the ends rather than wrapping: a path has two.
	if !key(platform.KeyRight) || b.FocusIndex() != 2 {
		t.Errorf("Right past the end left the keyboard on %d", b.FocusIndex())
	}
	if !key(platform.KeyReturn) || len(picked) != 1 || picked[0] != 2 {
		t.Errorf("Enter picked %v, want the step the keyboard was on", picked)
	}
	// Keys that are not the row's are left alone, so the window's own
	// shortcuts still work while the keyboard is on the path.
	for _, k := range []platform.Key{platform.KeyPageUp, platform.KeyTab, platform.KeyEscape} {
		if key(k) {
			t.Errorf("the row took %v", k)
		}
	}
}

// A reader hears a path as a path: a row of buttons, each with its place
// in it.
func TestThePathIsReadAsOne(t *testing.T) {
	var picked []int
	b := crumbRig(t, false, &picked)
	var n a11y.Node
	b.Describe(&n)
	if n.Role != a11y.RoleToolBar {
		t.Errorf("the row is a %v", n.Role)
	}
	items := b.AccessibleItems()
	if len(items) != 3 {
		t.Fatalf("%d items for 3 steps", len(items))
	}
	for i, it := range items {
		if it.Role != a11y.RoleButton || it.Name != b.Crumbs[i].Text {
			t.Errorf("item %d is a %v named %q", i, it.Role, it.Name)
		}
		if it.Index != i+1 || it.Count != 3 {
			t.Errorf("item %d says %d of %d", i, it.Index, it.Count)
		}
		if !it.Actions.Has(a11y.ActionDefault) {
			t.Errorf("item %d cannot be pressed", i)
		}
	}
	if !b.AccessibleAction(1, a11y.ActionDefault) || len(picked) != 1 || picked[0] != 1 {
		t.Errorf("the reader's press picked %v", picked)
	}
}
