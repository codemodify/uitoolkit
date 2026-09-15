package widgets

import (
	"fmt"
	"math"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ColorButton shows a colour and drops a picker down — GtkColorButton,
// QColorDialog's button, WPF toolkits' ColorPicker: a palette of swatches
// and, for anything else, a saturation / value square with a hue strip.
type ColorButton struct {
	widget.Base
	Color    paintengine2d.Color
	OnChange func(paintengine2d.Color)
	pressed  bool
	open     bool
}

// NewColorButton shows col.
func NewColorButton(col paintengine2d.Color, on func(paintengine2d.Color)) *ColorButton {
	b := &ColorButton{Color: col, OnChange: on}
	b.Init(b)
	b.SetWantsFocus(true)
	return b
}

// ColorHex is the colour as #rrggbb.
func ColorHex(c paintengine2d.Color) string {
	to8 := func(v float32) int {
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		return int(v*255 + 0.5)
	}
	return fmt.Sprintf("#%02x%02x%02x", to8(c.R), to8(c.G), to8(c.B))
}

func (b *ColorButton) Measure(c layout.Constraints) paintengine2d.Point {
	lk := b.Look()
	m := lk.Metrics()
	w := style.ControlFontOf(lk, style.RoleButton).Advance("#000000") + m.Pad*2 + style.Dip(lk, 40)
	return c.Constrain(paintengine2d.Pt(w, m.ControlH))
}

func (b *ColorButton) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }

func (b *ColorButton) Paint(ctx *paintengine2d.Context) {
	lk := b.Look()
	st := b.State()
	if b.pressed || b.open {
		st |= style.StatePressed
	}
	r := b.LocalBounds()
	// The look draws the button and centres the hex label; the swatch sits
	// just before it.
	label := ColorHex(b.Color)
	lk.DrawButton(ctx, r, st, label)
	f := style.ControlFontOf(lk, style.RoleButton)
	side := f.Height() * 0.9
	gap := style.Dip(lk, 6)
	x := r.Min.X + (r.Dx()-f.Advance(label))*0.5 - gap - side
	if x < r.Min.X+style.Dip(lk, 6) {
		x = r.Min.X + style.Dip(lk, 6)
	}
	sw := paintengine2d.XYWH(x, r.Min.Y+(r.Dy()-side)*0.5, side, side)
	drawSwatch(ctx, lk, sw, b.Color)
}

// drawSwatch paints a colour chip with a frame that reads on any ground.
func drawSwatch(ctx *paintengine2d.Context, lk style.LookAndFeel, r paintengine2d.Rect, col paintengine2d.Color) {
	rad := lk.Metrics().RadiusSmall * 0.5
	ctx.DrawRoundRect(r, rad, rad, paintengine2d.Fill(col))
	ctx.DrawRoundRect(r.Inset(0.5), rad, rad, paintengine2d.StrokePaint(lk.Palette().Text.WithAlpha(0.35), 1))
}

func (b *ColorButton) set(c paintengine2d.Color) {
	c.A = 1
	if c == b.Color {
		return
	}
	b.Color = c
	b.Invalidate()
	if b.OnChange != nil {
		b.OnChange(c)
	}
}

func (b *ColorButton) MousePress(e widget.MouseEvent) bool {
	if e.Button != platform.ButtonLeft {
		return false
	}
	b.RequestFocus()
	b.pressed = true
	b.Invalidate()
	return true
}

func (b *ColorButton) MouseRelease(e widget.MouseEvent) bool {
	was := b.pressed
	b.pressed = false
	b.Invalidate()
	if was && b.LocalBounds().Contains(e.Pos) {
		if b.open {
			b.Close()
		} else {
			b.Open()
		}
	}
	return true
}

func (b *ColorButton) KeyPress(e widget.KeyEvent) bool {
	switch e.Key {
	case platform.KeySpace, platform.KeyReturn, platform.KeyF4:
		b.Open()
		return true
	case platform.KeyDown:
		if e.Mods.Alt() {
			b.Open()
			return true
		}
	}
	return false
}

