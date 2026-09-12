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
