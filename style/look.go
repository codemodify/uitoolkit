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
	body     *Font
	title    *Font
	bold     *Font
	muted    *Font
	onAcc    *Font
	mono     *Font
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

func newClassic(name string, p Palette, m Metrics, corners CornerStyle, icons IconSetName, iconSize IconSize) *Classic {
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
	p = ResolveMenuChrome(p)
	return &Classic{
		palette:  p,
		metrics:  m,
		name:     name,
		pack:     StarterName(ParseTheme(name)),
		corners:  corners,
		icons:    icons,
		iconSize: iconSize,
		// OpenType atlases (Titillium / JetBrains Mono); Color tints at draw.
		body:  BakeFont(m.FontSize, p.Text),
		title: BakeTitleFont(m.TitleSize, p.Text),
		bold:  BakeFamily(FamilyUI, WeightBold, m.FontSize, p.Text),
		muted: BakeFont(m.FontSize, p.TextMuted),
		onAcc: BakeFont(m.FontSize, p.TextOnAccent),
		mono:  BakeMonoFont(m.FontSize, p.Text),
	}
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
	return newClassic(c.Name(), c.Palette(), ScaleMetrics(c.Metrics(), scale), c.Corners(), c.Icons(), c.IconSize()).setPack(c.Pack())
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

func (l *Classic) DrawPanel(ctx *paintengine2d.Context, b paintengine2d.Rect, raised bool) {
	p := l.palette
	m := l.metrics
	fill := p.Surface
	if raised {
		fill = p.SurfaceAlt
	}
	ctx.DrawRoundRect(b, m.Radius, m.Radius, paintengine2d.Fill(fill))
	ctx.DrawRoundRect(b.Inset(0.5), m.Radius, m.Radius, paintengine2d.StrokePaint(p.Border, m.Border))
}

func (l *Classic) DrawButton(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string) {
	p := l.palette
	m := l.metrics
	r := m.RadiusSmall
	if r > 0 {
		r += 1
	}
	top, bot := p.SurfaceAlt, p.Surface
	fg := l.body
	if st.Primary() {
		top, bot = p.AccentHover, p.Accent
		fg = l.onAcc
	}
	if st.Hovered() && !st.Disabled() {
		if st.Primary() {
			top = paintengine2d.RGB(clamp1(top.R+0.06), clamp1(top.G+0.06), clamp1(top.B+0.04))
		} else {
			top = p.Highlight
			bot = p.SurfaceAlt
		}
	}
	if st.Pressed() && !st.Disabled() {
		if st.Primary() {
			top, bot = p.AccentPress, p.AccentPress
		} else {
			top, bot = p.Surface, p.Surface
		}
		b = b.Translate(paintengine2d.Pt(0, 1))
	}
	if st.Disabled() {
		top = p.Surface
		bot = p.Surface
		fg = l.muted
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: b.Min, End: paintengine2d.Pt(b.Min.X, b.Max.Y),
		Stops: []paintengine2d.GradientStop{{Offset: 0, Color: top}, {Offset: 1, Color: bot}},
	}))
	border := p.Border
	if st.Primary() && !st.Disabled() {
		border = p.AccentPress
	}
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(border, m.Border))
	if st.Focused() {
		l.DrawFocusRing(ctx, b.Inset(-2))
	}
	l.drawCentered(ctx, fg, label, b)
}

func (l *Classic) DrawLabel(ctx *paintengine2d.Context, b paintengine2d.Rect, text string, col paintengine2d.Color, align Align) {
	f := l.fontFor(col)
	l.drawFittedText(ctx, f, text, b, col, align, 2)
}

