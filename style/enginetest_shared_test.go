package style

// Helpers shared by more than one engine's tests. They live here, with no
// build tag, because an engine that is not built must not take another
// engine's test helpers down with it: when engine_aqua.go is excluded by
// theme_engines_pick its test file goes too, and exerciseEngine was being
// used by engines that were still there.
//
// The same rule as the engines themselves — anything more than one of them
// needs belongs to all of them.

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// accHex prints a colour for failure messages.
func accHex(c paintengine2d.Color) string {
	q := func(v float32) int { return int(math.Round(float64(clamp1(v)) * 255)) }
	if c.A < 1 {
		return fmt.Sprintf("#%02x%02x%02x%02x", q(c.R), q(c.G), q(c.B), q(c.A))
	}
	return fmt.Sprintf("#%02x%02x%02x", q(c.R), q(c.G), q(c.B))
}

// accLook is the pack's look as an app that follows the desktop builds it
// while the desktop's accent is accent.
func accLook(t *testing.T, pack string, accent paintengine2d.Color, scale float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(pack)
	if !ok {
		t.Fatalf("pack %q not registered", pack)
	}
	SetDesktopAccent(accent, true)
	defer SetDesktopAccent(paintengine2d.Color{}, false)
	a := p.Appearance()
	a.FollowDesktop = true
	var lk LookAndFeel = a.Look()
	if scale != 1 {
		lk = WithScale(lk, scale)
	}
	return lk.(*Classic)
}

