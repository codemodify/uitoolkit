package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// RowGeometry is where a list's parts sit inside its own box, in its own
// coordinates, when a skin's fixed layout places them rather than the look
// measuring them: a panel's playlist is a well printed in the art with a
// groove beside it and rows ten design pixels tall, and none of that is
// anything the look knows.
//
// Every field is optional and the zero value is "the look's own", so a
// layout that names only a rows slot gets the look's row height and the
// look's bar inside it.
type RowGeometry struct {
	// Rows is the box the rows are drawn and hit in. Empty: the view
	// frame the look would put round them.
	Rows paintengine2d.Rect
	// Bar is the groove the scroll thumb rides, beside the rows. Empty
	// *with* a Rows of its own means the layout says there is no bar: the
	// wheel, the keys and ScrollTo still scroll, nothing is painted, and
	// no gutter is taken out of the rows.
	Bar paintengine2d.Rect
	// RowHeight is one row, in the same coordinates as Rows. Zero: the
	// look's RowHeight fitted to its font.
	RowHeight float32
	// ThumbLength fixes the thumb at that many of those coordinates — a
	// thumb that is a picture is one size, as an era's was — instead of
	// sizing it to how much of the content is in view.
	ThumbLength float32
}

// RowPainter paints row i in place of the look's row and reports whether it
// did, in the box the row would have been drawn in.
//
// It is the other half of [RowGeometry]: the geometry hook says where a
// skin's rows are, and this one says what they are made of. A skin engine
// hands DrawListRow to the pack underneath it — that is the rule, and it is
// what keeps a skinned form a form — so a list sitting in a panel's printed
// well paints the pack's rows in the pack's font, where the art wants
// eleven design pixels of green on black. A skin is a picture, and a
// picture has its own type in it.
//
// It paints the row and nothing else: the selection, the keyboard, the
// wheel, type-ahead, drag and drop and the accessibility tree are the
// list's either way, and a painter with nothing for this look returns false
// and the look's own row is drawn. A list with one paints row by row rather
// than from its scrolled row cache, since what the painter draws is the
// app's own and the list cannot know when it changed.
type RowPainter func(ctx *paintengine2d.Context, b paintengine2d.Rect, i int, st style.ControlState) bool

// ScrollPainter paints a list's scroll bar in place of the look's and
// reports whether it did; track is the groove and thumb the part that
// rides it, both in the box the bar would have been drawn in.
//
// It is for the skin whose thumb is a loose sprite of its own
// (style.DrawSkinSprite) riding a groove printed in the art. A skin pack's
// own scroll parts need none of this — its engine has already overridden
// them — so a painter that has no sprite for this look returns false and
// the look's bar is drawn, groove and all.
type ScrollPainter func(ctx *paintengine2d.Context, track, thumb paintengine2d.Rect, st style.ControlState) bool

// placed reports whether the geometry says where the rows go at all.
func (g *RowGeometry) placed() bool { return g != nil && !g.Rows.Empty() }

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
	// RowGeo, when set, is asked at every paint and every press where the
	// rows, the groove and one row are, so a skinned app can drive them
	// from its skin's layout slots (style.SkinSlotRect, widget.Slots) —
	// which is the one thing a list view could not take from a skin, and
	// the reason a player had to write its own list. A nil answer, or a
	// look that is not that skin, leaves the list exactly as it was.
	//
	// A placed list draws no view frame: the art has the well printed in
	// it already.
	RowGeo func(lk style.LookAndFeel) *RowGeometry
	// RowPaint paints the rows itself, and ScrollPaint the bar, where a
	// skin's art wants its own type and its own thumb ([RowPainter],
	// [ScrollPainter]). Both are the skin's way in and neither changes
	// what the list *is*.
	RowPaint    RowPainter
	ScrollPaint ScrollPainter
	// ItemDetail is a second column at the row's right-hand end: a
	// track's length beside its title, a file's size beside its name.
	//
	// A row with one is painted as the look's own table cells — a first
	// and a last, the detail aligned to the end — rather than as a list
	// row, because a list row is one label and a look's second column is
	// the look's business: its cell padding, and the column guide a look
	// that rules its columns draws between them. It is the same row a
	// TableView would paint, without a header over it, and a list of
	// three columns with a header on them is a TableView.
	//
	// Nil is the one-column list a list view has always been.
	ItemDetail func(i int) string
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

