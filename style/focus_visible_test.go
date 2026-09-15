package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Keyboard focus must be visible in every look: a focused control paints
// differently from an unfocused one. (Text fields are left out: Win95 and
// Motif show focus with the caret alone.)
func TestFocusIsVisibleInEveryLook(t *testing.T) {
	type control struct {
		name string
		w, h int
		draw func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState)
	}
	controls := []control{
		{"button", 120, 32, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawButton(ctx, b, st, "Button")
		}},
		{"checkbox", 140, 28, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawCheckbox(ctx, b, st, true, "Check")
		}},
		{"radio", 140, 28, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawRadio(ctx, b, st, true, "Radio")
		}},
		{"slider", 160, 28, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawSlider(ctx, b, st, 0.4)
		}},
		{"switch", 140, 30, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawSwitch(ctx, b, st, true, "Switch")
		}},
		{"tab", 110, 30, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawTab(ctx, b, st, "Tab", true)
		}},
		{"current row", 170, 24, func(lk *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Field))
			lk.DrawListRow(ctx, b, st|StateChecked, "Row")
			if st.Focused() {
				// Looks without a row mark ring the focused view instead.
				DrawViewFrameOf(lk, ctx, b, st)
				if ViewFrameInsetsOf(lk).Zero() {
					return
				}
			}
		}},
	}
	render := func(lk *Classic, c control, st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(c.w, c.h)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(c.w), float32(c.h)), paintengine2d.Fill(lk.Palette().Background))
		c.draw(lk, ctx, paintengine2d.XYWH(2, 2, float32(c.w-4), float32(c.h-4)), st)
		return img
	}
	for _, p := range ListBuiltinThemes() {
		lk := p.Look()
		for _, c := range controls {
			plain, focused := render(lk, c, StateNone), render(lk, c, StateFocused)
			diff := 0
			for i := 0; i < len(plain.Pix); i += 4 {
				if plain.Pix[i] != focused.Pix[i] || plain.Pix[i+1] != focused.Pix[i+1] || plain.Pix[i+2] != focused.Pix[i+2] {
					diff++
				}
			}
			if diff < 8 {
				t.Errorf("%s: a focused %s looks unfocused (%d pixels differ)", p.Name, c.name, diff)
			}
		}
	}
}
