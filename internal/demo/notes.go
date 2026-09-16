package demo

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Note is one row in the sample notes app.
type Note struct {
	Title    string
	Body     string
	Done     bool
	Priority int
}

// NotesApp is a small real desktop app: a filterable note list + editor.
func NotesApp(win *app.Window) widget.Component {
	notes := []Note{
		{Title: "Ship v0.1.7", Body: "Harden TextArea newline, Switch, Accordion exclusive.", Done: false, Priority: 1},
		{Title: "Damage pass", Body: "Resize should only present dirty rects.", Done: true, Priority: 3},
		{Title: "Theme polish", Body: "Check light theme contrast on sliders.", Done: false, Priority: 2},
		{Title: "X11 present", Body: "XPutImage dirty boxes; fallback offscreen.", Done: true, Priority: 4},
		{Title: "Grocery", Body: "Coffee, oats, lemons, bread.", Done: false, Priority: 5},
	}
	sel := 0
	filter := ""

	title := widgets.NewTextField(notes[0].Title, "Title", nil)
	body := widgets.NewTextArea(notes[0].Body, "Body", nil)
	body.MinRows = 4
	done := widgets.NewCheckbox("Done", notes[0].Done, nil)
	status := widgets.NewLabel(fmt.Sprintf("%d notes", len(notes)))
	priority := widgets.NewNumberField(1, 9, float64(notes[0].Priority), 1, nil)
	priority.Tip = "Lower number is sooner"

	match := func(n Note) bool {
		if filter == "" {
			return true
		}
		return containsFold(n.Title, filter) || containsFold(n.Body, filter)
	}
	visible := func() []int {
		var idx []int
		for i, n := range notes {
			if match(n) {
				idx = append(idx, i)
			}
		}
		return idx
	}

	var table *widgets.TableView
	loadEditor := func() {
		if sel < 0 || sel >= len(notes) {
			return
		}
		title.SetText(notes[sel].Title)
		body.SetText(notes[sel].Body)
		done.SetChecked(notes[sel].Done)
		priority.SetValue(float64(notes[sel].Priority))
	}
	refresh := func() {
		vis := visible()
		table.RowCount = len(vis)
		if sel >= len(notes) {
			sel = len(notes) - 1
		}
		table.Selected = indexOf(vis, sel)
		table.Invalidate()
		status.SetText(fmt.Sprintf("%d notes  ·  %d shown", len(notes), len(vis)))
	}

	table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "Title", Sortable: true},
		{Title: "Pri", Width: 48, Sortable: true, Align: style.AlignCenter},
		{Title: "Done", Width: 56, Sortable: true, Align: style.AlignCenter},
	}, len(notes), func(i, col int) string {
		vis := visible()
		if i < 0 || i >= len(vis) {
			return ""
		}
		n := notes[vis[i]]
		switch col {
		case 1:
			return fmt.Sprintf("%d", n.Priority)
		case 2:
			if n.Done {
				return "yes"
			}
			return "no"
		default:
			return n.Title
		}
	}, func(i int) {
		vis := visible()
		if i < 0 || i >= len(vis) {
			return
		}
		sel = vis[i]
		loadEditor()
	})
	table.Selected = 0
	// A note dragged out of the list goes as its text, so it can be
	// dropped into an editor, a mail body, or the note editor here. The
	// text in the editor drags out on its own: a press inside the
	// selection carries it (widgets/drag.go).
	table.OnDrag = func(rows []int) *widget.Drag {
		vis := visible()
		var parts []string
		for _, i := range rows {
			if i < 0 || i >= len(vis) {
				continue
			}
			n := notes[vis[i]]
			parts = append(parts, n.Title+"\n\n"+n.Body)
		}
		if len(parts) == 0 {
			return nil
		}
		d := widget.DragText(strings.Join(parts, "\n\n---\n\n"))
		if d == nil {
			return nil
		}
		label := fmt.Sprintf("%d notes", len(parts))
		if len(parts) == 1 {
			label = notes[vis[rows[0]]].Title
		}
		d.Image, d.Hotspot = widget.DragLabel(win.Look(), label, win.Scale())
		// The same drag reorders the list when it lands back in it: the
		// rows ride along in the payload, which never goes through a
		// type at all, so a drop here knows these are its own notes and
		// a drop anywhere else still gets the text.
		d.Payload = notesMove(append([]int(nil), rows...))
		d.Actions = platform.DragCopy | platform.DragMove
		d.Done = func(action platform.DragAction) {
			if action == platform.DragNone {
				status.SetText("Drag cancelled")
				return
			}
			status.SetText("Dragged " + label)
		}
		return d
	}
	// Dropping between two rows: notes from this list move there, and
	// text from anywhere else becomes a new note in that place.
	table.DropMimes = []string{"text/plain"}
	table.DropActions = platform.DragCopy | platform.DragMove
	table.OnDropAt = func(at int, e widget.DropEvent) bool {
		vis := visible()
		if at < 0 || at > len(vis) {
			return false
		}
		if mv, ok := e.Payload.(notesMove); ok && e.Source == table {
			moved := moveNotes(&notes, vis, mv, at)
			if moved < 0 {
				return false
			}
			sel = moved
			refresh()
			loadEditor()
			status.SetText(fmt.Sprintf("Moved %d note(s)", len(mv)))
			return true
		}
		text := strings.TrimSpace(e.Text)
		if text == "" {
			return false
		}
		n := Note{Title: firstLine(text), Body: text, Priority: 5}
		where := len(notes)
		if at < len(vis) {
			where = vis[at]
		}
		notes = append(notes, Note{})
		copy(notes[where+1:], notes[where:])
		notes[where] = n
		sel = where
		refresh()
		loadEditor()
		status.SetText("Added " + n.Title)
		return true
	}
	table.OnSort = func(col int, asc bool) {
		vis := visible()
		sort.SliceStable(vis, func(i, j int) bool {
			a, b := notes[vis[i]], notes[vis[j]]
			var less bool
			switch col {
			case 1:
				less = a.Priority < b.Priority
			case 2:
				less = !a.Done && b.Done
			default:
				less = a.Title < b.Title
			}
			if !asc {
				return !less
			}
			return less
		})
		reordered := make([]Note, 0, len(notes))
		seen := map[int]bool{}
		for _, i := range vis {
			reordered = append(reordered, notes[i])
			seen[i] = true
		}
		for i, n := range notes {
			if !seen[i] {
				reordered = append(reordered, n)
			}
		}
		if sel >= 0 && sel < len(notes) {
			cur := notes[sel]
			notes = reordered
			for i, n := range notes {
				if n.Title == cur.Title && n.Body == cur.Body {
					sel = i
					break
				}
			}
		} else {
			notes = reordered
		}
		refresh()
	}
	table.OnContext = func(i int, p paintengine2d.Point) {
		vis := visible()
		if i >= 0 && i < len(vis) {
			sel = vis[i]
			loadEditor()
		}
		widgets.ShowContextMenu(table, p,
			widgets.Item("Toggle done", func() {
				if sel >= 0 && sel < len(notes) {
					notes[sel].Done = !notes[sel].Done
					done.SetChecked(notes[sel].Done)
					refresh()
				}
			}),
			widgets.Item("Delete", func() {
				if len(notes) == 0 {
					return
				}
				notes = append(notes[:sel], notes[sel+1:]...)
				if sel >= len(notes) {
					sel = len(notes) - 1
				}
				if sel >= 0 {
					title.SetText(notes[sel].Title)
					body.SetText(notes[sel].Body)
					done.SetChecked(notes[sel].Done)
				}
				refresh()
			}),
			widgets.Sep(),
			widgets.Item("New note", func() {
				notes = append(notes, Note{Title: "Untitled", Body: "", Priority: 3})
				sel = len(notes) - 1
				loadEditor()
				refresh()
			}),
		)
	}

	title.OnChange = func(s string) {
		if sel >= 0 && sel < len(notes) {
			notes[sel].Title = s
			refresh()
		}
	}
	body.OnChange = func(s string) {
		if sel >= 0 && sel < len(notes) {
			notes[sel].Body = s
		}
	}
	done.OnChange = func(v bool) {
		if sel >= 0 && sel < len(notes) {
			notes[sel].Done = v
			refresh()
		}
	}
	priority.OnChange = func(v float64) {
		if sel >= 0 && sel < len(notes) {
			notes[sel].Priority = int(v)
			refresh()
		}
	}

	search := widgets.NewTextField("", "Filter notes", func(s string) {
		filter = s
		refresh()
	})

	add := widgets.NewButton("New note", func() {
		notes = append(notes, Note{Title: "Untitled", Body: "", Priority: 3})
		sel = len(notes) - 1
		loadEditor()
		refresh()
	})
	add.Primary = true
	add.Tip = "Create an empty note"

	del := widgets.NewButton("Delete", func() {
		if len(notes) == 0 {
			return
		}
		notes = append(notes[:sel], notes[sel+1:]...)
		if sel >= len(notes) {
			sel = len(notes) - 1
		}
		if sel >= 0 {
			loadEditor()
		}
		refresh()
	})
	del.Tip = "Remove the selected note"

	openNote := func() {
		widgets.ShowFileDialog(win.Content(), widgets.FileDialogOptions{
			Title:  "Open note file",
			Path:   ".",
			Filter: "*.md",
			Entries: []widgets.FileInfo{
				{Name: "docs", Dir: true},
				{Name: "inbox.md"},
				{Name: "ship-v017.md"},
				{Name: "theme-polish.md"},
			},
			OnPick: func(p string) {
				base := filepath.Base(p)
				notes = append(notes, Note{Title: base, Body: "Imported from " + p, Priority: 2})
				sel = len(notes) - 1
				loadEditor()
				refresh()
			},
		})
	}

	newItem := widgets.ToolIconBtn(style.IconNew, "New", func() { add.OnClick() })
	newItem.Tip = "New note"
	openItem := widgets.ToolIconBtn(style.IconOpen, "", openNote)
	openItem.Tip = "Open a file (stub picker)"
	cutItem := widgets.ToolIconBtn(style.IconCut, "", func() { del.OnClick() })
	cutItem.Tip = "Delete note"
	tools := widgets.NewToolBar(newItem, openItem, widgets.ToolDivider(), cutItem)

	sidebar := widgets.NewColumn(
		widgets.NewTitle("Notes"),
		tools,
		search,
		table,
		widgets.NewRow(add, del).WithGap(8),
		status,
	).WithGap(8).WithPad(12)
	sidebar.AddFlex(table, 1)

	editor := widgets.NewPanel("Editor",
		widgets.NewLabel("Title").For(title),
		title,
		widgets.NewLabel("Body").For(body),
		body,
		widgets.NewLabel("Priority").For(priority),
		priority,
		done,
		widgets.NewLabel("Table, textarea, spinner, file stub."),
	)
	right := widgets.NewPad(8, editor)
	split := widgets.NewSplitter(true, sidebar, right)
	split.Ratio = 0.38
	_ = win
	_ = layout.AlignStart
	return split
}

