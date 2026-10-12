//go:build theme_engine_all || theme_engine_hangar

package style

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func hangarLook(t *testing.T) *Classic {
	t.Helper()
	p, ok := LoadTheme("hangar")
	if !ok {
		t.Skip("the hangar pack is not in this build")
	}
	lk := p.Look()
	if lk == nil || lk.Engine().ID() != "hangar" {
		t.Fatalf("the pack paints with %v", lk.Engine().ID())
	}
	return lk
}

// shot draws into a small image over the pack's own ground.
func hangarShot(t *testing.T, lk *Classic, w, h int, draw func(*paintengine2d.Context, paintengine2d.Rect)) *paintengine2d.Image {
	t.Helper()
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	b := paintengine2d.XYWH(0, 0, float32(w), float32(h))
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Background))
	draw(ctx, b)
	img.Touch()
	return img
}

// near reports whether two colours are within tol on every channel.
func hangarNear(a, b paintengine2d.Color, tol float32) bool {
	d := func(x, y float32) bool { return x-y < tol && y-x < tol }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B)
}

// countNear is how many pixels of img are within tol of col.
func hangarCount(img *paintengine2d.Image, col paintengine2d.Color, tol float32) int {
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			c := img.NRGBAAt(x, y)
			if hangarNear(paintengine2d.RGBA(float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, 1), col, tol) {
				n++
			}
		}
	}
	return n
}

// **The ember is spent on what is on, never on a hover.** It is the one
// rule the whole look hangs from: a console where the hot colour means
// "the pointer is here" is one where it no longer means "this is on".
func TestHangarKeepsItsAccentForWhatIsOn(t *testing.T) {
	lk := hangarLook(t)
	accent := lk.Palette().Accent
	ember := func(st ControlState, d ButtonDraw) int {
		img := hangarShot(t, lk, 120, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawButton(ctx, b, st, d)
		})
		return hangarCount(img, accent, 0.06)
	}
	plain := ember(StateNone, ButtonDraw{Label: "Run"})
	hover := ember(StateHovered, ButtonDraw{Label: "Run"})
	press := ember(StatePressed, ButtonDraw{Label: "Run"})
	primary := ember(StatePrimary, ButtonDraw{Label: "Run"})

	if plain > 20 || hover > 20 || press > 20 {
		t.Errorf("an ordinary button wears the accent: plain %d, hover %d, pressed %d", plain, hover, press)
	}
	if primary < 1000 {
		t.Errorf("the default button is only %d pixels of ember; it is the accent's own button", primary)
	}
	// A hover is still visible — it is a wash of white, not nothing.
	lit := hangarShot(t, lk, 120, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawButton(ctx, b, StateHovered, ButtonDraw{Label: "Run"})
	})
	rest := hangarShot(t, lk, 120, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawButton(ctx, b, StateNone, ButtonDraw{Label: "Run"})
	})
	if diffPixels(lit, rest) < 200 {
		t.Error("a hover changed almost nothing: the wash is missing")
	}
}

// Chrome labels are painted in capitals and the application's words are
// untouched — the same rule a skin's "case": "upper" follows.
func TestHangarPaintsChromeLabelsInCapitals(t *testing.T) {
	lk := hangarLook(t)
	for _, role := range []Role{RoleButton, RoleTool, RoleTab, RoleCombo} {
		if got := lk.ControlLabel(role, "Run test"); got != "RUN TEST" {
			t.Errorf("role %v paints %q", role, got)
		}
	}
	// What a control shows of the person's own content is theirs.
	for _, role := range []Role{RoleField, RoleRow, RoleMenu} {
		if got := lk.ControlLabel(role, "Run test"); got != "Run test" {
			t.Errorf("role %v paints %q, want the words as written", role, got)
		}
	}
	// And a pack can turn it off.
	p, _ := LoadTheme("hangar")
	p.Tokens.Params = map[string]float32{"caps": 0}
	plain := p.Look()
	if got := plain.ControlLabel(RoleButton, "Run test"); got != "Run test" {
		t.Errorf("with caps off a button paints %q", got)
	}
}

// Everything is square and one pixel thick: no radius, no bevel.
func TestHangarIsSquareAndHairline(t *testing.T) {
	lk := hangarLook(t)
	m := lk.Metrics()
	if m.Radius != 0 || m.RadiusSmall != 0 {
		t.Errorf("radii %g / %g, want square", m.Radius, m.RadiusSmall)
	}
	p, _ := LoadTheme("hangar")
	if p.Tokens.Bevel != BevelNone {
		t.Errorf("bevel %q", p.Tokens.Bevel)
	}
	if !p.Tokens.Metrics.Square && !lk.Engine().DefaultMetrics().Square {
		t.Error("the look does not call itself square, so the Corners pref can round it")
	}
}

