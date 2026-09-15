package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A horizontal override moves only the horizontal bar's buttons.
func TestScrollGeometryHorizontalArrowOverride(t *testing.T) {
	p, ok := LoadTheme("next")
	if !ok {
		t.Fatal("next pack missing")
	}
	lk := p.Look()
	view := paintengine2d.XYWH(0, 0, 300, 200)
	h := ScrollGeometry(lk, view, false, 900, 300, 0, false)
	if h.Dec.Empty() || h.Inc.Empty() || h.Dec.Max.X > h.Track.Min.X+0.01 || h.Inc.Max.X > h.Track.Min.X+0.01 {
		t.Fatalf("horizontal NeXT arrows should group at the left: dec %v inc %v track %v", h.Dec, h.Inc, h.Track)
	}
	v := ScrollGeometry(lk, view, true, 900, 200, 0, false)
	if v.Dec.Min.Y < v.Track.Max.Y-0.01 || v.Inc.Min.Y < v.Track.Max.Y-0.01 {
		t.Fatalf("vertical NeXT arrows should group at the bottom: dec %v inc %v track %v", v.Dec, v.Inc, v.Track)
	}
}

// A memo builder may use another memoised value (it used to deadlock on
// the look's lock), and every caller gets the one value stored.
func TestMemoNestsAndKeepsOneValue(t *testing.T) {
	l := DarkLook()
	type inner struct{}
	type outer struct{}
	got := l.Memo(outer{}, func() any {
		return l.Memo(inner{}, func() any { return 41 }).(int) + 1
	}).(int)
	if got != 42 {
		t.Fatalf("nested memo = %d, want 42", got)
	}
	if again := l.Memo(outer{}, func() any { return -1 }).(int); again != 42 {
		t.Fatalf("second read rebuilt: %d", again)
	}
	done := make(chan int, 8)
	type racy struct{}
	for i := 0; i < 8; i++ {
		go func(i int) { done <- l.Memo(racy{}, func() any { return i }).(int) }(i)
	}
	first := <-done
	for i := 1; i < 8; i++ {
		if v := <-done; v != first {
			t.Fatalf("racing builds handed out different values: %d and %d", first, v)
		}
	}
}

func TestDarkAliasesNameTheirOwnPacks(t *testing.T) {
	for alias, want := range map[string]string{"adwaita-dark": "adwaita-night", "breeze-dark": "breeze-night", "fusion-dark": "fusion-night"} {
		if got := aliasThemeName(alias); got != want {
			t.Fatalf("alias %s -> %s, want %s", alias, got, want)
		}
		if _, ok := LoadTheme(alias); !ok {
			t.Fatalf("LoadTheme(%s) found nothing", alias)
		}
	}
}