func (l *Classic) DrawCheckbox(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, checked bool, label string) {
	p := l.palette
	m := l.metrics
	side := m.Checkbox
	box := paintengine2d.XYWH(b.Min.X, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	fill := p.Field
	if checked {
		fill = p.Accent
	}
	if st.Disabled() {
		if checked {
			fill = p.Divider
		} else {
			fill = p.Surface
		}
	} else if st.Hovered() {
		fill = fill.Lerp(p.AccentHover, 0.25)
	}
	cr := l.rx(4)
	ctx.DrawRoundRect(box, cr, cr, paintengine2d.Fill(fill))
	ctx.DrawRoundRect(box.Inset(0.5), cr, cr, paintengine2d.StrokePaint(p.FieldBorder, 1))
	if checked {
		chk := paintengine2d.NewPath()
		chk.MoveTo(box.Min.X+4, box.Min.Y+side*0.52)
		chk.LineTo(box.Min.X+side*0.42, box.Max.Y-4.5)
		chk.LineTo(box.Max.X-3.5, box.Min.Y+4)
		cap, join := paintengine2d.CapRound, paintengine2d.JoinRound
		if l.square() {
			cap, join = paintengine2d.CapSquare, paintengine2d.JoinMiter
		}
		ctx.DrawPath(chk, paintengine2d.Paint{
			Color:  p.TextOnAccent,
			Style:  paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: 2.1, Cap: cap, Join: join, MiterLimit: 4},
		})
	}
	if st.Focused() {
		l.DrawFocusRing(ctx, box.Inset(-2))
	}
	if label != "" {
		f := l.body
		if st.Disabled() {
			f = l.muted
		}
		lb := paintengine2d.XYWH(box.Max.X+8, b.Min.Y, b.Max.X-(box.Max.X+8), b.Dy())
		l.drawFittedText(ctx, f, label, lb, p.Text, AlignStart, 0)
	}
}

func (l *Classic) DrawSlider(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32) {
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
		l.DrawFocusRing(ctx, paintengine2d.XYWH(tx-rad-3, cy-rad-3, rad*2+6, rad*2+6))
	}
}

func (l *Classic) DrawTextField(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32, face *Font) {
	p := l.palette
	m := l.metrics
	r := m.RadiusSmall
	field := p.Field
	if st.Disabled() {
		field = p.Surface
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(field))
	border := p.FieldBorder
	if st.Disabled() {
		border = p.Divider
	} else if st.Focused() {
		border = p.Focus
	} else if st.Hovered() {
		border = p.Border
	}
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(border, m.Border+float32(btoi(st.Focused() && !st.Disabled()))))
	if st.Focused() && !st.Disabled() {
		l.DrawFocusRing(ctx, b.Inset(-2))
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
		ctx.DrawRect(paintengine2d.XYWH(x0, ty+1, x1-x0, f.Height()-2), paintengine2d.Fill(sel))
	}
	font.Draw(ctx, show, paintengine2d.Pt(ox, ty), p.Text)
	if st.Focused() && blink && text == show {
		cx := ox + f.CaretX(text, caret)
		ctx.DrawRect(paintengine2d.XYWH(cx, ty+2, 1.6, f.Height()-4), paintengine2d.Fill(p.Accent))
	}
	ctx.Restore()
}

func (l *Classic) DrawScrollBar(ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st ControlState) {
	p := l.palette
	r := l.rx(4)
	ctx.DrawRoundRect(track, r, r, paintengine2d.Fill(p.Track.WithAlpha(0.55)))
	col := p.Thumb
	if st.Hovered() || st.Pressed() {
		col = p.Accent
	}
	ctx.DrawRoundRect(thumb, r, r, paintengine2d.Fill(col))
}

func (l *Classic) DrawFocusRing(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	p := l.palette
	m := l.metrics
	r := m.Radius
	rs := m.RadiusSmall
	if rs > 0 {
		rs += 1
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.StrokePaint(p.Focus.WithAlpha(0.40), m.FocusWidth+1.5))
	ctx.DrawRoundRect(b.Inset(1.25), rs, rs, paintengine2d.StrokePaint(p.Focus.WithAlpha(0.92), 1.15))
}

func (l *Classic) DrawSplitter(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool, st ControlState) {
	p := l.palette
	col := p.Divider
	if st.Hovered() || st.Pressed() {
		col = p.Accent
	}
	if vertical {
		x := (b.Min.X + b.Max.X) * 0.5
		ctx.DrawRect(paintengine2d.XYWH(x-1, b.Min.Y+8, 2, b.Dy()-16), paintengine2d.Fill(col))
	} else {
		y := (b.Min.Y + b.Max.Y) * 0.5
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+8, y-1, b.Dx()-16, 2), paintengine2d.Fill(col))
	}
}

func (l *Classic) DrawListRow(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string) {
	p := l.palette
	r := l.rx(5)
	if selected {
		ctx.DrawRoundRect(b.Inset(2), r, r, paintengine2d.Fill(p.Accent.WithAlpha(0.28)))
	} else if hovered {
		ctx.DrawRoundRect(b.Inset(2), r, r, paintengine2d.Fill(p.Highlight))
	}
	lb := paintengine2d.XYWH(b.Min.X+10, b.Min.Y, b.Dx()-14, b.Dy())
	l.drawFittedText(ctx, l.body, label, lb, p.Text, AlignStart, 0)
}

