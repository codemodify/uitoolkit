package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// A row whose last item is of a different kind from the rest reads
// better against the far edge — Settings puts the renderer there, among
// the typefaces. The folding has to survive it, which is why this is a
// Wrap and not a Row with a spacer.
func TestWrapTrailRightEndsALineAtTheEdge(t *testing.T) {
	box := func(w float32) *Spacer { return NewSpacerSize(w, 10) }
	w := NewWrap(box(100), box(100), box(100))
	w.Gap = 10
	w.TrailRight = true
	w.SetHost(&scaledHost{look: style.WithScale(style.DarkLook(), 1)})

	// One line with room to spare: the last item ends at the edge, the
	// others stay where they were.
	w.Arrange(paintengine2d.XYWH(0, 0, 400, 40))
	kids := w.Children()
	if got := kids[0].Bounds().Min.X; got != 0 {
		t.Errorf("the first item moved to %v", got)
	}
	if got := kids[2].Bounds().Max.X; got != 400 {
		t.Errorf("the last item ends at %v, want the edge at 400", got)
	}
	if kids[1].Bounds().Max.X >= kids[2].Bounds().Min.X {
		t.Error("the pushed item overlaps the one before it")
	}

	// A line with no room is untouched: nothing moves left.
	w.Arrange(paintengine2d.XYWH(0, 0, 320, 40))
	if got := kids[2].Bounds().Max.X; got != 320 {
		t.Errorf("a full line moved its last item to %v", got)
	}
}

// A line of one keeps its place: pushing the only thing on a line to the
// far side reads as a second block, not as an aligned one.
func TestWrapTrailRightLeavesALineOfOne(t *testing.T) {
	box := func(w float32) *Spacer { return NewSpacerSize(w, 10) }
	w := NewWrap(box(200), box(200), box(200))
	w.Gap = 10
	w.TrailRight = true
	w.SetHost(&scaledHost{look: style.WithScale(style.DarkLook(), 1)})
	w.Arrange(paintengine2d.XYWH(0, 0, 420, 80))

	kids := w.Children()
	if kids[2].Bounds().Min.Y <= kids[0].Bounds().Min.Y {
		t.Fatal("the third item did not fold onto a line of its own")
	}
	if got := kids[2].Bounds().Min.X; got != 0 {
		t.Errorf("the only item on its line was pushed to %v", got)
	}
}
