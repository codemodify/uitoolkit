package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Overlay is a dimmed full-window layer with a centered card (dialog pattern).
type Overlay struct {
	widget.Base
	Card    widget.Component
	OnClose func()
	// MinCardW and MinCardH are the smallest the card may be, in 1x
	// design pixels: Arrange scales them by the look (style.Dip) before
	// comparing them with the measured size, which is in device pixels.
	MinCardW float32
	MinCardH float32
	// Modal overlays ignore clicks outside the card (message boxes,
	// dialogs); others close on them (light dismiss).
	Modal bool
	// InitialFocus is focused when the overlay is shown (the default
	// button of a message box); otherwise the first focusable in Card.
	InitialFocus widget.Component
	// OnPresented runs once the overlay is on a window (its look is known).
	OnPresented func()
	prevFocus   widget.Component
}

func NewOverlay(card widget.Component) *Overlay {
	o := &Overlay{Card: card}
	o.Init(o)
	if card != nil {
		o.Add(card)
	}
	return o
}

// Presented traps focus inside the card (widget.Presenter). Without it the
// widget that was focused before the modal opened keeps receiving text and
// Return, so typing "leaks" under the dimmer.
func (o *Overlay) Presented(from widget.Component) {
	o.prevFocus = widget.FocusOwner(from)
	if o.prevFocus == nil {
		o.prevFocus = from
	}
	if o.OnPresented != nil {
		o.OnPresented()
	}
	if f, ok := o.InitialFocus.(interface{ RequestFocus() }); ok && o.InitialFocus.Visible() && o.InitialFocus.Enabled() {
		f.RequestFocus()
		return
	}
	if o.Card != nil {
		widget.FocusFirstIn(o.Card)
	}
}

// FocusAnchor is the component focused before the overlay was shown.
func (o *Overlay) FocusAnchor() widget.Component { return o.prevFocus }

func (o *Overlay) Measure(c layout.Constraints) paintengine2d.Point {
	if c.HasMaxW() && c.HasMaxH() {
		return paintengine2d.Pt(c.MaxW, c.MaxH)
	}
	return paintengine2d.Pt(400, 300)
}

func (o *Overlay) Arrange(r paintengine2d.Rect) {
	o.SetBounds(r)
	if o.Card == nil {
		return
	}
	cs := o.Card.Measure(layout.Loose(r.Dx()*0.8, r.Dy()*0.8))
	// The floors are 1x design pixels; the measured size is device
	// pixels, so they only mean the same thing once scaled.
	lk := o.Look()
	minW, minH := style.Dip(lk, o.MinCardW), style.Dip(lk, o.MinCardH)
	if f := style.Dip(lk, 280); minW < f {
		minW = f
	}
	if f := style.Dip(lk, 140); minH < f {
		minH = f
	}
	// The width is settled first, because the height depends on it.
	if cs.X < minW {
		cs.X = minW
	}
	if cs.X > r.Dx()*0.92 {
		cs.X = r.Dx() * 0.92
	}
	// Height for the width the card is actually getting — Qt's
	// heightForWidth, one level up.
	//
	// The card was measured at 0.8 of the overlay and is arranged at
	// anything from the 280-pixel floor to 0.92 of it, so the two widths
	// routinely differ, and a card narrowed by that cap wraps into more
	// lines than were measured. Its last lines and its buttons were then
	// drawn outside it, where clicks on them missed — a confirmation with
	// a long program path in its text did exactly this. Asking again is
	// the only answer: no measurement made before the width is known can
	// be right about the height.
	if again := o.Card.Measure(layout.Constraints{MaxW: cs.X, MaxH: -1}); again.Y > cs.Y {
		cs.Y = again.Y
	}
	if cs.Y < minH {
		cs.Y = minH
	}
	if cs.Y > r.Dy()*0.88 {
		cs.Y = r.Dy() * 0.88
	}
	x := (r.Dx() - cs.X) * 0.5
	y := (r.Dy() - cs.Y) * 0.5
	o.Card.Arrange(paintengine2d.XYWH(x, y, cs.X, cs.Y))
}

func (o *Overlay) Paint(ctx *paintengine2d.Context) {
	lk := o.Look()
	lk.DrawOverlay(ctx, o.LocalBounds())
	if o.Card != nil && o.Card.Visible() {
		style.DrawPopupShadowOf(lk, ctx, o.Card.Bounds(), style.PopupDialog)
	}
}

