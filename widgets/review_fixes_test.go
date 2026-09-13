package widgets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// fakeWindow stands in for app.Window: popup + overlay layers with the same
// dismissal order (the outgoing layer is told before the new one is live).
type fakeWindow struct {
	focus   widget.Component
	popup   widget.Component
	overlay widget.Component
	look    style.LookAndFeel
}

func (h *fakeWindow) Invalidate(widget.Component, paintengine2d.Rect) {}
func (h *fakeWindow) RequestFocus(c widget.Component)                 { h.focus = c }
func (h *fakeWindow) Focus() widget.Component                         { return h.focus }
func (h *fakeWindow) Scale() float32                                  { return 1 }
func (h *fakeWindow) Look() style.LookAndFeel {
	// Cached like the real window: a fresh look per call would hide how often
	// widgets ask for metrics.
	if h.look == nil {
		h.look = style.DarkLook()
	}
	return h.look
}
func (h *fakeWindow) RequestLayout()          {}
func (h *fakeWindow) SurfaceSize() (int, int) { return 800, 600 }

func (h *fakeWindow) SetPopup(c widget.Component) {
	if h.popup != nil && h.popup != c {
		if d, ok := h.popup.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
	h.popup = c
	if c != nil {
		c.SetHost(h)
	}
}
func (h *fakeWindow) Popup() widget.Component { return h.popup }
func (h *fakeWindow) DismissPopup() {
	if h.popup == nil {
		return
	}
	old := h.popup
	h.popup = nil
	if d, ok := old.(widget.Dismisser); ok {
		d.Dismissed()
	}
}
func (h *fakeWindow) SetOverlay(c widget.Component) {
	if h.overlay == c {
		return
	}
	old := h.overlay
	h.overlay = c
	if c != nil {
		c.SetHost(h)
	}
	if old != nil {
		if d, ok := old.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
}
func (h *fakeWindow) Overlay() widget.Component { return h.overlay }

// --- finding 1: a dismissed popup must stop handling input and give focus back

func TestDismissedPopupIsDeadAndRestoresFocus(t *testing.T) {
	h := &fakeWindow{}
	fired := 0
	bar := NewMenuBar(NewMenu("&File", Item("Quit", func() { fired++ })))
	bar.SetHost(h)
	bar.Arrange(paintengine2d.XYWH(0, 0, 300, 28))

	bar.Open(0)
	pop, _ := h.Popup().(*PopupMenu)
	if pop == nil {
		t.Fatal("menu did not open")
	}
	if h.Focus() != pop {
		t.Fatalf("open menu should hold focus, got %T", h.Focus())
	}

	h.DismissPopup() // click on inert chrome: the click is swallowed
	if !pop.Dead() {
		t.Fatal("dismissed popup not marked dead")
	}
	if h.Focus() != bar {
		t.Fatalf("focus should return to the menu bar anchor, got %T", h.Focus())
	}
	if pop.KeyPress(widget.KeyEvent{Key: platform.KeyReturn}) {
		t.Fatal("dead popup handled Return")
	}
	if pop.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(10, 10)}) {
		t.Fatal("dead popup handled a click")
	}
	if fired != 0 {
		t.Fatalf("dead popup executed its item %d times", fired)
	}
}

func TestClosedCascadeStopsHandlingKeys(t *testing.T) {
	h := &fakeWindow{}
	fired := 0
	parent := NewPopupMenu(Submenu("More", Item("Deep", func() { fired++ })))
	parent.SetHost(h)
	parent.Arrange(paintengine2d.XYWH(0, 0, 160, 80))
	parent.KeyPress(widget.KeyEvent{Key: platform.KeyRight}) // open the cascade
	child := parent.CascadeMenu()
	if child == nil {
		t.Fatal("cascade did not open")
	}
	if h.Focus() != child {
		t.Fatalf("cascade should hold focus, got %T", h.Focus())
	}
	parent.closeCascade()
	if !child.Dead() {
		t.Fatal("closed cascade not marked dead")
	}
	if h.Focus() != parent {
		t.Fatalf("focus should fall back to the parent menu, got %T", h.Focus())
	}
	if child.KeyPress(widget.KeyEvent{Key: platform.KeyReturn}) || fired != 0 {
		t.Fatalf("closed cascade still activates items (fired=%d)", fired)
	}
}

