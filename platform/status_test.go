package platform

import (
	"os"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestFakeStatusItemRecordsCalls(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	clicked := 0
	item, err := NewStatusItem(StatusItemOptions{
		ID: "test", Title: "Test", Tooltip: "tip",
		OnClick: func() { clicked++ },
		Menu: []StatusMenuItem{
			{Text: "One", OnClick: func() { clicked += 10 }},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake, ok := item.(*FakeStatusItem)
	if !ok {
		t.Fatalf("backend %s", item.Backend())
	}
	if fake.Backend() != "fake" || !fake.Alive() {
		t.Fatalf("fake %+v", fake)
	}
	if err := fake.SetTooltip("hello"); err != nil || fake.Tooltip() != "hello" {
		t.Fatalf("tooltip %q", fake.Tooltip())
	}
	if err := fake.SetTitle("Mail"); err != nil || fake.Title() != "Mail" {
		t.Fatalf("title %q", fake.Title())
	}
	img := paintengine2d.NewImage(16, 16)
	if err := fake.SetIcon(StatusIcon{Name: "mail-unread", Image: img}); err != nil {
		t.Fatal(err)
	}
	if fake.Icon().Name != "mail-unread" {
		t.Fatalf("icon %+v", fake.Icon())
	}
	if err := fake.Notify(Notification{Title: "Ada", Body: "Hello"}); err != nil {
		t.Fatal(err)
	}
	if len(fake.Notes) != 1 || fake.Notes[0].Title != "Ada" {
		t.Fatalf("notes %+v", fake.Notes)
	}
	fake.Click()
	fake.ClickMenu(0)
	if clicked != 11 {
		t.Fatalf("clicks %d", clicked)
	}
	if err := fake.Close(); err != nil || fake.Alive() {
		t.Fatal("close should drop Alive")
	}
}

func TestStubStatusItemNoPanic(t *testing.T) {
	t.Setenv("UITK_TRAY", "stub")
	item, err := NewStatusItem(StatusItemOptions{Title: "x", OnClick: func() {}})
	if err != nil {
		t.Fatal(err)
	}
	if item.Backend() != "stub" || item.Alive() {
		t.Fatalf("stub %s alive=%v", item.Backend(), item.Alive())
	}
	_ = item.SetIcon(StatusIcon{Name: "x"})
	_ = item.SetTooltip("t")
	_ = item.SetTitle("t")
	_ = item.SetMenu(nil)
	_ = item.Notify(Notification{Title: "n", Body: "b"})
	_ = item.Close()
}

func TestStatusItemStubOption(t *testing.T) {
	os.Unsetenv("UITK_TRAY")
	item, err := NewStatusItem(StatusItemOptions{Stub: true, Title: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Backend() != "stub" {
		t.Fatalf("want stub, got %s", item.Backend())
	}
}

func TestOffscreenHostWindow(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 80, Height: 40})
	if !o.Visible() {
		t.Fatal("mapped by default")
	}
	o.Hide()
	if o.Visible() {
		t.Fatal("hidden")
	}
	o.Show()
	o.Raise()
	if !o.Visible() {
		t.Fatal("shown")
	}
}

func TestWakeSurfaceOffscreenNoPanic(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 40, Height: 20})
	before := o.Wakes()
	WakeSurface(o)
	if o.Wakes() <= before {
		t.Fatal("WakeSurface should count on Offscreen")
	}
	WakeSurface(nil)
	MoveSurface(o, 10, 20)
	MoveSurface(nil, 0, 0)
}

func TestMenuRowsEqual(t *testing.T) {
	a := []StatusMenuItem{{Text: "Show"}, {Separator: true}, {Text: "Quit"}}
	b := []StatusMenuItem{{Text: "Show"}, {Separator: true}, {Text: "Quit"}}
	if !menuRowsEqual(a, b) {
		t.Fatal("equal")
	}
	b[0].Text = "Show Mail"
	if menuRowsEqual(a, b) {
		t.Fatal("text change")
	}
	if !menuRowsEqual(nil, []StatusMenuItem{}) {
		t.Fatal("nil and empty are the same menu")
	}
}

func TestStatusIconNameAndClickable(t *testing.T) {
	if statusIconName(StatusIcon{Name: "mail-unread"}) != "mail-unread" {
		t.Fatal("keep explicit name")
	}
	if statusIconName(StatusIcon{}) != "application-default-icon" {
		t.Fatal("generic fallback, not mail-unread")
	}
	if menuItemClickable(StatusMenuItem{Separator: true, OnClick: func() {}}) {
		t.Fatal("separator")
	}
	if menuItemClickable(StatusMenuItem{Text: "x", Disabled: true, OnClick: func() {}}) {
		t.Fatal("disabled")
	}
	if !menuItemClickable(StatusMenuItem{Text: "x", OnClick: func() {}}) {
		t.Fatal("enabled")
	}
}

func TestFakeItemIsMenuClickDoesNotOnClick(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	n, opened := 0, 0
	item, err := NewStatusItem(StatusItemOptions{
		Title:      "Mail",
		ItemIsMenu: true,
		OnClick:    func() { n++ },
		OnMenu:     func(x, y int32) { opened++ },
	})
	if err != nil {
		t.Fatal(err)
	}
	item.(*FakeStatusItem).Click()
	if n != 0 || opened != 1 {
		t.Fatalf("ItemIsMenu Click n=%d opened=%d", n, opened)
	}
}

func TestShowRaiseAfterHideMakesVisible(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 80, Height: 40})
	HideSurface(o)
	if SurfaceVisible(o) {
		t.Fatal("hidden")
	}
	RaiseSurface(o)
	if !SurfaceVisible(o) {
		t.Fatal("Show/Raise after hide must set Visible")
	}
}

