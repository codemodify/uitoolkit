// Package settingsapp is the toolkit's appearance editor: the theme
// browser, the live preview with the whole widget showcase under it, the
// icon and corner options, and the Apply that writes look.json.
//
// It lives beside its command rather than inside it because the
// end-to-end driver and its own tests drive it as a library, and because
// it is a sample like the others: it is written against the published
// API — [showcase.Pane] for the gallery under the preview, the style
// package for themes — and nothing under internal/.
package settingsapp

import (
	"fmt"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/showcase"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Settings pages.
const (
	pageThemes = iota
	pageAppearance
	pagePacks
	pageAbout
)

var settingsPages = []string{"Themes", "Appearance", "Packs & icons", "About"}

// Theme browser filters: every built-in pack, one decade, or user packs.
var themeFilters = []string{"All decades", "1980s", "1990s", "2000s", "2010s", "2020s", "My themes"}

const filterUser = 6

// SettingsApp is the toolkit appearance editor. The Themes page is the
// toolkit's shop window: pick a pack and it is drawn at once in a live
// preview window and, under it, in the whole widget gallery — every
// control the toolkit has, in every state, in that theme — while Settings
// itself keeps the applied look (a ThemeScope each). Apply — the one
// button, at the right of the row under the pages — writes look.json and
// every app that watches it (and Settings) switches. Closing without
// Apply discards the staged change.
func SettingsApp(a *app.Application, win *app.Window) widget.Component {
	saved := style.LoadAppearance().Normalize()
	return buildSettings(a, win, saved, saved, pageThemes)
}

// SettingsAppStaged is SettingsApp with a theme already staged (not
// applied): screenshots and docs use it to show the preview and the
// gallery in a theme other than the one Settings runs in.
func SettingsAppStaged(a *app.Application, win *app.Window, theme string) widget.Component {
	return SettingsAppOpen(a, win, theme, "")
}

// SettingsAppOpen is SettingsApp with a theme staged and a page open by
// name ("themes", "appearance", "packs", "about"); either may be empty.
// cmd/uitksettings' -stage and -page take this path.
func SettingsAppOpen(a *app.Application, win *app.Window, theme, page string) widget.Component {
	saved := style.LoadAppearance().Normalize()
	staged := saved
	if pack, ok := style.LoadTheme(theme); ok {
		staged.Name, staged.Theme = pack.Name, pack.Palette
	}
	return buildSettings(a, win, saved, staged, SettingsPage(page))
}

// SettingsPage is the page a name opens; an unknown or empty name is the
// theme browser.
func SettingsPage(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "appearance":
		return pageAppearance
	case "packs", "packs & icons", "icons":
		return pagePacks
	case "about":
		return pageAbout
	default:
		return pageThemes
	}
}

// settingsState is the model behind one build of the Settings content.
type settingsState struct {
	a      *app.Application
	win    *app.Window
	saved  style.Appearance
	staged style.Appearance
	page   int
	filter int
	// query narrows the theme browser to what it names: a pack's id,
	// name, year, family, engine or summary.
	query string
	// listOff keeps the theme browser's scroll position across rebuilds:
	// picking a theme below the fold used to jump the list to the top.
	listOff float32
	// browserRatio and previewRatio are where the user left the two
	// splitters of the Themes page, galleryOff how far down the gallery
	// was scrolled: all three survive a rebuild.
	browserRatio float32
	// browserAuto is the ratio Settings worked out for itself, to tell it
	// apart from one the user dragged.
	browserAuto  float32
	previewRatio float32
	galleryOff   float32
	// scheme is the desktop's light / dark preference the page was built in.
	scheme style.ColorScheme

	// The live parts of the page the staged look feeds. Staging a change
	// repaints these where they stand instead of rebuilding the page: a
	// rebuild takes the caret out of the search field, the focus off the
	// theme list, and the gallery back to the top.
	rows       []themeRow
	list       *widgets.ListView
	bodySplit  *widgets.Splitter
	rightSplit *widgets.Splitter
	gallery    *widgets.ScrollView
	scopes     []*widgets.ThemeScope
	previewBox *widgets.Panel
	packTitle  *widgets.Label
	packMeta   *widgets.Label
	packNote   *widgets.Label
	galleryLbl *widgets.Label
	applyBtn   *widgets.Button
	hint       *widgets.Label
}

