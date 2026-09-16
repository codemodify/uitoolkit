package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ListView is a virtualized fixed-height row list. Only visible rows paint.
type ListView struct {
	widget.Base
	Count     int
	RowHeight float32
	Selected  int
	ItemText  func(i int) string
	OnSelect  func(i int)
	OnContext func(i int, windowPos paintengine2d.Point)
	OffsetY   float32
	// Frameless drops the look's view frame (a list that already sits in a
	// framed pane).
	Frameless bool
	// Sidebar paints the list as a sidebar (a settings page list, a
	// mail app's folders): macOS source lists, libadwaita's navigation
	// sidebar, WinUI's navigation pane, where the look has one.
	Sidebar bool
	// OnDrag is what a press on a selected row drags out of the list
	// (widget.DragSource); nil drags nothing. It is handed the selection,
	// the current row alone when nothing is selected — the list knows
	// about rows, not about what they hold.
	OnDrag func(rows []int) *widget.Drag
	// OnDropAt takes a drop *between* two rows, at the place among them
	// it would take — index 0 above the first row, Count below the last.
	// Setting it is what turns the list into one you can reorder: an
	// insertion caret then follows the pointer down the gaps. OnDropRow
	// takes a drop *onto* a row; with both set the top and bottom
	// quarters of a row are its gaps and the middle half is the row.
	// DropMimes narrows what they take (text/uri-list when empty).
	OnDropAt  func(index int, e widget.DropEvent) bool
	OnDropRow func(row int, e widget.DropEvent) bool
	DropMimes []string
	// DropActions is what a drop here may do — copying alone when zero.
	// A list that reorders itself wants a move as well, which is what
	// lets the drag's source take the original row away.
	DropActions platform.DragAction
	// Mode selects one row (the default) or several (SelectExtended: Ctrl
	// toggles, Shift extends, Ctrl+A). Selected stays the current row.
	Mode SelectionMode
	// OnSelectionChange reports the selected rows, ascending, whenever the
	// set changes in SelectExtended or SelectMulti.
	OnSelectionChange func(rows []int)
	sel               rowSelection
	hovered           int
	// dropRow is the row a drag is over, -1 for none; dropAt is the gap
	// it would be inserted into instead, -1 for none.
	dropRow int
	dropAt  int
	vbar    scrollDrag
	rows    rowSceneCache
	reveal  int // row+1 to bring into view at the next Arrange
	find    typeAhead
	// DisableTypeAhead turns off type-ahead find, for views whose letters
	// are commands (Mail's n / p / r).
	DisableTypeAhead bool
}

func NewListView(count int, text func(int) string, on func(int)) *ListView {
	l := &ListView{Count: count, RowHeight: 28, Selected: -1, ItemText: text, OnSelect: on, hovered: -1, dropRow: -1, dropAt: -1}
	l.Init(l)
	l.SetWantsFocus(true)
	return l
}

