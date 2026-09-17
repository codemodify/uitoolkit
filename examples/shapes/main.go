// Command shapes is the shaped-window demo: a window that is not a
// rectangle, with a hole through the middle you can see the desktop through
// and click through to whatever is behind it, and a panel of real glass —
// the desktop blurred by the compositor rather than by the window.
//
//	shapes                  # a clock: a disc with a hole at the hub
//	shapes -mode ring       # a rounded panel with a big hole in the middle
//	shapes -mode panel      # a rectangular panel
//	shapes -glass           # ask the desktop to blur what is behind
//	shapes -mode opaque     # the same window, unshaped: the baseline
//	shapes -mode backdrop   # a test card to put *behind* the others
//
// Drag it with the left button, press Escape to quit. It prints one line
// per silhouette with the rectangle count it needed, and counts the presses
// it actually receives: a click in the hole that the app never hears about
// is the proof that the hole is really a hole.
//
// Run it on any desktop. Where the compositor cannot blur (-glass on a
// plain X screen) the window says so and paints its own tint instead; on an
// X11 screen with no compositing manager at all the silhouette still works,
// hard-edged, through XShape's bounding shape.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func main() {
	mode := flag.String("mode", "clock", "clock | ring | panel | opaque | backdrop")
	glass := flag.Bool("glass", false, "ask the desktop to blur what is behind the window")
	size := flag.Int("size", 420, "window size in logical pixels")
	shot := flag.String("shot", "", "paint one frame offscreen, write this PNG and exit")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("shapes: ")

	headless := *shot != ""
	a := uitoolkit.New(uitoolkit.Options{Headless: headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Shapes — " + *mode, Width: *size, Height: *size,
		MinWidth: 160, MinHeight: 160, Headless: headless,
		// A shaped window draws its own silhouette, so it wants no frame
		// from anyone: a title bar around a disc would be a rectangle
		// around it.
		Decorations: platform.DecorationsNone,
	})
	if err != nil {
		log.Fatal(err)
	}

	f := newFace(*mode, win)
	win.SetContent(f)
	if *mode != "opaque" && *mode != "backdrop" {
		// The silhouette follows the window: SetShapeFunc is called again
		// at every size and scale, so its edge is drawn afresh at 1.75
		// rather than stretched from the shape of some other size.
		win.SetShapeFunc(func(sz paintengine2d.Point, scale float32) *platform.Shape {
			s := shapeFor(*mode, sz)
			if s != nil {
				r := s.Raster(int(sz.X), int(sz.Y))
				log.Printf("shape=%s size=%.0fx%.0f scale=%.2f rects=%d clamped=%v",
					*mode, sz.X, sz.Y, scale, len(r.Rects), r.Clamped)
			}
			return s
		})
	}
	if *glass {
		win.SetGlass(true)
		// Let plenty of the blurred desktop through: this demo is here to
		// show the blur, not to be a readable document window.
		win.SetGlassTint(a.Look().Palette().Background.WithAlpha(0.35))
	}

	if headless {
		if err := win.WritePNG(*shot); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", *shot)
		return
	}
	a.Post(func() {
		log.Printf("backend=%s scale=%.2f glass asked=%v available=%v",
			a.BackendName(), win.Scale(), *glass, win.GlassAvailable())
		if *glass && !win.GlassAvailable() {
			log.Print("this desktop does not blur behind a window; " +
				"the window paints its own tint instead")
		}
	})
	if *mode == "clock" {
		// The hands move, so the clock repaints twice a second; every
		// other mode is static and repaints only when something happens
		// to it. Post is the one safe way in from another goroutine.
		go func() {
			for range time.Tick(time.Second / 2) {
				a.Post(f.Invalidate)
			}
		}()
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

// shapeFor is each mode's silhouette, in the window's device pixels. A hole
// is a second contour of the same path, filled even-odd — the whole trick,
// and exactly what CombineRgn(RGN_XOR) did for Win32's SetWindowRgn.
func shapeFor(mode string, sz paintengine2d.Point) *platform.Shape {
	w, h := sz.X, sz.Y
	d := min(w, h)
	switch mode {
	case "clock":
		// A disc with a hole at the hub: the desktop shows through the
		// middle of the clock, and a click there lands behind it.
		p := paintengine2d.NewPath()
		p.AddCircle(paintengine2d.Pt(w/2, h/2), d/2-1)
		p.AddCircle(paintengine2d.Pt(w/2, h/2), d*0.11)
		return platform.NewShapeEvenOdd(p)
	case "ring":
		p := paintengine2d.NewPath()
		p.AddRoundRect(paintengine2d.XYWH(0, 0, w, h), d*0.07, d*0.07)
		p.AddCircle(paintengine2d.Pt(w/2, h/2), d*0.29)
		return platform.NewShapeEvenOdd(p)
	case "panel":
		r := d * 0.05
		return platform.ShapeRoundRect(paintengine2d.XYWH(0, 0, w, h), [4]float32{r, r, r, r})
	}
	return nil
}

// face is the demo's one widget: it paints what is inside the silhouette
// and counts the presses that reach it.
type face struct {
	widget.Base
	mode   string
	win    *app.Window
	clicks int
}

func newFace(mode string, win *app.Window) *face {
	f := &face{mode: mode, win: win}
	f.Init(f)
	f.SetWantsFocus(true)
	// No widget-level hit shape here on purpose. The window's own input
	// region already keeps every press in the hole away from this app, and
	// a second silhouette at the widget level would only be one more thing
	// to keep in step — it would still be refusing presses in the hole
	// after the window was maximized and gave its hole up.
	// widget.Base.SetHitShape is for a component that is a shape *inside*
	// a window, which is a different job.
	return f
}

func (f *face) Paint(ctx *paintengine2d.Context) {
	b := f.LocalBounds()
	w, h := b.Dx(), b.Dy()
	pal := f.Look().Palette()

	// With real glass the window's own background is already the tint over
	// the blurred desktop: painting another coat of it here would hide the
	// blur this demo exists to show.
	glass := f.win.Glass() && f.win.GlassAvailable()
	body := pal.Background.WithAlpha(0.93)
	if glass {
		body = paintengine2d.Transparent
	}

	// Maximized, full-screen and tiled windows give their silhouette up,
	// as they give their corners and their shadow up: paint the plain
	// rectangle the window actually is, not the hole it used to have.
	if s := f.shape(w, h); s != nil {
		path, rule := s.Path()
		if body.A > 0 {
			fill := paintengine2d.Fill(body)
			fill.FillRule, fill.AntiAlias = rule, true
			ctx.DrawPath(path, fill)
		}
		rim := paintengine2d.StrokePaint(pal.Accent, max(min(w, h)*0.012, 2))
		rim.FillRule, rim.AntiAlias = rule, true
		ctx.DrawPath(path, rim)
	} else if body.A > 0 {
		ctx.DrawRect(b, paintengine2d.Fill(body))
	}

	if f.mode == "backdrop" {
		f.paintBackdrop(ctx, b)
		return
	}
	if f.mode == "clock" && f.win.ShapeActive() {
		f.paintClock(ctx, b, pal)
	}
	// One pip per press the app actually received. A click in the hole
	// adds none: it never reached this window at all.
	for i := 0; i < f.clicks && i < 12; i++ {
		x := b.Min.X + w/2 - 60 + float32(i)*11
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y+h*0.76, 7, 7), paintengine2d.Fill(pal.Accent))
	}
	f.Look().DrawLabel(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y+h*0.82, w, h*0.1),
		f.caption(), pal.Text, style.AlignCenter)
}

