package widgets

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// colourAt is one pixel of img as r+g+b.
func colourAt(img *paintengine2d.Image, x, y int) int {
	r, g, b, _ := img.PremulAt(x, y)
	return int(r) + int(g) + int(b)
}

// Aero's buttons glow in over the look's fade time instead of switching;
// a press is instant; Win95 never fades.
func TestHoverCrossFades(t *testing.T) {
	pack, ok := style.LoadTheme("aero")
	if !ok {
		t.Fatal("no aero pack")
	}
	h := &timerHost{look: pack.Look(), now: time.Unix(2000, 0)}
	defer func(old func() time.Time) { fadeNow = old }(fadeNow)
	fadeNow = func() time.Time { return h.now }

	b := NewButton("Glow", nil)
	b.SetHost(h)
	b.Arrange(paintengine2d.XYWH(0, 0, 120, 32))
	const x, y = 60, 8 // the face, clear of the label
	idle := colourAt(immediatePaint(b, 120, 32), x, y)

	b.MouseEnter()
	b.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(x, y)})
	start := colourAt(immediatePaint(b, 120, 32), x, y)
	h.advance(80 * time.Millisecond)
	mid := colourAt(immediatePaint(b, 120, 32), x, y)
	h.advance(300 * time.Millisecond)
	hot := colourAt(immediatePaint(b, 120, 32), x, y)
	if idle == hot {
		t.Fatal("Aero's hover should change the face")
	}
	if start != idle {
		t.Errorf("the fade should start from the idle face: %d, idle %d", start, idle)
	}
	if !(mid != idle && mid != hot && (mid-idle)*(hot-mid) > 0) {
		t.Errorf("80ms in, the face %d should sit between idle %d and hot %d", mid, idle, hot)
	}

	// Win95 switches at once.
	pack95, _ := style.LoadTheme("win95")
	b95 := NewButton("Flat", nil)
	h95 := &timerHost{look: pack95.Look(), now: h.now}
	b95.SetHost(h95)
	b95.Arrange(paintengine2d.XYWH(0, 0, 120, 32))
	immediatePaint(b95, 120, 32)
	if f := style.LookHint(pack95.Look(), style.HintHoverFadeMs); f != 0 {
		t.Fatalf("Win95 fades %dms", f)
	}

	// UITK_ANIMATIONS=0: instant.
	t.Setenv(AnimationsEnv, "0")
	b2 := NewButton("Glow", nil)
	b2.SetHost(h)
	b2.Arrange(paintengine2d.XYWH(0, 0, 120, 32))
	immediatePaint(b2, 120, 32)
	b2.MouseEnter()
	b2.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(x, y)})
	if got := colourAt(immediatePaint(b2, 120, 32), x, y); got != hot {
		t.Errorf("with animations off the hover should show at once: %d, want %d", got, hot)
	}
}

// Only hover and focus fade: a press shows at once.
func TestPressNeverFades(t *testing.T) {
	var f stateFade
	h := &timerHost{now: time.Unix(3000, 0)}
	pack, _ := style.LoadTheme("aero")
	h.look = pack.Look()
	b := NewButton("x", nil)
	b.SetHost(h)
	defer func(old func() time.Time) { fadeNow = old }(fadeNow)
	fadeNow = func() time.Time { return h.now }
	img := paintengine2d.NewImage(4, 4)
	ctx := paintengine2d.NewContext(img)
	var drawn []style.ControlState
	draw := func(_ *paintengine2d.Context, st style.ControlState) { drawn = append(drawn, st) }
	r := paintengine2d.XYWH(0, 0, 4, 4)
	f.paint(b, ctx, r, style.StateHovered, draw)
	drawn = nil
	f.paint(b, ctx, r, style.StateHovered|style.StatePressed, draw)
	if len(drawn) != 1 || drawn[0] != style.StateHovered|style.StatePressed {
		t.Fatalf("a press should paint once in its own state, painted %v", drawn)
	}
}

