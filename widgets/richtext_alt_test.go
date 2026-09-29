package widgets_test

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// imgHTML is one paragraph holding a picture that will never load, with
// and without an alt attribute.
func imgHTML(alt string) string {
	a := ""
	if alt != "" {
		a = ` alt="` + alt + `"`
	}
	return `<p><img src="https://example.invalid/chart.png"` + a + ` width="220" height="90"></p>`
}

func altShot(t *testing.T, alt string) *paintengine2d.Image {
	t.Helper()
	var ed *widgets.RichText
	a, win := shotWindow(t, "breeze", 1, 420, 200, func() widget.Component {
		ed = widgets.NewRichTextHTML(imgHTML(alt))
		return widgets.NewColumn(ed).WithPad(10)
	})
	defer win.Close()
	a.PumpOnce()
	return win.Capture()
}

// differing is where two renders of the same document disagree: the
// count, and the bounding box of the disagreement.
func differing(a, b *paintengine2d.Image) (n int, minX, minY, maxX, maxY int) {
	minX, minY, maxX, maxY = 1<<30, 1<<30, -1, -1
	for y := 0; y < a.Height && y < b.Height; y++ {
		for x := 0; x < a.Width && x < b.Width; x++ {
			i, j := y*a.Stride+x*4, y*b.Stride+x*4
			if a.Pix[i] != b.Pix[j] || a.Pix[i+1] != b.Pix[j+1] || a.Pix[i+2] != b.Pix[j+2] {
				n++
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x), max(maxY, y)
			}
		}
	}
	return
}

// A picture that never loaded shows its alt text inside the placeholder
// box, rather than an empty rectangle.
//
// An image-heavy HTML mail with its pictures unresolved — a real
// newsletter — was a wall of blank boxes, which is worse than useless.
// The alt was parsed and stored all along (richtext.Image.Alt); the
// painter simply never looked at it.
//
// The two renders differ only in the alt attribute, so anything that
// differs between them *is* the alt being drawn — which is a surer
// measure than counting ink against a guessed background colour.
func TestUnresolvedImageDrawsItsAltText(t *testing.T) {
	with, without := altShot(t, "Quarterly revenue"), altShot(t, "")
	if with == nil || without == nil {
		t.Fatal("no render")
	}
	n, x0, y0, x1, y1 := differing(with, without)
	if n == 0 {
		t.Fatal("the two renders are identical: the alt text is not drawn")
	}
	// And within the picture's own 220x90, wherever the editor put it —
	// the box's position depends on the look's padding, so the size is
	// what is asserted and not the coordinates.
	if x1-x0 > 220 || y1-y0 > 90 {
		t.Errorf("the alt drew %dx%d, larger than the 220x90 placeholder", x1-x0, y1-y0)
	}
}

// A long alt is fitted to the box rather than running out of it and
// over whatever is beside it.
func TestLongAltIsFittedToThePlaceholder(t *testing.T) {
	long := strings.Repeat("a very long alternative description ", 6)
	with, without := altShot(t, long), altShot(t, "")
	if with == nil || without == nil {
		t.Fatal("no render")
	}
	n, x0, _, x1, _ := differing(with, without)
	if n == 0 {
		t.Fatal("a long alt drew nothing at all")
	}
	// Six times the text of the short one, and it must still occupy no
	// more than the picture's 220: that is what being fitted means.
	if w := x1 - x0; w > 220 {
		t.Errorf("a long alt drew %d wide, past the 220 the placeholder gives it", w)
	}
}
