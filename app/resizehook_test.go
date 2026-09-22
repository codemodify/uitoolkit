package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// TestOnResizeHearsEveryNewSize: the hook hears the size the window is
// first laid out at and every new one after, in logical pixels at every
// scale, before the layout that size brings — and never a size it has
// already heard, nor anything once removed.
func TestOnResizeHearsEveryNewSize(t *testing.T) {
	for _, s := range []float32{1, 1.75} {
		t.Setenv(platform.EnvDecorations, "")
		a := New(Options{Look: style.DarkLook(), Headless: true, Scale: s})
		w, err := a.NewWindow(platform.WindowOptions{Title: "Resize", Width: 400, Height: 300, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		col := widgets.NewColumn(widgets.NewLabel("x"))
		w.SetContent(col)
		var heard [][2]int
		// What the content was arranged at when the hook ran: the layout
		// that follows it is the one at the new size.
		var arrangedAt []float32
		remove := w.OnResize(func(width, height int) {
			heard = append(heard, [2]int{width, height})
			arrangedAt = append(arrangedAt, col.Bounds().Dx())
		})
		a.PumpOnce()
		w.Inject(platform.Event{Kind: platform.EventResize, Width: 640, Height: 480})
		a.PumpOnce()
		// The same size again (a compositor echoing its configure) is not news.
		w.Inject(platform.Event{Kind: platform.EventResize, Width: 640, Height: 480})
		a.PumpOnce()
		want := [][2]int{{400, 300}, {640, 480}}
		if len(heard) != len(want) || heard[0] != want[0] || heard[1] != want[1] {
			t.Fatalf("%.2fx: heard %v, want %v", s, heard, want)
		}
		if arrangedAt[1] >= col.Bounds().Dx() {
			t.Errorf("%.2fx: the hook ran after the layout at the new size (content %v then %v wide)",
				s, arrangedAt[1], col.Bounds().Dx())
		}
		remove()
		w.Inject(platform.Event{Kind: platform.EventResize, Width: 500, Height: 400})
		a.PumpOnce()
		if len(heard) != 2 {
			t.Errorf("%.2fx: a removed hook still heard %v", s, heard[2:])
		}
		w.Close()
	}
}

// TestOnResizeMayReplaceTheContent: a hook that swaps the content is laid
// out in the same frame, at the new size.
func TestOnResizeMayReplaceTheContent(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Swap", Width: 300, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("before"))
	a.PumpOnce()
	after := widgets.NewColumn(widgets.NewLabel("after"))
	w.OnResize(func(width, _ int) {
		if width > 300 {
			w.SetContent(after)
		}
	})
	w.Inject(platform.Event{Kind: platform.EventResize, Width: 420, Height: 200})
	a.PumpOnce()
	if w.Content() != after {
		t.Fatal("the hook's content was not kept")
	}
	if pw, _ := w.PixelSize(); after.Bounds().Dx() != float32(pw) {
		t.Errorf("the new content is %v wide, the window %d", after.Bounds().Dx(), pw)
	}
}
