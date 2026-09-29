package style

import (
	"github.com/codemodify/paintengine2d"
	"strings"
	"testing"
)

// The byte path measures what the string path measures, give or take the
// kerning a shaped run carries and this one deliberately does not.
func TestAdvanceBytesAgreesWithAdvance(t *testing.T) {
	f := BakeFamily("", WeightRegular, 14, paintengine2d.White)
	for _, s := range []string{"a", "hello", "Wave AV To", "correct horse battery staple"} {
		want := f.Advance(s)
		got := f.AdvanceBytes([]byte(s))
		if want == 0 {
			t.Fatalf("%q measured 0", s)
		}
		// Kerning only ever pulls pairs together, so the unkerned run is
		// the same width or a little wider, never narrower.
		if got < want-0.01 {
			t.Fatalf("%q: bytes %v < string %v", s, got, want)
		}
		if d := (got - want) / want; d > 0.1 {
			t.Fatalf("%q: bytes %v vs string %v (%.0f%% wider)", s, got, want, d*100)
		}
	}
	if f.AdvanceBytes(nil) != 0 || f.AdvanceBytes([]byte{}) != 0 {
		t.Fatal("empty is not zero")
	}
}

// Drawing or measuring a byte slice must leave nothing in the shape
// cache. That is the whole reason this path exists: a string cannot be
// wiped, and a cached run would hold the secret for the life of the face.
func TestBytePathCachesNothing(t *testing.T) {
	f := BakeFamily("", WeightRegular, 14, paintengine2d.White)
	// A string no other test measures: BakeFamily hands out copies that
	// share one shape cache, so a warm-up anywhere in the package would
	// otherwise be mistaken for this path having cached it.
	const secret = "zzq-secret-never-measured-as-a-string"
	// Warm the cache with something else, so it exists.
	f.Advance("something else entirely")
	before := shapeCacheKeys(f)

	f.AdvanceBytes([]byte(secret))
	f.CaretXBytes([]byte(secret), 7)
	f.IndexAtBytes([]byte(secret), 30)

	after := shapeCacheKeys(f)
	for _, k := range after {
		if strings.Contains(k, "zzq-secret") {
			t.Fatalf("the shape cache holds %q", k)
		}
	}
	if len(after) != len(before) {
		t.Fatalf("the cache grew from %d to %d entries", len(before), len(after))
	}
}

// shapeCacheKeys is what the face has shaped and kept.
func shapeCacheKeys(f *Font) []string {
	if f == nil || f.shaped == nil {
		return nil
	}
	f.shaped.mu.Lock()
	defer f.shaped.mu.Unlock()
	out := make([]string, 0, len(f.shaped.m))
	for k := range f.shaped.m {
		out = append(out, k)
	}
	return out
}

// Carets and hit-testing round-trip: the offset a click lands on is the
// offset whose caret is under the pointer.
func TestCaretAndHitTestOnBytes(t *testing.T) {
	f := BakeFamily("", WeightRegular, 14, paintengine2d.White)
	b := []byte("hello wörld")
	if f.CaretXBytes(b, 0) != 0 {
		t.Fatal("caret 0 is not at 0")
	}
	if got, want := f.CaretXBytes(b, len(b)), f.AdvanceBytes(b); got != want {
		t.Fatalf("caret at the end is %v, the run is %v", got, want)
	}
	for i := 0; i <= len(b); i++ {
		if i > 0 && i < len(b) && !isRuneStart(b[i]) {
			continue
		}
		if got := f.IndexAtBytes(b, f.CaretXBytes(b, i)); got != i {
			t.Fatalf("caret %d is at x=%v, which hit-tests to %d", i, f.CaretXBytes(b, i), got)
		}
	}
	// A hit inside a multi-byte rune snaps to a boundary, never into it.
	for x := float32(0); x < f.AdvanceBytes(b)+20; x += 1.5 {
		i := f.IndexAtBytes(b, x)
		if i < 0 || i > len(b) {
			t.Fatalf("x=%v gave offset %d", x, i)
		}
		if i > 0 && i < len(b) && !isRuneStart(b[i]) {
			t.Fatalf("x=%v landed inside a rune at %d", x, i)
		}
	}
}

func isRuneStart(c byte) bool { return c&0xC0 != 0x80 }
