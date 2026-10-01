// Command look-like-sourcegit builds a git client's shell: repository
// tabs in the title bar, a left panel of branches and remotes, the
// commit history in the middle, and the commit's detail beneath it.
//
// It is the four-pane shape a version-control GUI settles on, and the
// one that leans hardest on the splitter: three dividers, each with a
// pane that must not be squeezed past what it holds.
//
//	go run -tags theme_engine_web ./examples/uitoolkit-sample-look-like-sourcegit
//	go run -tags theme_engine_web ./examples/uitoolkit-sample-look-like-sourcegit -headless
package main

import (
	"flag"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

type commit struct{ graph, subject, author, when, hash string }

var history = []commit{
	{"│", "release: 0.22.5", "codemodify", "14:02", "d317b7b"},
	{"├─", "feat: chrome that lines up with a pane", "codemodify", "13:41", "26e71e5"},
	{"│ ", "feat: the button that is a mark", "codemodify", "12:55", "5431914"},
	{"├─", "fix: the three secretvault said were open", "codemodify", "11:30", "649af18"},
	{"│ ", "release: 0.22.4", "codemodify", "10:18", "7ea6f81"},
	{"├─", "feat: widget.MinWidthOf", "codemodify", "09:52", "5c346a0"},
	{"│ ", "fix: the block after one whose height changed", "codemodify", "09:11", "3a0e482"},
}

// themePack is the look this sample wears. Theme engines are chosen at
// build time (docs/engines.md), so a plain build has only the default
// one and the look falls back to the default — which the toolkit
// says at start-up, and app.Application.Diagnostics collects for a test.
const themePack = "sourcegit"

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write sourcegit.png")
	flag.Parse()

	// The shape is only half of looking like sourcegit; the other half is the
	// paint. The sample states the pack it wants and nothing else, so a
	// user's icon set, corner policy and typefaces still come from their
	// own settings — the pack drawn after SourceGit.
	a := uitoolkit.New(uitoolkit.Options{
		Headless: *headless,
		// Without this the id would be the binary's name, and a sample
		// built as "sourcegit" would claim the real sourcegit's identity on the
		// desktop — its task-bar slot, its icon, its window rules.
		AppID: "uitoolkit-sample-look-like-sourcegit",
		Theme: uitoolkit.ThemeOverride{Pack: themePack},
	})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit-sample-look-like-sourcegit", Width: 1240, Height: 760,
		MinWidth: 760, MinHeight: 480, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		log.Fatal(err)
	}
	// SourceGit's repository tabs are its title bar, the way Chromium's
	// are, so the window says so rather than letting the era decide.
	win.SetTitleBar(titleBar(win))
	win.SetCaptionStyle(uitoolkit.CaptionMerged)
	win.SetBorderless(true)
	win.SetContent(body())

	if *headless {
		if err := win.WritePNG("sourcegit.png"); err != nil {
			log.Fatal(err)
		}
		return
	}
	a.Run()
}

// titleBar: the open repositories as tabs, with the actions a git client
// keeps to hand at the end.
func titleBar(win *app.Window) *widgets.HeaderBar {
	repos := widgets.NewBrowserTabs("uitoolkit", "comms-mail", "secretvault")
	repos.OnNew = func() { repos.AddTab(widgets.BrowserTab{Title: "Open…"}) }
	repos.OnClose = func(i int) { repos.RemoveTab(i) }

	head := widgets.NewHeaderBar(
		[]widget.Component{
			widgets.NewIconButton(style.IconExternalLink, "Repositories", nil),
		},
		repos,
		[]widget.Component{
			widgets.NewToolButton("Fetch", style.IconDownload, nil),
			widgets.NewToolButton("Pull", style.IconArrowDown, nil),
			widgets.NewToolButton("Push", style.IconArrowUp, nil),
			widgets.NewMenuButton(style.IconMore, "More",
				&widgets.MenuItem{Text: "Stash All", Shortcut: "Ctrl+Shift+S"},
				&widgets.MenuItem{Text: "Apply Stash"},
				&widgets.MenuItem{Separator: true},
				&widgets.MenuItem{Text: "Preferences"},
				&widgets.MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: win.Close}),
		})
	head.ShowTitle = false
	return head
}

