package dock

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/internal/uitest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// box is the host's box in every test, big enough that no minimum bites
// unless a test means it to.
var box = paintengine2d.XYWH(0, 0, 1000, 700)

// rig is a mounted host with a few panels, which most tests start from.
type rig struct {
	*uitest.Session
	host *Host
	tree *Panel
	prop *Panel
	log  *Panel
}

// newRig mounts a host with a central widget and three panels: a tree on
// the left, properties on the right and a log along the bottom.
func newRig(t *testing.T) *rig {
	t.Helper()
	h := NewHost(widgets.NewButton("Centre", nil))
	// The contents take the focus, so the keyboard tests walk the same
	// path a real app's panels do.
	r := &rig{
		host: h,
		tree: NewPanel("tree", "Tree", widgets.NewButton("In the tree", nil)),
		prop: NewPanel("props", "Properties", widgets.NewButton("In the properties", nil)),
		log:  NewPanel("log", "Log", widgets.NewButton("In the log", nil)),
	}
	h.Dock(r.tree, SideLeft)
	h.Dock(r.prop, SideRight)
	h.Dock(r.log, SideBottom)
	r.Session = uitest.Mount(h, box)
	return r
}

// rectOf is a component's box in host coordinates.
func (r *rig) rectOf(c widget.Component) paintengine2d.Rect { return r.host.rectOf(c) }

// stackOf is the stack a panel sits in, failing the test when it has none.
func stackOf(t *testing.T, p *Panel) *Stack {
	t.Helper()
	st := p.Stack()
	if st == nil {
		t.Fatalf("%s is not in a stack", p.Name())
	}
	return st
}

// ---- the tree model -----------------------------------------------------

func TestDockPutsPanelsOnTheirSides(t *testing.T) {
	r := newRig(t)
	centre := r.rectOf(r.host.centre)
	for _, tc := range []struct {
		panel *Panel
		check func(pane paintengine2d.Rect) bool
		where string
	}{
		{r.tree, func(p paintengine2d.Rect) bool { return p.Max.X <= centre.Min.X }, "left of the centre"},
		{r.prop, func(p paintengine2d.Rect) bool { return p.Min.X >= centre.Max.X }, "right of the centre"},
		{r.log, func(p paintengine2d.Rect) bool { return p.Min.Y >= centre.Max.Y }, "below the centre"},
	} {
		pane := r.rectOf(stackOf(t, tc.panel))
		if pane.Empty() {
			t.Fatalf("%s got no room", tc.panel.Name())
		}
		if !tc.check(pane) {
			t.Errorf("%s at %v is not %s (centre %v)", tc.panel.Name(), pane, tc.where, centre)
		}
	}
}

func TestPanesAreExclusive(t *testing.T) {
	r := newRig(t)
	var rects []paintengine2d.Rect
	var names []string
	for _, st := range r.host.stacks() {
		rects = append(rects, r.rectOf(st))
		names = append(names, st.Current().Name())
	}
	rects = append(rects, r.rectOf(r.host.centre))
	names = append(names, "centre")
	for i := range rects {
		for j := i + 1; j < len(rects); j++ {
			if err := uitest.CheckExclusive(rects[i], rects[j]); err != nil {
				t.Errorf("%s and %s overlap: %v", names[i], names[j], err)
			}
		}
	}
}

func TestTreeInvariantsHold(t *testing.T) {
	r := newRig(t)
	for _, err := range uitest.TreeInvariants(r.host) {
		t.Errorf("invariant: %v", err)
	}
}

func TestDockIntoTabsPanels(t *testing.T) {
	r := newRig(t)
	r.host.DockInto(r.prop, r.tree)
	r.Layout()
	st := stackOf(t, r.tree)
	if got := stackOf(t, r.prop); got != st {
		t.Fatalf("props landed in another stack")
	}
	if n := len(st.Open()); n != 2 {
		t.Fatalf("stack holds %d open panels, want 2", n)
	}
	if !st.strip.Visible() {
		t.Error("a stack of two panels shows no tab strip")
	}
	if st.Current() != r.prop {
		t.Errorf("the dropped panel is not current: %v", st.Current())
	}
	if r.host.Area(SideRight).Empty() != true {
		t.Error("the right area kept a hollow branch after its only panel moved")
	}
}

func TestDockBesideSplitsTheStack(t *testing.T) {
	r := newRig(t)
	r.host.DockBeside(r.prop, r.tree, SideBottom)
	r.Layout()
	above, below := r.rectOf(stackOf(t, r.tree)), r.rectOf(stackOf(t, r.prop))
	if below.Min.Y < above.Max.Y {
		t.Errorf("props at %v is not below tree at %v", below, above)
	}
	if err := uitest.CheckExclusive(above, below); err != nil {
		t.Error(err)
	}
	if r.host.sideOf(stackOf(t, r.prop)) != SideLeft {
		t.Error("the split panel did not join the left area")
	}
}

