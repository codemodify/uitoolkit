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

func TestTrayContextMenuIsToolkitPopup(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("mail"))
	a.PumpOnce()
	picked := 0
	item, err := a.NewStatusItem(platform.StatusItemOptions{
		Title: "Mail",
		Menu: []platform.StatusMenuItem{
			{Text: "Show Mail", Icon: style.IconMail, OnClick: func() { picked++ }},
			{Separator: true},
			{Text: "Quit"},
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
	fake.ContextClick(380, 8)
	if w.Visible() {
		t.Fatal("context menu must not raise the main window")
	}
	menu := a.statusMenu
	if menu == nil || menu.Closed() || !menu.Visible() {
		t.Fatalf("want visible status-menu window, got %+v", menu)
	}
	pop, ok := menu.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		t.Fatalf("want toolkit PopupMenu, got %T", menu.Popup())
	}
	if len(pop.Items) != 3 || pop.Items[0].Text != "Show Mail" || pop.Items[0].Icon != style.IconMail {
		t.Fatalf("popup items %+v", pop.Items)
	}
	b := pop.Bounds()
	mw, mh := menu.SurfaceSize()
	if b.Min.X < 0 || b.Min.Y < 0 || b.Max.X > float32(mw)+1 || b.Max.Y > float32(mh)+1 {
		t.Fatalf("popup %v outside %dx%d", b, mw, mh)
	}
	if b.Dx() < 8 || b.Dy() < 8 {
		t.Fatalf("invisible popup %v", b)
	}
	menu.RequestFocus(pop)
	menu.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyReturn})
	if picked != 1 {
		t.Fatalf("popup activate %d", picked)
	}
	if !menu.Closed() {
		t.Fatal("activate should dismiss the status-menu window")
	}
	if w.Visible() {
		t.Fatal("activating Show Mail test hook must not show main")
	}
}

func TestShowStatusMenuLargeScreenCoordsStayInside(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("mail"))
	a.PumpOnce()
	w.Hide()
	a.ShowStatusMenu(3840, 2160, []platform.StatusMenuItem{
		{Text: "Show Mail"},
		{Separator: true},
		{Text: "Quit"},
	})
	if w.Visible() {
		t.Fatal("large screen coords must not show the main window")
	}
	menu := a.statusMenu
	if menu == nil || !menu.Visible() {
		t.Fatal("status-menu window must be visible")
	}
	pop, ok := menu.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		t.Fatalf("popup %T", menu.Popup())
	}
	b := pop.Bounds()
	ww, hh := menu.SurfaceSize()
	if b.Min.X < 0 || b.Min.Y < 0 || b.Max.X > float32(ww)+1 || b.Max.Y > float32(hh)+1 {
		t.Fatalf("popup %v outside %dx%d", b, ww, hh)
	}
	if b.Dx() < 8 || b.Dy() < 8 {
		t.Fatalf("invisible popup %v", b)
	}
}

func TestStatusMenuOriginIgnoresScreenCoords(t *testing.T) {
	p := statusMenuOrigin(400, 240, 3840, 2160)
	if p.X < 0 || p.Y < 0 || p.X > 400 || p.Y > 240 {
		t.Fatalf("origin %+v", p)
	}
	local := statusMenuOrigin(400, 240, 40, 20)
	if local.X != 40 || local.Y != 20 {
		t.Fatalf("in-window coords should stay, got %+v", local)
	}
}

func TestStatusMenuScreenPosAbovePanel(t *testing.T) {
	x, y := statusMenuScreenPos(100, 1060, 160, 80)
	if x != 100 || y != 980 {
		t.Fatalf("want 100,980 got %d,%d", x, y)
	}
	zx, zy := statusMenuScreenPos(0, 0, 160, 80)
	if zx != 0 || zy != 0 {
		t.Fatalf("0,0 means unset, got %d,%d", zx, zy)
	}
}

func TestStatusMenuEscapeDismissesWindow(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("mail"))
	a.PumpOnce()
	w.Hide()
	a.ShowStatusMenu(80, 80, []platform.StatusMenuItem{{Text: "Quit"}})
	menu := a.statusMenu
	if menu == nil {
		t.Fatal("status menu")
	}
	menu.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEscape})
	if !menu.Closed() || a.statusMenu != nil {
		t.Fatal("Escape should close the status-menu window")
	}
	if w.Visible() {
		t.Fatal("main window must stay hidden")
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
