package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// IconButton is a push button whose content is a mark: the look's own
// button face, with an icon centred in it and no text.
//
// It is the third case, and the toolkit had only two. [ToolIconBtn] is
// an icon on the *tool* face — flat, with no frame until the pointer is
// over it in most eras — which is right on a tool bar and wrong for a
// button somebody is meant to see is a button. [Button.Icon] is for a
// button that has a name: the engine centres the label, so the icon
// takes a reserved strip at each end, and with no text it leaves an
// empty label centred and the mark off to one side in a button 64 px
// wider than it needs.
//
// So this draws the face through the engine with no label and puts the
// icon in the middle itself, and measures a square of the control
// height. It is what a Fetch, a Write or a Search button on a mail
// client's header is, and what a toolbar-less application uses
// throughout.
//
// **Action is required in practice.** An icon alone cannot be read aloud,
// translated or searched, so it becomes both the tooltip and the
// accessibility name — a11ytest.Audit fails a tree with an unnamed
// interactive node, and this is exactly the control that would trip it
// (docs/contracts.md, rule 2).
type IconButton struct {
	Button
	// Icon is the mark. It is drawn in the label's colour, so it greys
	// with the button and follows the look.
	Icon style.ToolIcon
	// Action is what the button does, in words: its tooltip and the name
	// assistive technology reads. "Fetch", not "Download arrow".
	Action string
	// Pad is the space around the icon inside the button, as a 1x design
	// length. Zero takes the look's own.
	Pad float32
	// Flat draws the mark on the *tool* face instead of the button one:
	// no frame until the pointer is over it, the way a browser's
	// site-information lock, its bookmark star and its three-dot menu
	// are drawn, and every mark that sits inside another control.
	//
	// It is the same control either way — same action, same name, same
	// keyboard — and only the face differs. A button somebody is meant
	// to *see* is a button keeps the default.
	Flat bool
}

// NewIconButton builds a push button showing icon alone, named by name.
func NewIconButton(icon style.ToolIcon, name string, on func()) *IconButton {
	b := &IconButton{Icon: icon, Action: name}
	b.OnClick = on
	b.Tip = name
	b.Init(b)
	b.SetWantsFocus(true)
	b.SetFocusVisibleOnly(true)
	b.SetAccessibleName(name)
	b.Content = b.paintIcon
	return b
}

// Paint draws the look's button face with the mark on it, or the tool
// face when Flat is set.
func (b *IconButton) Paint(ctx *paintengine2d.Context) {
	if !b.Flat {
		b.Button.Paint(ctx)
		return
	}
	lk, r := b.Look(), b.LocalBounds()
	st := b.PaintState() | style.StateAutoRaise
	lk.DrawToolButton(ctx, r, st, "", style.IconNone)
	b.paintIcon(ctx, r, st)
}

// SetAction changes the tooltip and the accessible name together, so
// the two cannot drift apart.
func (b *IconButton) SetAction(name string) {
	b.Action, b.Tip = name, name
	b.SetAccessibleName(name)
}

// Measure is a square of the look's control height: a row of these reads
// as a row of buttons rather than a row of differently-shaped boxes.
func (b *IconButton) Measure(c layout.Constraints) paintengine2d.Point {
	h := b.Look().Metrics().ControlH
	return c.Constrain(paintengine2d.Pt(h, h))
}

// paintIcon draws the mark on the face the engine has already drawn,
// centred, in the colour the label would have been.
func (b *IconButton) paintIcon(ctx *paintengine2d.Context, r paintengine2d.Rect, st style.ControlState) {
	if b.Icon == style.IconNone || r.Empty() {
		return
	}
	lk := b.Look()
	pad := style.Dip(lk, 8)
	if b.Pad > 0 {
		pad = style.Dip(lk, b.Pad)
	}
	side := min(r.Dx(), r.Dy()) - pad*2
	if side <= 0 {
		return
	}
	col := lk.Palette().Text
	if st.Disabled() {
		col = lk.Palette().TextMuted
	}
	box := paintengine2d.XYWH(
		r.Min.X+(r.Dx()-side)*0.5,
		r.Min.Y+(r.Dy()-side)*0.5,
		side, side)
	if st.Pressed() && !st.Disabled() {
		// The mark travels with the face it is on.
		if d := style.Dip(lk, 1); d >= 1 {
			box = box.Translate(paintengine2d.Pt(d, d))
		}
	}
	style.DrawToolIcon(ctx, box, b.Icon, col, style.IconSetOf(lk))
}