func TestUndockPrunesEmptyBranches(t *testing.T) {
	r := newRig(t)
	r.host.DockBeside(r.prop, r.tree, SideBottom)
	r.Layout()
	before := stackOf(t, r.tree)
	r.host.Undock(r.prop)
	r.Layout()
	if r.prop.Stack() != nil {
		t.Error("an undocked panel is still in a stack")
	}
	// The split of two is down to one pane, so it must be gone and the
	// surviving stack lifted into its place.
	parent, _ := r.host.parentOf(before)
	if parent != r.host.Area(SideLeft) {
		t.Errorf("the left area still holds a split of one: parent %T", parent)
	}
}

func TestClosedPanelKeepsItsPlace(t *testing.T) {
	r := newRig(t)
	st := stackOf(t, r.tree)
	if !r.tree.Close() {
		t.Fatal("the panel refused to close")
	}
	r.Layout()
	if !r.tree.Closed() {
		t.Error("the panel does not say it is closed")
	}
	if r.rectOf(st).Dx() != 0 && !st.Empty() {
		t.Error("an empty stack still takes room")
	}
	r.tree.Show()
	r.Layout()
	if r.tree.Stack() != st {
		t.Error("the panel did not come back to the stack it left")
	}
	if r.rectOf(st).Empty() {
		t.Error("the panel came back with no room")
	}
}

func TestCloseCanBeVetoed(t *testing.T) {
	r := newRig(t)
	r.tree.OnClose = func() bool { return false }
	if r.tree.Close() {
		t.Fatal("Close reported success against a veto")
	}
	if r.tree.Closed() {
		t.Error("a vetoed panel closed anyway")
	}
}

func TestPanelWithoutCloseFeatureHasNoCloseButton(t *testing.T) {
	r := newRig(t)
	r.tree.SetFeatures(DefaultFeatures &^ FeatureClosable)
	for _, p := range stackOf(t, r.tree).head.buttons() {
		if p == headClose {
			t.Fatal("a panel that cannot close still shows a close button")
		}
	}
}

// ---- minimum sizes ------------------------------------------------------

func TestMinimumSizeIsHonoured(t *testing.T) {
	r := newRig(t)
	r.tree.SetMinSize(260, 100)
	r.Layout()
	got := r.rectOf(stackOf(t, r.tree))
	if got.Dx() < 260 {
		t.Errorf("the left panel is %.0f wide, under its 260 minimum", got.Dx())
	}
}

func TestSashStopsAtTheMinimum(t *testing.T) {
	r := newRig(t)
	r.tree.SetMinSize(240, 100)
	r.Layout()
	mid := r.host.middle
	// Drag the sash between the left area and the centre far to the left.
	mid.moveSash(0, 10)
	r.Layout()
	got := r.rectOf(stackOf(t, r.tree))
	if got.Dx() < 240 {
		t.Errorf("the sash squeezed the panel to %.0f, under its 240 minimum", got.Dx())
	}
}

func TestSashResizesBothNeighbours(t *testing.T) {
	r := newRig(t)
	mid := r.host.middle
	before := r.rectOf(stackOf(t, r.tree)).Dx()
	centreBefore := r.rectOf(r.host.centre).Dx()
	mid.moveSash(0, before+80)
	r.Layout()
	after := r.rectOf(stackOf(t, r.tree)).Dx()
	centreAfter := r.rectOf(r.host.centre).Dx()
	if after <= before {
		t.Errorf("the panel did not grow: %.0f then %.0f", before, after)
	}
	if centreAfter >= centreBefore {
		t.Errorf("the centre did not give the room up: %.0f then %.0f", centreBefore, centreAfter)
	}
	if d := (after - before) - (centreBefore - centreAfter); d > 1 || d < -1 {
		t.Errorf("the pair did not keep its total: gained %.1f, lost %.1f", after-before, centreBefore-centreAfter)
	}
}

func TestSashDragMovesThePane(t *testing.T) {
	r := newRig(t)
	pane := r.rectOf(stackOf(t, r.tree))
	sash := paintengine2d.Pt(pane.Max.X+1, pane.Min.Y+pane.Dy()*0.5)
	r.Drag(sash, paintengine2d.Pt(sash.X+120, sash.Y))
	r.Layout()
	if got := r.rectOf(stackOf(t, r.tree)).Dx(); got <= pane.Dx()+40 {
		t.Errorf("dragging the sash 120px right only took the panel from %.0f to %.0f", pane.Dx(), got)
	}
}

func TestSashShowsAResizeCursor(t *testing.T) {
	r := newRig(t)
	pane := r.rectOf(stackOf(t, r.tree))
	r.MouseMove(paintengine2d.Pt(pane.Max.X+1, pane.Min.Y+pane.Dy()*0.5))
	if got := r.Cursor(); got != platform.CursorColResize {
		t.Errorf("the sash shows cursor %v, want col-resize", got)
	}
}

