package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// solidCover fills everything it is given and counts what reaches it.
type solidCover struct {
	widget.Base
	col      paintengine2d.Color
	presses  int
	keys     int
	lastBox  paintengine2d.Rect
	hasFocus bool
}

func newSolidCover() *solidCover {
	c := &solidCover{col: paintengine2d.RGB(1, 0, 1)}
	c.Init(c)
	c.SetWantsFocus(true)
	return c
}

func (c *solidCover) Arrange(r paintengine2d.Rect) {
	c.SetBounds(r)
	c.lastBox = r
}

func (c *solidCover) Paint(ctx *paintengine2d.Context) {
	ctx.DrawRect(c.LocalBounds(), paintengine2d.Fill(c.col))
}

func (c *solidCover) MousePress(widget.MouseEvent) bool { c.presses++; return true }

func (c *solidCover) KeyPress(widget.KeyEvent) bool { c.keys++; return true }

// **A cover lies over the whole window inside its frame, title bar and
// all.** An overlay is laid over the content alone, a popup is a surface
// of its own, and every component is clipped to its own box: there was no
// way to draw over a window as one picture, so a card that turns the
// window over could not turn its title bar with it.
func TestACoverLiesOverTheTitleBarAndTheContent(t *testing.T) {
	// Both painters: the window records a display list for the scene
	// (platform.WantScene) or paints straight into the surface, and a
	// layer has to be in both or it is missing on half the machines.
	for _, scene := range []string{"1", "0"} {
		t.Run("scene="+scene, func(t *testing.T) {
			t.Setenv(platform.EnvScene, scene)
			coverPaints(t)
		})
	}
}

func coverPaints(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	if !r.w.geom.framed {
		t.Fatal("the rig is not drawing its own frame")
	}
	before := r.w.surf.Buffer().Clone()

	cover := newSolidCover()
	r.w.SetCover(cover)
	r.a.PumpOnce()
	r.w.frame()

	// The cover was given the window inside its frame: the caption's box
	// and the content's, as one.
	g := r.w.geom
	if got := cover.lastBox; got != g.inner {
		t.Errorf("the cover was arranged at %v, want the window's inside %v", got, g.inner)
	}
	if !holdsRect(cover.lastBox, g.caption) || !holdsRect(cover.lastBox, g.content) {
		t.Errorf("the cover's box %v does not hold the caption %v and the content %v",
			cover.lastBox, g.caption, g.content)
	}
	// And it is painted over both: the title bar's pixels changed too,
	// which is the whole point.
	after := r.w.surf.Buffer().Clone()
	mx := int(g.caption.Min.X + g.caption.Dx()*0.5)
	my := int(g.caption.Min.Y + g.caption.Dy()*0.5)
	if after.NRGBAAt(mx, my) == before.NRGBAAt(mx, my) {
		t.Error("the middle of the title bar is unchanged under the cover")
	}
	if n := diffPixels(t, before, after); n < 1000 {
		t.Errorf("only %d pixels changed: the cover is not covering the window", n)
	}
	// The frame around it is still the window's, so it can be resized.
	if region, edges := r.w.NonClientHit(r.edge(platform.EdgeRight, 200)); region != RegionResize || edges == 0 {
		t.Errorf("the right edge under a cover is %v/%v, want a resize edge", region, edges)
	}

	// Taking it down puts the window back as it was.
	r.w.SetCover(nil)
	r.a.PumpOnce()
	r.w.frame()
	if n := diffPixels(t, before, r.w.surf.Buffer().Clone()); n != 0 {
		t.Errorf("%d pixels differ after the cover came off", n)
	}
}

// holdsRect reports whether outer holds inner.
func holdsRect(outer, inner paintengine2d.Rect) bool {
	return inner.Min.X >= outer.Min.X && inner.Min.Y >= outer.Min.Y &&
		inner.Max.X <= outer.Max.X && inner.Max.Y <= outer.Max.Y
}

// What the cover covers, it owns: a press inside it reaches the cover and
// nothing under it — not the content, and not the title bar's own
// controls, which cannot be seen either.
func TestACoverOwnsWhatItCovers(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	cover := newSolidCover()
	r.w.SetCover(cover)
	r.a.PumpOnce()

	clicks := r.clicks
	tool := widget.DeviceOrigin(r.tools)
	r.click(tool.X+10, tool.Y+10)
	if r.clicks != clicks {
		t.Error("a tool bar item under the cover was clicked")
	}
	if cover.presses == 0 {
		t.Error("the press did not reach the cover")
	}

	// And the keyboard is the cover's.
	r.w.RequestFocus(cover)
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDelete})
	if cover.keys == 0 {
		t.Error("the key did not reach the cover")
	}
	// A key with the keyboard on nothing goes to the cover, not to the
	// content root behind it.
	cover.keys = 0
	r.w.RequestFocus(nil)
	r.w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDelete})
	if cover.keys != 1 {
		t.Errorf("the cover heard %d keys with the keyboard on nothing, want 1", cover.keys)
	}
}

