package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ButtonRole is what a dialog button does, which decides where its look's
// platform puts it.
type ButtonRole int

const (
	// RoleAccept confirms the dialog: OK, Save, Open, Send.
	RoleAccept ButtonRole = iota
	// RoleReject dismisses it: Cancel, Close.
	RoleReject
	// RoleDestructive discards: Don't Save, Delete.
	RoleDestructive
	// RoleAction does something without closing: Apply, Reset.
	RoleAction
	// RoleHelp opens help; it sits at the far left.
	RoleHelp
)

// ButtonBox lays a dialog's buttons out the way its look's platform did
// (Qt's QDialogButtonBox): under Windows and KDE looks the default button
// comes first ("OK  Cancel"), under Mac and GNOME looks last ("Cancel  OK")
// with a destructive button kept apart at the left ("Don't Save"). Help sits
// at the far left everywhere. The order follows the look live, so a theme
// switch moves the buttons too.
type ButtonBox struct {
	widget.Base
	Gap     float32 // between buttons, 1x (0 = 8)
	buttons []boxButton
}

type boxButton struct {
	b    *Button
	role ButtonRole
}

// NewButtonBox is an empty button box; AddButton buttons with their roles.
func NewButtonBox() *ButtonBox {
	bb := &ButtonBox{}
	bb.Init(bb)
	return bb
}

// AddButton appends btn in role and returns the box. The first accept
// button becomes the default (Primary) unless another button is.
func (bb *ButtonBox) AddButton(btn *Button, role ButtonRole) *ButtonBox {
	if btn == nil {
		return bb
	}
	bb.buttons = append(bb.buttons, boxButton{btn, role})
	bb.Base.Add(btn)
	if role == RoleAccept && !bb.hasPrimary() {
		btn.Primary = true
	}
	bb.Invalidate()
	return bb
}

func (bb *ButtonBox) hasPrimary() bool {
	for _, x := range bb.buttons {
		if x.b.Primary {
			return true
		}
	}
	return false
}

// Order is the buttons for the current look: the group at the left (help,
// and a Mac-style destructive button) and the group at the right, each
// left to right.
func (bb *ButtonBox) Order() (left, right []*Button) {
	primaryFirst := style.LookHint(bb.Look(), style.HintDialogPrimaryFirst) == 1
	pick := func(roles ...ButtonRole) []*Button {
		var out []*Button
		for _, r := range roles {
			for _, x := range bb.buttons {
				if x.role == r {
					out = append(out, x.b)
				}
			}
		}
		return out
	}
	if primaryFirst {
		// Windows, KDE: Help … OK, Don't Save, Cancel, Apply.
		return pick(RoleHelp), pick(RoleAccept, RoleDestructive, RoleReject, RoleAction)
	}
	// Mac, GNOME: Help, Don't Save … Apply, Cancel, OK.
	return pick(RoleHelp, RoleDestructive), pick(RoleAction, RoleReject, RoleAccept)
}

func (bb *ButtonBox) gap() float32 {
	g := bb.Gap
	if g <= 0 {
		g = 8
	}
	return style.Dip(bb.Look(), g)
}

func (bb *ButtonBox) Measure(c layout.Constraints) paintengine2d.Point {
	var w, h float32
	child := layout.Constraints{MaxW: c.MaxW, MaxH: c.MaxH}
	for i, x := range bb.buttons {
		if !x.b.Visible() {
			continue
		}
		sz := x.b.Measure(child)
		if i > 0 {
			w += bb.gap()
		}
		w += sz.X
		h = max(h, sz.Y)
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (bb *ButtonBox) Arrange(r paintengine2d.Rect) {
	bb.SetBounds(r)
	b := bb.LocalBounds()
	left, right := bb.Order()
	gap := bb.gap()
	place := func(btns []*Button, x float32, fromRight bool) {
		if fromRight {
			for i := len(btns) - 1; i >= 0; i-- {
				btn := btns[i]
				if !btn.Visible() {
					continue
				}
				w := btn.Measure(layout.Constraints{MaxW: b.Dx(), MaxH: b.Dy()}).X
				x -= w
				btn.Arrange(paintengine2d.XYWH(x, 0, w, b.Dy()))
				x -= gap
			}
			return
		}
		for _, btn := range btns {
			if !btn.Visible() {
				continue
			}
			w := btn.Measure(layout.Constraints{MaxW: b.Dx(), MaxH: b.Dy()}).X
			btn.Arrange(paintengine2d.XYWH(x, 0, w, b.Dy()))
			x += w + gap
		}
	}
	place(left, 0, false)
	place(right, b.Dx(), true)
}

// Paint draws nothing of its own; the buttons paint themselves.
func (bb *ButtonBox) Paint(*paintengine2d.Context) {}