// exerciseEngine paints every control of lk in every interesting state into
// its own rect with a transparent margin around it, and fails when a
// paint panics or touches the margin (engines paint inside their rect).
func exerciseEngine(t *testing.T, name string, lk *Classic) {
	t.Helper()
	states := []ControlState{
		StateNone, StateHovered, StatePressed | StateHovered, StateFocused, StateDisabled,
		StatePrimary, StatePrimary | StateFocused, StatePrimary | StatePressed, StateToggle, StateToggle | StateChecked,
	}
	rows := []MenuRow{
		{Label: "Open…", Shortcut: "Ctrl+O", Icon: IconOpen, Underline: 0},
		{Separator: true},
		{Label: "Word wrap", Checked: true},
		{Label: "Compact", Radio: true, Checked: true},
		{Label: "Recent", Submenu: true},
	}
	lines := []TextLine{{Text: "A text area", Start: 0, End: 11}, {Text: "two lines", Start: 12, End: 21}}
	var cells []engineCell
	add := func(n string, w, h float32, f func(ctx *paintengine2d.Context, b paintengine2d.Rect)) {
		cells = append(cells, engineCell{n, w, h, f, strings.HasPrefix(n, "menu item")})
	}
	for _, st := range states {
		st := st
		sn := fmt.Sprintf("%#x", uint32(st))
		add("button "+sn, 76, 28, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawButton(ctx, b, st, ButtonDraw{Label: "Button"})
		})
		add("tool "+sn, 40, 34, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawToolButton(ctx, b, st, "", IconSave) })
		add("tool label "+sn, 90, 34, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawToolButton(ctx, b, st, "Fetch", IconOpen)
		})
		for _, on := range []bool{false, true} {
			on := on
			add(fmt.Sprintf("check %s %v", sn, on), 108, 22, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawCheckbox(ctx, b, st, on, "Check") })
			add(fmt.Sprintf("radio %s %v", sn, on), 108, 22, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawRadio(ctx, b, st, on, "Radio") })
			add(fmt.Sprintf("switch %s %v", sn, on), 120, 26, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawSwitch(ctx, b, st, on, "Switch") })
			add(fmt.Sprintf("combo %s %v", sn, on), 130, 28, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawComboBox(ctx, b, st, "Choice", on) })
			add(fmt.Sprintf("tab %s %v", sn, on), 96, 32, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawTab(ctx, b, st, "Tab", on) })
			add(fmt.Sprintf("menu title %s %v", sn, on), 52, 26, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawMenuTitle(ctx, b, st, "File", 0, on) })
			add(fmt.Sprintf("header %s %v", sn, on), 70, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawTableHeader(ctx, b, st, "Name", on, !on)
			})
			add(fmt.Sprintf("accordion %s %v", sn, on), 130, 28, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawAccordionHeader(ctx, b, st, "Section", on)
			})
		}
		add("field "+sn, 180, 28, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawTextField(ctx, b, st, "Ada Lovelace", "", 3, 0, 3, true, 0, nil)
		})
		add("area "+sn, 220, 60, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawTextArea(ctx, b, st, lines, 5, 2, 6, true, 0, 0, "", nil)
		})
		for _, t0 := range []float32{0, 0.4, 1} {
			t0 := t0
			add(fmt.Sprintf("slider %s %v", sn, t0), 104, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawSlider(ctx, b, st, t0) })
			add(fmt.Sprintf("progress %s %v", sn, t0), 100, 18, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawProgressBar(ctx, b, st, t0, false, 0) })
			add(fmt.Sprintf("busy %s %v", sn, t0), 100, 18, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawProgressBar(ctx, b, st, 0, true, t0) })
		}
		add("spinner "+sn, 20, 28, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawSpinner(ctx, b, st, st.Hovered(), !st.Hovered(), st.Pressed(), st.Focused())
		})
		for _, row := range rows {
			row := row
			add("menu item "+row.Label+" "+sn, 214, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawMenuItem(ctx, b, st, row) })
		}
		add("splitter v "+sn, 10, 80, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawSplitter(ctx, b, true, st) })
		add("splitter h "+sn, 120, 10, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawSplitter(ctx, b, false, st) })
	}
	for _, sel := range []bool{false, true} {
		for _, hov := range []bool{false, true} {
			sel, hov := sel, hov
			add(fmt.Sprintf("list row %v %v", sel, hov), 170, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawListRow(ctx, b, RowState(sel, hov), "List row")
			})
			add(fmt.Sprintf("cell %v %v", sel, hov), 60, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawTableCell(ctx, b, RowState(sel, hov), "Cell", AlignEnd, nil)
			})
			for _, leaf := range []bool{false, true} {
				leaf := leaf
				add(fmt.Sprintf("tree %v %v %v", sel, hov, leaf), 170, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
					lk.DrawTreeRow(ctx, b, RowState(sel, hov), !leaf, leaf, 2, "Archives", sel)
				})
			}
		}
	}
	for _, raised := range []bool{false, true} {
		raised := raised
		add(fmt.Sprintf("panel %v", raised), 260, 60, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawPanel(ctx, b, raised) })
		add(fmt.Sprintf("group %v", raised), 260, 150, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawGroupBox(ctx, b, "Group box", raised) })
		add(fmt.Sprintf("group untitled %v", raised), 200, 90, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawGroupBox(ctx, b, "", raised) })
	}
	for _, ws := range []WindowState{
		{Active: true, CanClose: true}, {Active: true, CanClose: true, CloseHot: true},
		{Active: true, CanClose: true, CloseHot: true, ClosePress: true}, {Active: false, CanClose: true}, {Active: true},
	} {
		ws := ws
		add(fmt.Sprintf("window %+v", ws), 270, 150, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawWindowFrame(ctx, b, "Dialog", ws) })
	}
	for _, v := range []bool{true, false} {
		for _, ss := range []ScrollState{
			{}, {Hot: ScrollThumbPart, Hovered: true}, {Pressed: ScrollThumbPart}, {Pressed: ScrollDec, Hot: ScrollDec},
			{Pressed: ScrollInc}, {Pressed: ScrollPageInc}, {Disabled: true},
		} {
			v, ss := v, ss
			th := ScrollBarStyleOf(lk).Thickness / lk.Scale()
			w, h := th, float32(120)
			if !v {
				w, h = 220, th
			}
			add(fmt.Sprintf("scroll %v %+v", v, ss), w, h, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				content := float32(400)
				if ss.Disabled {
					content = 10
				}
				view := b
				along := b.Dy()
				if !v {
					along = b.Dx()
				}
				parts := ScrollGeometry(lk, view, v, content*lk.Scale(), along, 60*lk.Scale(), false)
				DrawScrollBarParts(lk, ctx, parts, v, ss)
			})
		}
		add(fmt.Sprintf("bare scroll %v", v), 15, 120, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawScrollBar(ctx, b, paintengine2d.XYWH(b.Min.X, b.Min.Y+b.Dy()*0.3, b.Dx(), b.Dy()*0.3), StatePressed)
		})
	}
	add("tab bar", 540, 32, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawTabBar(ctx, b) })
	add("menu bar", 540, 26, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawMenuBar(ctx, b) })
	add("menu frame", 230, 190, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawMenuFrame(ctx, b) })
	add("tool bar", 570, 40, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawToolBar(ctx, b) })
	add("status bar", 570, 26, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawStatusBar(ctx, b, []string{"Ready.", "Ln 1, Col 1", "v1.0"})
	})
	add("title bar", 570, 34, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawTitleBar(ctx, b, "Title bar", "subtitle")
	})
	add("tooltip", 150, 26, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawTooltip(ctx, b, "A helpful tip") })
	add("focus", 120, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawFocusRing(ctx, b) })
	add("overlay", 180, 40, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawOverlay(ctx, b) })
	add("label", 180, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawLabel(ctx, b, "Label", lk.Palette().Accent, AlignStart)
	})
	add("separator v", 10, 80, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawSeparator(ctx, b, true) })
	add("separator h", 120, 10, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawSeparator(ctx, b, false) })
	for _, ic := range []ToolIcon{IconInfo, IconWarning, IconError, IconQuestion} {
		ic := ic
		add(fmt.Sprintf("message icon %d", ic), 32, 32, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawMessageIcon(ctx, b, ic) })
	}

	sc := lk.Scale()
	ch := MenuChromeFor(lk)
	m := 6 * sc
	if ch.PadL+2 > m || ch.PadR+2 > m {
		m = ch.PadL + ch.PadR + 2
	}
	for _, c := range cells {
		w, h := c.w*sc, c.h*sc
		img := paintengine2d.NewImage(int(w+2*m), int(h+2*m))
		ctx := paintengine2d.NewContext(img)
		b := paintengine2d.XYWH(m, m, w, h)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s: %s panicked: %v", name, c.name, r)
				}
			}()
			c.draw(ctx, b)
		}()
		if ctx.SaveCount() != 0 {
			t.Fatalf("%s: %s left %d saved states", name, c.name, ctx.SaveCount())
		}
		allow := b
		if c.menu {
			// The toolkit lays menu rows inside the popup's padding and
			// lets highlights and separators reach the frame.
			allow = paintengine2d.XYWH(b.Min.X-ch.PadL, b.Min.Y, b.Dx()+ch.PadL+ch.PadR, b.Dy())
		}
		if x, y, ok := outsideRect(img, allow); ok {
			t.Errorf("%s: %s painted outside its rect at (%d,%d) of %v", name, c.name, x, y, b)
		}
	}
}

