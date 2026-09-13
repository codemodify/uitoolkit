package mail

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
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
	if tables < 1 || trees < 1 || areas < 1 || menus < 1 {
		t.Fatalf("chrome table=%d tree=%d area=%d status=%d menu=%d", tables, trees, areas, bars, menus)
	}
	if bars != 0 {
		t.Fatalf("status bar should be hidden, got %d", bars)
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
		if tv, ok := c.(*widgets.TableView); ok && len(tv.Columns) >= 5 && table == nil {
			table = tv
		}
	})
	if table == nil {
		t.Fatal("thread table")
	}
	ws := table.ColumnWidths()
	if len(ws) != 5 {
		t.Fatalf("cols %v", ws)
	}
	subj := ws[2]
	if subj < 200 {
		t.Fatalf("topic column too narrow at 1280: %v (all %v) table=%v", subj, ws, table.LocalBounds().Dx())
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
		t.Fatalf("topic after resize %v (all %v)", ws[2], ws)
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
	assertOutboxPinnedBottom(t, w.Content())

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
	if !HiddenFromFolderTree(FolderVIP) {
		t.Fatal("VIP folder should be hidden from the tree")
	}
	if HiddenFromFolderTree(FolderOutbox) || HiddenFromFolderTree(TagFolderID("Work")) {
		t.Fatal("outbox/tag should stay")
	}
}

func TestMailChromeHidesStatusBarAndVIPFolder(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	var bars, titles int
	var vipNode bool
	var qf widget.Component
	var split *widgets.Splitter
	widget.Walk(w.Content(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.StatusBar:
			bars++
		case *widgets.TitleBar:
			titles++
		case *widgets.Splitter:
			if split == nil {
				split = v
			}
		case *widgets.TreeView:
			var walk func([]*widgets.TreeNode)
			walk = func(nodes []*widgets.TreeNode) {
				for _, n := range nodes {
					if n == nil {
						continue
					}
					if id, ok := n.Data.(FolderID); ok && id == FolderVIP {
						vipNode = true
					}
					if n.Label == "VIP" || strings.HasPrefix(n.Label, "VIP (") {
						vipNode = true
					}
					walk(n.Children)
				}
			}
			walk(v.Roots)
		}
	})
	if bars != 0 {
		t.Fatalf("status bar widgets=%d", bars)
	}
	if titles != 0 {
		t.Fatalf("path/subtitle TitleBar should be gone, got %d", titles)
	}
	if vipNode {
		t.Fatal("VIP folder still in the sidebar tree")
	}
	qf = findQuickFilter(w.Content())
	if qf == nil {
		t.Fatal("quick filter field missing")
	}
	if qf.Visible() {
		t.Fatal("quick filter field should stay hidden until the Filter icon or Ctrl+F")
	}
	assertNoUnreadFolderFooter(t, w.Content())
	if split == nil || !widget.Contains(split, qf) {
		t.Fatal("quick filter should sit on the list toolbar row inside the splitter")
	}
	assertMailToolChrome(t, w.Content(), qf)
	w.Close()
}

func TestMailToolBarChrome(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	qf := findQuickFilter(w.Content())
	if qf == nil {
		t.Fatal("quick filter field missing")
	}
	assertMailToolChrome(t, w.Content(), qf)
	w.Close()
}

func TestMailFilterIconTogglesField(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	qf := findQuickFilter(w.Content())
	var btn *widgets.ToolItem
	widget.Walk(w.Content(), func(c widget.Component) {
		if v, ok := c.(*widgets.ToolBar); ok {
			if it := filterIconItem(v); it != nil {
				btn = it
			}
		}
	})
	if qf == nil || btn == nil || btn.OnClick == nil {
		t.Fatal("filter icon / field")
	}
	if qf.Visible() {
		t.Fatal("field should start hidden")
	}
	qf.SetText("lunch")
	btn.OnClick()
	a.PumpOnce()
	if !qf.Visible() {
		t.Fatal("Filter icon should reveal the field")
	}
	if qf.Text != "lunch" {
		t.Fatalf("should keep filter text, got %q", qf.Text)
	}
	if qf.Bounds().Dx() <= 100 {
		t.Fatalf("opened field crushed: width=%v", qf.Bounds().Dx())
	}
	if qf.OnEscape == nil {
		t.Fatal("Escape should hide the field")
	}
	qf.OnEscape()
	a.PumpOnce()
	if qf.Visible() {
		t.Fatal("Escape should hide the field")
	}
	if qf.Text != "lunch" {
		t.Fatalf("hide should keep text, got %q", qf.Text)
	}
	w.Close()
}

