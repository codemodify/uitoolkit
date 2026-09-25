package tourapp

import (
	"math"
	"strconv"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// A window need not be a rectangle. The demonstration cannot
// happen in this page — a hole in a page is just a drawing — so the page
// opens two windows: a test card, and over it a window with a hole in it.
// What proves the hole is real is that a press inside it is counted by
// the card underneath and not by the window on top. Both counters are on
// this page, side by side, which is the only way to show an input region
// without a video.

func init() {
	p := &tourPages[pageShapes]
	p.title = "Shaped and transparent windows"
	p.proof = "A window's silhouette is a path: this one has a hole you can see the desktop through " +
		"and click through, and glass behind it where the compositor offers any."
	p.try = "Open the pair, then click in the hole — the card behind counts it. On a touchpad, pinch the stamp."
	p.build = buildShapesPage
}

// The silhouettes on offer. A hole needs the even-odd rule: two
// subpaths, the inner one cutting the outer.
var tourShapeNames = []string{"Ring — a hole in the middle", "Clock — a hub hole", "Stadium", "Rounded panel"}

func tourShapeFor(kind int, sz paintengine2d.Point) *platform.Shape {
	w, h := sz.X, sz.Y
	d := min(w, h)
	switch kind {
	case 0:
		p := paintengine2d.NewPath()
		p.AddRoundRect(paintengine2d.XYWH(0, 0, w, h), d*0.08, d*0.08)
		p.AddCircle(paintengine2d.Pt(w/2, h/2), d*0.27)
		return platform.NewShapeEvenOdd(p)
	case 1:
		p := paintengine2d.NewPath()
		p.AddCircle(paintengine2d.Pt(w/2, h/2), d/2-1)
		p.AddCircle(paintengine2d.Pt(w/2, h/2), d*0.11)
		return platform.NewShapeEvenOdd(p)
	case 2:
		r := min(w, h) / 2
		return platform.ShapeRoundRect(paintengine2d.XYWH(0, 0, w, h), [4]float32{r, r, r, r})
	default:
		r := d * 0.06
		return platform.ShapeRoundRect(paintengine2d.XYWH(0, 0, w, h), [4]float32{r, r, r, r})
	}
}

type shapesPage struct {
	t *tourState
	// shaped is the window with the hole, card the test card behind it.
	shaped, card *app.Window
	shapedFace   *tourFace
	cardFace     *tourFace
	kind         int
	glass        bool
	facts        *widgets.TextArea
	glassSwitch  *widgets.Switch
	preview      *shapePreview
}

func buildShapesPage(t *tourState) widget.Component {
	p := &shapesPage{t: t}
	t.own(pageShapes, p)

	pick := widgets.NewComboBox(tourShapeNames, 0, func(i int) {
		p.kind = i
		if p.shaped != nil {
			// SetShapeFunc memoises on size and scale, so handing it a new
			// closure is what makes the window take the new silhouette.
			p.shaped.SetShapeFunc(func(sz paintengine2d.Point, _ float32) *platform.Shape {
				return tourShapeFor(p.kind, sz)
			})
			p.shapedFace.Invalidate()
		}
		p.refresh()
		if p.preview != nil {
			p.preview.Invalidate()
		}
	})
	pick.SetAccessibleName("Silhouette")

	p.glassSwitch = widgets.NewSwitch("Glass behind it", false, func(on bool) {
		p.glass = on
		if p.shaped != nil {
			p.shaped.SetGlass(on)
			p.shaped.SetGlassTint(p.shaped.Look().Palette().Background.WithAlpha(0.35))
			p.shapedFace.Invalidate()
		}
		if on && p.shaped != nil && !p.shaped.GlassAvailable() {
			p.note("This desktop does not blur behind a window; the window paints its own tint instead.")
		}
		p.refresh()
	})

	open := widgets.NewButton("Open the pair", func() { p.open() })
	open.Primary = true
	shut := widgets.NewButton("Close them", func() {
		p.closeWindows()
		p.note("Closed.")
	})
	reset := widgets.NewButton("Reset the counters", func() {
		if p.shapedFace != nil {
			p.shapedFace.clicks = 0
			p.shapedFace.Invalidate()
		}
		if p.cardFace != nil {
			p.cardFace.clicks = 0
			p.cardFace.Invalidate()
		}
		p.refresh()
	})

	silhouette := newShapePreview(p)
	p.preview = silhouette
	how := widgets.NewPanel("The pair",
		tourNote("\"Open the pair\" puts a test card on the screen and a shaped window over it. Drag "+
			"the shaped one by anywhere on it — a window with no frame moves itself — and the card "+
			"slides past behind the hole, which a painted fake of a hole could not do."),
		tourNote("Now click in the hole. The card's counter goes up and the ring's does not: the hole "+
			"is not in this application at all. The silhouette becomes the window's input region — "+
			"wl_surface.set_input_region on Wayland, XShape on X11 — so the press is delivered to "+
			"whatever is underneath."),
		tourNote("Escape closes either window."),
		widgets.NewSeparator(),
		widgets.NewRow(
			widgets.NewColumn(
				tourRow("Silhouette", pick),
				p.glassSwitch,
				widgets.NewRow(open, shut, reset).WithGap(6),
			).WithGap(8),
			silhouette,
		).WithGap(12),
	)
	how.Content().Spec.Gap = 8

	gave := widgets.NewPanel("What a shaped window gives up",
		tourNote("A maximized, tiled or full-screen window drops its silhouette, as it drops its "+
			"corners and its shadow: maximize the shaped window and the hole closes, because the "+
			"desktop has told it to be exactly that rectangle. Restore it and the hole is back."),
		tourNote("A shaped window also gives up its resize band — the frame's border is gone, so there "+
			"is nothing at the edge to grab. A real app with a silhouette provides its own handle."),
		tourNote("On an X11 screen with no compositing manager there is no alpha at all: the shape "+
			"survives, with a hard edge, and glass does not."),
	)
	gave.Content().Spec.Gap = 7

	panel, facts := tourReadout("The silhouette, measured")
	p.facts = facts

	stage := widgets.NewColumn(how, gave, widgets.NewSpacer()).WithGap(10)
	stage.AddFlex(widgets.NewSpacer(), 1)

	p.refresh()
	t.onClose(p.closeWindows)
	return tourStage(tourScroll("Shapes page", stage), panel)
}

// open puts the card up first and the shaped window over it, so the one
// with the hole in it is the one on top.
func (p *shapesPage) open() {
	p.closeWindows()
	a := p.t.a

	card, err := a.NewWindow(platform.WindowOptions{
		Title: "Backdrop — the window under the hole", Width: 620, Height: 420,
		MinWidth: 240, MinHeight: 180,
	})
	if err != nil {
		p.note("Could not open the backdrop: " + err.Error())
		return
	}
	p.card = card
	p.cardFace = newTourFace(card, p, true)
	card.SetContent(p.cardFace)

	shaped, err := a.NewWindow(platform.WindowOptions{
		Title: "Shaped", Width: 340, Height: 340, MinWidth: 160, MinHeight: 160,
		Decorations: platform.DecorationsNone,
	})
	if err != nil {
		p.note("Could not open the shaped window: " + err.Error())
		return
	}
	p.shaped = shaped
	shaped.SetShapeFunc(func(sz paintengine2d.Point, _ float32) *platform.Shape {
		return tourShapeFor(p.kind, sz)
	})
	if p.glass {
		shaped.SetGlass(true)
		shaped.SetGlassTint(shaped.Look().Palette().Background.WithAlpha(0.35))
	}
	p.shapedFace = newTourFace(shaped, p, false)
	shaped.SetContent(p.shapedFace)

	// X11 lets a client place its own windows, so the pair can be put one
	// over the other; a Wayland toplevel has no position at all and the
	// compositor decides. Either way the user can drag them together.
	if x, y, ok := card.Position(); ok && shaped.CanMove() {
		shaped.Move(x+140, y+40)
	}
	shaped.Raise()
	p.note("Click in the hole: the card behind counts it, the ring does not.")
	p.refresh()
}

func (p *shapesPage) closeWindows() {
	if p.shaped != nil {
		p.shaped.Close()
		p.shaped, p.shapedFace = nil, nil
	}
	if p.card != nil {
		p.card.Close()
		p.card, p.cardFace = nil, nil
	}
	p.refresh()
}

func (p *shapesPage) note(s string) {
	p.t.note(s)
	p.refresh()
}

func (p *shapesPage) refresh() {
	if p.facts == nil {
		return
	}
	if p.shaped == nil || p.shaped.Closed() {
		look := p.t.win.Look()
		p.facts.SetText(tourFacts(
			[2]string{"the pair", "not open — press \"Open the pair\""},
			[2]string{"backend", p.t.a.BackendName()},
			[2]string{"this theme", "shapes its windows: " + yesNo(style.WindowShaped(look))},
			[2]string{"glass here", yesNo(p.t.win.GlassAvailable())},
			[2]string{"opaque screen", yesNo(p.t.win.WindowState().Solid)},
		))
		return
	}
	w, h := p.shaped.Size()
	rects, clamped, opaque, clear := 0, false, 0, 0
	if s := p.shaped.Shape(); s != nil && w > 0 && h > 0 {
		r := s.Raster(w, h)
		rects, clamped, opaque, clear = len(r.Rects), r.Clamped, len(r.Opaque), len(r.Clear)
	}
	st := p.shaped.WindowState()
	shapedNow := p.shaped.ShapeActive()
	why := ""
	switch {
	case shapedNow:
		why = "the window is that path"
	case st.Maximized:
		why = "dropped — the window is maximized"
	case st.Fullscreen:
		why = "dropped — the window is full screen"
	case st.Tiled != 0:
		why = "dropped — the window is tiled " + st.Tiled.String()
	default:
		why = "no silhouette"
	}
	ring, cardHits := 0, 0
	if p.shapedFace != nil {
		ring = p.shapedFace.clicks
	}
	if p.cardFace != nil {
		cardHits = p.cardFace.clicks
	}
	p.facts.SetText(tourFacts(
		[2]string{"silhouette", tourShapeNames[p.kind]},
		[2]string{"active", yesNo(shapedNow) + " — " + why},
		[2]string{"window", strconv.Itoa(w) + " x " + strconv.Itoa(h) + " px"},
		[2]string{"input rects", strconv.Itoa(rects)},
		[2]string{"fully opaque", strconv.Itoa(opaque) + " rects"},
		[2]string{"not covered", strconv.Itoa(clear) + " rects (the hole, and outside)"},
		[2]string{"clamped", yesNo(clamped)},
		[2]string{"", ""},
		[2]string{"glass asked", yesNo(p.shaped.Glass())},
		[2]string{"glass here", yesNo(p.shaped.GlassAvailable())},
		[2]string{"opaque screen", yesNo(st.Solid)},
		[2]string{"", ""},
		[2]string{"presses: ring", strconv.Itoa(ring)},
		[2]string{"presses: card", strconv.Itoa(cardHits) + "  (the ones through the hole)"},
		[2]string{"", ""},
		[2]string{"backend", p.t.a.BackendName()},
		[2]string{"toplevel drag", yesNo(widget.DragsWindows(p.t.strip))},
	))
}

// shapePreview draws the chosen silhouette at the size of a stamp, so
// that the page says what "a hole in the middle" means before anything is
// opened — and so that a still of this page shows a shape at all. Two
// fingers pinching on a touchpad zoom it and turn it (a
// widget.GestureTarget); a double-click puts it back.
type shapePreview struct {
	widget.Base
	page *shapesPage
	// zoom and turn are what pinches made of it (1 and 0 untouched), and
	// zoom0 the zoom the pinch in progress started from.
	zoom, zoom0, turn float32
	lastClick         time.Time
}

func newShapePreview(page *shapesPage) *shapePreview {
	s := &shapePreview{page: page, zoom: 1}
	s.Init(s)
	s.SetAccessibleName("The silhouette")
	return s
}

// shapePreviewSide is the stamp's side in design pixels; like every
// other length here it is the look's to scale.
const shapePreviewSide = 112

func (s *shapePreview) Measure(c layout.Constraints) paintengine2d.Point {
	side := style.Dip(s.Look(), shapePreviewSide)
	return c.Constrain(paintengine2d.Pt(side, side))
}

func (s *shapePreview) Arrange(b paintengine2d.Rect) { s.SetBounds(b) }

// Gesture zooms and turns the stamp with a pinch.
func (s *shapePreview) Gesture(e widget.GestureEvent) bool {
	if e.Kind != platform.GesturePinch {
		return false
	}
	switch e.Phase {
	case platform.GestureBegin:
		s.zoom0 = s.zoom
	case platform.GestureUpdate:
		s.zoom = min(max(s.zoom0*e.Scale, 0.4), 3)
		s.turn = float32(math.Mod(float64(s.turn+e.Rotation), 360))
		s.Invalidate()
	case platform.GestureCancel:
		s.zoom = s.zoom0
		s.Invalidate()
	case platform.GestureEnd:
		s.page.note("Pinched to " + strconv.Itoa(int(s.zoom*100+0.5)) + "%, turned " +
			strconv.Itoa(int(s.turn)) + "° — double-click the stamp to put it back.")
	}
	return true
}

// MousePress puts a pinched stamp back on a double-click.
func (s *shapePreview) MousePress(e widget.MouseEvent) bool {
	if e.Button != platform.ButtonLeft {
		return false
	}
	now := time.Now()
	if now.Sub(s.lastClick) < 400*time.Millisecond && (s.zoom != 1 || s.turn != 0) {
		s.zoom, s.turn = 1, 0
		s.Invalidate()
	}
	s.lastClick = now
	return false
}

func (s *shapePreview) Describe(n *a11y.Node) {
	n.Role = a11y.RoleImage
	n.Name = "The silhouette"
	n.Description = tourShapeNames[s.page.kind]
}

func (s *shapePreview) Paint(ctx *paintengine2d.Context) {
	b := s.LocalBounds()
	lk := s.Look()
	pal := lk.Palette()
	// A chequer behind it, so that the hole reads as a hole rather than
	// as a patch of the page's own colour.
	cell := style.Dip(lk, 8)
	for y := float32(0); y < b.Dy(); y += cell {
		for x := float32(0); x < b.Dx(); x += cell {
			if int((x/cell))%2 == int((y/cell))%2 {
				continue
			}
			w, h := min(cell, b.Dx()-x), min(cell, b.Dy()-y)
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X+x, b.Min.Y+y, w, h),
				paintengine2d.Fill(pal.TextMuted.WithAlpha(0.18)))
		}
	}
	sh := tourShapeFor(s.page.kind, paintengine2d.Pt(b.Dx(), b.Dy()))
	if sh == nil {
		return
	}
	path, rule := sh.Path()
	ctx.Save()
	ctx.ClipRect(b)
	if s.zoom != 1 || s.turn != 0 {
		// About the stamp's centre, as the fingers see it.
		cx, cy := b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2
		ctx.Translate(cx, cy)
		ctx.Rotate(s.turn * math.Pi / 180)
		ctx.Scale(s.zoom, s.zoom)
		ctx.Translate(-cx, -cy)
	}
	defer ctx.Restore()
	// Tinted rather than filled with a surface colour: in the flatter
	// packs the surface is the page's own colour, and the shape would
	// then be nothing but its outline.
	fill := paintengine2d.Fill(pal.Accent.WithAlpha(0.30))
	fill.FillRule, fill.AntiAlias = rule, true
	ctx.Save()
	ctx.Translate(b.Min.X, b.Min.Y)
	ctx.DrawPath(path, fill)
	rim := paintengine2d.StrokePaint(pal.Accent, max(style.Dip(lk, 2), 2))
	rim.FillRule, rim.AntiAlias = rule, true
	ctx.DrawPath(path, rim)
	ctx.Restore()
}

