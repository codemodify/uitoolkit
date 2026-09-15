package style

import (
	"math"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// materialEngine paints Google's Material Design, from the published
// material.io guidelines taken as numbers: colour roles, emphasis and state
// opacities, dp sizes and corner radii, elevations.
//
// Material 2 (2014–2018, the "material" packs, param gen 2):
//
//   - the baseline theme: primary #6200EE, secondary #03DAC6 (its dark
//     variant #018786 under white marks), white background and surfaces
//     (the dark theme: #121212 lifted by white elevation overlays of 5% at
//     1dp to 16% at 24dp, primary #BB86FC);
//   - text in the on-surface colour at the three emphasis opacities (87%,
//     60%, 38% disabled; 87/60/38 of white in the dark theme);
//   - the default button is the contained button: primary fill, 4dp
//     corners, the label in capitals (bold: the toolkit has no medium
//     weight), resting at 2dp, 4dp under the pointer and 8dp while pressed,
//     with MDC's umbra, penumbra and ambient shadow layers painted inside
//     the control; other buttons are text buttons (a capitalised primary
//     label on nothing until the pointer brings the overlay);
//   - every interactive surface takes a state overlay of its content
//     colour: 4% hover, 12% focus, 12% pressed, 12% on the selected row
//     (8%, 24%, 24% of white on the primary);
//   - the outlined text field: a 1dp outline (on-surface 38%, 87% under
//     the pointer, a 2dp primary line when focused) whose placeholder is
//     the label and floats into a notch in the top edge once the field has
//     text or focus;
//   - 18dp check boxes and 20dp radios in the secondary colour (its dark
//     variant on light themes so the white mark reads), each in a round
//     touch target that carries the state overlay;
//   - the switch: a 36×14dp track and a 20dp thumb with a 1dp shadow,
//     secondary when on;
//   - the slider: a 4dp track, primary up to the thumb and primary at 24%
//     after it;
//   - tabs with capitalised labels and the 2dp primary indicator;
//   - menus and dialogs as surfaces at 8dp and 24dp elevation; the grey
//     tooltip at 90% in the small type.
//
// Material 3 / Material You (2021, the "material3" packs, param gen 3):
//
//   - every colour role comes from tonal palettes derived from one seed
//     colour (extra "seed"; see engine_material_tone.go, which approximates
//     HCT in CIELAB);
//   - fully rounded buttons: filled (the default button), outlined (other
//     buttons) and tonal (a latched toggle button);
//   - state layers of 8% hover and 12% focus and press;
//   - the switch with a 52×32dp track, a 16dp thumb off and a 24dp thumb
//     on (28dp while pressed), no icon;
//   - the 2021 slider (a 4dp track, a 20dp handle);
//   - navigation-style tabs: the selected one sits on a secondary-container
//     pill;
//   - tonal elevation: menus and dialogs are the surface tinted with the
//     primary colour (8% and 11%), with the two-layer key and ambient
//     shadows of levels 2 and 3; dialogs have 28dp corners;
//   - list and table selections are secondary-container pills.
//
// A sidebar (StateSidebar) is each generation's navigation drawer: Material
// 3's secondary-container pill on surface-container-low, Material 2's
// primary-tinted, 4dp-rounded activated item on the surface.
//
// Both animate their state changes: StyleHint answers HintHoverFadeMs 150.
//
// Pack data: params "gen" (2 or 3) and "plain" (how a plain push button
// looks: 0 text button (M2) / outlined (M3), 1 outlined, 2 contained or
// tonal). Material 2 colours come from extra "primary", "primaryVariant",
// "secondary", "secondaryVariant", "background", "surface", "error",
// "onPrimary", "onSecondary", "onSurface"; Material 3 schemes from extra
// "seed", and any role can be pinned by name ("primary", "surface", …).
type materialEngine struct{ BaseEngine }

func init() {
	RegisterEngine(materialEngine{})
	for _, p := range materialPacks() {
		RegisterPack(p)
	}
}

func (materialEngine) ID() string { return "material" }

// DefaultMetrics are Material 2's desktop sizes at the toolkit's 16px UI
// font: a 36dp button face (plus room for its shadow), dense 32dp menu
// rows, 48dp tabs, 18dp check boxes in 28px touch targets.
func (materialEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 1,
		Radius:    4, RadiusSmall: 4,
		ControlH: 40, FieldH: 46, ComboH: 46,
		Checkbox: 28, Radio: 28,
		MenuItemH: 32, MenuBarH: 36, TabH: 44, RowH: 36,
		TitleBar: 40, HeaderH: 40, ProgressH: 12, SliderH: 32, Thumb: 20,
		Scroll: 12, Pad: 14, FieldPad: 12, FocusWidth: 2, Border: 1,
		ToolBarH: 48, StatusBarH: 28, SpinnerW: 28, SwitchW: 40, SwitchH: 24,
	}
}

// ---- colours ----------------------------------------------------------------------------------

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

type mdKey struct{}

func mdColors(l *Classic) *mdSet {
	return l.Memo(mdKey{}, func() any { return mdBuild(l) }).(*mdSet)
}

func mdGen(l *Classic) int {
	if l.P("gen", 2) >= 3 {
		return 3
	}
	return 2
}

// mdOver is c at opacity a over base, flattened.
func mdOver(base, c paintengine2d.Color, a float32) paintengine2d.Color { return Mix(base, c, a) }

func mdBuild(l *Classic) *mdSet {
	dark := Luma(l.palette.Background) < 0.5
	c := &mdSet{m3: mdGen(l) == 3, dark: dark}
	if c.m3 {
		mdBuild3(l, c)
	} else {
		mdBuild2(l, c)
	}
	small := l.metrics.FontSize * 0.75
	c.small = BakeFamily(l.UIFamily(), WeightRegular, small, c.text2)
	c.label = l.BoldFont()
	c.navSel = mdOver(c.drawer(), c.primary, 0.12)
	c.navSelText = ReadableOn(c.navSel, 4.5, mdOver(c.navSel, c.primary, 0.87), c.primary, c.text)
	return c
}

// mdBuild2 resolves the Material 2 theme colours.
func mdBuild2(l *Classic, c *mdSet) {
	x := func(k, light, dark string) paintengine2d.Color {
		if c.dark {
			return l.X(k, Hex(dark))
		}
		return l.X(k, Hex(light))
	}
	c.primary = x("primary", "#6200ee", "#bb86fc")
	c.primaryC = x("primaryVariant", "#3700b3", "#3700b3")
	c.secondary = x("secondary", "#03dac6", "#03dac6")
	secVariant := x("secondaryVariant", "#018786", "#03dac6")
	c.errorC = x("error", "#b00020", "#cf6679")
	c.bg = x("background", "#ffffff", "#121212")
	c.surface = x("surface", "#ffffff", "#121212")
	c.onPrimary = x("onPrimary", "#ffffff", "#000000")
	c.onSecondary = x("onSecondary", "#000000", "#000000")
	c.onError = x("onError", "#ffffff", "#000000")
	c.onSurface = x("onSurface", "#000000", "#ffffff")
	c.onPrimaryC = c.onPrimary
	c.secondaryC = c.secondary
	c.onSecondaryC = c.onSecondary
	// Dark surfaces are lifted by a white overlay per dp of elevation.
	lift := func(pct float32) paintengine2d.Color {
		if !c.dark {
			return c.surface
		}
		return mdOver(c.surface, Hex("#ffffff"), pct)
	}
	c.card = lift(0.05)   // 1dp
	c.bar = lift(0.09)    // 4dp
	c.menu = lift(0.12)   // 8dp
	c.dialog = lift(0.16) // 24dp
	base := c.surface
	c.text = mdOver(base, c.onSurface, 0.87)
	c.text2 = mdOver(base, c.onSurface, 0.60)
	c.textDis = mdOver(base, c.onSurface, 0.38)
	c.onSurfaceVar = c.text2
	c.divider = mdOver(base, c.onSurface, 0.12)
	c.fieldLine = mdOver(base, c.onSurface, 0.38)
	c.outline = c.fieldLine
	c.outlineVar = c.divider
	c.surfaceVar = mdOver(base, c.onSurface, 0.04)
	c.invSurface = Hex("#616161")
	c.invOnSurface = Hex("#ffffff")
	c.tipBg = l.X("tooltip", Hex("#616161e6"))
	c.tipFg = l.X("tooltipText", Hex("#ffffff"))
	// Light themes fill checks with the dark secondary so the white mark
	// reads; the dark theme has the bright secondary under a black mark.
	c.ctlOn, c.ctlMark = secVariant, Hex("#ffffff")
	if c.dark {
		c.ctlOn, c.ctlMark = c.secondary, c.onSecondary
	}
	c.ctlOff = mdOver(base, c.onSurface, 0.54)
	// The switch thumb when off: white, grey 400 in the dark theme.
	c.thumbOff = Hex("#ffffff")
	if c.dark {
		c.thumbOff = Hex("#bdbdbd")
	}
	c.rowSel = mdOver(base, c.primary, 0.12)
	c.rowSelText = ReadableOn(c.rowSel, 4.5, c.primary, c.text)
	c.trackOff = c.primary.WithAlpha(0.24)
	c.aHover, c.aFocus, c.aPress = 0.04, 0.12, 0.12
	if c.dark {
		c.aHover = 0.08
	}
}

