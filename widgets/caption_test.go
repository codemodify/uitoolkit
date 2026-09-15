package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// frameHost is a widget.FrameHost for caption widgets under test.
type frameHost struct {
	host
	state  platform.WindowState
	caps   platform.WMCaps
	active bool
	calls  []string
	menuAt paintengine2d.Point
}

func (h *frameHost) Title() string                     { return "Window" }
func (h *frameHost) WindowState() platform.WindowState { return h.state }
func (h *frameHost) FrameCaps() platform.WMCaps        { return h.caps }
func (h *frameHost) Active() bool                      { return h.active }
func (h *frameHost) Minimize()                         { h.calls = append(h.calls, "minimize") }
func (h *frameHost) ToggleMaximize()                   { h.calls = append(h.calls, "maximize") }
func (h *frameHost) RequestClose()                     { h.calls = append(h.calls, "close") }
func (h *frameHost) ShowWindowMenu(p paintengine2d.Point) {
	h.calls = append(h.calls, "menu")
	h.menuAt = p
}

func newFrameHost() *frameHost {
	return &frameHost{active: true}
}

func arrange(c widget.Component, w, h float32) {
	c.Measure(layout.Tight(w, h))
	c.Arrange(paintengine2d.XYWH(0, 0, w, h))
}

func TestBuiltinCaptionAnswers(t *testing.T) {
	host := newFrameHost()
	tb := NewToolBar(ToolText("Fetch", nil), ToolDivider(), ToolText("Write", nil))
	tb.SetHost(host)
	arrange(tb, 400, 36)
	item := tb.ItemRect(0)
	if tb.CaptionAt(paintengine2d.Pt((item.Min.X+item.Max.X)/2, 18)) {
		t.Error("a tool button is a control")
	}
	div := tb.ItemRect(1)
	if !tb.CaptionAt(paintengine2d.Pt((div.Min.X+div.Max.X)/2, 18)) {
		t.Error("a divider is caption")
	}
	if !tb.CaptionAt(paintengine2d.Pt(390, 18)) {
		t.Error("the tool bar's empty end is caption")
	}

	tabs := NewTabBar("One", "Two")
	tabs.SetHost(host)
	arrange(tabs, 400, 30)
	if tabs.CaptionAt(paintengine2d.Pt(20, 15)) || !tabs.CaptionAt(paintengine2d.Pt(390, 15)) {
		t.Error("tab bar: a tab is a control, the space after the last is caption")
	}

	mb := NewMenuBar(NewMenu("&File"), NewMenu("&Edit"))
	mb.SetHost(host)
	arrange(mb, 400, 28)
	if mb.CaptionAt(mb.TitleRect(0).Min.Add(paintengine2d.Pt(4, 4))) || !mb.CaptionAt(paintengine2d.Pt(390, 14)) {
		t.Error("menu bar: a title is a control, the space after is caption")
	}

	for _, c := range []widget.CaptionHitTester{NewSpacer(), NewLabel("x"), NewSeparator(), NewRow(), NewStack(), NewPad(1, nil), NewTitleBar("t", "")} {
		if !c.CaptionAt(paintengine2d.Pt(1, 1)) {
			t.Errorf("%T is caption", c)
		}
	}
	if _, ok := interface{}(NewButton("x", nil)).(widget.CaptionHitTester); ok {
		t.Error("a button must not be caption")
	}
}

// Every component from the hit one up to the title bar must be caption: a
// label inside a button is not.
func TestIsCaptionWalksEveryAncestor(t *testing.T) {
	host := newFrameHost()
	inner := NewLabel("in a row")
	btnLabel := NewLabel("in a button")
	btn := NewButton("", nil)
	btn.Add(btnLabel)
	hb := NewHeaderBar([]widget.Component{NewRow(inner)}, nil, []widget.Component{btn})
	hb.SetHost(host)
	arrange(hb, 500, 36)
	pt := func(c widget.Component) paintengine2d.Point {
		b := widget.DeviceBounds(c)
		return paintengine2d.Pt(b.Min.X+1, b.Min.Y+1)
	}
	if !widget.IsCaption(inner, hb, pt(inner)) {
		t.Error("a label in a row in the header bar is caption")
	}
	if widget.IsCaption(btnLabel, hb, pt(btnLabel)) {
		t.Error("a label inside a button is not caption")
	}
	if widget.IsCaption(btn, hb, pt(btn)) {
		t.Error("a button is not caption")
	}
	other := NewLabel("elsewhere")
	if widget.IsCaption(other, hb, paintengine2d.Pt(1, 1)) {
		t.Error("a component outside the title bar is not caption")
	}
}