// ---- the two windows' content ---------------------------------------------------

// tourFace is what the shaped window and the card behind it are made of:
// one widget that paints the whole window, counts what it was pressed
// with, and moves its window when dragged.
type tourFace struct {
	widget.Base
	win    *app.Window
	page   *shapesPage
	card   bool
	clicks int
}

func newTourFace(win *app.Window, page *shapesPage, card bool) *tourFace {
	f := &tourFace{win: win, page: page, card: card}
	f.Init(f)
	f.SetWantsFocus(true)
	name := "Shaped window"
	if card {
		name = "Backdrop"
	}
	f.SetAccessibleName(name)
	// No widget-level hit shape: the window's own input region already
	// keeps a press in the hole out of this application, and a second
	// silhouette here would go on refusing presses after the window was
	// maximized and gave its hole up (examples/uitoolkit-sample-shapes says the same).
	return f
}

func (f *tourFace) Paint(ctx *paintengine2d.Context) {
	b := f.LocalBounds()
	w, h := b.Dx(), b.Dy()
	lk := f.Look()
	pal := lk.Palette()

	if f.card {
		f.paintCard(ctx, b, lk, pal)
		return
	}

	// Under real glass the window's background is already the tint over
	// the blurred desktop; another coat here would hide the blur.
	glass := f.win.Glass() && f.win.GlassAvailable()
	body := pal.Background.WithAlpha(0.94)
	if glass {
		body = paintengine2d.Transparent
	}
	if s := f.shape(w, h); s != nil {
		path, rule := s.Path()
		if body.A > 0 {
			fill := paintengine2d.Fill(body)
			fill.FillRule, fill.AntiAlias = rule, true
			ctx.DrawPath(path, fill)
		}
		rim := paintengine2d.StrokePaint(pal.Accent, max(min(w, h)*0.013, 2))
		rim.FillRule, rim.AntiAlias = rule, true
		ctx.DrawPath(path, rim)
	} else if body.A > 0 {
		ctx.DrawRect(b, paintengine2d.Fill(body))
	}

	// One pip per press this window actually received. A click in the
	// hole adds none, because it never arrived here.
	for i := 0; i < f.clicks && i < 14; i++ {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+w/2-70+float32(i)*10, b.Min.Y+h*0.72, 6, 6),
			paintengine2d.Fill(pal.Accent))
	}
	lk.DrawLabel(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+h*0.78, w, h*0.09),
		f.caption(), pal.Text, style.AlignCenter)
	lk.DrawLabel(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+h*0.86, w, h*0.09),
		"drag me · Esc closes", pal.TextMuted, style.AlignCenter)
}

