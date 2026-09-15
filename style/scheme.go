package style

import (
	"strings"
	"sync/atomic"
)

// ColorScheme is the desktop's light / dark preference: the setting GTK 4,
// libadwaita, Qt 6 and the browsers follow (the freedesktop portal's
// org.freedesktop.appearance color-scheme on Linux).
type ColorScheme int32

const (
	// SchemeNoPreference: the desktop does not say (or is not asked).
	SchemeNoPreference ColorScheme = iota
	// SchemeDark: the desktop prefers dark.
	SchemeDark
	// SchemeLight: the desktop prefers light.
	SchemeLight
)

// ParseColorScheme accepts dark / light; anything else is no preference.
func ParseColorScheme(s string) ColorScheme {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "dark", "prefer-dark":
		return SchemeDark
	case "light", "prefer-light":
		return SchemeLight
	}
	return SchemeNoPreference
}

func (s ColorScheme) String() string {
	switch s {
	case SchemeDark:
		return "dark"
	case SchemeLight:
		return "light"
	}
	return "no preference"
}

var desktopScheme atomic.Int32

// SetDesktopColorScheme records the desktop's light / dark preference for
// the process. Looks built from an Appearance that follows the desktop
// show the pack's sibling for it; the app package keeps it current.
func SetDesktopColorScheme(s ColorScheme) { desktopScheme.Store(int32(s)) }

// DesktopColorScheme is the preference last recorded by
// [SetDesktopColorScheme].
func DesktopColorScheme() ColorScheme { return ColorScheme(desktopScheme.Load()) }

// darkSiblings and lightSiblings pair packs whose names do not: every CDE
// palette darkens to Charcoal, every Window Maker scheme to Night Sky.
var (
	darkSiblings = map[string]string{
		"light":   "dark",
		"cde":     "cde-charcoal",
		"wmaker":  "wmaker-night",
		"win98":   "win95-dark",
		"win2000": "win95-dark",
		// The web engine's families: each palette's darkest scheme.
		"sourcegit":        "sourcegit-night",
		"catppuccin-latte": "catppuccin-mocha",
		"alucard":          "dracula",
		"tokyonight-day":   "tokyonight",
		"rosepine-dawn":    "rosepine",
	}
	lightSiblings = map[string]string{
		"dark":                 "light",
		"cde-charcoal":         "cde",
		"catppuccin-frappe":    "catppuccin-latte",
		"catppuccin-macchiato": "catppuccin-latte",
		"catppuccin-mocha":     "catppuccin-latte",
		"dracula":              "alucard",
		"tokyonight":           "tokyonight-day",
		"tokyonight-storm":     "tokyonight-day",
		"rosepine":             "rosepine-dawn",
		"rosepine-moon":        "rosepine-dawn",
	}
)

func packIsDark(p ThemePack) bool { return ParseTheme(string(p.Palette)) == ThemeDark }

// SchemeVariant is the pack that shows name in the scheme s: its dark
// sibling for SchemeDark (breeze → breeze-night, win95 → win95-dark,
// luna-olive → luna-night), its light one for SchemeLight (breeze-night →
// breeze, wmaker-night → wmaker-default). A pack already in s, a pack with
// no sibling (Amiga, OS/2 Warp, the high-contrast scheme), an unknown
// name and SchemeNoPreference come back unchanged.
func SchemeVariant(name string, s ColorScheme) string {
	if s == SchemeNoPreference {
		return name
	}
	p, ok := LoadTheme(name)
	if !ok {
		return name
	}
	dark := s == SchemeDark
	if packIsDark(p) == dark {
		return p.Name
	}
	// Try the pack, then its family: luna-olive, then luna.
	for base := p.Name; base != ""; base = trimVariant(base) {
		var cands []string
		if dark {
			cands = []string{darkSiblings[base], base + "-night", base + "-dark"}
		} else {
			cands = []string{lightSiblings[base], base, base + "-default"}
		}
		for _, c := range cands {
			if c == "" || c == p.Name {
				continue
			}
			if q, ok := LoadTheme(c); ok && packIsDark(q) == dark {
				return q.Name
			}
		}
		if !dark {
			// breeze-night → breeze: the suffix is the variant.
			for _, suf := range []string{"-night", "-dark"} {
				if cut, ok := strings.CutSuffix(base, suf); ok && cut != "" {
					if q, ok := LoadTheme(cut); ok && !packIsDark(q) {
						return q.Name
					}
				}
			}
		}
	}
	return p.Name
}

// trimVariant drops the last "-word" of a pack name ("" when none is left).
func trimVariant(name string) string {
	if i := strings.LastIndexByte(name, '-'); i > 0 {
		return name[:i]
	}
	return ""
}

// Effective is a as shown: when it follows the desktop, Name and Theme
// move to the pack's sibling for the desktop's current scheme. The saved
// choice stays in a, so a desktop that turns light again gets it back.
func (a Appearance) Effective() Appearance {
	a = a.Normalize()
	if !a.FollowDesktop {
		return a
	}
	name := SchemeVariant(a.Name, DesktopColorScheme())
	if name == a.Name {
		return a
	}
	if p, ok := LoadTheme(name); ok {
		a.Name, a.Theme = p.Name, p.Palette
	}
	return a
}
