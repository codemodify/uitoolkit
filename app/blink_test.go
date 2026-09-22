package app

import (
	"testing"
	"time"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func blinkWindow(t *testing.T) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	tf := widgets.NewTextField("hello", "", nil)
	w.SetContent(tf)
	a.PumpOnce()
	w.RequestFocus(tf)
	return a, w
}

// A caret stops blinking a while after the last input, lit, and the loop
// then has nothing to wake for; the next input starts it again.
func TestCaretBlinkTimesOut(t *testing.T) {
	a, w := blinkWindow(t)
	now := time.Now()
	if !a.blinking(now) {
		t.Fatal("a focused field blinks after input")
	}
	if !a.blinking(now.Add(caretBlinkTimeout - time.Second)) {
		t.Fatal("stopped before the timeout")
	}
	w.blink = false
	if a.blinking(now.Add(caretBlinkTimeout)) {
		t.Fatal("still blinking after the timeout with no input")
	}
	w.steadyCaret()
	if !w.CaretBlink() {
		t.Fatal("the caret stopped dark")
	}
	if d := a.waitTimeout(now, time.Time{}); d != -1 {
		t.Fatalf("idle caret still wakes the loop in %v", d)
	}
	w.dispatch(platform.Event{Kind: platform.EventText, Rune: 'x'})
	if !a.blinking(now.Add(caretBlinkTimeout + time.Second)) {
		t.Fatal("input did not start the blink again")
	}
}

// An inactive window's caret does not blink.
func TestCaretBlinkNeedsActiveWindow(t *testing.T) {
	a, w := blinkWindow(t)
	w.setActive(false)
	if w.wantsBlink() || a.blinking(time.Now()) {
		t.Fatal("an inactive window blinks its caret")
	}
	w.setActive(true)
	if !a.blinking(time.Now()) {
		t.Fatal("no blink once active again")
	}
}

// Reduced motion keeps the caret still.
func TestCaretBlinkReducedMotion(t *testing.T) {
	a, _ := blinkWindow(t)
	style.SetReduceMotion(true)
	defer style.SetReduceMotion(false)
	if a.blinking(time.Now()) {
		t.Fatal("the caret blinks with reduced motion")
	}
}