// Popups stay above it, so a cover holding a widget with a menu works.
func TestAPopupIsStillAboveACover(t *testing.T) {
	r := newFrameRig(t, platform.DecorationsClient)
	cover := newSolidCover()
	r.w.SetCover(cover)
	r.a.PumpOnce()

	pop := widgets.ShowContextMenu(cover, paintengine2d.Pt(40, 40), &widgets.MenuItem{Text: "Turn back"})
	if pop == nil {
		t.Fatal("no popup over the cover")
	}
	r.a.PumpOnce()
	at := paintengine2d.Pt(pop.Bounds().Min.X+8, pop.Bounds().Min.Y+8)
	if h := r.w.hit(at); h == nil || widget.LiveUnder(h, cover) {
		t.Errorf("a press on the popup landed on %T, want something in the popup", h)
	}
}

// **A Hangar window really is cut**: the three corners its look declares
// away are transparent in the buffer, so the desktop shows through them,
// and the fourth is not. A look can say anything about its silhouette;
// this is the end of the chain, where the pixels either went or did not.
func TestAHangarWindowsCornersAreCutOut(t *testing.T) {
	p, ok := style.LoadTheme("hangar")
	if !ok {
		t.Skip("the hangar pack is not in this build")
	}
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: p.Look(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Console", Width: 600, Height: 400,
		Headless: true, Decorations: platform.DecorationsClient})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Close)
	w.SetContent(widgets.NewLabel("x"))
	w.SetTitleBar(widgets.NewTitleBar("Console", ""))
	a.PumpOnce()
	w.frame()
	if !w.geom.framed {
		t.Fatal("the window is not drawing its own frame")
	}
	img := w.surf.Buffer()
	g := w.geom
	alpha := func(dx, dy float32) uint8 {
		return img.NRGBAAt(int(g.window.Min.X+dx), int(g.window.Min.Y+dy)).A
	}
	if a := alpha(2, 2); a < 250 {
		t.Errorf("the top left is %d/255 opaque; it is the corner that stays square", a)
	}
	for _, c := range []struct {
		name   string
		dx, dy float32
	}{
		{"top right", g.window.Dx() - 3, 2},
		{"bottom right", g.window.Dx() - 3, g.window.Dy() - 3},
		{"bottom left", 2, g.window.Dy() - 3},
	} {
		if a := alpha(c.dx, c.dy); a > 40 {
			t.Errorf("the %s is %d/255 opaque; the silhouette should have cut it away", c.name, a)
		}
	}
}

// **And it cuts them when the application's own bar is the caption.** A
// window with tabs in its title bar (`CaptionMerged`, which is what Files
// and the tour do) came out square: the silhouette was dropped for every
// look the moment a bar was merged.
//
// That rule belongs to BeOS, whose caption is a tab only as wide as its
// title and whose silhouette leaves the rest of the top edge to the
// desktop — merged, the bar runs the full width and would be drawn
// outside the window. A look that only takes the corners off has no such
// trouble: the bar is as wide as the window either way, and the corner it
// loses is the corner the look says is not there.
func TestAHangarWindowCutsItsCornersUnderAMergedCaption(t *testing.T) {
	p, ok := style.LoadTheme("hangar")
	if !ok {
		t.Skip("the hangar pack is not in this build")
	}
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: p.Look(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Console", Width: 600, Height: 400,
		Headless: true, Decorations: platform.DecorationsClient})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Close)
	w.SetContent(widgets.NewLabel("x"))
	w.SetTitleBar(widgets.NewTitleBar("Console", ""))
	w.SetCaptionStyle(style.CaptionMerged)
	a.PumpOnce()
	w.frame()
	if !w.geom.framed {
		t.Fatal("the window is not drawing its own frame")
	}
	if !w.caption.DecorationState().Merged {
		t.Fatal("the caption did not merge; this test is not testing what it says")
	}
	img := w.surf.Buffer()
	g := w.geom
	alpha := func(dx, dy float32) uint8 {
		return img.NRGBAAt(int(g.window.Min.X+dx), int(g.window.Min.Y+dy)).A
	}
	if a := alpha(2, 2); a < 250 {
		t.Errorf("the top left is %d/255 opaque; it is the corner that stays square", a)
	}
	for _, c := range []struct {
		name   string
		dx, dy float32
	}{
		{"top right", g.window.Dx() - 3, 2},
		{"bottom right", g.window.Dx() - 3, g.window.Dy() - 3},
		{"bottom left", 2, g.window.Dy() - 3},
	} {
		if a := alpha(c.dx, c.dy); a > 40 {
			t.Errorf("the %s is %d/255 opaque under a merged caption; the silhouette should have cut it away", c.name, a)
		}
	}
}
