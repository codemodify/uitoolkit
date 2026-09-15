package widgets

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// timerHost is a host with a hand-driven clock: AfterFunc callbacks run
// when advance passes their time.
type timerHost struct {
	host
	look  style.LookAndFeel
	now   time.Time
	queue []pendingTimer
}

type pendingTimer struct {
	at   time.Time
	fn   func()
	dead *bool
}

func (h *timerHost) Look() style.LookAndFeel { return h.look }

func (h *timerHost) AfterFunc(d time.Duration, fn func()) func() {
	dead := new(bool)
	h.queue = append(h.queue, pendingTimer{at: h.now.Add(d), fn: fn, dead: dead})
	return func() { *dead = true }
}

// advance moves the clock by d, running every callback that falls due
// (including ones scheduled while advancing).
func (h *timerHost) advance(d time.Duration) {
	end := h.now.Add(d)
	for {
		sort.SliceStable(h.queue, func(i, j int) bool { return h.queue[i].at.Before(h.queue[j].at) })
		if len(h.queue) == 0 || h.queue[0].at.After(end) {
			break
		}
		t := h.queue[0]
		h.queue = h.queue[1:]
		h.now = t.at
		if !*t.dead {
			t.fn()
		}
	}
	h.now = end
}

var _ widget.Timers = (*timerHost)(nil)

// Transient bars (libadwaita, WinUI) float over the content: the rows keep
// the full width, the bar shows only while the view scrolls or the pointer
// moves over it, and fades out after a second of rest.
func TestTransientScrollBarComesAndGoes(t *testing.T) {
	pack, ok := style.LoadTheme("adwaita")
	if !ok {
		t.Fatal("no adwaita pack")
	}
	h := &timerHost{look: pack.Look(), now: time.Unix(1000, 0)}
	defer func(old func() time.Time) { scrollNow = old }(scrollNow)
	scrollNow = func() time.Time { return h.now }

	lv := NewListView(200, func(i int) string { return fmt.Sprintf("item-%03d", i) }, nil)
	lv.SetHost(h)
	const w, hh = 180, 120
	lv.Arrange(paintengine2d.XYWH(0, 0, w, hh))
	if lv.MaxOffset() <= 0 {
		t.Fatal("the list should overflow")
	}
	if got, want := lv.rowsW(), lv.inner().Dx(); got != want {
		t.Fatalf("rows are %v wide under a transient bar, want the full %v", got, want)
	}

	// The bar's ink: pixels along the right edge that differ from the rows.
	barInk := func() int {
		img := immediatePaint(lv, w, hh)
		sp := lv.vparts()
		bar := fromView(sp.Bar, lv.frame())
		ref := img
		n := 0
		for y := int(bar.Min.Y) + 2; y < int(bar.Max.Y)-2; y++ {
			// Compare the bar column with the row a little to its left.
			for x := int(bar.Min.X); x < int(bar.Max.X); x++ {
				ar, ag, ab, _ := img.PremulAt(x, y)
				br, bg, bb, _ := ref.PremulAt(int(bar.Min.X)-8, y)
				if absInt(int(ar)-int(br))+absInt(int(ag)-int(bg))+absInt(int(ab)-int(bb)) > 12 {
					n++
				}
			}
		}
		return n
	}

	if n := barInk(); n != 0 {
		t.Fatalf("an untouched view shows its transient bar (%d px)", n)
	}
	lv.ScrollTo(60)
	if n := barInk(); n == 0 {
		t.Fatal("scrolling should show the bar")
	}
	h.advance(500 * time.Millisecond)
	if n := barInk(); n == 0 {
		t.Fatal("the bar left before its rest was over")
	}
	h.advance(2 * time.Second)
	if n := barInk(); n != 0 {
		t.Fatalf("the bar should have faded out after a second of rest (%d px)", n)
	}

	// Pointer motion over the view shows it; a pointer on the bar keeps it.
	sp := lv.vparts()
	onBar := fromView(sp.Thumb, lv.frame()).Center()
	lv.MouseMove(widget.MouseEvent{Pos: onBar})
	if n := barInk(); n == 0 {
		t.Fatal("the pointer over the view should show the bar")
	}
	h.advance(5 * time.Second)
	if n := barInk(); n == 0 {
		t.Fatal("a hovered bar must stay")
	}
	lv.MouseExit()
	h.advance(3 * time.Second)
	if n := barInk(); n != 0 {
		t.Fatalf("after the pointer left, the bar should fade (%d px)", n)
	}
}

func TestTransientBarsTakeNoGutter(t *testing.T) {
	for pack, transient := range map[string]bool{"adwaita": true, "fluent": true, "breeze": false, "win95": false, "adwaita-gtk3": false} {
		p, ok := style.LoadTheme(pack)
		if !ok {
			t.Fatalf("no %s pack", pack)
		}
		lk := p.Look()
		if got := style.ScrollBarStyleOf(lk).Transient; got != transient {
			t.Errorf("%s: transient %v, want %v", pack, got, transient)
		}
		if g := style.ScrollGutter(lk); (g == 0) != transient {
			t.Errorf("%s: gutter %v with transient %v", pack, g, transient)
		}
	}
}
