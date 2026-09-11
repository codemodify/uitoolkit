package layout

import "testing"

func TestClampScrollStopsAtEnd(t *testing.T) {
	if got := MaxScroll(100, 40); got != 60 {
		t.Fatalf("max %v", got)
	}
	if got := MaxScroll(20, 40); got != 0 {
		t.Fatalf("short content max %v", got)
	}
	if got := ClampScroll(-10, 100, 40); got != 0 {
		t.Fatalf("neg %v", got)
	}
	if got := ClampScroll(1000, 100, 40); got != 60 {
		t.Fatalf("past end %v", got)
	}
	if got := ClampScroll(30, 100, 40); got != 30 {
		t.Fatalf("mid %v", got)
	}
	if got := ClampScroll(12, 20, 40); got != 0 {
		t.Fatalf("fits %v", got)
	}
}
