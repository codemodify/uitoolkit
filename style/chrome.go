package style

import "github.com/codemodify/paintengine2d"

// chromeRole selects which token state a control face reads.
type chromeRole int

const (
	roleButton chromeRole = iota
	roleTool
	roleField
	roleCheck
	roleRow
	roleTab
	roleThumb
	roleTrack
	roleMenu
	roleCombo
	roleSplitter
	roleBar
	rolePanel
)

// LookTokens reads ThemeTokens from a Classic look (family fallback otherwise).
func LookTokens(lk LookAndFeel) ThemeTokens {
	if c, ok := lk.(*Classic); ok && c != nil {
		return c.Tokens()
	}
	if lk == nil {
		return ThemeTokens{}.Resolve()
	}
	t := ThemeTokens{Family: ParseTheme(lk.Name()), Palette: lk.Palette()}
	return t.Resolve()
}

func (l *Classic) Tokens() ThemeTokens {
	if l == nil {
		return ThemeTokens{}.Resolve()
	}
	if l.tokens.Empty() {
		return ThemeTokens{Family: ParseTheme(l.name), Palette: l.palette}.Resolve()
	}
	return l.tokens
}

func (l *Classic) tokensOr() ThemeTokens {
	return l.Tokens()
}

func (l *Classic) faceColors(role chromeRole, st ControlState) (fill, border, fg paintengine2d.Color) {
	p := l.palette
	t := l.tokensOr()
	fg = p.Text
	fill = p.SurfaceAlt
	border = p.Border

	switch role {
	case roleButton:
		fill = p.SurfaceAlt
		if colorUnset(fill) {
			fill = p.Surface
		}
		if st.Primary() && !st.Disabled() {
			fill = p.Accent
			fg = p.TextOnAccent
			border = p.AccentPress
		}
	case roleTool:
		fill = paintengine2d.Color{}
		border = paintengine2d.Color{}
		if st.Toggle() {
			fill = p.Field
			border = p.Border
			if st.Checked() {
				fill = t.Selected.Fill
				if colorUnset(fill) {
					fill = p.Accent.WithAlpha(0.32)
				}
				border = t.Selected.Border
				if colorUnset(border) {
					border = p.Accent
				}
			}
		}
	case roleField, roleCombo:
		fill = p.Field
		border = p.FieldBorder
	case roleCheck:
		fill = p.Field
		border = p.FieldBorder
		if st.Checked() {
			fill = p.Accent
			border = p.AccentPress
		}
	case roleRow:
		fill = paintengine2d.Color{}
		border = paintengine2d.Color{}
	case roleTab:
		fill = paintengine2d.Color{}
		border = paintengine2d.Color{}
	case roleThumb:
		fill = p.Thumb
		border = p.Border
	case roleTrack:
		fill = p.Track
		if t.Bevel != BevelClassic3D {
			fill = p.Track.WithAlpha(0.55)
		}
		border = p.Divider
	case roleMenu:
		fill = t.Hot.Fill
		border = t.Hot.Border
	case roleSplitter:
		fill = p.Divider
		border = p.Divider
	case roleBar, rolePanel:
		fill = p.SurfaceAlt
		border = p.Divider
	}

	if st.Disabled() {
		fg = p.TextMuted
		if role == roleButton || role == roleTool || role == roleCheck {
			fill = t.Disabled.Fill
			if colorUnset(fill) {
				fill = p.Surface
			}
			border = t.Disabled.Border
			if colorUnset(border) {
				border = p.Divider
			}
		}
		if role == roleField || role == roleCombo {
			fill = p.Surface
			border = p.Divider
		}
		return fill, border, fg
	}

	hot := st.Hovered() || (role == roleMenu && st.Pressed())
	if st.Pressed() && (role == roleButton || role == roleTool || role == roleCheck || role == roleCombo || role == roleTab) {
		if !st.Primary() || role != roleButton {
			if !colorUnset(t.Pressed.Fill) {
				fill = t.Pressed.Fill
			} else {
				fill = fill.Lerp(p.AccentPress, 0.22)
				fill.A = 1
			}
			if !colorUnset(t.Pressed.Border) {
				border = t.Pressed.Border
			}
		} else {
			fill = p.AccentPress
			border = p.AccentPress
			fg = p.TextOnAccent
		}
		return fill, border, fg
	}

	if st.Checked() && (role == roleRow || role == roleTab) {
		fill = t.Selected.Fill
		border = t.Selected.Border
		if t.Bevel == BevelClassic3D || t.Bevel == BevelLunaHottrack {
			if t.Selected.Fill.A > 0.7 {
				fg = p.TextOnAccent
			}
		}
		return fill, border, fg
	}

	if hot {
		switch role {
		case roleButton:
			if !st.Primary() {
				fill = t.Hot.Fill
				border = t.Hot.Border
				if t.Bevel == BevelClassic3D {
					// Win95 / Motif buttons barely wash; the bevel is the language.
					fill = p.SurfaceAlt.Lerp(p.BevelLight, 0.18)
					fill.A = 1
					border = p.BevelDark
				}
			} else {
				fill = p.AccentHover
				border = p.AccentPress
				fg = p.TextOnAccent
			}
		case roleTool, roleTab, roleCombo:
			fill = t.Hot.Fill
			border = t.Hot.Border
		case roleField:
			border = p.Border
			if t.Bevel == BevelLunaHottrack {
				border = t.Hot.Border
			}
		case roleCheck:
			if !st.Checked() {
				fill = p.Field.Lerp(t.Hot.Fill, 0.45)
				fill.A = 1
			}
			border = t.Hot.Border
		case roleThumb:
			if t.Bevel == BevelLunaHottrack {
				fill = t.Hot.Fill
				border = t.Hot.Border
			} else {
				fill = p.Accent
			}
		case roleRow:
			fill = t.Hot.Fill
			if t.Bevel == BevelNone || t.Bevel == BevelFluentAccent || t.Bevel == BevelSoftShadow {
				if fill.A > 0.95 {
					fill = fill.WithAlpha(0.55)
				}
			}
			border = t.Hot.Border
		case roleSplitter:
			fill = p.Accent
			border = p.Accent
		}
	}

	if st.Focused() && !hot && !st.Pressed() {
		switch role {
		case roleField, roleCombo:
			border = t.Focus.Border
			if colorUnset(border) {
				border = p.Focus
			}
		case roleButton:
			if t.Bevel == BevelLunaHottrack {
				border = t.Hot.Border
			}
		}
	}

	if st.Checked() && role == roleRow {
		fill = t.Selected.Fill
		border = t.Selected.Border
	}
	return fill, border, fg
}

