package dock

import (
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/internal/uitest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// tearRig is a mounted host whose panels float into fake windows, with the
// desktop's two answers — whether it carries a window under a drag, and
// where it says this window is — under the test's control.
type tearRig struct {
	*rig
	opener *fakeOpener
}

func newTearRig(t *testing.T, carries bool) *tearRig {
	t.Helper()
	r := &tearRig{rig: newRig(t), opener: &fakeOpener{}}
	r.host.SetWindowOpener(r.opener)
	r.Host.SetCarriesWindows(carries)
	r.Host.SetPosition(100, 50, true)
	return r
}

// tear is the tear-off drag the host started, failing the test when there
// is none.
func (r *tearRig) tear(t *testing.T) *uitest.TearOff {
	t.Helper()
	got := r.Host.TearOff()
	if got.Tear == nil {
		t.Fatal("no tear-off drag was started")
	}
	return got
}

// mountFloat floats p and lays its chrome out the way the window it now
// lives in would, so its title bar can be pressed.
func (r *tearRig) mountFloat(t *testing.T, p *Panel) *stackHead {
	t.Helper()
	if !r.host.FloatPanel(p, paintengine2d.XYWH(0, 0, 300, 400)) {
		t.Fatal("the panel refused to float")
	}
	r.Layout()
	st := stackOf(t, p)
	uitest.MountHost(r.Host, st, paintengine2d.XYWH(0, 0, 300, 400))
	return st.head
}

// belowTheHost is a point well clear of the host's bottom edge.
func (r *tearRig) belowTheHost() paintengine2d.Point {
	b := r.host.LocalBounds()
	return paintengine2d.Pt(b.Dx()*0.5, b.Max.Y+60)
}

// A panel dragged inside the window docks inside it; the same drag taken
// out of the window floats the panel into a window of its own, which the
// desktop then carries under the pointer.
func TestPanelDraggedOutOfTheWindowFloats(t *testing.T) {
	r := newTearRig(t, true)
	grip := r.gripOf(t, r.tree)
	r.MousePress(grip, platform.ButtonLeft)
	r.Session.MouseMove(paintengine2d.Pt(grip.X+40, grip.Y+40))
	if r.tree.Floating() || r.Host.TearOff().Tear != nil {
		t.Fatal("a drag inside the window keeps the panel in it")
	}
	r.Session.MouseMove(r.belowTheHost())

	if !r.tree.Floating() {
		t.Fatal("dragged out of the window, the panel should float")
	}
	if len(r.opener.wins) != 1 || !r.opener.wins[0].shown {
		t.Fatalf("opened %d windows", len(r.opener.wins))
	}
	if r.host.dragArmed() || r.host.ind.Visible() {
		t.Error("the host's own drag and its indicator end when the panel leaves")
	}
	tear := r.tear(t)
	d := tear.Drag
	if len(d.Types) != 1 || d.Types[0] != PanelMimeType {
		t.Fatalf("types %v", d.Types)
	}
	if b, ok := d.Read(PanelMimeType); !ok || string(b) != "tree" {
		t.Fatalf("the type carries the panel's layout name: %q ok=%v", b, ok)
	}
	if pan, _ := d.Payload.(*Panel); pan != r.tree {
		t.Fatalf("payload %#v", d.Payload)
	}
	if d.Preferred != platform.DragMove {
		t.Fatalf("a panel moves rather than copies: %v", d.Preferred)
	}
	if tear.Window == nil {
		t.Fatal("the drag has no window to carry")
	}
	// The window was asked for at the pointer, in the desktop's own
	// coordinates — which is the position the backend now reports.
	want := r.belowTheHost()
	geom := r.opener.wins[0].geom
	if geom.Min.X != 100+want.X-tear.Tear.Offset.X || geom.Min.Y != 50+want.Y-tear.Tear.Offset.Y {
		t.Fatalf("the window was asked for at %v, offset %v", geom, tear.Tear.Offset)
	}
}

// At a display scale the host lays out in device pixels and a window is
// placed and sized in logical ones: a panel torn out is asked for at the
// pointer less the point taken hold of, both divided by the scale, and at
// the logical size of the room it had docked.
func TestPanelTornOutAtAScaleIsPlacedInLogicalPixels(t *testing.T) {
	r := newTearRig(t, true)
	r.Host.SetScale(2)
	r.Layout()
	docked := r.rectOf(r.tree)
	grip := r.gripOf(t, r.tree)
	r.MousePress(grip, platform.ButtonLeft)
	r.Session.MouseMove(paintengine2d.Pt(grip.X+40, grip.Y+40))
	r.Session.MouseMove(r.belowTheHost())
	if !r.tree.Floating() {
		t.Fatal("dragged out of the window, the panel should float")
	}
	tear := r.tear(t)
	at, off := r.belowTheHost(), tear.Tear.Offset
	geom := r.opener.wins[0].geom
	wantX := 100 + float32(math.Round(float64((at.X-off.X)/2)))
	wantY := 50 + float32(math.Round(float64((at.Y-off.Y)/2)))
	if geom.Min.X != wantX || geom.Min.Y != wantY {
		t.Errorf("the window was asked for at %v, want %g,%g (pointer %v, offset %v, scale 2)",
			geom.Min, wantX, wantY, at, off)
	}
	if want := float32(math.Round(float64(docked.Dy() / 2))); geom.Dy() != want {
		t.Errorf("the window is %g logical pixels tall, want %g: the %g device pixels it had docked",
			geom.Dy(), want, docked.Dy())
	}
	if off.X < 0 || off.X >= geom.Dx()*2 || off.Y < 0 || off.Y >= geom.Dy()*2 {
		t.Errorf("the offset %v is outside the window it is in (%gx%g device pixels)", off, geom.Dx()*2, geom.Dy()*2)
	}
}

// How a torn-off panel's drag can end: put back where it was, left
// floating where it was dropped, or docked by whoever took it.
func TestTornOffPanelEndings(t *testing.T) {
	t.Run("cancelled docks it back", func(t *testing.T) {
		r := newTearRig(t, true)
		home := stackOf(t, r.tree)
		r.host.DockInto(r.prop, r.tree) // the home stack survives the float
		home.SelectPanel(r.tree)        // the title bar drags the panel showing
		r.Layout()
		r.Drag(r.gripOf(t, r.tree), r.belowTheHost())
		tear := r.tear(t)
		if !r.tree.Floating() {
			t.Fatal("the panel should be floating mid-drag")
		}
		tear.Tear.Done(widget.TearCancelled, tear.Window)
		r.Layout()
		if r.tree.Floating() {
			t.Fatal("a cancelled tear-off docks the panel back")
		}
		if stackOf(t, r.tree) != home {
			t.Error("it went back somewhere else")
		}
		if !r.opener.wins[0].closed {
			t.Error("the window it was carried in is still open")
		}
	})

	t.Run("kept leaves it floating", func(t *testing.T) {
		r := newTearRig(t, true)
		r.Drag(r.gripOf(t, r.tree), r.belowTheHost())
		tear := r.tear(t)
		r.opener.wins[0].geom = paintengine2d.XYWH(11, 22, 300, 400)
		tear.Tear.Done(widget.TearKept, tear.Window)
		if !r.tree.Floating() || r.opener.wins[0].closed {
			t.Fatal("a panel dropped on the desktop stays in its window")
		}
		if got := r.tree.FloatGeometry(); got.Min.X != 11 || got.Min.Y != 22 {
			t.Errorf("where the desktop left it is worth remembering: %v", got)
		}
	})
}

// Where the desktop cannot carry a window, the panel stays where it is
// until the drop and floats then — the same fallback the tab strip takes.
func TestPanelTearOffWithoutACarriedWindow(t *testing.T) {
	r := newTearRig(t, false)
	r.Drag(r.gripOf(t, r.tree), r.belowTheHost())
	tear := r.tear(t)
	if r.tree.Floating() || len(r.opener.wins) != 0 {
		t.Fatal("nothing floats while the drag runs")
	}
	win := tear.Tear.Open()
	if win == nil || !r.tree.Floating() || len(r.opener.wins) != 1 {
		t.Fatal("at the drop the panel floats")
	}
	if got := r.opener.wins[0].geom; got.Dx() <= 0 || got.Dy() <= 0 {
		t.Errorf("the window was asked for at %v, with no size", got)
	}
	tear.Tear.Done(widget.TearKept, win)
	if !r.tree.Floating() {
		t.Error("the panel should still be floating")
	}
}

// A floating panel's title bar drags the window itself where the desktop
// can carry one, and falls back to the desktop's interactive move where it
// cannot — which is what it always did.
func TestFloatingTitleBarDragsTheWindow(t *testing.T) {
	for _, tc := range []struct {
		name         string
		carries      bool
		tears, moves int
	}{
		{"carried by the desktop", true, 1, 0},
		{"the desktop's own move", false, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newTearRig(t, tc.carries)
			head := r.mountFloat(t, r.tree)
			hr := head.LocalBounds()
			head.MousePress(widget.MouseEvent{
				Pos:    paintengine2d.Pt(hr.Min.X+6, hr.Min.Y+hr.Dy()*0.5),
				Button: platform.ButtonLeft,
			})
			if got := r.Host.TearOff().Starts; got != tc.tears {
				t.Errorf("tear-offs started: %d want %d", got, tc.tears)
			}
			if got := r.opener.wins[0].moves; got != tc.moves {
				t.Errorf("interactive moves: %d want %d", got, tc.moves)
			}
		})
	}
}

