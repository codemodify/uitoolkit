package style

// Colour arithmetic that more than one engine needs: the OKLab/OKLCh
// conversions Material 3 derives its tonal palette with, the accent
// shifts several flat looks tint a face by, and the percentage lighten
// and darken that Fusion and the KDE engines share.
//
// None of it belongs to the engine it was first written for. okLab is a
// colour space, not a Material feature, and darkerPct is arithmetic.
// While they lived in engine_material.go and engine_fusion.go, an
// application that wanted neither still had to compile both.
//
// No build tag, so everything here is in every build.

import (
	"math"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// mdOver is c at opacity a over base, flattened.
func mdOver(base, c paintengine2d.Color, a float32) paintengine2d.Color { return Mix(base, c, a) }

// lighterPct returns c with f% of its HSV value (f > 100 lightens).
func lighterPct(c paintengine2d.Color, f float32) paintengine2d.Color {
	h, s, v := hsvOf(c)
	v *= f / 100
	if over := v - 1; over > 0 {
		s = max(0, s-over)
		v = 1
	}
	return hsvColor(h, s, v, c.A)
}

// darkerPct returns c with its HSV value divided by f/100.
func darkerPct(c paintengine2d.Color, f float32) paintengine2d.Color {
	h, s, v := hsvOf(c)
	return hsvColor(h, s, v*100/f, c.A)
}

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

// okTurn is the signed hue step from b to a in degrees (−180 … 180].
func okTurn(a, b float64) float64 {
	d := math.Mod(a-b+540, 360) - 180
	if d == -180 {
		d = 180
	}
	return d
}

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

func mdLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// okEncode gamma-encodes linear channels, each clipped to sRGB.
func okEncode(r, g, b float64) paintengine2d.Color {
	ch := func(v float64) float32 { return float32(mdGamma(math.Min(1, math.Max(0, v)))) }
	return paintengine2d.RGB(ch(r), ch(g), ch(b))
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

// hsvColor builds a colour from hue (degrees), saturation, value and alpha.
func hsvColor(h, s, v, a float32) paintengine2d.Color {
	if s <= 0 {
		return paintengine2d.RGBA(v, v, v, a)
	}
	// Each channel is v less a share of the chroma that depends on how far
	// the hue is from the channel's own primary.
	chroma := v * s
	ch := func(n float32) float32 {
		k := float32(math.Mod(float64(n+h/60), 6))
		d := min(k, 4-k, 1)
		if d < 0 {
			d = 0
		}
		return v - chroma*d
	}
	return paintengine2d.RGBA(ch(5), ch(3), ch(1), a)
}

// hsvOf splits c into hue (degrees), saturation and value (0..1).
func hsvOf(c paintengine2d.Color) (h, s, v float32) {
	r, g, b := clamp1(c.R), clamp1(c.G), clamp1(c.B)
	hi := max(r, g, b)
	lo := min(r, g, b)
	v = hi
	span := hi - lo
	if hi <= 0 || span <= 0 {
		return 0, 0, v
	}
	s = span / hi
	var sector float32
	switch hi {
	case r:
		sector = (g - b) / span
	case g:
		sector = 2 + (b-r)/span
	default:
		sector = 4 + (r-g)/span
	}
	h = sector * 60
	if h < 0 {
		h += 360
	}
	return h, s, v
}

func mdGamma(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// okGrey is the Oklch chroma below which a colour has no hue to speak of.
const okGrey = 0.02

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

// flatLighten is lighten(c, pct) (darken with a negative pct).
func flatLighten(c paintengine2d.Color, pct float64) paintengine2d.Color {
	h, s, l := flatHSL(c)
	return flatFromHSL(h, s, l+pct, c.A)
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

// mdSet is a look's resolved Material colours (built once per look). Both
// generations are described with Material 3's role names.
type mdSet struct {
	m3, dark bool

	primary, onPrimary, primaryC, onPrimaryC         paintengine2d.Color
	secondary, onSecondary, secondaryC, onSecondaryC paintengine2d.Color
	errorC, onError                                  paintengine2d.Color
	bg, surface, onSurface, onSurfaceVar             paintengine2d.Color
	surfaceVar, outline, outlineVar                  paintengine2d.Color
	invSurface, invOnSurface                         paintengine2d.Color

	// Opaque text at the three emphases, over the surface.
	text, text2, textDis paintengine2d.Color
	// Lines: dividers and the resting outline of fields.
	divider, fieldLine paintengine2d.Color
	// Surfaces at elevation: menus, dialogs, cards, bars.
	menu, dialog, card, bar paintengine2d.Color
	tipBg, tipFg            paintengine2d.Color
	// Selection controls: the "on" colour, its mark, the "off" outline.
	ctlOn, ctlMark, ctlOff paintengine2d.Color
	// Row selection fill and its text.
	rowSel, rowSelText paintengine2d.Color
	// Material 2's activated drawer item and its label.
	navSel, navSelText paintengine2d.Color
	// Slider / progress inactive track; the M2 switch thumb when off.
	trackOff, thumbOff paintengine2d.Color
	// State layer opacities.
	aHover, aFocus, aPress float32
	// Faces: the floated label and the button label.
	small, label *Font
}

func flatFromHSL(h, s, l float64, a float32) paintengine2d.Color {
	h = math.Mod(math.Mod(h, 360)+360, 360) / 360
	s, l = math.Min(100, math.Max(0, s))/100, math.Min(100, math.Max(0, l))/100
	if s == 0 {
		v := flat8(l)
		return paintengine2d.RGBA(v, v, v, a)
	}
	q := l + s - l*s
	if l < 0.5 {
		q = l * (1 + s)
	}
	p := 2*l - q
	ch := func(t float64) float32 {
		t = math.Mod(t+1, 1)
		switch {
		case t < 1.0/6:
			return flat8(p + (q-p)*6*t)
		case t < 0.5:
			return flat8(q)
		case t < 2.0/3:
			return flat8(p + (q-p)*(2.0/3-t)*6)
		}
		return flat8(p)
	}
	return paintengine2d.RGBA(ch(h+1.0/3), ch(h), ch(h-1.0/3), a)
}

func flatHSL(c paintengine2d.Color) (h, s, l float64) {
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l * 100
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s * 100, l * 100
}

// mdCore is the set of palettes a seed colour gives.
type mdCore struct {
	primary, secondary, tertiary, neutral, variant, err mdPalette
}

// mdPalette is one tonal palette: a hue and a chroma, sampled by tone.
type mdPalette struct{ h, c float64 }

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

// mdLab is a CIELAB colour in LCh form: lightness (tone), chroma, hue in
// degrees.
type mdLab struct{ l, c, h float64 }

func mdLabF(t float64) float64 {
	const e = 216.0 / 24389.0
	const k = 24389.0 / 27.0
	if t > e {
		return math.Cbrt(t)
	}
	return (k*t + 16) / 116
}

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

// D65 reference white.
const (
	mdXn = 0.95047
	mdYn = 1.0
	mdZn = 1.08883
)

func (p mdPalette) tone(t float64) paintengine2d.Color { return mdTone(p.h, p.c, t) }

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

func mdInGamut(r, g, b float64) bool {
	const eps = 1e-4
	return r >= -eps && r <= 1+eps && g >= -eps && g <= 1+eps && b >= -eps && b <= 1+eps
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

func mdLabFInv(f float64) float64 {
	const e = 216.0 / 24389.0
	const k = 24389.0 / 27.0
	if f3 := f * f * f; f3 > e {
		return f3
	}
	return (116*f - 16) / k
}

// layer is the state-layer opacity of st: the strongest of hover, focus
// and press (Material's layers do not stack).
func (c *mdSet) layer(st ControlState) float32 {
	if st.Disabled() {
		return 0
	}
	var a float32
	if st.Hovered() {
		a = c.aHover
	}
	if st.Focused() {
		a = max(a, c.aFocus)
	}
	if st.Pressed() {
		a = max(a, c.aPress)
	}
	return a
}

// face is the visible body of a push button inside its rect: Material 2
// keeps room around it for the elevation shadow.
func (c *mdSet) face(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = mdSnap(b)
	if c.m3 {
		return mdSnap(paintengine2d.Rect{
			Min: paintengine2d.Pt(b.Min.X+l.S(1), b.Min.Y+l.S(1)),
			Max: paintengine2d.Pt(b.Max.X-l.S(1), b.Max.Y-l.S(2)),
		})
	}
	return mdSnap(paintengine2d.Rect{
		Min: paintengine2d.Pt(b.Min.X+l.S(2), b.Min.Y+l.S(1)),
		Max: paintengine2d.Pt(b.Max.X-l.S(2), b.Max.Y-l.S(3)),
	})
}

// radius is the corner of a control face: 4dp in Material 2, fully round
// in Material 3.
func (c *mdSet) radius(l *Classic, f paintengine2d.Rect) float32 {
	if l.square() {
		return 0
	}
	if c.m3 {
		return min(f.Dx(), f.Dy()) * 0.5
	}
	return l.S(4)
}

// elevate paints the shadow of face (corner r) at elevation e (dp, or an
// M3 level's dp), clipped to clip — the control's own rect, which keeps
// room under the face for it.
func (c *mdSet) elevate(l *Classic, ctx *paintengine2d.Context, face, clip paintengine2d.Rect, r, e float32) {
	if e <= 0 || face.Empty() {
		return
	}
	k := float32(1)
	if c.dark {
		k = 1.8
	}
	ctx.Save()
	ctx.ClipRect(clip)
	for _, s := range mdLevel(c.m3, e) {
		if s.a > 0 {
			DropShadow(ctx, face, r, paintengine2d.RGBA(0, 0, 0, min(s.a*k, 0.6)), 0, l.S(s.dy), l.S(s.blur), l.S(s.spread))
		}
	}
	ctx.Restore()
}

// labelText is a button or tab label: Material 2 sets it in capitals when
// the capitals fit the room (the widget measured the label as written).
func (c *mdSet) labelText(f *Font, s string, room float32) string {
	if c.m3 || s == "" {
		return s
	}
	up := strings.ToUpper(s)
	if f.Advance(up) <= room {
		return up
	}
	return s
}

// button paints a push button face and returns its label colour.
func (c *mdSet) button(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	f := c.face(l, b)
	if f.Dx() < 4 || f.Dy() < 4 {
		return c.text
	}
	r := c.radius(l, f)
	lw := mdPx(l)
	a := c.layer(st)
	dis := st.Disabled()
	plain := int(l.P("plain", 0))
	switch {
	case st.Primary() || (!c.m3 && plain == 2):
		// Contained (M2) / filled (M3).
		fill, fg := c.primary, c.onPrimary
		if !st.Primary() {
			fill, fg = c.surface, c.primary
		}
		if dis {
			ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.onSurface.WithAlpha(0.12)))
			return c.textDis
		}
		e := float32(2)
		if c.m3 {
			e = 0
		}
		switch {
		case st.Pressed():
			e = 8
			if c.m3 {
				e = 0
			}
		case st.Hovered():
			e = 4
			if c.m3 {
				e = 1
			}
		}
		c.elevate(l, ctx, f, b, r, e)
		ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(fill))
		if a > 0 {
			// Content on the primary colour takes a doubled layer (M2).
			k := float32(1)
			if !c.m3 && st.Primary() {
				k = 2
			}
			ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(fg.WithAlpha(min(a*k, 0.32))))
		}
		return fg
	case c.m3 && (plain == 2 || (st.Toggle() && st.Checked())):
		// Filled tonal.
		if dis {
			ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.onSurface.WithAlpha(0.12)))
			return c.textDis
		}
		if st.Hovered() && !st.Pressed() {
			c.elevate(l, ctx, f, b, r, 1)
		}
		ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.secondaryC))
		if a > 0 {
			ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.onSecondaryC.WithAlpha(a)))
		}
		return c.onSecondaryC
	case c.m3 || plain == 1:
		// Outlined.
		edge := c.outline
		if !c.m3 {
			edge = c.divider
		}
		fg := c.primary
		if dis {
			edge, fg = c.onSurface.WithAlpha(0.12), c.textDis
		} else if st.Toggle() && st.Checked() {
			ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.primary.WithAlpha(0.12)))
		}
		if a > 0 {
			ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.primary.WithAlpha(a)))
		}
		if st.Focused() && !dis && c.m3 {
			edge = c.primary
		}
		mdRing(ctx, f, r, lw, edge)
		return fg
	default:
		// Text button (M2): the label alone, the layer under the pointer.
		if dis {
			return c.textDis
		}
		if st.Toggle() && st.Checked() {
			ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.primary.WithAlpha(0.12)))
		}
		if a > 0 {
			ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.primary.WithAlpha(a)))
		}
		return c.primary
	}
}

