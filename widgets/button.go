package widgets

import (
	"math"
	"os"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Button is a clickable labeled control.
type Button struct {
	widget.Base
	Text    string
	Primary bool
	Tip     string
	OnClick func()
	hovered bool
	pressed bool
	// outside is set while a press is dragged off the button: it pops up
	// (and releasing there does not click), then re-presses on re-entry —
	// Qt's QAbstractButton / Win32 BUTTON behaviour.
	outside bool
	fade    stateFade // hover / focus cross-fade (the look's HintHoverFadeMs)
	pulsing bool      // a default-button pulse frame is scheduled
}

func (b *Button) Tooltip() string { return b.Tip }

// PaintState is the state the button paints with (pressed only while the
// pointer is over it, hovered only when not dragged off).
func (b *Button) PaintState() style.ControlState {
	st := b.State()
	if b.hovered && !b.outside {
		st |= style.StateHovered
	}
	if b.pressed && !b.outside {
		st |= style.StatePressed
	}
	if b.Primary {
		st |= style.StatePrimary
	}
	return st
}

func NewButton(text string, onClick func()) *Button {
	b := &Button{Text: text, OnClick: onClick}
	b.Init(b)
	b.SetWantsFocus(true)
	b.SetFocusVisibleOnly(true)
	return b
}

func (b *Button) Measure(c layout.Constraints) paintengine2d.Point {
	lk := b.Look()
	m := lk.Metrics()
	w := style.ControlFontOf(lk, style.RoleButton).Advance(b.Text) + m.Pad*2 + 16
	h := m.ControlH
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (b *Button) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }

func (b *Button) Paint(ctx *paintengine2d.Context) {
	lk, r := b.Look(), b.LocalBounds()
	b.fade.paint(b, ctx, r, b.PaintState(), func(ctx *paintengine2d.Context, st style.ControlState) {
		if p, ok := b.pulse(st); ok {
			// The default button swells toward its hover look and back.
			ctx.DrawCrossFade(r, p,
				func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, r, st, b.Text) },
				func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, r, st|style.StateHovered, b.Text) })
			return
		}
		lk.DrawButton(ctx, r, st, b.Text)
	})
}

// pulseFrame is how often a pulsing default button repaints.
const pulseFrame = 40 * time.Millisecond

// pulse is how far the default button has swollen toward its hover look,
// for looks whose default button pulses (style.HintDefaultPulseMs), and
// asks for the next frame. Only a resting default button in an active
// window pulses.
func (b *Button) pulse(st style.ControlState) (float32, bool) {
	if !st.Primary() || st&(style.StateHovered|style.StatePressed|style.StateDisabled|style.StateBackdrop) != 0 {
		return 0, false
	}
	ms := style.LookHint(b.Look(), style.HintDefaultPulseMs)
	if ms <= 0 || os.Getenv(AnimationsEnv) == "0" {
		return 0, false
	}
	period := time.Duration(ms) * time.Millisecond
	t := float64(fadeNow().UnixNano()%int64(period)) / float64(period)
	if !b.pulsing {
		b.pulsing = true
		widget.After(b, pulseFrame, func() {
			b.pulsing = false
			b.Invalidate()
		})
	}
	return float32(0.5-0.5*math.Cos(2*math.Pi*t)) * 0.8, true
}

func (b *Button) MouseEnter() { b.hovered = true; b.Base.MouseEnter() }
func (b *Button) MouseExit() {
	b.hovered = false
	b.pressed = false
	b.Base.MouseExit()
}

func (b *Button) MousePress(e widget.MouseEvent) bool {
	if !b.Enabled() {
		return false
	}
	b.MarkPointerFocus()
	b.RequestFocus()
	b.pressed = true
	b.Invalidate()
	return true
}

// MouseMove tracks a press dragged off and back onto the button.
func (b *Button) MouseMove(e widget.MouseEvent) bool {
	if !b.pressed {
		return false
	}
	out := !b.LocalBounds().Contains(e.Pos)
	if out != b.outside {
		b.outside = out
		b.Invalidate()
	}
	return true
}

func (b *Button) MouseRelease(e widget.MouseEvent) bool {
	was := b.pressed
	b.pressed = false
	b.outside = false
	b.Invalidate()
	if was && b.LocalBounds().Contains(e.Pos) && b.OnClick != nil && b.Enabled() {
		b.OnClick()
	}
	return true
}

func (b *Button) KeyPress(e widget.KeyEvent) bool {
	if !b.Enabled() {
		return false
	}
	b.MarkKeyboardFocus()
	if e.Key == platform.KeyReturn || e.Key == platform.KeySpace {
		if b.OnClick != nil && b.Enabled() {
			b.OnClick()
		}
		return true
	}
	return false
}
