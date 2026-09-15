package style

import (
	"math"

	"github.com/codemodify/paintengine2d"
)

// material3xEngine paints Material 3 as Google's 2025 Expressive update
// draws it, on the spec as it stood after 2023–24: the material engine's era
// for packs with params "gen" 3 and "expressive" 1. The "material3" packs
// keep 2021's Material You. Numbers from the Material 3 token database and
// m3.material.io (CC BY 4.0 / Apache 2.0), taken as facts:
//
//   - tone-based surface containers replace the primary-tinted elevation:
//     cards and the navigation drawer surface-container-low, menus, tool
//     bars and list groups surface-container, dialogs
//     surface-container-high; focus and pressed layers are 10%; light
//     on-containers moved to tone 30;
//   - buttons change shape: the filled and outlined buttons are round
//     (full) and morph to 8dp corners while pressed, a selected toggle is
//     the square 12dp shape in the secondary container; outlined buttons
//     wear the outline-variant stroke and on-surface-variant labels;
//   - keyboard focus is a 3dp ring in the secondary colour 2dp outside the
//     control, in a margin it keeps; a focused text field's outline is 3dp;
//   - the 2025 slider: a 16dp track split round a 4dp bar handle by 6dp
//     gaps (2dp inner corners, full outer ones, a 4dp stop dot), the handle
//     2dp while pressed or focused, the inactive track secondary-container;
//   - wavy progress: the value a 4dp sine stroke (3dp amplitude, 40dp
//     wavelength; 20dp busy) 4dp clear of the straight track;
//   - lists are segmented: rows of 4dp corners with 2dp gaps in a
//     16dp-rounded group, the selected row secondary-container with 16dp
//     corners; menus have 16dp corners, rows with 4dp-rounded hover layers,
//     a selected choice in the tertiary container;
//   - primary tabs: the selected label in the primary colour over a 3dp
//     indicator as wide as the label, round at its top.
//
// On a desktop it keeps the precision-pointer sizes (40dp buttons, 40dp menu
// rows) and the Standard motion scheme's short fades.
type material3xEngine struct{ materialEngine }

func init() {
	for _, p := range material3xPacks() {
		RegisterPack(p)
	}
}

// EngineFor paints Material 3 packs with param "expressive" 1 with
// material3xEngine.
func (materialEngine) EngineFor(t ThemeTokens) Engine {
	if t.Params["expressive"] >= 1 && t.Params["gen"] >= 3 {
		return material3xEngine{}
	}
	return nil
}

// DefaultMetrics are Expressive's desktop sizes at the toolkit's 16px UI
// font: 40dp buttons in a 50px cell (the focus ring's room round them),
// 40dp menu rows, 48dp tabs, 40dp list segments with their 2dp gaps, the
// 44dp slider handle, the 52×32 switch.
func (material3xEngine) DefaultMetrics() ChromeMetrics {
	return ChromeMetrics{
		ViewFrame: 6,
		Radius:    20, RadiusSmall: 4,
		ControlH: 50, FieldH: 50, ComboH: 50,
		Checkbox: 30, Radio: 30,
		MenuItemH: 40, MenuBarH: 40, TabH: 48, RowH: 42,
		TitleBar: 48, HeaderH: 44, ProgressH: 14, SliderH: 44, Thumb: 4,
		Scroll: 12, Pad: 16, FieldPad: 14, FocusWidth: 3, Border: 1,
		ToolBarH: 56, StatusBarH: 28, SpinnerW: 30, SwitchW: 52, SwitchH: 32,
	}
}

// ---- colours ------------------------------------------------------------------------------------

// md3xRoles gives a Material 3 colour set the roles of the 2023 spec on:
// the tone-based surfaces (surface at tone 98 / 6), the containers for
// elevation (cards and the drawer surface-container-low, menus and bars
// surface-container, dialogs surface-container-high), 10% focus and pressed
// layers and Expressive's secondary-container inactive track. A pack pins
// any role by name.
func md3xRoles(l *Classic, c *mdSet) {
	n := mdCorePalette(l.X("seed", Hex("#6750a4"))).neutral
	tone := func(k string, light, dark float64) paintengine2d.Color {
		if c.dark {
			return l.X(k, n.tone(dark))
		}
		return l.X(k, n.tone(light))
	}
	c.surface = tone("surface", 98, 6)
	c.bg = tone("background", 98, 6)
	c.card = tone("surfaceContainerLow", 96, 10)
	c.menu = tone("surfaceContainer", 94, 12)
	c.bar = c.menu
	c.dialog = tone("surfaceContainerHigh", 92, 17)
	c.textDis = mdOver(c.surface, c.onSurface, 0.38)
	c.aFocus, c.aPress = 0.10, 0.10
	c.trackOff = c.secondaryC
}

