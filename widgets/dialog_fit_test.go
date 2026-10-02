package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A dialog is as tall as what it holds, not as tall as the window allows.
//
// A ScrollView offered a height takes all of it, and an Overlay measures
// its card in 0.8 of the window — so a card built of DialogContent was
// that tall whatever it held, and a one-line confirmation came out 560
// pixels tall in a 700-pixel window.
func TestAShortDialogIsShort(t *testing.T) {
	short := NewOverlay(DialogCard("Confirm", "Delete this item?", NewButton("OK", nil)))
	short.SetLook(style.DarkLook())
	short.SetHost(&host{})
	short.Arrange(paintengine2d.XYWH(0, 0, 900, 700))
	sh := widget.DeviceBounds(short.Card).Dy()

	long := NewOverlay(DialogCard("Plan", strings.Repeat("a line of the plan\n", 80), NewButton("OK", nil)))
	long.SetLook(style.DarkLook())
	long.SetHost(&host{})
	long.Arrange(paintengine2d.XYWH(0, 0, 900, 700))
	lh := widget.DeviceBounds(long.Card).Dy()

	if sh >= lh {
		t.Errorf("a one-line dialog is %g tall and an eighty-line one %g; the short one is not short", sh, lh)
	}
	if sh > 700*0.6 {
		t.Errorf("a one-line dialog takes %g of a 700-pixel window", sh)
	}
	// The long one is still held to the window.
	if lh > 700 {
		t.Errorf("the long dialog is %g tall in a 700-pixel window", lh)
	}
}

// A ButtonBox handed to DialogContent lays the buttons out itself, which
// is what it is for: squeezed into a justified row its left-hand group —
// Help, and a destructive button under Mac and GNOME looks — is no longer
// apart from the rest.
func TestAButtonBoxKeepsItsLayoutInADialog(t *testing.T) {
	bb := NewButtonBox()
	bb.AddButton(NewButton("Help", nil), RoleHelp)
	bb.AddButton(NewButton("Cancel", nil), RoleReject)
	bb.AddButton(NewButton("OK", nil), RoleAccept)
	content := DialogContent(NewLabel("Body"), bb)
	content.SetLook(style.DarkLook())
	content.SetHost(&host{})
	const W = 600
	content.Arrange(paintengine2d.XYWH(0, 0, W, 300))

	got := widget.DeviceBounds(bb).Dx()
	if got < W*0.8 {
		t.Errorf("the button box got %g of %d; it was squeezed instead of laying itself out", got, W)
	}
}