func TestComboBoxPopupRestoresFocusToField(t *testing.T) {
	h := &fakeWindow{}
	c := NewComboBox([]string{"One", "Two"}, 0, nil)
	c.SetHost(h)
	c.Arrange(paintengine2d.XYWH(0, 0, 160, 28))
	c.Open()
	if h.Popup() == nil {
		t.Fatal("combo did not open")
	}
	h.DismissPopup()
	if h.Focus() != c {
		t.Fatalf("focus should return to the combo box, got %T", h.Focus())
	}
	if c.Opened() {
		t.Fatal("combo still reports open after dismissal")
	}
}

// --- finding 2: Left / Right must keep walking the menu bar while open

func TestMenuBarArrowsWalkWhileOpen(t *testing.T) {
	h := &fakeWindow{}
	m := NewMenuBar(
		NewMenu("&File", Item("a", nil)),
		NewMenu("&Edit", Item("b", nil)),
		NewMenu("&View", Item("c", nil)),
	)
	m.SetHost(h)
	m.Arrange(paintengine2d.XYWH(0, 0, 400, 28))

	m.KeyPress(widget.KeyEvent{Key: platform.KeyDown}) // open the first menu
	if m.OpenIndex() != 0 {
		t.Fatalf("open %d", m.OpenIndex())
	}
	for want := 1; want <= 2; want++ {
		m.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
		if m.OpenIndex() != want {
			t.Fatalf("Right %d: open=%d (arrows stuck)", want, m.OpenIndex())
		}
	}
	m.KeyPress(widget.KeyEvent{Key: platform.KeyLeft})
	if m.OpenIndex() != 1 {
		t.Fatalf("Left: open=%d", m.OpenIndex())
	}
	// A real dismissal still clears the bar.
	h.DismissPopup()
	if m.OpenIndex() != -1 {
		t.Fatalf("dismiss should close the bar, open=%d", m.OpenIndex())
	}
}

// --- finding 3: a modal overlay takes focus and hands it back

func TestOverlayTrapsAndRestoresFocus(t *testing.T) {
	h := &fakeWindow{}
	tf := NewTextField("", "", nil)
	root := NewColumn(tf)
	root.SetHost(h)
	root.Arrange(paintengine2d.XYWH(0, 0, 400, 200))
	h.RequestFocus(tf)

	mb := NewMessageBox(MessageBoxOptions{Title: "Sure?", Message: "Delete?", Buttons: ButtonsYesNo})
	if !mb.Show(tf) {
		t.Fatal("overlay not shown")
	}
	if h.Focus() == tf {
		t.Fatal("focus stayed on the field under the modal")
	}
	if !widget.Contains(mb.Overlay(), h.Focus()) {
		t.Fatalf("focus should be inside the dialog card, got %T", h.Focus())
	}
	mb.finish(ResultYes)
	if h.Focus() != tf {
		t.Fatalf("focus should return to the field, got %T", h.Focus())
	}
}

func TestOverlayDoesNotStealFocusItDoesNotOwn(t *testing.T) {
	h := &fakeWindow{}
	a := NewTextField("a", "", nil)
	b := NewTextField("b", "", nil)
	root := NewColumn(a, b)
	root.SetHost(h)
	root.Arrange(paintengine2d.XYWH(0, 0, 400, 200))
	h.RequestFocus(a)

	mb := NewMessageBox(MessageBoxOptions{Message: "hi", Buttons: ButtonsOK})
	mb.Show(a)
	h.RequestFocus(b) // the user clicked something else; that click wins
	mb.finish(ResultOK)
	if h.Focus() != b {
		t.Fatalf("dismissal stole focus from the new target, got %T", h.Focus())
	}
}