func TestDragAreaAndNoDrag(t *testing.T) {
	host := newFrameHost()
	btn := NewButton("Logo", nil)
	drag := DragArea(btn)
	drag.SetHost(host)
	arrange(drag, 100, 30)
	if hit := drag.HitTest(paintengine2d.Pt(50, 15)); hit != widget.Component(drag) {
		t.Fatalf("a drag area takes the hit itself, got %T", hit)
	}
	if !drag.CaptionAt(paintengine2d.Pt(50, 15)) || !drag.Drag() {
		t.Error("drag area")
	}
	stop := NoDrag(NewLabel("status"))
	stop.SetHost(host)
	arrange(stop, 100, 30)
	if hit := stop.HitTest(paintengine2d.Pt(50, 15)); hit == widget.Component(stop) {
		t.Error("a no-drag region hit-tests into its content")
	}
	if stop.CaptionAt(paintengine2d.Pt(50, 15)) {
		t.Error("no-drag is never caption")
	}
}

func TestWindowControls(t *testing.T) {
	host := newFrameHost()
	c := NewWindowControls(platform.CaptionSpacer, platform.CaptionMinimize, platform.CaptionMaximize, platform.CaptionSpacer, platform.CaptionClose)
	c.SetHost(host)
	sz := c.Measure(layout.Unbounded())
	arrange(c, sz.X, 36)
	if got := c.Shown(); len(got) != 4 || got[0] != platform.CaptionMinimize || got[2] != platform.CaptionSpacer {
		t.Fatalf("shown %v (leading spacer dropped, inner kept)", got)
	}
	if !c.CaptionAt(center(c.rects()[2])) || c.CaptionAt(center(c.ButtonRect(platform.CaptionClose))) {
		t.Error("a spacer's gap is caption, a button is not")
	}
	press := func(b platform.CaptionButton) {
		p := center(c.ButtonRect(b))
		c.MousePress(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
		c.MouseRelease(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
	}
	press(platform.CaptionMinimize)
	press(platform.CaptionMaximize)
	press(platform.CaptionClose)
	if len(host.calls) != 3 || host.calls[0] != "minimize" || host.calls[1] != "maximize" || host.calls[2] != "close" {
		t.Fatalf("calls %v", host.calls)
	}
	// Released off the button: nothing.
	p := center(c.ButtonRect(platform.CaptionClose))
	c.MousePress(widget.MouseEvent{Pos: p, Button: platform.ButtonLeft})
	c.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(-5, -5), Button: platform.ButtonLeft})
	if len(host.calls) != 3 {
		t.Fatalf("a release off the button pressed it: %v", host.calls)
	}
	// Right-click: the window menu at the pointer.
	c.MousePress(widget.MouseEvent{Pos: p, Button: platform.ButtonRight})
	if host.calls[3] != "menu" || host.menuAt != p {
		t.Fatalf("right click %v at %v", host.calls, host.menuAt)
	}
	// Buttons for what the desktop cannot do are hidden.
	host.caps = platform.CapKnown | platform.CapWindowMenu
	if got := c.Shown(); len(got) != 1 || got[0] != platform.CaptionClose {
		t.Fatalf("without minimize and maximize %v", got)
	}
	host.caps = 0
	// Names for assistive technology; maximized means Restore.
	names := func() []string {
		var out []string
		for _, n := range c.AccessibleItems() {
			if n.Role != a11y.RoleButton || !n.Actions.Has(a11y.ActionDefault) {
				t.Errorf("item %+v", n)
			}
			out = append(out, n.Name)
		}
		return out
	}
	if got := names(); len(got) != 3 || got[0] != "Minimize" || got[1] != "Maximize" || got[2] != "Close" {
		t.Fatalf("names %v", got)
	}
	host.state.Maximized = true
	if got := names(); got[1] != "Restore" {
		t.Fatalf("maximized names %v", got)
	}
	if !c.AccessibleAction(0, a11y.ActionDefault) || host.calls[len(host.calls)-1] != "minimize" {
		t.Fatal("assistive technology presses a button")
	}
	if c.WantsFocus() {
		t.Error("caption buttons stay out of the Tab order")
	}
}

