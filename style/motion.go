package style

import (
	"os"
	"sync/atomic"
)

// AnimationsEnv set to "0" makes every state change instant (reproducible
// screenshots, tests), whatever the saved preference says.
const AnimationsEnv = "UITK_ANIMATIONS"

var reduceMotion atomic.Bool

// Animations reports whether controls animate: hover fades, the default
// button's pulse, busy bars, transient scroll bars fading out. Off with
// UITK_ANIMATIONS=0 or the user's "reduce motion" preference (look.json,
// like GTK's gtk-enable-animations).
func Animations() bool {
	return os.Getenv(AnimationsEnv) != "0" && !reduceMotion.Load()
}

// SetReduceMotion applies the "reduce motion" preference for the process.
func SetReduceMotion(v bool) { reduceMotion.Store(v) }
