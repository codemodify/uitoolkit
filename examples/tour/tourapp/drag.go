package demo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Page three: a drag is a conversation between two applications, and the
// same Drag has to hold both halves of it — the reorder that never
// leaves the list, and the drop into a file manager that leaves the
// process entirely. The two lists here reorder, move rows between
// themselves and drag real files out; the drop zone takes whatever the
// desktop hands it; and the readout says what the two sides settled on,
// which is the part of a drag nobody can see.

func init() {
	p := &tourPages[pageDrag]
	p.title = "Drags that leave the application"
	p.proof = "One Drag serves a reorder inside a list, a move between two lists and a drop into a " +
		"file manager: the caret says between which rows, and the modifiers say copy, move or link."
	p.try = "Drag a row between the lists, or out to a file manager."
	p.build = buildDragPage
}

// tourDoc is a row of the two lists: a little document that is also a
// real file the moment something outside the process asks for one.
type tourDoc struct {
	name string
	body string
	// path is the file written for a drag that leaves the process; empty
	// until one does.
	path string
}

// tourDocMove rides along inside the process, so a drop on our own lists
// gets the rows themselves rather than re-reading them out of a file.
type tourDocMove struct {
	from *docList
	docs []*tourDoc
}

type docList struct {
	page *dragPage
	name string
	rows []*tourDoc
	view *widgets.ListView
}

type dragPage struct {
	t     *tourState
	left  *docList
	right *docList
	facts *widgets.TextArea
	// dir holds the files a drag out of the process needs; it is made on
	// demand and removed with the window.
	dir string
	// last is what the last drag offered and what the last drop did.
	lastDrag string
	lastDrop string
}

func buildDragPage(t *tourState) widget.Component {
	p := &dragPage{t: t}
	t.own(pageDrag, p)
	p.left = p.newList("Drafts", []*tourDoc{
		{name: "Shipping forecast", body: "Dogger, Fisher, German Bight."},
		{name: "Release notes", body: "The tab strip is the caption now."},
		{name: "Postcard", body: "Wish you were here."},
	})
	p.right = p.newList("Filed", []*tourDoc{
		{name: "Receipts", body: "One toolkit, paid in full."},
	})

	leftPane, rightPane := p.left.panel(), p.right.panel()
	lists := widgets.NewRow(leftPane, rightPane).WithGap(10)
	lists.AddFlex(leftPane, 1)
	lists.AddFlex(rightPane, 1)

	// Anything from anywhere: the zone takes files and text, and says
	// which it got.
	zoneText := widgets.NewLabel("Drop files or text here — from a file manager, a browser, a terminal.")
	zoneText.Wrap = true
	zoneText.Align = style.AlignCenter
	zoneText.MinLines = 2
	zone := widgets.NewDropZone(widgets.NewPad(14, zoneText), nil)
	zone.OnFiles = func(paths []string) {
		p.lastDrop = "files from another application:\n  " + strings.Join(paths, "\n  ")
		zoneText.SetText(plural(len(paths), "file") + " dropped: " + filepath.Base(paths[0]))
		p.note("Took " + plural(len(paths), "file") + " from another application.")
	}
	zone.OnText = func(text string) {
		one := strings.Join(strings.Fields(text), " ")
		if len(one) > 90 {
			one = one[:90] + "…"
		}
		p.lastDrop = "text from another application:\n  " + one
		zoneText.SetText("Text dropped: " + one)
		p.note("Took text from another application.")
	}
	zone.SetAccessibleName("Drop zone")

	// The text widgets are drag sources for their own selection, which is
	// the other half of "drags out of the app" and needs no code at all.
	out := widgets.NewTextArea("Select this sentence and drag it into a text editor — a TextArea is a "+
		"drag source for its selection, and a drop target for text, with nothing wired up.", "", nil)
	out.SetAccessibleName("Draggable text")

	howto := widgets.NewPanel("What each gesture does",
		tourNote("Press a row and pull: past the drag threshold the list hands the window a Drag, and a "+
			"picture of what you are carrying follows the pointer. Over either list an insertion caret "+
			"opens in the gap the row would land in — the top and bottom quarters of a row are its gaps."),
		tourNote("Let go over the other list to file it. Hold Shift as you drop to move, Ctrl to copy, "+
			"both to link; with nothing held the source's own preference stands. On X11 the toolkit reads "+
			"those keys itself, on Wayland the compositor does and tells us what it decided."),
		tourNote("Drag a row onto a file manager and a real file leaves the process: the row is offered "+
			"as a uri-list first and as plain text after it, so a file manager takes the file and a "+
			"terminal takes the path."),
	)
	howto.Content().Spec.Gap = 7

	// The lists get height of their own rather than their natural one:
	// a list measures to its rows and its frame, and this page needs room
	// under the last row for the caret that says "at the end".
	rest := tourScroll("Drag and drop page", widgets.NewColumn(
		widgets.NewPanel("Anything from another application", zone),
		widgets.NewPanel("Text drags itself", out),
		howto,
	).WithGap(10))
	stage := widgets.NewSplitter(widgets.SplitRows, lists, rest)
	stage.Ratio = 0.36

	panel, facts := tourReadout("What the two sides settled on")
	p.facts = facts
	p.refresh()

	t.onClose(func() {
		if p.dir != "" {
			os.RemoveAll(p.dir)
		}
	})
	return tourStage(stage, panel)
}

