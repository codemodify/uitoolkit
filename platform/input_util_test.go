package platform

import (
	"testing"
	"time"
)

func TestKeyFromKeysymFallbackNonLatinLayout(t *testing.T) {
	const (
		cyrillicEs = 0x6d3 // what Ctrl+C reports under a Russian layout
		greekPsi   = 0x7f9
		latinC     = 'c'
		latinV     = 'v'
	)
	if got := KeyFromKeysymFallback(cyrillicEs, latinC); got != KeyC {
		t.Fatalf("Cyrillic_es with base 'c' = %v, want KeyC", got)
	}
	if got := KeyFromKeysymFallback(greekPsi, latinV); got != KeyV {
		t.Fatalf("Greek_psi with base 'v' = %v, want KeyV", got)
	}
	// A layout that does map the symbol must win over the base symbol.
	if got := KeyFromKeysymFallback('a', 'q'); got != KeyA {
		t.Fatalf("'a' with base 'q' = %v, want KeyA (layout symbol wins)", got)
	}
	// Nothing to fall back to.
	if got := KeyFromKeysymFallback(cyrillicEs, 0); got != KeyUnknown {
		t.Fatalf("no base symbol = %v, want KeyUnknown", got)
	}
	if got := KeyFromKeysymFallback(cyrillicEs, cyrillicEs); got != KeyUnknown {
		t.Fatalf("base == sym = %v, want KeyUnknown", got)
	}
	// Named keys are layout-independent already.
	if got := KeyFromKeysymFallback(0xff1b, 0); got != KeyEscape {
		t.Fatalf("Escape = %v", got)
	}
}

func TestLatin1ToUTF8(t *testing.T) {
	cases := []struct {
		in   []byte
		want string
	}{
		{[]byte("hello"), "hello"},
		{[]byte{0xe9}, "é"},
		{[]byte{0xfc, 0x62, 0xe4}, "übä"},
		{[]byte{0xa3}, "£"},
		{nil, ""},
	}
	for _, c := range cases {
		if got := string(latin1ToUTF8(c.in)); got != c.want {
			t.Fatalf("latin1ToUTF8(%v) = %q, want %q", c.in, got, c.want)
		}
	}
	// The pure-ASCII path must not copy.
	in := []byte("abc")
	if out := latin1ToUTF8(in); &out[0] != &in[0] {
		t.Fatal("ASCII input should be returned unchanged")
	}
}

func TestLatin1TextEventsProduceRealRunes(t *testing.T) {
	// The no-XIC X11 path: 0xE9 is é, not U+FFFD.
	evs := textEvents(latin1ToUTF8([]byte{0xe9}), 0)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	if evs[0].Rune != 'é' {
		t.Fatalf("rune = %q, want é", evs[0].Rune)
	}
}

func TestShouldArmKeyRepeat(t *testing.T) {
	cases := []struct {
		repeats     bool
		rate, delay int
		want        bool
	}{
		{true, 25, 600, true},
		{false, 25, 600, false}, // modifier: keymap says it does not repeat
		{true, 0, 600, false},   // compositor disabled repeat
		{true, 25, 0, false},
		{false, 0, 0, false},
	}
	for _, c := range cases {
		if got := ShouldArmKeyRepeat(c.repeats, c.rate, c.delay); got != c.want {
			t.Fatalf("ShouldArmKeyRepeat(%v,%d,%d) = %v, want %v",
				c.repeats, c.rate, c.delay, got, c.want)
		}
	}
}

func TestRepeatInterval(t *testing.T) {
	if got := repeatInterval(25); got != 40*time.Millisecond {
		t.Fatalf("25/s = %v, want 40ms", got)
	}
	if got := repeatInterval(0); got != 0 {
		t.Fatalf("rate 0 = %v, want 0", got)
	}
	// A hostile rate must not spin the loop.
	if got := repeatInterval(100000); got < 10*time.Millisecond {
		t.Fatalf("absurd rate = %v, want >= 10ms", got)
	}
	if got := repeatInterval(1); got > time.Second {
		t.Fatalf("slow rate = %v, want <= 1s", got)
	}
}
