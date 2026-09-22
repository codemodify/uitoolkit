package widgets

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// mdiHarness is an area of three windows in a fake window.
func mdiHarness(t *testing.T) (*MDIArea, *lookHost, []*MDIWindow, []*TextField) {
	t.Helper()
	hs := &lookHost{lk: style.LightLook()}
	a := NewMDIArea()
	a.SetHost(hs)
	a.Arrange(paintengine2d.XYWH(0, 0, 800, 600))
	var wins []*MDIWindow
	var fields []*TextField
	for _, name := range []string{"One", "Two", "Three"} {
		f := NewTextField(name, "first", nil)
		fields = append(fields, f)
		wins = append(wins, a.AddWindow(name, NewColumn(f, NewTextField("", "second", nil)).WithPad(8)))
	}
	a.Arrange(paintengine2d.XYWH(0, 0, 800, 600))
	return a, hs, wins, fields
}

// areaPoint is local point p of window w in the area's coordinates.
func areaPoint(w *MDIWindow, p paintengine2d.Point) paintengine2d.Point { return p.Add(w.Bounds().Min) }

func TestMDIActivationAndStacking(t *testing.T) {
	a, hs, wins, fields := mdiHarness(t)
	if a.ActiveWindow() != wins[2] || hs.Focus() != fields[2] {
		t.Fatalf("the newest window is active with the keyboard: %v", a.ActiveWindow().Title())
	}
	if st := a.StackingOrder(); st[len(st)-1] != wins[2] {
		t.Fatal("the active window is on top")
	}
	// Cascaded: each one a caption's step from the last.
	g0, g1 := wins[0].Geometry(), wins[1].Geometry()
	if g1.Min.X <= g0.Min.X || g1.Min.Y <= g0.Min.Y {
		t.Fatalf("not cascaded: %v %v", g0, g1)
	}
	var activated []string
	a.OnActivate = func(w *MDIWindow) { activated = append(activated, w.Title()) }
	a.Activate(wins[0])
	if st := a.StackingOrder(); st[len(st)-1] != wins[0] || hs.Focus() != fields[0] {
		t.Fatal("activate raises and focuses")
	}
	// The focus moving into another window activates it, and each window
	// remembers where the keyboard was in it.
	second := wins[1].Content().Children()[1]
	hs.RequestFocus(second)
	a.FocusMoved(second)
	if a.ActiveWindow() != wins[1] {
		t.Fatal("focus in a window activates it")
	}
	a.Activate(wins[0])
	a.Activate(wins[1])
	if hs.Focus() != second {
		t.Fatal("the window's last focus comes back")
	}
	if len(activated) != 4 {
		t.Fatalf("OnActivate %v", activated)
	}
	// Clicking a window's caption activates and raises it.
	cap := wins[2].caption()
	a.Arrange(a.Bounds())
	wins[2].MousePress(widget.MouseEvent{Pos: cap.Min.Add(paintengine2d.Pt(40, 4)), Button: platform.ButtonLeft})
	wins[2].MouseRelease(widget.MouseEvent{Pos: cap.Min.Add(paintengine2d.Pt(40, 4)), Button: platform.ButtonLeft})
	if a.ActiveWindow() != wins[2] {
		t.Fatal("caption click")
	}
}