// body is the left panel, the history and the detail beneath it.
func body() widget.Component {
	right := widgets.NewSplitter(widgets.SplitRows, historyTable(), detail())
	right.Ratio = 0.52

	split := widgets.NewSplitter(widgets.SplitColumns, widgets.NewScrollView(refs()), right)
	split.Ratio = 0.2

	col := widgets.NewColumn(split, statusBar()).WithGap(0)
	col.AddFlex(split, 1)
	return col
}

// refs is the tree of local branches, remotes and tags.
func refs() widget.Component {
	t := widgets.NewTreeView(
		&widgets.TreeNode{Label: "Local Branches", Expanded: true, Children: []*widgets.TreeNode{
			{Label: "dev", Icon: style.IconCheck, Bold: true},
			{Label: "main"},
		}},
		&widgets.TreeNode{Label: "Remotes", Expanded: true, Children: []*widgets.TreeNode{
			{Label: "origin/dev"},
			{Label: "origin/main"},
		}},
		&widgets.TreeNode{Label: "Tags", Expanded: true, Children: []*widgets.TreeNode{
			{Label: "v0.22.5", Icon: style.IconTag},
			{Label: "v0.22.4", Icon: style.IconTag},
			{Label: "v0.22.3", Icon: style.IconTag},
		}},
		&widgets.TreeNode{Label: "Submodules"},
	)
	return t
}

// historyTable is the commit graph. It scrolls sideways, because a
// history has more columns than a narrow pane holds and the hash is the
// one that falls off the end.
func historyTable() *widgets.TableView {
	t := widgets.NewTableView(
		[]widgets.TableColumn{
			{Title: "", Width: 44, MinWidth: 30},
			{Title: "Subject", Width: 380, MinWidth: 160},
			{Title: "Author", Width: 130, MinWidth: 80},
			{Title: "Time", Width: 70, MinWidth: 55},
			{Title: "Hash", Width: 90, MinWidth: 70},
		},
		len(history),
		func(row, col int) string {
			c := history[row]
			switch col {
			case 0:
				return c.graph
			case 1:
				return c.subject
			case 2:
				return c.author
			case 3:
				return c.when
			}
			return c.hash
		}, nil)
	t.Horizontal = true
	t.Mono = true
	t.Selected = 0
	return t
}

// detail is the selected commit: its message, and the files it touched.
func detail() widget.Component {
	head := widgets.NewForm()
	head.AddRow("Commit", mono("d317b7b0d314f3125288005080a7668b56fe13bf"))
	head.AddRow("Author", widgets.NewLabel("codemodify <codemodify@linux.com>"))
	head.AddRow("Date", widgets.NewLabel("Tue 30 Sep 2026 14:02:11"))

	files := widgets.NewTableView(
		[]widgets.TableColumn{
			{Title: "", Width: 30, MinWidth: 24},
			{Title: "Path", Width: 420, MinWidth: 160},
		},
		4,
		func(row, col int) string {
			paths := []string{"release-notes.md", "version.go", "docs/widgets.md", "docs/recipes.md"}
			marks := []string{"M", "M", "M", "M"}
			if col == 0 {
				return marks[row]
			}
			return paths[row]
		}, nil)

	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Changes", Content: files},
		widgets.Tab{Title: "Message", Content: widgets.NewPad(10, widgets.NewLabel("release: 0.22.5"))},
	)

	col := widgets.NewColumn(widgets.NewPad(8, head), tabs).WithGap(4)
	col.AddFlex(tabs, 1)
	return col
}

func mono(s string) *widgets.Label {
	l := widgets.NewLabel(s)
	l.Mono = true
	return l
}

func statusBar() widget.Component {
	gap := widgets.NewSpacer()
	row := widgets.NewRow(
		widgets.NewIconButton(style.IconExternalLink, "Branch", nil),
		widgets.NewLabel("dev"),
		// Ahead/behind as icons, not arrow runes: the bundled faces
		// carry no arrows, so "↑2 ↓0" draws two tofu boxes. docs/icons.md
		// is the rule — a mark is an icon, never a character.
		widgets.NewIconLabel(style.IconArrowUp, "2"),
		widgets.NewIconLabel(style.IconArrowDown, "0"),
		gap,
		widgets.NewLabel("7 commits"),
	).WithGap(8).WithPad(4)
	row.AddFlex(gap, 1)
	return row
}