func (p *dragPage) newList(name string, rows []*tourDoc) *docList {
	l := &docList{page: p, name: name, rows: rows}
	l.view = widgets.NewListView(len(rows), func(i int) string {
		if i < 0 || i >= len(l.rows) {
			return ""
		}
		return l.rows[i].name
	}, nil)
	l.view.SetAccessibleName(name)
	l.view.Mode = widgets.SelectExtended
	l.view.Selected = 0

	// Out of the list.
	l.view.OnDrag = func(rows []int) *widget.Drag { return l.drag(rows) }
	// Into it, between two rows: setting OnDropAt alone is what puts the
	// caret in every gap, with no dead band on the rows themselves.
	l.view.DropMimes = []string{"text/uri-list", "text/plain"}
	l.view.DropActions = platform.DragCopy | platform.DragMove | platform.DragLink
	l.view.OnDropAt = func(at int, e widget.DropEvent) bool { return l.dropAt(at, e) }
	return l
}

func (l *docList) sync() {
	l.view.Count = len(l.rows)
	if l.view.Selected >= len(l.rows) {
		l.view.Selected = len(l.rows) - 1
	}
	l.view.Invalidate()
	l.view.RequestLayout()
}

// drag is what a press on the selected rows carries out of the list: the
// rows themselves for a drop on our own lists, and real files for
// everything else.
func (l *docList) drag(rows []int) *widget.Drag {
	p := l.page
	var docs []*tourDoc
	var paths []string
	for _, i := range rows {
		if i < 0 || i >= len(l.rows) {
			continue
		}
		d := l.rows[i]
		docs = append(docs, d)
		if path, err := p.fileFor(d); err == nil {
			paths = append(paths, path)
		}
	}
	if len(docs) == 0 {
		return nil
	}
	label := docs[0].name
	if len(docs) > 1 {
		label = plural(len(docs), "document")
	}
	d := widget.DragFiles(paths...)
	if d == nil {
		// No file could be written, so the rows can still be moved inside
		// the process and offered as text outside it.
		d = widget.DragText(docs[0].body)
		if d == nil {
			return nil
		}
	}
	d.Payload = tourDocMove{from: l, docs: docs}
	d.Actions = platform.DragCopy | platform.DragMove | platform.DragLink
	// Filing a document is a move; the modifiers override it either way,
	// and a file manager's own convention wins where it has one.
	d.Preferred = platform.DragMove
	d.Image, d.Hotspot = widget.DragLabel(l.view.Look(), label, tourScale(p.t.win))
	p.lastDrag = tourFacts(
		[2]string{"dragging", label + " out of " + l.name},
		[2]string{"offers", strings.Join(d.Types, "\n")},
		[2]string{"allows", actionList(d.Actions)},
		[2]string{"prefers", d.Preferred.String()},
		[2]string{"picture", label},
	)
	d.Done = func(action platform.DragAction) {
		if action == platform.DragNone {
			p.lastDrop = "nothing took it — a cancelled drag and a refused one look the same"
			p.note("Drag cancelled.")
			p.refresh()
			return
		}
		if action == platform.DragMove {
			// The target says it took them, so the originals go.
			l.remove(docs)
		}
		p.note("The target performed a " + action.String() + ".")
		p.refresh()
	}
	p.refresh()
	return d
}

