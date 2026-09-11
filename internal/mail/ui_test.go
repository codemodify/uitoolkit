package mail

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestMailAppPaints(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	PrepareShot(w, -1)
	a.PumpOnce()

	var tables, trees, areas, bars, menus int
	widget.Walk(w.Content(), func(c widget.Component) {
		switch c.(type) {
		case *widgets.TableView:
			tables++
		case *widgets.TreeView:
			trees++
		case *widgets.TextArea:
			areas++
		case *widgets.StatusBar:
			bars++
		case *widgets.MenuBar:
			menus++
		}
	})
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && tv.CellText != nil && tv.RowCount > 0 {
			subj := tv.CellText(0, 2)
			t.Logf("row0 subject %q selected=%d rows=%d", subj, tv.Selected, tv.RowCount)
			if subj == "" {
				t.Error("empty first subject")
			}
		}
	})
	if tables < 1 || trees < 1 || areas < 1 || bars < 1 || menus < 1 {
		t.Fatalf("chrome table=%d tree=%d area=%d status=%d menu=%d", tables, trees, areas, bars, menus)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "mail.png")
	if err := w.WritePNG(path); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() < 2000 {
		t.Fatalf("png too small: %d", st.Size())
	}
	w.Close()
}

func TestMailHiDPIResizeStable(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Scale: 2, Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()

	var table *widgets.TableView
	var tree *widgets.TreeView
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && table == nil {
			table = tv
		}
		if tr, ok := c.(*widgets.TreeView); ok && tree == nil {
			tree = tr
		}
	})
	if table == nil || tree == nil {
		t.Fatal("missing table/tree")
	}
	font0 := table.Look().Font().Size
	row0 := table.Look().Metrics().RowH
	fit := style.FittedRowHeight(table.Look(), table.RowHeight)
	if font0 < 20 || fit < table.Look().Font().Height() {
		t.Fatalf("hidpi font %v row metric %v fitted %v", font0, row0, fit)
	}

	w.Inject(platform.Event{Kind: platform.EventResize, Width: 900, Height: 620})
	a.PumpOnce()
	w.Inject(platform.Event{Kind: platform.EventResize, Width: 1400, Height: 900})
	a.PumpOnce()
	w.Inject(platform.Event{Kind: platform.EventResize, Width: 1100, Height: 700})
	a.PumpOnce()

	if table.Look().Font().Size != font0 {
		t.Fatalf("font exploded %v -> %v", font0, table.Look().Font().Size)
	}
	if table.Look().Metrics().RowH != row0 {
		t.Fatalf("row metric %v -> %v", row0, table.Look().Metrics().RowH)
	}
	if tree.Look().Font().Size != font0 {
		t.Fatalf("tree font exploded %v", tree.Look().Font().Size)
	}
	w.Close()
}

func TestMailSubjectColumnFillsThreadPane(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	var table *widgets.TableView
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && len(tv.Columns) >= 6 && table == nil {
			table = tv
		}
	})
	if table == nil {
		t.Fatal("thread table")
	}
	ws := table.ColumnWidths()
	if len(ws) < 6 {
		t.Fatalf("cols %v", ws)
	}
	subj := ws[2]
	if subj < 200 {
		t.Fatalf("subject column too narrow at 1280: %v (all %v) table=%v", subj, ws, table.LocalBounds().Dx())
	}
	sum := float32(0)
	for _, x := range ws {
		sum += x
	}
	if d := sum - table.LocalBounds().Dx(); d > 1 || d < -1 {
		t.Fatalf("columns %v sum %v != table %v", ws, sum, table.LocalBounds().Dx())
	}
	w.Inject(platform.Event{Kind: platform.EventResize, Width: 1100, Height: 720})
	a.PumpOnce()
	ws = table.ColumnWidths()
	if ws[2] < 160 {
		t.Fatalf("subject after resize %v (all %v)", ws[2], ws)
	}
	w.Close()
}