func (l *Classic) faceRadius(role chromeRole) float32 {
	m := l.metrics
	r := m.RadiusSmall
	switch role {
	case roleBar, rolePanel, roleRow:
		r = m.Radius
		if r > 5 && (role == roleRow) {
			r = l.rx(5)
		}
	case roleCheck:
		return l.rx(4)
	case roleThumb, roleTrack:
		return l.rx(4)
	}
	if r > 0 && role == roleButton {
		r += 1
	}
	return r
}

func (l *Classic) paintFace(ctx *paintengine2d.Context, b paintengine2d.Rect, role chromeRole, st ControlState) (fg paintengine2d.Color) {
	if ctx == nil || b.Empty() {
		return
	}
	fill, border, fg := l.faceColors(role, st)
	l.paintBezel(ctx, b, fill, border, role, st)
	return fg
}

func (l *Classic) paintBezel(ctx *paintengine2d.Context, b paintengine2d.Rect, fill, border paintengine2d.Color, role chromeRole, st ControlState) {
	t := l.tokensOr()
	r := l.faceRadius(role)
	p := l.palette
	transparent := colorUnset(fill) || fill.A < 0.02

	switch t.Bevel {
	case BevelClassic3D:
		if !transparent {
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
		}
		inset := false
		switch role {
		case roleField, roleCombo, roleTrack, roleCheck:
			inset = true
		case roleButton, roleTool, roleThumb:
			inset = st.Pressed() && !st.Disabled()
		}
		depth := t.Metrics.BevelDepth
		if depth <= 0 {
			depth = 1
		}
		if role == roleBar || role == rolePanel || role == roleRow || role == roleSplitter {
			if role == roleBar {
				ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.BevelDark))
				ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), 1), paintengine2d.Fill(p.BevelLight))
			}
			if role == roleRow && !transparent {
				DrawBevel3D(ctx, b, p.BevelLight, p.BevelDark, 1, true)
			}
			return
		}
		DrawBevel3D(ctx, b, p.BevelLight, p.BevelDark, depth, inset)
	case BevelLunaHottrack:
		if !transparent {
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
		}
		show := !st.Disabled() && (st.Hovered() || st.Pressed() || st.Focused() || st.Checked() ||
			role == roleField || role == roleCombo || role == roleThumb || role == roleButton || role == roleMenu)
		if role == roleBar || role == rolePanel {
			if !transparent {
				ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
			}
			return
		}
		if show && !colorUnset(border) {
			ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(border, 1))
		} else if (role == roleField || role == roleCombo || role == roleButton) && !st.Disabled() {
			ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(p.Border.WithAlpha(0.45), 1))
		}
	case BevelSoftShadow:
		elev := t.Metrics.Elevation
		if role == roleButton || role == roleTool {
			if st.Hovered() && !st.Disabled() {
				elev++
			}
			if st.Pressed() && !st.Disabled() {
				elev = 0
			}
		}
		if role == roleField || role == roleCombo {
			elev = 0
		}
		if elev > 0 && !transparent {
			off := float32(elev)
			sh := p.Shadow
			if sh.A < 0.05 {
				sh = paintengine2d.RGBA(0, 0, 0, 0.20)
			}
			ctx.DrawRoundRect(b.Translate(paintengine2d.Pt(0, off*0.85)), r, r, paintengine2d.Fill(sh))
		}
		if !transparent {
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
		}
		if !colorUnset(border) && (st.Focused() || st.Hovered() || role == roleField || role == roleCombo || role == roleButton) {
			ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(border.WithAlpha(0.75), 1))
		}
	case BevelFluentAccent:
		if !transparent {
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
		}
		if role == roleBar || role == rolePanel {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
			return
		}
		if st.Focused() && !st.Disabled() && (role == roleButton || role == roleField || role == roleCombo || role == roleTool) {
			acc := t.Focus.Border
			if colorUnset(acc) {
				acc = p.Accent
			}
			ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(acc, 1.4))
			bar := float32(3)
			if s := LookScale(l); s > 1 {
				bar *= s
			}
			inset := r
			if inset < 2 {
				inset = 2
			}
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X+inset, b.Max.Y-bar, b.Dx()-inset*2, bar), paintengine2d.Fill(acc))
		} else if !colorUnset(border) {
			a := float32(0.55)
			if st.Hovered() {
				a = 0.85
			}
			ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(border.WithAlpha(a), 1))
		}
	default: // none — Breeze / FlatLaf
		if !transparent {
			ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(fill))
		}
		if role == roleBar || role == rolePanel {
			ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-1, b.Dx(), 1), paintengine2d.Fill(p.Divider))
			return
		}
		if !colorUnset(border) && (st.Focused() || st.Hovered() || st.Pressed() || st.Checked() ||
			role == roleField || role == roleCombo || role == roleButton || role == roleCheck || role == roleThumb) {
			w := l.metrics.Border
			if w < 1 {
				w = 1
			}
			if st.Focused() && (role == roleField || role == roleCombo || role == roleButton) {
				w += 1
				if colorUnset(border) {
					border = p.Focus
				}
			}
			ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(border, w))
		}
	}
}

