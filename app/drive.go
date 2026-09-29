package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// Driving a headless window from a test.
//
// A headless application is already drivable — [Window.Inject] takes a
// [platform.Event] and [Application.PumpOnce] runs a frame — and the
// toolkit's own suite has been built out of those two since the
// beginning. What was missing is the three lines everyone writes on top
// of them, which is why an application's tests kept reimplementing key
// events by hand and getting the modifier conventions subtly wrong.
//
// Each of these injects what the backend would have sent and pumps a
// frame, so the effect has happened by the time the call returns:
//
//	a := uitoolkit.New(uitoolkit.Options{Headless: true})
//	w, _ := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
//	w.SetContent(form)
//	a.PumpOnce()
//
//	w.FocusOn(field)
//	w.Type("correct horse")
//	w.Press(platform.KeyReturn)
//	w.ClickComponent(okButton)
//
// They work on a window with a real surface too, where they are the
// application driving itself rather than a test driving it, and
// [Window.Inject]'s own note applies: the event goes to this window's
// dispatch, not to the desktop.

// Type enters s as the user would: one text event per rune, then a
// frame. It is what a keyboard produces, not [widgets.TextField.SetText]
// — the field's own input handling runs, so OnInput fires, an Accept
// rejects what it would reject, and a secret field's buffer takes the
// same path it takes from a real key.
func (w *Window) Type(s string) {
	if w == nil || w.Closed() {
		return
	}
	for _, r := range s {
		w.Inject(platform.Event{Kind: platform.EventText, Rune: r})
	}
	w.pumpOnce()
}

// Press sends one key down and up with the given modifiers, then a
// frame. Press(platform.KeyA, platform.ModCtrl) is Ctrl+A.
func (w *Window) Press(key platform.Key, mods ...platform.Modifiers) {
	if w == nil || w.Closed() {
		return
	}
	var m platform.Modifiers
	for _, one := range mods {
		m |= one
	}
	w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: key, Mods: m})
	w.Inject(platform.Event{Kind: platform.EventKeyUp, Key: key, Mods: m})
	w.pumpOnce()
}

// ClickComponent presses and releases the left button in the middle of
// c, then runs a frame. c must be laid out — it is a component of this
// window's content, after a pump — because the click goes to the point
// its box is at, the way a real one does: through the window's hit
// testing, so a widget covered by a popup is not clicked and one that is
// disabled hears the press and does nothing.
//
// It reports whether there was a box to click.
func (w *Window) ClickComponent(c widget.Component) bool {
	if w == nil || w.Closed() || c == nil {
		return false
	}
	// Scroll to it first, as a person does: a click is aimed at something
	// the user can see. A button below a ScrollView's fold is at
	// coordinates outside the view, so clicking where its box says it is
	// pressed whatever happens to be there — or nothing — and the test
	// read as a button that did not work.
	w.reveal(c)
	b := c.Bounds()
	if b.Empty() {
		return false
	}
	o := widget.DeviceOrigin(c)
	at := paintengine2d.Pt(o.X+b.Dx()/2, o.Y+b.Dy()/2)
	return w.ClickAt(at)
}

// reveal scrolls every enclosing ScrollView so c is in view, and lays the
// window out again so the bounds read afterwards are the ones it now has.
func (w *Window) reveal(c widget.Component) {
	if c == nil {
		return
	}
	widget.RevealFocus(c)
	w.laid = false
	w.pumpOnce()
}

// ClickAt presses and releases the left button at a point in the
// window's own pixels, then runs a frame.
func (w *Window) ClickAt(at paintengine2d.Point) bool {
	if w == nil || w.Closed() {
		return false
	}
	w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: at})
	w.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: at, Button: platform.ButtonLeft})
	w.Inject(platform.Event{Kind: platform.EventMouseUp, Pos: at, Button: platform.ButtonLeft})
	w.pumpOnce()
	return true
}

// FocusOn puts the keyboard on c, then runs a frame — the Tab a test
// would otherwise have to count. ([Window.Focus] is the getter: what
// holds the keyboard now.)
func (w *Window) FocusOn(c widget.Component) {
	if w == nil || w.Closed() {
		return
	}
	w.RequestFocus(c)
	w.reveal(c)
	w.pumpOnce()
}

// pumpOnce runs a frame where there is an application to run it. A
// window made without one still delivers the event; it simply does not
// repaint, which is what a caller driving a detached window has already
// accepted.
func (w *Window) pumpOnce() {
	if w != nil && w.app != nil {
		w.app.PumpOnce()
	}
}
