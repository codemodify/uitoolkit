//go:build theme_engine_all || theme_engine_material || theme_engine_material_expressive

package style

import (
	"math"
	"reflect"

	"github.com/codemodify/paintengine2d"
)

// Material 3 tonal palettes, approximated in CIELAB.
//
// Material 3 builds every colour role of a scheme from a handful of tonal
// palettes: one hue and one chroma per palette, sampled at tones 0 (black)
// to 100 (white). Its colour model, HCT, takes hue and chroma from CAM16
// and tone from CIELAB: tone is L*. This file approximates HCT in CIELAB
// LCh(ab): the tone is exact, hue and chroma are the Lab ones. The palette
// rules are the documented ones — primary keeps the seed's hue and at
// least a chroma of 48, secondary keeps the hue at chroma 16, tertiary
// turns the hue by 60° at chroma 24, the neutrals sit at chroma 4 and 8,
// error is a fixed red — with the chromas and the tertiary turn rescaled
// from CAM16 to Lab units so that the baseline seed #6750A4 lands within a
// few ΔE of the published baseline scheme (see the test). A colour that
// does not fit sRGB at its tone gives up chroma until it does, as HCT's
// solver does. This is a from-first-principles implementation of the
// published colour science (sRGB, CIE XYZ D65, CIELAB), not a port.

// mdLab is a CIELAB colour in LCh form: lightness (tone), chroma, hue in
// degrees.

// mdToLab converts an sRGB colour (alpha ignored) to LCh(ab).

// mdLabRGB converts LCh(ab) to linear sRGB (unclamped).

// mdTone is the sRGB colour of tone (L*, 0…100) at hue h and chroma c,
// with the chroma reduced until the colour fits sRGB.

// mdPalette is one tonal palette: a hue and a chroma, sampled by tone.

// mdCore is the set of palettes a seed colour gives.

// mdCorePalette derives the core palettes from a seed colour.

// mdScheme is a Material 3 colour scheme: the colour roles.
type mdScheme struct {
	primary, onPrimary, primaryContainer, onPrimaryContainer         paintengine2d.Color
	secondary, onSecondary, secondaryContainer, onSecondaryContainer paintengine2d.Color
	tertiary, onTertiary, tertiaryContainer, onTertiaryContainer     paintengine2d.Color
	errorC, onError, errorContainer, onErrorContainer                paintengine2d.Color
	background, onBackground, surface, onSurface                     paintengine2d.Color
	surfaceVariant, onSurfaceVariant, outline, outlineVariant        paintengine2d.Color
	inverseSurface, inverseOnSurface, inversePrimary                 paintengine2d.Color
}

// mdSchemeFrom maps the core palettes to the 2021 Material 3 roles, light
// or dark.
func mdSchemeFrom(p mdCore, dark bool) mdScheme {
	// t picks the light or the dark tone of a role.
	t := func(pal mdPalette, light, darkTone float64) paintengine2d.Color {
		if dark {
			return pal.tone(darkTone)
		}
		return pal.tone(light)
	}
	return mdScheme{
		primary: t(p.primary, 40, 80), onPrimary: t(p.primary, 100, 20),
		primaryContainer: t(p.primary, 90, 30), onPrimaryContainer: t(p.primary, 10, 90),
		secondary: t(p.secondary, 40, 80), onSecondary: t(p.secondary, 100, 20),
		secondaryContainer: t(p.secondary, 90, 30), onSecondaryContainer: t(p.secondary, 10, 90),
		tertiary: t(p.tertiary, 40, 80), onTertiary: t(p.tertiary, 100, 20),
		tertiaryContainer: t(p.tertiary, 90, 30), onTertiaryContainer: t(p.tertiary, 10, 90),
		errorC: t(p.err, 40, 80), onError: t(p.err, 100, 20),
		errorContainer: t(p.err, 90, 30), onErrorContainer: t(p.err, 10, 90),
		background: t(p.neutral, 99, 10), onBackground: t(p.neutral, 10, 90),
		surface: t(p.neutral, 99, 10), onSurface: t(p.neutral, 10, 90),
		surfaceVariant: t(p.variant, 90, 30), onSurfaceVariant: t(p.variant, 30, 80),
		outline: t(p.variant, 50, 60), outlineVariant: t(p.variant, 80, 30),
		inverseSurface: t(p.neutral, 20, 90), inverseOnSurface: t(p.neutral, 95, 20),
		inversePrimary: t(p.primary, 80, 40),
	}
}

