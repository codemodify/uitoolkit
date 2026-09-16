package widgets

import (
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// DropZone wraps content that takes files or text dragged from other apps
// (a file manager, a browser): OnFiles gets the local paths, OnText the
// text. While a drag is over it, it shows a highlight around its content.
type DropZone struct {
	widget.Base
	OnFiles func(paths []string)
	OnText  func(text string)
	child   widget.Component
	over    bool
}

// NewDropZone wraps child.
func NewDropZone(child widget.Component, onFiles func([]string)) *DropZone {
	z := &DropZone{OnFiles: onFiles, child: child}
	z.Init(z)
	if child != nil {
		z.Add(child)
	}
	return z
}

// DropTypes: files when OnFiles is set, text when OnText is.
func (z *DropZone) DropTypes() []string {
	var out []string
	if z.OnFiles != nil {
		out = append(out, "text/uri-list")
	}
	if z.OnText != nil {
		out = append(out, "text/plain")
	}
	return out
}

func (z *DropZone) Drop(e widget.DropEvent) bool {
	switch {
	case len(e.Paths) > 0 && z.OnFiles != nil:
		z.OnFiles(e.Paths)
		return true
	case e.Text != "" && z.OnText != nil:
		z.OnText(e.Text)
		return true
	}
	return false
}

func (z *DropZone) DragOver(paintengine2d.Point) {
	if !z.over {
		z.over = true
		z.Invalidate()
	}
}

func (z *DropZone) DragLeave() {
	if z.over {
		z.over = false
		z.Invalidate()
	}
}

// ring is the room the highlight takes around the content.
func (z *DropZone) ring() float32 { return style.Dip(z.Look(), 3) }

func (z *DropZone) Measure(c layout.Constraints) paintengine2d.Point {
	if z.child == nil {
		return c.Constrain(paintengine2d.Pt(0, 0))
	}
	r := z.ring()
	sz := z.child.Measure(c.Inset(2*r, 2*r))
	return c.Constrain(paintengine2d.Pt(sz.X+2*r, sz.Y+2*r))
}

func (z *DropZone) Arrange(b paintengine2d.Rect) {
	z.SetBounds(b)
	if z.child != nil {
		z.child.Arrange(z.LocalBounds().Inset(z.ring()))
	}
}

func (z *DropZone) Paint(ctx *paintengine2d.Context) {
	if !z.over {
		return
	}
	lk := z.Look()
	acc := lk.Palette().Accent
	b := z.LocalBounds()
	r := lk.Metrics().Radius
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(acc.WithAlpha(0.12)))
	w := max(1, z.ring()*0.6)
	ctx.DrawRoundRect(b.Inset(w*0.5), r, r, paintengine2d.StrokePaint(acc, w))
}

// ---- text fields take dropped text ---------------------------------------------

func (t *TextField) DropTypes() []string { return []string{"text/plain"} }

// Drop inserts the text where it was dropped (one line).
func (t *TextField) Drop(e widget.DropEvent) bool {
	if !t.Enabled() || e.Text == "" {
		return false
	}
	// A drop of this field's own drag: the text is inserted here, so a
	// move must not also take the original away (drag.go).
	t.selfDrop = e.Source == widget.Component(t)
	t.caret = t.indexAt(e.Pos.X)
	t.selA, t.selB = t.caret, t.caret
	t.replaceSel(strings.Join(strings.Fields(strings.ReplaceAll(e.Text, "\n", " ")), " "))
	t.RequestFocus()
	return true
}

func (t *TextArea) DropTypes() []string { return []string{"text/plain"} }

// Drop inserts the text where it was dropped.
func (t *TextArea) Drop(e widget.DropEvent) bool {
	if !t.Enabled() || t.ReadOnly || e.Text == "" {
		return false
	}
	// A drop of this area's own drag: the text is inserted here, so a
	// move must not also take the original away (drag.go).
	t.selfDrop = e.Source == widget.Component(t)
	t.caret = t.indexAt(e.Pos.X, e.Pos.Y)
	t.selA, t.selB = t.caret, t.caret
	t.replaceSel(e.Text)
	t.RequestFocus()
	return true
}
