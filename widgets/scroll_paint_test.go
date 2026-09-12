package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func scenePaint(c widget.Component, w, h int) *paintengine2d.Image {
	rec := paintengine2d.NewRecorder(w, h)
	ctx := paintengine2d.NewContextDevice(rec)
	widget.RecordTree(c, rec, ctx, nil, nil, true)
	img := paintengine2d.NewImage(w, h)
	paintengine2d.DrawScene(rec.Finish(), paintengine2d.NewCPUDevice(img))
	return img
}

func immediatePaint(c widget.Component, w, h int) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	widget.PaintTree(c, ctx, nil)
	return img
}

func bandInk(img *paintengine2d.Image, y0, y1, x0, x1 int) int {
	if img == nil || y1 <= y0 || x1 <= x0 {
		return 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x0 < 0 {
		x0 = 0
	}
	if y1 > img.Height {
		y1 = img.Height
	}
	if x1 > img.Width {
		x1 = img.Width
	}
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, a := img.PremulAt(x, y)
			if a > 8 && int(r)+int(g)+int(b) > 40 {
				n++
			}
		}
	}
	return n
}

func nearlySameBand(a, b *paintengine2d.Image, y0, y1 int) bool {
	if a == nil || b == nil {
		return false
	}
	x1 := a.Width
	if b.Width < x1 {
		x1 = b.Width
	}
	if y0 < 0 {
		y0 = 0
	}
	if y1 > a.Height {
		y1 = a.Height
	}
	if y1 > b.Height {
		y1 = b.Height
	}
	diff := 0
	total := 0
	for y := y0; y < y1; y++ {
		for x := 0; x < x1; x++ {
			ar, ag, ab, _ := a.PremulAt(x, y)
			br, bg, bb, _ := b.PremulAt(x, y)
			total++
			dr := int(ar) - int(br)
			dg := int(ag) - int(bg)
			db := int(ab) - int(bb)
			if dr < 0 {
				dr = -dr
			}
			if dg < 0 {
				dg = -dg
			}
			if db < 0 {
				db = -db
			}
			if dr+dg+db > 48 {
				diff++
			}
		}
	}
	if total == 0 {
		return false
	}
	return diff*20 < total
}

func TestTableViewHeaderFlushAndClip(t *testing.T) {
	look := style.DarkLook()
	tv := NewTableView([]TableColumn{
		{Title: "Subject", Width: 160},
		{Title: "Date", Width: 80},
	}, 80, func(row, col int) string {
		if col == 0 {
			return fmt.Sprintf("row-%02d-subject", row)
		}
		return fmt.Sprintf("%02d:00", row)
	}, nil)
	tv.RowHeight = 20
	tv.Selected = 0
	tv.SetLook(look)
	tv.SetHost(&host{})
	const w, h = 240, 140
	tv.Arrange(paintengine2d.XYWH(0, 0, w, h))
	hh := int(tv.headerH() + 0.5)
	rh := tv.rowH()
	if hh < 8 || rh < 8 {
		t.Fatalf("header=%d rowH=%v", hh, rh)
	}

	lo, hi := tv.visibleRange()
	if lo != 0 {
		t.Fatalf("top lo=%d", lo)
	}
	if r := tv.rowRect(0); r.Min.Y < tv.headerH()-0.5 || r.Min.Y > tv.headerH()+0.5 {
		t.Fatalf("row 0 should sit flush under header, y=%v header=%v", r.Min.Y, tv.headerH())
	}

	top := scenePaint(tv, w, h)
	imm := immediatePaint(tv, w, h)
	headerInk := bandInk(top, 1, hh-1, 8, 140)
	rowInk := bandInk(top, hh+2, hh+int(rh)-2, 8, 200)
	if headerInk < 20 {
		t.Fatalf("header should paint labels, ink=%d", headerInk)
	}
	if rowInk < 20 {
		t.Fatalf("scrollY=0 first row should sit flush under header, body ink=%d", rowInk)
	}
	if !nearlySameBand(top, imm, hh, hh+int(rh)) {
		t.Fatal("scene first row should match immediate paint (no header-sized gap)")
	}

	tv.OffsetY = rh * 0.5
	tv.clamp()
	mid := scenePaint(tv, w, h)
	if !nearlySameBand(top, mid, 0, hh) {
		t.Fatal("scrolling down must clip rows below the sticky header")
	}
	for i := lo; i < hi; i++ {
		r := tv.rowRect(i)
		view := paintengine2d.XYWH(0, tv.headerH(), float32(w), float32(h)-tv.headerH())
		if !r.Overlaps(view) && i == lo {
			t.Fatalf("visible row %d bounds %+v miss body %+v", i, r, view)
		}
	}

	page := tv.bodyH() + rh
	tv.OffsetY = page
	tv.clamp()
	if tv.OffsetY < tv.bodyH()*0.5 {
		t.Fatalf("expected scroll past one page, offset=%v body=%v", tv.OffsetY, tv.bodyH())
	}
	lo, hi = tv.visibleRange()
	if lo < 2 {
		t.Fatalf("past first page lo=%d", lo)
	}
	past := scenePaint(tv, w, h)
	bodyInk := bandInk(past, hh+2, h-8, 8, 200)
	if bodyInk < 20 {
		t.Fatalf("past first page should still paint rows, ink=%d lo=%d hi=%d offset=%v", bodyInk, lo, hi, tv.OffsetY)
	}
	if !nearlySameBand(top, past, 0, hh) {
		t.Fatal("header must stay intact after scrolling past the first page")
	}
}

