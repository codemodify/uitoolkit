package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// The toolkit had two shapes and needed three. ToolIconBtn is an icon on
// the *tool* face — flat, with no frame until hovered in most eras — so
// it does not read as a button; Button.Icon is for a button with a name
// and leaves an empty label centred with the mark to one side. This is
// the third: the look's push-button face with a mark in the middle.
func TestIconButtonIsSquareAndNamed(t *testing.T) {
	b := NewIconButton(style.IconDownload, "Fetch", nil)
	atScale(t, b, 1)
	sz := b.Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	if sz.X != sz.Y {
		t.Errorf("measured %v — a row of these should be a row of squares", sz)
	}
	if got := b.Tooltip(); got != "Fetch" {
		t.Errorf("tooltip %q", got)
	}
	if got := b.AccessibleName(); got != "Fetch" {
		t.Errorf("accessible name %q — an icon alone cannot be read aloud", got)
	}
	// The two cannot drift apart.
	b.SetAction("Get mail")
	if b.Tooltip() != "Get mail" || b.AccessibleName() != "Get mail" {
		t.Errorf("after SetAction: tip %q name %q", b.Tooltip(), b.AccessibleName())
	}
}

// A latched button has to look latched. 53 of the 135 packs draw a
// checked button exactly as an ordinary one, so on those it is drawn
// pressed instead — which is how Windows 3.1, Motif, CDE and OPEN LOOK
// drew a toggle in the first place.
func TestLatchedButtonLooksLatchedEverywhere(t *testing.T) {
	packs := style.ListThemes()
	if len(packs) == 0 {
		t.Skip("no packs in this build")
	}
	for _, p := range packs {
		lk := style.Appearance{Name: p.Name, Theme: p.Palette}.Look()
		plain := style.LatchedState(lk, 0)
		latched := style.LatchedState(lk, style.StateChecked|style.StateToggle)
		if plain == latched {
			t.Errorf("%s: a latched button is drawn exactly as an ordinary one", p.Name)
		}
		if !style.CheckedFaceOf(lk) && !latched.Pressed() {
			t.Errorf("%s: the engine does not mark Checked and nothing was added", p.Name)
		}
	}
}

// A menu button opens on press, not release: the menu has to be under
// the pointer that opened it so a drag can run straight into it.
func TestMenuButtonOpensOnPress(t *testing.T) {
	clicked := 0
	b := NewMenuButton(style.IconMenu, "Menu",
		&MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: func() { clicked++ }})
	h := atScale(t, b, 1)
	_ = h
	sz := b.Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	b.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))

	if b.IsOpen() {
		t.Fatal("open before anything happened")
	}
	// Without a real popup host the menu cannot actually show; what is
	// pinned here is that the press is what tries, and the release does
	// not close what the press opened.
	b.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(4, 4), Button: platform.ButtonLeft})
	if !b.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(4, 4)}) {
		t.Error("the release should be swallowed, not treated as a click")
	}
}

// The one that bites: an application menu moved from a MenuBar to a
// Button loses its accelerators, and nothing says so — Ctrl+Q simply
// stops quitting.
func TestMenuButtonOwnsItsAccelerators(t *testing.T) {
	quit := 0
	b := NewMenuButton(style.IconMenu, "Menu",
		&MenuItem{Text: "Settings"},
		&MenuItem{Separator: true},
		&MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: func() { quit++ }})
	atScale(t, b, 1)

	if _, ok := any(b).(interface {
		HandleAccelerator(platform.Key, platform.Modifiers) bool
	}); !ok {
		t.Fatal("a menu button must answer the window's accelerator walk")
	}
	if !b.HandleAccelerator(platform.KeyQ, platform.ModCtrl) {
		t.Fatal("Ctrl+Q was not handled")
	}
	if quit != 1 {
		t.Errorf("Quit ran %d times", quit)
	}
	// And a shortcut nothing owns is not swallowed.
	if b.HandleAccelerator(platform.KeyW, platform.ModCtrl) {
		t.Error("an unrelated shortcut was swallowed")
	}
}

// A submenu's shortcuts count too, and a disabled item's does not.
func TestMenuButtonAcceleratorsReachSubmenus(t *testing.T) {
	deep, off := 0, 0
	b := NewMenuButton(style.IconMore, "More",
		&MenuItem{Text: "View", Submenu: []*MenuItem{
			{Text: "Zoom in", Shortcut: "Ctrl+Shift+Z", OnClick: func() { deep++ }},
		}},
		&MenuItem{Text: "Never", Shortcut: "Ctrl+N", Disabled: true, OnClick: func() { off++ }})
	atScale(t, b, 1)

	if !b.HandleAccelerator(platform.KeyZ, platform.ModCtrl|platform.ModShift) || deep != 1 {
		t.Errorf("a submenu's shortcut did not run: deep=%d", deep)
	}
	if b.HandleAccelerator(platform.KeyN, platform.ModCtrl) || off != 0 {
		t.Error("a disabled item's shortcut ran")
	}
}

// Build is asked at each opening, for a menu that depends on what is
// true when it opens.
func TestMenuButtonBuildsItsItems(t *testing.T) {
	n := 0
	b := NewMenuButton(style.IconMenu, "Menu")
	b.Build = func() []*MenuItem {
		n++
		return []*MenuItem{{Text: "Quit", Shortcut: "Ctrl+Q"}}
	}
	atScale(t, b, 1)
	b.HandleAccelerator(platform.KeyQ, platform.ModCtrl)
	if n == 0 {
		t.Error("Build was never asked")
	}
}

// A menu button is as often "File" as it is a hamburger. It embeds
// IconButton, which is square by definition, so a named one came out
// squeezed into a square showing "F…".
func TestNamedMenuButtonIsNotSquashed(t *testing.T) {
	mark := NewMenuButton(style.IconMenu, "Menu", &MenuItem{Text: "Quit"})
	named := NewTextMenuButton("Selection", &MenuItem{Text: "Select All"})
	atScale(t, mark, 1)
	atScale(t, named, 1)

	loose := layout.Constraints{MaxW: -1, MaxH: -1}
	m := mark.Measure(loose)
	if m.X != m.Y {
		t.Errorf("a mark-only menu button measured %v, want a square", m)
	}
	n := named.Measure(loose)
	if n.X <= n.Y {
		t.Errorf("a named menu button measured %v — it has no room for its word", n)
	}
	if n.X <= m.X {
		t.Errorf("%q measured %v, no wider than a bare mark at %v", named.Text, n.X, m.X)
	}
}
