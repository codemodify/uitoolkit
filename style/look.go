package style

import "github.com/codemodify/paintengine2d"

// Classic is the stock LookAndFeel (graphite chrome + blue accent).
// Theme, corner policy, and icon set are first-class (see Appearance).
type Classic struct {
	palette  Palette
	metrics  Metrics
	name     string
	pack     string
	corners  CornerStyle
	icons    IconSetName
	iconSize IconSize
	// density and scale are carried explicitly so a theme switch can
	// rebuild metrics from scratch instead of overlaying the new pack onto
	// the previous pack's values (which leaked control heights) or
	// re-deriving the display scale from FontSize (which is wrong as soon
	// as density changed the 1× size).
	density Density
	scale   float32
	tokens  ThemeTokens
	engine  Engine
	memo    lookMemo // engine-derived paint data, built once per look
	body    *Font
	title   *Font
	bold    *Font
	muted   *Font
	onAcc   *Font
	mono    *Font
}

// NewClassic builds fonts for p. Name is "dark" or "light" typically.
// Corners are round and icons are classic unless Metrics radii are already 0.
func NewClassic(name string, p Palette, m Metrics) *Classic {
	corners := CornersRound
	if m.FontSize >= 8 && m.Radius <= 0 && m.RadiusSmall <= 0 {
		corners = CornersSquare
	}
	return newClassic(name, p, m, corners, IconSetClassic, IconSizeMedium)
}

func newClassic(name string, p Palette, m Metrics, corners CornerStyle, icons IconSetName, iconSize IconSize, tok ...ThemeTokens) *Classic {
	if m.FontSize < 8 {
		m = DefaultMetrics()
	}
	corners = ParseCorners(string(corners))
	icons = ParseIconSet(string(icons))
	iconSize = ParseIconSize(string(iconSize))
	if corners == CornersSquare {
		m.Radius = 0
		m.RadiusSmall = 0
	}
	m = ApplyIconSize(m, iconSize)
	// Lock-in: Classic always ships Titillium Web + JetBrains Mono.
	// Metrics.FontFamily cannot select mononoki (or any other face).
	m.FontFamily = DefaultFontFamily
	m.MonoFamily = DefaultMonoFamily
	var tokens ThemeTokens
	if len(tok) > 0 {
		tokens = tok[0]
	}
	if tokens.Empty() {
		if pack, ok := builtinEraPack(name); ok {
			tokens = pack.Tokens
		} else {
			tokens = ThemeTokens{Family: ParseTheme(name), Palette: p}
		}
	}
	if colorUnset(tokens.Palette.Background) && colorUnset(tokens.Palette.Surface) {
		tokens.Palette = overlayPalette(p, tokens.Palette)
	}
	tokens = tokens.Resolve()
	p = tokens.Palette
	p = ResolveMenuChrome(p)
	p = ResolveBevelChromeFor(p, tokens.Family)
	density, scale := inferDensityScale(m)
	return &Classic{
		palette:  p,
		metrics:  m,
		name:     name,
		pack:     StarterName(ParseTheme(name)),
		corners:  corners,
		icons:    icons,
		iconSize: iconSize,
		density:  density,
		scale:    scale,
		tokens:   tokens,
		engine:   engineFor(tokens),
		// OpenType atlases (Titillium / JetBrains Mono); Color tints at draw.
		body:  BakeFont(m.FontSize, p.Text),
		title: BakeTitleFont(m.TitleSize, p.Text),
		bold:  BakeFamily(FamilyUI, WeightBold, m.FontSize, p.Text),
		muted: BakeFont(m.FontSize, p.TextMuted),
		onAcc: BakeFont(m.FontSize, p.TextOnAccent),
		mono:  BakeMonoFont(m.FontSize, p.Text),
	}
}

// inferDensityScale recovers the density and display scale of metrics that
// were not built through the style rebuild path (a caller-supplied
// [NewClassic] Metrics). An exact rebuild wins; otherwise default / 1×.
func inferDensityScale(m Metrics) (Density, float32) {
	best, bestScale, found := DensityDefault, float32(1), false
	near := func(a, b float32) bool {
		d := a - b
		return d < 0.05 && d > -0.05
	}
	for _, d := range []Density{DensityDefault, DensityCompact, DensityRelaxed} {
		base := ApplyDensity(DefaultMetrics(), d)
		if base.FontSize <= 0 || m.FontSize <= 0 {
			continue
		}
		s := m.FontSize / base.FontSize
		if s <= 0 {
			continue
		}
		want := ScaleMetrics(base, s)
		if !near(m.RowH, want.RowH) || !near(m.MenuItemH, want.MenuItemH) || !near(m.Pad, want.Pad) {
			continue
		}
		off := s - 1
		if off < 0 {
			off = -off
		}
		bestOff := bestScale - 1
		if bestOff < 0 {
			bestOff = -bestOff
		}
		if !found || off < bestOff {
			best, bestScale, found = d, s, true
		}
	}
	return best, bestScale
}

func (l *Classic) setDensity(d Density) *Classic {
	if l != nil {
		l.density = d
	}
	return l
}

func (l *Classic) setScale(s float32) *Classic {
	if l != nil && s > 0 {
		l.scale = s
	}
	return l
}

// Density is the chrome density this look was built at.
func (l *Classic) Density() Density {
	if l == nil {
		return DensityDefault
	}
	return l.density
}

// Scale is the display scale this look was built at (1 = unscaled).
func (l *Classic) Scale() float32 {
	if l == nil || l.scale <= 0 {
		return 1
	}
	return l.scale
}

// DarkLook is the default night skin (round + classic icons).
func DarkLook() *Classic { return NewClassic("dark", Dark(), DefaultMetrics()) }

// LightLook is the paper skin (round + classic icons).
func LightLook() *Classic { return NewClassic("light", Light(), DefaultMetrics()) }

// WithScale rebuilds a Classic look with scaled metrics and glyph atlases.
// Other LookAndFeel implementations are returned unchanged.
func WithScale(look LookAndFeel, scale float32) LookAndFeel {
	if look == nil || scale <= 0 || scale == 1 {
		return look
	}
	c, ok := look.(*Classic)
	if !ok {
		return look
	}
	return newClassic(c.Name(), c.Palette(), ScaleMetrics(c.Metrics(), scale), c.Corners(), c.Icons(), c.IconSize(), c.Tokens()).
		setPack(c.Pack()).setDensity(c.Density()).setScale(c.Scale() * scale)
}

func (l *Classic) Name() string           { return l.name }
func (l *Classic) Palette() Palette       { return l.palette }
func (l *Classic) Metrics() Metrics       { return l.metrics }
func (l *Classic) Corners() CornerStyle   { return ParseCorners(string(l.corners)) }
func (l *Classic) Icons() IconSetName     { return ParseIconSet(string(l.icons)) }
func (l *Classic) IconSize() IconSize     { return ParseIconSize(string(l.iconSize)) }
func (l *Classic) Appearance() Appearance { return LookAppearance(l) }

// Pack is the color theme name (look.json "theme"). Empty falls back
// to the embedded starter matching the palette.
func (l *Classic) Pack() string {
	if l == nil {
		return DefaultThemeName
	}
	if l.pack != "" {
		return l.pack
	}
	return StarterName(ParseTheme(l.name))
}

func (l *Classic) setPack(name string) *Classic {
	if l == nil {
		return l
	}
	if name != "" {
		l.pack = name
	}
	return l
}
func (l *Classic) Font() *Font      { return l.body }
func (l *Classic) TitleFont() *Font { return l.title }
func (l *Classic) BoldFont() *Font {
	if l.bold != nil {
		return l.bold
	}
	return l.body
}
func (l *Classic) MutedFont() *Font    { return l.muted }
func (l *Classic) OnAccentFont() *Font { return l.onAcc }
func (l *Classic) MonoFont() *Font     { return l.mono }

func (l *Classic) faceOrBody(face *Font) *Font {
	if face != nil {
		return face
	}
	return l.body
}

func (l *Classic) mutedFor(face *Font) *Font {
	if face != nil && face.Family == FamilyMono {
		return BakeMonoFont(face.Size, l.palette.TextMuted)
	}
	return l.muted
}

