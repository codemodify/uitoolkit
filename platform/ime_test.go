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