// md3xSet is the roles only the Expressive era paints with.
type md3xSet struct {
	lowest, highest        paintengine2d.Color // surface-container-lowest, -highest
	segment                paintengine2d.Color // a list segment: a step above its group
	tertiaryC, onTertiaryC paintengine2d.Color
	secondary              paintengine2d.Color // the focus ring
	medium                 *Font               // labels: medium weight
}

type md3xKey struct{}

func md3xColors(l *Classic) *md3xSet {
	return l.Memo(md3xKey{}, func() any { return md3xBuild(l) }).(*md3xSet)
}

func md3xBuild(l *Classic) *md3xSet {
	c := mdColors(l)
	core := mdCorePalette(l.X("seed", Hex("#6750a4")))
	pick := func(k string, p mdPalette, light, dark float64) paintengine2d.Color {
		if c.dark {
			return l.X(k, p.tone(dark))
		}
		return l.X(k, p.tone(light))
	}
	x := &md3xSet{
		lowest:      pick("surfaceContainerLowest", core.neutral, 100, 4),
		highest:     pick("surfaceContainerHighest", core.neutral, 90, 22),
		tertiaryC:   pick("tertiaryContainer", core.tertiary, 90, 30),
		onTertiaryC: pick("onTertiaryContainer", core.tertiary, 30, 90),
		secondary:   c.secondary,
		medium:      l.WeightFont(WeightMedium),
	}
	x.segment = x.lowest
	if c.dark {
		x.segment = c.dialog // surface-container-high
	}
	return x
}

// md3xRoleKeys are the roles a Material 3 Expressive pack pins: the
// published baseline scheme of its seed.
var md3xRoleKeys = [...]string{
	"primary", "onPrimary", "primaryContainer", "onPrimaryContainer",
	"secondary", "onSecondary", "secondaryContainer", "onSecondaryContainer",
	"tertiaryContainer", "onTertiaryContainer", "error", "onError",
	"background", "surface", "onSurface", "surfaceVariant", "onSurfaceVariant",
	"outline", "outlineVariant", "inverseSurface", "inverseOnSurface",
	"surfaceContainerLowest", "surfaceContainerLow", "surfaceContainer",
	"surfaceContainerHigh", "surfaceContainerHighest",
}

// md3xScheme is the 2024 scheme of a seed as role → colour: the tonal
// palettes of engine_material_tone.go at the 2023 surface tones and the
// 2024 light on-containers (tone 30).
func md3xScheme(seed paintengine2d.Color, dark bool) map[string]paintengine2d.Color {
	p := mdCorePalette(seed)
	t := func(pal mdPalette, light, darkTone float64) paintengine2d.Color {
		if dark {
			return pal.tone(darkTone)
		}
		return pal.tone(light)
	}
	return map[string]paintengine2d.Color{
		"primary": t(p.primary, 40, 80), "onPrimary": t(p.primary, 100, 20),
		"primaryContainer": t(p.primary, 90, 30), "onPrimaryContainer": t(p.primary, 30, 90),
		"secondary": t(p.secondary, 40, 80), "onSecondary": t(p.secondary, 100, 20),
		"secondaryContainer": t(p.secondary, 90, 30), "onSecondaryContainer": t(p.secondary, 30, 90),
		"tertiaryContainer": t(p.tertiary, 90, 30), "onTertiaryContainer": t(p.tertiary, 30, 90),
		"error": t(p.err, 40, 80), "onError": t(p.err, 100, 20),
		"background": t(p.neutral, 98, 6), "surface": t(p.neutral, 98, 6), "onSurface": t(p.neutral, 10, 90),
		"surfaceVariant": t(p.variant, 90, 30), "onSurfaceVariant": t(p.variant, 30, 80),
		"outline": t(p.variant, 50, 60), "outlineVariant": t(p.variant, 80, 30),
		"inverseSurface": t(p.neutral, 20, 90), "inverseOnSurface": t(p.neutral, 95, 20),
		"surfaceContainerLowest": t(p.neutral, 100, 4), "surfaceContainerLow": t(p.neutral, 96, 10),
		"surfaceContainer": t(p.neutral, 94, 12), "surfaceContainerHigh": t(p.neutral, 92, 17),
		"surfaceContainerHighest": t(p.neutral, 90, 22),
	}
}

