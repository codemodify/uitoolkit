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
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: windowTitle("Flip"), Width: 520, Height: 360, MinWidth: 420, MinHeight: 300,
	})
	if err != nil {
		log.Fatal(err)
	}

	card := &card{}
	card.build()
	card.panel.Perspective = float32(*persp)
	card.panel.Strips = *strips
	if *back {
		card.panel.ShowBack()
	}
	// The status line is outside the card, so it says what the card is
	// doing from somewhere that does not turn with it.
	status := widgets.NewStatusBar("", "")
	showSide := func(onBack bool) {
		side := "Front — the gear turns the card over"
		if onBack {
			side = "Settings — Done turns it back"
		}
		status.SetParts(side, card.summary())
	}
	card.panel.OnFlip = showSide
	showSide(card.panel.Showing())

	pad := widgets.NewPad(12, card.panel)
	body := widgets.NewColumn(pad, status).WithGap(0)
	body.AddFlex(pad, 1)
	win.SetContent(body)

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

	title  *widgets.Label
	detail *widgets.Label
}

var qualities = []string{"Low", "Normal", "High", "Lossless"}

func (c *card) build() {
	c.volume, c.quality = 65, 2
	// The panel first, empty: both faces carry a control that turns the
	// card over, so they need the panel that will hold them.
	c.panel = widgets.NewFlipPanel(nil, nil)
	c.panel.SetFront(c.front())
	c.panel.SetBack(c.settings())
}

// front is the face the application shows: a track, and the gear that
// turns the card over.
func (c *card) front() widget.Component {
	c.title = widgets.NewTitle("Weightless")
	c.detail = widgets.NewLabel("")
	c.detail.Wrap = true
	gear := widgets.NewIconButton(style.IconSettings, "Settings", func() { c.panel.ShowBack() })

	headGap := widgets.NewSpacer()
	head := widgets.NewRow(c.title, headGap, gear)
	head.AddFlex(headGap, 1)
	fill := widgets.NewSpacer()
	col := widgets.NewColumn(
		head,
		widgets.NewSeparator(),
		c.detail,
		fill,
		widgets.NewRow(
			widgets.NewButton("Previous", nil),
			widgets.NewButton("Play", nil),
			widgets.NewButton("Next", nil),
		),
	).WithGap(10)
	col.AddFlex(fill, 1)
	c.refresh()
	return face(col)
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
	return face(col)
}

// face is one side of the card: a panel for the border and the look's
// own surface, holding the content padded and stretched to fill it. A
// panel holds its child at the child's own height unless the child is
// given the flex, and a half-filled card would turn over to show where
// the content stopped.
func face(content widget.Component) widget.Component {
	pad := widgets.NewPad(14, content)
	panel := widgets.NewPanel("")
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
	c.detail.Text = c.summary()
	c.detail.Invalidate()
}

func (c *card) summary() string {
	shuffle := "in order"
	if c.shuffle {
		shuffle = "shuffled"
	}
	return fmt.Sprintf("%s · volume %.0f · %s", qualities[c.quality], c.volume, shuffle)
}

// windowTitle is what a window of this sample is called on the desktop:
// the sample's own name for the window under the toolkit's prefix, so
// that a desktop with several samples open says which toolkit they
// belong to. Every title this sample sets goes through here, so one
// computed while the app runs carries the prefix too.
func windowTitle(name string) string { return "uitoolkit - " + name }
