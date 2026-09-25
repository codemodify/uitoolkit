// Command popups is the popup demo: a window far too small for its own
// menus, whose menus, submenus, combo list, context menu and tooltips open
// anyway — as surfaces of their own, an xdg_popup on Wayland and an
// override-redirect window on X11, placed and flipped against the screen's
// edges rather than the window's.
//
//	popups                       # a 360x150 window: open File, Recent ▸, the list
//	popups -scale 1.75           # at a fractional scale
//	popups -theme macos-tahoe    # a glass look: real blur behind every menu
//	UITK_POPUPS=layer popups     # the same menus drawn inside the window, for comparison
//	popups -shot out.png         # paint one frame offscreen (menus inside), write, exit
//
// "Round" opens a menu that is a disc rather than a rectangle: its surface,
// its shadow and the region that takes the pointer are all cut to the
// silhouette its component states (widget.Base.SetHitShapeFunc).
//
// A right-click anywhere on the window's face opens a context menu at the
// pointer. The last line says which of the two a popup gets here.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/icons"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	theme := flag.String("theme", "", "the pack to run in (default: the desktop's)")
	scale := flag.Float64("scale", 0, "display scale (0: the desktop's, or $UITK_SCALE)")
	shot := flag.String("shot", "", "paint one frame offscreen, write this PNG and exit")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("popups: ")

	headless := *shot != ""
	opts := uitoolkit.Options{Headless: headless, Scale: float32(*scale)}
	if *theme != "" {
		pack, ok := style.LoadTheme(*theme)
		if !ok {
			log.Fatalf("no theme %q", *theme)
		}
		opts.Look = pack.Look()
	}
	a := uitoolkit.New(opts)
	// The window icon the desktop shows in its title bar, task bar and switcher.
	a.SetIcon(icons.AppIconRGB("more", 0x5a, 0x6b, 0x7d)...)
	win, err := a.NewWindow(platform.WindowOptions{
		Title: windowTitle("Popups"), Width: 360, Height: 150, Headless: headless,
	})
	if err != nil {
		log.Fatal(err)
	}
	d := newDemo(win)
	win.SetContent(d.root)

	if headless {
		if err := win.WritePNG(*shot); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", *shot)
		return
	}
	a.Post(func() {
		log.Printf("backend=%s scale=%.2f popups=%s", a.BackendName(), win.Scale(), d.mode())
		d.status.SetText(d.mode())
	})
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

type demo struct {
	win    *app.Window
	root   widget.Component
	status *widgets.Label
}

func newDemo(win *app.Window) *demo {
	d := &demo{win: win, status: widgets.NewLabel("")}
	say := func(s string) func() { return func() { d.status.SetText(s + " · " + d.mode()) } }
	var more []*widgets.MenuItem
	for i := 1; i <= 12; i++ {
		s := fmt.Sprintf("Command %d", i)
		more = append(more, widgets.Item(s, say(s)))
	}
	file := append([]*widgets.MenuItem{
		widgets.ItemAccel("&New", "Ctrl+N", say("New")),
		widgets.ItemAccel("&Open…", "Ctrl+O", say("Open")),
		widgets.Submenu("&Recent",
			widgets.Item("notes.txt", say("notes.txt")),
			widgets.Item("draft.md", say("draft.md")),
			widgets.Submenu("&Older",
				widgets.Item("2025.txt", say("2025.txt")),
				widgets.Item("2024.txt", say("2024.txt")),
			),
		),
		widgets.Sep(),
	}, more...)
	file = append(file, widgets.Sep(), widgets.ItemAccel("&Quit", "Ctrl+Q", func() { win.App().Quit() }))
	bar := widgets.NewMenuBar(
		widgets.NewMenu("&File", file...),
		widgets.NewMenu("&Edit",
			widgets.ItemAccel("Cu&t", "Ctrl+X", say("Cut")),
			widgets.ItemAccel("&Copy", "Ctrl+C", say("Copy")),
			widgets.ItemAccel("&Paste", "Ctrl+V", say("Paste")),
		),
	)
	var names []string
	for i := 1; i <= 40; i++ {
		names = append(names, fmt.Sprintf("Option %02d", i))
	}
	combo := widgets.NewComboBox(names, 0, func(i int) { say(names[i])() })
	tip := widgets.NewButton("Tip", nil)
	tip.Tip = "A tooltip wider than the window it belongs to, which it may now run past"
	round := widgets.NewButton("Round", nil)
	round.OnClick = func() { d.roundMenu(round) }
	row := widgets.NewRow(combo, tip, round)
	face := &pad{onRight: func(p paintengine2d.Point) {
		widgets.ShowContextMenu(d.root, p,
			widgets.Item("Context one", say("Context one")),
			widgets.Item("Context two", say("Context two")),
			widgets.Submenu("More", widgets.Item("Deeper", say("Deeper"))),
		)
	}}
	face.Init(face)
	d.root = widgets.NewColumn(bar, row, d.status, face)
	return d
}

// mode says where this window's popups go.
func (d *demo) mode() string {
	if d.win.PopupSurfaces() {
		return "popups on surfaces of their own"
	}
	return "popups inside the window"
}

// roundMenu opens a menu cut to a disc just under from.
func (d *demo) roundMenu(from widget.Component) {
	pop := widgets.NewPopupMenu(
		widgets.Item("", nil),
		widgets.Item("  North", func() { d.status.SetText("North · " + d.mode()) }),
		widgets.Item("  West · East", func() { d.status.SetText("West · East · " + d.mode()) }),
		widgets.Item("  South", func() { d.status.SetText("South · " + d.mode()) }),
		widgets.Item("", nil),
	)
	pop.SetHitShapeFunc(func(sz paintengine2d.Point) *platform.Shape {
		return platform.ShapeEllipse(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	})
	widget.PreparePopup(from, pop)
	b := widget.DeviceBounds(from)
	widget.PlacePopupForAnchor(from, pop, b, b.Dx()*2.2, 2)
	// A disc as tall as it is wide.
	pb := pop.Bounds()
	side := max(pb.Dx(), pb.Dy())
	pop.Arrange(paintengine2d.XYWH(pb.Min.X, pb.Min.Y, side, side))
	pop.RestoreFocusTo(from)
	if widget.ShowPopup(from, pop) {
		pop.RequestFocus()
	}
}

// pad is the window's empty face: a right-click opens a context menu.
type pad struct {
	widget.Base
	onRight func(paintengine2d.Point)
}

func (p *pad) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(0, 24))
}

func (p *pad) MousePress(e widget.MouseEvent) bool {
	if e.Button != platform.ButtonRight {
		return false
	}
	o := widget.DeviceOrigin(p)
	p.onRight(paintengine2d.Pt(o.X+e.Pos.X, o.Y+e.Pos.Y))
	return true
}

// windowTitle is what a window of this sample is called on the desktop:
// the sample's own name for the window under the toolkit's prefix, so
// that a desktop with several samples open says which toolkit they
// belong to. Every title this sample sets goes through here, so one
// computed while the app runs carries the prefix too.
func windowTitle(name string) string { return "uitoolkit - " + name }
