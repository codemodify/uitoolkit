package demo

import (
	"fmt"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Settings pages.
const (
	pageThemes = iota
	pagePacks
	pageAbout
)

var settingsPages = []string{"Themes", "Packs & icons", "About"}

// Theme browser filters: every built-in pack, one decade, or user packs.
var themeFilters = []string{"All decades", "1980s", "1990s", "2000s", "2010s", "2020s", "My themes"}

const filterUser = 6

// SettingsApp is the toolkit appearance editor. The theme browser previews
// the staged appearance — theme, corners, icon set, icon size — in a live,
// interactive mini application rendered in that theme (a ThemeScope), while
// Settings itself keeps the applied look. Apply writes look.json and every
// app that watches it (and Settings) switches; Revert drops the staged
// change. Closing without Apply discards it.
func SettingsApp(a *app.Application, win *app.Window) widget.Component {
	saved := style.LoadAppearance().Normalize()
	return buildSettings(a, win, saved, saved, pageThemes)
}

// SettingsAppStaged is SettingsApp with a theme already staged (not
// applied): screenshots and docs use it to show the preview in a theme
// other than the one Settings runs in.
func SettingsAppStaged(a *app.Application, win *app.Window, theme string) widget.Component {
	saved := style.LoadAppearance().Normalize()
	staged := saved
	if pack, ok := style.LoadTheme(theme); ok {
		staged.Name, staged.Theme = pack.Name, pack.Palette
	}
	return buildSettings(a, win, saved, staged, pageThemes)
}

// settingsState is the model behind one build of the Settings content.
type settingsState struct {
	a      *app.Application
	win    *app.Window
	saved  style.Appearance
	staged style.Appearance
	page   int
	filter int
	// listOff keeps the theme browser's scroll position across rebuilds:
	// picking a theme below the fold used to jump the list to the top.
	listOff float32
}

func (s *settingsState) rebuild() {
	s.win.SetContent(buildSettingsState(s))
}

func buildSettings(a *app.Application, win *app.Window, saved, staged style.Appearance, page int) widget.Component {
	return buildSettingsState(&settingsState{a: a, win: win, saved: saved.Normalize(), staged: staged.Normalize(), page: page})
}

func buildSettingsState(s *settingsState) widget.Component {
	if s.page < 0 || s.page >= len(settingsPages) {
		s.page = pageThemes
	}
	status := widgets.NewStatusBar(s.statusText(), style.AppearancePath(), "v"+uitoolkit.Version)
	chrome := widgets.NewTitleBar("Settings", "Themes from every decade for every uitoolkit app")

	// Stage a change: Settings keeps its look; the preview follows.
	stage := func(next style.Appearance) {
		s.staged = next.Normalize()
		s.rebuild()
	}
	apply := func() {
		next := s.staged.Normalize()
		if err := style.SaveAppearance(next); err != nil {
			status.Set(0, err.Error())
			return
		}
		s.saved = next
		s.a.SetLook(next.Look())
		s.rebuild()
	}
	revert := func() {
		s.staged = s.saved
		s.rebuild()
	}

	var page widget.Component
	switch s.page {
	case pagePacks:
		page = s.packsPage(status, stage)
	case pageAbout:
		page = widgets.NewScrollView(s.aboutPage())
	default:
		page = s.themesPage(status, stage)
	}

	nav := widgets.NewListView(len(settingsPages), func(i int) string { return settingsPages[i] }, func(i int) {
		s.page = i
		s.rebuild()
	})
	nav.Selected = s.page
	nav.RowHeight = 32
	side := widgets.NewColumn(
		widgets.NewTitle("Settings"),
		widgets.NewLabel("v"+uitoolkit.Version),
		nav,
	).WithGap(8).WithPad(10)
	side.AddFlex(nav, 1)

	right := widgets.NewPad(10, page)
	split := widgets.NewSplitter(true, side, right)
	split.Ratio = 0.18

	applyBtn := widgets.NewButton("Apply", apply)
	applyBtn.Primary = true
	revertBtn := widgets.NewButton("Revert", revert)
	if s.staged == s.saved {
		applyBtn.SetEnabled(false)
		revertBtn.SetEnabled(false)
	}
	hint := widgets.NewLabel("Apply writes look.json; running apps switch live. Closing without Apply discards the staged theme.")
	actions := widgets.NewRow(applyBtn, revertBtn, hint).WithGap(12).WithPad(8)

	root := widgets.NewColumn(chrome, split, actions, status)
	root.AddFlex(split, 1)
	return root
}

func (s *settingsState) statusText() string {
	st := s.staged.Name + " · " + string(s.staged.Corners) + " · " + string(s.staged.Icons) + " · " + string(s.staged.IconSize)
	if s.staged != s.saved {
		st += " · unapplied"
	}
	return st
}

// ---- Themes page --------------------------------------------------------------

// themeRow is one entry of the theme browser.
type themeRow struct {
	pack style.ThemePack
	user bool
}

func (s *settingsState) themeRows() []themeRow {
	var rows []themeRow
	if s.filter != filterUser {
		for _, p := range style.ListBuiltinThemes() {
			if s.filter == 0 || style.Decade(p.Year) == themeFilters[s.filter] {
				rows = append(rows, themeRow{pack: p})
			}
		}
	}
	if s.filter == 0 || s.filter == filterUser {
		for _, p := range style.ListUserThemes() {
			rows = append(rows, themeRow{pack: p, user: true})
		}
	}
	return rows
}

func themeRowText(r themeRow) string {
	switch {
	case r.user:
		return "User  ·  " + r.pack.Display()
	case r.pack.Year > 0:
		return fmt.Sprintf("%d  ·  %s", r.pack.Year, r.pack.Display())
	default:
		return r.pack.Display()
	}
}

func (s *settingsState) themesPage(status *widgets.StatusBar, stage func(style.Appearance)) widget.Component {
	rows := s.themeRows()
	sel := -1
	for i, r := range rows {
		if r.pack.Name == s.staged.Name {
			sel = i
		}
	}
	filters := widgets.NewComboBox(themeFilters, s.filter, func(i int) {
		s.filter = i
		s.listOff = 0
		s.rebuild()
	})
	var list *widgets.ListView
	list = widgets.NewListView(len(rows), func(i int) string { return themeRowText(rows[i]) }, func(i int) {
		if i < 0 || i >= len(rows) {
			return
		}
		s.listOff = list.OffsetY
		next := s.staged
		next.Name = rows[i].pack.Name
		next.Theme = rows[i].pack.Palette
		stage(next)
	})
	list.Selected = sel
	list.RowHeight = 28
	list.OffsetY = s.listOff
	// The staged theme may sit below the fold (Aqua is row 30): bring it
	// into view; a row the user just clicked is already there.
	list.EnsureVisible(sel)
	browser := widgets.NewColumn(widgets.NewLabel("Themes"), filters, list).WithGap(6)
	browser.AddFlex(list, 1)

	// Pack details.
	pack, _ := style.LoadTheme(s.staged.Name)
	meta := []string{}
	if pack.Year > 0 {
		meta = append(meta, fmt.Sprint(pack.Year))
	}
	if pack.Lineage != "" {
		meta = append(meta, pack.Lineage)
	}
	eng := pack.Tokens.Engine
	if eng == "" {
		eng = "base"
	}
	meta = append(meta, "engine "+eng)
	if pack.Source == style.ThemeSourceUser {
		meta = append(meta, "user pack")
	}
	info := widgets.NewColumn(
		widgets.NewTitle(pack.Display()),
		widgets.NewLabel(strings.Join(meta, "  ·  ")),
	).WithGap(2)
	if pack.Summary != "" {
		info.Add(widgets.NewLabel(pack.Summary))
	}

	// The live preview: a small application window in the staged theme —
	// its frame, caption and every control come from that theme.
	frame := widgets.NewPanel("Preview — "+pack.Display(), PreviewApp(func(msg string) { status.Set(0, msg) }))
	frame.Window = true
	scope := widgets.NewThemeScope(s.staged.Look(), frame)

	// Options that shape the staged look (shown live in the preview).
	cornerSel := 0
	switch s.staged.Corners {
	case style.CornersRound:
		cornerSel = 1
	case style.CornersSquare:
		cornerSel = 2
	}
	corners := widgets.NewComboBox([]string{"Theme shape", "Round", "Square"}, cornerSel, func(i int) {
		next := s.staged
		next.Corners = []style.CornerStyle{style.CornersTheme, style.CornersRound, style.CornersSquare}[i]
		stage(next)
	})
	sizeSel := 1
	switch s.staged.IconSize {
	case style.IconSizeSmall:
		sizeSel = 0
	case style.IconSizeLarge:
		sizeSel = 2
	}
	sizes := widgets.NewComboBox([]string{"Small", "Medium", "Large"}, sizeSel, func(i int) {
		next := s.staged
		next.IconSize = []style.IconSize{style.IconSizeSmall, style.IconSizeMedium, style.IconSizeLarge}[i]
		stage(next)
	})
	sets := append(style.ListBuiltinIconSets(), style.ListUserIconSets()...)
	names := make([]string, len(sets))
	iconSel := 0
	for i, set := range sets {
		names[i] = set.Label
		if set.Name == s.staged.Icons {
			iconSel = i
		}
	}
	icons := widgets.NewComboBox(names, iconSel, func(i int) {
		if i < 0 || i >= len(sets) {
			return
		}
		next := s.staged
		next.Icons = sets[i].Name
		stage(next)
	})
	options := widgets.NewRow(
		widgets.NewLabel("Corners"), corners,
		widgets.NewLabel("Icon size"), sizes,
		widgets.NewLabel("Icons"), icons,
	).WithGap(8).WithAlign(layout.AlignCenter)
	options.AddFlex(corners, 1)
	options.AddFlex(sizes, 1)
	options.AddFlex(icons, 1)

	preview := widgets.NewColumn(info, scope, options).WithGap(10)
	preview.AddFlex(scope, 1)

	body := widgets.NewRow(browser, preview).WithGap(14)
	body.AddFlex(browser, 4)
	body.AddFlex(preview, 7)
	return body
}

// PreviewApp is a small, fully interactive application used to preview a
// theme: menu bar, tool bar, tabs with every kind of control, lists, a
// tree, a table and a status bar. Settings shows it inside a ThemeScope.
func PreviewApp(say func(string)) widget.Component {
	if say == nil {
		say = func(string) {}
	}
	menu := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemIconAccel(style.IconNew, "&New", "Ctrl+N", func() { say("New") }),
			widgets.ItemIconAccel(style.IconOpen, "&Open…", "Ctrl+O", func() { say("Open") }),
			widgets.ItemIconAccel(style.IconSave, "&Save", "Ctrl+S", func() { say("Save") }),
			widgets.Sep(),
			widgets.Item("E&xit", func() { say("Exit") }),
		),
		widgets.NewMenu("&Edit",
			widgets.ItemIconAccel(style.IconCut, "Cu&t", "Ctrl+X", nil),
			widgets.ItemIconAccel(style.IconCopy, "&Copy", "Ctrl+C", nil),
			widgets.ItemIconAccel(style.IconPaste, "&Paste", "Ctrl+V", nil),
		),
		widgets.NewMenu("&View",
			widgets.CheckItem("Status &bar", true, nil),
			widgets.CheckItem("&Word wrap", false, nil),
		),
		widgets.NewMenu("&Help", widgets.Item("&About", func() { say("About") })),
	)
	bold := widgets.ToolIconBtn(style.IconPen, "", nil)
	bold.Toggle, bold.Down = true, true
	tools := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconNew, "", func() { say("New") }),
		widgets.ToolIconBtn(style.IconOpen, "", func() { say("Open") }),
		widgets.ToolIconBtn(style.IconSave, "", func() { say("Save") }),
		widgets.ToolDivider(),
		widgets.ToolIconBtn(style.IconCut, "", nil),
		widgets.ToolIconBtn(style.IconCopy, "", nil),
		widgets.ToolIconBtn(style.IconPaste, "", nil),
		widgets.ToolDivider(),
		bold,
		widgets.ToolIconBtn(style.IconMail, "Send", func() { say("Send") }),
	)

	ok := widgets.NewButton("Default", func() { say("Default button") })
	ok.Primary = true
	normal := widgets.NewButton("Button", func() { say("Button") })
	off := widgets.NewButton("Disabled", nil)
	off.SetEnabled(false)
	dialog := widgets.NewButton("Dialog…", nil)
	dialog.OnClick = func() {
		widgets.Confirm(dialog, "Save changes?", "Your document has unsaved changes.", func(yes bool) {
			say(fmt.Sprint("Save changes: ", yes))
		})
	}
	progress := widgets.NewProgressBar(0.62)
	left := widgets.NewColumn(
		widgets.NewRow(ok, normal).WithGap(8),
		widgets.NewRow(off, dialog).WithGap(8),
		widgets.NewCheckbox("Check box", true, nil),
		widgets.NewCheckbox("Unchecked", false, nil),
		widgets.NewRadioGroup([]string{"Radio one", "Radio two"}, 0, nil),
	).WithGap(8)
	right := widgets.NewColumn(
		widgets.NewTextField("Ada Lovelace", "Name", nil),
		widgets.NewRow(
			widgets.NewComboBox([]string{"Combo box", "Second choice", "Third choice"}, 0, nil),
			widgets.NewNumberField(0, 99, 3, 1, nil),
		).WithGap(8),
		widgets.NewSwitch("Switch", true, nil),
		widgets.NewSlider(0, 100, 40, nil),
		progress,
	).WithGap(8)
	controls := widgets.NewRow(left, right).WithGap(16).WithPad(8)
	controls.AddFlex(right, 1)

	tree := widgets.NewTreeView(previewTree())
	tree.SetPreferred(160, 0)
	table := widgets.NewTableView(
		[]widgets.TableColumn{{Title: "Name", Width: 120}, {Title: "Size", Width: 60, Align: style.AlignEnd}, {Title: "Kind", Width: 90}},
		8, func(row, col int) string {
			return [][]string{
				{"README.md", "4 KB", "Text"}, {"go.mod", "1 KB", "Module"}, {"look.go", "38 KB", "Go"},
				{"engine.go", "21 KB", "Go"}, {"theme.json", "2 KB", "JSON"}, {"icons", "—", "Folder"},
				{"fonts", "—", "Folder"}, {"LICENSE", "1 KB", "Text"},
			}[row][col]
		}, nil)
	table.Selected = 2
	lists := widgets.NewSplitter(true, tree, table)
	lists.Ratio = 0.36

	text := widgets.NewTextArea("Themes change shapes, not just colours:\nbevels, gel buttons, chamfered tabs,\nscrollbar arrows and window captions.", "", nil)
	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Controls", Content: controls},
		widgets.Tab{Title: "Lists", Content: lists},
		widgets.Tab{Title: "Text", Content: text},
	)

	sb := widgets.NewStatusBar("Ready", "Ln 1, Col 1", "100%")
	win := widgets.NewColumn(menu, tools, tabs, sb)
	win.AddFlex(tabs, 1)
	return win
}

