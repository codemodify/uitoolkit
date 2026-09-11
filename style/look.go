package style

import "github.com/codemodify/paintengine2d"

// Classic is the stock LookAndFeel (rounded graphite chrome + blue accent).
type Classic struct {
	palette Palette
	metrics Metrics
	name    string
	body    *Font
	title   *Font
	muted   *Font
	onAcc   *Font
}

// NewClassic builds fonts for p. Name is "dark" or "light" typically.
func NewClassic(name string, p Palette, m Metrics) *Classic {
	if m.FontSize < 8 {
		m = DefaultMetrics()
	}
	return &Classic{
		palette: p,
		metrics: m,
		name:    name,
		body:    BakeFont(m.FontSize, p.Text),
		title:   BakeFont(m.TitleSize, p.Text),
		muted:   BakeFont(m.FontSize, p.TextMuted),
		onAcc:   BakeFont(m.FontSize, p.TextOnAccent),
	}
}

// DarkLook is the default night skin.
func DarkLook() *Classic { return NewClassic("dark", Dark(), DefaultMetrics()) }

// LightLook is the paper skin.
func LightLook() *Classic { return NewClassic("light", Light(), DefaultMetrics()) }

func (l *Classic) Name() string        { return l.name }
func (l *Classic) Palette() Palette    { return l.palette }
func (l *Classic) Metrics() Metrics    { return l.metrics }
func (l *Classic) Font() *Font         { return l.body }
func (l *Classic) TitleFont() *Font    { return l.title }
func (l *Classic) MutedFont() *Font    { return l.muted }
func (l *Classic) OnAccentFont() *Font { return l.onAcc }

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
	r := m.RadiusSmall + 1
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
	tw := f.Advance(text)
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
	f.Draw(ctx, text, paintengine2d.Pt(x, y), col)
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
	if st.Hovered() && !st.Disabled() {
		fill = fill.Lerp(p.AccentHover, 0.25)
	}
	ctx.DrawRoundRect(box, 4, 4, paintengine2d.Fill(fill))
	ctx.DrawRoundRect(box.Inset(0.5), 4, 4, paintengine2d.StrokePaint(p.FieldBorder, 1))
	if checked {
		chk := paintengine2d.NewPath()
		chk.MoveTo(box.Min.X+4, box.Min.Y+side*0.52)
		chk.LineTo(box.Min.X+side*0.42, box.Max.Y-4.5)
		chk.LineTo(box.Max.X-3.5, box.Min.Y+4)
		ctx.DrawPath(chk, paintengine2d.Paint{
			Color:  p.TextOnAccent,
			Style:  paintengine2d.StyleStroke,
			Stroke: paintengine2d.Stroke{Width: 2.1, Cap: paintengine2d.CapRound, Join: paintengine2d.JoinRound, MiterLimit: 4},
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
		f.Draw(ctx, label, paintengine2d.Pt(box.Max.X+8, b.Min.Y+(b.Dy()-f.Height())*0.5), p.Text)
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
	ctx.DrawRoundRect(track, 2, 2, paintengine2d.Fill(p.Track))
	fillW := (x1 - x0) * t
	ctx.DrawRoundRect(paintengine2d.XYWH(x0, cy-2, fillW, 4), 2, 2, paintengine2d.Fill(p.Accent))
	tx := x0 + (x1-x0)*t
	rad := m.Thumb * 0.5
	if st.Hovered() || st.Pressed() {
		rad += 1
	}
	ctx.DrawCircle(paintengine2d.Pt(tx, cy+1), rad+1, paintengine2d.Fill(p.Shadow))
	col := p.Accent
	if st.Pressed() {
		col = p.AccentPress
	} else if st.Hovered() {
		col = p.AccentHover
	}
	ctx.DrawCircle(paintengine2d.Pt(tx, cy), rad, paintengine2d.Fill(col))
	ctx.DrawCircle(paintengine2d.Pt(tx-rad*0.25, cy-rad*0.25), rad*0.35, paintengine2d.Fill(p.Highlight))
	if st.Focused() {
		l.DrawFocusRing(ctx, paintengine2d.XYWH(tx-rad-3, cy-rad-3, rad*2+6, rad*2+6))
	}
}

func (l *Classic) DrawTextField(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState, text, placeholder string, caret, selA, selB int, blink bool, scrollX float32) {
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
	inner := paintengine2d.XYWH(b.Min.X+pad, b.Min.Y, b.Dx()-pad*2, b.Dy())
	ctx.Save()
	ctx.ClipRect(inner)
	f := l.body
	show := text
	font := f
	if text == "" && placeholder != "" && !st.Focused() {
		show = placeholder
		font = l.muted
	}
	ty := inner.Min.Y + (inner.Dy()-font.Height())*0.5
	ox := inner.Min.X - scrollX
	if selA != selB && text != "" {
		if selA > selB {
			selA, selB = selB, selA
		}
		x0 := ox + f.CaretX(text, selA)
		x1 := ox + f.CaretX(text, selB)
		ctx.DrawRect(paintengine2d.XYWH(x0, ty+1, x1-x0, f.Height()-2), paintengine2d.Fill(p.Selection))
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
	ctx.DrawRoundRect(track, 4, 4, paintengine2d.Fill(p.Track.WithAlpha(0.55)))
	col := p.Thumb
	if st.Hovered() || st.Pressed() {
		col = p.Accent
	}
	ctx.DrawRoundRect(thumb, 4, 4, paintengine2d.Fill(col))
}

func (l *Classic) DrawFocusRing(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	p := l.palette
	m := l.metrics
	ctx.DrawRoundRect(b, m.Radius, m.Radius, paintengine2d.StrokePaint(p.Focus.WithAlpha(0.40), m.FocusWidth+1.5))
	ctx.DrawRoundRect(b.Inset(1.25), m.RadiusSmall+1, m.RadiusSmall+1, paintengine2d.StrokePaint(p.Focus.WithAlpha(0.92), 1.15))
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
	if selected {
		ctx.DrawRoundRect(b.Inset(2), 5, 5, paintengine2d.Fill(p.Accent.WithAlpha(0.28)))
	} else if hovered {
		ctx.DrawRoundRect(b.Inset(2), 5, 5, paintengine2d.Fill(p.Highlight))
	}
	l.body.Draw(ctx, label, paintengine2d.Pt(b.Min.X+10, b.Min.Y+(b.Dy()-l.body.Height())*0.5), p.Text)
}

func (l *Classic) DrawOverlay(ctx *paintengine2d.Context, b paintengine2d.Rect) {
	ctx.DrawRect(b, paintengine2d.Fill(l.palette.Overlay))
}

func (l *Classic) drawCentered(ctx *paintengine2d.Context, f *Font, text string, b paintengine2d.Rect) {
	if f == nil || text == "" {
		return
	}
	tw := f.Advance(text)
	th := f.Height()
	col := f.Color
	if col == (paintengine2d.Color{}) {
		col = l.palette.Text
	}
	f.Draw(ctx, text, paintengine2d.Pt(b.Min.X+(b.Dx()-tw)*0.5, b.Min.Y+(b.Dy()-th)*0.5), col)
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
