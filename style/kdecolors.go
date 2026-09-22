package style

import (
	"fmt"
	"math"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// A KDE colour scheme for the desktop's own window frame. Where the desktop
// draws a window's title bar (KWin's Breeze frame, the user's "Use system
// title bar and borders"), KWin lets each window name a colour scheme for
// it — org_kde_kwin_server_decoration_palette on Wayland,
// _KDE_NET_WM_COLOR_SCHEME on X11 — which is how a KDE app's title bar
// follows the app's own colours. [KDEColorScheme] writes one from a look:
// the title bar in the look's own caption colours, so a Luna window under
// KWin wears a blue Breeze title bar, and the rest from its palette.
//
// The format is KDE's documented colour-scheme file (a KConfig INI file,
// "*.colors"): a [General] group naming it, one [Colors:<set>] group per
// colour set — Window, View, Button, Selection, Tooltip, Complementary and
// Header — holding the set's twelve roles as "r,g,b", a [<set>][Inactive]
// group where a set differs in an inactive window, and the [WM] group of
// title-bar colours that decorations without a Header set read. Nothing
// here is taken from KDE's own schemes: every colour is the look's.

// CaptionColors are a look's caption band and window title, active and in
// the backdrop, as opaque colours.
type CaptionColors struct {
	Active, Inactive         paintengine2d.Color
	ActiveText, InactiveText paintengine2d.Color
}

// CaptionColorsOf is lk's caption colours as its own frame paints them: the
// band's average colour and its title's ink, read off a frame painted
// offscreen. Reading them rather than asking every engine means a look's
// answer is its picture — Luna's gradient, Aqua's pinstripes, a skin's
// art — and a translucent caption (Aero's glass) is taken over the look's
// window background, the nearest thing a solid title bar has to what shows
// through it.
func CaptionColorsOf(lk LookAndFeel) CaptionColors {
	if lk == nil {
		return CaptionColors{}
	}
	var out CaptionColors
	out.Active, out.ActiveText = captionColors(lk, true)
	out.Inactive, out.InactiveText = captionColors(lk, false)
	return out
}

// captionColors paints lk's frame in the active or the backdrop state and
// reads the caption band's colour and its title's.
func captionColors(lk LookAndFeel, active bool) (band, text paintengine2d.Color) {
	pal := lk.Palette()
	whole := func(v float32) float32 { return float32(math.Round(float64(v))) }
	st := DecorationState{Active: active}
	spec := DecorationOf(lk, st)
	win := paintengine2d.XYWH(0, 0, whole(Dip(lk, 480)), whole(Dip(lk, 200)))
	inner := spec.Border.Apply(win)
	h := spec.Caption
	if !spec.Stacked {
		h = max(h, whole(Dip(lk, 32)))
	}
	h = max(whole(h), 4)
	f := DecorationFrame{Window: win, Caption: paintengine2d.XYWH(inner.Min.X, inner.Min.Y, inner.Dx(), h)}
	img := paintengine2d.NewImage(int(win.Dx()), int(win.Dy()))
	DrawDecorationOf(lk, paintengine2d.NewContext(img), f, st)
	// The band less a few pixels at its edges, where a frame puts its
	// bevels and outlines rather than its colour.
	in := whole(Dip(lk, 3))
	area := paintengine2d.Rect{
		Min: paintengine2d.Pt(f.Caption.Min.X+in, f.Caption.Min.Y+in),
		Max: paintengine2d.Pt(f.Caption.Max.X-in, f.Caption.Max.Y-in),
	}
	if area.Dx() < 1 || area.Dy() < 1 {
		area = f.Caption
	}
	band = over(averageColor(img, area), pal.Background)
	// The title, in the band's middle: its ink is the pixel it changed the
	// most. Anti-aliased edges, a glow (Aero) or a drop shadow (Luna) all
	// change a pixel less than the solid core of a letter does.
	before := img.Clone()
	tb := paintengine2d.XYWH(f.Caption.Min.X+f.Caption.Dx()*0.25, f.Caption.Min.Y, f.Caption.Dx()*0.5, f.Caption.Dy())
	DrawCaptionTitleOf(lk, paintengine2d.NewContext(img), tb, "Wm Hamburgefonts", st)
	best := float32(-1)
	text = pal.Text
	for y := int(tb.Min.Y); y < int(tb.Max.Y) && y < img.Height; y++ {
		for x := int(tb.Min.X); x < int(tb.Max.X) && x < img.Width; x++ {
			a, b := imageColorAt(img, x, y), imageColorAt(before, x, y)
			if d := colorDistance(over(a, band), over(b, band)); d > best {
				best, text = d, over(a, band)
			}
		}
	}
	if best < 0.1 {
		// No title drawn (a look whose frame carries none): the palette's
		// text on it, or its opposite where that would not read.
		text = readableOn(band, pal.Text)
	}
	return band, text
}

// averageColor is the mean colour of r's pixels, alpha-weighted, with its
// mean alpha.
func averageColor(img *paintengine2d.Image, r paintengine2d.Rect) paintengine2d.Color {
	var sr, sg, sb, sa float64
	n := 0
	for y := max(int(r.Min.Y), 0); y < int(r.Max.Y) && y < img.Height; y++ {
		for x := max(int(r.Min.X), 0); x < int(r.Max.X) && x < img.Width; x++ {
			p := img.NRGBAAt(x, y)
			a := float64(p.A) / 255
			sr += float64(p.R) / 255 * a
			sg += float64(p.G) / 255 * a
			sb += float64(p.B) / 255 * a
			sa += a
			n++
		}
	}
	if n == 0 || sa <= 0 {
		return paintengine2d.Color{}
	}
	return paintengine2d.RGBA(float32(sr/sa), float32(sg/sa), float32(sb/sa), float32(sa/float64(n)))
}

// imageColorAt is pixel (x, y) of img as a colour.
func imageColorAt(img *paintengine2d.Image, x, y int) paintengine2d.Color {
	p := img.NRGBAAt(x, y)
	return paintengine2d.RGBA(float32(p.R)/255, float32(p.G)/255, float32(p.B)/255, float32(p.A)/255)
}

// over is c composited over the opaque bg.
func over(c, bg paintengine2d.Color) paintengine2d.Color {
	a := max(min(c.A, 1), 0)
	return paintengine2d.RGB(c.R*a+bg.R*(1-a), c.G*a+bg.G*(1-a), c.B*a+bg.B*(1-a))
}

func colorDistance(a, b paintengine2d.Color) float32 {
	dr, dg, db := a.R-b.R, a.G-b.G, a.B-b.B
	return float32(math.Sqrt(float64(dr*dr + dg*dg + db*db)))
}

// luma is c's relative lightness, 0 to 1.
func luma(c paintengine2d.Color) float32 { return 0.3*c.R + 0.59*c.G + 0.11*c.B }

// readableOn is want where it reads on bg, else black or white, whichever
// does.
func readableOn(bg, want paintengine2d.Color) paintengine2d.Color {
	if d := luma(want) - luma(bg); d > 0.35 || d < -0.35 {
		return want
	}
	if luma(bg) > 0.5 {
		return paintengine2d.RGB(0, 0, 0)
	}
	return paintengine2d.RGB(1, 1, 1)
}

// KDEColorSchemeSets are the colour sets a KDE colour scheme holds, in the
// order KDE writes them, and KDEColorRoles the roles every set holds.
var (
	KDEColorSchemeSets = []string{"Window", "View", "Button", "Selection", "Tooltip", "Complementary", "Header"}
	KDEColorRoles      = []string{
		"BackgroundNormal", "BackgroundAlternate",
		"ForegroundNormal", "ForegroundInactive", "ForegroundActive",
		"ForegroundLink", "ForegroundVisited",
		"ForegroundNegative", "ForegroundNeutral", "ForegroundPositive",
		"DecorationFocus", "DecorationHover",
	}
	// KDEWMKeys are the [WM] group's title-bar colours.
	KDEWMKeys = []string{
		"activeBackground", "activeBlend", "activeForeground",
		"inactiveBackground", "inactiveBlend", "inactiveForeground",
		"frame", "inactiveFrame",
	}
)

// KDEColorScheme is a KDE colour-scheme file for lk, named name: the title
// bar in lk's caption colours ([CaptionColorsOf]) — the Header set and the
// [WM] group, active and inactive — and every other set from lk's palette.
func KDEColorScheme(lk LookAndFeel, name string) []byte {
	return kdeColorScheme(lk.Palette(), CaptionColorsOf(lk), name)
}

// kdeColorScheme writes the scheme from a palette and caption colours.
func kdeColorScheme(p Palette, cap CaptionColors, name string) []byte {
	bg := p.Background
	solid := func(c paintengine2d.Color, on paintengine2d.Color) paintengine2d.Color { return over(c, on) }
	field := solid(p.Field, bg)
	surface := solid(p.Surface, bg)
	alt := solid(p.SurfaceAlt, bg)
	text := solid(p.Text, bg)
	accent := solid(p.Accent, bg)
	focus := accent
	if p.Focus.A > 0 {
		focus = solid(p.Focus, bg)
	}
	hover := solid(p.AccentHover, bg)
	if p.AccentHover.A <= 0 {
		hover = accent
	}
	sel := solid(p.Selection, field)
	if p.Selection.A <= 0 {
		sel = accent
	}
	onSel := readableOn(sel, solid(p.TextOnAccent, sel))
	type set struct {
		name                          string
		bg, alt, fg, inactive, active paintengine2d.Color
	}
	roles := func(s set) []paintengine2d.Color {
		return []paintengine2d.Color{
			s.bg, s.alt,
			s.fg, s.inactive, s.active,
			readableOn(s.bg, accent), readableOn(s.bg, accent.Lerp(text, 0.35)),
			readableOn(s.bg, solid(p.Danger, s.bg)), readableOn(s.bg, solid(p.Warning, s.bg)), readableOn(s.bg, solid(p.Success, s.bg)),
			focus, hover,
		}
	}
	muted := func(fg, on paintengine2d.Color) paintengine2d.Color {
		if p.TextMuted.A > 0 {
			return readableOn(on, solid(p.TextMuted, on))
		}
		return fg.Lerp(on, 0.4)
	}
	window := set{"Window", bg, alt, readableOn(bg, text), muted(text, bg), readableOn(bg, accent)}
	view := set{"View", field, alt, readableOn(field, text), muted(text, field), readableOn(field, accent)}
	button := set{"Button", surface, alt, readableOn(surface, text), muted(text, surface), readableOn(surface, accent)}
	selection := set{"Selection", sel, hover, onSel, onSel.Lerp(sel, 0.3), onSel}
	tooltip := set{"Tooltip", surface, alt, readableOn(surface, text), muted(text, surface), readableOn(surface, accent)}
	// Complementary is the set a desktop shows inverted (Plasma's dark
	// panels on a light scheme): the palette's text and background
	// swapped.
	comp := set{"Complementary", text, text.Lerp(bg, 0.12), bg, bg.Lerp(text, 0.4), readableOn(text, accent)}
	header := set{"Header", cap.Active, cap.Active, cap.ActiveText, cap.ActiveText.Lerp(cap.Active, 0.35), cap.ActiveText}
	headerOff := set{"Header][Inactive", cap.Inactive, cap.Inactive, cap.InactiveText, cap.InactiveText.Lerp(cap.Inactive, 0.35), cap.InactiveText}

	var b strings.Builder
	group := func(g string) { fmt.Fprintf(&b, "\n[%s]\n", g) }
	key := func(k string, c paintengine2d.Color) { fmt.Fprintf(&b, "%s=%s\n", k, kdeRGB(c)) }
	colorSet := func(s set) {
		group("Colors:" + s.name)
		for i, c := range roles(s) {
			key(KDEColorRoles[i], c)
		}
	}
	name = strings.NewReplacer("\n", " ", "[", "(", "]", ")").Replace(name)
	fmt.Fprintf(&b, "[General]\nColorScheme=%s\nName=%s\n", name, name)
	for _, s := range []set{window, view, button, selection, tooltip, comp, header, headerOff} {
		colorSet(s)
	}
	// An inactive window's colours are the scheme's own (the Header set
	// says them outright); no effect is laid over them.
	b.WriteString("\n[ColorEffects:Inactive]\nEnable=false\n")
	group("WM")
	wm := []paintengine2d.Color{
		cap.Active, cap.Active, cap.ActiveText,
		cap.Inactive, cap.Inactive, cap.InactiveText,
		cap.Active, cap.Inactive,
	}
	for i, c := range wm {
		key(KDEWMKeys[i], c)
	}
	return []byte(b.String())
}

// kdeRGB is c as a colour scheme writes it: "r,g,b", 0 to 255.
func kdeRGB(c paintengine2d.Color) string {
	ch := func(v float32) int { return int(math.Round(float64(max(min(v, 1), 0) * 255))) }
	return fmt.Sprintf("%d,%d,%d", ch(c.R), ch(c.G), ch(c.B))
}