// Measure is every row and the view frame around them, so a list given its
// natural height — in a scroll view, a column — shows its last row whole
// rather than under the frame's bottom edge.
func (l *ListView) Measure(c layout.Constraints) paintengine2d.Point {
	in := l.frame()
	h := float32(l.Count)*l.rowH() + in.Top + in.Bottom
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

// geo is the skin's answer for this look, or nil.
func (l *ListView) geo() *RowGeometry {
	if l.RowGeo == nil {
		return nil
	}
	return l.RowGeo(l.Look())
}

func (l *ListView) rowH() float32 {
	if g := l.geo(); g.placed() && g.RowHeight > 0 {
		return g.RowHeight
	}
	return style.FittedRowHeight(l.Look(), l.RowHeight)
}

func (l *ListView) contentH() float32 { return float32(l.Count) * l.rowH() }

// frame is the look's view frame around the list (zero on flat looks, and
// zero for a placed list, whose art has the well printed in it).
func (l *ListView) frame() style.Insets {
	if l.geo().placed() {
		return style.Insets{}
	}
	return viewFrame(l.Look(), l.Frameless)
}

// pad is the inset from the list's own box to the rows: the view frame, or
// where the skin's layout put them. Everything but the frame's own painting
// works from this one — the rows are at its origin, and view space is where
// the list does all of its maths.
func (l *ListView) pad() style.Insets {
	if g := l.geo(); g.placed() {
		b := l.LocalBounds()
		return style.Insets{
			Left:   g.Rows.Min.X,
			Top:    g.Rows.Min.Y,
			Right:  b.Dx() - g.Rows.Max.X,
			Bottom: b.Dy() - g.Rows.Max.Y,
		}
	}
	return l.frame()
}

// inner is the viewport in view space (inside the frame, or on the rows
// slot the skin named).
func (l *ListView) inner() paintengine2d.Rect { return viewInner(l.LocalBounds(), l.pad()) }

// MaxOffset is max(0, content − viewport).
func (l *ListView) MaxOffset() float32 {
	return layout.MaxScroll(l.contentH(), l.inner().Dy())
}

func (l *ListView) clamp() {
	l.OffsetY = layout.ClampScroll(l.OffsetY, l.contentH(), l.inner().Dy())
}

func (l *ListView) vparts() style.ScrollParts {
	if g := l.geo(); g.placed() {
		return l.placedParts(g)
	}
	return vScrollParts(l.Look(), l.inner(), l.contentH(), l.OffsetY)
}

// placedParts is the bar in the groove the skin's layout named, in view
// space. A groove the layout leaves out is a list with no bar at all.
func (l *ListView) placedParts(g *RowGeometry) style.ScrollParts {
	if g.Bar.Empty() {
		return style.ScrollParts{}
	}
	in := l.pad()
	bar := g.Bar.Translate(paintengine2d.Pt(-in.Left, -in.Top))
	p := style.ScrollParts{Bar: bar, Track: bar}
	maxOff := l.MaxOffset()
	if maxOff <= 0 {
		return p
	}
	h := g.ThumbLength
	if h <= 0 {
		h = bar.Dy() * min(l.inner().Dy()/max(l.contentH(), 1), 1)
	}
	if lo := style.Dip(l.Look(), 8); h < lo {
		h = lo
	}
	if h > bar.Dy() {
		h = bar.Dy()
	}
	y := (bar.Dy() - h) * clampOff(l.OffsetY, maxOff) / maxOff
	p.Thumb = paintengine2d.XYWH(bar.Min.X, bar.Min.Y+y, bar.Dx(), h)
	p.Proportion = p.Thumb
	return p
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

// rowsW is the row width: the view minus the gutter a visible bar takes. A
// placed list gives up nothing — the skin's groove is beside its rows, not
// over them.
func (l *ListView) rowsW() float32 {
	if l.geo().placed() {
		return l.inner().Dx()
	}
	return l.inner().Dx() - scrollGutter(l.Look(), l.MaxOffset() > 0)
}

func (l *ListView) scrollTrack() (track, thumb paintengine2d.Rect) {
	sp := l.vparts()
	in := l.pad()
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
	if beginPlacedView(ctx, lk, l.LocalBounds(), l.frame(), l.pad(), l.viewState()) {
		defer ctx.Restore()
	}
	b := l.inner()
	rw := l.rowsW()
	if !l.geo().placed() {
		// A placed list sits on art that is already the background; the
		// look's would paint over the panel.
		ctx.DrawRect(b, paintengine2d.Fill(style.ViewBackgroundOf(lk, l.viewState())))
	}
	rh := l.rowH()
	lo, hi := l.visibleRange()
	if rec, ok := ctx.Device().(*paintengine2d.Recorder); ok && l.RowPaint == nil {
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
				if l.ItemDetail != nil {
					sig.str(l.ItemDetail(i))
				}
				return sig.sum()
			},
			func(i int) {
				l.drawRow(ctx, lk, paintengine2d.XYWH(0, 0, rw, rh), i)
			},
		)
	} else {
		ctx.Save()
		ctx.ClipRect(b)
		for i := lo; i < hi; i++ {
			y := float32(i)*rh - l.OffsetY
			l.drawRow(ctx, lk, paintengine2d.XYWH(0, y, rw, rh), i)
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
	if sp := l.vparts(); !l.paintedScroll(ctx, sp) {
		l.vbar.paint(l, ctx, lk, sp, true, l.OffsetY)
	}
	// The current row carries the focus mark; a focused list without one
	// rings itself.
	if l.Focused() && (l.Selected < 0 || l.Selected >= l.Count) {
		lk.DrawFocusRing(ctx, b)
	}
}

// drawRow paints row i in row: the app's own painter, the look's two cells
// where there is a detail column, or the look's row.
func (l *ListView) drawRow(ctx *paintengine2d.Context, lk style.LookAndFeel, row paintengine2d.Rect, i int) {
	st := l.rowState(i)
	if l.RowPaint != nil && l.RowPaint(ctx, row, i, st) {
		return
	}
	label := ""
	if l.ItemText != nil {
		label = l.ItemText(i)
	}
	if l.ItemDetail == nil {
		lk.DrawListRow(ctx, row, st, label)
		return
	}
	// Two table cells, as a table paints a row of them: the detail is
	// sized to itself and pushed to the end, so a column of lengths lines
	// up down the list without the list measuring every row it has not
	// drawn. The focus mark spans the row, since a cell does not know it.
	detail := l.ItemDetail(i)
	face := lk.Font()
	w := float32(0)
	if detail != "" {
		w = min(face.Advance(detail)+lk.Metrics().Pad*2, row.Dx())
	}
	lk.DrawTableCell(ctx, paintengine2d.XYWH(row.Min.X, row.Min.Y, row.Dx()-w, row.Dy()), st|style.StateFirst, label, style.AlignStart, face)
	lk.DrawTableCell(ctx, paintengine2d.XYWH(row.Max.X-w, row.Min.Y, w, row.Dy()), st|style.StateLast, detail, style.AlignEnd, face)
	if st.Focused() {
		style.DrawItemFocusOf(lk, ctx, row, st)
	}
}

// paintedScroll gives a ScrollPainter the groove and the thumb, and reports
// whether it took them. A list with nothing to scroll has no bar to paint,
// and the painter is not asked.
func (l *ListView) paintedScroll(ctx *paintengine2d.Context, sp style.ScrollParts) bool {
	if l.ScrollPaint == nil || sp.Thumb.Empty() {
		return false
	}
	st := l.State()
	if l.vbar.active {
		st |= style.StatePressed
	}
	if l.vbar.over {
		st |= style.StateHovered
	}
	return l.ScrollPaint(ctx, sp.Track, sp.Thumb, st)
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
		l.InvalidateRect(fromView(r, l.pad()).Inset(-1))
	}
}

func (l *ListView) MouseEnter() {}

func (l *ListView) MouseMove(e widget.MouseEvent) bool {
	p := toView(e.Pos, l.pad())
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
	p := toView(e.Pos, l.pad())
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
		l.OnContext(l.Selected, contextPoint(l, fromView(l.rowRect(l.Selected), l.pad())))
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