func (l *Classic) baseDrawPanel(ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	p := l.palette
	fill := p.Surface
	if raised {
		fill = p.SurfaceAlt
	}
	st := StateNone
	if raised {
		st = StateHovered
	}
	l.paintBezel(ctx, b, fill, p.Border, rolePanel, st)
}

func (l *Classic) baseDrawButton(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	if st.Pressed() && !st.Disabled() && l.tokensOr().Bevel == BevelClassic3D {
		b = b.Translate(paintengine2d.Pt(0, 1))
	}
	col := l.paintFace(ctx, b, roleButton, st)
	if st.Focused() {
		l.DrawFocusRing(ctx, b)
	}
	face := l.body
	if st.Disabled() {
		face = l.muted
	} else if nearColor(col, l.palette.TextOnAccent) {
		face = l.onAcc
	}
	l.drawFittedText(ctx, face, label, b, col, AlignCenter, 8)
}

func (l *Classic) baseDrawLabel(ctx *paintengine2d.Context, b paintengine2d.Rect, text string, col paintengine2d.Color, align Align) {
	col = l.onWindow(col)
	f := l.fontFor(col)
	l.drawFittedText(ctx, f, text, b, col, align, 2)
}

// onWindow nudges a coloured label or glyph until it reads on the window
// background (3:1): Classic 95 Dark drew accent links and info icons navy
// on dark grey. Colours that already read are returned unchanged.
func (l *Classic) onWindow(col paintengine2d.Color) paintengine2d.Color {
	bg := l.palette.Background
	if colorUnset(col) || colorUnset(bg) || ContrastRatio(col, bg) >= 3 {
		return col
	}
	toward := paintengine2d.RGB(1, 1, 1)
	if RelLuminance(bg) > 0.4 {
		toward = paintengine2d.RGB(0, 0, 0)
	}
	for t := float32(0.1); t <= 1; t += 0.1 {
		c := Mix(col, toward, t)
		if ContrastRatio(c, bg) >= 3 {
			c.A = col.A
			return c
		}
	}
	return col
}

func (l *Classic) baseDrawCheckbox(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	p := l.palette
	m := l.metrics
	side := m.Checkbox
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	if checked {
		st |= StateChecked
	}
	l.eng().CheckIndicator(l, ctx, box, st, checked)
	if label != "" {
		f := l.body
		if st.Disabled() {
			f = l.muted
		}
		lb := paintengine2d.XYWH(box.Max.X+8, b.Min.Y, b.Max.X-(box.Max.X+8), b.Dy())
		l.drawFittedText(ctx, f, label, lb, p.Text, AlignStart, 0)
		if st.Focused() {
			l.DrawFocusRing(ctx, labelFocusRect(f, label, lb, b))
		}
	} else if st.Focused() {
		l.DrawFocusRing(ctx, box.Intersect(b))
	}
}

func (l *Classic) baseDrawSlider(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	p := l.palette
	m := l.metrics
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	cy := (b.Min.Y + b.Max.Y) * 0.5
	x0 := b.Min.X + m.Thumb*0.5
	x1 := b.Max.X - m.Thumb*0.5
	track := paintengine2d.XYWH(x0, cy-2, x1-x0, 4)
	tr := l.rx(2)
	trackCol := p.Track
	fillCol := p.Accent
	if st.Disabled() {
		trackCol = p.Surface
		fillCol = p.Divider
	}
	ctx.DrawRoundRect(track, tr, tr, paintengine2d.Fill(trackCol))
	fillW := (x1 - x0) * t
	ctx.DrawRoundRect(paintengine2d.XYWH(x0, cy-2, fillW, 4), tr, tr, paintengine2d.Fill(fillCol))
	tx := x0 + (x1-x0)*t
	rad := m.Thumb * 0.5
	if !st.Disabled() && (st.Hovered() || st.Pressed()) {
		rad += 1
	}
	ctx.DrawCircle(paintengine2d.Pt(tx, cy+1), rad+1, paintengine2d.Fill(p.Shadow))
	col := fillCol
	if !st.Disabled() {
		if st.Pressed() {
			col = p.AccentPress
		} else if st.Hovered() {
			col = p.AccentHover
		}
	}
	ctx.DrawCircle(paintengine2d.Pt(tx, cy), rad, paintengine2d.Fill(col))
	if !st.Disabled() {
		ctx.DrawCircle(paintengine2d.Pt(tx-rad*0.25, cy-rad*0.25), rad*0.35, paintengine2d.Fill(p.Highlight))
	}
	if st.Focused() {
		l.DrawFocusRing(ctx, paintengine2d.XYWH(tx-rad-3, cy-rad-3, rad*2+6, rad*2+6).Intersect(b))
	}
}

func (l *Classic) baseDrawTextField(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	p := l.palette
	m := l.metrics
	l.paintFace(ctx, b, roleField, st)
	if st.Focused() && !st.Disabled() && l.eng().FieldFocusRing(l) {
		l.DrawFocusRing(ctx, b)
	}
	pad := m.FieldPad
	if pad <= 0 {
		pad = 8
	}
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy())
	ctx.Save()
	ctx.ClipRect(inner)
	f := l.faceOrBody(face)
	muted := l.mutedFor(face)
	show := text
	font := f
	if text == "" && placeholder != "" && !st.Focused() {
		show = placeholder
		font = muted
	}
	ty := inner.Min.Y + (inner.Dy()-font.Height())*0.5
	ox := inner.Min.X - scrollX
	var selBox paintengine2d.Rect
	var selCol paintengine2d.Color
	if selA != selB && text != "" {
		if selA > selB {
			selA, selB = selB, selA
		}
		x0 := ox + f.CaretX(text, selA)
		x1 := ox + f.CaretX(text, selB)
		sel := p.Selection
		if !st.Focused() {
			sel = p.Selection.WithAlpha(0.14)
		}
		selBox = paintengine2d.XYWH(x0, ty+1, x1-x0, f.Height()-2)
		ctx.DrawRect(selBox, paintengine2d.Fill(sel))
		selCol = l.selectedText(sel)
	}
	col := l.fieldText()
	if font == muted {
		col = p.TextMuted
	}
	font.Draw(ctx, show, paintengine2d.Pt(ox, ty), col)
	if !selBox.Empty() && selCol != col {
		// Selected text in the highlighted-text colour (Qt's
		// HighlightedText): a dark selection hid black text.
		ctx.Save()
		ctx.ClipRect(selBox)
		font.Draw(ctx, show, paintengine2d.Pt(ox, ty), selCol)
		ctx.Restore()
	}
	if st.Focused() && blink && text == show {
		cx := ox + f.CaretX(text, caret)
		ctx.DrawRect(paintengine2d.XYWH(cx, ty+2, 1.6, f.Height()-4), paintengine2d.Fill(l.caretColor()))
	}
	ctx.Restore()
}

func (l *Classic) baseDrawScrollBar(ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	l.paintFace(ctx, track, roleTrack, StateNone)
	if !thumb.Empty() {
		l.paintFace(ctx, thumb, roleThumb, st)
	}
	t := l.tokensOr()
	if t.Bevel == BevelClassic3D && track.Dy() >= 28 && track.Dx() >= 10 {
		// Arrow wells (visual only) at the ends of a vertical track.
		ah := float32(10)
		if s := LookScale(l); s > 1 {
			ah *= s
		}
		up := paintengine2d.XYWH(track.Min.X, track.Min.Y, track.Dx(), ah)
		dn := paintengine2d.XYWH(track.Min.X, track.Max.Y-ah, track.Dx(), ah)
		l.paintBezel(ctx, up, l.palette.SurfaceAlt, l.palette.Border, roleButton, StateNone)
		l.paintBezel(ctx, dn, l.palette.SurfaceAlt, l.palette.Border, roleButton, StateNone)
	}
}

