package style

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var lunaPackNames = []string{"luna", "luna-olive", "luna-silver", "luna-royale", "luna-night"}

func lunaLook(t *testing.T, name string, scale float32) *Classic {
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

func TestLunaPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	prevYear := 0
	for i, n := range lunaPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		if p.Tokens.Engine != "luna" {
			t.Fatalf("%s engine %q", n, p.Tokens.Engine)
		}
		if lk := p.Look(); lk.Engine().ID() != "luna" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
		if p.Lineage != "Windows" || p.Summary == "" || p.Label == "" {
			t.Fatalf("%s metadata %q %q %q", n, p.Lineage, p.Label, p.Summary)
		}
		if p.Year < prevYear {
			t.Fatalf("%s year %d before %d", n, p.Year, prevYear)
		}
		prevYear = p.Year
		if i > 0 && pos[n] < pos[lunaPackNames[i-1]] {
			t.Fatalf("packs not ordered by year: %s before %s", n, lunaPackNames[i-1])
		}
		wantFam := ThemeLight
		if n == "luna-night" {
			wantFam = ThemeDark
		}
		if p.Tokens.Family != wantFam {
			t.Fatalf("%s family %q want %q", n, p.Tokens.Family, wantFam)
		}
	}
	// The engine packs replace the legacy era packs of the same names.
	for n, label := range map[string]string{"luna": "Luna Blue", "luna-night": "Royale Noir"} {
		if p, _ := LoadTheme(n); p.Label != label {
			t.Fatalf("%s label %q, want the engine pack %q", n, p.Label, label)
		}
	}
}

// Every scheme key exists in the base table, every value parses, and
// building each pack's colours finds every key the engine asks for.
func TestLunaSchemeTables(t *testing.T) {
	for i, sc := range lunaSchemes {
		for k, v := range sc {
			if _, ok := lunaBlue[k]; !ok {
				t.Errorf("scheme %d: key %q is not in lunaBlue", i, k)
			}
			switch {
			case strings.HasPrefix(k, "g."):
				stops := lunaParseGrad(v)
				if len(stops) != strings.Count(v, ",")+1 || len(stops) < 2 {
					t.Errorf("scheme %d: %s = %q parsed to %d stops", i, k, v, len(stops))
					continue
				}
				if stops[0].Offset != 0 || stops[len(stops)-1].Offset != 1 {
					t.Errorf("scheme %d: %s does not span 0..1", i, k)
				}
				for j := 1; j < len(stops); j++ {
					if stops[j].Offset < stops[j-1].Offset {
						t.Errorf("scheme %d: %s offsets not ascending", i, k)
					}
				}
			case strings.HasPrefix(k, "f."):
			default:
				if _, ok := ParseHexColor(v); !ok {
					t.Errorf("scheme %d: %s = %q is not a colour", i, k, v)
				}
			}
		}
	}
	before := lunaKeyMiss.Load()
	for _, n := range lunaPackNames {
		lunaColors(lunaLook(t, n, 1))
	}
	if miss := lunaKeyMiss.Load() - before; miss != 0 {
		t.Fatalf("engine asked for %d keys missing from lunaBlue", miss)
	}
}

var lunaStates = []ControlState{
	StateNone, StateHovered, StatePressed | StateHovered, StateFocused, StateDisabled,
	StateChecked, StatePrimary, StatePrimary | StateFocused, StateToggle,
	StateToggle | StateChecked, StateToggle | StateChecked | StateHovered,
	StateDisabled | StateChecked, StateFocused | StateHovered,
}

// lunaCalls paints every control of the look into r, in state st.
func lunaCalls(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
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
	for role := RoleButton; role <= RolePanel; role++ {
		e.Face(lk, ctx, r, role, st)
	}
	e.CheckIndicator(lk, ctx, r, st, true)
	e.RadioIndicator(lk, ctx, r, st, true)
	e.MenuHighlight(lk, ctx, r, st.Focused())
	e.MenuTextColor(lk, st.Hovered())
}