func TestCollapsedPanelKeepsOnlyItsChrome(t *testing.T) {
	r := newRig(t)
	st := stackOf(t, r.log)
	full := r.rectOf(st).Dy()
	r.log.SetCollapsed(true)
	r.Layout()
	got := r.rectOf(st).Dy()
	if got >= full {
		t.Errorf("a collapsed panel still takes %.0f of %.0f", got, full)
	}
	if d := got - st.chromeH(); d > 1 || d < -1 {
		t.Errorf("a collapsed panel is %.0f high, want its chrome %.0f", got, st.chromeH())
	}
	if r.rectOf(r.log).Dy() > 0 {
		t.Error("a collapsed panel still gives its content room")
	}
	r.log.SetCollapsed(false)
	r.Layout()
	if got := r.rectOf(st).Dy(); got < full-1 {
		t.Errorf("expanding gave the panel %.0f back, want about %.0f", got, full)
	}
}

// ---- drop targets -------------------------------------------------------

func TestDropTargetsReadThePointer(t *testing.T) {
	r := newRig(t)
	b := r.host.LocalBounds()
	pane := r.rectOf(stackOf(t, r.tree))
	mid := paintengine2d.Pt(pane.Min.X+pane.Dx()*0.5, pane.Min.Y+pane.Dy()*0.5)
	for _, tc := range []struct {
		name string
		at   paintengine2d.Point
		want dropKind
		side Side
	}{
		{"the top band makes a top area", paintengine2d.Pt(b.Dx()*0.5, 2), dropArea, SideTop},
		{"the bottom band makes a bottom area", paintengine2d.Pt(b.Dx()*0.5, b.Max.Y-2), dropArea, SideBottom},
		{"the left band makes a left area", paintengine2d.Pt(2, b.Dy()*0.5), dropArea, SideLeft},
		{"the right band makes a right area", paintengine2d.Pt(b.Max.X-2, b.Dy()*0.5), dropArea, SideRight},
		{"the middle of a pane tabs", mid, dropTab, SideLeft},
		{"a pane's foot splits it", paintengine2d.Pt(mid.X, pane.Max.Y-4), dropSplit, SideBottom},
		{"off the host floats", paintengine2d.Pt(-20, -20), dropFloat, SideLeft},
	} {
		got := r.host.targetAt(tc.at)
		if got.kind != tc.want {
			t.Errorf("%s: got kind %d at %v, want %d", tc.name, got.kind, tc.at, tc.want)
			continue
		}
		if (tc.want == dropArea || tc.want == dropSplit) && got.side != tc.side {
			t.Errorf("%s: got side %v, want %v", tc.name, got.side, tc.side)
		}
		if tc.want != dropFloat && tc.want != dropNone && got.rect.Empty() {
			t.Errorf("%s: the indicator would have nothing to draw", tc.name)
		}
	}
}

func TestTabDropIndicatorCoversTheStack(t *testing.T) {
	r := newRig(t)
	pane := r.rectOf(stackOf(t, r.tree))
	got := r.host.targetAt(paintengine2d.Pt(pane.Min.X+pane.Dx()*0.5, pane.Min.Y+pane.Dy()*0.5))
	if got.rect != pane {
		t.Errorf("a tab drop marks %v, want the whole stack %v", got.rect, pane)
	}
}

// ---- dragging -----------------------------------------------------------

// gripOf is a point on a stack's title bar, clear of its buttons.
func (r *rig) gripOf(t *testing.T, p *Panel) paintengine2d.Point {
	t.Helper()
	head := stackOf(t, p).head
	hr := r.rectOf(head)
	return paintengine2d.Pt(hr.Min.X+6, hr.Min.Y+hr.Dy()*0.5)
}

func TestDraggingATitleBarDocksToAnotherSide(t *testing.T) {
	r := newRig(t)
	b := r.host.LocalBounds()
	r.Drag(r.gripOf(t, r.tree), paintengine2d.Pt(b.Dx()*0.5, 3))
	r.Layout()
	if r.host.Area(SideTop).Empty() {
		t.Fatal("the panel did not land in the top area")
	}
	if !r.host.Area(SideLeft).Empty() {
		t.Error("the panel is still in the left area too")
	}
	pane := r.rectOf(stackOf(t, r.tree))
	if pane.Max.Y > r.rectOf(r.host.centre).Min.Y+1 {
		t.Errorf("the panel at %v is not above the centre", pane)
	}
}

func TestDraggingOntoAPanelTabsThem(t *testing.T) {
	r := newRig(t)
	target := r.rectOf(stackOf(t, r.prop))
	r.Drag(r.gripOf(t, r.tree), paintengine2d.Pt(target.Min.X+target.Dx()*0.5, target.Min.Y+target.Dy()*0.5))
	r.Layout()
	st := stackOf(t, r.prop)
	if stackOf(t, r.tree) != st {
		t.Fatal("the dragged panel did not join the target's stack")
	}
	if n := len(st.Open()); n != 2 {
		t.Fatalf("the stack holds %d panels, want 2", n)
	}
}

func TestDraggingOntoAPanelsEdgeSplitsIt(t *testing.T) {
	r := newRig(t)
	target := r.rectOf(stackOf(t, r.prop))
	r.Drag(r.gripOf(t, r.tree), paintengine2d.Pt(target.Min.X+target.Dx()*0.5, target.Max.Y-4))
	r.Layout()
	if stackOf(t, r.tree) == stackOf(t, r.prop) {
		t.Fatal("the panels were tabbed rather than split")
	}
	above, below := r.rectOf(stackOf(t, r.prop)), r.rectOf(stackOf(t, r.tree))
	if below.Min.Y < above.Max.Y {
		t.Errorf("the dragged panel at %v is not below the target at %v", below, above)
	}
}