func center(r paintengine2d.Rect) paintengine2d.Point {
	return paintengine2d.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// Every caption glyph paints something inside its button, and the restore
// glyph differs from the maximize one.
func TestWindowControlsPaintGlyphs(t *testing.T) {
	host := newFrameHost()
	c := NewWindowControls(platform.CaptionMenu, platform.CaptionMinimize, platform.CaptionMaximize, platform.CaptionClose)
	c.SetHost(host)
	sz := c.Measure(layout.Unbounded())
	arrange(c, sz.X, 32)
	shoot := func() *paintengine2d.Image {
		img := paintengine2d.NewImage(int(sz.X), 32)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.RGB(0, 0, 0))
		c.Paint(ctx)
		return img
	}
	img := shoot()
	for _, b := range []platform.CaptionButton{platform.CaptionMenu, platform.CaptionMinimize, platform.CaptionMaximize, platform.CaptionClose} {
		r := c.ButtonRect(b)
		lit := 0
		for y := int(r.Min.Y); y < int(r.Max.Y); y++ {
			for x := int(r.Min.X); x < int(r.Max.X); x++ {
				if px := img.NRGBAAt(x, y); px.R > 60 || px.G > 60 || px.B > 60 {
					lit++
				}
			}
		}
		if lit < 8 {
			t.Errorf("%v glyph paints %d pixels", b, lit)
		}
	}
	maxBox := c.ButtonRect(platform.CaptionMaximize)
	before := crop(img, maxBox)
	host.state.Maximized = true
	after := crop(shoot(), maxBox)
	if before == after {
		t.Error("the restore glyph looks like the maximize glyph")
	}
}

func crop(img *paintengine2d.Image, r paintengine2d.Rect) string {
	var b []byte
	for y := int(r.Min.Y); y < int(r.Max.Y); y++ {
		for x := int(r.Min.X); x < int(r.Max.X); x++ {
			px := img.NRGBAAt(x, y)
			b = append(b, px.R, px.G, px.B)
		}
	}
	return string(b)
}

// Under the desktop's frame a header bar lays its items out exactly as the
// equivalent row: apps that move their chrome row into the title bar look
// the same with a system frame.
func TestHeaderBarMatchesRowWithoutFrame(t *testing.T) {
	host := newFrameHost()
	mk := func() (a, b, c widget.Component) {
		return NewToolBar(ToolText("Fetch", nil)), NewTextField("", "filter", nil), NewToolBar(ToolText("X", nil))
	}
	a1, b1, c1 := mk()
	hb := NewHeaderBar([]widget.Component{a1}, nil, []widget.Component{b1, c1})
	hb.SetHost(host)
	a2, b2, c2 := mk()
	slot := NewSpacer()
	row := NewRow(a2, slot, b2, c2).WithGap(8).WithAlign(layout.AlignCenter)
	row.AddFlex(slot, 1)
	row.SetHost(host)
	hs := hb.Measure(layout.Loose(900, 600))
	rs := row.Measure(layout.Loose(900, 600))
	if hs.Y != rs.Y {
		t.Fatalf("heights %v %v", hs, rs)
	}
	hb.Arrange(paintengine2d.XYWH(0, 0, 900, hs.Y))
	row.Arrange(paintengine2d.XYWH(0, 0, 900, rs.Y))
	for i, pair := range [][2]widget.Component{{a1, a2}, {b1, b2}, {c1, c2}} {
		if widget.DeviceBounds(pair[0]) != widget.DeviceBounds(pair[1]) {
			t.Errorf("item %d: %v vs %v", i, widget.DeviceBounds(pair[0]), widget.DeviceBounds(pair[1]))
		}
	}
	// Framed: the caption buttons take the sides, the items move in, and
	// the caption is at least tall enough for the buttons.
	hb.SetWindowControls(platform.ParseButtonLayout("close:minimize,maximize"), true)
	hs = hb.Measure(layout.Loose(900, 600))
	hb.Arrange(paintengine2d.XYWH(0, 0, 900, hs.Y))
	left, right := hb.Controls()
	if !left.Visible() || !right.Visible() || widget.DeviceBounds(left).Min.X != 0 || widget.DeviceBounds(right).Max.X != 900 {
		t.Fatalf("controls %v %v", widget.DeviceBounds(left), widget.DeviceBounds(right))
	}
	if widget.DeviceBounds(a1).Min.X < widget.DeviceBounds(left).Max.X || widget.DeviceBounds(c1).Max.X > widget.DeviceBounds(right).Min.X {
		t.Fatal("items overlap the caption buttons")
	}
	if hs.Y < left.MinHeight() || widget.DeviceBounds(right).Dy() != hs.Y {
		t.Fatalf("caption %v, buttons full height %v", hs, widget.DeviceBounds(right))
	}
}