// baseDrawFocusRing paints keyboard focus INSIDE b (the control's own
// rect). Widgets paint clipped to their bounds, so a ring drawn around the
// control — the old b.Inset(-2) convention — was clipped away and Tab
// focus was invisible on buttons, fields, lists and scroll views.
func (l *Classic) baseDrawFocusRing(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	p := l.palette
	m := l.metrics
	t := l.tokensOr()
	if b.Dx() < 4 || b.Dy() < 4 {
		return
	}
	r := m.Radius
	rs := m.RadiusSmall
	col := t.Focus.Border
	if colorUnset(col) {
		col = p.Focus
	}
	switch t.Bevel {
	case BevelClassic3D:
		// Windows / Motif: a dotted rectangle just inside the bevel.
		DottedRect(ctx, b.Inset(3), p.Text)
	case BevelLunaHottrack:
		DrawHotTrack(ctx, b.Inset(1.5), col.WithAlpha(0.08), col, l.rx(1))
	case BevelFluentAccent:
		ctx.DrawRoundRect(b.Inset(1), rs, rs, paintengine2d.StrokePaint(col, 1.5))
	default: // soft-shadow and none: soft outer band, crisp inner line
		fw := m.FocusWidth
		if fw < 1 {
			fw = 1
		}
		ctx.DrawRoundRect(b.Inset(fw*0.5+0.5), r, r, paintengine2d.StrokePaint(col.WithAlpha(0.40), fw+1))
		ctx.DrawRoundRect(b.Inset(fw+1.25), rs, rs, paintengine2d.StrokePaint(col.WithAlpha(0.92), 1.15))
	}
}

func (l *Classic) baseDrawSplitter(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	col, _, _ := l.faceColors(roleSplitter, st)
	if vertical {
		x := (b.Min.X + b.Max.X) * 0.5
		ctx.DrawRect(paintengine2d.XYWH(x-1, b.Min.Y+8, 2, b.Dy()-16), paintengine2d.Fill(col))
	} else {
		y := (b.Min.Y + b.Max.Y) * 0.5
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+8, y-1, b.Dx()-16, 2), paintengine2d.Fill(col))
	}
}

func (l *Classic) baseDrawListRow(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string) {
	st := StateNone
	if hovered {
		st |= StateHovered
	}
	if selected {
		st |= StateChecked
	}
	fg := l.fieldText()
	inner := b.Inset(2)
	if selected || hovered {
		fg = l.paintFace(ctx, inner, roleRow, st)
	}
	if l.menuInvertText() && selected {
		fg = l.palette.TextOnAccent
	}
	lb := paintengine2d.XYWH(b.Min.X+10, b.Min.Y, b.Dx()-14, b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, fg, AlignStart, 0)
}

func (l *Classic) baseDrawOverlay(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(l.palette.Overlay))
}

func (l *Classic) baseDrawMenuBar(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.paintBezel(ctx, b, l.palette.SurfaceAlt, l.palette.Divider, roleBar, StateNone)
}

func (l *Classic) baseDrawMenuTitle(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	if open || st.Pressed() || (st.Hovered() && !st.Disabled()) {
		// Open titles drop the bottom stroke so the popup shares an edge.
		hl := b.Inset(1)
		if open {
			hl.Max.Y = b.Max.Y
		}
		l.eng().MenuHighlight(l, ctx, hl, open)
	}
	fg := l.eng().MenuTextColor(l, open || st.Pressed() || (st.Hovered() && !st.Disabled()))
	l.drawLabeled(ctx, l.body, label, underline, b, fg)
	if st.Focused() && !open {
		l.DrawFocusRing(ctx, b.Inset(1))
	}
}

func (l *Classic) baseDrawMenuFrame(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	p := l.palette
	m := l.metrics
	// Keep the drop shadow inside the arranged box so PaintTree clip
	// cannot leave a sliver artifact under the last row.
	ctx.DrawRoundRect(b.Inset(1).Translate(paintengine2d.Pt(1, 2)), m.RadiusSmall, m.RadiusSmall, paintengine2d.Fill(p.Shadow))
	ctx.DrawRoundRect(b.Inset(1), m.RadiusSmall, m.RadiusSmall, paintengine2d.Fill(p.SurfaceAlt))
	// Classic XP icon gutter: a slightly darker strip to the label origin.
	ch := MenuChromeFor(l)
	inner := b.Inset(1)
	gw := ch.GutterW() - 1
	if gw > 2 && gw < inner.Dx()*0.55 {
		ctx.DrawRect(paintengine2d.XYWH(inner.Min.X, inner.Min.Y, gw, inner.Dy()), paintengine2d.Fill(p.MenuGutter))
	}
	ctx.DrawRoundRect(b.Inset(1.5), m.RadiusSmall, m.RadiusSmall, paintengine2d.StrokePaint(p.Border, m.Border+0.4))
}

func (l *Classic) baseMenuHighlight(ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	if b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	p := l.palette
	t := l.tokensOr()
	fill := t.Hot.Fill
	if colorUnset(fill) {
		fill = p.MenuHover
	}
	border := t.Hot.Border
	if colorUnset(border) {
		border = p.MenuHoverBorder
	}
	r := l.rx(1)
	switch t.Bevel {
	case BevelClassic3D:
		ctx.DrawRect(b, paintengine2d.Fill(fill))
		if fill.R != border.R || fill.G != border.G || fill.B != border.B {
			ctx.DrawRect(b.Inset(0.5), paintengine2d.StrokePaint(border, 1))
		}
	case BevelLunaHottrack:
		DrawHotTrack(ctx, b, fill, border, r)
	case BevelFluentAccent:
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
		acc := p.Accent
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y+2, 3, b.Dy()-4), paintengine2d.Fill(acc))
	default:
		ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
		if !colorUnset(border) {
			ctx.DrawRoundRect(b, r, r, paintengine2d.StrokePaint(border, 1))
		}
	}
	if attachBottom {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+1, b.Max.Y-1, b.Dx()-2, 1), paintengine2d.Fill(fill))
	}
}

func (l *Classic) menuItemHighlightBounds(item paintengine2d.Rect) paintengine2d.Rect {
	ch := MenuChromeFor(l)
	// Full row (icon gutter + label), 1px inside the popup frame / clip.
	return paintengine2d.XYWH(item.Min.X-ch.PadL+2, item.Min.Y+1, item.Dx()+ch.PadL+ch.PadR-4, item.Dy()-2)
}

func (l *Classic) baseDrawMenuItem(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	p := l.palette
	ch := MenuChromeFor(l)
	if row.Separator {
		y := (b.Min.Y + b.Max.Y) * 0.5
		x0 := ch.LabelMinX(b.Min.X)
		if w := b.Max.X - 8 - x0; w > 0 {
			ctx.DrawRect(paintengine2d.XYWH(x0, y, w, 1), paintengine2d.Fill(p.Divider))
		}
		return
	}
	if (st.Hovered() || st.Pressed()) && !st.Disabled() {
		l.eng().MenuHighlight(l, ctx, l.menuItemHighlightBounds(b), false)
	}
	ty := b.Min.Y + (b.Dy()-l.body.Height())*0.5
	if ty < b.Min.Y {
		ty = b.Min.Y
	}
	fg := p.Text
	font := l.body
	if st.Disabled() {
		fg = p.TextMuted
		font = l.muted
	} else {
		fg = l.eng().MenuTextColor(l, st.Hovered() || st.Pressed())
	}
	l.drawMenuGutter(ctx, b, ch, row, fg)
	label, shortcut, underline := row.Label, row.Shortcut, row.Underline
	// Clip to the item (and shortcut / submenu-arrow column), never tighter
	// than the reserved label box — glyph bearing may use the trailing pad.
	labelRight := b.Max.X
	if row.Submenu {
		aw := ch.SubmenuArrow
		if aw < 1 {
			aw = 10
		}
		ab := paintengine2d.XYWH(ch.ArrowMinX(b.Max.X), b.Min.Y, aw, b.Dy())
		drawMenuSubmenuArrow(ctx, ab, fg)
		labelRight = ab.Min.X
	}
	if shortcut != "" {
		tw := l.muted.InkWidth(shortcut)
		if tw <= 0 {
			tw = l.muted.Advance(shortcut)
		}
		sx := ch.LabelMaxX(b.Max.X) - tw
		if row.Submenu {
			sx = ch.ArrowMinX(b.Max.X) - tw
		}
		l.muted.Draw(ctx, shortcut, paintengine2d.Pt(sx, ty), p.TextMuted)
		labelRight = sx - ch.AccelGap
	}
	lx := ch.LabelMinX(b.Min.X)
	if labelRight < lx {
		labelRight = lx
	}
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(lx, b.Min.Y, labelRight-lx, b.Dy()))
	l.drawTextUnderline(ctx, font, label, underline, paintengine2d.Pt(lx, ty), fg)
	ctx.Restore()
}

