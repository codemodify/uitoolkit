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
// that page has two halves: the choices that are about something other
// than the window on the right go down the column on the left (which
// pack, and what the apps do besides drawing it), and the choices the
// preview can answer for itself stand with it on the right (where its
// colours come from, the shape and the icons it is drawn in, and where
// all of it is kept on disk).
const (
	sectionTheme     = iota // the column, at the top
	sectionBehaviour        // the column, under it
	sectionDesktop          // the right-hand pane, over the preview
	sectionPreview          // the preview itself: its icons and its corners
	sectionFiles            // the right-hand pane, under the preview
)

// settingsSections names the sections in that order. -page takes these
// names, and the names of the four pages they came from.
var settingsSections = []string{"Theme", "Behaviour", "Where the colours come from", "Preview", "Files"}

// inColumn reports whether a section is one of the two in the scrolling
// column: the others are always on screen, so there is nothing to scroll
// to.
func inColumn(section int) bool { return section == sectionTheme || section == sectionBehaviour }

// Theme browser filters: every built-in pack, one decade, or user packs.
var themeFilters = []string{"All decades", "1980s", "1990s", "2000s", "2010s", "2020s", "My themes"}

const filterUser = 6

// SettingsApp is the toolkit appearance editor, and it is one page: the
// choices down a scrolling column on the left — the theme browser, where
// its colours come from, the shape and weight it is drawn in, what the
// apps do besides drawing it, and the files it all lives in — and on the
// right, filling the rest of the window at every size, the thing those
// choices are about: a small application window whose frame, caption and
// every control come from the staged pack. Settings itself keeps the
// applied look (the preview is a ThemeScope), so a pack can be judged
// without living in it.
//
// Apply — the one button, at the right of the row under the page —
// writes look.json and every app that watches it (and Settings) switches.
// Closing without Apply discards the staged change.
func SettingsApp(a *app.Application, win *app.Window) widget.Component {
	saved := style.LoadAppearance().Normalize()
	return buildSettings(a, win, saved, saved, sectionTheme)
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
	saved := style.LoadAppearance().Normalize()
	staged := saved
	if pack, ok := style.LoadTheme(theme); ok {
		staged.Name, staged.Theme = pack.Name, pack.Palette
	}
	return buildSettings(a, win, saved, staged, SettingsPage(page))
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
// "about" is Files. Three of the five sections are beside the preview
// rather than in the column, and a name that resolves to one of those
// leaves the column where it is: what it names is already on screen.
func SettingsPage(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "appearance", "shape", "shape and weight", "shape and motion", "corners", "icons", "icon sets", "preview":
		return sectionPreview
	case "desktop", "the desktop", "colours", "colors", "where the colours come from":
		return sectionDesktop
	case "behaviour", "behavior", "windows", "motion":
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
	// What Delete acts on follows the staged choice, so both buttons are
	// always on the page and go grey when nothing of the user's own is
	// staged: a button that came and went would move everything under it
	// every time a pack was picked.
	delTheme *widgets.Button
	delIcons *widgets.Button
	// The controls the preview carries instead of showing: the sets the
	// chooser offers, the chooser itself and the size its glyphs are
	// drawn at, both on the previewed application's tool bar, and the
	// window-shape choice in its View menu. They follow the staged
	// appearance whatever staged it — a pack picked in the browser,
	// -stage, Apply — and not only their own clicks.
	iconSets []style.IconSetInfo
	iconPick *widgets.ComboBox
	iconSize *widgets.ComboBox
	corners  []*widgets.MenuItem
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

func buildSettings(a *app.Application, win *app.Window, saved, staged style.Appearance, section int) widget.Component {
	s := &settingsState{
		a: a, win: win, saved: saved.Normalize(), staged: staged.Normalize(), section: section,
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
	s.delTheme, s.delIcons = nil, nil
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
	if s.delIcons != nil {
		s.delIcons.SetEnabled(indexIcon(style.ListUserIconSets(), s.staged.Icons) >= 0)
	}
	if s.applyBtn != nil {
		s.applyBtn.SetEnabled(s.staged != s.saved)
	}
}

// ---- the column of choices ------------------------------------------------

// choicesColumn is the left-hand column: the two decisions the preview
// cannot make for itself — which pack, and what the apps do besides
// drawing it. The shape and the icons went to the preview's own chrome,
// where what shows a setting is what sets it; where the colours come
// from and where the files are went to the pane beside this one, over
// and under the preview. What is left still scrolls, because a list of
// 129 packs and four switches do not fit in a 520-pixel window and
// something has to give — and what must not give is the preview beside
// it, which is why it is not in here.
func (s *settingsState) choicesColumn() widget.Component {
	s.sections[sectionTheme] = s.themeSection()
	s.sections[sectionBehaviour] = s.behaviourSection()
	col := widgets.NewColumn(
		widgets.NewTitle("Settings"),
		widgets.NewLabel("v"+uitoolkit.Version),
		s.sections[sectionTheme],
		s.sections[sectionBehaviour],
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

// ---- Where the colours come from --------------------------------------------

// desktopSection is the one setting that changes where the look comes
// from rather than what it is: with it on, the pack Settings draws is not
// always the pack that was chosen. It stands over the preview because
// that is where its effect is read — the caption under it says "Preview
// — Breeze Dark" for a chosen Breeze — and it is one row rather than a
// panel of its own because every pixel it takes is a pixel off the
// window it is a footnote to.
func (s *settingsState) desktopSection() widget.Component {
	// GNOME's and Plasma's light / dark setting and accent colour: the
	// theme shows its sibling (Breeze and Breeze Dark) to match the
	// desktop, recoloured around its accent where the engine takes one.
	follow := widgets.NewSwitch("Follow the desktop's colours", s.staged.FollowDesktop, func(on bool) {
		next := s.staged
		next.FollowDesktop = on
		s.stage(next)
	})
	// The switch over its line of prose, not beside it: a line that wraps
	// beside a switch is a line the row measured at its unwrapped width
	// and then cut off. And one short line, not the paragraph this had
	// in the column: it stands on top of the window it is about, where
	// every line of prose is a line off that window, and the caption
	// right under it — "Preview — Breeze Dark" for a chosen Breeze — is
	// the rest of the explanation.
	col := optionSwitch(follow, "Light or dark and the accent, as the desktop asks.")
	col.SetAccessibleName(settingsSections[sectionDesktop])
	return col
}

// ---- what the preview sets ----------------------------------------------------

// previewControls are the settings the previewed application carries on
// its own chrome instead of showing what someone else chose: the icon
// set and the size its glyphs are drawn at, on the tool bar that draws
// in them, and the window's corner style, in the View menu of the window
// it shapes.
//
// They were a section called Shape and weight in the column on the left,
// with a strip of fifteen glyphs under them to show what a set draws.
// The strip was a picture of a tool bar; the preview has a real one,
// drawn in the staged set at the staged size, so the strip had nothing
// left to say and went. Corners came too, because a corner style is the
// shape of a window and the window is here.
func (s *settingsState) previewControls() PreviewControls {
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
	// no more, so that the tools it shares the bar with stay on it.
	s.iconPick.MinWidth = 1
	s.iconPick.SetAccessibleName("Icons")
	s.iconPick.Tip = "Icon set — a real setting. This tool bar is drawn in it; the rest of this window is a sample."
	s.iconPick.SetAccessibleDescription(s.iconPick.Tip)

	// The size is beside the set because it is not a separate choice: a
	// set's glyphs are drawn at it, some sets are made for one end of the
	// range, and a bar that showed a fixed size would be showing
	// something the user is not going to get. It lists the pixel sizes
	// rather than Small / Medium / Large because it is a size box on a
	// tool bar, where every application has written the number since the
	// first word processor — and because the three words are 50 pixels
	// the tools beside it would have paid for.
	s.iconSize = widgets.NewComboBox(iconSizeNames(), iconSizeIndex(s.staged.IconSize), func(i int) {
		next := s.staged
		next.IconSize = iconSizes[i]
		s.stage(next)
	})
	s.iconSize.MinWidth = 1
	s.iconSize.SetAccessibleName("Icon size")
	s.iconSize.Tip = "Icon size — a real setting: 16, 24 or 32 pixels, before the display scale."
	s.iconSize.SetAccessibleDescription(s.iconSize.Tip)

	// Corners are in the preview's View menu rather than on its tool bar:
	// they are the shape of the whole window — its frame, its buttons,
	// its fields — and not one bar's business, and a View menu is where
	// an application has always kept what its window looks like. It is
	// also what keeps that bar legible: a third chooser on it would not
	// fit beside the tools at the width Settings opens down to.
	s.corners = nil
	for _, c := range []struct {
		text  string
		shape style.CornerStyle
	}{
		{"&Theme shape", style.CornersTheme},
		{"&Round", style.CornersRound},
		{"S&quare", style.CornersSquare},
	} {
		shape := c.shape
		s.corners = append(s.corners, widgets.RadioItem(c.text, "corners", s.staged.Corners == shape, func() {
			next := s.staged
			next.Corners = shape
			s.stage(next)
		}))
	}
	return PreviewControls{Icons: s.iconPick, IconSize: s.iconSize, Corners: s.corners}
}

// ---- Behaviour ---------------------------------------------------------------

// behaviourSection is everything look.json carries that is not what the
// toolkit looks like but what it does: whether it moves, whose file
// dialogs it opens, who draws a window's frame and where that frame's
// buttons go. The old Appearance page had these in three groups of one
// and two; they are one group here, because the thing they have in
// common — none of them is the appearance of a pack — is the only thing
// a reader needs to know to skip the lot.
func (s *settingsState) behaviourSection() widget.Component {
	// Hover fades, the default button's pulse, busy bars: off for users
	// who get unwell from motion (GTK's gtk-enable-animations).
	motion := widgets.NewSwitch("Animations", !s.staged.ReduceMotion, func(on bool) {
		next := s.staged
		next.ReduceMotion = !on
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

	panel := widgets.NewPanel(settingsSections[sectionBehaviour],
		optionSwitch(motion, "Hover fades, the default button's pulse and busy bars."),
	)
	if style.DesktopReducesMotion() {
		panel.Add(wrapped("The desktop asks for reduced motion, so animations stay off."))
	}
	panel.Add(optionSwitch(native, "KDE's and GNOME's own Open and Save dialogs, through the XDG portal."))
	panel.Add(optionSwitch(system, "The desktop draws the title bar and borders of every window."))
	panel.Add(optionSwitch(themeButtons, "Close, minimise and maximise where the theme's era put them."))
	return panel
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
// The grid is what keeps them a table: a name, the path, and on the last
// line the one thing that can be done to what is in that directory.
func (s *settingsState) filesSection() widget.Component {
	line := func(name, path string, tail widget.Component) widget.Component {
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
		if tail != nil {
			row.Add(tail)
		}
		return row
	}
	panel := widgets.NewPanel(settingsSections[sectionFiles],
		line("Prefs", style.AppearancePath(), nil),
		line("Themes", style.ThemesDir()+"/<name>/theme.json", nil),
		line("Icons", style.IconsDir()+"/<set>/*.png", s.deleteIconsButton()),
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

// deleteIconsButton removes the staged icon set from disk when it is one
// the user copied in. It lost its neighbour when the two icon choosers
// went to the preview's tool bar, where there is no room for a button
// and no sense in one — a bar draws a set, it does not keep one. It is
// on the icon sets line of Files instead, the one place left that says
// where sets come from, because that is what deleting one is: taking a
// folder out of that directory.
func (s *settingsState) deleteIconsButton() widget.Component {
	var host widget.Component
	s.delIcons = widgets.NewButton("Delete icon set…", func() {
		name := s.staged.Icons
		if indexIcon(style.ListUserIconSets(), name) < 0 {
			return
		}
		widgets.Confirm(host, "Delete icon set?", "Remove "+string(name)+" from disk? This cannot be undone.", func(yes bool) {
			if !yes {
				return
			}
			if err := style.DeleteUserIconSet(name); err != nil {
				s.fail("Delete failed", err)
				return
			}
			next := s.staged
			if next.Icons == name {
				next.Icons = style.IconSetClassic
			}
			if s.saved.Icons == name {
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
	host = s.delIcons
	s.delIcons.SetEnabled(false)
	return s.delIcons
}

// ---- the previews -------------------------------------------------------------

// previewColumn is the right-hand side and the whole reason the page is
// shaped this way: the staged pack drawn as a small but entirely live
// application — and one that now answers for itself the three questions
// its own chrome can show, the icon set, the icon size and the shape of
// its corners.
//
// Three things are in this pane, in the order they are read: where the
// colours come from, over the window, because that switch decides which
// pack the window below it draws; the window; and where the files are,
// under it, because that is where what the window shows ends up. Both
// are single rows and neither scrolls: the window takes every pixel the
// two of them leave, at every window size, which is the promise this
// page has always made about its right-hand side. Nothing else is
// allowed in here — the strip of fifteen glyphs was tried above the
// window and folded onto four lines at the 720x520 minimum, leaving the
// preview a caption and a menu bar.
func (s *settingsState) previewColumn() widget.Component {
	s.sections[sectionDesktop] = s.desktopSection()
	s.previewBox = widgets.NewPanel("", PreviewAppWith(nil, s.previewControls()))
	s.previewBox.Window = true
	preview := s.scoped(s.previewBox)
	s.sections[sectionPreview] = preview
	s.sections[sectionFiles] = s.filesSection()
	col := widgets.NewColumn(s.sections[sectionDesktop], preview, s.sections[sectionFiles]).WithGap(8)
	col.AddFlex(preview, 1)
	return col
}

// showStagedControls puts the staged appearance back into the controls
// the preview carries. The bar and the window follow the staged look
// through the preview's scope, but what the choosers and the menu show
// is Settings' own and has to be told — and told whoever staged it, not
// only the controls themselves. Picking a pack in the browser, -stage,
// Apply, a desktop light / dark change and deleting a user icon set all
// come through here.
func (s *settingsState) showStagedControls() {
	for i, it := range s.corners {
		if it != nil && i < len(cornerStyles) {
			it.Checked = s.staged.Corners == cornerStyles[i]
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

// cornerStyles are the shapes the preview's View menu offers, in its
// order: the pack's own, then round, then square.
var cornerStyles = []style.CornerStyle{style.CornersTheme, style.CornersRound, style.CornersSquare}

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
// real settings, carried on the chrome that shows what they do: the icon
// choosers on the tool bar that is drawn in them, the corner styles in
// the View menu of the window they shape. Left empty, the preview is the
// sample it has always been.
type PreviewControls struct {
	// Icons and IconSize sit at the right-hand end of the tool bar,
	// after the free space, behind a rule that separates them from the
	// sample's own buttons.
	Icons, IconSize widget.Component
	// Corners are the View menu's window-shape choices, an exclusive
	// group under "Window corners".
	Corners []*widgets.MenuItem
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
// Settings passes the icon choosers and the corner styles, so that the
// tool bar a set is chosen on is the tool bar drawn in it and the window
// whose corners are chosen is the window they round.
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
		widgets.NewMenu("&View", viewItems(ctl)...),
		widgets.NewMenu("&Help", widgets.Item("&About", func() { say("About") })),
	)
	bold := widgets.ToolIconBtn(style.IconPen, "", nil)
	bold.Toggle, bold.Down = true, true
	items := []*widgets.ToolItem{
		widgets.ToolIconBtn(style.IconNew, "", func() { say("New") }),
		widgets.ToolIconBtn(style.IconOpen, "", func() { say("Open") }),
		widgets.ToolIconBtn(style.IconSave, "", func() { say("Save") }),
		widgets.ToolDivider(),
		widgets.ToolIconBtn(style.IconCut, "", nil),
		widgets.ToolIconBtn(style.IconCopy, "", nil),
		widgets.ToolDivider(),
		bold,
		widgets.ToolIconBtn(style.IconMail, "Send", func() { say("Send") }),
	}
	// The icon choosers go to the far end of the bar, behind its free
	// space: a tool bar's right-hand end is where an application keeps
	// what is not a command — a zoom, a style, a size — and the gap is
	// what says where the sample stops and the settings start. (A rule
	// as well was tried; beside a gap that wide it is noise, and it cost
	// the bar a tool button it could not spare.) The bar in between is
	// drawn in whatever they choose, which is what makes it a preview of
	// them and not merely the place they sit.
	if ctl.Icons != nil || ctl.IconSize != nil {
		items = append(items, widgets.ToolStretch())
		if ctl.Icons != nil {
			items = append(items, widgets.ToolWidget(ctl.Icons))
		}
		if ctl.IconSize != nil {
			items = append(items, widgets.ToolWidget(ctl.IconSize))
		}
	}
	tools := widgets.NewToolBar(items...)
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

	win := widgets.NewColumn(menu, tools, tabs, sb)
	win.AddFlex(tabs, 1)
	return win
}

// viewItems is the previewed application's View menu: what its window
// looks like. The corner styles are the one live thing in any of its
// menus, so they are named as a group of their own — a person who opens
// View sees "Window corners" and three shapes, not three loose words
// among the sample's own switches.
func viewItems(ctl PreviewControls) []*widgets.MenuItem {
	mock := []*widgets.MenuItem{
		widgets.CheckItem("Status &bar", true, nil),
		widgets.CheckItem("&Word wrap", false, nil),
	}
	if len(ctl.Corners) == 0 {
		return mock
	}
	return append([]*widgets.MenuItem{
		widgets.Submenu("Window &corners", ctl.Corners...),
		widgets.Sep(),
	}, mock...)
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

// optionSwitch is a switch whose own text is the setting, with the line
// that says what turning it on does.
func optionSwitch(sw *widgets.Switch, about string) *widgets.FlexBox {
	desc := widgets.NewLabel(about)
	desc.Wrap = true
	return widgets.NewColumn(sw, desc).WithGap(2)
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