// field paints the outlined text field box over box and returns it.
func (c *mdSet) field(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, notch0, notch1 float32) {
	if box.Dx() < 4 || box.Dy() < 4 {
		return
	}
	lw := mdPx(l)
	r := min(l.rx(4), box.Dy()*0.5)
	edge := c.fieldLine
	w := lw
	switch {
	case st.Disabled():
		edge = c.onSurface.WithAlpha(0.12)
	case st.Focused() || st.Pressed():
		edge, w = c.primary, 2*lw
	case st.Hovered():
		edge = c.text
	}
	ctx.DrawRoundRect(box, r, r, paintengine2d.Fill(c.surface))
	if notch1 > notch0 {
		// The floated label sits in a gap of the top edge.
		ctx.Save()
		p := paintengine2d.NewPath()
		p.AddRect(box)
		p.AddRect(paintengine2d.Rect{Min: paintengine2d.Pt(notch0, box.Min.Y), Max: paintengine2d.Pt(notch1, box.Min.Y+w+lw)})
		ctx.ClipPathRule(p, paintengine2d.FillEvenOdd)
		mdRing(ctx, box, r, w, edge)
		ctx.Restore()
		return
	}
	mdRing(ctx, box, r, w, edge)
}

// tool paints an icon / tool button face and returns its glyph colour.
func (c *mdSet) tool(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	b = mdSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return c.text2
	}
	r := l.rx(4)
	if c.m3 {
		r = min(b.Dx(), b.Dy()) * 0.5
	}
	fg := c.text2
	switch {
	case st.Disabled():
		return c.textDis
	case st.Checked() && c.m3:
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.secondaryC))
		fg = c.onSecondaryC
	case st.Checked():
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.primary.WithAlpha(0.12)))
		fg = c.primary
	}
	if a := c.layer(st); a > 0 {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.onSurface.WithAlpha(a)))
	}
	return fg
}