// md3xPalette is the shared palette of an Expressive scheme.
func md3xPalette(r map[string]paintengine2d.Color, dark bool) Palette {
	surface, on := r["surface"], r["onSurface"]
	cont := r["surfaceContainer"]
	hover := mdOver(cont, on, 0.08)
	success, warning := mdPalette{140, 40}.tone(40), mdPalette{70, 60}.tone(50)
	if dark {
		success, warning = mdPalette{140, 40}.tone(80), mdPalette{70, 60}.tone(80)
	}
	return Palette{
		Background: r["background"], Surface: surface, SurfaceAlt: r["surfaceContainerHighest"],
		Overlay: paintengine2d.RGBA(0, 0, 0, 0.32),
		Border:  r["outlineVariant"], Divider: r["outlineVariant"],
		Text: on, TextMuted: r["onSurfaceVariant"], TextOnAccent: r["onPrimary"],
		Accent: r["primary"], AccentHover: mdOver(r["primary"], r["onPrimary"], 0.08), AccentPress: mdOver(r["primary"], r["onPrimary"], 0.10),
		Danger: r["error"], Success: success, Warning: warning,
		Track: r["secondaryContainer"], Thumb: r["outline"],
		Field: surface, FieldBorder: r["outline"],
		Focus: r["primary"], Selection: r["primary"].WithAlpha(0.3),
		Shadow: paintengine2d.RGBA(0, 0, 0, 0.2), Highlight: hover,
		MenuHover: hover, MenuHoverBorder: hover, MenuGutter: cont,
		BevelLight: surface, BevelDark: r["outlineVariant"],
	}
}

// ---- shapes -------------------------------------------------------------------------------------

// md3xMargin is the room a control keeps round its face for the focus
// ring: 3dp of ring 2dp off the face.
func md3xMargin(l *Classic) float32 { return snap(l.S(5)) }

// md3xFace is a button's face inside its cell b: the ring's margin round
// it, or a pixel in a cell too short for one (the ring then lies over the
// face's edge).
func md3xFace(l *Classic, b paintengine2d.Rect) paintengine2d.Rect {
	b = mdSnap(b)
	m := md3xMargin(l)
	if b.Dx() < 4*m || b.Dy() < l.S(36) {
		m = mdPx(l)
	}
	if b.Dx() < 4*m || b.Dy() < 3*m {
		return b
	}
	return b.Inset(m)
}

// ring paints the focus indicator round face (corner r): 3dp of the
// secondary colour, 2dp out, kept inside clip.
func md3xRing(l *Classic, ctx *paintengine2d.Context, face, clip paintengine2d.Rect, r float32) {
	g, w := snap(l.S(2)), max(snap(l.S(3)), 2)
	col := md3xColors(l).secondary
	clip = mdSnap(clip)
	if face.Min.X-clip.Min.X < g+w-0.01 || face.Min.Y-clip.Min.Y < g+w-0.01 {
		// No room round the face (a short cell): the inner ring.
		mdRing(ctx, face, r, w, col)
		return
	}
	ctx.Save()
	ctx.ClipRect(clip)
	webBand(ctx, face.Inset(-(g + w)), r+g+w, face.Inset(-g), r+g, col)
	ctx.Restore()
}

// md3xButton paints a button face f (its cell b holds the shadow) and
// returns the label colour and the face's corner: the filled default
// button, a selected toggle in the secondary container (the square
// shape), the outlined button; each morphs to 8dp corners while pressed.
func md3xButton(l *Classic, ctx *paintengine2d.Context, f, b paintengine2d.Rect, st ControlState) (paintengine2d.Color, float32) {
	c := mdColors(l)
	if f.Dx() < 4 || f.Dy() < 4 {
		return c.text, 0
	}
	full := min(f.Dx(), f.Dy()) * 0.5
	r := full
	if l.square() {
		r = 0
	}
	dis := st.Disabled()
	pressed := st.Pressed() && !dis
	if st.Toggle() && st.Checked() {
		r = min(l.rx(12), full)
	}
	if pressed {
		r = min(l.rx(8), full)
	}
	a := c.layer(st)
	plain := int(l.P("plain", 0))
	switch {
	case st.Primary():
		if dis {
			mdFill(ctx, f, r, c.onSurface.WithAlpha(0.10))
			return c.textDis, r
		}
		if st.Hovered() && !pressed {
			c.elevate(l, ctx, f, b, r, 1)
		}
		mdFill(ctx, f, r, c.primary)
		if a > 0 {
			mdFill(ctx, f, r, c.onPrimary.WithAlpha(a))
		}
		return c.onPrimary, r
	case (st.Toggle() && st.Checked()) || plain == 2:
		if dis {
			mdFill(ctx, f, r, c.onSurface.WithAlpha(0.10))
			return c.textDis, r
		}
		mdFill(ctx, f, r, c.secondaryC)
		if a > 0 {
			mdFill(ctx, f, r, c.onSecondaryC.WithAlpha(a))
		}
		return c.onSecondaryC, r
	}
	edge, fg := c.outlineVar, c.onSurfaceVar
	if dis {
		edge, fg = c.onSurface.WithAlpha(0.12), c.textDis
	} else if a > 0 {
		mdFill(ctx, f, r, c.onSurfaceVar.WithAlpha(a))
	}
	mdRing(ctx, f, r, mdPx(l), edge)
	return fg, r
}

