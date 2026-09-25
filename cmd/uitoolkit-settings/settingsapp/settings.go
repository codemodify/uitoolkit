// Package settingsapp is the toolkit's appearance editor: the theme
// browser, the live preview of the staged pack, the icon and corner
// options, and the Apply that writes look.json.
//
// It lives beside its command rather than inside it because the
// end-to-end driver and its own tests drive it as a library, and because
// it is a sample like the others: it is written against the published
// API — the style package for themes, the widgets package for the
// preview — and nothing under internal/.
package settingsapp

import (
	"fmt"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
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

// The page names in the sidebar. "Packs & icons" became "Packs" when
// choosing an icon set moved to Themes: what is left of that page is the
// packs on disk, of both kinds — theme packs and icon sets — and what it
// does with them is install, export and delete.
var settingsPages = []string{"Themes", "Appearance", "Packs", "About"}

// Theme browser filters: every built-in pack, one decade, or user packs.
var themeFilters = []string{"All decades", "1980s", "1990s", "2000s", "2010s", "2020s", "My themes"}

const filterUser = 6

// SettingsApp is the toolkit appearance editor. The Themes page is the
// toolkit's shop window: pick a pack and it is drawn at once in a live
// preview — a small application window whose frame, caption and every
// control come from that theme — while Settings itself keeps the applied
// look (a ThemeScope around the preview). Apply — the one button, at the
// right of the row under the pages — writes look.json and every app that
// watches it (and Settings) switches. Closing without Apply discards the
// staged change.
func SettingsApp(a *app.Application, win *app.Window) widget.Component {
	saved := style.LoadAppearance().Normalize()
	return buildSettings(a, win, saved, saved, pageThemes)
}

// SettingsAppStaged is SettingsApp with a theme already staged (not
// applied): screenshots and docs use it to show the preview in a theme
// other than the one Settings runs in.
func SettingsAppStaged(a *app.Application, win *app.Window, theme string) widget.Component {
	return SettingsAppOpen(a, win, theme, "")
}

// SettingsAppOpen is SettingsApp with a theme staged and a page open by
// name ("themes", "appearance", "packs", "about"); either may be empty.
// cmd/uitoolkit-settings' -stage and -page take this path.
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
//
// The names of the pages Settings used to have keep working, because
// -page is in scripts and in the docs of two releases. "packs & icons" is
// the old name of Packs and still opens it. "icons" opened that page when
// it was where an icon set was chosen; choosing one is on Themes now, so
// that is where the name goes — a script that says -page icons wants the
// icon settings, which is the thing that moved, not the folder of
// installed sets that stayed behind.
func SettingsPage(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "appearance":
		return pageAppearance
	case "packs", "packs & icons", "theme packs":
		return pagePacks
	case "about":
		return pageAbout
	case "themes", "icons", "icon sets":
		return pageThemes
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
	// browserRatio is where the user left the splitter of the Themes
	// page; it survives a rebuild.
	browserRatio float32
	// browserAuto is the ratio Settings worked out for itself, to tell it
	// apart from one the user dragged.
	browserAuto float32
	// scheme is the desktop's light / dark preference the page was built in.
	scheme style.ColorScheme

	// The live parts of the page the staged look feeds. Staging a change
	// repaints these where they stand instead of rebuilding the page: a
	// rebuild takes the caret out of the search field, the focus off the
	// theme list, and the gallery back to the top.
	rows       []themeRow
	list       *widgets.ListView
	bodySplit  *widgets.Splitter
	scopes     []*widgets.ThemeScope
	previewBox *widgets.Panel
	applyBtn   *widgets.Button
	// The icons head of the Themes page: the sets the chooser offers, the
	// chooser itself, the size its glyphs are drawn at and the panel the
	// strip lives in. They follow the staged appearance like the theme
	// preview does, whatever staged it.
	iconSets  []style.IconSetInfo
	iconPick  *widgets.ComboBox
	iconSize  *widgets.ComboBox
	iconBox   *widgets.Panel
	iconScope *widgets.ThemeScope
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
}

func (s *settingsState) rebuild() {
	s.capture()
	s.win.SetContent(buildSettingsState(s))
}

func buildSettings(a *app.Application, win *app.Window, saved, staged style.Appearance, page int) widget.Component {
	s := &settingsState{
		a: a, win: win, saved: saved.Normalize(), staged: staged.Normalize(), page: page,
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
	s.rows, s.list, s.bodySplit = nil, nil, nil
	s.scopes, s.previewBox = nil, nil
	s.iconSets, s.iconPick, s.iconSize, s.iconBox, s.iconScope = nil, nil, nil, nil, nil

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
	// Apply closes the row on the right, where a dialog's accept button
	// belongs; the spacer before it is the whole of the rest of the row.
	gap := widgets.NewSpacer()
	actions := widgets.NewRow(gap, s.applyBtn).WithGap(12).WithPad(8)
	actions.AddFlex(gap, 1)

	s.showStaged()

	root := widgets.NewColumn(split, actions)
	root.AddFlex(split, 1)
	return root
}

// ---- staging ------------------------------------------------------------------

// stage shows next without rebuilding the page: the preview switches
// theme where it stands, its caption follows, and Apply turns on.
func (s *settingsState) stage(next style.Appearance) {
	s.staged = next.Normalize()
	s.showStaged()
}

func (s *settingsState) apply() {
	next := s.staged.Normalize()
	if err := style.SaveAppearance(next); err != nil {
		s.fail("Apply failed", err)
		return
	}
	s.saved = next
	s.a.ApplyAppearance(next)
	s.rebuild()
}

// fail puts an error in front of the user. Writing look.json, exporting
// a pack and deleting one all touch the disk and all can fail; with no
// status bar and no line beside Apply, a modal is the window's only
// honest place to say so — a failure nobody is told about looks exactly
// like nothing having happened.
func (s *settingsState) fail(what string, err error) {
	if s.applyBtn == nil || err == nil {
		return
	}
	widgets.ShowMessageBox(s.applyBtn, widgets.MessageBoxOptions{
		Title:   what,
		Message: err.Error(),
		Kind:    widgets.MessageError,
		Buttons: widgets.ButtonsOK,
	})
}

// showStaged paints the staged appearance into the parts of the page that
// follow it. Each is nil on the pages that do not show it.
func (s *settingsState) showStaged() {
	look := s.staged.Look()
	for _, sc := range s.scopes {
		sc.SetTheme(look)
	}
	if s.previewBox != nil {
		s.previewBox.Title = "Preview — " + s.shownPack().Display()
		s.previewBox.RequestLayout()
	}
	s.showStagedIcons()
	if s.list != nil {
		s.list.Selected = s.selectedRow()
		s.list.Invalidate()
	}
	if s.applyBtn != nil {
		s.applyBtn.SetEnabled(s.staged != s.saved)
	}
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
// pack's name. Its family, engine and summary are not written anywhere —
// the preview beside the list is what says what a pack is — but the
// search field reads all of them.
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

func (s *settingsState) selectedRow() int {
	for i, r := range s.rows {
		if r.pack.Name == s.staged.Name {
			return i
		}
	}
	return -1
}

// themesPage is the shop window: the browser on the left and, on the
// right, the staged pack drawn as a small application window — its
// frame, caption and every control that theme's — in a splitter the user
// can size, so the list and the preview never fight for the space.
//
// The whole widget gallery used to sit under the preview. It was the
// same widgets the tour now shows as three pages of its own, it halved
// the preview, and what it added to a theme browser was a second answer
// to a question the preview had already answered. The right pane is the
// preview alone.
func (s *settingsState) themesPage() widget.Component {
	if s.browserRatio == 0 {
		s.browserRatio = defaultBrowserRatio(s.win)
		s.browserAuto = s.browserRatio
	}
	browser := s.themeBrowser()

	s.previewBox = widgets.NewPanel("", PreviewApp(nil))
	s.previewBox.Window = true
	preview := s.scoped(s.previewBox)

	s.bodySplit = widgets.NewSplitter(widgets.SplitColumns, browser, preview)
	s.bodySplit.Ratio = s.browserRatio
	s.bodySplit.SetAccessibleName("Themes and preview")

	page := widgets.NewColumn(s.iconsHead(), s.bodySplit).WithGap(10)
	page.AddFlex(s.bodySplit, 1)
	return page
}

// iconsHead is the icons half of the look, at the head of the page and
// built like the half under it: what to choose from on the left, in
// Settings' own theme, and on the right, filling the rest of the width,
// what the choice looks like, drawn in the staged one.
//
// Icons and themes are one question — what the toolkit looks like — asked
// twice, which is why they are one page. The page says so by asking both
// the same way: the chooser column here stands where the theme browser
// stands, under a heading of its own, and the strip beside it stands
// where the theme preview stands. Read down, it is choose-and-see, then
// choose-and-see; it is not an icons page with a themes page under it.
//
// Two other places were tried. Inside the browser column, above the theme
// list, is where the width is worst: the column is about 240 logical
// pixels, which folds fifteen glyphs into five lines and leaves a list of
// 129 packs the rest — the icons would have cost the page the thing it is
// for. Beside the preview, in the right pane, keeps the width but hides
// the icon strip behind the splitter: drag the sash left to read the
// theme list and the icons go with it, and the chooser would have sat in
// a pane of previews. Across the head of both columns is the only place
// where the icon strip gets the width fifteen glyphs want and neither
// half of the page is inside the other.
func (s *settingsState) iconsHead() widget.Component {
	s.iconSets = append(style.ListBuiltinIconSets(), style.ListUserIconSets()...)
	names := make([]string, len(s.iconSets))
	for i, set := range s.iconSets {
		names[i] = set.Label
	}
	s.iconPick = widgets.NewComboBox(names, indexIcon(s.iconSets, s.staged.Icons), func(i int) {
		if i < 0 || i >= len(s.iconSets) {
			return
		}
		next := s.staged
		next.Icons = s.iconSets[i].Name
		s.stage(next)
	})
	s.iconPick.SetAccessibleName("Icons")

	// The size lives beside the set because it is not a separate choice: a
	// set's glyphs are drawn at it, some sets are made for one end of the
	// range, and a strip that showed a fixed size would be showing
	// something the user is not going to get. The strip is in the staged
	// scope, so it redraws at the staged size as the size is changed.
	s.iconSize = widgets.NewComboBox([]string{"Small", "Medium", "Large"}, iconSizeIndex(s.staged.IconSize), func(i int) {
		next := s.staged
		next.IconSize = iconSizes[i]
		s.stage(next)
	})
	s.iconSize.SetAccessibleName("Icon size")

	// "Icons" names this band as "Themes" names the column under it. The
	// two choosers sit on the line after it and the strip spans the page
	// under them, rather than the choosers standing in a column beside the
	// strip: a wrap that folds has to be measured at the width it is going
	// to be arranged at, and a flexed child of a row is measured at the
	// row's whole width — the strip laid out beside the choosers reported
	// one line, was given the width for two, and had its second line cut
	// off. Full width is also the width fifteen glyphs want.
	rule := widgets.NewSpacer()
	chooser := widgets.NewRow(widgets.NewLabel("Icons"), s.iconPick, s.iconSize, rule).WithGap(8)
	chooser.AddFlex(rule, 1)

	// The strip is in a scope of its own — Settings' own theme carrying
	// the staged icon set at the staged size — rather than in the staged
	// theme like the preview under it. It is the set that is being
	// previewed here, not the pack, and a strip drawn in the pack would
	// change height with it: the packs put different padding round a tool
	// button, the head would be a different height in every one of the
	// 129, and the preview panel under it would sit somewhere different in
	// each. The Theme Atlas crops that panel out of one screenshot per
	// pack at one rectangle (docs/settings.md, "Screenshot geometry"), so
	// "the same place in every pack" is a promise this page has to keep.
	s.iconScope = widgets.NewThemeScope(s.iconLook(), s.iconStrip())
	return widgets.NewColumn(chooser, s.iconScope).WithGap(6)
}

// iconLook is what the strip is drawn in: the theme Settings itself is
// running, with the staged icon set at the staged size laid over it. The
// glyphs and their size are the staged ones — that is the whole point of
// the strip — while everything around them is the applied look, so the
// head of the page keeps one height whatever pack is staged.
func (s *settingsState) iconLook() style.LookAndFeel {
	return style.WithIconSize(style.WithIcons(s.a.Look(), s.staged.Icons), s.staged.IconSize)
}

// iconStrip is the whole of what a set draws: every icon the toolkit asks
// one for by name — the whole of [style.AllToolIcons] — in the groups
// they belong to: the three file actions, the three clipboard ones and
// the two histories on one bar, then find, edit, mail and download, and
// last the four message-box faces. Those four are the ones a set gives a
// colour of its own, so they are also the ones that say whether a set can
// be read against a dark pack. (Question was missing from this strip for
// as long as it had one: a test now walks AllToolIcons, so a stem added
// to the toolkit and not to the preview fails rather than goes unseen.)
//
// Each group is a tool bar, and the groups are in a [widgets.Wrap]: one
// bar of fifteen would clip its tail on a window as narrow as the 720
// Settings opens down to, and fifteen loose tool buttons would not answer
// the question the size chooser beside them asks. A loose tool button is
// a control-height key, so the look's icon size is capped by its height
// and a set at Large draws exactly as it does at Medium; a tool bar sizes
// its buttons from the bar metrics, which are the ones the icon size
// grows. The strip has to show the size that is staged, so the strip is
// made of bars.
func (s *settingsState) iconStrip() widget.Component {
	// Three bars, not five: the file, clipboard and history icons are one
	// bar with dividers between the groups, because every bar pays for its
	// own ends and five of them fold onto a second line in a window where
	// three sit on one.
	groups := [][]style.ToolIcon{
		{style.IconNew, style.IconOpen, style.IconSave, style.IconNone,
			style.IconCut, style.IconCopy, style.IconPaste, style.IconNone,
			style.IconUndo, style.IconRedo},
		{style.IconSearch, style.IconPen, style.IconMail, style.IconDownload},
		{style.IconInfo, style.IconWarning, style.IconError, style.IconQuestion},
	}
	row := widgets.NewWrap()
	for _, group := range groups {
		items := make([]*widgets.ToolItem, 0, len(group))
		for _, i := range group {
			if i == style.IconNone {
				items = append(items, widgets.ToolDivider())
				continue
			}
			// Icon only, known by the action it stands for: the strip is
			// there to be looked at, and the name is what a screen reader
			// and the tooltip both need.
			it := widgets.ToolIconBtn(i, "", nil)
			it.Tip = i.Label()
			items = append(items, it)
		}
		row.Add(widgets.NewToolBar(items...))
	}
	s.iconBox = widgets.NewPanel(iconBoxTitle(s.staged.Icons), row)
	s.iconBox.SetAccessibleName("Icon preview")
	return s.iconBox
}

// iconBoxTitle names the set the strip is drawing, in the words the theme
// preview's caption names its pack in: the page asks the same question
// twice and captions both answers the same way.
func iconBoxTitle(name style.IconSetName) string { return "Preview — " + iconSetLabel(name) }

// showStagedIcons puts the staged appearance back into the icons head.
// The strip follows the staged look through its scope, but the two
// choosers and the caption are Settings' own and have to be told — and
// they have to be told whoever staged it, not only the choosers
// themselves. Nothing else stages an icon set today: a theme pack's
// theme.json carries no icon preference (the field is read and ignored;
// look.json owns icons), so picking a pack leaves the set alone and only
// changes the surface the strip is drawn on. Deleting a user set does
// stage one, Apply and a desktop light / dark change restage the whole
// appearance, and a pack that took its own set would come through here
// too.
func (s *settingsState) showStagedIcons() {
	if s.iconScope != nil {
		s.iconScope.SetTheme(s.iconLook())
	}
	if s.iconBox != nil {
		s.iconBox.Title = iconBoxTitle(s.staged.Icons)
		s.iconBox.RequestLayout()
	}
	if s.iconPick != nil {
		if i := indexIcon(s.iconSets, s.staged.Icons); i != s.iconPick.Selected {
			s.iconPick.Selected = i
			s.iconPick.Invalidate()
		}
	}
	if s.iconSize != nil {
		if i := iconSizeIndex(s.staged.IconSize); i != s.iconSize.Selected {
			s.iconSize.Selected = i
			s.iconSize.Invalidate()
		}
	}
}

// iconSizes are the sizes the chooser offers, in its order.
var iconSizes = []style.IconSize{style.IconSizeSmall, style.IconSizeMedium, style.IconSizeLarge}

func iconSizeIndex(sz style.IconSize) int {
	for i, s := range iconSizes {
		if s == sz {
			return i
		}
	}
	return 1 // medium, the default
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

	// The list runs to the foot of the column: the count of packs that
	// used to close it, and the box that named the selected pack and its
	// year, family and engine, are gone. Both said in words what the
	// preview beside them says in the thing itself, and both took rows
	// off a list of 129 packs to do it.
	col := widgets.NewColumn(widgets.NewLabel("Themes"), search, filters, s.list).WithGap(6)
	col.AddFlex(s.list, 1)
	return col
}

// shownPack is the pack the preview actually draws: the staged one, or
// its light or dark sibling when Settings is following the desktop and
// the desktop asks for the other. The preview's caption names it, which
// is how a user sees that Breeze is showing as Breeze Dark.
func (s *settingsState) shownPack() style.ThemePack {
	pack, _ := style.LoadTheme(s.staged.Name)
	if !s.staged.FollowDesktop {
		return pack
	}
	if eff := s.staged.Effective(); eff.Name != s.staged.Name {
		if p, ok := style.LoadTheme(eff.Name); ok {
			return p
		}
	}
	return pack
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

	// The icon set and the size its glyphs are drawn at used to be here,
	// between the corners and the animations. They are on Themes now,
	// where they are next to the strip that shows what they do and next to
	// the theme they are being chosen for; a combo box called "Icons" with
	// a line of prose under it never showed anybody what Phosphor's
	// scissors look like. What is left here is the shape the packs are
	// drawn in and whether they move — the two things that are every
	// pack's, and neither of them a picture.
	shape := settingsSection("Shape and motion",
		optionRow("Corners", corners, "The pack's own shape, or rounded or square everywhere."),
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
	// The staged theme beside the options, so corners and motion are seen
	// changing without going back to the theme browser. It keeps to its
	// own height at the top of the page.
	below := widgets.NewSpacer()
	sample := widgets.NewColumn(s.scoped(appearanceSample()), below)
	sample.AddFlex(below, 1)
	body := widgets.NewRow(options, sample).WithGap(16)
	body.AddFlex(options, 1)
	return widgets.NewColumn(
		widgets.NewTitle("Appearance"),
		wrapped("What every uitoolkit app does with the theme, besides the theme and its icons themselves — those are on Themes. Apply writes these to "+style.AppearancePath()+"."),
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
// small enough to sit next to them, wide enough to show the corners
// changing — and it keeps its tool bar, because the icons it draws are
// the staged set at the staged size and that is worth seeing on a page
// that no longer chooses them.
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

// ---- Packs page -------------------------------------------------------------

// packsPage is what is on disk: the theme packs and the icon sets, and
// what can be done to the files — export one, delete one. Choosing a
// theme is the theme browser's and choosing an icon set is now the head
// of that same page, so the lists here are a shelf rather than a picker.
// They still select, because Delete has to be told which one, and because
// a list of what is installed that would not let you try one would be
// being awkward on purpose.
func (s *settingsState) packsPage() widget.Component {
	stage := s.stage
	fail := s.fail
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
				fail("Export failed", err)
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
					fail("Delete failed", err)
					return
				}
				next := style.AfterUserThemeDeleted(s.staged, name)
				if s.saved.Name == name {
					if err := style.SaveAppearance(next); err != nil {
						fail("Delete failed", err)
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
	iconCol := widgets.NewColumn(widgets.NewTitle("Icon sets"),
		wrapped("Copy a folder of PNGs into "+style.IconsDir()+" to add one. The set every app draws is chosen on Themes, beside the strip that shows it."),
		builtinIcons, userIcons).WithGap(8)
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
					fail("Delete failed", err)
					return
				}
				next := s.staged
				if next.Icons == name {
					next.Icons = style.IconSetClassic
				}
				if s.saved.Icons == name {
					if err := style.SaveAppearance(next); err != nil {
						fail("Delete failed", err)
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
	// The strip that showed what the selected set draws was here, under
	// these lists. It is at the head of Themes now, where the set is
	// chosen: a preview belongs beside the choice it informs, and this
	// page is about the folders, not the glyphs.
	body := widgets.NewRow(themeCol, iconCol).WithGap(20)
	body.AddFlex(themeCol, 1)
	body.AddFlex(iconCol, 1)
	return body
}

// iconSetLabel is what the lists call a set ("Material Symbols"), or its
// bare id if it is not listed.
func iconSetLabel(name style.IconSetName) string {
	for _, set := range append(style.ListBuiltinIconSets(), style.ListUserIconSets()...) {
		if set.Name == name {
			return set.Label
		}
	}
	return string(name)
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