func TestListViewScenePaintsPastFirstPage(t *testing.T) {
	look := style.DarkLook()
	lv := NewListView(200, func(i int) string { return fmt.Sprintf("item-%03d", i) }, nil)
	lv.RowHeight = 16
	lv.SetLook(look)
	lv.SetHost(&host{})
	const w, h = 180, 96
	lv.Arrange(paintengine2d.XYWH(0, 0, w, h))
	page := lv.LocalBounds().Dy() + lv.rowH()*2
	lv.OffsetY = page
	lv.clamp()
	lo, hi := lv.visibleRange()
	if lo < 4 {
		t.Fatalf("expected window past first page, lo=%d hi=%d offset=%v", lo, hi, lv.OffsetY)
	}
	view := lv.LocalBounds()
	hit := 0
	for i := lo; i < hi; i++ {
		if lv.rowRect(i).Overlaps(view) {
			hit++
		}
	}
	if hit < 3 {
		t.Fatalf("expected visible rows in viewport, hit=%d lo=%d hi=%d", hit, lo, hi)
	}
	img := scenePaint(lv, w, h)
	if bandInk(img, 4, h-8, 8, w-16) < 20 {
		t.Fatalf("list scene empty after first page, lo=%d hi=%d offset=%v", lo, hi, lv.OffsetY)
	}
}

func TestCardListScenePaintsPastFirstPage(t *testing.T) {
	look := style.DarkLook()
	cl := NewCardList(80, func(i int) CardContent {
		return CardContent{Title: fmt.Sprintf("from-%02d", i), Subtitle: "subject", Meta: "now"}
	}, nil)
	cl.CardHeight = 48
	cl.SetLook(look)
	cl.SetHost(&host{})
	const w, h = 220, 160
	cl.Arrange(paintengine2d.XYWH(0, 0, w, h))
	cl.OffsetY = cl.LocalBounds().Dy() + cl.rowH()
	cl.clamp()
	lo, hi := cl.visibleRange()
	if lo < 1 {
		t.Fatalf("card lo=%d", lo)
	}
	img := scenePaint(cl, w, h)
	if bandInk(img, 4, h-8, 8, w-16) < 20 {
		t.Fatalf("card list empty after first page, lo=%d hi=%d", lo, hi)
	}
}

func TestVisibleRowsCoverViewportAfterScroll(t *testing.T) {
	look := style.DarkLook()
	lv := NewListView(80, func(i int) string { return fmt.Sprintf("item-%03d", i) }, nil)
	lv.SetLook(look)
	lv.SetHost(&host{})
	lv.Arrange(paintengine2d.XYWH(0, 0, 200, 160))
	lv.ScrollTo(48)
	lo, hi := lv.VisibleRange()
	if hi-lo < 3 {
		t.Fatalf("visible %d..%d", lo, hi)
	}
	view := lv.LocalBounds()
	var covered float32
	prev := view.Min.Y
	for i := lo; i < hi; i++ {
		r := lv.rowRect(i)
		if r.Max.Y <= view.Min.Y || r.Min.Y >= view.Max.Y {
			continue
		}
		top := r.Min.Y
		if top < view.Min.Y {
			top = view.Min.Y
		}
		if top > prev+1.5 {
			t.Fatalf("gap before row %d: prev=%v top=%v", i, prev, top)
		}
		bot := r.Max.Y
		if bot > view.Max.Y {
			bot = view.Max.Y
		}
		if bot > prev {
			covered += bot - prev
			prev = bot
		}
	}
	if covered < view.Dy()*0.85 {
		t.Fatalf("rows cover %v of viewport %v (lo=%d hi=%d off=%v)", covered, view.Dy(), lo, hi, lv.OffsetY)
	}
}