// md3xField paints the outlined text field box: a 1dp outline, 3dp of the
// primary colour while focused (Expressive), the on-surface colour under
// the pointer, 4dp corners; the floated label's notch between n0 and n1.
func md3xField(l *Classic, ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, n0, n1 float32) {
	c := mdColors(l)
	if box.Dx() < 4 || box.Dy() < 4 {
		return
	}
	lw := mdPx(l)
	r := min(l.rx(4), box.Dy()*0.5)
	edge, w := c.fieldLine, lw
	switch {
	case st.Disabled():
		edge = c.onSurface.WithAlpha(0.12)
	case st.Focused() || st.Pressed():
		edge, w = c.primary, max(snap(l.S(3)), 2)
	case st.Hovered():
		edge = c.text
	}
	ctx.DrawRoundRect(box, r, r, paintengine2d.Fill(c.surface))
	if n1 > n0 {
		ctx.Save()
		p := paintengine2d.NewPath()
		p.AddRect(box)
		p.AddRect(paintengine2d.Rect{Min: paintengine2d.Pt(n0, box.Min.Y), Max: paintengine2d.Pt(n1, box.Min.Y+w+lw)})
		ctx.ClipPathRule(p, paintengine2d.FillEvenOdd)
		mdRing(ctx, box, r, w, edge)
		ctx.Restore()
		return
	}
	mdRing(ctx, box, r, w, edge)
}

// ---- parts --------------------------------------------------------------------------------------

func (e material3xEngine) Face(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	c := mdColors(l)
	switch role {
	case RoleButton:
		fg, _ := md3xButton(l, ctx, md3xFace(l, b), mdSnap(b), st)
		return fg
	case RoleField, RoleCombo:
		md3xField(l, ctx, mdSnap(b), st, 0, 0)
		if st.Disabled() {
			return c.textDis
		}
		return c.text
	case RoleMenu:
		if (st.Hovered() || st.Pressed()) && !st.Disabled() {
			e.MenuHighlight(l, ctx, b, st.Pressed())
		}
		return c.text
	case RoleBar:
		ctx.DrawRect(b, paintengine2d.Fill(c.bar))
		return c.text
	}
	return e.materialEngine.Face(l, ctx, b, role, st)
}

// MenuHighlight is a menu row's state layer in its 4dp-rounded item shape
// (the pressed layer while held, or for an open menu-bar title).
func (material3xEngine) MenuHighlight(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	c := mdColors(l)
	a := c.aHover
	if attachBottom {
		a = c.aPress
	}
	b = mdSnap(b)
	mdFill(ctx, b, min(l.rx(4), b.Dy()*0.5), c.onSurface.WithAlpha(a))
}

// DrawFocusRing is the focus indicator inside b: 3dp of the secondary
// colour.
func (material3xEngine) DrawFocusRing(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	b = mdSnap(b)
	w := max(snap(l.S(3)), 2)
	if b.Dx() < 3*w || b.Dy() < 3*w {
		return
	}
	mdRing(ctx, b, min(l.rx(12), b.Dy()*0.5), w, md3xColors(l).secondary)
}

// ---- controls -----------------------------------------------------------------------------------

func (e material3xEngine) DrawButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	f := md3xFace(l, b)
	fg, r := md3xButton(l, ctx, f, mdSnap(b), st)
	l.drawFittedText(ctx, md3xColors(l).medium, label, f, fg, AlignCenter, l.S(16))
	if st.Focused() && !st.Disabled() {
		md3xRing(l, ctx, f, b, r)
	}
}