// dropAt takes a drop in the gap before row `at`.
func (l *docList) dropAt(at int, e widget.DropEvent) bool {
	p := l.page
	if at < 0 || at > len(l.rows) {
		at = len(l.rows)
	}
	var added []*tourDoc

	if mv, ok := e.Payload.(tourDocMove); ok {
		// One of ours. A move inside the same list is a reorder: take the
		// rows out first, so the index means what the caret showed.
		if mv.from == l && e.Action == platform.DragMove {
			before := 0
			for _, d := range mv.docs {
				if i := indexOfDoc(l.rows, d); i >= 0 && i < at {
					before++
				}
			}
			l.remove(mv.docs)
			at -= before
			if at < 0 {
				at = 0
			}
			added = mv.docs
		} else {
			for _, d := range mv.docs {
				switch e.Action {
				case platform.DragLink:
					added = append(added, &tourDoc{name: "→ " + d.name, body: d.body})
				case platform.DragMove:
					added = append(added, d)
				default:
					added = append(added, &tourDoc{name: d.name + " (copy)", body: d.body})
				}
			}
		}
		p.lastDrop = tourFacts(
			[2]string{"dropped on", l.name + ", gap " + strconv.Itoa(at)},
			[2]string{"read as", "the rows themselves (in-process payload)"},
			[2]string{"from", mv.from.name},
			[2]string{"action", e.Action.String()},
		)
	} else {
		// From somewhere else: files, or text.
		switch {
		case len(e.Paths) > 0:
			for _, path := range e.Paths {
				added = append(added, &tourDoc{name: filepath.Base(path), body: path, path: path})
			}
		case e.Text != "":
			added = append(added, &tourDoc{name: firstLine(e.Text), body: e.Text})
		default:
			return false
		}
		p.lastDrop = tourFacts(
			[2]string{"dropped on", l.name + ", gap " + strconv.Itoa(at)},
			[2]string{"read as", e.Mime},
			[2]string{"from", "another application"},
			[2]string{"action", e.Action.String()},
		)
	}
	if len(added) == 0 {
		return false
	}
	rest := append([]*tourDoc(nil), l.rows[at:]...)
	l.rows = append(append(l.rows[:at:at], added...), rest...)
	l.sync()
	p.note(plural(len(added), "row") + " " + e.Action.String() + "d into " + l.name + ".")
	p.refresh()
	return true
}

func (l *docList) remove(docs []*tourDoc) {
	keep := l.rows[:0]
	for _, r := range l.rows {
		if indexOfDoc(docs, r) < 0 {
			keep = append(keep, r)
		}
	}
	l.rows = keep
	l.sync()
}

func (l *docList) panel() widget.Component {
	p := widgets.NewPanel(l.name, l.view)
	p.Content().AddFlex(l.view, 1)
	return p
}

// fileFor writes a document out so that something outside the process can
// take it, once, into a directory that goes with the window.
func (p *dragPage) fileFor(d *tourDoc) (string, error) {
	if d.path != "" {
		return d.path, nil
	}
	if p.dir == "" {
		dir, err := os.MkdirTemp("", "uitk-tour-")
		if err != nil {
			return "", err
		}
		p.dir = dir
	}
	path := filepath.Join(p.dir, safeFileName(d.name)+".txt")
	if err := os.WriteFile(path, []byte(d.body+"\n"), 0o600); err != nil {
		return "", err
	}
	d.path = path
	return path, nil
}

func (p *dragPage) note(s string) { p.t.note(s) }

func (p *dragPage) refresh() {
	if p.facts == nil {
		return
	}
	drag := p.lastDrag
	if drag == "" {
		drag = "nothing has been dragged yet\n"
	}
	drop := p.lastDrop
	if drop == "" {
		drop = "nothing has been dropped yet\n"
	}
	where := "the files a drag out of the process offers: none written yet"
	if p.dir != "" {
		where = "files written for drags out of the process:\n  " + p.dir
	}
	p.facts.SetText("THE LAST DRAG\n" + drag +
		"\nTHE LAST DROP\n" + drop +
		"\n" + where + "\n")
}

// ---- small helpers -------------------------------------------------------------

func indexOfDoc(rows []*tourDoc, d *tourDoc) int {
	for i, r := range rows {
		if r == d {
			return i
		}
	}
	return -1
}

// safeFileName keeps a document's name usable as one.
func safeFileName(s string) string {
	repl := func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		}
		return '-'
	}
	out := strings.Map(repl, s)
	if out == "" {
		return "document"
	}
	return out
}

func plural(n int, thing string) string {
	if n == 1 {
		return "1 " + thing
	}
	return strconv.Itoa(n) + " " + thing + "s"
}

// actionList names every action a set carries, which String does not: it
// answers with the one that would be performed.
func actionList(a platform.DragAction) string {
	var out []string
	for _, one := range []platform.DragAction{platform.DragCopy, platform.DragMove, platform.DragLink} {
		if a.Has(one) {
			out = append(out, one.String())
		}
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ", ")
}

// tourScale is the window's scale, for the pictures a page draws itself
// (a drag's icon has to be as sharp as the window it came out of).
func tourScale(win *app.Window) float32 {
	if win == nil {
		return 1
	}
	if s := win.Scale(); s > 0 {
		return s
	}
	return 1
}
