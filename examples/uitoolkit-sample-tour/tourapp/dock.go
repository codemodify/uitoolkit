package tourapp

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/dock"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Panels around a centre. The gestures are the point — a panel
// dragged by its title bar to another side, onto another panel to tab
// with it, or clean out of the window into one of its own — so the page
// is mostly the dock host itself, with a row of buttons that does the
// same things without a drag, and the layout JSON beside it so that the
// arrangement is visibly a value and not a mood.

func init() {
	p := &tourPages[pageDock]
	p.title = "Panels around a centre"
	p.proof = "Panels dock to the four sides of a centre, tab together, split, float into windows of " +
		"their own and come back — and the whole arrangement is JSON you can write out and read back."
	p.try = "Drag a panel by its title bar: to a side, onto another panel, or out."
	p.build = buildDockPage
}

type dockPage struct {
	t     *tourState
	host  *dock.Host
	names []string
	pick  *widgets.ComboBox
	facts *widgets.TextArea
	state *widgets.Label
	// saved is the layout the user put by with "Remember", to show that a
	// layout survives being written down and read back.
	saved []byte
	// floating is which panels were in windows of their own at the last
	// change, so a panel that floats or docks by a drag — which no button
	// here hears about — still gets the status line it would have got.
	floating map[string]bool
}

