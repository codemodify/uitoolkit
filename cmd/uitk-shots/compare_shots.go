package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// compareThumbNames are isolated widget frames for docs/widgets.md.
var compareThumbNames = []string{
	"compare/label.png",
	"compare/button.png",
	"compare/checkbox.png",
	"compare/switch.png",
	"compare/radio.png",
	"compare/slider.png",
	"compare/textfield.png",
	"compare/textarea.png",
	"compare/numberfield.png",
	"compare/combobox.png",
	"compare/progress.png",
	"compare/listview.png",
	"compare/cardlist.png",
	"compare/tableview.png",
	"compare/treeview.png",
	"compare/tabview.png",
	"compare/menubar.png",
	"compare/popupmenu.png",
	"compare/toolbar.png",
	"compare/statusbar.png",
	"compare/titlebar.png",
	"compare/scrollview.png",
	"compare/splitter.png",
	"compare/panel.png",
	"compare/accordion.png",
	"compare/messagebox.png",
	"compare/filedialog.png",
	"compare/tooltip.png",
	"compare/layout.png",
}

func writeCompareThumbs(dir string) error {
	out := filepath.Join(dir, "compare")
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	type thumb struct {
		name   string
		width  int
		height int
		build  func() widget.Component
		after  func(a *app.Application, w *app.Window)
	}
	thumbs := []thumb{
		{
			name: "label.png", width: 360, height: 120,
			build: func() widget.Component {
				return frame("Label / Title",
					widgets.NewTitle("Widget gallery"),
					widgets.NewLabel("Static text  ·  Titillium Web"),
				)
			},
		},
		{
			name: "button.png", width: 400, height: 120,
			build: func() widget.Component {
				primary := widgets.NewButton("Primary action", nil)
				primary.Primary = true
				disabled := widgets.NewButton("Disabled", nil)
				disabled.SetEnabled(false)
				return frame("Button",
					widgets.NewRow(primary, widgets.NewButton("Secondary", nil), disabled).WithGap(10),
				)
			},
			after: func(a *app.Application, w *app.Window) {
				widget.Walk(w.Content(), func(c widget.Component) {
					if b, ok := c.(*widgets.Button); ok && b.Primary {
						b.MouseEnter()
					}
				})
			},
		},
		{
			name: "checkbox.png", width: 360, height: 140,
			build: func() widget.Component {
				return frame("Checkbox",
					widgets.NewCheckbox("Enable notifications", true, nil),
					widgets.NewCheckbox("Launch at login", false, nil),
				)
			},
		},
		{
			name: "switch.png", width: 360, height: 140,
			build: func() widget.Component {
				return frame("Switch",
					widgets.NewSwitch("Dark chrome", true, nil),
					widgets.NewSwitch("Compact layout", false, nil),
				)
			},
		},
		{
			name: "radio.png", width: 380, height: 160,
			build: func() widget.Component {
				return frame("RadioGroup",
					widgets.NewRadioGroup([]string{"Dark graphite", "Light paper", "System follow"}, 0, nil),
				)
			},
		},
		{
			name: "slider.png", width: 400, height: 120,
			build: func() widget.Component {
				return frame("Slider",
					widgets.NewLabel("Volume  72%"),
					widgets.NewSlider(0, 100, 72, nil),
				)
			},
		},
		{
			name: "textfield.png", width: 400, height: 140,
			build: func() widget.Component {
				return frame("TextField",
					widgets.NewTextField("Ada Lovelace", "Display name", nil),
					widgets.NewMonoTextField("/var/log/uitoolkit.log", "log path", nil),
				)
			},
			after: func(a *app.Application, w *app.Window) {
				widget.Walk(w.Content(), func(c widget.Component) {
					if tf, ok := c.(*widgets.TextField); ok && !tf.Mono {
						w.RequestFocus(tf)
						tf.SetSelection(0, 3)
					}
				})
			},
		},
		{
			name: "textarea.png", width: 420, height: 200,
			build: func() widget.Component {
				area := widgets.NewTextArea(
					"Multi-line TextArea.\nWrap or scroll; Return inserts a line.",
					"Notes",
					nil,
				)
				area.MinRows = 4
				return frame("TextArea", area)
			},
		},
		{
			name: "numberfield.png", width: 360, height: 120,
			build: func() widget.Component {
				return frame("NumberField / Spinner",
					widgets.NewNumberField(1, 12, 3, 1, nil),
				)
			},
		},
		{
			name: "combobox.png", width: 400, height: 220,
			build: func() widget.Component {
				return frame("ComboBox",
					widgets.NewComboBox([]string{"paintengine2d", "Software raster", "Offscreen pixmap"}, 0, nil),
					widgets.NewLabel("Drop-down lists named engines."),
				)
			},
			after: func(a *app.Application, w *app.Window) {
				widget.Walk(w.Content(), func(c widget.Component) {
					if cb, ok := c.(*widgets.ComboBox); ok {
						cb.Open()
					}
				})
			},
		},
		{
			name: "progress.png", width: 400, height: 140,
			build: func() widget.Component {
				return frame("ProgressBar / BusyBar",
					widgets.NewLabel("Build  68%"),
					widgets.NewProgressBar(0.68),
					widgets.NewLabel("Busy"),
					widgets.NewBusyBar(0.35),
				)
			},
		},
		{
			name: "listview.png", width: 360, height: 240,
			build: func() widget.Component {
				files := []string{"README.md", "export.go", "version.go", "go.mod", "LICENSE"}
				list := widgets.NewListView(len(files), func(i int) string { return files[i] }, nil)
				list.Selected = 0
				col := widgets.NewColumn(widgets.NewTitleBar("ListView", "virtual rows"), list).WithGap(0)
				col.AddFlex(list, 1)
				return col
			},
		},
		{
			name: "cardlist.png", width: 420, height: 260,
			build: func() widget.Component {
				cards := widgets.NewCardList(3, func(i int) widgets.CardContent {
					rows := []widgets.CardContent{
						{Title: "Ada Lovelace", Subtitle: "Notes on the Analytical Engine", Meta: "now", Snippet: "Thunderbird-style multi-line card", Bold: true},
						{Title: "Grace Hopper", Subtitle: "Compiler notes", Meta: "Tue", Snippet: "Second card in the virtual list"},
						{Title: "Katherine Johnson", Subtitle: "Trajectory tables", Meta: "Mon", Snippet: "Third row, unread style off"},
					}
					return rows[i]
				}, nil)
				cards.Selected = 0
				cards.CardHeight = 64
				col := widgets.NewColumn(widgets.NewTitleBar("CardList", "mail-style rows"), cards).WithGap(0)
				col.AddFlex(cards, 1)
				return col
			},
		},
		{
			name: "tableview.png", width: 440, height: 260,
			build: func() widget.Component {
				rows := [][3]string{
					{"widget", "412", "stable"},
					{"widgets", "2140", "wip"},
					{"app", "288", "stable"},
				}
				table := widgets.NewTableView([]widgets.TableColumn{
					{Title: "Package", Sortable: true},
					{Title: "Lines", Width: 72, Sortable: true, Align: style.AlignEnd},
					{Title: "Status", Width: 80, Sortable: true},
				}, len(rows), func(row, col int) string {
					return rows[row][col]
				}, nil)
				table.Selected = 0
				col := widgets.NewColumn(widgets.NewTitleBar("TableView", "sortable columns"), table).WithGap(0)
				col.AddFlex(table, 1)
				return col
			},
		},
		{
			name: "treeview.png", width: 360, height: 240,
			build: func() widget.Component {
				src := widgets.NewTreeNode("src",
					widgets.NewTreeNode("app", widgets.NewTreeNode("window.go")),
					widgets.NewTreeNode("widgets", widgets.NewTreeNode("tree.go"), widgets.NewTreeNode("menu.go")),
				)
				root := widgets.NewTreeNode("uitoolkit", src, widgets.NewTreeNode("go.mod"))
				tree := widgets.NewTreeView(root)
				tree.Selected = src
				col := widgets.NewColumn(widgets.NewTitleBar("TreeView", "expand / select"), tree).WithGap(0)
				col.AddFlex(tree, 1)
				return col
			},
		},
		{
			name: "tabview.png", width: 400, height: 200,
			build: func() widget.Component {
				tabs := widgets.NewTabView(
					widgets.Tab{Title: "Scroll", Content: widgets.NewPad(10, widgets.NewLabel("ScrollView page"))},
					widgets.Tab{Title: "List", Content: widgets.NewPad(10, widgets.NewLabel("ListView page"))},
					widgets.Tab{Title: "Tree", Content: widgets.NewPad(10, widgets.NewLabel("TreeView page"))},
				)
				col := widgets.NewColumn(widgets.NewTitleBar("TabView", "TabBar + pages"), tabs).WithGap(0)
				col.AddFlex(tabs, 1)
				return col
			},
		},
		{
			name: "menubar.png", width: 420, height: 240,
			build: func() widget.Component {
				bar := widgets.NewMenuBar(
					widgets.NewMenu("&File",
						widgets.ItemAccel("&New window", "Ctrl+N", nil),
						widgets.ItemAccel("&Open…", "Ctrl+O", nil),
						widgets.Sep(),
						widgets.ItemAccel("&Quit", "Ctrl+Q", nil),
					),
					widgets.NewMenu("&Edit",
						widgets.ItemAccel("&Copy", "Ctrl+C", nil),
					),
				)
				return widgets.NewColumn(
					bar,
					widgets.NewPad(16, widgets.NewLabel("MenuBar drop-down")),
				).WithGap(0)
			},
			after: func(a *app.Application, w *app.Window) {
				widget.Walk(w.Content(), func(c widget.Component) {
					if mb, ok := c.(*widgets.MenuBar); ok {
						mb.Open(0)
					}
				})
			},
		},
		{
			name: "popupmenu.png", width: 360, height: 200,
			build: func() widget.Component {
				return widgets.NewPad(12, widgets.NewLabel("Right-click target  ·  PopupMenu"))
			},
			after: func(a *app.Application, w *app.Window) {
				widgets.ShowContextMenu(w.Content(), paintengine2d.Pt(28, 36),
					widgets.Item("Open", nil),
					widgets.Item("Copy path", nil),
					widgets.Sep(),
					widgets.Item("Reveal in list", nil),
				)
			},
		},
		{
			name: "toolbar.png", width: 480, height: 140,
			build: func() widget.Component {
				bar := widgets.NewToolBar(
					widgets.ToolIconBtn(style.IconNew, "New", nil),
					widgets.ToolIconBtn(style.IconOpen, "Open", nil),
					widgets.ToolIconBtn(style.IconSave, "Save", nil),
					widgets.ToolDivider(),
					widgets.ToolToggle("Snap", true, nil),
					widgets.ToolText("Inspect", nil),
				)
				return widgets.NewColumn(
					widgets.NewTitleBar("ToolBar", "icon · text · toggle"),
					bar,
				).WithGap(0)
			},
		},
		{
			name: "statusbar.png", width: 440, height: 100,
			build: func() widget.Component {
				return widgets.NewColumn(
					widgets.NewPad(12, widgets.NewLabel("Window body")),
					widgets.NewStatusBar("Ready.", "Ln 1, Col 1", "v"+uitoolkit.Version),
				).WithGap(0)
			},
		},
		{
			name: "titlebar.png", width: 440, height: 90,
			build: func() widget.Component {
				return widgets.NewColumn(
					widgets.NewTitleBar("Widget gallery", "v"+uitoolkit.Version),
					widgets.NewPad(12, widgets.NewLabel("Client-side caption + subtitle")),
				).WithGap(0)
			},
		},
		{
			name: "scrollview.png", width: 360, height: 220,
			build: func() widget.Component {
				long := widgets.NewColumn()
				for i := 1; i <= 16; i++ {
					long.Add(widgets.NewLabel(fmt.Sprintf("Row %02d  —  scrollable content", i)))
				}
				scroll := widgets.NewScrollView(long)
				col := widgets.NewColumn(widgets.NewTitleBar("ScrollView", "clip + bars"), scroll).WithGap(0)
				col.AddFlex(scroll, 1)
				return col
			},
			after: func(a *app.Application, w *app.Window) {
				widget.Walk(w.Content(), func(c widget.Component) {
					if sv, ok := c.(*widgets.ScrollView); ok {
						sv.ScrollTo(sv.MaxOffset() * 0.4)
					}
				})
			},
		},
		{
			name: "splitter.png", width: 420, height: 180,
			build: func() widget.Component {
				left := widgets.NewPanel("Left", widgets.NewLabel("Pane A"))
				right := widgets.NewPanel("Right", widgets.NewLabel("Pane B"))
				split := widgets.NewSplitter(widgets.SplitColumns, left, right)
				split.Ratio = 0.42
				col := widgets.NewColumn(widgets.NewTitleBar("Splitter", "drag sash"), split).WithGap(0)
				col.AddFlex(split, 1)
				return col
			},
		},
		{
			name: "panel.png", width: 380, height: 160,
			build: func() widget.Component {
				return widgets.NewPad(10, widgets.NewPanel("Panel",
					widgets.NewLabel("Grouped chrome with a title."),
					widgets.NewButton("Inside", nil),
				))
			},
		},
		{
			name: "accordion.png", width: 400, height: 240,
			build: func() widget.Component {
				session := widgets.NewExpander("Session", true, widgets.NewColumn(
					widgets.NewSwitch("Autosave", true, nil),
					widgets.NewLabel("Writes a sidecar next to the project."),
				).WithGap(6))
				more := widgets.NewExpander("Advanced", false, widgets.NewLabel("Hidden until expanded."))
				return frame("Accordion / Expander", widgets.NewAccordion(true, session, more))
			},
		},
		{
			name: "messagebox.png", width: 480, height: 280,
			build: func() widget.Component {
				return widgets.NewPad(8, widgets.NewLabel("MessageBox overlay"))
			},
			after: func(a *app.Application, w *app.Window) {
				widgets.ShowMessageBox(w.Content(), widgets.MessageBoxOptions{
					Title:   "Unsaved changes",
					Message: "Discard the draft theme and revert to Dark graphite?",
					Kind:    widgets.MessageWarning,
					Buttons: widgets.ButtonsYesNoCancel,
				})
			},
		},
		{
			name: "filedialog.png", width: 480, height: 320,
			build: func() widget.Component {
				return widgets.NewPad(8, widgets.NewLabel("FileDialog stub"))
			},
			after: func(a *app.Application, w *app.Window) {
				widgets.ShowFileDialog(w.Content(), widgets.FileDialogOptions{
					Title:  "Open project file",
					Path:   "/project/uitoolkit",
					Filter: "*.go",
					Entries: []widgets.FileInfo{
						{Name: "app", Dir: true},
						{Name: "widgets", Dir: true},
						{Name: "export.go"},
						{Name: "version.go"},
					},
				})
			},
		},
		{
			name: "tooltip.png", width: 400, height: 160,
			build: func() widget.Component {
				save := widgets.ToolIconBtn(style.IconSave, "", nil)
				save.Tip = "Save project"
				bar := widgets.NewToolBar(save, widgets.ToolText("About", nil))
				return widgets.NewColumn(
					widgets.NewTitleBar("Tooltip", "delayed hover"),
					bar,
					widgets.NewPad(12, widgets.NewLabel("Hover a tool button.")),
				).WithGap(0)
			},
			after: func(a *app.Application, w *app.Window) {
				widget.Walk(w.Content(), func(c widget.Component) {
					if tb, ok := c.(*widgets.ToolBar); ok {
						tb.Hover(0)
						o := widget.DeviceOrigin(tb)
						c := tb.ItemCenter(0)
						w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+c.X, o.Y+c.Y)})
					}
				})
				a.PumpOnce()
				w.RevealTooltip()
			},
		},
		{
			name: "layout.png", width: 420, height: 180,
			build: func() widget.Component {
				row := widgets.NewRow().WithGap(8)
				row.Add(widgets.NewButton("A", nil))
				row.Add(widgets.NewVSeparator())
				row.Add(widgets.NewButton("B", nil))
				sp := widgets.NewSpacer()
				row.AddFlex(sp, 1)
				row.Add(widgets.NewLabel("end"))
				return frame("Column / Row / Separator / Spacer",
					row,
					widgets.NewSeparator(),
					widgets.NewLabel("FlexBox row + spacer pushes the label."),
				)
			},
		},
	}
	for _, t := range thumbs {
		if err := paintCompareThumb(filepath.Join(out, t.name), t.width, t.height, t.build, t.after); err != nil {
			return err
		}
	}
	return verifyDistinctPNGs(dir, compareThumbNames)
}

func frame(title string, children ...widget.Component) widget.Component {
	return widgets.NewPad(10, widgets.NewPanel(title, children...))
}

func paintCompareThumb(path string, width, height int, build func() widget.Component, after func(*app.Application, *app.Window)) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit compare", Width: width, Height: height, Headless: true,
	})
	if err != nil {
		return err
	}
	w.SetContent(build())
	a.PumpOnce()
	if after != nil {
		after(a, w)
		a.PumpOnce()
	}
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}