// paintCard is the test card the shaped window sits over: a coarse grid
// whose cells are told apart by their pips, so a patch of it seen through
// a hole says which patch it is and moving the window shows another.
func (f *tourFace) paintCard(ctx *paintengine2d.Context, b paintengine2d.Rect, lk style.LookAndFeel, pal style.Palette) {
	cols, rows := 7, 5
	cw, ch := b.Dx()/float32(cols), b.Dy()/float32(rows)
	swatch := []paintengine2d.Color{
		paintengine2d.RGBA(0.86, 0.20, 0.24, 1), paintengine2d.RGBA(0.96, 0.60, 0.13, 1),
		paintengine2d.RGBA(0.98, 0.87, 0.21, 1), paintengine2d.RGBA(0.30, 0.76, 0.35, 1),
		paintengine2d.RGBA(0.16, 0.56, 0.86, 1), paintengine2d.RGBA(0.48, 0.30, 0.78, 1),
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X+float32(c)*cw, b.Min.Y+float32(r)*ch, cw, ch),
				paintengine2d.Fill(swatch[(r*cols+c)%len(swatch)]))
			px := b.Min.X + float32(c)*cw + cw*0.5
			py := b.Min.Y + float32(r)*ch + ch*0.5
			for k := 0; k <= (r*cols+c)%6; k++ {
				ctx.DrawRect(paintengine2d.XYWH(px-24+float32(k)*9, py-5, 6, 10),
					paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.95)))
			}
		}
	}
	// The count of presses that landed here — the ones that went through
	// the hole are in this number and in no other. It goes in the bottom
	// left on a plate of its own: the middle is behind the shaped window
	// the user is aiming through, and over these colours plain white
	// text cannot be read at all.
	pw, ph := b.Dx()*0.52, style.Dip(lk, 30)
	plate := paintengine2d.XYWH(b.Min.X+b.Dx()*0.03, b.Max.Y-ph-b.Dy()*0.04, pw, ph)
	r := style.Dip(lk, 6)
	ctx.DrawRoundRect(plate, r, r, paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.72)))
	lk.DrawLabel(ctx, plate, "presses on the card: "+strconv.Itoa(f.clicks),
		paintengine2d.RGBA(1, 1, 1, 1), style.AlignCenter)
	_ = pal
}