func buildDockPage(t *tourState) widget.Component {
	p := &dockPage{t: t}
	t.own(pageDock, p)

	centre := widgets.NewPanel("The centre",
		tourNote("The centre is the thing the panels are arranged around — an editor, a canvas, a "+
			"message list. It is an ordinary widget: the dock host never asks it to be anything."),
		tourNote("Everything else on this page is a panel. Drag one by its title bar: the host shows "+
			"where it would land — a whole side, a split beside the panel you are over, or its tab strip "+
			"— and drops it there. Drag it past the window's edge and it floats."),
	)
	centre.Content().Spec.Gap = 8

	outline := widgets.NewTreeView(
		widgets.NewTreeNode("Window",
			widgets.NewTreeNode("HeaderBar",
				widgets.NewTreeNode("BrowserTabs"),
				widgets.NewTreeNode("WindowControls")),
			widgets.NewTreeNode("Dock host",
				widgets.NewTreeNode("Outline"),
				widgets.NewTreeNode("Notes"),
				widgets.NewTreeNode("Log"))))
	outline.SetAccessibleName("Outline")

	notes := widgets.NewTextArea("A panel holds any widget at all.\n\nType here, float this panel "+
		"into a window of its own, and dock it back: it is the same widget throughout, so the caret "+
		"stays where you left it.", "", nil)
	notes.SetAccessibleName("Notes")

	logView := widgets.NewMonoTextView("dock: ready\n", "")
	logView.SetAccessibleName("Log")

	swatches := widgets.NewWrap()
	for _, s := range []string{"Accent", "Surface", "Text", "Muted"} {
		swatches.Add(widgets.NewButton(s, nil))
	}

	p.host = dock.NewHost(centre)
	mk := func(name, title string, c widget.Component, w, h float32) *dock.Panel {
		pan := dock.NewPanel(name, title, c)
		pan.SetMinSize(w, h)
		return pan
	}
	panels := []*dock.Panel{
		mk("outline", "Outline", outline, 150, 90),
		mk("notes", "Notes", widgets.NewPad(4, notes), 190, 110),
		mk("palette", "Palette", tourScroll("Palette", widgets.NewPad(6, swatches)), 150, 80),
		mk("log", "Log", widgets.NewPad(4, logView), 170, 70),
	}
	p.host.Dock(panels[0], dock.SideLeft)
	p.host.Dock(panels[1], dock.SideRight)
	p.host.Dock(panels[2], dock.SideRight)
	p.host.Dock(panels[3], dock.SideBottom)
	p.host.SetDefaultLayout()
	for _, pan := range panels {
		p.names = append(p.names, pan.Title())
	}

	// DockHost gives the host somewhere to put a floating panel, and takes
	// the window's close hook to shut those windows again; the tour wants
	// its own hook back, with that one folded into it.
	app.DockHost(t.win, p.host)
	t.onClose(p.host.CloseFloating)
	t.installCloseHook()

	p.host.OnLayoutChanged = func() {
		p.noteDrags()
		p.refresh()
	}

	// ---- the same moves, without a drag ----------------------------------

	p.pick = widgets.NewComboBox(p.names, 0, func(int) { p.refresh() })
	p.pick.SetAccessibleName("Panel")

	side := func(label string, s dock.Side) *widgets.Button {
		return widgets.NewButton(label, func() {
			pan := p.chosen()
			if pan == nil {
				return
			}
			if pan.Floating() {
				p.host.DockPanel(pan)
			}
			p.host.Dock(pan, s)
			pan.Show()
			p.note(pan.Title() + " docked " + s.String())
		})
	}

	float := widgets.NewButton("Float", func() {
		pan := p.chosen()
		if pan == nil {
			return
		}
		if pan.Floating() {
			p.note(pan.Title() + " is already in a window of its own.")
			return
		}
		if !pan.Float() {
			p.note("This desktop would not give " + pan.Title() + " a window.")
			return
		}
		p.note(pan.Title() + " is in a window of its own — drag it back over the host to dock it.")
	})
	float.Primary = true
	back := widgets.NewButton("Dock back", func() {
		pan := p.chosen()
		if pan == nil {
			return
		}
		if !pan.Floating() {
			p.note(pan.Title() + " is already docked.")
			return
		}
		pan.Dock()
		p.note(pan.Title() + " came back to where it was.")
	})

	tabWith := widgets.NewButton("Tab with the next panel", func() {
		pan := p.chosen()
		if pan == nil {
			return
		}
		other := p.host.Panel(p.nameOf((p.pick.Selected + 1) % len(p.names)))
		if other == nil || other == pan {
			return
		}
		if pan.Floating() {
			p.host.DockPanel(pan)
		}
		p.host.DockInto(pan, other)
		p.note(pan.Title() + " and " + other.Title() + " share a tab strip now.")
	})
	splitWith := widgets.NewButton("Split under the next panel", func() {
		pan := p.chosen()
		if pan == nil {
			return
		}
		other := p.host.Panel(p.nameOf((p.pick.Selected + 1) % len(p.names)))
		if other == nil || other == pan {
			return
		}
		if pan.Floating() {
			p.host.DockPanel(pan)
		}
		p.host.DockBeside(pan, other, dock.SideBottom)
		p.note(pan.Title() + " split off the bottom of " + other.Title() + ".")
	})

	collapse := widgets.NewButton("Collapse", func() {
		pan := p.chosen()
		if pan == nil {
			return
		}
		pan.ToggleCollapsed()
		p.note(pan.Title() + " " + map[bool]string{true: "collapsed to its title bar", false: "expanded"}[pan.Collapsed()])
	})
	hide := widgets.NewButton("Close / show", func() {
		pan := p.chosen()
		if pan == nil {
			return
		}
		if pan.Closed() {
			pan.Show()
			p.note(pan.Title() + " is back.")
			return
		}
		pan.Close()
		p.note(pan.Title() + " is closed — a closed panel keeps its place in the layout.")
	})

	remember := widgets.NewButton("Remember this layout", func() {
		b, err := p.host.LayoutJSON()
		if err != nil {
			p.note("The layout could not be written: " + err.Error())
			return
		}
		p.saved = b
		p.note("Layout remembered — " + strconv.Itoa(len(b)) + " bytes of JSON.")
		p.refresh()
	})
	restore := widgets.NewButton("Put it back", func() {
		if len(p.saved) == 0 {
			p.note("Nothing has been remembered yet.")
			return
		}
		if err := p.host.ApplyLayoutJSON(p.saved); err != nil {
			p.note("The layout could not be read: " + err.Error())
			return
		}
		p.note("Layout read back from JSON.")
	})
	reset := widgets.NewButton("Reset", func() {
		if p.host.ResetLayout() {
			p.note("Back to the layout the page started with.")
		}
	})

	p.state = tourNote("")
	moves := widgets.NewPanel("Move a panel without dragging it",
		tourRow("Panel", p.pick),
		widgets.NewRow(side("Left", dock.SideLeft), side("Right", dock.SideRight),
			side("Top", dock.SideTop), side("Bottom", dock.SideBottom)).WithGap(6),
		widgets.NewRow(float, back, collapse, hide).WithGap(6),
		widgets.NewRow(tabWith, splitWith).WithGap(6),
		widgets.NewSeparator(),
		widgets.NewRow(remember, restore, reset).WithGap(6),
		p.state,
	)
	moves.Content().Spec.Gap = 8

	panel, facts := tourReadout("The layout, as JSON")
	p.facts = facts

	stage := widgets.NewColumn(p.host, tourScroll("Panel moves", moves)).WithGap(10)
	stage.AddFlex(p.host, 3)
	stage.AddFlex(stage.Children()[1], 2)

	p.refresh()
	return tourStage(stage, panel)
}

