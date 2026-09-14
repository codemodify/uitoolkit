package widgets

import (
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// TableColumn describes one header cell.
// Width 0 shares leftover space (flex). Width > 0 is a preferred size
// that may shrink to MinWidth when the row is tight.
type TableColumn struct {
	Title    string
	Width    float32
	MinWidth float32
	Sortable bool
	Align    style.Align
}

// TableView is a virtualized row/column grid with optional sort headers.
type TableView struct {
	widget.Base
	Columns    []TableColumn
	RowCount   int
	RowHeight  float32
	Selected   int
	SortCol    int
	SortAsc    bool
	CellText   func(row, col int) string
	CellBold   func(row, col int) bool
	OnSelect   func(row int)
	OnActivate func(row int) // Return / Space / double click; falls back to OnSelect
	OnSort     func(col int, asc bool)
	OnContext  func(row int, windowPos paintengine2d.Point)
	Mono       bool
	OffsetY    float32
	hovered    int
	hoverCol   int
	pressCol   int
	resizeCol  int
	resizeX    float32
	resizeW    float32
	vbar       scrollDrag
	rows       rowSceneCache
	lastRow    int
	lastAt     time.Time
}

// doubleClickInterval is the window for a second press on the same row to
// count as an activation.
const doubleClickInterval = 400 * time.Millisecond

// NewTableView builds a table. selected starts at -1.
func NewTableView(cols []TableColumn, rows int, cell func(row, col int) string, on func(int)) *TableView {
	t := &TableView{
		Columns: cols, RowCount: rows, RowHeight: 28, Selected: -1,
		SortCol: -1, SortAsc: true, CellText: cell, OnSelect: on,
		hovered: -1, hoverCol: -1, pressCol: -1, resizeCol: -1, lastRow: -1,
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

// MaxOffset is max(0, content − body viewport).
func (t *TableView) MaxOffset() float32 {
	return layout.MaxScroll(t.contentH(), t.bodyH())
}

func (t *TableView) clamp() {
	t.OffsetY = layout.ClampScroll(t.OffsetY, t.contentH(), t.bodyH())
}

func (t *TableView) vparts() style.ScrollParts {
	return vScrollParts(t.Look(), paintengine2d.XYWH(0, t.headerH(), t.LocalBounds().Dx(), t.bodyH()), t.contentH(), t.OffsetY)
}

func (t *TableView) vaxis() scrollAxis {
	return scrollAxis{
		vertical: true,
		parts:    t.vparts,
		get:      func() (float32, float32) { return t.OffsetY, t.MaxOffset() },
		set:      func(y float32) { t.OffsetY = y; t.clamp(); t.Invalidate() },
		steps:    func() (float32, float32) { return t.rowH(), t.bodyH() * 0.9 },
	}
}

// rowsW is the row width: the view minus the gutter a visible bar takes.
func (t *TableView) rowsW() float32 {
	return t.LocalBounds().Dx() - scrollGutter(t.Look(), t.MaxOffset() > 0)
}

func (t *TableView) scrollTrack() (track, thumb paintengine2d.Rect) {
	sp := t.vparts()
	return sp.Track, sp.Thumb
}

// VisibleRange is the half-open [lo, hi) window of body rows that Paint draws.
func (t *TableView) VisibleRange() (lo, hi int) { return t.visibleRange() }

// HeaderHeight is the sticky header band (device pixels).
func (t *TableView) HeaderHeight() float32 { return t.headerH() }

// BodyHeight is the viewport under the sticky header.
func (t *TableView) BodyHeight() float32 { return t.bodyH() }

// ScrollTrack is the overflow bar geometry in local coordinates.
func (t *TableView) ScrollTrack() (track, thumb paintengine2d.Rect) { return t.scrollTrack() }

// RowBounds is the current on-screen rect of row i (may sit above the header
// in local space; Paint clips it to the body).
func (t *TableView) RowBounds(i int) paintengine2d.Rect { return t.rowRect(i) }

// ScrollTo sets OffsetY (clamped) without requiring a wheel event.
func (t *TableView) ScrollTo(y float32) {
	t.OffsetY = y
	t.clamp()
	t.Invalidate()
}

func (t *TableView) bodyH() float32 {
	h := t.LocalBounds().Dy() - t.headerH()
	if h < 0 {
		return 0
	}
	return h
}

// ColumnWidths is the current header/cell layout in device pixels.
func (t *TableView) ColumnWidths() []float32 { return t.colWidths() }

// RowsWidth is the width the columns fill: the view minus the gutter a
// visible scrollbar takes, so no cell text runs under the bar.
func (t *TableView) RowsWidth() float32 { return t.rowsW() }

func (t *TableView) colWidths() []float32 {
	n := len(t.Columns)
	out := make([]float32, n)
	if n == 0 {
		return out
	}
	total := t.rowsW()
	if total < 1 {
		return out
	}
	lk := t.Look()
	min := make([]float32, n)
	flex := make([]bool, n)
	nflex := 0
	for i, c := range t.Columns {
		if c.Width <= 0 {
			flex[i] = true
			nflex++
			floor := c.MinWidth
			if floor <= 0 {
				floor = 80
			}
			min[i] = style.Dip(lk, floor)
			out[i] = min[i]
			continue
		}
		pref := style.Dip(lk, c.Width)
		floor := c.MinWidth
		if floor <= 0 {
			floor = c.Width
		}
		min[i] = style.Dip(lk, floor)
		if min[i] > pref {
			min[i] = pref
		}
		if min[i] < 1 {
			min[i] = 1
		}
		out[i] = pref
	}
	used := float32(0)
	for _, w := range out {
		used += w
	}
	remain := total - used
	if remain > 0 && nflex > 0 {
		share := remain / float32(nflex)
		for i := range out {
			if flex[i] {
				out[i] += share
			}
		}
	} else if remain > 0 {
		out[n-1] += remain
	} else if remain < 0 {
		shrinkBy(out, min, -remain, flex, false)
		used = 0
		for _, w := range out {
			used += w
		}
		if used > total {
			shrinkBy(out, min, used-total, flex, true)
		}
		used = 0
		for _, w := range out {
			used += w
		}
		if used > total {
			scale := total / used
			for i := range out {
				out[i] *= scale
			}
		}
	}
	used = 0
	for _, w := range out {
		used += w
	}
	if drift := total - used; drift != 0 {
		give := 0
		for i, f := range flex {
			if f {
				give = i
				break
			}
		}
		if !flex[give] {
			give = n - 1
		}
		out[give] += drift
		if out[give] < 1 {
			out[give] = 1
		}
	}
	return out
}

// shrinkBy removes deficit from columns, preferring non-flex first, then
// flex, never going below min unless force.
func shrinkBy(out, min []float32, deficit float32, flex []bool, force bool) {
	if deficit <= 0 {
		return
	}
	pass := func(wantFlex bool) {
		if deficit <= 0 {
			return
		}
		slack := float32(0)
		for i := range out {
			if flex[i] != wantFlex {
				continue
			}
			s := out[i] - min[i]
			if force {
				s = out[i] - 1
			}
			if s > 0 {
				slack += s
			}
		}
		if slack <= 0 {
			return
		}
		take := deficit
		if take > slack {
			take = slack
		}
		for i := range out {
			if flex[i] != wantFlex {
				continue
			}
			s := out[i] - min[i]
			if force {
				s = out[i] - 1
			}
			if s <= 0 {
				continue
			}
			cut := take * (s / slack)
			out[i] -= cut
			deficit -= cut
		}
	}
	pass(false)
	pass(true)
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

const colResizeHit = 5

func (t *TableView) colEdgeAt(x float32) int {
	acc := float32(0)
	for i, w := range t.colWidths() {
		acc += w
		if x >= acc-colResizeHit && x < acc+colResizeHit {
			return i
		}
	}
	return -1
}

// ColumnDividerAt is the column whose right edge is near x, or -1.
func (t *TableView) ColumnDividerAt(x float32) int { return t.colEdgeAt(x) }

// ResizingColumn is the header divider being dragged, or -1.
func (t *TableView) ResizingColumn() int { return t.resizeCol }

// CursorAt is col-resize on a header divider (and while dragging).
func (t *TableView) CursorAt(local paintengine2d.Point) platform.Cursor {
	if t.resizeCol >= 0 {
		return platform.CursorColResize
	}
	if local.Y >= 0 && local.Y < t.headerH() && t.colEdgeAt(local.X) >= 0 {
		return platform.CursorColResize
	}
	return platform.CursorDefault
}

func (t *TableView) applyCursor(local paintengine2d.Point) {
	widget.ApplyCursor(t.Host(), t.CursorAt(local))
}

func (t *TableView) minColPx(i int) float32 {
	if i < 0 || i >= len(t.Columns) {
		return 24
	}
	floor := t.Columns[i].MinWidth
	if floor <= 0 {
		floor = 40
	}
	px := style.Dip(t.Look(), floor)
	if px < 24 {
		px = 24
	}
	return px
}

func (t *TableView) setColWidthPx(i int, px float32) {
	if i < 0 || i >= len(t.Columns) {
		return
	}
	if min := t.minColPx(i); px < min {
		px = min
	}
	scale := style.LookScale(t.Look())
	if scale < 0.01 {
		scale = 1
	}
	if t.Columns[i].Width == px/scale {
		return
	}
	t.Columns[i].Width = px / scale
	t.rows.reset()
	t.Invalidate()
}

func (t *TableView) paintHeader(ctx *paintengine2d.Context, lk style.LookAndFeel, widths []float32, hh float32) {
	x := float32(0)
	for i, col := range t.Columns {
		w := widths[i]
		hb := paintengine2d.XYWH(x, 0, w, hh)
		// A press on a body row must not paint every header pressed.
		st := t.State() &^ (style.StateHovered | style.StatePressed)
		if i == t.hoverCol {
			st |= style.StateHovered
		}
		if i == t.pressCol {
			st |= style.StatePressed
		}
		lk.DrawTableHeader(ctx, hb, st, col.Title, t.SortCol == i, t.SortAsc)
		x += w
	}
}

func (t *TableView) paintRow(ctx *paintengine2d.Context, lk style.LookAndFeel, widths []float32, row int, y, rh float32) {
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
	t.clamp()
	b := t.LocalBounds()
	lk := t.Look()
	rw := t.rowsW()
	hh := t.headerH()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Field))
	widths := t.colWidths()
	body := paintengine2d.XYWH(0, hh, b.Dx(), b.Dy()-hh)
	rh := t.rowH()
	lo, hi := t.visibleRange()
	if rec, ok := ctx.Device().(*paintengine2d.Recorder); ok {
		// The body viewport goes on the band group, so the sticky header
		// keeps its own clip and the rows slide under it.
		o := rowOrigin(ctx)
		t.rows.ready(o.X, o.Y, rw, rh, lookSig(lk))
		recordScrollingRows(rec, ctx, &t.rows, t.ID()^(1<<32), body, rw, rh, t.OffsetY, hh, lo, hi,
			func(i int) uint64 { return t.ID()<<32 | uint64(i) + 1 },
			func(i int) uint64 {
				extra := bits32(rw)
				if t.Mono {
					extra ^= 0x4d
				}
				if t.CellBold != nil && t.CellBold(i, 0) {
					extra ^= 0xb01d
				}
				for col := range t.Columns {
					extra ^= bits32(widths[col]) << uint(col%16)
				}
				sig := newRowSig(i == t.Selected, i == t.hovered, extra)
				for col := range t.Columns {
					if t.CellText != nil {
						sig.str(t.CellText(i, col))
					} else {
						sig.str("")
					}
				}
				return sig.sum()
			},
			func(i int) {
				t.paintRow(ctx, lk, widths, i, 0, rh)
			},
		)
	} else {
		ctx.Save()
		ctx.ClipRect(body)
		for row := lo; row < hi; row++ {
			y := hh + float32(row)*rh - t.OffsetY
			t.paintRow(ctx, lk, widths, row, y, rh)
		}
		ctx.Restore()
	}
	// Sticky header after the body so a leaked row cannot cover the labels.
	t.paintHeader(ctx, lk, widths, hh)
	t.vbar.paint(ctx, lk, t.vparts(), true)
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

// Invalidate drops retained row scenes so CellText / flag changes
// repaint on the next frame (visualSig is an optimization, not the API).
func (t *TableView) Invalidate() {
	t.rows.reset()
	t.Base.Invalidate()
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
	if t.resizeCol >= 0 {
		t.setColWidthPx(t.resizeCol, t.resizeW+(e.Pos.X-t.resizeX))
		t.applyCursor(e.Pos)
		return true
	}
	if handled, dirty := t.vbar.move(e.Pos, t.vaxis()); handled || dirty {
		if dirty {
			t.Invalidate()
		}
		if handled {
			return true
		}
	}
	if e.Pos.Y < t.headerH() {
		t.applyCursor(e.Pos)
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
	t.vbar.exit()
	if t.resizeCol < 0 {
		widget.ApplyCursor(t.Host(), platform.CursorDefault)
	}
	t.invalidateRow(oldRow)
	if hadCol {
		t.invalidateHeader()
	}
}

func (t *TableView) MousePress(e widget.MouseEvent) bool {
	t.RequestFocus()
	if t.vbar.press(t, e.Pos, t.vaxis()) {
		t.Invalidate()
		return true
	}
	if e.Pos.Y < t.headerH() {
		if edge := t.colEdgeAt(e.Pos.X); edge >= 0 {
			widths := t.colWidths()
			t.resizeCol = edge
			t.resizeX = e.Pos.X
			t.resizeW = widths[edge]
			t.pressCol = -1
			t.applyCursor(e.Pos)
			t.Invalidate()
			return true
		}
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
		if e.Button == platform.ButtonLeft {
			if i == t.lastRow && time.Since(t.lastAt) < doubleClickInterval {
				t.lastRow = -1
				t.activate(i)
			} else {
				t.lastRow = i
				t.lastAt = time.Now()
			}
		}
	}
	if e.Button == platform.ButtonRight && t.OnContext != nil {
		o := widget.DeviceOrigin(t)
		t.OnContext(i, paintengine2d.Pt(o.X+e.Pos.X, o.Y+e.Pos.Y))
	}
	return true
}

func (t *TableView) MouseRelease(e widget.MouseEvent) bool {
	t.pressCol = -1
	if t.resizeCol >= 0 {
		t.resizeCol = -1
		t.applyCursor(e.Pos)
		t.Invalidate()
		return true
	}
	if t.vbar.release() {
		t.Invalidate()
		return true
	}
	t.Invalidate()
	return true
}

// activate reports a row the user committed to (Return, Space, double click).
// Callers that only want selection changes keep using OnSelect.
func (t *TableView) activate(row int) {
	if row < 0 || row >= t.RowCount {
		return
	}
	if t.OnActivate != nil {
		t.OnActivate(row)
		return
	}
	if t.OnSelect != nil {
		t.OnSelect(row)
	}
}

// Activate commits row (the programmatic form of Return / double click).
func (t *TableView) Activate(row int) { t.activate(row) }

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

// MouseWheel scrolls, and reports false when it cannot: an unscrollable or
// already-at-the-edge view must let the wheel bubble to an outer scroll pane
// instead of swallowing it.
func (t *TableView) MouseWheel(e widget.MouseEvent) bool {
	if t.MaxOffset() <= 0 {
		return false
	}
	before := t.OffsetY
	t.OffsetY += wheelDelta(e.Scroll.Y, t.rowH())
	t.clamp()
	if t.OffsetY == before {
		return false
	}
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
		t.activate(t.Selected)
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