// paintBackdrop is the test card that goes behind a shaped or glassy
// window: a coarse colour grid with two diagonals across it. A patch of it
// seen through a hole says exactly which patch it is, so moving the window
// shows a different one — which a painted fake of the desktop could not do
// — and a blur of it is unmistakable, which a flat grey one is not.
func (f *face) paintBackdrop(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	cols, rows := 8, 5
	cw, ch := b.Dx()/float32(cols), b.Dy()/float32(rows)
	pal := []paintengine2d.Color{
		paintengine2d.RGBA(0.86, 0.20, 0.24, 1), paintengine2d.RGBA(0.96, 0.60, 0.13, 1),
		paintengine2d.RGBA(0.98, 0.87, 0.21, 1), paintengine2d.RGBA(0.30, 0.76, 0.35, 1),
		paintengine2d.RGBA(0.16, 0.56, 0.86, 1), paintengine2d.RGBA(0.48, 0.30, 0.78, 1),
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X+float32(c)*cw, b.Min.Y+float32(r)*ch, cw, ch),
				paintengine2d.Fill(pal[(r*cols+c)%len(pal)]))
			// A row of white pips whose count names the cell.
			px := b.Min.X + float32(c)*cw + cw*0.5
			py := b.Min.Y + float32(r)*ch + ch*0.5
			for k := 0; k <= (r*cols+c)%6; k++ {
				ctx.DrawRect(paintengine2d.XYWH(px-26+float32(k)*10, py-5, 7, 11),
					paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.95)))
			}
		}
	}
	d := paintengine2d.NewPath()
	d.MoveTo(b.Min.X, b.Min.Y)
	d.LineTo(b.Max.X, b.Max.Y)
	d.MoveTo(b.Max.X, b.Min.Y)
	d.LineTo(b.Min.X, b.Max.Y)
	ctx.DrawPath(d, paintengine2d.StrokePaint(paintengine2d.RGBA(0, 0, 0, 0.85), 14))
}

