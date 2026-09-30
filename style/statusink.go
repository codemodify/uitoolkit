package style

import "github.com/codemodify/paintengine2d"

// Status colours as ink.
//
// Palette.Danger, Success and Warning are the pack's own colours, and
// they are era-faithful on purpose: Clearlooks really did use #c4a000
// for a warning, KDE 2 really did use #c00000 for an error. They are
// right, and they must not be adjusted, because most of their uses are
// fills and marks where the colour *is* the information and the
// surrounding chrome carries the contrast.
//
// Text is the other case. A word or an icon drawn in one of them on the
// window's own background has nothing behind it to help, and the
// measured answer across the shipped packs is that most of those pairs
// do not reach 4.5:1 — #c4a000 on Clearlooks' #ededed is 2.9:1, and the
// worst pairs are near 1.6:1, which is a colour you can see is there and
// cannot read.
//
// So the toolkit keeps both: the declared colour for fills, and Ink for
// the same colour lifted far enough from the background to be read. An
// application drawing its own status text wants Ink; one filling a badge
// or a bar wants the palette's colour with its own text colour on top.

// MinInkContrast is the contrast ratio [Palette.Ink] lifts a colour to:
// WCAG AA for body text.
const MinInkContrast = 4.5

// Ink is c adjusted to be readable as text or an icon on this palette's
// background, keeping its hue.
//
// A colour that already reaches [MinInkContrast] is returned unchanged,
// which is most of the accent and text colours and a fair number of the
// status ones. The rest are moved along their own lightness — darker on
// a light background, lighter on a dark one — until they reach it or run
// out of room. Hue and saturation are kept, so a lifted red is still
// recognisably that pack's red rather than a generic one.
//
// The alpha of c is preserved and not counted: a translucent colour is
// measured as if it were solid, because what it is drawn over is not
// known here.
func (p Palette) Ink(c paintengine2d.Color) paintengine2d.Color {
	return ReadableInk(c, p.Background)
}

// DangerInk, WarningInk and SuccessInk are the three status colours as
// ink on this palette's background. They are what a status label, a
// message icon or a validation message is drawn in.
func (p Palette) DangerInk() paintengine2d.Color  { return p.Ink(p.Danger) }
func (p Palette) WarningInk() paintengine2d.Color { return p.Ink(p.Warning) }
func (p Palette) SuccessInk() paintengine2d.Color { return p.Ink(p.Success) }

// ReadableInk is want, moved along its own lightness until it reaches
// [MinInkContrast] against bg, or until there is no further to go.
//
// It is the general form of [Palette.Ink], for a caller drawing on
// something other than the window background — a card, a selected row,
// a tooltip.
//
// Where the colour cannot reach the ratio at any lightness (a mid-grey
// background leaves nowhere to go), the better of black and white is
// used: an unreadable coloured word is worse than a readable plain one,
// and this is the case where the pack has left no room for the colour to
// survive.
func ReadableInk(want, bg paintengine2d.Color) paintengine2d.Color {
	if ContrastRatio(want, bg) >= MinInkContrast {
		return want
	}
	h, s, _ := flatHSL(want)
	// Away from the background: darker on a light one, lighter on a dark
	// one. Which direction is decided by the background alone, so two
	// status colours on the same surface move the same way and go on
	// looking like a set.
	dark := RelLuminance(bg) > 0.5
	best, bestRatio := want, ContrastRatio(want, bg)
	// A walk rather than a binary search: contrast against a fixed
	// background is monotonic in lightness on each side of it, but the
	// first lightness that clears the bar is the one that keeps most of
	// the colour, and that is what stepping finds.
	// Lightness in percent, which is what flatHSL gives and flatFromHSL
	// takes. Walking it from 0 to 1 instead put every candidate within
	// one percent of black: on a light background the first already
	// cleared the bar, so the ink came out all but black, and on a dark
	// one none did and the fallback made it white. Every promise this
	// file makes about keeping the pack's own colour was void.
	const steps = 64
	for i := 1; i <= steps; i++ {
		l := 100 * float64(i) / steps
		if dark {
			l = 100 - l
		}
		cand := flatFromHSL(h, s, l, want.A)
		r := ContrastRatio(cand, bg)
		if r > bestRatio {
			best, bestRatio = cand, r
		}
		if r >= MinInkContrast {
			return cand
		}
	}
	// Nowhere on this hue is far enough from the background.
	black := paintengine2d.Color{R: 0, G: 0, B: 0, A: want.A}
	white := paintengine2d.Color{R: 1, G: 1, B: 1, A: want.A}
	plain, plainRatio := black, ContrastRatio(black, bg)
	if r := ContrastRatio(white, bg); r > plainRatio {
		plain, plainRatio = white, r
	}
	if plainRatio > bestRatio {
		return plain
	}
	return best
}
