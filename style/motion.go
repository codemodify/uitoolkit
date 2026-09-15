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
