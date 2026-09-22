package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Popups as surfaces of their own, on the offscreen desktop's simulation of
// what X11 does (platform.Offscreen.SimulatePopups): a work area the test
// chooses, SolvePopup placing within it, and every event injected into a
// popup arriving on its window translated.

// desk is the simulated screen's work area: 1280x800 less a 40 px panel at
// the bottom, in the desktop's device pixels.
var desk = platform.FrameRect{X: 0, Y: 0, W: 1280, H: 760}

type popRig struct {
	a   *Application
	w   *Window
	o   *platform.Offscreen
	bar *widgets.MenuBar
	// picked is the last menu item run.
	picked string
}

// newPopRig is Minim's 275x116 strip at (x, y) with a menu bar: File has a
// dozen items (far taller than the window) and a submenu, Edit two items.
func newPopRig(t *testing.T, scale float32, x, y int, surfaces bool) *popRig {
	t.Helper()
	r := &popRig{}
	r.a = New(Options{Look: style.LightLook(), Headless: true, Scale: scale})
	w, err := r.a.NewWindow(platform.WindowOptions{Width: 275, Height: 116, Headless: true, X: x, Y: y})
	if err != nil {
		t.Fatal(err)
	}
	r.w = w
	r.o = w.Surface().(*platform.Offscreen)
	if surfaces {
		// The desktop's own pixels: the same screen at every scale.
		r.o.SimulatePopups(platform.FrameRect{W: int(float32(desk.W) * scale), H: int(float32(desk.H) * scale)})
	}
	pick := func(s string) func() { return func() { r.picked = s } }
	var file []*widgets.MenuItem
	for i := 0; i < 12; i++ {
		s := fmt.Sprintf("Item %d", i)
		file = append(file, widgets.Item(s, pick(s)))
	}
	file = append(file, widgets.Submenu("Recent", widgets.Item("one.txt", pick("one")), widgets.Item("two.txt", pick("two"))))
	r.bar = widgets.NewMenuBar(
		widgets.NewMenu("&File", file...),
		widgets.NewMenu("&Edit", widgets.Item("Cut", pick("cut")), widgets.Item("Copy", pick("copy"))),
	)
	col := widgets.NewColumn(r.bar, widgets.NewLabel("strip"))
	w.SetContent(col)
	r.a.PumpOnce()
	return r
}

func (r *popRig) menu() *widgets.PopupMenu {
	p, _ := r.w.Popup().(*widgets.PopupMenu)
	return p
}

// rowCentre is item i of p in window device pixels.
func rowCentre(p *widgets.PopupMenu, i int) paintengine2d.Point {
	b := p.Bounds()
	row := p.ItemBounds(i)
	return paintengine2d.Pt(b.Min.X+(row.Min.X+row.Max.X)/2, b.Min.Y+(row.Min.Y+row.Max.Y)/2)
}