func TestFakeDbusMenuClickInvokesOnClick(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	n := 0
	item, err := NewStatusItem(StatusItemOptions{
		Title: "Mail",
		Menu: []StatusMenuItem{
			{Text: "Show Mail", OnClick: func() { n++ }},
			{Separator: true},
			{Text: "Quit", OnClick: func() { n += 10 }},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := item.(*FakeStatusItem)
	fake.ClickMenu(0)
	if n != 1 {
		t.Fatalf("Show Mail click %d", n)
	}
	fake.ClickMenu(2)
	if n != 11 {
		t.Fatalf("Quit click %d", n)
	}
}

func TestSNIPixmapIsStraightAlpha(t *testing.T) {
	// A 50% transparent pure red pixel: premultiplied storage is
	// (0x80,0,0,0x80); the SNI wire format wants straight ARGB, so the
	// red channel must come back at full strength.
	img := paintengine2d.NewImage(1, 1)
	img.SetColor(0, 0, paintengine2d.RGBA(1, 0, 0, 0.5))
	w, h, pix := sniARGB(img)
	if w != 1 || h != 1 || len(pix) != 4 {
		t.Fatalf("sniARGB = %d×%d, %d bytes", w, h, len(pix))
	}
	a, r, g, b := pix[0], pix[1], pix[2], pix[3]
	if a < 0x7c || a > 0x84 {
		t.Fatalf("alpha = %#02x, want ~0x80", a)
	}
	if r < 0xf0 {
		t.Fatalf("red = %#02x, want ~0xff (straight alpha, not premultiplied)", r)
	}
	if g != 0 || b != 0 {
		t.Fatalf("green/blue = %#02x/%#02x, want 0", g, b)
	}
}

func TestUnpremul(t *testing.T) {
	cases := []struct{ c, a, want uint8 }{
		{0, 0, 0},
		{0xff, 0xff, 0xff},
		{0x80, 0x80, 0xff}, // fully saturated at half alpha
		{0x40, 0x80, 0x80},
		{0x00, 0x80, 0x00},
	}
	for _, c := range cases {
		if got := unpremul(c.c, c.a); got != c.want {
			t.Fatalf("unpremul(%#02x,%#02x) = %#02x, want %#02x", c.c, c.a, got, c.want)
		}
	}
}

func TestSNIPixmapOpaqueRoundTrip(t *testing.T) {
	img := paintengine2d.NewImage(2, 1)
	img.SetColor(0, 0, paintengine2d.RGBA(0, 1, 0, 1))
	img.SetColor(1, 0, paintengine2d.RGBA(0, 0, 1, 1))
	_, _, pix := sniARGB(img)
	if pix[0] != 0xff || pix[2] < 0xf0 {
		t.Fatalf("opaque green = %v", pix[:4])
	}
	if pix[4] != 0xff || pix[7] < 0xf0 {
		t.Fatalf("opaque blue = %v", pix[4:8])
	}
}

func TestFakeStatusItemKeepsSubmenuTree(t *testing.T) {
	t.Setenv("UITK_TRAY", "fake")
	clicks := map[string]int{}
	hit := func(name string) func() { return func() { clicks[name]++ } }
	item, err := NewStatusItem(StatusItemOptions{
		Title: "Mail",
		Menu: []StatusMenuItem{
			{Text: "Show Mail", OnClick: hit("show")},
			{Text: "Folders", OnClick: hit("folders"), Submenu: []StatusMenuItem{
				{Text: "Inbox", OnClick: hit("inbox")},
				{Separator: true},
				{Text: "Archive", Submenu: []StatusMenuItem{
					{Text: "2025", OnClick: hit("2025")},
				}},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake, ok := item.(*FakeStatusItem)
	if !ok {
		t.Fatalf("backend %T", item)
	}
	menu := fake.Menu()
	if len(menu) != 2 || len(menu[1].Submenu) != 3 {
		t.Fatalf("the fake must record the tree as given: %+v", menu)
	}
	if !menu[1].Submenu[1].Separator || len(menu[1].Submenu[2].Submenu) != 1 {
		t.Fatalf("submenu rows %+v", menu[1].Submenu)
	}
	fake.ClickMenuPath(1, 0)
	fake.ClickMenuPath(1, 2, 0)
	if clicks["inbox"] != 1 || clicks["2025"] != 1 {
		t.Fatalf("nested clicks %v", clicks)
	}
	// A parent is not a command, and a separator never was.
	fake.ClickMenuPath(1)
	fake.ClickMenuPath(1, 1)
	if clicks["folders"] != 0 {
		t.Fatalf("cascade parent fired OnClick: %v", clicks)
	}
	// The snapshot is a copy: mutating it cannot reach the live menu.
	menu[1].Submenu[0].Text = "gone"
	if fake.Menu()[1].Submenu[0].Text != "Inbox" {
		t.Fatal("Menu() must hand out a deep copy")
	}
}
