// Command glass is the toolkit's glass, shown where glass can be seen.
//
// Three things have to be true at once before a pane of glass reads as
// glass rather than as paint, and the first one is a layout decision:
//
//  1. **Something has to be behind it.** The list here runs the whole
//     height of the window and scrolls *under* the chrome
//     ([widgets.FloatOver]), instead of starting below it as a column
//     would put it. Chrome with a flat window background behind it has
//     nothing to be translucent of, and the finest glass material in the
//     world draws as a grey rectangle.
//  2. **The material has to be clear.** The tool bar capsules are an
//     eighth of an alpha over the list, not a frosted sheet: a backdrop
//     blurred hard is a flat colour, and a flat colour cannot be bent.
//  3. **The pane has to have thickness.** Its edge displaces what is
//     behind it ([paintengine2d.Context.BackdropRefract]), which is the
//     difference between glass and frosting.
//
// Scroll the list and watch the rows bend as they pass under the
// capsules' rims. Move the pointer across a capsule and the specular rim
// follows it. -glass asks the compositor to blur the desktop behind the
// whole window as well, where it can.
//
//	go run ./examples/uitoolkit-sample-glass
//	go run ./examples/uitoolkit-sample-glass -glass      # desktop blurred behind too
//	go run ./examples/uitoolkit-sample-glass -theme tahoe-night   # Apple's own, for comparison
//	go run ./examples/uitoolkit-sample-glass -flat       # the same window, no glass
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	theme := flag.String("theme", "lumen-night", "the pack to run in")
	behind := flag.Bool("glass", false, "ask the compositor to blur the desktop behind the window")
	flat := flag.Bool("flat", false, "no glass at all, for comparison")
	shell := flag.Bool("shell", false, "the whole window: a glass sidebar beside an opaque document, under one title bar")
	shot := flag.String("shot", "", "paint one frame offscreen, write this PNG and exit")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("glass: ")

	// A demo of an animation and a material has to be allowed to animate:
	// a desktop with its animations switched off would otherwise take the
	// moving specular with it. UITK_ANIMATIONS is the documented way to
	// say so and belongs to whoever starts the program.
	if os.Getenv(style.AnimationsEnv) == "" {
		os.Setenv(style.AnimationsEnv, "1")
	}

	headless := *shot != ""
	opts := uitoolkit.Options{Headless: headless}
	if *theme != "" {
		pack, ok := style.LoadTheme(*theme)
		if !ok {
			log.Fatalf("no theme %q — build with -tags theme_engine_all, or name one style.EngineIDs() lists", *theme)
		}
		opts.Look = pack.Look()
	}
	a := uitoolkit.New(opts)
	wo := platform.WindowOptions{
		Title: windowTitle("Glass"), Width: 760, Height: 560, MinWidth: 520, MinHeight: 380,
	}
	if *shell {
		// The toolkit draws the frame, so the window's corners, its
		// shadow and its caption buttons come from the look and the
		// title bar under them is the application's own.
		wo.Decorations = platform.DecorationsClient
	}
	win, err := a.NewWindow(wo)
	if err != nil {
		log.Fatal(err)
	}
	if *behind && !*flat {
		// The desktop, blurred behind the whole window, where the
		// compositor offers it. The tint has to stay translucent or the
		// blur it asked for shows through nothing.
		win.SetGlass(true)
	}

	lib := newLibrary()
	if *shell {
		shellWindow(win, lib, *behind && !*flat)
		runOrShoot(a, win, headless, *shot, *theme, *flat)
		return
	}
	rows := widgets.NewScrollView(lib.list(*flat))
	bar := lib.toolBar(*flat)
	now := lib.nowPlaying(*flat)

	var content widget.Component = rows
	if *flat {
		// The comparison: chrome above the list rather than over it, which
		// is what every column layout gives and what leaves a translucent
		// material with nothing behind it.
		col := widgets.NewColumn(bar, rows, now).WithGap(0)
		col.AddFlex(rows, 1)
		win.SetContent(col)
	} else {
		over := widgets.NewFloatOver(content)
		over.Inset = 10
		over.SetTop(bar)
		over.SetBottom(now)
		win.SetContent(over)
	}

	runOrShoot(a, win, headless, *shot, *theme, *flat)
}