// shape is the silhouette to paint: none while the window's own is
// dropped, so a maximized window paints the rectangle it really is.
func (f *tourFace) shape(w, h float32) *platform.Shape {
	if f.card || !f.win.ShapeActive() {
		return nil
	}
	return tourShapeFor(f.page.kind, paintengine2d.Pt(w, h))
}

func (f *tourFace) caption() string {
	switch {
	case !f.win.ShapeActive():
		return "maximized: the silhouette is dropped"
	case f.win.Glass() && f.win.GlassAvailable():
		return "glass: the desktop behind is blurred"
	case f.win.Glass():
		return "glass asked for; this desktop cannot blur"
	}
	return "click the hole — it goes behind"
}

func (f *tourFace) MousePress(e widget.MouseEvent) bool {
	f.clicks++
	f.Invalidate()
	if f.page != nil {
		f.page.refresh()
	}
	if f.card {
		// A press that came through the hole is a press on this window,
		// so the desktop raises it — and the shaped window the user is
		// aiming through disappears behind it after one go. Putting the
		// ring back on top is what a real heads-up window would do, and
		// it is what makes the hole worth clicking twice.
		if f.page != nil && f.page.shaped != nil && !f.page.shaped.Closed() {
			f.page.shaped.Raise()
		}
		return true
	}
	// Dragging the window is what proves the hole belongs to it: the card
	// behind slides past the hole as the window moves.
	if e.Button == platform.ButtonLeft {
		f.win.StartMove()
	}
	return true
}

func (f *tourFace) KeyPress(e widget.KeyEvent) bool {
	if e.Key == platform.KeyEscape {
		f.win.RequestClose()
		return true
	}
	return false
}