func (l *Classic) drawMenuGutter(ctx *paintengine2d.Context, b paintengine2d.Rect, ch MenuChrome, row MenuRow, fg paintengine2d.Color) {
	side := IconSizePixels(l.IconSize())
	if s := LookScale(l); s > 1.01 {
		side *= s
	}
	gw := ch.CheckCol()
	if side > gw-2 {
		side = gw - 2
	}
	if side < 10 {
		side = 10
	}
	ib := paintengine2d.XYWH(b.Min.X+(gw-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	// Office XP: check/radio takes the gutter when present; otherwise the action icon.
	if row.Radio {
		drawMenuRadio(ctx, ib, row.Checked, fg)
		return
	}
	if row.Checked {
		drawMenuCheck(ctx, ib, fg)
		return
	}
	if row.Icon != IconNone {
		l.drawToolIcon(ctx, ib, row.Icon, fg)
	}
}

func drawMenuCheck(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	if ctx == nil || b.Empty() {
		return
	}
	p := paintengine2d.NewPath()
	p.MoveTo(b.Min.X+b.Dx()*0.16, b.Min.Y+b.Dy()*0.52)
	p.LineTo(b.Min.X+b.Dx()*0.40, b.Min.Y+b.Dy()*0.78)
	p.LineTo(b.Min.X+b.Dx()*0.86, b.Min.Y+b.Dy()*0.20)
	ctx.DrawPath(p, iconStroke(col, 1.85, paintengine2d.CapRound, paintengine2d.JoinRound))
}

func drawMenuSubmenuArrow(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	if ctx == nil || b.Empty() {
		return
	}
	cx := b.Min.X + b.Dx()*0.42
	cy := (b.Min.Y + b.Max.Y) * 0.5
	hw := b.Dx() * 0.22
	if hw < 2.2 {
		hw = 2.2
	}
	hh := b.Dy() * 0.16
	if hh < 3.2 {
		hh = 3.2
	}
	p := paintengine2d.NewPath()
	p.MoveTo(cx-hw*0.35, cy-hh)
	p.LineTo(cx+hw, cy)
	p.LineTo(cx-hw*0.35, cy+hh)
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(col))
}

func drawMenuRadio(ctx *paintengine2d.Context, b paintengine2d.Rect, on bool, col paintengine2d.Color) {
	if ctx == nil || b.Empty() {
		return
	}
	cx := (b.Min.X + b.Max.X) * 0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	rad := b.Dx()
	if b.Dy() < rad {
		rad = b.Dy()
	}
	rad *= 0.30
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), rad, paintengine2d.StrokePaint(col, 1.35))
	if on {
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), rad*0.48, paintengine2d.Fill(col))
	}
}

func (l *Classic) baseDrawTabBar(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.paintBezel(ctx, b, l.palette.SurfaceAlt, l.palette.Divider, roleBar, StateNone)
}

func (l *Classic) baseDrawTab(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	if selected {
		st |= StateChecked
	}
	box := b.Inset(3)
	if selected {
		box = paintengine2d.XYWH(b.Min.X+2, b.Min.Y+4, b.Dx()-4, b.Dy()-4)
	}
	fg := l.palette.Text
	if selected || (st.Hovered() && !st.Disabled()) || st.Pressed() {
		fg = l.paintFace(ctx, box, roleTab, st)
	}
	if selected && l.tokensOr().Bevel == BevelFluentAccent {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+6, b.Max.Y-3, b.Dx()-12, 3), paintengine2d.Fill(l.palette.Accent))
	}
	font := l.body
	if !selected {
		font = l.muted
		fg = l.palette.TextMuted
	}
	l.drawFittedText(ctx, font, label, b, fg, AlignCenter, 8)
	if st.Focused() {
		l.DrawFocusRing(ctx, b.Inset(2))
	}
}

func (l *Classic) baseDrawTreeRow(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered, expanded, leaf bool, depth int, label string, bold bool) {
	p := l.palette
	m := l.metrics
	st := StateNone
	if hovered {
		st |= StateHovered
	}
	if selected {
		st |= StateChecked
	}
	fg := l.fieldText()
	if selected || hovered {
		fg = l.paintFace(ctx, b.Inset(2), roleRow, st)
	}
	indent := m.TreeIndent
	if indent <= 0 {
		indent = 16
	}
	pad := float32(8)
	if m.RowPad > 0 {
		pad = m.RowPad + 2
	}
	x := b.Min.X + pad + float32(depth)*indent
	cy := (b.Min.Y + b.Max.Y) * 0.5
	guide := p.Divider.WithAlpha(0.85)
	for d := 0; d < depth; d++ {
		gx := b.Min.X + pad + float32(d)*indent + 3
		ctx.DrawRect(paintengine2d.XYWH(gx, b.Min.Y, 1, b.Dy()), paintengine2d.Fill(guide))
	}
	if !leaf {
		u := indent * 0.45
		if u < 6 {
			u = 6
		}
		if u > 12 {
			u = 12
		}
		chev := paintengine2d.NewPath()
		if expanded {
			chev.MoveTo(x, cy-u*0.35)
			chev.LineTo(x+u, cy-u*0.35)
			chev.LineTo(x+u*0.5, cy+u*0.45)
			chev.Close()
		} else {
			chev.MoveTo(x, cy-u*0.55)
			chev.LineTo(x+u*0.85, cy)
			chev.LineTo(x, cy+u*0.55)
			chev.Close()
		}
		ctx.DrawPath(chev, paintengine2d.Fill(p.TextMuted))
	}
	face := l.body
	if bold {
		face = l.bold
	}
	ctx.Save()
	ctx.ClipRect(b)
	face.Draw(ctx, label, paintengine2d.Pt(x+14, b.Min.Y+(b.Dy()-face.Height())*0.5), fg)
	ctx.Restore()
}

func (l *Classic) baseDrawStatusBar(ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	p := l.palette
	l.paintBezel(ctx, b, p.SurfaceAlt, p.Divider, roleBar, StateNone)
	if len(parts) == 0 {
		return
	}
	n := float32(len(parts))
	slot := b.Dx() / n
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		if i > 0 {
			ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y+6, 1, b.Dy()-12), paintengine2d.Fill(p.Divider))
		}
		// Each part fits its own slot (a long path used to run into the
		// next part — Settings' status bar).
		l.drawFittedText(ctx, l.body, s, paintengine2d.XYWH(x+10, b.Min.Y, slot-16, b.Dy()), p.TextMuted, AlignStart, 0)
	}
}

func (l *Classic) baseDrawToolBar(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.paintBezel(ctx, b, l.palette.SurfaceAlt, l.palette.Divider, roleBar, StateNone)
}

// ToolItemGap is the horizontal space between adjacent tool buttons.
const ToolItemGap = float32(8)

// ToolButtonChrome is pad, icon side, and icon→label gap for a tool
// button of height h. Measure and paint share this so labels cannot
// run into the next icon.
func ToolButtonChrome(h float32) (pad, iconSide, iconGap float32) {
	return ToolButtonChromeFor(nil, h)
}

// ToolButtonChromeFor is ToolButtonChrome using the look's iconSize
// (small 16 / medium 24 / large 32), scaled on HiDPI.
func ToolButtonChromeFor(lk LookAndFeel, h float32) (pad, iconSide, iconGap float32) {
	pad = 10
	iconGap = 8
	iconSide = IconSizePixels(LookIconSize(lk))
	if s := LookScale(lk); s > 1.01 {
		iconSide *= s
	}
	max := h - 8
	if max < 8 {
		max = 8
	}
	if iconSide > max {
		iconSide = max
	}
	if iconSide < 8 {
		iconSide = 8
	}
	return pad, iconSide, iconGap
}

