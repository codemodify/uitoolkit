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

// ---- accent shades, shared by the engines that take an accent ------------------------------
//
// Desktops hand a look one accent colour; each platform derives the rest of
// its accent family from it (hover and pressed shades, pale selection
// washes, the text-safe variant). The helpers below work in Oklab (Björn
// Ottosson's published perceptual space, also CSS Color 4's), where
// lightness and hue move independently.

// okLab converts an sRGB colour (alpha ignored) to Oklab.
func okLab(c paintengine2d.Color) (L, a, b float64) {
	r, g, bl := mdLinear(float64(clamp1(c.R))), mdLinear(float64(clamp1(c.G))), mdLinear(float64(clamp1(c.B)))
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*bl)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*bl)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*bl)
	return 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s
}

// okLinear converts Oklab to linear sRGB (unclamped).
func okLinear(L, a, b float64) (r, g, bl float64) {
	l := L + 0.3963377774*a + 0.2158037573*b
	m := L - 0.1055613458*a - 0.0638541728*b
	s := L - 0.0894841775*a - 1.2914855480*b
	l, m, s = l*l*l, m*m*m, s*s*s
	return 4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s
}

// okEncode gamma-encodes linear channels, each clipped to sRGB.
func okEncode(r, g, b float64) paintengine2d.Color {
	ch := func(v float64) float32 { return float32(mdGamma(math.Min(1, math.Max(0, v)))) }
	return paintengine2d.RGB(ch(r), ch(g), ch(b))
}

// okClip is the Oklab colour (L, a, b) in sRGB with each channel clipped,
// the way GTK renders a CSS relative colour that leaves the gamut.
func okClip(L, a, b float64) paintengine2d.Color { return okEncode(okLinear(L, a, b)) }

// okLCh is c in Oklch: lightness 0…1, chroma, hue in degrees.
func okLCh(c paintengine2d.Color) (L, C, h float64) {
	L, a, b := okLab(c)
	h = math.Atan2(b, a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return L, math.Hypot(a, b), h
}

// okInGamut reports Oklch (L, C, h) inside sRGB.
func okInGamut(L, C, h float64) bool {
	hr := h * math.Pi / 180
	r, g, b := okLinear(L, C*math.Cos(hr), C*math.Sin(hr))
	const eps = 1e-6
	return r >= -eps && r <= 1+eps && g >= -eps && g <= 1+eps && b >= -eps && b <= 1+eps
}

// okMaxChroma is the most chroma sRGB holds at Oklab lightness L and hue h.
func okMaxChroma(L, h float64) float64 {
	lo, hi := 0.0, 0.4
	for i := 0; i < 32; i++ {
		if mid := (lo + hi) * 0.5; okInGamut(L, mid, h) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo
}

// okFit is the sRGB colour of Oklch (L, C, h), giving up chroma at that
// lightness and hue until it fits the gamut.
func okFit(L, C, h float64) paintengine2d.Color {
	L = math.Min(1, math.Max(0, L))
	if C > 0 && !okInGamut(L, C, h) {
		C = okMaxChroma(L, h)
	}
	hr := h * math.Pi / 180
	return okEncode(okLinear(L, C*math.Cos(hr), C*math.Sin(hr)))
}

// okTurn is the signed hue step from b to a in degrees (−180 … 180].
func okTurn(a, b float64) float64 {
	d := math.Mod(a-b+540, 360) - 180
	if d == -180 {
		d = 180
	}
	return d
}

// okGrey is the Oklch chroma below which a colour has no hue to speak of.
const okGrey = 0.02

// accentShift recolours target, a shade that a pack derived from its own
// accent ref (a lighter or darker accent: a hover or pressed face, a
// border, Windows' Dark1), around accent: the step from ref to target,
// taken in Oklch, is taken again from accent. Lightness moves by the same
// share of the room left towards white (or black), chroma scales by the
// same ratio, and a hue turn (a palette's own, as Windows 11 turns its light
// blues towards cyan) applies in full at ref's hue and fades out 90° away
// from it. The result keeps target's alpha, and accentShift(ref, ref,
// target) is target.
func accentShift(accent, ref, target paintengine2d.Color) paintengine2d.Color {
	aL, aC, ah := okLCh(accent)
	rL, rC, rh := okLCh(ref)
	tL, tC, th := okLCh(target)
	L := tL
	switch {
	case tL >= rL && rL < 1:
		L = 1 - (1-aL)*(1-tL)/(1-rL)
	case tL < rL && rL > 0:
		L = aL * tL / rL
	}
	C, h := tC, ah
	if rC > okGrey {
		C = aC * tC / rC
		if w := math.Cos(okTurn(ah, rh) * math.Pi / 180); w > 0 && tC > 0 {
			h = ah + okTurn(th, rh)*w
		}
	}
	c := okFit(L, C, h)
	c.A = target.A
	return c
}

// accentWash recolours target, a wash of the pack's accent ref over a grey
// surface (Explorer's pale selection, the accent at 20% over white; a dark
// theme's tinted hover), around accent. It reads how much of ref the wash
// holds — the slope of target's channels against ref's, as an alpha
// composite over a grey has it — and moves the wash by that share of the
// step from ref to accent, so a pale blue over white becomes the same
// pale tint of any accent and a dark wash stays dark. The result keeps
// target's alpha, and accentWash(ref, ref, target) is target.
func accentWash(accent, ref, target paintengine2d.Color) paintengine2d.Color {
	r := [3]float64{float64(ref.R), float64(ref.G), float64(ref.B)}
	t := [3]float64{float64(target.R), float64(target.G), float64(target.B)}
	a := [3]float64{float64(accent.R), float64(accent.G), float64(accent.B)}
	mr, mt := (r[0]+r[1]+r[2])/3, (t[0]+t[1]+t[2])/3
	var cov, v float64
	for i := range r {
		cov += (r[i] - mr) * (t[i] - mt)
		v += (r[i] - mr) * (r[i] - mr)
	}
	if v < 1e-9 {
		return target
	}
	k := math.Min(1, math.Max(0, cov/v))
	ch := func(i int) float32 { return float32(math.Min(1, math.Max(0, t[i]+k*(a[i]-r[i])))) }
	return paintengine2d.RGBA(ch(0), ch(1), ch(2), target.A)
}

// accentX is the pack colour key k as its look will read it (Classic.X):
// the pack's "extra" entry, else def.
func accentX(tok ThemeTokens, k string, def paintengine2d.Color) paintengine2d.Color {
	if c, ok := tok.Extra[k]; ok {
		return c
	}
	return def
}

// accentP is the pack number key k as its look will read it (Classic.P).
func accentP(tok ThemeTokens, k string, def float32) float32 {
	if v, ok := tok.Params[k]; ok {
		return v
	}
	return def
}

// accentDark reports a dark pack, as the engines' colour builders decide.
func accentDark(tok ThemeTokens) bool { return Luma(tok.Palette.Background) < 0.5 }

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