func TestMailCollapsedFiltersDoesNotStealOutbox(t *testing.T) {
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
		if v, ok := c.(*widgets.TreeView); ok && tree == nil {
			tree = v
		}
	})
	if tree == nil {
		t.Fatal("folder tree")
	}
	filters := findTreeLabel(tree.Roots, "Filters")
	if filters == nil {
		t.Fatal("Filters")
	}
	tree.Toggle(filters)
	if filters.Expanded {
		t.Fatal("Filters should collapse")
	}
	_, pin := mailSidebarTrees(w.Content())
	if pin == nil || pin.OnSelect == nil || len(pin.Roots) == 0 {
		t.Fatal("Outbox pin")
	}
	pin.OnSelect(pin.Roots[0])
	a.PumpOnce()
	filters = findTreeLabel(tree.Roots, "Filters")
	if filters == nil {
		t.Fatal("Filters after rebuild")
	}
	if filters.Expanded {
		t.Fatal("Outbox click re-opened Filters (rebuild lost collapse, or hit-test stole the row)")
	}
	_, pin = mailSidebarTrees(w.Content())
	if pin == nil || pin.Selected == nil || !strings.HasPrefix(pin.Selected.Label, "Outbox") {
		label := ""
		if pin != nil && pin.Selected != nil {
			label = pin.Selected.Label
		}
		t.Fatalf("selected %q want Outbox", label)
	}
	w.Close()
}

func findTreeLabel(nodes []*widgets.TreeNode, label string) *widgets.TreeNode {
	var found *widgets.TreeNode
	var walk func([]*widgets.TreeNode)
	walk = func(ns []*widgets.TreeNode) {
		for _, n := range ns {
			if n == nil || found != nil {
				continue
			}
			if n.Label == label {
				found = n
				return
			}
			walk(n.Children)
		}
	}
	walk(nodes)
	return found
}

func visibleTreeRows(roots []*widgets.TreeNode) []*widgets.TreeNode {
	var out []*widgets.TreeNode
	var walk func([]*widgets.TreeNode)
	walk = func(nodes []*widgets.TreeNode) {
		for _, n := range nodes {
			if n == nil {
				continue
			}
			out = append(out, n)
			if n.Expanded {
				walk(n.Children)
			}
		}
	}
	walk(roots)
	return out
}

func assertMailToolChrome(t *testing.T, root widget.Component, qf widget.Component) {
	t.Helper()
	var split *widgets.Splitter
	var table *widgets.TableView
	var mainBar, listBar, filterBar *widgets.ToolBar
	var tree *widgets.TreeView
	widget.Walk(root, func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.Splitter:
			if split == nil {
				split = v
			}
		case *widgets.TableView:
			if table == nil && len(v.Columns) >= 5 {
				table = v
			}
		case *widgets.TreeView:
			if tree == nil {
				tree = v
			}
		case *widgets.ToolBar:
			texts := toolTexts(v)
			if texts["Classic"] || texts["Get Messages"] || texts["Write"] {
				mainBar = v
			}
			if texts["Tag"] && texts["Archive"] && texts["Junk"] && texts["Delete"] {
				listBar = v
			}
			if filterIconItem(v) != nil {
				filterBar = v
			}
			for _, name := range []string{"Unread", "Starred", "Attachment", "From", "To", "Subject", "Body"} {
				if texts[name] {
					t.Fatalf("filter pin %q still on a toolbar", name)
				}
			}
		case *widgets.ComboBox:
			t.Fatal("Tags ComboBox should be gone from Mail chrome")
		}
	})
	if mainBar == nil {
		t.Fatal("main toolbar")
	}
	if listBar == nil {
		t.Fatal("list toolbar (Tag / Archive / Junk / Delete)")
	}
	if filterBar == nil {
		t.Fatal("icon-only Filter button missing after Delete")
	}
	if split == nil || table == nil {
		t.Fatal("splitter/table")
	}
	main := toolTexts(mainBar)
	for _, name := range []string{"Reply", "Forward", "Tag", "Archive", "Junk", "Delete"} {
		if main[name] {
			t.Fatalf("main toolbar still has %s", name)
		}
	}
	if !main["Write"] || !main["Classic"] {
		t.Fatalf("main toolbar missing Write/Classic: %v", main)
	}
	if widget.Contains(mainBar, qf) {
		t.Fatal("filter field should sit after Delete on the list toolbar, not the main toolbar")
	}
	if !widget.Contains(split, listBar) {
		t.Fatal("list toolbar should sit in the thread pane")
	}
	if !widget.Contains(split, qf) || !widget.Contains(split, filterBar) {
		t.Fatal("quick filter chrome should sit on the list toolbar row inside the splitter")
	}
	if !filterAfterListActions(root, filterBar) {
		t.Fatal("Filter button should share the list toolbar row after Delete (right-aligned)")
	}
	if separateFilterRow(root) {
		t.Fatal("separate Quick Filter row still under the main toolbar")
	}
	if main["Quick Filter"] {
		t.Fatal("left Quick Filter visibility toggle should be gone (View menu / Ctrl+F)")
	}
	barOrigin := widget.DeviceOrigin(listBar)
	iconOrigin := widget.DeviceOrigin(filterBar)
	if iconOrigin.X+0.5 < barOrigin.X+listBar.Bounds().Dx() {
		t.Fatalf("Filter icon should sit after Delete: bar=%v..%v icon=%v",
			barOrigin.X, barOrigin.X+listBar.Bounds().Dx(), iconOrigin.X)
	}
	if qf.Visible() {
		t.Fatal("quick filter field should be hidden until the Filter icon opens it")
	}
	assertFiltersTree(t, tree)
}

