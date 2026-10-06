//go:build windows

package platform

import (
	"sync/atomic"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

// The Windows secret clipboard.
//
// Windows records what is copied in three places the user did not ask
// for: the clipboard history (Win+V), the cloud clipboard that syncs it
// to the account's other machines, and whatever clipboard manager is
// installed. Each is turned off for one copy by putting an empty entry
// under a registered format name, which is the documented way and what
// every Windows password manager writes:
//
//	ExcludeClipboardContentFromMonitorProcessing  managers, generally
//	CanIncludeInClipboardHistory                  = 0, Win+V
//	CanUploadToCloudClipboard                     = 0, the sync
//
// The first is a marker whose presence is the whole message; the other
// two carry a DWORD 0. All three are set inside the same
// OpenClipboard / EmptyClipboard as the text, because they describe that
// one entry.
var (
	procRegisterClipboardFormatW = user32.NewProc("RegisterClipboardFormatW")

	cfExcludeFromMonitor = registerClipboardFormat("ExcludeClipboardContentFromMonitorProcessing")
	cfCanIncludeHistory  = registerClipboardFormat("CanIncludeInClipboardHistory")
	cfCanUploadCloud     = registerClipboardFormat("CanUploadToCloudClipboard")
)

func registerClipboardFormat(name string) uintptr {
	u, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0
	}
	r, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(u)))
	return r
}

// It reports whether the copy was actually made, and notes the sequence
// number only then.
//
// The number used to be noted in a defer, whether or not anything was copied.
// Two ways that happens: another program can hold the clipboard open for all
// of withClipboard's tries, and SetClipboardData can refuse the block after
// EmptyClipboard has already emptied the clipboard. Either way the program was
// told it had copied — a SecretField paste was served the held secret while
// the clipboard held something else, ClipboardHoldsSecret said a secret was
// there, and the clear at its time found the number unchanged and emptied a
// clipboard holding the person's own work, which is the one thing that check
// exists to prevent.
func clipboardNativeSetSecret(b []byte) bool {
	// UTF-16 without going through a Go string: a string of the
	// passphrase could not be wiped, and this one can.
	u := utf16FromBytes(b)
	defer wipeUint16(u)
	ok := withClipboard(func() bool {
		procEmptyClipboard.Call()
		raw := unsafe.Slice((*byte)(unsafe.Pointer(&u[0])), len(u)*2)
		h := winGlobalOf(raw)
		if h == 0 {
			return false
		}
		if r, _, _ := procSetClipboardData.Call(cfUnicodeText, h); r == 0 {
			// The block holds the passphrase as UTF-16 and the clipboard
			// did not take it, so this process still owns it: zeroed
			// before it is released, because GlobalFree does not clear it
			// and a passphrase in a freed allocation is readable by
			// whatever gets that memory next.
			winGlobalWipeFree(h, len(raw))
			return false
		}
		// The three hints, on the same entry. A failure to set one is
		// not a failure to copy: the passphrase is on the clipboard and
		// the timeout will still take it off.
		setClipboardFlag(cfExcludeFromMonitor, nil)
		zero := []byte{0, 0, 0, 0}
		setClipboardFlag(cfCanIncludeHistory, zero)
		setClipboardFlag(cfCanUploadCloud, zero)
		return true
	})
	if ok {
		winClipSeq.Store(clipboardSeq())
	}
	return ok
}

// winGlobalWipeFree zeroes a global block before releasing it, for a block
// this process still owns because the clipboard refused it.
func winGlobalWipeFree(h uintptr, n int) {
	if h == 0 {
		return
	}
	if p, _, _ := procGlobalLock.Call(h); p != 0 {
		blk := unsafe.Slice(lparamAs[byte](p), n)
		for i := range blk {
			blk[i] = 0
		}
		procGlobalUnlock.Call(h)
	}
	procGlobalFree.Call(h)
}

// setClipboardFlag puts one marker format on the open clipboard. An
// empty payload is a one-byte block, because SetClipboardData with a
// null handle means "render on demand" and nothing here will.
func setClipboardFlag(format uintptr, payload []byte) {
	if format == 0 {
		return
	}
	if len(payload) == 0 {
		payload = []byte{0}
	}
	h := winGlobalOf(payload)
	if h == 0 {
		return
	}
	if r, _, _ := procSetClipboardData.Call(format, h); r == 0 {
		procGlobalFree.Call(h)
	}
}