func TestPopupSurfaceRunsPastItsWindow(t *testing.T) {
	for _, sc := range []float32{1, 1.75} {
		t.Run(fmt.Sprint(sc), func(t *testing.T) {
			r := newPopRig(t, sc, 200, 100, true)
			r.bar.Open(0)
			r.a.PumpOnce()
			pop := r.menu()
			if pop == nil || len(r.w.pops) != 1 {
				t.Fatalf("menu not on a surface: popup %v, surfaces %d", pop != nil, len(r.w.pops))
			}
			pl := r.w.pops[0]
			_, wh := r.w.PixelSize()
			b := pop.Bounds()
			if b.Max.Y <= float32(wh)+50 {
				t.Fatalf("menu %v stops at the %d px window: it should run past it", b, wh)
			}
			if ch := pop.ContentSize().Y; b.Dy() < ch-1 || b.Dy() > ch+2 {
				t.Fatalf("menu cut to %v of its %v: nothing on this desktop constrains it", b.Dy(), ch)
			}
			// Placed flush under the File title, left edges aligned.
			a, ok := widget.PopupAnchorOf(pop)
			if !ok || a.Side != widget.PopupBelow {
				t.Fatalf("anchor %+v %v", a, ok)
			}
			if d := b.Min.Y - a.Rect.Max.Y; d < -2 || d > 2 {
				t.Fatalf("menu top %v, title bottom %v", b.Min.Y, a.Rect.Max.Y)
			}
			if d := b.Min.X - a.Rect.Min.X; d < -2 || d > 2 {
				t.Fatalf("menu left %v, title left %v", b.Min.X, a.Rect.Min.X)
			}
			// The component sits exactly where the surface's visible box is.
			o := pl.surf.Origin()
			m := pl.frame.Margin
			if b.Min.X != o.X+float32(m.Left) || b.Min.Y != o.Y+float32(m.Top) {
				t.Fatalf("component at %v, surface box at %v+%v", b.Min, o, m)
			}
			if !pl.surf.(*platform.Offscreen).Grabbed() {
				t.Fatal("a menu takes the pointer and the keyboard")
			}
			// Its own shadow in its own margin, and the menu painted in its
			// own buffer: the window did not paint it.
			img := pl.surf.Buffer()
			if m.Bottom < 1 {
				t.Fatalf("no shadow margin: %+v", m)
			}
			if _, _, _, sa := img.At(m.Left+int(b.Dx())/2, m.Top+int(b.Dy())+1).RGBA(); sa == 0 {
				t.Fatal("no shadow under the menu")
			}
			if _, _, _, ma := img.At(m.Left+int(b.Dx())/2, m.Top+int(b.Dy())/2).RGBA(); ma != 0xffff {
				t.Fatal("menu not painted in its surface")
			}
		})
	}
}

// changed counts the pixels of box that differ between two captures.
func changed(a, b *paintengine2d.Image, box paintengine2d.Rect) int {
	n := 0
	x0, y0, x1, y1 := box.IntBounds()
	for y := max(y0, 0); y < min(y1, a.Height, b.Height); y++ {
		for x := max(x0, 0); x < min(x1, a.Width, b.Width); x++ {
			if a.At(x, y) != b.At(x, y) {
				n++
			}
		}
	}
	return n
}

// The window stops painting its popups once they have surfaces: what is
// under the menu in the window's buffer is the window's own content.
func TestWindowDoesNotPaintItsPopupSurfaces(t *testing.T) {
	r := newPopRig(t, 1, 200, 100, true)
	before := r.w.Capture()
	r.bar.Open(0)
	r.a.PumpOnce()
	after := r.w.Capture()
	b := r.menu().Bounds()
	// The File title shows it is open; below it nothing may change.
	b.Min.Y += 2
	if n := changed(before, after, b); n != 0 {
		t.Fatalf("the window painted %d pixels of the menu", n)
	}
	// And the same menu drawn inside the window does change them.
	r2 := newPopRig(t, 1, 200, 100, false)
	before = r2.w.Capture()
	r2.bar.Open(0)
	r2.a.PumpOnce()
	if n := changed(before, r2.w.Capture(), r2.menu().Bounds()); n < 100 {
		t.Fatalf("in-window menu changed only %d pixels: the check above proves nothing", n)
	}
}

func TestPopupFlipsAndSlidesInTheWorkArea(t *testing.T) {
	// Near the bottom of the work area: a combo box's list goes above.
	r := newPopRig(t, 1, 900, 700, true)
	combo := widgets.NewComboBox([]string{"a", "b", "c", "d", "e", "f", "g", "h"}, 0, nil)
	r.w.SetContent(widgets.NewColumn(widgets.NewLabel("x"), combo))
	r.a.PumpOnce()
	combo.Open()
	r.a.PumpOnce()
	pop := r.w.Popup()
	if pop == nil || len(r.w.pops) != 1 {
		t.Fatal("no list surface")
	}
	cb := widget.DeviceBounds(combo)
	pb := pop.Bounds()
	if pb.Max.Y > cb.Min.Y+1 {
		t.Fatalf("list %v below combo %v with no room below", pb, cb)
	}
	// Screen coordinates: the list is inside the work area.
	placed := r.w.pops[0].surf.Placed()
	if 700+placed.Y+placed.H > desk.H {
		t.Fatalf("list runs past the work area: placed %+v", placed)
	}

	// At the right edge of the screen: a context menu slides or flips back.
	r2 := newPopRig(t, 1, 1100, 200, true)
	pop2 := widgets.ShowContextMenu(r2.w.Content(), paintengine2d.Pt(260, 60),
		widgets.Item("A fairly long menu item label", nil), widgets.Item("Another", nil))
	r2.a.PumpOnce()
	if pop2 == nil || len(r2.w.pops) != 1 {
		t.Fatal("no context menu surface")
	}
	p2 := r2.w.pops[0].surf.Placed()
	if 1100+p2.X+p2.W > desk.W {
		t.Fatalf("context menu past the screen's right edge: %+v", p2)
	}
	if p2.X+p2.W > 260 {
		// It went left of the pointer, the other side of the anchor.
		t.Fatalf("context menu %+v should have flipped left of the pointer", p2)
	}
}