// A meter is segmented: blocks with gaps, not one bar. The fill is the
// accent and the gaps are not, so a half-full meter has both.
func TestHangarMetersAreSegmented(t *testing.T) {
	lk := hangarLook(t)
	img := hangarShot(t, lk, 120, 10, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawProgressBar(ctx, b, StateNone, 1, false, 0)
	})
	// Along the middle row the ember must start and stop more than once.
	runs, was := 0, false
	y := img.Height / 2
	for x := 0; x < img.Width; x++ {
		c := img.NRGBAAt(x, y)
		on := hangarNear(paintengine2d.RGBA(float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, 1), lk.Palette().Accent, 0.08)
		if on && !was {
			runs++
		}
		was = on
	}
	if runs < 3 {
		t.Errorf("a full meter is %d blocks of ember, want it segmented", runs)
	}
}

// The window ground is a gradient: the top-left corner is the gunmetal
// and the bottom-right is warmer, and the whole of it depends on the
// window's box alone, so a partial repaint paints the same pixels.
func TestHangarGroundIsAGradientOfTheWindow(t *testing.T) {
	lk := hangarLook(t)
	const w, h = 160, 100
	whole := hangarShot(t, lk, w, h, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawWindowBackground(ctx, b)
	})
	tl := whole.NRGBAAt(2, 2)
	br := whole.NRGBAAt(w-3, h-3)
	if !(br.R > tl.R+6 && br.B <= tl.B+2) {
		t.Errorf("the ground runs from %v to %v; want it warming towards the far corner", tl, br)
	}
	// The same call under a clip paints the same pixels: that is what
	// "depends on the window's box alone" means in practice.
	clipped := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(clipped)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(lk.Palette().Background))
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(40, 20, 50, 40))
	lk.DrawWindowBackground(ctx, paintengine2d.XYWH(0, 0, w, h))
	ctx.Restore()
	clipped.Touch()
	for y := 21; y < 59; y++ {
		for x := 41; x < 89; x++ {
			if whole.NRGBAAt(x, y) != clipped.NRGBAAt(x, y) {
				t.Fatalf("a clipped repaint differs at %d,%d: %v against %v",
					x, y, clipped.NRGBAAt(x, y), whole.NRGBAAt(x, y))
			}
		}
	}
}

// The close button is the ember block — the one button on the panel you
// are not meant to press by accident, and the only one that is marked.
func TestHangarMarksItsCloseButton(t *testing.T) {
	lk := hangarLook(t)
	st := DecorationState{Active: true}
	ember := func(k CaptionButton) int {
		img := hangarShot(t, lk, 30, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			DrawCaptionButtonOf(lk, ctx, b, k, StateNone, st)
		})
		return hangarCount(img, lk.Palette().Accent, 0.06)
	}
	if n := ember(CaptionClose); n < 400 {
		t.Errorf("the close button is %d pixels of ember, want the block", n)
	}
	if n := ember(CaptionMinimize); n > 20 {
		t.Errorf("the minimize button wears %d pixels of ember", n)
	}
}

// The pack says what it is, the way every pack in here has to.
func TestHangarPackDescribesItself(t *testing.T) {
	p, ok := LoadTheme("hangar")
	if !ok {
		t.Skip("the hangar pack is not in this build")
	}
	if p.Label == "" || p.Lineage == "" || p.Era == "" || p.Year == 0 {
		t.Errorf("pack %+v", p)
	}
	if s := strings.TrimSpace(p.Summary); s == "" || strings.Contains(s, "\n") {
		t.Errorf("summary %q must be one line", p.Summary)
	}
	if p.Palette != ThemeDark {
		t.Errorf("family %v, want dark", p.Palette)
	}
	if len(p.Tokens.Fonts.UI) == 0 || len(p.Tokens.Fonts.Mono) == 0 {
		t.Error("the pack names no typefaces, so the genre's face is left to the machine")
	}
}

