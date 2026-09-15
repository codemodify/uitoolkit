package style

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var aeroPackNames = []string{"aero", "aero-basic"}

func TestAeroPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winCheckPacks(t, "aero", []winPackWant{
		{"aero", "Aero", 2007, ThemeLight},
		{"aero-basic", "Windows 7 Basic", 2009, ThemeLight},
	})
}

// Every override key exists in the base table and no resolved colour is
// the missing-key sentinel.
func TestAeroColourTable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for k := range aeroBasic {
		if _, ok := aeroBase[k]; !ok {
			t.Errorf("aeroBasic key %q is not in aeroBase", k)
		}
	}
	for _, n := range aeroPackNames {
		lk := winLook(t, n, 1)
		c := aeroColors(lk)
		if c != aeroColors(lk) {
			t.Fatalf("%s: colours are rebuilt per paint", n)
		}
		winNoSentinel(t, n, reflect.ValueOf(*c))
	}
	if c := aeroColors(winLook(t, "aero-basic", 1)); c.glass {
		t.Fatal("Windows 7 Basic must not paint the glass frame")
	}
}

func TestAeroPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsEverything(t, aeroPackNames)
}

func TestAeroPaintsInsideRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsInsideRect(t, aeroPackNames)
}

func TestAeroCloseRectAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range aeroPackNames {
		winCheckClose(t, n)
	}
	// The close button is Windows 7's red glass.
	lk := winLook(t, "aero", 1)
	b := paintengine2d.XYWH(4, 4, 300, 180)
	img := winFrame(lk, b, WindowState{Active: true, CanClose: true})
	cr := lk.WindowCloseRect(b)
	r, g, bl, _ := img.PremulAt(int(cr.Min.X+cr.Dx()*0.2), int(cr.Min.Y+cr.Dy()*0.75))
	if int(r) < int(g)+50 || int(r) < int(bl)+50 {
		t.Fatalf("close button is %d,%d,%d, not red", r, g, bl)
	}
}

func TestAeroStyleHints(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range aeroPackNames {
		winCheckHints(t, n)
	}
}

func TestAeroLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range aeroPackNames {
		lk := winLook(t, n, 1)
		c := aeroColors(lk)
		mid := func(s []paintengine2d.GradientStop) paintengine2d.Color { return s[len(s)/2].Color }
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on face", c.text, c.face, 4.5},
			{"text on field", lk.fieldText(), c.field, 4.5},
			{"selected text", c.selText, c.sel, 3},
			{"button label", c.text, mid(c.btn.face), 4.5},
			{"hot button label", c.text, mid(c.btnHot.face), 4.5},
			{"pressed button label", c.text, mid(c.btnPress.face), 4.5},
			{"disabled label", c.disText, c.btnDis.face[0].Color, 3},
			{"hot menu label", lk.Engine().MenuTextColor(lk, true), mid(c.menuHot.face), 4.5},
			{"selected row", lk.fieldText(), mid(c.selBox.face), 4.5},
			{"tool tip", c.tipText, mid(c.tip), 4.5},
			{"header", c.hdrText, c.hdrStops[0].Color, 4.5},
			{"main instruction", c.instr, c.face, 4.5},
			{"group header", c.groupHead, c.field, 4.5},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// Glass: the push button's face is two-tone, lighter over its top half;
// hot turns it blue.
func TestAeroButtonIsTwoToneGlass(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := winLook(t, "aero", 1)
	paint := func(st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(100, 40)
		lk.DrawButton(paintengine2d.NewContext(img), paintengine2d.XYWH(5, 5, 90, 30), st, "")
		return img
	}
	img := paint(StateNone)
	tr, _, _, _ := img.PremulAt(20, 12)
	br, _, _, _ := img.PremulAt(20, 28)
	if int(tr) < int(br)+12 {
		t.Fatalf("top half %d not lighter than bottom half %d", tr, br)
	}
	hot := paint(StateHovered)
	r, _, b, _ := hot.PremulAt(20, 28)
	if int(b) < int(r)+25 {
		t.Fatalf("hot face %d,_,%d is not blue", r, b)
	}
}

// Windows 7's scroll arrows are glyphs at rest; their button chrome shows
// only while the pointer is over the bar.
func TestAeroScrollArrowsShowChromeOnHover(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := winLook(t, "aero", 1)
	view := paintengine2d.XYWH(0, 0, 40, 200)
	p := ScrollGeometry(lk, view, true, 800, 200, 100, false)
	if p.Dec.Empty() || p.Inc.Empty() {
		t.Fatalf("aero bars have arrow buttons: %+v", p)
	}
	rest := winInk(lk, view, p, ScrollState{}, p.Dec)
	hover := winInk(lk, view, p, ScrollState{Hovered: true}, p.Dec)
	if hover <= rest+int(p.Dec.Dx()) {
		t.Fatalf("arrow button chrome: %d pixels differ from the shaft at rest, %d on hover", rest, hover)
	}
}

// ---- helpers shared by the aero, metro and fluent tests ----------------------------------------

type winPackWant struct {
	name, label string
	year        int
	fam         ThemeName
}

func winLook(t *testing.T, name string, scale float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("pack %q not found", name)
	}
	var lk LookAndFeel = p.Look()
	if scale != 1 {
		lk = WithScale(lk, scale)
	}
	c, ok := lk.(*Classic)
	if !ok {
		t.Fatalf("%s: look is %T", name, lk)
	}
	return c
}