func TestSubmenuIsANestedSurface(t *testing.T) {
	r := newPopRig(t, 1, 200, 100, true)
	r.bar.Open(0)
	r.a.PumpOnce()
	pop := r.menu()
	sub := len(pop.Items) - 1
	r.w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: rowCentre(pop, sub)})
	r.a.PumpOnce()
	if pop.CascadeMenu() == nil || len(r.w.pops) != 2 {
		t.Fatalf("hovering Recent: cascade %v, surfaces %d", pop.CascadeMenu() != nil, len(r.w.pops))
	}
	child := r.w.pops[1].surf.(*platform.Offscreen)
	if child.PopupOptions().Parent != r.w.pops[0].surf {
		t.Fatal("the submenu hangs from the window, not from its menu")
	}
	if cb, pb := pop.CascadeMenu().Bounds(), pop.Bounds(); cb.Min.X < pb.Max.X-2 {
		t.Fatalf("submenu %v not beside its menu %v", cb, pb)
	}
	// Click the submenu's first item through its own surface: the event
	// is local to it, and arrives on the window translated.
	cm := pop.CascadeMenu()
	at := rowCentre(cm, 0)
	o := child.Origin()
	local := paintengine2d.Pt(at.X-o.X, at.Y-o.Y)
	child.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: local})
	child.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: local, Button: platform.ButtonLeft})
	child.Inject(platform.Event{Kind: platform.EventMouseUp, Pos: local, Button: platform.ButtonLeft})
	r.a.PumpOnce()
	if r.picked != "one" {
		t.Fatalf("picked %q", r.picked)
	}
	if r.w.Popup() != nil || len(r.w.pops) != 0 || len(r.o.Popups()) != 0 {
		t.Fatalf("menus still up: layer %v, surfaces %d/%d", r.w.Popup() != nil, len(r.w.pops), len(r.o.Popups()))
	}
}

func TestPopupDismissal(t *testing.T) {
	// A click outside everything — past the window, where only a grab can
	// see it — takes the menu down.
	r := newPopRig(t, 1, 200, 100, true)
	r.bar.Open(0)
	r.a.PumpOnce()
	r.w.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: paintengine2d.Pt(-300, -40), Button: platform.ButtonLeft})
	r.a.PumpOnce()
	if r.w.Popup() != nil || len(r.o.Popups()) != 0 {
		t.Fatal("a click outside left the menu up")
	}
	// The window system taking it down (a click in another application).
	r.bar.Open(1)
	r.a.PumpOnce()
	if len(r.w.pops) != 1 {
		t.Fatal("no surface")
	}
	r.w.pops[0].surf.(*platform.Offscreen).SimulatePopupDone()
	r.a.PumpOnce()
	if r.w.Popup() != nil || len(r.o.Popups()) != 0 {
		t.Fatal("popup_done left the menu up")
	}
	// Escape, from the keyboard.
	r.bar.Open(0)
	r.a.PumpOnce()
	r.w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	r.a.PumpOnce()
	if r.w.Popup() != nil || len(r.o.Popups()) != 0 {
		t.Fatal("Escape left the menu up")
	}
}