// row paints an item row (list, tree, table cell with span) and returns its
// text colour. rad > 0 rounds it (Material 3 pills).
func (c *mdSet) row(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, rad float32) paintengine2d.Color {
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	if st.Checked() {
		fill := c.rowSel
		if st.Backdrop() {
			// An inactive window keeps a neutral selection.
			fill = mdOver(c.surface, c.onSurface, 0.08)
			mdFill(ctx, b, rad, fill)
			return fg
		}
		mdFill(ctx, b, rad, fill)
		if !st.Disabled() {
			fg = c.rowSelText
		}
	}
	a := float32(0)
	if st.Hovered() && !st.Disabled() {
		a = c.aHover
	}
	if st.Pressed() && !st.Disabled() {
		a = c.aPress
	}
	if a > 0 {
		mdFill(ctx, b, rad, c.onSurface.WithAlpha(a))
	}
	return fg
}

// cardR is the corner of cards, views and group boxes: 4dp in Material 2,
// 12dp (medium) in Material 3.
func (c *mdSet) cardR(l *Classic) float32 {
	if c.m3 {
		return l.rx(12)
	}
	return l.rx(4)
}

// cardBox paints a card inside b: elevated (a 1dp shadow inside b) or
// outlined.
func (c *mdSet) cardBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	b = mdSnap(b)
	if b.Dx() < 6 || b.Dy() < 6 {
		return
	}
	r := c.cardR(l)
	if raised {
		f := mdSnap(paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X+l.S(1), b.Min.Y+l.S(1)), Max: paintengine2d.Pt(b.Max.X-l.S(1), b.Max.Y-l.S(2))})
		c.elevate(l, ctx, f, b, r, 1)
		ctx.DrawRoundRect(f, r, r, paintengine2d.Fill(c.card))
		return
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.surface))
	mdRing(ctx, b, r, mdPx(l), c.divider)
}

