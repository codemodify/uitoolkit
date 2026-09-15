package style

import "github.com/codemodify/paintengine2d"

// FaceLook paints a control face for a role (an engine's Face).
type FaceLook interface {
	DrawFace(ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color
}

// DrawFace implements [FaceLook]: the engine's face for role in state st,
// returning the colour to draw a label or glyph on it with.
func (l *Classic) DrawFace(ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	return l.eng().Face(l, ctx, b, role, st)
}

// DrawFaceOf paints lk's face for role in state st over b and returns the
// foreground colour for it: widgets that draw their own glyph on a themed
// face (window caption buttons) use it. Looks without engine faces paint a
// tool button and answer their text colour.
func DrawFaceOf(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, role Role, st ControlState) paintengine2d.Color {
	if f, ok := lk.(FaceLook); ok {
		return f.DrawFace(ctx, b, role, st)
	}
	lk.DrawToolButton(ctx, b, st, "", IconNone)
	return lk.Palette().Text
}
