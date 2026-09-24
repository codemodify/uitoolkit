package widgets_test

import (
	"fmt"
	"testing"

	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// dialogCardLooks is the spread the dialog card is judged in: the era the
// report came from (win8), the classic desktops, the Unix workstations,
// the Macs, today's Linux and web looks, and a skin pack.
var dialogCardLooks = []string{
	"win8", "win95", "win2000", "motif", "cde", "platinum", "aqua",
	"breeze6", "adwaita48", "fluent", "material3", "tahoe", "deck",
}

func TestDialogCardShots(t *testing.T) {
	entries := []widgets.FileInfo{
		{Name: "Documents", Dir: true},
		{Name: "Pictures", Dir: true},
		{Name: "notes.txt", Size: 1240},
		{Name: "report.pdf", Size: 94210},
		{Name: "budget.csv", Size: 5120},
	}
	for _, pack := range dialogCardLooks {
		for _, scale := range []float32{1, 1.75} {
			if testing.Short() && scale != 1 {
				continue
			}
			name := fmt.Sprintf("filedialog-%s@%g", pack, scale)
			t.Run(name, func(t *testing.T) {
				a, win := shotWindow(t, pack, scale, 820, 620, func() widget.Component {
					fd := widgets.NewFileDialog(widgets.FileDialogOptions{
						Title: "Open file", Path: "/home/ada", Filter: "*.txt *.md",
						Entries: entries, OnNavigate: func(string) []widgets.FileInfo { return entries },
					})
					return fd.Overlay()
				})
				defer win.Close()
				a.PumpOnce()
				writeShot(t, win, name)
			})
			name = fmt.Sprintf("messagebox-%s@%g", pack, scale)
			t.Run(name, func(t *testing.T) {
				a, win := shotWindow(t, pack, scale, 820, 620, func() widget.Component {
					mb := widgets.NewMessageBox(widgets.MessageBoxOptions{
						Title: "Discard changes?", Kind: widgets.MessageWarning,
						Message:  "The document has unsaved changes. They are lost if you close it now.",
						Buttons:  widgets.ButtonsYesNoCancel,
						OnResult: func(widgets.MessageResult) {},
					})
					return mb.Overlay()
				})
				defer win.Close()
				a.PumpOnce()
				writeShot(t, win, name)
			})
		}
	}
}

// TestPlainPanelShots shows the panels that are NOT raised — the tour's
// pages, Settings' sections — are untouched by the card change: they are
// still the look's group box, legend and all.
func TestPlainPanelShots(t *testing.T) {
	for _, pack := range dialogCardLooks {
		name := fmt.Sprintf("plainpanel-%s@1", pack)
		t.Run(name, func(t *testing.T) {
			_, win := shotWindow(t, pack, 1, 560, 320, func() widget.Component {
				a := widgets.NewPanel("Appearance",
					widgets.NewCheckbox("Animate windows", true, nil),
					widgets.NewCheckbox("Show shadows", false, nil))
				b := widgets.NewPanel("Behaviour", widgets.NewLabel("Double-click speed"))
				return widgets.NewColumn(a, b).WithGap(12).WithPad(12)
			})
			defer win.Close()
			writeShot(t, win, name)
		})
	}
}