func (l *ListView) Measure(c layout.Constraints) paintengine2d.Point {
	h := float32(l.Count) * l.rowH()
	if c.HasMaxH() && h > c.MaxH {
		h = c.MaxH
	}
	w := style.Dip(l.Look(), 200)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (l *ListView) Arrange(r paintengine2d.Rect) {
	l.SetBounds(r)
	if l.reveal > 0 {
		i := l.reveal - 1
		l.reveal = 0
		l.ensureVisible(i)
	}
	l.clamp()
}

// EnsureVisible scrolls the least needed to bring row i into view. Called
// before the list is laid out, it applies at the first Arrange.
func (l *ListView) EnsureVisible(i int) {
	if i < 0 || i >= l.Count {
		return
	}
	if l.inner().Dy() <= 0 {
		l.reveal = i + 1
		return
	}
	l.ensureVisible(i)
	l.Invalidate()
}

func (l *ListView) rowH() float32 {
	return style.FittedRowHeight(l.Look(), l.RowHeight)
}

func (l *ListView) contentH() float32 { return float32(l.Count) * l.rowH() }

// frame is the look's view frame around the list (zero on flat looks).
func (l *ListView) frame() style.Insets { return viewFrame(l.Look(), l.Frameless) }

// inner is the viewport in view space (inside the frame).
func (l *ListView) inner() paintengine2d.Rect { return viewInner(l.LocalBounds(), l.frame()) }

// MaxOffset is max(0, content − viewport).
func (l *ListView) MaxOffset() float32 {
	return layout.MaxScroll(l.contentH(), l.inner().Dy())
}

func (l *ListView) clamp() {
	l.OffsetY = layout.ClampScroll(l.OffsetY, l.contentH(), l.inner().Dy())
}

func (l *ListView) vparts() style.ScrollParts {
	return vScrollParts(l.Look(), l.inner(), l.contentH(), l.OffsetY)
}

func (l *ListView) vaxis() scrollAxis {
	return scrollAxis{
		vertical: true,
		parts:    l.vparts,
		get:      func() (float32, float32) { return l.OffsetY, l.MaxOffset() },
		set:      func(y float32) { l.OffsetY = y; l.clamp(); l.Invalidate() },
		steps:    func() (float32, float32) { return l.rowH(), l.inner().Dy() * 0.9 },
	}
}

// rowsW is the row width: the view minus the gutter a visible bar takes.
func (l *ListView) rowsW() float32 {
	return l.inner().Dx() - scrollGutter(l.Look(), l.MaxOffset() > 0)
}

func (l *ListView) scrollTrack() (track, thumb paintengine2d.Rect) {
	sp := l.vparts()
	in := l.frame()
	return fromView(sp.Track, in), fromView(sp.Thumb, in)
}

// HoverIndex is the row under the pointer, or -1.
func (l *ListView) HoverIndex() int { return l.hovered }

// VisibleRange is the half-open [lo, hi) window of rows that Paint draws.
func (l *ListView) VisibleRange() (lo, hi int) { return l.visibleRange() }

// ScrollTrack is the overflow bar geometry (empty thumb when content fits).
func (l *ListView) ScrollTrack() (track, thumb paintengine2d.Rect) { return l.scrollTrack() }

// ScrollTo sets OffsetY (clamped) without requiring a wheel event.
func (l *ListView) ScrollTo(y float32) {
	l.OffsetY = y
	l.clamp()
	l.Invalidate()
}

func (l *ListView) visibleRange() (lo, hi int) {
	rh := l.rowH()
	if rh <= 0 {
		return 0, 0
	}
	lo = int(l.OffsetY / rh)
	hi = int((l.OffsetY+l.inner().Dy())/rh) + 1
	if lo < 0 {
		lo = 0
	}
	if hi > l.Count {
		hi = l.Count
	}
	return
}

func (l *ListView) Paint(ctx *paintengine2d.Context) {
	l.clamp()
	lk := l.Look()
	if beginViewFrame(ctx, lk, l.LocalBounds(), l.frame(), l.viewState()) {
		defer ctx.Restore()
	}
	b := l.inner()
	rw := l.rowsW()
	ctx.DrawRect(b, paintengine2d.Fill(style.ViewBackgroundOf(lk, l.viewState())))
	rh := l.rowH()
	lo, hi := l.visibleRange()
	if rec, ok := ctx.Device().(*paintengine2d.Recorder); ok {
		// Rows carry the viewport on their band group, not in their own
		// clips, so no ctx.ClipRect(b) here.
		o := rowOrigin(ctx)
		l.rows.ready(o.X, o.Y, rw, rh, lookSig(lk))
		recordScrollingRows(rec, ctx, &l.rows, l.ID()^(1<<32), b, rw, rh, l.OffsetY, 0, lo, hi,
			func(i int) uint64 { return l.ID()<<32 | uint64(i) + 1 },
			func(i int) uint64 {
				label := ""
				if l.ItemText != nil {
					label = l.ItemText(i)
				}
				// Every state bit counts, the named states above bit 24
				// (sidebar, a selected neighbour) included.
				st := uint64(l.rowState(i))
				sig := newRowSig(l.IsSelected(i), i == l.hovered, bits32(rw)^st<<40^st>>24)
				sig.str(label)
				return sig.sum()
			},
			func(i int) {
				label := ""
				if l.ItemText != nil {
					label = l.ItemText(i)
				}
				lk.DrawListRow(ctx, paintengine2d.XYWH(0, 0, rw, rh), l.rowState(i), label)
			},
		)
	} else {
		ctx.Save()
		ctx.ClipRect(b)
		for i := lo; i < hi; i++ {
			y := float32(i)*rh - l.OffsetY
			row := paintengine2d.XYWH(0, y, rw, rh)
			label := ""
			if l.ItemText != nil {
				label = l.ItemText(i)
			}
			lk.DrawListRow(ctx, row, l.rowState(i), label)
		}
		ctx.Restore()
	}
	// Where a drag would land, over the rows and under the scrollbar: a
	// drop onto a row tints it, one between two rows gets the caret.
	if l.dropRow >= 0 && l.dropRow < l.Count {
		ctx.Save()
		ctx.ClipRect(b)
		paintDropRow(ctx, lk, paintengine2d.XYWH(0, float32(l.dropRow)*rh-l.OffsetY, rw, rh))
		ctx.Restore()
	}
	if l.dropAt >= 0 {
		paintDropCaret(ctx, lk, l.dropAt, l.Count, 0, rw, 0, rh, l.OffsetY, b)
	}
	l.vbar.paint(l, ctx, lk, l.vparts(), true, l.OffsetY)
	// The current row carries the focus mark; a focused list without one
	// rings itself.
	if l.Focused() && (l.Selected < 0 || l.Selected >= l.Count) {
		lk.DrawFocusRing(ctx, b)
	}
}

func (l *ListView) indexAt(y float32) int {
	return rowIndexAt(y, l.OffsetY, l.rowH(), l.Count)
}

func (l *ListView) rowRect(i int) paintengine2d.Rect {
	if i < 0 {
		return paintengine2d.Rect{}
	}
	rh := l.rowH()
	y := float32(i)*rh - l.OffsetY
	return paintengine2d.XYWH(0, y, l.inner().Dx(), rh)
}

func (l *ListView) invalidateRow(i int) {
	if r := l.rowRect(i); !r.Empty() {
		l.InvalidateRect(fromView(r, l.frame()).Inset(-1))
	}
}

func (l *ListView) MouseEnter() {}

func (l *ListView) MouseMove(e widget.MouseEvent) bool {
	p := toView(e.Pos, l.frame())
	if handled, dirty := l.vbar.move(l, p, l.vaxis()); handled || dirty {
		if dirty {
			l.Invalidate()
		}
		if handled {
			return true
		}
	}
	h := l.indexAt(p.Y)
	if p.Y < 0 || p.Y >= l.inner().Dy() {
		h = -1
	}
	if h != l.hovered {
		old := l.hovered
		l.hovered = h
		l.invalidateRow(old)
		l.invalidateRow(h)
	}
	return true
}

func (l *ListView) MouseExit() {
	old := l.hovered
	l.hovered = -1
	l.vbar.exit()
	l.invalidateRow(old)
}

func (l *ListView) MouseRelease(widget.MouseEvent) bool {
	if l.vbar.release() {
		l.Invalidate()
		return true
	}
	return false
}

func (l *ListView) MousePress(e widget.MouseEvent) bool {
	l.RequestFocus()
	p := toView(e.Pos, l.frame())
	if l.vbar.press(l, p, l.vaxis()) {
		l.Invalidate()
		return true
	}
	i := l.indexAt(p.Y)
	if p.Y < 0 || p.Y >= l.inner().Dy() {
		i = -1
	}
	if i >= 0 {
		changed := false
		if l.Mode != SelectSingle {
			if e.Button == platform.ButtonRight {
				changed = l.sel.contextClick(i)
			} else {
				changed = l.sel.click(l.Mode, i, e.Mods)
			}
		}
		l.Selected = i
		l.Invalidate()
		if l.OnSelect != nil {
			l.OnSelect(i)
		}
		if changed {
			l.selectionChanged()
		}
	}
	if e.Button == platform.ButtonRight && l.OnContext != nil {
		o := widget.DeviceOrigin(l)
		l.OnContext(i, paintengine2d.Pt(o.X+e.Pos.X, o.Y+e.Pos.Y))
	}
	return true
}

// MouseWheel scrolls, and reports false when it cannot: an unscrollable or
// already-at-the-edge view must let the wheel bubble to an outer scroll pane
// instead of swallowing it.
func (l *ListView) MouseWheel(e widget.MouseEvent) bool {
	if l.MaxOffset() <= 0 {
		return false
	}
	before := l.OffsetY
	l.OffsetY += wheelDelta(e.Scroll.Y, l.rowH(), e.Precise)
	l.clamp()
	if l.OffsetY == before {
		return false
	}
	l.Invalidate()
	return true
}

func (l *ListView) KeyPress(e widget.KeyEvent) bool {
	if !l.Enabled() || l.Count <= 0 {
		return false
	}
	if contextKey(e) {
		if l.OnContext == nil {
			return false
		}
		l.OnContext(l.Selected, contextPoint(l, fromView(l.rowRect(l.Selected), l.frame())))
		return true
	}
	next := l.Selected
	page := int(l.inner().Dy()/l.rowH()) - 1
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
		next = l.Count - 1
	case platform.KeySpace:
		// Space toggles the current row in Multi (Ctrl+Space in Extended).
		if l.Mode == SelectMulti || (l.Mode == SelectExtended && e.Mods.Ctrl()) {
			if l.Selected >= 0 {
				l.sel.toggle(l.Selected)
				l.sel.anchor = l.Selected
				l.Invalidate()
				l.selectionChanged()
			}
			return true
		}
		if l.Selected >= 0 && l.OnSelect != nil {
			l.OnSelect(l.Selected)
		}
		return true
	case platform.KeyReturn:
		if l.Selected >= 0 && l.OnSelect != nil {
			l.OnSelect(l.Selected)
		}
		return true
	case platform.KeyA:
		if e.Mods.Ctrl() && l.Mode != SelectSingle {
			l.SelectAll()
			return true
		}
		return false
	default:
		return false
	}
	l.navigate(next, e.Mods)
	return true
}

