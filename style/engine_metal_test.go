package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var metalPackNames = []string{"metal-steel", "metal-ocean"}

func TestMetalPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	want := map[string]struct {
		label string
		year  int
	}{
		"metal-steel": {"Metal (Steel)", 1998},
		"metal-ocean": {"Metal (Ocean)", 2004},
	}
	for _, n := range metalPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		w := want[n]
		if p.Tokens.Engine != "metal" || p.Label != w.label || p.Lineage != "Java" || p.Year != w.year {
			t.Fatalf("%s: engine %q label %q lineage %q year %d", n, p.Tokens.Engine, p.Label, p.Lineage, p.Year)
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if p.Tokens.Family != ThemeLight {
			t.Fatalf("%s family %q", n, p.Tokens.Family)
		}
		if lk := p.Look(); lk.Engine().ID() != "metal" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
	for _, o := range [][2]string{{"win95", "metal-steel"}, {"metal-steel", "luna"}, {"luna", "metal-ocean"}} {
		if pos[o[0]] > pos[o[1]] {
			t.Fatalf("packs not ordered by year: %s after %s", o[0], o[1])
		}
	}
}

// The themes carry the documented Metal colours: DefaultMetalTheme's
// eight (as the design guidelines tabulate them) and Ocean's primaries and
// secondaries.
func TestMetalThemeColours(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for n, want := range map[string][8]string{
		"metal-steel": {"#666699", "#9999cc", "#ccccff", "#666666", "#999999", "#cccccc", "#000000", "#ffffff"},
		"metal-ocean": {"#6382bf", "#a3b8cc", "#b8cfe5", "#7a8a99", "#b8cfe5", "#eeeeee", "#333333", "#ffffff"},
	} {
		c := mtlColors(mustLook(t, n))
		got := [8]string{colorHexPadded(c.p1), colorHexPadded(c.p2), colorHexPadded(c.p3),
			colorHexPadded(c.s1), colorHexPadded(c.s2), colorHexPadded(c.s3), colorHexPadded(c.black), colorHexPadded(c.white)}
		if got != want {
			t.Errorf("%s theme colours %v, want %v", n, got, want)
		}
		p := mustLook(t, n).Palette()
		if !nearColor(p.Background, c.s3) || !nearColor(p.Selection, c.p3) || !nearColor(p.Focus, c.p2) || !nearColor(p.Accent, c.p1) {
			t.Errorf("%s palette drifts from the theme: bg %s sel %s focus %s accent %s", n,
				colorHexPadded(p.Background), colorHexPadded(p.Selection), colorHexPadded(p.Focus), colorHexPadded(p.Accent))
		}
	}
	if c := mtlColors(mustLook(t, "metal-ocean")); !c.ocean || mtlColors(mustLook(t, "metal-steel")).ocean {
		t.Fatal("only Ocean paints the Ocean faces")
	}
}

func TestMetalPaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metalPackNames {
		p, _ := LoadTheme(n)
		for _, scale := range []float32{1, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), lk)
			mtlRowsInside(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}

// Every Draw* survives degenerate rects and every state.
func TestMetalPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metalPackNames {
		for _, sc := range []float32{1, 2} {
			lk := lunaLook(t, n, sc)
			img := paintengine2d.NewImage(int(420*sc), int(200*sc))
			ctx := paintengine2d.NewContext(img)
			for _, st := range lunaStates {
				for _, r := range []paintengine2d.Rect{
					paintengine2d.XYWH(4*sc, 4*sc, 120*sc, 28*sc),
					paintengine2d.XYWH(4*sc, 4*sc, 17*sc, 100*sc),
					paintengine2d.XYWH(4.5*sc, 3.25*sc, 96.5*sc, 22.75*sc),
					{}, paintengine2d.XYWH(0, 0, 1, 1), paintengine2d.XYWH(2, 2, 3, 3),
					paintengine2d.XYWH(0, 0, 5, 40), paintengine2d.XYWH(0, 0, 40, 5),
				} {
					lunaCalls(lk, ctx, r, st)
				}
			}
			lunaStateless(lk, ctx, paintengine2d.XYWH(4*sc, 4*sc, 400*sc, 180*sc))
			for _, r := range []paintengine2d.Rect{{}, paintengine2d.XYWH(0, 0, 1, 1), paintengine2d.XYWH(0, 0, 40, 5)} {
				lunaStateless(lk, ctx, r)
			}
			if ctx.SaveCount() != 0 {
				t.Fatalf("%s@%gx left %d saved states", n, sc, ctx.SaveCount())
			}
		}
	}
}