func previewTree() *widgets.TreeNode {
	inbox := widgets.NewTreeNode("Inbox")
	inbox.Bold = true
	return widgets.NewTreeNode("Mail",
		inbox,
		widgets.NewTreeNode("Archives", widgets.NewTreeNode("2025"), widgets.NewTreeNode("2026")),
		widgets.NewTreeNode("Sent"),
		widgets.NewTreeNode("Trash"),
	)
}

// ---- Packs & icons page ---------------------------------------------------

func (s *settingsState) packsPage(status *widgets.StatusBar, stage func(style.Appearance)) widget.Component {
	userThemes := style.ListUserThemes()
	userSel := indexTheme(userThemes, s.staged.Name)
	themes := pickerSection("User", len(userThemes), func(i int) string {
		return userThemes[i].Display()
	}, userSel, func(i int) {
		if i >= 0 && i < len(userThemes) {
			next := s.staged
			next.Name = userThemes[i].Name
			next.Theme = userThemes[i].Palette
			stage(next)
		}
	})
	themeCol := widgets.NewColumn(widgets.NewTitle("Theme packs"),
		widgets.NewLabel("Built-in packs live in the theme browser. Export saves the staged theme's colours and metrics as a pack you can edit."),
		themes).WithGap(8)
	var exportHost widget.Component
	exportBtn := widgets.NewButton("Export current theme…", func() {
		promptExportName(exportHost, func(name string) {
			pack, err := style.ExportAppearance(strings.TrimSpace(name), s.staged)
			if err != nil {
				status.Set(0, err.Error())
				return
			}
			next := s.staged
			next.Name = pack.Name
			next.Theme = pack.Palette
			stage(next)
		})
	})
	exportHost = exportBtn
	themeCol.Add(exportBtn)
	if userSel >= 0 {
		var delHost widget.Component
		del := widgets.NewButton("Delete", func() {
			name := userThemes[userSel].Name
			widgets.Confirm(delHost, "Delete theme?", "Remove "+name+" from disk? This cannot be undone.", func(yes bool) {
				if !yes {
					return
				}
				if err := style.DeleteUserTheme(name); err != nil {
					status.Set(0, err.Error())
					return
				}
				next := style.AfterUserThemeDeleted(s.staged, name)
				if s.saved.Name == name {
					if err := style.SaveAppearance(next); err != nil {
						status.Set(0, err.Error())
						return
					}
					s.saved = next
				}
				s.a.SetLook(s.saved.Look())
				stage(next)
			})
		})
		delHost = del
		themeCol.Add(del)
	}
	themeCol.AddFlex(themes, 1)

	iconBuiltin := style.ListBuiltinIconSets()
	iconUser := style.ListUserIconSets()
	onIcon := func(set style.IconSetInfo) {
		next := s.staged
		next.Icons = set.Name
		stage(next)
	}
	builtinIcons := pickerSection("Built-in", len(iconBuiltin), func(i int) string {
		return iconBuiltin[i].Label
	}, indexIcon(iconBuiltin, s.staged.Icons), func(i int) {
		if i >= 0 && i < len(iconBuiltin) {
			onIcon(iconBuiltin[i])
		}
	})
	userIconSel := indexIcon(iconUser, s.staged.Icons)
	userIcons := pickerSection("User", len(iconUser), func(i int) string {
		return iconUser[i].Label
	}, userIconSel, func(i int) {
		if i >= 0 && i < len(iconUser) {
			onIcon(iconUser[i])
		}
	})
	iconCol := widgets.NewColumn(widgets.NewTitle("Icon sets"), builtinIcons, userIcons).WithGap(8)
	iconCol.AddFlex(builtinIcons, 1)
	if len(iconUser) > 0 {
		iconCol.AddFlex(userIcons, 1)
	}
	if userIconSel >= 0 {
		var delHost widget.Component
		del := widgets.NewButton("Delete", func() {
			name := iconUser[userIconSel].Name
			widgets.Confirm(delHost, "Delete icon set?", "Remove "+string(name)+" from disk? This cannot be undone.", func(yes bool) {
				if !yes {
					return
				}
				if err := style.DeleteUserIconSet(name); err != nil {
					status.Set(0, err.Error())
					return
				}
				next := s.staged
				if next.Icons == name {
					next.Icons = style.IconSetClassic
				}
				if s.saved.Icons == name {
					if err := style.SaveAppearance(next); err != nil {
						status.Set(0, err.Error())
						return
					}
					s.saved = next
				}
				s.a.SetLook(s.saved.Look())
				stage(next)
			})
		})
		delHost = del
		iconCol.Add(del)
	}
	body := widgets.NewRow(themeCol, iconCol).WithGap(20)
	body.AddFlex(themeCol, 1)
	body.AddFlex(iconCol, 1)
	return body
}