func (l *Classic) DrawOverlay(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(l.palette.Overlay))
}

func (l *Classic) DrawMenuBar(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	p := l.palette
	ctx.DrawRect(b, paintengine2d.Fill(p.SurfaceAlt))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
}

func (l *Classic) DrawMenuTitle(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, underline int, open bool) {
	p := l.palette
	if open || st.Pressed() || (st.Hovered() && !st.Disabled()) {
		// Open titles drop the bottom stroke so the popup shares an edge.
		hl := b.Inset(1)
		if open {
			hl.Max.Y = b.Max.Y
		}
		l.drawMenuHighlight(ctx, hl, open)
	}
	l.drawLabeled(ctx, l.body, label, underline, b, p.Text)
	if st.Focused() && !open {
		l.DrawFocusRing(ctx, b.Inset(1))
	}
}

func (l *Classic) DrawMenuFrame(ctx *paintengine2d.Context, b paintengine2d.Rect) {
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

func (l *Classic) drawMenuHighlight(ctx *paintengine2d.Context, b paintengine2d.Rect, attachBottom bool) {
	if b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	p := l.palette
	// Flat Office XP hot-track (1px border). Square looks stay square;
	// round themes get a 1px radius so the chrome is not a Win95 bevel.
	r := l.rx(1)
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(p.MenuHover))
	ctx.DrawRoundRect(b, r, r, paintengine2d.StrokePaint(p.MenuHoverBorder, 1))
	if attachBottom {
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+1, b.Max.Y-1, b.Dx()-2, 1), paintengine2d.Fill(p.MenuHover))
	}
}

func (l *Classic) menuItemHighlightBounds(item paintengine2d.Rect) paintengine2d.Rect {
	ch := MenuChromeFor(l)
	// Full row (icon gutter + label), 1px inside the popup frame / clip.
	return paintengine2d.XYWH(item.Min.X-ch.PadL+2, item.Min.Y+1, item.Dx()+ch.PadL+ch.PadR-4, item.Dy()-2)
}

func (l *Classic) DrawMenuItem(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, row MenuRow) {
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
		l.drawMenuHighlight(ctx, l.menuItemHighlightBounds(b), false)
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
	}
	l.drawMenuGutter(ctx, b, ch, row, fg)
	label, shortcut, underline := row.Label, row.Shortcut, row.Underline
	// Clip to the item (and shortcut column), never tighter than the
	// reserved label box — glyph bearing may use the trailing item pad.
	labelRight := b.Max.X
	if shortcut != "" {
		tw := l.muted.InkWidth(shortcut)
		if tw <= 0 {
			tw = l.muted.Advance(shortcut)
		}
		sx := ch.LabelMaxX(b.Max.X) - tw
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

func (l *Classic) DrawTabBar(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	p := l.palette
	ctx.DrawRect(b, paintengine2d.Fill(p.SurfaceAlt))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
}

func (l *Classic) DrawTab(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, selected bool) {
	p := l.palette
	r := l.rx(5)
	if selected {
		ctx.DrawRoundRect(paintengine2d.XYWH(b.Min.X+2, b.Min.Y+4, b.Dx()-4, b.Dy()-4), r, r, paintengine2d.Fill(p.Surface))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+6, b.Max.Y-3, b.Dx()-12, 3), paintengine2d.Fill(p.Accent))
	} else if st.Pressed() {
		ctx.DrawRoundRect(b.Inset(3), r, r, paintengine2d.Fill(p.Surface))
	} else if st.Hovered() && !st.Disabled() {
		ctx.DrawRoundRect(b.Inset(3), r, r, paintengine2d.Fill(p.Highlight))
	}
	font := l.body
	if !selected {
		font = l.muted
	}
	l.drawCentered(ctx, font, label, b)
	if st.Focused() {
		l.DrawFocusRing(ctx, b.Inset(2))
	}
}

