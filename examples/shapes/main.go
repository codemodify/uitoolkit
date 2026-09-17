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
	mode := flag.String("mode", "clock", "clock | ring | panel | opaque")
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
	if *mode != "opaque" {
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
	if mode != "opaque" {
		// The widget takes input only where the window really is. The
		// window's input region already tells the compositor that; saying
		// it here too is what a widget inside a shaped window does, and it
		// is what keeps hover and the cursor honest at the edge.
		f.SetHitShapeFunc(func(sz paintengine2d.Point) *platform.Shape {
			return shapeFor(mode, sz)
		})
	}
	return f
}

func (f *face) Paint(ctx *paintengine2d.Context) {
	b := f.LocalBounds()
	w, h := b.Dx(), b.Dy()
	pal := f.Look().Palette()

	// How much of what is behind shows through. With real glass the
	// compositor has blurred the desktop already, so far more of it can
	// show without the window becoming unreadable.
	alpha := float32(0.93)
	if f.win.Glass() && f.win.GlassAvailable() {
		alpha = 0.45
	}
	body := pal.Background.WithAlpha(alpha)

	if s := shapeFor(f.mode, paintengine2d.Pt(w, h)); s != nil {
		path, rule := s.Path()
		fill := paintengine2d.Fill(body)
		fill.FillRule, fill.AntiAlias = rule, true
		ctx.DrawPath(path, fill)
		rim := paintengine2d.StrokePaint(pal.Accent, max(min(w, h)*0.012, 2))
		rim.FillRule, rim.AntiAlias = rule, true
		ctx.DrawPath(path, rim)
	} else {
		ctx.DrawRect(b, paintengine2d.Fill(pal.Background))
	}

	if f.mode == "clock" {
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

func (f *face) caption() string {
	switch {
	case f.mode == "opaque":
		return "unshaped baseline"
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
