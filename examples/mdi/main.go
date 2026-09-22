// Command mdi is a multiple-document editor: rich-text documents in
// windows inside the main window, with a Window menu that arranges and
// lists them and asks before closing an edited one.
//
//	go run ./examples/mdi
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write mdi.png")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{Title: "Documents", Width: 960, Height: 640, MinWidth: 480, MinHeight: 320})
	if err != nil {
		log.Fatal(err)
	}
	area := uitoolkit.NewMDIArea()
	status := widgets.NewStatusBar("Ctrl+Tab switches documents, Ctrl+F4 closes one.")
	n := 0
	newDoc := func() {
		n++
		title := fmt.Sprintf("Document %d", n)
		ed := uitoolkit.NewRichText("Type here")
		ed.SetAccessibleName(title)
		edited := false
		ed.OnChange = func() { edited = true }
		body := widgets.NewColumn(uitoolkit.NewRichTextBar(ed), ed).WithGap(4).WithPad(4)
		body.AddFlex(ed, 1)
		w := area.AddWindow(title, body)
		w.SetInitialFocus(ed)
		// An edited document asks first; the answer closes it or not.
		w.OnCloseRequest = func() bool {
			if !edited {
				return true
			}
			widgets.Confirm(area, "Close "+title, "Close it without saving?", func(ok bool) {
				if ok {
					w.Dismiss()
				}
			})
			return false
		}
	}
	for range 3 {
		newDoc()
	}
	area.OnActivate = func(w *widgets.MDIWindow) { status.Set(0, w.Title()) }
	menu := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemAccel("&New", "Ctrl+N", newDoc),
			widgets.ItemAccel("&Close", "Ctrl+F4", func() { area.CloseActive() }),
			widgets.Sep(),
			widgets.ItemAccel("&Quit", "Ctrl+Q", func() {
				if area.CloseAll() {
					a.Quit()
				}
			}),
		),
		area.WindowMenu(),
	)
	root := widgets.NewColumn(menu, area, status)
	root.AddFlex(area, 1)
	win.SetContent(root)
	if *headless {
		a.PumpOnce()
		if err := win.WritePNG("mdi.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote mdi.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
