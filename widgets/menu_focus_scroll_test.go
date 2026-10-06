package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A menu too long for its host scrolls, and every way the keyboard moves the
// focus must bring the focused row into view before Return can invoke it.
//
// Four did not: the first Up, the first Down, Home and End set the focus
// directly and skipped the scroll that moveFocus already did. In a player's
// 275x116 window the appearance list is twelve items in a 108-pixel view, and
// End selected a row at y 292..318 while the offset stayed at 0 — Return then
// chose a theme the person could not see.

// longMenu is a menu with more items than its box can show.
func longMenu(t *testing.T, n int) *PopupMenu {
	t.Helper()
	items := make([]*MenuItem, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, Item(fmt.Sprintf("Appearance %d", i+1), nil))
	}
	p := NewPopupMenu(items...)
	p.SetLook(style.DarkLook())
	p.SetHost(&host{})
	// The height the player's window leaves it, well short of the content.
	p.Arrange(paintengine2d.XYWH(0, 0, 240, 108))
	if p.MaxOffset() <= 0 {
		t.Skip("the menu fits its box; there is nothing to scroll")
	}
	return p
}

// focusedRowVisible reports whether the focused row is inside the viewport.
func focusedRowVisible(p *PopupMenu) (top, bot, view float32, ok bool) {
	i := p.focus
	if i < 0 || i >= len(p.Items) {
		return 0, 0, 0, false
	}
	top = p.rowTop(i) - p.OffsetY
	bot = top + p.rowH(p.Items[i])
	view = p.LocalBounds().Dy()
	return top, bot, view, top >= -0.01 && bot <= view+0.01
}

func TestEveryKeyboardFocusChangeScrollsTheRowIntoView(t *testing.T) {
	for _, tc := range []struct {
		what string
		do   func(p *PopupMenu)
	}{
		{"End", func(p *PopupMenu) {
			p.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
		}},
		{"an initial Up", func(p *PopupMenu) {
			p.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
		}},
		{"an initial Down", func(p *PopupMenu) {
			p.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
		}},
		{"Home after arrowing to the end", func(p *PopupMenu) {
			// Exactly to the last item: the first Down lands on the first,
			// and one more wraps round to the top, which would hide the
			// scroll this is about.
			for i := 0; i < len(p.Items); i++ {
				p.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
			}
			if p.focus != len(p.Items)-1 {
				t.Fatalf("arrowing landed on %d, want the last item", p.focus)
			}
			p.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
		}},
		{"End after Home", func(p *PopupMenu) {
			p.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
			p.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
		}},
	} {
		p := longMenu(t, 12)
		tc.do(p)
		top, bot, view, ok := focusedRowVisible(p)
		if !ok {
			t.Errorf("after %s the focused row is at %.0f..%.0f of a %.0f view (offset %.0f)",
				tc.what, top, bot, view, p.OffsetY)
		}
	}
}

// Arrowing, which already scrolled, still does.
func TestArrowingStillScrollsTheMenu(t *testing.T) {
	p := longMenu(t, 12)
	for i := 0; i < 11; i++ {
		p.KeyPress(widget.KeyEvent{Key: platform.KeyDown})
	}
	if _, _, _, ok := focusedRowVisible(p); !ok {
		t.Error("arrowing to the last item left it out of view")
	}
	if p.OffsetY <= 0 {
		t.Error("arrowing to the end scrolled nothing")
	}
}

// A menu that fits is never scrolled by any of them.
func TestAMenuThatFitsIsNeverScrolled(t *testing.T) {
	p := NewPopupMenu(Item("One", nil), Item("Two", nil))
	p.SetLook(style.DarkLook())
	p.SetHost(&host{})
	p.Arrange(paintengine2d.XYWH(0, 0, 240, 400))
	for _, k := range []platform.Key{platform.KeyEnd, platform.KeyHome, platform.KeyUp, platform.KeyDown} {
		p.KeyPress(widget.KeyEvent{Key: k})
		if p.OffsetY != 0 {
			t.Errorf("%v scrolled a menu that fits to %.0f", k, p.OffsetY)
		}
	}
}