// **Three corners are cut and the fourth is not.** The cut is the
// genre's most recognisable mark, and the top left is kept square
// because that is where a panel's name and its ember tick go — a window
// whose title started in a diagonal would read as a mistake.
func TestHangarCutsThreeCornersAndKeepsTheFourth(t *testing.T) {
	lk := hangarLook(t)
	const w, h = 120, 80
	img := hangarShot(t, lk, w, h, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawPanel(ctx, b, false)
	})
	ground := lk.Palette().Background
	corner := func(x, y int) bool { // is this corner still the ground?
		c := img.NRGBAAt(x, y)
		return hangarNear(paintengine2d.RGBA(float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, 1), ground, 0.02)
	}
	if corner(2, 2) {
		t.Error("the top left is cut; it is the corner the eye starts from")
	}
	for _, c := range []struct {
		name string
		x, y int
	}{{"top right", w - 3, 2}, {"bottom right", w - 3, h - 3}, {"bottom left", 2, h - 3}} {
		if !corner(c.x, c.y) {
			t.Errorf("the %s is not cut", c.name)
		}
	}
	// The chamfer is one length for everything big enough to take it,
	// and a box that cannot spare it loses less rather than coming out a
	// hexagon — a tiny one loses nothing at all.
	full := hangarChamfer(lk)
	if got := hangarCut(lk, paintengine2d.XYWH(0, 0, 400, 300)); got != full {
		t.Errorf("a panel is cut by %g, want the look's %g", got, full)
	}
	if got := hangarCut(lk, paintengine2d.XYWH(0, 0, 18, 14)); got <= 0 || got >= full {
		t.Errorf("an 18x14 box is cut by %g, want less than the %g a panel gets", got, full)
	}
	if got := hangarCut(lk, paintengine2d.XYWH(0, 0, 9, 9)); got != 0 {
		t.Errorf("a 9x9 box is cut by %g, want left square", got)
	}
}

// The window states that outline as its silhouette, so the desktop shows
// through the cut rather than the look painting a triangle of its own
// background there.
func TestHangarWindowSaysItsOutline(t *testing.T) {
	lk := hangarLook(t)
	win := paintengine2d.XYWH(0, 0, 800, 600)
	f := DecorationFrame{Window: win, Caption: paintengine2d.XYWH(0, 0, 800, 30)}
	s := WindowShapeOf(lk, f, DecorationState{Active: true})
	if s.Empty() {
		t.Fatal("the look declares no silhouette, so its corners are painted and not cut")
	}
	if s.Path == nil {
		t.Error("the silhouette is a mask; a chamfer should be a path, exact at every scale")
	}
	// A window too small for the cut is a rectangle rather than a lozenge.
	tiny := paintengine2d.XYWH(0, 0, 20, 16)
	if got := WindowShapeOf(lk, DecorationFrame{Window: tiny}, DecorationState{Active: true}); !got.Empty() {
		t.Error("a 20x16 window was given a silhouette")
	}
}

// **The close button is cut with the window, by the same diagonal.** It
// sits in the corner the frame cuts; a button that worked its chamfer out
// from its own box took nine pixels where the frame took twelve, and the
// sliver of caption band between the two read as a smudge.
func TestHangarCloseButtonSharesTheFramesChamfer(t *testing.T) {
	lk := hangarLook(t)
	spec := DecorationOf(lk, DecorationState{Active: true})
	if spec.Button.Y != spec.Caption {
		t.Errorf("a caption button is %g tall in a %g caption; it should be the whole height",
			spec.Button.Y, spec.Caption)
	}
	if spec.ButtonPad != (Insets{}) || spec.ButtonGap != 0 || spec.CloseGap != 0 {
		t.Errorf("the buttons are padded (%v, gap %g, close gap %g); they are cut out of the band",
			spec.ButtonPad, spec.ButtonGap, spec.CloseGap)
	}
	// At 2x as well as at 1x: the two candidate rules — the look's one
	// chamfer, or one worked out from the button's own box — differ by a
	// pixel at 1x and by three at 2x, so a single scale cannot tell them
	// apart and a test at one scale would pass either way.
	for _, scale := range []float32{1, 2} {
		hangarCloseCut(t, WithScale(lk, scale).(*Classic), scale)
	}
}

func hangarCloseCut(t *testing.T, lk *Classic, scale float32) {
	t.Helper()
	spec := DecorationOf(lk, DecorationState{Active: true})
	box := paintengine2d.XYWH(0, 0, spec.Button.X, spec.Button.Y)
	img := hangarShot(t, lk, int(box.Dx()), int(box.Dy()), func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		DrawCaptionButtonOf(lk, ctx, b, CaptionClose, StateNone, DecorationState{Active: true})
	})
	cut := hangarChamfer(lk)
	ember := func(x, y int) bool {
		c := img.NRGBAAt(x, y)
		return hangarNear(paintengine2d.RGBA(float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, 1), lk.Palette().Accent, 0.06)
	}
	if ember(int(box.Dx())-2, 2) {
		t.Error("the close button's top right corner is square")
	}
	// Measure the diagonal rather than probe it: on the row y, the cut
	// edge stands at x = width - chamfer + y, so the first ember pixel
	// scanning in from the right says how long the chamfer is. A button
	// that worked its own out of its own box would answer ten where the
	// frame answers twelve, and the test would never know from one probe.
	const y = 3
	got := float32(-1)
	for x := int(box.Dx()) - 1; x >= 0; x-- {
		if ember(x, y) {
			got = box.Dx() - float32(x) + y - 1
			break
		}
	}
	if got < cut-1.5 || got > cut+1.5 {
		t.Errorf("at %gx the close button's chamfer measures %g, want the frame's %g", scale, got, cut)
	}
}

