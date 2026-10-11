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

// **The pointing hand says "this can be clicked."** It is the shape a
// link has in every toolkit (CSS pointer, GTK pointer, Qt's
// PointingHandCursor, macOS pointingHandCursor, IDC_HAND), and the only
// thing a page with nothing on it but a picture can say so with. Every
// backend has to reach the desktop's own hand, not fall back to an arrow.
func TestThePointingHandIsOnEveryBackend(t *testing.T) {
	if CursorPointer.String() != "pointer" {
		t.Errorf("the shape is named %q", CursorPointer.String())
	}
	if got := waylandCursorShape(CursorPointer); got != wlShapePointer {
		t.Errorf("cursor-shape-v1 %d, want the pointer shape %d", got, wlShapePointer)
	}
	// A theme that predates the CSS names still has hand2: the X11 list
	// asks for that first, the Wayland one for the modern name.
	wl, x11 := waylandThemeCursorNames(CursorPointer), x11ThemeCursorNames(CursorPointer)
	if len(wl) == 0 || wl[0] != "pointer" {
		t.Errorf("wayland theme names %v", wl)
	}
	if len(x11) == 0 || x11[0] != "hand2" {
		t.Errorf("x11 theme names %v", x11)
	}
	if got := x11FontCursorShape(CursorPointer); got != 58 {
		t.Errorf("X cursor font shape %d, want XC_hand2 (58)", got)
	}
	if got := win32CursorID(CursorPointer); got != winIDCHand {
		t.Errorf("win32 id %d, want IDC_HAND %d", got, winIDCHand)
	}
	if got := darwinCursorKind(CursorPointer); got != akCursorPointingHand {
		t.Errorf("NSCursor kind %d, want the pointing hand %d", got, akCursorPointingHand)
	}
}
