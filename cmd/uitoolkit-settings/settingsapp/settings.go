// Package settingsapp is the toolkit's appearance editor: the theme
// browser, the block of settings over the preview — four on/off options
// and the four choosers for the icon set, its size, the window's corners
// and the device that paints them — the live preview of the staged pack
// under them, and the Apply that writes look.json.
//
// It lives beside its command rather than inside it because the
// end-to-end driver and its own tests drive it as a library, and because
// it is a sample like the others: it is written against the published
// API — the style package for themes, the widgets package for the
// preview — and nothing under internal/.
package settingsapp

import (
	"fmt"
	"os"
	"strings"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The sections of the page. Settings had four pages behind a sidebar —
// Themes, Appearance, Packs, About — and they were four answers to one
// question: what does this desktop look like. They are one page now, and
// the line through that page is no longer between kinds of choice but
// between the browser and the thing it is browsing for: the list of 129
// packs is the column on the left, and everything else stands with the
// preview on the right — the settings in a folding block over it, and
// where it all lives on disk under it.
const (
	sectionTheme     = iota // the column on the left, and the whole of it
	sectionBehaviour        // the four on/off options, first in the block over the preview
	sectionPreview          // the four choosers after them: the icons, the corners and the renderer
	sectionFiles            // the right-hand pane, under the preview
)

// settingsSections names the sections in that order. -page takes these
// names, and the names of the four pages they came from.
var settingsSections = []string{"Theme", "Behaviour", "Preview", "Files"}

// Theme browser filters: every built-in pack, one decade, or user packs.
var themeFilters = []string{"All decades", "1980s", "1990s", "2000s", "2010s", "2020s", "My themes"}

const filterUser = 6

// SettingsApp is the toolkit appearance editor, and it is one page: the
// theme browser down a column on the left, and on the right,
// filling the rest of the window at every size, the thing it is
// browsing for — a small application window whose frame, caption and
// every control come from the staged pack, with the settings in a
// folding row over it and the files it all lives in under it. Settings
// itself keeps the applied look (the preview is a ThemeScope), so a pack
// can be judged without living in it.
//
// Apply — the one button, at the right of the row under the page —
// writes look.json and every app that watches it (and Settings) switches.
// Closing without Apply discards the staged change.
func SettingsApp(a *app.Application, win *app.Window) widget.Component {
	saved := style.LoadAppearance().Normalize()
	return buildSettings(a, win, saved, saved, false)
}

// SettingsAppStaged is SettingsApp with a theme already staged (not
// applied): screenshots and docs use it to show the preview in a theme
// other than the one Settings runs in.
func SettingsAppStaged(a *app.Application, win *app.Window, theme string) widget.Component {
	return SettingsAppOpen(a, win, theme, "")
}

// SettingsAppOpen is SettingsApp with a theme staged and the page opened
// at a section by name; either may be empty.
// cmd/uitoolkit-settings' -stage and -page take this path.
func SettingsAppOpen(a *app.Application, win *app.Window, theme, page string) widget.Component {
	return SettingsAppWith(a, win, SettingsOptions{Theme: theme, Page: page})
}

// SettingsOptions is how the command opens Settings.
type SettingsOptions struct {
	// Theme is a pack staged in the preview (not applied), by id; empty
	// stages the applied one. Page is the section that was asked for, by
	// any of the names [SettingsPage] takes. It is accepted and it does
	// nothing: there is one page, nothing on it is under a fold, and
	// every name resolves to something already on screen. See
	// [SettingsPage] for why it is still taken.
	Theme, Page string
	// PlainPreview makes the previewed window make-believe all through:
	// its File menu's Open… and Save… say their name on the sample's
	// status bar instead of opening a real file dialog.
	//
	// It is for pictures, not for people — the Theme Atlas renders 129
	// tiles out of this preview, and a tile is looked at rather than
	// clicked. It used to take the bar of live settings off the head of
	// the window as well; that bar is gone, and the icon set, the icon
	// size and the corner style are chosen on the page beside the
	// preview, outside anything the atlas crops. Settings itself never
	// sets it; cmd/uitoolkit-settings' -plain-preview does.
	PlainPreview bool
}

// SettingsAppWith is SettingsApp opened as opt says.
func SettingsAppWith(a *app.Application, win *app.Window, opt SettingsOptions) widget.Component {
	saved := style.LoadAppearance().Normalize()
	staged := saved
	if pack, ok := style.LoadTheme(opt.Theme); ok {
		staged.Name, staged.Theme = pack.Name, pack.Palette
	}
	return buildSettings(a, win, saved, staged, opt.PlainPreview)
}

// SettingsPage is the section a name means; an unknown or empty name is
// the theme browser, which is the column on the left.
//
// There is one page, nothing on it scrolls out of reach, and so -page
// switches nothing and scrolls nothing: every name it takes resolves to
// something that is already on screen. The flag and this function are
// kept anyway, because a flag that has stopped mattering must still not
// be an error: the names are in scripts, in the atlas tooling, in
// internal/apptest and in the docs of two releases. The names of the four pages
// Settings used to have keep working, because -page is in scripts, in
// the atlas tooling and in the docs of two releases — each one resolves
// to the section that swallowed it. "appearance", "corners" and "icons"
// are the Preview: the choosers that say what the previewed window is
// drawn with and what draws it, which stand with the options over it; "packs" (and its
// old name "packs & icons") is Theme, because
// exporting a pack and deleting one are under the theme browser;
// "about" is Files; "desktop" and "colours" are Behaviour, because
// following the desktop's colours is one of the four options that lead
// that row.
func SettingsPage(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "appearance", "shape", "shape and weight", "shape and motion", "corners", "icons", "icon sets", "preview":
		return sectionPreview
	case "behaviour", "behavior", "windows", "motion",
		"desktop", "the desktop", "colours", "colors", "where the colours come from":
		return sectionBehaviour
	case "about", "files", "paths":
		return sectionFiles
	case "packs", "packs & icons", "theme packs", "themes", "theme":
		return sectionTheme
	default:
		return sectionTheme
	}
}

