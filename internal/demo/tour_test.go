package demo

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/dock"
	"github.com/codemodify/uitoolkit/internal/uitest"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The tour is checked in the packs that disagree most about everything —
// two that predate anti-aliasing, one that made gradients normal, two
// that are mostly blur, one web-era, and a skin whose chrome is
// pictures and which cuts the window's outline. A page that reads in all
// of those reads everywhere.
var tourLooks = []string{"win95", "system7", "luna", "bigsur", "tahoe", "sourcegit", "deck"}

// tourWindow opens a headless tour window holding one page, laid out and
// painted, in the named pack at the given scale.
func tourWindow(t *testing.T, pack string, scale float32, page int) (*app.Application, *app.Window) {
	t.Helper()
	look := style.DarkLook()
	if p, ok := style.LoadTheme(pack); ok {
		look = p.Look()
	} else if pack != "" {
		t.Fatalf("unknown pack %q", pack)
	}
	a := uitoolkit.New(uitoolkit.Options{
		Look: look, Headless: true, Scale: scale, DisableLookWatch: true,
	})
	// A window's size is logical pixels at every scale, so the same
	// 1180 x 820 page comes back as 1180*scale device pixels to capture.
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "tour", Width: 1180, Height: 820, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	win.SetContent(TourAppOpen(a, win, page))
	// Twice: the first lays the page out, the second runs what the page
	// posted for after its first frame (the accessibility read).
	a.PumpOnce()
	a.PumpOnce()
	return a, win
}

// Every page builds, lays out and paints something in every extreme
// pack, at 1x and at a fractional scale. "Something" is the point: a
// page that measures to nothing, or paints the background and stops,
// passes every other test in this file.
func TestTourPagesPaintInEveryLook(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "0")
	for _, pack := range tourLooks {
		for _, scale := range []float32{1, 1.75} {
			if testing.Short() && scale != 1 {
				continue
			}
			for page := range tourPages {
				name := pack + "/" + strings.ReplaceAll(tourPages[page].name, " ", "-")
				if scale != 1 {
					name += "@1.75"
				}
				t.Run(name, func(t *testing.T) {
					a, win := tourWindow(t, pack, scale, page)
					defer win.Close()
					img := win.Capture()
					if img == nil {
						t.Fatal("no frame")
					}
					w, h := img.Width, img.Height
					if got := int(1180 * scale); w != got {
						t.Errorf("frame is %d px wide, want %d", w, got)
					}
					if got := win.Scale(); got != scale {
						t.Errorf("the window is at scale %.2f, want %.2f", got, scale)
					}
					// A page that drew nothing is one flat colour.
					box := paintengine2d.XYWH(0, 0, float32(w), float32(h))
					ink := uitest.CountNonColor(img, box, win.Look().Palette().Background, 12)
					if min := w * h / 100; ink < min {
						t.Errorf("only %d of %d pixels differ from the background; the page looks blank",
							ink, w*h)
					}
					_ = a
				})
			}
		}
	}
}

// Every page passes the accessibility linter, in more than one pack: a
// pack changes metrics, and a control squeezed to nothing loses its box.
func TestTourPagesAreAccessible(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "0")
	for _, pack := range []string{"win95", "tahoe", "deck"} {
		for page := range tourPages {
			t.Run(pack+"/"+strings.ReplaceAll(tourPages[page].name, " ", "-"), func(t *testing.T) {
				_, win := tourWindow(t, pack, 1, page)
				defer win.Close()
				tree := win.AccessibleTree()
				if len(tree.Children) == 0 {
					t.Fatal("empty accessibility tree")
				}
				var lines []string
				for _, p := range a11y.Check(tree) {
					lines = append(lines, p.String())
				}
				if len(lines) > 0 {
					t.Errorf("%d accessibility problems:\n%s", len(lines), strings.Join(lines, "\n"))
				}
			})
		}
	}
}