// navigate makes row next current the way keyboard navigation does: plain
// moves select it, Shift extends and Ctrl only moves (SelectExtended).
func (l *ListView) navigate(next int, mods platform.Modifiers) {
	if next < 0 {
		next = 0
	}
	if next >= l.Count {
		next = l.Count - 1
	}
	changed := false
	if l.Mode != SelectSingle {
		changed = l.sel.moveTo(l.Mode, next, mods)
	}
	if next != l.Selected || changed {
		l.Selected = next
		l.ensureVisible(next)
		l.Invalidate()
		if l.OnSelect != nil {
			l.OnSelect(next)
		}
		if changed {
			l.selectionChanged()
		}
	}
}

// TextInput is type-ahead find: typing jumps to the next row whose text
// starts with what was typed.
func (l *ListView) TextInput(r rune) bool {
	if !l.Enabled() || l.ItemText == nil || l.DisableTypeAhead {
		return false
	}
	i, searched := l.find.next(r, l.Selected, l.Count, l.ItemText)
	if i >= 0 {
		l.navigate(i, 0)
	}
	return searched
}

// rowState is row i's item state for the look.
func (l *ListView) rowState(i int) style.ControlState {
	sel := l.IsSelected(i)
	st := widget.RowItemState(l, i, sel, i == l.hovered, i == l.Selected)
	if sel {
		// Consecutive selected rows: looks that box a selection join them.
		if l.IsSelected(i - 1) {
			st |= style.StateSelectedAbove
		}
		if l.IsSelected(i + 1) {
			st |= style.StateSelectedBelow
		}
	}
	if l.Sidebar {
		st |= style.StateSidebar
	}
	return st
}

