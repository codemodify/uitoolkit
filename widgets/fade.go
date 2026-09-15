package widgets

import (
	"os"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// AnimationsEnv set to "0" makes every state change instant (reduced
// motion, reproducible screenshots).
const AnimationsEnv = "UITK_ANIMATIONS"

// fadeNow is the clock fades run by (tests replace it).
var fadeNow = time.Now

// fadeFrame is how often a running fade repaints.
const fadeFrame = 16 * time.Millisecond

// stateFade cross-fades a control when only its hover or focus changed, for
// the look's style.HintHoverFadeMs: the state it showed is painted, then the
// new one over it as one layer at the fade's progress, eased out (a face of
// many shapes fades as a whole). Presses, checks and everything else switch
// at once.
type stateFade struct {
	from, to style.ControlState
	start    time.Time
	running  bool
	painted  bool
	pending  bool // a repaint is scheduled
}

// paint draws the control in st through draw, cross-fading from the state
// it showed last when the look fades that change.
func (f *stateFade) paint(owner widget.Component, ctx *paintengine2d.Context, st style.ControlState, draw func(*paintengine2d.Context, style.ControlState)) {
	ms := 0
	if owner != nil && os.Getenv(AnimationsEnv) != "0" {
		ms = style.LookHint(owner.Look(), style.HintHoverFadeMs)
	}
	now := fadeNow()
	if st != f.to {
		gentle := (st^f.to)&^(style.StateHovered|style.StateFocused) == 0
		f.running = gentle && f.painted && ms > 0
		f.from, f.to, f.start = f.to, st, now
	}
	f.painted = true
	if !f.running {
		draw(ctx, st)
		return
	}
	t := float32(now.Sub(f.start)) / float32(time.Duration(ms)*time.Millisecond)
	if t >= 1 {
		f.running = false
		draw(ctx, st)
		return
	}
	t = 1 - (1-t)*(1-t) // ease out
	draw(ctx, f.from)
	ctx.DrawLayer(owner.LocalBounds(), t, func(lc *paintengine2d.Context) { draw(lc, st) })
	if !f.pending {
		f.pending = true
		widget.After(owner, fadeFrame, func() {
			f.pending = false
			repaintBar(owner)
		})
	}
}
