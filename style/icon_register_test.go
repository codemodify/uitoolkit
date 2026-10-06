package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// An application whose vocabulary is wider than the toolkit's 56 typed ids
// and 80 stems had nowhere to put a mark of its own: IconByStem answers only
// what the toolkit ships, AddSearchPath art can stand in only for a stem the
// toolkit already knows, and a stem-only id draws the missing-icon mark in
// the drawn sets, which every pack uses unless the person picks otherwise. So
// a mail client drew its mail *server* itself, outside the icon system, and
// left three other rows without a mark at all.

// crossIcon is a drawing nothing in the toolkit makes, and an unmistakable
// one: it fills its whole box. No icon the toolkit ships does that, and
// neither does the missing-icon mark, so "was this the application's drawing"
// is a question the pixels answer on their own.
func crossIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
	ctx.DrawRect(b, paintengine2d.Fill(col))
}

func TestARegisteredIconIsDrawnAndFoundByItsStem(t *testing.T) {
	icon, err := RegisterIcon("uitk-test-server", crossIcon)
	if err != nil {
		t.Fatalf("RegisterIcon: %v", err)
	}
	if icon == IconNone {
		t.Fatal("RegisterIcon answered IconNone")
	}
	// Found by stem, which is how an application that ships a name rather
	// than an id reaches it — and how a file icon set replaces it.
	if got, ok := IconByStem("uitk-test-server"); !ok || got != icon {
		t.Errorf("IconByStem answered (%v, %v), want %v", got, ok, icon)
	}
	if got := StemOf(icon); got != "uitk-test-server" {
		t.Errorf("StemOf = %q, want the registered stem", got)
	}
	// The toolkit can draw it, which is what Drawable means.
	if !Drawable(icon) {
		t.Error("a registered icon is not Drawable")
	}

	// And it really is the application's drawing in each drawn set, not the
	// missing-icon mark: the drawer fills its box, and nothing the toolkit
	// draws does.
	const box = 24 * 24
	for _, set := range []IconSetName{IconSetClassic, IconSetSharp} {
		ink := registeredIconInk(t, icon, set)
		if ink < box*9/10 {
			t.Errorf("%s: the registered icon inked %d of %d pixels — that is not its drawing", set, ink, box)
		}
	}
	// A stem the toolkit has never heard of still draws nothing of its own.
	unknown, ok := IconByStem("uitk-test-never-registered")
	if ok {
		t.Errorf("an unregistered stem answered %v", unknown)
	}
}

// Registering the same stem again replaces the drawing and keeps the id, so a
// library may register at start-up and an application may override it.
func TestRegisteringAStemAgainKeepsItsID(t *testing.T) {
	first, err := RegisterIcon("uitk-test-compact", crossIcon)
	if err != nil {
		t.Fatal(err)
	}
	var drew bool
	again, err := RegisterIcon("uitk-test-compact", func(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) {
		drew = true
		crossIcon(ctx, b, col)
	})
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Errorf("re-registering gave %v, want the first id %v", again, first)
	}
	registeredIconInk(t, first, IconSetClassic)
	if !drew {
		t.Error("the replacement drawing was not used")
	}
}

// The toolkit's own names and stems are refused: they have drawings in every
// set, and replacing one quietly would make an application's icons differ
// from everybody's for no visible reason.
func TestRegisteringOverTheToolkitsOwnIsRefused(t *testing.T) {
	for _, name := range []string{"save", "chevron-down"} {
		if _, err := RegisterIcon(name, crossIcon); err == nil {
			t.Errorf("registering over %q was allowed", name)
		}
	}
	if _, err := RegisterIcon("", crossIcon); err == nil {
		t.Error("an empty stem was allowed")
	}
	if _, err := RegisterIcon("uitk-test-nodraw", nil); err == nil {
		t.Error("a nil drawing was allowed")
	}
}

// inkOf draws the icon and counts the pixels it put down.
func registeredIconInk(t *testing.T, icon ToolIcon, set IconSetName) int {
	t.Helper()
	const n = 24
	img := paintengine2d.NewImage(n, n)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	DrawToolIcon(ctx, paintengine2d.XYWH(0, 0, n, n), icon, paintengine2d.RGB(1, 1, 1), set)
	img.Touch()
	count := 0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if _, _, _, a := img.PremulAt(x, y); a > 128 {
				count++
			}
		}
	}
	return count
}
