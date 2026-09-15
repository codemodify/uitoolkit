package style

import (
	"reflect"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// webPackNames are every pack the web engine registers.
func webPackNames() []string {
	var out []string
	for _, s := range webSpecs() {
		out = append(out, s.name)
	}
	return out
}

func TestWebPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		if _, dup := pos[n]; dup {
			t.Fatalf("pack %q listed twice", n)
		}
		pos[n] = i
	}
	names := webPackNames()
	if len(names) != 27 {
		t.Fatalf("%d web packs, want 27: %v", len(names), names)
	}
	for _, n := range names {
		p, ok := LoadTheme(n)
		if !ok {
			t.Fatalf("pack %q not registered", n)
		}
		if p.Tokens.Engine != "web" || p.Year < 2013 || p.Year > 2026 || p.Label == "" || p.Lineage == "" || p.Era == "" {
			t.Fatalf("%s: engine %q year %d label %q lineage %q era %q", n, p.Tokens.Engine, p.Year, p.Label, p.Lineage, p.Era)
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if p.Tokens.Bevel != BevelNone {
			t.Fatalf("%s: bevel %q", n, p.Tokens.Bevel)
		}
		if lk := p.Look(); lk.Engine().ID() != "web" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
		wantDark := strings.HasSuffix(n, "-night") || strings.HasSuffix(n, "-dimmed") || n == "dracula" ||
			strings.Contains(n, "frappe") || strings.Contains(n, "macchiato") || strings.Contains(n, "mocha") ||
			n == "tokyonight" || n == "tokyonight-storm" || n == "rosepine" || n == "rosepine-moon"
		if (p.Tokens.Family == ThemeDark) != wantDark {
			t.Fatalf("%s: family %q", n, p.Tokens.Family)
		}
	}
	// Registered packs sort by year: Dracula (2013) before Nord (2016),
	// Tokyo Night (2020), SourceGit (2024), shadcn (2025) and Linear (2026).
	order := []string{"dracula", "nord", "tokyonight", "catppuccin-mocha", "primer", "sourcegit", "shadcn", "linear"}
	for i := 1; i < len(order); i++ {
		if pos[order[i-1]] > pos[order[i]] {
			t.Fatalf("%s sorts after %s", order[i-1], order[i])
		}
	}
}

// Every colour a pack names parses, and no resolved colour is the magenta
// sentinel of a missing key.
func TestWebColourTables(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, s := range webSpecs() {
		for k, v := range s.c {
			if _, ok := ParseHexColor(v); !ok {
				t.Errorf("%s: %s = %q is not a colour", s.name, k, v)
			}
		}
		lk := winLook(t, s.name, 1)
		c := webColors(lk)
		if c != webColors(lk) {
			t.Fatalf("%s: colours are rebuilt per paint", s.name)
		}
		winNoSentinel(t, s.name, reflect.ValueOf(*c))
	}
}

// The packs read in their own text sizes: SourceGit and Linear 13px, the
// web systems and palettes 14px.
func TestWebPackTextSizes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for n, want := range map[string]float32{"sourcegit": 13, "sourcegit-night": 13, "linear": 13, "primer": 14, "shadcn-night": 14, "catppuccin-latte": 14} {
		lk := winLook(t, n, 1)
		if got := lk.Metrics().FontSize; got != want {
			t.Errorf("%s: text %vpx, want %v", n, got, want)
		}
		if got := winLook(t, n, 2).Metrics().FontSize; got != 2*want {
			t.Errorf("%s@2x: text %vpx, want %v", n, got, 2*want)
		}
	}
}

func TestWebPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsEverything(t, webPackNames())
}

func TestWebPaintsInsideRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	names := webPackNames()
	winPaintsInsideRect(t, names)
	for _, n := range names {
		for _, sc := range []float32{1, 2} {
			lk := winLook(t, n, sc)
			aquaExercise(t, n, lk)
			kdeExtras(t, n, lk)
		}
	}
	// Browser tabs and every tab style paint inside their rects too.
	for _, n := range []string{"sourcegit", "shadcn", "primer"} {
		for _, sc := range []float32{1, 2} {
			lk := webLookWith(t, n, sc, map[string]float32{"tabStyle": webTabBrowser})
			for _, st := range winStates {
				img := paintengine2d.NewImage(int(140*sc), int(72*sc))
				r := paintengine2d.XYWH(20*sc, 20*sc, 100*sc, 32*sc)
				ctx := paintengine2d.NewContext(img)
				lk.DrawTabBar(ctx, r)
				lk.DrawTab(ctx, r, st, "sourcegit", st.Checked())
				DrawBrowserTabOf(lk, ctx, r, st, "tab", !st.Checked())
				if k := winOutside(img, r); k > 0 {
					t.Errorf("%s %gx browser tab %#x: %d pixels outside", n, sc, uint32(st), k)
				}
			}
		}
	}
}

func TestWebCloseRectAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range []string{"sourcegit", "sourcegit-night", "primer", "shadcn", "geist-night", "linear", "catppuccin-mocha"} {
		winCheckClose(t, n)
	}
	// SourceGit's close cell turns pure red under the pointer.
	for _, sc := range []float32{1, 2} {
		lk := winLook(t, "sourcegit", sc)
		b := paintengine2d.XYWH(10*sc, 10*sc, 300*sc, 180*sc)
		cr := lk.WindowCloseRect(b)
		img := winFrame(lk, b, WindowState{Active: true, CanClose: true, CloseHot: true})
		if !winNear(img, int(cr.Min.X+cr.Dx()*0.2), int(cr.Min.Y+cr.Dy()*0.3), Hex("#ff0000")) {
			t.Fatalf("sourcegit@%gx: the hot close cell is not red", sc)
		}
		if cr.Dx() != snap(48*sc) {
			t.Fatalf("sourcegit@%gx: close cell %v wide, want 48", sc, cr.Dx()/sc)
		}
	}
}

// Dialog button order follows each design system: SourceGit's popups put
// OK before Cancel; GitHub, shadcn, Geist and Linear put the primary action
// last, as the palettes' web ports do. Mnemonics show while Alt is held.
func TestWebStyleHints(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range webPackNames() {
		lk := winLook(t, n, 1)
		want := 0
		if strings.HasPrefix(n, "sourcegit") {
			want = 1
		}
		if got := LookHint(lk, HintDialogPrimaryFirst); got != want {
			t.Errorf("%s: primary first %d, want %d", n, got, want)
		}
		if LookHint(lk, HintMnemonics) != MnemonicsOnAlt || LookHint(lk, HintTabsCentered) != 0 || LookHint(lk, HintFormLabelsRight) != 0 {
			t.Errorf("%s: mnemonics on Alt, left tabs and labels", n)
		}
		s := ScrollBarStyleOf(lk)
		if !s.Overlay || !s.Transient {
			t.Errorf("%s: transient overlay scroll bars, got %+v", n, s)
		}
	}
	if s := ScrollBarStyleOf(winLook(t, "sourcegit", 1)); s.Thickness != 8 || s.Arrows != ArrowsEnds {
		t.Errorf("sourcegit: an 8px bar with arrow cells at the edge, got %+v", s)
	}
}

// webLookWith is pack n's look with some params changed.
func webLookWith(t *testing.T, n string, scale float32, params map[string]float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(n)
	if !ok {
		t.Fatalf("pack %q not found", n)
	}
	tok := CloneTokenMaps(p.Tokens)
	for k, v := range params {
		tok.Params[k] = v
	}
	lk := newClassic(string(tok.Family), tok.Palette, packMetrics(DensityDefault, tok, CornersTheme, IconSizeMedium),
		CornersTheme, IconSetClassic, IconSizeMedium, withEraFonts(tok, n)).setPack(n)
	if scale != 1 {
		return WithScale(lk, scale).(*Classic)
	}
	return lk
}

