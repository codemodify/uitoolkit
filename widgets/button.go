package widgets

import (
	"math"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ButtonPainter paints a button in place of the look's face and reports
// whether it did. It is how an app whose skin draws each key as a picture
// of its own puts that picture on an ordinary button: the button is still
// the component — its focus, its keys, its name in the accessibility tree,
// its tooltip — and the art is only what it looks like. A look the painter
// has nothing for answers false and the look's own face is drawn, so a
// half-skinned app is a coherent app in a real look.
type ButtonPainter func(ctx *paintengine2d.Context, b paintengine2d.Rect, st style.ControlState) bool

// ButtonShaper is the silhouette of what a [ButtonPainter] paints, for a
// box of that size ([widget.ArtShape]): a round key painted as a picture
// takes the pointer on the picture and a press in the corner of its box
// falls through to whatever is behind. Nil, or a nil answer, leaves the
// button the shape of the face it paints.
type ButtonShaper func(lk style.LookAndFeel, size paintengine2d.Point) *style.Silhouette

// Button is a clickable labeled control.
type Button struct {
	widget.Base
	Text    string
	Primary bool
	Tip     string
	OnClick func()
	// Painter and Shaper are the skin's way in: the art, and where the
	// art really is. The focus ring is drawn over a painted button either
	// way — a skin never removes it (docs/skins.md).
	Painter ButtonPainter
	Shaper  ButtonShaper
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

// ShapeRole is the face a button paints: its whole box is the look's push
// button, so a look that gives that face a silhouette — a skin whose button
// art is a disc — decides where the button takes the pointer. Every look
// without one keeps the box, which is every pack in the toolkit.
func (b *Button) ShapeRole() style.Role { return style.RoleButton }

// ArtShape is the silhouette of the picture Painter paints, when the button
// is painted as one. It comes before ShapeRole: the art is what the eye
// sees, so it is what the pointer meets.
func (b *Button) ArtShape(lk style.LookAndFeel, size paintengine2d.Point) *style.Silhouette {
	if b.Shaper == nil {
		return nil
	}
	return b.Shaper(lk, size)
}

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
	w := style.ControlFontOf(lk, style.RoleButton).Advance(b.Text) + m.Pad*2 + style.Dip(lk, 16)
	h := m.ControlH
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (b *Button) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }

func (b *Button) Paint(ctx *paintengine2d.Context) {
	lk, r := b.Look(), b.LocalBounds()
	if paintedAsArt(ctx, lk, r, b.PaintState(), b.Painter) {
		return
	}
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
	if ms <= 0 || !style.Animations() {
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

// paintedAsArt gives a painter the first refusal on a control's box, and
// puts the focus ring back over whatever it drew.
//
// The ring is the one thing the art does not get to decide. It is the same
// rule the skin engine keeps (style/engine_skin.go): a control the keyboard
// is on says so whatever it is wearing, so no picture can ship a keyboard
// trap.
func paintedAsArt(ctx *paintengine2d.Context, lk style.LookAndFeel, r paintengine2d.Rect, st style.ControlState, p ButtonPainter) bool {
	if p == nil || !p(ctx, r, st) {
		return false
	}
	if st.Focused() {
		lk.DrawFocusRing(ctx, r)
	}
	return true
}

// ToolButton is a push button on the look's tool face: the mark that grows
// a face under the pointer on a toolbar, as a component of its own.
//
// ToolBar is the strip of them, and is what a window's toolbar should be.
// This is the loose one — a key on a panel, a control beside a field —
// and it is what an app skinning its own chrome hangs art on, since a
// player's transport row is a row of keys and not a toolbar.
type ToolButton struct {
	widget.Base
	Text string
	Icon style.ToolIcon
	Tip  string
	// Checked draws the button in its on state and says so in the
	// accessibility tree; Toggle makes it a toggle for a screen reader
	// rather than a button that happens to look pressed.
	Checked bool
	Toggle  bool
	OnClick func()
	// Painter and Shaper are as Button's: the art, and where it really is.
	Painter ButtonPainter
	Shaper  ButtonShaper
	hovered bool
	pressed bool
	// outside is set while a press is dragged off the button, exactly as
	// Button tracks it.
	outside bool
	fade    stateFade // hover / focus cross-fade (the look's HintHoverFadeMs)
}

// NewToolButton is a tool-faced button with a label, an icon or both.
func NewToolButton(text string, icon style.ToolIcon, onClick func()) *ToolButton {
	b := &ToolButton{Text: text, Icon: icon, OnClick: onClick}
	b.Init(b)
	b.SetWantsFocus(true)
	b.SetFocusVisibleOnly(true)
	return b
}

func (b *ToolButton) Tooltip() string { return b.Tip }

// ShapeRole is the tool face, so a skin whose tool art is a disc decides
// where the button takes the pointer.
func (b *ToolButton) ShapeRole() style.Role { return style.RoleTool }

// ArtShape is the silhouette of the picture Painter paints, and comes
// before the face's.
func (b *ToolButton) ArtShape(lk style.LookAndFeel, size paintengine2d.Point) *style.Silhouette {
	if b.Shaper == nil {
		return nil
	}
	return b.Shaper(lk, size)
}

// PaintState is the state the face is painted in. A tool button is
// auto-raise: at rest it is a mark on the strip, not a face.
func (b *ToolButton) PaintState() style.ControlState {
	st := b.State() | style.StateAutoRaise
	if b.hovered && !b.outside {
		st |= style.StateHovered
	}
	if b.pressed && !b.outside {
		st |= style.StatePressed
	}
	if b.Checked {
		st |= style.StateChecked
	}
	return st
}

// SetChecked turns the button on or off.
func (b *ToolButton) SetChecked(on bool) {
	if b.Checked == on {
		return
	}
	b.Checked = on
	b.Invalidate()
}

func (b *ToolButton) Measure(c layout.Constraints) paintengine2d.Point {
	lk := b.Look()
	h := lk.Metrics().ControlH
	pad, iconSide, iconGap := style.ToolButtonChromeFor(lk, h)
	w := pad * 2
	if b.Icon != style.IconNone {
		w += iconSide
		if b.Text != "" {
			w += iconGap
		}
	}
	if b.Text != "" {
		w += style.ControlFontOf(lk, style.RoleTool).Advance(b.Text)
	}
	if b.Text == "" && b.Icon == style.IconNone && w < h {
		w = h
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (b *ToolButton) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }

func (b *ToolButton) Paint(ctx *paintengine2d.Context) {
	lk, r := b.Look(), b.LocalBounds()
	st := b.PaintState()
	if paintedAsArt(ctx, lk, r, st, b.Painter) {
		return
	}
	b.fade.paint(b, ctx, r, st, func(ctx *paintengine2d.Context, st style.ControlState) {
		lk.DrawToolButton(ctx, r, st, b.Text, b.Icon)
	})
}

func (b *ToolButton) MouseEnter() { b.hovered = true; b.Base.MouseEnter() }

func (b *ToolButton) MouseExit() {
	b.hovered = false
	b.pressed = false
	b.Base.MouseExit()
}

// MousePress takes the primary button only: a right-click on a key is a
// right-click on whatever the key sits on, and bubbles there.
func (b *ToolButton) MousePress(e widget.MouseEvent) bool {
	if !b.Enabled() || e.Button == platform.ButtonRight {
		return false
	}
	b.MarkPointerFocus()
	b.RequestFocus()
	b.pressed = true
	b.Invalidate()
	return true
}

// MouseMove tracks a press dragged off and back onto the button.
func (b *ToolButton) MouseMove(e widget.MouseEvent) bool {
	if !b.pressed {
		return false
	}
	if out := !b.LocalBounds().Contains(e.Pos); out != b.outside {
		b.outside = out
		b.Invalidate()
	}
	return true
}

func (b *ToolButton) MouseRelease(e widget.MouseEvent) bool {
	was := b.pressed
	b.pressed, b.outside = false, false
	b.Invalidate()
	if was && b.LocalBounds().Contains(e.Pos) && b.Enabled() {
		b.fire()
	}
	return true
}

func (b *ToolButton) KeyPress(e widget.KeyEvent) bool {
	if !b.Enabled() {
		return false
	}
	b.MarkKeyboardFocus()
	if e.Key == platform.KeyReturn || e.Key == platform.KeySpace {
		b.fire()
		return true
	}
	return false
}

func (b *ToolButton) fire() {
	if b.Toggle {
		b.Checked = !b.Checked
		b.Invalidate()
	}
	if b.OnClick != nil {
		b.OnClick()
	}
}
