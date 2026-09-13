package widgets

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

type benchHost struct{ look style.LookAndFeel }

func (h *benchHost) Invalidate(widget.Component, paintengine2d.Rect) {}
func (h *benchHost) RequestFocus(widget.Component)                   {}
func (h *benchHost) Focus() widget.Component                         { return nil }
func (h *benchHost) Scale() float32                                  { return 1 }
func (h *benchHost) Look() style.LookAndFeel                         { return h.look }
func (h *benchHost) RequestLayout()                                  {}

func benchTable(b *testing.B) *TableView {
	b.Helper()
	cols := []TableColumn{{Title: "From", Width: 160}, {Title: "Subject", Width: 280}, {Title: "Date", Width: 120}}
	t := NewTableView(cols, 400,
		func(r, c int) string { return fmt.Sprintf("cell %d/%d text", r, c) }, nil)
	t.SetHost(&benchHost{look: style.DarkLook()})
	t.Measure(layout.Tight(600, 400))
	t.Arrange(paintengine2d.XYWH(0, 0, 600, 400))
	return t
}

func recordOnce(t widget.Component, w, h int) {
	rec := paintengine2d.NewRecorder(w, h)
	ctx := paintengine2d.NewContextDevice(rec)
	widget.RecordTree(t, rec, ctx, nil, nil, true)
	rec.Finish()
}

func BenchmarkTableRowsAtRest(b *testing.B) {
	t := benchTable(b)
	recordOnce(t, 600, 400)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recordOnce(t, 600, 400)
	}
}

func BenchmarkTableRowsScrolling(b *testing.B) {
	t := benchTable(b)
	recordOnce(t, 600, 400)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t.OffsetY = float32(i%200) * 3
		recordOnce(t, 600, 400)
	}
}

func BenchmarkListRowsScrolling(b *testing.B) {
	l := NewListView(400, func(i int) string { return fmt.Sprintf("row %02d body text", i) }, nil)
	l.SetHost(&benchHost{look: style.DarkLook()})
	l.Measure(layout.Tight(320, 400))
	l.Arrange(paintengine2d.XYWH(0, 0, 320, 400))
	recordOnce(l, 320, 400)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.OffsetY = float32(i%200) * 3
		recordOnce(l, 320, 400)
	}
}
