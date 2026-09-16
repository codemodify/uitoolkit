package widgets

import (
	"math"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Tearing a tab out of its window and putting it back into another one:
// the gesture browsers have had for twenty years. A tab dragged along the
// strip is being reordered; dragged clear of the strip it leaves the
// window, into a window of its own that follows the pointer, and dropped
// on another strip it joins that one.
//
// The drag is an ordinary [widget.Drag] carrying a private type
// ([TabMimeType]) so a strip of this application's knows one of its own
// tabs, with whatever the app offers for the document behind it
// (BrowserTabs.OnTabDrag) after it — so the same drag that merges a tab
// into another window drops its folder into a file manager.
//
// The drop half is here too: the strip takes tabs (OnMergeTab) and files
// on a tab (OnDropTab), and marks the two differently — a caret in the
// gap a tab would land in, a highlight on the tab a file would land on.

// TabMimeType is the private type a torn-off tab is offered under. It
// carries the tab's title and nothing else: a drop from another
// application has only that, while a drop from this one gets the whole
// [BrowserTab] in [widget.DropEvent.Payload]. An app with two kinds of
// tab strip gives each its own type with BrowserTabs.TabMime.
const TabMimeType = "application/x-uitoolkit-tab"

// tabMime is the type this strip's tabs are offered and taken under.
func (t *BrowserTabs) tabMime() string {
	if t.TabMime != "" {
		return t.TabMime
	}
	return TabMimeType
}

// ---- out of the strip ----------------------------------------------------------

// tearBand is how far out of the strip a dragged tab has to be pulled
// before it leaves the window: far enough that a wobbling hand goes on
// reordering, close enough that pulling the tab down out of the strip is
// one gesture rather than a journey.
func (t *BrowserTabs) tearBand() float32 { return t.dip(16) }

// tornOut reports whether the tab being dragged has been pulled clear of
// the strip, and tears it off when it has. A strip with one tab keeps it:
// there would be nothing left behind, and dragging a window by its
// caption is what that gesture already means.
func (t *BrowserTabs) tornOut(p paintengine2d.Point) bool {
	if t.OnTearOff == nil || len(t.tabs) < 2 || !t.drag.active {
		return false
	}
	band := t.tearBand()
	b := t.LocalBounds()
	if p.Y >= -band && p.Y <= b.Dy()+band {
		return false
	}
	return t.TearOffTab(t.drag.tab, p)
}

// TearOffTab takes tab i out of the strip and into a window of its own
// (OnTearOff), carrying it under the pointer for as long as the button is
// held: a drop on another window's strip merges it there, a drop on the
// desktop leaves the window where it fell, and Escape puts the tab back.
// at is where the pointer is, in the strip's own coordinates; it is what
// keeps the tab under the pointer as the window is picked up.
//
// It is what a tab dragged out of the strip does, and a menu item ("Move
// Tab to New Window") can call it with the middle of the tab. It reports
// whether the tab left.
func (t *BrowserTabs) TearOffTab(i int, at paintengine2d.Point) bool {
	if t.OnTearOff == nil || i < 0 || i >= len(t.tabs) {
		return false
	}
	tab := t.tabs[i]
	d := t.tabDrag(i, tab)
	if d == nil {
		return false
	}
	// The reorder is over either way: from here the tab is leaving.
	t.drag = tabDrag{}
	t.press = stripHit{tab: -1}
	// Where the desktop carries the window under the pointer the tab
	// leaves now, so the user sees it in the window they are dragging.
	// Where it does not, the window is made at the drop and the tab stays
	// in the strip until then — an empty gap under the pointer for the
	// length of the drag would be worse than none (Chromium's fallback).
	carried := widget.DragsWindows(t)
	if carried {
		t.RemoveTab(i)
	}
	tear := &widget.TearOff{
		// The pointer keeps its place in the new window, so the tab it
		// tore off lands roughly under it.
		Offset: widget.DeviceOrigin(t).Add(at),
		Open:   func() widget.TearOffWindow { return t.OnTearOff(i, tab) },
		Done: func(res widget.TearResult, win widget.TearOffWindow) {
			switch res {
			case widget.TearCancelled:
				// Nothing happened: the window the tab was carried in has
				// no reason to exist and the tab goes back where it was.
				if win != nil {
					win.Close()
				}
				if carried {
					t.InsertTab(i, tab)
					t.Select(i)
				}
			case widget.TearMerged:
				// Another strip holds the tab now.
				if win != nil {
					win.Close()
				}
				fallthrough
			case widget.TearKept:
				if !carried {
					t.RemoveTab(i)
				}
			}
			t.Invalidate()
		},
	}
	if widget.StartTearOff(t, d, tear) {
		return true
	}
	if carried {
		t.InsertTab(i, tab)
		t.Select(i)
	}
	return false
}

// tabDrag is what tab i carries out of the window: the strip's own
// private type first, so another window of this application takes the tab
// whole, and after it whatever the app offers for the document behind it
// — the folder as a uri-list, the page as a URL — so everything else
// takes what it knows.
func (t *BrowserTabs) tabDrag(i int, tab BrowserTab) *widget.Drag {
	d := &widget.Drag{}
	if t.OnTabDrag != nil {
		if got := t.OnTabDrag(i, tab); got != nil {
			d = got
		}
	}
	mime := t.tabMime()
	if !d.Offers(mime) {
		d.Types = append([]string{mime}, d.Types...)
	}
	title := tab.Title
	inner := d.Data
	d.Data = func(m string) ([]byte, bool) {
		if m == mime {
			return []byte(title), true
		}
		if inner != nil {
			return inner(m)
		}
		return nil, false
	}
	// The tab itself rides along in the process, so a strip of ours takes
	// it with its own data untouched.
	d.Payload, d.Source = tab, t
	// A tab moves: it is in one window or the other, never both. An app
	// that also offers the document may have it copied, and a copy leaves
	// the torn-off window standing.
	d.Actions |= platform.DragMove
	d.Preferred = platform.DragMove
	if d.Image == nil {
		d.Image, d.Hotspot = widget.DragLabel(t.Look(), tab.Title, dragScale(t))
	}
	return d
}

// ---- into the strip ------------------------------------------------------------

// DropTypes: the private tab type when the strip takes tabs from other
// windows, and what OnDropTab takes after it.
func (t *BrowserTabs) DropTypes() []string {
	var out []string
	if t.OnMergeTab != nil {
		out = append(out, t.tabMime())
	}
	if t.OnDropTab != nil {
		if len(t.DropMimes) > 0 {
			out = append(out, t.DropMimes...)
		} else {
			out = append(out, "text/uri-list")
		}
	}
	return out
}

// Drop takes a tab dragged out of another window into this strip, at the
// gap the caret marked; anything else goes to the tab under the pointer —
// dropping a file on a folder tab is how a file manager moves it there.
func (t *BrowserTabs) Drop(e widget.DropEvent) bool {
	mime := t.tabMime()
	hit := t.hitAt(e.Pos)
	at := t.insertIndexAt(e.Pos)
	t.clearDropMarks()
	if e.Mime == mime && t.OnMergeTab != nil {
		tab, ok := tabFromDrop(e)
		if !ok {
			return false
		}
		return t.OnMergeTab(at, tab, e)
	}
	if hit.tab < 0 || hit.tab >= t.Len() || t.OnDropTab == nil {
		return false
	}
	return t.OnDropTab(hit.tab, e)
}

// tabFromDrop is the tab a drop carries: the whole tab when the drag
// started in this application, and the title alone when it came from
// another one — the private type carries nothing else, since what a tab
// holds is the app's own. An app that wants more offers a type of its own
// beside it (OnTabDrag) and reads that in OnMergeTab.
func tabFromDrop(e widget.DropEvent) (BrowserTab, bool) {
	if tab, ok := e.Payload.(BrowserTab); ok {
		return tab, true
	}
	title := strings.TrimSpace(string(e.Data))
	if title == "" {
		return BrowserTab{}, false
	}
	return BrowserTab{Title: title}, true
}

// DragOverMime marks what a drop would do: a tab from another window
// lands between two tabs, so the caret goes in that gap; anything else
// lands on a tab, which lights up.
func (t *BrowserTabs) DragOverMime(pos paintengine2d.Point, mime string) {
	if mime != "" && mime == t.tabMime() && t.OnMergeTab != nil {
		t.mark(-1, t.insertIndexAt(pos))
		return
	}
	t.mark(t.hitAt(pos).tab, -1)
}

// DragOver marks the tab under a drag, for a host that does not say what
// is being dragged.
func (t *BrowserTabs) DragOver(pos paintengine2d.Point) { t.mark(t.hitAt(pos).tab, -1) }

func (t *BrowserTabs) DragLeave() { t.clearDropMarks() }

// mark sets the strip's two drag marks at once: the tab a drop would land
// on, and the gap a torn-off tab would go into.
func (t *BrowserTabs) mark(tab, at int) {
	if t.dropTab == tab && t.dropAt == at {
		return
	}
	t.dropTab, t.dropAt = tab, at
	t.Invalidate()
}

func (t *BrowserTabs) clearDropMarks() { t.mark(-1, -1) }

// insertIndexAt is where a tab dropped at local point p would go: before
// the tab whose left half the pointer is over, after the one whose right
// half it is over, and at the end past the last one.
func (t *BrowserTabs) insertIndexAt(p paintengine2d.Point) int {
	g := t.geom()
	for i, s := range g.slots {
		if p.X < (s.Min.X+s.Max.X)*0.5 {
			return i
		}
	}
	return len(g.slots)
}

// caretRect is the gap before tab i, as a bar the width of a caret: at
// the left edge of that tab's slot, or after the last tab.
func (t *BrowserTabs) caretRect(i int, g stripGeom) paintengine2d.Rect {
	w := max(t.dip(2), 2)
	x := g.view.Min.X
	top, h := g.view.Max.Y-g.tabH, g.tabH
	switch {
	case len(g.slots) == 0:
	case i < len(g.slots):
		x = g.slots[i].Min.X
		top, h = g.slots[i].Min.Y, g.slots[i].Dy()
	default:
		last := g.slots[len(g.slots)-1]
		x = last.Max.X - style.TabOverlapOf(t.Look())
		top, h = last.Min.Y, last.Dy()
	}
	x = float32(math.Round(float64(x - w*0.5)))
	return paintengine2d.XYWH(x, top, w, h).Intersect(t.LocalBounds())
}

// paintInsertCaret draws the bar in the gap a torn-off tab would land in.
func (t *BrowserTabs) paintInsertCaret(ctx *paintengine2d.Context, g stripGeom) {
	r := t.caretRect(t.dropAt, g)
	if r.Empty() {
		return
	}
	acc := t.Look().Palette().Accent
	ctx.DrawRect(r, paintengine2d.Fill(acc))
	// A wedge at the top, so the caret reads as an insertion point and
	// not as the edge of a tab.
	w := r.Dx() * 2
	p := paintengine2d.NewPath()
	p.MoveTo(r.Min.X-w, r.Min.Y)
	p.LineTo(r.Max.X+w, r.Min.Y)
	p.LineTo((r.Min.X+r.Max.X)*0.5, r.Min.Y+w*1.5)
	p.Close()
	ctx.DrawPath(p, paintengine2d.Fill(acc))
}