// drawer is the navigation drawer's container. Material 3's is
// surface-container-low: in the 2021 roles this engine derives it is the
// surface at elevation level 1 (tinted by the primary at 5%), the colour
// the 2023 role took over (#F7F2FA for the baseline seed), a step off the
// surface in both schemes. Material 2's standard drawer is the surface.
func (c *mdSet) drawer() paintengine2d.Color {
	if c.m3 {
		return c.card
	}
	return c.surface
}

// drawerItem is a navigation drawer item's box inside its row and the box's
// corner. Material 3's active indicator is a full-height pill (corner full:
// 28dp on the 56dp item) inset 12dp from the drawer's sides (336dp in a
// 360dp drawer); Material 2's item a 4dp-rounded box inset 8dp from the
// sides and 4dp from the top and bottom (40dp in its 48dp slot).
func (c *mdSet) drawerItem(l *Classic, b paintengine2d.Rect) (paintengine2d.Rect, float32) {
	b = mdSnap(b)
	h, v := snap(l.S(12)), float32(0)
	if !c.m3 {
		h = snap(l.S(8))
		v = snap(min(l.S(4), max((b.Dy()-l.S(24))*0.5, 0)))
	}
	if b.Dx() <= 4*h || b.Dy() <= 2*v+4 {
		return b, 0
	}
	box := paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X+h, b.Min.Y+v), Max: paintengine2d.Pt(b.Max.X-h, b.Max.Y-v)}
	if c.m3 {
		return box, min(l.rx(28), box.Dy()*0.5)
	}
	return box, min(l.rx(4), box.Dy()*0.5)
}

