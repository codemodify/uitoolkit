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

func TestStatusMenuScreenRectAbovePanel(t *testing.T) {
	restore := platform.SimulateScreenRect(platform.FrameRect{W: 1920, H: 1080})
	defer restore()
	gap := platform.ScreenMenuPointerGap
	// A click on a bottom panel: no room for 80 pixels below 1060, so the
	// menu flips above the point, a gap clear of it.
	box := statusMenuScreenRect(100, 1060, 160, 80, 1)
	if box.X != 100+gap || box.Y != 1060-gap-80 {
		t.Fatalf("bottom panel: %+v, want x=%d y=%d", box, 100+gap, 1060-gap-80)
	}
	if box.Y+box.H > 1060 {
		t.Fatalf("menu %+v covers the click at y=1060", box)
	}
	// A click on a top panel: the menu hangs below the point, still clear
	// of it.
	if box := statusMenuScreenRect(100, 20, 160, 80, 1); box.Y != 20+gap {
		t.Fatalf("top panel: %+v, want y=%d", box, 20+gap)
	}
	zero := statusMenuScreenRect(0, 0, 160, 80, 1)
	if zero.X != 0 || zero.Y != 0 {
		t.Fatalf("0,0 means unset, got %+v", zero)
	}
}

// The two faults reported from a real KDE Wayland session (2880x1800
// panel, wl_output.scale 2, fractional scale 175%, so a logical desktop
// of 1645x1029): a tray menu opened at the icon covered the icon, and its
// submenus — which live in the same surface, to the right of the parent —
// were off the right edge of the screen entirely.
func TestStatusMenuScreenRectKDETrayReport(t *testing.T) {
	const (
		screenW, screenH = 1645, 1029 // 2880x1800 at 175%, from xdg_output
		scale            = 1.75
	)
	restore := platform.SimulateScreenRect(platform.FrameRect{W: screenW, H: screenH})
	defer restore()
	// What the session logged: OnMenu(2305,28) in device pixels, a menu
	// surface of 988x221 device = 565x126 logical.
	menuW := platform.LogicalPixels(988, scale)
	menuH := platform.LogicalPixels(221, scale)
	box := statusMenuScreenRect(2305, 28, menuW, menuH, scale)
	if box.X+box.W > screenW || box.Y+box.H > screenH || box.X < 0 || box.Y < 0 {
		t.Fatalf("menu %+v is not inside the %dx%d screen", box, screenW, screenH)
	}
	if box.W != menuW || box.H != menuH {
		t.Fatalf("menu %+v was shrunk; it fits the screen whole", box)
	}
	// It must not cover the icon the tray named.
	px := platform.LogicalPosition(2305, scale)
	py := platform.LogicalPosition(28, scale)
	if px >= box.X && px < box.X+box.W && py >= box.Y && py < box.Y+box.H {
		t.Fatalf("menu %+v covers the tray point %d,%d", box, px, py)
	}
	// Near the right edge there is no room to the right, so it goes to
	// the left of the point — flipped, not slid, which is what keeps the
	// cascade on screen too.
	if box.X >= px {
		t.Fatalf("menu %+v should have flipped left of %d", box, px)
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

func TestStatusMenuToItemsNestsCascades(t *testing.T) {
	picked := 0
	rows := StatusMenuToItems([]platform.StatusMenuItem{
		{Text: "Show Mail"},
		{Separator: true},
		{Text: "Folders", Icon: style.IconMail, Disabled: true, OnClick: func() { picked += 100 },
			Submenu: []platform.StatusMenuItem{
				{Text: "Inbox", OnClick: func() { picked++ }},
				{Separator: true},
				{Text: "Archive", Submenu: []platform.StatusMenuItem{{Text: "2025"}}},
			}},
	})
	if len(rows) != 3 {
		t.Fatalf("rows %d", len(rows))
	}
	folders := rows[2]
	if !folders.HasSubmenu() || folders.Text != "Folders" {
		t.Fatalf("cascade row %+v", folders)
	}
	if !folders.Disabled || folders.Icon != style.IconMail {
		t.Fatalf("cascade row lost its own properties: %+v", folders)
	}
	if folders.OnClick != nil {
		t.Fatal("a parent is not a command: OnClick must not survive")
	}
	if len(folders.Submenu) != 3 {
		t.Fatalf("children %d", len(folders.Submenu))
	}
	if !folders.Submenu[1].Separator {
		t.Fatalf("a separator inside a submenu must survive: %+v", folders.Submenu[1])
	}
	if !folders.Submenu[2].HasSubmenu() || folders.Submenu[2].Submenu[0].Text != "2025" {
		t.Fatalf("grandchild %+v", folders.Submenu[2])
	}
	folders.Submenu[0].OnClick()
	if picked != 1 {
		t.Fatalf("child OnClick %d", picked)
	}
}

func TestStatusMenuFromItemsKeepsCascades(t *testing.T) {
	menu := StatusMenuFromItems([]*widgets.MenuItem{
		widgets.Item("Show Mail", nil),
		widgets.Submenu("Folders",
			widgets.Item("Inbox", nil),
			widgets.Sep(),
		),
	})
	if len(menu) != 2 || len(menu[1].Submenu) != 2 {
		t.Fatalf("round trip %+v", menu)
	}
	if menu[1].Text != "Folders" || !menu[1].Submenu[1].Separator {
		t.Fatalf("cascade %+v", menu[1])
	}
	back := StatusMenuToItems(menu)
	if !back[1].HasSubmenu() || back[1].Submenu[0].Text != "Inbox" {
		t.Fatalf("widgets → tray → widgets lost the cascade: %+v", back[1])
	}
}

func TestStatusMenuWindowLeavesRoomForTheCascade(t *testing.T) {
	host := &statusMeasureHost{look: style.DarkLook(), scale: 1}
	children := []platform.StatusMenuItem{
		{Text: "A folder with a deliberately long name"},
		{Text: "Inbox"},
		{Text: "Drafts"},
	}
	flat := StatusMenuToItems([]platform.StatusMenuItem{
		{Text: "Show Mail"},
		{Text: "Folders"},
		{Text: "Quit"},
	})
	nested := StatusMenuToItems([]platform.StatusMenuItem{
		{Text: "Show Mail"},
		{Text: "Folders", Submenu: children},
		{Text: "Quit"},
	})
	flatW, _ := statusMenuBox(host, flat)
	childW, childH := statusMenuBox(host, StatusMenuToItems(children))
	nestedW, nestedH := statusMenuBox(host, nested)
	// The cascade opens inside this one window and is clamped to it, so
	// the box is the parent menu plus the widest child, not either alone.
	if nestedW < flatW+childW {
		t.Fatalf("cascade box %.0f narrower than parent %.0f + child %.0f", nestedW, flatW, childW)
	}
	if nestedH < childH {
		t.Fatalf("cascade box height %.0f shorter than the child menu %.0f", nestedH, childH)
	}
}

func TestTrayCascadeOpensInsideStatusMenuWindow(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("mail"))
	a.PumpOnce()
	w.Hide()
	a.ShowStatusMenu(300, 900, []platform.StatusMenuItem{
		{Text: "Show Mail"},
		{Text: "Folders", Submenu: []platform.StatusMenuItem{
			{Text: "Inbox"},
			{Separator: true},
			{Text: "Archive"},
		}},
		{Text: "Quit"},
	})
	menu := a.statusMenu
	if menu == nil || !menu.Visible() {
		t.Fatal("status-menu window must be visible")
	}
	pop, ok := menu.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		t.Fatalf("popup %T", menu.Popup())
	}
	if !pop.Items[1].HasSubmenu() {
		t.Fatalf("row 1 lost its cascade: %+v", pop.Items[1])
	}
	menu.RequestFocus(pop)
	menu.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDown})
	menu.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyDown})
	menu.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyRight})
	child := pop.CascadeMenu()
	if child == nil {
		t.Fatal("Right on a cascade row must open the child menu")
	}
	b := child.Bounds()
	ww, hh := menu.SurfaceSize()
	if b.Dx() < 8 || b.Dy() < 8 {
		t.Fatalf("invisible cascade %v", b)
	}
	if b.Min.X < 0 || b.Min.Y < 0 || b.Max.X > float32(ww)+1 || b.Max.Y > float32(hh)+1 {
		t.Fatalf("cascade %v outside the status-menu window %dx%d", b, ww, hh)
	}
}

