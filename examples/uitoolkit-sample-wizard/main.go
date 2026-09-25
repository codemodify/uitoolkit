// Command wizard sets up an imaginary mail account in a wizard: a page
// that must be filled in, one that checks what was typed, an optional
// page, and one that only a choice brings in. Finish prints the result.
//
//	go run ./examples/uitoolkit-sample-wizard
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write wizard.png")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{Title: windowTitle("New account"), Width: 680, Height: 460, MinWidth: 520, MinHeight: 360})
	if err != nil {
		log.Fatal(err)
	}
	name := widgets.NewTextField("", "Full name", nil)
	email := widgets.NewTextField("", "name@example.com", nil)
	form := widgets.NewForm()
	form.AddRow("Name", name)
	form.AddRow("Email", email)
	proxy := widgets.NewCheckbox("Connect through a proxy", false, nil)
	host := widgets.NewTextField("", "proxy.example.com:3128", nil)
	host.SetAccessibleName("Proxy")

	var wiz *widgets.Wizard
	name.OnChange = func(string) { wiz.UpdateButtons() }
	wiz = uitoolkit.NewWizard("New account",
		&widgets.WizardPage{Title: "Welcome", Subtitle: "This assistant sets up a mail account.",
			Content: widgets.NewLabel("Press Next to begin, or Escape to give up.")},
		&widgets.WizardPage{Title: "Your account", Subtitle: "The name and address people see.", Content: form,
			Complete: func() bool { return strings.TrimSpace(name.Text) != "" },
			Validate: func() error {
				if !strings.Contains(email.Text, "@") {
					return errors.New("The address needs an @ between the name and the domain.")
				}
				return nil
			}},
		&widgets.WizardPage{Title: "Connection", Subtitle: "How to reach the server.", Optional: true, Content: proxy},
		&widgets.WizardPage{Title: "Proxy", Subtitle: "The proxy to go through.", Content: host,
			Skip: func() bool { return !proxy.Checked }},
		&widgets.WizardPage{Title: "Summary", Subtitle: "Everything is ready.",
			Content: widgets.NewLabel("Press Finish to create the account.")},
	)
	wiz.OnFinish = func() {
		fmt.Printf("account: %s <%s>, proxy %q\n", name.Text, email.Text, host.Text)
		a.Quit()
	}
	wiz.OnCancel = func() bool { a.Quit(); return true }
	win.SetContent(wiz)
	if *headless {
		a.PumpOnce()
		if err := win.WritePNG("wizard.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote wizard.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

// windowTitle is what a window of this sample is called on the desktop:
// the sample's own name for the window under the toolkit's prefix, so
// that a desktop with several samples open says which toolkit they
// belong to. Every title this sample sets goes through here, so one
// computed while the app runs carries the prefix too.
func windowTitle(name string) string { return "uitoolkit - " + name }
