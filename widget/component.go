package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// Component is a retained UI node (JUCE Component analogue).
type Component interface {
	ID() uint64
	Name() string
	SetName(string)

	Parent() Component
	setParent(Component)
	Children() []Component
	Add(child Component)
	Remove(child Component)

	Bounds() paintengine2d.Rect
	SetBounds(paintengine2d.Rect)
	LocalBounds() paintengine2d.Rect

	Visible() bool
	SetVisible(bool)
	Enabled() bool
	SetEnabled(bool)

	WantsFocus() bool
	SetWantsFocus(bool)
	ManagesChildren() bool

	Look() style.LookAndFeel
	SetLook(style.LookAndFeel)

	Host() Host
	SetHost(Host)

	Measure(c layout.Constraints) paintengine2d.Point
	Arrange(r paintengine2d.Rect)
	Paint(ctx *paintengine2d.Context)

	HitTest(local paintengine2d.Point) Component

	// MousePress reports whether the component *took* the press.
	//
	// A press is offered to the deepest component under the pointer and
	// then, while nobody has taken it, to each of its ancestors in turn —
	// the walk the wheel makes from the pointer and keys make from the
	// focus. So a container can act on a press its children ignored (a
	// panel of labels that drags the window) without every child knowing
	// about it.
	//
	// "Took it" is not "noticed it": it means this component is now acting
	// on the gesture and nothing above it should also act. A component
	// that only repaints on a press still returns false, so the container
	// around it stays free to do something with the same press.
	//
	// Whoever takes it becomes the window's pointer capture: every
	// MouseMove until the button goes up, and the MouseRelease, go there
	// wherever the pointer has travelled. When nobody takes it the capture
	// is the deepest component hit, so a widget that ignores presses still
	// hears the release over it.
	//
	// Focus is not part of this. A click focuses the component it landed
	// on, if that one wants focus, and never the ancestor that took the
	// press: dragging a window by its face must not take the keyboard away
	// from the field the user was typing in.
	MousePress(e MouseEvent) bool
	MouseRelease(e MouseEvent) bool
	MouseMove(e MouseEvent) bool
	MouseEnter()
	MouseExit()
	// MouseWheel reports whether the component took the notch; one that
	// did not lets it bubble to its ancestors.
	MouseWheel(e MouseEvent) bool
	// KeyPress reports whether the component took the key; one that did
	// not lets it bubble to its ancestors, and then to the window's
	// accelerators.
	KeyPress(e KeyEvent) bool
	KeyRelease(e KeyEvent) bool
	TextInput(r rune) bool
	FocusGained()
	FocusLost()

	Invalidate()
	InvalidateRect(r paintengine2d.Rect)
}

// Host is implemented by app.Window: damage, focus, scale, layout.
type Host interface {
	Invalidate(c Component, local paintengine2d.Rect)
	RequestFocus(c Component)
	Focus() Component
	Scale() float32
	Look() style.LookAndFeel
	RequestLayout()
}

// AltHeld reports whether the Alt key is held in c's window (hosts that
// track it implement AltHeld() bool); looks that hide mnemonic underlines
// show them then.
func AltHeld(c Component) bool {
	if c == nil {
		return false
	}
	if h, ok := c.Host().(interface{ AltHeld() bool }); ok {
		return h.AltHeld()
	}
	return false
}

// Self is used so an embedded Base can return the outer Component.
type Self interface {
	Component
}