func runOrShoot(a *uitoolkit.Application, win *uitoolkit.Window, headless bool, shot, theme string, flat bool) {
	if headless {
		if err := win.WritePNG(shot); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", shot)
		return
	}
	log.Printf("theme=%s glass asked=%v available=%v under=%v floating=%v",
		theme, win.Glass(), win.GlassAvailable(), win.GlassUnder() != nil, !flat)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

// shellWindow is the shape a modern application window actually has: one
// title bar with the caption buttons in it, a sidebar of glass down the
// left, and an opaque document beside it.
//
// **The sidebar is glass by being the only part of the window nothing
// paints over.** A window that asks for glass is translucent all
// through and the compositor blurs the desktop behind the whole of it;
// the document then paints itself opaque on top, so the desktop shows
// only where the sidebar is. SetGlassUnder tells the compositor as much,
// so it blurs that strip rather than the whole surface — the rest of
// that work would be thrown away behind the document.
func shellWindow(win *uitoolkit.Window, lib *library, glass bool) {
	// The application's own title bar *is* the caption: the look draws
	// its buttons into it rather than stacking a strip of its own above
	// it, which is what Chromium and VS Code do with theirs.
	title := widgets.NewLabel(windowTitle("Glass"))
	title.Bold = true
	gapL, gapR := widgets.NewSpacer(), widgets.NewSpacer()
	bar := widgets.NewRow(gapL, title, gapR)
	bar.AddFlex(gapL, 1)
	bar.AddFlex(gapR, 1)
	win.SetTitleBar(widgets.NewPad(6, bar))
	win.SetCaptionStyle(uitoolkit.CaptionMerged)

	// The sidebar paints nothing of its own, so what shows there is the
	// window's own translucent background — and behind that, the
	// desktop. It takes its natural width; the document gets the rest.
	side := widgets.NewPad(12, lib.sources())
	doc := lib.document(win.Look())
	row := widgets.NewRow(side, doc).WithGap(0)
	row.AddFlex(doc, 1)
	win.SetContent(row)

	if glass {
		win.SetGlass(true)
		// An explicit tint: the window background has to be translucent
		// for the compositor's blur to show through it at all, and the
		// look's own is only as translucent as that look wanted.
		win.SetGlassTint(paintengine2d.RGBA(0.11, 0.11, 0.13, 0.55))
		win.SetGlassUnder(side)
	}
}

// sources is the sidebar's list: plain text on nothing, so the glass
// behind it is what the eye sees.
func (l *library) sources() widget.Component {
	col := widgets.NewColumn().WithGap(4)
	head := widgets.NewLabel("Library")
	head.Bold = true
	col.Add(head)
	for _, s := range []string{"Recently Added", "Artists", "Albums", "Songs", "Playlists"} {
		col.Add(widgets.NewLabel(s))
	}
	fill := widgets.NewSpacer()
	col.Add(fill)
	col.AddFlex(fill, 1)
	return col
}

// document is the opaque half.
//
// It has to paint itself solid, and a panel is not enough: a panel's
// background is a wash of a few percent over whatever is behind it,
// which over a translucent window is the desktop. Measured, the
// document came back at 161/255 alpha against the sidebar's 140 —
// showing the desktop through a document, which is not a window anybody
// designs. A stretched block of the palette's own background under it
// is what makes the window opaque exactly where it is meant to be.
func (l *library) document(lk style.LookAndFeel) widget.Component {
	solid := widgets.NewPicture(block(1, 1, lk.Palette().Background))
	solid.Fit = widgets.FitStretch
	return widgets.NewStack(solid, widgets.NewScrollView(l.list(true)))
}

// library is the content the glass sits over: rows with enough colour and
// structure in them that a pane bending one is plain to see. A wash of
// one flat colour would hide every effect in this program.
type library struct {
	tracks []track
	status *widgets.Label
}

type track struct {
	title, artist string
	hue           float32
}

var artists = []string{
	"Hollow Coast", "Marta Vey", "The Long Shore", "Ilse Brandt",
	"Pelagic", "Ruth Okonjo", "Saltmarsh", "Night Ferry",
}

var titles = []string{
	"Weightless", "An Ending", "Avril", "Music for Airports",
	"Harbour Lights", "Slow Tide", "Northern Line", "Glasshouse",
	"Pale Sky", "Undertow", "First Frost", "Quiet Hours",
}

func newLibrary() *library {
	l := &library{}
	for i := 0; i < 48; i++ {
		l.tracks = append(l.tracks, track{
			title:  titles[i%len(titles)],
			artist: artists[(i*5)%len(artists)],
			hue:    float32(i*37%360) / 360,
		})
	}
	return l
}

// list is the scrolling content. Each row carries a block of colour, so
// the rows passing under a capsule are a pattern rather than a wash.
func (l *library) list(flat bool) widget.Component {
	col := widgets.NewColumn().WithGap(6)
	for i, t := range l.tracks {
		col.Add(l.row(i, t))
	}
	pad := widgets.NewPad(10, col)
	if !flat {
		// **The content is inset by the chrome, inside the scroll.** The
		// viewport keeps the whole window so rows pass under the bars as
		// they scroll, but the scrolled column begins below the top bar
		// and ends above the bottom one, so at rest nothing is hidden —
		// which is iOS's content inset and macOS's safe area, and what
		// the first version of this sample forgot: the chrome landed on
		// the first and last rows and read as broken rather than as
		// glass.
		pad.T, pad.B = chromeInset, chromeInset
	}
	return pad
}

// chromeInset is how far the content keeps clear of a floating bar: the
// bar's own height plus the gap it floats in, as a 1x design length.
const chromeInset = 54

func (l *library) row(i int, t track) widget.Component {
	art := widgets.NewPicture(block(52, 52, hue(t.hue)))

	title := widgets.NewLabel(fmt.Sprintf("%2d.  %s", i+1, t.title))
	title.Bold = true
	sub := widgets.NewLabel(t.artist)
	text := widgets.NewColumn(title, sub).WithGap(2)

	row := widgets.NewRow(art, text).WithGap(10)
	row.AddFlex(text, 1)
	return widgets.NewHeightBox(52, row)
}

// block is a square of flat colour, which a Picture shows at its natural
// size. The art a real library would load is a picture too; this keeps
// the sample to one file and gives the glass something with edges in it
// to bend.
func block(w, h int, c paintengine2d.Color) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(w), float32(h)), paintengine2d.Fill(c))
	img.Touch()
	return img
}

