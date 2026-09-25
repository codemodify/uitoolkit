// Package settingsapp is the toolkit's appearance editor: the theme
// browser, the live preview of the staged pack, the icon and corner
// options it carries on a bar of its own, the five on/off options in a
// row over it, and the Apply that writes look.json.
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

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
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
// preview on the right — the five options in a row over it, the icons
// and the corners on its own bar, and where it all lives on disk under
// it.
const (
	sectionTheme     = iota // the column on the left, and the whole of it
	sectionBehaviour        // the row of options over the preview
	sectionPreview          // the preview itself: its icons and its corners
	sectionFiles            // the right-hand pane, under the preview
)

// settingsSections names the sections in that order. -page takes these
// names, and the names of the four pages they came from.
var settingsSections = []string{"Theme", "Behaviour", "Preview", "Files"}

// inColumn reports whether a section is in the scrolling column: the
// others are always on screen, so there is nothing to scroll to. Only
// the theme browser is in it now that the options stand over the
// preview, and the browser is the top of it, so nothing -page names is
// under the fold — the column opens where it opens.
func inColumn(section int) bool { return section == sectionTheme }

// Theme browser filters: every built-in pack, one decade, or user packs.
var themeFilters = []string{"All decades", "1980s", "1990s", "2000s", "2010s", "2020s", "My themes"}

const filterUser = 6

// SettingsApp is the toolkit appearance editor, and it is one page: the
// theme browser down a scrolling column on the left, and on the right,
// filling the rest of the window at every size, the thing it is
// browsing for — a small application window whose frame, caption and
// every control come from the staged pack, with the five on/off options
// in a row over it and the files it all lives in under it. Settings
// itself keeps the applied look (the preview is a ThemeScope), so a pack
// can be judged without living in it.
//
// Apply — the one button, at the right of the row under the page —
// writes look.json and every app that watches it (and Settings) switches.
// Closing without Apply discards the staged change.
func SettingsApp(a *app.Application, win *app.Window) widget.Component {
	saved := style.LoadAppearance().Normalize()
	return buildSettings(a, win, saved, saved, sectionTheme, false)
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
	// stages the applied one. Page is the section the page opens at, by
	// any of the names [SettingsPage] takes; empty is the top.
	Theme, Page string
	// PlainPreview leaves the bar of live settings off the preview, so
	// that the window in the right-hand pane is the sample application
	// and nothing else: no icon set, no icon size and no corner style
	// can be chosen while it is set.
	//
	// It is for pictures, not for people — the Theme Atlas renders 129
	// tiles out of this preview, and a tile is read as a picture of a
	// pack rather than used as a control panel. Settings itself never
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
	return buildSettings(a, win, saved, staged, SettingsPage(opt.Page), opt.PlainPreview)
}

// SettingsPage is the section a name opens the page at; an unknown or
// empty name is the top of it, which is the theme browser.
//
// There is one page now, so -page no longer switches anything: it says
// which part of the page to start at, and Settings scrolls its column
// there when the section named is in it. The names of the four pages
// Settings used to have keep working, because -page is in scripts, in
// the atlas tooling and in the docs of two releases — each one resolves
// to the section that swallowed it. "appearance", "corners" and "icons"
// are the Preview, which is where the shape and the icon set are chosen
// now; "packs" (and its old name "packs & icons") is Theme, because
// exporting a pack and deleting one are under the theme browser;
// "about" is Files; "desktop" and "colours" are Behaviour, because
// following the desktop's colours is one of the five options in that
// row. Three of the four sections are beside the preview rather than in
// the column, and a name that resolves to one of those leaves the
// column where it is: what it names is already on screen.
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
	// section is where the page opens; it is used once, on the first
	// build, and a rebuild keeps the scroll offset instead.
	section  int
	revealed bool
	filter   int
	// query narrows the theme browser to what it names: a pack's id,
	// name, year, family, engine or summary.
	query string
	// listOff keeps the theme browser's scroll position across rebuilds:
	// picking a theme below the fold used to jump the list to the top.
	listOff float32
	// choicesOff is where the user left the column of choices; Apply
	// rebuilds the page and must not throw them back to the top of it.
	choicesOff float32
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
	choices    *widgets.ScrollView
	sections   []widget.Component
	scopes     []*widgets.ThemeScope
	previewBox *widgets.Panel
	applyBtn   *widgets.Button
	// What Delete acts on follows the staged choice, so the button is
	// always on the page and goes grey when the staged pack is not one
	// of the user's own: a button that came and went would move
	// everything under it every time a pack was picked.
	delTheme *widgets.Button
	// The controls the preview carries instead of showing: the sets the
	// chooser offers, the chooser itself, the size its glyphs are drawn
	// at and the shape of the window's corners. All three stand on a bar
	// of Settings' own at the head of the previewed window, each behind
	// the word that says what it sets. They follow the staged appearance
	// whatever staged it — a pack picked in the browser, -stage, Apply —
	// and not only their own clicks.
	iconSets []style.IconSetInfo
	iconPick *widgets.ComboBox
	iconSize *widgets.ComboBox
	corners  *widgets.ComboBox
	// plain drops that bar: the preview is then the sample application
	// and nothing else. See [SettingsOptions].
	plain bool
}