func TestComposeAppPaints(t *testing.T) {
	sock, stop, err := StartDemo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	cli, err := DialWait(sock, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	a := uitoolkit.New(uitoolkit.Options{Look: style.LightLook(), Headless: true})
	w, err := OpenCompose(a, cli, ComposeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()
	var body *widgets.TextArea
	widget.Walk(w.Content(), func(c widget.Component) {
		if ta, ok := c.(*widgets.TextArea); ok && !ta.ReadOnly {
			body = ta
		}
	})
	if body == nil {
		t.Fatal("compose body should be an editable TextArea")
	}
	if body.ReadOnly {
		t.Fatal("compose must stay editable")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.png")
	if err := w.WritePNG(path); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() < 1000 {
		t.Fatalf("compose png too small: %d", st.Size())
	}
	w.Close()
}

func TestMailPreviewReadOnlyAndListClamp(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	PrepareShot(w, -1)
	a.PumpOnce()

	var table *widgets.TableView
	var views []*widgets.TextArea
	var splits []*widgets.Splitter
	widget.Walk(w.Content(), func(c widget.Component) {
		switch t := c.(type) {
		case *widgets.TableView:
			if len(t.Columns) >= 5 && table == nil {
				table = t
			}
		case *widgets.TextArea:
			views = append(views, t)
		case *widgets.Splitter:
			splits = append(splits, t)
		}
	})
	if table == nil {
		t.Fatal("thread table")
	}
	if len(views) < 1 {
		t.Fatal("message TextView")
	}
	for _, ta := range views {
		if !ta.ReadOnly {
			t.Fatalf("mail view TextArea must be ReadOnly, got editable %q", ta.Placeholder)
		}
		before := ta.Text
		if ta.TextInput('Z') {
			t.Fatal("preview accepted typing")
		}
		if ta.Text != before {
			t.Fatalf("preview mutated %q -> %q", before, ta.Text)
		}
	}

	for i := 0; i < 80; i++ {
		table.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 40)})
	}
	if table.OffsetY > table.MaxOffset()+0.5 {
		t.Fatalf("thread list scrolled past end offset=%v max=%v", table.OffsetY, table.MaxOffset())
	}
	if table.MaxOffset() > 0 && table.OffsetY != table.MaxOffset() {
		t.Fatalf("wheel spam should pin to max, offset=%v max=%v", table.OffsetY, table.MaxOffset())
	}

	if len(splits) == 0 {
		t.Fatal("expected splitters")
	}
	for _, s := range splits {
		a, b := s.PaneA(), s.PaneB()
		if a.Overlaps(b) {
			t.Fatalf("splitter panes overlap A=%+v B=%+v", a, b)
		}
		pa, pb := s.PaneA(), s.PaneB()
		var sash, mid paintengine2d.Point
		if s.Vertical {
			sash = paintengine2d.Pt((pa.Max.X+pb.Min.X)*0.5, 8)
			mid = paintengine2d.Pt(s.LocalBounds().Dx()*0.75, 8)
		} else {
			sash = paintengine2d.Pt(8, (pa.Max.Y+pb.Min.Y)*0.5)
			mid = paintengine2d.Pt(8, s.LocalBounds().Dy()*0.75)
		}
		s.MousePress(widget.MouseEvent{Pos: sash, Button: platform.ButtonLeft})
		s.MouseMove(widget.MouseEvent{Pos: mid, Button: platform.ButtonLeft})
		s.MouseRelease(widget.MouseEvent{Pos: mid})
		a, b = s.PaneA(), s.PaneB()
		if a.Overlaps(b) {
			t.Fatalf("after drag overlap A=%+v B=%+v", a, b)
		}
	}
	w.Close()
}

func TestWriteScreenshotsDistinct(t *testing.T) {
	dir := t.TempDir()
	if err := WriteScreenshots(dir); err != nil {
		t.Fatal(err)
	}
	seen := map[int64]string{}
	for _, name := range ScreenshotNames {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if st.Size() < 200 {
			t.Fatalf("%s too small: %d", name, st.Size())
		}
		if other, ok := seen[st.Size()]; ok {
			t.Fatalf("%s same size as %s (%d) — likely duplicate", name, other, st.Size())
		}
		seen[st.Size()] = name
	}
}

