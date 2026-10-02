package platform

import (
	"sync"
	"time"
)

// The secret clipboard: a copy that is meant to be pasted once and then
// to stop existing.
//
// [ClipboardSet] is wrong for a password in four ways, and each one is a
// way it leaks:
//
//   - it writes the X11 PRIMARY selection as well as CLIPBOARD, so the
//     next middle click anywhere on the desktop pastes it;
//   - it says nothing to the clipboard managers — Klipper, GPaste,
//     Windows' clipboard history and cloud sync, the Mac's universal
//     clipboard — which therefore record it, sometimes to another
//     machine;
//   - there is no way to take it back, so it stays until something else
//     is copied;
//   - it takes a string, which cannot be wiped.
//
// ClipboardSetSecret answers all four. It writes CLIPBOARD only, with
// whatever each platform's managers honour as "do not record this", it
// takes a []byte whose ownership it assumes and wipes, and it clears
// itself after clearAfter — and on quit, whichever comes first.
//
// Clearing is conditional, as every password manager's is: if something
// else has taken the clipboard in the meantime, that is the user's copy
// and this must not wipe it.
type SecretClip struct {
	mu      sync.Mutex
	data    []byte
	timer   *time.Timer
	cleared bool
	onClear func()
}

var (
	secretMu   sync.Mutex
	secretHeld *SecretClip
)

// DefaultSecretClipboardTimeout is the usual "clear it after a while",
// and what ClipboardSetSecret uses when clearAfter is zero. Forty-five
// seconds is KeePassXC's default and about what 1Password and Bitwarden
// use; it is long enough to switch window and paste, and short enough
// that a forgotten copy is not still there at lunch.
const DefaultSecretClipboardTimeout = 45 * time.Second

// ClipboardSetSecret puts data on the CLIPBOARD selection — never
// PRIMARY — hinted so clipboard managers do not record it, and clears it
// again after clearAfter (zero: [DefaultSecretClipboardTimeout];
// negative: never, and the caller clears it).
//
// It **takes ownership of data**: the caller must not use or wipe the
// slice afterwards. It is wiped here, when the clipboard is cleared.
//
// The returned handle clears this copy and only this copy, so a caller
// that has moved on does not wipe somebody else's. Call it from the UI
// goroutine, like every other clipboard call.
func ClipboardSetSecret(data []byte, clearAfter time.Duration) *SecretClip {
	if clearAfter == 0 {
		clearAfter = DefaultSecretClipboardTimeout
	}
	c := &SecretClip{data: data}

	secretMu.Lock()
	prev := secretHeld
	secretHeld = c
	secretMu.Unlock()
	// A previous secret is dropped rather than left for its own timer:
	// it is no longer on the clipboard, so there is nothing to clear,
	// only a buffer to wipe.
	if prev != nil {
		prev.forget()
	}

	// The in-memory fallback must not hold it as a string: nothing is
	// put in `clip`, and a paste in this process goes through
	// ClipboardGetSecret, which reads the bytes.
	clipMu.Lock()
	clip = ""
	clipMu.Unlock()

	clipboardNativeSetSecret(data)

	if clearAfter > 0 {
		c.mu.Lock()
		c.timer = time.AfterFunc(clearAfter, c.Clear)
		c.mu.Unlock()
	}
	return c
}

// Clear takes this secret off the clipboard and wipes the toolkit's copy
// of it. It does nothing if something else has since taken the
// clipboard — that is the user's own copy, and wiping it would be a
// clipboard that empties itself at random — and nothing if it has
// already been cleared.
func (c *SecretClip) Clear() {
	if c == nil {
		return
	}
	secretMu.Lock()
	mine := secretHeld == c
	if mine {
		secretHeld = nil
	}
	secretMu.Unlock()
	if mine {
		clipboardNativeClear()
	}
	c.forget()
}

// forget wipes the buffer and stops the timer, without touching the
// clipboard.
func (c *SecretClip) forget() {
	c.mu.Lock()
	if c.cleared {
		c.mu.Unlock()
		return
	}
	c.cleared = true
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	for i := range c.data {
		c.data[i] = 0
	}
	c.data = nil
	fn := c.onClear
	c.onClear = nil
	c.mu.Unlock()
	// Outside the lock: the callback is the application's, it runs on
	// whatever goroutine cleared the copy — the timer's, most often — and
	// holding the clip's lock across it would let it deadlock by asking
	// this copy anything about itself.
	if fn != nil {
		fn()
	}
}