func (l *Classic) DrawTreeRow(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered, expanded, leaf bool, depth int, label string, bold bool) {
	p := l.palette
	m := l.metrics
	r := l.rx(5)
	if selected {
		ctx.DrawRoundRect(b.Inset(2), r, r, paintengine2d.Fill(p.Accent.WithAlpha(0.28)))
	} else if hovered {
		ctx.DrawRoundRect(b.Inset(2), r, r, paintengine2d.Fill(p.Highlight))
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
	face.Draw(ctx, label, paintengine2d.Pt(x+14, b.Min.Y+(b.Dy()-face.Height())*0.5), p.Text)
	ctx.Restore()
}

func (l *Classic) DrawStatusBar(ctx *paintengine2d.Context, b paintengine2d.Rect, parts []string) {
	p := l.palette
	ctx.DrawRect(b, paintengine2d.Fill(p.SurfaceAlt))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), 1), paintengine2d.Fill(p.Divider))
	if len(parts) == 0 {
		return
	}
	n := float32(len(parts))
	slot := b.Dx() / n
	ty := b.Min.Y + (b.Dy()-l.body.Height())*0.5
	for i, s := range parts {
		x := b.Min.X + slot*float32(i)
		if i > 0 {
			ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y+6, 1, b.Dy()-12), paintengine2d.Fill(p.Divider))
		}
		l.body.Draw(ctx, s, paintengine2d.Pt(x+10, ty), p.TextMuted)
	}
}

func (l *Classic) DrawToolBar(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	p := l.palette
	ctx.DrawRect(b, paintengine2d.Fill(p.SurfaceAlt))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
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

func (l *Classic) DrawToolButton(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, icon ToolIcon) {
	p := l.palette
	r := l.metrics.RadiusSmall
	if st.Toggle() {
		well := b.Inset(1)
		if st.Checked() {
			ctx.DrawRoundRect(well, r, r, paintengine2d.Fill(p.Accent.WithAlpha(0.32)))
			ctx.DrawRoundRect(well.Inset(0.5), r, r, paintengine2d.StrokePaint(p.Accent, 1.15))
		} else {
			fill := p.Field
			if st.Hovered() && !st.Disabled() {
				fill = p.Highlight
			}
			ctx.DrawRoundRect(well, r, r, paintengine2d.Fill(fill))
			ctx.DrawRoundRect(well.Inset(0.5), r, r, paintengine2d.StrokePaint(p.Border, 1))
		}
		if st.Pressed() && !st.Disabled() {
			ctx.DrawRoundRect(well.Inset(1), r, r, paintengine2d.Fill(p.Accent.WithAlpha(0.18)))
		}
	} else if st.Pressed() && !st.Disabled() {
		ctx.DrawRoundRect(b.Inset(2), r, r, paintengine2d.Fill(p.Accent.WithAlpha(0.28)))
	} else if st.Hovered() && !st.Disabled() {
		ctx.DrawRoundRect(b.Inset(2), r, r, paintengine2d.Fill(p.Highlight))
	}
	if st.Focused() {
		l.DrawFocusRing(ctx, b.Inset(1))
	}
	fg := p.Text
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

func (l *Classic) DrawProgressBar(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, t float32, indeterminate bool, phase float32) {
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

func (l *Classic) DrawRadio(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, selected bool, label string) {
	p := l.palette
	m := l.metrics
	side := m.Radio
	if side <= 0 {
		side = m.Checkbox
	}
	if side <= 0 {
		side = 18
	}
	cx := b.Min.X + side*0.5
	cy := (b.Min.Y + b.Max.Y) * 0.5
	fill := p.Field
	if selected {
		fill = p.Accent
	}
	if st.Disabled() {
		if selected {
			fill = p.Divider
		} else {
			fill = p.Surface
		}
	} else if st.Hovered() {
		fill = fill.Lerp(p.AccentHover, 0.25)
	}
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), side*0.5, paintengine2d.Fill(fill))
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), side*0.5-0.5, paintengine2d.StrokePaint(p.FieldBorder, 1))
	if selected {
		ctx.DrawCircle(paintengine2d.Pt(cx, cy), side*0.18+1.2, paintengine2d.Fill(p.TextOnAccent))
	}
	if st.Focused() {
		l.DrawFocusRing(ctx, paintengine2d.XYWH(cx-side*0.5-2, cy-side*0.5-2, side+4, side+4))
	}
	if label != "" {
		f := l.body
		if st.Disabled() {
			f = l.muted
		}
		lb := paintengine2d.XYWH(b.Min.X+side+8, b.Min.Y, b.Max.X-(b.Min.X+side+8), b.Dy())
		l.drawFittedText(ctx, f, label, lb, p.Text, AlignStart, 0)
	}
}