// outsideRect finds a painted pixel outside b.
func outsideRect(img *paintengine2d.Image, b paintengine2d.Rect) (int, int, bool) {
	x0, y0 := int(b.Min.X), int(b.Min.Y)
	x1, y1 := int(b.Max.X+0.999), int(b.Max.Y+0.999)
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			if x >= x0 && x < x1 && y >= y0 && y < y1 {
				continue
			}
			if _, _, _, a := img.PremulAt(x, y); a > 2 {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// kdeClose reports whether a and b are within tol (0–255) per channel.
func kdeClose(a, b paintengine2d.Color, tol int) bool {
	ar, ag, ab, _ := a.Premul8()
	br, bg, bb, _ := b.Premul8()
	d := func(p, q uint8) bool { return int(p)-int(q) <= tol && int(q)-int(p) <= tol }
	return d(ar, br) && d(ag, bg) && d(ab, bb)
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
		if x, y, ok := outsideRect(img, r); ok {
			t.Errorf("%s: row state %#x painted outside at (%d,%d)", name, uint32(st), x, y)
		}
	}
}

// engineCell is one control painted into a w×h rect (1× design pixels).
// Menu rows may also paint into the popup's side padding (padX).
type engineCell struct {
	name string
	w, h float32
	draw func(ctx *paintengine2d.Context, b paintengine2d.Rect)
	menu bool
}

// kdeExtras paints what exerciseEngine does not reach — the current item's
// focus mark, item views' frames, rows in every item state, the tab pane
// and the window background — and fails when one panics, leaves saved
// states or paints outside its rect.
func kdeExtras(t *testing.T, name string, lk *Classic) {
	t.Helper()
	type cell struct {
		n    string
		w, h float32
		draw func(ctx *paintengine2d.Context, b paintengine2d.Rect)
	}
	var cells []cell
	for _, st := range []ControlState{
		StateNone, StateHovered, StateFocused, StateChecked, StateChecked | StateFocused,
		StateChecked | StateInactive, StateChecked | StateFocused | StateInactive, StateChecked | StateBackdrop,
		StateChecked | StateHovered, StateDisabled, StateDisabled | StateChecked | StateFocused,
	} {
		st := st
		sn := fmt.Sprintf("%#x", uint32(st))
		cells = append(cells,
			cell{"item focus " + sn, 170, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawItemFocus(ctx, b, st) }},
			cell{"view frame " + sn, 200, 120, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawViewFrame(ctx, b, st) }},
			cell{"list row " + sn, 170, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawListRow(ctx, b, st, "List row") }},
			cell{"tree row " + sn, 170, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawTreeRow(ctx, b, st, true, false, 1, "Archives", false)
			}},
			cell{"table cell " + sn, 70, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawTableCell(ctx, b, st, "Cell", AlignEnd, nil)
			}},
		)
	}
	cells = append(cells,
		cell{"tab pane", 300, 160, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawTabPane(ctx, b) }},
		cell{"window background", 320, 240, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawWindowBackground(ctx, b) }},
	)
	sc := lk.Scale()
	m := 6 * sc
	for _, c := range cells {
		w, h := c.w*sc, c.h*sc
		img := paintengine2d.NewImage(int(w+2*m), int(h+2*m))
		ctx := paintengine2d.NewContext(img)
		b := paintengine2d.XYWH(m, m, w, h)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s: %s panicked: %v", name, c.n, r)
				}
			}()
			c.draw(ctx, b)
		}()
		if ctx.SaveCount() != 0 {
			t.Fatalf("%s: %s left %d saved states", name, c.n, ctx.SaveCount())
		}
		if x, y, ok := outsideRect(img, b); ok {
			t.Errorf("%s: %s painted outside its rect at (%d,%d) of %v", name, c.n, x, y, b)
		}
	}
}

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

