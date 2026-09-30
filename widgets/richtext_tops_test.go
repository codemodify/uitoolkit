package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
)

// Block tops are a prefix sum over heights: tops[k+1] = tops[k] +
// heights[k]. So changing heights[i] invalidates tops[i+1] onward, and
// the first valid entry after an edit is tops[i].
//
// setHeight kept tops[i+1] — one too many — so the block *directly
// after* one whose height changed was placed at the offset the old
// height gave. A table row that turned out two lines tall after its
// height had been estimated at one therefore had the next row drawn over
// its second line, with no rule between them.
//
// Heights are estimated before a block is laid out and corrected by
// setHeight when it is, so this fires whenever an estimate is wrong —
// which for a wrapping table cell is most widths. It was invisible at
// others, and a resize cleared it, because that recomputes every top
// from zero.
func TestBlockTopsFollowAHeightThatChanges(t *testing.T) {
	for _, w := range []float32{380, 420, 440, 456, 520, 700} {
		rt := NewRichText("")
		atScale(t, rt, 1)
		rt.SetHTML(`<table>
		  <tr><th>Item</th><th>Price</th><th>Notes</th></tr>
		  <tr><td><b>Tea</b></td><td>&pound;3</td><td>see <a href="http://x/">the list</a> and <code>code</code></td></tr>
		  <tr><td>Cake with a long name that should wrap inside its cell</td><td>&pound;4.50</td><td></td></tr>
		</table>`)
		sz := rt.Measure(layout.Constraints{MaxW: w, MaxH: -1})
		rt.Arrange(paintengine2d.XYWH(0, 0, w, sz.Y))

		// What Paint does: ask for the document's extent (which walks
		// the tops over estimated heights), then lay each block out,
		// which corrects them.
		_ = rt.docH()
		for i := 0; i < rt.doc.Len(); i++ {
			_ = rt.lay(i)
		}

		// Every top must now be the true running sum of the heights.
		var want float32
		for i := 0; i < rt.doc.Len(); i++ {
			if got := rt.top(i); got != want {
				t.Errorf("w=%v: block %d is placed at %v, but the blocks above it are %v tall",
					w, i, got, want)
			}
			want += rt.heights[i]
		}
	}
}

// And the whole document is as tall as its blocks, so a scroll bar is
// not sized from one estimate and a layout from another.
func TestDocumentHeightIsTheSumOfItsBlocks(t *testing.T) {
	rt := NewRichText("")
	atScale(t, rt, 1)
	rt.SetHTML(`<table>
	  <tr><td>a</td><td>a cell with enough words to need two lines of its column</td></tr>
	  <tr><td>b</td><td>c</td></tr>
	</table><p>after</p>`)
	sz := rt.Measure(layout.Constraints{MaxW: 400, MaxH: -1})
	rt.Arrange(paintengine2d.XYWH(0, 0, 400, sz.Y))
	_ = rt.docH()
	for i := 0; i < rt.doc.Len(); i++ {
		_ = rt.lay(i)
	}

	var sum float32
	for _, h := range rt.heights {
		sum += h
	}
	if got := rt.docH(); got != sum {
		t.Errorf("the document is %v tall but its blocks add up to %v", got, sum)
	}
}