// capture remembers where the user left the parts of the page they can
// move, so the next build puts them back.
func (s *settingsState) capture() {
	if s.list != nil {
		s.listOff = s.list.OffsetY
	}
	if s.bodySplit != nil {
		// Only what the user dragged is kept: a ratio still ours is
		// worked out again from the window, which by now has the size the
		// desktop gave it rather than the one it asked for.
		s.browserRatio = 0
		if s.bodySplit.Ratio != s.browserAuto {
			s.browserRatio = s.bodySplit.Ratio
		}
	}
	if s.rightSplit != nil {
		s.previewRatio = s.rightSplit.Ratio
	}
	if s.gallery != nil {
		s.galleryOff = s.gallery.OffsetY
	}
}

func (s *settingsState) rebuild() {
	s.capture()
	s.win.SetContent(buildSettingsState(s))
}

func buildSettings(a *app.Application, win *app.Window, saved, staged style.Appearance, page int) widget.Component {
	s := &settingsState{
		a: a, win: win, saved: saved.Normalize(), staged: staged.Normalize(), page: page,
		previewRatio: 0.62,
	}
	// The preview draws the staged theme's light or dark sibling: redraw
	// it when the desktop switches.
	a.OnLookChange(func() {
		if now := style.DesktopColorScheme(); now != s.scheme {
			s.rebuild()
		}
	})
	// The theme browser keeps its share as the window is resized, until
	// the user drags the sash: then the split is theirs.
	win.OnResize(func(int, int) { s.followWindow() })
	return buildSettingsState(s)
}

func buildSettingsState(s *settingsState) widget.Component {
	s.scheme = style.DesktopColorScheme()
	if s.page < 0 || s.page >= len(settingsPages) {
		s.page = pageThemes
	}
	// Every live part belongs to the build that made it.
	s.rows, s.list, s.bodySplit, s.rightSplit, s.gallery = nil, nil, nil, nil, nil
	s.scopes, s.previewBox = nil, nil
	s.packTitle, s.packMeta, s.packNote, s.galleryLbl = nil, nil, nil, nil

	var page widget.Component
	switch s.page {
	case pageAppearance:
		page = widgets.NewScrollView(s.appearancePage())
	case pagePacks:
		page = s.packsPage()
	case pageAbout:
		page = widgets.NewScrollView(s.aboutPage())
	default:
		page = s.themesPage()
	}

	nav := widgets.NewListView(len(settingsPages), func(i int) string { return settingsPages[i] }, func(i int) {
		s.page = i
		s.rebuild()
	})
	nav.SetAccessibleName("Pages")
	nav.Selected = s.page
	nav.RowHeight = 32
	nav.Sidebar = true
	side := widgets.NewColumn(
		widgets.NewTitle("Settings"),
		widgets.NewLabel("v"+uitoolkit.Version),
		nav,
	).WithGap(8).WithPad(10)
	side.AddFlex(nav, 1)

	right := widgets.NewPad(10, page)
	split := widgets.NewSplitter(widgets.SplitColumns, side, right)
	split.Ratio = 0.18

	s.applyBtn = widgets.NewButton("Apply", s.apply)
	s.applyBtn.Primary = true
	// One line, elided: a hint that wrapped pushed the button off a
	// small window. It leads the row and Apply closes it on the right,
	// where a dialog's accept button belongs.
	s.hint = widgets.NewLabel("")
	actions := widgets.NewRow(s.hint, s.applyBtn).WithGap(12).WithPad(8)
	actions.AddFlex(s.hint, 1)

	s.showStaged()

	root := widgets.NewColumn(split, actions)
	root.AddFlex(split, 1)
	return root
}

// ---- staging ------------------------------------------------------------------

// stage shows next without rebuilding the page: the preview and the
// gallery switch theme where they stand, the pack's details follow, and
// Apply and the line beside it say what is staged.
func (s *settingsState) stage(next style.Appearance) {
	s.staged = next.Normalize()
	s.showStaged()
}