// mdBuild3 derives the Material 3 roles from the seed.
func mdBuild3(l *Classic, c *mdSet) {
	s := mdSchemeFrom(mdCorePalette(l.X("seed", Hex("#6750a4"))), c.dark)
	x := func(k string, def paintengine2d.Color) paintengine2d.Color { return l.X(k, def) }
	c.primary, c.onPrimary = x("primary", s.primary), x("onPrimary", s.onPrimary)
	c.primaryC, c.onPrimaryC = x("primaryContainer", s.primaryContainer), x("onPrimaryContainer", s.onPrimaryContainer)
	c.secondary, c.onSecondary = x("secondary", s.secondary), x("onSecondary", s.onSecondary)
	c.secondaryC, c.onSecondaryC = x("secondaryContainer", s.secondaryContainer), x("onSecondaryContainer", s.onSecondaryContainer)
	c.errorC, c.onError = x("error", s.errorC), x("onError", s.onError)
	c.bg, c.surface = x("background", s.background), x("surface", s.surface)
	c.onSurface, c.onSurfaceVar = x("onSurface", s.onSurface), x("onSurfaceVariant", s.onSurfaceVariant)
	c.surfaceVar = x("surfaceVariant", s.surfaceVariant)
	c.outline, c.outlineVar = x("outline", s.outline), x("outlineVariant", s.outlineVariant)
	c.invSurface, c.invOnSurface = x("inverseSurface", s.inverseSurface), x("inverseOnSurface", s.inverseOnSurface)
	// Tonal elevation: the surface tinted with the primary per level.
	tint := func(pct float32) paintengine2d.Color { return mdOver(c.surface, c.primary, pct) }
	c.card = tint(0.05)   // level 1
	c.menu = tint(0.08)   // level 2
	c.bar = tint(0.08)    // level 2
	c.dialog = tint(0.11) // level 3
	c.text = c.onSurface
	c.text2 = c.onSurfaceVar
	c.textDis = mdOver(c.surface, c.onSurface, 0.38)
	c.divider = c.outlineVar
	c.fieldLine = c.outline
	c.tipBg, c.tipFg = c.invSurface, c.invOnSurface
	c.ctlOn, c.ctlMark, c.ctlOff = c.primary, c.onPrimary, c.onSurfaceVar
	c.rowSel, c.rowSelText = c.secondaryC, c.onSecondaryC
	c.trackOff = c.surfaceVar
	c.aHover, c.aFocus, c.aPress = 0.08, 0.12, 0.12
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

// ---- helpers ----------------------------------------------------------------------------------

// mdPx is one device pixel: 1 at 1x, 2 at 2x.
func mdPx(l *Classic) float32 {
	v := float32(math.Round(float64(l.S(1))))
	if v < 1 {
		v = 1
	}
	return v
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

// mdFill fills a round rect (radius clamped to the shape).
func mdFill(ctx *paintengine2d.Context, b paintengine2d.Rect, r float32, col paintengine2d.Color) {
	if b.Empty() || col.A <= 0 {
		return
	}
	r = min(r, min(b.Dx(), b.Dy())*0.5)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(col))
}

// mdCentered is a w×h rect centred in b, on whole pixels.
func mdCentered(b paintengine2d.Rect, w, h float32) paintengine2d.Rect {
	return mdSnap(paintengine2d.XYWH((b.Min.X+b.Max.X-w)*0.5, (b.Min.Y+b.Max.Y-h)*0.5, w, h))
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

// mdTick strokes the check mark into g.
func mdTick(ctx *paintengine2d.Context, g paintengine2d.Rect, col paintengine2d.Color, w float32) {
	p := paintengine2d.NewPath()
	p.MoveTo(g.Min.X+g.Dx()*0.17, g.Min.Y+g.Dy()*0.53)
	p.LineTo(g.Min.X+g.Dx()*0.40, g.Min.Y+g.Dy()*0.76)
	p.LineTo(g.Min.X+g.Dx()*0.84, g.Min.Y+g.Dy()*0.28)
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapSquare, Join: paintengine2d.JoinMiter, MiterLimit: 4}})
}

// mdChevron strokes the Material chevron (expand_more / chevron_right).
func mdChevron(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	s := min(b.Dx(), b.Dy(), l.S(12))
	if s < 3 {
		return
	}
	Chevron(ctx, mdCentered(b, s, s), dir, col, max(l.S(1.6), 1))
}

// mdTriangle fills Material's small drop-down triangle (arrow_drop_down).
func mdTriangle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	w := min(l.S(10), b.Dx()*0.8, b.Dy()*1.6)
	if w < 2 {
		return
	}
	h := w * 0.5
	cx, cy := (b.Min.X+b.Max.X)*0.5, (b.Min.Y+b.Max.Y)*0.5
	p := paintengine2d.NewPath()
	switch dir {
	case DirUp:
		p.MoveTo(cx-w*0.5, cy+h*0.5)
		p.LineTo(cx+w*0.5, cy+h*0.5)
		p.LineTo(cx, cy-h*0.5)
	case DirLeft:
		p.MoveTo(cx+h*0.5, cy-w*0.5)
		p.LineTo(cx+h*0.5, cy+w*0.5)
		p.LineTo(cx-h*0.5, cy)
	case DirRight:
		p.MoveTo(cx-h*0.5, cy-w*0.5)
		p.LineTo(cx-h*0.5, cy+w*0.5)
		p.LineTo(cx+h*0.5, cy)
	default:
		p.MoveTo(cx-w*0.5, cy-h*0.5)
		p.LineTo(cx+w*0.5, cy-h*0.5)
		p.LineTo(cx, cy+h*0.5)
	}
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

// ---- parts ------------------------------------------------------------------------------------

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

func (e materialEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := mdColors(l)
	switch role {
	case RoleButton:
		return c.button(l, ctx, b, st)
	case RoleTool:
		return c.tool(l, ctx, b, st)
	case RoleField, RoleCombo:
		c.field(l, ctx, mdSnap(b), st, 0, 0)
		if st.Disabled() {
			return c.textDis
		}
		return c.text
	case RoleCheck:
		mdRing(ctx, mdSnap(b), l.rx(2), 2*mdPx(l), c.ctlOff)
		return c.text
	case RoleRow:
		return c.row(l, ctx, mdSnap(b), st, 0)
	case RoleMenu:
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			e.MenuHighlight(l, ctx, b, false)
		}
		return c.text
	case RoleTab:
		if a := c.layer(st); a > 0 {
			ctx.DrawRect(b, paintengine2d.Fill(c.primary.WithAlpha(a)))
		}
		return c.text2
	case RoleThumb:
		mdFill(ctx, b, min(b.Dx(), b.Dy())*0.5, c.onSurface.WithAlpha(0.38))
		return c.text
	case RoleTrack:
		mdFill(ctx, b, min(b.Dx(), b.Dy())*0.5, c.trackOff)
		return c.text
	case RoleSplitter:
		return c.text
	case RoleBar:
		ctx.DrawRect(b, paintengine2d.Fill(c.surface))
		return c.text
	case RolePanel:
		ctx.DrawRect(b, paintengine2d.Fill(c.bg))
		return c.text
	}
	return c.text
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