func TestAShortDragIsAClickAndMovesNothing(t *testing.T) {
	r := newRig(t)
	before := r.rectOf(stackOf(t, r.tree))
	grip := r.gripOf(t, r.tree)
	r.Drag(grip, paintengine2d.Pt(grip.X+2, grip.Y+2))
	r.Layout()
	if got := r.rectOf(stackOf(t, r.tree)); got != before {
		t.Errorf("a 2px drag moved the panel from %v to %v", before, got)
	}
}

func TestTheIndicatorShowsWhileDraggingAndGoesOnRelease(t *testing.T) {
	r := newRig(t)
	b := r.host.LocalBounds()
	grip := r.gripOf(t, r.tree)
	r.MousePress(grip, platform.ButtonLeft)
	if r.host.ind.Visible() {
		t.Error("the indicator showed on a bare press")
	}
	r.Session.MouseMove(paintengine2d.Pt(b.Dx()*0.5, 3))
	if !r.host.ind.Visible() {
		t.Error("the indicator did not show once the drag was under way")
	}
	r.MouseRelease(paintengine2d.Pt(b.Dx()*0.5, 3))
	if r.host.ind.Visible() {
		t.Error("the indicator stayed up after the drop")
	}
}

func TestEscapeGivesUpTheDrag(t *testing.T) {
	r := newRig(t)
	b := r.host.LocalBounds()
	before := r.rectOf(stackOf(t, r.tree))
	grip := r.gripOf(t, r.tree)
	r.MousePress(grip, platform.ButtonLeft)
	r.Session.MouseMove(paintengine2d.Pt(b.Dx()*0.5, 3))
	if !r.host.KeyPress(widget.KeyEvent{Key: platform.KeyEscape}) {
		t.Fatal("Escape was not taken while dragging")
	}
	r.MouseRelease(paintengine2d.Pt(b.Dx()*0.5, 3))
	r.Layout()
	if got := r.rectOf(stackOf(t, r.tree)); got != before {
		t.Errorf("Escape still let the panel move, from %v to %v", before, got)
	}
}

// ---- the title bar and tabs --------------------------------------------

func TestTitleBarCloseButtonClosesThePanel(t *testing.T) {
	r := newRig(t)
	head := stackOf(t, r.tree).head
	i := -1
	for j, p := range head.buttons() {
		if p == headClose {
			i = j
		}
	}
	if i < 0 {
		t.Fatal("no close button")
	}
	at := head.buttonRect(i)
	hostAt := r.rectOf(head)
	click := paintengine2d.Pt(hostAt.Min.X+at.Min.X+at.Dx()*0.5, hostAt.Min.Y+at.Dy()*0.5)
	r.MousePress(click, platform.ButtonLeft)
	r.MouseRelease(click)
	if !r.tree.Closed() {
		t.Error("the close button did not close the panel")
	}
}

func TestTabStripSelectsPanels(t *testing.T) {
	r := newRig(t)
	r.host.DockInto(r.prop, r.tree)
	r.Layout()
	st := stackOf(t, r.tree)
	strip := st.strip
	sr := r.rectOf(strip)
	rects := strip.rects()
	if len(rects) != 2 {
		t.Fatalf("the strip has %d tabs, want 2", len(rects))
	}
	click := paintengine2d.Pt(sr.Min.X+rects[0].Min.X+rects[0].Dx()*0.5, sr.Min.Y+sr.Dy()*0.5)
	r.MousePress(click, platform.ButtonLeft)
	r.MouseRelease(click)
	if st.Current() != r.tree {
		t.Errorf("clicking the first tab shows %v, want the tree", st.Current())
	}
}

// ---- floating -----------------------------------------------------------

// fakeOpener stands in for a desktop: it hands out windows that only
// record what was asked of them.
type fakeOpener struct{ wins []*fakeWindow }

func (f *fakeOpener) OpenFloat(title string, geom paintengine2d.Rect) (FloatWindow, error) {
	w := &fakeWindow{title: title, geom: geom}
	f.wins = append(f.wins, w)
	return w, nil
}

type fakeWindow struct {
	title   string
	content widget.Component
	geom    paintengine2d.Rect
	shown   bool
	closed  bool
	onClose func() bool
}

func (w *fakeWindow) SetTitle(t string)                { w.title = t }
func (w *fakeWindow) SetContent(c widget.Component)    { w.content = c }
func (w *fakeWindow) SetOnCloseRequest(fn func() bool) { w.onClose = fn }
func (w *fakeWindow) Geometry() paintengine2d.Rect     { return w.geom }
func (w *fakeWindow) Show()                            { w.shown = true }
func (w *fakeWindow) Hide()                            { w.shown = false }
func (w *fakeWindow) Raise()                           {}
func (w *fakeWindow) Close()                           { w.closed = true }