// Open drops the picker down.
func (b *ColorButton) Open() {
	if b.open {
		return
	}
	pop := newColorPopup(b)
	o := widget.DeviceOrigin(b)
	lb := b.LocalBounds()
	widget.PlacePopupForAnchor(b, pop, paintengine2d.XYWH(o.X, o.Y, lb.Dx(), lb.Dy()), 0, 2)
	if widget.ShowPopup(b, pop) {
		b.open = true
		b.Invalidate()
		pop.RequestFocus()
	}
}

// Close takes the picker down.
func (b *ColorButton) Close() {
	if !b.open {
		return
	}
	b.open = false
	widget.DismissPopup(b)
	b.Invalidate()
}

// colorPalette is the swatch grid: nine hues in five tones, then greys (the
// GNOME palette's layout).
var colorPalette = func() []paintengine2d.Color {
	hues := []float64{0, 30, 50, 90, 150, 190, 215, 260, 320}
	tones := [][2]float64{{0.35, 0.95}, {0.6, 0.95}, {0.85, 0.85}, {0.85, 0.6}, {0.85, 0.38}}
	var out []paintengine2d.Color
	for _, t := range tones {
		for _, h := range hues {
			out = append(out, hsv(h, t[0], t[1]))
		}
	}
	for i := 0; i < 9; i++ {
		v := float32(i) / 8
		out = append(out, paintengine2d.RGB(v, v, v))
	}
	return out
}()

const paletteCols = 9

// hsv converts hue (degrees), saturation and value (0..1) to a colour.
func hsv(h, s, v float64) paintengine2d.Color {
	h = math.Mod(math.Mod(h, 360)+360, 360) / 60
	i := math.Floor(h)
	f := h - i
	p, q, t := v*(1-s), v*(1-s*f), v*(1-s*(1-f))
	var r, g, b float64
	switch int(i) {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}
	return paintengine2d.RGB(float32(r), float32(g), float32(b))
}