// ---- About page -----------------------------------------------------------------

func (s *settingsState) aboutPage() widget.Component {
	mono := func(text string) widget.Component {
		v := widgets.NewMonoTextView(text, "")
		v.MinRows = 2
		v.Wrap = true
		return v
	}
	engines := strings.Join(style.EngineIDs(), ", ")
	return widgets.NewColumn(
		widgets.NewTitle("About"),
		widgets.NewLabel("uitoolkit v"+uitoolkit.Version),
		widgets.NewLabel(fmt.Sprintf("%d built-in themes from %d theme engines: %s.", len(style.ListBuiltinThemes()), len(style.EngineIDs()), engines)),
		widgets.NewLabel("A theme is a pack (colours, metrics) painted by an engine (shapes): Windows 95 bevels, Aqua gel, Motif shadows…"),
		widgets.NewLabel("Prefs file (theme + corners + icons + iconSize, written on Apply):"),
		mono(style.AppearancePath()),
		widgets.NewLabel("User theme packs (exported; edit the JSON to make your own):"),
		mono(style.ThemesDir()+"/<name>/theme.json"),
		widgets.NewLabel("Icon sets (copy the repo icons/ folders here after every pull):"),
		mono(style.IconsDir()+"/<set>/*.png"),
		widgets.NewLabel("Other apps watch look.json and switch live on Apply."),
		widgets.NewButton("Browse themes", func() {
			s.page = pageThemes
			s.rebuild()
		}),
	).WithGap(10).WithPad(4)
}

