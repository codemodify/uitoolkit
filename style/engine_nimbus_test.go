package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestNimbusPackRegistered(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	if _, ok := pos["nimbus"]; !ok {
		t.Fatal("pack nimbus not registered")
	}
	p, _ := LoadTheme("nimbus")
	if p.Tokens.Engine != "nimbus" || p.Label != "Nimbus" || p.Lineage != "Java" || p.Year != 2008 || p.Tokens.Family != ThemeLight {
		t.Fatalf("nimbus: engine %q label %q lineage %q year %d family %q", p.Tokens.Engine, p.Label, p.Lineage, p.Year, p.Tokens.Family)
	}
	if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
		t.Fatalf("nimbus: summary must be one line: %q", p.Summary)
	}
	if lk := p.Look(); lk.Engine().ID() != "nimbus" {
		t.Fatalf("nimbus look paints with %q", lk.Engine().ID())
	}
	if pos["metal-ocean"] > pos["nimbus"] {
		t.Fatal("Nimbus (2008) sorts after Metal Ocean (2004)")
	}
	// The palette carries the published Nimbus colours.
	pal := mustLook(t, "nimbus").Palette()
	for what, pair := range map[string][2]paintengine2d.Color{
		"control":                   {pal.Background, Hex("#d6d9df")},
		"nimbusBase":                {pal.Accent, Hex("#33628c")},
		"nimbusFocus":               {pal.Focus, Hex("#73a4d1")},
		"nimbusSelectionBackground": {pal.Selection, Hex("#39698a")},
		"nimbusSelectedText":        {pal.TextOnAccent, Hex("#ffffff")},
		"nimbusDisabledText":        {pal.TextMuted, Hex("#8e8f91")},
		"nimbusLightBackground":     {pal.Field, Hex("#ffffff")},
		"nimbusBorder":              {pal.Border, Hex("#9297a1")},
	} {
		if !nearColor(pair[0], pair[1]) {
			t.Errorf("palette %s is %s, want %s", what, colorHexPadded(pair[0]), colorHexPadded(pair[1]))
		}
	}
}

func TestNimbusPaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, _ := LoadTheme("nimbus")
	for _, scale := range []float32{1, 2} {
		lk := WithScale(p.Look(), scale).(*Classic)
		aquaExercise(t, fmt.Sprintf("nimbus@%gx", scale), lk)
		mtlRowsInside(t, fmt.Sprintf("nimbus@%gx", scale), lk)
	}
}

// Every Draw* survives degenerate rects and every state.
func TestNimbusPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, sc := range []float32{1, 2} {
		lk := lunaLook(t, "nimbus", sc)
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
			t.Fatalf("nimbus@%gx left %d saved states", sc, ctx.SaveCount())
		}
	}
}

// The close button sits at the right of the title bar where hit-testing
// looks for it: its × crosses at the centre, dark on the grey glass, white
// on the red glass under the pointer.
func TestNimbusCloseButtonAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, scale := range []float32{1, 2} {
		lk := WithScale(mustLook(t, "nimbus"), scale).(*Classic)
		c := nbColors(lk)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Max.X > b.Max.X || cr.Min.X < b.Max.X-b.Dx()*0.2 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("@%gx: close rect %v not at the right of the title bar of %v (top inset %v)", scale, cr, b, in.Top)
		}
		for _, hot := range []bool{false, true} {
			img := paintengine2d.NewImage(int(b.Max.X+4*scale), int(b.Max.Y+4*scale))
			lk.DrawWindowFrame(paintengine2d.NewContext(img), b, "Dialog", WindowState{Active: true, CanClose: true, CloseHot: hot})
			ctr := cr.Center()
			glyph := c.arrowDark
			if hot {
				glyph = c.white
			}
			if !nxNear(img, int(ctr.X), int(ctr.Y), glyph) {
				r, g, bl, _ := img.PremulAt(int(ctr.X), int(ctr.Y))
				t.Fatalf("@%gx hot=%v: close centre rgb(%d,%d,%d), want the × in %s", scale, hot, r, g, bl, colorHexPadded(glyph))
			}
			if hot {
				r, g, bl, _ := img.PremulAt(int(cr.Min.X+3*c.u), int(ctr.Y))
				if int(r) < int(g)+50 || int(r) < int(bl)+50 {
					t.Fatalf("@%gx: the hot close button is not red: rgb(%d,%d,%d)", scale, r, g, bl)
				}
			}
		}
		if r := lk.WindowCloseRect(paintengine2d.XYWH(0, 0, 20, 20)); !r.Empty() {
			t.Fatalf("@%gx: a frame too small for a title bar reports close rect %v", scale, r)
		}
	}
}

