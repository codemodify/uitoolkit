package style

import (
	"os"
	"strings"
	"sync/atomic"
)

// AnimationsEnv decides motion for the process, over every preference,
// and is read live so a test may set it:
//
//	"0"     every state change is instant — reproducible screenshots, tests
//	"1"     animate, whatever anything else says
//	unset   the application decides (the default)
const AnimationsEnv = "UITK_ANIMATIONS"

var reduceMotion, desktopReduce, followDesktopMotion atomic.Bool

// Animations reports whether controls animate: hover fades, the default
// button's pulse, busy bars, transient scroll bars fading out, a
// two-sided panel turning over.
//
// **The toolkit decides this for itself.** It is on unless the
// application turned it off ([SetReduceMotion], which is what an app's
// own settings page writes through look.json) or whoever started the
// program said otherwise (UITK_ANIMATIONS).
//
// The desktop's reduced-motion setting is read and kept current
// ([DesktopReducesMotion]) but is *not* obeyed unless the application
// asks for it with [SetFollowDesktopMotion]. It used to be obeyed always,
// and the cost was plain: a Plasma session with AnimationDurationFactor=0
// — a common thing to set, and reported through the portal as
// reduced-motion — silently turned off every animation in every uitoolkit
// app, with no way for an application to say otherwise even when the
// animation was the thing it existed to show. A toolkit that carries its
// own widgets so it looks the same everywhere should not take its
// behaviour from the session it happens to be started in.
//
// That makes honouring reduced motion the application's call rather than
// the toolkit's. It is a real accessibility setting — it is there for
// people whom movement makes ill — so an application that has no reason
// to do otherwise should pass the desktop's wish on, either with
// SetFollowDesktopMotion(true) or by offering the preference in its own
// settings.
func Animations() bool {
	switch strings.TrimSpace(os.Getenv(AnimationsEnv)) {
	case "0":
		return false
	case "1":
		return true
	}
	if reduceMotion.Load() {
		return false
	}
	return !(followDesktopMotion.Load() && desktopReduce.Load())
}

// Motion is what one thing in an application has to say about moving,
// over the preference that governs the rest.
//
// The preference ([SetReduceMotion], look.json's `reduceMotion`) is the
// right default for motion that decorates: a hover fading in, a scroll
// bar fading out, the default button's pulse. It is the wrong answer for
// motion that *is* the thing — a card that turns over to show it has a
// back, a window that flips to its settings. Switched off there, the
// back face simply appears, and the one cue that said "this is the same
// object, seen from behind" is gone; the application looks broken rather
// than calm, and nothing the application could say made any difference.
//
// So a component, a dialog or a window may state its own:
//
//	MotionAuto    the preference decides — the default, for everything
//	MotionAlways  this motion carries meaning: run it
//	MotionNever   this one does not move, whatever the preference says
//
// [widget.MotionOf] reads it from a component or the nearest ancestor
// that states one, so one call covers a dialog and everything in it.
//
// **MotionAlways is not a way to ignore the preference.** It is a way to
// say "this is content, not decoration", which is the distinction the
// preference is about in the first place. Use it for the motion that
// would leave a hole in the meaning if it went, and leave the rest alone:
// an application that marks everything Always has simply turned the
// preference off for the person who asked for it. UITK_ANIMATIONS=0 still
// beats it, because reproducible screenshots and tests need one switch
// nothing overrides.
type Motion uint8

const (
	// MotionAuto leaves it to the preference. Every component starts here.
	MotionAuto Motion = iota
	// MotionAlways runs this motion whatever the preference says.
	MotionAlways
	// MotionNever keeps this one still, whatever the preference says.
	MotionNever
)

// AnimationsWith is [Animations] for one thing that has stated its own
// [Motion]. MotionAuto is exactly Animations.
func AnimationsWith(m Motion) bool {
	switch strings.TrimSpace(os.Getenv(AnimationsEnv)) {
	case "0":
		return false
	case "1":
		return true
	}
	switch m {
	case MotionAlways:
		return true
	case MotionNever:
		return false
	}
	return Animations()
}

// SetFollowDesktopMotion makes the desktop's reduced-motion setting apply
// to this process, for an application that would rather take the
// session's word for it than decide. Off by default: see [Animations].
func SetFollowDesktopMotion(v bool) { followDesktopMotion.Store(v) }

// FollowsDesktopMotion reports whether the desktop's reduced-motion
// setting applies to this process.
func FollowsDesktopMotion() bool { return followDesktopMotion.Load() }

// SetReduceMotion applies the "reduce motion" preference for the process.
func SetReduceMotion(v bool) { reduceMotion.Store(v) }

// ReduceMotion reports the user's "reduce motion" preference as it was
// last applied — the preference alone, not the desktop's setting or
// UITK_ANIMATIONS (see [Animations] for whether controls animate).
func ReduceMotion() bool { return reduceMotion.Load() }

// SetDesktopReduceMotion records the desktop's reduced-motion setting for
// the process; the app package keeps it current.
func SetDesktopReduceMotion(v bool) { desktopReduce.Store(v) }

// DesktopReducesMotion reports the desktop's reduced-motion setting.
func DesktopReducesMotion() bool { return desktopReduce.Load() }

var nativeDialogs atomic.Bool

// SetNativeDialogs applies the "desktop's file dialogs" preference for the
// process (look.json nativeDialogs); the app package keeps it current.
func SetNativeDialogs(v bool) { nativeDialogs.Store(v) }

// NativeDialogs reports whether file dialogs should be the desktop's own.
func NativeDialogs() bool { return nativeDialogs.Load() }

var comboWheel atomic.Bool

// SetComboWheel applies the "wheel over a closed combo box" preference
// for the process (look.json comboWheel); the app package keeps it
// current.
func SetComboWheel(v bool) { comboWheel.Store(v) }

// ComboWheel reports whether the mouse wheel over a *closed* combo box
// steps its selection, the way Qt's QComboBox and GTK 2's option menu
// did. It is off unless the user asks for it, and that is a considered
// default rather than a timid one: GTK removed the behaviour in GTK 4
// because a wheel over a form is nearly always somebody scrolling the
// page past the form, and a control that changes its value while being
// scrolled past changes it silently — the pointer is over the combo,
// not the thing the eye is reading, and nothing on screen says a value
// moved.
//
// What the preference turns on is therefore deliberately narrow, and
// [ComboBox] spells the rest of it out: a combo that cannot step
// further in the direction of the wheel does not swallow the notch, so
// a list inside a [ScrollView] keeps scrolling when the combo it passes
// over has nowhere left to go. An *open* combo's popup scrolls with the
// wheel whatever this says — that is the list under the pointer doing
// what a list does, and it was never the contentious part.
func ComboWheel() bool { return comboWheel.Load() }
