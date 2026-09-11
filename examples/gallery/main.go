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
	"github.com/codemodify/uitoolkit/internal/mail"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	shot := flag.String("screenshot", "", "write PNG gallery into this directory and exit")
	headless := flag.Bool("headless", false, "paint offscreen (no X11/Wayland)")
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
	return demo.Gallery(a, win, light)
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
		{
			name: "gallery-accordion.png",
			look: style.LightLook(),
			setup: func(a *app.Application, w *app.Window) {
				selectGalleryTab(w, 4)
				openGalleryAccordion(w, 1)
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
	if err := writeTextAreaShot(filepath.Join(dir, "gallery-textarea.png")); err != nil {
		return err
	}
	if err := writeInspectorShot(filepath.Join(dir, "inspector.png")); err != nil {
		return err
	}
	if err := writeFilesShot(filepath.Join(dir, "files.png")); err != nil {
		return err
	}
	if err := writeFontsShot(filepath.Join(dir, "fonts.png")); err != nil {
		return err
	}
	if err := mail.WriteScreenshots(dir); err != nil {
		return err
	}
	if err := writeCompareThumbs(dir); err != nil {
		return err
	}
	return verifyDistinctPNGs(dir, screenshotNames)
}

var screenshotNames = []string{
	"gallery-dark.png", "gallery-light.png", "gallery-dialog.png",
	"gallery-scroll.png", "gallery-menu.png", "gallery-tree.png",
	"gallery-toolbar.png", "gallery-combo.png", "gallery-message.png",
	"gallery-table.png", "gallery-file.png", "gallery-tooltip.png",
	"gallery-textarea.png", "gallery-accordion.png",
	"notes.png", "inspector.png", "files.png", "widgets.png", "fonts.png",
	"mail-dark.png", "mail-light.png", "mail-classic.png", "mail-compose.png", "mail-prefs.png",
	"mail-cards.png", "mail-compact.png", "mail-filters.png",
	"mail-empty.png", "mail-account.png", "mail-smart.png",
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

func openGalleryAccordion(w *app.Window, i int) {
	widget.Walk(w.Content(), func(c widget.Component) {
		if acc, ok := c.(*widgets.Accordion); ok {
			secs := acc.Sections()
			if i >= 0 && i < len(secs) {
				secs[i].SetExpanded(true)
			}
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
		Title: "Controls", Width: 520, Height: 620, Headless: true,
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
	area := widgets.NewTextArea("Remembered draft\nSecond line.", "Notes", nil)
	area.MinRows = 2
	logField := widgets.NewMonoTextField("/var/log/uitoolkit.log", "log path", nil)
	w.SetContent(widgets.NewPanel("Themed controls",
		widgets.NewRow(primary, widgets.NewButton("Cancel", nil)).WithGap(10),
		field,
		logField,
		combo,
		spin,
		widgets.NewSlider(0, 100, 72, nil),
		bar,
		radios,
		widgets.NewSwitch("Dark chrome", true, nil),
		widgets.NewCheckbox("Remember window size", true, nil),
		widgets.NewCheckbox("Show hidden files", false, nil),
		widgets.NewSeparator(),
		area,
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

func writeTextAreaShot(path string) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "TextArea", Width: 560, Height: 380, Headless: true,
	})
	if err != nil {
		return err
	}
	area := widgets.NewTextArea(
		"Ship notes for v0.1.7.\nTextArea wraps this paragraph onto the next visual line and scrolls when the body is taller than the field.\nReturn inserts a newline.",
		"Write something…",
		nil,
	)
	area.MinRows = 8
	w.SetContent(widgets.NewPanel("TextArea",
		widgets.NewLabel("Multi-line edit  ·  wrap  ·  selection"),
		widgets.NewRow(
			widgets.NewSwitch("Word wrap", true, nil),
			widgets.NewVSeparator(),
			widgets.NewLabel("Ln 2, Col 8"),
		).WithGap(10),
		area,
	))
	a.PumpOnce()
	w.RequestFocus(area)
	area.SetSelection(0, 21) // "Ship notes for v0.1.7"
	area.SetCaretBlink(true)
	a.PumpOnce()
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}

func writeFontsShot(path string) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Fonts", Width: 640, Height: 420, Headless: true,
	})
	if err != nil {
		return err
	}
	uiField := widgets.NewTextField("Search toolkit…", "Titillium Web", nil)
	monoField := widgets.NewMonoTextField("/work/projects/uitoolkit", "path", nil)
	code := widgets.NewMonoTextArea(
		"func main() {\n    fmt.Println(\"JetBrains Mono\")\n}\n",
		"code",
		nil,
	)
	code.MinRows = 4
	code.Wrap = false
	uiLbl := widgets.NewLabel("UI role  ·  Titillium Web")
	monoLbl := widgets.NewLabel("Mono role  ·  JetBrains Mono")
	monoLbl.Mono = true
	w.SetContent(widgets.NewPanel("LookAndFeel typefaces",
		uiLbl,
		uiField,
		widgets.NewSeparator(),
		monoLbl,
		monoField,
		code,
	))
	a.PumpOnce()
	w.RequestFocus(code)
	code.SetSelection(0, 11) // "func main()"
	a.PumpOnce()
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}

func writeFilesShot(path string) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Files", Width: 1040, Height: 680, Headless: true,
	})
	if err != nil {
		return err
	}
	w.SetContent(demo.FilesApp(w))
	a.PumpOnce()
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TabView); ok {
			tv.Select(0)
		}
		if table, ok := c.(*widgets.TableView); ok && len(table.Columns) >= 3 {
			table.Selected = 0
			if table.OnSelect != nil {
				table.OnSelect(0)
			}
		}
	})
	a.PumpOnce()
	openGalleryMenu(w, 0)
	a.PumpOnce()
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}

func writeInspectorShot(path string) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Inspector", Width: 860, Height: 580, Headless: true,
	})
	if err != nil {
		return err
	}
	w.SetContent(demo.InspectorApp(w))
	a.PumpOnce()
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TabView); ok {
			tv.Select(1)
		}
	})
	a.PumpOnce()
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok {
			tv.Selected = 2
			if tv.OnSelect != nil {
				tv.OnSelect(2)
			}
			tv.Invalidate()
		}
		if ta, ok := c.(*widgets.TextArea); ok {
			w.RequestFocus(ta)
			ta.SetSelection(0, 14)
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