func (e material3xEngine) DrawTextField(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
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
	md3xField(l, ctx, box, st, n0, n1)
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

func (e material3xEngine) DrawComboBox(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	c := mdColors(l)
	b = mdSnap(b)
	if open {
		st |= StateFocused
	}
	md3xField(l, ctx, b, st, 0, 0)
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

func (e material3xEngine) DrawSpinner(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	if !st.Frameless() {
		md3xField(l, ctx, mdSnap(b), st&^StateFocused, 0, 0)
	}
	e.materialEngine.DrawSpinner(l, ctx, b, st|StateFrameless, upHover, downHover, upPress, downPress)
}

// DrawTabBar is the primary tabs' strip: the surface over an
// outline-variant divider.
func (material3xEngine) DrawTabBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mdColors(l)
	b = mdSnap(b)
	ctx.DrawRect(b, paintengine2d.Fill(c.surface))
	px := mdPx(l)
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-px, b.Dx(), px), paintengine2d.Fill(c.outlineVar))
}

// DrawTab is a primary tab: the label in on-surface-variant (the selected
// one in the primary colour over a 3dp indicator as wide as the label,
// rounded at its top), the state layer across the tab.
func (e material3xEngine) DrawTab(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	c := mdColors(l)
	b = mdSnap(b)
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	if a := c.layer(st); a > 0 {
		ctx.DrawRect(b, paintengine2d.Fill(c.onSurface.WithAlpha(a)))
	}
	f := md3xColors(l).medium
	fg := c.onSurfaceVar
	if selected {
		fg = c.primary
	}
	if st.Disabled() {
		fg = c.textDis
	}
	l.drawFittedText(ctx, f, label, b, fg, AlignCenter, l.S(12))
	if selected {
		w := max(min(f.Advance(label), b.Dx()-l.S(8)), l.S(24))
		h := max(snap(l.S(3)), 2)
		ind := paintengine2d.XYWH(snap((b.Min.X+b.Max.X-w)*0.5), b.Max.Y-h, snap(w), h)
		col := c.primary
		if st.Disabled() {
			col = c.textDis
		}
		ctx.DrawPath(RoundRectPath(ind, h, h, 0, 0), paintengine2d.Fill(col))
	}
	if st.Focused() && !st.Disabled() {
		e.DrawFocusRing(l, ctx, b.Inset(snap(l.S(2))))
	}
}

// md3xTravel is where the slider handle's centre runs.
func (material3xEngine) SliderTravel(l *Classic, b paintengine2d.Rect) (x0, x1 float32) {
	b = mdSnap(b)
	m := snap(l.S(2))
	return b.Min.X + m, b.Max.X - m
}

// DrawSlider is the 2025 slider: a 16dp track split round the 4dp bar
// handle by 6dp gaps, the active part primary, the inactive
// secondary-container with a 4dp stop dot at its end; inner corners 2dp,
// outer ones full. Pressed or focused, the handle narrows to 2dp.
func (e material3xEngine) DrawSlider(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	c := mdColors(l)
	b = mdSnap(b)
	t = clamp1(t)
	th := min(snap(l.S(16)), snap(b.Dy()*0.45))
	hh := min(snap(l.S(44)), b.Dy())
	if th < 4 || b.Dx() < l.S(24) {
		return
	}
	dis := st.Disabled()
	hw := snap(l.S(4))
	if (st.Pressed() || st.Focused()) && !dis {
		hw = max(snap(l.S(2)), 1)
	}
	gap := snap(l.S(6))
	x0, x1 := e.SliderTravel(l, b)
	hx := snap(x0 + (x1-x0)*t)
	cy := snap((b.Min.Y + b.Max.Y) * 0.5)
	on, off, dot := c.primary, c.secondaryC, c.primary
	if dis {
		on, off, dot = c.onSurface.WithAlpha(0.38), c.onSurface.WithAlpha(0.12), c.onSurface.WithAlpha(0.38)
	}
	outer, inner := th*0.5, min(l.rx(2), th*0.5)
	if l.square() {
		outer = 0
	}
	top := cy - th*0.5
	if a := hx - hw*0.5 - gap - b.Min.X; a >= 1 {
		ctx.DrawPath(RoundRectPath(paintengine2d.XYWH(b.Min.X, top, a, th), min(outer, a*0.5), min(inner, a*0.5), min(inner, a*0.5), min(outer, a*0.5)), paintengine2d.Fill(on))
	}
	if s := hx + hw*0.5 + gap; b.Max.X-s >= 1 {
		w := b.Max.X - s
		ctx.DrawPath(RoundRectPath(paintengine2d.XYWH(s, top, w, th), min(inner, w*0.5), min(outer, w*0.5), min(outer, w*0.5), min(inner, w*0.5)), paintengine2d.Fill(off))
		if w > th {
			ctx.DrawCircle(paintengine2d.Pt(b.Max.X-th*0.5, cy), max(l.S(2), 1), paintengine2d.Fill(dot))
		}
	}
	handle := paintengine2d.XYWH(hx-hw*0.5, cy-hh*0.5, hw, hh)
	ctx.DrawRoundRect(handle, hw*0.5, hw*0.5, paintengine2d.Fill(on))
}

