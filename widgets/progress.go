package widgets

import (
	"strconv"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ProgressBar is a determinate (0..1) or indeterminate meter. A busy bar
// animates itself while it is on screen (Qt's busy QProgressBar, Win32's
// marquee, Mac OS X's barber pole): Phase is where the animation starts.
// Manual stops that, and the bar shows only the Phase the app sets (GTK's
// pulse). UITK_ANIMATIONS=0 holds every bar still.
type ProgressBar struct {
	widget.Base
	Value         float32
	Indeterminate bool
	Phase         float32
	Manual        bool
	// ShowText writes the progress ("42%") in the bar, or beside it where
	// the look's bar is too thin to hold text (QProgressBar's textVisible,
	// GtkProgressBar's show-text). Format renders it (default "42%").
	ShowText bool
	Format   func(v float32) string
	started  time.Time
	pending  bool
}

func (p *ProgressBar) text() string {
	if p.Format != nil {
		return p.Format(p.Value)
	}
	return strconv.Itoa(int(p.Value*100+0.5)) + "%"
}

// textInside reports whether the look's bar is tall enough to hold the
// text; otherwise it goes beside the bar.
func (p *ProgressBar) textInside() bool {
	return p.barH() >= p.Look().Font().Height()+2
}

func (p *ProgressBar) barH() float32 {
	h := p.Look().Metrics().ProgressH
	if h <= 0 {
		h = 14
	}
	return h
}

// textW is the room kept beside a thin bar for its text.
func (p *ProgressBar) textW() float32 {
	if !p.ShowText || p.Indeterminate || p.textInside() {
		return 0
	}
	f := p.Look().Font()
	return max(f.Advance("100%"), f.Advance(p.text())) + style.Dip(p.Look(), 8)
}

// busyPeriod is one cycle of a busy bar's animation; busyFrame how often
// it repaints; busyHidden how often a bar scrolled out of sight looks
// whether it is back, without repainting.
const (
	busyPeriod = 1600 * time.Millisecond
	busyFrame  = 33 * time.Millisecond
	busyHidden = 500 * time.Millisecond
)

// NewProgressBar builds a determinate bar at value (clamped to 0..1).
func NewProgressBar(value float32) *ProgressBar {
	p := &ProgressBar{Value: clamp01(value)}
	p.Init(p)
	return p
}

// NewBusyBar builds an indeterminate bar.
func NewBusyBar(phase float32) *ProgressBar {
	p := &ProgressBar{Indeterminate: true, Phase: phase}
	p.Init(p)
	return p
}

func (p *ProgressBar) SetValue(v float32) {
	v = clamp01(v)
	if p.Value == v && !p.Indeterminate {
		return
	}
	p.Value = v
	p.Indeterminate = false
	p.Invalidate()
}

func (p *ProgressBar) SetBusy(phase float32) {
	p.Indeterminate = true
	p.Phase = phase
	p.Invalidate()
}

func (p *ProgressBar) Measure(c layout.Constraints) paintengine2d.Point {
	h := p.barH()
	if p.ShowText && !p.Indeterminate && !p.textInside() {
		h = max(h, p.Look().Font().Height())
	}
	w := float32(160) + p.textW()
	if c.HasMaxW() && c.MaxW < w {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (p *ProgressBar) Arrange(r paintengine2d.Rect) { p.SetBounds(r) }

func (p *ProgressBar) Paint(ctx *paintengine2d.Context) {
	lk := p.Look()
	b := p.LocalBounds()
	tw := p.textW()
	bar := b
	if tw > 0 {
		// Beside a thin bar, centred on it.
		bar.Max.X -= tw
		h := p.barH()
		bar.Min.Y = b.Min.Y + (b.Dy()-h)*0.5
		bar.Max.Y = bar.Min.Y + h
	}
	lk.DrawProgressBar(ctx, bar, p.State(), p.Value, p.Indeterminate, p.phase())
	if !p.ShowText || p.Indeterminate {
		return
	}
	f := lk.Font()
	pal := lk.Palette()
	text := p.text()
	col := pal.Text
	if p.State().Disabled() {
		col = pal.TextMuted
	}
	if tw > 0 {
		f.Draw(ctx, text, paintengine2d.Pt(b.Max.X-tw+style.Dip(lk, 8), b.Min.Y+(b.Dy()-f.Height())*0.5), col)
		return
	}
	// In the bar: the text changes colour where the fill ends (Qt's
	// Fusion, Windows classic).
	at := paintengine2d.Pt(bar.Min.X+(bar.Dx()-f.Advance(text))*0.5, bar.Min.Y+(bar.Dy()-f.Height())*0.5)
	fill := bar.Min.X + bar.Dx()*min(max(p.Value, 0), 1)
	ctx.Save()
	ctx.ClipRect(paintengine2d.Rect{Min: bar.Min, Max: paintengine2d.Pt(fill, bar.Max.Y)})
	f.Draw(ctx, text, at, style.ReadableOn(pal.Accent, 3, pal.TextOnAccent, pal.Text))
	ctx.Restore()
	ctx.Save()
	ctx.ClipRect(paintengine2d.Rect{Min: paintengine2d.Pt(fill, bar.Min.Y), Max: bar.Max})
	f.Draw(ctx, text, at, col)
	ctx.Restore()
}

// phase is where a busy bar's animation is now, and asks for the next
// frame: the loop runs only while the bar is painted, so a bar that is not
// in the tree costs nothing.
func (p *ProgressBar) phase() float32 {
	if !p.Indeterminate || p.Manual || !style.Animations() {
		return p.Phase
	}
	now := fadeNow()
	if p.started.IsZero() {
		p.started = now
	}
	t := float32(now.Sub(p.started)%busyPeriod) / float32(busyPeriod)
	if !p.pending {
		p.pending = true
		widget.After(p, busyFrame, p.tick)
	}
	ph := p.Phase + t
	return ph - float32(int(ph))
}

// tick repaints the bar for its next frame — if it can be seen. A scroll
// view records all of its content, so a bar scrolled out of sight is still
// painted, and it kept the gallery repainting 30 times a second behind a
// still window. Out of sight it only looks again now and then, and a
// scroll that brings it back finds it moving within busyHidden.
func (p *ProgressBar) tick() {
	if widget.Exposed(p) {
		p.pending = false
		p.Invalidate()
		return
	}
	widget.After(p, busyHidden, p.tick)
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