func TestMDIMoveAndResize(t *testing.T) {
	a, _, wins, _ := mdiHarness(t)
	w := wins[2]
	g := w.normal
	// Drag the caption 100, 50.
	cap := w.caption()
	p0 := cap.Min.Add(paintengine2d.Pt(60, cap.Dy()*0.5))
	w.lastClick = time.Time{}
	w.MousePress(widget.MouseEvent{Pos: p0, Button: platform.ButtonLeft})
	// Moves arrive in the window's own coordinates, which travel with it.
	target := areaPoint(w, p0).Add(paintengine2d.Pt(100, 50))
	w.MouseMove(widget.MouseEvent{Pos: target.Sub(w.Bounds().Min), Button: platform.ButtonLeft})
	a.Arrange(a.Bounds())
	w.MouseRelease(widget.MouseEvent{Pos: target.Sub(w.Bounds().Min), Button: platform.ButtonLeft})
	if d := w.normal.Min.Sub(g.Min); d.X != 100 || d.Y != 50 {
		t.Fatalf("moved by %v", d)
	}
	// A move cannot lose the caption: dragged far up it stops at the top.
	w.lastClick = time.Time{}
	w.MousePress(widget.MouseEvent{Pos: p0, Button: platform.ButtonLeft})
	w.MouseMove(widget.MouseEvent{Pos: p0.Add(paintengine2d.Pt(0, -2000)), Button: platform.ButtonLeft})
	w.MouseRelease(widget.MouseEvent{Pos: p0, Button: platform.ButtonLeft})
	if w.normal.Min.Y != 0 {
		t.Fatalf("dragged off the top: %v", w.normal)
	}
	a.Arrange(a.Bounds())
	// Resize by the bottom-right corner.
	wr := w.win()
	corner := wr.Max.Sub(paintengine2d.Pt(2, 2))
	if e := w.edgesAt(corner); e != style.EdgeBottom|style.EdgeRight {
		t.Fatalf("corner edges %v", e)
	}
	if w.CursorAt(corner) != platform.CursorResizeSE {
		t.Fatal("corner cursor")
	}
	if hit := w.HitTest(corner); hit != widget.Component(w) {
		t.Fatalf("the corner is the frame's, got %T", hit)
	}
	size := w.normal.Size()
	w.MousePress(widget.MouseEvent{Pos: corner, Button: platform.ButtonLeft})
	w.MouseMove(widget.MouseEvent{Pos: corner.Add(paintengine2d.Pt(40, 30)), Button: platform.ButtonLeft})
	w.MouseRelease(widget.MouseEvent{Pos: corner, Button: platform.ButtonLeft})
	if got := w.normal.Size(); got.X != size.X+40 || got.Y != size.Y+30 {
		t.Fatalf("resized to %v from %v", got, size)
	}
	// Never smaller than its least size.
	a.Arrange(a.Bounds())
	corner = w.win().Max.Sub(paintengine2d.Pt(2, 2))
	w.MousePress(widget.MouseEvent{Pos: corner, Button: platform.ButtonLeft})
	w.MouseMove(widget.MouseEvent{Pos: corner.Sub(paintengine2d.Pt(2000, 2000)), Button: platform.ButtonLeft})
	w.MouseRelease(widget.MouseEvent{Pos: corner, Button: platform.ButtonLeft})
	if w.normal.Dx() < w.minW() || w.normal.Dy() < w.minH() {
		t.Fatalf("shrunk to %v", w.normal)
	}
	// Geometry round-trips in logical pixels.
	w.SetGeometry(paintengine2d.XYWH(10, 20, 300, 200))
	if g := w.Geometry(); g != paintengine2d.XYWH(10, 20, 300, 200) {
		t.Fatalf("geometry %v", g)
	}
}

func TestMDIMinimizeMaximize(t *testing.T) {
	a, _, wins, _ := mdiHarness(t)
	w := wins[1]
	a.Activate(w)
	w.Maximize()
	a.Arrange(a.Bounds())
	if !w.IsMaximized() || w.Bounds() != a.LocalBounds() {
		t.Fatalf("maximised to %v, area %v", w.Bounds(), a.LocalBounds())
	}
	// Activating another window hands the maximised state on.
	a.Activate(wins[0])
	if !wins[0].IsMaximized() || w.IsMaximized() {
		t.Fatal("the maximised state should follow the active window")
	}
	wins[0].Restore()
	if wins[0].IsMaximized() {
		t.Fatal("restore")
	}
	// The maximise button, clicked.
	a.Activate(w)
	a.Arrange(a.Bounds())
	var maxBtn paintengine2d.Rect
	for _, b := range w.buttonRects() {
		if b.k == platform.CaptionMaximize {
			maxBtn = b.r
		}
	}
	if maxBtn.Empty() {
		t.Fatal("no maximise button")
	}
	c := maxBtn.Min.Add(paintengine2d.Pt(maxBtn.Dx()*0.5, maxBtn.Dy()*0.5))
	w.MousePress(widget.MouseEvent{Pos: c, Button: platform.ButtonLeft})
	w.MouseRelease(widget.MouseEvent{Pos: c, Button: platform.ButtonLeft})
	if !w.IsMaximized() {
		t.Fatal("maximise button")
	}
	// A double click on the caption restores.
	a.Arrange(a.Bounds())
	cp := w.caption().Min.Add(paintengine2d.Pt(80, 6))
	w.lastClick = time.Time{}
	for range 2 {
		w.MousePress(widget.MouseEvent{Pos: cp, Button: platform.ButtonLeft})
		w.MouseRelease(widget.MouseEvent{Pos: cp, Button: platform.ButtonLeft})
	}
	if w.IsMaximized() {
		t.Fatal("double click restores")
	}
	// Minimised windows line up along the foot, out of the tiling.
	wins[0].Minimize()
	wins[2].Minimize()
	a.Arrange(a.Bounds())
	b0, b2 := wins[0].Bounds(), wins[2].Bounds()
	if b0.Max.Y != a.LocalBounds().Max.Y || b0.Min.Y != b2.Min.Y || b2.Min.X <= b0.Min.X {
		t.Fatalf("minimised at %v and %v", b0, b2)
	}
	if wins[0].client.Visible() {
		t.Fatal("a minimised window shows its content")
	}
	a.Tile()
	a.Arrange(a.Bounds())
	if !wins[0].IsMinimized() || w.normal.Max.Y > a.work().Max.Y {
		t.Fatal("tiling keeps minimised windows minimised and above their row")
	}
	wins[0].Restore()
	if wins[0].IsMinimized() {
		t.Fatal("restore from minimised")
	}
}

