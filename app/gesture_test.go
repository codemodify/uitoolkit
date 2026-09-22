package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// gestureBox takes the gestures of the kinds in takes and records every one
// it is told about; navs records its history steps.
type gestureBox struct {
	widget.Base
	takes map[platform.GestureKind]bool
	got   []widget.GestureEvent
	nav   bool
	navs  []bool
	press int
}

func newGestureBox(takes ...platform.GestureKind) *gestureBox {
	b := &gestureBox{takes: map[platform.GestureKind]bool{}}
	for _, k := range takes {
		b.takes[k] = true
	}
	b.Init(b)
	return b
}

func (b *gestureBox) Measure(c layout.Constraints) paintengine2d.Point {
	return paintengine2d.Pt(c.MaxW, c.MaxH)
}

func (b *gestureBox) Gesture(e widget.GestureEvent) bool {
	b.got = append(b.got, e)
	return b.takes[e.Kind]
}

func (b *gestureBox) NavigateHistory(forward bool) bool {
	b.navs = append(b.navs, forward)
	return b.nav
}

func (b *gestureBox) MousePress(widget.MouseEvent) bool { b.press++; return true }

// gestureRig is a 400×300 window: an outer box that fills it and an inner
// one on its lower half.
func gestureRig(t *testing.T, outer, inner *gestureBox) *Window {
	t.Helper()
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Gestures", Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	outer.Add(inner)
	w.SetContent(outer)
	a.PumpOnce()
	inner.SetBounds(paintengine2d.XYWH(0, 150, 400, 150))
	return w
}

func gesture(kind platform.GestureKind, phase platform.GesturePhase, x, y float32) platform.Event {
	return platform.Event{Kind: platform.EventGesture, Gesture: kind, Phase: phase, Fingers: 2,
		Pos: paintengine2d.Pt(x, y), Scale: 1}
}

// The begin goes to the component under the pointer, then up, until one
// takes it; that one hears the rest wherever the pointer goes, in its own
// coordinates, and nobody else does.
func TestGestureGoesToWhoTookTheBegin(t *testing.T) {
	outer, inner := newGestureBox(platform.GesturePinch), newGestureBox()
	w := gestureRig(t, outer, inner)
	w.dispatch(gesture(platform.GesturePinch, platform.GestureBegin, 100, 200))
	up := gesture(platform.GesturePinch, platform.GestureUpdate, 100, 50)
	up.Scale, up.Rotation, up.Delta = 1.5, 12, paintengine2d.Pt(3, 4)
	w.dispatch(up)
	w.dispatch(gesture(platform.GesturePinch, platform.GestureEnd, 100, 50))
	// Offered to the inner box first, which declined.
	if len(inner.got) != 1 || inner.got[0].Phase != platform.GestureBegin || inner.got[0].Pos != paintengine2d.Pt(100, 50) {
		t.Fatalf("inner heard %+v", inner.got)
	}
	if len(outer.got) != 3 {
		t.Fatalf("outer heard %d events: %+v", len(outer.got), outer.got)
	}
	g := outer.got[1]
	if g.Phase != platform.GestureUpdate || g.Scale != 1.5 || g.Rotation != 12 || g.Delta != paintengine2d.Pt(3, 4) || g.Fingers != 2 {
		t.Fatalf("update %+v", g)
	}
	if outer.got[2].Phase != platform.GestureEnd {
		t.Fatalf("end %+v", outer.got[2])
	}
	// Over now: a stray update goes nowhere.
	w.dispatch(gesture(platform.GesturePinch, platform.GestureUpdate, 10, 10))
	if len(outer.got) != 3 || len(inner.got) != 1 {
		t.Fatal("an update after the end was delivered")
	}
}

// A gesture nobody takes at its begin goes nowhere; a disabled component
// is passed over; a component taken out mid-gesture hears no more.
func TestGestureNobodyTookIt(t *testing.T) {
	outer, inner := newGestureBox(), newGestureBox(platform.GesturePinch)
	w := gestureRig(t, outer, inner)
	w.dispatch(gesture(platform.GestureSwipe, platform.GestureBegin, 100, 200))
	w.dispatch(gesture(platform.GestureSwipe, platform.GestureUpdate, 100, 200))
	w.dispatch(gesture(platform.GestureSwipe, platform.GestureEnd, 100, 200))
	if len(inner.got) != 1 || len(outer.got) != 1 {
		t.Fatalf("an untaken swipe: inner %d outer %d", len(inner.got), len(outer.got))
	}
	inner.SetEnabled(false)
	w.dispatch(gesture(platform.GesturePinch, platform.GestureBegin, 100, 200))
	if len(inner.got) != 1 {
		t.Fatal("a disabled component was offered a gesture")
	}
	inner.SetEnabled(true)
	w.dispatch(gesture(platform.GesturePinch, platform.GestureBegin, 100, 200))
	outer.Remove(inner)
	w.dispatch(gesture(platform.GesturePinch, platform.GestureUpdate, 100, 200))
	if n := len(inner.got); n != 2 {
		t.Fatalf("a removed component heard %d events", n)
	}
}

// A sideways swipe nobody takes goes back (fingers moving right) or forward
// (left) in the nearest history, once it ends far enough; a short one, a
// mostly vertical one or a cancelled one does not.
func TestSwipeNavigatesHistory(t *testing.T) {
	outer, inner := newGestureBox(), newGestureBox()
	outer.nav = true
	w := gestureRig(t, outer, inner)
	swipe := func(dx, dy float32, end platform.GesturePhase) {
		w.dispatch(gesture(platform.GestureSwipe, platform.GestureBegin, 100, 200))
		for i := 0; i < 4; i++ {
			up := gesture(platform.GestureSwipe, platform.GestureUpdate, 100, 200)
			up.Delta = paintengine2d.Pt(dx/4, dy/4)
			w.dispatch(up)
		}
		w.dispatch(gesture(platform.GestureSwipe, end, 100, 200))
	}
	swipe(160, 10, platform.GestureEnd)
	swipe(-160, -20, platform.GestureEnd)
	swipe(40, 0, platform.GestureEnd)
	swipe(120, 100, platform.GestureEnd)
	swipe(200, 0, platform.GestureCancel)
	// Asked of the inner box first, which went nowhere, then the outer.
	if len(inner.navs) != 2 || len(outer.navs) != 2 || outer.navs[0] != false || outer.navs[1] != true {
		t.Fatalf("inner %v outer %v; want back then forward", inner.navs, outer.navs)
	}
}

// The mouse's back and forward buttons navigate too, and are not clicks:
// nothing is pressed by them.
func TestThumbButtonsNavigate(t *testing.T) {
	outer, inner := newGestureBox(), newGestureBox()
	inner.nav = true
	w := gestureRig(t, outer, inner)
	for _, b := range []platform.MouseButton{platform.ButtonBack, platform.ButtonForward} {
		w.dispatch(platform.Event{Kind: platform.EventMouseDown, Button: b, Pos: paintengine2d.Pt(100, 200)})
		w.dispatch(platform.Event{Kind: platform.EventMouseUp, Button: b, Pos: paintengine2d.Pt(100, 200)})
	}
	if len(inner.navs) != 2 || inner.navs[0] || !inner.navs[1] || len(outer.navs) != 0 {
		t.Fatalf("inner %v outer %v", inner.navs, outer.navs)
	}
	if inner.press != 0 || outer.press != 0 {
		t.Fatal("a thumb button pressed something")
	}
}