func findQuickFilter(root widget.Component) *widgets.TextField {
	var qf *widgets.TextField
	walkAll(root, func(c widget.Component) {
		v, ok := c.(*widgets.TextField)
		if ok && qf == nil && strings.Contains(v.Placeholder, "Quick Filter") {
			qf = v
		}
	})
	return qf
}

func walkAll(c widget.Component, fn func(widget.Component)) {
	if c == nil {
		return
	}
	fn(c)
	for _, ch := range c.Children() {
		walkAll(ch, fn)
	}
}

func filterIconItem(bar *widgets.ToolBar) *widgets.ToolItem {
	if bar == nil {
		return nil
	}
	for _, it := range bar.Items() {
		if it != nil && it.Text == "" && it.Icon == style.IconSearch {
			return it
		}
	}
	return nil
}

func toolTexts(bar *widgets.ToolBar) map[string]bool {
	out := map[string]bool{}
	if bar == nil {
		return out
	}
	for _, it := range bar.Items() {
		if it != nil && it.Text != "" {
			out[it.Text] = true
		}
	}
	return out
}

func filterAfterListActions(root, qf widget.Component) bool {
	var row *widgets.FlexBox
	widget.Walk(root, func(c widget.Component) {
		f, ok := c.(*widgets.FlexBox)
		if !ok || row != nil {
			return
		}
		var hasList, hasQF bool
		widget.Walk(f, func(ch widget.Component) {
			if bar, ok := ch.(*widgets.ToolBar); ok {
				texts := toolTexts(bar)
				if texts["Delete"] && texts["Tag"] {
					hasList = true
				}
			}
			if ch == qf {
				hasQF = true
			}
		})
		if hasList && hasQF {
			row = f
		}
	})
	return row != nil
}

func assertFiltersTree(t *testing.T, tree *widgets.TreeView) {
	t.Helper()
	if tree == nil {
		t.Fatal("folder tree")
	}
	var filters *widgets.TreeNode
	for _, n := range tree.Roots {
		if n == nil {
			continue
		}
		if n.Label == "Tags" {
			t.Fatal("sidebar Tags node should be renamed Filters")
		}
		if n.Label == "Filters" || n.Data == AccountTags {
			filters = n
		}
	}
	if filters == nil || filters.Label != "Filters" {
		t.Fatal("Filters missing from the tree")
	}
	want := []string{"Unread", "Starred", "Attachment"}
	ban := []string{"From", "To", "Subject", "Body", "Work", "Personal", "Later"}
	got := map[string]bool{}
	var important bool
	for _, n := range filters.Children {
		if n == nil {
			continue
		}
		label := strings.TrimPrefix(n.Label, "✓ ")
		if i := strings.Index(label, " ("); i > 0 {
			label = label[:i]
		}
		got[label] = true
		if label == "Important" {
			important = true
		}
	}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("Filters tree missing pin %q", name)
		}
	}
	for _, name := range ban {
		if got[name] {
			t.Fatalf("Filters tree still has %q", name)
		}
	}
	if !important {
		t.Fatal("Filters tree missing Important")
	}
}

func separateFilterRow(root widget.Component) bool {
	var col *widgets.FlexBox
	widget.Walk(root, func(c widget.Component) {
		f, ok := c.(*widgets.FlexBox)
		if !ok || col != nil {
			return
		}
		var hasMenu, hasSplit bool
		for _, ch := range f.Children() {
			if _, ok := ch.(*widgets.MenuBar); ok {
				hasMenu = true
			}
			if _, ok := ch.(*widgets.Splitter); ok {
				hasSplit = true
			}
		}
		if hasMenu && hasSplit {
			col = f
		}
	})
	if col == nil {
		return true
	}
	chs := col.Children()
	if len(chs) < 3 {
		return true
	}
	if _, ok := chs[0].(*widgets.MenuBar); !ok {
		return true
	}
	if _, ok := chs[2].(*widgets.Splitter); !ok {
		return true
	}
	// A leftover filter strip would be a sibling between the main row and the split.
	return len(chs) > 3
}