// md3xWave strokes the wavy indicator from x0 to x1 about cy: a sine of
// amplitude amp and wavelength wl, phase shifted by ph (radians), the
// amplitude easing in over the first and last quarter wavelength.
func md3xWave(ctx *paintengine2d.Context, x0, x1, cy, amp, wl, ph, w float32, col paintengine2d.Color) {
	if x1-x0 < 0.5 || col.A <= 0 {
		return
	}
	p := paintengine2d.NewPath()
	k := 2 * math.Pi / float64(wl)
	ease := wl * 0.25
	step := max(w*0.4, 1)
	first := true
	for x := x0; ; x += step {
		if x > x1 {
			x = x1
		}
		a := amp * min((x-x0)/ease, (x1-x)/ease, 1)
		y := cy + a*float32(math.Sin(k*float64(x-x0)+float64(ph)))
		if first {
			p.MoveTo(x, y)
			first = false
		} else {
			p.LineTo(x, y)
		}
		if x >= x1 {
			break
		}
	}
	ctx.DrawPath(p, paintengine2d.Paint{Color: col, Style: paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: w, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
}

// DrawProgressBar is Expressive's wavy linear indicator: the value a 4dp
// primary sine stroke (3dp amplitude, 40dp wavelength), 4dp clear of the
// straight secondary-container track, which ends in a 4dp primary stop dot;
// busy, a wavelength-20dp wave travels across.
func (e material3xEngine) DrawProgressBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	c := mdColors(l)
	b = mdSnap(b)
	sw := max(snap(l.S(4)), 2)
	if b.Dx() < 4*sw || b.Dy() < sw {
		return
	}
	amp := min(l.S(3), max((b.Dy()-sw)*0.5, 0))
	cy := (b.Min.Y + b.Max.Y) * 0.5
	x0, x1 := b.Min.X+sw*0.5, b.Max.X-sw*0.5
	gap := snap(l.S(4))
	on, off := c.primary, c.secondaryC
	if st.Disabled() {
		on, off = c.onSurface.WithAlpha(0.38), c.onSurface.WithAlpha(0.12)
	}
	line := func(a, z float32) {
		if z-a < 0.5 {
			return
		}
		ctx.DrawLine(paintengine2d.Pt(a, cy), paintengine2d.Pt(z, cy), paintengine2d.Paint{Color: off, Style: paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: sw, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4}})
	}
	ctx.Save()
	defer ctx.Restore()
	ctx.ClipRect(b)
	if indeterminate {
		phase -= float32(math.Floor(float64(phase)))
		span := (x1 - x0) * 0.45
		s := x0 + (x1-x0+span)*phase - span
		z := s + span
		line(x0, s-gap-sw)
		line(z+gap+sw, x1)
		md3xWave(ctx, max(s, x0), min(z, x1), cy, amp, l.S(20), -phase*2*math.Pi*4, sw, on)
		return
	}
	xe := x0 + (x1-x0)*clamp1(t)
	if t > 0 {
		md3xWave(ctx, x0, xe, cy, amp, l.S(40), 0, sw, on)
	}
	start := x0
	if t > 0 {
		start = xe + gap + sw
	}
	if x1-start >= 0.5 {
		line(start, x1)
		ctx.DrawCircle(paintengine2d.Pt(x1, cy), sw*0.5, paintengine2d.Fill(on))
	}
}

// DrawMenuFrame is a vertical menu: surface-container with 16dp corners
// (the level 2 shadow is PopupShadow's).
func (material3xEngine) DrawMenuFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	c := mdColors(l)
	b = mdSnap(b)
	r := min(l.rx(16), b.Dy()*0.25)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.menu))
}

