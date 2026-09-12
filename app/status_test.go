package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestHeadlessStatusItemIsStub(t *testing.T) {
	t.Setenv("UITK_TRAY", "")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	item, err := a.NewStatusItem(platform.StatusItemOptions{Title: "Mail"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Backend() != "stub" || item.Alive() {
		t.Fatalf("headless must stub, got %s alive=%v", item.Backend(), item.Alive())
	}
}

func TestFakeStatusItemClickShowsWindow(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("mail"))
	a.PumpOnce()
	shown := 0
	item, err := a.NewStatusItem(platform.StatusItemOptions{
		Title: "Mail",
		OnClick: func() {
			shown++
			w.Show()
			w.Raise()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake, ok := item.(*platform.FakeStatusItem)
	if !ok {
		t.Fatalf("backend %s", item.Backend())
	}
	w.Hide()
	if w.Visible() {
		t.Fatal("hidden")
	}
	fake.Click()
	if shown != 1 || !w.Visible() {
		t.Fatalf("shown=%d visible=%v", shown, w.Visible())
	}
}

func TestShowRaiseAfterCloseToTray(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("mail"))
	a.PumpOnce()
	w.SetCloseHides(true)
	w.dispatch(platform.Event{Kind: platform.EventClose})
	if w.Visible() || w.Closed() {
		t.Fatal("close-to-tray should hide")
	}
	w.Show()
	w.Raise()
	if !w.Visible() {
		t.Fatal("Show/Raise after hide")
	}
}

func TestApplicationPostRunsImmediatelyOutsideLoop(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	n := 0
	a.Post(func() { n++ })
	if n != 1 {
		t.Fatalf("Post outside Run should be sync, n=%d", n)
	}
}

func TestCloseHidesKeepsWindow(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()
	w.SetCloseHides(true)
	w.dispatch(platform.Event{Kind: platform.EventClose})
	if w.Closed() {
		t.Fatal("close-to-tray must not destroy")
	}
	if w.Visible() {
		t.Fatal("should hide")
	}
	w.Show()
	if !w.Visible() {
		t.Fatal("show")
	}
}

func TestStatusMenuFromItems(t *testing.T) {
	n := 0
	got := StatusMenuFromItems([]*widgets.MenuItem{
		widgets.Item("Show", func() { n++ }),
		widgets.Sep(),
	})
	if len(got) != 2 || got[0].Text != "Show" || !got[1].Separator {
		t.Fatalf("%+v", got)
	}
	got[0].OnClick()
	if n != 1 {
		t.Fatal("onclick")
	}
}

func TestStatusIconFromToolPaints(t *testing.T) {
	icon := StatusIconFromTool(style.IconInfo, style.DarkLook(), 22)
	if icon.Image == nil || icon.Image.Width != 22 {
		t.Fatalf("image %+v", icon.Image)
	}
	n := 0
	for y := 0; y < icon.Image.Height; y++ {
		for x := 0; x < icon.Image.Width; x++ {
			_, _, _, a := icon.Image.PremulAt(x, y)
			if a > 10 {
				n++
			}
		}
	}
	if n < 20 {
		t.Fatalf("expected tray ink, n=%d", n)
	}
}
