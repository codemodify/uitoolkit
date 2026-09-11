package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// TableColumn describes one header cell. Width 0 shares leftover space.
type TableColumn struct {
	Title    string
	Width    float32
	Sortable bool
	Align    style.Align
}

// TableView is a virtualized row/column grid with optional sort headers.
type TableView struct {
	widget.Base
	Columns   []TableColumn
	RowCount  int
	RowHeight float32
	Selected  int
	SortCol   int
	SortAsc   bool
	CellText  func(row, col int) string
	CellBold  func(row, col int) bool
	OnSelect  func(row int)
	OnSort    func(col int, asc bool)
	OnContext func(row int, windowPos paintengine2d.Point)
	Mono      bool
	OffsetY   float32
	hovered   int
	hoverCol  int
	pressCol  int
	rows      rowSceneCache
}

// NewTableView builds a table. selected starts at -1.
func NewTableView(cols []TableColumn, rows int, cell func(row, col int) string, on func(int)) *TableView {
	t := &TableView{
		Columns: cols, RowCount: rows, RowHeight: 28, Selected: -1,
		SortCol: -1, SortAsc: true, CellText: cell, OnSelect: on,
		hovered: -1, hoverCol: -1, pressCol: -1,
	}
	t.Init(t)
	t.SetWantsFocus(true)
	return t
}

