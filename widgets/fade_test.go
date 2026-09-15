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
	f.paint(b, ctx, style.StateHovered, draw)
	drawn = nil
	f.paint(b, ctx, style.StateHovered|style.StatePressed, draw)
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