// drawerRow paints a navigation drawer item and returns its box and label
// colour. Material 3: the active item is the secondary-container pill with
// its label on-secondary-container; other labels are on-surface-variant,
// on-surface when hovered, focused or pressed. The state layer — 8%
// hovered, 12% focused or pressed — is on-secondary-container over the
// active item and a pressed one, on-surface elsewhere. Material 2: the
// activated item is the primary at 12% (16% hovered, 24% focused or
// pressed) with its label the primary at 87%; other items take the
// on-surface overlay. An inactive window keeps a neutral selection, as the
// engine's lists do.
func (c *mdSet) drawerRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) (paintengine2d.Rect, paintengine2d.Color) {
	box, r := c.drawerItem(l, b)
	pane := c.drawer()
	dis := st.Disabled()
	fg := c.text
	if c.m3 && !st.Hovered() && !st.Focused() && !st.Pressed() {
		fg = c.onSurfaceVar
	}
	if dis {
		fg = c.textDis
	}
	a := c.layer(st)
	if st.Checked() {
		if st.Backdrop() || dis {
			mdFill(ctx, box, r, mdOver(pane, c.onSurface, 0.08))
			return box, fg
		}
		if c.m3 {
			mdFill(ctx, box, r, c.secondaryC)
			if a > 0 {
				mdFill(ctx, box, r, c.onSecondaryC.WithAlpha(a))
			}
			return box, c.onSecondaryC
		}
		if a <= 0 {
			mdFill(ctx, box, r, c.navSel)
			return box, c.navSelText
		}
		sel := mdOver(pane, c.primary, 0.12+a)
		mdFill(ctx, box, r, sel)
		return box, ReadableOn(sel, 4.5, c.navSelText, c.primary, c.text)
	}
	if a > 0 {
		layer := c.onSurface
		if c.m3 && st.Pressed() {
			layer = c.onSecondaryC
		}
		mdFill(ctx, box, r, layer.WithAlpha(a))
	}
	return box, fg
}

// halo is the round state layer around a selection control.
func (c *mdSet) halo(ctx *paintengine2d.Context, slot, clip paintengine2d.Rect, st ControlState, on bool) {
	a := c.layer(st)
	if a <= 0 || slot.Empty() {
		return
	}
	col := c.onSurface
	if on {
		col = c.ctlOn
	}
	ctx.Save()
	ctx.ClipRect(clip)
	ctr := paintengine2d.Pt((slot.Min.X+slot.Max.X)*0.5, (slot.Min.Y+slot.Max.Y)*0.5)
	ctx.DrawCircle(ctr, min(slot.Dx(), slot.Dy())*0.5, paintengine2d.Fill(col.WithAlpha(a)))
	ctx.Restore()
}

func (c *mdSet) toggleLabel(l *Classic, ctx *paintengine2d.Context, lb paintengine2d.Rect, st ControlState, label string) {
	if label == "" || lb.Dx() <= 0 {
		return
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
}

// fieldBox is the outlined box inside a field rect: it drops by half the
// floated label's height when the field has a label to float.
func (c *mdSet) fieldBox(l *Classic, b paintengine2d.Rect, labelled bool) paintengine2d.Rect {
	b = mdSnap(b)
	if !labelled {
		return b
	}
	top := snap(c.small.Height() * 0.5)
	if b.Dy()-top < c.small.Height()*1.6 {
		return b
	}
	return paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, b.Min.Y+top), Max: b.Max}
}

// rowBox is where a row's selection paints and its corner.
func (c *mdSet) rowBox(l *Classic, b paintengine2d.Rect) (paintengine2d.Rect, float32) {
	b = mdSnap(b)
	if !c.m3 {
		return b, 0
	}
	p := mdRowPill(l, b)
	return p, p.Dy() * 0.5
}