func TestFolderTreeOmitsVirtualSections(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()

	var tree *widgets.TreeView
	widget.Walk(w.Content(), func(c widget.Component) {
		if tr, ok := c.(*widgets.TreeView); ok && tree == nil {
			tree = tr
		}
	})
	if tree == nil {
		t.Fatal("missing tree")
	}
	var walk func([]*widgets.TreeNode)
	walk = func(nodes []*widgets.TreeNode) {
		for _, n := range nodes {
			if n == nil {
				continue
			}
			if n.Label == "Unified Folders" || n.Label == "Smart Folders" || n.Label == "Categories" {
				t.Fatalf("removed section still in tree: %q", n.Label)
			}
			walk(n.Children)
		}
	}
	walk(tree.Roots)

	banned := []string{"Unified Inbox", "New Smart Folder", "Smart Folders", "Move sender to Primary"}
	widget.Walk(w.Content(), func(c widget.Component) {
		mb, ok := c.(*widgets.MenuBar)
		if !ok {
			return
		}
		for _, m := range mb.Menus() {
			for _, it := range m.Items {
				for _, bad := range banned {
					if it.Text == bad {
						t.Fatalf("menu still has %q", bad)
					}
				}
			}
		}
	})
	w.Close()
}

func TestSelectAccountKeepsInboxList(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()

	var tree *widgets.TreeView
	var table *widgets.TableView
	widget.Walk(w.Content(), func(c widget.Component) {
		if tr, ok := c.(*widgets.TreeView); ok && tree == nil {
			tree = tr
		}
		if tv, ok := c.(*widgets.TableView); ok && table == nil {
			table = tv
		}
	})
	if tree == nil || table == nil {
		t.Fatal("missing table/tree")
	}
	if table.RowCount < 1 {
		t.Fatal("inbox should show messages")
	}
	var acct *widgets.TreeNode
	for _, n := range tree.Roots {
		if n == nil {
			continue
		}
		if id, ok := n.Data.(string); ok && id != "" && id != AccountTags {
			acct = n
			break
		}
	}
	if acct == nil {
		t.Fatal("no account node")
	}
	before := table.RowCount
	if tree.OnSelect != nil {
		tree.OnSelect(acct)
	}
	a.PumpOnce()
	if table.RowCount == 0 {
		t.Fatalf("selecting account wiped the list (had %d)", before)
	}
	if !table.Visible() {
		t.Fatal("thread list hidden after account select")
	}
	w.Close()
}

func TestClickUnreadKeepsList(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()

	var table *widgets.TableView
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && table == nil {
			table = tv
		}
	})
	if table == nil || table.RowCount < 1 {
		t.Fatal("thread table")
	}
	unread := -1
	for i := 0; i < table.RowCount; i++ {
		if table.CellBold != nil && table.CellBold(i, 2) {
			unread = i
			break
		}
	}
	if unread < 0 {
		unread = 0
	}
	before := table.RowCount
	if table.OnSelect != nil {
		table.OnSelect(unread)
	}
	a.PumpOnce()
	if table.RowCount == 0 {
		t.Fatal("clicking a message wiped the list")
	}
	if table.RowCount != before {
		t.Fatalf("row count %d -> %d after click", before, table.RowCount)
	}
	w.Close()
}

func TestHiddenFromFolderTree(t *testing.T) {
	if !HiddenFromFolderTree(FolderUnifiedInbox) || !HiddenFromFolderTree(FolderUnifiedUnread) {
		t.Fatal("unified")
	}
	if !HiddenFromFolderTree(FolderCatPrimary) || !HiddenFromFolderTree(FolderID("smart/sf-invoices")) {
		t.Fatal("cat/smart")
	}
	if HiddenFromFolderTree(FolderVIP) || HiddenFromFolderTree(FolderOutbox) || HiddenFromFolderTree(TagFolderID("Work")) {
		t.Fatal("vip/outbox/tag should stay")
	}
}