func (s *settingsState) apply() {
	next := s.staged.Normalize()
	if err := style.SaveAppearance(next); err != nil {
		s.say(err.Error())
		return
	}
	s.saved = next
	s.a.ApplyAppearance(next)
	s.rebuild()
}

// say puts a message where the state line sits, beside Apply: it is the
// one line of prose the window has left for what went wrong.
func (s *settingsState) say(msg string) {
	if s.hint != nil {
		s.hint.SetText(msg)
	}
}

// showStaged paints the staged appearance into the parts of the page that
// follow it. Each is nil on the pages that do not show it.
func (s *settingsState) showStaged() {
	pack, _ := style.LoadTheme(s.staged.Name)
	shown := pack
	note := s.followNote(pack, &shown)
	look := s.staged.Look()
	for _, sc := range s.scopes {
		sc.SetTheme(look)
	}
	if s.previewBox != nil {
		s.previewBox.Title = "Preview — " + shown.Display()
		s.previewBox.RequestLayout()
	}
	if s.galleryLbl != nil {
		s.galleryLbl.SetText("Every control the toolkit has, in " + shown.Display())
	}
	if s.packTitle != nil {
		s.packTitle.SetText(pack.Display())
		s.packMeta.SetText(packMetaLine(pack))
		s.packNote.SetText(note)
	}
	if s.list != nil {
		s.list.Selected = s.selectedRow()
		s.list.Invalidate()
	}
	if s.applyBtn != nil {
		s.applyBtn.SetEnabled(s.staged != s.saved)
		s.hint.SetText(s.hintText())
	}
}

// hintText says, beside Apply, where the staged look has got to.
func (s *settingsState) hintText() string {
	if s.staged == s.saved {
		return "Applied — every uitoolkit app is using this look."
	}
	return "Staged, not applied — only the preview and the gallery show it."
}

// ---- Themes page --------------------------------------------------------------

// themeRow is one entry of the theme browser. A pack is whatever an
// engine — or a skin of pictures — makes of its tokens; the browser only
// reads what the pack says about itself.
type themeRow struct {
	pack style.ThemePack
	user bool
}

func (s *settingsState) themeRows() []themeRow {
	var rows []themeRow
	if s.filter != filterUser {
		for _, p := range style.ListBuiltinThemes() {
			if (s.filter == 0 || style.Decade(p.Year) == themeFilters[s.filter]) && matchesQuery(p, s.query) {
				rows = append(rows, themeRow{pack: p})
			}
		}
	}
	if s.filter == 0 || s.filter == filterUser {
		for _, p := range style.ListUserThemes() {
			if matchesQuery(p, s.query) {
				rows = append(rows, themeRow{pack: p, user: true})
			}
		}
	}
	return rows
}

// matchesQuery is what the search field looks at: a pack's id, name,
// year, family, era, engine and summary, so "kde", "1995", "gel" and
// "oxygen" all find something. Every word has to match.
func matchesQuery(p style.ThemePack, q string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return true
	}
	hay := strings.ToLower(strings.Join([]string{
		p.Name, p.Display(), p.Lineage, p.Era, p.Summary, p.Tokens.Engine, fmt.Sprint(p.Year),
	}, " "))
	for _, word := range strings.Fields(strings.ToLower(q)) {
		if !strings.Contains(hay, word) {
			return false
		}
	}
	return true
}

// themeRowText is one row of the browser: when the look shipped and the
// pack's name. Its family, engine and summary are under the list, where
// there is room for them; the search field reads all of them.
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

// packMetaLine is the line under the pack's name: year, family, and the
// engine that paints it (a skin pack names its own).
func packMetaLine(p style.ThemePack) string {
	var meta []string
	if p.Year > 0 {
		meta = append(meta, fmt.Sprint(p.Year))
	}
	if p.Lineage != "" {
		meta = append(meta, p.Lineage)
	}
	eng := p.Tokens.Engine
	if eng == "" {
		eng = "base"
	}
	meta = append(meta, "engine "+eng)
	if p.Source == style.ThemeSourceUser {
		meta = append(meta, "user pack")
	}
	return strings.Join(meta, "  ·  ")
}