func TestMailChromeHasNoSidebarAccountPicker(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()

	var split *widgets.Splitter
	var tree *widgets.TreeView
	var prefs bool
	widget.Walk(w.Content(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.Splitter:
			if split == nil {
				split = v
			}
		case *widgets.TreeView:
			if tree == nil {
				tree = v
			}
		case *widgets.Label:
			if v.Title && (v.Text == "Folders" || v.Text == "Account" || v.Text == "Tags") {
				t.Fatalf("sidebar section header still present: %q", v.Text)
			}
		case *widgets.MenuBar:
			for _, m := range v.Menus() {
				for _, it := range m.Items {
					if it == nil {
						continue
					}
					label, _, _ := widgets.ParseMnemonic(it.Text)
					if label == "Preferences" && it.OnClick != nil {
						prefs = true
					}
				}
			}
		}
	})
	if split == nil {
		t.Fatal("splitter")
	}
	widget.Walk(split, func(c widget.Component) {
		if _, ok := c.(*widgets.ComboBox); ok {
			t.Fatal("account ComboBox still in the sidebar splitter")
		}
	})
	if tree == nil {
		t.Fatal("folder tree")
	}
	var acctNode, tagsNode bool
	for _, n := range tree.Roots {
		if n == nil {
			continue
		}
		if id, ok := n.Data.(string); ok && id != "" && id != AccountTags {
			acctNode = true
		}
		if n.Label == "Tags" {
			t.Fatal("sidebar Tags node should be renamed Filters")
		}
		if n.Label == "Filters" || n.Data == AccountTags {
			tagsNode = true
		}
	}
	if !acctNode {
		t.Fatal("account folders missing from the tree")
	}
	if !tagsNode {
		t.Fatal("Filters missing from the tree")
	}
	if !prefs {
		t.Fatal("M → Preferences missing")
	}
	w.Close()
}

func TestMailChromeHasNoActiveFilterBanner(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()

	qf := findQuickFilter(w.Content())
	var tree *widgets.TreeView
	var table *widgets.TableView
	widget.Walk(w.Content(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.TableView:
			if table == nil {
				table = v
			}
		case *widgets.TreeView:
			if tree == nil {
				tree = v
			}
		}
	})
	if qf == nil {
		t.Fatal("quick filter field missing")
	}
	if table == nil {
		t.Fatal("thread table")
	}
	if tree == nil || tree.OnSelect == nil {
		t.Fatal("Filters tree")
	}
	assertNoFilterBanner(t, w.Content())

	before := table.RowCount
	qf.SetText("lunch")
	a.PumpOnce()
	assertNoFilterBanner(t, w.Content())
	if table.RowCount <= 0 || table.RowCount > before {
		t.Fatalf("quick filter should narrow the list, rows=%d before=%d", table.RowCount, before)
	}

	qf.SetText("")
	a.PumpOnce()
	assertNoFilterBanner(t, w.Content())
	if table.RowCount != before {
		t.Fatalf("emptying quick filter should restore the list, rows=%d want %d", table.RowCount, before)
	}

	unread := findFilterPin(tree, "Unread")
	if unread == nil {
		t.Fatal("Unread pin missing from Filters tree")
	}
	tree.OnSelect(unread)
	a.PumpOnce()
	assertNoFilterBanner(t, w.Content())
	if table.RowCount <= 0 || table.RowCount > before {
		t.Fatalf("Unread pin should narrow the list, rows=%d before=%d", table.RowCount, before)
	}
	unread = findFilterPin(tree, "Unread")
	if unread == nil {
		t.Fatal("Unread pin missing after toggle")
	}
	tree.OnSelect(unread)
	a.PumpOnce()
	assertNoFilterBanner(t, w.Content())
	w.Close()
}

func findFilterPin(tree *widgets.TreeView, name string) *widgets.TreeNode {
	if tree == nil {
		return nil
	}
	var found *widgets.TreeNode
	var walk func([]*widgets.TreeNode)
	walk = func(nodes []*widgets.TreeNode) {
		for _, n := range nodes {
			if n == nil || found != nil {
				continue
			}
			if _, ok := n.Data.(filterPin); ok && strings.TrimPrefix(n.Label, "✓ ") == name {
				found = n
				return
			}
			walk(n.Children)
		}
	}
	walk(tree.Roots)
	return found
}

