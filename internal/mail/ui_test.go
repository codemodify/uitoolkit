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
	assertOutboxLastRoot(t, tree)

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
	var qf *widgets.TextField
	widget.Walk(w.Content(), func(c widget.Component) {
		if v, ok := c.(*widgets.TextField); ok && strings.Contains(v.Placeholder, "Quick Filter") && qf == nil {
			qf = v
		}
	})
	if qf == nil {
		t.Fatal("quick filter field missing")
	}
	assertMailToolChrome(t, w.Content(), qf)
	w.Close()
}

func assertMailToolChrome(t *testing.T, root widget.Component, qf widget.Component) {
	t.Helper()
	var split *widgets.Splitter
	var table *widgets.TableView
	var mainBar, listBar *widgets.ToolBar
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
		case *widgets.ToolBar:
			texts := toolTexts(v)
			if texts["Classic"] || texts["Get Messages"] || texts["Write"] {
				mainBar = v
			}
			if texts["Tag"] && texts["Archive"] && texts["Junk"] && texts["Delete"] {
				listBar = v
			}
		}
	})
	if mainBar == nil {
		t.Fatal("main toolbar")
	}
	if listBar == nil {
		t.Fatal("list toolbar (Tag / Archive / Junk / Delete)")
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
		t.Fatal("filter field should sit in the flex slot after Classic, not inside the left tool cluster")
	}
	if !widget.Contains(split, listBar) {
		t.Fatal("list toolbar should sit in the thread pane")
	}
	if widget.Contains(split, qf) {
		t.Fatal("quick filter still in the splitter")
	}
	if !filterAfterClassic(root, qf) {
		t.Fatal("quick filter should share the main toolbar row after Classic (right-aligned)")
	}
	if separateFilterRow(root) {
		t.Fatal("separate Quick Filter row still under the main toolbar")
	}
	if main["Quick Filter"] {
		t.Fatal("left Quick Filter visibility toggle should be gone (View menu / Ctrl+F)")
	}
	barOrigin := widget.DeviceOrigin(mainBar)
	qfOrigin := widget.DeviceOrigin(qf)
	if qfOrigin.X+0.5 < barOrigin.X+mainBar.Bounds().Dx() {
		t.Fatalf("filter should sit after Classic: bar=%v..%v qf=%v",
			barOrigin.X, barOrigin.X+mainBar.Bounds().Dx(), qfOrigin.X)
	}
	if qf.Bounds().Dx() <= 100 {
		t.Fatalf("quick filter crushed: width=%v toolbar=%v", qf.Bounds().Dx(), mainBar.Bounds().Dx())
	}
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

func filterAfterClassic(root, qf widget.Component) bool {
	var row *widgets.FlexBox
	widget.Walk(root, func(c widget.Component) {
		f, ok := c.(*widgets.FlexBox)
		if !ok || row != nil {
			return
		}
		var hasClassic, hasQF bool
		widget.Walk(f, func(ch widget.Component) {
			if bar, ok := ch.(*widgets.ToolBar); ok && toolTexts(bar)["Classic"] {
				hasClassic = true
			}
			if ch == qf {
				hasQF = true
			}
		})
		if hasClassic && hasQF {
			row = f
		}
	})
	return row != nil
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

	var open func()
	widget.Walk(w.Content(), func(c widget.Component) {
		mb, ok := c.(*widgets.MenuBar)
		if !ok {
			return
		}
		for _, m := range mb.Menus() {
			for _, it := range m.Items {
				if it == nil {
					continue
				}
				label, _, _ := widgets.ParseMnemonic(it.Text)
				if label == "Message Source" && it.OnClick != nil {
					open = it.OnClick
				}
			}
		}
	})
	if open == nil {
		t.Fatal("View/Message Source menu missing")
	}
	open()
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
	var tree *widgets.TreeView
	widget.Walk(w.Content(), func(c widget.Component) {
		if tr, ok := c.(*widgets.TreeView); ok && tree == nil {
			tree = tr
		}
	})
	assertOutboxLastRoot(t, tree)
	w.Close()
}

func assertOutboxLastRoot(t *testing.T, tree *widgets.TreeView) {
	t.Helper()
	if tree == nil || len(tree.Roots) == 0 {
		t.Fatal("empty folder tree")
	}
	last := tree.Roots[len(tree.Roots)-1]
	if last == nil {
		t.Fatal("nil last root")
	}
	id, ok := last.Data.(FolderID)
	if !ok || id != FolderOutbox {
		t.Fatalf("last root data=%v label=%q, want Outbox", last.Data, last.Label)
	}
	if !strings.HasPrefix(last.Label, "Outbox") {
		t.Fatalf("last root label %q", last.Label)
	}
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