func (e materialEngine) CheckIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	c := mdColors(l)
	s := snap(min(l.S(18), box.Dx(), box.Dy()))
	if s < 4 {
		return
	}
	g := mdCentered(box, s, s)
	lw := max(2*mdPx(l), snap(s/9))
	r := l.rx(2)
	switch {
	case checked:
		col := c.ctlOn
		if st.Disabled() {
			col = c.onSurface.WithAlpha(0.38)
		}
		ctx.DrawRoundRect(g, r, r, paintengine2d.Fill(col))
		mark := c.ctlMark
		if st.Disabled() {
			mark = c.surface
		}
		mdTick(ctx, g.Inset(s*0.1), mark, max(l.S(2), 1.2))
	default:
		col := c.ctlOff
		if st.Disabled() {
			col = c.onSurface.WithAlpha(0.38)
		}
		mdRing(ctx, g, r, lw, col)
	}
}

func (e materialEngine) RadioIndicator(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	c := mdColors(l)
	d := snap(min(l.S(20), box.Dx(), box.Dy()))
	if d < 4 {
		return
	}
	g := mdCentered(box, d, d)
	ctr := paintengine2d.Pt((g.Min.X+g.Max.X)*0.5, (g.Min.Y+g.Max.Y)*0.5)
	col := c.ctlOff
	if selected {
		col = c.ctlOn
	}
	if st.Disabled() {
		col = c.onSurface.WithAlpha(0.38)
	}
	lw := max(2*mdPx(l), snap(d/10))
	mdRing(ctx, g, d*0.5, lw, col)
	if selected {
		ctx.DrawCircle(ctr, d*0.25, paintengine2d.Fill(col))
	}
}

// Arrow is the Material chevron (scroll steps, spinners, submenus).
func (materialEngine) Arrow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, dir Direction, col paintengine2d.Color) {
	mdChevron(l, ctx, b, dir, col)
}

// Expander is the tree's chevron: right when closed, down when open.
func (materialEngine) Expander(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, expanded bool, col paintengine2d.Color) {
	dir := DirRight
	if expanded {
		dir = DirDown
	}
	mdChevron(l, ctx, b, dir, col)
}

// MenuHighlight is the state layer over a hot menu row (and an open menu
// title).
func (materialEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := mdColors(l)
	a := c.aHover
	if attachBottom {
		a = c.aPress
	}
	ctx.DrawRect(mdSnap(b), paintengine2d.Fill(c.onSurface.WithAlpha(max(a, 0.06))))
}

func (materialEngine) MenuTextColor(l *Classic, hot bool) paintengine2d.Color {
	return mdColors(l).text
}

// Fields show focus with their 2dp primary outline (painted by the field).
func (materialEngine) FieldFocusRing(l *Classic) bool { return false }

// DrawFocusRing is the focus state layer: 12% of the content colour over
// the control's shape.
func (materialEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mdColors(l)
	b = mdSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r := l.rx(4)
	if c.m3 {
		r = min(b.Dx(), b.Dy()) * 0.5
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.onSurface.WithAlpha(c.aFocus)))
}

// ---- scroll bars ------------------------------------------------------------------------------

// ScrollBarStyle: Material's scroll indicator comes and goes with
// scrolling: a thin thumb at rest that widens under the pointer.
func (materialEngine) ScrollBarStyle(l *Classic) ScrollBarStyle {
	return ScrollBarStyle{Thickness: 12, Overlay: true, Transient: true, Inset: 1, MinThumb: 32, EndPad: 4}
}

func (materialEngine) DrawScrollBarParts(l *Classic, ctx *paintengine2d.Context, p ScrollParts, vertical bool, st ScrollState) {
	c := mdColors(l)
	bar := mdSnap(p.Bar)
	if bar.Empty() || p.Thumb.Empty() || st.Disabled {
		return
	}
	hover := st.Hovered || st.Hot != ScrollNone || st.Pressed != ScrollNone
	w := snap(l.S(4))
	col := c.onSurface.WithAlpha(0.32)
	if hover {
		w = snap(l.S(8))
		col = c.onSurface.WithAlpha(0.46)
		if st.Pressed == ScrollThumbPart {
			col = c.onSurface.WithAlpha(0.6)
		}
		// The track shows under the pointer.
		tr := mdSnap(p.Track)
		if vertical {
			tr = paintengine2d.XYWH(snap((bar.Min.X+bar.Max.X-w)*0.5), tr.Min.Y, w, tr.Dy())
		} else {
			tr = paintengine2d.XYWH(tr.Min.X, snap((bar.Min.Y+bar.Max.Y-w)*0.5), tr.Dx(), w)
		}
		mdFill(ctx, tr, w*0.5, c.onSurface.WithAlpha(0.08))
	}
	t := mdSnap(p.Thumb)
	if vertical {
		x := bar.Max.X - w - snap(l.S(1))
		if hover {
			x = snap((bar.Min.X + bar.Max.X - w) * 0.5)
		}
		t = paintengine2d.XYWH(x, t.Min.Y, w, t.Dy())
	} else {
		y := bar.Max.Y - w - snap(l.S(1))
		if hover {
			y = snap((bar.Min.Y + bar.Max.Y - w) * 0.5)
		}
		t = paintengine2d.XYWH(t.Min.X, y, t.Dx(), w)
	}
	mdFill(ctx, t, w*0.5, col)
}

func (e materialEngine) DrawScrollBar(l *Classic, ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	ss := ScrollState{Hovered: st.Hovered() || st.Pressed()}
	if st.Pressed() {
		ss.Pressed = ScrollThumbPart
	}
	e.DrawScrollBarParts(l, ctx, ScrollParts{Bar: track, Track: track, Thumb: thumb}, track.Dy() >= track.Dx(), ss)
}

// ---- frames -----------------------------------------------------------------------------------

// cardR is the corner of cards, views and group boxes: 4dp in Material 2,
// 12dp (medium) in Material 3.
func (c *mdSet) cardR(l *Classic) float32 {
	if c.m3 {
		return l.rx(12)
	}
	return l.rx(4)
}

// GroupBoxInsets: a titled card; the title sits inside it.
func (materialEngine) GroupBoxInsets(l *Classic, hasTitle bool) Insets {
	pad := l.metrics.Pad
	top := pad
	if hasTitle {
		top = mdHeadH(l) + pad*0.5
	}
	return Insets{Top: top, Right: pad, Bottom: pad, Left: pad}
}

func mdHeadH(l *Classic) float32 {
	return snap(l.BoldFont().Height() + l.S(14))
}

// DrawGroupBox is a card: outlined when flat, elevated when raised, the
// title in the card's top-left.
func (materialEngine) DrawGroupBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, raised bool) {
	c := mdColors(l)
	c.cardBox(l, ctx, b, raised)
	if title == "" {
		return
	}
	pad := l.metrics.Pad
	hh := mdHeadH(l)
	l.drawFittedText(ctx, l.BoldFont(), title, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+l.S(4), b.Dx()-pad*2, hh-l.S(4)), c.text, AlignStart, 0)
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

// mdCaptionH is a dialog's title band.
func mdCaptionH(l *Classic) float32 {
	return snap(max(l.BoldFont().Height()+l.S(24), l.S(44)))
}

func (materialEngine) WindowFrameInsets(l *Classic) Insets {
	px := mdPx(l)
	return Insets{Top: mdCaptionH(l), Right: px, Bottom: px, Left: px}
}

// WindowCloseRect is the close icon button at the right of the title band.
func (materialEngine) WindowCloseRect(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = mdSnap(b)
	h := mdCaptionH(l)
	s := snap(min(l.S(32), h-l.S(8)))
	if b.Dx() < s*3 || b.Dy() < h {
		return paintengine2d.Rect{}
	}
	return paintengine2d.XYWH(b.Max.X-s-snap(l.S(8)), b.Min.Y+snap((h-s)*0.5), s, s)
}