// winCheckPacks checks that each pack is registered once, paints with
// engine, carries its metadata and that the list is in year order.
func winCheckPacks(t *testing.T, engine string, want []winPackWant) {
	t.Helper()
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		if _, dup := pos[n]; dup {
			t.Fatalf("pack %q listed twice", n)
		}
		pos[n] = i
	}
	prev := -1
	for _, w := range want {
		i, ok := pos[w.name]
		if !ok {
			t.Fatalf("pack %q not registered", w.name)
		}
		if i < prev {
			t.Fatalf("pack %q out of year order", w.name)
		}
		prev = i
		p, _ := LoadTheme(w.name)
		if p.Tokens.Engine != engine || p.Lineage != "Windows" || p.Year != w.year || p.Label != w.label || p.Summary == "" {
			t.Fatalf("%s: engine %q lineage %q year %d label %q summary %q", w.name, p.Tokens.Engine, p.Lineage, p.Year, p.Label, p.Summary)
		}
		if p.Tokens.Family != w.fam {
			t.Fatalf("%s: family %q want %q", w.name, p.Tokens.Family, w.fam)
		}
		if lk := p.Look(); lk.Engine().ID() != engine {
			t.Fatalf("%s look paints with %q", w.name, lk.Engine().ID())
		}
	}
}

// winNoSentinel fails when a resolved colour set holds the magenta the
// builders return for a key missing from their tables.
func winNoSentinel(t *testing.T, name string, v reflect.Value) {
	t.Helper()
	sentinel := Hex("#ff00ff")
	colorT := reflect.TypeOf(paintengine2d.Color{})
	var walk func(v reflect.Value, path string)
	walk = func(v reflect.Value, path string) {
		switch {
		case v.Type() == colorT:
			c := paintengine2d.RGBA(float32(v.FieldByName("R").Float()), float32(v.FieldByName("G").Float()),
				float32(v.FieldByName("B").Float()), float32(v.FieldByName("A").Float()))
			if nearColor(c, sentinel) && c.A > 0.99 {
				t.Errorf("%s: %s is the missing-key sentinel", name, path)
			}
		case v.Kind() == reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i), path+"."+v.Type().Field(i).Name)
			}
		case v.Kind() == reflect.Array || v.Kind() == reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	walk(v, "set")
}

var winStates = []ControlState{
	StateNone, StateHovered, StatePressed | StateHovered, StateFocused, StateDisabled,
	StateChecked, StatePrimary, StatePrimary | StateFocused, StatePrimary | StatePressed, StateToggle,
	StateToggle | StateChecked, StateToggle | StateChecked | StateHovered,
	StateDisabled | StateChecked, StateFocused | StateHovered, StateChecked | StateInactive,
	StateChecked | StateInactive | StateBackdrop, StateChecked | StateFocused | StateHovered,
	StateFirst, StateLast, StateAlternate | StateHovered,
}