func (t *TableView) Measure(c layout.Constraints) paintengine2d.Point {
	h := t.headerH() + float32(t.RowCount)*t.rowH()
	if pref := t.Preferred(); pref.Y > h {
		h = pref.Y
	}
	if h < t.headerH()+t.rowH()*4 {
		h = t.headerH() + t.rowH()*4
	}
	if c.HasMaxH() && h > c.MaxH {
		h = c.MaxH
	}
	w := float32(280)
	if pref := t.Preferred(); pref.X > w {
		w = pref.X
	}
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (t *TableView) Arrange(r paintengine2d.Rect) { t.SetBounds(r); t.clamp() }

func (t *TableView) rowH() float32 {
	return style.FittedRowHeight(t.Look(), t.RowHeight)
}

func (t *TableView) headerH() float32 {
	h := t.Look().Metrics().HeaderH
	if h <= 0 {
		h = 28
	}
	if rh := t.rowH(); rh > h {
		h = rh
	}
	return h
}

func (t *TableView) contentH() float32 { return float32(t.RowCount) * t.rowH() }

func (t *TableView) clamp() {
	mx := t.contentH() - t.bodyH()
	if mx < 0 {
		mx = 0
	}
	if t.OffsetY < 0 {
		t.OffsetY = 0
	}
	if t.OffsetY > mx {
		t.OffsetY = mx
	}
}

func (t *TableView) bodyH() float32 {
	h := t.LocalBounds().Dy() - t.headerH()
	if h < 0 {
		return 0
	}
	return h
}

func (t *TableView) colWidths() []float32 {
	n := len(t.Columns)
	out := make([]float32, n)
	if n == 0 {
		return out
	}
	total := t.LocalBounds().Dx()
	fixed := float32(0)
	flex := 0
	lk := t.Look()
	for i, c := range t.Columns {
		if c.Width > 0 {
			out[i] = style.Dip(lk, c.Width)
			fixed += out[i]
		} else {
			flex++
		}
	}
	remain := total - fixed
	if remain < 0 {
		remain = 0
	}
	share := remain
	if flex > 0 {
		share = remain / float32(flex)
	}
	for i, c := range t.Columns {
		if c.Width <= 0 {
			out[i] = share
			if out[i] < 40 {
				out[i] = 40
			}
		}
	}
	return out
}

func (t *TableView) colAt(x float32) int {
	acc := float32(0)
	for i, w := range t.colWidths() {
		if x >= acc && x < acc+w {
			return i
		}
		acc += w
	}
	return -1
}

func (t *TableView) visibleRange() (lo, hi int) {
	rh := t.rowH()
	if rh <= 0 {
		return 0, 0
	}
	lo = int(t.OffsetY / rh)
	hi = int((t.OffsetY+t.bodyH())/rh) + 1
	if lo < 0 {
		lo = 0
	}
	if hi > t.RowCount {
		hi = t.RowCount
	}
	return
}

func (t *TableView) Paint(ctx *paintengine2d.Context) {
	b := t.LocalBounds()
	lk := t.Look()
	hh := t.headerH()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Field))
	widths := t.colWidths()
	x := float32(0)
	for i, col := range t.Columns {
		w := widths[i]
		hb := paintengine2d.XYWH(x, 0, w, hh)
		st := t.State()
		if i != t.hoverCol {
			st &^= style.StateHovered
		} else {
			st |= style.StateHovered
		}
		if i == t.pressCol {
			st |= style.StatePressed
		}
		lk.DrawTableHeader(ctx, hb, st, col.Title, t.SortCol == i, t.SortAsc)
		x += w
	}
	body := paintengine2d.XYWH(0, hh, b.Dx(), b.Dy()-hh)
	ctx.Save()
	ctx.ClipRect(body)
	rh := t.rowH()
	lo, hi := t.visibleRange()
	if rec, ok := ctx.Device().(*paintengine2d.Recorder); ok {
		ob := t.Bounds()
		t.rows.ready(ob.Min.X, ob.Min.Y, b.Dx(), rh, lookSig(lk))
		recordScrollingRows(rec, &t.rows, t.ID()^(1<<32), t.OffsetY, hh, lo, hi,
			func(i int) uint64 { return t.ID()<<32 | uint64(i) + 1 },
			func(i int) uint64 {
				extra := bits32(b.Dx())
				if t.Mono {
					extra ^= 0x4d
				}
				if t.CellBold != nil && t.CellBold(i, 0) {
					extra ^= 0xb01d
				}
				parts := make([]string, len(t.Columns))
				for col := range t.Columns {
					if t.CellText != nil {
						parts[col] = t.CellText(i, col)
					}
					extra ^= bits32(widths[col]) << uint(col%16)
				}
				return visualSig(i == t.Selected, i == t.hovered, extra, parts...)
			},
			func(i int) {
				y := float32(i) * rh
				cx := float32(0)
				for col := range t.Columns {
					w := widths[col]
					cell := paintengine2d.XYWH(cx, y, w, rh)
					label := ""
					if t.CellText != nil {
						label = t.CellText(i, col)
					}
					face := lk.Font()
					if t.Mono {
						face = lk.MonoFont()
					}
					if t.CellBold != nil && t.CellBold(i, col) {
						face = lk.BoldFont()
					}
					lk.DrawTableCell(ctx, cell, i == t.Selected, i == t.hovered, label, t.Columns[col].Align, face)
					cx += w
				}
			},
		)
	} else {
		for row := lo; row < hi; row++ {
			y := hh + float32(row)*rh - t.OffsetY
			cx := float32(0)
			for col := range t.Columns {
				w := widths[col]
				cell := paintengine2d.XYWH(cx, y, w, rh)
				label := ""
				if t.CellText != nil {
					label = t.CellText(row, col)
				}
				face := lk.Font()
				if t.Mono {
					face = lk.MonoFont()
				}
				if t.CellBold != nil && t.CellBold(row, col) {
					face = lk.BoldFont()
				}
				lk.DrawTableCell(ctx, cell, row == t.Selected, row == t.hovered, label, t.Columns[col].Align, face)
				cx += w
			}
		}
	}
	ctx.Restore()
	if t.Focused() {
		lk.DrawFocusRing(ctx, b.Inset(-2))
	}
}

func (t *TableView) indexAt(y float32) int {
	hh := t.headerH()
	if y < hh {
		return -1
	}
	i := int((y - hh + t.OffsetY) / t.rowH())
	if i < 0 || i >= t.RowCount {
		return -1
	}
	return i
}

func (t *TableView) rowRect(i int) paintengine2d.Rect {
	if i < 0 {
		return paintengine2d.Rect{}
	}
	rh := t.rowH()
	y := t.headerH() + float32(i)*rh - t.OffsetY
	return paintengine2d.XYWH(0, y, t.LocalBounds().Dx(), rh)
}

func (t *TableView) invalidateRow(i int) {
	if r := t.rowRect(i); !r.Empty() {
		t.InvalidateRect(r.Inset(-1))
	}
}

