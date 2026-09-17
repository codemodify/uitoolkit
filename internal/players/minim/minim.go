// Package minim is the compact player: a strip 275 by 116 design pixels
// with an equaliser and a playlist that stick to its edges and travel with
// it.
//
// It is a **visual demo**. Nothing is decoded and nothing is played; the
// clock counts, the seek bar scrubs the clock, the analyser is a function of
// where the clock is, and the equaliser's ten faders move and filter
// nothing. The window says so and so does its About box.
//
// What it is a demo *of* is three things the toolkit gained and no toolkit
// of the era had all of:
//
//   - A skin is a pack. Minim wears the "minim" skin, which is a pixel sheet
//     over win95; run it with UITK_THEME=breeze-night and it is the same app
//     in an ordinary theme, with every control still a control.
//   - A skinned window need not be a rectangle. The silhouette is the skin's
//     (window.shape), so the app never mentions it, and the strip stands on
//     a stepped chin with the desktop showing through beside it.
//   - The proportions are design pixels rather than a frozen size. 275 by
//     116 is what this shape was at 1×; at 1.75 it is 481 by 203 and the art
//     is drawn from the 2× sheet at whole multiples, because the skin is
//     declared pixelated.
package minim

import (
	"fmt"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/internal/players"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The window sizes, in logical pixels. They are the design pixels the skin
// is drawn on, so the strip is exactly the shape it was drawn as at 1× and
// exactly that shape scaled at every other scale.
const (
	StripW = 275
	StripH = 116
	EqH    = 116
	ListH  = 232
)

// Skin is the pack this player wears. It is an ordinary pack id: anything
// else in Settings runs the same app.
const Skin = "minim"

// Player is the three windows and the one model under them.
type Player struct {
	App  *app.Application
	Desk *players.Desk

	Transport *players.Transport
	Spectrum  *players.Spectrum
	Equaliser *players.Equalizer

	Main, Eq, List *app.Window
	strip          *strip
	eqPane         *eqPane
	listPane       *listPane
	pulse          *players.Pulse

	// iMain, iEq, iList are the panes in the desk's rack.
	iMain, iEq, iList int
	// stacked is whether the three have been put in a stack yet, and tries
	// how many times settle has asked — see settle, which cannot run until
	// the desktop has placed them and may not be obeyed when it does.
	stacked bool
	tries   int
}

// Options is what the command line passes in.
type Options struct {
	Headless bool
	// Scale is the display scale; 0 takes the desktop's.
	Scale float32
	// NoEq and NoList open the main strip on its own.
	NoEq, NoList bool
}

// New opens the player's windows and wires them to one model.
func New(a *app.Application, opts Options) (*Player, error) {
	list := players.NewLibrary()
	p := &Player{
		App:       a,
		Transport: players.NewTransport(list),
		Spectrum:  players.NewSpectrum(20),
		Equaliser: players.NewEqualizer(),
		iEq:       -1,
		iList:     -1,
	}
	p.Spectrum.Fall = 0.7

	main, err := p.open(a, opts, "Minim — a visual demo", StripW, StripH)
	if err != nil {
		return nil, err
	}
	p.Main = main
	p.strip = newStrip(p)
	main.SetContent(p.strip)

	// The rack's reach grows with the display, like every other measurement
	// in the toolkit: ten pixels of slop at 1× is a fingertip, and ten
	// device pixels at 2× is half of one.
	p.Desk = players.NewDesk(int(players.DefaultReach * max(main.Scale(), 1)))
	p.iMain = p.Desk.Add("main", main)

	if !opts.NoEq {
		w, err := p.open(a, opts, "Minim equaliser", StripW, EqH)
		if err != nil {
			return nil, err
		}
		p.Eq = w
		p.eqPane = newEqPane(p)
		w.SetContent(p.eqPane)
		p.iEq = p.Desk.Add("equaliser", w)
		p.Desk.Attach(p.iEq, p.iMain, players.SideBottom)
	}
	if !opts.NoList {
		w, err := p.open(a, opts, "Minim playlist", StripW, ListH)
		if err != nil {
			return nil, err
		}
		p.List = w
		p.listPane = newListPane(p)
		w.SetContent(p.listPane)
		p.iList = p.Desk.Add("playlist", w)
		p.Desk.Attach(p.iList, maxInt(p.iEq, p.iMain), players.SideBottom)
	}

	p.Transport.Changed = p.refresh
	p.Equaliser.Changed = p.refresh
	p.pulse = players.NewPulse(70*time.Millisecond, p.advance)
	p.refresh()
	// Something in every window has the keyboard from the first frame. A
	// player's keys are bare letters that bubble up from whatever is
	// focused, and a window with no focus at all has nothing for them to
	// bubble from — so the equaliser and the playlist are given one too,
	// or the transport keys would be dead in two windows out of three.
	p.Main.RequestFocus(p.strip.play)
	if p.eqPane != nil {
		p.Eq.RequestFocus(p.eqPane.preamp)
	}
	if p.listPane != nil {
		p.List.RequestFocus(p.listPane.list)
	}
	return p, nil
}

// open makes one of the player's windows. All three take the toolkit's own
// frame: the skin's caption band is the title strip a player of this shape
// had, and a *look's* silhouette is only asked of a window the toolkit
// frames — where the desktop draws the frame, the window is the rectangle
// inside it and there is nothing to cut.
func (p *Player) open(a *app.Application, opts Options, title string, w, h int) (*app.Window, error) {
	return players.OpenSized(a, platform.WindowOptions{
		Title:       title,
		Headless:    opts.Headless,
		Decorations: platform.DecorationsClient,
	}, w, h, true)
}

// Start begins the clock and shows the windows.
func (p *Player) Start() {
	p.Transport.Play()
	p.pulse.Start(p.Main)
}

// Pulse is the clock, for a test that wants to drive it by hand.
func (p *Player) Pulse() *players.Pulse { return p.pulse }

// Pose puts the player in a stated position with the analyser advanced to
// match, and touches no clock at all.
//
// It is what the screenshots and the golden tests use. A player posed at
// ninety-seven seconds is the same picture every time it is posed there —
// the clock reads the same, the seek bar is at the same place and the
// analyser is in the same shape, because the analyser is a function of the
// position and not of a timer.
func (p *Player) Pose(pos time.Duration) {
	p.Transport.State = players.Playing
	p.Transport.Pos = pos
	p.Spectrum.Reset()
	p.Spectrum.Advance(pos, true, 250*time.Millisecond)
	p.refresh()
}

// advance is one tick: the transport, the analyser, and a look at where the
// desktop has put the windows.
func (p *Player) advance(dt time.Duration) {
	p.Transport.Tick(dt)
	p.Spectrum.Advance(p.Transport.Pos, p.Transport.State == players.Playing, dt)
	p.settle()
	p.Desk.Follow()
	p.refresh()
}

// settle builds the stack once the desktop has said where it put the
// windows, and then never again.
//
// It cannot be done when the windows are made: a window manager places a
// window as it maps it and the app is told afterwards, so a stack arranged
// at construction is arranged around a strip that is still at the origin.
// On a desktop that will not place windows at all this never runs, which is
// right — there is nothing to arrange.
func (p *Player) settle() {
	if p.stacked || !p.Desk.Places() || !p.Desk.Adopt() {
		return
	}
	p.tries++
	if p.iEq >= 0 {
		p.Desk.Attach(p.iEq, p.iMain, players.SideBottom)
	}
	if p.iList >= 0 {
		p.Desk.Attach(p.iList, maxInt(p.iEq, p.iMain), players.SideBottom)
	}
	// It has taken when the panes are where their bonds say. A window
	// manager may refuse a move that would hang a window off the screen,
	// so it is asked a few times and then left alone; the panes are then
	// loose, and dragging one back to an edge snaps it as it always does.
	if p.placed() || p.tries >= settleTries {
		p.stacked = true
	}
}

// settleTries is how many clock ticks the stack is asked for before the
// desktop's answer is taken as final.
const settleTries = 20

// placed reports whether every pane's window is where the rack says.
func (p *Player) placed() bool {
	for i := 0; i < p.Desk.Rack.Len(); i++ {
		pane, w := p.Desk.Rack.Pane(i), p.Desk.Window(i)
		if pane == nil || w == nil || !pane.Shown {
			continue
		}
		if x, y, ok := w.Position(); !ok || x != pane.Box.X || y != pane.Box.Y {
			return false
		}
	}
	return true
}

// refresh puts the model back into the controls and repaints. Everything
// here is cheap and idempotent, so it runs on every tick rather than being
// threaded through twenty callbacks.
func (p *Player) refresh() {
	if p.strip != nil {
		p.strip.sync()
	}
	if p.eqPane != nil {
		p.eqPane.sync()
	}
	if p.listPane != nil {
		p.listPane.sync()
	}
}

// Command runs one of the shared transport commands, and keeps the clock
// running when it starts something.
func (p *Player) Command(c players.Command) {
	p.Transport.Do(c)
	if p.Transport.State == players.Playing {
		p.pulse.Start(p.Main)
	}
}

// Keys is the handler every one of the three windows shares, so the keys
// work wherever the focus is. It returns false for anything it does not
// know, which is what lets Tab, the arrows inside a list and the focus ring
// keep working.
//
// It is handed the window the key came from rather than reading the main
// one: the guard below asks what has the keyboard, and in a player with
// three windows that is a different answer in each.
func (p *Player) Keys(w *app.Window, e widget.KeyEvent) bool {
	if e.Key == platform.KeyEscape {
		p.App.Quit()
		return true
	}
	if players.Typing(w.Focus()) {
		// The focus is in something that takes text: its letters are its
		// own. Nothing the transport answers to is a modifier chord, so
		// there is nothing left to try.
		return false
	}
	if c := players.CommandFor(e); c != players.CmdNone {
		p.Command(c)
		return true
	}
	return false
}

// Status is the one line the strip shows about itself, and what the
// screenshots are read against.
func (p *Player) Status() string {
	t := p.Transport
	where := "loose"
	if pane := p.Desk.Rack.Pane(p.iEq); pane != nil && pane.To >= 0 {
		where = "stacked"
	}
	if !p.Desk.Places() {
		where = "this desktop places windows"
	}
	return fmt.Sprintf("%s · %s · %s", t.State, players.Percent(t.Volume), where)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---- the strip ---------------------------------------------------------------

// strip is the main window's content: a display, a seek bar and a transport
// row, laid out by hand because a compact player's proportions *are* its
// design and a flex box would give them away.
type strip struct {
	widget.Base
	p *Player

	readout  *readout
	analyser *players.AnalyserView
	seek     *widgets.Slider
	volume   *widgets.Slider

	prev, play, stop, next *players.GlyphButton
	mute                   *players.GlyphButton
	shuffle, repeat        *players.GlyphButton
	eqBtn, listBtn         *players.GlyphButton

	// seeking is set while the seek bar is being scrubbed, so the clock
	// does not fight the pointer for the thumb.
	seeking bool
}

func newStrip(p *Player) *strip {
	s := &strip{p: p}
	s.Init(s)
	s.SetManagesChildren(true)

	s.readout = newReadout(p)
	s.analyser = players.NewAnalyserView(p.Spectrum)
	s.analyser.Drag = p.Main
	s.analyser.Pixelated = true
	s.analyser.Gap = 1
	s.analyser.MinBarW = 2
	s.analyser.Peaks = true

	s.seek = widgets.NewSlider(0, 1, 0, func(v float32) {
		if s.seeking {
			p.Transport.Seek(v)
		}
	})
	s.seek.SetAccessibleName("Seek")
	s.volume = widgets.NewSlider(0, 100, p.Transport.Volume*100, func(v float32) {
		p.Transport.SetVolume(v / 100)
	})
	s.volume.SetAccessibleName("Volume")

	s.prev = players.NewGlyphButton(players.GlyphPrev, "Previous track", func() { p.Command(players.CmdPrev) })
	s.play = players.NewGlyphButton(players.GlyphPlay, "Play", func() { p.Command(players.CmdPlayPause) })
	s.stop = players.NewGlyphButton(players.GlyphStop, "Stop", func() { p.Command(players.CmdStop) })
	s.next = players.NewGlyphButton(players.GlyphNext, "Next track", func() { p.Command(players.CmdNext) })
	s.mute = players.NewGlyphButton(players.GlyphVolume, "Mute", func() { p.Command(players.CmdMute) })
	s.mute.Toggle = true

	s.shuffle = players.NewGlyphButton(players.GlyphShuffle, "Shuffle", func() { p.Command(players.CmdShuffle) })
	s.shuffle.Toggle = true
	s.repeat = players.NewGlyphButton(players.GlyphRepeat, "Repeat", func() { p.Command(players.CmdRepeat) })
	s.repeat.Toggle = true
	s.eqBtn = players.NewGlyphButton(players.GlyphSliders, "Equaliser window", func() { p.toggle(p.iEq) })
	s.eqBtn.Toggle = true
	s.listBtn = players.NewGlyphButton(players.GlyphList, "Playlist window", func() { p.toggle(p.iList) })
	s.listBtn.Toggle = true

	for _, c := range s.kids() {
		s.Add(c)
	}
	return s
}

func (s *strip) kids() []widget.Component {
	return []widget.Component{
		s.readout, s.analyser, s.seek,
		s.prev, s.play, s.stop, s.next,
		s.mute, s.volume,
		s.shuffle, s.repeat, s.eqBtn, s.listBtn,
	}
}

// toggle opens or closes one of the satellite windows.
func (p *Player) toggle(i int) {
	pane := p.Desk.Rack.Pane(i)
	if pane == nil {
		return
	}
	p.Desk.Show(i, !pane.Shown)
	p.refresh()
}

// sync puts the model into the controls: the play button's glyph, the
// toggles, the seek bar's thumb.
func (s *strip) sync() {
	t := s.p.Transport
	if t.State == players.Playing {
		s.play.SetGlyph(players.GlyphPause, "Pause")
	} else {
		s.play.SetGlyph(players.GlyphPlay, "Play")
	}
	s.mute.SetGlyph(glyphForVolume(t), muteLabel(t))
	s.mute.SetChecked(t.Muted())
	s.shuffle.SetChecked(t.Random)
	s.repeat.SetChecked(t.Repeat != players.RepeatOff)
	if t.Repeat == players.RepeatOne {
		s.repeat.SetGlyph(players.GlyphRepeatOne, "Repeat one")
	} else {
		s.repeat.SetGlyph(players.GlyphRepeat, t.Repeat.String())
	}
	if p := s.p.Desk.Rack.Pane(s.p.iEq); p != nil {
		s.eqBtn.SetChecked(p.Shown)
	}
	if p := s.p.Desk.Rack.Pane(s.p.iList); p != nil {
		s.listBtn.SetChecked(p.Shown)
	}
	if !s.seeking {
		s.seek.Value = t.Fraction()
	}
	s.volume.Value = t.Volume * 100
	s.readout.Invalidate()
	s.analyser.Invalidate()
	s.seek.Invalidate()
	s.volume.Invalidate()
}

func glyphForVolume(t *players.Transport) players.Glyph {
	if t.Muted() {
		return players.GlyphMute
	}
	return players.GlyphVolume
}

func muteLabel(t *players.Transport) string {
	if t.Muted() {
		return "Unmute"
	}
	return "Mute"
}

// Arrange is the whole design of the compact player: three rows in eighty
// design pixels, stated in the units the art was drawn in.
func (s *strip) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	lk := s.Look()
	dip := func(v float32) float32 { return style.Dip(lk, v) }
	b := s.LocalBounds()
	w := b.Dx()

	// Row A: the display, with the analyser inside its right-hand end.
	displayH := dip(38)
	display := paintengine2d.XYWH(0, 0, w, displayH)
	s.readout.Arrange(display)
	an := dip(AnalyserWidth) - dip(6)
	s.analyser.Arrange(paintengine2d.XYWH(display.Max.X-an-dip(3), display.Min.Y+dip(4), an, displayH-dip(9)))

	// Row B: the seek bar, the full width of the strip.
	y := displayH + dip(5)
	seekH := dip(10)
	s.seek.Arrange(paintengine2d.XYWH(0, y, w, seekH))

	// Row C: transport, volume, toggles.
	y += seekH + dip(5)
	btn := dip(21)
	x := float32(0)
	for _, c := range []*players.GlyphButton{s.prev, s.play, s.stop, s.next} {
		c.Arrange(paintengine2d.XYWH(x, y, btn, btn))
		x += btn + dip(1)
	}
	x += dip(5)
	s.mute.Arrange(paintengine2d.XYWH(x, y, dip(18), btn))
	x += dip(19)

	// The toggles are pinned to the right-hand end and the volume slider
	// takes whatever is between: the row keeps its shape when the strip is
	// wider than it was drawn.
	small := dip(19)
	toggles := []*players.GlyphButton{s.shuffle, s.repeat, s.eqBtn, s.listBtn}
	tw := float32(len(toggles))*small + float32(len(toggles)-1)*dip(1)
	tx := w - tw
	volW := max(tx-dip(6)-x, dip(24))
	s.volume.Arrange(paintengine2d.XYWH(x, y+(btn-dip(10))/2, volW, dip(10)))
	for _, c := range toggles {
		c.Arrange(paintengine2d.XYWH(tx, y+(btn-small)/2, small, small))
		tx += small + dip(1)
	}
}

func (s *strip) Measure(c layout.Constraints) paintengine2d.Point {
	lk := s.Look()
	return c.Constrain(paintengine2d.Pt(style.Dip(lk, StripW), style.Dip(lk, 80)))
}

// MousePress on the strip itself starts a window move, so the shell between
// the controls is a drag handle — which is what the whole face of a player
// this size has always been.
func (s *strip) MousePress(widget.MouseEvent) bool {
	s.p.Main.StartMove()
	return true
}

func (s *strip) KeyPress(e widget.KeyEvent) bool { return s.p.Keys(s.p.Main, e) }

// CaptionAt tells the frame that the strip's own background is caption: a
// press there moves the window rather than landing in the app.
func (s *strip) CaptionAt(paintengine2d.Point) bool { return true }

func (s *strip) Describe(n *a11y.Node) {
	n.Role = a11y.RolePane
	if n.Name == "" {
		n.Name = "Player"
	}
	n.Description = players.Disclaimer
}

// ---- the display -------------------------------------------------------------

// readout is the phosphor well: the clock, the track, and the flags a player
// of this shape printed beside them.
//
// It is painted rather than assembled from labels because the whole of it is
// one object — a display — and because the colours come from the look's
// palette, so the same component is a phosphor well under the skin and an
// ordinary sunken field under any of the other packs.
type readout struct {
	players.DragsWindow
	p *Player
}

func newReadout(p *Player) *readout {
	r := &readout{p: p}
	r.Init(r)
	r.Win = p.Main
	return r
}

func (r *readout) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(style.Dip(r.Look(), 200), style.Dip(r.Look(), 38)))
}