// lunaStateless paints the controls that take no state.
func lunaStateless(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect) {
	e := lk.Engine()
	lk.DrawPanel(ctx, r, true)
	lk.DrawPanel(ctx, r, false)
	lk.DrawLabel(ctx, r, "Label", lk.Palette().Accent, AlignStart)
	lk.DrawFocusRing(ctx, r)
	for _, sel := range []bool{false, true} {
		for _, hov := range []bool{false, true} {
			lk.DrawListRow(ctx, r, sel, hov, "Row")
			lk.DrawTableCell(ctx, r, sel, hov, "Cell", AlignStart, nil)
			lk.DrawTableCell(ctx, r, sel, hov, "12", AlignEnd, nil)
			lk.DrawTableCell(ctx, r, sel, hov, "Row", AlignCenter, nil)
			lk.DrawTreeRow(ctx, r, sel, hov, true, false, 0, "Inbox", true)
			lk.DrawTreeRow(ctx, r, sel, hov, false, true, 2, "2026", false)
		}
	}
	lk.DrawOverlay(ctx, r)
	lk.DrawMenuBar(ctx, r)
	lk.DrawMenuFrame(ctx, r)
	lk.DrawTabBar(ctx, r)
	lk.DrawStatusBar(ctx, r, []string{"Ready.", "Ln 1, Col 1", "v1.0"})
	lk.DrawStatusBar(ctx, r, nil)
	lk.DrawToolBar(ctx, r)
	lk.DrawTitleBar(ctx, r, "Title", "subtitle")
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
	for _, dir := range []Direction{DirUp, DirDown, DirLeft, DirRight} {
		e.Arrow(lk, ctx, r, dir, lk.Palette().Text)
	}
	e.Expander(lk, ctx, r, true, lk.Palette().Text)
	e.Expander(lk, ctx, r, false, lk.Palette().Text)
	for _, vertical := range []bool{true, false} {
		for _, st := range []ScrollState{
			{}, {Hot: ScrollThumbPart, Hovered: true}, {Pressed: ScrollThumbPart, Hot: ScrollThumbPart},
			{Hot: ScrollInc}, {Pressed: ScrollDec, Hot: ScrollDec}, {Pressed: ScrollPageDec}, {Pressed: ScrollPageInc},
			{Disabled: true},
		} {
			parts := ScrollGeometry(lk, r, vertical, 400, r.Dy(), 80, false)
			DrawScrollBarParts(lk, ctx, parts, vertical, st)
		}
	}
}

// Every Draw* paints every state without panicking, for every pack at 1x
// and 2x, including degenerate rects.
func TestLunaPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range lunaPackNames {
		for _, sc := range []float32{1, 2} {
			lk := lunaLook(t, n, sc)
			img := paintengine2d.NewImage(int(420*sc), int(200*sc))
			ctx := paintengine2d.NewContext(img)
			big := paintengine2d.XYWH(4*sc, 4*sc, 400*sc, 180*sc)
			for _, st := range lunaStates {
				for _, r := range []paintengine2d.Rect{
					paintengine2d.XYWH(4*sc, 4*sc, 120*sc, 28*sc),
					paintengine2d.XYWH(4*sc, 4*sc, 17*sc, 100*sc),
					paintengine2d.XYWH(4.5*sc, 3.25*sc, 96.5*sc, 22.75*sc),
				} {
					lunaCalls(lk, ctx, r, st)
				}
			}
			lunaStateless(lk, ctx, big)
			lunaStateless(lk, ctx, paintengine2d.XYWH(4*sc, 4*sc, 200*sc, 60*sc))
			for _, r := range []paintengine2d.Rect{{}, paintengine2d.XYWH(0, 0, 1, 1), paintengine2d.XYWH(2, 2, 3, 3),
				paintengine2d.XYWH(0, 0, 5, 40), paintengine2d.XYWH(0, 0, 40, 5)} {
				for _, st := range lunaStates {
					lunaCalls(lk, ctx, r, st)
				}
				lunaStateless(lk, ctx, r)
			}
		}
	}
}