func (l *Classic) baseDrawToolButton(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	p := l.palette
	well := b.Inset(1)
	fg := p.Text
	if st.Toggle() || (st.Hovered() && !st.Disabled()) || (st.Pressed() && !st.Disabled()) {
		fg = l.paintFace(ctx, well, roleTool, st)
	}
	if st.Focused() {
		l.DrawFocusRing(ctx, b.Inset(1))
	}
	font := l.body
	if st.Disabled() {
		fg = p.TextMuted
		font = l.muted
	}
	pad, iconSide, iconGap := ToolButtonChromeFor(l, b.Dy())
	x := b.Min.X + pad
	if icon != IconNone {
		ib := paintengine2d.XYWH(x, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		if label == "" {
			ib = paintengine2d.XYWH(b.Min.X+(b.Dx()-iconSide)*0.5, b.Min.Y+(b.Dy()-iconSide)*0.5, iconSide, iconSide)
		}
		l.drawToolIcon(ctx, ib, icon, fg)
		x = ib.Max.X + iconGap
	}
	if label != "" {
		right := b.Max.X - pad
		if right < x {
			right = x
		}
		ctx.Save()
		ctx.ClipRect(paintengine2d.XYWH(x, b.Min.Y, right-x, b.Dy()))
		font.Draw(ctx, label, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-font.Height())*0.5), fg)
		ctx.Restore()
	}
}

func (l *Classic) baseDrawProgressBar(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	p := l.palette
	m := l.metrics
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	if phase < 0 {
		phase = 0
	}
	phase = phase - float32(int(phase))
	if phase < 0 {
		phase += 1
	}
	r := m.RadiusSmall
	if r > b.Dy()*0.5 {
		r = b.Dy() * 0.5
	}
	track := p.Track
	fillCol := p.Accent
	if st.Disabled() {
		track = p.Surface
		fillCol = p.Divider
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(track))
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(p.Border.WithAlpha(0.55), 1))
	inner := b.Inset(2)
	if inner.Empty() {
		return
	}
	if indeterminate {
		span := inner.Dx() * 0.34
		x := inner.Min.X + (inner.Dx()+span)*phase - span
		fill := paintengine2d.XYWH(x, inner.Min.Y, span, inner.Dy())
		if fill.Min.X < inner.Min.X {
			fill.Min.X = inner.Min.X
		}
		if fill.Max.X > inner.Max.X {
			fill.Max.X = inner.Max.X
		}
		if fill.Dx() > 1 {
			ctx.DrawRoundRect(fill, r-1, r-1, paintengine2d.Fill(fillCol))
		}
		return
	}
	w := inner.Dx() * t
	if w > 1 {
		ctx.DrawRoundRect(paintengine2d.XYWH(inner.Min.X, inner.Min.Y, w, inner.Dy()), r-1, r-1, paintengine2d.Fill(fillCol))
	}
}

func (l *Classic) baseDrawRadio(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	p := l.palette
	m := l.metrics
	side := m.Radio
	if side <= 0 {
		side = m.Checkbox
	}
	if side <= 0 {
		side = 18
	}
	if selected {
		st |= StateChecked
	}
	cx := b.Min.X + side*0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	l.eng().RadioIndicator(l, ctx, paintengine2d.XYWH(cx-side*0.5, cy-side*0.5, side, side), st, selected)
	if label != "" {
		f := l.body
		if st.Disabled() {
			f = l.muted
		}
		lb := paintengine2d.XYWH(b.Min.X+side+8, b.Min.Y, b.Max.X-(b.Min.X+side+8), b.Dy())
		l.drawFittedText(ctx, f, label, lb, p.Text, AlignStart, 0)
		if st.Focused() {
			l.DrawFocusRing(ctx, labelFocusRect(f, label, lb, b))
		}
	} else if st.Focused() {
		l.DrawFocusRing(ctx, paintengine2d.XYWH(cx-side*0.5, cy-side*0.5, side, side).Intersect(b))
	}
}

func (l *Classic) baseDrawComboBox(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	p := l.palette
	m := l.metrics
	if open {
		st |= StatePressed
	}
	faceFg := l.paintFace(ctx, b, roleCombo, st)
	if st.Focused() && !open && l.eng().FieldFocusRing(l) {
		l.DrawFocusRing(ctx, b)
	}
	chevW := float32(22)
	btn := paintengine2d.XYWH(b.Max.X-chevW, b.Min.Y, chevW, b.Dy())
	if l.tokensOr().Bevel == BevelClassic3D || l.tokensOr().Bevel == BevelLunaHottrack {
		l.paintFace(ctx, btn.Inset(1), roleButton, st)
	}
	pad := m.FieldPad
	if pad <= 0 {
		pad = 8
	}
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad-chevW, b.Dy())
	f := l.body
	col := faceFg
	if text == "" {
		f = l.muted
		col = p.TextMuted
	}
	if st.Disabled() {
		f = l.muted
		col = p.TextMuted
	}
	ty := inner.Min.Y + (inner.Dy()-f.Height())*0.5
	ctx.Save()
	ctx.ClipRect(inner)
	if text != "" {
		show := f.Fit(text, inner.Dx())
		f.Draw(ctx, show, paintengine2d.Pt(inner.Min.X, ty), col)
	}
	ctx.Restore()
	cx := b.Max.X - chevW*0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	chev := paintengine2d.NewPath()
	if open {
		chev.MoveTo(cx-4.5, cy+2)
		chev.LineTo(cx+4.5, cy+2)
		chev.LineTo(cx, cy-4)
		chev.Close()
	} else {
		chev.MoveTo(cx-4.5, cy-2)
		chev.LineTo(cx+4.5, cy-2)
		chev.LineTo(cx, cy+4)
		chev.Close()
	}
	chevCol := p.TextMuted
	if ContrastRatio(chevCol, l.faceBackdrop(roleCombo, st)) < 3 {
		chevCol = faceFg
	}
	ctx.DrawPath(chev, paintengine2d.Fill(chevCol))
}

// faceBackdrop is the opaque colour a face of role / state ends up as.
func (l *Classic) faceBackdrop(role chromeRole, st ControlState) paintengine2d.Color {
	fill, _, _ := l.faceColorsRaw(role, st)
	base := l.roleBase(role)
	if colorUnset(fill) {
		return base
	}
	if fill.A < 1 {
		return Mix(base, fill, fill.A)
	}
	return fill
}

func (l *Classic) baseDrawTitleBar(ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	p := l.palette
	l.paintBezel(ctx, b, p.SurfaceAlt, p.Divider, roleBar, StateNone)
	pad := l.metrics.Pad
	if pad <= 0 {
		pad = 12
	}
	x := b.Min.X + pad
	if title == "" && subtitle == "" {
		return
	}
	f := l.body
	if subtitle == "" && b.Dy() >= l.title.Height()+12 {
		f = l.title
	}
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	if title != "" {
		f.Draw(ctx, title, paintengine2d.Pt(x, ty), p.Text)
		x += f.Advance(title) + 16
	}
	if subtitle != "" {
		l.muted.Draw(ctx, subtitle, paintengine2d.Pt(x, b.Min.Y+(b.Dy()-l.muted.Height())*0.5), p.TextMuted)
	}
}

func (l *Classic) baseDrawMessageIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	if icon == IconNone {
		return
	}
	l.drawToolIcon(ctx, b, icon, l.onWindow(l.messageIconColor(icon)))
}

func (l *Classic) messageIconColor(icon ToolIcon) paintengine2d.Color {
	switch icon {
	case IconWarning:
		return l.palette.Warning
	case IconError:
		return l.palette.Danger
	case IconQuestion:
		return l.palette.Accent
	case IconInfo:
		return l.palette.Accent
	default:
		return l.palette.Text
	}
}

func (l *Classic) drawToolIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon, col paintengine2d.Color) {
	DrawToolIcon(ctx, b, icon, col, l.Icons())
}

// rx is the paint radius for a 1× design value. Square looks force 0.
func (l *Classic) rx(design float32) float32 {
	if l == nil || l.square() {
		return 0
	}
	s := LookScale(l)
	if s <= 0 {
		s = 1
	}
	return design * s
}

