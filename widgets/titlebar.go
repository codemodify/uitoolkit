package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// TitleBar is optional client-side window chrome (caption + muted subtitle).
type TitleBar struct {
	widget.Base
	Title    string
	Subtitle string
}

// NewTitleBar constructs a caption strip.
func NewTitleBar(title, subtitle string) *TitleBar {
	t := &TitleBar{Title: title, Subtitle: subtitle}
	t.Init(t)
	return t
}

func (t *TitleBar) SetTitle(s string) {
	if t.Title == s {
		return
	}
	t.Title = s
	t.Invalidate()
}

func (t *TitleBar) SetSubtitle(s string) {
	if t.Subtitle == s {
		return
	}
	t.Subtitle = s
	t.Invalidate()
}

func (t *TitleBar) barH() float32 {
	h := t.Look().Metrics().TitleBar
	if h <= 0 {
		h = 36
	}
	if t.Subtitle != "" && h < 40 {
		h = 40
	}
	return h
}

func (t *TitleBar) Measure(c layout.Constraints) paintengine2d.Point {
	w := float32(200)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, t.barH()))
}

func (t *TitleBar) Arrange(r paintengine2d.Rect) { t.SetBounds(r) }

func (t *TitleBar) Paint(ctx *paintengine2d.Context) {
	t.Look().DrawTitleBar(ctx, t.LocalBounds(), t.Title, t.Subtitle)
}
