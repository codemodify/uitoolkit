package style

import (
	"math"

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
type mdLab struct{ l, c, h float64 }

// D65 reference white.
const (
	mdXn = 0.95047
	mdYn = 1.0
	mdZn = 1.08883
)

func mdLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func mdGamma(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

func mdLabF(t float64) float64 {
	const e = 216.0 / 24389.0
	const k = 24389.0 / 27.0
	if t > e {
		return math.Cbrt(t)
	}
	return (k*t + 16) / 116
}

func mdLabFInv(f float64) float64 {
	const e = 216.0 / 24389.0
	const k = 24389.0 / 27.0
	if f3 := f * f * f; f3 > e {
		return f3
	}
	return (116*f - 16) / k
}

// mdToLab converts an sRGB colour (alpha ignored) to LCh(ab).
func mdToLab(c paintengine2d.Color) mdLab {
	r, g, b := mdLinear(float64(clamp1(c.R))), mdLinear(float64(clamp1(c.G))), mdLinear(float64(clamp1(c.B)))
	x := 0.4124564*r + 0.3575761*g + 0.1804375*b
	y := 0.2126729*r + 0.7151522*g + 0.0721750*b
	z := 0.0193339*r + 0.1191920*g + 0.9503041*b
	fx, fy, fz := mdLabF(x/mdXn), mdLabF(y/mdYn), mdLabF(z/mdZn)
	L := 116*fy - 16
	a := 500 * (fx - fy)
	bb := 200 * (fy - fz)
	h := math.Atan2(bb, a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return mdLab{l: L, c: math.Hypot(a, bb), h: h}
}

// mdLabRGB converts LCh(ab) to linear sRGB (unclamped).
func mdLabRGB(L, C, H float64) (r, g, b float64) {
	hr := H * math.Pi / 180
	a, bb := C*math.Cos(hr), C*math.Sin(hr)
	fy := (L + 16) / 116
	fx := fy + a/500
	fz := fy - bb/200
	x, y, z := mdXn*mdLabFInv(fx), mdYn*mdLabFInv(fy), mdZn*mdLabFInv(fz)
	r = 3.2404542*x - 1.5371385*y - 0.4985314*z
	g = -0.9692660*x + 1.8760108*y + 0.0415560*z
	b = 0.0556434*x - 0.2040259*y + 1.0572252*z
	return r, g, b
}

func mdInGamut(r, g, b float64) bool {
	const eps = 1e-4
	return r >= -eps && r <= 1+eps && g >= -eps && g <= 1+eps && b >= -eps && b <= 1+eps
}

// mdTone is the sRGB colour of tone (L*, 0…100) at hue h and chroma c,
// with the chroma reduced until the colour fits sRGB.
func mdTone(h, c, tone float64) paintengine2d.Color {
	switch {
	case tone <= 0:
		return paintengine2d.RGB(0, 0, 0)
	case tone >= 100:
		return paintengine2d.RGB(1, 1, 1)
	}
	lo, hi := 0.0, c
	if r, g, b := mdLabRGB(tone, hi, h); mdInGamut(r, g, b) {
		lo = hi
	} else {
		for i := 0; i < 24; i++ {
			mid := (lo + hi) * 0.5
			if r, g, b := mdLabRGB(tone, mid, h); mdInGamut(r, g, b) {
				lo = mid
			} else {
				hi = mid
			}
		}
	}
	r, g, b := mdLabRGB(tone, lo, h)
	ch := func(v float64) float32 {
		return float32(math.Min(1, math.Max(0, mdGamma(math.Min(1, math.Max(0, v))))))
	}
	return paintengine2d.RGB(ch(r), ch(g), ch(b))
}

// mdPalette is one tonal palette: a hue and a chroma, sampled by tone.
type mdPalette struct{ h, c float64 }

func (p mdPalette) tone(t float64) paintengine2d.Color { return mdTone(p.h, p.c, t) }

// The palette rules in Lab units (see the file comment): CAM16's 48, 16,
// 24, 4 and 8 and its 60° tertiary turn, rescaled.
const (
	mdPrimaryChroma   = 48
	mdSecondaryChroma = 14
	mdTertiaryChroma  = 20.5
	mdTertiaryTurn    = 53
	mdNeutralChroma   = 2.8
	mdVariantChroma   = 6.2
	mdErrorHue        = 35
	mdErrorChroma     = 68
)

// mdCore is the set of palettes a seed colour gives.
type mdCore struct {
	primary, secondary, tertiary, neutral, variant, err mdPalette
}

// mdCorePalette derives the core palettes from a seed colour.
func mdCorePalette(seed paintengine2d.Color) mdCore {
	s := mdToLab(seed)
	return mdCore{
		primary:   mdPalette{s.h, math.Max(s.c, mdPrimaryChroma)},
		secondary: mdPalette{s.h, mdSecondaryChroma},
		tertiary:  mdPalette{math.Mod(s.h+mdTertiaryTurn, 360), mdTertiaryChroma},
		neutral:   mdPalette{s.h, mdNeutralChroma},
		variant:   mdPalette{s.h, mdVariantChroma},
		err:       mdPalette{mdErrorHue, mdErrorChroma},
	}
}

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
