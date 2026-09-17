// Gallery is the widget showcase used by examples/gallery, the
// uitest-driver and Settings.
package demo

import (
	"fmt"
	"sort"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// GalleryHost is what the showcase needs from the application around it:
// where its dialogs go up, and what its Window, Theme and Quit controls
// do. The standalone window fills all of it in; Settings, which shows the
// gallery in a ThemeScope under the theme preview, leaves out what would
// re-theme or close Settings itself (a nil field greys its control out).
type GalleryHost struct {
	// Light says which palette the gallery is drawn in: it sets the Dark
	// chrome switch and the View menu's radio.
	Light bool
	// Root is where modal dialogs are shown — the window's content, read
	// when a dialog opens rather than while the gallery is being built.
	Root func() widget.Component
	// NewWindow opens the Window button's second OS window.
	NewWindow func() error
	// SwitchTheme flips the palette the gallery is drawn in.
	SwitchTheme func(light bool)
	// Quit ends the run loop (File ▸ Quit).
	Quit func()
}

// Gallery is the showcase as a window of its own: menu bar, title bar,
// tool bar, the buttons and fields beside the tabbed views, and a status
// bar. examples/gallery, the uitest-driver and the screenshot fixtures
// draw this one.
func Gallery(a *app.Application, win *app.Window, light bool) widget.Component {
	return GalleryWindow(GalleryHost{
		Light: light,
		Root:  func() widget.Component { return win.Content() },
		NewWindow: func() error {
			w2, err := a.NewWindow(platform.WindowOptions{Title: "Second window", Width: 420, Height: 280})
			if err != nil {
				return err
			}
			w2.SetContent(widgets.NewPanel("Dialog window",
				widgets.NewLabel("This is a second OS window."),
				widgets.NewButton("Close", func() { w2.Close() }),
			))
			return nil
		},
		SwitchTheme: func(light bool) {
			palette := style.ThemeDark
			if light {
				palette = style.ThemeLight
			}
			a.SetLook(style.WithTheme(win.Look(), palette))
			win.SetContent(Gallery(a, win, light))
		},
		Quit: a.Quit,
	})
}

// GalleryWindow assembles the showcase the way a window wants it: the
// chrome across the top, the buttons and fields in a scrolling column
// beside the tabbed views, the status bar along the bottom.
func GalleryWindow(host GalleryHost) widget.Component {
	p := buildGalleryParts(host)
	leftCol := widgets.NewColumn(
		widgets.NewTitle("uitoolkit"),
		widgets.NewLabel("paintengine2d  ·  v"+uitoolkit.Version),
		p.buttons,
		p.fields,
	).WithGap(10).WithPad(12)
	left := widgets.NewScrollView(leftCol)

	tabs := widgets.NewTabView(p.tabs()...)
	tabs.OnChange = func(i int) {
		if i >= 0 && i < len(p.views) {
			p.status.Set(0, "Tab: "+p.views[i].title)
		}
	}
	right := widgets.NewPad(10, tabs)

	split := widgets.NewSplitter(true, left, right)
	split.Ratio = 0.46

	root := widgets.NewColumn(p.menu, p.chrome, p.tools, split, p.status).WithGap(0)
	root.AddFlex(split, 1)
	return root
}

// GalleryPane assembles the same showcase as one column that scrolls,
// without the menu bar and title bar a window owns: the tool bar, the
// buttons and fields, then every tabbed view stacked under them as its
// own panel, and the status bar. Settings shows it in a ThemeScope under
// the theme preview, so a pack can be judged on every control at once
// instead of a tab at a time.
func GalleryPane(host GalleryHost) *widgets.ScrollView {
	p := buildGalleryParts(host)
	col := widgets.NewColumn(
		p.tools,
		widgets.NewWrap(p.buttons, p.fields),
		widgets.NewSpacerSize(0, 2),
	).WithGap(10).WithPad(10)
	for _, v := range p.views {
		col.Add(boxed(v.height, v.content))
	}
	col.Add(p.status)
	return widgets.NewScrollView(col)
}

// galleryView is one of the showcase's big views: a tab of the window, a
// panel stacked in the Settings pane. Height is what the pane holds it to,
// where the scrolling column bounds nothing and a list would otherwise be
// as tall as all its rows; 0 leaves the view its own height.
type galleryView struct {
	title   string
	content widget.Component
	height  float32
}

// galleryParts is one built set of the showcase's widgets. The window and
// the Settings pane arrange the same parts differently; neither owns them.
type galleryParts struct {
	menu    *widgets.MenuBar
	chrome  *widgets.TitleBar
	tools   *widgets.ToolBar
	buttons *widgets.Panel
	fields  *widgets.Panel
	views   []galleryView
	status  *widgets.StatusBar
}

func (p galleryParts) tabs() []widgets.Tab {
	out := make([]widgets.Tab, len(p.views))
	for i, v := range p.views {
		out[i] = widgets.Tab{Title: v.title, Content: v.content}
	}
	return out
}

// fixedHeight holds its child to a height given in 1x pixels and follows
// the look's display scale, so a view that would otherwise grow without
// bound — a list as tall as all its rows, a tree as tall as its nodes —
// keeps its size in a column that scrolls.
type fixedHeight struct {
	widget.Base
	h float32
}

// boxed is child at h 1x pixels tall; height 0 leaves it its own.
func boxed(h float32, child widget.Component) widget.Component {
	if h <= 0 {
		return child
	}
	b := &fixedHeight{h: h}
	b.Init(b)
	if child != nil {
		b.Add(child)
	}
	return b
}

func (b *fixedHeight) Measure(c layout.Constraints) paintengine2d.Point {
	h := style.Dip(b.Look(), b.h)
	var w float32
	if kids := b.Children(); len(kids) > 0 {
		w = kids[0].Measure(layout.Constraints{MinW: c.MinW, MaxW: c.MaxW, MaxH: h}).X
	}
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (b *fixedHeight) Arrange(r paintengine2d.Rect) {
	b.SetBounds(r)
	if kids := b.Children(); len(kids) > 0 {
		kids[0].Arrange(paintengine2d.XYWH(0, 0, r.Dx(), r.Dy()))
	}
}

func buildGalleryParts(host GalleryHost) galleryParts {
	light := host.Light
	root := func() widget.Component { return nil }
	if host.Root != nil {
		root = host.Root
	}
	status := widgets.NewStatusBar("Ready.", "Ln 1, Col 1", "v"+uitoolkit.Version)
	chrome := widgets.NewTitleBar("Widget gallery", "v"+uitoolkit.Version)

	volume := widgets.NewLabel("Volume  60%")
	slider := widgets.NewSlider(0, 100, 60, func(v float32) {
		volume.SetText(fmt.Sprintf("Volume  %d%%", int(v+0.5)))
	})

	name := widgets.NewTextField("Ada Lovelace", "Display name", nil)
	columns := widgets.NewNumberField(1, 12, 3, 1, func(v float64) {
		status.Set(0, fmt.Sprintf("Columns  %d", int(v)))
	})
	columns.Tip = "Layout columns (spinner)"
	columns.SetAccessibleName("Columns")
	slider.SetAccessibleName("Volume")
	slider.Ticks, slider.TickInterval = widgets.TicksBelow, 10
	engine := widgets.NewComboBox([]string{
		"paintengine2d", "Software raster", "Offscreen pixmap",
	}, 0, func(i int) {
		names := []string{"paintengine2d", "Software raster", "Offscreen pixmap"}
		if i >= 0 && i < len(names) {
			status.Set(0, "Engine: "+names[i])
		}
	})
	engine.Placeholder = "Paint engine"
	// An editable combo box: type a size or pick one.
	size := widgets.NewComboBox([]string{"8", "9", "10", "11", "12", "14", "16", "18", "24", "36", "48", "72"}, 4, func(i int) {
		status.Set(0, "Font size picked")
	})
	size.SetEditable(true)
	size.SetAccessibleName("Font size")
	size.OnSubmit = func(s string) { status.Set(0, "Font size "+s) }

	progress := widgets.NewProgressBar(0.42)
	progress.ShowText = true
	progressLbl := widgets.NewLabel("Build  42%")
	busy := widgets.NewBusyBar(0.35)

	radios := widgets.NewRadioGroup([]string{"Dark graphite", "Light paper", "System follow"}, 0, func(i int) {
		switch i {
		case 1:
			status.Set(0, "Radio: Light paper")
		case 2:
			status.Set(0, "Radio: System follow")
		default:
			status.Set(0, "Radio: Dark graphite")
		}
	})

	checks := widgets.NewColumn(
		widgets.NewCheckbox("Enable notifications", true, nil),
		widgets.NewCheckbox("Launch at login", false, nil),
		widgets.NewCheckbox("Hardware acceleration", true, nil),
	).WithGap(4)
	toggles := widgets.NewColumn(
		widgets.NewSwitch("Dark chrome", !light, nil),
		widgets.NewSwitch("Compact layout", false, nil),
	).WithGap(4)
	blurb := widgets.NewTextArea(
		"Multi-line TextArea.\nWrap or scroll; Return inserts a line.",
		"Notes",
		func(s string) { status.Set(0, "TextArea edited") },
	)
	blurb.MinRows = 3
	more := widgets.NewExpander("Advanced", false, widgets.NewColumn(
		widgets.NewSwitch("Diagnostic overlay", false, nil),
		widgets.NewLabel("Hidden until the header is expanded."),
	).WithGap(6))
	session := widgets.NewExpander("Session", true, widgets.NewColumn(
		widgets.NewSwitch("Autosave", true, nil),
		widgets.NewLabel("Writes a sidecar next to the project."),
	).WithGap(6))
	formAcc := widgets.NewAccordion(true, session, more)

	clicks := 0
	clickLbl := widgets.NewLabel("Clicked 0 times")
	primary := widgets.NewButton("Primary action", func() {
		clicks++
		clickLbl.SetText(fmt.Sprintf("Clicked %d times", clicks))
	})
	primary.Primary = true
	plain := widgets.NewButton("Secondary", func() {
		status.Set(0, "Secondary clicked")
	})
	disabled := widgets.NewButton("Disabled", nil)
	disabled.SetEnabled(false)

	about := widgets.NewButton("About…", func() {
		widgets.Info(root(), "About uitoolkit",
			"Pure Go desktop UI. Paints only with paintengine2d.", nil)
	})
	ask := widgets.NewButton("Confirm…", func() {
		widgets.Confirm(root(), "Quit gallery?",
			"Close the showcase window and leave the run loop.",
			func(yes bool) {
				if yes {
					status.Set(0, "Confirmed")
				} else {
					status.Set(0, "Cancelled")
				}
			})
	})
	warn := widgets.NewButton("Warn…", func() {
		widgets.Warn(root(), "Unsaved changes",
			"The current theme preset is not written to disk.", nil)
	})

	other := widgets.NewButton("Window", func() {
		if host.NewWindow == nil {
			return
		}
		if err := host.NewWindow(); err != nil {
			status.Set(0, err.Error())
		}
	})
	if host.NewWindow == nil {
		other.SetEnabled(false)
	}

	// The palette the gallery draws in: the button flips it, the View menu
	// picks a side. Both are inert where the host owns the look (Settings).
	switchTheme := func(light bool) {
		if host.SwitchTheme != nil {
			host.SwitchTheme(light)
		}
	}
	themeBtn := widgets.NewButton("Theme", func() { switchTheme(!light) })
	if host.SwitchTheme == nil {
		themeBtn.SetEnabled(false)
	}

	buttons := widgets.NewPanel("Buttons",
		// Each group folds onto another line when the column is narrow.
		wrapRow(primary, plain, disabled),
		wrapRow(about, ask, warn),
		wrapRow(other, themeBtn),
		clickLbl,
	)

	openFile := widgets.NewButton("Open file…", func() {
		widgets.ShowFileDialog(root(), widgets.FileDialogOptions{
			Title:  "Open project file",
			Path:   "/project/uitoolkit",
			Filter: "*.go",
			Entries: []widgets.FileInfo{
				{Name: "app", Dir: true},
				{Name: "widgets", Dir: true},
				{Name: "export.go"},
				{Name: "version.go"},
				{Name: "doc.go"},
			},
			OnPick: func(p string) { status.Set(0, "Open "+p) },
		})
	})
	openFile.Tip = "Stub file picker (modal list + path)"

	fields := widgets.NewPanel("Fields",
		widgets.NewLabel("Name"),
		name,
		widgets.NewLabel("Columns"),
		columns,
		widgets.NewLabel("Paint engine"),
		engine,
		widgets.NewLabel("Font size (type or pick)"),
		size,
		widgets.NewLabel("Theme preset"),
		radios,
		volume,
		slider,
		progressLbl,
		progress,
		widgets.NewLabel("Busy"),
		busy,
		widgets.NewLabel("Options"),
		checks,
		widgets.NewSeparator(),
		widgets.NewLabel("Toggles"),
		toggles,
		widgets.NewLabel("Notes"),
		blurb,
		openFile,
	)

	newBtn := widgets.ToolIconBtn(style.IconNew, "", func() { status.Set(0, "New") })
	newBtn.Tip = "New window"
	openBtn := widgets.ToolIconBtn(style.IconOpen, "", func() { openFile.OnClick() })
	openBtn.Tip = "Open file"
	saveBtn := widgets.ToolIconBtn(style.IconSave, "", func() { status.Set(0, "Save") })
	saveBtn.Tip = "Save project"
	cutBtn := widgets.ToolIconBtn(style.IconCut, "", func() { status.Set(0, "Cut") })
	cutBtn.Tip = "Cut"
	copyBtn := widgets.ToolIconBtn(style.IconCopy, "", func() {
		if s := name.SelectedText(); s != "" {
			platform.ClipboardSet(s)
		}
		status.Set(0, "Copy")
	})
	copyBtn.Tip = "Copy"
	pasteBtn := widgets.ToolIconBtn(style.IconPaste, "", func() {
		name.SetText(name.Text + platform.ClipboardGet())
		status.Set(0, "Paste")
	})
	pasteBtn.Tip = "Paste"
	aboutTool := widgets.ToolText("About", func() { about.OnClick() })
	aboutTool.Tip = "About uitoolkit"
	snap := widgets.ToolToggle("Snap", true, func() { status.Set(0, "Snap toggled") })
	snap.Tip = "Snap to grid"
	toolbar := widgets.NewToolBar(
		newBtn, openBtn, saveBtn, widgets.ToolDivider(),
		cutBtn, copyBtn, pasteBtn, widgets.ToolDivider(),
		aboutTool, snap,
	)

	long := widgets.NewColumn()
	for i := 1; i <= 40; i++ {
		long.Add(widgets.NewLabel(fmt.Sprintf("Row %02d  —  scrollable content", i)))
	}
	scroll := widgets.NewScrollView(long)

	files := []string{
		"README.md", "go.mod", "LICENSE", "widget/base.go", "style/look.go",
		"examples/gallery/main.go", "examples/notes/main.go", "platform/x11_linux.go",
		"app/window.go", "layout/flex.go",
	}
	for i := 0; i < 30; i++ {
		files = append(files, fmt.Sprintf("item-%02d.txt", i+1))
	}
	selected := widgets.NewLabel("Selected: README.md")
	list := widgets.NewListView(len(files), func(i int) string { return files[i] }, func(i int) {
		selected.SetText("Selected: " + files[i])
	})
	list.Selected = 0
	list.OnContext = func(i int, p paintengine2d.Point) {
		if i >= 0 && i < len(files) {
			status.Set(0, "Context: "+files[i])
			selected.SetText("Selected: " + files[i])
		}
		widgets.ShowContextMenu(list, p,
			widgets.Item("Open", func() {
				if i >= 0 && i < len(files) {
					status.Set(0, "Open "+files[i])
				}
			}),
			widgets.Item("Copy path", func() {
				if i >= 0 && i < len(files) {
					platform.ClipboardSet(files[i])
					status.Set(0, "Copied "+files[i])
				}
			}),
			widgets.Sep(),
			widgets.Item("Reveal in list", nil),
		)
	}
	cardDemo := widgets.NewCardList(4, func(i int) widgets.CardContent {
		return widgets.CardContent{
			Title: files[i%len(files)], Subtitle: "CardList row",
			Meta: "now", Snippet: "Thunderbird-style multi-line card",
			Bold: i == 0,
		}
	}, nil)
	cardDemo.CardHeight = 56
	listPane := widgets.NewColumn(selected, list, widgets.NewLabel("Cards"), cardDemo).WithGap(6)
	listPane.AddFlex(list, 1)
	listPane.AddFlex(cardDemo, 1)

	treeSel := widgets.NewLabel("Selected: src")
	srcApp := widgets.NewTreeNode("app",
		widgets.NewTreeNode("app.go"),
		widgets.NewTreeNode("window.go"),
	)
	srcWidgets := widgets.NewTreeNode("widgets",
		widgets.NewTreeNode("menu.go"),
		widgets.NewTreeNode("tabs.go"),
		widgets.NewTreeNode("tree.go"),
	)
	srcWidgets.Expanded = true
	src := widgets.NewTreeNode("src", srcApp, srcWidgets)
	src.Expanded = true
	docs := widgets.NewTreeNode("docs", widgets.NewTreeNode("screenshots"))
	rootNode := widgets.NewTreeNode("uitoolkit", src, docs, widgets.NewTreeNode("go.mod"))
	rootNode.Expanded = true
	tree := widgets.NewTreeView(rootNode)
	tree.Selected = src
	tree.OnSelect = func(n *widgets.TreeNode) {
		if n != nil {
			treeSel.SetText("Selected: " + n.Label)
			status.Set(1, n.Label)
		}
	}
	tree.OnContext = func(n *widgets.TreeNode, p paintengine2d.Point) {
		if n == nil {
			return
		}
		items := []*widgets.MenuItem{
			widgets.Item("Select "+n.Label, func() { tree.OnSelect(n) }),
		}
		if !n.Leaf() {
			label := "Expand"
			if n.Expanded {
				label = "Collapse"
			}
			items = append(items, widgets.Item(label, func() { tree.Toggle(n) }))
		}
		widgets.ShowContextMenu(tree, p, items...)
	}
	treePane := widgets.NewColumn(treeSel, tree).WithGap(6)
	treePane.AddFlex(tree, 1)

	type pkgRow struct{ name, lines, status string }
	pkgs := []pkgRow{
		{"widget", "412", "stable"},
		{"widgets", "2140", "wip"},
		{"app", "288", "stable"},
		{"style", "960", "stable"},
		{"platform", "540", "linux"},
		{"layout", "190", "stable"},
		{"examples", "720", "demo"},
	}
	tableSel := widgets.NewLabel("Selected: widget")
	var table *widgets.TableView
	table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "Package", Sortable: true},
		{Title: "Lines", Width: 72, Sortable: true, Align: style.AlignEnd},
		{Title: "Status", Width: 80, Sortable: true},
	}, len(pkgs), func(row, col int) string {
		if row < 0 || row >= len(pkgs) {
			return ""
		}
		switch col {
		case 1:
			return pkgs[row].lines
		case 2:
			return pkgs[row].status
		default:
			return pkgs[row].name
		}
	}, func(i int) {
		if i >= 0 && i < len(pkgs) {
			tableSel.SetText("Selected: " + pkgs[i].name)
			status.Set(0, "Table: "+pkgs[i].name)
		}
	})
	table.Selected = 0
	table.OnSort = func(col int, asc bool) {
		sort.SliceStable(pkgs, func(i, j int) bool {
			var less bool
			switch col {
			case 1:
				less = pkgs[i].lines < pkgs[j].lines
			case 2:
				less = pkgs[i].status < pkgs[j].status
			default:
				less = pkgs[i].name < pkgs[j].name
			}
			if !asc {
				return !less
			}
			return less
		})
		table.Invalidate()
	}
	tablePane := widgets.NewColumn(tableSel, table).WithGap(6)
	tablePane.AddFlex(table, 1)

	formNotes := widgets.NewTextArea(
		"Form tab notes.\nAccordion below; this field wraps.",
		"Form notes",
		nil,
	)
	formNotes.MinRows = 4
	// A dialog form: labels line up in their own column (right-aligned in
	// Mac looks), fields take the rest.
	account := widgets.NewForm()
	account.AddRow("Name", widgets.NewTextField("Ada Lovelace", "Full name", nil))
	account.AddRow("Email", widgets.NewTextField("ada@example.com", "Address", nil))
	account.AddRow("Server type", widgets.NewComboBox([]string{"IMAP", "POP3", "Exchange"}, 0, nil))
	account.AddRow("Port", widgets.NewNumberField(1, 65535, 993, 1, nil))
	account.AddRow("Renews", widgets.NewDateField(time.Date(2026, time.September, 14, 0, 0, 0, 0, time.Local), nil))
	account.AddRow("Label colour", widgets.NewColorButton(paintengine2d.RGB(0.13, 0.43, 0.47), nil))
	account.AddRow("Show as", widgets.NewSegmented([]string{"List", "Cards", "Columns"}, 1, nil))
	account.AddWide(widgets.NewCheckbox("Use TLS", true, nil))
	form := widgets.NewPanel("Form",
		account,
		widgets.NewSeparator(),
		widgets.NewLabel("TextArea, accordion, switch, separator, spacer."),
		formNotes,
		widgets.NewSeparator(),
		widgets.NewRow(
			widgets.NewSwitch("Live preview", true, nil),
			widgets.NewVSeparator(),
			widgets.NewLabel("v"+uitoolkit.Version),
		).WithGap(8),
		formAcc,
		widgets.NewSpacerSize(0, 4),
	)

	menubar := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemIconAccel(style.IconNew, "&New window", "Ctrl+N", func() { other.OnClick() }),
			widgets.ItemIconAccel(style.IconOpen, "&Open…", "Ctrl+O", func() { openFile.OnClick() }),
			widgets.ItemAccel("&About", "F1", func() { about.OnClick() }),
			widgets.Sep(),
			widgets.ItemAccel("&Quit", "Ctrl+Q", func() {
				if host.Quit != nil {
					host.Quit()
				}
			}),
		),
		widgets.NewMenu("&Edit",
			widgets.ItemAccel("&Copy", "Ctrl+C", func() {
				if s := name.SelectedText(); s != "" {
					platform.ClipboardSet(s)
					status.Set(0, "Copied")
				}
			}),
			widgets.ItemAccel("&Paste", "Ctrl+V", func() {
				name.SetText(name.Text + platform.ClipboardGet())
				status.Set(0, "Pasted")
			}),
			widgets.Sep(),
			&widgets.MenuItem{Text: "Undo", Shortcut: "Ctrl+Z", Disabled: true},
		),
		widgets.NewMenu("&View",
			widgets.RadioItem("&Dark", "palette", !light, func() { switchTheme(false) }),
			widgets.RadioItem("&Light", "palette", light, func() { switchTheme(true) }),
		),
	)

	return galleryParts{
		menu:    menubar,
		chrome:  chrome,
		tools:   toolbar,
		buttons: buttons,
		fields:  fields,
		status:  status,
		views: []galleryView{
			{"Scroll", widgets.NewPanel("ScrollView", scroll), 240},
			{"List", widgets.NewPanel("ListView", listPane), 340},
			{"Tree", widgets.NewPanel("TreeView", treePane), 0},
			{"Table", widgets.NewPanel("TableView", tablePane), 0},
			{"Form", form, 0},
		},
	}
}

// wrapRow is a row of controls that wraps when it runs out of width.
func wrapRow(children ...widget.Component) *widgets.Wrap {
	w := widgets.NewWrap(children...)
	w.Gap = 10
	return w
}
