package widgets_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// wizardShotSample is a four-step account set-up, shown on its second
// step with a validation message up.
func wizardShotSample() (*widgets.Wizard, *widgets.TextField) {
	name := widgets.NewTextField("Ada Lovelace", "Full name", nil)
	email := widgets.NewTextField("ada.example.com", "Address", nil)
	form := widgets.NewForm()
	form.AddRow("Name", name)
	form.AddRow("Email", email)
	w := widgets.NewWizard("New account",
		&widgets.WizardPage{Title: "Welcome", Subtitle: "This assistant sets up a mail account.", Content: widgets.NewLabel("Press Next to begin.")},
		&widgets.WizardPage{Title: "Your account", Subtitle: "The name and address people see.", Content: form,
			Validate: func() error {
				if !strings.Contains(email.Text, "@") {
					return errors.New("The address needs an @ between the name and the domain.")
				}
				return nil
			}},
		&widgets.WizardPage{Title: "Connection", Subtitle: "How to reach the server.", Optional: true,
			Content: widgets.NewCheckbox("Connect through a proxy", false, nil)},
		&widgets.WizardPage{Title: "Summary", Subtitle: "Everything is ready.", Content: widgets.NewLabel("Press Finish to create the account.")},
	)
	w.OnHelp = func(int) {}
	return w, email
}

func TestWizardShots(t *testing.T) {
	for _, pack := range breadthLooks {
		for _, scale := range []float32{1, 1.75} {
			if testing.Short() && scale != 1 {
				continue
			}
			name := fmt.Sprintf("wizard-%s@%g", pack, scale)
			t.Run(name, func(t *testing.T) {
				var wiz *widgets.Wizard
				a, win := shotWindow(t, pack, scale, 640, 420, func() widget.Component {
					wiz, _ = wizardShotSample()
					return wiz
				})
				defer win.Close()
				wiz.Next()
				wiz.Next() // refused: the message shows
				a.PumpOnce()
				a.PumpOnce()
				writeShot(t, win, name)
				// The other layout in the same look, with the list of steps.
				if wiz.Style == widgets.WizardAuto {
					other := widgets.WizardModern
					if wiz.ShownStyle() == widgets.WizardModern {
						other = widgets.WizardClassic
					}
					wiz.Style, wiz.Steps = other, widgets.StepsShown
					wiz.RequestLayout()
					a.PumpOnce()
					a.PumpOnce()
					writeShot(t, win, name+"-other")
				}
			})
		}
	}
}