// ---- helpers ------------------------------------------------------------------

func pickerSection(title string, count int, text func(int) string, selected int, on func(int)) widget.Component {
	head := widgets.NewLabel(title)
	if count == 0 {
		return widgets.NewColumn(head, widgets.NewLabel("None yet")).WithGap(4)
	}
	list := widgets.NewListView(count, text, on)
	list.Selected = selected
	list.RowHeight = 28
	col := widgets.NewColumn(head, list).WithGap(4)
	col.AddFlex(list, 1)
	return col
}

func indexTheme(packs []style.ThemePack, name string) int {
	for i, p := range packs {
		if p.Name == name {
			return i
		}
	}
	return -1
}

func indexIcon(sets []style.IconSetInfo, name style.IconSetName) int {
	for i, s := range sets {
		if s.Name == name {
			return i
		}
	}
	return -1
}

func promptExportName(from widget.Component, on func(string)) {
	if from == nil {
		return
	}
	field := widgets.NewTextField("", "theme name", nil)
	var overlay *widgets.Overlay
	finish := func(ok bool) {
		name := strings.TrimSpace(field.Text)
		widget.DismissOverlay(overlay)
		if ok && on != nil {
			on(name)
		}
	}
	cancel := widgets.NewButton("Cancel", func() { finish(false) })
	ok := widgets.NewButton("Export", func() { finish(true) })
	ok.Primary = true
	field.OnSubmit = func(string) { finish(true) }
	card := widgets.NewPanel("Export theme",
		widgets.NewLabel("Name the theme pack. It is written to ~/.config/uitoolkit/themes/<name>/theme.json with the theme's colours, engine and metrics. Corners, icons and icon size stay in look.json."),
		field,
		widgets.NewRow(cancel, ok).WithGap(8).WithJustify(layout.JustifyEnd),
	)
	card.Window = true
	card.Raised = true
	card.OnClose = func() { finish(false) }
	overlay = widgets.NewOverlay(card)
	overlay.Modal = true
	overlay.InitialFocus = field
	overlay.MinCardH = 200
	widget.ShowOverlay(from, overlay)
}
