//go:build linux && cgo

package platform

import (
	"os"
	"os/exec"
	"testing"
	"time"
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

// wlPrimaryOwner makes another program hold the primary selection, and reports
// whether it took. wl-copy is a second client, which is the only way to have a
// selection that is not this process's — the same reason the X11 tests need
// tools/e2e/clipown.
func wlPrimaryOwner(t *testing.T, text string) {
	t.Helper()
	if _, err := exec.LookPath("wl-copy"); err != nil {
		t.Skip("wl-clipboard is not installed")
	}
	// --foreground, started and killed on the way out, like tools/e2e/clipown:
	// the default wl-copy forks a server that inherits the pipes, so waiting
	// for the command to finish waits for the selection to be given up again.
	cmd := exec.Command("wl-copy", "--foreground", "--primary", text)
	if err := cmd.Start(); err != nil {
		t.Skip("wl-copy did not start: ", err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
	// And it has to reach the compositor before anything is asked of it.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if got, ok := wlPrimaryText(t); ok && got == text {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Skip("the compositor did not give wl-copy the primary selection")
}

// wlPrimaryText is what the primary selection holds, asked of the compositor
// through another program rather than of this process's own cache.
func wlPrimaryText(t *testing.T) (string, bool) {
	t.Helper()
	out, err := exec.Command("wl-paste", "--primary", "--no-newline").Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

// A secret copy leaves another program's primary selection alone.
//
// It gives the primary selection up on purpose — a middle click anywhere would
// otherwise paste the passphrase — but it did so with set_selection(null)
// whenever a primary device existed at all, and that unsets whichever client's
// selection it is. So copying a password threw away whatever the person had
// selected somewhere else, and the middle click after it pasted nothing. The
// copy button's click supplies a fresh serial, so a compositor that checks one
// accepts it.
//
// wlClipClear has guarded this since the same bug was found there; the copy
// path never did.
func TestAWaylandSecretCopyLeavesAnotherProgramsPrimarySelectionAlone(t *testing.T) {
	c := wlHeld(t)
	t.Cleanup(ClipboardClear)

	const theirs = "what the person had selected in another program"
	wlPrimaryOwner(t, theirs)

	// A window, and a serial from it. set_selection carries the serial of the
	// input event that justifies it, and a compositor refuses one it did not
	// issue — so a windowless test process, whose serial is 0, cannot reach
	// this bug at all: every request it makes is dropped, including the wrong
	// one. The application that hits it has a serial because the person
	// clicked a copy button. This is the smallest version of that.
	wlSerialFromAWindow(t)

	// This process owns no primary source — it has copied nothing — so there
	// is nothing of its own to give up, and anything it gives up is somebody
	// else's.
	if _, prim, _ := sources(c); prim {
		t.Fatal("this process already owns the primary selection; the test would prove nothing")
	}

	ClipboardSetSecret([]byte("the-passphrase"), -1)
	if clip, _, _ := sources(c); !clip {
		t.Skip("the secret copy did not take the clipboard")
	}

	got, ok := wlPrimaryText(t)
	if !ok || got != theirs {
		t.Errorf("after a secret copy the primary selection reads %q (ok=%v), want the other program's %q", got, ok, theirs)
	}
}

// wlSerialFromAWindow gives the connection a serial the compositor will
// accept, by making a window and letting it be entered. Nothing is injected:
// in a nested compositor the keyboard or pointer enter arrives on its own
// within a frame or two.
func wlSerialFromAWindow(t *testing.T) {
	t.Helper()
	raw, err := (wlBackend{}).NewSurface(WindowOptions{Title: "uitk-serial", Width: 300, Height: 200})
	if err != nil {
		t.Skip("no surface: ", err)
	}
	s, ok := raw.(*wlSurface)
	if !ok {
		t.Skip("not a Wayland surface")
	}
	t.Cleanup(func() { s.Close() })
	s.Present(nil)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s.Poll()
		wlMu.Lock()
		ser := s.conn.serial
		wlMu.Unlock()
		if ser != 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Skip("the window never got an input serial; the compositor would refuse every selection request")
}
