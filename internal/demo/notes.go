package demo

import (
	"fmt"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Note is one row in the sample notes app.
type Note struct {
	Title string
	Body  string
	Done  bool
}

// NotesApp is a small real desktop app: a filterable note list + editor.
func NotesApp(win *app.Window) widget.Component {
	notes := []Note{
		{Title: "Ship v0.1", Body: "Write README, screenshots, and tag the module.", Done: false},
		{Title: "Damage pass", Body: "Resize should only present dirty rects.", Done: true},
		{Title: "Theme polish", Body: "Check light theme contrast on sliders.", Done: false},
		{Title: "X11 present", Body: "XPutImage dirty boxes; fallback offscreen.", Done: true},
		{Title: "Grocery", Body: "Coffee, oats, lemons, bread.", Done: false},
	}
	sel := 0
	filter := ""

	title := widgets.NewTextField(notes[0].Title, "Title", nil)
	body := widgets.NewTextField(notes[0].Body, "Body", nil)
	done := widgets.NewCheckbox("Done", notes[0].Done, nil)
	status := widgets.NewLabel(fmt.Sprintf("%d notes", len(notes)))

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

	var list *widgets.ListView
	refresh := func() {
		vis := visible()
		list.Count = len(vis)
		if sel >= len(notes) {
			sel = len(notes) - 1
		}
		list.Selected = indexOf(vis, sel)
		list.Invalidate()
		status.SetText(fmt.Sprintf("%d notes  ·  %d shown", len(notes), len(vis)))
	}

	list = widgets.NewListView(len(notes), func(i int) string {
		vis := visible()
		if i < 0 || i >= len(vis) {
			return ""
		}
		n := notes[vis[i]]
		mark := "·"
		if n.Done {
			mark = "+"
		}
		return mark + "  " + n.Title
	}, func(i int) {
		vis := visible()
		if i < 0 || i >= len(vis) {
			return
		}
		sel = vis[i]
		title.SetText(notes[sel].Title)
		body.SetText(notes[sel].Body)
		done.SetChecked(notes[sel].Done)
	})
	list.Selected = 0
	list.OnContext = func(i int, p paintengine2d.Point) {
		vis := visible()
		if i >= 0 && i < len(vis) {
			sel = vis[i]
			title.SetText(notes[sel].Title)
			body.SetText(notes[sel].Body)
			done.SetChecked(notes[sel].Done)
		}
		widgets.ShowContextMenu(list, p,
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
				notes = append(notes, Note{Title: "Untitled", Body: ""})
				sel = len(notes) - 1
				title.SetText("Untitled")
				body.SetText("")
				done.SetChecked(false)
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

	search := widgets.NewTextField("", "Filter notes", func(s string) {
		filter = s
		refresh()
	})

	add := widgets.NewButton("New note", func() {
		notes = append(notes, Note{Title: "Untitled", Body: ""})
		sel = len(notes) - 1
		title.SetText("Untitled")
		body.SetText("")
		done.SetChecked(false)
		refresh()
	})
	add.Primary = true

	del := widgets.NewButton("Delete", func() {
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
	})

	tools := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconNew, "New", func() { add.OnClick() }),
		widgets.ToolIconBtn(style.IconOpen, "", func() {}),
		widgets.ToolDivider(),
		widgets.ToolIconBtn(style.IconCut, "", func() { del.OnClick() }),
	)

	sidebar := widgets.NewColumn(
		widgets.NewTitle("Notes"),
		tools,
		search,
		list,
		widgets.NewRow(add, del).WithGap(8),
		status,
	).WithGap(8).WithPad(12)
	sidebar.AddFlex(list, 1)

	editor := widgets.NewPanel("Editor",
		widgets.NewLabel("Title"),
		title,
		widgets.NewLabel("Body"),
		body,
		done,
		widgets.NewLabel("A small desktop app on uitoolkit — list, filter, editor, actions."),
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