func (l *Classic) DrawComboBox(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text string, open bool) {
	p := l.palette
	m := l.metrics
	r := m.RadiusSmall
	field := p.Field
	if st.Disabled() {
		field = p.Surface
	}
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(field))
	border := p.FieldBorder
	if st.Disabled() {
		border = p.Divider
	} else if st.Focused() || open {
		border = p.Focus
	} else if st.Hovered() {
		border = p.Border
	}
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(border, m.Border+float32(btoi(st.Focused() || open))))
	if st.Focused() && !open {
		l.DrawFocusRing(ctx, b.Inset(-2))
	}
	if open {
		ctx.DrawRoundRect(b.Inset(1), r, r, paintengine2d.Fill(p.Highlight))
	}
	pad := m.FieldPad
	if pad <= 0 {
		pad = 8
	}
	chevW := float32(22)
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad-chevW, b.Dy())
	f := l.body
	col := p.Text
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
	ctx.DrawPath(chev, paintengine2d.Fill(p.TextMuted))
}

func (l *Classic) DrawTitleBar(ctx *paintengine2d.Context, b paintengine2d.Rect, title, subtitle string) {
	p := l.palette
	ctx.DrawRect(b, paintengine2d.Fill(p.SurfaceAlt))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
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

func (l *Classic) DrawMessageIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon) {
	if icon == IconNone {
		return
	}
	l.drawToolIcon(ctx, b, icon, l.messageIconColor(icon))
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

func (l *Classic) DrawTableHeader(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, label string, sorted, asc bool) {
	p := l.palette
	ctx.DrawRect(b, paintengine2d.Fill(p.SurfaceAlt))
	if st.Pressed() {
		ctx.DrawRect(b.Inset(1), paintengine2d.Fill(p.Accent.WithAlpha(0.22)))
	} else if st.Hovered() && !st.Disabled() {
		ctx.DrawRect(b.Inset(1), paintengine2d.Fill(p.Highlight))
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
	l.body.Draw(ctx, l.body.Fit(label, inner.Dx()), paintengine2d.Pt(inner.Min.X, ty), p.Text)
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

func (l *Classic) DrawTableCell(ctx *paintengine2d.Context, b paintengine2d.Rect, selected, hovered bool, label string, align Align, face *Font) {
	p := l.palette
	if selected {
		ctx.DrawRect(b, paintengine2d.Fill(p.Accent.WithAlpha(0.28)))
	} else if hovered {
		ctx.DrawRect(b, paintengine2d.Fill(p.Highlight))
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
	f.Draw(ctx, label, paintengine2d.Pt(x, ty), p.Text)
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

func (l *Classic) DrawSpinner(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, upHover, downHover, upPress, downPress bool) {
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

func (l *Classic) DrawTextArea(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, lines []TextLine, caret, selA, selB int, blink bool, scrollX, scrollY float32, placeholder string, face *Font) {
	p := l.palette
	m := l.metrics
	r := m.RadiusSmall
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(p.Field))
	border := p.FieldBorder
	if st.Focused() {
		border = p.Focus
	} else if st.Hovered() {
		border = p.Border
	}
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(border, m.Border+float32(btoi(st.Focused()))))
	if st.Focused() {
		l.DrawFocusRing(ctx, b.Inset(-2))
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
			ctx.DrawRect(paintengine2d.XYWH(x0, y+1, x1-x0, f.Height()-2), paintengine2d.Fill(p.Selection))
		}
		if line.Text != "" {
			f.Draw(ctx, line.Text, paintengine2d.Pt(ox, y), p.Text)
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
				ctx.DrawRect(paintengine2d.XYWH(cx, y+1, 1.6, f.Height()-2), paintengine2d.Fill(p.Accent))
			}
		}
	}
	ctx.Restore()
}

func (l *Classic) DrawSwitch(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, on bool, label string) {
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
		l.DrawFocusRing(ctx, track.Inset(-3))
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

func (l *Classic) DrawAccordionHeader(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, title string, expanded bool) {
	p := l.palette
	if st.Pressed() {
		ctx.DrawRect(b, paintengine2d.Fill(p.Accent.WithAlpha(0.22)))
	} else if st.Hovered() && !st.Disabled() {
		ctx.DrawRect(b, paintengine2d.Fill(p.Highlight))
	} else {
		ctx.DrawRect(b, paintengine2d.Fill(p.SurfaceAlt))
	}
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
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

func (l *Classic) DrawSeparator(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	p := l.palette
	if vertical {
		x := (b.Min.X + b.Max.X) * 0.5
		ctx.DrawRect(paintengine2d.XYWH(x, b.Min.Y+4, 1, b.Dy()-8), paintengine2d.Fill(p.Divider))
		return
	}
	y := (b.Min.Y + b.Max.Y) * 0.5
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), 1), paintengine2d.Fill(p.Divider))
}

func (l *Classic) DrawTooltip(ctx *paintengine2d.Context, b paintengine2d.Rect, text string) {
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