func TestPanelFloatsIntoItsOwnWindow(t *testing.T) {
	r := newRig(t)
	o := &fakeOpener{}
	r.host.SetWindowOpener(o)
	if !r.host.FloatPanel(r.tree, paintengine2d.XYWH(40, 60, 300, 420)) {
		t.Fatal("the panel refused to float")
	}
	r.Layout()
	if len(o.wins) != 1 {
		t.Fatalf("opened %d windows, want 1", len(o.wins))
	}
	w := o.wins[0]
	if w.title != "Tree" {
		t.Errorf("the window is called %q, want the panel's title", w.title)
	}
	if !w.shown {
		t.Error("the window was never shown")
	}
	if !r.tree.Floating() {
		t.Error("the panel does not say it floats")
	}
	if r.tree.Stack() == nil {
		t.Error("a floating panel lost the chrome it needs to be docked back")
	}
	if !r.host.Area(SideLeft).Empty() {
		t.Error("the left area kept the panel it gave up")
	}
	// The float button on a floating panel offers to dock it back.
	if got := headPartName(headFloat, stackOf(t, r.tree)); got != "Dock" {
		t.Errorf("a floating panel's float button is called %q, want Dock", got)
	}
}

func TestFloatingPanelDocksBackWhereItWas(t *testing.T) {
	r := newRig(t)
	r.host.SetWindowOpener(&fakeOpener{})
	home := stackOf(t, r.tree)
	r.host.DockInto(r.prop, r.tree) // so the home stack survives the float
	r.Layout()
	r.host.FloatPanel(r.tree, paintengine2d.XYWH(0, 0, 300, 400))
	r.Layout()
	if !r.host.DockPanel(r.tree) {
		t.Fatal("the panel refused to dock back")
	}
	r.Layout()
	if r.tree.Floating() {
		t.Error("the panel still says it floats")
	}
	if got := stackOf(t, r.tree); got != home {
		t.Error("the panel did not come back to the stack it left")
	}
}

func TestFloatingPanelDocksBackToItsSideWhenTheStackIsGone(t *testing.T) {
	r := newRig(t)
	r.host.SetWindowOpener(&fakeOpener{})
	r.host.FloatPanel(r.tree, paintengine2d.XYWH(0, 0, 300, 400))
	r.Layout()
	r.host.DockPanel(r.tree)
	r.Layout()
	if r.host.sideOf(stackOf(t, r.tree)) != SideLeft {
		t.Error("the panel did not come back to the left area")
	}
}

func TestClosingTheFloatingWindowHidesThePanel(t *testing.T) {
	r := newRig(t)
	o := &fakeOpener{}
	r.host.SetWindowOpener(o)
	r.host.FloatPanel(r.tree, paintengine2d.XYWH(0, 0, 300, 400))
	w := o.wins[0]
	if w.onClose == nil {
		t.Fatal("the window was given no close hook")
	}
	if w.onClose() {
		t.Error("the close hook let the window die; it should hide with the panel")
	}
	if !r.tree.Closed() {
		t.Error("closing the window did not close the panel")
	}
	if w.shown {
		t.Error("the window is still shown")
	}
}

func TestDraggingAFloatingPanelBackDocksIt(t *testing.T) {
	r := newRig(t)
	r.host.SetWindowOpener(&fakeOpener{})
	r.host.FloatPanel(r.tree, paintengine2d.XYWH(0, 0, 300, 400))
	r.Layout()
	// The panel's own chrome drives the drop, wherever its window is.
	r.host.applyDrop(r.tree, dropTarget{kind: dropArea, side: SideTop})
	r.Layout()
	if r.tree.Floating() {
		t.Error("the panel still floats after being dropped on the host")
	}
	if r.host.Area(SideTop).Empty() {
		t.Error("the panel did not land in the top area")
	}
}

func TestWithoutAnOpenerNothingFloats(t *testing.T) {
	r := newRig(t)
	if r.host.FloatPanel(r.tree, paintengine2d.Rect{}) {
		t.Error("a host with no window opener floated a panel")
	}
	for _, p := range stackOf(t, r.tree).head.buttons() {
		if p == headFloat {
			t.Error("a host that cannot float shows a float button")
		}
	}
}

// ---- layouts ------------------------------------------------------------

// shape is a stable description of an arrangement, for comparing one with
// another.
func shape(h *Host) string {
	var b strings.Builder
	for s := range h.areas {
		b.WriteString(Side(s).String())
		b.WriteByte('[')
		writeNode(&b, h.areas[s])
		b.WriteString("] ")
	}
	for _, p := range h.panels {
		if p.Floating() {
			b.WriteString("float:" + p.Name() + " ")
		}
		if p.Closed() {
			b.WriteString("closed:" + p.Name() + " ")
		}
		if p.Collapsed() {
			b.WriteString("collapsed:" + p.Name() + " ")
		}
	}
	return strings.TrimSpace(b.String())
}

func writeNode(b *strings.Builder, n Node) {
	switch v := n.(type) {
	case *Stack:
		for i, p := range v.panels {
			if i > 0 {
				b.WriteByte('|')
			}
			b.WriteString(p.Name())
			if i == v.current {
				b.WriteByte('*')
			}
		}
	case *Split:
		sep := "-"
		if v.vertical {
			sep = "/"
		}
		for i, k := range v.kids {
			if i > 0 {
				b.WriteString(sep)
			}
			b.WriteByte('(')
			writeNode(b, k)
			b.WriteByte(')')
		}
	}
}