var lunaStates = []ControlState{
	StateNone, StateHovered, StatePressed | StateHovered, StateFocused, StateDisabled,
	StateChecked, StatePrimary, StatePrimary | StateFocused, StateToggle,
	StateToggle | StateChecked, StateToggle | StateChecked | StateHovered,
	StateDisabled | StateChecked, StateFocused | StateHovered,
}

func macLook(t *testing.T, name string, scale float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("pack %q not found", name)
	}
	var lk LookAndFeel = p.Look()
	if scale != 1 {
		lk = WithScale(lk, scale)
	}
	return lk.(*Classic)
}

var macosPackNames = []string{"yosemite", "bigsur", "bigsur-night"}

func mustLook(t *testing.T, name string) *Classic {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("missing %s", name)
	}
	return p.Look()
}

func nxNear(img *paintengine2d.Image, x, y int, want paintengine2d.Color) bool {
	r, g, b, _ := img.PremulAt(x, y)
	wr, wg, wb, _ := want.Premul8()
	d := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	return d(r, wr) < 40 && d(g, wg) < 40 && d(b, wb) < 40
}

// retroLook is pack name at a display scale.
func retroLook(t *testing.T, name string, scale float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("missing %s", name)
	}
	return WithScale(p.Look(), scale).(*Classic)
}

// kdeCheckClose checks that the close button sits at the right of the
// title bar and that WindowCloseRect covers exactly what the button paints:
// a frame with the button and one without differ only inside the rect.
func kdeCheckClose(t *testing.T, n string) {
	t.Helper()
	for _, scale := range []float32{1, 2} {
		lk := WithScale(mustLook(t, n), scale).(*Classic)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Max.X > b.Max.X || cr.Min.X < b.Max.X-b.Dx()*0.2 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("%s@%gx: close rect %v not at the right of the title bar of %v (top inset %v)", n, scale, cr, b, in.Top)
		}
		for _, ws := range []WindowState{{Active: true}, {Active: false}} {
			with, without := ws, ws
			with.CanClose = true
			a := kdeFrame(lk, b, with)
			z := kdeFrame(lk, b, without)
			x0, y0, x1, y1 := a.Width, a.Height, -1, -1
			for y := 0; y < a.Height; y++ {
				for x := 0; x < a.Width; x++ {
					if kdeDiffers(a, z, x, y) {
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

// kdeCheckPacks checks that every pack in labels is registered once, paints
// with engine, and carries its label, year and lineage.
func kdeCheckPacks(t *testing.T, engine, lineage string, year int, labels map[string]string) {
	t.Helper()
	pos := kdePositions()
	for n, label := range labels {
		i, ok := pos[n]
		if !ok {
			t.Fatalf("pack %q not registered", n)
		}
		if i < 0 {
			t.Fatalf("pack %q listed twice", n)
		}
		p, _ := LoadTheme(n)
		if p.Tokens.Engine != engine || p.Lineage != lineage || p.Year != year || p.Label != label {
			t.Fatalf("%s: engine %q lineage %q year %d label %q", n, p.Tokens.Engine, p.Lineage, p.Year, p.Label)
		}
		if lk := p.Look(); lk.Engine().ID() != engine {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
}

func kdeDiffers(a, b *paintengine2d.Image, x, y int) bool {
	r0, g0, b0, a0 := a.PremulAt(x, y)
	r1, g1, b1, a1 := b.PremulAt(x, y)
	d := func(p, q uint8) bool { return int(p) > int(q)+2 || int(q) > int(p)+2 }
	return d(r0, r1) || d(g0, g1) || d(b0, b1) || d(a0, a1)
}

func kdeFrame(lk *Classic, b paintengine2d.Rect, ws WindowState) *paintengine2d.Image {
	img := paintengine2d.NewImage(int(b.Max.X+b.Min.X), int(b.Max.Y+b.Min.Y))
	ctx := paintengine2d.NewContext(img)
	lk.DrawWindowFrame(ctx, b, "", ws)
	return img
}

func kdePositions() map[string]int {
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		if _, dup := pos[n]; dup {
			pos[n] = -1 // flagged below
			continue
		}
		pos[n] = i
	}
	return pos
}
