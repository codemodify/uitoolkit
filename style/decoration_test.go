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

// themePack is the built-in pack named name.
func themePack(t *testing.T, name string) ThemePack {
	t.Helper()
	for _, p := range ListBuiltinThemes() {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("no built-in pack %q", name)
	return ThemePack{}
}

// Every look's frame is sane in every state and at every scale: a caption
// that holds its buttons, borders on whole device pixels, none while
// maximized.
func TestDecorationSpecs(t *testing.T) {
	for _, p := range ListBuiltinThemes() {
		for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
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
				if s.CloseButton.Y > 0 && s.ButtonPad.Top+s.CloseButton.Y > s.Caption+0.01 {
					t.Fatalf("%s @%vx %+v: the close button overhangs the caption", p.Name, scale, st)
				}
				if st.Maximized && NativeDecoration(lk) && s.Radius != ([4]float32{}) {
					t.Fatalf("%s @%vx: a maximized frame is square: %v", p.Name, scale, s.Radius)
				}
			}
		}
	}
}

// paintedOutside reports the first pixel with ink outside r (grown by slack)
// in an image that started transparent; it looks at the pixels outside only.
func paintedOutside(img *paintengine2d.Image, r paintengine2d.Rect, slack float32) (int, int, bool) {
	g := r.Inset(-slack)
	x0, y0 := int(math.Floor(float64(g.Min.X))), int(math.Floor(float64(g.Min.Y)))
	x1, y1 := int(math.Ceil(float64(g.Max.X))), int(math.Ceil(float64(g.Max.Y)))
	ink := func(x, y int) bool {
		_, _, _, a := img.At(x, y).RGBA()
		return a != 0
	}
	for y := 0; y < img.Height; y++ {
		inside := y >= y0 && y < y1
		for x := 0; x < img.Width; x++ {
			if inside && x >= x0 && x < x1 {
				x = x1 - 1
				continue
			}
			if ink(x, y) {
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
		for _, scale := range []float32{1, 1.5, 2} {
			lk := p.Look().setScale(scale)
			for _, st := range decorationStates {
				s := DecorationOf(lk, st)
				win := paintengine2d.XYWH(8, 8, float32(math.Round(float64(lk.S(360)))), float32(math.Round(float64(lk.S(200)))))
				inner := s.Border.Apply(win)
				capH := float32(math.Ceil(float64(s.Caption)))
				f := DecorationFrame{Window: win, Caption: paintengine2d.XYWH(inner.Min.X, inner.Min.Y, inner.Dx(), capH)}
				if s.Stacked && st.Custom {
					f.Bar = paintengine2d.XYWH(inner.Min.X, f.Caption.Max.Y, inner.Dx(), lk.S(36))
				}
				img := paintengine2d.NewImage(int(win.Max.X)+8, int(win.Max.Y)+8)
				DrawDecorationOf(lk, paintengine2d.NewContext(img), f, st)
				if x, y, bad := paintedOutside(img, win, 0); bad {
					t.Fatalf("%s @%vx %+v: frame paints (%d,%d) outside the window %v", p.Name, scale, st, x, y, win)
				}
				// The title on a canvas of its own, 8px round its box.
				tb := paintengine2d.XYWH(8, 8, f.Caption.Dx()-lk.S(120), capH)
				img = paintengine2d.NewImage(int(tb.Max.X)+8, int(tb.Max.Y)+8)
				DrawCaptionTitleOf(lk, paintengine2d.NewContext(img), tb, "Window title", st)
				if x, y, bad := paintedOutside(img, tb, 0); bad {
					t.Fatalf("%s @%vx %+v: title paints (%d,%d) outside its box %v", p.Name, scale, st, x, y, tb)
				}
				if st.Custom || st.Tiled != 0 {
					// The buttons do not change with these.
					continue
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

// Every engine paints its era's own frame; only the base look, which is no
// era, falls back to the plain one.
func TestEveryEnginePaintsItsOwnFrame(t *testing.T) {
	for _, p := range ListBuiltinThemes() {
		lk := p.Look()
		if lk.eng().ID() == baseEngine.ID() {
			if NativeDecoration(lk) {
				t.Fatalf("%s: the base engine has no frame of its own", p.Name)
			}
			continue
		}
		if !NativeDecoration(lk) {
			t.Fatalf("%s (engine %s): no DecorationEngine — the frame would be adapted", p.Name, lk.eng().ID())
		}
	}
}

// The adapter is what an engine without a frame of its own would get, and
// nothing in the toolkit reaches it any more — so it is driven directly:
// it takes the in-app window's caption height, borders and close button,
// leaves the frame square and states no layout of its own.
func TestDecorationAdapterUsesTheInAppFrame(t *testing.T) {
	for _, name := range []string{"win31", "os2warp", "next", "keramik", "oxygen"} {
		p := themePack(t, name)
		lk := p.Look()
		in := lk.eng().WindowFrameInsets(lk)
		s := frameAdapter{}.Decoration(lk, DecorationState{Active: true})
		if !s.Stacked || s.Caption != snap(in.Top) || s.Border.Left != snap(in.Left) || s.Border.Bottom != snap(in.Bottom) {
			t.Fatalf("%s: adapted spec %+v from insets %+v", p.Name, s, in)
		}
		if cr := lk.eng().WindowCloseRect(lk, adapterProbe(lk)); !cr.Empty() && s.Button != winSnap(cr).Size() {
			t.Fatalf("%s: buttons %v, the in-app close is %v", p.Name, s.Button, cr.Size())
		}
		if s.Radius != ([4]float32{}) || s.Layout != "" {
			t.Fatalf("%s: the adapter states a look of its own: %+v", p.Name, s)
		}
		// It paints inside the boxes it is given, like any other frame.
		win := paintengine2d.XYWH(8, 8, 360, 200)
		inner := s.Border.Apply(win)
		f := DecorationFrame{Window: win, Caption: paintengine2d.XYWH(inner.Min.X, inner.Min.Y, inner.Dx(), float32(math.Ceil(float64(s.Caption))))}
		img := paintengine2d.NewImage(int(win.Max.X)+8, int(win.Max.Y)+8)
		frameAdapter{}.DrawDecoration(lk, paintengine2d.NewContext(img), f, DecorationState{Active: true})
		if x, y, bad := paintedOutside(img, win, 0); bad {
			t.Fatalf("%s: the adapter paints (%d,%d) outside the window", p.Name, x, y)
		}
	}
}

// An engine's frame paints its caption band: one that drew nothing there
// would leave the window's background showing through as its title bar.
// (The plain frame is the exception by design: its caption *is* the window
// background, under a rule.)
func TestDecorationPaintsItsCaption(t *testing.T) {
	for _, p := range ListBuiltinThemes() {
		lk := p.Look()
		if !NativeDecoration(lk) {
			continue
		}
		s := DecorationOf(lk, DecorationState{Active: true})
		win := paintengine2d.XYWH(0, 0, 360, 200)
		inner := s.Border.Apply(win)
		capH := float32(math.Ceil(float64(s.Caption)))
		f := DecorationFrame{Window: win, Caption: paintengine2d.XYWH(inner.Min.X, inner.Min.Y, inner.Dx(), capH)}
		img := paintengine2d.NewImage(int(win.Max.X), int(win.Max.Y))
		DrawDecorationOf(lk, paintengine2d.NewContext(img), f, DecorationState{Active: true})
		x, y := int(f.Caption.Min.X+f.Caption.Dx()*0.5), int(f.Caption.Min.Y+capH*0.5)
		if _, _, _, a := img.At(x, y).RGBA(); a == 0 {
			t.Fatalf("%s: nothing painted in the caption at (%d,%d)", p.Name, x, y)
		}
	}
}

// Corners and shadows are the era's: a desktop that drew frames straight on
// the screen had neither, the ones that shaped their decorations rounded
// their title bar's top corners, and only a compositing desktop cast a
// shadow.
func TestDecorationCornersAndShadowsPerEra(t *testing.T) {
	type era struct{ rounded, shadow bool }
	eras := map[string]era{
		// Drawn straight on the screen: square, no shadow.
		"system1": {}, "system7": {}, "platinum": {}, "platinum-lime": {},
		"amiga13": {}, "amiga31": {}, "beos": {}, "os2warp": {},
		"win31": {}, "win-hotdog": {}, "openlook": {},
		"next": {}, "openstep": {}, "wmaker-default": {},
		"metal-steel": {}, "metal-ocean": {},
		// Bluecurve is square everywhere, by Red Hat's design.
		"bluecurve": {},
		// Shaped decorations, still no compositor.
		"keramik": {rounded: true}, "plastik": {rounded: true}, "plastique": {rounded: true},
		"clearlooks": {rounded: true}, "cleanlooks": {rounded: true}, "human": {rounded: true},
		// Composited desktops.
		"oxygen": {rounded: true, shadow: true},
		"fusion": {rounded: true, shadow: true}, "fusion-night": {rounded: true, shadow: true},
		"nimbus": {rounded: true, shadow: true},
	}
	for name, want := range eras {
		s := DecorationOf(themePack(t, name).Look(), DecorationState{Active: true})
		rounded := s.Radius[0] > 0 && s.Radius[1] > 0
		if rounded != want.rounded || s.Radius[2] != 0 || s.Radius[3] != 0 {
			t.Errorf("%s: corners %v, wanted rounded=%v on the top two only", name, s.Radius, want.rounded)
		}
		if got := s.Shadow != (Insets{}); got != want.shadow {
			t.Errorf("%s: shadow %+v, wanted %v", name, s.Shadow, want.shadow)
		}
	}
}

// A look's own button layout is the one its desktop had, on the side it had
// it: the Mac, the Amiga and BeOS close at the left, NeXT miniaturizes at
// the left and closes at the right, Windows 3.1 and OS/2 open the window
// menu at the left, OPEN LOOK has nothing but that menu.
func TestDecorationThemeLayouts(t *testing.T) {
	want := map[string]string{
		"system7": "close:maximize", "platinum": "close:minimize,maximize",
		"amiga31": "close:maximize", "beos": "close:maximize",
		"next": "minimize:close", "wmaker-default": "minimize:close",
		"win31": "icon:minimize,maximize", "os2warp": "icon:close,minimize,maximize",
		"openlook": "icon:", "keramik": "icon:minimize,maximize,close",
		"clearlooks": "icon:minimize,maximize,close", "metal-ocean": "icon:minimize,maximize,close",
		"nimbus": ":minimize,maximize,close", "fusion": ":minimize,maximize,close",
	}
	for name, layout := range want {
		if got := DecorationOf(themePack(t, name).Look(), DecorationState{Active: true}).Layout; got != layout {
			t.Errorf("%s: layout %q, wanted %q", name, got, layout)
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