// Controls paint strictly inside the rect they are given (widgets paint
// clipped to their bounds; anything outside would be lost).
func TestLunaPaintsInsideRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
		}},
		{"radio", 120, 22, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawRadio(ctx, r, st, true, "Selected")
		}},
		{"switch", 130, 26, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawSwitch(ctx, r, st, true, "On")
		}},
		{"slider", 110, 24, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawSlider(ctx, r, st, 1)
			lk.DrawSlider(ctx, r, st, 0)
		}},
		{"progress", 110, 18, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawProgressBar(ctx, r, st, 1, false, 0)
			lk.DrawProgressBar(ctx, r, st, 0, true, 0.5)
		}},
		{"field", 150, 28, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawTextField(ctx, r, st, "Ada Lovelace", "", 3, 0, 3, true, 0, nil)
		}},
		{"combo", 130, 28, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawComboBox(ctx, r, st, "Choice", st.Pressed())
		}},
		{"spinner", 20, 28, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawSpinner(ctx, r, st, true, false, false, st.Pressed())
		}},
		{"tab", 96, 32, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawTab(ctx, r, st, "Tab", st.Checked())
		}},
		{"menutitle", 60, 26, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawMenuTitle(ctx, r, st, "File", 0, st.Pressed())
		}},
		{"header", 80, 24, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawTableHeader(ctx, r, st, "Name", true, true)
		}},
		{"accordion", 140, 28, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawAccordionHeader(ctx, r, st, "Section", st.Checked())
		}},
		{"rows", 170, 24, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawListRow(ctx, r, st.Checked(), st.Hovered(), "Row")
			lk.DrawTreeRow(ctx, r, st.Checked(), st.Hovered(), false, false, 1, "Archives", false)
			lk.DrawTableCell(ctx, r, st.Checked(), st.Hovered(), "Cell", AlignEnd, nil)
		}},
		{"bars", 300, 34, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawMenuBar(ctx, r)
			lk.DrawTabBar(ctx, r)
			lk.DrawToolBar(ctx, r)
			lk.DrawStatusBar(ctx, r, []string{"Ready.", "Ln 1"})
			lk.DrawTitleBar(ctx, r, "Title", "subtitle")
			lk.DrawSeparator(ctx, r, st.Checked())
		}},
		{"frames", 260, 150, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			lk.DrawGroupBox(ctx, r, "Group box", st.Checked())
			lk.DrawPanel(ctx, r, st.Checked())
			lk.DrawMenuFrame(ctx, r)
			lk.DrawTooltip(ctx, r, "Tip")
			lk.DrawFocusRing(ctx, r)
			lk.DrawWindowFrame(ctx, r, "Dialog", WindowState{Active: !st.Disabled(), CanClose: true, CloseHot: st.Hovered(), ClosePress: st.Pressed()})
		}},
		{"icons", 32, 32, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			for _, ic := range []ToolIcon{IconInfo, IconWarning, IconError, IconQuestion} {
				lk.DrawMessageIcon(ctx, r, ic)
			}
		}},
		{"scrollbar", 17, 120, func(lk *Classic, ctx *paintengine2d.Context, r paintengine2d.Rect, st ControlState) {
			ss := ScrollState{Hot: ScrollThumbPart, Hovered: st.Hovered()}
			if st.Pressed() {
				ss = ScrollState{Pressed: ScrollDec, Hot: ScrollDec}
			}
			DrawScrollBarParts(lk, ctx, ScrollGeometry(lk, r, true, 500, r.Dy(), 100, false), true, ss)
		}},
	}
	for _, n := range lunaPackNames {
		for _, sc := range []float32{1, 2} {
			lk := lunaLook(t, n, sc)
			for _, cl := range calls {
				for _, st := range lunaStates {
					img := paintengine2d.NewImage(int((cl.w+40)*sc), int((cl.h+40)*sc))
					ctx := paintengine2d.NewContext(img)
					r := paintengine2d.XYWH(20*sc, 20*sc, cl.w*sc, cl.h*sc)
					cl.fn(lk, ctx, r, st)
					if n := lunaOutside(img, r); n > 0 {
						t.Errorf("%s %gx %s state %b: %d pixels painted outside the rect", lk.Pack(), sc, cl.name, st, n)
					}
				}
			}
		}
	}
}