// winCalls paints every stateful control of the look into r.
func winCalls(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
	e := lk.Engine()
	lines := []TextLine{{Text: "A text area", Start: 0, End: 11}, {Text: "two", Start: 12, End: 15}}
	lk.DrawButton(ctx, r, st, "Button")
	lk.DrawToolButton(ctx, r, st, "", IconSave)
	lk.DrawToolButton(ctx, r, st, "Fetch", IconOpen)
	lk.DrawCheckbox(ctx, r, st, true, "Checked")
	lk.DrawCheckbox(ctx, r, st, false, "")
	lk.DrawRadio(ctx, r, st, true, "Selected")
	lk.DrawRadio(ctx, r, st, false, "")
	lk.DrawSwitch(ctx, r, st, true, "On")
	lk.DrawSwitch(ctx, r, st, false, "")
	for _, v := range []float32{-1, 0, 0.5, 1, 2} {
		lk.DrawSlider(ctx, r, st, v)
		lk.DrawProgressBar(ctx, r, st, v, false, 0)
		lk.DrawProgressBar(ctx, r, st, 0, true, v*3.7)
	}
	lk.DrawTextField(ctx, r, st, "Ada Lovelace", "Placeholder", 3, 0, 3, true, 0, nil)
	lk.DrawTextField(ctx, r, st, "", "Placeholder", 0, 0, 0, false, 0, nil)
	lk.DrawTextArea(ctx, r, st, lines, 5, 2, 6, true, 0, 0, "", nil)
	lk.DrawComboBox(ctx, r, st, "Choice", false)
	lk.DrawComboBox(ctx, r, st, "Open", true)
	lk.DrawSpinner(ctx, r, st, true, false, false, true)
	lk.DrawSpinner(ctx, r, st, false, true, true, false)
	lk.DrawTab(ctx, r, st, "Selected", true)
	lk.DrawTab(ctx, r, st, "Tab", false)
	lk.DrawMenuTitle(ctx, r, st, "File", 0, true)
	lk.DrawMenuTitle(ctx, r, st, "Edit", 0, false)
	for _, row := range []MenuRow{
		{Label: "Open…", Shortcut: "Ctrl+O", Icon: IconOpen, Underline: 0},
		{Separator: true},
		{Label: "Word wrap", Checked: true},
		{Label: "Bold", Checked: true, Icon: IconSave},
		{Label: "Compact", Radio: true, Checked: true},
		{Label: "Relaxed", Radio: true},
		{Label: "Recent", Submenu: true, Shortcut: "F2"},
	} {
		lk.DrawMenuItem(ctx, r, st, row)
	}
	lk.DrawTableHeader(ctx, r, st, "Name", true, true)
	lk.DrawTableHeader(ctx, r, st, "Size", true, false)
	lk.DrawTableHeader(ctx, r, st, "Plain", false, false)
	lk.DrawAccordionHeader(ctx, r, st, "Section", true)
	lk.DrawAccordionHeader(ctx, r, st, "Section", false)
	lk.DrawSplitter(ctx, r, true, st)
	lk.DrawSplitter(ctx, r, false, st)
	lk.DrawScrollBar(ctx, r, r.Inset(2), st)
	lk.DrawListRow(ctx, r, st, "Row")
	lk.DrawTreeRow(ctx, r, st, true, false, 1, "Inbox", true)
	lk.DrawTreeRow(ctx, r, st, false, true, 2, "2026", false)
	lk.DrawTableCell(ctx, r, st, "Cell", AlignEnd, nil)
	lk.DrawItemFocus(ctx, r, st)
	lk.DrawViewFrame(ctx, r, st)
	for role := RoleButton; role <= RolePanel; role++ {
		e.Face(lk, ctx, r, role, st)
	}
	e.CheckIndicator(lk, ctx, r, st, true)
	e.RadioIndicator(lk, ctx, r, st, true)
	e.MenuHighlight(lk, ctx, r, st.Focused())
	e.MenuTextColor(lk, st.Hovered())
}