func (s *settingsState) selectedRow() int {
	for i, r := range s.rows {
		if r.pack.Name == s.staged.Name {
			return i
		}
	}
	return -1
}

// themesPage is the shop window: the browser on the left, and on the
// right the staged pack drawn twice — a small application window above,
// the whole widget gallery below — in a splitter the user can size, so
// the two never fight for the space.
func (s *settingsState) themesPage() widget.Component {
	if s.browserRatio == 0 {
		s.browserRatio = defaultBrowserRatio(s.win)
		s.browserAuto = s.browserRatio
	}
	browser := s.themeBrowser()

	// The live preview: a small application window in the staged theme —
	// its frame, caption and every control come from that theme.
	s.previewBox = widgets.NewPanel("", PreviewApp(nil))
	s.previewBox.Window = true
	preview := s.scoped(s.previewBox)

	// Under it the same theme over the whole toolkit: every control in
	// every state, scrolling on its own.
	s.galleryLbl = widgets.NewLabel("")
	var galleryRoot widget.Component
	s.gallery = showcase.Pane(showcase.Host{
		Light: s.staged.Effective().Theme == style.ThemeLight,
		// Dialogs go up from inside the scope, so they are drawn in the
		// staged theme too. The gallery here previews a pack rather than
		// being an app of its own: its Window, Theme and Quit controls
		// would open a window in Settings' look, re-theme Settings or
		// close it, so the host leaves them out and they show disabled.
		Root: func() widget.Component { return galleryRoot },
	})
	galleryRoot = s.gallery
	s.gallery.OffsetY = s.galleryOff
	galleryScope := s.scoped(s.gallery)
	galleryCol := widgets.NewColumn(s.galleryLbl, galleryScope).WithGap(6)
	galleryCol.AddFlex(galleryScope, 1)

	s.rightSplit = widgets.NewSplitter(widgets.SplitRows, preview, galleryCol)
	s.rightSplit.Ratio = s.previewRatio
	s.rightSplit.SetAccessibleName("Preview and gallery")

	s.bodySplit = widgets.NewSplitter(widgets.SplitColumns, browser, s.rightSplit)
	s.bodySplit.Ratio = s.browserRatio
	s.bodySplit.SetAccessibleName("Themes and preview")
	return s.bodySplit
}

// followWindow works the browser's share out again for the window's new
// size, unless the user has dragged the sash since Settings last set it.
func (s *settingsState) followWindow() {
	if s.bodySplit == nil || s.bodySplit.Ratio != s.browserAuto {
		return
	}
	s.browserAuto = defaultBrowserRatio(s.win)
	s.bodySplit.Ratio = s.browserAuto
	s.bodySplit.RequestLayout()
}

// defaultBrowserRatio aims the theme browser at about 240 logical pixels
// whatever the window and the display scale are: a narrow window gives it
// a bigger share so its rows stay readable, a wide one hands the room to
// the preview and the gallery. Dragging the sash replaces it.
func defaultBrowserRatio(win *app.Window) float32 {
	const want, pad = 240, 30
	// Window.Size is already logical pixels, which is what `want` is in.
	lw, _ := win.Size()
	// The page is what the navigation sidebar and its padding leave.
	page := float32(lw)*(1-0.18) - pad
	if page <= 0 {
		return 0.3
	}
	return min(max(want/page, 0.24), 0.45)
}

// scoped draws c in the staged theme while Settings keeps the applied one.
func (s *settingsState) scoped(c widget.Component) *widgets.ThemeScope {
	scope := widgets.NewThemeScope(s.staged.Look(), c)
	s.scopes = append(s.scopes, scope)
	return scope
}