// keyScript drives the menus from the keyboard alone and says what it saw.
func keyScript(t *testing.T, surfaces bool) (string, *a11y.Node) {
	r := newPopRig(t, 1, 200, 100, surfaces)
	var log []string
	step := func(k platform.Key, mods platform.Modifiers) {
		r.w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: k, Mods: mods})
		r.a.PumpOnce()
		p := r.menu()
		state := "closed"
		if p != nil {
			state = fmt.Sprintf("open@%d", p.HighlightedIndex())
			if c := p.CascadeMenu(); c != nil {
				state += fmt.Sprintf(">%d", c.HighlightedIndex())
			}
		}
		log = append(log, state)
	}
	step(platform.KeyF, platform.ModAlt)
	step(platform.KeyDown, 0)
	step(platform.KeyEnd, 0)
	step(platform.KeyRight, 0)
	tree := r.w.AccessibleTree()
	step(platform.KeyDown, 0)
	step(platform.KeyLeft, 0)
	step(platform.KeyRight, 0)
	step(platform.KeyReturn, 0)
	log = append(log, "picked "+r.picked)
	return strings.Join(log, " "), tree
}

func TestPopupKeyboardAndA11yUnchanged(t *testing.T) {
	layer, lt := keyScript(t, false)
	surf, st := keyScript(t, true)
	if layer != surf {
		t.Fatalf("keyboard differs:\n layer   %s\n surface %s", layer, surf)
	}
	if !strings.HasSuffix(surf, "picked one") {
		t.Fatalf("script: %s", surf)
	}
	// Every node, role, name and state is the same, bar one bit: a row the
	// in-window menu had to cut off at the window's edge was off screen,
	// and on its own surface the whole menu is on screen.
	shape := func(n *a11y.Node) string {
		var b strings.Builder
		var walk func(n *a11y.Node, depth int)
		walk = func(n *a11y.Node, depth int) {
			fmt.Fprintf(&b, "%s%v %q %v\n", strings.Repeat(" ", depth), n.Role, n.Name, n.State&^a11y.StateOffscreen)
			for _, c := range n.Children {
				walk(c, depth+1)
			}
		}
		walk(n, 0)
		return b.String()
	}
	if shape(lt) != shape(st) {
		t.Fatalf("a11y tree differs:\n--- layer\n%s--- surface\n%s", shape(lt), shape(st))
	}
	if probs := a11y.Check(st); len(probs) != 0 {
		t.Fatalf("a11y problems: %v", probs)
	}
}

// Headless, and wherever the window system refuses a popup, menus are drawn
// inside the window as they always were.
func TestPopupFallsBackInsideTheWindow(t *testing.T) {
	check := func(t *testing.T, r *popRig) {
		t.Helper()
		before := r.w.Capture()
		r.bar.Open(0)
		r.a.PumpOnce()
		if len(r.w.pops) != 0 {
			t.Fatal("a popup surface in the fallback")
		}
		b := r.menu().Bounds()
		_, wh := r.w.PixelSize()
		if b.Max.Y > float32(wh) {
			t.Fatalf("in-window menu %v runs out of the %d px window", b, wh)
		}
		if n := changed(before, r.w.Capture(), b); n < 100 {
			t.Fatalf("the window did not paint its menu (%d pixels changed)", n)
		}
		// Keyboard and pick work in the fallback too.
		r.w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDown})
		r.w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyReturn})
		r.a.PumpOnce()
		if r.picked == "" {
			t.Fatal("no pick")
		}
	}
	t.Run("headless", func(t *testing.T) { check(t, newPopRig(t, 1, 200, 100, false)) })
	t.Run("refused", func(t *testing.T) {
		r := newPopRig(t, 1, 200, 100, true)
		r.o.SimulatePopupsRefused(true)
		check(t, r)
		if !r.w.popsRefused {
			t.Fatal("refusal not remembered")
		}
	})
	t.Run("env", func(t *testing.T) {
		t.Setenv(platform.EnvPopups, "layer")
		check(t, newPopRig(t, 1, 200, 100, true))
	})
}