// winStateless paints the controls that take no state.
func winStateless(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect) {
	e := lk.Engine()
	lk.DrawPanel(ctx, r, true)
	lk.DrawPanel(ctx, r, false)
	lk.DrawLabel(ctx, r, "Label", lk.Palette().Accent, AlignStart)
	lk.DrawFocusRing(ctx, r)
	lk.DrawOverlay(ctx, r)
	lk.DrawMenuBar(ctx, r)
	lk.DrawMenuFrame(ctx, r)
	lk.DrawTabBar(ctx, r)
	lk.DrawTabPane(ctx, r)
	lk.DrawWindowBackground(ctx, r)
	lk.DrawStatusBar(ctx, r, []string{"Ready.", "Ln 1, Col 1", "v1.0"})
	lk.DrawStatusBar(ctx, r, nil)
	lk.DrawToolBar(ctx, r)
	lk.DrawTitleBar(ctx, r, "Title", "subtitle")
	lk.DrawTitleBar(ctx, r, "Title", "")
	for _, ic := range []ToolIcon{IconInfo, IconWarning, IconError, IconQuestion, IconSave, IconNone} {
		lk.DrawMessageIcon(ctx, r, ic)
	}
	lk.DrawTooltip(ctx, r, "A helpful tip")
	lk.DrawSeparator(ctx, r, true)
	lk.DrawSeparator(ctx, r, false)
	lk.DrawGroupBox(ctx, r, "Group box", false)
	lk.DrawGroupBox(ctx, r, "Card", true)
	lk.DrawGroupBox(ctx, r, "", false)
	for _, ws := range []WindowState{
		{Active: true, CanClose: true},
		{Active: true, CanClose: true, CloseHot: true},
		{Active: true, CanClose: true, CloseHot: true, ClosePress: true},
		{CanClose: true},
		{Active: true},
	} {
		lk.DrawWindowFrame(ctx, r, "Dialog", ws)
	}
	lk.WindowFrameInsets()
	lk.WindowCloseRect(r)
	lk.GroupBoxInsets(true)
	lk.ToolBarInsets()
	lk.TabOutset()
	for _, kind := range []PopupKind{PopupMenu, PopupTooltip, PopupDialog} {
		lk.PopupShadow(kind)
	}
	for _, dir := range []Direction{DirUp, DirDown, DirLeft, DirRight} {
		e.Arrow(lk, ctx, r, dir, lk.Palette().Text)
	}
	e.Expander(lk, ctx, r, true, lk.Palette().Text)
	e.Expander(lk, ctx, r, false, lk.Palette().Text)
	for _, vertical := range []bool{true, false} {
		for _, st := range []ScrollState{
			{}, {Hovered: true}, {Hot: ScrollThumbPart, Hovered: true}, {Pressed: ScrollThumbPart, Hot: ScrollThumbPart},
			{Hot: ScrollInc, Hovered: true}, {Pressed: ScrollDec, Hot: ScrollDec}, {Pressed: ScrollPageDec}, {Pressed: ScrollPageInc},
			{Disabled: true},
		} {
			parts := ScrollGeometry(lk, r, vertical, 400, r.Dy(), 80, false)
			DrawScrollBarParts(lk, ctx, parts, vertical, st)
		}
	}
}