func containsFold(s, sub string) bool {
	if sub == "" {
		return true
	}
	S, Sub := []rune(s), []rune(sub)
	for i := 0; i+len(Sub) <= len(S); i++ {
		ok := true
		for j := range Sub {
			if fold(S[i+j]) != fold(Sub[j]) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func fold(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r - 'A' + 'a'
	}
	return r
}

func indexOf(xs []int, v int) int {
	for i, x := range xs {
		if x == v {
			return i
		}
	}
	return -1
}

// notesMove is what a drag of notes carries inside the process: the rows
// it picked up, in the order the list showed them.
type notesMove []int

// moveNotes moves the notes at the given visible rows into the gap before
// visible row at, keeping their order. It reports where the first of them
// ended up, or -1 when there was nothing to move.
//
// The gap belongs to the row below it, so the note that row holds is the
// anchor the block is inserted before; past the last row the anchor is the
// end of the list. Taking the moved notes out cannot shift the anchor,
// because it is found while walking the old list, not by index.
func moveNotes(notes *[]Note, vis []int, rows notesMove, at int) int {
	picked := map[int]bool{}
	var order []int
	for _, r := range rows {
		if r < 0 || r >= len(vis) || picked[vis[r]] {
			continue
		}
		picked[vis[r]] = true
		order = append(order, vis[r])
	}
	if len(order) == 0 {
		return -1
	}
	sort.Ints(order)
	anchor := -1 // -1: the end of the list
	if at < len(vis) {
		anchor = vis[at]
	}
	old := *notes
	out := make([]Note, 0, len(old))
	first := -1
	insert := func() {
		first = len(out)
		for _, i := range order {
			out = append(out, old[i])
		}
	}
	for i := range old {
		if i == anchor {
			insert()
		}
		if !picked[i] {
			out = append(out, old[i])
		}
	}
	if first < 0 {
		insert()
	}
	*notes = out
	return first
}

// firstLine is a title for a note made out of dropped text.
func firstLine(text string) string {
	line := text
	if i := strings.IndexAny(line, "\r\n"); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	if line == "" {
		line = "Note"
	}
	if r := []rune(line); len(r) > 40 {
		line = string(r[:40]) + "…"
	}
	return line
}
