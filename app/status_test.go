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
	a.DrainPosted() // tray callbacks are queued for the UI goroutine
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

func TestApplicationPostNeverRunsOnCallerGoroutine(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	n := 0
	a.Post(func() { n++ })
	if n != 0 {
		t.Fatalf("Post must not run fn inline, n=%d", n)
	}
	if a.Looping() {
		t.Fatal("Looping outside Run")
	}
	a.DrainPosted()
	if n != 1 {
		t.Fatalf("DrainPosted should run queued fn, n=%d", n)
	}
	// PumpOnce drains too, so a headless pump loop needs no special care.
	a.Post(func() { n++ })
	a.PumpOnce()
	if n != 2 {
		t.Fatalf("PumpOnce should drain, n=%d", n)
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

func fakeStatus(item platform.StatusItem) *platform.FakeStatusItem {
	if f, ok := item.(*platform.FakeStatusItem); ok {
		return f
	}
	if t, ok := item.(*trackingStatusItem); ok {
		if f, ok := t.StatusItem.(*platform.FakeStatusItem); ok {
			return f
		}
	}
	return nil
}

func TestHostMenuDoesNotOpenToolkitPopup(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("mail"))
	a.PumpOnce()
	item, err := a.NewStatusItem(platform.StatusItemOptions{
		Title:      "Mail",
		MenuChrome: platform.HostMenu,
		Menu: []platform.StatusMenuItem{
			{Text: "Show Mail"},
			{Separator: true},
			{Text: "Quit"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := fakeStatus(item)
	if fake == nil {
		t.Fatalf("backend %T", item)
	}
	fake.ContextClick(80, 80)
	if a.statusMenu != nil {
		t.Fatal("HostMenu must not open a toolkit popup")
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
		Title:      "Mail",
		MenuChrome: platform.ToolkitMenu,
		Menu: []platform.StatusMenuItem{
			{Text: "Show Mail", Icon: style.IconMail, OnClick: func() { picked++ }},
			{Separator: true},
			{Text: "Quit"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := fakeStatus(item)
	if fake == nil {
		t.Fatalf("backend %T %s", item, item.Backend())
	}
	w.Hide()
	fake.ContextClick(380, 8)
	a.DrainPosted() // tray callbacks are queued for the UI goroutine
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
	if menu.Closed() {
		t.Fatal("activate must hide, not destroy, the status-menu window")
	}
	if menu.Visible() {
		t.Fatal("activate should hide the status-menu window")
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
	if menu.Closed() {
		t.Fatal("Escape should hide, not destroy, the status-menu window")
	}
	if menu.Visible() {
		t.Fatal("Escape should hide the status-menu window")
	}
	if a.statusMenu != menu {
		t.Fatal("status-menu window should be reused")
	}
	if w.Visible() {
		t.Fatal("main window must stay hidden")
	}
}

func TestShowStatusMenuReusesWindow(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("mail"))
	a.PumpOnce()
	before := len(a.Windows())
	rows := []platform.StatusMenuItem{{Text: "Show Mail"}, {Separator: true}, {Text: "Quit"}}
	a.ShowStatusMenu(80, 80, rows)
	first := a.statusMenu
	if first == nil || first.Closed() {
		t.Fatal("first ShowStatusMenu")
	}
	afterFirst := len(a.Windows())
	if afterFirst != before+1 {
		t.Fatalf("windows %d → %d", before, afterFirst)
	}
	a.hideStatusMenu()
	if first.Closed() || first.Visible() {
		t.Fatal("hide must keep the window, unmapped")
	}
	a.ShowStatusMenu(90, 90, rows)
	if a.statusMenu != first {
		t.Fatal("second ShowStatusMenu must reuse the window")
	}
	if len(a.Windows()) != afterFirst {
		t.Fatalf("window leak: %d vs %d", len(a.Windows()), afterFirst)
	}
	if !first.Visible() {
		t.Fatal("reused menu must be visible")
	}
}

func TestStatusMenuFocusOutIgnoresUntilArmed(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("mail"))
	a.PumpOnce()
	a.ShowStatusMenu(80, 80, []platform.StatusMenuItem{{Text: "Quit"}})
	menu := a.statusMenu
	if menu == nil {
		t.Fatal("status menu")
	}
	menu.dispatch(platform.Event{Kind: platform.EventFocusOut})
	if !menu.Visible() || menu.Closed() {
		t.Fatal("FocusOut before arm must not dismiss")
	}
	menu.statusMenuArmed = true
	menu.dispatch(platform.Event{Kind: platform.EventFocusOut})
	if menu.Visible() {
		t.Fatal("FocusOut after arm should hide")
	}
	if menu.Closed() {
		t.Fatal("FocusOut must not destroy the reused window")
	}
}

func TestApplicationPostWakesDisplay(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 80, Height: 40, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()
	o, ok := w.Surface().(*platform.Offscreen)
	if !ok {
		t.Fatalf("surface %T", w.Surface())
	}
	a.mu.Lock()
	a.looping = true
	a.mu.Unlock()
	beforeSurf := o.Wakes()
	beforeLoop := platform.LoopWakes()
	ran := 0
	a.Post(func() { ran++ })
	if ran != 0 {
		t.Fatal("Post during loop must queue")
	}
	if o.Wakes() <= beforeSurf && platform.LoopWakes() <= beforeLoop {
		t.Fatal("Post should wake the display")
	}
	a.runPosted()
	if ran != 1 {
		t.Fatalf("runPosted %d", ran)
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
