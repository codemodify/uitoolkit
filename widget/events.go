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

// KeyEvent is keyboard input for the focused component, and for its
// ancestors after that (see [Component.KeyPress]).
//
// Which field a shortcut reads:
//
//   - Key for anything the keyboard *does* — Escape, Tab, Return, the
//     arrows, Home and End, the function keys. These have no Rune.
//   - Rune for a shortcut written as a letter, which is most of them:
//     e.Rune == 's' && e.Mods.Ctrl() is Ctrl+S, and it stays Ctrl+S when
//     the letter is typed with Shift held or under a layout that puts S
//     somewhere else.
//
// Rune is the key's identity as a character, not what typing it produces:
// no modifier has been applied, so Shift+A and A are both 'a' and the
// shift is in Mods. Text being *typed* never arrives here — it comes as
// characters through [Component.TextInput], one per character the layout,
// Shift, the dead keys and the compose key actually produced, which is
// what a text field reads and what an input method drives.
type KeyEvent struct {
	Key platform.Key
	// Rune is the character the key stands for ([platform.KeyChar]), or 0
	// for a key that stands for none.
	Rune rune
	Mods platform.Modifiers
}