// DrawMenuItem is a vertical menu's row: the state layer in its 4dp item
// shape inset from the menu's sides, a selected choice in the tertiary
// container with 12dp corners, checks, radios and icons in the leading
// column, dim shortcuts, the submenu triangle.
func (e material3xEngine) DrawMenuItem(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	c := mdColors(l)
	x := md3xColors(l)
	ch := MenuChromeFor(l)
	px := mdPx(l)
	k := min(snap(l.S(4)), ch.PadL)
	item := mdSnap(paintengine2d.XYWH(b.Min.X-ch.PadL+k, b.Min.Y+px, b.Dx()+ch.PadL+ch.PadR-2*k, b.Dy()-2*px))
	if row.Separator {
		y := snap((b.Min.Y + b.Max.Y) * 0.5)
		ctx.DrawRect(paintengine2d.XYWH(item.Min.X+l.S(8), y, item.Dx()-l.S(16), px), paintengine2d.Fill(c.outlineVar))
		return
	}
	fg, sc := c.text, c.onSurfaceVar
	if row.Radio && row.Checked && !st.Disabled() {
		mdFill(ctx, item, min(l.rx(12), item.Dy()*0.5), x.tertiaryC)
		fg, sc = x.onTertiaryC, x.onTertiaryC
	}
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		e.MenuHighlight(l, ctx, item, st.Pressed())
	}
	font := l.body
	if st.Disabled() {
		fg, sc = c.textDis, c.textDis
		font = l.muted
	}
	mdGutter(l, ctx, b, ch, row, fg)
	ty := b.Min.Y + (b.Dy()-font.Height())*0.5
	right := b.Max.X
	if row.Submenu {
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, ch.SubmenuArrow, b.Dy())
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

// ViewFrameInsets: a list's segments sit 6px inside its 16dp-rounded group.
func (material3xEngine) ViewFrameInsets(l *Classic) Insets {
	v := snap(l.S(6))
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// DrawViewFrame is a list's group: surface-container in 16dp corners (a
// navigation drawer keeps its container), ringed by the focus indicator
// while the view has focus.
func (e material3xEngine) DrawViewFrame(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	c := mdColors(l)
	if st.Sidebar() {
		if !b.Empty() {
			ctx.DrawRect(b, paintengine2d.Fill(c.drawer()))
		}
		return
	}
	b = mdSnap(b)
	if b.Dx() < 8 || b.Dy() < 8 {
		return
	}
	r := min(l.rx(16), b.Dy()*0.25, b.Dx()*0.25)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(c.menu))
	if st.Focused() && !st.Disabled() {
		mdRing(ctx, b, r, max(snap(l.S(3)), 2), md3xColors(l).secondary)
	}
}

// ViewBackground: a list's rows sit on its group's container.
func (e material3xEngine) ViewBackground(l *Classic, st ControlState) paintengine2d.Color {
	if st.Sidebar() {
		return e.materialEngine.ViewBackground(l, st)
	}
	return mdColors(l).menu
}

// md3xSegment paints a segmented list row into b and returns its label
// colour: a container a step brighter than its group's (the lowest in the
// light scheme, the high one in the dark) with 4dp corners and 2dp gaps
// between rows, the selected row secondary-container with 16dp corners (a
// neutral one in an inactive window), the state layer over it.
func md3xSegment(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) paintengine2d.Color {
	c := mdColors(l)
	x := md3xColors(l)
	b = mdSnap(b)
	v := snap(l.S(1))
	box := paintengine2d.Rect{Min: paintengine2d.Pt(b.Min.X, b.Min.Y+v), Max: paintengine2d.Pt(b.Max.X, b.Max.Y-v)}
	if box.Dy() < 4 {
		box = b
	}
	r := min(l.rx(4), box.Dy()*0.5)
	fill, fg, layer := x.segment, c.text, c.onSurface
	if st.Checked() {
		r = min(l.rx(16), box.Dy()*0.5)
		fill, fg, layer = c.secondaryC, c.onSecondaryC, c.onSecondaryC
		if st.Backdrop() {
			fill, fg = mdOver(x.segment, c.onSurface, 0.08), c.text
		}
	}
	if st.Disabled() {
		fg = c.textDis
	}
	mdFill(ctx, box, r, fill)
	a := float32(0)
	switch {
	case st.Disabled():
	case st.Pressed():
		a = c.aPress
	case st.Hovered():
		a = c.aHover
	}
	if a > 0 {
		mdFill(ctx, box, r, layer.WithAlpha(a))
	}
	if st.Focused() && !st.Disabled() {
		mdFill(ctx, box, r, layer.WithAlpha(c.aFocus))
	}
	return fg
}

func (e material3xEngine) DrawListRow(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	if st.Sidebar() {
		e.materialEngine.DrawListRow(l, ctx, b, st, label)
		return
	}
	fg := md3xSegment(l, ctx, b, st)
	pad := l.S(16)
	l.drawFittedText(ctx, l.body, label, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-2*pad, b.Dy()), fg, AlignStart, 0)
}

