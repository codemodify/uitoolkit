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

// SettingsApp is the toolkit appearance editor (theme packs + icon sets).
// Picker changes preview locally via Application.SetLook. Apply writes
// look.json {theme, icons} so other apps watching the file reload.
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

	st := staged.Name + " · " + string(staged.Icons)
	if staged != saved {
		st += " · unapplied"
	}
	status := widgets.NewStatusBar(st, style.AppearancePath(), "v"+uitoolkit.Version)
	chrome := widgets.NewTitleBar("Settings", "Theme packs and PNG icon sets for every uitoolkit app")

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

	themes := style.ListThemes()
	sel := 0
	for i, p := range themes {
		if p.Name == staged.Name {
			sel = i
			break
		}
	}
	picker := widgets.NewListView(len(themes), func(i int) string {
		if i < 0 || i >= len(themes) {
			return ""
		}
		return themes[i].Display()
	}, func(i int) {
		if i < 0 || i >= len(themes) {
			return
		}
		next := themes[i].Appearance()
		next.Icons = staged.Icons
		preview(next)
	})
	picker.Selected = sel
	picker.RowHeight = 28

	iconSets := style.ListIconSets()
	iconSel := 0
	for i, s := range iconSets {
		if s.Name == staged.Icons {
			iconSel = i
			break
		}
	}
	iconPicker := widgets.NewListView(len(iconSets), func(i int) string {
		if i < 0 || i >= len(iconSets) {
			return ""
		}
		return iconSets[i].Label
	}, func(i int) {
		if i < 0 || i >= len(iconSets) {
			return
		}
		next := staged
		next.Icons = iconSets[i].Name
		preview(next)
	})
	iconPicker.Selected = iconSel
	iconPicker.RowHeight = 28

	var exportHost widget.Component
	exportBtn := widgets.NewButton("Export current look…", func() {
		promptExportName(exportHost, func(name string) {
			name = strings.TrimSpace(name)
			pack, err := style.ExportAppearance(name, staged)
			if err != nil {
				status.Set(0, err.Error())
				return
			}
			preview(pack.Appearance())
		})
	})
	exportHost = exportBtn

	detail := widgets.NewColumn(
		widgets.NewLabel("Pack  "+staged.Name),
		widgets.NewLabel("Palette  "+string(staged.Theme)),
		widgets.NewLabel("Corners  "+string(staged.Corners)),
		widgets.NewLabel("Icons  "+string(staged.Icons)+"  (on top of the pack)"),
		widgets.NewLabel("Copy repo icons/<set>/ PNGs into ~/.config/uitoolkit/icons/ to install sets."),
		widgets.NewLabel("User packs override builtins of the same name."),
		widgets.NewRow(exportBtn).WithGap(8),
	).WithGap(6)
	pickerCol := widgets.NewColumn(widgets.NewLabel("Theme"), picker).WithGap(4)
	pickerCol.AddFlex(picker, 1)
	iconCol := widgets.NewColumn(widgets.NewLabel("Icons"), iconPicker).WithGap(4)
	iconCol.AddFlex(iconPicker, 1)
	choices := widgets.NewRow(pickerCol, iconCol, detail).WithGap(16)
	choices.AddFlex(pickerCol, 3)
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
		widgets.NewLabel("Prefs file (theme + icons, written on Apply):"),
		aboutPath,
		widgets.NewLabel("User theme packs (exported; override builtins by name):"),
		themesPath,
		widgets.NewLabel("Icon sets (copy repo lucide/phosphor/tabler/heroicons/material-symbols here):"),
		iconsPath,
		widgets.NewLabel("Eight starter packs are embedded. Icon PNGs are not."),
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
	hint := widgets.NewLabel("Apply writes theme + icons to look.json. Close without Apply discards staged changes.")
	actions := widgets.NewRow(applyBtn, hint).WithGap(12).WithPad(8)

	root := widgets.NewColumn(chrome, split, actions, status)
	root.AddFlex(split, 1)
	return root
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
		widgets.NewLabel("Name the pack. Written to ~/.config/uitoolkit/themes/<name>/theme.json so you can tweak it."),
		field,
		widgets.NewRow(cancel, ok).WithGap(8),
	)
	card.Raised = true
	overlay = widgets.NewOverlay(card)
	overlay.MinCardH = 200
	widget.ShowOverlay(from, overlay)
}
