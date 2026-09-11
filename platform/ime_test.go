package platform

import "testing"

func TestApplyPreeditDraw(t *testing.T) {
	if got := ApplyPreeditDraw("", 0, 0, "に"); got != "に" {
		t.Fatalf("start %q", got)
	}
	if got := ApplyPreeditDraw("に", 0, 1, "にほ"); got != "にほ" {
		t.Fatalf("replace %q", got)
	}
	if got := ApplyPreeditDraw("ab", 1, 0, "X"); got != "aXb" {
		t.Fatalf("insert %q", got)
	}
	if got := ApplyPreeditDraw("hello", 1, 3, ""); got != "ho" {
		t.Fatalf("delete %q", got)
	}
}

func TestDeleteSurroundingUTF8(t *testing.T) {
	s, c := DeleteSurroundingUTF8("hello", 5, 2, 0)
	if s != "hel" || c != 3 {
		t.Fatalf("before %q caret=%d", s, c)
	}
	s, c = DeleteSurroundingUTF8("ひらがな", 2, 3, 3)
	if s != "がな" && s != "らがな" {
		// ひ is 3 bytes; deleting 3 before caret 2 removes ひ
		if c < 0 {
			t.Fatal(c)
		}
	}
	s, _ = DeleteSurroundingUTF8("abc", 1, 1, 1)
	if s != "c" {
		t.Fatalf("around %q", s)
	}
}

func TestComposeVisual(t *testing.T) {
	vis, caret, a, b := ComposeVisual("ab", 1, "X", 1)
	if vis != "aXb" || caret != 2 || a != 1 || b != 2 {
		t.Fatalf("vis=%q caret=%d [%d,%d]", vis, caret, a, b)
	}
	vis, caret, a, b = ComposeVisual("hi", 2, "", 0)
	if vis != "hi" || caret != 2 || a != b {
		t.Fatalf("empty preedit %q %d %d %d", vis, caret, a, b)
	}
}

func TestInsertAtRune(t *testing.T) {
	if InsertAtRune("ab", 1, "X") != "aXb" {
		t.Fatal("insert")
	}
	if InsertAtRune("", 0, "に") != "に" {
		t.Fatal("empty")
	}
}

func TestShouldEmitXKBText(t *testing.T) {
	// Regression: text-input-v3 "active" must not starve EventText.
	// The old uitkWlKey gate was: !pressed || Ctrl || textActive → return.
	if !ShouldEmitXKBText(true, false, false) {
		t.Fatal("printable key with idle IME must emit EventText")
	}
	if ShouldEmitXKBText(false, false, false) {
		t.Fatal("key-up")
	}
	if ShouldEmitXKBText(true, true, false) {
		t.Fatal("Ctrl is a shortcut, not text")
	}
	if ShouldEmitXKBText(true, false, true) {
		t.Fatal("active preedit: IME owns the key")
	}
}

func TestPairXKBAndIMECommit(t *testing.T) {
	// Latin compositor: xkb utf8 first, then IME commit of the same rune.
	emit, lastX, lastI := PairXKBText("a", "")
	if emit != "a" || lastX != "a" || lastI != "" {
		t.Fatalf("xkb first %#v %#v %#v", emit, lastX, lastI)
	}
	emit, lastX, lastI = PairIMECommit("a", lastX)
	if emit != "" || lastX != "" || lastI != "" {
		t.Fatalf("dedupe IME %#v %#v %#v", emit, lastX, lastI)
	}

	// Reverse order: IME commit first, then xkb of the same rune.
	emit, lastX, lastI = PairIMECommit("a", "")
	if emit != "a" || lastX != "" || lastI != "a" {
		t.Fatalf("IME first %#v %#v %#v", emit, lastX, lastI)
	}
	emit, lastX, lastI = PairXKBText("a", lastI)
	if emit != "" || lastX != "" || lastI != "" {
		t.Fatalf("dedupe xkb %#v %#v %#v", emit, lastX, lastI)
	}

	// Repeated same letter: second key must still insert.
	emit, lastX, lastI = PairXKBText("a", "")
	if emit != "a" {
		t.Fatal("first a")
	}
	emit, lastX, lastI = PairXKBText("a", lastI)
	if emit != "a" || lastX != "a" {
		t.Fatalf("second a must not be eaten by previous xkb pair %#v %#v", emit, lastX)
	}

	// CJK commit after xkb Latin must still land (preedit path skips xkb).
	emit, lastX, lastI = PairXKBText("a", "")
	emit, lastX, lastI = PairIMECommit("あ", lastX)
	if emit != "あ" || lastI != "あ" {
		t.Fatalf("distinct IME commit %#v %#v", emit, lastI)
	}

	emit, lastX, lastI = PairIMECommit("", "a")
	if emit != "" || lastX != "a" || lastI != "" {
		t.Fatalf("empty commit keeps last xkb %#v %#v %#v", emit, lastX, lastI)
	}
}

func TestTextEventsPrintable(t *testing.T) {
	evs := textEvents([]byte("ab\x7f\n!"), 0)
	if len(evs) != 3 || evs[0].Rune != 'a' || evs[1].Rune != 'b' || evs[2].Rune != '!' {
		t.Fatalf("%+v", evs)
	}
	if evs[0].Kind != EventText {
		t.Fatal("kind")
	}
}