// **The surfaces are light enough to read.** The first draft of this
// pack was about half the luminance of the thing it copies all the way
// down — dark enough that the room can be dark is not the same as dark
// enough that nothing can be seen — so the ramp is pinned here against
// the colours a frame of the console actually quantises to.
func TestHangarSurfacesAreTheConsolesOwn(t *testing.T) {
	lk := hangarLook(t)
	p := lk.Palette()
	lum := func(c paintengine2d.Color) float32 { return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B }
	// The ramp, darkest first: a well is sunk into a panel, a panel sits
	// on the window, a band is raised off it, and the hairline is
	// lighter than all of them.
	for _, step := range []struct {
		name string
		a, b paintengine2d.Color
	}{
		{"the field is darker than the window", p.Field, p.Background},
		{"the window is darker than a panel", p.Background, p.Surface},
		{"a panel is darker than a raised face", p.Surface, p.SurfaceAlt},
		{"a raised face is darker than the hairline", p.SurfaceAlt, p.Border},
	} {
		if lum(step.a) >= lum(step.b) {
			t.Errorf("%s: %.3f against %.3f", step.name, lum(step.a), lum(step.b))
		}
	}
	// And the panel is in the measured range rather than half of it: the
	// console's body quantises to #2b3a49, luminance about 0.215.
	if got := lum(p.Surface); got < 0.17 || got > 0.27 {
		t.Errorf("the panel's luminance is %.3f, want the console's ~0.215", got)
	}
	// Light ink on the ember, not dark: a dark mark on that orange reads
	// as a hole punched in the block.
	if lum(p.TextOnAccent) < 0.8 {
		t.Errorf("the ink on the accent is %.3f, want the console's near-white", lum(p.TextOnAccent))
	}
}

// The bracket hairlines are brighter than the frame's own line, because
// they are the mark that says "panel" and a mark nobody can see is not
// one.
func TestHangarBracketsAreBrighterThanTheFrame(t *testing.T) {
	lk := hangarLook(t)
	c := hangarColorsOf(lk)
	lum := func(x paintengine2d.Color) float32 { return 0.2126*x.R + 0.7152*x.G + 0.0722*x.B }
	if lum(c.bracket) <= lum(c.line) {
		t.Errorf("the brackets (%.3f) are no brighter than the frame's line (%.3f)", lum(c.bracket), lum(c.line))
	}
	// They are drawn, and inside the box rather than on its edge.
	const w, h = 160, 110
	plain := hangarShot(t, lk, w, h, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		hangarBox(lk, ctx, b, lk.Palette().Surface, lk.Palette().Border)
	})
	marked := hangarShot(t, lk, w, h, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		hangarBox(lk, ctx, b, lk.Palette().Surface, lk.Palette().Border)
		hangarBrackets(lk, ctx, b, hangarCut(lk, b), c.bracket)
	})
	if n := diffPixels(plain, marked); n < 150 {
		t.Errorf("the brackets changed %d pixels; they are meant to be seen", n)
	}
}

// **One ember mark, at the head of a caption, and nowhere else.** The
// first draft dashed every panel, tool bar and tip along its top edge,
// which read as a scratch above the title and a stray line over the
// first menu item — and is not in the thing being copied.
func TestHangarMarksCaptionsAndNothingElse(t *testing.T) {
	lk := hangarLook(t)
	accent := lk.Palette().Accent
	const w, h = 200, 90
	ember := func(draw func(*paintengine2d.Context, paintengine2d.Rect)) int {
		return hangarCount(hangarShot(t, lk, w, h, draw), accent, 0.06)
	}
	if n := ember(func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawTitleBar(ctx, paintengine2d.XYWH(0, 0, w, 30), "Report", "")
	}); n < 30 {
		t.Errorf("a caption has %d pixels of ember; it should wear the mark", n)
	}
	for _, tc := range []struct {
		name string
		draw func(*paintengine2d.Context, paintengine2d.Rect)
	}{
		{"a panel", func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawPanel(ctx, b, false) }},
		{"a tool bar", func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawToolBar(ctx, paintengine2d.XYWH(0, 0, w, 30))
		}},
		{"a menu bar", func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawMenuBar(ctx, paintengine2d.XYWH(0, 0, w, 30))
		}},
		{"a tip", func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawTooltip(ctx, paintengine2d.XYWH(0, 0, w, 40), "a tip")
		}},
	} {
		if n := ember(tc.draw); n > 10 {
			t.Errorf("%s wears %d pixels of ember; the mark belongs to a caption alone", tc.name, n)
		}
	}
}
