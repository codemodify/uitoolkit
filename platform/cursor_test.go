package platform

import "testing"

func TestOffscreenSetCursor(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 40, Height: 30})
	if o.Cursor() != CursorDefault {
		t.Fatalf("start %v", o.Cursor())
	}
	SetCursor(o, CursorColResize)
	if o.Cursor() != CursorColResize {
		t.Fatalf("col %v", o.Cursor())
	}
	SetCursor(o, CursorDefault)
	if o.Cursor() != CursorDefault {
		t.Fatalf("restore %v", o.Cursor())
	}
}
