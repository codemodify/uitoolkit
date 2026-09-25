package tourapp

import (
	"strconv"
	"strings"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The strip at the top of this window is not a tab bar under the
// caption, it is the caption. Every gesture the page describes is
// made on the strip itself, and every one of them has a button here too
// — partly so the page can be driven from the keyboard, and partly
// because "drag a tab out of the window" is worth being able to try
// without committing a drag.

func init() {
	p := &tourPages[pageTabs]
	p.title = "Tabs in the title bar"
	p.proof = "The strip above is the window's caption, not a widget under one: the tabs are controls, " +
		"the space around them still drags the window, and a tab pulled clear becomes a window of its own."
	p.try = "Drag a tab along the strip, or pull one down out of it."
	p.build = buildTabsPage
}

type tabsPage struct {
	t     *tourState
	facts *widgets.TextArea
	// where says, above the buttons, whether the strip is caption or the
	// window's first row — which is the frame the desktop gave us, and
	// changes under the user on the Frames page.
	where *widgets.Label
}

func buildTabsPage(t *tourState) widget.Component {
	p := &tabsPage{t: t}
	t.own(pageTabs, p)

	move := func(d int) {
		i := t.strip.Selected()
		to := i + d
		if i < 0 || to < 0 || to >= t.strip.Len() {
			t.note("This page is already at the end of the strip.")
			return
		}
		t.strip.MoveTab(i, to)
		t.note("Moved " + tourPages[t.pageAt(to)].name + " to position " + strconv.Itoa(to+1) + ".")
		p.refresh()
	}

	left := widgets.NewButton("Move left", func() { move(-1) })
	left.Tip = "The same as dragging this tab one place down the strip"
	right := widgets.NewButton("Move right", func() { move(1) })
	right.Tip = "The same as dragging this tab one place up the strip"
	out := widgets.NewButton("Move to a new window", func() {
		t.tearOff(t.strip.Selected())
		p.refresh()
	})
	out.Tip = "The same as pulling this tab clear of the strip"
	out.Primary = true
	add := widgets.NewButton("Open a page…", func() { t.openMenu() })
	add.Tip = "The same as the + at the end of the strip"
	shut := widgets.NewButton("Close this page", func() {
		t.strip.CloseTab(t.strip.Selected())
		p.refresh()
	})

	p.where = tourNote("")
	p.facts = nil

	panel, facts := tourReadout("What the strip is doing")
	p.facts = facts

	gestures := widgets.NewPanel("The gestures, on the strip itself",
		tourNote("Press a tab to select it; drag it along the strip to reorder it (the tabs move under "+
			"the pointer as you go). The × closes a tab and a middle click does the same. The + at the end "+
			"asks this app for a new one. Right-click a tab for its menu, and right-click the empty strip "+
			"for the window's."),
		tourNote("Pull a tab down out of the strip and it leaves the window: a second tour window opens "+
			"under the pointer running that page. Drop it on this window's strip — or on the other "+
			"window's — and it joins that one at the caret. Drop it on the desktop and the new window "+
			"stays. Escape puts it back."),
		widgets.NewSeparator(),
		p.where,
	)
	gestures.Content().Spec.Gap = 7

	keyboard := widgets.NewPanel("The same things, from here",
		widgets.NewRow(left, right, shut).WithGap(8),
		widgets.NewRow(out, add).WithGap(8),
		tourNote("Ctrl+Tab and Ctrl+Shift+Tab walk the strip from anywhere in the window, Ctrl+W closes "+
			"the page and Ctrl+T opens one — the strip is handed those keys by the window's content root, "+
			"because a tab strip almost never holds the focus itself."),
	)
	keyboard.Content().Spec.Gap = 8

	stage := widgets.NewColumn(gestures, keyboard, widgets.NewSpacer()).WithGap(10)
	stage.AddFlex(widgets.NewSpacer(), 1)

	p.refresh()
	return tourStage(tourScroll("Tabs page", stage), panel)
}

// refresh restates what the strip is, which the user changes from the
// Frames page and from the strip itself.
func (p *tabsPage) refresh() {
	t := p.t
	if p.where != nil {
		switch {
		case t.win.Decorations() == platform.DecorationsClient && stripStacked(t):
			p.where.SetText("Right now the toolkit draws this window's frame, and this pack stacks it: " +
				"the era it comes from had a caption strip of its own, so the tabs sit in a row under it " +
				"rather than in it — Windows 95, Windows XP, classic Mac OS and Motif all did that. Pick " +
				"a pack whose frame is merged, on the Skins page, and the same tabs move up into the " +
				"caption beside the caption buttons: GTK, Windows 10, macOS and the web-era packs.")
		case t.win.Decorations() == platform.DecorationsClient:
			p.where.SetText("Right now the toolkit draws this window's frame and this pack merges it, so " +
				"the strip above is the caption itself: the caption buttons sit beside the tabs and the " +
				"empty part of the strip moves the window. Switch the frame on the Frames page and the " +
				"same strip becomes the window's first row under the desktop's title bar, which is what " +
				"Chromium does.")
		default:
			p.where.SetText("Right now the desktop draws this window's frame, so the strip above is the " +
				"window's first row under the desktop's title bar — the same widget, laid out as an " +
				"ordinary row, which is what Chromium falls back to. Switch the frame on the Frames page " +
				"to put the tabs in the caption itself.")
		}
	}
	if p.facts == nil {
		return
	}
	var open []string
	for i := 0; i < t.strip.Len(); i++ {
		name := tourPages[t.pageAt(i)].name
		if i == t.strip.Selected() {
			name = "[" + name + "]"
		}
		open = append(open, name)
	}
	carried := widget.DragsWindows(t.strip)
	when := "at the drop (the window is made where it falls)"
	if carried {
		when = "at the press (the desktop carries the window under the pointer)"
	}
	p.facts.SetText(tourFacts(
		[2]string{"open here", strings.Join(open, "  ")},
		[2]string{"selected", strconv.Itoa(t.strip.Selected()+1) + " of " + strconv.Itoa(t.strip.Len())},
		[2]string{"decorations", t.win.Decorations().String()},
		[2]string{"strip is", captionOrRow(t)},
		[2]string{"caption", captionStyle(t)},
		[2]string{"tab type", widgets.TabMimeType},
		[2]string{"a tab moves", platform.DragMove.String()},
		[2]string{"toplevel drag", yesNo(carried)},
		[2]string{"a tab leaves", when},
		[2]string{"backend", t.a.BackendName()},
	))
}

func captionOrRow(t *tourState) string {
	switch {
	case t.win.Decorations() != platform.DecorationsClient:
		return "the window's first row"
	case stripStacked(t):
		return "a row under the caption"
	}
	return "the window's caption"
}

// stripStacked reports that the look keeps a caption strip of its own
// above the header bar (Windows 95, XP, classic Mac, Motif) rather than
// merging the two into one (GTK, Windows 10, macOS, the web-era packs).
func stripStacked(t *tourState) bool {
	hb := t.win.Caption()
	return hb != nil && hb.Stacked()
}

func captionStyle(t *tourState) string {
	if t.win.Decorations() != platform.DecorationsClient {
		return "the desktop's"
	}
	if stripStacked(t) {
		return "stacked — an era caption above the strip"
	}
	return "merged — the strip is the caption"
}