// mtlRowsInside paints the item rows in their view states (current,
// unfocused view, odd row, tree chains, table cell spans) and checks they
// stay in their rect.
func mtlRowsInside(t *testing.T, name string, lk *Classic) {
	t.Helper()
	sc := lk.Scale()
	states := []ControlState{
		StateFocused, StateChecked | StateFocused, StateChecked | StateInactive, StateChecked | StateBackdrop,
		StateAlternate, StateAlternate | StateChecked, StateExpanderHot | StateHovered,
		TreeChain(0b101) | StateFocused, TreeChain(0) | StateChecked,
		StateFirst | StateChecked, StateLast | StateChecked, StateChecked,
	}
	for _, st := range states {
		img := paintengine2d.NewImage(int(260*sc), int(60*sc))
		ctx := paintengine2d.NewContext(img)
		r := paintengine2d.XYWH(20*sc, 20*sc, 200*sc, 24*sc)
		lk.DrawListRow(ctx, r, st, "List row")
		lk.DrawTreeRow(ctx, r, st, st.Checked(), !st.Checked(), 2, "Tree row", false)
		lk.DrawTableCell(ctx, r, st, "Cell", AlignCenter, nil)
		lk.DrawItemFocus(ctx, r, st)
		lk.DrawViewFrame(ctx, r, st)
		if x, y, ok := aquaOutside(img, r); ok {
			t.Errorf("%s: row state %#x painted outside at (%d,%d)", name, uint32(st), x, y)
		}
	}
}

// The flush 3D border: a dark ring with white inside its top and left and
// outside its right and bottom, left open at the top-right and bottom-left
// corners; a default button's ring is two pixels thick.
func TestMetalFlushBorder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "metal-steel")
	c := mtlColors(lk)
	const w, h = 60, 24
	for _, primary := range []bool{false, true} {
		img := paintengine2d.NewImage(w+8, h+8)
		ctx := paintengine2d.NewContext(img)
		st := StateNone
		if primary {
			st = StatePrimary
		}
		lk.DrawButton(ctx, paintengine2d.XYWH(4, 4, w, h), st, "")
		px := func(x, y int, want paintengine2d.Color, what string) {
			t.Helper()
			if !nxNear(img, 4+x, 4+y, want) {
				r, g, b, _ := img.PremulAt(4+x, 4+y)
				t.Fatalf("primary=%v: %s at (%d,%d) is rgb(%d,%d,%d), want %s", primary, what, x, y, r, g, b, colorHexPadded(want))
			}
		}
		px(0, 0, c.s1, "outer ring")
		px(w-2, h/2, c.s1, "right edge")
		px(w/2, h-2, c.s1, "bottom edge")
		px(w-1, h/2, c.white, "outer highlight right")
		px(w/2, h-1, c.white, "outer highlight bottom")
		px(w/2, h/2, c.s3, "face")
		if primary {
			px(1, 1, c.s1, "second ring")
			px(2, h/2, c.white, "inner highlight")
			px(w-3, 2, c.s3, "open corner")
		} else {
			px(1, h/2, c.white, "inner highlight left")
			px(w/2, 1, c.white, "inner highlight top")
			px(w-2, 1, c.s3, "open top-right corner")
			px(1, h-2, c.s3, "open bottom-left corner")
		}
	}
	// Pressed: secondary 2 without the inner highlight.
	img := paintengine2d.NewImage(w+8, h+8)
	lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(4, 4, w, h), StatePressed|StateHovered, "")
	if !nxNear(img, 4+w/2, 4+h/2, c.s2) || !nxNear(img, 4+1, 4+h/2, c.s2) {
		t.Fatal("a pressed Metal button is secondary 2 up to its dark ring")
	}
}

