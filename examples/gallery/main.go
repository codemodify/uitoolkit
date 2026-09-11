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

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/internal/demo"
	"github.com/codemodify/uitoolkit/layout"
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
		Title: "uitoolkit gallery", Width: 960, Height: 720, MinWidth: 640, MinHeight: 420,
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

	volume := widgets.NewLabel("Volume  60%")
	slider := widgets.NewSlider(0, 100, 60, func(v float32) {
		volume.SetText(fmt.Sprintf("Volume  %d%%", int(v+0.5)))
	})

	name := widgets.NewTextField("Ada Lovelace", "Display name", nil)
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
		var overlay *widgets.Overlay
		closeBtn := widgets.NewButton("Close", func() { win.SetOverlay(nil) })
		closeBtn.Primary = true
		card := widgets.DialogCard(
			"About uitoolkit",
			"Pure Go desktop UI. Paints only with paintengine2d.",
			closeBtn,
		)
		overlay = widgets.NewOverlay(card)
		overlay.OnClose = func() { win.SetOverlay(nil) }
		win.SetOverlay(overlay)
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
		widgets.NewRow(primary, plain).WithGap(10),
		widgets.NewRow(disabled, clickLbl).WithGap(10),
		widgets.NewRow(about, other, themeBtn).WithGap(10),
	)

	fields := widgets.NewPanel("Fields",
		widgets.NewLabel("Name"),
		name,
		volume,
		slider,
		widgets.NewLabel("Options"),
		checks,
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
	treePane := widgets.NewColumn(treeSel, tree).WithGap(6)
	treePane.AddFlex(tree, 1)

	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Scroll", Content: widgets.NewPanel("ScrollView", scroll)},
		widgets.Tab{Title: "List", Content: widgets.NewPanel("ListView", listPane)},
		widgets.Tab{Title: "Tree", Content: widgets.NewPanel("TreeView", treePane)},
	)
	tabs.OnChange = func(i int) {
		names := []string{"Scroll", "List", "Tree"}
		if i >= 0 && i < len(names) {
			status.Set(0, "Tab: "+names[i])
		}
	}

	menubar := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.Item("About…", func() { about.OnClick() }),
			widgets.Sep(),
			widgets.Item("Quit", func() { a.Quit() }),
		),
		widgets.NewMenu("&Edit",
			widgets.Item("Copy", func() {
				if s := name.SelectedText(); s != "" {
					platform.ClipboardSet(s)
					status.Set(0, "Copied")
				}
			}),
			widgets.Item("Paste", func() {
				name.SetText(name.Text + platform.ClipboardGet())
				status.Set(0, "Pasted")
			}),
		),
		widgets.NewMenu("&View",
			widgets.Item("Dark theme", func() {
				a.SetLook(style.DarkLook())
				win.SetContent(buildGallery(a, win, false))
			}),
			widgets.Item("Light theme", func() {
				a.SetLook(style.LightLook())
				win.SetContent(buildGallery(a, win, true))
			}),
		),
	)

	left := widgets.NewColumn(
		widgets.NewTitle("uitoolkit"),
		widgets.NewLabel("paintengine2d  ·  v"+uitoolkit.Version),
		buttons,
		fields,
	).WithGap(10).WithPad(12)
	left.AddFlex(buttons, 0)
	left.AddFlex(fields, 1)

	right := widgets.NewPad(10, tabs)

	split := widgets.NewSplitter(true, left, right)
	split.Ratio = 0.46

	header := widgets.NewRow(
		widgets.NewTitle("Widget gallery"),
		widgets.NewLabel("menus  ·  tabs  ·  tree  ·  focus"),
	).WithGap(16).WithPad(12).WithAlign(layout.AlignCenter)

	root := widgets.NewColumn(menubar, header, split, status).WithGap(0)
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
				closeBtn := widgets.NewButton("Close", func() {})
				closeBtn.Primary = true
				card := widgets.DialogCard(
					"About uitoolkit",
					"Pure Go desktop UI. Paints only with paintengine2d.",
					closeBtn,
				)
				w.SetOverlay(widgets.NewOverlay(card))
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
	}
	for _, s := range shots {
		a := uitoolkit.New(uitoolkit.Options{Look: s.look, Headless: true})
		w, err := a.NewWindow(platform.WindowOptions{
			Title: "uitoolkit gallery", Width: 960, Height: 720, Headless: true,
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
	return verifyDistinctPNGs(dir, screenshotNames)
}

var screenshotNames = []string{
	"gallery-dark.png", "gallery-light.png", "gallery-dialog.png",
	"gallery-scroll.png", "gallery-menu.png", "gallery-tree.png",
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
		if lv, ok := c.(*widgets.ListView); ok {
			lv.MousePress(widget.MouseEvent{
				Pos:    paintengine2d.Pt(72, 16),
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
		Title: "Controls", Width: 520, Height: 360, Headless: true,
	})
	if err != nil {
		return err
	}
	primary := widgets.NewButton("Save changes", nil)
	primary.Primary = true
	field := widgets.NewTextField("Search toolkit…", "Placeholder", nil)
	w.SetContent(widgets.NewPanel("Themed controls",
		widgets.NewRow(primary, widgets.NewButton("Cancel", nil)).WithGap(10),
		field,
		widgets.NewSlider(0, 100, 72, nil),
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