func (r *readout) Arrange(b paintengine2d.Rect) { r.SetBounds(b) }

// AnalyserWidth is how much of the display's right-hand end the analyser
// takes, in design pixels. The strip's Arrange puts it there and the
// readout leaves that much room, so the two agree in one place.
const AnalyserWidth = 74

func (r *readout) Paint(ctx *paintengine2d.Context) {
	lk, b := r.Look(), r.LocalBounds()
	pal := lk.Palette()
	t := r.p.Transport
	dip := func(v float32) float32 { return style.Dip(lk, v) }
	line := max(dip(1), 1)

	// The well. Palette colours, so this is a phosphor display under the
	// skin and an ordinary sunken field under any other pack.
	ctx.DrawRect(b, paintengine2d.Fill(pal.Field))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), line), paintengine2d.Fill(pal.FieldBorder))
	ctx.DrawRect(paintengine2d.XYWH(b.Min.X, b.Max.Y-line, b.Dx(), line), paintengine2d.Fill(pal.FieldBorder))

	pad := dip(4)
	right := b.Max.X - dip(AnalyserWidth)

	// The clock, in the pack's mono face at a display size: the one number
	// a player of this shape is read by.
	clock := players.Clock(t.Pos)
	if t.State == players.Stopped {
		clock = players.Clock(0)
	}
	big := players.ScaledFace(lk.MonoFont(), dip(17))
	cw := big.Advance(clock)
	players.DrawTextIn(ctx, big, paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+dip(2), cw, dip(22)),
		clock, pal.Accent, style.AlignStart)

	// The flags beside it: what the transport is doing, and the invented
	// rate a player of this era printed whether anyone read it or not.
	tr := t.Track()
	x := b.Min.X + pad + cw + dip(6)
	small := players.ScaledFace(lk.Font(), dip(10))
	players.DrawTextIn(ctx, small, paintengine2d.XYWH(x, b.Min.Y+dip(3), max(right-x-dip(4), 0), dip(20)),
		fmt.Sprintf("%s · %d kHz · %s", shortState(t.State), tr.Rate, channels(tr)), pal.TextMuted, style.AlignStart)

	// The title under both, scrolling when it does not fit.
	players.DrawScrollingText(ctx, lk, lk.Font(),
		paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+dip(23), max(right-b.Min.X-2*pad, 0), dip(14)),
		tr.Label(), pal.Text, t.Pos)
}

func shortState(s players.State) string {
	switch s {
	case players.Playing:
		return "play"
	case players.Paused:
		return "pause"
	}
	return "stop"
}

func channels(t players.Track) string {
	if t.Channels >= 2 {
		return "stereo"
	}
	return "mono"
}

// Describe is the display as a screen reader meets it: one label whose name
// is everything the well says, so a reader announces the track and the time
// rather than a blank rectangle.
func (r *readout) Describe(n *a11y.Node) {
	t := r.p.Transport
	n.Role = a11y.RoleLabel
	n.Name = fmt.Sprintf("%s. %s of %s. %s.",
		t.Track().Label(), players.Clock(t.Pos), players.Clock(t.Length()), t.State)
	n.Description = players.Disclaimer
}