func TestNimbusStyleHintsAndHooks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "nimbus")
	if LookHint(lk, HintDialogPrimaryFirst) != 0 {
		t.Fatal("Nimbus option panes put the default button last (OptionPane.isYesLast)")
	}
	for _, h := range []StyleHint{HintTabsCentered, HintFormLabelsRight, HintHoverFadeMs, HintDefaultPulseMs} {
		if LookHint(lk, h) != 0 {
			t.Fatalf("hint %d = %d, want 0", h, LookHint(lk, h))
		}
	}
	s := ScrollBarStyleOf(lk)
	if s.Arrows != ArrowsEnds || s.Overlay || s.Transient || s.Thickness != 15 || s.MinThumb != 29 {
		t.Fatalf("scroll bar %+v: want Nimbus's 15px gutter, arrows at the ends, 29px minimum thumb", s)
	}
	if !lk.eng().FieldFocusRing(lk) {
		t.Fatal("Nimbus rings focused fields")
	}
	for _, k := range []PopupKind{PopupMenu, PopupTooltip, PopupDialog} {
		if lk.PopupShadow(k).Zero() {
			t.Fatalf("Nimbus floats popups on soft shadows (kind %d)", k)
		}
	}
	if lk.PopupShadow(PopupDialog).Bottom <= lk.PopupShadow(PopupTooltip).Bottom {
		t.Fatal("a dialog's shadow reaches further than a tool tip's")
	}
	if lk.ControlFont(RoleButton) != lk.Font() || lk.ControlFont(RoleMenu) != lk.Font() {
		t.Fatal("Nimbus labels controls in its plain font")
	}
	if m := lk.Metrics(); m.Checkbox != 18 || m.Radio != 18 || m.Thumb != 17 || m.ProgressH != 19 {
		t.Fatalf("Nimbus icon and bar sizes: check %v radio %v thumb %v progress %v", m.Checkbox, m.Radio, m.Thumb, m.ProgressH)
	}
}

// Focus is the nimbusFocus ring in the margin round the control's body.
func TestNimbusFocusRing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "nimbus")
	c := nbColors(lk)
	count := func(st ControlState, draw func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState)) int {
		img := paintengine2d.NewImage(140, 40)
		draw(paintengine2d.NewContext(img), paintengine2d.XYWH(4, 4, 120, 30), st)
		n := 0
		for y := 0; y < 40; y++ {
			for x := 0; x < 140; x++ {
				if nxNear(img, x, y, c.focus) {
					n++
				}
			}
		}
		return n
	}
	for name, draw := range map[string]func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState){
		"button": func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawButton(ctx, b, st, "OK")
		},
		"field": func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawTextField(ctx, b, st, "text", "", 0, 0, 0, false, 0, nil)
		},
		"combo": func(ctx *paintengine2d.Context, b paintengine2d.Rect, st ControlState) {
			lk.DrawComboBox(ctx, b, st, "Choice", false)
		},
	} {
		if plain, focused := count(StateNone, draw), count(StateFocused, draw); plain > 4 || focused < 150 {
			t.Errorf("%s: %d focus-ring pixels unfocused, %d focused", name, plain, focused)
		}
	}
}

