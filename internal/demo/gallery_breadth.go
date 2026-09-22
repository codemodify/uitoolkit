package demo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The gallery's document views: the rich-text editor, windows inside a
// window and a wizard. Each is what an application would build, small.

// RichTextSample is the document the gallery's editor opens with: a bit
// of everything the editor does.
const RichTextSample = `<h1>Release notes</h1>
<p>The editor keeps <b>bold</b>, <i>italic</i>, <u>underlined</u> and <s>struck</s> text,
<code>monospace</code>, <span style="font-size:18px">larger</span> and
<span style="color:#c0392b">coloured</span> words, a <span style="background-color:#ffe066">highlight</span>
and <a href="https://example.com/notes">a link</a> (Ctrl+click follows it).</p>
<h2>Lists</h2>
<ul><li>Bullets<ul><li>that nest (Ctrl+])</li><li>and come back (Ctrl+[)</li></ul></li><li>Return on an empty item ends the list</li></ul>
<ol><li>Numbered items</li><li>count themselves<ol><li>with letters under them</li></ol></li></ol>
<p style="text-align:center">Select, drag and drop; copy keeps the formatting.</p>`

// richTextView is the format bar over an editor holding the sample.
func richTextView(status *widgets.StatusBar) widget.Component {
	ed := widgets.NewRichTextHTML(RichTextSample)
	ed.SetAccessibleName("Document")
	ed.MinRows = 10
	ed.OnLink = func(href string) { status.Set(0, "Link: "+href) }
	ed.OnSelectionChange = func() {
		a, c := ed.Selection()
		if a == c {
			status.Set(1, fmt.Sprintf("Offset %d", c))
		} else {
			status.Set(1, fmt.Sprintf("%d selected", max(a, c)-min(a, c)))
		}
	}
	bar := widgets.NewRichTextBar(ed)
	col := widgets.NewColumn(bar, ed).WithGap(6)
	col.AddFlex(ed, 1)
	return col
}

// mdiView is an area of three documents with the buttons that arrange
// them, a New that opens another, and the tabbed view.
func mdiView(status *widgets.StatusBar) widget.Component {
	area := widgets.NewMDIArea()
	area.SetAccessibleName("Documents")
	n := 0
	open := func() {
		n++
		title := fmt.Sprintf("Untitled %d", n)
		body := widgets.NewTextArea(fmt.Sprintf("%s\n\nDrag the caption to move it, an edge to resize it.", title), "", nil)
		body.SetAccessibleName(title)
		w := area.AddWindow(title, widgets.NewPad(4, body))
		w.SetGeometry(w.Geometry()) // keep the cascade place
	}
	for range 3 {
		open()
	}
	area.OnActivate = func(w *widgets.MDIWindow) { status.Set(0, "Active: "+w.Title()) }
	tabbed := widgets.NewSwitch("Tabs", false, func(on bool) {
		if on {
			area.SetViewMode(widgets.MDITabbed)
		} else {
			area.SetViewMode(widgets.MDISubWindows)
		}
	})
	row := widgets.NewRow(
		widgets.NewButton("New", open),
		widgets.NewButton("Cascade", area.Cascade),
		widgets.NewButton("Tile", area.Tile),
		widgets.NewSpacer(),
		tabbed,
	).WithGap(6)
	col := widgets.NewColumn(row, area).WithGap(6)
	col.AddFlex(area, 1)
	return col
}

// wizardView is an account set-up: a required name, an address that
// must hold an @, an optional page and a page only a proxy brings in.
func wizardView(status *widgets.StatusBar) widget.Component {
	name := widgets.NewTextField("", "Full name", nil)
	email := widgets.NewTextField("", "name@example.com", nil)
	form := widgets.NewForm()
	form.AddRow("Name", name)
	form.AddRow("Email", email)
	proxy := widgets.NewCheckbox("Connect through a proxy", false, nil)
	proxyHost := widgets.NewTextField("", "proxy.example.com:3128", nil)
	proxyHost.SetAccessibleName("Proxy")
	var wiz *widgets.Wizard
	name.OnChange = func(string) { wiz.UpdateButtons() }
	wiz = widgets.NewWizard("New account",
		&widgets.WizardPage{Title: "Welcome", Subtitle: "This assistant sets up a mail account.",
			Content: widgets.NewLabel("It takes a minute. Press Next to begin.")},
		&widgets.WizardPage{Title: "Your account", Subtitle: "The name and address people see.", Content: form,
			Complete: func() bool { return strings.TrimSpace(name.Text) != "" },
			Validate: func() error {
				if !strings.Contains(email.Text, "@") {
					return errors.New("The address needs an @ between the name and the domain.")
				}
				return nil
			}},
		&widgets.WizardPage{Title: "Connection", Subtitle: "How to reach the server.", Optional: true, Content: proxy},
		&widgets.WizardPage{Title: "Proxy", Subtitle: "The proxy to go through.", Content: proxyHost,
			Skip: func() bool { return !proxy.Checked }},
		&widgets.WizardPage{Title: "Summary", Subtitle: "Everything is ready.",
			Content: widgets.NewLabel("Press Finish to create the account.")},
	)
	wiz.OnFinish = func() {
		status.Set(0, "Account created for "+name.Text)
		wiz.Restart()
	}
	wiz.OnCancel = func() bool {
		status.Set(0, "Set-up cancelled")
		wiz.Restart()
		return true
	}
	wiz.OnHelp = func(page int) { status.Set(0, "Help for "+wiz.Pages()[page].Title) }
	return wiz
}