// webPaint paints f into a w×h image on the look's window colour.
func webPaint(lk *Classic, w, h int, f func(ctx *paintengine2d.Context)) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(w), float32(h)), paintengine2d.Fill(webColors(lk).window))
	f(ctx)
	return img
}

// SourceGit's controls as its Styles.axaml draws them.
func TestWebSourceGitControls(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := winLook(t, "sourcegit", 1)
	c := webColors(lk)
	b := paintengine2d.XYWH(10, 10, 120, 28)

	// Square text fields: the corner pixel is the border.
	field := webPaint(lk, 140, 48, func(ctx *paintengine2d.Context) { lk.DrawTextField(ctx, b, StateNone, "", "", 0, 0, 0, false, 0, nil) })
	if !winNear(field, 10, 10, c.border1) {
		t.Errorf("a text field's corner is not square: %v", pixelColor(field, 10, 10))
	}
	// ...that take the accent border under the pointer.
	if hot := webPaint(lk, 140, 48, func(ctx *paintengine2d.Context) {
		lk.DrawTextField(ctx, b, StateHovered, "", "", 0, 0, 0, false, 0, nil)
	}); !winNear(hot, 60, 10, c.accent) {
		t.Errorf("a hovered field's border is not the accent")
	}

	// 3px buttons: the corner pixel is the window, two pixels in the face.
	btn := webPaint(lk, 140, 48, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, b, StateNone, "") })
	if !winNear(btn, 10, 10, c.window) || !winNear(btn, 13, 13, c.btn) || !winNear(btn, 60, 10, c.btnBorder) {
		t.Errorf("a button is not a 3px-rounded flat face in Border2")
	}

	// The selected tab: an accent label over a 1px accent pipe under it,
	// two pixels above the bottom.
	tab := paintengine2d.XYWH(10, 10, 120, 30)
	sel := webPaint(lk, 140, 50, func(ctx *paintengine2d.Context) { lk.DrawTab(ctx, tab, StateNone, "INFORMATION", true) })
	if !winNear(sel, 70, 37, c.accent) || winNear(sel, 70, 38, c.accent) || winNear(sel, 70, 36, c.accent) {
		t.Errorf("no 1px accent pipe 2px above the selected tab's bottom")
	}
	if winNear(webPaint(lk, 140, 50, func(ctx *paintengine2d.Context) { lk.DrawTab(ctx, tab, StateNone, "INFORMATION", false) }), 70, 37, c.accent) {
		t.Errorf("an unselected tab has the pipe")
	}

	// An unfilled check box with an accent tick.
	cb := paintengine2d.XYWH(10, 10, 100, 22)
	off := webPaint(lk, 120, 40, func(ctx *paintengine2d.Context) { lk.DrawCheckbox(ctx, cb, StateNone, false, "") })
	on := webPaint(lk, 120, 40, func(ctx *paintengine2d.Context) { lk.DrawCheckbox(ctx, cb, StateNone, true, "") })
	box := flatCentered(paintengine2d.XYWH(10, 13, 16, 16), 16, 16)
	if !winNear(off, int(box.Min.X)+3, int(box.Min.Y)+3, c.window) || !winNear(on, int(box.Min.X)+2, int(box.Min.Y)+2, c.window) {
		t.Errorf("a check box is filled")
	}
	accent := 0
	for y := int(box.Min.Y); y < int(box.Max.Y); y++ {
		for x := int(box.Min.X); x < int(box.Max.X); x++ {
			if winNear(on, x, y, c.accent) {
				accent++
			}
		}
	}
	if accent < 8 {
		t.Errorf("the checked box has no accent tick (%d accent pixels)", accent)
	}
	// Keyboard focus: a 2px accent border, and the box fills when checked.
	if f := webPaint(lk, 120, 40, func(ctx *paintengine2d.Context) { lk.DrawCheckbox(ctx, cb, StateFocused, true, "") }); !winNear(f, int(box.Min.X)+3, int(box.Min.Y)+3, c.accent) {
		t.Errorf("a focused checked box is not filled with the accent")
	}

	// The dotted focus adorner on a focused button.
	foc := webPaint(lk, 140, 48, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, b, StateFocused, "") })
	// Dots of one pixel every three along the edge (Avalonia's 1,2 dash).
	if !winNear(foc, 10, 19, c.text) || !winNear(foc, 10, 22, c.text) || winNear(foc, 10, 20, c.text) {
		t.Errorf("no dotted focus adorner on the button's edge")
	}
}