// A compositor with no zwlr_layer_shell_v1 cannot put the toolkit's menu
// window where the tray clicked, so an item that asks for ToolkitMenu is
// given HostMenu instead — and is told, in the options it ends up with.
func TestStatusMenuChromeFallsBackWithoutPlacement(t *testing.T) {
	defer platform.SimulateScreenPlacement(false)()
	if got, _ := StatusMenuChromeFor(platform.ToolkitMenu); got != platform.HostMenu {
		t.Fatalf("chrome = %v, want HostMenu", got)
	}
	if _, why := StatusMenuChromeFor(platform.ToolkitMenu); why == "" {
		t.Fatal("no reason given for the demotion")
	}
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	item, err := a.NewStatusItem(platform.StatusItemOptions{
		Title:      "Mail",
		MenuChrome: platform.ToolkitMenu,
		Menu:       StatusMenuFromItems([]*widgets.MenuItem{widgets.Item("Quit", func() {})}),
	})
	if err != nil {
		t.Fatal(err)
	}
	fake, ok := item.(*platform.FakeStatusItem)
	if !ok {
		t.Fatalf("backend %s", item.Backend())
	}
	if fake.MenuChrome() != platform.HostMenu {
		t.Fatalf("item opened with %v, want HostMenu: a demoted item must export a real host menu", fake.MenuChrome())
	}
}