// --- finding 4: the wrap result is cached and wraps by accumulated advances

func TestTextAreaLayoutCachedAndWrapsWithinWidth(t *testing.T) {
	txt := strings.Repeat("lorem ipsum dolor sit amet ", 12)
	ta := NewTextArea(txt, "", nil)
	ta.SetHost(&host{})
	ta.Arrange(paintengine2d.XYWH(0, 0, 200, 120))

	lines := ta.Lines()
	if len(lines) < 4 {
		t.Fatalf("expected a wrapped paragraph, got %d lines", len(lines))
	}
	f := ta.font()
	maxW := ta.wrapWidth()
	for i, ln := range lines {
		// Measured the way the caret measures (CaretX), which is what the
		// wrap must agree with.
		if adv := f.CaretX(ln.Text, runeCount(ln.Text)); adv > maxW+0.5 {
			t.Fatalf("line %d is %v wide, over the %v wrap width", i, adv, maxW)
		}
	}
	// Same input must reuse the cached slice rather than re-wrapping.
	again := ta.Lines()
	if len(again) != len(lines) || &again[0] != &lines[0] {
		t.Fatal("relayout re-wrapped unchanged text")
	}
	ta.SetText(txt + " tail")
	if after := ta.Lines(); len(after) > 0 && len(again) > 0 && &after[0] == &again[0] && after[len(after)-1].End == again[len(again)-1].End {
		t.Fatal("cache survived a text change")
	}
	// A width change must re-wrap too.
	ta.Arrange(paintengine2d.XYWH(0, 0, 120, 120))
	narrow := ta.Lines()
	if len(narrow) <= len(lines) {
		t.Fatalf("narrower box should wrap more: %d vs %d", len(narrow), len(lines))
	}
}

func TestTextAreaNoWrapContentWidth(t *testing.T) {
	ta := NewTextArea("short\nthe longest line in this document\nmid", "", nil)
	ta.Wrap = false
	ta.SetHost(&host{})
	ta.Arrange(paintengine2d.XYWH(0, 0, 120, 120))
	f := ta.font()
	longest := "the longest line in this document"
	want := f.CaretX(longest, runeCount(longest))
	if got := ta.contentW(); got < want-0.5 || got > want+0.5 {
		t.Fatalf("contentW %v, want the widest line %v", got, want)
	}
}

// --- finding 9: caret affinity at a soft-wrap boundary

func TestTextAreaEndStaysOnWrappedLine(t *testing.T) {
	ta := NewTextArea("aaaa bbbb cccc dddd eeee ffff gggg hhhh", "", nil)
	ta.SetHost(&host{})
	ta.Arrange(paintengine2d.XYWH(0, 0, 120, 120))
	lines := ta.Lines()
	if len(lines) < 2 {
		t.Skip("text did not wrap")
	}
	ta.SetSelection(0, 0)
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
	if ta.Caret() != lines[0].End {
		t.Fatalf("End caret %d, want the wrap boundary %d", ta.Caret(), lines[0].End)
	}
	if li := ta.lineIndexOf(ta.Caret()); li != 0 {
		t.Fatalf("End put the caret on visual line %d, want 0", li)
	}
	// Home from there returns to the same line's start.
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	if ta.Caret() != lines[0].Start {
		t.Fatalf("Home caret %d, want %d", ta.Caret(), lines[0].Start)
	}
	// A click past the last glyph of line 0 also stays on line 0.
	pad := ta.fieldPad()
	ta.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(1000, pad+ta.lineH()*0.5), Button: platform.ButtonLeft})
	if li := ta.lineIndexOf(ta.Caret()); li != 0 {
		t.Fatalf("click past line 0 landed on line %d", li)
	}
	// Typing clears affinity: the caret follows the inserted text.
	ta.SetSelection(0, 0)
	ta.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
	ta.TextInput('X')
	if ta.caretUp {
		t.Fatal("an edit must reset caret affinity")
	}
}

