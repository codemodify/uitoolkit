// Command gallery is the uitoolkit widget showcase.
package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/internal/demo"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	shot := flag.String("screenshot", "", "write PNG gallery into this directory and exit")
	headless := flag.Bool("headless", false, "paint offscreen (no X11)")
	flag.Parse()

	if *shot != "" {
		if err := writeScreenshots(*shot); err != nil {
			log.Fatal(err)
		}
		return
	}
	a := uitoolkit.New(uitoolkit.Options{Look: uitoolkit.DarkLook(), Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit gallery", Width: 1000, Height: 760, MinWidth: 720, MinHeight: 480,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(buildGallery(a, win, false))
	if *headless {
		_ = win.WritePNG("gallery.png")
		fmt.Println("wrote gallery.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

func buildGallery(a *app.Application, win *app.Window, light bool) widget.Component {
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
		if win.Look().Name() == "dark" {
			a.SetLook(style.LightLook())
		} else {
			a.SetLook(style.DarkLook())
		}
		win.SetContent(buildGallery(a, win, win.Look().Name() == "light"))
	})

	buttons := widgets.NewPanel("Buttons",
		widgets.NewRow(primary, plain, disabled).WithGap(10),
		widgets.NewRow(about, ask, warn).WithGap(10),
		widgets.NewRow(other, themeBtn, clickLbl).WithGap(10),
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
	listPane := widgets.NewColumn(selected, list).WithGap(6)
	listPane.AddFlex(list, 1)

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

	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Scroll", Content: widgets.NewPanel("ScrollView", scroll)},
		widgets.Tab{Title: "List", Content: widgets.NewPanel("ListView", listPane)},
		widgets.Tab{Title: "Tree", Content: widgets.NewPanel("TreeView", treePane)},
		widgets.Tab{Title: "Table", Content: widgets.NewPanel("TableView", tablePane)},
	)
	tabs.OnChange = func(i int) {
		names := []string{"Scroll", "List", "Tree", "Table"}
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
				a.SetLook(style.DarkLook())
				win.SetContent(buildGallery(a, win, false))
			}),
			widgets.CheckItem("&Light", light, func() {
				a.SetLook(style.LightLook())
				win.SetContent(buildGallery(a, win, true))
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

func writeScreenshots(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	type shot struct {
		name  string
		look  style.LookAndFeel
		setup func(a *app.Application, w *app.Window)
	}
	shots := []shot{
		{name: "gallery-dark.png", look: style.DarkLook(), setup: nil},
		{
			name: "gallery-light.png",
			look: style.LightLook(),
			setup: func(a *app.Application, w *app.Window) {
				selectGalleryTab(w, 1)
			},
		},
		{
			name: "gallery-dialog.png",
			look: style.DarkLook(),
			setup: func(a *app.Application, w *app.Window) {
				widgets.Info(w.Content(), "About uitoolkit",
					"Pure Go desktop UI. Paints only with paintengine2d.", nil)
			},
		},
		{
			name: "gallery-combo.png",
			look: style.DarkLook(),
			setup: func(a *app.Application, w *app.Window) {
				openGalleryCombo(w)
			},
		},
		{
			name: "gallery-message.png",
			look: style.DarkLook(),
			setup: func(a *app.Application, w *app.Window) {
				widgets.ShowMessageBox(w.Content(), widgets.MessageBoxOptions{
					Title:   "Unsaved changes",
					Message: "Discard the draft theme and revert to Dark graphite?",
					Kind:    widgets.MessageWarning,
					Buttons: widgets.ButtonsYesNoCancel,
				})
			},
		},
		{
			name: "gallery-scroll.png",
			look: style.DarkLook(),
			setup: func(a *app.Application, w *app.Window) {
				selectGalleryTab(w, 0)
				scrollGallery(w)
				w.Inject(platform.Event{
					Kind:   platform.EventScroll,
					Pos:    paintengine2d.Pt(720, 280),
					Scroll: paintengine2d.Pt(0, 80),
				})
			},
		},
		{
			name: "gallery-menu.png",
			look: style.DarkLook(),
			setup: func(a *app.Application, w *app.Window) {
				openGalleryMenu(w, 0)
			},
		},
		{
			name: "gallery-tree.png",
			look: style.DarkLook(),
			setup: func(a *app.Application, w *app.Window) {
				selectGalleryTab(w, 2)
			},
		},
		{
			name: "gallery-table.png",
			look: style.DarkLook(),
			setup: func(a *app.Application, w *app.Window) {
				selectGalleryTab(w, 3)
				sortGalleryTable(w)
			},
		},
		{
			name: "gallery-file.png",
			look: style.DarkLook(),
			setup: func(a *app.Application, w *app.Window) {
				widgets.ShowFileDialog(w.Content(), widgets.FileDialogOptions{
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
				})
			},
		},
		{
			name: "gallery-tooltip.png",
			look: style.DarkLook(),
			setup: func(a *app.Application, w *app.Window) {
				hoverGalleryTool(w, 2)
				a.PumpOnce()
				w.RevealTooltip()
			},
		},
	}
	for _, s := range shots {
		a := uitoolkit.New(uitoolkit.Options{Look: s.look, Headless: true})
		w, err := a.NewWindow(platform.WindowOptions{
			Title: "uitoolkit gallery", Width: 1000, Height: 760, Headless: true,
		})
		if err != nil {
			return err
		}
		w.SetContent(buildGallery(a, w, s.look.Name() == "light"))
		a.PumpOnce()
		if s.setup != nil {
			s.setup(a, w)
			a.PumpOnce()
		}
		path := filepath.Join(dir, s.name)
		if err := w.WritePNG(path); err != nil {
			return err
		}
		fmt.Println("wrote", path)
		w.Close()
	}
	if err := writeNotesShot(filepath.Join(dir, "notes.png")); err != nil {
		return err
	}
	if err := writeWidgetsCloseup(filepath.Join(dir, "widgets.png")); err != nil {
		return err
	}
	if err := writeToolbarShot(filepath.Join(dir, "gallery-toolbar.png")); err != nil {
		return err
	}
	return verifyDistinctPNGs(dir, screenshotNames)
}

var screenshotNames = []string{
	"gallery-dark.png", "gallery-light.png", "gallery-dialog.png",
	"gallery-scroll.png", "gallery-menu.png", "gallery-tree.png",
	"gallery-toolbar.png", "gallery-combo.png", "gallery-message.png",
	"gallery-table.png", "gallery-file.png", "gallery-tooltip.png",
	"notes.png", "widgets.png",
}

func selectGalleryTab(w *app.Window, i int) {
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TabView); ok {
			tv.Select(i)
		}
	})
}

func openGalleryMenu(w *app.Window, i int) {
	widget.Walk(w.Content(), func(c widget.Component) {
		if mb, ok := c.(*widgets.MenuBar); ok {
			mb.Open(i)
		}
	})
}

func sortGalleryTable(w *app.Window) {
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && len(tv.Columns) >= 3 {
			tv.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(20, 10)})
			tv.Selected = 2
			if tv.OnSelect != nil {
				tv.OnSelect(2)
			}
			tv.Invalidate()
		}
	})
}