// themeBrowser is the left column: a search field, the decade filter and
// the packs that pass both. Typing filters the list in place, so the
// caret stays in the field.
func (s *settingsState) themeBrowser() widget.Component {
	s.rows = s.themeRows()
	s.list = widgets.NewListView(len(s.rows), func(i int) string {
		if i < 0 || i >= len(s.rows) {
			return ""
		}
		return themeRowText(s.rows[i])
	}, func(i int) {
		if i < 0 || i >= len(s.rows) {
			return
		}
		s.listOff = s.list.OffsetY
		next := s.staged
		next.Name = s.rows[i].pack.Name
		next.Theme = s.rows[i].pack.Palette
		s.stage(next)
	})
	s.list.RowHeight = 28
	s.list.OffsetY = s.listOff
	s.list.Selected = s.selectedRow()
	// The staged theme may sit below the fold (Aqua is row 30): bring it
	// into view; a row the user just clicked is already there.
	s.list.EnsureVisible(s.list.Selected)
	s.list.SetAccessibleName("Themes")

	count := widgets.NewLabel("")
	setCount := func() {
		count.SetText(fmt.Sprintf("%d of %d packs", len(s.rows), len(style.ListThemes())))
	}
	setCount()
	// Narrowing the list rebuilds nothing: the rows behind it change and
	// the list redraws, so the field keeps the caret and the focus.
	refilter := func() {
		s.rows = s.themeRows()
		s.list.Count = len(s.rows)
		s.list.Selected = s.selectedRow()
		s.list.OffsetY = 0
		s.list.EnsureVisible(s.list.Selected)
		s.list.RequestLayout()
		s.list.Invalidate()
		setCount()
	}
	search := widgets.NewTextField(s.query, "Search themes", func(q string) {
		s.query = q
		refilter()
	})
	search.SetAccessibleName("Search themes")
	// Return stages the first pack the search found; Escape empties the
	// field, which is the whole list again.
	search.OnSubmit = func(string) {
		if len(s.rows) > 0 {
			s.list.OnSelect(0)
		}
	}
	search.OnEscape = func() {
		if s.query == "" {
			return
		}
		search.SetText("")
	}
	filters := widgets.NewComboBox(themeFilters, s.filter, func(i int) {
		s.filter = i
		s.listOff = 0
		refilter()
	})
	filters.SetAccessibleName("Decade")

	col := widgets.NewColumn(widgets.NewLabel("Themes"), search, filters, s.list, count, s.packDetails()).WithGap(6)
	col.AddFlex(s.list, 1)
	return col
}

// packDetails is what the list has no room for: the staged pack's name,
// the year and family it comes from, the engine (or skin) that paints it,
// and what following the desktop does to it. The box is a fixed height so
// the list above it does not resize as the user arrows down the packs.
func (s *settingsState) packDetails() widget.Component {
	s.packTitle = widgets.NewTitle("")
	s.packMeta = widgets.NewLabel("")
	s.packMeta.Wrap = true
	s.packNote = widgets.NewLabel("")
	s.packNote.Wrap = true
	s.packNote.MinLines = 1
	// A note longer than the box scrolls rather than being cut off, and
	// on a short column the box gives the list its rows back.
	return widgets.NewHeightBoxShare(96, 0.20, widgets.NewScrollView(
		widgets.NewColumn(s.packTitle, s.packMeta, s.packNote).WithGap(2)))
}

// followNote says what following the desktop does to the staged pack, and
// sets shown to the pack the preview draws.
func (s *settingsState) followNote(pack style.ThemePack, shown *style.ThemePack) string {
	if !s.staged.FollowDesktop {
		return ""
	}
	var note string
	scheme := s.a.DesktopColorScheme()
	eff := s.staged.Effective()
	switch {
	case scheme == style.SchemeNoPreference:
		note = "The desktop has no light or dark preference: the theme shows as it is."
	case eff.Name != s.staged.Name:
		if p, ok := style.LoadTheme(eff.Name); ok {
			*shown = p
		}
		note = fmt.Sprintf("The desktop prefers %s: %s shows as %s.", scheme, pack.Display(), shown.Display())
	case (scheme == style.SchemeDark) != (pack.Palette == style.ThemeDark):
		note = fmt.Sprintf("The desktop prefers %s, but %s has no %s version.", scheme, pack.Display(), scheme)
	}
	if _, ok := style.DesktopAccent(); ok && style.TakesAccent(*shown) {
		note = strings.TrimSpace(note + " It takes the desktop's accent colour.")
	}
	return note
}