// capture remembers where the user left the parts of the page they can
// move, so the next build puts them back.
func (s *settingsState) capture() {
	if s.list != nil {
		s.listOff = s.list.OffsetY
	}
	if s.choices != nil {
		s.choicesOff = s.choices.OffsetY
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

func buildSettings(a *app.Application, win *app.Window, saved, staged style.Appearance, section int, plain bool) widget.Component {
	s := &settingsState{
		a: a, win: win, saved: saved.Normalize(), staged: staged.Normalize(), section: section, plain: plain,
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
	if s.section < 0 || s.section >= len(settingsSections) {
		s.section = sectionTheme
	}
	// Every live part belongs to the build that made it.
	s.rows, s.list, s.bodySplit, s.choices = nil, nil, nil, nil
	s.sections = make([]widget.Component, len(settingsSections))
	s.scopes, s.previewBox = nil, nil
	s.delTheme = nil
	s.iconSets, s.iconPick, s.iconSize, s.corners = nil, nil, nil, nil

	if s.browserRatio == 0 {
		s.browserRatio = defaultChoicesRatio(s.win)
		s.browserAuto = s.browserRatio
	}
	s.choices = widgets.NewScrollView(s.choicesColumn())
	s.choices.SetAccessibleName("Settings")
	s.choices.OffsetY = s.choicesOff

	s.bodySplit = widgets.NewSplitter(widgets.SplitColumns, s.openAt(), s.previewColumn())
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

// openAt is the column of choices, wrapped — on the first build only —
// in the thing that scrolls it to the section -page named. Where that
// section starts is not known until the column has been measured at the
// width it got, so it cannot be worked out here; the wrapper does it at
// the end of the first layout and then gets out of the way.
func (s *settingsState) openAt() widget.Component {
	was := s.revealed
	s.revealed = true
	if was || s.section == sectionTheme || !inColumn(s.section) || s.sections[s.section] == nil {
		// The top of the column, or a section that is not in it: what
		// -page named is beside the preview and already on screen.
		return s.choices
	}
	return newScrollTo(s.choices, s.sections[s.section])
}

// ---- staging ------------------------------------------------------------------

// stage shows next without rebuilding the page: the previews switch
// theme where they stand, their captions follow, and Apply turns on.
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
	// Delete says what it would delete, and is only alive while that is
	// something of the user's own.
	if s.delTheme != nil {
		s.delTheme.SetEnabled(indexTheme(style.ListUserThemes(), s.staged.Name) >= 0)
	}
	if s.applyBtn != nil {
		s.applyBtn.SetEnabled(s.staged != s.saved)
	}
}

// ---- the column of choices ------------------------------------------------

// choicesColumn is the left-hand column, and it is the theme browser
// alone: the one decision on this page that is not about the window on
// the right but about which of 129 packs that window is to be. The shape
// and the icons went to the preview's own chrome, where what shows a
// setting is what sets it; the five on/off options and the three paths
// went to the pane beside this one, over and under the preview. What is
// left still scrolls, because a list of 129 packs does not fit in a
// 520-pixel window and something has to give — and what must not give is
// the preview beside it, which is why nothing else is in here.
func (s *settingsState) choicesColumn() widget.Component {
	s.sections[sectionTheme] = s.themeSection()
	col := widgets.NewColumn(
		widgets.NewTitle("Settings"),
		widgets.NewLabel("v"+uitoolkit.Version),
		s.sections[sectionTheme],
	).WithGap(12).WithPad(4)
	return col
}

// ---- Theme ------------------------------------------------------------------

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

// themeSection is the head of the column and the first decision: a
// search field, the decade filter, the packs that pass both, and the two
// things that can be done to a pack of the user's own — write one out,
// and remove one.
//
// Export and Delete were a page of their own (Packs) with a second copy
// of the browser on it to say which pack they meant. The browser here is
// the only one there is now, so they sit under it and act on what it has
// staged: Export writes the staged look out as a pack, Delete removes the
// staged pack when it is one of the user's. Nothing has to be selected
// twice.
func (s *settingsState) themeSection() widget.Component {
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

	// The list is held to a height rather than run to the foot of the
	// window: it is in a column that scrolls now, and a view as tall as
	// all 129 of its rows would have made the column ten screens long and
	// given the list no scrollbar of its own. Nine rows is enough to
	// browse in and leaves the four sections under it within reach.
	return widgets.NewPanel("Theme",
		wrapped("The pack every uitoolkit app is drawn from. Picking one stages it in the preview; nothing is written until Apply."),
		search, filters,
		widgets.NewHeightBox(252, s.list),
		s.exportButton(), s.deleteThemeButton(),
	)
}

// exportButton writes the staged look out as a theme pack of the user's
// own. It is under the browser because the browser is what says which
// look is staged, and because the pack it writes appears in that same
// list a line later, as "User · <name>".
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

// deleteThemeButton removes the staged pack from disk when it is one the
// user exported. It lives under the browser — the one place that lists
// the user's own packs — and it is grey rather than absent while a
// built-in pack is staged, so that picking a pack never shuffles the
// rest of the column.
func (s *settingsState) deleteThemeButton() widget.Component {
	var host widget.Component
	s.delTheme = widgets.NewButton("Delete theme…", func() {
		name := s.staged.Name
		if indexTheme(style.ListUserThemes(), name) < 0 {
			return
		}
		widgets.Confirm(host, "Delete theme?", "Remove "+name+" from disk? This cannot be undone.", func(yes bool) {
			if !yes {
				return
			}
			if err := style.DeleteUserTheme(name); err != nil {
				s.fail("Delete failed", err)
				return
			}
			next := style.AfterUserThemeDeleted(s.staged, name)
			if s.saved.Name == name {
				if err := style.SaveAppearance(next); err != nil {
					s.fail("Delete failed", err)
					return
				}
				s.saved = next
			}
			s.a.ApplyAppearance(s.saved)
			s.staged = next.Normalize()
			s.rebuild()
		})
	})
	host = s.delTheme
	s.delTheme.SetEnabled(false)
	return s.delTheme
}

// ---- the options, over the preview --------------------------------------------

// option is one of the five on/off settings in the row over the preview:
// a check box with a short word on it, the full name a screen reader
// says, and the one sentence that says what turning it on does.
//
// The word on the box is short and the spoken name contains it — "File
// dialogs" is read out as "Use the desktop's file dialogs" — which is
// the rule the preview's own settings bar follows two inches below:
// the visible word must be inside the spoken name, never beside it, or
// the control has two names. What the eye gets from the row these five
// stand in, the ear gets from the rest of the name.
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

// optionsRow is the five things look.json carries that are neither a
// pack nor the shape and the icons the preview sets for itself: where
// the colours come from, and the four that are not what the toolkit
// looks like but what it does — whether it moves, whose file dialogs it
// opens, who draws a window's frame, and where that frame's buttons go.
//
// They are one row over the preview rather than a panel in the column
// beside it. Four of them spent a release in that column, where 300
// logical pixels elided every one of them ("Use the desktop's file
// dia…") and each carried a line of prose under it, and the fifth stood
// here alone. A row of five words over a window is furniture you glance
// at; a stack of five sentences is documentation, and documentation
// about five booleans is not worth a third of the column it was costing.
//
// They are check boxes, not switches, for two reasons that agree. The
// honest one is that nothing on this page takes effect when it is
// touched — Apply writes look.json and nothing else does — and a switch
// is the control that says "this is live now", while a check box is the
// control that says "this is what I am asking for". The measured one is
// that a switch's pill is 42 logical pixels of chrome before its word,
// and five of those in the 392-pixel pane of a 720x520 window fold onto
// three lines and take a fifth of the preview's height with them; five
// check boxes fold onto two, and onto one at 1024. The preview is what
// this pane is for.
//
// [widgets.Wrap] is what folds them: one line while the pane is wide,
// two when it is not, and never a control cut off by the window frame
// the way a tool bar's shedding would leave one.
func (s *settingsState) optionsRow() widget.Component {
	// Where the colours come from is last, which puts it nearest the
	// window it changes: it is the one of the five that decides which
	// pack is drawn under it, and the caption right below says
	// "Preview — Breeze Dark" for a chosen Breeze, which is the rest of
	// this option's explanation.
	colours := s.option("Desktop colours",
		"Desktop colours: follow the desktop's light or dark mode and its accent",
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
	// apps can use, instead of the themed ones.
	native := s.option("File dialogs", "Use the desktop's file dialogs",
		"KDE's and GNOME's own Open and Save dialogs, through the XDG portal, instead of the themed ones.",
		s.staged.NativeDialogs, func(on bool) {
			next := s.staged
			next.NativeDialogs = on
			s.stage(next)
		})
	// Chromium's switch: windows that draw their own title bar (Mail's,
	// with its tool bar in it) get the desktop's title bar and borders
	// instead, and their title bar becomes the first row.
	system := s.option("System frame", "System frame: the desktop's title bar and borders",
		"The desktop draws the title bar and borders of every window, instead of the toolkit.",
		s.staged.Decorations == style.DecorationsSystem, func(on bool) {
			next := s.staged
			next.Decorations = style.DecorationsAuto
			if on {
				next.Decorations = style.DecorationsSystem
			}
			s.stage(next)
		})
	// Where a title bar the toolkit draws puts its caption buttons: the
	// desktop's layout, or the theme's own (the Mac's traffic lights on
	// the left). This is the one of the five that is not really an on and
	// an off but a choice between two layouts, and the only one whose
	// short word had to be found rather than cut out of the long one:
	// "Place window buttons as the theme does" shortens to nothing that
	// is inside it and still says which of the two it means.
	themeButtons := s.option("Theme buttons", "Theme buttons: the caption buttons where the theme puts them",
		"Close, minimise and maximise where the theme's era put them, instead of in the desktop's order.",
		s.staged.CaptionButtons == style.CaptionButtonsTheme, func(on bool) {
			next := s.staged
			next.CaptionButtons = style.CaptionButtonsDesktop
			if on {
				next.CaptionButtons = style.CaptionButtonsTheme
			}
			s.stage(next)
		})

	// The order is the order they are read, and it is also what makes
	// them fold well: the three narrowest first, so a narrow pane gets a
	// line of three and a line of two rather than a ragged four and one.
	// The frame pair stay next to each other across the fold — the
	// desktop's frame first, because the theme's button places only mean
	// anything while the toolkit is drawing the frame itself.
	row := widgets.NewWrap(motion, native, system, themeButtons, colours)
	row.Gap = 8
	row.SetAccessibleName(settingsSections[sectionBehaviour])
	return row
}

// ---- what the preview sets ----------------------------------------------------

// previewControls are the settings the previewed window carries on a bar
// of its own instead of showing what someone else chose: the icon set,
// the size its glyphs are drawn at, and the shape of its corners.
//
// They were a section called Shape and weight in the column on the left,
// with a strip of fifteen glyphs under them to show what a set draws.
// The strip was a picture of a tool bar; the preview has a real one,
// drawn in the staged set at the staged size, so the strip had nothing
// left to say and went. Corners came too, because a corner style is the
// shape of a window and the window is here.
//
// All three are combo boxes on one bar now, each behind the word that
// says what it sets. The set and the size spent a release at the far end
// of the sample's own tool bar, where they were mistaken for the
// sample's own style and size boxes, and corners spent it in the
// sample's View menu, where the owner of this toolkit could not find
// them at all. A menu hides; a labelled bar does not.
func (s *settingsState) previewControls() PreviewControls {
	if s.plain {
		// Pictures only: the window in the pane is the sample and
		// nothing else. See [SettingsOptions].
		return PreviewControls{}
	}
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
	// A tool bar item, not a form field: it fits its longest set name and
	// no more, so that all three choosers and their words stay on a bar
	// as narrow as the 720-pixel window gives it.
	s.iconPick.MinWidth = 1
	// The word on the bar is "Icons"; what a screen reader says is the
	// same word, because a name that disagreed with the one on the
	// screen would be two names for one control.
	s.iconPick.SetAccessibleName("Icons")
	s.iconPick.Tip = "Icon set — a real setting. This window's tools are drawn in it; the rest of it is a sample."
	s.iconPick.SetAccessibleDescription(s.iconPick.Tip)

	// The size is beside the set because it is not a separate choice: a
	// set's glyphs are drawn at it, some sets are made for one end of the
	// range, and a bar that showed a fixed size would be showing
	// something the user is not going to get. It lists the pixel sizes
	// rather than Small / Medium / Large because it is a size box on a
	// tool bar, where every application has written the number since the
	// first word processor.
	s.iconSize = widgets.NewComboBox(iconSizeNames(), iconSizeIndex(s.staged.IconSize), func(i int) {
		next := s.staged
		next.IconSize = iconSizes[i]
		s.stage(next)
	})
	s.iconSize.MinWidth = 1
	// "Size" on the bar, "Icon size" to a screen reader: the bar's word
	// is short because the chooser it names is the second half of a
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
	// make-believe. It is the third box on the bar now, and it is a
	// setting of the same kind as the other two: one word, one chooser.
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
	s.corners.Tip = "Window corners — a real setting: the pack's own shape, round, or square. This window is drawn with it."
	s.corners.SetAccessibleDescription(s.corners.Tip)

	return PreviewControls{Icons: s.iconPick, IconSize: s.iconSize, Corners: s.corners}
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
// A name and the path, three times over, and nothing else: Delete icon
// set… rode at the end of the icons line for a release and is gone. It
// was the last thing left of the old Packs page, it acted on a set
// chosen two inches away on the preview's own bar, and it put a button
// that destroys a directory on the one part of the page that was meant
// to change nothing. style.DeleteUserIconSet is still there for an
// application that wants it.
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
	panel := widgets.NewPanel(settingsSections[sectionFiles],
		line("Prefs", style.AppearancePath()),
		line("Themes", style.ThemesDir()+"/<name>/theme.json"),
		line("Icons", style.IconsDir()+"/<set>/*.png"),
	)
	// Three lines of one thing each: the 8 pixels a panel puts between
	// its children are for paragraphs, not for a table.
	panel.Content().WithGap(2)
	return panel
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
// application — and one that carries, on a labelled bar over its menu
// bar, the three settings it is itself the answer to: the icon set, the
// icon size and the shape of its corners.
//
// Three things are in this pane, in the order they are read: the five
// options, over the window, because the first of them decides which pack
// the window below it draws and the other four decide what the apps
// drawn in it will do; the window; and where the files are, under it,
// because that is where what the window shows ends up. The options are
// one row that folds to two in a narrow pane and the paths are three
// lines, and neither scrolls: the window takes every pixel the two of
// them leave, at every window size, which is the promise this page has
// always made about its right-hand side. Nothing else is allowed in
// here — the strip of fifteen glyphs was tried above the window and
// folded onto four lines at the 720x520 minimum, leaving the preview a
// caption and a menu bar.
func (s *settingsState) previewColumn() widget.Component {
	s.sections[sectionBehaviour] = s.optionsRow()
	s.previewBox = widgets.NewPanel("", PreviewAppWith(nil, s.previewControls()))
	s.previewBox.Window = true
	preview := s.scoped(s.previewBox)
	s.sections[sectionPreview] = preview
	s.sections[sectionFiles] = s.filesSection()
	col := widgets.NewColumn(s.sections[sectionBehaviour], preview, s.sections[sectionFiles]).WithGap(8)
	col.AddFlex(preview, 1)
	return col
}

// showStagedControls puts the staged appearance back into the controls
// the preview carries. The bars and the window follow the staged look
// through the preview's scope, but what the three choosers show is
// Settings' own and has to be told — and told whoever staged it, not
// only the controls themselves. Picking a pack in the browser, -stage,
// Apply and a desktop light / dark change all come through here.
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

// defaultChoicesRatio aims the column of choices at about 300 logical
// pixels whatever the window and the display scale are: a narrow window
// gives it a bigger share so its rows and its prose stay readable, a wide
// one hands the room to the previews. It was 240 while the column was the
// theme browser and nothing else; an option row is a label, a control and
// a line of prose, and 240 wrapped every one of them to three lines.
// Dragging the sash replaces it.
func defaultChoicesRatio(win *app.Window) float32 {
	const want, pad = 300, 30
	// Window.Size is already logical pixels, which is what `want` is in.
	lw, _ := win.Size()
	page := float32(lw) - pad
	if page <= 0 {
		return 0.3
	}
	return min(max(want/page, 0.18), 0.5)
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

// PreviewControls are the parts of the preview that are not a sample.
// Everything else in the previewed application is make-believe — Send
// sends nothing, the tree lists a mailbox nobody has — but these are the
// real settings of the desktop, and the window they stand in is drawn
// with them. Left empty, the preview is the sample it has always been.
//
// They go on a bar of their own at the head of the window, above the
// sample's menu bar, each behind a short word ([PreviewSettingsWords]).
type PreviewControls struct {
	// Icons chooses the icon set the window's tools are drawn in,
	// IconSize the pixel size of their glyphs, and Corners the shape of
	// the window itself. Any may be nil; all three nil is no bar.
	Icons, IconSize, Corners widget.Component
}

// PreviewSettingsWords are the words on that bar, in its order: the
// label in front of each chooser. They are short because the bar is as
// narrow as the preview pane of a 720-pixel window, and each one is the
// beginning of what its chooser is called to a screen reader ("Icons",
// "Icon size", "Window corners") rather than another name for it.
var PreviewSettingsWords = [3]string{"Icons", "Size", "Corners"}

// settingsBar is that bar: the live settings, over the sample's own menu
// bar and tool bar.
//
// Over, not under. The sample's tool bar belongs where a tool bar
// belongs, under the menu bar of the window it commands, and a second
// strip below it would read as the same application's second row of
// tools — which is exactly the mistake the last arrangement invited,
// when the two choosers rode at the end of the sample's own bar and were
// taken for its style and size boxes. Nothing in any application sits
// above its menu bar, so a strip that does is not the application's; it
// is the frame around it, in the same voice as the caption over it,
// which does not say a document's name either but "Preview — Windows
// 95". The seam is clean: caption and settings bar are Settings talking,
// and everything from the menu bar down is the sample.
//
// Each chooser carries a word. The bar sheds them from the right as it
// narrows and keeps the choosers whole, so at the smallest window the
// three boxes stand on their own with their tooltips — see the tool bar
// rule in [widgets.ToolStretch].
func settingsBar(ctl PreviewControls) *widgets.ToolBar {
	boxes := [3]widget.Component{ctl.Icons, ctl.IconSize, ctl.Corners}
	var items []*widgets.ToolItem
	for i, box := range boxes {
		if box == nil {
			continue
		}
		if i == 2 && len(items) > 0 {
			// The set and the size are one question about what is drawn;
			// the shape of the window is another.
			items = append(items, widgets.ToolDivider())
		}
		items = append(items, widgets.ToolLabel(PreviewSettingsWords[i]), widgets.ToolWidget(box))
	}
	if len(items) == 0 {
		return nil
	}
	// The free space is what makes the bar fill the window's width and
	// what lets it shed a word when it cannot: a bar without one is only
	// as wide as its items and drops nothing.
	items = append(items, widgets.ToolStretch())
	bar := widgets.NewToolBar(items...)
	bar.SetAccessibleName("Appearance")
	return bar
}

// PreviewApp is a small, fully interactive application used to preview a
// theme: menu bar, tool bar, tabs with every kind of control, lists, a
// tree, a table and a status bar. Nothing in it does anything outside
// itself. Settings shows it inside a ThemeScope.
func PreviewApp(say func(string)) widget.Component {
	return PreviewAppWith(say, PreviewControls{})
}

// PreviewAppWith is [PreviewApp] carrying controls of the application
// that shows it: settings the previewed window is itself the answer to.
// Settings passes the icon set, the icon size and the corner style, and
// they stand on a bar of their own over the sample's menu bar, so that
// the tools a set is chosen on are drawn in it and the window whose
// corners are chosen is the window they round.
func PreviewAppWith(say func(string), ctl PreviewControls) widget.Component {
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
	// The sample's own bar, and nothing but: eight commands in three
	// groups, the way a text editor's is. Paste is back beside Cut and
	// Copy — it was the button the choosers cost this bar when they rode
	// at the end of it, and a clipboard group of two was a group with a
	// hole in it. The bar is drawn in the staged set at the staged size,
	// which is what makes it the preview of them.
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
	// each for a long time. No window has those, and they were the room
	// the settings bar needed.)
	sample := widgets.NewColumn(menu, tools, tabs, sb).WithGap(0)
	sample.AddFlex(tabs, 1)
	bar := settingsBar(ctl)
	if bar == nil {
		return sample
	}
	// And that gap is what the settings bar stands in: it is not part of
	// the window under it, so it does not touch it.
	win := widgets.NewColumn(bar, sample)
	win.AddFlex(sample, 1)
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

// ---- helpers ------------------------------------------------------------------

// wrapped is a label that wraps: prose on this page is read, and a line
// elided at the column's edge hides the part that says where a file is.
func wrapped(text string) *widgets.Label {
	l := widgets.NewLabel(text)
	l.Wrap = true
	return l
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

// scrollTo puts one section of a scrolling column at the top of it, once,
// at the end of the first layout. -page names a section now that Settings
// is one page, and where a section starts is not known until the column
// has been measured at the width the splitter gave it — which happens
// after the page is built, so it cannot be an offset set when the page is
// made. This holds the scroll view, lets it lay out, then scrolls it and
// does nothing ever after.
type scrollTo struct {
	widget.Base
	view    *widgets.ScrollView
	section widget.Component
	done    bool
}

func newScrollTo(view *widgets.ScrollView, section widget.Component) widget.Component {
	if view == nil || section == nil {
		return view
	}
	b := &scrollTo{view: view, section: section}
	b.Init(b)
	b.Add(view)
	return b
}

func (b *scrollTo) Measure(c layout.Constraints) paintengine2d.Point {
	return b.view.Measure(c)
}

func (b *scrollTo) Arrange(r paintengine2d.Rect) {
	b.SetBounds(r)
	b.view.Arrange(paintengine2d.XYWH(0, 0, r.Dx(), r.Dy()))
	if b.done || r.Empty() {
		return
	}
	b.done = true
	// The section's top in the scrolled content: every offset from it up
	// to the view, plus the offset the view is already at (the content is
	// arranged at -OffsetY).
	top := b.view.OffsetY
	for p := widget.Component(b.section); p != nil && p != widget.Component(b.view); p = p.Parent() {
		top += p.Bounds().Min.Y
	}
	b.view.ScrollTo(top - 8)
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
