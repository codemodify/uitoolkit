package style

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// ink counts the pixels an icon actually put down.
func inkOf(img *paintengine2d.Image) int {
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0x2000 {
				n++
			}
		}
	}
	return n
}

// A person's icon set is copied into their own directory once and an
// application is forbidden to write there (icons/README.md), so the day
// the toolkit gains a typed id every installed copy of every set is one
// stem short of it — for everybody, every time. That is falling behind,
// not being installed wrong, and the toolkit has its own vector for every
// typed id, so it draws that and says the set is behind.
//
// A stem-only icon is the other case: nothing anywhere can draw it, and
// the missing-icon mark is the only honest answer.
func TestAStaleFileSetGetsTheDrawnMarkNotABox(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	ResetSearchPathsForTest()
	t.Cleanup(ResetSearchPathsForTest)

	// A set with one icon in it and nothing else — the shape of a copy
	// made before the toolkit grew the rest.
	setDir := filepath.Join(dir, "uitoolkit", "icons", "stale")
	if err := os.MkdirAll(setDir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	for y := 4; y < 20; y++ {
		for x := 4; x < 20; x++ {
			seed.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, seed); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(setDir, "open.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateIconCache()

	const set = IconSetName("stale")
	if !IsFileIconSet(set) {
		t.Skipf("the temporary set did not register as a file set")
	}

	draw := func(icon ToolIcon) *paintengine2d.Image {
		img := paintengine2d.NewImage(24, 24)
		ctx := paintengine2d.NewContext(img)
		DrawToolIcon(ctx, paintengine2d.XYWH(0, 0, 24, 24), icon, paintengine2d.RGB(0, 0, 0), set)
		img.Touch()
		return img
	}

	// The stem the set does have.
	var open, menu ToolIcon
	for _, i := range AllToolIcons() {
		switch ToolIconName(i) {
		case "open":
			open = i
		case "menu":
			menu = i
		}
	}
	if open == IconNone || menu == IconNone {
		t.Skip("this build has no open/menu id")
	}
	if got := inkOf(draw(open)); got == 0 {
		t.Error("the set's own icon drew nothing")
	}

	// The typed stem it does not: the toolkit's own mark, not a box. The
	// drawn menu mark is three bars, which is what the vector puts down.
	stale := draw(menu)
	want := paintengine2d.NewImage(24, 24)
	wctx := paintengine2d.NewContext(want)
	drawClassicIcon(wctx, paintengine2d.XYWH(0, 0, 24, 24), menu, paintengine2d.RGB(0, 0, 0))
	want.Touch()
	if inkOf(want) == 0 {
		t.Fatal("the toolkit has no drawn menu mark to fall back to")
	}
	if got, exp := inkOf(stale), inkOf(want); got != exp {
		t.Errorf("a stale set drew %d ink for a typed id, the toolkit's own mark is %d — it fell to the placeholder", got, exp)
	}

	// A stem-only icon has nothing behind it: the mark is right.
	if stem, ok := IconByStem("bluetooth"); ok {
		if Drawable(stem) {
			t.Fatalf("bluetooth resolved to a typed id, so it is the wrong probe")
		}
		box := draw(stem)
		if inkOf(box) == 0 {
			t.Error("a stem-only icon the set lacks drew nothing at all")
		}
	}
}