// DrawWindowFrame is a Material dialog: the dialog surface (4dp corners in
// Material 2, 28dp in Material 3), the title in the top-left, the close
// icon button at the right.
func (e materialEngine) DrawWindowFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st WindowState) {
	c := mdColors(l)
	b = mdSnap(b)
	if b.Empty() {
		return
	}
	r := l.rx(4)
	if c.m3 {
		r = min(l.rx(28), min(b.Dx(), b.Dy())*0.25)
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.dialog))
	h := mdCaptionH(l)
	right := b.Max.X
	if st.CanClose {
		cb := e.WindowCloseRect(l, b)
		if !cb.Empty() {
			fg := c.text2
			if !st.Active {
				fg = c.textDis
			}
			a := float32(0)
			switch {
			case st.ClosePress:
				a = c.aPress
			case st.CloseHot:
				a = c.aHover * 2
			}
			if a > 0 {
				ctx.DrawCircle(paintengine2d.Pt((cb.Min.X+cb.Max.X)*0.5, (cb.Min.Y+cb.Max.Y)*0.5), cb.Dx()*0.5, paintengine2d.Fill(c.onSurface.WithAlpha(a)))
			}
			g := cb.Inset(cb.Dx() * 0.32)
			DrawCross(ctx, g, fg, max(l.S(1.6), 1))
			right = cb.Min.X - l.S(4)
		}
	}
	if title == "" {
		return
	}
	fg := c.text
	if !st.Active {
		fg = c.text2
	}
	x := b.Min.X + l.S(20)
	l.drawFittedText(ctx, l.BoldFont(), title, paintengine2d.XYWH(x, b.Min.Y, right-x, h), fg, AlignStart, 0)
}

func (materialEngine) DrawWindowBackground(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(mdColors(l).bg))
}

// PopupShadow: menus float at 8dp (Material 2) or level 2 (Material 3),
// dialogs at 24dp / level 3; tooltips have no elevation.
func (materialEngine) PopupShadow(l *Classic, kind PopupKind) Insets {
	var in Insets
	for _, s := range mdPopupShadows(l, kind) {
		if s.a > 0 {
			in = in.Max(ShadowReach(0, s.dy, s.blur, s.spread))
		}
	}
	return in
}

func (materialEngine) DrawPopupShadow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, kind PopupKind) {
	c := mdColors(l)
	r := l.rx(4)
	if kind == PopupDialog && c.m3 {
		r = min(l.rx(28), min(b.Dx(), b.Dy())*0.25)
	}
	for _, s := range mdPopupShadows(l, kind) {
		if s.a > 0 {
			DropShadow(ctx, b, r, paintengine2d.RGBA(0, 0, 0, s.a), 0, s.dy, s.blur, s.spread)
		}
	}
}

// mdShadow is one layer of an elevation shadow: opacity, offset, blur and
// spread (dp at 1x).
type mdShadow struct{ a, dy, blur, spread float32 }

// mdPopupShadows are the layers of a floating layer's shadow: menus at 8dp
// (Material 2) or level 2 (Material 3), dialogs at 24dp or level 3.
func mdPopupShadows(l *Classic, kind PopupKind) [3]mdShadow {
	c := mdColors(l)
	k := l.P("shadow", 1)
	if c.dark {
		k *= 1.6
	}
	var s [3]mdShadow
	switch kind {
	case PopupTooltip:
		return s
	case PopupDialog:
		s = mdLevel(c.m3, 24)
	default:
		s = mdLevel(c.m3, 3)
		if !c.m3 {
			s = mdLevel(false, 8)
		}
	}
	for i := range s {
		s[i].a = min(s[i].a*k, 0.9)
		s[i].dy, s[i].blur, s[i].spread = l.S(s[i].dy), l.S(s[i].blur), l.S(s[i].spread)
	}
	return s
}

// ItemFocus is the focus state layer over the current row.
func (materialEngine) ItemFocus(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := mdColors(l)
	b = mdSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r := float32(0)
	if c.m3 {
		b = mdRowPill(l, b)
		r = b.Dy() * 0.5
	}
	mdFill(ctx, b, r, c.onSurface.WithAlpha(c.aFocus))
}

// ViewFrameInsets: views sit in a 1dp outlined card.
func (materialEngine) ViewFrameInsets(l *Classic) Insets {
	v := mdPx(l)
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

func (materialEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := mdColors(l)
	if st.Sidebar() {
		// A navigation drawer: its container, no outline; focus shows on
		// the item.
		if !b.Empty() {
			ctx.DrawRect(b, paintengine2d.Fill(c.drawer()))
		}
		return
	}
	b = mdSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r := min(l.rx(4), b.Dy()*0.25)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.surface))
	edge := c.divider
	if st.Focused() && !st.Disabled() {
		edge = c.primary
	}
	mdRing(ctx, b, r, mdPx(l), edge)
}

// ViewBackground: a sidebar is a navigation drawer, whose items sit on its
// container; other views on the field colour.
func (e materialEngine) ViewBackground(l *Classic, st ControlState) paintengine2d.Color {
	if st.Sidebar() {
		return mdColors(l).drawer()
	}
	return e.BaseEngine.ViewBackground(l, st)
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

// StyleHint: Material's state changes fade (about 150ms); dialogs put the
// confirming action last; tabs start at the left.
func (materialEngine) StyleHint(l *Classic, h StyleHint) int {
	if h == HintHoverFadeMs {
		return 150
	}
	if h == HintMnemonics {
		return MnemonicsNever // Material has no mnemonics
	}
	return 0
}

// ControlFont: buttons and tabs are labelled in the medium (here bold)
// weight.
func (materialEngine) ControlFont(l *Classic, role Role) *Font {
	if role == RoleButton || role == RoleTab {
		return l.BoldFont()
	}
	return l.body
}

// SpinBoxStyle: the step buttons sit inside the outlined field.
func (materialEngine) SpinBoxStyle(l *Classic) SpinBoxStyle { return SpinBoxStyle{Inside: true} }

func (materialEngine) DrawTabPane(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(mdColors(l).surface))
}

// ---- controls ---------------------------------------------------------------------------------

func (e materialEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := mdColors(l)
	fg := c.button(l, ctx, b, st)
	f := c.face(l, b)
	room := f.Dx() - l.S(8)
	l.drawFittedText(ctx, c.label, c.labelText(c.label, label, room), f, fg, AlignCenter, l.S(8))
}

func (e materialEngine) DrawToolButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	c := mdColors(l)
	in := mdSnap(b).Inset(mdPx(l))
	fg := c.tool(l, ctx, in, st)
	if label != "" && fg == c.text2 && !st.Disabled() {
		fg = c.text
	}
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(b.Min.X+(b.Dx()-iconSide)*0.5, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib.Intersect(b), icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-pad*0.5-x, b.Dy()), fg, AlignStart, 0)
	}
}

// toggleBox is the touch target of a check box, radio or switch at the left
// of b, and the label rect after it.
func mdToggleLayout(l *Classic, b paintengine2d.Rect, side float32) (slot, lb paintengine2d.Rect) {
	side = min(side, b.Dx())
	slot = paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side).Intersect(b)
	lb = paintengine2d.XYWH(b.Min.X+side+l.S(8), b.Min.Y, b.Max.X-(b.Min.X+side+l.S(8)), b.Dy())
	return slot, lb
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

func (e materialEngine) DrawCheckbox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	c := mdColors(l)
	slot, lb := mdToggleLayout(l, b, l.metrics.Checkbox)
	c.halo(ctx, slot, b, st, checked)
	e.CheckIndicator(l, ctx, slot, st, checked)
	c.toggleLabel(l, ctx, lb, st, label)
}