func (o *Overlay) MousePress(e widget.MouseEvent) bool {
	if o.Card != nil && o.Card.Bounds().Contains(e.Pos) {
		return false
	}
	if o.Modal {
		// A modal box stays until answered: a stray click (or the second
		// click of the double-click that opened it) must not dismiss it.
		return true
	}
	widget.DismissOverlay(o)
	return true
}

func (o *Overlay) KeyPress(e widget.KeyEvent) bool {
	if e.Key == platform.KeyEscape {
		widget.DismissOverlay(o)
		return true
	}
	return false
}

func (o *Overlay) Dismissed() {
	// Only while the card still owns focus: a dismissal triggered by clicking
	// elsewhere must not steal focus back from what the user just clicked.
	widget.RestoreFocus(o, o.prevFocus, true)
	o.prevFocus = nil
	if o.OnClose != nil {
		o.OnClose()
	}
}

// splitButtonBox finds the one ButtonBox among the actions, and what
// else there is.
func splitButtonBox(actions []widget.Component) (*ButtonBox, []widget.Component) {
	var bb *ButtonBox
	rest := make([]widget.Component, 0, len(actions))
	for _, a := range actions {
		if b, ok := a.(*ButtonBox); ok && bb == nil {
			bb = b
			continue
		}
		rest = append(rest, a)
	}
	return bb, rest
}

// DialogCard is an in-app window with a title, a message and an action
// row; the look paints its frame and caption.
// DialogContent is a dialog's body over its action row, arranged the way
// the platforms' own dialogs are: the body scrolls and the row keeps its
// place at the foot.
//
// An [Overlay] holds its card to the window, and a card taller than that
// used to lay its last lines and its buttons out past its own foot, where
// they could be neither seen nor reached. The Overlay cannot fix it for a
// card an application builds — it is handed one component and cannot tell
// which part of it is the buttons — so this is the piece to build that
// card out of. [DialogCard] is made of it.
//
//	card := widgets.NewPanel("Recovery share", widgets.DialogContent(
//	    widgets.NewColumn(qr, widgets.NewLabel(words)),
//	    cancel, save))
func DialogContent(body widget.Component, actions ...widget.Component) *FlexBox {
	scroll := NewScrollView(body)
	// The card is as tall as what it holds, up to what the window allows.
	//
	// A scroll view offered a height takes all of it, and an Overlay
	// measures its card in 0.8 of the window — so a dialog built of this
	// was that tall whatever it held, and a one-line confirmation came
	// out 560 pixels tall in a 700-pixel window. It scrolls when there is
	// more than there is room for, which is the point, and otherwise it
	// asks for what it has.
	scroll.ShrinkToContent = true

	// One action that lays itself out — a ButtonBox, which arranges a
	// dialog's buttons the way the look's platform does — is given the
	// width rather than squeezed to its natural size at the right end of
	// a row. A ButtonBox put in a justified row loses the whole point of
	// it: its left-hand group, Help and a destructive button under Mac
	// and GNOME looks, is no longer apart from the rest.
	// A ButtonBox anywhere among the actions takes the width, not only
	// one handed over by itself. A dialog's foot often holds something
	// beside the buttons — "Caps Lock is on", a progress note — and that
	// is exactly the foot that most needs the box to lay itself out: the
	// box keeps its left-hand group apart (Help, and a destructive button
	// under Mac and GNOME looks) only if it has the room to.
	var foot widget.Component
	if bb, rest := splitButtonBox(actions); bb != nil {
		if len(rest) == 0 {
			foot = bb
		} else {
			// What else there is on the left, the box filling the rest.
			row := NewRow(append(append([]widget.Component{}, rest...), bb)...).WithGap(8)
			row.AddFlex(bb, 1)
			foot = row
		}
	} else {
		foot = NewRow(actions...).WithGap(8).WithJustify(layout.JustifyEnd)
	}
	col := NewColumn(scroll, foot).WithGap(12).WithPad(4)
	col.AddFlex(scroll, 1)
	return col
}

func DialogCard(title, body string, actions ...widget.Component) *Panel {
	// The body scrolls and the action row does not. An Overlay holds its
	// card to 0.88 of the window, so a card with more text than that
	// used to lay its last lines and its buttons out past its own foot,
	// where they could be neither seen nor reached — a long plan, or a
	// secret with its fingerprint under it, in a small window. The row
	// keeps its place at the foot and the text moves under it.
	msg := NewLabel(body)
	msg.Wrap = true
	col := DialogContent(msg, actions...)
	p := NewPanel(title, col)
	p.Window = true
	p.Raised = true
	p.OnClose = func() { widget.DismissOverlay(p) }
	return p
}
