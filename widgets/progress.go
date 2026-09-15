package widgets

import (
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
	started       time.Time
	pending       bool
}

// busyPeriod is one cycle of a busy bar's animation; busyFrame how often
// it repaints.
const (
	busyPeriod = 1600 * time.Millisecond
	busyFrame  = 33 * time.Millisecond
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
	h := p.Look().Metrics().ProgressH
	if h <= 0 {
		h = 14
	}
	w := float32(160)
	if c.HasMaxW() && c.MaxW < w {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (p *ProgressBar) Arrange(r paintengine2d.Rect) { p.SetBounds(r) }

func (p *ProgressBar) Paint(ctx *paintengine2d.Context) {
	p.Look().DrawProgressBar(ctx, p.LocalBounds(), p.State(), p.Value, p.Indeterminate, p.phase())
}

// phase is where a busy bar's animation is now, and asks for the next
// frame: the loop runs only while the bar is painted, so a hidden bar costs
// nothing.
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
		widget.After(p, busyFrame, func() {
			p.pending = false
			p.Invalidate()
		})
	}
	ph := p.Phase + t
	return ph - float32(int(ph))
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
