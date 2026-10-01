package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// An application that asks for nothing gets the look, whole.
//
// There are three ways to build a window with this toolkit and they are
// not a ladder: a themed application whose controls the look arranges, an
// application that states part of its own chrome (a browser's tab strip,
// an editor's title row), and a skin, where the art and the hit areas are
// a file. The second and third are *opt-in* — a developer who writes no
// extra line gets the first, under every pack, for ever.
//
// This holds that promise as a test, because it is the kind that erodes
// one convenient default at a time.
func TestEverythingCustomIsOptedInto(t *testing.T) {
	a := New(Options{Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "plain", Width: 600, Height: 400})
	if err != nil {
		t.Fatal(err)
	}

	// The window wears the look's own frame and caption.
	if got := w.CaptionStyle(); got != style.CaptionFollowsLook {
		t.Errorf("caption style is %v, want the look's own", got)
	}
	if w.Borderless() {
		t.Error("a plain window dropped the look's border")
	}
	if w.TitleBar() != nil {
		t.Error("a plain window has a title bar of its own")
	}

	// No chrome tint: the look's surfaces are the look's.
	plain := New(Options{Headless: true}).Look()
	if lk, ok := w.Look().(*style.Classic); ok {
		if p, ok2 := plain.(*style.Classic); ok2 {
			zero := paintengine2d.Color{}
			for _, k := range []string{"titleBar", "toolBar", "sidebar", "window"} {
				if lk.X(k, zero) != p.X(k, zero) {
					t.Errorf("a plain application's %q differs from the pack's", k)
				}
			}
		}
	}

	// Controls wear the look's faces, not the flat ones.
	if b := widgets.NewIconButton(style.IconNew, "New", nil); b.Flat {
		t.Error("an icon button is flat by default")
	}
	if it := widgets.ToolWidget(widgets.NewLabel("x")); it.Grow || it.Stretch {
		t.Error("a tool item takes the bar's spare width by default")
	}
	if l := widgets.NewLabel("x"); l.Icon != style.IconNone {
		t.Error("a label has a mark by default")
	}
	if f := widgets.NewTextField("", "", nil); f.Frameless {
		t.Error("a text field is frameless by default")
	}
}