func (t *TableView) invalidateHeader() {
	hh := t.headerH()
	if hh <= 0 {
		return
	}
	t.InvalidateRect(paintengine2d.XYWH(0, 0, t.LocalBounds().Dx(), hh).Inset(-1))
}

func (t *TableView) MouseEnter() {}

func (t *TableView) MouseMove(e widget.MouseEvent) bool {
	if e.Pos.Y < t.headerH() {
		c := t.colAt(e.Pos.X)
		if c != t.hoverCol || t.hovered != -1 {
			oldRow := t.hovered
			t.hoverCol = c
			t.hovered = -1
			t.invalidateRow(oldRow)
			t.invalidateHeader()
		}
		return true
	}
	h := t.indexAt(e.Pos.Y)
	if h != t.hovered || t.hoverCol != -1 {
		oldRow, oldCol := t.hovered, t.hoverCol
		t.hovered = h
		t.hoverCol = -1
		t.invalidateRow(oldRow)
		t.invalidateRow(h)
		if oldCol != -1 {
			t.invalidateHeader()
		}
	}
	return true
}

func (t *TableView) MouseExit() {
	oldRow := t.hovered
	hadCol := t.hoverCol != -1
	t.hovered = -1
	t.hoverCol = -1
	t.pressCol = -1
	t.invalidateRow(oldRow)
	if hadCol {
		t.invalidateHeader()
	}
}

func (t *TableView) MousePress(e widget.MouseEvent) bool {
	t.RequestFocus()
	if e.Pos.Y < t.headerH() {
		c := t.colAt(e.Pos.X)
		t.pressCol = c
		if c >= 0 && c < len(t.Columns) && t.Columns[c].Sortable {
			t.sortBy(c)
		}
		t.Invalidate()
		return true
	}
	i := t.indexAt(e.Pos.Y)
	if i >= 0 {
		t.Selected = i
		t.Invalidate()
		if t.OnSelect != nil {
			t.OnSelect(i)
		}
	}
	if e.Button == platform.ButtonRight && t.OnContext != nil {
		o := widget.DeviceOrigin(t)
		t.OnContext(i, paintengine2d.Pt(o.X+e.Pos.X, o.Y+e.Pos.Y))
	}
	return true
}

func (t *TableView) MouseRelease(widget.MouseEvent) bool {
	t.pressCol = -1
	t.Invalidate()
	return true
}

func (t *TableView) sortBy(col int) {
	if t.SortCol == col {
		t.SortAsc = !t.SortAsc
	} else {
		t.SortCol = col
		t.SortAsc = true
	}
	t.Invalidate()
	if t.OnSort != nil {
		t.OnSort(col, t.SortAsc)
	}
}

func (t *TableView) MouseWheel(e widget.MouseEvent) bool {
	dy := e.Scroll.Y
	if dy > -8 && dy < 8 && dy != 0 {
		dy *= t.rowH() * 3
	}
	t.OffsetY += dy
	t.clamp()
	t.Invalidate()
	return true
}

func (t *TableView) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() || t.RowCount <= 0 {
		return false
	}
	next := t.Selected
	page := int(t.bodyH()/t.rowH()) - 1
	if page < 1 {
		page = 1
	}
	switch e.Key {
	case platform.KeyDown:
		next++
	case platform.KeyUp:
		next--
	case platform.KeyPageDown:
		next += page
	case platform.KeyPageUp:
		next -= page
	case platform.KeyHome:
		next = 0
	case platform.KeyEnd:
		next = t.RowCount - 1
	case platform.KeyReturn, platform.KeySpace:
		if t.Selected >= 0 && t.OnSelect != nil {
			t.OnSelect(t.Selected)
		}
		return true
	default:
		return false
	}
	if next < 0 {
		next = 0
	}
	if next >= t.RowCount {
		next = t.RowCount - 1
	}
	if next != t.Selected {
		t.Selected = next
		t.ensureVisible(next)
		t.Invalidate()
		if t.OnSelect != nil {
			t.OnSelect(next)
		}
	}
	return true
}

func (t *TableView) ensureVisible(i int) {
	rh := t.rowH()
	top := float32(i) * rh
	bot := top + rh
	view := t.bodyH()
	if top < t.OffsetY {
		t.OffsetY = top
	}
	if bot > t.OffsetY+view {
		t.OffsetY = bot - view
	}
	t.clamp()
}