// Consecutive selected sidebar rows join into one box: the shared corners
// go square, the outer ones stay round.
func TestWebJoinedInsetSelection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := winLook(t, "sourcegit", 1)
	c := webColors(lk)
	row := func(st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(200, 24)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, 200, 24), paintengine2d.Fill(c.sidebar))
		lk.DrawListRow(ctx, paintengine2d.XYWH(0, 0, 200, 24), st|StateSidebar, "")
		return img
	}
	fill := webOver(c.sidebar, webA(c.sel, 0.65))
	alone := row(StateChecked)
	in := 6 // the sidebar inset
	if winNear(alone, in, 0, fill) || winNear(alone, in, 23, fill) || !winNear(alone, in+6, 12, fill) {
		t.Errorf("a lone selected row is not an inset round box")
	}
	if winNear(alone, 2, 12, fill) {
		t.Errorf("the selection is not inset from the sidebar's side")
	}
	below := row(StateChecked | StateSelectedBelow)
	if winNear(below, in, 0, fill) || !winNear(below, in, 23, fill) {
		t.Errorf("a row joined to the one below keeps its top corner and squares its bottom one")
	}
	above := row(StateChecked | StateSelectedAbove)
	if !winNear(above, in, 0, fill) || winNear(above, in, 23, fill) {
		t.Errorf("a row joined to the one above squares its top corner")
	}
	// Without focus the selection is the neutral wash.
	if !winNear(row(StateChecked|StateInactive), in+10, 12, webOver(c.sidebar, c.wash)) {
		t.Errorf("an unfocused sidebar selection is not the 10%% wash")
	}
}

