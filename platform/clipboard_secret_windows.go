//go:build windows

package platform

import (
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

func clipboardNativeSetSecret(b []byte) {
	// UTF-16 without going through a Go string: a string of the
	// passphrase could not be wiped, and this one can.
	u := utf16FromBytes(b)
	defer wipeUint16(u)
	withClipboard(func() bool {
		procEmptyClipboard.Call()
		raw := unsafe.Slice((*byte)(unsafe.Pointer(&u[0])), len(u)*2)
		h := winGlobalOf(raw)
		if h == 0 {
			return false
		}
		if r, _, _ := procSetClipboardData.Call(cfUnicodeText, h); r == 0 {
			procGlobalFree.Call(h)
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

func clipboardNativeClear() {
	withClipboard(func() bool {
		procEmptyClipboard.Call()
		return true
	})
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