func assertNoFilterBanner(t *testing.T, root widget.Component) {
	t.Helper()
	widget.Walk(root, func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.Button:
			if v.Text == "Clear filter" {
				t.Fatal("Clear filter button still above the message list")
			}
		case *widgets.Label:
			if strings.Contains(v.Text, "Filter on") ||
				strings.Contains(v.Text, "No messages match this filter") ||
				v.Text == "This folder is empty." {
				t.Fatalf("active-filter banner still present: %q", v.Text)
			}
		}
	})
}

func TestMailColumnsTopicWhoWhen(t *testing.T) {
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
	if table == nil {
		t.Fatal("thread table")
	}
	want := []string{"★", "📎", "Topic", "Who", "When"}
	if len(table.Columns) != len(want) {
		t.Fatalf("columns %d %v", len(table.Columns), titlesOf(table))
	}
	for i, title := range want {
		if table.Columns[i].Title != title {
			t.Fatalf("col %d %q want %q", i, table.Columns[i].Title, title)
		}
	}
	w.Close()
}

func titlesOf(tv *widgets.TableView) []string {
	out := make([]string, len(tv.Columns))
	for i, c := range tv.Columns {
		out[i] = c.Title
	}
	return out
}

func TestMailStarRendersAfterToggle(t *testing.T) {
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
	if table == nil || table.CellText == nil || table.RowCount < 1 {
		t.Fatal("thread table")
	}
	row := -1
	for i := 0; i < table.RowCount; i++ {
		if table.CellText(i, 0) == "" {
			row = i
			break
		}
	}
	if row < 0 {
		t.Fatal("no unstarred row")
	}
	table.Selected = row
	if table.OnSelect != nil {
		table.OnSelect(row)
	}
	if table.OnContext == nil {
		t.Fatal("thread context menu")
	}
	table.OnContext(row, paintengine2d.Pt(10, 10))
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil {
		t.Fatal("context menu Star")
	}
	var star func()
	for _, it := range pop.Items {
		if it != nil && it.Text == "Star" && it.OnClick != nil {
			star = it.OnClick
		}
	}
	if star == nil {
		t.Fatal("context menu Star")
	}
	star()
	a.PumpOnce()
	if got := table.CellText(row, 0); got != "★" {
		t.Fatalf("after star CellText=%q", got)
	}
	img := paintengine2d.NewImage(32, 28)
	ctx := paintengine2d.NewContext(img)
	table.Look().DrawTableCell(ctx, paintengine2d.XYWH(0, 0, 28, 28), true, false, "★", style.AlignStart, table.Look().Font())
	if ink := cellInk(img, 0, 26); ink < 8 {
		t.Fatalf("star glyph missing in 28px cell, ink=%d", ink)
	}
	star()
	a.PumpOnce()
	if got := table.CellText(row, 0); got != "" {
		t.Fatalf("after unstar CellText=%q", got)
	}
	w.Close()
}

func TestMailAttachmentSelectOpenSaveAs(t *testing.T) {
	t.Setenv("UITK_MAIL_NO_OPEN", "1")
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()

	hits, opens, saves, saveAll := findAttachChrome(w.Content())
	if len(hits) < 2 || len(opens) < 2 || len(saves) < 2 {
		t.Fatalf("per-row chrome hits=%d open=%d saveAs=%d", len(hits), len(opens), len(saves))
	}
	if saveAll == nil || !saveAll.Enabled() || saveAll.OnClick == nil {
		t.Fatal("Save All should sit in the attachment toolbar and be enabled")
	}
	for i, b := range opens {
		if !b.Enabled() {
			t.Fatalf("row Open %d should be enabled without a shared selection", i)
		}
	}
	for i, b := range saves {
		if !b.Enabled() {
			t.Fatalf("row Save As %d should be enabled without a shared selection", i)
		}
	}

	before := globMailOpenDirs()
	if hits[0].OnPress == nil {
		t.Fatal("attachment row press")
	}
	hits[0].OnPress()
	a.PumpOnce()
	if !hits[0].Selected || hits[1].Selected {
		t.Fatal("single click should select that row only")
	}
	if n := globMailOpenDirs(); len(n) != len(before) {
		t.Fatalf("single click opened a part: before=%d after=%d", len(before), len(n))
	}

	hits[0].OnPress()
	a.PumpOnce()
	afterOpen := globMailOpenDirs()
	if len(afterOpen) <= len(before) {
		t.Fatal("double click should Open (messages.openPart cache)")
	}

	beforeBtn := globMailOpenDirs()
	if opens[1].OnClick == nil {
		t.Fatal("row Open")
	}
	opens[1].OnClick()
	a.PumpOnce()
	if n := globMailOpenDirs(); len(n) <= len(beforeBtn) {
		t.Fatal("row Open should call messages.openPart")
	}

	dest := filepath.Join(t.TempDir(), "mail-shortcuts.txt")
	if saves[0].OnClick == nil {
		t.Fatal("Save As")
	}
	saves[0].OnClick()
	a.PumpOnce()
	if err := confirmFileDialog(t, w, dest); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("saved file: %v", err)
	}
	if !strings.Contains(string(raw), "mail-shortcuts.txt") {
		t.Fatalf("saved bytes %q", raw)
	}

	dir := t.TempDir()
	saveAll.OnClick()
	a.PumpOnce()
	if err := confirmFileDialog(t, w, dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"mail-shortcuts.txt", "uitoolkit-v080.png"} {
		p := filepath.Join(dir, name)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("Save All %s: %v", name, err)
		}
		if !strings.Contains(string(b), name) {
			t.Fatalf("Save All bytes %s %q", name, b)
		}
	}
	w.Close()
}