type titleHost struct{ frameHost }

func (titleHost) Title() string { return "Hello caption" }

// The default caption shows the window's title in its free space.
func TestHeaderBarTitle(t *testing.T) {
	h := NewHeaderBar(nil, nil, nil)
	h.ShowTitle = true
	h.SetHost(&titleHost{frameHost{active: true}})
	h.SetWindowControls(platform.ParseButtonLayout(":close"), true)
	sz := h.Measure(layout.Loose(300, 100))
	h.Arrange(paintengine2d.XYWH(0, 0, 300, sz.Y))
	img := paintengine2d.NewImage(300, int(sz.Y))
	ctx := paintengine2d.NewContext(img)
	bg := paintengine2d.RGB(0.16, 0.16, 0.16) // the dark look's window
	ctx.Clear(bg)
	h.Paint(ctx)
	r := h.titleRect()
	if r.Dy() != sz.Y || r.Dx() < 100 {
		t.Fatalf("title rect %v in a %v caption", r, sz)
	}
	lit := 0
	for y := 0; y < img.Height; y++ {
		for x := int(r.Min.X); x < int(r.Max.X); x++ {
			if p := img.NRGBAAt(x, y); p.R > 120 {
				lit++
			}
		}
	}
	if lit < 20 {
		t.Fatalf("the title painted %d pixels", lit)
	}
	if items := h.AccessibleItems(); len(items) != 1 || items[0].Name != "Hello caption" {
		t.Fatalf("title item %+v", items)
	}
}

// lookFrameHost is a frameHost in a given look.
type lookFrameHost struct {
	frameHost
	look style.LookAndFeel
}

func (h *lookFrameHost) Look() style.LookAndFeel { return h.look }

// A look that centres the title (GNOME's) centres it on the window; where
// the buttons leave no room at the centre they push it aside instead of
// cutting it short.
func TestHeaderBarCentredTitle(t *testing.T) {
	pack, ok := style.LoadTheme("adwaita-gtk3")
	if !ok {
		t.Fatal("no adwaita-gtk3 pack")
	}
	lk := pack.Look()
	h := NewHeaderBar(nil, nil, nil)
	h.ShowTitle = true
	h.SetHost(&lookFrameHost{frameHost: frameHost{active: true}, look: lk})
	h.SetWindowControls(platform.ParseButtonLayout(":minimize,maximize,close"), true)
	if !h.spec().CenterTitle {
		t.Fatal("GNOME centres its title")
	}
	// The title and the look's pads round it.
	need := lk.BoldFont().Advance("Window") + 2*style.Dip(lk, 8)
	sz := h.Measure(layout.Loose(800, 100))
	h.Arrange(paintengine2d.XYWH(0, 0, 800, sz.Y))
	buttons := 800 - h.trail.Bounds().Min.X
	// Too narrow for a box on the window's centre to hold the title.
	narrow := 2*buttons + need*0.8
	for _, w := range []float32{800, narrow} {
		sz := h.Measure(layout.Loose(w, 100))
		h.Arrange(paintengine2d.XYWH(0, 0, w, sz.Y))
		r, ctl := h.titleRect(), h.trail.Bounds()
		if r.Dx() < need || r.Min.X < 0 || r.Max.X > ctl.Min.X+0.5 {
			t.Fatalf("%v wide: title %v (needs %v) beside the buttons at %v", w, r, need, ctl)
		}
		if mid := (r.Min.X + r.Max.X) * 0.5; w == 800 && (mid < w/2-1 || mid > w/2+1) {
			t.Fatalf("title %v off the window's centre %v", r, w/2)
		}
	}
}