// drawerPad is where a drawer item's label starts inside its box: 16dp into
// Material 3's indicator, past Material 2's 8dp padding.
func (c *mdSet) drawerPad(l *Classic) float32 {
	if c.m3 {
		return l.S(16)
	}
	return l.S(8)
}

// mdFill fills a round rect (radius clamped to the shape).
func mdFill(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, col paintengine2d.Color) {
	if b.Empty() || col.A <= 0 {
		return
	}
	r = min(r, min(b.Dx(), b.Dy())*0.5)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(col))
}

// mdLevel is the shadow of a Material elevation, as layers of offset,
// blur and spread (dp at 1x): Material 2's umbra, penumbra and ambient
// light (dp 1, 2, 4, 8, 24), Material 3's key and ambient light (levels 1
// to 3).
func mdLevel(m3 bool, e float32) [3]mdShadow {
	if m3 {
		switch {
		case e >= 6:
			return [3]mdShadow{{0.3, 1, 3, 0}, {0.15, 4, 8, 3}}
		case e >= 3:
			return [3]mdShadow{{0.3, 1, 2, 0}, {0.15, 2, 6, 2}}
		case e > 0:
			return [3]mdShadow{{0.3, 1, 2, 0}, {0.15, 1, 3, 1}}
		}
		return [3]mdShadow{}
	}
	switch {
	case e >= 24:
		return [3]mdShadow{{0.2, 11, 15, -7}, {0.14, 24, 38, 3}, {0.12, 9, 46, 8}}
	case e >= 8:
		return [3]mdShadow{{0.2, 5, 5, -3}, {0.14, 8, 10, 1}, {0.12, 3, 14, 2}}
	case e >= 4:
		return [3]mdShadow{{0.2, 2, 4, -1}, {0.14, 4, 5, 0}, {0.12, 1, 10, 0}}
	case e >= 2:
		return [3]mdShadow{{0.2, 3, 1, -2}, {0.14, 2, 2, 0}, {0.12, 1, 5, 0}}
	case e > 0:
		return [3]mdShadow{{0.2, 2, 1, -1}, {0.14, 1, 1, 0}, {0.12, 1, 3, 0}}
	}
	return [3]mdShadow{}
}

// mdPx is one device pixel: 1 at 1x, 2 at 2x.
func mdPx(l *Classic) float32 {
	v := float32(math.Round(float64(l.S(1))))
	if v < 1 {
		v = 1
	}
	return v
}

// mdRing fills the lw-wide band just inside the round rect b (one
// even-odd path).
func mdRing(ctx *paintengine2d.Context, b paintengine2d.Rect, r, lw float32, col paintengine2d.Color) {
	if b.Empty() || col.A <= 0 {
		return
	}
	r = min(r, min(b.Dx(), b.Dy())*0.5)
	if b.Dx() <= 2*lw || b.Dy() <= 2*lw {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(col))
		return
	}
	p := paintengine2d.NewPath()
	p.AddRoundRect(b, r, r)
	ri := max(r-lw, 0)
	p.AddRoundRect(b.Inset(lw), ri, ri)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleFill, FillRule: paintengine2d.FillEvenOdd})
}

// mdRowPill is a Material 3 row's pill inside its row rect.
func mdRowPill(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	in := snap(l.S(4))
	return mdSnap(paintengine2d.XYWH(b.Min.X+in, b.Min.Y+snap(l.S(2)), b.Dx()-2*in, b.Dy()-2*snap(l.S(2))))
}

// mdSnap puts b on whole pixels; edges round half down so the rect never
// covers a pixel whose centre lies outside b.
func mdSnap(b paintengine2d.Rect) paintengine2d.Rect {
	f := func(v float32) float32 { return float32(math.Ceil(float64(v) - 0.5 - 1e-3)) }
	x0, y0, x1, y1 := f(b.Min.X), f(b.Min.Y), f(b.Max.X), f(b.Max.Y)
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	return paintengine2d.Rect{Min: paintengine2d.Pt(x0, y0), Max: paintengine2d.Pt(x1, y1)}
}

// mdShadow is one layer of an elevation shadow: opacity, offset, blur and
// spread (dp at 1x).
type mdShadow struct{ a, dy, blur, spread float32 }
