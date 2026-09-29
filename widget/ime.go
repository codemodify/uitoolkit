package widget

import "github.com/codemodify/paintengine2d"

// IMETarget is implemented by text widgets that show composition.
type IMETarget interface {
	IMEPreedit(s string, caret int)
	IMECommit(s string)
	IMEReset()
	IMEDeleteSurrounding(beforeBytes, afterBytes int)
	IMESurrounding() (text string, cursor, anchor int)
	IMECaretRect() paintengine2d.Rect
}

// SecretTarget is a widget whose contents must not reach an input
// method: a passphrase field.
//
// An input method is a second process that sees every keystroke, learns
// from what it is given and offers it back as a suggestion later. That
// is what makes it useful for the scripts it exists for, and what makes
// it the wrong place for a passphrase. A widget that says yes here gets
// no input method at all while it has the focus — text input is turned
// off on the window, rather than merely hidden — so there is no preedit,
// no candidate window and nothing for the method to remember.
//
// It is separate from [IMETarget] because the two answer different
// questions. A widget that is not an IMETarget shows no composition and
// so gets no input method either; one that is both — a field that would
// otherwise compose, marked secret — is the case this exists for.
type SecretTarget interface {
	// IsSecret reports whether what is typed here is a secret now. It is
	// asked on every focus change, so a field may answer differently as
	// its own mode changes.
	IsSecret() bool
}

// IsSecretTarget reports whether c is a [SecretTarget] that says yes.
func IsSecretTarget(c Component) bool {
	s, ok := c.(SecretTarget)
	return ok && s.IsSecret()
}