// mdDeltaE is the CIE76 distance between two colours.
func mdDeltaE(a, b paintengine2d.Color) float64 {
	la, lb := mdToLab(a), mdToLab(b)
	ha, hb := la.h*math.Pi/180, lb.h*math.Pi/180
	da := la.c*math.Cos(ha) - lb.c*math.Cos(hb)
	db := la.c*math.Sin(ha) - lb.c*math.Sin(hb)
	return math.Sqrt((la.l-lb.l)*(la.l-lb.l) + da*da + db*db)
}

// ---- accent shades, shared by the engines that take an accent ------------------------------
//
// Desktops hand a look one accent colour; each platform derives the rest of
// its accent family from it (hover and pressed shades, pale selection
// washes, the text-safe variant). The helpers below work in Oklab (Björn
// Ottosson's published perceptual space, also CSS Color 4's), where
// lightness and hue move independently.

// okLab converts an sRGB colour (alpha ignored) to Oklab.

// okLinear converts Oklab to linear sRGB (unclamped).

// okEncode gamma-encodes linear channels, each clipped to sRGB.

// okClip is the Oklab colour (L, a, b) in sRGB with each channel clipped,
// the way GTK renders a CSS relative colour that leaves the gamut.

// okLCh is c in Oklch: lightness 0…1, chroma, hue in degrees.

// okInGamut reports Oklch (L, C, h) inside sRGB.

// okMaxChroma is the most chroma sRGB holds at Oklab lightness L and hue h.

// okFit is the sRGB colour of Oklch (L, C, h), giving up chroma at that
// lightness and hue until it fits the gamut.

// okTurn is the signed hue step from b to a in degrees (−180 … 180].

// okGrey is the Oklch chroma below which a colour has no hue to speak of.

// accentShift recolours target, a shade that a pack derived from its own
// accent ref (a lighter or darker accent: a hover or pressed face, a
// border, Windows' Dark1), around accent: the step from ref to target,
// taken in Oklch, is taken again from accent. Lightness moves by the same
// share of the room left towards white (or black), chroma scales by the
// same ratio, and a hue turn (a palette's own, as Windows 11 turns its light
// blues towards cyan) applies in full at ref's hue and fades out 90° away
// from it. The result keeps target's alpha, and accentShift(ref, ref,
// target) is target.

// accentWash recolours target, a wash of the pack's accent ref over a grey
// surface (Explorer's pale selection, the accent at 20% over white; a dark
// theme's tinted hover), around accent. It reads how much of ref the wash
// holds — the slope of target's channels against ref's, as an alpha
// composite over a grey has it — and moves the wash by that share of the
// step from ref to accent, so a pale blue over white becomes the same
// pale tint of any accent and a dark wash stays dark. The result keeps
// target's alpha, and accentWash(ref, ref, target) is target.

// accentX is the pack colour key k as its look will read it (Classic.X):
// the pack's "extra" entry, else def.

// accentP is the pack number key k as its look will read it (Classic.P).

// accentDark reports a dark pack, as the engines' colour builders decide.

// accentRepalette is p with every colour that matches (within 1/255) the
// same entry of was replaced by that entry of now: the colours a pack took
// from its old accent follow the new one, and those the pack set itself
// stay.
func accentRepalette(p, was, now Palette) Palette {
	pv := reflect.ValueOf(&p).Elem()
	wv, nv := reflect.ValueOf(was), reflect.ValueOf(now)
	for i := 0; i < pv.NumField(); i++ {
		c, ok := pv.Field(i).Interface().(paintengine2d.Color)
		if ok && accentSame(c, wv.Field(i).Interface().(paintengine2d.Color)) {
			pv.Field(i).Set(nv.Field(i))
		}
	}
	return p
}

// accentSame reports two colours within 1/255 on every channel.
func accentSame(a, b paintengine2d.Color) bool {
	const tol = 1.0 / 255
	d := func(x, y float32) bool { return x-y <= tol && y-x <= tol }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}