func (e materialEngine) DrawRadio(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	c := mdColors(l)
	side := l.metrics.Radio
	if side <= 0 {
		side = l.metrics.Checkbox
	}
	slot, lb := mdToggleLayout(l, b, side)
	c.halo(ctx, slot, b, st, selected)
	e.RadioIndicator(l, ctx, slot, st, selected)
	c.toggleLabel(l, ctx, lb, st, label)
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

func (e materialEngine) DrawSwitch(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	c := mdColors(l)
	sw, sh := l.metrics.SwitchW, min(l.metrics.SwitchH, b.Dy())
	slot := mdSnap(paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-sh)*0.5, min(sw, b.Dx()), sh))
	if slot.Dx() < 8 || slot.Dy() < 6 {
		return
	}
	lw := mdPx(l)
	dis := st.Disabled()
	if c.m3 {
		// A 52×32 track; the thumb grows from 16 (off) to 24 (on), 28 pressed.
		k := slot.Dy() / 32
		track := mdCentered(slot, min(52*k, slot.Dx()), 32*k)
		r := track.Dy() * 0.5
		d := 16 * k
		if on {
			d = 24 * k
		}
		if st.Pressed() && !dis {
			d = 28 * k
		}
		cx := track.Min.X + r
		if on {
			cx = track.Max.X - r
		}
		cy := (track.Min.Y + track.Max.Y) * 0.5
		switch {
		case on && dis:
			ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(c.onSurface.WithAlpha(0.12)))
		case on:
			ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(c.primary))
		case dis:
			ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(c.surfaceVar.WithAlpha(0.12)))
			mdRing(ctx, track, r, 2*lw, c.onSurface.WithAlpha(0.12))
		default:
			ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(c.surfaceVar))
			mdRing(ctx, track, r, 2*lw, c.outline)
		}
		if a := c.layer(st); a > 0 {
			col := c.onSurface
			if on {
				col = c.primary
			}
			ctx.Save()
			ctx.ClipRect(b)
			ctx.DrawCircle(paintengine2d.Pt(cx, cy), min(20*k, slot.Dy()*0.5+2*lw), paintengine2d.Fill(col.WithAlpha(a)))
			ctx.Restore()
		}
		kc := c.outline
		switch {
		case on && dis:
			kc = c.surface
		case on && (st.Hovered() || st.Pressed() || st.Focused()):
			kc = c.primaryC
		case on:
			kc = c.onPrimary
		case dis:
			kc = c.onSurface.WithAlpha(0.38)
		case st.Hovered() || st.Pressed():
			kc = c.onSurfaceVar
		}
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), d*0.5, paintengine2d.Fill(kc))
	} else {
		// A 36×14 track under a 20dp thumb with a 1dp shadow.
		k := min(slot.Dy()/24, slot.Dx()/40)
		track := mdCentered(slot, 36*k, 14*k)
		d := 20 * k
		// The thumb overhangs the track's ends, inside the slot.
		cx := slot.Min.X + d*0.5
		if on {
			cx = slot.Max.X - d*0.5
		}
		cy := (track.Min.Y + track.Max.Y) * 0.5
		knob := paintengine2d.XYWH(cx-d*0.5, cy-d*0.5, d, d)
		tc := c.onSurface.WithAlpha(0.38)
		kc := c.thumbOff
		if on {
			tc, kc = c.ctlOn.WithAlpha(0.54), c.ctlOn
		}
		if dis {
			tc = c.onSurface.WithAlpha(0.12)
			kc = mdOver(c.surface, c.onSurface, 0.2)
		}
		ctx.DrawRoundRect(track, track.Dy()*0.5, track.Dy()*0.5, paintengine2d.Fill(tc))
		if a := c.layer(st); a > 0 {
			col := c.onSurface
			if on {
				col = c.ctlOn
			}
			ctx.Save()
			ctx.ClipRect(b)
			ctx.DrawCircle(paintengine2d.Pt(cx, cy), min(d, slot.Dy()*0.5+3*lw), paintengine2d.Fill(col.WithAlpha(a)))
			ctx.Restore()
		}
		ctx.Save()
		ctx.ClipRect(b)
		DropShadow(ctx, knob, d*0.5, paintengine2d.RGBA(0, 0, 0, 0.3), 0, l.S(1), l.S(2), 0)
		ctx.Restore()
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), d*0.5, paintengine2d.Fill(kc))
	}
	lb := paintengine2d.XYWH(slot.Max.X+l.S(10), b.Min.Y, b.Max.X-slot.Max.X-l.S(10), b.Dy())
	c.toggleLabel(l, ctx, lb, st, label)
}

func (e materialEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := mdColors(l)
	b = mdSnap(b)
	t = clamp1(t)
	d := min(snap(l.S(20)), b.Dy())
	if b.Dx() < d+4 || d < 6 {
		return
	}
	th := snap(l.S(4))
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	x0, x1 := b.Min.X+d*0.5, b.Max.X-d*0.5
	tx := x0 + (x1-x0)*t
	on, off := c.primary, c.trackOff
	if st.Disabled() {
		on, off = c.onSurface.WithAlpha(0.38), c.onSurface.WithAlpha(0.12)
	}
	track := paintengine2d.XYWH(x0, cy-th*0.5, x1-x0, th)
	mdFill(ctx, track, th*0.5, off)
	if tx > x0 {
		mdFill(ctx, paintengine2d.XYWH(x0, cy-th*0.5, tx-x0, th), th*0.5, on)
	}
	ctr := paintengine2d.Pt(tx, cy)
	if a := c.layer(st); a > 0 {
		ctx.Save()
		ctx.ClipRect(b)
		ctx.DrawCircle(ctr, min(d, b.Dy()*0.5), paintengine2d.Fill(on.WithAlpha(a*2)))
		ctx.Restore()
	}
	if !st.Disabled() {
		// The handle rests at 1dp (Material 2) / level 1 (Material 3).
		ctx.Save()
		ctx.ClipRect(b)
		DropShadow(ctx, paintengine2d.XYWH(tx-d*0.5, cy-d*0.5, d, d), d*0.5, paintengine2d.RGBA(0, 0, 0, 0.24), 0, l.S(1), l.S(2), 0)
		ctx.Restore()
	}
	kc := on
	if st.Disabled() {
		kc = mdOver(c.surface, c.onSurface, 0.38)
	}
	ctx.DrawCircle(ctr, d*0.5, paintengine2d.Fill(kc))
}

func (e materialEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := mdColors(l)
	b = mdSnap(b)
	h := min(snap(l.S(4)), b.Dy())
	if b.Dx() < 4 || h < 1 {
		return
	}
	bar := paintengine2d.XYWH(b.Min.X, snap((b.Min.Y+b.Max.Y-h)*0.5), b.Dx(), h)
	on, off := c.primary, c.trackOff
	if st.Disabled() {
		on, off = c.onSurface.WithAlpha(0.38), c.onSurface.WithAlpha(0.12)
	}
	// Square ends, as both generations drew them until the 2023 refresh.
	r := float32(0)
	mdFill(ctx, bar, r, off)
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		span := bar.Dx() * 0.4
		x := bar.Min.X + (bar.Dx()+span)*phase - span
		seg := paintengine2d.XYWH(x, bar.Min.Y, span, h).Intersect(bar)
		if seg.Dx() >= 1 {
			mdFill(ctx, seg, r, on)
		}
		return
	}
	if w := snap(bar.Dx() * clamp1(t)); w >= 1 {
		mdFill(ctx, paintengine2d.XYWH(bar.Min.X, bar.Min.Y, w, h), r, on)
	}
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

func (e materialEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	c := mdColors(l)
	if st.Frameless() {
		l.baseDrawTextField(ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
		return
	}
	box := c.fieldBox(l, b, placeholder != "")
	float := placeholder != "" && (text != "" || st.Focused()) && box.Min.Y > b.Min.Y
	pad := l.metrics.FieldPad
	var n0, n1 float32
	if float {
		n0 = box.Min.X + pad - l.S(4)
		n1 = min(box.Min.X+pad+c.small.Advance(placeholder)+l.S(4), box.Max.X-pad)
	}
	c.field(l, ctx, box, st, n0, n1)
	if float {
		col := c.text2
		switch {
		case st.Disabled():
			col = c.textDis
		case st.Focused():
			col = c.primary
		}
		lb := paintengine2d.XYWH(box.Min.X+pad, b.Min.Y, n1-l.S(4)-(box.Min.X+pad), c.small.Height())
		ctx.Save()
		ctx.ClipRect(lb.Intersect(b))
		c.small.Draw(ctx, placeholder, lb.Min, col)
		ctx.Restore()
		placeholder = ""
	}
	l.baseDrawTextField(ctx, box, st|StateFrameless, text, placeholder, caret, selA, selB, blink, scrollX, face)
}

func (e materialEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := mdColors(l)
	b = mdSnap(b)
	if open {
		st |= StateFocused
	}
	c.field(l, ctx, b, st, 0, 0)
	aw := min(snap(l.S(32)), b.Dx()*0.4)
	arrow := paintengine2d.XYWH(b.Max.X-aw, b.Min.Y, aw-l.S(6), b.Dy())
	col := c.text2
	switch {
	case st.Disabled():
		col = c.textDis
	case open:
		col = c.primary
	}
	dir := DirDown
	if open {
		dir = DirUp
	}
	mdTriangle(l, ctx, arrow, dir, col)
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	pad := l.metrics.FieldPad
	l.drawFittedText(ctx, l.body, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, arrow.Min.X-b.Min.X-pad, b.Dy()), fg, AlignStart, 0)
}