// shape is the silhouette to paint: none while the window's own is dropped.
func (f *face) shape(w, h float32) *platform.Shape {
	if !f.win.ShapeActive() {
		return nil
	}
	return shapeFor(f.mode, paintengine2d.Pt(w, h))
}

func (f *face) caption() string {
	switch {
	case f.mode == "opaque":
		return "unshaped baseline"
	case f.mode == "backdrop":
		return ""
	case !f.win.ShapeActive():
		return "maximized: the silhouette is dropped"
	case f.win.Glass() && f.win.GlassAvailable():
		return "glass: the desktop is blurred"
	case f.win.Glass():
		return "glass: this desktop cannot blur"
	}
	return "click the hole — it goes behind"
}

// paintClock draws the dial and the hands, so the hole is obviously part of
// a running app rather than a picture of one.
func (f *face) paintClock(ctx *paintengine2d.Context, b paintengine2d.Rect, pal style.Palette) {
	cx, cy := b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2
	rad := min(b.Dx(), b.Dy())/2 - 1
	// Twelve ticks, the quarters longer.
	ticks := paintengine2d.NewPath()
	for i := 0; i < 12; i++ {
		a := float64(i) * math.Pi / 6
		in := rad * 0.86
		if i%3 == 0 {
			in = rad * 0.78
		}
		sx, sy := float32(math.Sin(a)), float32(-math.Cos(a))
		ticks.MoveTo(cx+sx*in, cy+sy*in)
		ticks.LineTo(cx+sx*rad*0.93, cy+sy*rad*0.93)
	}
	ctx.DrawPath(ticks, paintengine2d.StrokePaint(pal.TextMuted, max(rad*0.012, 1)))

	now := time.Now()
	hand := func(turns float64, length, width float32, col paintengine2d.Color) {
		a := turns * 2 * math.Pi
		sx, sy := float32(math.Sin(a)), float32(-math.Cos(a))
		p := paintengine2d.NewPath()
		// The hands start outside the hub's hole: there is nothing to
		// draw on in the middle of this clock.
		p.MoveTo(cx+sx*rad*0.13, cy+sy*rad*0.13)
		p.LineTo(cx+sx*length, cy+sy*length)
		ctx.DrawPath(p, paintengine2d.StrokePaint(col, width))
	}
	h, m, s := now.Hour()%12, now.Minute(), now.Second()
	hand((float64(h)+float64(m)/60)/12, rad*0.52, max(rad*0.035, 2), pal.Text)
	hand((float64(m)+float64(s)/60)/60, rad*0.74, max(rad*0.024, 2), pal.Text)
	hand(float64(s)/60, rad*0.80, max(rad*0.010, 1), pal.Accent)
}

func (f *face) MousePress(e widget.MouseEvent) bool {
	f.clicks++
	f.Invalidate()
	log.Printf("press at %.0f,%.0f — the app got it (total %d)", e.Pos.X, e.Pos.Y, f.clicks)
	// Dragging the window is what proves the hole belongs to it: the
	// desktop behind slides past the hole as the window moves.
	if e.Button == platform.ButtonLeft {
		f.win.StartMove()
	}
	return true
}

func (f *face) KeyPress(e widget.KeyEvent) bool {
	if e.Key == platform.KeyEscape {
		f.win.RequestClose()
		return true
	}
	return false
}
