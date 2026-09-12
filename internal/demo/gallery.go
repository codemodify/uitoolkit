// Gallery is the widget showcase used by examples/gallery and the uitest-driver.
package demo

import (
	"fmt"
	"sort"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func Gallery(a *app.Application, win *app.Window, light bool) widget.Component {
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
	engine := widgets.NewComboBox([]string{
		"paintengine2d", "Software raster", "Offscreen pixmap",
	}, 0, func(i int) {
		names := []string{"paintengine2d", "Software raster", "Offscreen pixmap"}
		if i >= 0 && i < len(names) {
			status.Set(0, "Engine: "+names[i])
		}
	})
	engine.Placeholder = "Paint engine"

	progress := widgets.NewProgressBar(0.42)
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
		widgets.Info(win.Content(), "About uitoolkit",
			"Pure Go desktop UI. Paints only with paintengine2d.", nil)
	})
	ask := widgets.NewButton("Confirm…", func() {
		widgets.Confirm(win.Content(), "Quit gallery?",
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
		widgets.Warn(win.Content(), "Unsaved changes",
			"The current theme preset is not written to disk.", nil)
	})

	other := widgets.NewButton("Window", func() {
		w2, err := a.NewWindow(platform.WindowOptions{Title: "Second window", Width: 420, Height: 280})
		if err != nil {
			status.Set(0, err.Error())
			return
		}
		w2.SetContent(widgets.NewPanel("Dialog window",
			widgets.NewLabel("This is a second OS window."),
			widgets.NewButton("Close", func() { w2.Close() }),
		))
	})

	themeBtn := widgets.NewButton("Theme", func() {
		next := style.LookAppearance(win.Look())
		if next.Theme == style.ThemeLight {
			next.Theme = style.ThemeDark
		} else {
			next.Theme = style.ThemeLight
		}
		a.SetLook(style.WithAppearance(win.Look(), next))
		win.SetContent(Gallery(a, win, next.Theme == style.ThemeLight))
	})

	buttons := widgets.NewPanel("Buttons",
		widgets.NewRow(primary, plain, disabled).WithGap(10),
		widgets.NewRow(about, ask, warn).WithGap(10),
		widgets.NewRow(other, themeBtn).WithGap(10),
		clickLbl,
	)

	openFile := widgets.NewButton("Open file…", func() {
		widgets.ShowFileDialog(win.Content(), widgets.FileDialogOptions{
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
	form := widgets.NewPanel("Form",
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
	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Scroll", Content: widgets.NewPanel("ScrollView", scroll)},
		widgets.Tab{Title: "List", Content: widgets.NewPanel("ListView", listPane)},
		widgets.Tab{Title: "Tree", Content: widgets.NewPanel("TreeView", treePane)},
		widgets.Tab{Title: "Table", Content: widgets.NewPanel("TableView", tablePane)},
		widgets.Tab{Title: "Form", Content: form},
	)
	tabs.OnChange = func(i int) {
		names := []string{"Scroll", "List", "Tree", "Table", "Form"}
		if i >= 0 && i < len(names) {
			status.Set(0, "Tab: "+names[i])
		}
	}

	menubar := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemAccel("&New window", "Ctrl+N", func() { other.OnClick() }),
			widgets.ItemAccel("&Open…", "Ctrl+O", func() { openFile.OnClick() }),
			widgets.ItemAccel("&About", "F1", func() { about.OnClick() }),
			widgets.Sep(),
			widgets.ItemAccel("&Quit", "Ctrl+Q", func() { a.Quit() }),
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
			widgets.CheckItem("&Dark", !light, func() {
				a.SetLook(style.WithTheme(win.Look(), style.ThemeDark))
				win.SetContent(Gallery(a, win, false))
			}),
			widgets.CheckItem("&Light", light, func() {
				a.SetLook(style.WithTheme(win.Look(), style.ThemeLight))
				win.SetContent(Gallery(a, win, true))
			}),
		),
	)

	leftCol := widgets.NewColumn(
		widgets.NewTitle("uitoolkit"),
		widgets.NewLabel("paintengine2d  ·  v"+uitoolkit.Version),
		buttons,
		fields,
	).WithGap(10).WithPad(12)
	left := widgets.NewScrollView(leftCol)

	right := widgets.NewPad(10, tabs)

	split := widgets.NewSplitter(true, left, right)
	split.Ratio = 0.46

	root := widgets.NewColumn(menubar, chrome, toolbar, split, status).WithGap(0)
	root.AddFlex(split, 1)
	_ = light
	return root
}
