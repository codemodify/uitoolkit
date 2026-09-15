package widgets

import (
	"testing"

	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// altHost is a test host that reports the Alt key.
type altHost struct {
	host
	alt bool
}

func (h *altHost) AltHeld() bool { return h.alt }

// Mnemonic underlines follow the look: always in Windows 95, while Alt is
// held or the keyboard drives the menu in XP and Plasma, never on the Mac.
func TestMnemonicCuesFollowTheLook(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	look := func(name string) style.LookAndFeel {
		p, ok := style.LoadTheme(name)
		if !ok {
			t.Fatalf("no pack %s", name)
		}
		return p.Look()
	}
	for _, c := range []struct {
		pack                   string
		idle, alt, keyboardNav bool
	}{
		{"win95", true, true, true},
		{"win2000", false, true, true},
		{"luna", false, true, true},
		{"breeze", false, true, true},
		{"aqua", false, false, false},
		{"system7", false, false, false},
	} {
		h := &altHost{}
		bar := NewMenuBar(NewMenu("&File"))
		bar.SetLook(look(c.pack))
		bar.SetHost(h)
		if got := mnemonicShown(bar, false); got != c.idle {
			t.Errorf("%s idle: %v, want %v", c.pack, got, c.idle)
		}
		h.alt = true
		if got := mnemonicShown(bar, false); got != c.alt {
			t.Errorf("%s with Alt: %v, want %v", c.pack, got, c.alt)
		}
		h.alt = false
		if got := mnemonicShown(bar, true); got != c.keyboardNav {
			t.Errorf("%s keyboard-driven: %v, want %v", c.pack, got, c.keyboardNav)
		}
	}
	if widget.AltHeld(nil) {
		t.Fatal("no component, no Alt")
	}
}
