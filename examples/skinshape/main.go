// Command skinshape is the shaped-*look* demo: a window whose outline
// belongs to the theme rather than to the app, and a round control whose hit
// area is its own artwork.
//
//	skinshape                       # Deck: a shoulder, a narrower body, round buttons
//	skinshape -theme beos           # BeOS's tab as the window's real outline
//	skinshape -theme breeze-night   # the same app in a look that frames a rectangle
//	skinshape -mode backdrop        # a test card to put *behind* the others
//	skinshape -shot out.png         # paint one frame offscreen, write it, exit
//
// Put the backdrop up first and the shaped window over it. Everywhere the
// silhouette is not, the test card shows through and a press lands on it —
// the window never hears about it at all.
//
// Inside the window the same rule applies one layer down. The disc is an
// ordinary widgets.Button; Deck's art for a push button is a stadium, so the
// corners of the disc's box are not the button. The counter says which of
// the two a press reached, and pressing a corner adds to "panel", not to
// "button". Run it in a look that draws rectangles and every press is the
// button's, because then the button really is its whole box.
//
// Escape quits.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	theme := flag.String("theme", "deck", "the pack to run in (a skin, or any of the 121)")
	mode := flag.String("mode", "window", "window | backdrop")
	width := flag.Int("width", 560, "window width in logical pixels")
	height := flag.Int("height", 340, "window height in logical pixels")
	scale := flag.Float64("scale", 0, "display scale (0: the desktop's, or $UITK_SCALE)")
	shot := flag.String("shot", "", "paint one frame offscreen, write this PNG and exit")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("skinshape: ")

	if *theme != "" && *mode != "backdrop" {
		// The same override UITK_THEME=<pack> gives any app; the look is
		// built when the application is, so it has to be set first.
		os.Setenv(style.ThemeEnv, *theme)
	}
	headless := *shot != ""
	a := uitoolkit.New(uitoolkit.Options{Headless: headless, Scale: float32(*scale)})
	opts := platform.WindowOptions{
		Title: "Deck", Width: *width, Height: *height,
		MinWidth: 320, MinHeight: 200, Headless: headless,
	}
	if *mode == "backdrop" {
		opts.Title = "Backdrop"
		opts.Width, opts.Height = 900, 620
		opts.Decorations = platform.DecorationsNone
	}
	win, err := a.NewWindow(opts)
	if err != nil {
		log.Fatal(err)
	}
	if *mode == "backdrop" {
		win.SetContent(newCard())
	} else {
		win.SetContent(newPanel(win))
	}

	if headless {
		if err := win.WritePNG(*shot); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", *shot)
		return
	}
	a.Post(func() {
		log.Printf("backend=%s mode=%s theme=%s scale=%.2f shaped=%v",
			a.BackendName(), *mode, *theme, win.Scale(), win.ShapeActive())
	})
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

// ---- the window's content ---------------------------------------------------

// panel lays its three buttons out itself, so the demo can put a button in a
// box that is exactly square and let the look decide what shape that is.
//
// It also counts the presses that reach *it* rather than a button, which is
// the whole point: a press in the corner of the disc's box is not the
// button's, so it falls through to here, exactly as a press outside the
// window's silhouette falls through to the desktop.
type panel struct {
	widget.Base
	win            *app.Window
	disc, pill     *widgets.Button
	off            *widgets.Button
	onDisc, onPill int
	onPanel        int
}

func newPanel(win *app.Window) *panel {
	p := &panel{win: win}
	p.Init(p)
	p.SetManagesChildren(true)
	p.disc = widgets.NewButton("●", func() { p.onDisc++; p.Invalidate() })
	p.pill = widgets.NewButton("Play", func() { p.onPill++; p.Invalidate() })
	p.pill.Primary = true
	p.off = widgets.NewButton("Stop", nil)
	p.off.SetEnabled(false)
	p.Add(p.disc)
	p.Add(p.pill)
	p.Add(p.off)
	return p
}