func TestLayoutRoundTrips(t *testing.T) {
	r := newRig(t)
	r.host.DockInto(r.prop, r.tree)
	r.host.DockBeside(r.log, r.tree, SideBottom)
	r.log.SetCollapsed(true)
	r.Layout()
	want := shape(r.host)
	data, err := r.host.LayoutJSON()
	if err != nil {
		t.Fatal(err)
	}
	// Pull the arrangement apart, then read it back.
	r.host.Dock(r.tree, SideRight)
	r.host.Dock(r.prop, SideTop)
	r.host.Dock(r.log, SideBottom)
	r.log.SetCollapsed(false)
	r.Layout()
	if shape(r.host) == want {
		t.Fatal("the test did not actually change the arrangement")
	}
	if err := r.host.ApplyLayoutJSON(data); err != nil {
		t.Fatal(err)
	}
	r.Layout()
	if got := shape(r.host); got != want {
		t.Errorf("the layout came back as\n  %s\nwant\n  %s", got, want)
	}
}

func TestLayoutKeepsSashPositions(t *testing.T) {
	r := newRig(t)
	r.host.middle.moveSash(0, 320)
	r.Layout()
	want := r.rectOf(stackOf(t, r.tree)).Dx()
	data, err := r.host.LayoutJSON()
	if err != nil {
		t.Fatal(err)
	}
	r.host.middle.moveSash(0, 90)
	r.Layout()
	if err := r.host.ApplyLayoutJSON(data); err != nil {
		t.Fatal(err)
	}
	r.Layout()
	if got := r.rectOf(stackOf(t, r.tree)).Dx(); got < want-2 || got > want+2 {
		t.Errorf("the left area came back %.0f wide, want %.0f", got, want)
	}
}

func TestLayoutRoundTripsFloatingGeometry(t *testing.T) {
	r := newRig(t)
	o := &fakeOpener{}
	r.host.SetWindowOpener(o)
	want := paintengine2d.XYWH(120, 90, 340, 480)
	r.host.FloatPanel(r.tree, want)
	r.Layout()
	data, err := r.host.LayoutJSON()
	if err != nil {
		t.Fatal(err)
	}
	r.host.DockPanel(r.tree)
	r.Layout()
	if err := r.host.ApplyLayoutJSON(data); err != nil {
		t.Fatal(err)
	}
	r.Layout()
	if !r.tree.Floating() {
		t.Fatal("the panel did not float again")
	}
	if got := r.tree.FloatGeometry(); got != want {
		t.Errorf("the window came back at %v, want %v", got, want)
	}
}

func TestLayoutRoundTripsClosedPanels(t *testing.T) {
	r := newRig(t)
	r.tree.Close()
	r.Layout()
	data, _ := r.host.LayoutJSON()
	r.tree.Show()
	r.Layout()
	if err := r.host.ApplyLayoutJSON(data); err != nil {
		t.Fatal(err)
	}
	if !r.tree.Closed() {
		t.Error("the closed panel came back open")
	}
}

func TestLayoutFromAnotherVersionIsRefused(t *testing.T) {
	r := newRig(t)
	err := r.host.ApplyLayoutJSON([]byte(`{"version":99}`))
	if err == nil {
		t.Fatal("a layout from another version was accepted")
	}
	if err != ErrLayoutVersion {
		t.Errorf("got %v, want ErrLayoutVersion", err)
	}
}

func TestLayoutKeepsPanelsItNeverHeardOf(t *testing.T) {
	r := newRig(t)
	data, _ := r.host.LayoutJSON()
	extra := NewPanel("extra", "Extra", widgets.NewLabel("x"))
	r.host.Dock(extra, SideRight)
	r.Layout()
	if err := r.host.ApplyLayoutJSON(data); err != nil {
		t.Fatal(err)
	}
	r.Layout()
	if extra.Stack() == nil {
		t.Error("a panel the saved layout never mentioned was dropped")
	}
	if r.rectOf(stackOf(t, extra)).Empty() {
		t.Error("the new panel came back with no room")
	}
}

func TestLayoutSkipsPanelsThatAreGone(t *testing.T) {
	r := newRig(t)
	data := []byte(`{"version":1,"areas":{"left":{"panels":["tree","ghost"]}}}`)
	if err := r.host.ApplyLayoutJSON(data); err != nil {
		t.Fatal(err)
	}
	r.Layout()
	st := stackOf(t, r.tree)
	if len(st.panels) != 1 {
		t.Errorf("the stack holds %d panels, want only the one that exists", len(st.panels))
	}
}