// PreviewApp is a small, fully interactive application used to preview a
// theme: menu bar, tool bar, tabs with every kind of control, lists, a
// tree, a table and a status bar. Settings shows it inside a ThemeScope.
func PreviewApp(say func(string)) widget.Component {
	// The preview's own status bar is where what it says goes when the
	// caller wants it nowhere else: Settings has no status bar of its own.
	sb := widgets.NewStatusBar("Ready", "Ln 1, Col 1", "100%")
	if say == nil {
		say = func(msg string) { sb.Set(0, msg) }
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
	// Four rows a side, the check boxes beside the radios: one control
	// per row ran to six, which a pack with 36-pixel controls cannot fit
	// in the preview at Settings' default split.
	checks := widgets.NewColumn(
		widgets.NewCheckbox("Check box", true, nil),
		widgets.NewCheckbox("Unchecked", false, nil),
	).WithGap(4) // a radio group's own gap, so the two pairs line up
	left := widgets.NewColumn(
		widgets.NewRow(ok, normal).WithGap(8),
		widgets.NewRow(off, dialog).WithGap(8),
		widgets.NewRow(checks, widgets.NewRadioGroup([]string{"Radio one", "Radio two"}, 0, nil)).WithGap(16),
	).WithGap(8)
	combo := widgets.NewComboBox([]string{"Combo box", "Second", "Third"}, 0, nil)
	combo.SetAccessibleName("Choice")
	spin := widgets.NewNumberField(0, 99, 3, 1, nil)
	spin.SetAccessibleName("Count")
	slider := widgets.NewSlider(0, 100, 40, nil)
	slider.SetAccessibleName("Level")
	// The combo box gives way: the spin box's arrows are what a narrow
	// preview would otherwise cut off.
	choice := widgets.NewRow(combo, spin).WithGap(8)
	choice.AddFlex(combo, 1)
	toggle := widgets.NewRow(widgets.NewSwitch("Switch", true, nil), slider).WithGap(12)
	toggle.AddFlex(slider, 1)
	right := widgets.NewColumn(
		widgets.NewTextField("Ada Lovelace", "Name", nil),
		choice,
		toggle,
		progress,
	).WithGap(8)
	controls := widgets.NewRow(left, right).WithGap(16).WithPad(8)
	controls.AddFlex(right, 1)

	tree := widgets.NewTreeView(previewTree())
	tree.SetAccessibleName("Folders")
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
	table.SetAccessibleName("Files")
	table.Selected = 2
	lists := widgets.NewSplitter(widgets.SplitColumns, tree, table)
	lists.Ratio = 0.36

	text := widgets.NewTextArea("Themes change shapes, not just colours:\nbevels, gel buttons, chamfered tabs,\nscrollbar arrows and window captions.", "", nil)
	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Controls", Content: controls},
		widgets.Tab{Title: "Lists", Content: lists},
		widgets.Tab{Title: "Text", Content: text},
	)

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

// ---- Appearance page ------------------------------------------------------

// appearancePage holds everything look.json carries besides the pack
// itself. Each control says in a line what it does; a strip of the staged
// theme beside them shows corners, icons and motion as they change, and
// the Themes page shows them over the whole toolkit.
func (s *settingsState) appearancePage() widget.Component {
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
		s.stage(next)
	})
	corners.SetAccessibleName("Corners")

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
		s.stage(next)
	})
	sizes.SetAccessibleName("Icon size")

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
		s.stage(next)
	})
	icons.SetAccessibleName("Icons")

	// Hover fades, the default button's pulse, busy bars: off for users
	// who get unwell from motion (GTK's gtk-enable-animations).
	motion := widgets.NewSwitch("Animations", !s.staged.ReduceMotion, func(on bool) {
		next := s.staged
		next.ReduceMotion = !on
		s.stage(next)
	})
	// GNOME's and Plasma's light / dark setting and accent colour: the
	// theme shows its sibling (Breeze and Breeze Dark) to match the
	// desktop, recoloured around its accent where the engine takes one.
	follow := widgets.NewSwitch("Follow the desktop's colours", s.staged.FollowDesktop, func(on bool) {
		next := s.staged
		next.FollowDesktop = on
		s.stage(next)
	})
	// The desktop's own file dialogs (the XDG portal's), as Qt and GTK
	// apps can use, instead of the themed ones.
	native := widgets.NewSwitch("Use the desktop's file dialogs", s.staged.NativeDialogs, func(on bool) {
		next := s.staged
		next.NativeDialogs = on
		s.stage(next)
	})
	// Chromium's switch: windows that draw their own title bar (Mail's,
	// with its tool bar in it) get the desktop's title bar and borders
	// instead, and their title bar becomes the first row.
	system := widgets.NewSwitch("Use system title bar and borders", s.staged.Decorations == style.DecorationsSystem, func(on bool) {
		next := s.staged
		next.Decorations = style.DecorationsAuto
		if on {
			next.Decorations = style.DecorationsSystem
		}
		s.stage(next)
	})
	// Where a title bar the toolkit draws puts its caption buttons: the
	// desktop's layout, or the theme's own (the Mac's traffic lights on
	// the left).
	themeButtons := widgets.NewSwitch("Place window buttons as the theme does", s.staged.CaptionButtons == style.CaptionButtonsTheme, func(on bool) {
		next := s.staged
		next.CaptionButtons = style.CaptionButtonsDesktop
		if on {
			next.CaptionButtons = style.CaptionButtonsTheme
		}
		s.stage(next)
	})

	shape := settingsSection("Shape and icons",
		optionRow("Corners", corners, "The pack's own shape, or rounded or square everywhere."),
		optionRow("Icon size", sizes, "How big tool bar and menu icons are drawn: 16, 24 or 32 pixels."),
		optionRow("Icons", icons, "The chrome icon set. Copy a folder into "+style.IconsDir()+" to add one."),
		optionSwitch(motion, "Hover fades, the default button's pulse and busy bars."),
	)
	if style.DesktopReducesMotion() {
		shape.Add(wrapped("The desktop asks for reduced motion, so animations stay off."))
	}
	desktop := settingsSection("The desktop",
		optionSwitch(follow, "Light or dark, and the accent colour, as the desktop asks for them."),
		optionSwitch(native, "KDE's and GNOME's own Open and Save dialogs, through the XDG portal."),
	)
	// Windows: who draws the frame and where its buttons go. A window that
	// is shaped, transparent or glass behind belongs in this section too.
	windows := settingsSection("Windows",
		optionSwitch(system, "The desktop draws the title bar and borders of every window."),
		optionSwitch(themeButtons, "Close, minimise and maximise where the theme's era put them."),
	)

	options := widgets.NewColumn(shape, desktop, windows).WithGap(12)
	// The staged theme beside the options, so corners, icons and their
	// size are seen changing without going back to the theme browser. It
	// keeps to its own height at the top of the page.
	below := widgets.NewSpacer()
	sample := widgets.NewColumn(s.scoped(appearanceSample()), below)
	sample.AddFlex(below, 1)
	body := widgets.NewRow(options, sample).WithGap(16)
	body.AddFlex(options, 1)
	return widgets.NewColumn(
		widgets.NewTitle("Appearance"),
		wrapped("What every uitoolkit app does with the theme. Apply writes these to "+style.AppearancePath()+"."),
		body,
	).WithGap(12).WithPad(4)
}

