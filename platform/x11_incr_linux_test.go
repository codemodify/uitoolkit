//go:build linux && cgo

package platform

import (
	"bytes"
	"testing"
)

// An INCR transfer whose last pieces arrive after the paste gave up waiting
// is nobody's, and is let go rather than kept.
//
// A long selection — the private key that made it long enough to need INCR in
// the first place — arrives in 4 KiB steps, and readSelection has a deadline.
// Past it, the pieces still come: the completed value was put in pasteData,
// where nothing would ever take it, and sat in this process's heap until some
// later paste happened to overwrite it.
//
// No display: this is the state machine, and the point is what it does when
// nothing is listening, which is not a thing an X server decides.
func TestALateINCRTransferIsNotKept(t *testing.T) {
	secret := []byte("-----BEGIN PRIVATE KEY-----...")
	c := &x11Conn{}
	c.incrRecv = incrRecvState{active: true}
	c.incrRecv.buf = append([]byte(nil), secret...)
	buf := c.incrRecv.buf

	// Nobody is waiting: the paste timed out and returned long ago.
	c.pasteWaiting = false
	c.finishINCRLocked()

	if len(c.pasteData) != 0 {
		t.Errorf("pasteData holds %d bytes nothing will ever read", len(c.pasteData))
	}
	if c.pasteDone {
		t.Error("the paste was reported done to a reader that had gone")
	}
	if c.incrRecv.active {
		t.Error("the transfer was left active")
	}
	if !bytes.Equal(buf, make([]byte, len(buf))) {
		t.Errorf("the transfer's buffer was not zeroed: %q", buf)
	}
}

// And a transfer somebody is waiting for becomes the paste, which is the
// whole job: a test that dropped both would pass the one above and break
// every long paste there is.
func TestAnAwaitedINCRTransferBecomesThePaste(t *testing.T) {
	want := []byte("-----BEGIN PRIVATE KEY-----...")
	c := &x11Conn{}
	c.incrRecv = incrRecvState{active: true}
	c.incrRecv.buf = append([]byte(nil), want...)
	buf := c.incrRecv.buf

	c.pasteWaiting = true
	c.finishINCRLocked()

	if !bytes.Equal(c.pasteData, want) {
		t.Errorf("pasteData = %q, want the transferred value", c.pasteData)
	}
	if !c.pasteDone {
		t.Error("the paste was not reported done")
	}
	if c.incrRecv.active {
		t.Error("the transfer was left active")
	}
	// The transfer's own copy still goes: pasteData is a copy of its own,
	// and two copies of a key in the heap is one more than is needed.
	if !bytes.Equal(buf, make([]byte, len(buf))) {
		t.Errorf("the transfer's buffer was not zeroed: %q", buf)
	}
}

// Finishing a transfer lets go of whatever the previous paste had read. The
// value being replaced is as much somebody's passphrase as the new one.
func TestFinishingATransferWipesTheOldPaste(t *testing.T) {
	c := &x11Conn{}
	c.pasteData = []byte("the previous paste")
	old := c.pasteData
	c.incrRecv = incrRecvState{active: true, buf: []byte("the new one")}
	c.pasteWaiting = true
	c.finishINCRLocked()
	if !bytes.Equal(old, make([]byte, len(old))) {
		t.Errorf("the previous paste was dropped without being wiped: %q", old)
	}
}