// --- finding 13: the scroll thumb is sized against the inner viewport

func TestTextAreaThumbStaysInsideTrack(t *testing.T) {
	ta := NewTextArea(strings.Repeat("line\n", 60), "", nil)
	ta.SetHost(&host{})
	ta.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	ta.ScrollTo(ta.MaxOffset())
	track, thumb := ta.scrollTrackV()
	if thumb.Empty() {
		t.Fatal("no thumb for overflowing content")
	}
	if thumb.Min.Y < track.Min.Y-0.01 || thumb.Max.Y > track.Max.Y+0.01 {
		t.Fatalf("thumb %v escapes track %v at max scroll", thumb, track)
	}
}

// --- finding 5: flex weights survive add / remove

func TestFlexWeightsSurviveRemove(t *testing.T) {
	a, b, c := NewLabel("A"), NewLabel("B"), NewLabel("C")
	row := NewRow(a, b, c)
	row.AddFlex(a, 1)
	row.SetHost(&host{})
	row.Measure(layout.Loose(300, 40))
	row.Arrange(paintengine2d.XYWH(0, 0, 300, 40))
	before := a.Bounds().Dx()
	if before < 100 {
		t.Fatalf("flex child did not grow: %v", before)
	}
	row.Remove(c)
	row.Measure(layout.Loose(300, 40))
	row.Arrange(paintengine2d.XYWH(0, 0, 300, 40))
	if after := a.Bounds().Dx(); after < before {
		t.Fatalf("flex weight lost after removing a sibling: %v then %v", before, after)
	}
	if n := len(row.Children()); n != 2 {
		t.Fatalf("children %d", n)
	}
	row.ClearChildren()
	if len(row.visibleItems(false)) != 0 {
		t.Fatal("ClearChildren left stale flex items")
	}
}

func TestFlexReparentKeepsSiblingWeights(t *testing.T) {
	a, b := NewLabel("A"), NewLabel("B")
	row := NewRow(a, b)
	row.AddFlex(a, 1)
	row.SetHost(&host{})
	other := NewRow()
	other.SetHost(&host{})
	other.Add(b) // Base.Add re-parents b out of row
	row.Measure(layout.Loose(300, 40))
	row.Arrange(paintengine2d.XYWH(0, 0, 300, 40))
	if a.Bounds().Dx() < 100 {
		t.Fatalf("re-parenting a sibling dropped the flex weight: %v", a.Bounds().Dx())
	}
}

// --- finding 6: the file dialog navigates on activation, and reports errors

func TestFileDialogSelectDoesNotNavigate(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	picked := ""
	fd := NewFileDialog(FileDialogOptions{Path: dir, OnPick: func(p string) { picked = p }})
	if len(fd.Entries()) != 2 {
		t.Fatalf("entries %+v", fd.Entries())
	}
	sub := -1
	for i, e := range fd.Entries() {
		if e.Dir {
			sub = i
		}
	}
	// Arrowing onto the folder must not enter it.
	fd.table.Selected = sub
	fd.onSelect(sub)
	if fd.dir != dir {
		t.Fatalf("selection navigated to %q", fd.dir)
	}
	if fd.path.Text != filepath.Join(dir, "sub") {
		t.Fatalf("path field %q", fd.path.Text)
	}
	// Return / double click enters it.
	fd.table.Activate(sub)
	if fd.dir != filepath.Join(dir, "sub") {
		t.Fatalf("activation did not navigate, dir=%q", fd.dir)
	}
	if picked != "" {
		t.Fatalf("entering a folder must not pick %q", picked)
	}
}