// The whole tour in one window passes too: the pages share a window, and
// two of them naming the same thing would be two nodes with one id.
func TestTourWholeWindowIsAccessible(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "0")
	a := uitoolkit.New(uitoolkit.Options{
		Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true,
	})
	win, err := a.NewWindow(platform.WindowOptions{Title: "tour", Width: 1180, Height: 820, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	win.SetContent(TourApp(a, win))
	a.PumpOnce()

	strip := tourStripOf(t, win)
	for i := 0; i < strip.Len(); i++ {
		strip.Select(i)
		a.PumpOnce()
		a.PumpOnce()
		var lines []string
		for _, p := range a11y.Check(win.AccessibleTree()) {
			lines = append(lines, p.String())
		}
		if len(lines) > 0 {
			t.Errorf("page %q: %d problems:\n%s", strip.Tab(i).Title, len(lines), strings.Join(lines, "\n"))
		}
	}
}

// Tab reaches every control on every page and comes back to where it
// started, and every control it stops on says what it is.
func TestTourKeyboardReachesEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "0")
	for page := range tourPages {
		t.Run(strings.ReplaceAll(tourPages[page].name, " ", "-"), func(t *testing.T) {
			a, win := tourWindow(t, "luna", 1, page)
			defer win.Close()

			want := widget.Focusables(win.Content())
			if len(want) < 2 {
				t.Fatalf("only %d focusable controls on this page", len(want))
			}
			for _, c := range want {
				if focusNameOf(c) == "" {
					t.Errorf("a %T takes the focus and describes itself as nothing", c)
				}
			}

			seen := map[widget.Component]bool{}
			for i := 0; i <= len(want); i++ {
				win.Inject(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab})
				a.PumpOnce()
				if f := win.Focus(); f != nil {
					seen[f] = true
				}
			}
			var missed []string
			for _, c := range want {
				if !seen[c] {
					missed = append(missed, focusNameOf(c))
				}
			}
			if len(missed) > 0 {
				t.Errorf("Tab never reached %d of %d controls: %s",
					len(missed), len(want), strings.Join(missed, ", "))
			}
		})
	}
}

func focusNameOf(c widget.Component) string {
	var n a11y.Node
	if d, ok := c.(interface{ Describe(*a11y.Node) }); ok {
		d.Describe(&n)
	}
	if n.Name != "" {
		return n.Name
	}
	if g, ok := c.(interface{ AccessibleName() string }); ok {
		return g.AccessibleName()
	}
	return ""
}

// tourStripOf is the tab strip a tour window put in its title bar.
func tourStripOf(t *testing.T, win *app.Window) *widgets.BrowserTabs {
	t.Helper()
	hb := win.Caption()
	if hb == nil {
		t.Fatal("the window has no caption")
	}
	if s, ok := hb.Center().(*widgets.BrowserTabs); ok {
		return s
	}
	t.Fatalf("the caption's centre is %T, not a tab strip", hb.Center())
	return nil
}

// ---- page one: the strip ---------------------------------------------------------

