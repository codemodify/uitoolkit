package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
)

// MouseEvent is pointer input in the receiver's local coordinates.
type MouseEvent struct {
	Pos    paintengine2d.Point
	Button platform.MouseButton
	Scroll paintengine2d.Point
	// Precise: Scroll is in device pixels (a touchpad); otherwise it
	// counts wheel notches.
	Precise bool
	Mods    platform.Modifiers
}

// KeyEvent is keyboard input for the focused component.
type KeyEvent struct {
	Key  platform.Key
	Rune rune
	Mods platform.Modifiers
}