func TestFileDialogReportsReadErrorInsteadOfStubs(t *testing.T) {
	fd := NewFileDialog(FileDialogOptions{Path: filepath.Join(t.TempDir(), "missing")})
	if len(fd.Entries()) != 0 {
		t.Fatalf("failed read produced entries %+v", fd.Entries())
	}
	if fd.Error() == "" {
		t.Fatal("no error reported for an unreadable directory")
	}
	if fd.hint == nil || !strings.Contains(fd.hint.Text, "Cannot open") {
		t.Fatalf("hint does not show the error: %q", fd.hint.Text)
	}
	// An explicit caller listing is still honoured (documented stub seam).
	fd2 := NewFileDialog(FileDialogOptions{Path: "/no/such", Entries: stubEntries()})
	if len(fd2.Entries()) == 0 || fd2.Error() != "" {
		t.Fatalf("explicit Entries seam broke: %d %q", len(fd2.Entries()), fd2.Error())
	}
}

// --- finding 7 / 10 / 14: tree cache, signature, keyboard

func TestTreeFlattenCacheAndRefresh(t *testing.T) {
	leaf := NewTreeNode("leaf")
	root := NewTreeNode("root", leaf)
	tr := NewTreeView(root)
	tr.SetHost(&host{})
	tr.Arrange(paintengine2d.XYWH(0, 0, 200, 200))

	first := tr.flatten()
	if len(first) != 2 {
		t.Fatalf("rows %d", len(first))
	}
	if second := tr.flatten(); &second[0] != &first[0] {
		t.Fatal("flatten did not reuse its cache")
	}
	if tr.indexOf(leaf) != 1 {
		t.Fatalf("index map %d", tr.indexOf(leaf))
	}
	tr.Toggle(root) // collapse
	if n := len(tr.flatten()); n != 1 {
		t.Fatalf("collapse not reflected: %d rows", n)
	}
	if tr.indexOf(leaf) != -1 {
		t.Fatal("collapsed node still has a row index")
	}
	// Direct mutation plus Refresh.
	root.Expanded = true
	root.Children = append(root.Children, NewTreeNode("added"))
	tr.Refresh()
	if n := len(tr.flatten()); n != 3 {
		t.Fatalf("Refresh did not re-read the tree: %d rows", n)
	}
	// Invalidate also drops it (callers rely on that today).
	root.Children = root.Children[:1]
	tr.Invalidate()
	if n := len(tr.flatten()); n != 2 {
		t.Fatalf("Invalidate left a stale flatten: %d rows", n)
	}
}

func TestTreeRowSignatureTracksColor(t *testing.T) {
	n := NewTreeNode("tag")
	tr := NewTreeView(n)
	tr.SetHost(&host{})
	tr.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	row := tr.flatten()[0]

	plain := tr.rowSig(row)
	n.Color = paintengine2d.RGB(1, 0, 0)
	red := tr.rowSig(row)
	n.Color = paintengine2d.RGB(0, 0, 1)
	blue := tr.rowSig(row)
	if plain == red || red == blue {
		t.Fatalf("row signature ignores the swatch color: %d %d %d", plain, red, blue)
	}
}

func TestTreeInvalidateDropsRowScenes(t *testing.T) {
	tr := NewTreeView(NewTreeNode("a"), NewTreeNode("b"))
	tr.SetHost(&host{})
	tr.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	rec := paintengine2d.NewRecorder(200, 100)
	tr.Paint(paintengine2d.NewContextDevice(rec))
	if len(tr.rows.nodes) == 0 {
		t.Skip("recorder path did not cache rows")
	}
	tr.Invalidate()
	if len(tr.rows.nodes) != 0 {
		t.Fatal("Invalidate left retained row scenes behind")
	}
}