func TestResetGoesBackToTheDefault(t *testing.T) {
	r := newRig(t)
	r.host.SetDefaultLayout()
	want := shape(r.host)
	r.host.DockInto(r.prop, r.tree)
	r.host.Dock(r.log, SideTop)
	r.Layout()
	if shape(r.host) == want {
		t.Fatal("the test did not change anything")
	}
	if !r.host.ResetLayout() {
		t.Fatal("reset found no default")
	}
	r.Layout()
	if got := shape(r.host); got != want {
		t.Errorf("reset gave\n  %s\nwant\n  %s", got, want)
	}
}

func TestResetWithoutADefaultReportsSo(t *testing.T) {
	r := newRig(t)
	if r.host.ResetLayout() {
		t.Error("reset claimed to work with no default recorded")
	}
}

func TestLayoutChangesAreAnnounced(t *testing.T) {
	r := newRig(t)
	n := 0
	r.host.OnLayoutChanged = func() { n++ }
	r.host.Dock(r.tree, SideTop)
	if n == 0 {
		t.Error("docking a panel did not announce a layout change")
	}
	was := n
	r.host.middle.moveSash(0, 300)
	if n == was {
		t.Error("moving a sash did not announce a layout change")
	}
}

// ---- keyboard -----------------------------------------------------------

func TestF6MovesBetweenPanels(t *testing.T) {
	r := newRig(t)
	widget.FocusFirstIn(r.tree)
	first := r.host.FocusedPanel()
	if first == nil {
		t.Fatal("nothing took the focus to begin with")
	}
	if !r.host.cyclePanel(true) {
		t.Fatal("F6 moved nothing")
	}
	second := r.host.FocusedPanel()
	if second == nil {
		t.Fatal("F6 left the focus outside every panel")
	}
	if second == first {
		t.Error("F6 stayed in the same panel")
	}
	// Round the houses and back.
	r.host.cyclePanel(true)
	r.host.cyclePanel(true)
	if got := r.host.FocusedPanel(); got != first {
		t.Errorf("F6 three times round three panels landed on %v, want %v", got, first)
	}
}

func TestShiftF6GoesBack(t *testing.T) {
	r := newRig(t)
	widget.FocusFirstIn(r.tree)
	first := r.host.FocusedPanel()
	r.host.cyclePanel(true)
	r.host.cyclePanel(false)
	if got := r.host.FocusedPanel(); got != first {
		t.Errorf("forward then back landed on %v, want %v", got, first)
	}
}

func TestCtrlWClosesTheFocusedPanel(t *testing.T) {
	r := newRig(t)
	widget.FocusFirstIn(r.tree)
	if !r.host.KeyPress(widget.KeyEvent{Key: platform.KeyW, Mods: platform.ModCtrl}) {
		t.Fatal("Ctrl+W was not taken")
	}
	if !r.tree.Closed() {
		t.Error("Ctrl+W did not close the focused panel")
	}
}

func TestCtrlWLeavesAPanelThatCannotClose(t *testing.T) {
	r := newRig(t)
	r.tree.SetFeatures(DefaultFeatures &^ FeatureClosable)
	widget.FocusFirstIn(r.tree)
	r.host.KeyPress(widget.KeyEvent{Key: platform.KeyW, Mods: platform.ModCtrl})
	if r.tree.Closed() {
		t.Error("Ctrl+W closed a panel that says it cannot close")
	}
}

func TestEveryTitleBarControlIsReachableByKeyboard(t *testing.T) {
	r := newRig(t)
	head := stackOf(t, r.tree).head
	if !head.WantsFocus() {
		t.Fatal("a panel's title bar is not in the tab order")
	}
	r.Focus(head)
	head.FocusGained()
	btns := head.buttons()
	seen := map[headPart]bool{}
	for range btns {
		if head.focus < 0 || head.focus >= len(btns) {
			t.Fatalf("the keyboard fell off the title bar at %d", head.focus)
		}
		seen[btns[head.focus]] = true
		head.KeyPress(widget.KeyEvent{Key: platform.KeyLeft})
	}
	for _, p := range btns {
		if !seen[p] {
			t.Errorf("button %d cannot be reached with the arrow keys", p)
		}
	}
}

func TestSpaceActivatesTheFocusedTitleBarButton(t *testing.T) {
	r := newRig(t)
	head := stackOf(t, r.tree).head
	r.Focus(head)
	head.FocusGained()
	for i, p := range head.buttons() {
		if p == headClose {
			head.focus = i
		}
	}
	head.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	if !r.tree.Closed() {
		t.Error("Space on the close button did not close the panel")
	}
}

func TestArrowKeysWalkTheTabs(t *testing.T) {
	r := newRig(t)
	r.host.DockInto(r.prop, r.tree)
	r.Layout()
	st := stackOf(t, r.tree)
	r.Focus(st.strip)
	st.strip.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	if st.Current() != r.tree {
		t.Errorf("Home showed %v, want the first tab", st.Current())
	}
	st.strip.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	if st.Current() != r.prop {
		t.Errorf("Right showed %v, want the second tab", st.Current())
	}
}

// ---- accessibility ------------------------------------------------------

// tree builds the accessible tree under the host.
func accTree(h *Host) *a11y.Node {
	root := &a11y.Node{Role: a11y.RoleWindow, Name: "Test"}
	widget.AccessibleTree(root, h)
	return root
}

