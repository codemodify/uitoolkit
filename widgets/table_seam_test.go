package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// A selected row is one unbroken bar. The layout may hand the table a
// fractional origin and the columns fractional widths; cells that meet on a
// fraction each half-cover the shared pixel and leave a light seam at every
// column edge (it showed in every engine in the gallery).
func TestTableSelectedRowHasNoColumnSeams(t *testing.T) {
	for _, name := range []string{"luna", "aero", "win10", "win95"} {
		pack, ok := style.LoadTheme(name)
		if !ok {
			t.Fatalf("no %s pack", name)
		}
		tv := NewTableView([]TableColumn{
			{Title: "Subject", Width: 101.5},
			{Title: "From", Width: 70.25},
			{Title: "Date", Width: 60},
		}, 20, func(row, col int) string { return fmt.Sprintf("r%d c%d", row, col) }, nil)
		tv.SetSelectedRows([]int{1})
		tv.Selected = 1
		const w, h = 262, 120
		root := NewColumn(tv)
		root.SetLook(pack.Look())
		root.SetHost(&host{})
		root.Arrange(paintengine2d.XYWH(0, 0, w, h))
		for _, ox := range []float32{0.37, 0.5} {
			tv.Arrange(paintengine2d.XYWH(ox, 0.5, w-2, h-1))
			if o := tv.Bounds().Min; o.X != float32(int(o.X)) || o.Y != float32(int(o.Y)) {
				t.Fatalf("%s: bounds origin %v is not on the pixel grid", name, o)
			}
			for path, img := range map[string]*paintengine2d.Image{
				"scene":     scenePaint(root, w, h),
				"immediate": immediatePaint(root, w, h),
			} {
				r := fromView(tv.rowRect(1), tv.frame())
				y := int(tv.Bounds().Min.Y + r.Min.Y + r.Dy()/2)
				x := tv.Bounds().Min.X + tv.frame().Left
				widths := tv.colWidths()
				for i := 0; i < len(widths)-1; i++ {
					x += widths[i]
					// The pixels either side of the edge and the edge
					// pixel itself: one colour across the bar.
					xi := int(x)
					ar, ag, ab, _ := img.PremulAt(xi-3, y)
					for xx := xi - 2; xx <= xi+2; xx++ {
						br, bg, bb, _ := img.PremulAt(xx, y)
						if d := absInt(int(ar)-int(br)) + absInt(int(ag)-int(bg)) + absInt(int(ab)-int(bb)); d > 6 {
							t.Errorf("%s %s ox=%.2f: seam at x=%d (column edge %.2f): %d,%d,%d vs %d,%d,%d",
								name, path, ox, xx, x, br, bg, bb, ar, ag, ab)
							break
						}
					}
				}
			}
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
