package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// floatOverFixture is a list under a bar, both big enough to tell apart.
func floatOverFixture(t *testing.T) (*FloatOver, *Panel, *Panel) {
	t.Helper()
	content := NewPanel("", NewHeightBox(400, NewLabel("content")))
	bar := NewPanel("", NewHeightBox(40, NewLabel("bar")))
	f := NewFloatOver(content)
	f.SetTop(bar)
	h := &host{}
	f.SetHost(h)
	content.SetHost(h)
	bar.SetHost(h)
	f.Arrange(paintengine2d.XYWH(0, 0, 300, 200))
	return f, content, bar
}

// **The content keeps the whole box and the bar sits on top of it.** A
// column would have given the list what was left under the bar; here it
// runs the full height, which is what puts something behind the glass.
func TestFloatingChromeLeavesTheContentTheWholeBox(t *testing.T) {
	f, content, bar := floatOverFixture(t)
	if got := content.Bounds(); got.Dy() != 200 || got.Min.Y != 0 {
		t.Errorf("the content got %v, want the whole 300x200 box", got)
	}
	b := bar.Bounds()
	if b.Min.Y != 0 {
		t.Errorf("the bar is at y=%v, want the top edge", b.Min.Y)
	}
	if b.Dy() >= 200 {
		t.Errorf("the bar is %v tall — it took the whole box instead of its own height", b.Dy())
	}
	if b.Dy() <= 0 {
		t.Errorf("the bar has no height")
	}
	if b.Dx() != 300 {
		t.Errorf("the bar is %v wide, want the full 300", b.Dx())
	}
	_ = f
}

// **Clicks fall through.** A press on the bar reaches the bar, and a
// press below it reaches the content — chrome that floats must not
// swallow the view it floats over.
func TestAPressBelowTheFloatingBarReachesTheContent(t *testing.T) {
	f, content, bar := floatOverFixture(t)
	onBar := paintengine2d.Pt(150, bar.Bounds().Dy()/2)
	below := paintengine2d.Pt(150, 150)

	if got := f.HitTest(onBar); !widget.Contains(bar, got) {
		t.Errorf("a press on the bar hit %T, want the bar", got)
	}
	if got := f.HitTest(below); !widget.Contains(content, got) {
		t.Errorf("a press below the bar hit %T, want the content", got)
	}
}

// The bar paints after the content, or it would be behind the thing it
// floats over.
func TestTheFloatingBarPaintsOverTheContent(t *testing.T) {
	f, content, bar := floatOverFixture(t)
	kids := f.Children()
	ci, bi := -1, -1
	for i, ch := range kids {
		switch ch {
		case widget.Component(content):
			ci = i
		case widget.Component(bar):
			bi = i
		}
	}
	if ci < 0 || bi < 0 {
		t.Fatalf("content at %d and bar at %d among %d children", ci, bi, len(kids))
	}
	if bi <= ci {
		t.Errorf("the bar is child %d and the content %d: the content would paint over the bar", bi, ci)
	}
}

// A bottom piece goes against the bottom edge, not the top.
func TestABottomPieceSitsAtTheBottom(t *testing.T) {
	f, _, _ := floatOverFixture(t)
	foot := NewPanel("", NewHeightBox(30, NewLabel("foot")))
	foot.SetHost(f.Host())
	f.SetBottom(foot)
	f.Arrange(paintengine2d.XYWH(0, 0, 300, 200))
	b := foot.Bounds()
	if b.Max.Y != 200 {
		t.Errorf("the foot ends at y=%v, want the bottom edge 200", b.Max.Y)
	}
	if b.Min.Y <= 100 {
		t.Errorf("the foot starts at y=%v — it is not against the bottom", b.Min.Y)
	}
}

// Measure is the content's, so floating chrome never makes a view ask
// for more room than the view wanted.
func TestFloatingChromeDoesNotChangeTheMeasure(t *testing.T) {
	f, content, _ := floatOverFixture(t)
	c := layout.Loose(300, 1000)
	want := content.Measure(c)
	if got := f.Measure(c); got != want {
		t.Errorf("the container measures %v and its content %v", got, want)
	}
}

// **A floating bar paints no strip of its own.** A bar normally has a
// row to itself and fills it, which is opaque — and an opaque fill
// behind a translucent capsule leaves the material with nothing to be
// translucent of. Over content, the bar must let the view through
// between its buttons.
func TestAFloatingToolBarPaintsNoStrip(t *testing.T) {
	pack, ok := style.LoadTheme("tahoe-night")
	if !ok {
		t.Skipf("the %q pack is not in this build", "tahoe-night")
	}
	const w, h = 300, 60
	paint := func(floating bool) *paintengine2d.Image {
		bar := NewToolBar(
			ToolIconBtn(style.IconStar, "Star", nil),
			ToolIconBtn(style.IconTag, "Tag", nil),
		)
		bar.Floating = floating
		bar.SetLook(pack.Look())
		bar.SetHost(&host{})
		bar.Arrange(paintengine2d.XYWH(0, 0, w, h))
		img := paintengine2d.NewImage(w, h)
		ctx := paintengine2d.NewContext(img)
		// A loud ground: anything the bar paints over it shows up.
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(paintengine2d.RGB(0.95, 0.15, 0.6)))
		widget.PaintTree(bar, ctx, nil)
		img.Touch()
		return img
	}
	// The far right of the bar, past the buttons and any capsule round
	// them: a strip would have covered it, a floating bar leaves it.
	ground := func(img *paintengine2d.Image) bool {
		i := ((h/2)*img.Width + w - 4) * 4
		return img.Pix[i] > 200 && img.Pix[i+1] < 90 && img.Pix[i+2] > 120
	}
	if !ground(paint(true)) {
		t.Error("a floating bar painted over the view behind it")
	}
	if ground(paint(false)) {
		t.Error("an ordinary bar left the view showing; it is meant to fill its own strip, so this test proves nothing")
	}
}
