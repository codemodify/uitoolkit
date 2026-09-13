package app

import (
	"sync"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func runAsync(t *testing.T, a *Application) chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- a.Run() }()
	return done
}

func waitRun(t *testing.T, done chan error, why string) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s: %v", why, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Run did not return: %s", why)
	}
}

// Closing the last window must end Run even when nothing else wakes the
// loop (no caret, no tooltip, no look watcher): the idle wait is otherwise
// indefinite and the process hangs after its window is gone.
func TestRunReturnsWhenLastWindowCloses(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 120, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("bye"))
	w.Inject(platform.Event{Kind: platform.EventClose})
	waitRun(t, runAsync(t, a), "close last window")
	if !w.Closed() {
		t.Fatal("window should be closed")
	}
}

// Quit from another goroutine must wake an idle loop, not wait for the next
// display event.
func TestQuitWakesIdleLoop(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 120, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("hi"))
	done := runAsync(t, a)
	time.Sleep(80 * time.Millisecond)
	a.Quit()
	waitRun(t, done, "quit")
	if !a.Quitting() {
		t.Fatal("Quitting should report the request")
	}
}

// A quit flag left over from a previous Run must not make the next Run
// return immediately.
func TestRunClearsPreviousQuit(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 120, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("hi"))
	a.Quit()
	if !a.Quitting() {
		t.Fatal("Quit should set the flag")
	}
	done := runAsync(t, a)
	time.Sleep(60 * time.Millisecond)
	if a.Quitting() {
		t.Fatal("Run should clear a stale quit request")
	}
	a.Quit()
	waitRun(t, done, "second run")
}

// A hidden status-menu window is toolkit chrome, not an application window:
// it must neither hold the run loop open nor survive the last real window.
func TestHiddenStatusMenuDoesNotHoldRunLoop(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("main"))
	a.PumpOnce()
	a.ShowStatusMenu(10, 10, []platform.StatusMenuItem{{Text: "Quit", OnClick: func() {}}})
	a.PumpOnce()
	if a.statusMenu == nil {
		t.Fatal("expected a status-menu window")
	}
	a.hideStatusMenu()
	if a.statusMenu.Visible() {
		t.Fatal("status menu should be hidden")
	}
	if n := a.aliveWindows(); n != 1 {
		t.Fatalf("hidden status menu counted as alive: %d windows hold the loop", n)
	}
	w.Inject(platform.Event{Kind: platform.EventClose})
	waitRun(t, runAsync(t, a), "close last real window with a hidden status menu")
	for _, win := range a.Windows() {
		if !win.Closed() {
			t.Fatalf("window %p survived the loop", win)
		}
	}
}

// EventClose is a request, not a fact: a close-to-tray window must stay
// alive and keep working after the window manager's close button.
func TestCloseToTrayVetoesClose(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetCloseHides(true)
	btn := widgets.NewButton("OK", nil)
	w.SetContent(btn)
	a.PumpOnce()

	w.dispatch(platform.Event{Kind: platform.EventClose})
	if w.Closed() {
		t.Fatal("close-to-tray window was destroyed by a close request")
	}
	if w.Visible() {
		t.Fatal("close-to-tray window should be hidden")
	}
	if w.surf.Closed() {
		t.Fatal("surface must stay alive for a vetoed close")
	}
	// Still a live window: it can be shown again and still paints.
	w.Show()
	if !w.Visible() {
		t.Fatal("hidden window should come back with Show")
	}
	btn.Invalidate()
	a.PumpOnce()
	if w.needsPaint() {
		t.Fatal("re-shown window did not repaint")
	}
}

// A surface the display server destroyed is reaped, and that is the only
// thing Closed() may be read as.
func TestDeadSurfaceIsReaped(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 120, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("x"))
	w.SetCloseHides(true)
	_ = w.surf.Close() // server-side death
	waitRun(t, runAsync(t, a), "dead surface")
	if !w.Closed() {
		t.Fatal("a destroyed surface must reap its window even with closeHides")
	}
}

// Close must not leave the window pointing at its widget tree.
func TestCloseReleasesTree(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 120, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	tf := widgets.NewTextField("x", "", nil)
	w.SetContent(tf)
	a.PumpOnce()
	w.RequestFocus(tf)
	w.Close()
	if w.Content() != nil || w.Focus() != nil {
		t.Fatalf("closed window still holds content=%v focus=%v", w.Content(), w.Focus())
	}
	if w.layers.Len() != 0 {
		t.Fatalf("closed window kept %d recorded groups", w.layers.Len())
	}
	// Must not panic or repaint.
	w.frame()
	w.dispatch(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(5, 5)})
}

// Quit and Run must not race on the quit flag (go test -race).
func TestQuitFromAnotherGoroutineIsRaceFree(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 120, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("x"))
	done := runAsync(t, a)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = a.Quitting()
			a.Quit()
		}()
	}
	wg.Wait()
	waitRun(t, done, "concurrent quit")
}

// Removing the focused subtree without a layer swap must still drop the
// dangling focus at the next layout.
func TestRemovedSubtreeLosesFocus(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 240, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	tf := widgets.NewTextField("", "", nil)
	col := widgets.NewColumn(widgets.NewLabel("keep"), tf)
	w.SetContent(col)
	a.PumpOnce()
	w.RequestFocus(tf)
	if w.Focus() == nil {
		t.Fatal("field should be focused")
	}
	col.Remove(tf)
	w.RequestLayout()
	a.PumpOnce()
	if w.Focus() != nil {
		t.Fatalf("focus survived subtree removal: %T", w.Focus())
	}
	w.dispatch(platform.Event{Kind: platform.EventText, Rune: 'z'})
	if tf.Text != "" {
		t.Fatalf("typing reached the removed field: %q", tf.Text)
	}
}