func (e materialEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	c := mdColors(l)
	b = mdSnap(b)
	if b.Dx() < 4 || b.Dy() < 6 {
		return
	}
	if !st.Frameless() {
		c.field(l, ctx, b, st&^StateFocused, 0, 0)
	}
	mid := snap((b.Min.Y + b.Max.Y) * 0.5)
	half := func(r paintengine2d.Rect, dir Direction, hot, press bool) {
		r = r.Inset(mdPx(l))
		a := float32(0)
		switch {
		case st.Disabled():
		case press:
			a = c.aPress
		case hot:
			a = c.aHover * 2
		}
		if a > 0 {
			mdFill(ctx, r, l.rx(4), c.onSurface.WithAlpha(a))
		}
		col := c.text2
		if st.Disabled() {
			col = c.textDis
		}
		mdTriangle(l, ctx, r, dir, col)
	}
	half(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y), DirUp, upHover, upPress)
	half(paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid), DirDown, downHover, downPress)
}

func (materialEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mdColors(l)
	b = mdSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.surface))
	px := mdPx(l)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.divider))
}

// DrawTab: Material 2's tab (an uppercase label, the 2dp primary indicator
// under the selected one, a primary-tinted layer under the pointer) or
// Material 3's navigation-style tab (the selected label on a
// secondary-container pill).
func (materialEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := mdColors(l)
	b = mdSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	f := c.label
	if c.m3 {
		pw := min(f.Advance(label)+l.S(32), b.Dx()-l.S(4))
		ph := min(snap(l.S(32)), b.Dy()-l.S(6))
		pill := mdCentered(b, pw, ph)
		fg := c.text2
		if selected {
			mdFill(ctx, pill, ph*0.5, c.secondaryC)
			fg = c.onSecondaryC
		}
		if a := c.layer(st); a > 0 {
			col := c.onSurface
			if selected {
				col = c.onSecondaryC
			}
			mdFill(ctx, pill, ph*0.5, col.WithAlpha(a))
		}
		if st.Disabled() {
			fg = c.textDis
		}
		l.drawFittedText(ctx, f, label, pill, fg, AlignCenter, l.S(8))
		return
	}
	if a := c.layer(st); a > 0 {
		ctx.DrawRect(b, paintengine2d.Fill(c.primary.WithAlpha(a)))
	}
	fg := c.text2
	if selected {
		fg = c.primary
		ih := snap(l.S(2))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-ih, b.Dx(), ih), paintengine2d.Fill(c.primary))
	}
	if st.Disabled() {
		fg = c.textDis
	}
	// The bar measured the label in the bold face with 28px to spare.
	room := b.Dx() - l.S(12)
	l.drawFittedText(ctx, f, c.labelText(f, label, room), b, fg, AlignCenter, l.S(12))
}

func (materialEngine) DrawPanel(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	c := mdColors(l)
	if raised {
		c.cardBox(l, ctx, b, true)
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(c.bg))
}

func (materialEngine) DrawMenuBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mdColors(l)
	b = mdSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.surface))
	px := mdPx(l)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.divider))
}

func (e materialEngine) DrawMenuTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	c := mdColors(l)
	b = mdSnap(b)
	a := float32(0)
	switch {
	case st.Disabled():
	case open || st.Pressed():
		a = c.aPress
	case st.Focused():
		a = c.aFocus
	case st.Hovered():
		a = max(c.aHover, 0.06)
	}
	if a > 0 {
		hl := b.Inset(snap(l.S(3)))
		mdFill(ctx, hl, l.rx(4), c.onSurface.WithAlpha(a))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
}

// DrawMenuFrame is a menu: the surface at 8dp (Material 2: white, or the
// lifted dark surface) or at level 2 (Material 3: primary-tinted), 4dp
// corners; the shadow is PopupShadow's.
func (materialEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mdColors(l)
	b = mdSnap(b)
	r := min(l.rx(4), b.Dy()*0.25)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.menu))
}

func (e materialEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := mdColors(l)
	ch := MenuChromeFor(l)
	px := mdPx(l)
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		x0, x1 := b.Min.X-ch.PadL, b.Max.X+ch.PadR
		ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, px), paintengine2d.Fill(c.divider))
		return
	}
	hot := (st.Hovered() || st.Pressed()) && !st.Disabled()
	if hot {
		hb := paintengine2d.XYWH(b.Min.X-ch.PadL, b.Min.Y, b.Dx()+ch.PadL+ch.PadR, b.Dy())
		a := max(c.aHover, 0.06)
		if st.Pressed() {
			a = c.aPress
		}
		ctx.DrawRect(mdSnap(hb), paintengine2d.Fill(c.onSurface.WithAlpha(a)))
	}
	fg, sc := c.text, c.text2
	font := l.body
	if st.Disabled() {
		fg, sc = c.textDis, c.textDis
		font = l.muted
	}
	mdGutter(l, ctx, b, ch, row, fg)
	ty := b.Min.Y + (b.Dy()-font.Height())*0.5
	right := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		mdTriangle(l, ctx, ab, DirRight, sc)
		right = ab.Min.X
	}
	if row.Shortcut != "" {
		tw := l.body.Advance(row.Shortcut)
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		l.body.Draw(ctx, row.Shortcut, paintengine2d.Pt(sx, ty), sc)
		right = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if right < lx {
		right = lx
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, right-lx, b.Dy()))
	l.drawTextUnderline(ctx, font, row.Label, row.Underline, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

// mdGutter paints a menu row's leading column: the check mark, the radio
// button or the icon, sized to the row (24dp at most).
func mdGutter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, ch MenuChrome, row MenuRow, fg paintengine2d.Color) {
	gw := ch.CheckCol()
	side := snap(min(b.Dy()-l.S(6), gw-l.S(2), l.S(20)))
	if side < 4 {
		return
	}
	box := mdSnap(paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side))
	switch {
	case row.Radio:
		ctr := paintengine2d.Pt((box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5)
		d := side * 0.8
		mdRing(ctx, mdCentered(box, d, d), d*0.5, max(l.S(1.6), 1), fg)
		if row.Checked {
			ctx.DrawCircle(ctr, d*0.25, paintengine2d.Fill(fg))
		}
	case row.Checked:
		mdTick(ctx, box.Inset(side*0.1), fg, max(l.S(1.8), 1.2))
	case row.Icon != IconNone:
		l.drawToolIcon(ctx, box, row.Icon, fg)
	}
}

// mdRowPill is a Material 3 row's pill inside its row rect.
func mdRowPill(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	in := snap(l.S(4))
	return mdSnap(paintengine2d.XYWH(b.Min.X+in, b.Min.Y+snap(l.S(2)), b.Dx()-2*in, b.Dy()-2*snap(l.S(2))))
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

func (e materialEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	c := mdColors(l)
	if st.Sidebar() {
		// The focus layer is part of the drawer item's state layer.
		box, fg := c.drawerRow(l, ctx, b, st)
		pad := c.drawerPad(l)
		l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(box.Min.X+pad, b.Min.Y, box.Dx()-pad-l.S(8), b.Dy()), fg, AlignStart, 0)
		return
	}
	box, r := c.rowBox(l, b)
	fg := c.row(l, ctx, box, st, r)
	pad := l.S(16)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad-l.S(8), b.Dy()), fg, AlignStart, 0)
	if st.Focused() {
		e.ItemFocus(l, ctx, b, st)
	}
}

func (e materialEngine) DrawTreeRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, expanded, leaf bool, depth int, label string, bold bool) {
	c := mdColors(l)
	side := st.Sidebar()
	var fg paintengine2d.Color
	x0, right := b.Min.X+l.S(8), b.Max.X-l.S(4)
	if side {
		box, col := c.drawerRow(l, ctx, b, st)
		fg, x0, right = col, box.Min.X+c.drawerPad(l)-l.S(8), box.Max.X-l.S(8)
	} else {
		box, r := c.rowBox(l, b)
		fg = c.row(l, ctx, box, st, r)
	}
	indent := l.metrics.TreeIndent
	if indent <= 0 {
		indent = l.S(16)
	}
	x := x0 + float32(depth)*indent
	if !leaf {
		col := c.text2
		if st.ExpanderHot() || (st.Checked() && !st.Backdrop()) {
			col = fg
		}
		e.Expander(l, ctx, paintengine2d.XYWH(x, b.Min.Y, indent, b.Dy()), expanded, col)
	}
	f := l.body
	if bold {
		f = l.BoldFont()
	}
	lx := x + indent + l.S(6)
	l.drawFittedText(ctx, f, label, paintengine2d.XYWH(lx, b.Min.Y, right-lx, b.Dy()), fg, AlignStart, 0)
	if st.Focused() && !side {
		e.ItemFocus(l, ctx, b, st)
	}
}

