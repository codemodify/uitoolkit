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
		Title: "uitoolkit gallery", Width: 960, Height: 680, MinWidth: 640, MinHeight: 420,
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
	status := widgets.NewLabel("Ready.")
	status.Color = win.Look().Palette().TextMuted

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
		status.SetText("Secondary clicked")
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

	other := widgets.NewButton("New window", func() {
		w2, err := a.NewWindow(platform.WindowOptions{Title: "Second window", Width: 420, Height: 280})
		if err != nil {
			status.SetText(err.Error())
			return
		}
		w2.SetContent(widgets.NewPanel("Dialog window",
			widgets.NewLabel("This is a second OS window."),
			widgets.NewButton("Close", func() { w2.Close() }),
		))
	})

	themeBtn := widgets.NewButton("Toggle theme", func() {
		if win.Look().Name() == "dark" {
			a.SetLook(style.LightLook())
		} else {
			a.SetLook(style.DarkLook())
		}
		win.SetContent(buildGallery(a, win, win.Look().Name() == "light"))
		status.SetText("Theme: " + win.Look().Name())
	})

	buttons := widgets.NewPanel("Buttons",
		widgets.NewRow(primary, plain, disabled).WithGap(10),
		clickLbl,
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
	listPane := widgets.NewColumn(selected, list).WithGap(6)
	listPane.AddFlex(list, 1)

	left := widgets.NewColumn(
		widgets.NewTitle("uitoolkit"),
		widgets.NewLabel("paintengine2d  ·  v"+uitoolkit.Version),
		buttons,
		fields,
	).WithGap(10).WithPad(12)
	left.AddFlex(buttons, 0)
	left.AddFlex(fields, 1)

	right := widgets.NewColumn(
		widgets.NewPanel("ScrollView", scroll),
		widgets.NewPanel("ListView", listPane),
		status,
	).WithGap(10).WithPad(12)
	// give scroll room
	right.AddFlex(right.Children()[0], 1)
	right.AddFlex(right.Children()[1], 1)

	split := widgets.NewSplitter(true, left, right)
	split.Ratio = 0.46

	header := widgets.NewRow(
		widgets.NewTitle("Widget gallery"),
		widgets.NewLabel("Linux X11  ·  damage  ·  focus  ·  themes"),
	).WithGap(16).WithPad(12).WithAlign(layout.AlignCenter)

	root := widgets.NewColumn(header, split).WithGap(0)
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
		{name: "gallery-light.png", look: style.LightLook(), setup: nil},
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
				scrollGallery(w)
				// Wheel over the ScrollView (right column, upper pane).
				w.Inject(platform.Event{
					Kind:   platform.EventScroll,
					Pos:    paintengine2d.Pt(720, 220),
					Scroll: paintengine2d.Pt(0, 80),
				})
			},
		},
	}
	for _, s := range shots {
		a := uitoolkit.New(uitoolkit.Options{Look: s.look, Headless: true})
		w, err := a.NewWindow(platform.WindowOptions{
			Title: "uitoolkit gallery", Width: 960, Height: 680, Headless: true,
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
	return verifyDistinctPNGs(dir, []string{
		"gallery-dark.png", "gallery-light.png", "gallery-dialog.png",
		"gallery-scroll.png", "notes.png", "widgets.png",
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