// A busy bar animates itself while painted; Manual and UITK_ANIMATIONS=0
// hold it at the app's phase.
func TestBusyBarAnimatesItself(t *testing.T) {
	h := &timerHost{look: style.DarkLook(), now: time.Unix(4000, 0)}
	defer func(old func() time.Time) { fadeNow = old }(fadeNow)
	fadeNow = func() time.Time { return h.now }
	p := NewBusyBar(0.25)
	p.SetHost(h)
	p.Arrange(paintengine2d.XYWH(0, 0, 160, 14))
	a := p.phase()
	h.advance(400 * time.Millisecond)
	b := p.phase()
	if a != 0.25 || b <= a || b >= 1 {
		t.Fatalf("phase %v then %v, want 0.25 then a quarter cycle on", a, b)
	}
	if len(h.queue) == 0 {
		t.Fatal("a painted busy bar should ask for its next frame")
	}
	p.Manual = true
	if got := p.phase(); got != 0.25 {
		t.Fatalf("a manual bar shows the app's phase, got %v", got)
	}
	p.Manual = false
	t.Setenv(AnimationsEnv, "0")
	if got := p.phase(); got != 0.25 {
		t.Fatalf("animations off: phase %v", got)
	}
}

// A flat tool button's hover box fades out too: its idle state paints no
// box, so painting it over the hot state would leave the box up.
func TestToolHoverFadesOut(t *testing.T) {
	pack, _ := style.LoadTheme("aero")
	h := &timerHost{look: pack.Look(), now: time.Unix(5000, 0)}
	defer func(old func() time.Time) { fadeNow = old }(fadeNow)
	fadeNow = func() time.Time { return h.now }
	tb := NewToolBar(&ToolItem{Text: "Cut"}, &ToolItem{Text: "Copy"})
	tb.SetHost(h)
	tb.Arrange(paintengine2d.XYWH(0, 0, 200, 34))
	r := tb.itemRects()[0]
	x, y := int(r.Min.X)+3, int(r.Min.Y)+4 // the box's edge, clear of the label
	idle := colourAt(immediatePaint(tb, 200, 34), x, y)
	tb.MouseMove(widget.MouseEvent{Pos: r.Center()})
	immediatePaint(tb, 200, 34)
	h.advance(time.Second)
	hot := colourAt(immediatePaint(tb, 200, 34), x, y)
	if hot == idle {
		t.Fatal("hovering the tool should show its box")
	}
	tb.MouseExit()
	immediatePaint(tb, 200, 34)
	h.advance(60 * time.Millisecond)
	mid := colourAt(immediatePaint(tb, 200, 34), x, y)
	if mid == hot || mid == idle || (mid-idle)*(hot-mid) < 0 {
		t.Errorf("fading out, the box %d should sit between hot %d and idle %d", mid, hot, idle)
	}
	h.advance(time.Second)
	if got := colourAt(immediatePaint(tb, 200, 34), x, y); got != idle {
		t.Errorf("after the fade the box should be gone: %d, idle %d", got, idle)
	}
}

// Aqua's default button throbs in an active window: it swells toward its
// hover look and back over the look's period. Not in an inactive window,
// not under the pointer, not in Win95.
func TestDefaultButtonPulses(t *testing.T) {
	pack, _ := style.LoadTheme("aqua")
	h := &timerHost{look: pack.Look(), now: time.Unix(6000, 0)}
	defer func(old func() time.Time) { fadeNow = old }(fadeNow)
	fadeNow = func() time.Time { return h.now }
	b := NewButton("OK", nil)
	b.Primary = true
	b.SetHost(h)
	b.Arrange(paintengine2d.XYWH(0, 0, 100, 30))
	st := b.PaintState()
	seen := map[int]bool{}
	for i := 0; i < 8; i++ {
		p, ok := b.pulse(st)
		if !ok || p < 0 || p > 1 {
			t.Fatalf("pulse %v ok=%v", p, ok)
		}
		seen[int(p*10)] = true
		h.advance(200 * time.Millisecond)
	}
	if len(seen) < 3 {
		t.Fatalf("the pulse should sweep over its period, saw %v", seen)
	}
	if _, ok := b.pulse(st | style.StateBackdrop); ok {
		t.Error("an inactive window's default button is still")
	}
	if _, ok := b.pulse(st | style.StateHovered); ok {
		t.Error("under the pointer the default button shows its hover look")
	}
	win95, _ := style.LoadTheme("win95")
	b.SetLook(win95.Look())
	if _, ok := b.pulse(st); ok {
		t.Error("Win95's default button does not pulse")
	}
}