// The params change shapes: a pack's corners, focus style, tab style,
// check style and primary style are its own.
func TestWebParamsChangeShapes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	b := paintengine2d.XYWH(10, 10, 120, 30)
	paint := func(lk *Classic, st ControlState) *paintengine2d.Image {
		return webPaint(lk, 140, 50, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, b, st, "") })
	}
	r3, r8 := webLookWith(t, "sourcegit", 1, map[string]float32{"radius": 3}), webLookWith(t, "sourcegit", 1, map[string]float32{"radius": 8})
	a, z := paint(r3, StateNone), paint(r8, StateNone)
	if winSame(a, z) {
		t.Fatal("radius 3 and radius 8 paint the same button")
	}
	c := webColors(r8)
	if !winNear(a, 12, 12, c.btn) || winNear(z, 12, 12, c.btn) {
		t.Errorf("an 8px corner does not cut deeper than a 3px one")
	}
	// The Corners pref squares any radius.
	sq := webLookWith(t, "sourcegit", 1, map[string]float32{"radius": 8})
	sq.corners = CornersSquare
	if !winNear(paint(sq, StateNone), 10, 12, c.btnBorder) {
		t.Errorf("square corners keep the pack's radius")
	}
	// Every focus style shows, each its own way.
	for _, fs := range []float32{webFocusOutside, webFocusInside, webFocusDotted} {
		lk := webLookWith(t, "primer", 1, map[string]float32{"focusStyle": fs})
		if winSame(paint(lk, StateNone), paint(lk, StateFocused)) {
			t.Errorf("focus style %v: focus does not show", fs)
		}
	}
	out := webLookWith(t, "shadcn", 1, nil)
	if f := paint(out, StateFocused); !winDiffers(f, paint(out, StateNone), 11, 25) {
		t.Errorf("shadcn's ring is not outside the face")
	}
	// Geist rings a focused button in blue beyond a gap, but a focused field
	// takes a grey halo round its darker border.
	geist := winLook(t, "geist", 1)
	gc := webColors(geist)
	if !winNear(paint(geist, StateFocused), 70, 10, gc.focus) {
		t.Errorf("geist's focused button has no blue ring at its rect's edge")
	}
	gf := webPaint(geist, 140, 50, func(ctx *paintengine2d.Context) {
		geist.DrawTextField(ctx, b, StateFocused, "", "", 0, 0, 0, false, 0, nil)
	})
	if halo := webOver(gc.window, gc.fieldRing); !winNear(gf, 70, 12, halo) || winNear(gf, 70, 12, gc.focus) {
		t.Errorf("geist's focused field has no grey halo: %v", pixelColor(gf, 70, 12))
	}
	if !winNear(gf, 70, 14, webOver(gc.field, gc.fieldFocus)) {
		t.Errorf("geist's focused field border is not gray-alpha-600: %v", pixelColor(gf, 70, 14))
	}
	// Underline, segmented and browser tabs differ.
	tabs := map[float32]*paintengine2d.Image{}
	for _, ts := range []float32{webTabUnderline, webTabSegmented, webTabBrowser} {
		lk := webLookWith(t, "primer", 1, map[string]float32{"tabStyle": ts})
		tabs[ts] = webPaint(lk, 140, 50, func(ctx *paintengine2d.Context) { lk.DrawTab(ctx, b, StateNone, "Tab", true) })
	}
	if winSame(tabs[0], tabs[1]) || winSame(tabs[1], tabs[2]) || winSame(tabs[0], tabs[2]) {
		t.Errorf("tab styles paint alike")
	}
	// A filled check box against SourceGit's unfilled one.
	for _, n := range []string{"primer", "shadcn"} {
		lk := winLook(t, n, 1)
		c := webColors(lk)
		img := webPaint(lk, 120, 40, func(ctx *paintengine2d.Context) {
			lk.DrawCheckbox(ctx, paintengine2d.XYWH(10, 9, 100, 22), StateNone, true, "")
		})
		g := flatCentered(paintengine2d.XYWH(10, 9, 22, 22), 16, 16)
		if !winNear(img, int(g.Min.X)+2, int(g.Max.Y)-3, c.checkOn) {
			t.Errorf("%s: a checked box is not filled with %s", n, colorHexPadded(c.checkOn))
		}
	}
	// Primer's green, shadcn's near-black and SourceGit's accent defaults.
	for n, want := range map[string]string{"primer": "#1f883d", "shadcn": "#171717", "sourcegit": "#0078d7", "geist-night": "#ededed"} {
		lk := winLook(t, n, 1)
		if !winNear(paint(lk, StatePrimary), 70, 25, Hex(want)) {
			t.Errorf("%s: the default button is not %s", n, want)
		}
	}
}

// SourceGit follows the desktop's accent (its shades as Avalonia derives
// them); the design systems and palettes keep their own.
func TestWebAccent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range webPackNames() {
		p, _ := LoadTheme(n)
		if got, want := TakesAccent(p), strings.HasPrefix(n, "sourcegit"); got != want {
			t.Errorf("%s: takes the accent %v, want %v", n, got, want)
		}
	}
	lk := accLook(t, "sourcegit", Hex("#e95420"), 1)
	c := webColors(lk)
	if !accentSame(c.accent, Hex("#e95420")) || !accentSame(c.accentHover, webLight1(Hex("#e95420"))) || !accentSame(c.primary, Hex("#e95420")) {
		t.Errorf("sourcegit with Ubuntu's orange: accent %s hover %s primary %s", accHex(c.accent), accHex(c.accentHover), accHex(c.primary))
	}
	// Avalonia's Light1 of the fallback #0078D7 is #269FFF.
	if got := webLight1(Hex("#0078d7")); !accentSame(got, Hex("#269fff")) {
		t.Errorf("Light1 of #0078d7 is %s, want #269fff", accHex(got))
	}
}