// winClipSeq is GetClipboardSequenceNumber as of the last secret copy.
// Windows bumps it on every change by anyone, so an unchanged number
// means the clipboard still holds what this process put there.
//
// Atomic because the copy is made on the UI goroutine and the clear is made
// on the timer's, and both touch it.
var winClipSeq atomic.Uint32

func clipboardSeq() uint32 {
	r, _, _ := procGetClipboardSeq.Call()
	return uint32(r)
}

func clipboardNativeClear() {
	// Only while the clipboard is still ours. EmptyClipboard takes
	// whatever is there, and after the timeout — thirty seconds in a
	// password manager — that is as likely to be something the person
	// copied since. The toolkit's own promise is that a clear "does
	// nothing if something else has since taken the clipboard"; it was
	// not keeping it.
	if !clipboardStillHoldsOurSecret() {
		return
	}
	withClipboard(func() bool {
		procEmptyClipboard.Call()
		return true
	})
	winClipSeq.Store(0)
}

// clipboardStillHoldsOurSecret reports whether the clipboard still holds the
// secret this process last put there. Windows tells a program nothing when
// another one copies, so the only way to know is to ask for the sequence
// number again and compare.
//
// A zero noted number means no secret copy is outstanding, in which case
// there is nothing for this to be wrong about; a zero *current* number is the
// call failing, and is treated the same way, because refusing to clear would
// leave a passphrase on the clipboard.
func clipboardStillHoldsOurSecret() bool {
	was := winClipSeq.Load()
	if was == 0 {
		return true
	}
	seq := clipboardSeq()
	return seq == 0 || seq == was
}

// utf16FromBytes encodes UTF-8 bytes as a NUL-terminated UTF-16 buffer,
// without a Go string anywhere in the middle.
func utf16FromBytes(b []byte) []uint16 {
	out := make([]uint16, 0, len(b)+1)
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		if n == 0 {
			break
		}
		i += n
		if r1, r2 := utf16.EncodeRune(r); r1 != 0xFFFD {
			out = append(out, uint16(r1), uint16(r2))
			continue
		}
		out = append(out, uint16(r))
	}
	return append(out, 0)
}

func wipeUint16(u []uint16) {
	for i := range u {
		u[i] = 0
	}
}

// clipboardNativeGetSecret reads CF_UNICODETEXT as UTF-8 bytes without
// making a Go string of it.
//
// The ordinary read goes through syscall.UTF16ToString, which builds a
// string, so a passphrase pasted from another program — a terminal, or
// another password manager — was in this process's heap as a string that
// nothing could wipe. This decodes into a byte slice the caller owns and
// zeroes everything it worked through on the way.
func clipboardNativeGetSecret() ([]byte, bool) {
	var out []byte
	ok := withClipboard(func() bool {
		if r, _, _ := procIsFormatAvail.Call(cfUnicodeText); r == 0 {
			// Nothing textual: empty, not unavailable.
			return true
		}
		h, _, _ := procGetClipboardData.Call(cfUnicodeText)
		if h == 0 {
			return false
		}
		out = winSecretFromGlobal(h)
		return true
	})
	return out, ok && len(out) > 0
}

// winSecretFromGlobal is winTextFromGlobal without the string.
//
// The clipboard's own buffer belongs to whichever program owns the
// selection, so it is read and not written; what this zeroes is the
// UTF-16 copy and the runes it decoded through, which are this process's.
func winSecretFromGlobal(h uintptr) []byte {
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return nil
	}
	defer procGlobalUnlock.Call(h)
	src := lparamAs[[1 << 20]uint16](ptr)
	n := 0
	for n < len(src) && src[n] != 0 {
		n++
	}
	if n == 0 {
		return nil
	}
	u := make([]uint16, n)
	copy(u, src[:n])
	runes := utf16.Decode(u)
	out := make([]byte, 0, n*3)
	var buf [4]byte
	for _, r := range runes {
		out = append(out, buf[:utf8.EncodeRune(buf[:], r)]...)
	}
	for i := range u {
		u[i] = 0
	}
	for i := range runes {
		runes[i] = 0
	}
	for i := range buf {
		buf[i] = 0
	}
	return out
}

// clipboardSecretBytesPath reports whether this platform can read a secret
// from the clipboard without making a string of it. Where it can, a failed
// read is a failed read: falling back to the string reader would undo the
// whole point of the bytes path on the try that happens to succeed.
func clipboardSecretBytesPath() bool { return true }