// wrapped is a label that wraps: prose on these pages is read, and a line
// elided at the window's edge hides the part that says where a file is.
func wrapped(text string) *widgets.Label {
	l := widgets.NewLabel(text)
	l.Wrap = true
	return l
}

// settingsSection is a titled group of option rows.
func settingsSection(title string, rows ...widget.Component) *widgets.Panel {
	return widgets.NewPanel(title, rows...)
}

// optionRow is a named control with the line that says what it does.
func optionRow(label string, control widget.Component, about string) widget.Component {
	desc := widgets.NewLabel(about)
	desc.Wrap = true
	gap := widgets.NewSpacer()
	head := widgets.NewRow(widgets.NewLabel(label), gap, control).WithGap(10)
	head.AddFlex(gap, 1)
	return widgets.NewColumn(head, desc).WithGap(2)
}

// optionSwitch is a switch whose own text is the setting, with the line
// that says what turning it on does.
func optionSwitch(sw *widgets.Switch, about string) widget.Component {
	desc := widgets.NewLabel(about)
	desc.Wrap = true
	return widgets.NewColumn(sw, desc).WithGap(2)
}

// appearanceSample is a strip of the staged theme beside the options:
// small enough to sit next to them, wide enough to show corners, icons
// and the icon size changing.
func appearanceSample() widget.Component {
	primary := widgets.NewButton("Default", nil)
	primary.Primary = true
	tools := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconNew, "", nil),
		widgets.ToolIconBtn(style.IconOpen, "", nil),
		widgets.ToolIconBtn(style.IconSave, "", nil),
		widgets.ToolDivider(),
		widgets.ToolIconBtn(style.IconCut, "", nil),
		widgets.ToolIconBtn(style.IconCopy, "", nil),
	)
	choice := widgets.NewComboBox([]string{"Combo box", "Second choice"}, 0, nil)
	choice.SetAccessibleName("Sample choice")
	level := widgets.NewSlider(0, 100, 40, nil)
	level.SetAccessibleName("Sample level")
	sample := widgets.NewPanel("Sample",
		tools,
		widgets.NewRow(primary, widgets.NewButton("Button", nil)).WithGap(8),
		widgets.NewTextField("Ada Lovelace", "Name", nil),
		choice,
		widgets.NewCheckbox("Check box", true, nil),
		level,
		widgets.NewProgressBar(0.62),
	)
	sample.Window = true
	return sample
}