func (p *panel) Arrange(r paintengine2d.Rect) {
	p.SetBounds(r)
	lk := p.Look()
	pad := style.Dip(lk, 16)
	// A square box at the control's own height. Deck's button art is a
	// stadium whose caps are half the control height, so at that size the
	// two caps meet and the face is a circle — at 1.75 as at 1, because the
	// caps are design pixels like everything else in a skin.
	h := lk.Metrics().ControlH
	y := style.Dip(lk, 84)
	p.disc.Arrange(paintengine2d.XYWH(pad, y, h, h))
	x := pad + h + style.Dip(lk, 20)
	p.pill.Arrange(paintengine2d.XYWH(x, y, style.Dip(lk, 110), h))
	p.off.Arrange(paintengine2d.XYWH(x+style.Dip(lk, 126), y, style.Dip(lk, 100), h))
}

// MousePress takes every press the buttons did not — the corners of the
// disc's box among them — and counts it.
func (p *panel) MousePress(widget.MouseEvent) bool {
	p.onPanel++
	p.Invalidate()
	return true
}

func (p *panel) Paint(ctx *paintengine2d.Context) {
	lk, b := p.Look(), p.LocalBounds()
	pal := lk.Palette()
	pad := style.Dip(lk, 16)
	line := func(y float32, text string, col paintengine2d.Color) {
		lk.DrawLabel(ctx, paintengine2d.XYWH(pad, y, max(b.Dx()-2*pad, 0), style.Dip(lk, 22)), text, col, style.AlignStart)
	}
	shaped := "a rectangle"
	if p.win.ShapeActive() {
		shaped = "cut to the look's own outline"
	}
	line(style.Dip(lk, 8), "This window is "+shaped+".", pal.Text)
	line(style.Dip(lk, 30), "Press the corners of the round button: they are not the button.", pal.TextMuted)
	line(style.Dip(lk, 148), fmt.Sprintf("disc %d   ·   play %d   ·   panel %d",
		p.onDisc, p.onPill, p.onPanel), pal.Text)
}

func (p *panel) KeyPress(e widget.KeyEvent) bool {
	if e.Key == platform.KeyEscape {
		p.win.Close()
		return true
	}
	return false
}

// ---- the backdrop -----------------------------------------------------------

// card is a test card to put behind the shaped window: a grid whose lines
// run under the silhouette, so a window dragged across it is obviously
// letting the desktop through rather than painting a picture of it.
//
// It counts the presses it receives, which is the other half of the proof. A
// press beside the shaped window's body is not that window's, and it does
// not stop in mid-air either: it lands here, on the window behind.
type card struct {
	widget.Base
	hits int
}

func newCard() *card {
	c := &card{}
	c.Init(c)
	return c
}

func (c *card) MousePress(widget.MouseEvent) bool {
	c.hits++
	c.Invalidate()
	return true
}

func (c *card) Paint(ctx *paintengine2d.Context) {
	b := c.LocalBounds()
	ctx.DrawRect(b, paintengine2d.Fill(paintengine2d.RGB(0.86, 0.31, 0.16)))
	step := float32(24)
	ink := paintengine2d.Fill(paintengine2d.RGBA(1, 1, 1, 0.35))
	for x := float32(0); x < b.Dx(); x += step {
		ctx.DrawRect(paintengine2d.XYWH(x, 0, 2, b.Dy()), ink)
	}
	for y := float32(0); y < b.Dy(); y += step {
		ctx.DrawRect(paintengine2d.XYWH(0, y, b.Dx(), 2), ink)
	}
	lk := c.Look()
	box := paintengine2d.XYWH(16, b.Dy()-40, b.Dx()-32, 28)
	ctx.DrawRect(box, paintengine2d.Fill(paintengine2d.RGBA(0, 0, 0, 0.55)))
	lk.DrawLabel(ctx, box.Inset(8), fmt.Sprintf("backdrop presses: %d", c.hits),
		paintengine2d.RGB(1, 1, 1), style.AlignStart)
}