// toolBar is the chrome the list runs under. The divider splits it into
// two runs, and the look draws a glass capsule under each
// (style.ToolGroupsOf) — those capsules are the panes that refract.
func (l *library) toolBar(flat bool) widget.Component {
	bar := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconArrowLeft, "Back", nil),
		widgets.ToolIconBtn(style.IconArrowRight, "Forward", nil),
		widgets.ToolDivider(),
		widgets.ToolIconBtn(style.IconStar, "Favourite", nil),
		widgets.ToolIconBtn(style.IconTag, "Tag", nil),
		widgets.ToolIconBtn(style.IconMore, "More", nil),
		widgets.ToolStretch(),
		widgets.ToolDivider(),
		widgets.ToolIconBtn(style.IconSearch, "Search", nil),
	)
	// Over content, the bar paints no strip of its own: a strip is
	// opaque, and an opaque strip behind a translucent capsule leaves
	// the material with nothing to be translucent of.
	bar.Floating = !flat
	return bar
}

// nowPlaying is a second pane of the same glass against the other edge,
// so there is one to watch at each end of a scroll.
func (l *library) nowPlaying(flat bool) widget.Component {
	l.status = widgets.NewLabel("Weightless — Hollow Coast")
	bar := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconArrowLeft, "Previous", nil),
		widgets.ToolIconBtn(style.IconCheck, "Play", nil),
		widgets.ToolIconBtn(style.IconArrowRight, "Next", nil),
		widgets.ToolDivider(),
		widgets.ToolWidget(l.status),
	)
	bar.Floating = !flat
	return bar
}

// hue is a saturated colour round the wheel, for the row blocks.
func hue(h float32) paintengine2d.Color {
	r, g, b := hsv(h, 0.55, 0.92)
	return paintengine2d.RGB(r, g, b)
}

func hsv(h, s, v float32) (r, g, b float32) {
	i := int(h * 6)
	f := h*6 - float32(i)
	p, q, t := v*(1-s), v*(1-f*s), v*(1-(1-f)*s)
	switch i % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	}
	return v, p, q
}

// windowTitle is what a window of this sample is called on the desktop:
// the sample's own name for the window under the toolkit's prefix, so
// that a desktop with several samples open says which toolkit they
// belong to. Every title this sample sets goes through here, so one
// computed while the app runs carries the prefix too.
func windowTitle(name string) string { return "uitoolkit - " + name }