// A floating panel dropped on its host docks where the indicator said,
// with the same targets a drag inside the window has.
func TestFloatingPanelDroppedOnTheHostDocks(t *testing.T) {
	r := newTearRig(t, true)
	r.mountFloat(t, r.tree)
	if got := r.host.DropTypes(); len(got) != 1 || got[0] != PanelMimeType {
		t.Fatalf("DropTypes %v", got)
	}
	if got := r.host.DropActionFor(platform.DragCopy | platform.DragMove); got != platform.DragMove {
		t.Fatalf("docking a panel moves it: %v", got)
	}

	// Over the middle of the properties pane: the indicator covers it and
	// a drop there tabs the two together.
	pane := r.rectOf(stackOf(t, r.prop))
	mid := paintengine2d.Pt(pane.Min.X+pane.Dx()*0.5, pane.Min.Y+pane.Dy()*0.5)
	r.host.DragOver(mid)
	if !r.host.ind.Visible() {
		t.Fatal("the drop indicator should show where the panel would land")
	}
	if got := r.host.ind.target.rect; got != pane {
		t.Errorf("the indicator covers %v, want the pane %v", got, pane)
	}
	if !r.host.Drop(widget.DropEvent{Pos: mid, Mime: PanelMimeType, Payload: r.tree, Action: platform.DragMove}) {
		t.Fatal("the host refused the panel")
	}
	r.Layout()
	if r.tree.Floating() {
		t.Fatal("the panel should be docked again")
	}
	if stackOf(t, r.tree) != stackOf(t, r.prop) {
		t.Error("it did not land in the stack it was dropped on")
	}
	if !r.opener.wins[0].closed {
		t.Error("the window it came from is still open")
	}
	if r.host.ind.Visible() {
		t.Error("the indicator stayed up after the drop")
	}
}

