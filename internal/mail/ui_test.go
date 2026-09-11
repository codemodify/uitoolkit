package mail

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	var qf, qfInSplit widget.Component
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
		case *widgets.TextField:
			if strings.Contains(v.Placeholder, "Quick Filter") && qf == nil {
				qf = v
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
	if qf != nil && split != nil && widget.Contains(split, qf) {
		qfInSplit = qf
	}
	if bars != 0 {
		t.Fatalf("status bar widgets=%d", bars)
	}
	if titles != 0 {
		t.Fatalf("path/subtitle TitleBar should be gone, got %d", titles)
	}
	if vipNode {
		t.Fatal("VIP folder still in the sidebar tree")
	}
	if qf == nil {
		t.Fatal("quick filter field missing")
	}
	if qfInSplit != nil {
		t.Fatal("quick filter still lives above the message list inside the splitter")
	}
	w.Close()
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
	var addAcct, removeAcct, acctCentral bool
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
					switch it.Text {
					case "Add Account…", "Add &Account…":
						addAcct = true
					case "Remove Account…", "Remove &Account…":
						removeAcct = true
					case "Account Central":
						acctCentral = true
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
		if n.Label == "Tags" || n.Data == AccountTags {
			tagsNode = true
		}
	}
	if !acctNode {
		t.Fatal("account folders missing from the tree")
	}
	if !tagsNode {
		t.Fatal("Tags missing from the tree")
	}
	if !addAcct || !removeAcct || !acctCentral {
		t.Fatalf("account menus add=%v remove=%v central=%v", addAcct, removeAcct, acctCentral)
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

	var qf *widgets.TextField
	var unread *widgets.ToolItem
	var table *widgets.TableView
	widget.Walk(w.Content(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.TextField:
			if strings.Contains(v.Placeholder, "Quick Filter") && qf == nil {
				qf = v
			}
		case *widgets.TableView:
			if table == nil {
				table = v
			}
		case *widgets.ToolBar:
			for _, it := range v.Items() {
				if it != nil && it.Text == "Unread" && unread == nil {
					unread = it
				}
			}
		}
	})
	if qf == nil {
		t.Fatal("quick filter field missing")
	}
	if table == nil {
		t.Fatal("thread table")
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

	if unread == nil || unread.OnClick == nil {
		t.Fatal("Unread pin missing")
	}
	unread.OnClick()
	a.PumpOnce()
	assertNoFilterBanner(t, w.Content())
	if table.RowCount <= 0 || table.RowCount > before {
		t.Fatalf("Unread pin should narrow the list, rows=%d before=%d", table.RowCount, before)
	}
	unread.OnClick()
	a.PumpOnce()
	assertNoFilterBanner(t, w.Content())
	w.Close()
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
	var sortWhen, sortTopic, sortWho, sortSize bool
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && table == nil {
			table = tv
		}
		if mb, ok := c.(*widgets.MenuBar); ok {
			for _, m := range mb.Menus() {
				for _, it := range m.Items {
					if it == nil {
						continue
					}
					switch it.Text {
					case "Sort by When":
						sortWhen = true
					case "Sort by Topic":
						sortTopic = true
					case "Sort by Who":
						sortWho = true
					case "Sort by Size", "Sort by Date", "Sort by Subject", "Sort by Correspondent":
						sortSize = true
					}
				}
			}
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
	if !sortWhen || !sortTopic || !sortWho {
		t.Fatalf("view sort labels when=%v topic=%v who=%v", sortWhen, sortTopic, sortWho)
	}
	if sortSize {
		t.Fatal("old Size/Date/Subject/Correspondent sort labels still present")
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
	var star func()
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && table == nil {
			table = tv
		}
		if mb, ok := c.(*widgets.MenuBar); ok {
			for _, m := range mb.Menus() {
				for _, it := range m.Items {
					if it != nil && it.Text == "Star" && it.OnClick != nil {
						star = it.OnClick
					}
				}
			}
		}
	})
	if table == nil || table.CellText == nil || table.RowCount < 1 {
		t.Fatal("thread table")
	}
	if star == nil {
		t.Fatal("Message → Star")
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