// settingsState is the model behind one build of the Settings content.
type settingsState struct {
	a      *app.Application
	win    *app.Window
	saved  style.Appearance
	staged style.Appearance
	filter int
	// query narrows the theme browser to what it names: a pack's id,
	// name, year, family, engine or summary.
	query string
	// listOff keeps the theme browser's scroll position across rebuilds:
	// picking a theme below the fold used to jump the list to the top.
	listOff float32
	// browserRatio is where the user left the splitter between the
	// choices and the previews; it survives a rebuild.
	browserRatio float32
	// browserAuto is the ratio Settings worked out for itself, to tell it
	// apart from one the user dragged.
	browserAuto float32
	// scheme is the desktop's light / dark preference the page was built in.
	scheme style.ColorScheme

	// The live parts of the page the staged look feeds. Staging a change
	// repaints these where they stand instead of rebuilding the page: a
	// rebuild takes the caret out of the search field, the focus off the
	// theme list, and the column back to the top.
	rows       []themeRow
	list       *widgets.ListView
	bodySplit  *widgets.Splitter
	scopes     []*widgets.ThemeScope
	previewBox *widgets.Panel
	applyBtn   *widgets.Button
	// The three settings that are not an on and an off: the sets the
	// chooser offers, the chooser itself, the size its glyphs are drawn
	// at and the shape of the window's corners. They stand on the page
	// after the four options, each behind the word that says what it
	// sets. They follow the staged appearance whatever staged it — a
	// pack picked in the browser, -stage, Apply — and not only their own
	// clicks.
	iconSets []style.IconSetInfo
	iconPick *widgets.ComboBox
	iconSize *widgets.ComboBox
	corners  *widgets.ComboBox
	// render is the fourth chooser, and the odd one out: it says which
	// device paints, not what is painted, and what it asks for reaches
	// the windows opened after Apply rather than the ones already up.
	// See [settingsState.renderer].
	render *widgets.ComboBox
	// plain makes the previewed window make-believe all through. See
	// [SettingsOptions].
	plain bool
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

func buildSettings(a *app.Application, win *app.Window, saved, staged style.Appearance, plain bool) widget.Component {
	s := &settingsState{
		a: a, win: win, saved: saved.Normalize(), staged: staged.Normalize(), plain: plain,
	}
	// The preview draws the staged theme's light or dark sibling: redraw
	// it when the desktop switches.
	a.OnLookChange(func() {
		if now := style.DesktopColorScheme(); now != s.scheme {
			s.rebuild()
		}
	})
	// The column of choices keeps its share as the window is resized,
	// until the user drags the sash: then the split is theirs.
	win.OnResize(func(int, int) { s.followWindow() })
	return buildSettingsState(s)
}

func buildSettingsState(s *settingsState) widget.Component {
	s.scheme = style.DesktopColorScheme()
	// Every live part belongs to the build that made it.
	s.rows, s.list, s.bodySplit = nil, nil, nil
	s.scopes, s.previewBox = nil, nil
	s.iconSets, s.iconPick, s.iconSize, s.corners, s.render = nil, nil, nil, nil, nil

	if s.browserRatio == 0 {
		s.browserRatio = defaultChoicesRatio(s.win)
		s.browserAuto = s.browserRatio
	}
	s.bodySplit = widgets.NewSplitter(widgets.SplitColumns, s.choicesColumn(), s.previewColumn())
	s.bodySplit.Ratio = s.browserRatio
	s.bodySplit.SetAccessibleName("Settings and preview")

	s.applyBtn = widgets.NewButton("Apply", s.apply)
	s.applyBtn.Primary = true
	// Apply closes the row at the foot of the window, where a dialog's
	// accept button belongs; the spacer before it is the whole of the
	// rest of the row. It is outside the scrolling column on purpose:
	// the one thing that writes anything must not be able to scroll away.
	gap := widgets.NewSpacer()
	actions := widgets.NewRow(gap, s.applyBtn).WithGap(12).WithPad(8)
	actions.AddFlex(gap, 1)

	s.showStaged()

	body := widgets.NewPad(10, s.bodySplit)
	root := widgets.NewColumn(body, actions)
	root.AddFlex(body, 1)
	return root
}

// ---- staging ------------------------------------------------------------------

// stage shows next without rebuilding the page: the previews switch
// theme where they stand, their captions follow, and Apply turns on.
func (s *settingsState) stage(next style.Appearance) {
	s.staged = next.Normalize()
	s.showStaged()
}

// apply writes the staged appearance to look.json and then wears it.
//
// Settings puts its own window into the look it has just applied, rather
// than previewing something it is not: [app.Application.ApplyAppearance]
// switches every window of this application to next, which is the same
// call the look watcher makes in every *other* running app when it sees
// the file change. The page is then rebuilt, because the parts of it that
// are not a ThemeScope — the browser's rows, the options, Apply itself —
// were built in the old look's metrics.
//
// There is one mechanism, not two: the command opens with
// DisableLookWatch (see cmd/uitoolkit-settings/main.go), so the app that
// writes look.json is the one app that does not watch it. Watching it as
// well would mean this window took the new look twice — once here and
// again up to 300ms later, when the poll noticed its own write — and the
// second one would arrive after the user had already staged something
// else and quietly restyle the page under them.
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
// follow it.
func (s *settingsState) showStaged() {
	look := s.staged.Look()
	for _, sc := range s.scopes {
		sc.SetTheme(look)
	}
	if s.previewBox != nil {
		s.previewBox.Title = "Preview — " + s.shownPack().Display()
		s.previewBox.RequestLayout()
	}
	s.showStagedControls()
	if s.list != nil {
		s.list.Selected = s.selectedRow()
		s.list.Invalidate()
	}
	if s.applyBtn != nil {
		s.applyBtn.SetEnabled(s.staged != s.saved)
	}
}

// ---- the column: the theme browser --------------------------------------------

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

// choicesColumn is the left-hand column, and it is four bare controls:
// a search field, the decade filter, the list of packs that pass both,
// and Export. There is no caption over them and no group box around
// them, because there is nothing for one to tell them apart from: the
// column is one thing, the browser, and a legend reading Theme over the
// only list on the page said no more than the list says by being a list
// of themes. What a screen reader needs instead of that legend is on the
// two controls themselves — the list is called Themes and the field is
// called Search themes — and that is where it should have been all
// along, because a group box's legend names a group, not the list
// inside it.
//
// The heading that read Settings and the version under it went with
// them. A window says what application it is in its title bar, which is
// the desktop's to draw and says "uitoolkit - Settings" already; a
// heading repeating it inside the window was the last of the four-page
// sidebar, which needed something at the top of the column to own the
// pages under it. The version is not gone: it is what
// `uitoolkit-settings -version` prints.
//
// Everything else that was ever in here has gone to the pane on the
// right: the four on/off options and the four choosers for the shape,
// the icons and the renderer in one folding row over the preview, and
// the three paths under it.
func (s *settingsState) choicesColumn() widget.Component {
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

	// The list runs to the foot of the column: it takes every pixel the
	// search field, the filter and Export leave, at every window size,
	// and scrolls its 129 rows inside that on a scrollbar of its own.
	//
	// It was held to 252 pixels inside a column that scrolled, which was
	// the only way to have both a list with a scrollbar and a column
	// with one — a view as tall as all 129 of its rows would have made
	// the column ten screens long. That column had nothing else in it
	// but the browser, so what the arrangement really bought was two
	// scrollbars an inch apart and 250 pixels of nothing below the
	// buttons. The column does not scroll at all now. Nothing in it can
	// overflow: three controls of fixed height and a list that takes
	// what is left, which is the same promise the preview makes on the
	// other side of the sash.
	col := widgets.NewColumn(search, filters, s.list, s.exportButton()).WithGap(8).WithPad(4)
	col.AddFlex(s.list, 1)
	// The window opens on the list, not on the search field above it.
	// A window with no initial focus of its own starts on the first
	// control a click would focus, which is the field, and a focused
	// field shows its caret instead of its placeholder — so the top of a
	// column with no caption over it was an empty box, and the one word
	// on the page that says what the column is ("Search themes") was the
	// one word the page would not draw. Opening on the list shows it,
	// and it is the better place to land anyway: this column is a
	// browser, and the arrow keys walk it and stage what they reach.
	s.win.SetInitialFocus(s.list)
	return col
}

// exportButton writes the staged look out as a theme pack of the user's
// own. It is under the browser because the browser is what says which
// look is staged, and because the pack it writes appears in that same
// list a line later, as "User · <name>". It is the only thing under the
// list now: Delete theme… stood beside it for a release and is gone,
// the call made for Delete icon set… the release before.
func (s *settingsState) exportButton() widget.Component {
	var host widget.Component
	btn := widgets.NewButton("Export current theme…", func() {
		promptExportName(host, func(name string) {
			pack, err := style.ExportAppearance(strings.TrimSpace(name), s.staged)
			if err != nil {
				s.fail("Export failed", err)
				return
			}
			next := s.staged
			next.Name = pack.Name
			next.Theme = pack.Palette
			s.staged = next.Normalize()
			s.rebuild()
		})
	})
	host = btn
	return btn
}

// ---- the settings, over the preview --------------------------------------------

// option is one of the four on/off settings that lead the block over the
// preview: a check box with a short word on it, the full name a screen
// reader says, and the one sentence that says what turning it on does.
//
// The word on the box is short and the spoken name contains it — "OS
// window borders" is read out as "OS window borders: the desktop's title
// bar and borders" — which is the rule the choosers after them follow too:
// the visible word must be inside the spoken name, never beside it, or
// the control has two names. What the eye gets from the row these stand
// in, the ear gets from the rest of the name.
//
// The sentence is the tooltip and the accessible description, not a line
// under the box. It was a line under the box while these were in the
// column, where there was nothing under them but more of themselves;
// over the preview every line of prose is a line off the window the
// whole page is about.
func (s *settingsState) option(word, name, about string, on bool, set func(bool)) widget.Component {
	box := widgets.NewCheckbox(word, on, set)
	box.SetAccessibleName(name)
	box.SetAccessibleDescription(about)
	return widgets.NewTip(about, box)
}

// optionsRow is everything look.json carries that is not a pack: the four
// things that are not what the toolkit looks like but what it does —
// whether it moves, whose file dialogs it opens, who draws a window's
// frame, and where the colours come from — and then the four choosers:
// the shape of a window's corners, the icon set, the size its glyphs are
// drawn at, and the device that paints the lot.
//
// It is one block over the preview rather than a panel in the column
// beside it. Four of the options spent a release in that column, where
// 300 logical pixels elided every one of them ("Use the desktop's file
// dia…") and each carried a line of prose under it. A row of short words
// over a window is furniture you glance at; a stack of sentences is
// documentation, and documentation about four booleans is not worth a
// third of the column it was costing.
//
// The options are check boxes, not switches, for two reasons that agree.
// The honest one is that nothing on this page takes effect when it is
// touched — Apply writes look.json and nothing else does — and a switch
// is the control that says "this is live now", while a check box is the
// control that says "this is what I am asking for". The measured one is
// that a switch's pill is 42 logical pixels of chrome before its word,
// and four of those in the 453-pixel row of a 720x520 window take a
// third of the pane the preview is the point of.
//
// # One block, not two rows
//
// Three of the four choosers came out of the previewed window, where
// they stood on a bar of Settings' own over the sample's menu bar — "the
// preview configures itself". They are settings of the page like the
// four before them, so they are read where the page keeps its settings,
// and the window below is a sample again with nothing live in its
// chrome. The fourth, the renderer, was never anywhere but UITK_PAINT.
//
// They are in the *same* [widgets.Wrap] as the options rather than a
// second row under it, and that is a measurement, not a preference. The
// pane is 453 logical pixels at the 720x520 minimum and the settings
// come to about 990 of them: a row that folds packs them into three
// lines there, and two rows that each fold on their own take four —
// which is 78 pixels off a preview that had 273, and the preview stops
// being what the pane is for. One row that folds also *reads* as two
// rows wherever there is room for it to: at 1024x860 the four options
// fill the first line and the four choosers fall onto the second, which
// is the arrangement, arrived at by folding rather than by decree.
//
// What the fold must never do is break a chooser from the word in front
// of it, or the icon set from the size its glyphs are drawn at. So the
// choosers go in as two children, not six: the pair that says what is
// drawn, and the one that says what shape the window is — the two groups
// the divider on the old bar stood between.
func (s *settingsState) optionsRow() widget.Component {
	// Where the colours come from is last of the four, which puts it
	// nearest the window it changes: it is the one that decides which
	// pack is drawn under it, and the caption below says
	// "Preview — Breeze Dark" for a chosen Breeze, which is the rest of
	// this option's explanation.
	colours := s.option("OS colors",
		"OS colors: follow the desktop's light or dark mode and its accent",
		"Light or dark and the accent, as the desktop asks: a chosen Breeze shows as Breeze Dark.",
		s.staged.FollowDesktop, func(on bool) {
			next := s.staged
			next.FollowDesktop = on
			s.stage(next)
		})
	// Hover fades, the default button's pulse, busy bars: off for users
	// who get unwell from motion (GTK's gtk-enable-animations).
	motionAbout := "Hover fades, the default button's pulse and busy bars."
	if style.DesktopReducesMotion() {
		// The desktop's setting wins over the preference (style.Animations),
		// so a ticked box would be promising something it cannot give.
		// This used to be a line of prose under the box; it is what the
		// box says about itself now.
		motionAbout += " The desktop is asking for reduced motion, so they stay off whatever this says."
	}
	motion := s.option("Animations", "Animations", motionAbout,
		!s.staged.ReduceMotion, func(on bool) {
			next := s.staged
			next.ReduceMotion = !on
			s.stage(next)
		})
	// The desktop's own file dialogs (the XDG portal's), as Qt and GTK
	// apps can use, instead of the themed ones. "OS", not "System",
	// because the three of these that hand something to the desktop
	// should say so in the same word, and "OS colors" already did.
	//
	// The preview is where this one can be looked at rather than read:
	// the sample's File ▸ Open… and Save… open the dialog this box is
	// asking for, the desktop's or the toolkit's, and open nothing else.
	// See [PreviewOptions].
	native := s.option("OS open/save dialogs", "OS open/save dialogs: the desktop's own Open and Save",
		"KDE's and GNOME's own Open and Save dialogs, through the XDG portal, instead of the themed ones. The preview's File menu opens the one this asks for.",
		s.staged.NativeDialogs, func(on bool) {
			next := s.staged
			next.NativeDialogs = on
			s.stage(next)
		})
	// Chromium's switch: windows that draw their own title bar (Mail's,
	// with its tool bar in it) get the desktop's title bar and borders
	// instead, and their title bar becomes the first row.
	//
	// The box has two states, so it writes the two definite preferences —
	// "system" and "toolkit" — and never "auto". Auto is a third thing:
	// the toolkit frames a window that has a title bar of its own and
	// leaves every other window to the desktop, which is the right
	// default for a file nobody has edited but is not what unticking this
	// box says. Unticked used to write auto, and a window without a title
	// bar of its own — Settings' own, the sample's, most windows — then
	// kept the desktop's frame: the box said the toolkit would draw the
	// borders and nothing happened. Untouched, the staged preference
	// stays whatever look.json holds, so applying a theme never turns
	// client-side frames on behind the user's back.
	//
	// It carries the caption buttons with it, and that is the whole of
	// what is left of a fifth box called "Theme buttons". Where the
	// buttons of a title bar go was a question about a title bar that
	// only exists while this box is unticked: with the desktop drawing
	// the frame there is no toolkit caption to put buttons on, and with
	// the toolkit drawing it the theme is the only thing on this page
	// with an opinion about where they go — the Mac's traffic lights on
	// the left, GNOME's lone close, KDE's window menu. So the rule is
	// implicit: the toolkit draws the frame, the theme places the
	// buttons. style.CaptionButtonsPref and app.Application.SetCaptionButtons
	// stay, because an application may still want to choose; Settings is
	// what stopped asking. A look.json that already says
	// "captionButtons" keeps saying it — the staged appearance is
	// whatever was loaded until a box is touched, and this is the box
	// that touches it.
	system := s.option("OS window borders", "OS window borders: the desktop's title bar and borders",
		"The desktop draws the title bar and borders of every window, instead of the toolkit. Unticked, the toolkit draws them and the theme places the caption buttons.",
		s.staged.Decorations == style.DecorationsSystem, func(on bool) {
			next := s.staged
			next.Decorations, next.CaptionButtons = style.DecorationsToolkit, style.CaptionButtonsTheme
			if on {
				next.Decorations, next.CaptionButtons = style.DecorationsSystem, style.CaptionButtonsDesktop
			}
			s.stage(next)
		})

	shape, glyphs, paint := s.choosers()
	// The order is the order they are read: what the toolkit does on its
	// own account first, then the three that hand a piece of the window
	// to the desktop, in the order of how much they hand over — its
	// colours, then the dialogs it opens, then the frame around it —
	// and then what a pack is drawn with and what draws it.
	//
	// The fold follows from that order, and the order was settled by the
	// fold as much as by the reading. "OS borders" became "OS window
	// borders" and a fourth chooser joined the end: 63 px on one box and
	// 136 more at the end of the row. With the colours still last of the
	// four and the corners still last of the choosers, the 453-pixel row
	// of a 720x520 window folded onto four lines — and a fourth line is
	// 30 px off a preview that has 281, which takes it under the 60% of
	// its pane this page promises. Three narrow options lead now, so
	// that row's first line holds three of the four; the borders box
	// goes down onto the second with the corners, and the icon set, its
	// size and the renderer take the third. At 1024x860 the four options
	// still fill the first line and the three chooser groups fall onto
	// the second by themselves.
	row := widgets.NewWrap(motion, colours, native, system, shape, glyphs, paint)
	row.Gap = 8
	// Closer between the lines than along them: a block that folds has
	// to read as one block and not as rows of unrelated furniture.
	row.LineGap = 4
	// Named for the options that lead it; the four choosers after them
	// carry their own names, each with the word in front of it in the
	// tree as the static text it is.
	row.SetAccessibleName(settingsSections[sectionBehaviour])
	return row
}

// ---- what the packs are drawn with ---------------------------------------------

// choosers are the four settings that are not an on and an off: the icon
// set, the size its glyphs are drawn at, the shape of the window's
// corners, and which device paints them. Each stands behind the word
// that says what it sets.
//
// They come back as three children of the wrapping row, not eight,
// because a child of that row is what never folds down the middle of
// itself: the set and the size are one question about what is drawn,
// the shape of the window is another, the renderer a third, and a pair
// must not be split by a line break any more than a word may be split
// from its box. It is the same seam the divider marked when the first
// three rode on a bar inside the preview.
//
// They were a section called Shape and weight in the column on the left,
// with a strip of fifteen glyphs under them to show what a set draws.
// The strip was a picture of a tool bar; the preview has a real one,
// drawn in the staged set at the staged size, so the strip had nothing
// left to say and went. It has not come back: the preview's tool bar is
// still drawn in whatever the chooser says — the whole previewed window
// is, through its theme scope — so what shows a set is still a bar full
// of that set's icons, and it is a bar of the size the size chooser
// says, which a strip of loose glyphs never was.
func (s *settingsState) choosers() (shape, glyphs, paint widget.Component) {
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
	// It fits its longest set name and no more: four choosers and their
	// words share a line with four check boxes, and a combo box that
	// measured itself at the stock 160 would fold the block a line
	// further at every window size.
	s.iconPick.MinWidth = 1
	// The word on the page is "Icons"; what a screen reader says is the
	// same word, because a name that disagreed with the one on the
	// screen would be two names for one control.
	s.iconPick.SetAccessibleName("Icons")
	s.iconPick.Tip = "Icon set — a real setting. The preview's tools are drawn in it; the rest of that window is a sample."
	s.iconPick.SetAccessibleDescription(s.iconPick.Tip)

	// The size is beside the set because it is not a separate choice: a
	// set's glyphs are drawn at it, some sets are made for one end of the
	// range, and a page that showed a fixed size would be showing
	// something the user is not going to get. It lists the pixel sizes
	// rather than Small / Medium / Large because it is a size box beside
	// a set, where every application has written the number since the
	// first word processor.
	s.iconSize = widgets.NewComboBox(iconSizeNames(), iconSizeIndex(s.staged.IconSize), func(i int) {
		next := s.staged
		next.IconSize = iconSizes[i]
		s.stage(next)
	})
	s.iconSize.MinWidth = 1
	// "Size" on the page, "Icon size" to a screen reader: the visible
	// word is short because the chooser it names is the second half of a
	// pair, and the spoken name carries the half the eye gets from where
	// the box stands. The one contains the other, which is the rule —
	// the visible word must be in the spoken name, never beside it.
	s.iconSize.SetAccessibleName("Icon size")
	s.iconSize.Tip = "Icon size — a real setting: 16, 24 or 32 pixels, before the display scale."
	s.iconSize.SetAccessibleDescription(s.iconSize.Tip)

	// The shape of the window's corners. It was three radio items in the
	// preview's View ▸ Window corners, which is where an application has
	// always kept what its window looks like — and which nobody opened,
	// because a preview's menus are the one part of it a reader takes for
	// make-believe.
	s.corners = widgets.NewComboBox(cornerNames(), cornerIndex(s.staged.Corners), func(i int) {
		if i < 0 || i >= len(cornerStyles) {
			return
		}
		next := s.staged
		next.Corners = cornerStyles[i]
		s.stage(next)
	})
	s.corners.MinWidth = 1
	s.corners.SetAccessibleName("Window corners")
	s.corners.Tip = "Window corners — a real setting: the pack's own shape, round, or square. The previewed window is drawn with it."
	s.corners.SetAccessibleDescription(s.corners.Tip)

	// A word sits close to the box it names and further from the one
	// before it — six pixels one side, fourteen the other — so that the
	// four read as four settings rather than eight controls. The gap
	// between one group and the next is the row's own eight plus six
	// carried on the left-hand group's right edge, which makes it the
	// same fourteen; on the group's right rather than the next group's
	// left, because a line a group starts is a line that must start at
	// the margin.
	pair := func(word string, box *widgets.ComboBox) *widgets.FlexBox {
		return widgets.NewRow(widgets.NewLabel(word), box).WithGap(6).WithAlign(layout.AlignCenter)
	}
	shape = pair(settingWords[0], s.corners).WithPadding(0, 0, 6, 0)
	glyphs = widgets.NewRow(pair(settingWords[1], s.iconPick), pair(settingWords[2], s.iconSize)).
		WithGap(14).WithAlign(layout.AlignCenter).WithPadding(0, 0, 6, 0)
	paint = pair(settingWords[3], s.renderer())
	return shape, glyphs, paint
}

// settingWords are the words in front of the four choosers, in the order
// they stand on the page. They are short because they share a folding
// row with four check boxes in a pane as narrow as 453 logical pixels,
// and each one is the beginning of what its chooser is called to a
// screen reader ("Window corners", "Icons", "Icon size", "Paint
// renderer") rather than another name for it.
//
// Corners leads them, where it used to come last of three. The reading
// is outside in — the shape of the window, then what is drawn inside
// it, then what draws the lot — and, as with the four options before
// them, the fold is what settled it: the borders box and the corners
// share the second line of a 453-pixel row, and the icon set, its size
// and the renderer the third. With the corners last, as they were, the
// icons had a line to themselves and the row took four.
var settingWords = [4]string{"Corners", "Icons", "Size", "Paint"}

// rendererPrefs are the devices the paint chooser offers, in its order:
// let the toolkit decide, then the two definite answers.
var rendererPrefs = []style.RendererPref{style.RendererAuto, style.RendererGPU, style.RendererCPU}

// renderer is the chooser for the device that paints — the EGL/GLES one
// or the CPU rasterizer — and it is the odd one out on this row twice
// over.
//
// It is not appearance. Everything else here changes what a window looks
// like; this changes what draws it, and on a working GPU the two paths
// are meant to be the same picture. It is on this row because it is a
// setting of the toolkit the user owns, this is where the toolkit's
// settings are read, and the alternative was a second preferences page
// for one enum.
//
// And it is the one setting on the page that a window already open
// cannot take. Every other Apply reaches the windows that exist:
// [app.Application.ApplyAppearance] re-looks them where they stand. A
// surface binds its paint device when it is created — wl_egl_window and
// eglSwapBuffers for the GPU, wl_shm or XPutImage for the CPU — and
// swapping that under a mapped window means tearing an EGL context down
// and building another, with a frame of nothing in between. So this
// chooser is honest about its reach instead: what it says, and what it
// is read out as, is that the windows opened after Apply get it and the
// ones already up keep what they started with. A chooser that quietly
// did nothing until the next start is the shape of bug that cost this
// page two releases.
//
// It also says what is actually being painted with, which a preference
// on its own never does. "Auto" is a question, not an answer: a machine
// whose EGL will not initialise runs the whole toolkit on the CPU and
// the page would otherwise show a cheerful "Auto" over it. So the tip
// and the accessible description name the device this very window is
// presenting through ([app.Window.PaintBackend]), and say so when
// UITK_PAINT is what decided it.
func (s *settingsState) renderer() *widgets.ComboBox {
	names := make([]string, len(rendererPrefs))
	for i, p := range rendererPrefs {
		names[i] = p.Label()
	}
	// Auto's item was "Auto (CPU)" for an afternoon — the resolved device
	// on the face of the control, where nobody has to hover for it. It
	// does not fit: the suffix is 49 px, the combo measures itself on its
	// longest item, and the third line of the 453-pixel row at the
	// 720x520 minimum has 42 px spare. It folded onto a fourth line and
	// took the preview under 60% of its pane. So what is really painting
	// is in the tip and in the accessible description, which are the same
	// sentence and say it in full — see [settingsState.rendererTip].
	s.render = widgets.NewComboBox(names, rendererIndex(s.staged.Renderer), func(i int) {
		if i < 0 || i >= len(rendererPrefs) {
			return
		}
		next := s.staged
		next.Renderer = rendererPrefs[i]
		s.stage(next)
	})
	s.render.MinWidth = 1
	// "Paint" on the page, "Paint renderer" to a screen reader: the
	// visible word is inside the spoken name, as everywhere else on this
	// row. "Paint" rather than "Renderer" because it is the word the
	// environment variable and the docs have always used (UITK_PAINT),
	// and because five characters is what the row has room for.
	s.render.SetAccessibleName("Paint renderer")
	s.render.Tip = s.rendererTip()
	s.render.SetAccessibleDescription(s.render.Tip)
	return s.render
}

// rendererTip is what the paint chooser says about itself: what the
// choice means, what is really painting this window, what decided that,
// and how far an Apply reaches. It is rebuilt whenever the staged choice
// changes, because all four answers can move.
func (s *settingsState) rendererTip() string {
	var b strings.Builder
	b.WriteString("Paint renderer — a real setting: Auto takes the GPU (EGL/GLES) where it starts and the CPU rasterizer where it does not. ")
	// The window this page is drawn in, not the preference and not the
	// application's best window: the sentence says "this window", so it
	// has to ask this window's surface.
	got := "the CPU"
	if s.win.PaintBackend() == style.RendererGPU {
		got = "the GPU"
	}
	b.WriteString("This window is painting on " + got + ".")
	if env, ok := app.RendererOverride(); ok {
		b.WriteString(" " + app.RendererEnv + "=" + strings.ToLower(env.Label()) +
			" is set here and overrides the saved choice in this process; Apply still writes it for everything else.")
	}
	// The one thing on this page that Apply cannot give a window that is
	// already up. Said on the control rather than in a footnote: this is
	// where a person is when they need it.
	b.WriteString(" Apply writes it, and the windows opened after that are drawn with it — the ones already open, this one included, keep the device they were created with.")
	return b.String()
}

func rendererIndex(p style.RendererPref) int {
	for i, want := range rendererPrefs {
		if want == style.ParseRendererPref(string(p)) {
			return i
		}
	}
	return 0 // auto, the default
}

// ---- Files --------------------------------------------------------------------

// filesSection is the old About page: the three paths Settings reads and
// writes. It is under the preview because it is the only part of the
// page that changes nothing — it says where what the rest of the page
// changed ends up — and it is three lines rather than the six it was
// because it is under the preview: a line of prose and a three-row text
// box for each path cost the window they sit beneath a third of its
// height, and a path in a pane 700 pixels wide needs one row and says
// what it is by its own name.
//
// Three lines and no box. It was a group box with "Files" on its legend,
// and the legend and the frame cost 43 of the 101 pixels the block took
// — 43 pixels off the window the page is about, to put a word over three
// lines each of which is a name and a path and so says what it is by
// being one. It is the same call the Theme legend lost in the column on
// the left: a group box's legend names a group, and three paths under a
// preview are not a group anyone has to be told about. The pixels went
// to the preview, which is what paid for the choosers that came out of
// it.
//
// A name and the path, three times over, and nothing else: Delete icon
// set… rode at the end of the icons line for a release and is gone. It
// was the last thing left of the old Packs page, it acted on a set
// chosen two inches away, and it put a button that destroys a directory
// on the one part of the page that was meant to change nothing.
// style.DeleteUserIconSet is still there for an application that wants
// it.
func (s *settingsState) filesSection() widget.Component {
	line := func(name, path string) widget.Component {
		// A label, not a text box: a box that can be selected from keeps
		// three rows and grows a scrollbar of its own the moment the path
		// is longer than the pane, and there are three of them under a
		// window that wants every pixel. A label gives the pane back and
		// elides what will not fit.
		v := widgets.NewLabel(shortPath(path))
		v.Mono = true
		v.SetAccessibleName(name + ": " + path)
		row := widgets.NewRow(widgets.NewLabel(name), v).WithGap(10).WithAlign(layout.AlignCenter)
		row.AddFlex(v, 1)
		return row
	}
	// Three lines of one thing each: two pixels between them, because the
	// eight a column puts between its children are for paragraphs, not
	// for a table.
	return widgets.NewColumn(
		line("Prefs", style.AppearancePath()),
		line("Themes", style.ThemesDir()+"/<name>/theme.json"),
		line("Icons", style.IconsDir()+"/<set>/*.png"),
	).WithGap(2)
}

// shortPath is a path with the user's home written as ~, the way a shell
// writes it: these three lines are under the preview, and every
// character of /home/<someone> is a character of the part that matters
// elided away.
func shortPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || !strings.HasPrefix(p, home+"/") {
		return p
	}
	return "~" + strings.TrimPrefix(p, home)
}

