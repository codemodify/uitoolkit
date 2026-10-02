//go:build linux && cgo

package platform

import "testing"

// The password-manager hint is answered with the word "secret", never
// with the secret.
//
// The convention — KeePassXC writes it, Klipper reads it — is that a
// clipboard manager asks for x-kde-passwordManagerHint and takes the
// value "secret" to mean "do not record this copy". The Wayland data
// source advertised the type and then answered every request with the
// selection, so Klipper asked for the hint, found a passphrase where it
// looks for "secret", concluded the copy was ordinary, and kept it in the
// history it writes to disk. X11 has always answered it correctly.
func TestTheWaylandHintIsAnsweredSecretNotTheSecret(t *testing.T) {
	c := &wlConn{}
	secret := []byte("correct horse battery staple")
	c.clip.setBytes(secret)
	c.clipSecret = true

	got, ok := wlHintAnswerFor(c, wlPasswordHintMime)
	if !ok {
		t.Fatal("a request for the hint was not recognised as one")
	}
	if string(got) != "secret" {
		t.Errorf("the hint is answered %q, want \"secret\"", got)
	}
	if string(got) == string(secret) {
		t.Fatal("the hint was answered with the passphrase itself")
	}

	// The text types still get the bytes.
	if _, ok := wlHintAnswerFor(c, "text/plain;charset=utf-8"); ok {
		t.Error("a text request was treated as the hint")
	}
	// And an ordinary copy has no hint to answer at all.
	c.clipSecret = false
	if _, ok := wlHintAnswerFor(c, wlPasswordHintMime); ok {
		t.Error("an ordinary selection answered the password hint")
	}
}