// DrawBevel3D paints Motif/Win95 highlight (top/left) and shadow (bottom/right)
// edges. inset flips the pair (pressed buttons, fields, wells).
func DrawBevel3D(ctx *paintengine2d.Context, b paintengine2d.Rect, hi, lo paintengine2d.Color, depth float32, inset bool) {
	if ctx == nil || b.Dx() < 2 || b.Dy() < 2 {
		return
	}
	if inset {
		hi, lo = lo, hi
	}
	n := int(depth + 0.5)
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	for i := 0; i < n; i++ {
		f := float32(i)
		w := b.Dx() - 2*f
		h := b.Dy() - 2*f
		if w < 2 || h < 2 {
			break
		}
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+f, b.Min.Y+f, w, 1), paintengine2d.Fill(hi))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+f, b.Min.Y+f, 1, h), paintengine2d.Fill(hi))
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+f, b.Max.Y-1-f, w, 1), paintengine2d.Fill(lo))
		ctx.DrawRect(paintengine2d.XYWH(b.Max.X-1-f, b.Min.Y+f, 1, h), paintengine2d.Fill(lo))
	}
}

// DrawHotTrack paints an Office XP / Luna pale fill plus a 1px border.
func DrawHotTrack(ctx *paintengine2d.Context, b paintengine2d.Rect, fill, border paintengine2d.Color, radius float32) {
	if ctx == nil || b.Empty() {
		return
	}
	ctx.DrawRoundRect(b, radius, radius, paintengine2d.Fill(fill))
	ctx.DrawRoundRect(b.Inset(0.5), radius, radius, paintengine2d.StrokePaint(border, 1))
}

func (l *Classic) paintFocus(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
	if ctx == nil || !st.Focused() || st.Disabled() {
		return
	}
	l.DrawFocusRing(ctx, b)
}

func (l *Classic) menuInvertText() bool {
	t := l.tokensOr()
	switch t.Bevel {
	case BevelClassic3D:
		return true
	default:
		// NeXT / solid-select packs also invert when hover is near-black.
		h := t.Hot.Fill
		return h.R+h.G+h.B < 0.35 && h.A > 0.7
	}
}