// ---- the previews -------------------------------------------------------------

// previewColumn is the right-hand side and the whole reason the page is
// shaped this way: the staged pack drawn as a small but entirely live
// application, frame and caption and every control.
//
// Three things are in this pane, in the order they are read: the block
// of settings, over the window, because they are what it is drawing;
// the window; and where the files are, under it, because that is where
// what the window shows ends up. The settings are one row that folds and
// the paths are three lines, and neither scrolls: the window takes every
// pixel the two of them leave, at every window size, which is the
// promise this page has always made about its right-hand side. Nothing
// else is allowed in here — the strip of fifteen glyphs was tried above
// the window and folded onto four lines at the 720x520 minimum, leaving
// the preview a caption and a menu bar.
func (s *settingsState) previewColumn() widget.Component {
	options := s.optionsRow()
	s.previewBox = widgets.NewPanel("", PreviewAppWith(nil, s.previewOptions()))
	s.previewBox.Window = true
	preview := s.scoped(s.previewBox)
	col := widgets.NewColumn(options, preview, s.filesSection()).WithGap(8)
	col.AddFlex(preview, 1)
	return col
}

// previewOptions is what Settings lends the sample: which file dialog its
// File menu should open, read from the *staged* setting every time the
// menu is used rather than captured when the page was built, so that
// ticking the box and going straight to File ▸ Open… shows what was
// ticked. A plain preview lends it nothing and is make-believe all
// through.
func (s *settingsState) previewOptions() PreviewOptions {
	if s.plain {
		return PreviewOptions{}
	}
	return PreviewOptions{NativeDialogs: func() bool { return s.staged.NativeDialogs }}
}