func TestMDITileAndCascade(t *testing.T) {
	a, _, wins, _ := mdiHarness(t)
	wins = append(wins, a.AddWindow("Four", NewLabel("4")))
	a.Tile()
	a.Arrange(a.Bounds())
	// Four windows tile two by two, filling the area without overlap.
	var total float32
	for i, w := range wins {
		total += w.normal.Dx() * w.normal.Dy()
		for j := i + 1; j < len(wins); j++ {
			if !w.normal.Intersect(wins[j].normal).Empty() {
				t.Fatalf("%s and %s overlap", w.Title(), wins[j].Title())
			}
		}
	}
	if area := a.LocalBounds().Dx() * a.LocalBounds().Dy(); total < area*0.99 {
		t.Fatalf("tiles cover %.0f of %.0f", total, area)
	}
	if wins[0].normal.Dx() != 400 || wins[0].normal.Dy() != 300 {
		t.Fatalf("tile %v", wins[0].normal)
	}
	a.TileColumns()
	if wins[1].normal.Dy() != 600 || wins[1].normal.Dx() != 200 {
		t.Fatalf("columns %v", wins[1].normal)
	}
	a.TileRows()
	if wins[1].normal.Dx() != 800 || wins[1].normal.Dy() != 150 {
		t.Fatalf("rows %v", wins[1].normal)
	}
	a.Activate(wins[1])
	a.Cascade()
	// In the order they were opened, the active one moved to the front.
	order := []*MDIWindow{wins[0], wins[2], wins[3], wins[1]}
	for i := 1; i < len(order); i++ {
		d := order[i].normal.Min.Sub(order[i-1].normal.Min)
		if d.X <= 0 || d.X != d.Y {
			t.Fatalf("cascade step %v", d)
		}
	}
	if st := a.StackingOrder(); st[len(st)-1] != wins[1] {
		t.Fatal("the active window stays on top of a cascade")
	}
}

func TestMDIKeyboard(t *testing.T) {
	a, hs, wins, fields := mdiHarness(t)
	// Keys bubble from the focused field up to the area.
	bubble := func(e widget.KeyEvent) bool {
		for c := hs.Focus(); c != nil; c = c.Parent() {
			if c.KeyPress(e) {
				return true
			}
		}
		return false
	}
	bubble(widget.KeyEvent{Key: platform.KeyTab, Mods: platform.ModCtrl})
	if a.ActiveWindow() != wins[0] || hs.Focus() != fields[0] {
		t.Fatalf("Ctrl+Tab wraps to the first window: %s", a.ActiveWindow().Title())
	}
	bubble(widget.KeyEvent{Key: platform.KeyF6, Mods: platform.ModCtrl})
	if a.ActiveWindow() != wins[1] {
		t.Fatal("Ctrl+F6")
	}
	bubble(widget.KeyEvent{Key: platform.KeyTab, Mods: platform.ModCtrl | platform.ModShift})
	if a.ActiveWindow() != wins[0] {
		t.Fatal("Ctrl+Shift+Tab")
	}
	// Ctrl+F4 asks the window, which may refuse.
	asked := 0
	wins[0].OnCloseRequest = func() bool { asked++; return asked > 1 }
	bubble(widget.KeyEvent{Key: platform.KeyF4, Mods: platform.ModCtrl})
	if len(a.Windows()) != 3 || asked != 1 {
		t.Fatal("a refused close kept the window")
	}
	closed := false
	wins[0].OnClose = func() { closed = true }
	bubble(widget.KeyEvent{Key: platform.KeyW, Rune: 'w', Mods: platform.ModCtrl})
	if len(a.Windows()) != 2 || !closed {
		t.Fatal("Ctrl+W closed nothing")
	}
	if a.ActiveWindow() == nil || !widget.FocusWithin(a.ActiveWindow()) {
		t.Fatal("closing hands the keyboard to the next window")
	}
	// A keyboard move: the arrows move, Escape puts it back, Return keeps.
	w := a.ActiveWindow()
	g := w.normal
	a.startKeyboard(w, false)
	a.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	a.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if w.normal.Min.X != g.Min.X+10 || w.normal.Min.Y != g.Min.Y+10 {
		t.Fatalf("keyboard move %v from %v", w.normal, g)
	}
	a.KeyPress(widget.KeyEvent{Key: platform.KeyEscape})
	if w.normal != g {
		t.Fatal("Escape puts it back")
	}
	a.startKeyboard(w, true)
	a.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	a.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if w.normal.Dx() != g.Dx()+10 || a.kb != nil {
		t.Fatal("keyboard size")
	}
}

