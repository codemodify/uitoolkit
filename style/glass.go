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