// The bumps: a light dot every four pixels with a dark dot one down and to
// the right, rows two pixels apart and staggered — at 2x each dot is 2×2.
func TestMetalBumpsTexture(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, scale := range []float32{1, 2} {
		lk := WithScale(mustLook(t, "metal-steel"), scale).(*Classic)
		c := mtlColors(lk)
		u := int(c.u)
		img := paintengine2d.NewImage(64*u, 64*u)
		ctx := paintengine2d.NewContext(img)
		bg := Hex("#123456")
		ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(64*u), float32(64*u)), paintengine2d.Fill(bg))
		o := 8 * u
		c.bumps(ctx, paintengine2d.XYWH(float32(o), float32(o), float32(32*u), float32(32*u)), c.gripBumps)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				want := bg
				switch {
				case x%4 == 0 && y%4 == 0, x%4 == 2 && y%4 == 2:
					want = c.white
				case x%4 == 1 && y%4 == 1, x%4 == 3 && y%4 == 3:
					want = c.s1
				}
				for dy := 0; dy < u; dy++ {
					for dx := 0; dx < u; dx++ {
						if !nxNear(img, o+x*u+dx, o+y*u+dy, want) {
							r, g, b, _ := img.PremulAt(o+x*u+dx, o+y*u+dy)
							t.Fatalf("@%gx bump (%d,%d) is rgb(%d,%d,%d), want %s", scale, x, y, r, g, b, colorHexPadded(want))
						}
					}
				}
			}
		}
	}
}

// Ocean's button wash: #dde8f3 at the top edge, white about a third of
// the way down, back to #dde8f3 by sixty percent, #b8cfe5 at the bottom.
func TestMetalOceanWash(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "metal-ocean")
	c := mtlColors(lk)
	if len(c.btnGrad) != 4 || c.btnGrad[1].Offset != 0.3 || c.btnGrad[2].Offset != 0.6 {
		t.Fatalf("Ocean wash stops %+v", c.btnGrad)
	}
	const h = 40
	img := paintengine2d.NewImage(40, h+4)
	lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 2, 40, h), StateNone, "")
	at := func(y int) paintengine2d.Color {
		r, g, b, _ := img.PremulAt(20, 2+y)
		return paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255)
	}
	if top, peak, low := at(2), at(12), at(h-3); !(Luma(peak) > Luma(top) && Luma(top) > Luma(low)) || Luma(peak) < 0.97 {
		t.Fatalf("Ocean wash top %s peak %s bottom %s", colorHexPadded(top), colorHexPadded(peak), colorHexPadded(low))
	}
	if !nxNear(img, 20, 2, c.s1) || !nxNear(img, 0, 20, c.s1) {
		t.Fatal("an Ocean button is ringed in secondary 1")
	}
}

// The close button is at the right of the title bar and hit-testing
// agrees with the paint: its × crosses at the centre, the button's face
// lies inside its flush ring.
func TestMetalCloseButtonAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metalPackNames {
		for _, scale := range []float32{1, 2} {
			lk := WithScale(mustLook(t, n), scale).(*Classic)
			c := mtlColors(lk)
			b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Max.X > b.Max.X || cr.Min.X < b.Max.X-b.Dx()*0.2 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close rect %v not at the right of the title bar of %v (top inset %v)", n, scale, cr, b, in.Top)
			}
			if cr.Dx() != cr.Dy() {
				t.Fatalf("%s@%gx: close button %vx%v is not square", n, scale, cr.Dx(), cr.Dy())
			}
			img := paintengine2d.NewImage(int(b.Max.X+4*scale), int(b.Max.Y+4*scale))
			ctx := paintengine2d.NewContext(img)
			lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
			g := paintengine2d.XYWH(cr.Min.X, cr.Min.Y, cr.Dx()-c.u, cr.Dy()-c.u).Center()
			if !nxNear(img, int(g.X), int(g.Y), c.black) {
				r, gg, bl, _ := img.PremulAt(int(g.X), int(g.Y))
				t.Fatalf("%s@%gx: close centre rgb(%d,%d,%d), want the × in %s", n, scale, r, gg, bl, colorHexPadded(c.black))
			}
			face := paintengine2d.Pt(cr.Min.X+2.5*c.u, cr.Max.Y-3.5*c.u)
			if nxNear(img, int(face.X), int(face.Y), c.black) || nxNear(img, int(face.X), int(face.Y), c.p1) {
				t.Fatalf("%s@%gx: the close button has no face inside its ring", n, scale)
			}
			if r := lk.WindowCloseRect(paintengine2d.XYWH(0, 0, 20, 20)); !r.Empty() {
				t.Fatalf("%s@%gx: a frame too small for a title bar reports close rect %v", n, scale, r)
			}
		}
	}
}