// DrawTableHeader is a data table header: medium-emphasis labels over a
// divider, the sorted column in full emphasis with its arrow.
func (e materialEngine) DrawTableHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	c := mdColors(l)
	b = mdSnap(b)
	if b.Empty() {
		return
	}
	px := mdPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.surface))
	if a := c.layer(st &^ StateFocused); a > 0 {
		ctx.DrawRect(b, paintengine2d.Fill(c.onSurface.WithAlpha(a)))
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.divider))
	fg := c.text2
	if sorted {
		fg = c.text
	}
	if st.Disabled() {
		fg = c.textDis
	}
	aw := float32(0)
	if sorted {
		aw = l.S(16)
		dir := DirDown
		if asc {
			dir = DirUp
		}
		mdChevron(l, ctx, paintengine2d.XYWH(b.Max.X-aw-l.S(4), b.Min.Y, aw, b.Dy()), dir, fg)
	}
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+l.S(10), b.Min.Y, b.Dx()-l.S(12)-aw, b.Dy()), fg, AlignStart, 0)
}

// DrawTableCell: rows divided by hairlines; a selected row is one box
// across its cells (a Material 3 pill), tinted by the selection.
func (e materialEngine) DrawTableCell(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, align Align, face *Font) {
	c := mdColors(l)
	cell := mdSnap(b)
	px := mdPx(l)
	ctx.Save()
	ctx.ClipRect(cell)
	span := CellSpan(cell, st, l.S(24))
	box, r := c.rowBox(l, span)
	fg := c.row(l, ctx, box, st, r)
	if !c.m3 && !st.Checked() {
		ctx.DrawRect(paintengine2d.XYWH(cell.Min.X, cell.Max.Y-px, cell.Dx(), px), paintengine2d.Fill(c.divider))
	}
	ctx.Restore()
	f := l.faceOrBody(face)
	pad := l.tableCellPad(b.Dx(), f.Advance(label))
	if st.First() {
		pad = max(pad, l.S(12))
	}
	avail := max(b.Dx()-pad*2, 4)
	label = f.Fit(label, avail)
	tw := f.Advance(label)
	x := b.Min.X + pad
	switch align {
	case AlignCenter:
		x = b.Min.X + (b.Dx()-tw)*0.5
	case AlignEnd:
		x = b.Max.X - tw - pad
	}
	ctx.Save()
	ctx.ClipRect(cell)
	f.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-f.Height())*0.5), fg)
	ctx.Restore()
}

func (materialEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mdColors(l)
	b = mdSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.surface))
	px := mdPx(l)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.divider))
}

func (materialEngine) DrawStatusBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	c := mdColors(l)
	b = mdSnap(b)
	px := mdPx(l)
	ctx.DrawRect(b, paintengine2d.Fill(c.surface))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), px), paintengine2d.Fill(c.divider))
	if len(parts) == 0 {
		return
	}
	slot := b.Dx() / float32(len(parts))
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		if i > 0 {
			ctx.DrawRect(paintengine2d.XYWH(snap(x), b.Min.Y+l.S(6), px, b.Dy()-l.S(12)), paintengine2d.Fill(c.divider))
		}
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+l.S(12), b.Min.Y, slot-l.S(18), b.Dy()), c.text2, AlignStart, 0)
	}
}

func (materialEngine) DrawTitleBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	c := mdColors(l)
	b = mdSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.surface))
	x := b.Min.X + l.metrics.Pad
	f := l.BoldFont()
	if title != "" {
		l.drawFittedText(ctx, f, title, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), c.text, AlignStart, 0)
		x += f.Advance(title) + l.S(12)
	}
	if subtitle != "" && x < b.Max.X {
		l.drawFittedText(ctx, l.body, subtitle, paintengine2d.XYWH(x, b.Min.Y, b.Max.X-x, b.Dy()), c.text2, AlignStart, 0)
	}
}

// DrawAccordionHeader is an expansion panel header: the title and the
// trailing chevron, a state layer under the pointer.
func (e materialEngine) DrawAccordionHeader(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	c := mdColors(l)
	b = mdSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r := l.rx(4)
	if c.m3 {
		r = l.rx(12)
	}
	r = min(r, b.Dy()*0.5)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.card))
	mdRing(ctx, b, r, mdPx(l), c.divider)
	if a := c.layer(st); a > 0 {
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.onSurface.WithAlpha(a)))
	}
	fg := c.text
	if st.Disabled() {
		fg = c.textDis
	}
	dir := DirDown
	if expanded {
		dir = DirUp
	}
	cw := l.S(24)
	mdChevron(l, ctx, paintengine2d.XYWH(b.Max.X-cw-l.S(8), b.Min.Y, cw, b.Dy()), dir, c.text2)
	l.drawFittedText(ctx, l.body, title, paintengine2d.XYWH(b.Min.X+l.S(16), b.Min.Y, b.Dx()-cw-l.S(28), b.Dy()), fg, AlignStart, 0)
}

func (materialEngine) DrawSeparator(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c := mdColors(l)
	px := mdPx(l)
	if vertical {
		x := snap((b.Min.X + b.Max.X - px) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, px, b.Dy()), paintengine2d.Fill(c.divider))
		return
	}
	y := snap((b.Min.Y + b.Max.Y - px) * 0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), px), paintengine2d.Fill(c.divider))
}

func (materialEngine) DrawSplitter(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	c := mdColors(l)
	px := mdPx(l)
	col := c.divider
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		col = c.primary
		px *= 2
	}
	if vertical {
		x := snap((b.Min.X + b.Max.X - px) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y, px, b.Dy()), paintengine2d.Fill(col))
		return
	}
	y := snap((b.Min.Y + b.Max.Y - px) * 0.5)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), px), paintengine2d.Fill(col))
}

// DrawTooltip: Material 2's grey tooltip at 90%, Material 3's plain tooltip
// in the inverse surface; 4dp corners.
func (materialEngine) DrawTooltip(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	c := mdColors(l)
	b = mdSnap(b)
	r := min(l.rx(4), b.Dy()*0.5)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.tipBg))
	pad := l.metrics.TooltipPad
	if pad <= 0 {
		pad = l.S(8)
	}
	// The small type fits when the tip was measured in the body face.
	f := c.small
	if f.Advance(text) > b.Dx()-pad*2 {
		f = l.body
	}
	l.drawFittedText(ctx, f, text, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy()), c.tipFg, AlignStart, 0)
}

// ---- packs ------------------------------------------------------------------------------------

func mdPack(name, label string, year int, summary string, fam ThemeName, pal Palette, metrics ChromeMetrics, extra map[string]string, params map[string]float32) ThemePack {
	tok := ThemeTokens{
		Engine:  "material",
		Bevel:   BevelSoftShadow,
		Family:  fam,
		Palette: pal,
		Metrics: metrics,
		Params:  params,
	}
	for k, v := range extra {
		if tok.Extra == nil {
			tok.Extra = map[string]paintengine2d.Color{}
		}
		tok.Extra[k] = hexColor(v)
	}
	mdChrome(&tok)
	return ThemePack{
		Name: name, Label: label, Year: year, Lineage: "Google", Summary: summary,
		Era: EraMaterial, Palette: fam, Tokens: tok,
	}
}

// mdChrome sets the chrome states Material derives from its palette
// (pressed and disabled come from the resolver).
func mdChrome(tok *ThemeTokens) {
	pal := tok.Palette
	tok.Hot = ChromeState{Fill: pal.MenuHover, Border: pal.MenuHover}
	tok.Selected = ChromeState{Fill: pal.Selection, Border: pal.Accent}
	tok.Focus = ChromeState{Fill: pal.Accent.WithAlpha(0.12), Border: pal.Accent}
}