// A tab dragged along the strip lands where it was dropped, the page
// follows it, and the body shows the page the selected tab holds.
func TestTourTabsReorderAndSelect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	win, err := a.NewWindow(platform.WindowOptions{Title: "tour", Width: 1180, Height: 820, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	win.SetContent(TourApp(a, win))
	a.PumpOnce()
	strip := tourStripOf(t, win)

	if got := strip.Len(); got != len(tourPages) {
		t.Fatalf("the strip holds %d tabs, want %d", got, len(tourPages))
	}
	first, second := strip.Tab(0).Title, strip.Tab(1).Title
	strip.MoveTab(0, 1)
	if strip.Tab(0).Title != second || strip.Tab(1).Title != first {
		t.Errorf("after moving tab 0 to 1 the strip reads %q, %q", strip.Tab(0).Title, strip.Tab(1).Title)
	}
	// The page travels with its tab: the data is the page, not the index.
	if p, ok := strip.Tab(1).Data.(int); !ok || tourPages[p].name != first {
		t.Errorf("tab 1 now holds %v, want the page called %q", strip.Tab(1).Data, first)
	}

	// Selecting shows that page's heading.
	strip.Select(3)
	a.PumpOnce()
	p := strip.Tab(3).Data.(int)
	if !windowShowsText(win, tourPages[p].title) {
		t.Errorf("selecting tab 3 did not put %q on the page", tourPages[p].title)
	}
}

// A window keeps its last page: the only tab has no close button, and
// closing it does nothing.
func TestTourLastPageStays(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, win := tourWindow(t, "", 1, pageTabs)
	defer win.Close()
	strip := tourStripOf(t, win)
	if strip.Len() != 1 {
		t.Fatalf("a one-page window has %d tabs", strip.Len())
	}
	if !strip.Tab(0).NoClose {
		t.Error("the only tab still shows a close button")
	}
	strip.CloseTab(0)
	a.PumpOnce()
	if strip.Len() != 1 {
		t.Errorf("closing the last tab left %d tabs", strip.Len())
	}
}

// A page moved out of the strip goes into a window of its own, and the
// same page dropped back on a strip joins it at the caret.
func TestTourTabTearsOffAndMergesBack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	win, err := a.NewWindow(platform.WindowOptions{Title: "tour", Width: 1180, Height: 820, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	win.SetContent(TourApp(a, win))
	a.PumpOnce()
	strip := tourStripOf(t, win)
	tour := stateOf(t, win)

	before := strip.Len()
	windowsBefore := len(a.Windows())
	moved := strip.Tab(2)
	tour.tearOff(2)
	a.PumpOnce()

	if strip.Len() != before-1 {
		t.Errorf("after moving a page out the strip holds %d tabs, want %d", strip.Len(), before-1)
	}
	if len(a.Windows()) != windowsBefore+1 {
		t.Fatalf("moving a page out opened %d windows", len(a.Windows())-windowsBefore)
	}
	// The new window runs that page and nothing else.
	other := a.Windows()[len(a.Windows())-1]
	otherStrip := tourStripOf(t, other)
	if otherStrip.Len() != 1 || otherStrip.Tab(0).Title != moved.Title {
		t.Errorf("the new window holds %d tabs, the first %q; want 1 × %q",
			otherStrip.Len(), otherStrip.Tab(0).Title, moved.Title)
	}
	defer other.Close()

	// And back: a drop of that tab on the first window's strip, as the
	// drag would deliver it.
	at := paintengine2d.Pt(1, 1)
	took := strip.Drop(widget.DropEvent{
		Pos:     at,
		Mime:    widgets.TabMimeType,
		Data:    []byte(moved.Title),
		Payload: moved,
		Action:  platform.DragMove,
		Source:  otherStrip,
	})
	a.PumpOnce()
	if !took {
		t.Fatal("the strip refused a tab dropped on it")
	}
	if strip.Len() != before {
		t.Errorf("after the drop the strip holds %d tabs, want %d", strip.Len(), before)
	}
	if strip.Tab(0).Title != moved.Title {
		t.Errorf("the tab landed at %q, not at the caret (tab 0)", strip.Tab(0).Title)
	}
	// A page already open is refused rather than opened twice.
	if strip.Drop(widget.DropEvent{Pos: at, Mime: widgets.TabMimeType,
		Data: []byte(moved.Title), Payload: moved, Action: platform.DragMove}) {
		t.Error("the strip took a second copy of a page it already holds")
	}
}

// Closing a page and opening it again from "+" gets the same page back.
func TestTourPageClosesAndReopens(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	win, err := a.NewWindow(platform.WindowOptions{Title: "tour", Width: 1180, Height: 820, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	win.SetContent(TourApp(a, win))
	a.PumpOnce()
	strip := tourStripOf(t, win)
	tour := stateOf(t, win)

	strip.CloseTab(strip.Len() - 1)
	a.PumpOnce()
	if tour.isOpen(pageAccess) {
		t.Fatal("the closed page is still open")
	}
	tour.openPage(pageAccess)
	a.PumpOnce()
	if !tour.isOpen(pageAccess) {
		t.Fatal("the page did not come back")
	}
	if got := tour.current(); got != pageAccess {
		t.Errorf("reopening selected page %d, want %d", got, pageAccess)
	}
}

// ---- page two: docking ------------------------------------------------------------

func TestTourDockFloatsTabsAndRestores(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, win := tourWindow(t, "", 1, pageDock)
	defer win.Close()
	p := stateOf(t, win).page(pageDock).(*dockPage)

	outline := p.host.Panel("outline")
	notes := p.host.Panel("notes")
	if outline == nil || notes == nil {
		t.Fatal("the page did not build its panels")
	}

	// Floating and docking back.
	if !outline.Float() {
		t.Fatal("the panel would not float")
	}
	a.PumpOnce()
	if !outline.Floating() {
		t.Error("the panel floated but does not say so")
	}
	if !outline.Dock() {
		t.Fatal("the panel would not dock back")
	}
	a.PumpOnce()
	if outline.Floating() {
		t.Error("the panel docked back but still says it is floating")
	}

	// Tabbing two panels together puts them in one stack.
	p.host.DockInto(outline, notes)
	a.PumpOnce()
	if outline.Stack() == nil || outline.Stack() != notes.Stack() {
		t.Error("tabbing the two panels did not put them in one stack")
	}

	// The arrangement survives being written down and read back.
	saved, err := p.host.LayoutJSON()
	if err != nil {
		t.Fatal(err)
	}
	p.host.Dock(outline, dock.SideBottom)
	a.PumpOnce()
	if outline.Stack() == notes.Stack() {
		t.Fatal("moving the panel away left it in the same stack")
	}
	if err := p.host.ApplyLayoutJSON(saved); err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()
	if outline.Stack() == nil || outline.Stack() != notes.Stack() {
		t.Error("the layout read back from JSON did not put the panels together again")
	}

	// A closed panel keeps its place; showing it puts it back.
	notes.Close()
	a.PumpOnce()
	if !notes.Closed() {
		t.Error("the panel did not close")
	}
	notes.Show()
	a.PumpOnce()
	if notes.Closed() {
		t.Error("the panel did not come back")
	}
}

// ---- page three: drag and drop -----------------------------------------------------

// A row dragged from one list to the other is moved: it lands at the gap
// the caret marked and leaves the list it came from.
func TestTourDropMovesARowBetweenLists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, win := tourWindow(t, "", 1, pageDrag)
	defer win.Close()
	p := stateOf(t, win).page(pageDrag).(*dragPage)

	from, to := p.left, p.right
	if len(from.rows) < 2 || len(to.rows) < 1 {
		t.Fatalf("the page starts with %d and %d rows", len(from.rows), len(to.rows))
	}
	from.view.Selected = 1
	moving := from.rows[1]
	wasFrom, wasTo := len(from.rows), len(to.rows)

	d := from.drag([]int{1})
	if d == nil {
		t.Fatal("a selected row dragged nothing")
	}
	if !d.Offers("text/uri-list") {
		t.Errorf("the drag offers %v; a file manager needs a uri-list", d.Types)
	}
	if !d.Allowed().Has(platform.DragMove) {
		t.Error("the drag does not allow a move, so a row could never be filed")
	}

	// Dropped at the top of the other list.
	e := d.DropEvent(paintengine2d.Pt(1, 1), "text/uri-list", nil, platform.DragMove)
	if !to.dropAt(0, e) {
		t.Fatal("the other list refused the drop")
	}
	if len(to.rows) != wasTo+1 || to.rows[0] != moving {
		t.Errorf("the row did not land at the caret: %v", names(to.rows))
	}
	// The source removes its own row when the target says it moved.
	d.Done(platform.DragMove)
	if len(from.rows) != wasFrom-1 {
		t.Errorf("after a move the source still holds %d rows, want %d", len(from.rows), wasFrom-1)
	}
	for _, r := range from.rows {
		if r == moving {
			t.Error("the row that was moved is still in the list it came from")
		}
	}

	// A copy leaves the original where it was.
	wasFrom = len(from.rows)
	d2 := from.drag([]int{0})
	e2 := d2.DropEvent(paintengine2d.Pt(1, 1), "text/uri-list", nil, platform.DragCopy)
	if !to.dropAt(len(to.rows), e2) {
		t.Fatal("the other list refused a copy")
	}
	d2.Done(platform.DragCopy)
	if len(from.rows) != wasFrom {
		t.Errorf("a copy took %d rows out of the source", wasFrom-len(from.rows))
	}
}

// A drop from another application arrives as files or as text, and is
// read as such.
func TestTourListTakesAForeignDrop(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, win := tourWindow(t, "", 1, pageDrag)
	defer win.Close()
	p := stateOf(t, win).page(pageDrag).(*dragPage)

	was := len(p.right.rows)
	e := widget.NewDropEvent(paintengine2d.Pt(1, 1), "text/uri-list",
		[]byte("file:///tmp/one.txt\r\nfile:///tmp/two.txt\r\n"))
	e.Action = platform.DragCopy
	if !p.right.dropAt(0, e) {
		t.Fatal("the list refused a uri-list from another application")
	}
	if len(p.right.rows) != was+2 {
		t.Fatalf("two files became %d rows", len(p.right.rows)-was)
	}
	if p.right.rows[0].name != "one.txt" {
		t.Errorf("the first row is %q, want the file's name", p.right.rows[0].name)
	}

	was = len(p.left.rows)
	et := widget.NewDropEvent(paintengine2d.Pt(1, 1), "text/plain", []byte("Dropped from a browser"))
	et.Action = platform.DragCopy
	if !p.left.dropAt(len(p.left.rows), et) {
		t.Fatal("the list refused text from another application")
	}
	if len(p.left.rows) != was+1 || p.left.rows[len(p.left.rows)-1].name != "Dropped from a browser" {
		t.Errorf("the text landed as %v", names(p.left.rows))
	}
}

// ---- page four: frames ---------------------------------------------------------------

// Asking for the toolkit's frame and then the desktop's changes what the
// window reports, and the Tabs page's account of the strip follows it.
func TestTourFramesSwitchAndTabsPageFollows(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	win, err := a.NewWindow(platform.WindowOptions{Title: "tour", Width: 1180, Height: 820, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	win.SetContent(TourAppOpen(a, win, pageTabs, pageFrames))
	a.PumpOnce()
	tour := stateOf(t, win)

	// Open both pages so that both are built and listening.
	tour.strip.Select(1)
	a.PumpOnce()
	frames := tour.page(pageFrames).(*framesPage)
	tour.strip.Select(0)
	a.PumpOnce()

	if frames.decor.Selected() != 0 {
		t.Errorf("the page opens on %d, want auto", frames.decor.Selected())
	}
	// An offscreen window is always the desktop's to frame, so what is
	// asserted here is the preference and the account of it, not a frame
	// the headless backend cannot draw.
	frames.decor.Select(2)
	a.PumpOnce()
	if got := a.Decorations(); got != style.DecorationsToolkit {
		t.Errorf("after asking for the toolkit's frame the preference is %q", got)
	}
	frames.decor.Select(1)
	a.PumpOnce()
	if got := a.Decorations(); got != style.DecorationsSystem {
		t.Errorf("after asking for the desktop's frame the preference is %q", got)
	}

	// And a change of pack must not undo it: the tour starts every change
	// from Application.Appearance, which carries the preference through.
	tour.apply(func(ap *style.Appearance) { ap.Name = "win95" })
	a.PumpOnce()
	if got := a.Decorations(); got != style.DecorationsSystem {
		t.Errorf("changing pack reset the frame preference to %q", got)
	}

	// Caption buttons follow the theme when asked.
	frames.caps.Select(1)
	a.PumpOnce()
	if a.CaptionButtons() != style.CaptionButtonsTheme {
		t.Error("the caption buttons did not switch to the theme's layout")
	}
}

// ---- page five: shapes -----------------------------------------------------------------

// Every silhouette the page offers is a real region with an inside, and
// the ones with a hole leave the middle out of it.
func TestTourSilhouettesHaveHoles(t *testing.T) {
	const side = 240
	for kind, name := range tourShapeNames {
		t.Run(strings.Fields(name)[0], func(t *testing.T) {
			s := tourShapeFor(kind, paintengine2d.Pt(side, side))
			if s == nil {
				t.Fatal("no silhouette")
			}
			r := s.Raster(side, side)
			if r == nil || r.Empty() {
				t.Fatal("the silhouette rasterised to nothing")
			}
			if !r.Contains(side/2, 6) {
				t.Error("the top edge is not inside the silhouette")
			}
			holed := kind == 0 || kind == 1
			if got := r.Contains(side/2, side/2); got == holed {
				t.Errorf("the middle is inside=%v; %q should have a hole: %v", got, name, holed)
			}
			if len(r.Clear) == 0 {
				t.Error("no pixels at all are outside the silhouette")
			}
		})
	}
}

// The page opens its pair, and the shaped one is shaped.
func TestTourShapesOpenAPair(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, win := tourWindow(t, "", 1, pageShapes)
	defer win.Close()
	p := stateOf(t, win).page(pageShapes).(*shapesPage)

	was := len(a.Windows())
	p.open()
	a.PumpOnce()
	if len(a.Windows()) != was+2 {
		t.Fatalf("opening the pair made %d windows, want 2", len(a.Windows())-was)
	}
	if p.shaped == nil || p.card == nil {
		t.Fatal("the page did not keep both windows")
	}
	if !p.shaped.ShapeActive() {
		t.Error("the shaped window is not shaped")
	}
	if p.card.ShapeActive() {
		t.Error("the backdrop should be an ordinary rectangle")
	}
	// A press on each face is counted by that window and no other, which
	// is what the page's two counters claim.
	p.shapedFace.MousePress(widget.MouseEvent{Button: platform.ButtonLeft})
	p.cardFace.MousePress(widget.MouseEvent{Button: platform.ButtonLeft})
	p.cardFace.MousePress(widget.MouseEvent{Button: platform.ButtonLeft})
	if p.shapedFace.clicks != 1 || p.cardFace.clicks != 2 {
		t.Errorf("counters read ring=%d card=%d, want 1 and 2", p.shapedFace.clicks, p.cardFace.clicks)
	}

	p.closeWindows()
	a.PumpOnce()
	if p.shaped != nil || p.card != nil {
		t.Error("the pair did not close")
	}
}

// ---- page six: skins ---------------------------------------------------------------------

// The pack list filters, and applying one changes the look every window
// of the application runs in without rebuilding anything.
func TestTourSkinsApplyLive(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, win := tourWindow(t, "luna", 1, pageSkins)
	defer win.Close()
	p := stateOf(t, win).page(pageSkins).(*skinsPage)

	all := len(p.rows)
	if all < 50 {
		t.Fatalf("the list shows %d packs; the toolkit has more than that", all)
	}
	p.only = 1 // skins
	p.reload()
	if len(p.rows) == 0 || len(p.rows) >= all {
		t.Fatalf("the skins filter shows %d of %d packs", len(p.rows), all)
	}
	for _, pack := range p.rows {
		if !style.IsSkin(pack.Name) {
			t.Errorf("%q is in the skins list but is not a skin", pack.Name)
		}
	}

	// A search narrows it further, and an impossible one to nothing.
	p.query = "deck"
	p.reload()
	if len(p.rows) != 1 || p.rows[0].Name != "deck" {
		t.Errorf("searching for deck found %v", packNames(p.rows))
	}

	// Applying it changes the running look, and the tree is the same tree.
	beforeContent := win.Content()
	beforeNodes := countNodes(win.AccessibleTree())
	i := p.list.Selected
	p.choose(i)
	pack, _ := p.selected()
	ap := style.LookAppearance(a.Look())
	ap.Name = pack.Name
	ap.Theme = pack.Palette
	a.ApplyAppearance(ap)
	a.PumpOnce()
	if got := style.LookAppearance(a.Look()).Name; got != pack.Name {
		t.Errorf("the application is in %q, want %q", got, pack.Name)
	}
	if win.Content() != beforeContent {
		t.Error("switching pack rebuilt the window's content; it should only relayout")
	}
	if got := countNodes(win.AccessibleTree()); got != beforeNodes {
		t.Errorf("the accessibility tree went from %d nodes to %d across a theme switch",
			beforeNodes, got)
	}
}

// ---- page eight: access -------------------------------------------------------------------

// The page reads the tree of the window it is in, and says so honestly:
// with the page laid out, the linter has nothing to report.
func TestTourAccessPageReadsItsOwnWindow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "0")
	_, win := tourWindow(t, "", 1, pageAccess)
	defer win.Close()
	p := stateOf(t, win).page(pageAccess).(*accessPage)

	if len(p.found) != 1 || !strings.HasPrefix(p.found[0], "Nothing") {
		t.Errorf("the page reports %d problems with its own window:\n%s",
			len(p.found), strings.Join(p.found, "\n"))
	}
	if !strings.Contains(p.order.Text, "Accessibility tree") {
		t.Errorf("the tab order does not list the page's own tree view:\n%s", p.order.Text)
	}
	if n := countNodes(win.AccessibleTree()); n < 40 {
		t.Errorf("the window describes itself in only %d nodes", n)
	}
}

// ---- the page index -----------------------------------------------------------------------

func TestTourPageNamesRoundTrip(t *testing.T) {
	for i, name := range TourPageNames() {
		if got := TourPageIndex(name); got != i {
			t.Errorf("%q is page %d, want %d", name, got, i)
		}
		if got := TourPageIndex(strings.ToUpper(name)); got != i {
			t.Errorf("%q (upper case) is page %d, want %d", name, got, i)
		}
	}
	for _, short := range []string{"tabs", "dock", "drag", "frames", "shapes", "skins", "desktop", "a11y"} {
		if TourPageIndex(short) < 0 {
			t.Errorf("%q names no page", short)
		}
	}
	if TourPageIndex("nonsense") != -1 {
		t.Error("an unknown name found a page")
	}
	// Every page says what it is and what it proves.
	for i, p := range tourPages {
		if p.name == "" || p.title == "" || p.proof == "" || p.try == "" || p.build == nil {
			t.Errorf("page %d is incomplete: %+v", i, p)
		}
	}
}

// ---- helpers ---------------------------------------------------------------------------------

// stateOf digs the tour's own state out of a window. The tour keeps it in
// the closures of the strip's hooks, so the way in is the strip.
func stateOf(t *testing.T, win *app.Window) *tourState {
	t.Helper()
	s := tourIn(win)
	if s == nil {
		t.Fatalf("this window's content is %T, not a tour", win.Content())
	}
	return s
}

func names(rows []*tourDoc) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.name
	}
	return out
}

func packNames(rows []style.ThemePack) []string {
	out := make([]string, len(rows))
	for i, p := range rows {
		out[i] = p.Name
	}
	return out
}

func windowShowsText(win *app.Window, want string) bool {
	found := false
	widget.Walk(win.Content(), func(c widget.Component) {
		if l, ok := c.(*widgets.Label); ok && l.Text == want {
			found = true
		}
	})
	return found
}

// The status line follows a panel docked back by a drag as it follows the
// buttons: a panel that has left its window does not stay "in a window of
// its own".
func TestTourDockNoteFollowsADrag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, win := tourWindow(t, "", 1, pageDock)
	defer win.Close()
	tour := stateOf(t, win)
	p := tour.page(pageDock).(*dockPage)
	outline := p.host.Panel("outline")

	clickNamed(t, win.Content(), "Float")
	a.PumpOnce()
	if !outline.Floating() || !strings.Contains(tour.status.Parts()[1], "window of its own") {
		t.Fatalf("after Float: floating=%v, the status says %q", outline.Floating(), tour.status.Parts()[1])
	}
	// Back into the host the way a drop does it — the host's own call, no
	// button of the page's involved.
	p.host.DockPanel(outline)
	a.PumpOnce()
	if outline.Floating() {
		t.Fatal("the panel did not dock")
	}
	if got := tour.status.Parts()[1]; strings.Contains(got, "window of its own") || !strings.Contains(got, "docked") {
		t.Errorf("after docking by drag the status says %q", got)
	}
}