func TestMetalStyleHintsAndHooks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metalPackNames {
		lk := mustLook(t, n)
		if LookHint(lk, HintDialogPrimaryFirst) != 1 {
			t.Fatalf("%s: Swing's option panes put the default button first", n)
		}
		for _, h := range []StyleHint{HintTabsCentered, HintFormLabelsRight, HintHoverFadeMs, HintDefaultPulseMs} {
			if LookHint(lk, h) != 0 {
				t.Fatalf("%s: hint %d = %d, want 0", n, h, LookHint(lk, h))
			}
		}
		s := ScrollBarStyleOf(lk)
		if s.Arrows != ArrowsEnds || s.Overlay || s.Transient || s.Thickness != 17 {
			t.Fatalf("%s scroll bar %+v: want a 17px gutter with an arrow at each end", n, s)
		}
		if lk.eng().FieldFocusRing(lk) {
			t.Fatalf("%s: Metal fields show focus with the caret alone", n)
		}
		for _, k := range []PopupKind{PopupMenu, PopupTooltip, PopupDialog} {
			if !lk.PopupShadow(k).Zero() {
				t.Fatalf("%s: Metal popups have no shadow (kind %d)", n, k)
			}
		}
		if lk.ControlFont(RoleButton) != lk.BoldFont() || lk.ControlFont(RoleMenu) != lk.BoldFont() || lk.ControlFont(RoleTab) != lk.BoldFont() {
			t.Fatalf("%s: buttons, menus and tabs are labelled in bold", n)
		}
		if lk.ControlFont(RoleField) != lk.Font() || lk.ControlFont(RoleRow) != lk.Font() {
			t.Fatalf("%s: text and rows read in the plain font", n)
		}
		if in := lk.ToolBarInsets(); in.Left < 12 {
			t.Fatalf("%s: tool bars keep room for the bumps grip, got %+v", n, in)
		}
		if lk.TabOverlap() != 1 || !lk.TabOutset().Zero() {
			t.Fatalf("%s: tabs share a one-pixel border and keep their slots", n)
		}
		if in := lk.ViewFrameInsets(); in.Top != 1 || in.Left != 1 || in.Right != 2 || in.Bottom != 2 {
			t.Fatalf("%s: view frame %+v, want the flush border's 1/2 pixels", n, in)
		}
	}
	// swing.boldMetal=false: the pack param turns the bold labels off.
	p, _ := LoadTheme("metal-steel")
	tok := p.Tokens
	tok.Params = map[string]float32{"bold": 0}
	if lk := newClassic("metal-steel", tok.Palette, DefaultMetrics(), CornersSquare, IconSetClassic, IconSizeMedium, tok); lk.ControlFont(RoleButton) != lk.Font() {
		t.Fatal("bold=0 labels controls in the plain font")
	}
}

// Focus is a thin primary 2 rectangle round the label.
func TestMetalFocusRectangle(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metalPackNames {
		lk := mustLook(t, n)
		c := mtlColors(lk)
		img := paintengine2d.NewImage(140, 40)
		lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(4, 4, 120, 28), StateFocused, "Save")
		found := 0
		for y := 0; y < 40; y++ {
			for x := 0; x < 140; x++ {
				if nxNear(img, x, y, c.p2) {
					found++
				}
			}
		}
		if found < 60 {
			t.Fatalf("%s: a focused button shows %d primary 2 focus pixels", n, found)
		}
		if nxNear(img, 64, 8, c.p2) {
			t.Fatalf("%s: the focus rectangle hugs the label, not the whole face", n)
		}
	}
}

