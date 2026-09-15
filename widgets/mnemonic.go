package widgets

import (
	"strings"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// mnemonicShown reports whether c paints its mnemonic underlines: always
// in looks that always showed them, while Alt is held or the keyboard
// drives the menus (keyNav) in looks that hid them (Windows 2000 onward,
// GNOME, Plasma), never in Mac OS and Material looks.
func mnemonicShown(c widget.Component, keyNav bool) bool {
	switch style.LookHint(c.Look(), style.HintMnemonics) {
	case style.MnemonicsNever:
		return false
	case style.MnemonicsOnAlt:
		return keyNav || widget.AltHeld(c)
	}
	return true
}

// ParseMnemonic strips the '&' marker from label text and reports the
// accelerator. "&File" → ("File", KeyF, 0). No marker → (s, KeyUnknown, -1).
// "&&" is an escaped literal ampersand: "Save && Exit" → ("Save & Exit",
// KeyUnknown, -1). index is a rune offset into the returned label, so an
// escaped ampersand before the marker does not shift the underline.
func ParseMnemonic(s string) (label string, key platform.Key, index int) {
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	key = platform.KeyUnknown
	index = -1
	for i := 0; i < len(runes); i++ {
		if runes[i] != '&' {
			out = append(out, runes[i])
			continue
		}
		if i+1 >= len(runes) {
			// Trailing lone '&' is literal text, not a marker.
			out = append(out, '&')
			continue
		}
		if runes[i+1] == '&' {
			out = append(out, '&')
			i++
			continue
		}
		if index < 0 {
			index = len(out)
			key = platform.LetterKey(runes[i+1])
		}
		// Drop the marker; the next iteration appends the marked rune.
	}
	return string(out), key, index
}

// ParseAccel reads a menu shortcut label ("Ctrl+N", "Ctrl+Shift+S",
// "Alt+F4", "F1", "Del") into the key and modifiers that trigger it.
// Cmd / ⌘ read as Ctrl so Mac-style labels still work.
func ParseAccel(s string) (platform.Key, platform.Modifiers, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return platform.KeyUnknown, 0, false
	}
	parts := strings.Split(s, "+")
	var mods platform.Modifiers
	for _, part := range parts[:len(parts)-1] {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "ctrl", "control", "cmd", "command", "⌘":
			mods |= platform.ModCtrl
		case "shift", "⇧":
			mods |= platform.ModShift
		case "alt", "option", "opt", "⌥":
			mods |= platform.ModAlt
		case "super", "meta", "win":
			mods |= platform.ModSuper
		default:
			return platform.KeyUnknown, 0, false
		}
	}
	key := accelKey(strings.TrimSpace(parts[len(parts)-1]))
	if key == platform.KeyUnknown {
		return platform.KeyUnknown, 0, false
	}
	return key, mods, true
}

func accelKey(name string) platform.Key {
	if r := []rune(name); len(r) == 1 {
		if k := platform.LetterKey(r[0]); k != platform.KeyUnknown {
			return k
		}
		switch r[0] {
		case ',':
			return platform.KeyComma
		case '#':
			return platform.KeyHash
		case '3':
			return platform.Key3
		}
		return platform.KeyUnknown
	}
	switch strings.ToLower(name) {
	case "del", "delete":
		return platform.KeyDelete
	case "esc", "escape":
		return platform.KeyEscape
	case "enter", "return":
		return platform.KeyReturn
	case "space":
		return platform.KeySpace
	case "tab":
		return platform.KeyTab
	case "backspace":
		return platform.KeyBackspace
	case "home":
		return platform.KeyHome
	case "end":
		return platform.KeyEnd
	case "pgup", "pageup":
		return platform.KeyPageUp
	case "pgdn", "pagedown":
		return platform.KeyPageDown
	case "left":
		return platform.KeyLeft
	case "right":
		return platform.KeyRight
	case "up":
		return platform.KeyUp
	case "down":
		return platform.KeyDown
	}
	if len(name) >= 2 && (name[0] == 'F' || name[0] == 'f') {
		n := 0
		for _, c := range name[1:] {
			if c < '0' || c > '9' {
				return platform.KeyUnknown
			}
			n = n*10 + int(c-'0')
		}
		if n >= 1 && n <= 12 {
			return platform.KeyF1 + platform.Key(n-1)
		}
	}
	return platform.KeyUnknown
}