// OnCleared installs a callback for the moment this copy goes: the
// timeout expiring, another copy replacing it, or the program quitting.
//
// It is what a "copied — clears in 45s" indicator is built from. Without
// it an application has to poll [SecretClip.Cleared] to find out, and a
// poll is both a timer of its own and a window in which the interface
// says the passphrase is still on the clipboard when it is not.
//
// The callback runs at most once and on the goroutine that cleared the
// copy, which is usually not the UI one — hand it to the application's
// dispatcher rather than touching widgets in it. Installing it on a copy
// that has already been cleared calls it straight away, so a caller
// cannot lose the event to a timeout that beat it.
func (c *SecretClip) OnCleared(fn func()) {
	if c == nil {
		if fn != nil {
			fn()
		}
		return
	}
	c.mu.Lock()
	if c.cleared {
		c.mu.Unlock()
		if fn != nil {
			fn()
		}
		return
	}
	c.onClear = fn
	c.mu.Unlock()
}

// Cleared reports whether this copy is gone.
func (c *SecretClip) Cleared() bool {
	if c == nil {
		return true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cleared
}

// ClipboardClear empties the clipboard, whatever is on it, and wipes
// whatever secret the toolkit was holding. An application calls it when
// it locks.
//
// It is deliberately unconditional, unlike [ClipboardClearSecret]: a
// vault that has just locked is saying "take everything back", and it
// cannot know whether what is on the clipboard came from one of its own
// fields by some path the toolkit did not label.
func ClipboardClear() {
	secretMu.Lock()
	held := secretHeld
	secretHeld = nil
	secretMu.Unlock()
	if held != nil {
		held.forget()
	}
	clipMu.Lock()
	clip = ""
	clipMu.Unlock()
	clipboardNativeClear()
}

// ClipboardClearSecret clears the clipboard only if what is on it is a
// secret this process put there, and does nothing otherwise.
//
// That is the difference between taking a password back and throwing
// away whatever the user copied. The toolkit calls it on quit: a copied
// secret must not outlive the application that copied it, and an
// ordinary copy must still be there afterwards — both backends go on
// serving a selection after the last window closes precisely so that
// another application's paste does not hang on it.
func ClipboardClearSecret() {
	if !ClipboardHoldsSecret() {
		return
	}
	ClipboardClear()
}

// ClipboardGetSecret reads the clipboard as bytes, for pasting into a
// field that must not make a string of it.
//
// When what is there is the toolkit's own secret it is copied straight
// out of the buffer and no string is made at any point. When it is
// somebody else's, the platform hands the toolkit a string — that is
// what the clipboard API of every one of them returns — and this
// converts it. The string is the session's, not the toolkit's; what this
// can promise is that no further copy is kept.
//
// The caller owns the result and wipes it.
func ClipboardGetSecret() []byte {
	secretMu.Lock()
	held := secretHeld
	secretMu.Unlock()
	if held != nil {
		held.mu.Lock()
		if !held.cleared && len(held.data) > 0 {
			out := make([]byte, len(held.data))
			copy(out, held.data)
			held.mu.Unlock()
			return out
		}
		held.mu.Unlock()
	}
	// Another program's secret: read it as bytes where the backend can,
	// so a passphrase copied out of a terminal or another password
	// manager does not become a Go string in this process's heap, which
	// nothing could then wipe. Where it cannot, the ordinary read is
	// still better than refusing to paste.
	if b, ok := clipboardNativeGetSecret(); ok {
		return b
	}
	s := ClipboardGet()
	if s == "" {
		return nil
	}
	return []byte(s)
}

// ClipboardHoldsSecret reports whether the clipboard holds a secret this
// process put there and has not cleared. It is for a lock screen that
// wants to say so, and for tests.
func ClipboardHoldsSecret() bool {
	secretMu.Lock()
	held := secretHeld
	secretMu.Unlock()
	return held != nil && !held.Cleared()
}