func (p *dockPage) nameOf(i int) string {
	pans := p.host.Panels()
	if i < 0 || i >= len(pans) {
		return ""
	}
	return pans[i].Name()
}

func (p *dockPage) chosen() *dock.Panel {
	pans := p.host.Panels()
	i := p.pick.Selected
	if i < 0 || i >= len(pans) {
		return nil
	}
	return pans[i]
}

func (p *dockPage) note(s string) {
	p.t.note(s)
	p.refresh()
}

// noteDrags says what a change of layout did to a panel's window: a panel
// dragged out of the host is in one of its own now, and one dragged back
// over it has left it. A button that floats or docks a panel says the
// same thing, and a more particular one, after this has run.
func (p *dockPage) noteDrags() {
	if p.floating == nil {
		p.floating = map[string]bool{}
	}
	for _, pan := range p.host.Panels() {
		now := pan.Floating()
		was := p.floating[pan.Name()]
		p.floating[pan.Name()] = now
		switch {
		case now && !was:
			p.t.note(pan.Title() + " is in a window of its own — drag it back over the host to dock it.")
		case was && !now:
			p.t.note(pan.Title() + " is docked again.")
		}
	}
}

// refresh restates where every panel is, and prints the layout the host
// would save right now.
func (p *dockPage) refresh() {
	if p.state != nil {
		var where []string
		for _, pan := range p.host.Panels() {
			switch {
			case pan.Closed():
				where = append(where, pan.Title()+": closed")
			case pan.Floating():
				where = append(where, pan.Title()+": floating")
			case pan.Collapsed():
				where = append(where, pan.Title()+": collapsed")
			}
		}
		if len(where) == 0 {
			p.state.SetText("Every panel is docked and open.")
		} else {
			p.state.SetText(strings.Join(where, " · "))
		}
	}
	if p.facts == nil {
		return
	}
	b, err := p.host.LayoutJSON()
	if err != nil {
		p.facts.SetText("the layout could not be written: " + err.Error())
		return
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, b, "", "  ") != nil {
		pretty.Reset()
		pretty.Write(b)
	}
	head := foldValue("This is what the host would save now. A panel the layout does not "+
		"name is docked back where it belongs, so a layout written by an older "+
		"version never loses a panel added since.", tourFactCols, "") + "\n\n"
	if len(p.saved) > 0 {
		head = foldValue("Remembered: "+strconv.Itoa(len(p.saved))+" bytes. Below is the layout as it "+
			"stands now; \"Put it back\" reads the remembered one in again.", tourFactCols, "") + "\n\n"
	}
	p.facts.SetText(head + pretty.String())
}