func TestMDIWindowMenuAndTabs(t *testing.T) {
	a, _, wins, _ := mdiHarness(t)
	m := a.WindowMenu()
	count := func() (n int, checked string) {
		for _, it := range m.Items {
			if it.RadioGroup == "mdi-windows" {
				n++
				if it.Checked {
					checked = it.Text
				}
			}
		}
		return
	}
	if n, c := count(); n != 3 || c != "&3 Three" {
		t.Fatalf("menu lists %d, checked %q", n, c)
	}
	wins[0].SetTitle("Renamed")
	a.Activate(wins[0])
	if n, c := count(); n != 3 || c != "&1 Renamed" {
		t.Fatalf("menu after rename: %d %q", n, c)
	}
	wins[1].Dismiss()
	if n, _ := count(); n != 2 {
		t.Fatal("menu follows a close")
	}
	// The tabbed view: a tab a window, the active one shown alone.
	a.SetViewMode(MDITabbed)
	a.Arrange(a.Bounds())
	if a.tabs == nil || a.tabs.Len() != 2 || !a.tabs.Visible() {
		t.Fatal("tab strip")
	}
	if !wins[0].Visible() || wins[2].Visible() {
		t.Fatal("only the active window shows in the tabbed view")
	}
	a.tabs.Select(1)
	a.Arrange(a.Bounds())
	if a.ActiveWindow() != wins[2] || !wins[2].Visible() {
		t.Fatal("a tab activates its window")
	}
	if !wins[2].caption().Empty() || wins[2].contentRect() != wins[2].LocalBounds() {
		t.Fatal("no frame in the tabbed view")
	}
	a.SetViewMode(MDISubWindows)
	a.Arrange(a.Bounds())
	if a.tabs.Visible() || !wins[0].Visible() {
		t.Fatal("back to windows")
	}
}

func TestMDIAccessibility(t *testing.T) {
	a, hs, wins, _ := mdiHarness(t)
	wins[0].Minimize()
	root := &a11y.Node{ID: 1, Role: a11y.RoleWindow, Name: "w", Bounds: paintengine2d.XYWH(0, 0, 800, 600)}
	col := NewColumn(a)
	col.SetHost(hs)
	a.Arrange(paintengine2d.XYWH(0, 0, 800, 600))
	widget.AccessibleTree(root, col)
	var pane *a11y.Node
	frames := 0
	root.Walk(func(n *a11y.Node) bool {
		switch n.Role {
		case a11y.RoleDesktopPane:
			pane = n
		case a11y.RoleInternalFrame:
			frames++
			names := map[string]bool{}
			for _, c := range n.Children {
				if c.Role == a11y.RoleButton {
					names[c.Name] = true
				}
			}
			if !names["Close"] {
				t.Errorf("%s has no Close button: %v", n.Name, names)
			}
			if n.Name == "One" && n.Description != "minimized" {
				t.Errorf("minimised window says %q", n.Description)
			}
		}
		return true
	})
	if pane == nil || frames != 3 {
		t.Fatalf("pane %v, %d frames", pane, frames)
	}
	for _, p := range a11y.Check(root) {
		t.Error(p)
	}
	// Pressing a caption button through assistive technology.
	for i, n := range wins[1].AccessibleItems() {
		if n.Name == "Maximize" {
			wins[1].AccessibleAction(i, a11y.ActionDefault)
		}
	}
	if !wins[1].IsMaximized() {
		t.Fatal("maximise by action")
	}
	// Every window's fields are reachable by Tab, and a minimised window
	// (whose fields are hidden) is a stop of its own, which Return restores.
	f := widget.Focusables(a)
	if len(f) != 5 || f[0] != widget.Component(wins[0]) {
		t.Fatalf("%d focusable", len(f))
	}
	hs.RequestFocus(wins[0])
	wins[0].KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	if wins[0].IsMinimized() {
		t.Fatal("Return on a minimised window restores it")
	}
}