// md2Baseline is Material 2's baseline primary, #6200EE (Purple 500 of its
// palette); its variant is #3700B3 (the 700) and the dark theme's primary
// #BB86FC (the 200).
var md2Baseline = Hex("#6200ee")

// Accented recolours Material around the desktop's accent. Material 3
// (Material You) takes it as the seed: every tonal palette — primary,
// secondary, tertiary and the neutrals, which carry the seed's hue — is
// regenerated from it, so the whole scheme follows (engine_material_tone.go).
// Material 2 takes it as the primary colour: the contained button, the
// slider, progress, tab indicator, focused field line and selection tints;
// the primary variant (the 700 shade), the dark theme's lighter primary
// (the 200) and its pressed shade make the step from the accent that the
// baseline palette makes from #6200EE (accentShift). The teal secondary of
// Material 2's check boxes and switches stays. On-colours keep 4.5:1.
func (materialEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
	dark := accentDark(tok)
	tok = CloneTokenMaps(tok)
	if accentP(tok, "gen", 2) >= 3 {
		// Every colour the pack's palette took from its seed is taken
		// from the accent; a colour the pack set itself stays.
		was := mdPalette3(accentX(tok, "seed", Hex("#6750a4")), dark)
		tok.Palette = accentRepalette(tok.Palette, was, mdPalette3(accent, dark))
		tok.Extra["seed"] = accent
		p := &tok.Palette
		p.TextOnAccent = ReadableOn(p.Accent, 4.5, p.TextOnAccent)
	} else {
		pick := func(light, darkHex string) paintengine2d.Color {
			if dark {
				return Hex(darkHex)
			}
			return Hex(light)
		}
		primary0 := accentX(tok, "primary", pick("#6200ee", "#bb86fc"))
		// The primary this pack's is the Material 2 shade of: #6200EE for
		// both shipped packs.
		ref := accentShift(primary0, pick("#6200ee", "#bb86fc"), md2Baseline)
		primary := accentShift(accent, ref, primary0)
		variant := accentShift(accent, ref, accentX(tok, "primaryVariant", Hex("#3700b3")))
		p := &tok.Palette
		on := ReadableOn(primary, 4.5, accentX(tok, "onPrimary", pick("#ffffff", "#000000")), pick("#000000", "#ffffff"))
		tok.Extra["primary"], tok.Extra["primaryVariant"], tok.Extra["onPrimary"] = primary, variant, on
		p.Accent, p.Focus, p.TextOnAccent = primary, primary, on
		p.AccentHover = mdOver(primary, on, 0.08)
		p.AccentPress = accentShift(accent, ref, p.AccentPress)
		p.Track, p.Selection = primary.WithAlpha(0.24), primary.WithAlpha(0.24)
	}
	mdChrome(&tok)
	tok.Pressed, tok.Disabled = ChromeState{}, ChromeState{} // the resolver's, from the new palette
	return tok.Resolve()
}

// mdPalette2 is a Material 2 palette over surface (text at the emphasis
// opacities of onSurface).
func mdPalette2(dark bool) Palette {
	bg, surface, on := Hex("#ffffff"), Hex("#ffffff"), Hex("#000000")
	primary, press, onPrimary := Hex("#6200ee"), Hex("#3700b3"), Hex("#ffffff")
	danger := Hex("#b00020")
	if dark {
		bg, surface, on = Hex("#121212"), Hex("#121212"), Hex("#ffffff")
		primary, press, onPrimary = Hex("#bb86fc"), Hex("#9965f4"), Hex("#000000")
		danger = Hex("#cf6679")
	}
	em := func(a float32) paintengine2d.Color { return mdOver(surface, on, a) }
	lift := surface
	if dark {
		lift = mdOver(surface, Hex("#ffffff"), 0.05)
	}
	hover := em(0.04)
	if dark {
		hover = mdOver(surface, on, 0.08)
	}
	p := Palette{
		Background: bg, Surface: lift, SurfaceAlt: mdOver(surface, on, 0.04),
		Overlay: paintengine2d.RGBA(0, 0, 0, 0.32),
		Border:  em(0.12), Divider: em(0.12),
		Text: em(0.87), TextMuted: em(0.6), TextOnAccent: onPrimary,
		Accent: primary, AccentHover: mdOver(primary, onPrimary, 0.08), AccentPress: press,
		Danger: danger, Success: Hex("#388e3c"), Warning: Hex("#f57c00"),
		Track: primary.WithAlpha(0.24), Thumb: em(0.38),
		Field: surface, FieldBorder: em(0.38),
		Focus: primary, Selection: primary.WithAlpha(0.24),
		Shadow: paintengine2d.RGBA(0, 0, 0, 0.2), Highlight: hover,
		MenuHover: hover, MenuHoverBorder: hover, MenuGutter: surface,
		BevelLight: surface, BevelDark: em(0.12),
	}
	if dark {
		p.Success, p.Warning = Hex("#81c784"), Hex("#ffb74d")
	}
	return p
}

// mdPalette3 is a Material 3 palette from a seed.
func mdPalette3(seed paintengine2d.Color, dark bool) Palette {
	s := mdSchemeFrom(mdCorePalette(seed), dark)
	hover := mdOver(s.surface, s.onSurface, 0.08)
	tone := func(light, darkTone float64) float64 {
		if dark {
			return darkTone
		}
		return light
	}
	return Palette{
		Background: s.background, Surface: mdOver(s.surface, s.primary, 0.05), SurfaceAlt: s.surfaceVariant,
		Overlay: paintengine2d.RGBA(0, 0, 0, 0.32),
		Border:  s.outlineVariant, Divider: s.outlineVariant,
		Text: s.onSurface, TextMuted: s.onSurfaceVariant, TextOnAccent: s.onPrimary,
		Accent: s.primary, AccentHover: mdOver(s.primary, s.onPrimary, 0.08), AccentPress: mdOver(s.primary, s.onPrimary, 0.12),
		Danger: s.errorC, Success: mdPalette{140, 40}.tone(tone(40, 80)), Warning: mdPalette{70, 60}.tone(tone(50, 80)),
		Track: s.surfaceVariant, Thumb: s.outline,
		Field: s.surface, FieldBorder: s.outline,
		Focus: s.primary, Selection: s.primary.WithAlpha(0.24),
		Shadow: paintengine2d.RGBA(0, 0, 0, 0.2), Highlight: hover,
		MenuHover: hover, MenuHoverBorder: hover, MenuGutter: s.surface,
		BevelLight: s.surface, BevelDark: s.outlineVariant,
	}
}

func materialPacks() []ThemePack {
	m3 := ChromeMetrics{
		Radius: 20, RadiusSmall: 4,
		ControlH: 42, FieldH: 50, ComboH: 50,
		Checkbox: 30, Radio: 30,
		MenuItemH: 36, MenuBarH: 40, TabH: 48, RowH: 40,
		TitleBar: 44, HeaderH: 44, ProgressH: 12, SliderH: 34, Thumb: 20,
		Pad: 16, FieldPad: 14, SpinnerW: 30, SwitchW: 52, SwitchH: 32, ToolBarH: 52,
	}
	seed := "#6750a4"
	return []ThemePack{
		mdPack("material", "Material", 2014,
			"Material Design 2: contained buttons on 2dp shadows, outlined fields with floating labels, #6200EE.",
			ThemeLight, mdPalette2(false), ChromeMetrics{}, nil, map[string]float32{"gen": 2}),
		mdPack("material-night", "Material Dark", 2014,
			"Material Design 2's dark theme: #121212 surfaces lifted by elevation, primary #BB86FC.",
			ThemeDark, mdPalette2(true), ChromeMetrics{}, nil, map[string]float32{"gen": 2}),
		mdPack("material3", "Material 3", 2021,
			"Material You: tonal palettes from one seed colour, pill buttons, state layers, the big switch.",
			ThemeLight, mdPalette3(Hex(seed), false), m3, map[string]string{"seed": seed}, map[string]float32{"gen": 3}),
		mdPack("material3-night", "Material 3 Dark", 2021,
			"Material You's dark scheme from the same seed: tone 80 primaries on tone 10 surfaces.",
			ThemeDark, mdPalette3(Hex(seed), true), m3, map[string]string{"seed": seed}, map[string]float32{"gen": 3}),
	}
}