func TestAttachFileNameHelpers(t *testing.T) {
	if got := attachFileName("../x/mail-shortcuts.txt"); got != "mail-shortcuts.txt" {
		t.Fatalf("base %q", got)
	}
	if got := attachFileName(".."); got != "attachment" {
		t.Fatalf("dotdot %q", got)
	}
	used := map[string]int{}
	if got := uniqueFileName("a.txt", used); got != "a.txt" {
		t.Fatalf("first %q", got)
	}
	if got := uniqueFileName("a.txt", used); got != "a-2.txt" {
		t.Fatalf("second %q", got)
	}
	dir := t.TempDir()
	got, err := saveAllDir(filepath.Join(dir, "nested-folder"))
	if err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(got); err != nil || !st.IsDir() {
		t.Fatalf("saveAllDir create: %s %v", got, err)
	}
	file := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	parent, err := saveAllDir(file)
	if err != nil || parent != dir {
		t.Fatalf("saveAllDir file -> parent: %q %v", parent, err)
	}
}

func findAttachChrome(root widget.Component) (hits []*attachHit, opens, saves []*widgets.Button, saveAll *widgets.Button) {
	widget.Walk(root, func(c widget.Component) {
		switch v := c.(type) {
		case *attachHit:
			hits = append(hits, v)
		case *widgets.Button:
			switch v.Text {
			case "Open":
				opens = append(opens, v)
			case "Save As":
				saves = append(saves, v)
			case "Save All":
				saveAll = v
			}
		}
	})
	return
}

func confirmFileDialog(t *testing.T, w *app.Window, path string) error {
	t.Helper()
	if w.Overlay() == nil {
		return fmt.Errorf("file dialog overlay")
	}
	var pathField *widgets.TextField
	var confirm *widgets.Button
	widget.Walk(w.Overlay(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.TextField:
			if pathField == nil {
				pathField = v
			}
		case *widgets.Button:
			if v.Text == "Save" {
				confirm = v
			}
		}
	})
	if pathField == nil || confirm == nil || confirm.OnClick == nil {
		return fmt.Errorf("dialog path/Save")
	}
	pathField.SetText(path)
	confirm.OnClick()
	return nil
}

func globMailOpenDirs() []string {
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "uitk-mail-*"))
	return matches
}

func TestOrderFolderChildrenOutboxLast(t *testing.T) {
	got := orderFolderChildren([]Folder{
		{Name: "Outbox", ID: FolderOutbox, Kind: FolderCustom},
		{Name: "Projects", Kind: FolderCustom},
		{Name: "Inbox", Kind: FolderInbox},
		{Name: "Trash", Kind: FolderTrash},
		{Name: "Junk", Kind: FolderJunk},
		{Name: "Archive", Kind: FolderArchive},
		{Name: "Drafts", Kind: FolderDrafts},
		{Name: "Sent", Kind: FolderSent},
	})
	want := []string{"Inbox", "Drafts", "Sent", "Archive", "Junk", "Trash", "Projects", "Outbox"}
	if len(got) != len(want) {
		t.Fatalf("len %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Fatalf("pos %d: got %q want %q", i, got[i].Name, want[i])
		}
	}
	named := orderFolderChildren([]Folder{
		{Name: "Outbox", Kind: FolderCustom},
		{Name: "Inbox", Kind: FolderInbox},
		{Name: "Notes", Kind: FolderCustom},
	})
	if named[len(named)-1].Name != "Outbox" {
		t.Fatalf("named Outbox not last: %#v", named)
	}
}