// lunaOutside counts non-transparent pixels outside r.
func lunaOutside(img *paintengine2d.Image, r paintengine2d.Rect) int {
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

// Labels read on every fill the engine paints them on.
func TestLunaLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range lunaPackNames {
		lk := lunaLook(t, n, 1)
		c := lunaColors(lk)
		e := lk.Engine()
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on face", c.text, c.face, 4.5},
			{"text on field", lk.fieldText(), c.field, 4.5},
			{"selected text", c.selText, c.sel, 3},
			{"hot menu text", e.MenuTextColor(lk, true), c.hotFill, 4.5},
			{"button label", c.text, c.btnFace[len(c.btnFace)/2].Color, 4.5},
			{"tooltip", c.infoText, c.info, 4.5},
			{"tab label", c.text, c.tabFace[len(c.tabFace)/2].Color, 4.5},
			{"header label", c.text, c.hdrFace, 4.5},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// Menus: Office 2003 on the XP schemes, Office XP (the owner's pale blue
// hot-track) on Royale; the palette carries the colours the engine paints.
func TestLunaMenuHotTrack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for n, want := range map[string][2]string{
		"luna":        {"#ffeec2", "#000080"},
		"luna-olive":  {"#ffeec2", "#3f5d38"},
		"luna-silver": {"#ffeec2", "#4b4b6f"},
		"luna-royale": {"#c2cfe5", "#335ea8"},
	} {
		lk := lunaLook(t, n, 1)
		c := lunaColors(lk)
		if colorHexPadded(c.hotFill) != want[0] || colorHexPadded(c.hotBorder) != want[1] {
			t.Errorf("%s hot-track %s / %s, want %s / %s", n, colorHexPadded(c.hotFill), colorHexPadded(c.hotBorder), want[0], want[1])
		}
		p := lk.Palette()
		if !nearColor(p.MenuHover, c.hotFill) || !nearColor(p.MenuHoverBorder, c.hotBorder) {
			t.Errorf("%s palette menu colours drift from the engine's", n)
		}
	}
}

func TestLunaStyleHintAndCloseRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, sc := range []float32{1, 2} {
		lk := lunaLook(t, "luna", sc)
		if LookHint(lk, HintDialogPrimaryFirst) != 1 {
			t.Fatal("Windows dialogs put the default button first")
		}
		if LookHint(lk, HintTabsCentered) != 0 {
			t.Fatal("XP tabs are left aligned")
		}
		b := paintengine2d.XYWH(10*sc, 10*sc, 270*sc, 150*sc)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Max.X || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("%gx close %+v outside the caption of %+v (top %v)", sc, cr, b, in.Top)
		}
		if cr.Dx() != cr.Dy() {
			t.Fatalf("close button %vx%v is not square", cr.Dx(), cr.Dy())
		}
		// The button is painted where hit-testing looks for it.
		img := paintengine2d.NewImage(int(300*sc), int(180*sc))
		ctx := paintengine2d.NewContext(img)
		lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
		x, y := int((cr.Min.X+cr.Max.X)*0.5-cr.Dx()*0.3), int((cr.Min.Y+cr.Max.Y)*0.5)
		r, g, bl, _ := img.PremulAt(x, y)
		if int(r) < int(g)+60 || int(r) < int(bl)+60 {
			t.Fatalf("%gx close button centre-left (%d,%d) is %d,%d,%d, not red", sc, x, y, r, g, bl)
		}
	}
}
