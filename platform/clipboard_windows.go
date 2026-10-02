//go:build windows

package platform

import (
	"syscall"
	"time"
	"unsafe"
)

// The Windows clipboard: OpenClipboard, GetClipboardData,
// SetClipboardData, with CF_UNICODETEXT.
//
// Before this every non-Linux build fell to clipboard_stub.go, which
// answered nothing and threw writes away — so copy and paste did nothing
// at all on Windows, in a toolkit whose text fields have had Ctrl+C since
// the beginning.
//
// Three things about this API are worth knowing because they decide the
// shape of the code.
//
// It is a lock, not a handle. OpenClipboard takes an exclusive lock on a
// single system-wide resource, and any other application that wants it
// while we hold it is blocked; every path through here closes it, and
// nothing that can block happens in between. It also fails while another
// application holds it, which is normal rather than exceptional — a
// retry over a few milliseconds is what every toolkit does.
//
// SetClipboardData gives the memory away. The HGLOBAL passed to it
// belongs to the clipboard afterwards and must not be freed here; the
// block is only ours to free if the call fails.
//
// There is no PRIMARY selection. X11's middle-click clipboard has no
// counterpart, so ClipboardPrimaryGet answers from the same place as
// ClipboardGet rather than pretending to a second one.

var (
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procGetClipboardSeq  = user32.NewProc("GetClipboardSequenceNumber")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procIsFormatAvail    = user32.NewProc("IsClipboardFormatAvailable")
)

// clipboardTries is how many times to ask for the lock, and how long to
// wait between. Another application holding the clipboard for a moment
// is ordinary; holding it for a quarter of a second is not, and at that
// point failing is better than blocking the loop.
const clipboardTries = 12

func withClipboard(fn func() bool) bool {
	for i := 0; i < clipboardTries; i++ {
		if r, _, _ := procOpenClipboard.Call(0); r != 0 {
			ok := fn()
			procCloseClipboard.Call()
			return ok
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func clipboardNativeGet() (string, bool) {
	var out string
	ok := withClipboard(func() bool {
		if r, _, _ := procIsFormatAvail.Call(cfUnicodeText); r == 0 {
			// Nothing textual on the clipboard is a real answer: it is
			// empty, not unavailable, and reporting true with "" keeps
			// the in-memory buffer from being handed back as though it
			// were the session's.
			return true
		}
		h, _, _ := procGetClipboardData.Call(cfUnicodeText)
		if h == 0 {
			return false
		}
		out = string(winTextFromGlobal(h))
		return true
	})
	return out, ok
}

// clipboardNativePrimaryGet is the same as the clipboard: Windows has no
// PRIMARY selection, and answering from somewhere else would invent one.
func clipboardNativePrimaryGet() (string, bool) { return clipboardNativeGet() }

func clipboardNativeSet(s string) {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return
	}
	withClipboard(func() bool {
		procEmptyClipboard.Call()
		h := winGlobalOf(unsafe.Slice((*byte)(unsafe.Pointer(&u[0])), len(u)*2))
		if h == 0 {
			return false
		}
		if r, _, _ := procSetClipboardData.Call(cfUnicodeText, h); r == 0 {
			// It refused, so the block is still ours.
			procGlobalFree.Call(h)
			return false
		}
		// It took it. The clipboard owns the memory now.
		return true
	})
}
