package skingen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"

	"github.com/codemodify/uitoolkit/skingen"
	"github.com/codemodify/uitoolkit/style"
)

// A generated layout can be a *resizable* panel.
//
// The runtime has read stretchX, stretchY, fromRight and fromBottom on a slot
// rect since layouts existed — they are what let a playlist's row viewport grow
// while its buttons stay on its foot — but SlotSpec had only Name, At and Art.
// So a resizable skin could be written by hand and not generated, and an
// application that used the generator had to reopen the manifest it had just
// written and add the anchors itself.
func responsivePlan() *skingen.Plan {
	fill := func(hex string) func(*paintengine2d.Context, float32, float32) {
		return func(ctx *paintengine2d.Context, w, h float32) {
			ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(skingen.Hex(hex)))
		}
	}
	return &skingen.Plan{
		Name:  "stretchy",
		Label: "Stretchy",
		Base:  "breeze-night",
		Sheets: []*skingen.Sheet{{
			Name: "chrome", W: 64, H: 32,
			Cells: []skingen.Cell{
				{Name: "face", X: 0, Y: 0, W: 32, H: 32, Draw: fill("#202428")},
				{Name: "key", X: 32, Y: 0, W: 32, H: 32, Draw: fill("#8899aa")},
			},
		}},
		Layouts: []skingen.LayoutSpec{{
			Name: "playlist", W: 240, H: 160, Art: "face",
			Slots: []skingen.SlotSpec{
				// The row viewport: both margins, so it grows with the window.
				{Name: "rows", At: [4]int{4, 4, 4, 28}, StretchX: true, StretchY: true},
				// A button pinned to the bottom right corner.
				{Name: "close", At: [4]int{4, 4, 20, 20}, FromRight: true, FromBottom: true,
					Art: [][2]string{{"normal", "key"}}},
				// And one that only follows the foot.
				{Name: "add", At: [4]int{4, 4, 20, 20}, FromBottom: true},
			},
		}},
		Window: &skingen.WindowSpec{
			Caption: 20,
			Shape: []skingen.ShapeRect{
				{At: [4]int{0, 0, 0, 0}, StretchX: true, StretchY: true},
				{At: [4]int{0, 0, 8, 8}, FromRight: true, FromBottom: true},
			},
		},
	}
}

// The anchors reach the manifest, and the manifest the generator writes is one
// the loader reads.
func TestGeneratedSlotAnchorsLoad(t *testing.T) {
	dir := t.TempDir()
	p := responsivePlan()
	if err := skingen.Write(dir, p); err != nil {
		t.Fatalf("Write: %v", err)
	}
	sk, err := style.LoadSkinFS(os.DirFS(filepath.Join(dir, p.Name)), p.Name)
	if err != nil {
		t.Fatalf("a plan with anchored slots does not load: %v", err)
	}
	lay := sk.Layouts["playlist"]
	if lay == nil {
		t.Fatal("the layout did not reach the skin")
	}
	for _, c := range []struct {
		slot                                   string
		stretchX, stretchY, fromRight, fromBtm bool
	}{
		{"rows", true, true, false, false},
		{"close", false, false, true, true},
		{"add", false, false, false, true},
	} {
		sl := lay.Slots[c.slot]
		if sl == nil {
			t.Errorf("%s: the slot did not reach the skin", c.slot)
			continue
		}
		r := sl.Rect
		if r.StretchX != c.stretchX || r.StretchY != c.stretchY ||
			r.FromRight != c.fromRight || r.FromBottom != c.fromBtm {
			t.Errorf("%s: stretchX=%v stretchY=%v fromRight=%v fromBottom=%v, want %v %v %v %v",
				c.slot, r.StretchX, r.StretchY, r.FromRight, r.FromBottom,
				c.stretchX, c.stretchY, c.fromRight, c.fromBtm)
		}
	}
}

