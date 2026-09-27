package style

import (
	"os"
	"sync/atomic"
)

// AnimationsEnv set to "0" makes every state change instant (reproducible
// screenshots, tests), whatever the saved preference says.
const AnimationsEnv = "UITK_ANIMATIONS"

var reduceMotion, desktopReduce atomic.Bool

// Animations reports whether controls animate: hover fades, the default
// button's pulse, busy bars, transient scroll bars fading out. Off with
// UITK_ANIMATIONS=0, the user's "reduce motion" preference (look.json), or
// the desktop's reduced-motion setting (GNOME's and Plasma's animations
// switch, which GTK's gtk-enable-animations follows too).
func Animations() bool {
	return os.Getenv(AnimationsEnv) != "0" && !reduceMotion.Load() && !desktopReduce.Load()
}

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
