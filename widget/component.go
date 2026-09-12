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

	MousePress(e MouseEvent) bool
	MouseRelease(e MouseEvent) bool
	MouseMove(e MouseEvent) bool
	MouseEnter()
	MouseExit()
	MouseWheel(e MouseEvent) bool
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

// PixelScroller is an optional Host assertion (app.Window). Scrolling
// widgets blit the viewport with paintengine2d.Context.Scroll, then only
// paint the newly exposed strip. Test hosts omit it and fall back to a
// full Invalidate.
type PixelScroller interface {
	ScrollPixels(c Component, local paintengine2d.Rect, dx, dy float32) bool
}

// Self is used so an embedded Base can return the outer Component.
type Self interface {
	Component
}