func hoverGalleryTool(w *app.Window, i int) {
	widget.Walk(w.Content(), func(c widget.Component) {
		if tb, ok := c.(*widgets.ToolBar); ok {
			tb.Hover(i)
			o := widget.DeviceOrigin(tb)
			c := tb.ItemCenter(i)
			w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+c.X, o.Y+c.Y)})
		}
	})
}

func openGalleryCombo(w *app.Window) {
	widget.Walk(w.Content(), func(c widget.Component) {
		if cb, ok := c.(*widgets.ComboBox); ok {
			cb.Open()
		}
	})
}

func scrollGallery(w *app.Window) {
	widget.Walk(w.Content(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.ScrollView:
			v.ScrollTo(v.MaxOffset() * 0.48)
		case *widgets.ListView:
			if v.Count > 10 {
				rh := v.RowHeight
				if rh <= 0 {
					rh = 28
				}
				v.OffsetY = rh * 5
				sel := 7
				v.Selected = sel
				if v.OnSelect != nil {
					v.OnSelect(sel)
				}
				v.Invalidate()
			}
		}
	})
}

func verifyDistinctPNGs(dir string, names []string) error {
	seen := map[string]string{}
	for _, name := range names {
		path := filepath.Join(dir, name)
		sum, err := fileMD5(path)
		if err != nil {
			return err
		}
		if other, ok := seen[sum]; ok {
			return fmt.Errorf("screenshot %s is a duplicate of %s", name, other)
		}
		seen[sum] = name
	}
	return nil
}

