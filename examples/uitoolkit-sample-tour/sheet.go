package main

import (
	"fmt"
	"strconv"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/examples/uitoolkit-sample-tour/tourapp"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// sheetCols is how many pages a contact sheet puts side by side.
const sheetCols = 2

// writeSheet puts every page on one contact sheet: each page painted as
// its own still would be, halved, under its name and heading, two to a
// row, in a window of the tour's own look. The sheet is laid out by the
// toolkit, so its captions are the pack's type and colours like the pages.
func writeSheet(a *app.Application, path string, pages []int) error {
	pages = shotPages(pages)
	// Gaps and margins are device pixels to a layout: scale them with the
	// pages so the sheet is one design at every scale.
	dip := max(a.Scale(), 1)
	var cells []widget.Component
	for _, p := range pages {
		img, err := shootPage(a, p)
		if err != nil {
			return err
		}
		name := widgets.NewTitle(tourapp.TourPageNames()[p])
		title := widgets.NewLabel(tourapp.TourPageTitle(p))
		head := widgets.NewRow(name, title).WithGap(10 * dip).WithAlign(layout.AlignCenter)
		cells = append(cells, widgets.NewColumn(head, newThumb(halve(img))).WithGap(6*dip))
	}
	gap := 18 * dip
	grid := widgets.NewGrid()
	grid.ColGap, grid.RowGap = gap, gap
	for i, c := range cells {
		grid.Place(c, i/sheetCols, i%sheetCols)
	}
	ap := a.Appearance()
	label := ap.Name
	if pack, ok := style.LoadTheme(ap.Name); ok {
		label = pack.Display()
	}
	body := widgets.NewPad(gap, grid)
	root := widgets.NewColumn(
		widgets.NewTitleBar("uitoolkit tour", "every page, in "+label+" at "+
			strconv.FormatFloat(float64(a.Scale()), 'f', -1, 32)+"×"),
		body,
	)

	// The sheet is as big as what is on it: lay it out once to measure,
	// then size the window to that.
	win, err := a.NewWindow(platform.WindowOptions{Title: tourapp.WindowTitle("Tour pages"), Width: 1200, Height: 800, Headless: true})
	if err != nil {
		return err
	}
	defer win.Close()
	win.SetContent(root)
	a.PumpOnce()
	// The pages decide the width (the title bar would take any), and the
	// height is what everything needs at that width.
	w := body.Measure(layout.Loose(1e6, 1e6)).X
	h := root.Measure(layout.Loose(w, 1e6)).Y
	sc := win.Scale()
	win.SetSize(platform.LogicalPixels(int(w+0.999), sc), platform.LogicalPixels(int(h+0.999), sc))
	img := win.Capture()
	if img == nil {
		return fmt.Errorf("the sheet has no picture")
	}
	return img.WritePNGFile(path)
}

// halve is img at half its size, each pixel the mean of the four it
// replaces: a box filter, so text and one-pixel rules stay even rather
// than shimmering the way a sampled downscale leaves them.
func halve(img *paintengine2d.Image) *paintengine2d.Image {
	w, h := img.Width/2, img.Height/2
	out := paintengine2d.NewImage(w, h)
	ss, ds := img.RowStride(), out.RowStride()
	for y := 0; y < h; y++ {
		r0, r1 := img.Pix[2*y*ss:], img.Pix[(2*y+1)*ss:]
		d := out.Pix[y*ds:]
		for x := 0; x < w; x++ {
			i := 8 * x
			for k := 0; k < 4; k++ {
				sum := int(r0[i+k]) + int(r0[i+4+k]) + int(r1[i+k]) + int(r1[i+4+k])
				d[4*x+k] = uint8((sum + 2) / 4)
			}
		}
	}
	return out
}

// thumb shows an image at exactly its pixels: one image pixel to one
// device pixel, whatever the display scale, so a halved page is drawn as
// it was halved and not resampled again.
type thumb struct {
	widget.Base
	img *paintengine2d.Image
}

func newThumb(img *paintengine2d.Image) *thumb {
	t := &thumb{img: img}
	t.Init(t)
	return t
}

func (t *thumb) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(float32(t.img.Width), float32(t.img.Height)))
}

func (t *thumb) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

func (t *thumb) Paint(ctx *paintengine2d.Context) {
	w, h := float32(t.img.Width), float32(t.img.Height)
	r := paintengine2d.XYWH(0, 0, w, h)
	ctx.DrawImageRectPaint(t.img, r, r, paintengine2d.Paint{Filter: paintengine2d.FilterNearest})
	// A hairline round the page, in the pack's own border colour, so a
	// page whose background is the sheet's does not run into it.
	ctx.DrawRect(paintengine2d.XYWH(0.5, 0.5, w-1, h-1), paintengine2d.StrokePaint(t.Look().Palette().Border, 1))
}
