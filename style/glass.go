package style

import (
	"sync/atomic"

	"github.com/codemodify/paintengine2d"
)

// Glass: the desktop blurred behind a window, and the one switch that
// decides whether a look gets it or paints its own approximation.
//
// Three looks have faked glass since they were written — Fluent's acrylic,
// Big Sur's vibrancy and Tahoe's Liquid Glass — because a window can only
// blur its own pixels ([paintengine2d.Context.BackdropBlur]). That is right
// for a menu, which really does sit over the window's own content, and
// wrong for a window background, where there is nothing behind but the
// desktop the window cannot see. Those looks pre-flatten their translucent
// tints over the window colour instead, which is why a Big Sur sidebar is
// a solid grey rather than a pane of glass.
//
// [ext_background_effect_v1] on Wayland and _KDE_NET_WM_BLUR_BEHIND_REGION
// on X11 let the compositor do it properly. Not every desktop can, so both
// paths have to stay, and the choice between them has to be made in exactly
// one place — here — or a look ends up blurring twice on one machine and
// not at all on another.
//
// [ext_background_effect_v1]: https://wayland.app/protocols/ext-background-effect-v1

// glassAvailable is whether this desktop blurs behind a window. The app
// package sets it from the platform capability as windows come and go, and
// every look reads it through [GlassBehind]. It is an atomic because a look
// may be asked to paint from the frame loop while the compositor's answer
// arrives on another: the value is a hint either way, never correctness.
var glassAvailable atomic.Bool

// SetGlassAvailable records whether the desktop can blur behind a window.
// The app package calls it; an app that wants the painted approximation
// everywhere, for a screenshot or a comparison, may call it with false.
func SetGlassAvailable(on bool) { glassAvailable.Store(on) }

// GlassAvailable reports whether the desktop can blur behind a window.
func GlassAvailable() bool { return glassAvailable.Load() }

// GlassBehind reports whether lk's translucent materials are real glass —
// the compositor blurring the desktop behind the window — rather than the
// approximation the look paints for itself.
//
// This is the switch. A look that wants glass says so once, in its
// [DecorationSpec.Glass]; everything else asks here, so the painted
// approximation and the real thing are never both on and never both off.
func GlassBehind(lk LookAndFeel) bool {
	return glassAvailable.Load() && WantsGlass(lk)
}

// WantsGlass reports whether lk asks for glass at all, whatever the desktop
// can do: its decoration says so. It is what a look consults when it needs
// to know its own intent rather than what it is getting.
func WantsGlass(lk LookAndFeel) bool {
	if lk == nil {
		return false
	}
	return DecorationOf(lk, DecorationState{Active: true}).Glass
}

// GlassFrameOnly reports whether lk's glass is its frame's alone
// ([DecorationSpec.GlassFrame]): the window's content stays opaque.
func GlassFrameOnly(lk LookAndFeel) bool {
	if lk == nil {
		return false
	}
	return DecorationOf(lk, DecorationState{Active: true}).GlassFrame
}

// GlassTint is the colour a window's background takes when it is really
// glass: the look's own "glassWindow" token where its pack defines one, and
// otherwise its window background at alpha. The result is always
// translucent — an opaque tint over a blurred desktop shows nothing of the
// blur, which is the commonest way to get this wrong.
//
// alpha is how much of the desktop shows through: 0.6 to 0.85 is where
// Fluent's acrylic, Big Sur's vibrancy and Tahoe's glass each land.
func GlassTint(lk LookAndFeel, alpha float32) paintengine2d.Color {
	if lk == nil {
		return paintengine2d.Transparent
	}
	bg := lk.Palette().Background
	if c, ok := lk.(*Classic); ok {
		bg = c.X("glassWindow", bg)
	}
	if bg.A >= 1 {
		bg = bg.WithAlpha(clamp01(alpha))
	}
	return bg
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

// Backdrop is what lies behind a floating layer — a menu, a flyout, a
// list — that a look paints a translucent material on.
type Backdrop uint8

const (
	// BackdropWindow: the layer is drawn inside its window, over the
	// window's own content, which the look blurs itself
	// ([paintengine2d.Context.BackdropBlur]). What every popup had before
	// popups became surfaces, and what they still have wherever they are
	// not (headless, UITK_POPUPS=layer).
	BackdropWindow Backdrop = iota
	// BackdropGlass: the layer is a surface of its own and the compositor
	// blurs the desktop behind it. There is nothing of the window's to
	// blur; the material keeps its real alpha.
	BackdropGlass
	// BackdropNone: the layer is a surface of its own over a desktop
	// nobody blurs. A translucent tint would show the desktop raw, so the
	// look flattens it over its own background, as it does for a window
	// without glass.
	BackdropNone
)

// layerBackdrop is the backdrop of the layer being painted now. Painting
// happens on the UI goroutine one layer at a time; the app package sets it
// around each popup surface it paints.
var layerBackdrop atomic.Uint32

// SetLayerBackdrop says what lies behind the layer about to be painted and
// returns what was said before, so the caller can put it back.
func SetLayerBackdrop(b Backdrop) Backdrop {
	return Backdrop(layerBackdrop.Swap(uint32(b)))
}

// LayerBackdrop is what lies behind the layer being painted: what a look's
// menu or flyout material asks before it blurs, tints or flattens.
func LayerBackdrop() Backdrop { return Backdrop(layerBackdrop.Load()) }

// FlattenOver is c composited over an opaque bg: a translucent tint made
// solid, what a material is on a backdrop nobody blurs.
func FlattenOver(c, bg paintengine2d.Color) paintengine2d.Color {
	if c.A >= 1 {
		return c
	}
	a := clamp01(c.A)
	return paintengine2d.RGBA(c.R*a+bg.R*(1-a), c.G*a+bg.G*(1-a), c.B*a+bg.B*(1-a), 1)
}

// LayerMaterial is how a look paints a translucent material on a floating
// layer — a menu's vibrancy, a flyout's acrylic — given what is behind the
// layer now: blur is whether to blur the window's own content under it
// (the layer is drawn inside its window), and tint the colour to lay down,
// its real alpha over the compositor's glass, flattened over bg where
// nobody blurs the desktop behind a surface of its own.
func LayerMaterial(tint, bg paintengine2d.Color) (paintengine2d.Color, bool) {
	switch LayerBackdrop() {
	case BackdropGlass:
		return tint, false
	case BackdropNone:
		return FlattenOver(tint, bg), false
	}
	return tint, true
}