// And they do what they say: the stretching slot grows with the panel and the
// pinned one stays in its corner. A manifest that merely parses is not the
// claim — the claim is that a generated pack is responsive.
func TestGeneratedSlotAnchorsResolve(t *testing.T) {
	dir := t.TempDir()
	p := responsivePlan()
	if err := skingen.Write(dir, p); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := style.RegisterSkinFS(os.DirFS(filepath.Join(dir, p.Name)), p.Name); err != nil {
		t.Fatalf("RegisterSkinFS: %v", err)
	}
	pack, ok := style.LoadTheme(p.Name)
	if !ok {
		t.Skip("the generated skin did not register as a theme")
	}
	lk := pack.Look()

	small := paintengine2d.XYWH(0, 0, 240, 160)
	big := paintengine2d.XYWH(0, 0, 480, 320)
	rowsS, ok1 := style.SkinSlotRect(lk, "playlist", "rows", small)
	rowsB, ok2 := style.SkinSlotRect(lk, "playlist", "rows", big)
	closeS, ok3 := style.SkinSlotRect(lk, "playlist", "close", small)
	closeB, ok4 := style.SkinSlotRect(lk, "playlist", "close", big)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		t.Fatal("the generated layout's slots cannot be resolved")
	}
	if rowsB.Dx() <= rowsS.Dx() || rowsB.Dy() <= rowsS.Dy() {
		t.Errorf("the stretching slot is %gx%g in the small panel and %gx%g in the big one",
			rowsS.Dx(), rowsS.Dy(), rowsB.Dx(), rowsB.Dy())
	}
	if closeB.Dx() != closeS.Dx() || closeB.Dy() != closeS.Dy() {
		t.Errorf("the pinned slot changed size: %gx%g then %gx%g",
			closeS.Dx(), closeS.Dy(), closeB.Dx(), closeB.Dy())
	}
	if gap := big.Max.X - closeB.Max.X; gap != small.Max.X-closeS.Max.X {
		t.Errorf("the slot pinned to the right edge is %g from it in the big panel and %g in the small one",
			gap, small.Max.X-closeS.Max.X)
	}
	if gap := big.Max.Y - closeB.Max.Y; gap != small.Max.Y-closeS.Max.Y {
		t.Errorf("the slot pinned to the bottom edge is %g from it in the big panel and %g in the small one",
			gap, small.Max.Y-closeS.Max.Y)
	}
}

// A plan that says nothing about anchors writes none, and the manifest stays
// as terse as it was: four false flags in every rect would make a file nobody
// reads, and the format's default is already a fixed rect at the top left.
//
// The plan here has a layout and a shape in it, because a plan with neither
// never reaches the code that writes them — which is how the first version of
// this test passed a generator that wrote "stretchX": false into everything.
func TestAPlanWithNoAnchorsWritesNone(t *testing.T) {
	p := responsivePlan()
	for i := range p.Layouts {
		for j := range p.Layouts[i].Slots {
			sl := &p.Layouts[i].Slots[j]
			sl.StretchX, sl.StretchY, sl.FromRight, sl.FromBottom = false, false, false, false
		}
	}
	for i := range p.Window.Shape {
		r := &p.Window.Shape[i]
		r.StretchX, r.StretchY, r.FromRight, r.FromBottom = false, false, false, false
		// A rect that does not stretch needs a size: under stretchX the third
		// number is a margin and may be 0, and without it it may not.
		r.At = [4]int{0, 0, 8, 8}
	}
	doc, err := skingen.Manifest(p)
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if !strings.Contains(string(doc), `"slots"`) {
		t.Fatal("this plan has no slots in its manifest, so it tests nothing")
	}
	for _, k := range []string{"stretchX", "stretchY", "fromRight", "fromBottom"} {
		if strings.Contains(string(doc), k) {
			t.Errorf("the manifest mentions %s though the plan asked for no anchors", k)
		}
	}
}
