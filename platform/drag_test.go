package platform

import "testing"

func TestDragActionHasAndOne(t *testing.T) {
	both := DragCopy | DragMove
	if !both.Has(DragCopy) || !both.Has(DragMove) || both.Has(DragLink) {
		t.Fatalf("Has: %v", both)
	}
	if DragNone.Has(DragNone) {
		t.Fatal("nothing has DragNone")
	}
	if got := both.One(); got != DragCopy {
		t.Fatalf("One prefers copying: got %v", got)
	}
	if got := (DragMove | DragLink).One(); got != DragMove {
		t.Fatalf("One: got %v", got)
	}
	if got := DragNone.One(); got != DragNone {
		t.Fatalf("One of nothing: got %v", got)
	}
}

func TestDragActionString(t *testing.T) {
	for _, tc := range []struct {
		a    DragAction
		want string
	}{
		{DragCopy, "copy"},
		{DragMove, "move"},
		{DragLink, "link"},
		{DragNone, "none"},
		{DragMove | DragLink, "move"},
	} {
		if got := tc.a.String(); got != tc.want {
			t.Fatalf("%d: got %q want %q", tc.a, got, tc.want)
		}
	}
}

func TestNegotiateDragAction(t *testing.T) {
	all := DragCopy | DragMove | DragLink
	for _, tc := range []struct {
		name                        string
		offered, preferred, allowed DragAction
		want                        DragAction
	}{
		{"the source's preference wins", all, DragMove, all, DragMove},
		{"a preference the target refuses falls back", all, DragMove, DragCopy, DragCopy},
		{"no preference takes the best of both", all, DragNone, DragMove | DragLink, DragMove},
		{"nothing in common refuses", DragMove, DragMove, DragCopy, DragNone},
		{"a source offering nothing refuses", DragNone, DragNone, all, DragNone},
		{"a target allowing nothing refuses", all, DragCopy, DragNone, DragNone},
		{"one action both sides allow", DragCopy, DragNone, all, DragCopy},
		// A preference outside the offered set is meaningless; the
		// negotiation still has to land on something both allow.
		{"a preference not offered is ignored", DragCopy, DragLink, all, DragCopy},
	} {
		got := NegotiateDragAction(tc.offered, tc.preferred, tc.allowed)
		if got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