func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func writeNotesShot(path string) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Notes", Width: 860, Height: 540, Headless: true,
	})
	if err != nil {
		return err
	}
	w.SetContent(demo.NotesApp(w))
	a.PumpOnce()
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok {
			tv.MousePress(widget.MouseEvent{
				Pos:    paintengine2d.Pt(72, tv.RowHeight+20),
				Button: platform.ButtonRight,
			})
		}
	})
	a.PumpOnce()
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}

func writeWidgetsCloseup(path string) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Controls", Width: 520, Height: 520, Headless: true,
	})
	if err != nil {
		return err
	}
	primary := widgets.NewButton("Save changes", nil)
	primary.Primary = true
	field := widgets.NewTextField("Search toolkit…", "Placeholder", nil)
	combo := widgets.NewComboBox([]string{"Dark", "Light", "High contrast"}, 0, nil)
	radios := widgets.NewRadioGroup([]string{"UTF-8", "Latin-1"}, 0, nil)
	bar := widgets.NewProgressBar(0.68)
	spin := widgets.NewNumberField(0, 24, 14, 1, nil)
	w.SetContent(widgets.NewPanel("Themed controls",
		widgets.NewRow(primary, widgets.NewButton("Cancel", nil)).WithGap(10),
		field,
		combo,
		spin,
		widgets.NewSlider(0, 100, 72, nil),
		bar,
		radios,
		widgets.NewCheckbox("Remember window size", true, nil),
		widgets.NewCheckbox("Show hidden files", false, nil),
	))
	a.PumpOnce()
	w.RequestFocus(field)
	field.SetSelection(7, 14) // "toolkit"
	primary.MouseEnter()
	a.PumpOnce()
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}

func writeToolbarShot(path string) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Toolbar", Width: 640, Height: 220, Headless: true,
	})
	if err != nil {
		return err
	}
	bar := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconNew, "New", nil),
		widgets.ToolIconBtn(style.IconOpen, "Open", nil),
		widgets.ToolIconBtn(style.IconSave, "Save", nil),
		widgets.ToolDivider(),
		widgets.ToolIconBtn(style.IconCut, "", nil),
		widgets.ToolIconBtn(style.IconCopy, "", nil),
		widgets.ToolIconBtn(style.IconPaste, "", nil),
		widgets.ToolDivider(),
		widgets.ToolToggle("Snap", true, nil),
		widgets.ToolText("Inspect", nil),
	)
	w.SetContent(widgets.NewColumn(
		widgets.NewTitleBar("Project", "toolbar  ·  icon and text"),
		bar,
		widgets.NewPad(16, widgets.NewColumn(
			widgets.NewLabel("Stock ToolBar — icon buttons, text, separators, toggle."),
			widgets.NewProgressBar(0.55),
		).WithGap(10)),
	).WithGap(0))
	a.PumpOnce()
	bar.Hover(2)
	a.PumpOnce()
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}