// DrawToolBar is a docked tool bar: surface-container, no divider.
func (material3xEngine) DrawToolBar(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(mdSnap(b), paintengine2d.Fill(mdColors(l).bar))
}

// ---- accent -------------------------------------------------------------------------------------

// Accented takes the desktop's accent as the seed, as Material You does:
// the pack's pinned baseline roles give way to the scheme the seed makes
// (engine_material_tone.go's palettes at the 2023–24 tones). The pack's own
// seed gives the pack back: its published scheme is that seed's.
func (material3xEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
	if accentSame(accentX(tok, "seed", Hex("#6750a4")), accent) {
		return tok
	}
	dark := accentDark(tok)
	tok = CloneTokenMaps(tok)
	for _, k := range md3xRoleKeys {
		delete(tok.Extra, k)
	}
	tok.Extra["seed"] = accent
	tok.Palette = md3xPalette(md3xScheme(accent, dark), dark)
	mdChrome(&tok)
	tok.Pressed, tok.Disabled = ChromeState{}, ChromeState{} // the resolver's
	return tok.Resolve()
}

// ---- packs --------------------------------------------------------------------------------------

// md3xBaseline is Material 3's baseline scheme (seed #6750A4) as the token
// database publishes it, with the 2023 surface containers and the 2024
// tone-30 light on-containers.
var md3xBaseline = map[bool]map[string]string{
	false: {
		"seed": "#6750a4", "primary": "#6750a4", "onPrimary": "#ffffff", "primaryContainer": "#eaddff", "onPrimaryContainer": "#4f378b",
		"secondary": "#625b71", "onSecondary": "#ffffff", "secondaryContainer": "#e8def8", "onSecondaryContainer": "#4a4458",
		"tertiaryContainer": "#ffd8e4", "onTertiaryContainer": "#633b48", "error": "#b3261e", "onError": "#ffffff",
		"background": "#fef7ff", "surface": "#fef7ff", "onSurface": "#1d1b20", "surfaceVariant": "#e7e0ec", "onSurfaceVariant": "#49454f",
		"outline": "#79747e", "outlineVariant": "#cac4d0", "inverseSurface": "#322f35", "inverseOnSurface": "#f5eff7",
		"surfaceContainerLowest": "#ffffff", "surfaceContainerLow": "#f7f2fa", "surfaceContainer": "#f3edf7",
		"surfaceContainerHigh": "#ece6f0", "surfaceContainerHighest": "#e6e0e9",
	},
	true: {
		"seed": "#6750a4", "primary": "#d0bcff", "onPrimary": "#381e72", "primaryContainer": "#4f378b", "onPrimaryContainer": "#eaddff",
		"secondary": "#ccc2dc", "onSecondary": "#332d41", "secondaryContainer": "#4a4458", "onSecondaryContainer": "#e8def8",
		"tertiaryContainer": "#633b48", "onTertiaryContainer": "#ffd8e4", "error": "#f2b8b5", "onError": "#601410",
		"background": "#141218", "surface": "#141218", "onSurface": "#e6e0e9", "surfaceVariant": "#49454f", "onSurfaceVariant": "#cac4d0",
		"outline": "#938f99", "outlineVariant": "#49454f", "inverseSurface": "#e6e0e9", "inverseOnSurface": "#322f35",
		"surfaceContainerLowest": "#0f0d13", "surfaceContainerLow": "#1d1b20", "surfaceContainer": "#211f26",
		"surfaceContainerHigh": "#2b2930", "surfaceContainerHighest": "#36343b",
	},
}

func material3xPacks() []ThemePack {
	pack := func(name, label, summary string, fam ThemeName) ThemePack {
		dark := fam == ThemeDark
		roles := md3xBaseline[dark]
		r := make(map[string]paintengine2d.Color, len(roles))
		for k, v := range roles {
			r[k] = Hex(v)
		}
		return mdPack(name, label, 2025, summary, fam, md3xPalette(r, dark), ChromeMetrics{}, roles,
			map[string]float32{"gen": 3, "expressive": 1})
	}
	return []ThemePack{
		pack("material3x", "Material 3 Expressive",
			"Material 3 Expressive: tone-based surface containers, round buttons that square off when pressed, the split slider, wavy progress, segmented lists, a secondary focus ring.",
			ThemeLight),
		pack("material3x-night", "Material 3 Expressive Dark",
			"Material 3 Expressive's dark baseline: #141218 surfaces, #211f26 containers, the #d0bcff primary.",
			ThemeDark),
	}
}