// Follow the desktop moves within each family.
func TestWebSchemeSiblings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range []struct {
		name   string
		scheme ColorScheme
		want   string
	}{
		{"sourcegit", SchemeDark, "sourcegit-night"}, {"sourcegit-night", SchemeLight, "sourcegit"},
		{"primer", SchemeDark, "primer-night"}, {"primer-dimmed", SchemeLight, "primer"},
		{"shadcn", SchemeDark, "shadcn-night"}, {"geist-night", SchemeLight, "geist"}, {"linear", SchemeDark, "linear-night"},
		{"catppuccin-latte", SchemeDark, "catppuccin-mocha"}, {"catppuccin-frappe", SchemeLight, "catppuccin-latte"},
		{"catppuccin-mocha", SchemeLight, "catppuccin-latte"}, {"catppuccin-macchiato", SchemeDark, "catppuccin-macchiato"},
		{"nord", SchemeDark, "nord-night"}, {"nord-night", SchemeLight, "nord"},
		{"alucard", SchemeDark, "dracula"}, {"dracula", SchemeLight, "alucard"},
		{"tokyonight-day", SchemeDark, "tokyonight"}, {"tokyonight-storm", SchemeLight, "tokyonight-day"}, {"tokyonight", SchemeLight, "tokyonight-day"},
		{"rosepine-dawn", SchemeDark, "rosepine"}, {"rosepine-moon", SchemeLight, "rosepine-dawn"}, {"rosepine", SchemeLight, "rosepine-dawn"},
		{"vscode", SchemeDark, "vscode-night"}, {"vscode-night", SchemeLight, "vscode"},
	} {
		if got := SchemeVariant(c.name, c.scheme); got != c.want {
			t.Errorf("SchemeVariant(%q, %v) = %q, want %q", c.name, c.scheme, got, c.want)
		}
	}
}

// Labels read on what the engine paints them on, in every pack.
func TestWebLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	systems := map[string]bool{}
	for _, s := range webSpecs() {
		systems[s.name] = s.m != paletteMetrics
	}
	for _, n := range webPackNames() {
		lk := winLook(t, n, 1)
		c := webColors(lk)
		// The palettes keep their own pairs, some under 4.5:1 (Tokyo Night
		// Day's #3760bf text on its #d0d5e3 popups is 3.99:1).
		floor := 4.5
		if !systems[n] {
			floor = 3.5
		}
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on window", c.text, c.window, 4.5},
			{"text on field", lk.fieldText(), c.field, 4.5},
			{"text on popup", c.text, c.popup, floor},
			{"button label", c.btnText, c.flat(c.btn), floor},
			{"primary label", c.onPrimary, c.flat(c.primary), 3},
			{"hot menu label", c.menuText, webOver(c.popup, c.menuHover), floor},
			{"tool tip", c.tipText, webOver(c.popup, c.tip), floor},
			{"selected list row", c.rowLabel(StateChecked, webList), webOver(c.field, c.rowFill(StateChecked, webList)), 4.5},
			{"selected table row", c.rowLabel(StateChecked, webTable), webOver(c.field, c.rowFill(StateChecked, webTable)), 4.5},
			{"muted text", c.text2, c.window, 3},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s %s on %s is %.2f:1 < %.1f", n, chk.what, colorHexPadded(chk.fg), colorHexPadded(chk.bg), r, chk.min)
			}
		}
	}
}
