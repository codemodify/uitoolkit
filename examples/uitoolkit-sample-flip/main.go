// Command flip puts an application's settings on the back of the window
// and turns the window over to reach them — the gesture Dashboard's
// widgets used, where the ⓘ in the corner flipped the thing you were
// using to the thing that configured it.
//
// The front is a small player. The gear in its corner turns the card over
// to the settings, and Done turns it back. Both sides are live: the
// settings are real controls that change what the front shows, not a
// picture of a settings page, because a [widgets.FlipPanel] only draws a
// face as a picture for the length of the turn.
//
//	go run ./examples/uitoolkit-sample-flip
//	go run ./examples/uitoolkit-sample-flip -back    # open showing the settings
//	go run ./examples/uitoolkit-sample-flip -perspective 1.4
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write flip.png")
	back := flag.Bool("back", false, "open showing the settings side")
	persp := flag.Float64("perspective", 0, "camera distance in half-widths (0 takes the default)")
	strips := flag.Int("strips", 0, "columns the face is cut into (0 takes the default)")
	motion := flag.Bool("motion", false, "animate even where the desktop asks for reduced motion")
	whole := flag.Bool("window", false, "turn the whole window over, not a card inside it")
	flag.Parse()

	// A desktop with its animations switched off (Plasma's
	// AnimationDurationFactor=0, which the portal reports as
	// reduced-motion) turns off every animation in every uitoolkit app —
	// correctly, and including the turn this sample exists to show. The
	// variable is the documented way to say otherwise, and it belongs to
	// whoever starts the program, which here is this flag.
	if *motion {
		os.Setenv(style.AnimationsEnv, "1")
	}

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	// The window says which projection it is running, so several of these
	// open at once can be told apart in the panel and compared turning
	// side by side.
	name := "Flip"
	switch {
	case *persp > 0 && *strips > 0:
		name = fmt.Sprintf("Flip (camera %.2g, %d strips)", *persp, *strips)
	case *persp > 0:
		name = fmt.Sprintf("Flip (camera %.2g)", *persp)
	case *strips > 0:
		name = fmt.Sprintf("Flip (%d strips)", *strips)
	}
	opts := platform.WindowOptions{
		Title: windowTitle(name), Width: 520, Height: 360, MinWidth: 420, MinHeight: 300,
	}
	if *whole {
		// No frame from anybody: a window that turns over draws its own,
		// on the faces, so that the frame turns with them. The desktop's
		// frame cannot turn (the toolkit does not draw it) and the
		// toolkit's own sits outside the content, so either would be the
		// one part that stayed still — and under DecorationsClient the
		// first version of this wore two captions, the window's and the
		// face's. DecorationsNone is the mode for exactly this: "no
		// frame at all; the app draws whatever it wants".
		opts.Decorations = platform.DecorationsNone
	}
	win, err := a.NewWindow(opts)
	if err != nil {
		log.Fatal(err)
	}

	card := &card{volume: 65, quality: 2}
	if *whole {
		// The whole application turns over: the two faces are the
		// window's content, and the window takes the card's silhouette
		// while it moves, so what rotates is the window itself. There is
		// no status line here — a bar outside the card would be a strip
		// that did not turn with the window it belongs to.
		//
		// The frame has to go *on the faces*. A window wears its frame
		// outside its content, so a frame left to the window would be
		// the one part that stayed still while everything inside it
		// turned; and dropping the frame without putting one back is how
		// the first version of this came out with no border at all. Each
		// face is an in-app window instead — the look's own frame,
		// caption and close button, drawn by the era in its own style —
		// so the border turns with the card that wears it.
		// The faces carry the chrome, so they carry what the chrome does:
		// the caption hands the press to the desktop for a move, or the
		// window has a title bar that cannot move it.
		card.chrome, card.onClose, card.onMove = true, a.Quit, win.StartMove
		card.adopt(win.SetFlipFaces(card.front(), card.settings()))
	} else {
		// A card inside an ordinary window: the window does not move, and
		// the status line sits outside the card saying what it is doing.
		card.adopt(widgets.NewFlipPanel(nil, nil))
		card.panel.SetFront(card.front())
		card.panel.SetBack(card.settings())
		status := widgets.NewStatusBar("", "")
		show := func(onBack bool) {
			side := "Front — Settings turns the card over"
			if onBack {
				side = "Settings — Done turns it back"
			}
			status.SetParts(side, card.summary())
		}
		card.onFlip = show
		show(false)
		pad := widgets.NewPad(12, card.panel)
		body := widgets.NewColumn(pad, status).WithGap(0)
		body.AddFlex(pad, 1)
		win.SetContent(body)
	}
	card.panel.Perspective = float32(*persp)
	card.panel.Strips = *strips
	if *back {
		card.panel.ShowBack()
	}

	if *headless {
		if err := win.WritePNG("flip.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote flip.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

// card is the two-sided card: a player on the front, its settings on the
// back, and the state they share.
type card struct {
	panel *widgets.FlipPanel

	// What the settings change and the front shows.
	shuffle bool
	volume  float32
	quality int
	track   int
	playing bool

	title  *widgets.Label
	detail *widgets.Label
	play   *widgets.Button
	// onFlip is what the window around the card wants to hear, where
	// there is anything outside the card to tell.
	onFlip func(back bool)
	// chrome puts the look's window frame on each face, for the mode
	// where the faces are the window; onClose and onMove are what its
	// close button and its caption do.
	chrome  bool
	onClose func()
	onMove  func() bool
}

var (
	qualities = []string{"Low", "Normal", "High", "Lossless"}
	tracks    = []string{"Weightless", "An Ending (Ascent)", "Avril 14th", "Music for Airports"}
)

// front is the face the application shows: a track, and the gear that
// turns the card over.
func (c *card) front() widget.Component {
	c.title = widgets.NewTitle("")
	c.detail = widgets.NewLabel("")
	c.detail.Wrap = true

	// A button that says what it does, not a bare mark in the corner:
	// the first person to run this sample pressed Play and reported that
	// nothing happened, which is exactly right — the control that turns
	// the card over was an unlabelled icon and the big obvious buttons
	// were dead.
	gear := widgets.NewButton("Settings", func() { c.panel.ShowBack() })
	gear.Icon = style.IconSettings
	gear.Tip = "Turn the card over"

	headGap := widgets.NewSpacer()
	head := widgets.NewRow(c.title, headGap, gear)
	head.AddFlex(headGap, 1)
	fill := widgets.NewSpacer()
	c.play = widgets.NewButton("", c.playPause)
	col := widgets.NewColumn(
		head,
		widgets.NewSeparator(),
		c.detail,
		fill,
		widgets.NewRow(
			widgets.NewButton("Previous", func() { c.step(-1) }),
			c.play,
			widgets.NewButton("Next", func() { c.step(1) }),
		),
	).WithGap(10)
	col.AddFlex(fill, 1)
	c.refresh()
	return c.face(windowTitle("Flip"), col)
}

// adopt points the card at the panel that will hold its faces — the one
// the window built, or one of its own — so the Settings and Done buttons
// turn that one.
func (c *card) adopt(p *widgets.FlipPanel) {
	c.panel = p
	p.OnFlip = func(onBack bool) {
		c.refresh()
		if c.onFlip != nil {
			c.onFlip(onBack)
		}
	}
}

// step moves through the tracks, and playPause starts and stops. Neither
// plays anything — there is no audio here — but a demo with buttons that
// do nothing teaches the reader that the toolkit's buttons do nothing.
func (c *card) step(by int) {
	c.track = (c.track + by + len(tracks)) % len(tracks)
	c.refresh()
}

func (c *card) playPause() {
	c.playing = !c.playing
	c.refresh()
}

// settings is the face on the back: the same state, as controls.
func (c *card) settings() widget.Component {
	form := widgets.NewForm()
	form.AddRow("Quality", widgets.NewComboBox(qualities, c.quality, func(i int) {
		c.quality = i
		c.refresh()
	}))
	form.AddRow("Volume", widgets.NewSlider(0, 100, c.volume, func(v float32) {
		c.volume = v
		c.refresh()
	}))
	form.AddRow("", widgets.NewSwitch("Shuffle", c.shuffle, func(v bool) {
		c.shuffle = v
		c.refresh()
	}))

	done := widgets.NewButton("Done", func() { c.panel.ShowFront() })
	footGap := widgets.NewSpacer()
	foot := widgets.NewRow(footGap, done)
	foot.AddFlex(footGap, 1)
	fill := widgets.NewSpacer()
	col := widgets.NewColumn(
		widgets.NewTitle("Settings"),
		widgets.NewSeparator(),
		form,
		fill,
		foot,
	).WithGap(10)
	col.AddFlex(fill, 1)
	return c.face(windowTitle("Flip — settings"), col)
}

// face is one side of the card: a panel for the border and the look's
// own surface, holding the content padded and stretched to fill it. A
// panel holds its child at the child's own height unless the child is
// given the flex, and a half-filled card would turn over to show where
// the content stopped.
func (c *card) face(title string, content widget.Component) widget.Component {
	pad := widgets.NewPad(14, content)
	panel := widgets.NewPanel("")
	if c.chrome {
		// The look's own window frame: Windows 95's navy caption, Aqua's
		// traffic lights, Adwaita's rounded header — whatever the era
		// gives a window, drawn here on the face so that it turns.
		panel.Window = true
		panel.Title = title
		panel.OnClose = c.onClose
		panel.OnCaptionPress = c.onMove
	}
	panel.Add(pad)
	panel.Content().AddFlex(pad, 1)
	return panel
}

// refresh restates on the front what the settings now say. The front is a
// live subtree whichever way the card is facing, so there is nothing to
// redraw by hand and nothing to copy across at the turn.
func (c *card) refresh() {
	if c.detail == nil {
		return
	}
	c.title.Text = tracks[c.track]
	c.title.Invalidate()
	c.detail.Text = c.summary()
	c.detail.Invalidate()
	if c.play != nil {
		c.play.Text = "Play"
		if c.playing {
			c.play.Text = "Pause"
		}
		c.play.Invalidate()
	}
}

func (c *card) summary() string {
	shuffle := "in order"
	if c.shuffle {
		shuffle = "shuffled"
	}
	state := "stopped"
	if c.playing {
		state = "playing"
	}
	return fmt.Sprintf("%s · %s · volume %.0f · %s", state, qualities[c.quality], c.volume, shuffle)
}

// windowTitle is what a window of this sample is called on the desktop:
// the sample's own name for the window under the toolkit's prefix, so
// that a desktop with several samples open says which toolkit they
// belong to. Every title this sample sets goes through here, so one
// computed while the app runs carries the prefix too.
func windowTitle(name string) string { return "uitoolkit - " + name }