func TestMailMessageSourceShowsRFC822(t *testing.T) {
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

	var tabs *widgets.TabView
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TabView); ok && tabs == nil {
			tabs = tv
		}
	})
	if tabs == nil {
		t.Fatal("preview tabs")
	}
	tabs.Select(1)
	a.PumpOnce()
	var sourceTab *widgets.TextArea
	widget.Walk(w.Content(), func(c widget.Component) {
		if ta, ok := c.(*widgets.TextArea); ok && ta.Mono && ta.ReadOnly {
			sourceTab = ta
		}
	})
	if sourceTab == nil || !strings.Contains(sourceTab.Text, "MIME-Version:") {
		t.Fatalf("Source tab missing RFC822: %q", textPreview(sourceTab))
	}
	if strings.Contains(sourceTab.Text, "X-Flag:") {
		t.Fatal("Source tab used reconstructed headers")
	}

	var mb *widgets.MenuBar
	widget.Walk(w.Content(), func(c widget.Component) {
		if m, ok := c.(*widgets.MenuBar); ok && mb == nil {
			mb = m
		}
	})
	if mb == nil {
		t.Fatal("menu bar")
	}
	mb.RequestFocus()
	w.Inject(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyU, Mods: platform.ModCtrl})
	a.PumpOnce()
	var srcWin *app.Window
	for _, win := range a.Windows() {
		if win == w {
			continue
		}
		var body *widgets.TextArea
		widget.Walk(win.Content(), func(c widget.Component) {
			if ta, ok := c.(*widgets.TextArea); ok && ta.Mono && strings.Contains(ta.Text, "MIME-Version:") {
				body = ta
			}
		})
		if body != nil {
			srcWin = win
			break
		}
	}
	if srcWin == nil {
		t.Fatal("source window not opened")
	}
	var body *widgets.TextArea
	widget.Walk(srcWin.Content(), func(c widget.Component) {
		if ta, ok := c.(*widgets.TextArea); ok && ta.Mono {
			body = ta
		}
	})
	if body == nil || !body.ReadOnly || !strings.Contains(body.Text, "MIME-Version:") {
		t.Fatalf("source window body %q", textPreview(body))
	}
	srcWin.Close()
	w.Close()
}

func textPreview(ta *widgets.TextArea) string {
	if ta == nil {
		return "<nil>"
	}
	if len(ta.Text) > 200 {
		return ta.Text[:200]
	}
	return ta.Text
}

func TestMailFolderTreePutsOutboxLast(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	assertOutboxPinnedBottom(t, w.Content())
	w.Close()
}

func TestMailMenuHoverDoesNotRebuildTree(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	var mb *widgets.MenuBar
	var folder *widgets.TreeView
	widget.Walk(w.Content(), func(c widget.Component) {
		if m, ok := c.(*widgets.MenuBar); ok && mb == nil {
			mb = m
		}
	})
	folder, _ = mailSidebarTrees(w.Content())
	if mb == nil || folder == nil {
		t.Fatal("mail chrome")
	}
	roots := folder.Roots
	mb.Open(0)
	a.PumpOnce()
	pop, ok := w.Popup().(*widgets.PopupMenu)
	if !ok || pop == nil || len(pop.Items) < 3 {
		t.Fatal("M menu")
	}
	for i := 0; i < 3; i++ {
		r := pop.ItemBounds(i)
		o := widget.DeviceOrigin(pop)
		w.Inject(platform.Event{
			Kind: platform.EventMouseMove,
			Pos:  paintengine2d.Pt(o.X+(r.Min.X+r.Max.X)*0.5, o.Y+(r.Min.Y+r.Max.Y)*0.5),
		})
		a.PumpOnce()
	}
	if folder.Roots[0] != roots[0] {
		t.Fatal("menu hover rebuilt the folder tree")
	}
	w.Close()
}

func mailSidebarTrees(root widget.Component) (folder, outbox *widgets.TreeView) {
	widget.Walk(root, func(c widget.Component) {
		tv, ok := c.(*widgets.TreeView)
		if !ok {
			return
		}
		if treeHasOutboxRoot(tv) {
			if outbox == nil {
				outbox = tv
			}
			return
		}
		if folder == nil {
			folder = tv
		}
	})
	return folder, outbox
}

func treeHasOutboxRoot(tv *widgets.TreeView) bool {
	if tv == nil {
		return false
	}
	for _, n := range tv.Roots {
		if n == nil {
			continue
		}
		if id, ok := n.Data.(FolderID); ok && id == FolderOutbox {
			return true
		}
		if strings.HasPrefix(n.Label, "Outbox") {
			return true
		}
	}
	return false
}