// find is the first node with the given role and name.
func find(n *a11y.Node, role a11y.Role, name string) *a11y.Node {
	var hit *a11y.Node
	n.Walk(func(x *a11y.Node) bool {
		if hit == nil && x.Role == role && (name == "" || x.Name == name) {
			hit = x
		}
		return hit == nil
	})
	return hit
}

func TestAccessibleTreeNamesEveryPanel(t *testing.T) {
	r := newRig(t)
	root := accTree(r.host)
	for _, p := range []*Panel{r.tree, r.prop, r.log} {
		if find(root, a11y.RoleGroup, p.Title()) == nil {
			t.Errorf("no accessible group called %q", p.Title())
		}
	}
}

func TestAccessibleTreeHasSplittersWithNames(t *testing.T) {
	r := newRig(t)
	root := accTree(r.host)
	n := 0
	root.Walk(func(x *a11y.Node) bool {
		if x.Role != a11y.RoleSplitter {
			return true
		}
		n++
		if x.Name == "" {
			t.Error("a splitter has no name")
		}
		if !x.HasRange {
			t.Errorf("splitter %q publishes no position", x.Name)
		}
		return true
	})
	if n == 0 {
		t.Error("the tree has no splitters at all")
	}
}

func TestAccessibleTabsAppearWhenPanelsAreStacked(t *testing.T) {
	r := newRig(t)
	if find(accTree(r.host), a11y.RoleTabList, "") != nil {
		t.Error("a stack of one panel already claims a tab list")
	}
	r.host.DockInto(r.prop, r.tree)
	r.Layout()
	root := accTree(r.host)
	list := find(root, a11y.RoleTabList, "")
	if list == nil {
		t.Fatal("stacked panels published no tab list")
	}
	if len(list.Children) != 2 {
		t.Fatalf("the tab list has %d tabs, want 2", len(list.Children))
	}
	sel := 0
	for _, c := range list.Children {
		if c.Role != a11y.RoleTab {
			t.Errorf("a tab list child has role %v", c.Role)
		}
		if c.Name == "" {
			t.Error("a tab has no name")
		}
		if c.State.Has(a11y.StateSelected) {
			sel++
		}
	}
	if sel != 1 {
		t.Errorf("%d tabs say they are selected, want 1", sel)
	}
}

func TestAccessibleTitleBarButtonsAreActionable(t *testing.T) {
	r := newRig(t)
	head := stackOf(t, r.tree).head
	items := head.AccessibleItems()
	if len(items) == 0 {
		t.Fatal("the title bar published no buttons")
	}
	for _, n := range items {
		if n.Role != a11y.RoleButton {
			t.Errorf("title bar item %q has role %v, want button", n.Name, n.Role)
		}
		if n.Name == "" {
			t.Error("a title bar button has no name")
		}
		if !n.Actions.Has(a11y.ActionDefault) {
			t.Errorf("title bar button %q cannot be pressed", n.Name)
		}
	}
	// Pressing the close button through the accessibility tree works.
	for i, p := range head.buttons() {
		if p == headClose {
			if !head.AccessibleAction(i, a11y.ActionDefault) {
				t.Fatal("the close button refused an accessible press")
			}
		}
	}
	if !r.tree.Closed() {
		t.Error("an accessible press on close did not close the panel")
	}
}

func TestAccessibleTreePassesTheChecker(t *testing.T) {
	r := newRig(t)
	r.host.DockInto(r.prop, r.tree)
	r.Layout()
	for _, err := range a11y.Check(accTree(r.host)) {
		t.Errorf("a11y.Check: %v", err)
	}
}

func TestCollapsedPanelSaysSoToAssistiveTechnology(t *testing.T) {
	r := newRig(t)
	r.tree.SetCollapsed(true)
	r.Layout()
	n := find(accTree(r.host), a11y.RoleGroup, "Tree")
	if n == nil {
		t.Fatal("no node for the tree panel")
	}
	if n.State.Has(a11y.StateExpanded) {
		t.Error("a collapsed panel still says it is expanded")
	}
	if !n.State.Has(a11y.StateExpandable) {
		t.Error("a collapsible panel does not say it can expand")
	}
}

// ---- painting -----------------------------------------------------------

func TestThePanelsPaintWithoutTrouble(t *testing.T) {
	r := newRig(t)
	r.host.DockInto(r.prop, r.tree)
	r.Layout()
	if img := r.Paint(); img == nil {
		t.Fatal("painting the host produced nothing")
	}
	// The retained-scene path records the same tree.
	if sc := r.Record(); sc == nil {
		t.Fatal("recording the host produced nothing")
	}
}

func TestTheDropIndicatorPaints(t *testing.T) {
	r := newRig(t)
	pane := r.rectOf(stackOf(t, r.tree))
	before := r.Paint()
	r.host.showIndicator(r.host.targetAt(paintengine2d.Pt(pane.Min.X+pane.Dx()*0.5, pane.Min.Y+pane.Dy()*0.5)))
	r.Layout()
	after := r.Paint()
	if uitest.ColorDiff(before, after, 8) == 0 {
		t.Error("showing the drop indicator changed no pixels")
	}
}
