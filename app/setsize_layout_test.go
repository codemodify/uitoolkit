package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// Asking for a size leaves the layout to be done again, whether or not the
// window system says anything about it.
//
// It used not to, and the silent case is the one that bit: X11's Resize sizes
// the backing image at once and then asks the server, and the ConfigureNotify
// that comes back becomes an EventResize only where the native size differs
// from that image — so a window manager that granted the request *exactly*
// produced no event at all. The cached geometry stayed with it, and with it
// Size, PixelSize, WindowRect and every hit test: a window asked for 700x450
// went on reporting 520x340 until the application called RequestLayout itself.
//
// Checked on the flag rather than through a backend, because no backend here
// can be made to answer a request exactly and silently on purpose — which is
// also why the bug survived three releases.
func TestAskingForASizeLeavesTheLayoutToBeDoneAgain(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Size", Width: 520, Height: 340, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewColumn(widgets.NewLabel("body")))
	a.PumpOnce()
	if !w.laid {
		t.Fatal("the window was not laid out by its first pump, so this proves nothing")
	}

	w.SetSize(700, 450)

	if w.laid {
		t.Error("the window still believes its layout is current after being resized")
	}
	a.PumpOnce()
	if gw, gh := w.Size(); gw != 700 || gh != 450 {
		t.Errorf("the window reports %dx%d after being sized to 700x450", gw, gh)
	}
}

// A request for the size it already has changes nothing, so the layout is not
// thrown away for it: a window that is asked for its own size every frame —
// an application restoring a saved geometry on a timer — must not re-lay out
// every frame.
func TestAskingForTheSizeItAlreadyHasDoesNotRelayout(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Size", Width: 520, Height: 340, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewColumn(widgets.NewLabel("body")))
	a.PumpOnce()
	gw, gh := w.Size()

	w.SetSize(gw, gh)

	if !w.laid {
		t.Error("the layout was thrown away for a size the window already had")
	}
	a.PumpOnce()
	if w2, h2 := w.Size(); w2 != gw || h2 != gh {
		t.Errorf("the window is %dx%d after being asked for the %dx%d it had", w2, h2, gw, gh)
	}
}