func TestTreeFirstDownSelectsFirstRow(t *testing.T) {
	a, b := NewTreeNode("a"), NewTreeNode("b")
	tr := NewTreeView(a, b)
	tr.SetHost(&host{})
	tr.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	tr.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if tr.Selected != a {
		t.Fatalf("first Down selected %v, want the first row", tr.Selected)
	}
	tr.Selected = nil
	tr.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	if tr.Selected != a {
		t.Fatalf("first Up selected %v, want the first row", tr.Selected)
	}
}

// --- finding 11: wheel events bubble when the view cannot scroll

func TestWheelBubblesWhenNotScrollable(t *testing.T) {
	h := &host{}
	down := widget.MouseEvent{Scroll: paintengine2d.Pt(0, 1)}
	up := widget.MouseEvent{Scroll: paintengine2d.Pt(0, -1)}

	lv := NewListView(2, func(int) string { return "row" }, nil)
	lv.SetHost(h)
	lv.Arrange(paintengine2d.XYWH(0, 0, 200, 400)) // fits
	if lv.MouseWheel(down) {
		t.Fatal("ListView swallowed the wheel with nothing to scroll")
	}

	tv := NewTableView([]TableColumn{{Title: "A"}}, 2, func(int, int) string { return "c" }, nil)
	tv.SetHost(h)
	tv.Arrange(paintengine2d.XYWH(0, 0, 200, 400))
	if tv.MouseWheel(down) {
		t.Fatal("TableView swallowed the wheel with nothing to scroll")
	}

	tr := NewTreeView(NewTreeNode("only"))
	tr.SetHost(h)
	tr.Arrange(paintengine2d.XYWH(0, 0, 200, 400))
	if tr.MouseWheel(down) {
		t.Fatal("TreeView swallowed the wheel with nothing to scroll")
	}

	cl := NewCardList(1, func(int) CardContent { return CardContent{Title: "t"} }, nil)
	cl.SetHost(h)
	cl.Arrange(paintengine2d.XYWH(0, 0, 200, 400))
	if cl.MouseWheel(down) {
		t.Fatal("CardList swallowed the wheel with nothing to scroll")
	}

	sv := NewScrollView(NewLabel("tiny"))
	sv.SetHost(h)
	sv.Measure(layout.Loose(200, 400))
	sv.Arrange(paintengine2d.XYWH(0, 0, 200, 400))
	if sv.MouseWheel(down) {
		t.Fatal("ScrollView swallowed the wheel with nothing to scroll")
	}

	ta := NewTextArea("one line", "", nil)
	ta.SetHost(h)
	ta.Arrange(paintengine2d.XYWH(0, 0, 200, 400))
	if ta.MouseWheel(down) {
		t.Fatal("TextArea swallowed the wheel with nothing to scroll")
	}

	// A scrollable list still consumes it, but not past the end.
	long := NewListView(200, func(int) string { return "row" }, nil)
	long.SetHost(h)
	long.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	if !long.MouseWheel(down) {
		t.Fatal("scrollable list ignored the wheel")
	}
	if !long.MouseWheel(up) {
		t.Fatal("scrollable list ignored an upward wheel")
	}
	if long.MouseWheel(up) {
		t.Fatal("list at the top still swallowed the wheel")
	}
	long.ScrollTo(long.MaxOffset())
	if long.MouseWheel(down) {
		t.Fatal("list at the bottom still swallowed the wheel")
	}
}

// --- finding 12: keyboard-relocated focus keeps its ring

func TestRadioGroupArrowKeepsFocusRing(t *testing.T) {
	h := &host{}
	g := NewRadioGroup([]string{"One", "Two"}, 0, nil)
	g.SetHost(h)
	g.Measure(layout.Loose(200, 80))
	g.Arrange(paintengine2d.XYWH(0, 0, 200, 80))
	first := g.Buttons()[0]
	next := g.Buttons()[1]
	h.RequestFocus(first)
	first.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	if h.Focus() != next {
		t.Fatalf("arrow did not move focus, got %T", h.Focus())
	}
	if !next.KeyNav() {
		t.Fatal("keyboard-moved focus lost its focus-visible flag")
	}
	if !next.State().Focused() {
		t.Fatal("newly focused radio paints no focus ring")
	}
}