func (l *Classic) square() bool {
	if l == nil {
		return false
	}
	if l.corners == CornersSquare {
		return true
	}
	return l.metrics.Radius <= 0 && l.metrics.RadiusSmall <= 0
}

func (l *Classic) baseDrawTableHeader(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	p := l.palette
	l.paintBezel(ctx, b, p.SurfaceAlt, p.Divider, roleBar, StateNone)
	fg := p.Text
	if st.Pressed() || (st.Hovered() && !st.Disabled()) {
		fg = l.paintFace(ctx, b.Inset(1), roleRow, st)
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-1, b.Min.Y+6, 1, b.Dy()-12), paintengine2d.Fill(p.Divider))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
	pad := float32(8)
	chevW := float32(0)
	if sorted {
		chevW = 14
	}
	ty := b.Min.Y + (b.Dy()-l.body.Height())*0.5
	tw := l.body.Advance(label)
	innerW := b.Dx() - pad - chevW - 4
	if tw > 0 && tw <= b.Dx()-chevW-4 && innerW < tw {
		pad = 2
		innerW = b.Dx() - pad - chevW - 4
	}
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, innerW, b.Dy())
	ctx.Save()
	ctx.ClipRect(inner)
	l.body.Draw(ctx, l.body.Fit(label, inner.Dx()), paintengine2d.Pt(inner.Min.X, ty), fg)
	ctx.Restore()
	if sorted {
		cx := b.Max.X - 12
		cy := (b.Min.Y + b.Max.Y) * 0.5
		chev := paintengine2d.NewPath()
		if asc {
			chev.MoveTo(cx-3.5, cy+2)
			chev.LineTo(cx+3.5, cy+2)
			chev.LineTo(cx, cy-3)
		} else {
			chev.MoveTo(cx-3.5, cy-2)
			chev.LineTo(cx+3.5, cy-2)
			chev.LineTo(cx, cy+3)
		}
		chev.Close()
		ctx.DrawPath(chev, paintengine2d.Fill(p.Accent))
	}
}

func (l *Classic) baseDrawTableCell(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string, align Align, face *Font) {
	p := l.palette
	st := StateNone
	if hovered {
		st |= StateHovered
	}
	if selected {
		st |= StateChecked
	}
	fg := l.fieldText()
	if selected || hovered {
		fg = l.paintFace(ctx, b, roleRow, st)
	}
	f := l.faceOrBody(face)
	pad := tableCellPad(b.Dx(), f.Advance(label))
	avail := b.Dx() - pad*2
	if avail < 4 {
		avail = 4
	}
	label = f.Fit(label, avail)
	tw := f.Advance(label)
	x := b.Min.X + pad
	switch align {
	case AlignCenter:
		x = b.Min.X + (b.Dx()-tw)*0.5
	case AlignEnd:
		x = b.Max.X - tw - pad
	}
	ty := b.Min.Y + (b.Dy()-f.Height())*0.5
	ctx.Save()
	ctx.ClipRect(b.Inset(1))
	f.Draw(ctx, label, paintengine2d.Pt(x, ty), fg)
	ctx.Restore()
	ctx.DrawRect(paintengine2d.XYWH(b.Max.X-1, b.Min.Y, 1, b.Dy()), paintengine2d.Fill(p.Divider.WithAlpha(0.55)))
}

// tableCellPad keeps the default 8px inset unless the full label already
// fits the cell with a tighter pad (28px ★/📎 columns). Fit must not
// replace a glyph that the column can show.
func tableCellPad(cellW, textW float32) float32 {
	pad := float32(8)
	if textW > 0 && textW <= cellW-4 && cellW-pad*2 < textW {
		pad = (cellW - textW) * 0.5
		if pad < 2 {
			pad = 2
		}
	}
	return pad
}

func (l *Classic) baseDrawSpinner(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	p := l.palette
	m := l.metrics
	r := m.RadiusSmall
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(p.SurfaceAlt))
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(p.FieldBorder, 1))
	mid := (b.Min.Y + b.Max.Y) * 0.5
	up := paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), mid-b.Min.Y)
	down := paintengine2d.XYWH(b.Min.X, mid, b.Dx(), b.Max.Y-mid)
	if upPress {
		ctx.DrawRect(up.Inset(1), paintengine2d.Fill(p.Accent.WithAlpha(0.30)))
	} else if upHover && !st.Disabled() {
		ctx.DrawRect(up.Inset(1), paintengine2d.Fill(p.Highlight))
	}
	if downPress {
		ctx.DrawRect(down.Inset(1), paintengine2d.Fill(p.Accent.WithAlpha(0.30)))
	} else if downHover && !st.Disabled() {
		ctx.DrawRect(down.Inset(1), paintengine2d.Fill(p.Highlight))
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X+3, mid, b.Dx()-6, 1), paintengine2d.Fill(p.Divider))
	cx := (b.Min.X + b.Max.X) * 0.5
	upC := paintengine2d.NewPath()
	upC.MoveTo(cx-3.2, mid-3)
	upC.LineTo(cx+3.2, mid-3)
	upC.LineTo(cx, b.Min.Y+4)
	upC.Close()
	dnC := paintengine2d.NewPath()
	dnC.MoveTo(cx-3.2, mid+3)
	dnC.LineTo(cx+3.2, mid+3)
	dnC.LineTo(cx, b.Max.Y-4)
	dnC.Close()
	col := p.TextMuted
	if st.Disabled() {
		col = p.Divider
	}
	ctx.DrawPath(upC, paintengine2d.Fill(col))
	ctx.DrawPath(dnC, paintengine2d.Fill(col))
}

func (l *Classic) baseDrawTextArea(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	p := l.palette
	m := l.metrics
	l.paintFace(ctx, b, roleField, st)
	if st.Focused() && l.eng().FieldFocusRing(l) {
		l.DrawFocusRing(ctx, b)
	}
	pad := m.FieldPad
	if pad <= 0 {
		pad = 8
	}
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+pad, b.Dx()-pad*2, b.Dy()-pad*2)
	ctx.Save()
	ctx.ClipRect(inner)
	f := l.faceOrBody(face)
	muted := l.mutedFor(face)
	lh := f.Height() + 2
	empty := len(lines) == 0 || (len(lines) == 1 && lines[0].Text == "" && lines[0].End <= lines[0].Start)
	if empty && placeholder != "" && !st.Focused() {
		muted.Draw(ctx, placeholder, paintengine2d.Pt(inner.Min.X, inner.Min.Y), p.TextMuted)
		ctx.Restore()
		return
	}
	if selA > selB {
		selA, selB = selB, selA
	}
	for i, line := range lines {
		y := inner.Min.Y + float32(i)*lh - scrollY
		if y+lh < inner.Min.Y || y > inner.Max.Y {
			continue
		}
		ox := inner.Min.X - scrollX
		if selA != selB && selB > line.Start && selA < line.End {
			a := selA
			if a < line.Start {
				a = line.Start
			}
			b0 := selB
			if b0 > line.End {
				b0 = line.End
			}
			col0 := a - line.Start
			col1 := b0 - line.Start
			if col0 < 0 {
				col0 = 0
			}
			n := 0
			for range line.Text {
				n++
			}
			x0 := ox + f.CaretX(line.Text, col0)
			x1 := ox + f.CaretX(line.Text, col1)
			if col1 > n || selB > line.Start+n {
				x1 = inner.Max.X
			}
			if x1 < x0 {
				x1 = x0
			}
			if x1-x0 < 3 && selB > line.End-1 {
				x1 = inner.Max.X
			}
			selBox := paintengine2d.XYWH(x0, y+1, x1-x0, f.Height()-2)
			ctx.DrawRect(selBox, paintengine2d.Fill(p.Selection))
			if line.Text != "" {
				f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), l.fieldText())
				ctx.Save()
				ctx.ClipRect(selBox)
				f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), l.selectedText(p.Selection))
				ctx.Restore()
			}
		} else if line.Text != "" {
			f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), l.fieldText())
		}
		if st.Focused() && blink && caret >= line.Start && caret <= line.End {
			onThis := caret < line.End || i == len(lines)-1
			if caret == line.End && i < len(lines)-1 && lines[i+1].Start == line.End {
				onThis = false
			}
			if onThis {
				col := caret - line.Start
				if col < 0 {
					col = 0
				}
				cx := ox + f.CaretX(line.Text, col)
				ctx.DrawRect(paintengine2d.XYWH(cx, y+1, 1.6, f.Height()-2), paintengine2d.Fill(l.caretColor()))
			}
		}
	}
	ctx.Restore()
}

