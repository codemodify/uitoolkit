package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// scrollable is a view with more content than box, parked at the top.
func scrollable(t *testing.T) *ScrollView {
	t.Helper()
	s := NewScrollView(newSized(200, 1000))
	s.SetLook(style.DarkLook())
	s.SetHost(&host{})
	s.Measure(layout.Loose(200, 200))
	s.Arrange(paintengine2d.XYWH(0, 0, 200, 200))
	if s.MaxOffset() <= 0 {
		t.Fatalf("the fixture does not scroll: max offset %v", s.MaxOffset())
	}
	return s
}

// **Ctrl+Page Up and Ctrl+Page Down are the previous and the next tab,
// not a page of scrolling.** A page inside a tab is nearly always inside
// a scroll view, so a view that took them would swallow the shortcut and
// it would never reach the thing it names. Shift+Page is a page, and so
// is a bare one.
func TestScrollViewLeavesPageKeysWithAModifier(t *testing.T) {
	for _, tc := range []struct {
		name  string
		mods  platform.Modifiers
		takes bool
	}{
		{"bare", 0, true},
		{"with Shift", platform.ModShift, true},
		{"with Caps Lock on", platform.ModCapsLock, true},
		{"with Num Lock on", platform.ModNumLock, true},
		{"with Ctrl", platform.ModCtrl, false},
		{"with Ctrl and Shift", platform.ModCtrl | platform.ModShift, false},
		{"with Alt", platform.ModAlt, false},
		{"with Super", platform.ModSuper, false},
	} {
		for _, key := range []platform.Key{platform.KeyPageDown, platform.KeyPageUp} {
			s := scrollable(t)
			if key == platform.KeyPageUp {
				s.ScrollTo(s.MaxOffset())
			}
			before := s.OffsetY
			took := s.KeyPress(widget.KeyEvent{Key: key, Mods: tc.mods})
			moved := s.OffsetY != before
			if took != tc.takes || moved != tc.takes {
				verb := "take"
				if !tc.takes {
					verb = "leave"
				}
				t.Errorf("%v %s: handled=%v moved=%v, want the view to %s it",
					key, tc.name, took, moved, verb)
			}
		}
	}
}

// The rest of the keys are unchanged: this is about Page Up and Down
// alone, and a bare Down still scrolls a line.
func TestScrollViewStillTakesTheOtherKeys(t *testing.T) {
	s := scrollable(t)
	if !s.KeyPress(widget.KeyEvent{Key: platform.KeyDown}) || s.OffsetY <= 0 {
		t.Errorf("a bare Down did not scroll: handled=%v offset=%v", true, s.OffsetY)
	}
	if !s.KeyPress(widget.KeyEvent{Key: platform.KeyEnd}) || s.OffsetY != s.MaxOffset() {
		t.Errorf("End left the offset at %v, want the maximum %v", s.OffsetY, s.MaxOffset())
	}
}
