package platform

// Which modifier a shortcut is held with, and the one place that
// differs by platform.
//
// A menu says "Ctrl+S" on Linux and Windows and "⌘S" on a Mac, and it
// is the same shortcut. The toolkit writes it once — ParseAccel reads
// Cmd and ⌘ as [ModCtrl] already — so what is left is the other half:
// turning the modifiers a *key press* carried into the ones an
// accelerator table means.
//
// It is done here and not in the backend. The AppKit backend reports
// Command as [ModSuper] and Control as [ModCtrl], faithfully, because
// that is what the keyboard did and [Event].Mods is what the keyboard
// did; a backend that handed the toolkit a ModCtrl nobody pressed
// would make every other reader of Mods wrong. The translation is a
// question about what a *shortcut* is, so it happens where shortcuts
// are — once, on the way from the platform to the widgets.

// PrimaryModifier is the modifier a shortcut is held with on this
// platform: Command on macOS, Control everywhere else.
func PrimaryModifier() Modifiers { return primaryModifier }

// AccelMods turns the modifiers a key or button event carried into the
// ones a shortcut is written with, so that a table saying Ctrl+S fires
// on ⌘S.
//
// On macOS the two swap places rather than Command merely becoming
// Control, so they stay distinct: ⌘A is Select All, and a Mac's
// Control+A is the emacs-ism for the start of the line, which should
// not fire it. Everywhere else this is the identity.
func AccelMods(m Modifiers) Modifiers {
	if primaryModifier == ModCtrl {
		return m
	}
	swap := m &^ (ModCtrl | ModSuper)
	if m&ModSuper != 0 {
		swap |= ModCtrl
	}
	if m&ModCtrl != 0 {
		swap |= ModSuper
	}
	return swap
}
