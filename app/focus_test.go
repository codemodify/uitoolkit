package app

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// A dismissed popup must not keep the keyboard: Enter after Escape used to
// activate a menu item that was no longer on screen.
func TestDismissedPopupLosesKeyboard(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	root := widgets.NewLabel("content")
	w.SetContent(root)
	a.PumpOnce()

	fired := 0
	item := &widgets.MenuItem{Text: "Delete everything", OnClick: func() { fired++ }}
	pop := widgets.ShowContextMenu(root, paintengine2d.Pt(20, 20), item)
	if pop == nil {
		t.Fatal("no popup")
	}
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDown})
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	if w.Popup() != nil {
		t.Fatal("escape should dismiss the popup")
	}
	if widget.LiveUnder(w.Focus(), w.Content()) == false && w.Focus() != nil {
		t.Fatalf("focus left on a detached node: %T", w.Focus())
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyReturn})
	if fired != 0 {
		t.Fatalf("dismissed menu item activated %d times", fired)
	}
}

// Replacing the content must drop focus, hover and the caret wake that went
// with it.
func TestSetContentDropsStaleFocus(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	tf := widgets.NewTextField("", "", nil)
	w.SetContent(tf)
	a.PumpOnce()
	w.RequestFocus(tf)
	if !w.wantsBlink() || !a.anyCaret() {
		t.Fatal("focused field should want a caret")
	}
	w.SetContent(widgets.NewLabel("replaced"))
	a.PumpOnce()
	if w.Focus() == widget.Component(tf) {
		t.Fatal("focus stayed on the detached field")
	}
	if w.wantsBlink() || a.anyCaret() {
		t.Fatal("detached field still drives the caret blink wake")
	}
	w.dispatch(platform.Event{Kind: platform.EventText, Rune: 'x'})
	if tf.Text != "" {
		t.Fatalf("typing reached the detached field: %q", tf.Text)
	}
}

// Removing the hovered widget must send it MouseExit and forget it.
func TestSetContentClearsHover(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	btn := widgets.NewButton("hover me", nil)
	w.SetContent(btn)
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(20, 20)})
	if w.hover == nil {
		t.Fatal("expected a hovered widget")
	}
	w.SetContent(widgets.NewLabel("gone"))
	if w.hover != nil {
		t.Fatalf("hover kept a detached widget: %T", w.hover)
	}
	if btn.Hovered() {
		t.Fatal("detached widget never got MouseExit")
	}
}

// Losing window focus must not leave a widget stuck in its hover state.
func TestFocusOutClearsHover(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	btn := widgets.NewButton("hover me", nil)
	w.SetContent(btn)
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(20, 20)})
	if !btn.Hovered() {
		t.Fatal("expected hover")
	}
	w.dispatch(platform.Event{Kind: platform.EventFocusOut})
	if btn.Hovered() || w.hover != nil {
		t.Fatal("hover survived focus loss")
	}
}

// A modal overlay contains the keyboard: text must not leak into the field
// underneath it.
func TestModalOverlayContainsKeyboard(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 320, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	under := widgets.NewTextField("", "", nil)
	w.SetContent(under)
	a.PumpOnce()
	w.RequestFocus(under)

	w.SetOverlay(widgets.NewLabel("modal"))
	a.PumpOnce()
	w.dispatch(platform.Event{Kind: platform.EventText, Rune: 'x'})
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyA})
	if under.Text != "" {
		t.Fatalf("text leaked past the modal overlay: %q", under.Text)
	}
	w.dispatch(platform.Event{Kind: platform.EventIMECommit, Text: "y"})
	if under.Text != "" {
		t.Fatalf("IME commit leaked past the modal overlay: %q", under.Text)
	}
}

// A key handled by nothing in the popup must not then be delivered to the
// popup a second time via the focus chain, nor leak to the content behind.
func TestPopupKeyDispatchedOnce(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	behind := widgets.NewTextField("", "", nil)
	w.SetContent(behind)
	a.PumpOnce()
	w.RequestFocus(behind)

	counter := &keyCounter{}
	counter.Init(counter)
	counter.SetWantsFocus(true)
	counter.Arrange(paintengine2d.XYWH(0, 0, 100, 40))
	w.SetPopup(counter)
	w.RequestFocus(counter)
	a.PumpOnce()

	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyA})
	if counter.keys != 1 {
		t.Fatalf("popup saw the key %d times, want 1", counter.keys)
	}
	w.dispatch(platform.Event{Kind: platform.EventText, Rune: 'q'})
	if behind.Text != "" {
		t.Fatalf("text leaked behind the popup: %q", behind.Text)
	}
}

