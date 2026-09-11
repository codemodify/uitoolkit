package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func TestTableViewSelectSortKeys(t *testing.T) {
	sorted := 0
	var asc bool
	tv := NewTableView([]TableColumn{
		{Title: "Name", Sortable: true},
		{Title: "N", Width: 48, Sortable: true, Align: 2},
	}, 8, func(row, col int) string {
		if col == 1 {
			return "x"
		}
		return "r"
	}, nil)
	tv.OnSort = func(col int, a bool) {
		sorted = col
		asc = a
	}
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 240, 160))
	tv.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(20, tv.headerH()+10)})
	if tv.Selected != 0 {
		t.Fatalf("select %d", tv.Selected)
	}
	tv.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(20, 8)})
	if tv.SortCol != 0 || !tv.SortAsc || sorted != 0 || !asc {
		t.Fatalf("sort col=%d asc=%v cb=%d %v", tv.SortCol, tv.SortAsc, sorted, asc)
	}
	tv.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(20, 8)})
	if tv.SortAsc {
		t.Fatal("toggle desc")
	}
	tv.KeyPress(widget.KeyEvent{Key: platform.KeyEnd})
	if tv.Selected != 7 {
		t.Fatalf("end %d", tv.Selected)
	}
	tv.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	if tv.Selected != 6 {
		t.Fatalf("up %d", tv.Selected)
	}
}

func TestTableViewFlagCellRepaintsOnInvalidate(t *testing.T) {
	star := ""
	tv := NewTableView([]TableColumn{
		{Title: "★", Width: 28, MinWidth: 24},
		{Title: "Topic", MinWidth: 80},
	}, 1, func(row, col int) string {
		if col == 0 {
			return star
		}
		return "hello"
	}, nil)
	tv.SetLook(style.DarkLook())
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 240, 80))

	paint := func() int {
		rec := paintengine2d.NewRecorder(240, 80)
		ctx := paintengine2d.NewContextDevice(rec)
		tv.Paint(ctx)
		return rec.Finish().Reused
	}
	if n := paint(); n != 0 {
		t.Fatalf("first record reused=%d", n)
	}
	star = "★"
	tv.Invalidate()
	if n := paint(); n != 0 {
		t.Fatalf("Invalidate after flag change reused stale row %d", n)
	}

	img := paintengine2d.NewImage(240, 80)
	tv.Paint(paintengine2d.NewContext(img))
	n := 0
	hh := int(tv.headerH()) + 4
	for y := hh; y < hh+20 && y < img.Height; y++ {
		for x := 2; x < 26; x++ {
			_, _, _, a := img.PremulAt(x, y)
			if a > 20 {
				n++
			}
		}
	}
	if n < 8 {
		t.Fatalf("star cell empty after toggle+Invalidate, ink=%d", n)
	}
}

func TestNumberFieldStepAndFilter(t *testing.T) {
	n := 0.0
	nf := NewNumberField(0, 10, 4, 1, func(v float64) { n = v })
	nf.SetHost(&host{})
	nf.Arrange(paintengine2d.XYWH(0, 0, 140, 34))
	nf.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	if nf.Value != 5 || n != 5 {
		t.Fatalf("up %v n=%v", nf.Value, n)
	}
	nf.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	if nf.Value != 0 {
		t.Fatalf("home %v", nf.Value)
	}
	nf.Field().SetSelection(0, 1)
	nf.Field().TextInput('3')
	if nf.Field().Text != "3" {
		t.Fatalf("type %q", nf.Field().Text)
	}
	nf.Field().TextInput('x')
	if nf.Field().Text != "3" {
		t.Fatalf("reject %q", nf.Field().Text)
	}
	nf.Field().TextInput('.')
	if nf.Field().Text != "3" {
		t.Fatalf("int reject dot %q", nf.Field().Text)
	}
	nf.SetValue(99)
	if nf.Value != 10 {
		t.Fatalf("clamp %v", nf.Value)
	}
	sb := nf.spinnerBox()
	nf.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(sb.Min.X+4, sb.Min.Y+4)})
	if nf.Value != 10 {
		t.Fatalf("nudge at max %v", nf.Value)
	}
	sz := nf.Measure(layout.Loose(200, 40))
	if sz.Y < 20 || sz.X < 40 {
		t.Fatalf("size %v", sz)
	}
	if NewSpinner(0, 1, 0, 0.5, nil).Decimals != 2 {
		t.Fatal("float spinner decimals")
	}
}

func TestTextFieldAccept(t *testing.T) {
	tf := NewTextField("12", "", nil)
	tf.Accept = func(s string) bool {
		for _, r := range s {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	tf.SetHost(&host{})
	tf.Arrange(paintengine2d.XYWH(0, 0, 120, 32))
	tf.TextInput('3')
	if tf.Text != "123" {
		t.Fatalf("%q", tf.Text)
	}
	tf.TextInput('a')
	if tf.Text != "123" {
		t.Fatalf("accept %q", tf.Text)
	}
}

func TestFileDialogStubPick(t *testing.T) {
	got := ""
	cancel := 0
	fd := NewFileDialog(FileDialogOptions{
		Title:  "Open project",
		Path:   "/stub/project",
		Filter: "*.md",
		Entries: []FileInfo{
			{Name: "docs", Dir: true},
			{Name: "README.md"},
			{Name: "export.go"},
		},
		OnPick:   func(p string) { got = p },
		OnCancel: func() { cancel++ },
	})
	ents := fd.Entries()
	if len(ents) != 2 { // folder + md; go filtered
		t.Fatalf("filter %d %+v", len(ents), ents)
	}
	fd.table.Selected = 1
	fd.finish(true)
	if got == "" || got == "/stub/project" && fd.Path() == "" {
		t.Fatalf("pick %q path=%q", got, fd.Path())
	}
	if cancel != 0 {
		t.Fatalf("cancel %d", cancel)
	}

	fd2 := NewFileDialog(FileDialogOptions{
		Path:     "/no/such",
		Entries:  stubEntries(),
		OnCancel: func() { cancel++ },
	})
	fd2.finish(false)
	if cancel != 1 {
		t.Fatalf("cancel cb %d", cancel)
	}
}

func TestTipWrapReports(t *testing.T) {
	inner := NewLabel("X")
	wrap := NewTip("Helpful", inner)
	if wrap.Tooltip() != "Helpful" {
		t.Fatal(wrap.Tooltip())
	}
	sz := wrap.Measure(layout.Loose(200, 40))
	if sz.X < 2 {
		t.Fatalf("measure %v", sz)
	}
	b := NewButton("Save", nil)
	b.Tip = "Write file"
	if b.Tooltip() != "Write file" {
		t.Fatal("button tip")
	}
}

func TestReadDirEntriesWorkspace(t *testing.T) {
	ents, err := ReadDirEntries(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) < 3 {
		t.Fatalf("entries %d", len(ents))
	}
}