// winPaintsEverything paints every control in every state for every pack
// at 1x and 2x, including degenerate rects: no panic, no leaked Save.
func winPaintsEverything(t *testing.T, names []string) {
	t.Helper()
	for _, n := range names {
		for _, sc := range []float32{1, 2} {
			lk := winLook(t, n, sc)
			img := paintengine2d.NewImage(int(420*sc), int(200*sc))
			ctx := paintengine2d.NewContext(img)
			rects := []paintengine2d.Rect{
				paintengine2d.XYWH(4*sc, 4*sc, 120*sc, 28*sc),
				paintengine2d.XYWH(4*sc, 4*sc, 17*sc, 100*sc),
				paintengine2d.XYWH(4.5*sc, 3.25*sc, 96.5*sc, 22.75*sc),
			}
			// Degenerate rects in a few states: nothing may panic.
			tiny := []paintengine2d.Rect{{}, paintengine2d.XYWH(0, 0, 1, 1), paintengine2d.XYWH(2, 2, 3, 3),
				paintengine2d.XYWH(0, 0, 5, 40), paintengine2d.XYWH(0, 0, 40, 5)}
			few := []ControlState{StateNone, StatePressed | StateHovered, StateDisabled | StateChecked, StateChecked | StateFocused | StatePrimary}
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("%s@%gx panicked: %v", n, sc, r)
					}
				}()
				for _, r := range rects {
					for _, st := range winStates {
						winCalls(lk, ctx, r, st)
					}
					winStateless(lk, ctx, r)
				}
				for _, r := range tiny {
					for _, st := range few {
						winCalls(lk, ctx, r, st)
					}
					winStateless(lk, ctx, r)
				}
				winStateless(lk, ctx, paintengine2d.XYWH(4*sc, 4*sc, 400*sc, 180*sc))
			}()
			if ctx.SaveCount() != 0 {
				t.Fatalf("%s@%gx: %d saved states left", n, sc, ctx.SaveCount())
			}
		}
	}
}