// showStagedControls puts the staged appearance back into the three
// choosers. The previewed window follows the staged look through its
// scope, but what a chooser shows is Settings' own and has to be told —
// and told whoever staged it, not only the chooser itself. Picking a
// pack in the browser, -stage, Apply and a desktop light / dark change
// all come through here.
func (s *settingsState) showStagedControls() {
	if s.corners != nil {
		if i := cornerIndex(s.staged.Corners); i != s.corners.Selected {
			s.corners.Selected = i
			s.corners.Invalidate()
		}
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
	if s.render != nil {
		if i := rendererIndex(s.staged.Renderer); i != s.render.Selected {
			s.render.Selected = i
			s.render.Invalidate()
		}
		// What the chooser says about itself moves with the choice: what
		// Auto resolves to on this machine is one sentence, what a
		// definite GPU or CPU asks for is another.
		if tip := s.rendererTip(); tip != s.render.Tip {
			s.render.Tip = tip
			s.render.SetAccessibleDescription(tip)
		}
	}
}

// iconSizes are the sizes the chooser offers, in its order.
var iconSizes = []style.IconSize{style.IconSizeSmall, style.IconSizeMedium, style.IconSizeLarge}

// iconSizeNames are what the size chooser lists: the pixel sizes the
// three named sizes are.
func iconSizeNames() []string {
	out := make([]string, len(iconSizes))
	for i, sz := range iconSizes {
		out[i] = fmt.Sprint(style.IconSizePixels(sz))
	}
	return out
}

// cornerStyles are the shapes the corner chooser offers, in its order:
// the pack's own, then round, then square.
var cornerStyles = []style.CornerStyle{style.CornersTheme, style.CornersRound, style.CornersSquare}

// cornerNames are what that chooser lists. "Theme shape" is the pack's
// own corners, whatever they are — the words the View menu used, kept
// because they are the ones the docs and two releases of screenshots
// say.
func cornerNames() []string { return []string{"Theme shape", "Round", "Square"} }

func cornerIndex(c style.CornerStyle) int {
	for i, want := range cornerStyles {
		if want == c {
			return i
		}
	}
	return 0 // the pack's own shape, the default
}

func iconSizeIndex(sz style.IconSize) int {
	for i, s := range iconSizes {
		if s == sz {
			return i
		}
	}
	return 1 // medium, the default
}

// followWindow works the choices column's share out again for the
// window's new size, unless the user has dragged the sash since Settings
// last set it.
func (s *settingsState) followWindow() {
	if s.bodySplit == nil || s.bodySplit.Ratio != s.browserAuto {
		return
	}
	s.browserAuto = defaultChoicesRatio(s.win)
	s.bodySplit.Ratio = s.browserAuto
	s.bodySplit.RequestLayout()
}

// defaultChoicesRatio aims the column at about 300 logical pixels
// whatever the window and the display scale are: a narrow window gives
// it a bigger share so its rows stay readable, a wide one hands the room
// to the preview. It was 240 when the column was the theme browser and
// nothing else the first time, and went to 300 for the option rows that
// have since left. It stays at 300: what sets it now is a row reading
// "1995  ·  Windows 95" and the word on the Export button, and 240
// elides the one and wraps the other.
// Dragging the sash replaces it.
func defaultChoicesRatio(win *app.Window) float32 {
	const want, narrow, pad = 300, 800, 30
	// Window.Size is already logical pixels, which is what `want` is in.
	lw, _ := win.Size()
	page := float32(lw) - pad
	if page <= 0 {
		return 0.3
	}
	// A browser is a list and three controls; a preview is a window with
	// a row of settings over it. Below 800 the column asks for less, or
	// it takes two fifths of the window to show fourteen packs while the
	// options fold onto a third line and the preview stops being the
	// biggest thing on the page. 240 still shows a decade filter and a
	// readable pack name.
	w := float32(want)
	if lw < narrow {
		w = 240
	}
	return min(max(w/page, 0.18), 0.5)
}

// scoped draws c in the staged theme while Settings keeps the applied one.
func (s *settingsState) scoped(c widget.Component) *widgets.ThemeScope {
	scope := widgets.NewThemeScope(s.staged.Look(), c)
	s.scopes = append(s.scopes, scope)
	return scope
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

// PreviewOptions is what the application showing the preview lends it.
// Everything in the previewed window is make-believe — Send sends
// nothing, the tree lists a mailbox nobody has, the check boxes tick
// themselves — with one exception, and this is how the exception is
// handed in. Left empty, the preview is the sample it has always been.
type PreviewOptions struct {
	// NativeDialogs, when it is not nil, makes the sample's File ▸ Open…
	// and Save… (and the Open and Save on its tool bar) open a real file
	// dialog, and says which one: the desktop's own, through the XDG
	// portal, when it returns true, and the toolkit's themed one when it
	// returns false. That is the "OS open/save dialogs" option made
	// visible — the one of Settings' options whose effect is a window
	// nobody can picture from four words — and it is a preview in the
	// strict sense: the dialog reads a directory and nothing else, and
	// the path it comes back with is said on the sample's status bar and
	// dropped.
	//
	// It is a function, not a bool, because the answer has to be the
	// *staged* setting at the moment the menu is used. Settings has not
	// applied anything yet, so the process-wide style.NativeDialogs()
	// still holds what look.json said when it started; a preview that
	// asked that would be previewing the applied setting, which is the
	// one thing it must not do.
	NativeDialogs func() bool
}

// showPreviewDialog opens the file dialog native asks for.
//
// It is a variable so that a test can see which dialog was asked for. It
// cannot see it any other way: there is no portal on a test's private
// bus, so the desktop's dialog would silently fall back to the toolkit's
// and both branches would end in the same window.
var showPreviewDialog = func(from widget.Component, native bool, opts widgets.FileDialogOptions) {
	if native {
		// ShowFileDialog asks the portal and falls back to the toolkit's
		// own dialog when the desktop has none, which is exactly what
		// the setting promises.
		opts.Native = true
		widgets.ShowFileDialog(from, opts)
		return
	}
	// The staged setting says the themed dialog — and ShowFileDialog
	// would hand this one to the portal all the same, because it ORs
	// style.NativeDialogs(), which is process-wide and still says what
	// was applied. Untick the box on a desktop whose dialogs are applied
	// and the preview would go on opening KDE's. Building the dialog is
	// the published way to say "this one, whatever the process thinks".
	widgets.NewFileDialog(opts).Show(from)
}

// PreviewApp is a small, fully interactive application used to preview a
// theme: menu bar, tool bar, tabs with every kind of control, lists, a
// tree, a table and a status bar. Nothing in it does anything outside
// itself. Settings shows it inside a ThemeScope.
func PreviewApp(say func(string)) widget.Component {
	return PreviewAppWith(say, PreviewOptions{})
}

// PreviewAppWith is [PreviewApp] with the one thing in it the showing
// application lends it: the file dialog its File menu opens. See
// [PreviewOptions].
func PreviewAppWith(say func(string), opt PreviewOptions) widget.Component {
	// The preview's own status bar is where what it says goes when the
	// caller wants it nowhere else: Settings has no status bar of its own.
	sb := widgets.NewStatusBar("Ready", "Ln 1, Col 1", "100%")
	if say == nil {
		say = func(msg string) { sb.Set(0, msg) }
	}
	// File ▸ Open… and Save… are the one thing in this window that is not
	// make-believe: they open the file dialog the staged "OS open/save
	// dialogs" setting asks for, so that the option can be looked at
	// instead of read. host is the window they open over — it is inside
	// the preview's theme scope, so the toolkit's own dialog comes up in
	// the pack being staged, which is the other half of what there is to
	// see. Nothing is read and nothing is written: the path that comes
	// back is said on the status bar and dropped.
	var host widget.Component
	files := func(mode widgets.FileDialogMode) {
		what := "Open"
		if mode == widgets.FileSave {
			what = "Save"
		}
		if opt.NativeDialogs == nil || host == nil {
			// A plain preview, or [PreviewApp]: the sample says what was
			// asked of it, the way Send and About do.
			say(what)
			return
		}
		start, err := os.UserHomeDir()
		if err != nil || start == "" {
			start = "."
		}
		showPreviewDialog(host, opt.NativeDialogs(), widgets.FileDialogOptions{
			// The title says so in both dialogs, the desktop's included:
			// a file chooser that has opened over an editor of themes is
			// the one place a person would reasonably expect a file to be
			// opened.
			Title: what + " — preview only",
			Path:  start,
			Mode:  mode,
			OnPick: func(p string) {
				say(what + " " + shortPath(p) + " — a preview: nothing was read or written")
			},
			OnCancel: func() { say(what + " cancelled") },
		})
	}
	menu := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemIconAccel(style.IconNew, "&New", "Ctrl+N", func() { say("New") }),
			widgets.ItemIconAccel(style.IconOpen, "&Open…", "Ctrl+O", func() { files(widgets.FileOpen) }),
			widgets.ItemIconAccel(style.IconSave, "&Save", "Ctrl+S", func() { files(widgets.FileSave) }),
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
	// The sample's own bar, and nothing but: eight commands in three
	// groups, the way a text editor's is. It is the bar it was before the
	// choosers rode at the end of it and cost it Paste, and nothing has
	// been added to fill the room they left — a tool bar is not a shelf,
	// and eight commands in three groups is what this sample does. The
	// bar is drawn in the staged set at the staged size, which is what
	// makes it the preview of both.
	//
	// Open and Save are the menu's Open and Save: a tool button and the
	// item it doubles are one command, so they open the same dialog.
	tools := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconNew, "", func() { say("New") }),
		widgets.ToolIconBtn(style.IconOpen, "", func() { files(widgets.FileOpen) }),
		widgets.ToolIconBtn(style.IconSave, "", func() { files(widgets.FileSave) }),
		widgets.ToolDivider(),
		widgets.ToolIconBtn(style.IconCut, "", nil),
		widgets.ToolIconBtn(style.IconCopy, "", nil),
		widgets.ToolIconBtn(style.IconPaste, "", nil),
		widgets.ToolDivider(),
		bold,
		widgets.ToolIconBtn(style.IconMail, "Send", func() { say("Send") }),
		// The free space at the end is the window's right edge: a bar
		// that knows where that is sheds its last tools when the preview
		// is squeezed to the 720-pixel minimum, instead of showing half
		// a button cut off by the frame.
		widgets.ToolStretch(),
	)
	tools.SetAccessibleName("Tools")

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

	// The sample's own furniture is flush, the way a window's is: a menu
	// bar, the tool bar under it, the document, the status bar at the
	// foot, with no air between them. (There were eight pixels between
	// each for a long time. No window has those.)
	sample := widgets.NewColumn(menu, tools, tabs, sb).WithGap(0)
	sample.AddFlex(tabs, 1)
	// The window a dialog opens over, now that there is a window: the
	// column is inside the preview's theme scope, so the dialog wears the
	// staged pack.
	host = sample
	return sample
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

// ---- helpers ------------------------------------------------------------------

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