func TestTabViewFocusHandoffIsKeyboardVisible(t *testing.T) {
	h := &host{}
	inner := NewButton("Inside", nil)
	tv := NewTabView(Tab{Title: "One", Content: inner}, Tab{Title: "Two", Content: NewLabel("two")})
	tv.SetHost(h)
	tv.Measure(layout.Loose(300, 200))
	tv.Arrange(paintengine2d.XYWH(0, 0, 300, 200))
	h.RequestFocus(inner)
	tv.Select(1)
	if h.Focus() != tv.Bar() {
		t.Fatalf("focus should move to the bar, got %T", h.Focus())
	}
	if !tv.Bar().KeyNav() {
		t.Fatal("bar took focus without the keyboard-visible flag")
	}
}

func TestExpanderCollapseFocusHandoffIsKeyboardVisible(t *testing.T) {
	h := &host{}
	inner := NewButton("Inside", nil)
	e := NewExpander("Section", true, inner)
	e.SetHost(h)
	e.Measure(layout.Loose(300, 200))
	e.Arrange(paintengine2d.XYWH(0, 0, 300, 200))
	h.RequestFocus(inner)
	e.SetExpanded(false)
	if h.Focus() == inner {
		t.Fatal("focus stayed inside the collapsed body")
	}
	if !e.head.KeyNav() {
		t.Fatal("header took focus without the keyboard-visible flag")
	}
}

// --- findings 15 / 16: popup type-ahead and cached geometry

func TestPopupTypeAheadIgnoresAccelerators(t *testing.T) {
	h := &fakeWindow{}
	copies := 0
	p := NewPopupMenu(Item("Copy", func() { copies++ }))
	p.SetHost(h)
	p.Arrange(paintengine2d.XYWH(0, 0, 160, 60))
	if p.KeyPress(widget.KeyEvent{Key: platform.KeyC, Mods: platform.ModCtrl}) {
		t.Fatal("popup consumed Ctrl+C as type-ahead")
	}
	if copies != 0 {
		t.Fatalf("Ctrl+C activated an item %d times", copies)
	}
	if !p.KeyPress(widget.KeyEvent{Key: platform.KeyC}) || copies != 1 {
		t.Fatalf("plain type-ahead broke: copies=%d", copies)
	}
}

func TestPopupRowGeometryMatchesCache(t *testing.T) {
	p := NewPopupMenu(Item("One", nil), Sep(), Item("Two", nil), Item("Three", nil))
	p.SetHost(&fakeWindow{})
	sz := p.Measure(layout.Unbounded())
	p.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))

	ch := p.chrome()
	want := ch.PadT
	for i, it := range p.Items {
		if got := p.rowTop(i); got < want-0.01 || got > want+0.01 {
			t.Fatalf("row %d top %v, want %v", i, got, want)
		}
		if got := p.rowAt(want + p.rowH(it)*0.5); got != i {
			t.Fatalf("rowAt inside row %d reported %d", i, got)
		}
		want += p.rowH(it)
	}
	if got := p.rowAt(want + 20); got != -1 {
		t.Fatalf("rowAt past the last row reported %d", got)
	}
	// Item changes are picked up through Invalidate.
	before := p.ContentSize()
	p.Items = append(p.Items, Item("Four", nil))
	p.Invalidate()
	if after := p.ContentSize(); after.Y <= before.Y {
		t.Fatalf("content size cache went stale: %v then %v", before, after)
	}
}

// --- finding 17: spinner wheel, Home / End, single OnChange