func TestTooltipSurfaceTakesNoInput(t *testing.T) {
	r := newPopRig(t, 1, 200, 100, true)
	btn := widgets.NewButton("Hover me", nil)
	btn.Tip = "A tip that is wider than the strip it is shown from, by some way"
	r.w.SetContent(widgets.NewColumn(btn))
	r.a.PumpOnce()
	c := widget.DeviceBounds(btn).Center()
	r.w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: c})
	r.a.PumpOnce()
	r.w.RevealTooltip()
	r.a.PumpOnce()
	if r.w.tipPop == nil {
		t.Fatal("tooltip not on a surface")
	}
	s := r.w.tipPop.surf.(*platform.Offscreen)
	if s.Grabbed() {
		t.Fatal("a tooltip must not grab")
	}
	if f := s.Frame(); f.Shape == nil || len(f.Shape) != 0 {
		t.Fatalf("tooltip input region %v: want the empty region", f.Shape)
	}
	tb := r.w.tooltip.Bounds()
	ww, _ := r.w.PixelSize()
	if tb.Max.X <= float32(ww) {
		t.Fatalf("tooltip %v kept inside the %d px window", tb, ww)
	}
	if tb.Min.Y < c.Y {
		t.Fatalf("tooltip %v not below the pointer at %v", tb, c)
	}
}

// A popup may be any shape: its component's own hit shape cuts its surface,
// its input region and its shadow.
func TestShapedPopupSurface(t *testing.T) {
	r := newPopRig(t, 1, 200, 100, true)
	pop := widgets.NewPopupMenu(widgets.Item("One", nil), widgets.Item("Two", nil), widgets.Item("Three", nil))
	pop.SetHitShapeFunc(func(sz paintengine2d.Point) *platform.Shape {
		return platform.ShapeEllipse(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	})
	widget.PreparePopup(r.w.Content(), pop)
	widget.PlacePopup(pop, paintengine2d.Pt(40, 40), 360, 480)
	widget.ShowPopup(r.w.Content(), pop)
	r.a.PumpOnce()
	if len(r.w.pops) != 1 {
		t.Fatal("no surface")
	}
	pl := r.w.pops[0]
	f := pl.surf.Frame()
	if len(f.Shape) < 4 {
		t.Fatalf("input region %v: want the ellipse's rows", f.Shape)
	}
	b := pop.Bounds()
	m := f.Margin
	img := pl.surf.Buffer()
	// The ellipse's bounding box corner is outside it: transparent.
	if _, _, _, a := img.At(m.Left+1, m.Top+1).RGBA(); a != 0 {
		t.Fatalf("corner alpha %d: the silhouette was not cut", a)
	}
	if _, _, _, a := img.At(m.Left+int(b.Dx())/2, m.Top+int(b.Dy())/2).RGBA(); a != 0xffff {
		t.Fatal("the popup's middle is not painted")
	}
	// Its shadow follows the ellipse: under the bottom of it, not in the
	// bounding box's corner.
	if _, _, _, a := img.At(m.Left+int(b.Dx())/2, m.Top+int(b.Dy())+2).RGBA(); a == 0 {
		t.Fatal("no shadow below the ellipse")
	}
}

// The look can shape its popups too (style.PopupShapeEngine).
func TestLookShapedPopup(t *testing.T) {
	style.RegisterEngine(diamondPopups{})
	style.RegisterPack(style.ThemePack{Name: "diamond-popups-test", Palette: style.ThemeLight,
		Tokens: style.ThemeTokens{Engine: "diamond-popups"}})
	lk := style.ThemePack{Name: "diamond-popups-test", Palette: style.ThemeLight}.Look()
	if _, ok := lk.Engine().(style.PopupShapeEngine); !ok {
		t.Fatalf("engine %T", lk.Engine())
	}
	r := newPopRig(t, 1, 200, 100, true)
	r.a.SetLook(lk)
	r.a.PumpOnce()
	r.bar.Open(1)
	r.a.PumpOnce()
	if len(r.w.pops) != 1 {
		t.Fatal("no surface")
	}
	if f := r.w.pops[0].surf.Frame(); len(f.Shape) < 4 {
		t.Fatalf("input region %v: want the look's diamond", f.Shape)
	}
}

type diamondPopups struct{ style.BaseEngine }

func (diamondPopups) ID() string { return "diamond-popups" }

func (diamondPopups) PopupShape(l *style.Classic, b paintengine2d.Rect, kind style.PopupKind) *style.Silhouette {
	p := paintengine2d.NewPath()
	p.MoveTo(b.Dx()/2, 0)
	p.LineTo(b.Dx(), b.Dy()/2)
	p.LineTo(b.Dx()/2, b.Dy())
	p.LineTo(0, b.Dy()/2)
	p.Close()
	return &style.Silhouette{Path: p}
}