// Where the window can be placed — X11, or a compositor with layer shell
// — ToolkitMenu is honoured and nothing changes.
func TestStatusMenuChromeKeptWithPlacement(t *testing.T) {
	defer platform.SimulateScreenPlacement(true)()
	if got, _ := StatusMenuChromeFor(platform.ToolkitMenu); got != platform.ToolkitMenu {
		t.Fatalf("chrome = %v, want ToolkitMenu", got)
	}
	t.Setenv("UITK_TRAY", "fake")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	item, err := a.NewStatusItem(platform.StatusItemOptions{
		Title:      "Mail",
		MenuChrome: platform.ToolkitMenu,
		Menu:       StatusMenuFromItems([]*widgets.MenuItem{widgets.Item("Quit", func() {})}),
		// Its own OnMenu, so the item is not wrapped for menu tracking
		// and the fake underneath can be read straight out.
		OnMenu: func(int32, int32) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake, ok := item.(*platform.FakeStatusItem)
	if !ok {
		t.Fatalf("backend %s", item.Backend())
	}
	if fake.MenuChrome() != platform.ToolkitMenu {
		t.Fatalf("item opened with %v, want ToolkitMenu", fake.MenuChrome())
	}
}

// The default — an app that says nothing — is the desktop's own menu on
// Linux whether or not the compositor can place windows: the fallback
// only ever takes something away from ToolkitMenu.
func TestStatusMenuChromeDefaultUnchanged(t *testing.T) {
	if !platform.HostMenuNative() {
		t.Skip("no native host menu on this OS")
	}
	for _, place := range []bool{true, false} {
		restore := platform.SimulateScreenPlacement(place)
		got, _ := StatusMenuChromeFor(platform.HostMenu)
		restore()
		if got != platform.HostMenu {
			t.Fatalf("placement=%v: default chrome = %v, want HostMenu", place, got)
		}
	}
}

// ShowStatusMenu puts its window where the tray said, on a desktop that
// places windows: the whole path from SNI coordinates through
// statusMenuScreenRect and the logical-pixel conversion to the surface.
func TestShowStatusMenuPlacesItsWindow(t *testing.T) {
	// The offscreen desktop has no monitors to report, so the screen the
	// menu is constrained to is simulated: without one nothing would flip
	// and the bottom-panel rule below could not be checked.
	restore := platform.SimulateScreenRect(platform.FrameRect{W: 1920, H: 1080})
	defer restore()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	a.ShowStatusMenu(900, 1040, StatusMenuFromItems([]*widgets.MenuItem{
		widgets.Item("Raise", func() {}),
		widgets.Sep(),
		widgets.Item("Quit", func() {}),
	}))
	w := a.statusMenu
	if w == nil || w.Closed() {
		t.Fatal("no status-menu window")
	}
	mw, mh := measureStatusMenu(a.Look(), a.Scale(), StatusMenuToItems(StatusMenuFromItems(
		[]*widgets.MenuItem{widgets.Item("Raise", func() {}), widgets.Sep(), widgets.Item("Quit", func() {})})))
	want := statusMenuScreenRect(900, 1040,
		platform.LogicalPixels(mw, a.Scale()), platform.LogicalPixels(mh, a.Scale()), a.Scale())
	wantX, wantY := want.X, want.Y
	x, y, ok := platform.SurfacePosition(w.Surface())
	if !ok {
		t.Fatal("the offscreen desktop does not say where the window is")
	}
	if x != wantX || y != wantY {
		t.Fatalf("status menu at %d,%d, want %d,%d", x, y, wantX, wantY)
	}
	// A bottom-panel click opens the menu above the click, never below.
	if click := platform.LogicalPosition(1040, a.Scale()); wantY >= click {
		t.Fatalf("menu top %d is not above the click at %d", wantY, click)
	}
}
