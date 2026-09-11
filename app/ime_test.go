package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestWindowDispatchesIME(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 320, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	tf := widgets.NewTextField("", "", nil)
	w.SetContent(tf)
	a.PumpOnce()
	w.RequestFocus(tf)
	w.dispatch(platform.Event{Kind: platform.EventIMEPreedit, Text: "か", IMECaret: 1})
	if tf.Preedit() != "か" {
		t.Fatalf("preedit %q", tf.Preedit())
	}
	w.dispatch(platform.Event{Kind: platform.EventIMECommit, Text: "か"})
	if tf.Text != "か" || tf.Preedit() != "" {
		t.Fatalf("commit %q pre=%q", tf.Text, tf.Preedit())
	}
	w.dispatch(platform.Event{Kind: platform.EventIMEPreedit, Text: "x", IMECaret: 1})
	w.dispatch(platform.Event{Kind: platform.EventIMECancel})
	if tf.Preedit() != "" {
		t.Fatal("cancel")
	}
}
