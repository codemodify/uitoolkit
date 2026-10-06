//go:build linux && cgo

package platform

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// X11 is *told* when another client takes a selection — SelectionClear — and
// until v0.23.3 nothing listened on the secret's behalf: the bytes the owner
// served were zeroed and the SecretClip was left held, its own copy of the
// passphrase in it. For the rest of its time a paste into a SecretField took
// that copy in preference to the clipboard, so a passphrase copied from
// another program pasted as the one copied before it.
//
// It takes a second client, which is what tools/e2e/clipown is: no part of
// this can be reached from inside the process under test. The whole test
// therefore lives behind a display and that helper, and runs under
// tools/test-display.sh.

// clipOwner starts another program that takes CLIPBOARD and serves text on
// it, and returns once the X server has acknowledged the ownership.
func clipOwner(t *testing.T, text string) {
	t.Helper()
	bin, err := filepath.Abs("../tools/e2e/clipown")
	if err != nil {
		t.Skip(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Skip("tools/e2e/clipown is not built (tools/e2e/build.sh)")
	}
	cmd := exec.Command(bin, text, "20")
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Skip("clipown did not start: ", err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || line != "owned\n" {
		t.Skip("clipown did not take CLIPBOARD")
	}
}

// pumpX11 runs the X11 event loop, which is where SelectionClear is read.
func pumpX11(t *testing.T) {
	t.Helper()
	c, err := x11Get()
	if err != nil || c == nil {
		t.Skip("no X11 connection")
	}
	x11Mu.Lock()
	c.drainLocked()
	x11Mu.Unlock()
}

// After another program takes CLIPBOARD, the secret this process was holding
// is gone from it — so the copy here goes too, and a paste reads what was
// actually copied.
func TestAnotherClientTakingTheClipboardEndsTheHeldSecretOnX11(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		// A secret copy takes both selections there, and losing X11's while
		// the Wayland one still serves it is the case that must *not* end the
		// copy — wlStillServesTheSecret. Run this under the x11 pass.
		t.Skip("a Wayland session too: the Wayland selection would still serve it")
	}
	t.Cleanup(ClipboardClear)

	secret := []byte("the-held-passphrase")
	c := ClipboardSetSecret(secret, -1)
	if !ClipboardHoldsSecret() {
		t.Skip("the secret copy did not take")
	}

	const theirs = "what the other program copied"
	clipOwner(t, theirs)

	// The news arrives on the X11 loop and the copy ends on a goroutine of
	// its own, so this waits for it rather than assuming either has run.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		pumpX11(t)
		if !ClipboardHoldsSecret() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if ClipboardHoldsSecret() {
		t.Fatal("the clipboard still claims to hold this process's secret")
	}
	if !c.Cleared() {
		t.Error("the held copy was not ended")
	}
	if !bytes.Equal(secret, make([]byte, len(secret))) {
		t.Errorf("the held bytes were not wiped: %q", secret)
	}
	got := ClipboardGetSecret()
	defer func() {
		for i := range got {
			got[i] = 0
		}
	}()
	if string(got) != theirs {
		t.Errorf("a secret paste read %q, want the other program's copy %q", got, theirs)
	}
}

// Losing PRIMARY does not end a secret. A secret copy gives PRIMARY up on
// purpose (a middle click anywhere would otherwise paste it), so the word
// that it has gone arrives in the ordinary course of things and must not be
// read as the secret having been taken.
func TestLosingPrimaryDoesNotEndTheHeldSecretOnX11(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	t.Cleanup(ClipboardClear)

	// An ordinary copy first, so this process owns PRIMARY to lose.
	ClipboardSet("something selected")
	secret := []byte("still-ours")
	ClipboardSetSecret(secret, -1)
	if !ClipboardHoldsSecret() {
		t.Skip("the secret copy did not take")
	}
	for i := 0; i < 20; i++ {
		pumpX11(t)
		time.Sleep(5 * time.Millisecond)
	}
	if !ClipboardHoldsSecret() {
		t.Error("the held secret ended without another client taking CLIPBOARD")
	}
}

// A loss still on its way is not pinned on the copy just made.
//
// Another program copies, and milliseconds later this one copies a secret. The
// SelectionClear for the first is still in the queue, and the copy's own round
// trip for a fresh selection timestamp is what drains it — so the loss of the
// *previous* copy is handled inside the new copy's own call. The backend only
// ever asked which copy is held, and the new one was already installed, so it
// was the new one that was ended: a password manager that said "cleared" the
// instant the user copied, with the passphrase still on the clipboard.
func TestALossOnItsWayIsNotPinnedOnTheCopyJustMade(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		t.Skip("a Wayland session too: the mirror check would rescue the copy anyway")
	}
	t.Cleanup(ClipboardClear)

	// A reference for the length of the test, which is what a window loop
	// holds. Without one the connection runs its own background drainer
	// (serviceSelections) and *that* reads the SelectionClear, at a moment
	// when the copy it is about is still the held one — the case that always
	// worked. The bug needs the clear to still be in the queue when the next
	// copy starts, which is exactly what an application with a window does:
	// its loop reads events between one turn of the UI and the next.
	conn, err := x11Retain()
	if err != nil {
		t.Skip("no X11 connection: ", err)
	}
	defer conn.release()

	// One secret of this process's own, so there is a copy to lose.
	first := ClipboardSetSecret([]byte("the-first-passphrase"), -1)
	if !ClipboardHoldsSecret() {
		t.Skip("the secret copy did not take")
	}

	// Another program takes CLIPBOARD, and the SelectionClear it causes is
	// deliberately *not* read: it sits in this process's queue, which is the
	// whole point.
	clipOwner(t, "what the other program copied")
	time.Sleep(50 * time.Millisecond) // time for the event to arrive, not to be read

	want := []byte("the-second-passphrase")
	secret := append([]byte(nil), want...)
	second := ClipboardSetSecret(secret, -1)

	// The end runs on a goroutine, so give it every chance to happen.
	for i := 0; i < 20; i++ {
		pumpX11(t)
		time.Sleep(10 * time.Millisecond)
	}

	if second.Cleared() {
		t.Error("the copy just made was ended by a loss of the one before it")
	}
	if !ClipboardHoldsSecret() {
		t.Error("the clipboard no longer claims the secret this process just put there")
	}
	if !bytes.Equal(secret, want) {
		t.Errorf("the new copy's bytes are %q, want %q — wiped under the call that was serving them", secret, want)
	}
	// And the one that really was lost is gone.
	if !first.Cleared() {
		t.Error("the copy that was replaced is still held")
	}
}