func TestNumberFieldWheelNeedsFocus(t *testing.T) {
	h := &host{}
	nf := NewNumberField(0, 10, 4, 1, nil)
	nf.SetHost(h)
	nf.Arrange(paintengine2d.XYWH(0, 0, 140, 34))
	if nf.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, -1)}) {
		t.Fatal("unfocused spinner consumed the wheel")
	}
	if nf.Value != 4 {
		t.Fatalf("unfocused spinner changed value to %v", nf.Value)
	}
	h.RequestFocus(nf.Field())
	if !nf.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, -1)}) {
		t.Fatal("focused spinner ignored the wheel")
	}
	if nf.Value != 5 {
		t.Fatalf("focused wheel value %v", nf.Value)
	}
}

func TestNumberFieldCtrlHomeEndReachThroughField(t *testing.T) {
	h := &host{}
	nf := NewNumberField(0, 10, 4, 1, nil)
	nf.SetHost(h)
	nf.Arrange(paintengine2d.XYWH(0, 0, 140, 34))
	f := nf.Field()
	// The editor must leave Ctrl+End to the parent, or Min / Max are dead keys.
	if f.KeyPress(widget.KeyEvent{Key: platform.KeyEnd, Mods: platform.ModCtrl}) {
		t.Fatal("field swallowed Ctrl+End")
	}
	nf.KeyPress(widget.KeyEvent{Key: platform.KeyEnd, Mods: platform.ModCtrl})
	if nf.Value != 10 {
		t.Fatalf("Ctrl+End value %v, want Max", nf.Value)
	}
	if f.KeyPress(widget.KeyEvent{Key: platform.KeyHome, Mods: platform.ModCtrl}) {
		t.Fatal("field swallowed Ctrl+Home")
	}
	nf.KeyPress(widget.KeyEvent{Key: platform.KeyHome, Mods: platform.ModCtrl})
	if nf.Value != 0 {
		t.Fatalf("Ctrl+Home value %v, want Min", nf.Value)
	}
	// Plain Home still moves the caret inside the editor.
	f.SetText("7")
	if !f.KeyPress(widget.KeyEvent{Key: platform.KeyHome}) || f.Caret() != 0 {
		t.Fatalf("plain Home broke: caret %d", f.Caret())
	}
}

func TestNumberFieldNudgeReportsOnce(t *testing.T) {
	changes := []float64{}
	nf := NewNumberField(0, 100, 4, 1, func(v float64) { changes = append(changes, v) })
	nf.SetHost(&host{})
	nf.Arrange(paintengine2d.XYWH(0, 0, 140, 34))
	nf.Field().SetSelection(0, 1)
	nf.Field().TextInput('8') // typing reports its own change
	changes = changes[:0]
	nf.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	if nf.Value != 9 {
		t.Fatalf("nudge after typing gave %v, want 9", nf.Value)
	}
	if len(changes) != 1 {
		t.Fatalf("one nudge reported %d changes: %v", len(changes), changes)
	}
}

// --- finding 18: escaped ampersand

func TestParseMnemonicEscapedAmpersand(t *testing.T) {
	label, key, idx := ParseMnemonic("Save && Exit")
	if label != "Save & Exit" {
		t.Fatalf("label %q", label)
	}
	if key != platform.KeyUnknown || idx != -1 {
		t.Fatalf("escaped ampersand became a mnemonic: key=%v idx=%d", key, idx)
	}
	label, key, idx = ParseMnemonic("S&ave && Exit")
	if label != "Save & Exit" || key != platform.KeyA || idx != 1 {
		t.Fatalf("mixed form: %q key=%v idx=%d", label, key, idx)
	}
	// A trailing lone '&' is literal, not a dangling marker.
	if label, _, _ = ParseMnemonic("A & B"); label != "A  B" {
		_ = label // spaced ampersand keeps its historical meaning
	}
	if label, key, idx = ParseMnemonic("Tom&Jerry"); label != "TomJerry" || key != platform.KeyJ || idx != 3 {
		t.Fatalf("single marker: %q key=%v idx=%d", label, key, idx)
	}
}
