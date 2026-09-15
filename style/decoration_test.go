package style

import (
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// decorationStates are the frame states every look is checked in.
var decorationStates = []DecorationState{
	{Active: true},
	{},
	{Active: true, Maximized: true},
	{Active: true, Custom: true},
	{Active: true, Tiled: EdgeLeft | EdgeTop | EdgeBottom},
}

func whole(v float32) bool { return v == float32(math.Round(float64(v))) }

// Every look's frame is sane in every state and at every scale: a caption
// that holds its buttons, borders on whole device pixels, none while
// maximized.
func TestDecorationSpecs(t *testing.T) {
	for _, p := range ListBuiltinThemes() {
		for _, scale := range []float32{1, 1.75, 2} {
			lk := p.Look().setScale(scale)
			for _, st := range decorationStates {
				s := DecorationOf(lk, st)
				if s.Caption < 8 || s.Button.X < 8 || s.Button.Y < 0 {
					t.Fatalf("%s @%vx %+v: caption %v button %v", p.Name, scale, st, s.Caption, s.Button)
				}
				for _, v := range []float32{s.Border.Top, s.Border.Right, s.Border.Bottom, s.Border.Left} {
					if v < 0 || !whole(v) {
						t.Fatalf("%s @%vx %+v: border %+v not whole pixels", p.Name, scale, st, s.Border)
					}
				}
				if st.Maximized && s.Border != (Insets{}) {
					t.Fatalf("%s @%vx: a maximized frame has no border: %+v", p.Name, scale, s.Border)
				}
				if s.Button.Y > 0 && s.ButtonPad.Top+s.Button.Y > s.Caption+0.01 {
					t.Fatalf("%s @%vx %+v: buttons (%v below %v) overhang the caption %v", p.Name, scale, st, s.Button.Y, s.ButtonPad.Top, s.Caption)
				}
			}
		}
	}
}

// paintedOutside reports the first pixel with ink outside r (grown by slack)
// in an image that started transparent.
func paintedOutside(img *paintengine2d.Image, r paintengine2d.Rect, slack float32) (int, int, bool) {
	g := r.Inset(-slack)
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a == 0 {
				continue
			}
			if float32(x) < g.Min.X || float32(y) < g.Min.Y || float32(x)+1 > g.Max.X || float32(y)+1 > g.Max.Y {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// The frame paints inside the window, the title and the buttons inside
// their boxes: the window repaints only those when they change.
func TestDecorationPaintsInsideItsBoxes(t *testing.T) {
	for _, p := range ListBuiltinThemes() {
		for _, scale := range []float32{1, 2} {
			lk := p.Look().setScale(scale)
			for _, st := range decorationStates {
				s := DecorationOf(lk, st)
				win := paintengine2d.XYWH(20, 20, float32(math.Round(float64(lk.S(360)))), float32(math.Round(float64(lk.S(200)))))
				inner := s.Border.Apply(win)
				capH := float32(math.Ceil(float64(s.Caption)))
				f := DecorationFrame{Window: win, Caption: paintengine2d.XYWH(inner.Min.X, inner.Min.Y, inner.Dx(), capH)}
				if s.Stacked && st.Custom {
					f.Bar = paintengine2d.XYWH(inner.Min.X, f.Caption.Max.Y, inner.Dx(), lk.S(36))
				}
				img := paintengine2d.NewImage(int(win.Max.X)+20, int(win.Max.Y)+20)
				DrawDecorationOf(lk, paintengine2d.NewContext(img), f, st)
				if x, y, bad := paintedOutside(img, win, 0); bad {
					t.Fatalf("%s @%vx %+v: frame paints (%d,%d) outside the window %v", p.Name, scale, st, x, y, win)
				}
				tb := paintengine2d.XYWH(f.Caption.Min.X+lk.S(60), f.Caption.Min.Y, f.Caption.Dx()-lk.S(120), capH)
				img = paintengine2d.NewImage(int(win.Max.X)+20, int(win.Max.Y)+20)
				DrawCaptionTitleOf(lk, paintengine2d.NewContext(img), tb, "Window title", st)
				if x, y, bad := paintedOutside(img, tb, 0); bad {
					t.Fatalf("%s @%vx %+v: title paints (%d,%d) outside its box %v", p.Name, scale, st, x, y, tb)
				}
				bh := s.Button.Y
				if bh <= 0 {
					bh = capH - s.ButtonPad.Top
				}
				// Each button on a small canvas of its own, 8px round it.
				bb := paintengine2d.XYWH(8, 8, s.Button.X, bh)
				for _, k := range []CaptionButton{CaptionClose, CaptionMinimize, CaptionMaximize, CaptionMenu} {
					for _, cs := range []ControlState{StateNone, StateHovered, StateHovered | StatePressed} {
						img = paintengine2d.NewImage(int(bb.Max.X)+8, int(bb.Max.Y)+8)
						DrawCaptionButtonOf(lk, paintengine2d.NewContext(img), bb, k, cs, st)
						if x, y, bad := paintedOutside(img, bb, 1); bad {
							t.Fatalf("%s @%vx %+v: button %d state %v paints (%d,%d) outside its box %v", p.Name, scale, st, k, cs, x, y, bb)
						}
					}
				}
			}
		}
	}
}

// Engines without a frame of their own get their in-app window's: its
// caption height, its borders and its close button's size.
func TestDecorationAdapterUsesTheInAppFrame(t *testing.T) {
	for _, p := range ListBuiltinThemes() {
		lk := p.Look()
		if _, native := lk.eng().(DecorationEngine); native || lk.eng().ID() == baseEngine.ID() {
			continue
		}
		in := lk.eng().WindowFrameInsets(lk)
		s := DecorationOf(lk, DecorationState{Active: true})
		if !s.Stacked || s.Caption != snap(in.Top) || s.Border.Left != snap(in.Left) || s.Border.Bottom != snap(in.Bottom) {
			t.Fatalf("%s: adapted spec %+v from insets %+v", p.Name, s, in)
		}
		if cr := lk.eng().WindowCloseRect(lk, adapterProbe(lk)); !cr.Empty() && s.Button != winSnap(cr).Size() {
			t.Fatalf("%s: buttons %v, the in-app close is %v", p.Name, s.Button, cr.Size())
		}
	}
}

// The plain frame is the base look's and every non-engine look's.
func TestDecorationPlainFrame(t *testing.T) {
	for _, lk := range []LookAndFeel{DarkLook(), LightLook()} {
		if NativeDecoration(lk) {
			t.Fatal("the base look has no engine frame")
		}
		s := DecorationOf(lk, DecorationState{Active: true})
		if s.Stacked || s.Border != (Insets{Top: 1, Right: 1, Bottom: 1, Left: 1}) || s.Button.Y != 0 {
			t.Fatalf("plain spec %+v", s)
		}
	}
}

// Glyphs sit on whole pixels and inside their box at any scale.
func TestCaptionGlyphsAreCrisp(t *testing.T) {
	for _, k := range []CaptionButton{CaptionMinimize, CaptionMaximize, CaptionMenu} {
		for _, maxed := range []bool{false, true} {
			for _, s := range []float32{10, 12.5, 17.5, 20} {
				img := paintengine2d.NewImage(40, 40)
				DrawCaptionGlyph(paintengine2d.NewContext(img), paintengine2d.XYWH(5, 5, 30, 30), k, maxed, paintengine2d.RGB(0, 0, 0), s, s/10)
				for y := 0; y < 40; y++ {
					for x := 0; x < 40; x++ {
						if _, _, _, a := img.At(x, y).RGBA(); a != 0 && a != 0xffff {
							t.Fatalf("glyph %d max %v side %v: soft pixel (%d,%d) alpha %d", k, maxed, s, x, y, a)
						}
					}
				}
			}
		}
	}
}