func assertOutboxPinnedBottom(t *testing.T, root widget.Component) {
	t.Helper()
	folder, pin := mailSidebarTrees(root)
	if folder == nil || len(folder.Roots) == 0 {
		t.Fatal("folder tree")
	}
	if treeHasOutboxRoot(folder) {
		t.Fatal("Outbox should not sit in the folder tree next to Filters")
	}
	last := folder.Roots[len(folder.Roots)-1]
	if last == nil || last.Label != "Filters" {
		label := ""
		if last != nil {
			label = last.Label
		}
		t.Fatalf("folder tree last root %q, want Filters", label)
	}
	if pin == nil || !treeHasOutboxRoot(pin) {
		t.Fatal("Outbox pin missing")
	}
	if pin.Bounds().Dy() > 48 {
		t.Fatalf("Outbox pin too tall %v (should be one row at the bottom)", pin.Bounds().Dy())
	}
	fo := widget.DeviceOrigin(folder)
	po := widget.DeviceOrigin(pin)
	if po.Y+0.5 < fo.Y+folder.Bounds().Dy()-pin.Bounds().Dy()-8 {
		t.Fatalf("Outbox not at the bottom of the folder pane: folder=%v..%v pin=%v",
			fo.Y, fo.Y+folder.Bounds().Dy(), po.Y)
	}
}

func assertNoUnreadFolderFooter(t *testing.T, root widget.Component) {
	t.Helper()
	walkAll(root, func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.StatusBar:
			t.Fatal("status bar should stay gone")
		case *widgets.Label:
			if strings.Contains(strings.ToLower(v.Text), "unread in all folders") ||
				strings.Contains(strings.ToLower(v.Text), "unread folders") {
				t.Fatalf("unread-count footer still present: %q", v.Text)
			}
		}
	})
}

func TestMailMenuBarIsOnlyM(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	var mb *widgets.MenuBar
	widget.Walk(w.Content(), func(c widget.Component) {
		if m, ok := c.(*widgets.MenuBar); ok && mb == nil {
			mb = m
		}
	})
	if mb == nil {
		t.Fatal("menu bar")
	}
	menus := mb.Menus()
	if len(menus) != 1 {
		t.Fatalf("menus %d want 1", len(menus))
	}
	title, _, _ := widgets.ParseMnemonic(menus[0].Title)
	if title != "M" {
		t.Fatalf("menu title %q want M", menus[0].Title)
	}
	bannedTitles := []string{"File", "Edit", "View", "Go", "Message", "Tools", "Help"}
	for _, name := range bannedTitles {
		if title == name {
			t.Fatalf("old top-level menu %q", name)
		}
	}
	var labels []string
	var quit, prefs, view, threaded, muted bool
	gone := []string{
		"Mail Toolbar", "Quick Filter Bar",
		"Sort by When", "Sort by Topic", "Sort by Who",
		"Dark", "Light", "Message Source",
	}
	for _, it := range menus[0].Items {
		if it == nil {
			continue
		}
		label, _, _ := widgets.ParseMnemonic(it.Text)
		labels = append(labels, label)
		for _, g := range gone {
			if label == g {
				t.Fatalf("removed item still on M: %q %v", label, labels)
			}
		}
		switch label {
		case "Quit":
			quit = it.Shortcut == "Ctrl+Q" && it.OnClick != nil
		case "Preferences":
			prefs = it.OnClick != nil
		case "View":
			view = it.HasSubmenu()
			want := []string{
				"Vertical (3-pane)", "Classic (preview below)",
				"Table view", "Card view",
				"Compact", "Default density", "Relaxed",
			}
			got := menuItemLabels(it.Submenu)
			for _, name := range want {
				if !containsLabel(got, name) {
					t.Fatalf("View submenu missing %q: %v", name, got)
				}
			}
			for _, name := range gone {
				if containsLabel(got, name) {
					t.Fatalf("removed item under View: %q %v", name, got)
				}
			}
		case "Threaded":
			threaded = it.Checkable
		case "Hide muted threads":
			muted = it.Checkable
		}
	}
	if !view || !threaded || !muted || !prefs || !quit {
		t.Fatalf("M items view=%v threaded=%v muted=%v prefs=%v quit=%v %v", view, threaded, muted, prefs, quit, labels)
	}
	pos := map[string]int{}
	for i, l := range labels {
		if _, ok := pos[l]; !ok {
			pos[l] = i
		}
	}
	if pos["View"] > pos["Threaded"] || pos["Threaded"] > pos["Preferences"] || pos["Preferences"] > pos["Quit"] {
		t.Fatalf("order %v want View, Threaded…, Preferences, Quit", labels)
	}
	w.Close()
}

func menuItemLabels(items []*widgets.MenuItem) []string {
	var out []string
	for _, it := range items {
		if it == nil || it.Separator {
			continue
		}
		label, _, _ := widgets.ParseMnemonic(it.Text)
		out = append(out, label)
	}
	return out
}

func containsLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}

func cellInk(img *paintengine2d.Image, x0, x1 int) int {
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := x0; x < x1 && x < img.Width; x++ {
			_, _, _, a := img.PremulAt(x, y)
			if a > 20 {
				n++
			}
		}
	}
	return n
}
