// Command richtext is a small rich-text editor: the format bar over a
// document, opened from and saved to HTML, exported as plain text.
//
//	go run ./examples/uitoolkit-sample-richtext [file.html]
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/showcase"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write richtext.png")
	flag.Parse()
	path := flag.Arg(0)

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{Title: "Rich text", Width: 760, Height: 560, MinWidth: 420, MinHeight: 300})
	if err != nil {
		log.Fatal(err)
	}
	ed := uitoolkit.NewRichTextHTML(showcase.RichTextSample)
	ed.SetAccessibleName("Document")
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			ed.SetHTML(string(b))
		}
	}
	status := widgets.NewStatusBar("Ready.")
	save := func() {
		if path == "" {
			path = "document.html"
		}
		if err := os.WriteFile(path, []byte(ed.HTML()), 0o644); err != nil {
			status.Set(0, err.Error())
			return
		}
		status.Set(0, "Saved "+path)
	}
	export := func() {
		out := strings.TrimSuffix(path, ".html") + ".txt"
		if path == "" {
			out = "document.txt"
		}
		if err := os.WriteFile(out, []byte(ed.PlainText()+"\n"), 0o644); err != nil {
			status.Set(0, err.Error())
			return
		}
		status.Set(0, "Exported "+out)
	}
	ed.OnLink = func(href string) { status.Set(0, "Link: "+href) }
	menu := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemAccel("&Save", "Ctrl+S", save),
			widgets.Item("&Export as text", export),
			widgets.Sep(),
			widgets.ItemAccel("&Quit", "Ctrl+Q", a.Quit),
		),
		widgets.NewMenu("&Edit",
			widgets.ItemAccel("&Undo", "Ctrl+Z", ed.Undo),
			widgets.ItemAccel("&Redo", "Ctrl+Shift+Z", ed.Redo),
			widgets.Sep(),
			widgets.ItemAccel("Cu&t", "Ctrl+X", ed.Cut),
			widgets.ItemAccel("&Copy", "Ctrl+C", ed.Copy),
			widgets.ItemAccel("&Paste", "Ctrl+V", func() { ed.Paste(false) }),
		),
	)
	body := widgets.NewColumn(uitoolkit.NewRichTextBar(ed), ed).WithGap(6).WithPad(8)
	body.AddFlex(ed, 1)
	root := widgets.NewColumn(menu, body, status)
	root.AddFlex(body, 1)
	win.SetContent(root)
	win.RequestFocus(ed)
	if *headless {
		a.PumpOnce()
		if err := win.WritePNG("richtext.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote richtext.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