// winPaintsInsideRect checks that every control paints strictly inside
// the rect it is given, in every state, at 1x and 2x.
func winPaintsInsideRect(t *testing.T, names []string) {
	t.Helper()
	type call struct {
		name string
		w, h float32
		fn   func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState)
	}
	calls := []call{
		{"button", 90, 30, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawButton(ctx, r, st, "Button")
		}},
		{"tool", 90, 34, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawToolButton(ctx, r, st, "Fetch", IconOpen)
		}},
		{"checkbox", 120, 22, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawCheckbox(ctx, r, st, true, "Checked")
			lk.DrawCheckbox(ctx, r, st, false, "")
		}},
		{"radio", 120, 22, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawRadio(ctx, r, st, true, "Selected")
			lk.DrawRadio(ctx, r, st, false, "")
		}},
		{"switch", 130, 26, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawSwitch(ctx, r, st, true, "On")
			lk.DrawSwitch(ctx, r, st, false, "")
		}},
		{"slider", 110, 24, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawSlider(ctx, r, st, 1)
			lk.DrawSlider(ctx, r, st, 0)
		}},
		{"progress", 110, 18, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawProgressBar(ctx, r, st, 1, false, 0)
			lk.DrawProgressBar(ctx, r, st, 0.4, false, 0)
			lk.DrawProgressBar(ctx, r, st, 0, true, 0.5)
			lk.DrawProgressBar(ctx, r, st, 0, true, 0.95)
		}},
		{"field", 150, 28, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawTextField(ctx, r, st, "Ada Lovelace", "", 3, 0, 3, true, 0, nil)
		}},
		{"textarea", 150, 60, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawTextArea(ctx, r, st, []TextLine{{Text: "A text area", Start: 0, End: 11}}, 3, 0, 3, true, 0, 0, "", nil)
		}},
		{"combo", 130, 28, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawComboBox(ctx, r, st, "Choice", st.Pressed())
		}},
		{"spinner", 20, 28, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawSpinner(ctx, r, st, true, false, false, st.Pressed())
		}},
		{"tab", 96, 32, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawTab(ctx, r, st, "Tab", st.Checked())
			lk.DrawTab(ctx, r, st, "Tab", !st.Checked())
		}},
		{"menutitle", 60, 26, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawMenuTitle(ctx, r, st, "File", 0, st.Pressed())
		}},
		{"menuitem", 200, 26, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			// Menu rows paint across the popup's padding: give them the
			// frame's room, as PopupMenu does.
			ch := MenuChromeFor(lk)
			row := paintengine2d.XYWH(r.Min.X+ch.PadL, r.Min.Y, r.Dx()-ch.PadL-ch.PadR, r.Dy())
			for _, mr := range []MenuRow{{Label: "Open", Shortcut: "Ctrl+O", Icon: IconOpen}, {Separator: true},
				{Label: "Wrap", Checked: true}, {Label: "Radio", Radio: true, Checked: true}, {Label: "More", Submenu: true}} {
				lk.DrawMenuItem(ctx, row, st, mr)
			}
		}},
		{"header", 80, 24, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawTableHeader(ctx, r, st, "Name", true, true)
			lk.DrawTableHeader(ctx, r, st, "Name", true, false)
		}},
		{"accordion", 140, 28, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawAccordionHeader(ctx, r, st, "Section", st.Checked())
		}},
		{"rows", 170, 24, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawListRow(ctx, r, st, "Row")
			lk.DrawTreeRow(ctx, r, st, false, false, 1, "Archives", false)
			lk.DrawTreeRow(ctx, r, st, true, false, 0, "Inbox", true)
			lk.DrawTableCell(ctx, r, st, "Cell", AlignEnd, nil)
			lk.DrawItemFocus(ctx, r, st)
		}},
		{"viewframe", 200, 120, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawViewFrame(ctx, r, st)
		}},
		{"bars", 300, 34, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawMenuBar(ctx, r)
			lk.DrawTabBar(ctx, r)
			lk.DrawToolBar(ctx, r)
			lk.DrawStatusBar(ctx, r, []string{"Ready.", "Ln 1"})
			lk.DrawTitleBar(ctx, r, "Title", "subtitle")
			lk.DrawSeparator(ctx, r, st.Checked())
			lk.DrawSplitter(ctx, r, st.Checked(), st)
		}},
		{"frames", 260, 150, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawGroupBox(ctx, r, "Group box", st.Checked())
			lk.DrawPanel(ctx, r, st.Checked())
			lk.DrawMenuFrame(ctx, r)
			lk.DrawTooltip(ctx, r, "Tip")
			lk.DrawFocusRing(ctx, r)
			lk.DrawTabPane(ctx, r)
			lk.DrawWindowBackground(ctx, r)
			lk.DrawWindowFrame(ctx, r, "Dialog", WindowState{Active: !st.Disabled(), CanClose: true, CloseHot: st.Hovered(), ClosePress: st.Pressed()})
		}},
		{"icons", 32, 32, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			for _, ic := range []ToolIcon{IconInfo, IconWarning, IconError, IconQuestion} {
				lk.DrawMessageIcon(ctx, r, ic)
			}
			lk.Engine().Expander(lk, ctx, r, st.Checked(), lk.Palette().Text)
		}},
		{"scrollbar", 17, 120, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			ss := ScrollState{Hot: ScrollThumbPart, Hovered: st.Hovered()}
			if st.Pressed() {
				ss = ScrollState{Pressed: ScrollDec, Hot: ScrollDec, Hovered: true}
			}
			for _, vertical := range []bool{true, false} {
				DrawScrollBarParts(lk, ctx, ScrollGeometry(lk, r, vertical, 500, r.Dy(), 100, false), vertical, ss)
			}
		}},
	}
	for _, n := range names {
		for _, sc := range []float32{1, 2} {
			lk := winLook(t, n, sc)
			for _, cl := range calls {
				for _, st := range winStates {
					img := paintengine2d.NewImage(int((cl.w+40)*sc), int((cl.h+40)*sc))
					ctx := paintengine2d.NewContext(img)
					r := paintengine2d.XYWH(20*sc, 20*sc, cl.w*sc, cl.h*sc)
					cl.fn(lk, ctx, r, st)
					if k := winOutside(img, r); k > 0 {
						t.Errorf("%s %gx %s state %#x: %d pixels painted outside the rect", n, sc, cl.name, uint32(st), k)
					}
					if ctx.SaveCount() != 0 {
						t.Errorf("%s %gx %s: %d saved states left", n, sc, cl.name, ctx.SaveCount())
					}
				}
			}
		}
	}
}

// winOutside counts non-transparent pixels outside r.
func winOutside(img *paintengine2d.Image, r paintengine2d.Rect) int {
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			px, py := float32(x)+0.5, float32(y)+0.5
			if px >= r.Min.X && px < r.Max.X && py >= r.Min.Y && py < r.Max.Y {
				continue
			}
			if _, _, _, a := img.PremulAt(x, y); a != 0 {
				n++
			}
		}
	}
	return n
}