// Labels read on the fills they are painted on.
func TestMetalLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metalPackNames {
		lk := mustLook(t, n)
		c := mtlColors(lk)
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on the canvas", c.text, c.s3, 4.5},
			{"text on a field", c.fieldText, c.field, 4.5},
			{"selected text", c.selText, c.sel, 4.5},
			{"hot menu text", c.hotText, c.menuHot, 4.5},
			{"tool tip", c.infoText, c.info, 4.5},
			{"unselected tab", c.text, c.tabIdle, 4.5},
			{"selected tab", c.text, c.tabSel, 4.5},
			{"pressed button", c.text, c.pressed, 4.5},
			{"accelerator", c.accel, c.s3, 3},
			{"hot accelerator", c.accelHot, c.menuHot, 3},
			{"titled border", c.groupText, c.s3, 3},
			{"disabled tab", c.tabDis, c.tabIdle, 1.6},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// Metal's tabs cut their top-left corner on the diagonal and stand on the
// strip; the selected one opens the page's edge.
func TestMetalTabSlant(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metalPackNames {
		lk := mustLook(t, n)
		c := mtlColors(lk)
		img := paintengine2d.NewImage(140, 40)
		ctx := paintengine2d.NewContext(img)
		bar := paintengine2d.XYWH(0, 0, 140, 30)
		lk.DrawTabBar(ctx, bar)
		lk.DrawTab(ctx, paintengine2d.XYWH(10, 0, 100, 30), StateNone, "Tab", false)
		if !nxNear(img, 11, 3, c.s3) {
			t.Fatalf("%s: the tab's top-left corner is not cut away", n)
		}
		if !nxNear(img, 60, 1, c.tabEdgeIdle) || !nxNear(img, 10, 20, c.tabEdgeIdle) || !nxNear(img, 109, 20, c.tabEdgeIdle) {
			t.Fatalf("%s: the tab is not outlined along its top and sides", n)
		}
		if !nxNear(img, 60, 15, c.tabIdle) {
			t.Fatalf("%s: an unselected tab is filled with %s", n, colorHexPadded(c.tabIdle))
		}
		lk.DrawTab(ctx, paintengine2d.XYWH(10, 0, 100, 30), StateNone, "", true)
		if !nxNear(img, 60, 29, c.tabSel) {
			t.Fatalf("%s: the selected tab does not open the page edge", n)
		}
	}
}

// Steel's scroll thumb carries the bumps; Ocean's the three-line grip.
func TestMetalThumbTexture(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range metalPackNames {
		lk := mustLook(t, n)
		c := mtlColors(lk)
		img := paintengine2d.NewImage(17, 200)
		ctx := paintengine2d.NewContext(img)
		p := ScrollGeometry(lk, paintengine2d.XYWH(0, 0, 17, 200), true, 400, 200, 100, false)
		DrawScrollBarParts(lk, ctx, p, true, ScrollState{})
		ctr := p.Thumb.Center()
		dark, light := 0, 0
		for y := int(p.Thumb.Min.Y) + 4; y < int(p.Thumb.Max.Y)-4; y++ {
			for x := 3; x < 13; x++ {
				switch {
				case nxNear(img, x, y, c.p1):
					dark++
				case nxNear(img, x, y, c.p3) || nxNear(img, x, y, c.white):
					light++
				}
			}
		}
		if c.ocean {
			if !nxNear(img, 8, int(ctr.Y)-3, c.p1) || dark > 40 {
				t.Fatalf("%s: the Ocean thumb has a short grip (%d dark pixels)", n, dark)
			}
		} else if dark < 20 || light < 20 {
			t.Fatalf("%s: the Steel thumb shows %d dark and %d light bumps", n, dark, light)
		}
	}
}