// viewState is the list's own state for its frame.
func (l *ListView) viewState() style.ControlState {
	if l.Sidebar {
		return l.State() | style.StateSidebar
	}
	return l.State()
}

// IsSelected reports whether row i is selected (in SelectSingle, whether it
// is the current row).
func (l *ListView) IsSelected(i int) bool {
	if i < 0 || i >= l.Count {
		return false
	}
	if l.Mode == SelectSingle {
		return i == l.Selected
	}
	return l.sel.has(i)
}

// SelectedRows is the selection in ascending order.
func (l *ListView) SelectedRows() []int {
	if l.Mode == SelectSingle {
		if l.Selected >= 0 && l.Selected < l.Count {
			return []int{l.Selected}
		}
		return nil
	}
	l.sel.drop(l.Count)
	return l.sel.rows()
}

// SetSelectedRows replaces the selection; the last row given becomes the
// current row and the anchor. It does not call the callbacks.
func (l *ListView) SetSelectedRows(rows []int) {
	l.sel.clear()
	for _, r := range rows {
		if r >= 0 && r < l.Count {
			l.sel.add(r)
			l.Selected = r
			l.sel.anchor = r
		}
	}
	if len(rows) == 0 {
		l.Selected = -1
	}
	l.Invalidate()
}

// SelectAll selects every row (SelectExtended / SelectMulti) and reports it.
func (l *ListView) SelectAll() {
	if l.Mode == SelectSingle || l.Count == 0 {
		return
	}
	l.sel.addRange(0, l.Count-1)
	l.Invalidate()
	l.selectionChanged()
}

func (l *ListView) selectionChanged() {
	if l.OnSelectionChange != nil && l.Mode != SelectSingle {
		l.OnSelectionChange(l.SelectedRows())
	}
}

func (l *ListView) ensureVisible(i int) {
	rh := l.rowH()
	top := float32(i) * rh
	bot := top + rh
	view := l.inner().Dy()
	if top < l.OffsetY {
		l.OffsetY = top
	}
	if bot > l.OffsetY+view {
		l.OffsetY = bot - view
	}
	l.clamp()
}
