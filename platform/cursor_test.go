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

func TestCursorString(t *testing.T) {
	if CursorDefault.String() != "default" || CursorColResize.String() != "col-resize" {
		t.Fatal(CursorDefault.String(), CursorColResize.String())
	}
	if CursorRowResize.String() != "row-resize" || CursorText.String() != "text" {
		t.Fatal(CursorRowResize.String(), CursorText.String())
	}
}

func TestWaylandCursorShapeMap(t *testing.T) {
	if waylandCursorShape(CursorDefault) != wlShapeDefault {
		t.Fatalf("default %d", waylandCursorShape(CursorDefault))
	}
	if waylandCursorShape(CursorColResize) != wlShapeColResize {
		t.Fatalf("col %d", waylandCursorShape(CursorColResize))
	}
	if waylandCursorShape(CursorRowResize) != wlShapeRowResize {
		t.Fatalf("row %d", waylandCursorShape(CursorRowResize))
	}
	if waylandCursorShape(CursorText) != wlShapeText {
		t.Fatalf("text %d", waylandCursorShape(CursorText))
	}
}

func TestHostCursorNames(t *testing.T) {
	wl := waylandThemeCursorNames(CursorColResize)
	if len(wl) < 2 || wl[0] != "col-resize" {
		t.Fatalf("wayland col %v", wl)
	}
	x11 := x11ThemeCursorNames(CursorText)
	if x11[0] != "xterm" {
		t.Fatalf("x11 text %v", x11)
	}
	if x11FontCursorShape(CursorDefault) != 68 || x11FontCursorShape(CursorColResize) != 108 {
		t.Fatalf("font shapes %d %d", x11FontCursorShape(CursorDefault), x11FontCursorShape(CursorColResize))
	}
	if x11FontCursorShape(CursorRowResize) != 116 || x11FontCursorShape(CursorText) != 152 {
		t.Fatalf("font shapes row/text")
	}
	if win32CursorID(CursorDefault) != winIDCArrow || win32CursorID(CursorColResize) != winIDCSizeWE {
		t.Fatal("win32 ids")
	}
	if win32CursorID(CursorRowResize) != winIDCSizeNS || win32CursorID(CursorText) != winIDCIbeam {
		t.Fatal("win32 text/row")
	}
	if darwinCursorKind(CursorDefault) != 0 || darwinCursorKind(CursorColResize) != 1 {
		t.Fatal("darwin kinds")
	}
	if darwinCursorKind(CursorRowResize) != 2 || darwinCursorKind(CursorText) != 3 {
		t.Fatal("darwin text/row")
	}
}
