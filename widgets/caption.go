package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// Caption hit-testing (widget.CaptionHitTester) for the built-in widgets that
// can sit in a window's title bar: layout containers, spacers, labels,
// separators and pictures are caption; strips of controls are caption only
// where no control is. In a title bar, pressing caption space moves the
// window and double-clicking it maximizes; every other widget is a control.

// CaptionAt: a row or column's own space is caption.
func (f *FlexBox) CaptionAt(paintengine2d.Point) bool { return true }

// CaptionAt: a stack's own space is caption.
func (s *Stack) CaptionAt(paintengine2d.Point) bool { return true }

// CaptionAt: padding is caption.
func (p *Pad) CaptionAt(paintengine2d.Point) bool { return true }

// CaptionAt: a spacer is caption (the free space of a title bar).
func (s *Spacer) CaptionAt(paintengine2d.Point) bool { return true }

// CaptionAt: a separator is caption.
func (s *Separator) CaptionAt(paintengine2d.Point) bool { return true }

// CaptionAt: a label is caption (a window title, a subtitle).
func (l *Label) CaptionAt(paintengine2d.Point) bool { return true }

// CaptionAt: a picture is caption (an app icon or logo).
func (p *Picture) CaptionAt(paintengine2d.Point) bool { return true }

// CaptionAt: a title strip is caption.
func (t *TitleBar) CaptionAt(paintengine2d.Point) bool { return true }

// CaptionAt: a tool bar is caption between its buttons and on dividers.
func (t *ToolBar) CaptionAt(p paintengine2d.Point) bool {
	i := t.itemAt(p)
	return i < 0 || t.items[i] == nil || t.items[i].Sep
}

// CaptionAt: a tab bar is caption where there is no tab (after the last).
func (t *TabBar) CaptionAt(p paintengine2d.Point) bool { return t.indexAt(p.X) < 0 }

// CaptionAt: a menu bar is caption after its last title.
func (m *MenuBar) CaptionAt(p paintengine2d.Point) bool { return m.titleAt(p.X) < 0 }

// CaptionRegion forces its content to be caption or to be controls,
// whatever the content says: Electron's app-region: drag / no-drag, Tauri's
// data-tauri-drag-region. Make one with DragArea or NoDrag.
type CaptionRegion struct {
	widget.Base
	drag bool
}

// DragArea makes c caption in a window's title bar: pressing anywhere on it
// moves the window and double-clicking maximizes it. c itself no longer gets
// pointer input (a logo, a decorative title), as with app-region: drag.
func DragArea(c widget.Component) *CaptionRegion { return newCaptionRegion(c, true) }

// NoDrag keeps all of c out of the caption: even c's empty space takes
// clicks as a control would (app-region: no-drag).
func NoDrag(c widget.Component) *CaptionRegion { return newCaptionRegion(c, false) }

func newCaptionRegion(c widget.Component, drag bool) *CaptionRegion {
	r := &CaptionRegion{drag: drag}
	r.Init(r)
	if c != nil {
		r.Add(c)
	}
	return r
}

// Drag reports whether the region is a drag area.
func (r *CaptionRegion) Drag() bool { return r.drag }

func (r *CaptionRegion) Measure(c layout.Constraints) paintengine2d.Point {
	if ch := r.Children(); len(ch) > 0 {
		return ch[0].Measure(c)
	}
	return c.Constrain(paintengine2d.Point{})
}

func (r *CaptionRegion) Arrange(b paintengine2d.Rect) {
	r.SetBounds(b)
	if ch := r.Children(); len(ch) > 0 {
		ch[0].Arrange(paintengine2d.XYWH(0, 0, b.Dx(), b.Dy()))
	}
}

// HitTest: a drag area takes every point itself, so nothing inside it is a
// control; a no-drag region hit-tests as usual.
func (r *CaptionRegion) HitTest(local paintengine2d.Point) widget.Component {
	if r.drag {
		if !r.Visible() || !r.LocalBounds().Contains(local) {
			return nil
		}
		return r
	}
	return r.Base.HitTest(local)
}

// CaptionAt implements widget.CaptionHitTester.
func (r *CaptionRegion) CaptionAt(paintengine2d.Point) bool { return r.drag }