// Tables stripe their odd rows with alternateRow and select in
// nimbusSelectionBackground with white text; lists do not stripe.
func TestNimbusStripedTableRows(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "nimbus")
	c := nbColors(lk)
	at := func(st ControlState, draw func(ctx *paintengine2d.Context, b paintengine2d.Rect)) (paintengine2d.Color, bool) {
		img := paintengine2d.NewImage(80, 24)
		draw(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 80, 24))
		r, g, b, a := img.PremulAt(4, 12)
		return paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255), a > 0
	}
	cell := func(st ControlState) func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		return func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawTableCell(ctx, b, st, "", AlignStart, nil)
		}
	}
	if col, ok := at(StateAlternate, cell(StateAlternate)); !ok || colorHexPadded(col) != "#f2f2f2" {
		t.Fatalf("odd table row %s (painted %v), want #f2f2f2", colorHexPadded(col), ok)
	}
	if _, ok := at(StateNone, cell(StateNone)); ok {
		t.Fatal("even table rows leave the view's white showing")
	}
	if col, _ := at(StateChecked|StateAlternate, cell(StateChecked|StateAlternate)); colorHexPadded(col) != "#39698a" {
		t.Fatalf("selected row %s, want nimbusSelectionBackground", colorHexPadded(col))
	}
	if _, ok := at(StateAlternate, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawListRow(ctx, b, StateAlternate, "") }); ok {
		t.Fatal("list rows are not striped")
	}
	if ContrastRatio(c.selText, c.sel) < 4.5 {
		t.Fatal("selected text reads on the selection")
	}
}

// Indeterminate progress runs diagonal stripes of light and deep orange
// across the whole well, and they move with the phase.
func TestNimbusIndeterminateStripes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "nimbus")
	row := func(phase float32) []byte {
		img := paintengine2d.NewImage(200, 19)
		lk.DrawProgressBar(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 200, 19), StateNone, 0, true, phase)
		out := make([]byte, 0, 160*4)
		for x := 20; x < 180; x++ {
			r, g, b, a := img.PremulAt(x, 10)
			out = append(out, r, g, b, a)
		}
		return out
	}
	a := row(0)
	light, deep := 0, 0
	for i := 0; i < len(a); i += 4 {
		r, g, b := int(a[i]), int(a[i+1]), int(a[i+2])
		if r < g || g < b {
			t.Fatalf("stripe pixel rgb(%d,%d,%d) is not orange", r, g, b)
		}
		if g > 150 {
			light++
		} else if g < 120 {
			deep++
		}
	}
	if light < 30 || deep < 30 {
		t.Fatalf("indeterminate bar: %d light and %d deep orange pixels, want both stripes", light, deep)
	}
	if bytesEqual(a, row(0.3)) {
		t.Fatal("the stripes do not move with the phase")
	}
	// Determinate: the fill stops at the value.
	img := paintengine2d.NewImage(200, 19)
	lk.DrawProgressBar(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 200, 19), StateNone, 0.5, false, 0)
	if r, g, _, _ := img.PremulAt(60, 10); int(r) < int(g)+40 {
		t.Fatal("the progress fill is nimbusOrange")
	}
	if r, g, b, _ := img.PremulAt(160, 10); int(r)-int(b) > 30 || int(g)-int(b) > 30 {
		t.Fatal("past the value the well stays grey")
	}
}

// Labels read on the fills they are painted on.
func TestNimbusLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "nimbus")
	c := nbColors(lk)
	mid := func(g []paintengine2d.GradientStop) paintengine2d.Color { return g[len(g)/2].Color }
	for _, chk := range []struct {
		what   string
		fg, bg paintengine2d.Color
		min    float64
	}{
		{"text on the canvas", c.text, c.control, 4.5},
		{"rows on white", c.rowText, c.light, 4.5},
		{"rows on the stripe", c.rowText, c.alt, 4.5},
		{"selected text", c.selText, c.sel, 4.5},
		{"button label", c.text, mid(c.grey.face), 4.5},
		{"pressed button label", c.text, mid(c.greyPress.face), 4.5},
		{"default button label", c.text, mid(c.blue.face), 4.5},
		{"menu text", c.menuText, c.menuBg, 4.5},
		{"accelerator", c.accel, c.menuBg, 4.5},
		{"hot menu text", c.selText, mid(c.hotGrad), 4.5},
		{"tool tip", c.tipText, c.control, 4.5},
		{"disabled text", c.dis, c.control, 1.8},
	} {
		if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
			t.Errorf("%s contrast %.2f < %.1f", chk.what, r, chk.min)
		}
	}
}
