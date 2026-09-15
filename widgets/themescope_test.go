package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A scope's subtree resolves the scope's look (scaled to the host), the
// rest of the window keeps the window's, and switching themes relayouts.
func TestThemeScopeLookResolution(t *testing.T) {
	win95, ok := style.LoadTheme("win95")
	if !ok {
		t.Fatal("win95 pack")
	}
	outside := NewButton("Outside", nil)
	inside := NewButton("Inside", nil)
	scope := NewThemeScope(win95.Look(), NewColumn(inside))
	root := NewColumn(outside, scope)
	h := &fakeWindow{look: style.DarkLook()}
	root.SetHost(h)
	root.Measure(layout.Loose(400, 300))
	root.Arrange(paintengine2d.XYWH(0, 0, 400, 300))

	if outside.Look() != h.Look() {
		t.Fatal("widgets outside the scope must use the window's look")
	}
	got, ok := inside.Look().(*style.Classic)
	if !ok || got.Engine().ID() != "win95" {
		t.Fatalf("widgets inside the scope must use the scope's look, got %T", inside.Look())
	}
	if inside.Look() != scope.Look() {
		t.Fatal("the scope caches its scaled look")
	}
	light, _ := style.LoadTheme("light")
	scope.SetTheme(light.Look())
	if c, _ := inside.Look().(*style.Classic); c == nil || c.Engine().ID() == "win95" {
		t.Fatal("SetTheme must switch the subtree's look")
	}
}