// The host takes its own panels and nothing else.
func TestHostRefusesWhatIsNotItsPanel(t *testing.T) {
	r := newTearRig(t, true)
	mid := r.host.LocalBounds().Min.Add(paintengine2d.Pt(400, 300))
	if r.host.Drop(widget.DropEvent{Pos: mid, Mime: PanelMimeType}) {
		t.Error("a drop with no panel in it is not a panel")
	}
	other := NewHost(nil)
	stray := NewPanel("stray", "Stray", nil)
	other.Dock(stray, SideLeft)
	if r.host.Drop(widget.DropEvent{Pos: mid, Mime: PanelMimeType, Payload: stray}) {
		t.Error("another host's panel belongs to that host")
	}
	r.host.DragLeave()
	if r.host.ind.Visible() {
		t.Error("the indicator goes with the drag")
	}
}

// Without being told where its window is — every Wayland toplevel — the
// float asks for no position at all and lets the compositor place it.
func TestFloatGeometryWithoutAWindowPosition(t *testing.T) {
	r := newTearRig(t, true)
	r.Host.SetPosition(0, 0, false)
	r.Drag(r.gripOf(t, r.tree), r.belowTheHost())
	if len(r.opener.wins) != 1 {
		t.Fatal("no window")
	}
	g := r.opener.wins[0].geom
	if g.Min.X != 0 || g.Min.Y != 0 || g.Dx() <= 0 || g.Dy() <= 0 {
		t.Fatalf("asked for %v, want a size and the desktop's own position", g)
	}
	// The offset still says where in the window the pointer is, which is
	// what the compositor carries it by.
	if off := r.tear(t).Tear.Offset; off.X < 0 || off.X > g.Dx() || off.Y < 0 || off.Y > g.Dy() {
		t.Fatalf("the offset %v is not inside the window %v", off, g)
	}
}

// A panel that goes out of the window and comes back leaves nothing
// behind: the area it lands in holds the one stack and no empty pane.
func TestPanelOutAndBackLeavesNoEmptyPane(t *testing.T) {
	r := newTearRig(t, true)
	r.Drag(r.gripOf(t, r.tree), r.belowTheHost())
	tear := r.tear(t)
	if !r.tree.Floating() {
		t.Fatal("the panel should be floating mid-drag")
	}
	b := r.host.LocalBounds()
	at := paintengine2d.Pt(b.Min.X+4, b.Min.Y+b.Dy()*0.5)
	r.host.DragOver(at)
	if !r.host.Drop(widget.DropEvent{Pos: at, Mime: PanelMimeType, Payload: r.tree, Action: platform.DragMove}) {
		t.Fatal("the host refused the panel")
	}
	tear.Tear.Done(widget.TearMerged, tear.Window)
	r.Layout()
	area := r.host.Area(SideLeft)
	if len(area.kids) != 1 {
		t.Fatalf("the left area holds %d panes, want the one the panel came back into", len(area.kids))
	}
	if got, want := r.host.rectOf(stackOf(t, r.tree)), r.host.rectOf(area); got != want {
		t.Errorf("the stack at %v does not fill its area %v", got, want)
	}
}
