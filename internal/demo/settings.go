package demo

import (
	"fmt"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// SettingsApp is the toolkit appearance editor. Theme (palette), corners,
// and icon set are independent. Picker changes preview locally via
// Application.SetLook. Apply writes look.json {theme, corners, icons}.
// Closing the window without Apply discards staged changes.
func SettingsApp(a *app.Application, win *app.Window) widget.Component {
	saved := style.LoadAppearance().Normalize()
	return buildSettings(a, win, saved, saved, 0)
}

func buildSettings(a *app.Application, win *app.Window, saved, staged style.Appearance, section int) widget.Component {
	saved = saved.Normalize()
	staged = staged.Normalize()
	if section < 0 || section > 1 {
		section = 0
	}

	st := staged.Name + " · " + string(staged.Corners) + " · " + string(staged.Icons)
	if staged != saved {
		st += " · unapplied"
	}
	status := widgets.NewStatusBar(st, style.AppearancePath(), "v"+uitoolkit.Version)
	chrome := widgets.NewTitleBar("Settings", "Theme, corners, and icon sets for every uitoolkit app")

	preview := func(next style.Appearance) {
		next = next.Normalize()
		a.SetLook(next.Look())
		win.SetContent(buildSettings(a, win, saved, next, section))
	}
	persist := func() {
		next := staged.Normalize()
		if err := style.SaveAppearance(next); err != nil {
			status.Set(0, err.Error())
			return
		}
		a.SetLook(next.Look())
		win.SetContent(buildSettings(a, win, next, next, section))
	}

	themeBuiltin := style.ListBuiltinThemes()
	themeUser := style.ListUserThemes()
	onTheme := func(p style.ThemePack) {
		next := staged
		next.Name = p.Name
		next.Theme = p.Palette
		preview(next)
	}
	builtinTheme := pickerSection("Built-in", len(themeBuiltin), func(i int) string {
		return themeBuiltin[i].Display()
	}, indexTheme(themeBuiltin, staged.Name), func(i int) {
		if i >= 0 && i < len(themeBuiltin) {
			onTheme(themeBuiltin[i])
		}
	})
	userTheme := pickerSection("User", len(themeUser), func(i int) string {
		return themeUser[i].Display()
	}, indexTheme(themeUser, staged.Name), func(i int) {
		if i >= 0 && i < len(themeUser) {
			onTheme(themeUser[i])
		}
	})

	iconBuiltin := style.ListBuiltinIconSets()
	iconUser := style.ListUserIconSets()
	onIcon := func(s style.IconSetInfo) {
		next := staged
		next.Icons = s.Name
		preview(next)
	}
	builtinIcons := pickerSection("Built-in", len(iconBuiltin), func(i int) string {
		return iconBuiltin[i].Label
	}, indexIcon(iconBuiltin, staged.Icons), func(i int) {
		if i >= 0 && i < len(iconBuiltin) {
			onIcon(iconBuiltin[i])
		}
	})
	userIcons := pickerSection("User", len(iconUser), func(i int) string {
		return iconUser[i].Label
	}, indexIcon(iconUser, staged.Icons), func(i int) {
		if i >= 0 && i < len(iconUser) {
			onIcon(iconUser[i])
		}
	})

	cornerSel := 0
	if staged.Corners == style.CornersSquare {
		cornerSel = 1
	}
	corners := widgets.NewRadioGroup([]string{"Round", "Square"}, cornerSel, func(i int) {
		next := staged
		if i == 1 {
			next.Corners = style.CornersSquare
		} else {
			next.Corners = style.CornersRound
		}
		preview(next)
	})
	cornerCol := widgets.NewColumn(widgets.NewLabel("Corners"), corners).WithGap(4)

	var exportHost widget.Component
	exportBtn := widgets.NewButton("Export current theme…", func() {
		promptExportName(exportHost, func(name string) {
			name = strings.TrimSpace(name)
			pack, err := style.ExportAppearance(name, staged)
			if err != nil {
				status.Set(0, err.Error())
				return
			}
			next := staged
			next.Name = pack.Name
			next.Theme = pack.Palette
			preview(next)
		})
	})
	exportHost = exportBtn

	detail := widgets.NewColumn(
		widgets.NewLabel("Theme  "+staged.Name),
		widgets.NewLabel("Palette  "+string(staged.Theme)),
		widgets.NewLabel("Corners  "+string(staged.Corners)),
		widgets.NewLabel("Icons  "+string(staged.Icons)),
		widgets.NewLabel("look.json stores theme, corners, and icons independently."),
		widgets.NewLabel("Export writes the color theme only; corners and icons stay prefs."),
		widgets.NewLabel("Built-in vs User: stock dark/light and premiere icon names, then custom folders."),
		widgets.NewRow(exportBtn).WithGap(8),
	).WithGap(6)
	pickerCol := widgets.NewColumn(widgets.NewLabel("Theme"), builtinTheme, userTheme).WithGap(8)
	pickerCol.AddFlex(builtinTheme, 1)
	if len(themeUser) > 0 {
		pickerCol.AddFlex(userTheme, 1)
	}
	iconCol := widgets.NewColumn(widgets.NewLabel("Icons"), builtinIcons, userIcons).WithGap(8)
	iconCol.AddFlex(builtinIcons, 1)
	if len(iconUser) > 0 {
		iconCol.AddFlex(userIcons, 1)
	}
	choices := widgets.NewRow(pickerCol, cornerCol, iconCol, detail).WithGap(16)
	choices.AddFlex(pickerCol, 3)
	choices.AddFlex(cornerCol, 1)
	choices.AddFlex(iconCol, 2)
	choices.AddFlex(detail, 2)

	primary := widgets.NewButton("Primary action", func() { status.Set(0, "Primary") })
	primary.Primary = true
	secondary := widgets.NewButton("Secondary", func() { status.Set(0, "Secondary") })
	disabled := widgets.NewButton("Disabled", nil)
	disabled.SetEnabled(false)

	field := widgets.NewTextField("Ada Lovelace", "Display name", func(s string) {
		status.Set(0, "Field: "+s)
	})
	combo := widgets.NewComboBox([]string{"Classic look", "Squared chrome", "Sharp icons"}, 0, func(i int) {
		status.Set(0, fmt.Sprintf("Combo %d", i))
	})
	check := widgets.NewCheckbox("Enable notifications", true, nil)
	sw := widgets.NewSwitch("Compact density", false, nil)
	slider := widgets.NewSlider(0, 100, 60, func(v float32) {
		status.Set(0, fmt.Sprintf("Slider %d", int(v+0.5)))
	})

	newBtn := widgets.ToolIconBtn(style.IconNew, "", func() { status.Set(0, "New") })
	newBtn.Tip = "New"
	openBtn := widgets.ToolIconBtn(style.IconOpen, "", func() { status.Set(0, "Open") })
	openBtn.Tip = "Open"
	saveBtn := widgets.ToolIconBtn(style.IconSave, "", func() { status.Set(0, "Save") })
	saveBtn.Tip = "Save"
	cutBtn := widgets.ToolIconBtn(style.IconCut, "", nil)
	copyBtn := widgets.ToolIconBtn(style.IconCopy, "", nil)
	pasteBtn := widgets.ToolIconBtn(style.IconPaste, "", nil)
	searchBtn := widgets.ToolIconBtn(style.IconSearch, "", nil)
	infoBtn := widgets.ToolIconBtn(style.IconInfo, "", nil)
	warnBtn := widgets.ToolIconBtn(style.IconWarning, "", nil)
	errBtn := widgets.ToolIconBtn(style.IconError, "", nil)
	toolbar := widgets.NewToolBar(
		newBtn, openBtn, saveBtn, widgets.ToolDivider(),
		cutBtn, copyBtn, pasteBtn, widgets.ToolDivider(),
		searchBtn, infoBtn, warnBtn, errBtn,
	)

	previewPane := widgets.NewPanel("Live preview",
		widgets.NewRow(primary, secondary, disabled).WithGap(10),
		toolbar,
		widgets.NewRow(field, combo).WithGap(10),
		widgets.NewRow(check, sw).WithGap(16),
		slider,
	)

	appearance := widgets.NewColumn(
		widgets.NewTitle("Appearance"),
		widgets.NewLabel("Shared by Mail, gallery, and any app that calls PreferredLook()."),
		choices,
		previewPane,
	).WithGap(10).WithPad(4)
	appearance.AddFlex(choices, 1)

	aboutPath := widgets.NewMonoTextView(style.AppearancePath(), "")
	aboutPath.MinRows = 2
	aboutPath.Wrap = true
	themesPath := widgets.NewMonoTextView(style.ThemesDir()+"/<name>/theme.json", "")
	themesPath.MinRows = 2
	themesPath.Wrap = true
	iconsPath := widgets.NewMonoTextView(style.IconsDir()+"/<set>/*.png", "")
	iconsPath.MinRows = 2
	iconsPath.Wrap = true
	about := widgets.NewColumn(
		widgets.NewTitle("About"),
		widgets.NewLabel("uitoolkit v"+uitoolkit.Version),
		widgets.NewLabel("LookAndFeel Classic — Titillium Web + JetBrains Mono."),
		widgets.NewLabel("Prefs file (theme + corners + icons, written on Apply):"),
		aboutPath,
		widgets.NewLabel("User color themes (exported palette; listed under User; override builtins by name):"),
		themesPath,
		widgets.NewLabel("Icon sets: Built-in = lucide/phosphor/tabler/heroicons/material-symbols when copied, plus drawn classic/sharp. User = any other folder:"),
		iconsPath,
		widgets.NewLabel("Two starter palettes are embedded (dark, light). Corners and icons are separate prefs."),
		widgets.NewLabel("Other apps watch look.json and call SetLook(PreferredLook())."),
		widgets.NewButton("Open appearance", func() {
			win.SetContent(buildSettings(a, win, saved, staged, 0))
		}),
	).WithGap(10).WithPad(4)

	sections := []string{"Appearance", "About"}
	nav := widgets.NewListView(len(sections), func(i int) string { return sections[i] }, func(i int) {
		win.SetContent(buildSettings(a, win, saved, staged, i))
	})
	nav.Selected = section
	nav.RowHeight = 32
	side := widgets.NewColumn(
		widgets.NewTitle("Settings"),
		widgets.NewLabel("v"+uitoolkit.Version),
		nav,
	).WithGap(8).WithPad(10)

	var page widget.Component = appearance
	if section == 1 {
		page = about
	}
	// Pad (not ScrollView): a ScrollView as a Splitter pane is a SceneLayer
	// that Splitter.Paint never recordNode-walks, so UITK_SCENE skips its
	// children. Column/Pad paint through PaintTree instead.
	right := widgets.NewPad(12, page)
	split := widgets.NewSplitter(true, side, right)
	split.Ratio = 0.24

	applyBtn := widgets.NewButton("Apply", persist)
	applyBtn.Primary = true
	if staged == saved {
		applyBtn.SetEnabled(false)
	}
	hint := widgets.NewLabel("Apply writes theme, corners, and icons to look.json. Close without Apply discards staged changes.")
	actions := widgets.NewRow(applyBtn, hint).WithGap(12).WithPad(8)

	root := widgets.NewColumn(chrome, split, actions, status)
	root.AddFlex(split, 1)
	return root
}

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
		widgets.NewLabel("Name the color theme (palette only). Written to ~/.config/uitoolkit/themes/<name>/theme.json. Corners and icons stay in look.json."),
		field,
		widgets.NewRow(cancel, ok).WithGap(8),
	)
	card.Raised = true
	overlay = widgets.NewOverlay(card)
	overlay.MinCardH = 200
	widget.ShowOverlay(from, overlay)
}