// toHSV is the inverse of hsv.
func toHSV(c paintengine2d.Color) (h, s, v float64) {
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	v = mx
	d := mx - mn
	if mx > 0 {
		s = d / mx
	}
	if d == 0 {
		return 0, s, v
	}
	switch mx {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, s, v
}

// colorPopup is the drop-down picker.
type colorPopup struct {
	widget.Base
	btn     *ColorButton
	h, s, v float64
	hot     int // swatch under the pointer, or -1
	drag    int // 1 dragging the square, 2 the hue strip
}

func newColorPopup(b *ColorButton) *colorPopup {
	p := &colorPopup{btn: b, hot: -1}
	p.h, p.s, p.v = toHSV(b.Color)
	p.Init(p)
	p.SetWantsFocus(true)
	return p
}

// geometry lays the popup out: swatch cells, the square and the strip.
func (p *colorPopup) geometry() (pad, cell float32, grid, square, strip paintengine2d.Rect) {
	lk := p.Look()
	pad = style.Dip(lk, 8)
	cell = style.Dip(lk, 22)
	rows := (len(colorPalette) + paletteCols - 1) / paletteCols
	grid = paintengine2d.XYWH(pad, pad, cell*paletteCols, cell*float32(rows))
	square = paintengine2d.XYWH(pad, grid.Max.Y+pad, grid.Dx(), style.Dip(lk, 110))
	strip = paintengine2d.XYWH(pad, square.Max.Y+pad*0.75, grid.Dx(), style.Dip(lk, 14))
	return
}

func (p *colorPopup) Measure(c layout.Constraints) paintengine2d.Point {
	pad, _, grid, _, strip := p.geometry()
	return c.Constrain(paintengine2d.Pt(grid.Dx()+pad*2, strip.Max.Y+pad))
}

func (p *colorPopup) Arrange(r paintengine2d.Rect) { p.SetBounds(r) }

func (p *colorPopup) Paint(ctx *paintengine2d.Context) {
	lk := p.Look()
	lk.DrawMenuFrame(ctx, p.LocalBounds())
	_, cell, grid, square, strip := p.geometry()
	cur := p.btn.Color
	for i, col := range colorPalette {
		r := paintengine2d.XYWH(grid.Min.X+float32(i%paletteCols)*cell, grid.Min.Y+float32(i/paletteCols)*cell, cell, cell).Inset(2)
		drawSwatch(ctx, lk, r, col)
		if i == p.hot || ColorHex(col) == ColorHex(cur) {
			w := style.Dip(lk, 2)
			ctx.DrawRect(r.Inset(-w*0.5-1), paintengine2d.StrokePaint(lk.Palette().Text, w))
		}
	}
	// Saturation left to right, value top to bottom, over the hue.
	hue := hsv(p.h, 1, 1)
	ctx.DrawRect(square, style.HGradient(square, style.Stop(0, paintengine2d.RGB(1, 1, 1)), style.Stop(1, hue)))
	ctx.DrawRect(square, style.VGradient(square, style.Stop(0, paintengine2d.RGBA(0, 0, 0, 0)), style.Stop(1, paintengine2d.RGB(0, 0, 0))))
	mx := square.Min.X + float32(p.s)*square.Dx()
	my := square.Min.Y + float32(1-p.v)*square.Dy()
	ring := style.Dip(lk, 5)
	ctx.DrawCircle(paintengine2d.Pt(mx, my), ring, paintengine2d.StrokePaint(paintengine2d.RGB(1, 1, 1), 2))
	ctx.DrawCircle(paintengine2d.Pt(mx, my), ring+1.5, paintengine2d.StrokePaint(paintengine2d.RGBA(0, 0, 0, 0.6), 1))
	var stops []paintengine2d.GradientStop
	for i := 0; i <= 6; i++ {
		stops = append(stops, style.Stop(float32(i)/6, hsv(float64(i)*60, 1, 1)))
	}
	ctx.DrawRect(strip, style.HGradient(strip, stops...))
	hx := strip.Min.X + float32(p.h/360)*strip.Dx()
	ctx.DrawRect(paintengine2d.XYWH(hx-2, strip.Min.Y-2, 4, strip.Dy()+4), paintengine2d.StrokePaint(lk.Palette().Text, 1.5))
}

func (p *colorPopup) swatchAt(pt paintengine2d.Point) int {
	_, cell, grid, _, _ := p.geometry()
	if !grid.Contains(pt) {
		return -1
	}
	i := int((pt.Y-grid.Min.Y)/cell)*paletteCols + int((pt.X-grid.Min.X)/cell)
	if i < 0 || i >= len(colorPalette) {
		return -1
	}
	return i
}

// track applies a drag at pt to the square or the strip.
func (p *colorPopup) track(pt paintengine2d.Point) {
	_, _, _, square, strip := p.geometry()
	clamp := func(v float32) float64 { return math.Max(0, math.Min(1, float64(v))) }
	switch p.drag {
	case 1:
		p.s = clamp((pt.X - square.Min.X) / square.Dx())
		p.v = 1 - clamp((pt.Y-square.Min.Y)/square.Dy())
	case 2:
		p.h = clamp((pt.X-strip.Min.X)/strip.Dx()) * 360
	default:
		return
	}
	p.btn.set(hsv(p.h, p.s, p.v))
	p.Invalidate()
}

func (p *colorPopup) MouseMove(e widget.MouseEvent) bool {
	if p.drag != 0 {
		p.track(e.Pos)
		return true
	}
	if h := p.swatchAt(e.Pos); h != p.hot {
		p.hot = h
		p.Invalidate()
	}
	return true
}

func (p *colorPopup) MousePress(e widget.MouseEvent) bool {
	_, _, _, square, strip := p.geometry()
	switch {
	case square.Contains(e.Pos):
		p.drag = 1
	case strip.Inset(-4).Contains(e.Pos):
		p.drag = 2
	default:
		if i := p.swatchAt(e.Pos); i >= 0 {
			p.btn.set(colorPalette[i])
			p.btn.Close()
		}
		return true
	}
	p.track(e.Pos)
	return true
}

func (p *colorPopup) MouseRelease(e widget.MouseEvent) bool {
	p.drag = 0
	return true
}

func (p *colorPopup) KeyPress(e widget.KeyEvent) bool {
	switch e.Key {
	case platform.KeyEscape, platform.KeyReturn:
		p.btn.Close()
		return true
	}
	return false
}

// Dismissed hands focus back to the button and clears its open state.
func (p *colorPopup) Dismissed() {
	if p.btn.open {
		p.btn.open = false
		p.btn.Invalidate()
	}
	widget.RestoreFocus(p, p.btn, true)
}
