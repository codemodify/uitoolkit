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
