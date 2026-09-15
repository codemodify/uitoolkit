package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A horizontal override moves only the horizontal bar's buttons.
func TestScrollGeometryHorizontalArrowOverride(t *testing.T) {
	p, ok := LoadTheme("next")
	if !ok {
		t.Fatal("next pack missing")
	}
	lk := p.Look()
	view := paintengine2d.XYWH(0, 0, 300, 200)
	h := ScrollGeometry(lk, view, false, 900, 300, 0, false)
	if h.Dec.Empty() || h.Inc.Empty() || h.Dec.Max.X > h.Track.Min.X+0.01 || h.Inc.Max.X > h.Track.Min.X+0.01 {
		t.Fatalf("horizontal NeXT arrows should group at the left: dec %v inc %v track %v", h.Dec, h.Inc, h.Track)
	}
	v := ScrollGeometry(lk, view, true, 900, 200, 0, false)
	if v.Dec.Min.Y < v.Track.Max.Y-0.01 || v.Inc.Min.Y < v.Track.Max.Y-0.01 {
		t.Fatalf("vertical NeXT arrows should group at the bottom: dec %v inc %v track %v", v.Dec, v.Inc, v.Track)
	}
}
