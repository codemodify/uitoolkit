package minim

import (
	"fmt"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/internal/players"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The two windows that stick to the strip.
//
// They are separate top-level windows rather than panels that fold out of
// the main one, because that is the thing being demonstrated: three windows
// that behave as one object, snapping flush and travelling together. The
// arithmetic is in players.Rack and the desktop half is players.Desk; all
// either window does here is lay itself out.

// ---- the equaliser -----------------------------------------------------------

// eqPane is a preamp, ten bands and a preset picker.
type eqPane struct {
	widget.Base
	p *Player

	on     *players.GlyphButton
	flat   *widgets.Button
	preset *widgets.ComboBox
	preamp *players.Fader
	bands  []*players.Fader
}

func newEqPane(p *Player) *eqPane {
	e := &eqPane{p: p}
	e.Init(e)
	e.SetManagesChildren(true)

	e.on = players.NewGlyphButton(players.GlyphSliders, "Equaliser on", func() {
		p.Equaliser.On = !p.Equaliser.On
		p.refresh()
	})
	e.on.Toggle = true
	e.on.Checked = p.Equaliser.On

	e.flat = widgets.NewButton("Flat", func() { p.Equaliser.Flat() })
	e.flat.Tip = "Take every band back to zero"

	names := make([]string, 0, len(players.EqPresets()))
	for _, pr := range players.EqPresets() {
		names = append(names, pr.Name)
	}
	e.preset = widgets.NewComboBox(names, 0, func(i int) {
		if i >= 0 && i < len(names) {
			p.Equaliser.Apply(names[i])
		}
	})
	e.preset.SetAccessibleName("Preset")

	e.preamp = players.NewFader(-players.EqRange, players.EqRange, p.Equaliser.Preamp, "Preamp",
		func(v float32) { p.Equaliser.SetPreamp(v) })
	e.preamp.Format = players.Decibels
	for i, name := range players.EqBands {
		band := i
		f := players.NewFader(-players.EqRange, players.EqRange, p.Equaliser.Gains[i], name+" Hz",
			func(v float32) { p.Equaliser.Set(band, v) })
		f.Format = players.Decibels
		e.bands = append(e.bands, f)
	}

	e.Add(e.on)
	e.Add(e.flat)
	e.Add(e.preset)
	e.Add(e.preamp)
	for _, f := range e.bands {
		e.Add(f)
	}
	return e
}

// sync puts the model back into the faders — after a preset, after Flat.
func (e *eqPane) sync() {
	eq := e.p.Equaliser
	e.on.SetChecked(eq.On)
	e.preamp.Set(eq.Preamp)
	for i, f := range e.bands {
		if i < len(eq.Gains) {
			f.Set(eq.Gains[i])
		}
	}
	for i, pr := range players.EqPresets() {
		if pr.Name == eq.Preset {
			e.preset.Selected = i
		}
	}
	e.preset.Invalidate()
	e.Invalidate()
}

func (e *eqPane) Measure(c layout.Constraints) paintengine2d.Point {
	lk := e.Look()
	return c.Constrain(paintengine2d.Pt(style.Dip(lk, StripW), style.Dip(lk, 80)))
}

func (e *eqPane) Arrange(r paintengine2d.Rect) {
	e.SetBounds(r)
	lk := e.Look()
	dip := func(v float32) float32 { return style.Dip(lk, v) }
	b := e.LocalBounds()

	row := dip(18)
	e.on.Arrange(paintengine2d.XYWH(0, 0, row, row))
	e.flat.Arrange(paintengine2d.XYWH(row+dip(4), 0, dip(38), row))
	px := row + dip(4) + dip(38) + dip(4)
	e.preset.Arrange(paintengine2d.XYWH(px, 0, max(b.Dx()-px, dip(60)), row))

	// The faders: the preamp, a gap, then the ten bands sharing what is
	// left. Every one is the same width, so the row reads as a set.
	top := row + dip(6)
	h := max(b.Dy()-top-dip(11), dip(20))
	pre := dip(22)
	e.preamp.Arrange(paintengine2d.XYWH(0, top, pre, h))
	x := pre + dip(8)
	span := b.Dx() - x
	n := float32(len(e.bands))
	each := span / n
	for i, f := range e.bands {
		f.Arrange(paintengine2d.XYWH(x+float32(i)*each, top, each-dip(1), h))
	}
}

// Paint draws the zero rule behind the faders, which is the one mark that
// says which way is up on a control whose whole range is a number nobody
// reads.
func (e *eqPane) Paint(ctx *paintengine2d.Context) {
	lk, b := e.Look(), e.LocalBounds()
	if len(e.bands) == 0 {
		return
	}
	pal := lk.Palette()
	first := e.bands[0].Bounds()
	y := (first.Min.Y + first.Max.Y) / 2
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, y, b.Dx(), max(style.Dip(lk, 1), 1)),
		paintengine2d.Fill(pal.Divider))

	// The frequencies, under the faders they belong to. Nine design pixels
	// is small, and it is what the row of a control this size is: the
	// numbers are a scale, not a label anybody reads one of.
	tiny := players.ScaledFace(lk.Font(), style.Dip(lk, 9))
	row := paintengine2d.XYWH(0, first.Max.Y+style.Dip(lk, 1), 0, style.Dip(lk, 10))
	pre := e.preamp.Bounds()
	players.DrawTextIn(ctx, tiny, paintengine2d.XYWH(pre.Min.X, row.Min.Y, pre.Dx(), row.Dy()),
		"pre", pal.TextMuted, style.AlignCenter)
	for i, f := range e.bands {
		fb := f.Bounds()
		players.DrawTextIn(ctx, tiny, paintengine2d.XYWH(fb.Min.X, row.Min.Y, fb.Dx(), row.Dy()),
			players.EqBands[i], pal.TextMuted, style.AlignCenter)
	}
}

