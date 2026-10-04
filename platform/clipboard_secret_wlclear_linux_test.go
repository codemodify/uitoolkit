//go:build linux && cgo

package platform

import (
	"os"
	"testing"
)

// A clear gives up the selections this process owns. It used to copy an empty
// string onto them, which is still a copy: it takes the primary selection a
// middle click pastes, so a secret's timeout threw away whatever the person
// had selected in another program since — and left this process owning a
// selection it had nothing to serve.
//
// Under the compositor rig (tools/test-display.sh), because a selection needs
// something to hold it. The connection is retained for the length of each
// test: a compositor gives the selection only to a client with a window and
// the focus, and a `go test` process has neither, so the source it has just
// made is cancelled a few milliseconds later and the connection closes itself.
// What is under test here is what the *clear* does to the sources this process
// holds at the moment it is called, which is a question that can be asked
// before any of that.

func wlHeld(t *testing.T) *wlConn {
	t.Helper()
	if os.Getenv("WAYLAND_DISPLAY") == "" || !waylandProbe() {
		t.Skip("no compositor")
	}
	c, err := wlRetain()
	if err != nil || c == nil {
		t.Skip("no Wayland connection: ", err)
	}
	t.Cleanup(c.release)
	return c
}

// sources is what this process owns, from the backend's own state.
func sources(c *wlConn) (clip, prim, primCached bool) {
	wlMu.Lock()
	defer wlMu.Unlock()
	return c.dataSrc != nil, c.primSrc != nil, c.prim.owned
}

// Clearing a secret leaves no primary selection of this process's behind.
//
// The old clear made a *new* primary source offering an empty string, which
// is how the person's own selection went missing thirty seconds after a
// password manager was used.
func TestAWaylandSecretClearDoesNotTakeThePrimarySelection(t *testing.T) {
	c := wlHeld(t)
	t.Cleanup(ClipboardClear)

	// Something ordinary first, so this process has both selections — the
	// state a person is in when a password manager is reached for.
	ClipboardSet("what the person had selected")
	if clip, prim, _ := sources(c); !clip || !prim {
		t.Skip("an ordinary copy did not take both selections")
	}
	sc := ClipboardSetSecret([]byte("the-passphrase"), -1)
	// A secret copy gives the primary selection up on purpose: a middle
	// click anywhere would otherwise paste it.
	if _, prim, _ := sources(c); prim {
		t.Error("a secret copy left this process owning the primary selection")
	}

	sc.Clear()

	clip, prim, cached := sources(c)
	if clip {
		t.Error("the clipboard selection was not given up: this process still serves it")
	}
	if prim {
		t.Error("the clear took the primary selection with an empty offer")
	}
	if cached {
		t.Error("the primary cache still claims this process owns the selection")
	}
}

// An ordinary clear gives up both, because this process owns both: a vault
// that has locked is saying take everything back.
func TestAWaylandClearGivesUpBothSelectionsItOwns(t *testing.T) {
	c := wlHeld(t)
	t.Cleanup(ClipboardClear)

	ClipboardSet("a line of text")
	if clip, prim, _ := sources(c); !clip || !prim {
		t.Skip("an ordinary copy did not take both selections")
	}

	ClipboardClear()

	clip, prim, cached := sources(c)
	if clip || prim || cached {
		t.Errorf("after a clear: clipboard=%v primary=%v cached=%v, want none", clip, prim, cached)
	}
}

// Clearing when this process owns nothing gives nothing up. The guard is the
// whole protection: set_selection with a null source unsets whichever client's
// selection it is, so an unguarded clear would reach across to another
// program's.
func TestAWaylandClearWithNothingOwnedGivesNothingUp(t *testing.T) {
	_ = wlHeld(t)
	ClipboardClear()
	if gave := wlClipClear(); gave {
		t.Error("a clear with nothing owned reported giving a selection up")
	}
}

// A secret copy gives the primary selection up, and the cache says so.
//
// It recorded an empty string as this process's own instead, so a read of the
// primary selection was answered "" from that cache rather than falling
// through to whichever client actually held it: after a password manager had
// been used, a middle-click paste read nothing rather than what was selected.
func TestASecretCopyDoesNotLeaveThePrimarySelectionOwned(t *testing.T) {
	c := wlHeld(t)
	t.Cleanup(ClipboardClear)

	ClipboardSet("what the person had selected")
	if _, prim, _ := sources(c); !prim {
		t.Skip("an ordinary copy did not take the primary selection")
	}
	ClipboardSetSecret([]byte("the-passphrase"), -1)

	if _, _, cached := sources(c); cached {
		t.Error("the primary cache still claims this process owns the selection")
	}
	wlMu.Lock()
	got, ok := c.prim.get()
	wlMu.Unlock()
	if ok {
		t.Errorf("a read of the primary selection is answered %q from this process's cache", got)
	}
}