func (l *Classic) baseDrawSwitch(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	p := l.palette
	m := l.metrics
	tw, th := m.SwitchW, m.SwitchH
	if tw <= 0 {
		tw = 42
	}
	if th <= 0 {
		th = 22
	}
	track := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-th)*0.5, tw, th)
	fill := p.Track
	if on {
		fill = p.Accent
	}
	if st.Hovered() && !st.Disabled() {
		fill = fill.Lerp(p.AccentHover, 0.28)
	}
	if st.Disabled() {
		fill = p.Surface
	}
	rr := th * 0.5
	if l.square() {
		rr = 0
	}
	ctx.DrawRoundRect(track, rr, rr, paintengine2d.Fill(fill))
	ctx.DrawRoundRect(track.Inset(0.5), rr, rr, paintengine2d.StrokePaint(p.FieldBorder.WithAlpha(0.7), 1))
	pad := float32(2.4)
	kr := th*0.5 - pad
	kx := track.Min.X + pad + kr
	if on {
		kx = track.Max.X - pad - kr
	}
	cy := (track.Min.Y + track.Max.Y) * 0.5
	knob := p.TextOnAccent
	if !on {
		knob = p.SurfaceAlt
	}
	if st.Disabled() {
		knob = p.Thumb
	}
	ctx.DrawCircle(paintengine2d.Pt(kx, cy+0.6), kr+0.6, paintengine2d.Fill(p.Shadow))
	ctx.DrawCircle(paintengine2d.Pt(kx, cy), kr, paintengine2d.Fill(knob))
	if st.Focused() {
		l.DrawFocusRing(ctx, track)
	}
	if label != "" {
		f := l.body
		if st.Disabled() {
			f = l.muted
		}
		lb := paintengine2d.XYWH(track.Max.X+8, b.Min.Y, b.Max.X-(track.Max.X+8), b.Dy())
		l.drawFittedText(ctx, f, label, lb, p.Text, AlignStart, 0)
	}
}

func (l *Classic) baseDrawAccordionHeader(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	p := l.palette
	l.paintBezel(ctx, b, p.SurfaceAlt, p.Divider, roleBar, StateNone)
	if st.Pressed() || (st.Hovered() && !st.Disabled()) {
		l.paintFace(ctx, b, roleRow, st)
	}
	x := b.Min.X + 10
	cy := (b.Min.Y + b.Max.Y) * 0.5
	chev := paintengine2d.NewPath()
	if expanded {
		chev.MoveTo(x, cy-3)
		chev.LineTo(x+8, cy-3)
		chev.LineTo(x+4, cy+4)
		chev.Close()
	} else {
		chev.MoveTo(x, cy-5)
		chev.LineTo(x+7, cy)
		chev.LineTo(x, cy+5)
		chev.Close()
	}
	ctx.DrawPath(chev, paintengine2d.Fill(p.TextMuted))
	l.body.Draw(ctx, title, paintengine2d.Pt(x+16, b.Min.Y+(b.Dy()-l.body.Height())*0.5), p.Text)
	if st.Focused() {
		l.DrawFocusRing(ctx, b.Inset(1))
	}
}

func (l *Classic) baseDrawSeparator(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	p := l.palette
	if vertical {
		x := (b.Min.X + b.Max.X) * 0.5
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y+4, 1, b.Dy()-8), paintengine2d.Fill(p.Divider))
		return
	}
	y := (b.Min.Y + b.Max.Y) * 0.5
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), 1), paintengine2d.Fill(p.Divider))
}

func (l *Classic) baseDrawTooltip(ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	p := l.palette
	m := l.metrics
	r := m.RadiusSmall
	ctx.DrawRoundRect(b.Translate(paintengine2d.Pt(1.5, 2)), r, r, paintengine2d.Fill(p.Shadow))
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(p.SurfaceAlt))
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(p.Border, m.Border))
	pad := m.TooltipPad
	if pad <= 0 {
		pad = 8
	}
	ty := b.Min.Y + (b.Dy()-l.body.Height())*0.5
	l.body.Draw(ctx, text, paintengine2d.Pt(b.Min.X+pad, ty), p.Text)
}

func (l *Classic) drawLabeled(ctx *paintengine2d.Context, f *Font, text string, underline int, b paintengine2d.Rect, col paintengine2d.Color) {
	if f == nil || text == "" {
		return
	}
	tw := f.Advance(text)
	th := f.Height()
	origin := paintengine2d.Pt(b.Min.X+(b.Dx()-tw)*0.5, b.Min.Y+(b.Dy()-th)*0.5)
	l.drawTextUnderline(ctx, f, text, underline, origin, col)
}

func (l *Classic) drawTextUnderline(ctx *paintengine2d.Context, f *Font, text string, underline int, origin paintengine2d.Point, col paintengine2d.Color) {
	if f == nil || text == "" {
		return
	}
	f.Draw(ctx, text, origin, col)
	if underline < 0 || underline >= len([]rune(text)) {
		return
	}
	x0 := origin.X + f.CaretX(text, underline)
	x1 := origin.X + f.CaretX(text, underline+1)
	if x1-x0 < 4 {
		x1 = x0 + 6
	}
	y := origin.Y + f.Height() - 2
	ctx.DrawRect(paintengine2d.XYWH(x0, y, x1-x0, 1.2), paintengine2d.Fill(col))
}

func (l *Classic) drawCentered(ctx *paintengine2d.Context, f *Font, text string, b paintengine2d.Rect) {
	if f == nil || text == "" {
		return
	}
	col := f.Color
	if col == (paintengine2d.Color{}) {
		col = l.palette.Text
	}
	l.drawFittedText(ctx, f, text, b, col, AlignCenter, 8)
}

func (l *Classic) drawFittedText(ctx *paintengine2d.Context, f *Font, text string, b paintengine2d.Rect, col paintengine2d.Color, align Align, pad float32) {
	if f == nil || text == "" || b.Empty() {
		return
	}
	ctx.Save()
	ctx.ClipRect(b)
	maxW := b.Dx() - pad
	if maxW < 4 {
		maxW = 4
	}
	show := text
	if f.Advance(show) > maxW {
		show = f.Fit(show, maxW)
	}
	tw := f.Advance(show)
	th := f.Height()
	x := b.Min.X
	switch align {
	case AlignCenter:
		x = b.Min.X + (b.Dx()-tw)*0.5
	case AlignEnd:
		x = b.Max.X - tw - 2
	}
	y := b.Min.Y + (b.Dy()-th)*0.5
	if y < b.Min.Y {
		y = b.Min.Y
	}
	f.Draw(ctx, show, paintengine2d.Pt(x, y), col)
	ctx.Restore()
}

func (l *Classic) fontFor(col paintengine2d.Color) *Font {
	if nearColor(col, l.palette.TextMuted) {
		return l.muted
	}
	if nearColor(col, l.palette.TextOnAccent) {
		return l.onAcc
	}
	if col != (paintengine2d.Color{}) && !nearColor(col, l.palette.Text) {
		return BakeFont(l.metrics.FontSize, col)
	}
	return l.body
}

type fieldTextKey struct{}

// fieldText is the text colour on the field / view background: the window
// text when it reads there, else black or white (NeXT has dark chrome and
// white fields; its lists and text fields drew white on white).
func (l *Classic) fieldText() paintengine2d.Color {
	return l.Memo(fieldTextKey{}, func() any {
		if colorUnset(l.palette.Field) {
			return l.palette.Text
		}
		return ReadableOn(l.palette.Field, 4.5, l.palette.Text)
	}).(paintengine2d.Color)
}