// ---- Packs & icons page ---------------------------------------------------

func (s *settingsState) packsPage() widget.Component {
	stage := s.stage
	say := s.say
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
		wrapped("Built-in packs live in the theme browser. Export saves the staged theme's colours and metrics as a pack you can edit."),
		themes).WithGap(8)
	var exportHost widget.Component
	exportBtn := widgets.NewButton("Export current theme…", func() {
		promptExportName(exportHost, func(name string) {
			pack, err := style.ExportAppearance(strings.TrimSpace(name), s.staged)
			if err != nil {
				say(err.Error())
				return
			}
			next := s.staged
			next.Name = pack.Name
			next.Theme = pack.Palette
			s.staged = next.Normalize()
			s.rebuild()
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
					say(err.Error())
					return
				}
				next := style.AfterUserThemeDeleted(s.staged, name)
				if s.saved.Name == name {
					if err := style.SaveAppearance(next); err != nil {
						say(err.Error())
						return
					}
					s.saved = next
				}
				s.a.ApplyAppearance(s.saved)
				s.staged = next.Normalize()
				s.rebuild()
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
					say(err.Error())
					return
				}
				next := s.staged
				if next.Icons == name {
					next.Icons = style.IconSetClassic
				}
				if s.saved.Icons == name {
					if err := style.SaveAppearance(next); err != nil {
						say(err.Error())
						return
					}
					s.saved = next
				}
				s.a.ApplyAppearance(s.saved)
				s.staged = next.Normalize()
				s.rebuild()
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
	mono := func(name, text string) widget.Component {
		v := widgets.NewMonoTextView(text, "")
		v.MinRows = 2
		v.Wrap = true
		v.SetAccessibleName(name)
		return v
	}
	return widgets.NewColumn(
		widgets.NewTitle("About"),
		wrapped("Prefs file (theme + corners + icons + iconSize, written on Apply):"),
		mono("Prefs file", style.AppearancePath()),
		wrapped("User theme packs (exported; edit the JSON to make your own):"),
		mono("User theme packs", style.ThemesDir()+"/<name>/theme.json"),
		wrapped("Icon sets (copy the repo icons/ folders here after every pull):"),
		mono("Icon sets", style.IconsDir()+"/<set>/*.png"),
	).WithGap(10).WithPad(4)
}

// ---- helpers ------------------------------------------------------------------

func pickerSection(title string, count int, text func(int) string, selected int, on func(int)) widget.Component {
	head := widgets.NewLabel(title)
	if count == 0 {
		return widgets.NewColumn(head, widgets.NewLabel("None yet")).WithGap(4)
	}
	list := widgets.NewListView(count, text, on)
	list.SetAccessibleName(title)
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
		widgets.NewButtonBox().AddButton(cancel, widgets.RoleReject).AddButton(ok, widgets.RoleAccept),
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