func (e *eqPane) KeyPress(ev widget.KeyEvent) bool { return e.p.Keys(e.p.Eq, ev) }

func (e *eqPane) CaptionAt(paintengine2d.Point) bool { return true }

func (e *eqPane) Describe(n *a11y.Node) {
	n.Role = a11y.RoleGroup
	if n.Name == "" {
		n.Name = "Equaliser"
	}
	n.Description = "Ten bands and a preamp. " + players.Disclaimer
}

// ---- the playlist ------------------------------------------------------------

// listPane is the queue: a row per track, and a footer that counts them.
type listPane struct {
	widget.Base
	p *Player

	list   *widgets.ListView
	foot   *foot
	toTop  *widgets.Button
	shrink *players.GlyphButton
}

func newListPane(p *Player) *listPane {
	l := &listPane{p: p}
	l.Init(l)
	l.SetManagesChildren(true)

	pl := p.Transport.List
	// The time goes before the title rather than after it: a row that runs
	// out of width loses the end of the title, and a playlist that hides
	// the one number it is there to show would be a strange playlist.
	l.list = widgets.NewListView(pl.Len(), func(i int) string {
		t := pl.At(i)
		return fmt.Sprintf("%2d.  %s   %s", i+1, players.Clock(t.Length), t.Label())
	}, func(i int) {
		p.Transport.SelectTrack(i)
		if p.Transport.State != players.Playing {
			p.Command(players.CmdPlayPause)
		}
	})
	l.list.SetAccessibleName("Playlist")
	l.foot = &foot{p: p}
	l.foot.Init(l.foot)
	l.toTop = widgets.NewButton("Top", func() {
		p.Transport.SelectTrack(0)
	})
	l.toTop.Tip = "Play the first track"
	l.shrink = players.NewGlyphButton(players.GlyphShrink, "Close the playlist", func() {
		p.toggle(p.iList)
	})

	l.Add(l.list)
	l.Add(l.foot)
	l.Add(l.toTop)
	l.Add(l.shrink)
	return l
}

func (l *listPane) sync() {
	if i := l.p.Transport.List.Index(); i >= 0 && l.list.Selected != i {
		l.list.Selected = i
		l.list.EnsureVisible(i)
	}
	l.list.Invalidate()
	l.foot.Invalidate()
}

func (l *listPane) Measure(c layout.Constraints) paintengine2d.Point {
	lk := l.Look()
	return c.Constrain(paintengine2d.Pt(style.Dip(lk, StripW), style.Dip(lk, 196)))
}

func (l *listPane) Arrange(r paintengine2d.Rect) {
	l.SetBounds(r)
	lk := l.Look()
	dip := func(v float32) float32 { return style.Dip(lk, v) }
	b := l.LocalBounds()
	row := dip(32)
	l.list.Arrange(paintengine2d.XYWH(0, 0, b.Dx(), max(b.Dy()-row-dip(4), dip(40))))
	y := b.Dy() - row
	btn := dip(38)
	ctl := dip(20)
	l.foot.Arrange(paintengine2d.XYWH(0, y, max(b.Dx()-btn-ctl-dip(8), 0), row))
	l.toTop.Arrange(paintengine2d.XYWH(b.Dx()-btn-ctl-dip(4), y+(row-ctl)/2, btn, ctl))
	l.shrink.Arrange(paintengine2d.XYWH(b.Dx()-ctl, y+(row-ctl)/2, ctl, ctl))
}

func (l *listPane) KeyPress(e widget.KeyEvent) bool { return l.p.Keys(l.p.List, e) }

func (l *listPane) Describe(n *a11y.Node) {
	n.Role = a11y.RoleGroup
	if n.Name == "" {
		n.Name = "Playlist"
	}
}

// foot is the playlist's status line: how many tracks and how long they run.
type foot struct {
	widget.Base
	p *Player
}

func (f *foot) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(style.Dip(f.Look(), 120), style.Dip(f.Look(), 32)))
}

func (f *foot) Arrange(b paintengine2d.Rect) { f.SetBounds(b) }

// Paint is two lines: what the queue holds, and what this application is.
//
// The second line is here rather than in the strip because the strip is two
// hundred and sixty-seven design pixels wide with a display, a seek bar and
// thirteen controls in it, and there is no honest place to put a sentence.
// The playlist window has the room, so this is where the compact player
// says out loud what it is — and it says it in the accessibility tree as
// well, where a window this small is read rather than looked at.
func (f *foot) Paint(ctx *paintengine2d.Context) {
	lk, b := f.Look(), f.LocalBounds()
	pal := lk.Palette()
	half := b.Dy() / 2
	lk.DrawLabel(ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), half),
		f.count(), pal.TextMuted, style.AlignStart)
	tiny := players.ScaledFace(lk.Font(), style.Dip(lk, 10))
	players.DrawTextIn(ctx, tiny, paintengine2d.XYWH(b.Min.X, b.Min.Y+half, b.Dx(), half),
		players.Short, style.Mix(pal.TextMuted, pal.Background, 0.2), style.AlignStart)
}

func (f *foot) count() string {
	pl := f.p.Transport.List
	return fmt.Sprintf("%d tracks · %s", pl.Len(), players.Clock(pl.Total()))
}

func (f *foot) text() string { return f.count() + " · " + players.Disclaimer }

func (f *foot) Describe(n *a11y.Node) {
	n.Role = a11y.RoleStatusBar
	n.Name = f.text()
}

func (f *foot) CaptionAt(paintengine2d.Point) bool { return true }