// selectedText is the colour of text over a text selection of colour sel.
func (l *Classic) selectedText(sel paintengine2d.Color) paintengine2d.Color {
	bg := sel
	if colorUnset(bg) {
		return l.fieldText()
	}
	if bg.A < 1 {
		base := l.palette.Field
		if colorUnset(base) {
			base = l.palette.Background
		}
		bg = Mix(base, sel, sel.A)
	}
	return ReadableOn(bg, 4.5, l.fieldText(), l.palette.TextOnAccent)
}

// caretColor is the text caret: the pack's "caret" extra, else the text
// colour (the old accent caret vanished on dark fields — navy on grey).
func (l *Classic) caretColor() paintengine2d.Color {
	return l.X("caret", l.fieldText())
}

// labelFocusRect is the focus rectangle around a check / radio label (the
// Windows convention), kept inside the control rect.
func labelFocusRect(f *Font, label string, lb, b paintengine2d.Rect) paintengine2d.Rect {
	w := lb.Dx()
	if f != nil {
		if tw := f.Advance(label); tw+6 < w {
			w = tw + 6
		}
	}
	h := b.Dy() - 2
	if f != nil && f.Height()+6 < h {
		h = f.Height() + 6
	}
	r := paintengine2d.XYWH(lb.Min.X-3, b.Min.Y+(b.Dy()-h)*0.5, w, h)
	return r.Intersect(b)
}

func nearColor(a, b paintengine2d.Color) bool {
	dr, dg, db := a.R-b.R, a.G-b.G, a.B-b.B
	return dr*dr+dg*dg+db*db < 0.002
}

func clamp1(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ---- LookAndFeel: every control is painted by the look's engine ----------

func (l *Classic) DrawPanel(ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	l.eng().DrawPanel(l, ctx, b, raised)
}

func (l *Classic) DrawButton(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	l.eng().DrawButton(l, ctx, b, st, label)
}

func (l *Classic) DrawLabel(ctx *paintengine2d.Context, b paintengine2d.Rect, text string, col paintengine2d.Color, align Align) {
	l.eng().DrawLabel(l, ctx, b, text, col, align)
}

func (l *Classic) DrawCheckbox(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	l.eng().DrawCheckbox(l, ctx, b, st, checked, label)
}

func (l *Classic) DrawSlider(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
	l.eng().DrawSlider(l, ctx, b, st, t)
}

func (l *Classic) DrawTextField(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	l.eng().DrawTextField(l, ctx, b, st, text, placeholder, caret, selA, selB, blink, scrollX, face)
}

func (l *Classic) DrawScrollBar(ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	l.eng().DrawScrollBar(l, ctx, track, thumb, st)
}

func (l *Classic) DrawFocusRing(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.eng().DrawFocusRing(l, ctx, b)
}

func (l *Classic) DrawSplitter(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	l.eng().DrawSplitter(l, ctx, b, vertical, st)
}

func (l *Classic) DrawListRow(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string) {
	l.eng().DrawListRow(l, ctx, b, selected, hovered, label)
}

func (l *Classic) DrawOverlay(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.eng().DrawOverlay(l, ctx, b)
}

func (l *Classic) DrawMenuBar(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.eng().DrawMenuBar(l, ctx, b)
}

func (l *Classic) DrawMenuTitle(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	l.eng().DrawMenuTitle(l, ctx, b, st, label, underline, open)
}

func (l *Classic) DrawMenuFrame(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.eng().DrawMenuFrame(l, ctx, b)
}

func (l *Classic) DrawMenuItem(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
	l.eng().DrawMenuItem(l, ctx, b, st, row)
}

func (l *Classic) DrawTabBar(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.eng().DrawTabBar(l, ctx, b)
}

func (l *Classic) DrawTab(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	l.eng().DrawTab(l, ctx, b, st, label, selected)
}

func (l *Classic) DrawTreeRow(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered, expanded, leaf bool, depth int, label string, bold bool) {
	l.eng().DrawTreeRow(l, ctx, b, selected, hovered, expanded, leaf, depth, label, bold)
}

func (l *Classic) DrawStatusBar(ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	l.eng().DrawStatusBar(l, ctx, b, parts)
}

func (l *Classic) DrawToolBar(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	l.eng().DrawToolBar(l, ctx, b)
}

func (l *Classic) DrawToolButton(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	l.eng().DrawToolButton(l, ctx, b, st, label, icon)
}

func (l *Classic) DrawProgressBar(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
	l.eng().DrawProgressBar(l, ctx, b, st, t, indeterminate, phase)
}

func (l *Classic) DrawRadio(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	l.eng().DrawRadio(l, ctx, b, st, selected, label)
}

func (l *Classic) DrawComboBox(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	l.eng().DrawComboBox(l, ctx, b, st, text, open)
}

func (l *Classic) DrawTitleBar(ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	l.eng().DrawTitleBar(l, ctx, b, title, subtitle)
}

func (l *Classic) DrawMessageIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	l.eng().DrawMessageIcon(l, ctx, b, icon)
}

func (l *Classic) DrawTableHeader(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	l.eng().DrawTableHeader(l, ctx, b, st, label, sorted, asc)
}

func (l *Classic) DrawTableCell(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string, align Align, face *Font) {
	l.eng().DrawTableCell(l, ctx, b, selected, hovered, label, align, face)
}

func (l *Classic) DrawSpinner(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
	l.eng().DrawSpinner(l, ctx, b, st, upHover, downHover, upPress, downPress)
}

func (l *Classic) DrawTooltip(ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
	l.eng().DrawTooltip(l, ctx, b, text)
}

func (l *Classic) DrawTextArea(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	l.eng().DrawTextArea(l, ctx, b, st, lines, caret, selA, selB, blink, scrollX, scrollY, placeholder, face)
}

func (l *Classic) DrawSwitch(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
	l.eng().DrawSwitch(l, ctx, b, st, on, label)
}

func (l *Classic) DrawAccordionHeader(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	l.eng().DrawAccordionHeader(l, ctx, b, st, title, expanded)
}

func (l *Classic) DrawSeparator(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	l.eng().DrawSeparator(l, ctx, b, vertical)
}

// baseCheckIndicator is the stock checkbox well + tick.
func (l *Classic) baseCheckIndicator(ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, checked bool) {
	side := box.Dx()
	if checked {
		st |= StateChecked
	}
	l.paintFace(ctx, box, roleCheck, st)
	if !checked {
		return
	}
	chk := paintengine2d.NewPath()
	chk.MoveTo(box.Min.X+4, box.Min.Y+side*0.52)
	chk.LineTo(box.Min.X+side*0.42, box.Max.Y-4.5)
	chk.LineTo(box.Max.X-3.5, box.Min.Y+4)
	cap, join := paintengine2d.CapRound, paintengine2d.JoinRound
	if l.square() {
		cap, join = paintengine2d.CapSquare, paintengine2d.JoinMiter
	}
	ctx.DrawPath(chk, paintengine2d.Paint{
		Color:  l.palette.TextOnAccent,
		Style:  paintengine2d.StyleStroke,
		Stroke: paintengine2d.Stroke{Width: 2.1, Cap: cap, Join: join, MiterLimit: 4},
	})
}

// baseRadioIndicator is the stock round radio well + dot.
func (l *Classic) baseRadioIndicator(ctx *paintengine2d.Context, box paintengine2d.Rect, st ControlState, selected bool) {
	p := l.palette
	side := box.Dx()
	if selected {
		st |= StateChecked
	}
	cx := (box.Min.X + box.Max.X) * 0.5
	cy := (box.Min.Y + box.Max.Y) * 0.5
	fill, border, _ := l.faceColors(roleCheck, st)
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), side*0.5, paintengine2d.Fill(fill))
	if l.tokensOr().Bevel == BevelClassic3D {
		// Inset well: light on bottom-right, dark on top-left.
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), side*0.5-0.5, paintengine2d.StrokePaint(p.BevelDark, 1))
		ctx.DrawCircle(paintengine2d.Pt(cx+0.6, cy+0.6), side*0.5-1.4, paintengine2d.StrokePaint(p.BevelLight, 1))
	} else {
		if colorUnset(border) {
			border = p.FieldBorder
		}
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), side*0.5-0.5, paintengine2d.StrokePaint(border, 1))
	}
	if selected {
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), side*0.18+1.2, paintengine2d.Fill(p.TextOnAccent))
	}
}
