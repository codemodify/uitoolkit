package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A dialog with more body than the window can hold keeps its buttons in
// view, and scrolls the text under them.
//
// An Overlay holds its card to 0.88 of the window, so a card with a long
// body used to lay its last lines and its action row out past its own
// foot: the buttons were drawn where they could not be seen, and a click
// or a Tab that reached them landed nowhere.
func TestATallDialogKeepsItsButtonsInView(t *testing.T) {
	ok := NewButton("OK", nil)
	card := DialogCard("Plan", strings.Repeat("a line of the plan\n", 60), ok)
	ov := NewOverlay(card)
	ov.SetLook(style.DarkLook())
	ov.SetHost(&host{})

	const W, H = 420, 300
	ov.Arrange(paintengine2d.XYWH(0, 0, W, H))

	cb := widget.DeviceBounds(card)
	if cb.Max.Y > H {
		t.Fatalf("the card runs to %g, past the %d-pixel window", cb.Max.Y, H)
	}
	bb := widget.DeviceBounds(ok)
	if bb.Max.Y > cb.Max.Y+0.5 {
		t.Errorf("the OK button ends at %g, past the card's foot at %g", bb.Max.Y, cb.Max.Y)
	}
	if bb.Max.Y > H {
		t.Errorf("the OK button ends at %g, outside the %d-pixel window", bb.Max.Y, H)
	}
	if bb.Dy() <= 0 {
		t.Error("the OK button has no height")
	}
}