// winCheckClose checks that the close button sits at the right of the
// caption and that WindowCloseRect covers what the button paints: a frame
// with the button and one without differ only inside the rect.
func winCheckClose(t *testing.T, n string) {
	t.Helper()
	for _, scale := range []float32{1, 2} {
		lk := winLook(t, n, scale)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Max.X > b.Max.X || cr.Min.X < b.Max.X-b.Dx()*0.3 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("%s@%gx: close rect %v not at the right of the caption of %v (top inset %v)", n, scale, cr, b, in.Top)
		}
		for _, ws := range []WindowState{{Active: true}, {Active: false}, {Active: true, CloseHot: true}} {
			with, without := ws, ws
			with.CanClose = true
			without.CloseHot = false
			a := winFrame(lk, b, with)
			z := winFrame(lk, b, without)
			x0, y0, x1, y1 := a.Width, a.Height, -1, -1
			for y := 0; y < a.Height; y++ {
				for x := 0; x < a.Width; x++ {
					if winDiffers(a, z, x, y) {
						x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
					}
				}
			}
			if x1 < 0 {
				t.Fatalf("%s@%gx %+v: no close button painted", n, scale, with)
			}
			slack := int(scale)
			if x0 < int(cr.Min.X)-slack || y0 < int(cr.Min.Y)-slack || x1 > int(cr.Max.X)+slack || y1 > int(cr.Max.Y)+slack {
				t.Fatalf("%s@%gx %+v: close button painted over (%d,%d)-(%d,%d), hit rect %v", n, scale, with, x0, y0, x1, y1, cr)
			}
		}
	}
}

func winFrame(lk *Classic, b paintengine2d.Rect, ws WindowState) *paintengine2d.Image {
	img := paintengine2d.NewImage(int(b.Max.X+b.Min.X), int(b.Max.Y+b.Min.Y))
	lk.DrawWindowFrame(paintengine2d.NewContext(img), b, "", ws)
	return img
}

func winDiffers(a, b *paintengine2d.Image, x, y int) bool {
	r0, g0, b0, a0 := a.PremulAt(x, y)
	r1, g1, b1, a1 := b.PremulAt(x, y)
	d := func(p, q uint8) bool { return int(p) > int(q)+2 || int(q) > int(p)+2 }
	return d(r0, r1) || d(g0, g1) || d(b0, b1) || d(a0, a1)
}

// winCheckHints: Windows puts OK first, left-aligns tabs and form labels.
func winCheckHints(t *testing.T, n string) {
	t.Helper()
	lk := winLook(t, n, 1)
	if LookHint(lk, HintDialogPrimaryFirst) != 1 {
		t.Fatalf("%s: Windows dialogs put the default button first", n)
	}
	if LookHint(lk, HintTabsCentered) != 0 {
		t.Fatalf("%s: Windows tabs are left-aligned", n)
	}
	if LookHint(lk, HintFormLabelsRight) != 0 {
		t.Fatalf("%s: Windows form labels are left-aligned", n)
	}
}

// winInk counts the pixels inside part that differ from the same bar
// painted with nothing hot (the shaft or the empty gutter).
func winInk(lk *Classic, view paintengine2d.Rect, p ScrollParts, st ScrollState, part paintengine2d.Rect) int {
	paint := func(st ScrollState, parts ScrollParts) *paintengine2d.Image {
		img := paintengine2d.NewImage(int(view.Max.X), int(view.Max.Y))
		DrawScrollBarParts(lk, paintengine2d.NewContext(img), parts, true, st)
		return img
	}
	bare := p
	bare.Dec, bare.Inc = paintengine2d.Rect{}, paintengine2d.Rect{}
	a, z := paint(st, p), paint(st, bare)
	n := 0
	for y := int(part.Min.Y); y < int(part.Max.Y); y++ {
		for x := int(part.Min.X); x < int(part.Max.X); x++ {
			if winDiffers(a, z, x, y) {
				n++
			}
		}
	}
	return n
}