type keyCounter struct {
	widget.Base
	keys int
}

func (k *keyCounter) KeyPress(widget.KeyEvent) bool {
	k.keys++
	return false
}

// Escape and a click mean "no tip for this hover": the bubble must not come
// straight back when the delay elapses again without pointer motion.
func TestTooltipStaysDismissed(t *testing.T) {
	for _, how := range []string{"escape", "click"} {
		t.Run(how, func(t *testing.T) {
			a := New(Options{Look: style.DarkLook(), Headless: true})
			w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 200, Headless: true})
			if err != nil {
				t.Fatal(err)
			}
			btn := widgets.NewButton("OK", nil)
			btn.Tip = "help"
			w.SetContent(btn)
			now := time.Unix(1000, 0)
			w.SetClock(func() time.Time { return now })
			a.PumpOnce()

			w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(20, 20)})
			now = now.Add(time.Second)
			a.PumpOnce()
			if w.Tooltip() == nil {
				t.Fatal("tooltip should be up")
			}
			if how == "escape" {
				w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
			} else {
				w.dispatch(platform.Event{Kind: platform.EventMouseDown, Pos: paintengine2d.Pt(20, 20), Button: platform.ButtonLeft})
				w.dispatch(platform.Event{Kind: platform.EventMouseUp, Pos: paintengine2d.Pt(20, 20), Button: platform.ButtonLeft})
			}
			if w.Tooltip() != nil {
				t.Fatal("tooltip should be down")
			}
			now = now.Add(2 * time.Second)
			a.PumpOnce()
			if w.Tooltip() != nil {
				t.Fatal("tooltip came back without pointer motion")
			}
			if _, ok := w.tipDeadline(now); ok {
				t.Fatal("a dismissed tooltip must not schedule a wake")
			}
			// Real motion re-arms it.
			w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(24, 22)})
			now = now.Add(time.Second)
			a.PumpOnce()
			if w.Tooltip() == nil {
				t.Fatal("moving the pointer should re-arm the tooltip")
			}
		})
	}
}

// The window tracks the Alt key for mnemonic underlines; losing focus
// (Alt+Tab) forgets it.
func TestWindowTracksAlt(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("x"))
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyAlt, Mods: platform.ModAlt})
	if !w.AltHeld() {
		t.Fatal("Alt down")
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyUp, Key: platform.KeyAlt})
	if w.AltHeld() {
		t.Fatal("Alt up")
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyAlt, Mods: platform.ModAlt})
	w.dispatch(platform.Event{Kind: platform.EventFocusOut})
	if w.AltHeld() {
		t.Fatal("focus out forgets Alt")
	}
}

// In a look that underlines mnemonics only while Alt is held (XP), the
// frame after Alt must redraw them: a full present alone replayed the
// retained menu bar, so no underline showed on a live compositor.
func TestAltRedrawsMnemonicUnderlines(t *testing.T) {
	p, ok := style.LoadTheme("luna")
	if !ok || style.LookHint(p.Look(), style.HintMnemonics) != style.MnemonicsOnAlt {
		t.Fatal("luna underlines mnemonics on Alt")
	}
	a := New(Options{Look: p.Look(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewMenuBar(
		widgets.NewMenu("&File", widgets.Item("&Open", nil)),
		widgets.NewMenu("&Edit", widgets.Item("&Copy", nil)),
	))
	w.frame()
	idle := w.surf.Buffer().Clone()
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyAlt, Mods: platform.ModAlt})
	w.frame()
	held := w.surf.Buffer().Clone()
	if diffPixels(t, idle, held) == 0 {
		t.Fatal("holding Alt left the menu bar without its underlines")
	}
	if n := diffPixels(t, held, fullRepaint(w)); n != 0 {
		t.Fatalf("the frame with Alt held differs from a full repaint in %d pixels", n)
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyUp, Key: platform.KeyAlt})
	w.frame()
	if n := diffPixels(t, idle, w.surf.Buffer().Clone()); n != 0 {
		t.Fatalf("releasing Alt left %d pixels changed", n)
	}
}
