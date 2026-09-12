package demo

import (
	"fmt"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// SettingsApp is the toolkit appearance editor (theme, corners, icon set).
// Radio changes preview locally via Application.SetLook. Apply writes
// look.json (the source of truth) so other apps watching the file reload.
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

	st := staged.String()
	if staged != saved {
		st += " · unapplied"
	}
	status := widgets.NewStatusBar(st, style.AppearancePath(), "v"+uitoolkit.Version)
	chrome := widgets.NewTitleBar("Settings", "Appearance for every uitoolkit app")

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

	themeSel := 0
	if staged.Theme == style.ThemeLight {
		themeSel = 1
	}
	theme := widgets.NewRadioGroup([]string{"Dark graphite", "Light paper"}, themeSel, func(i int) {
		next := staged
		if i == 1 {
			next.Theme = style.ThemeLight
		} else {
			next.Theme = style.ThemeDark
		}
		preview(next)
	})

	cornerSel := 0
	if staged.Corners == style.CornersSquare {
		cornerSel = 1
	}
	corners := widgets.NewRadioGroup([]string{"Round corners", "Square corners"}, cornerSel, func(i int) {
		next := staged
		if i == 1 {
			next.Corners = style.CornersSquare
		} else {
			next.Corners = style.CornersRound
		}
		preview(next)
	})

	iconSel := 0
	if staged.Icons == style.IconSetSharp {
		iconSel = 1
	}
	icons := widgets.NewRadioGroup([]string{"Classic (rounded stroke)", "Sharp (geometric)"}, iconSel, func(i int) {
		next := staged
		if i == 1 {
			next.Icons = style.IconSetSharp
		} else {
			next.Icons = style.IconSetClassic
		}
		preview(next)
	})

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

	themeCol := widgets.NewColumn(widgets.NewLabel("Theme"), theme).WithGap(4)
	cornerCol := widgets.NewColumn(widgets.NewLabel("Corners"), corners).WithGap(4)
	iconCol := widgets.NewColumn(widgets.NewLabel("Icon set"), icons).WithGap(4)
	choices := widgets.NewRow(themeCol, cornerCol, iconCol).WithGap(24)
	choices.AddFlex(themeCol, 1)
	choices.AddFlex(cornerCol, 1)
	choices.AddFlex(iconCol, 1)
	appearance := widgets.NewColumn(
		widgets.NewTitle("Appearance"),
		widgets.NewLabel("Shared by Mail, gallery, and any app that calls PreferredLook()."),
		choices,
		previewPane,
	).WithGap(10).WithPad(4)

	aboutPath := widgets.NewMonoTextView(style.AppearancePath(), "")
	aboutPath.MinRows = 2
	aboutPath.Wrap = true
	about := widgets.NewColumn(
		widgets.NewTitle("About"),
		widgets.NewLabel("uitoolkit v"+uitoolkit.Version),
		widgets.NewLabel("LookAndFeel Classic — Titillium Web + JetBrains Mono."),
		widgets.NewLabel("Prefs file (written on Apply):"),
		aboutPath,
		widgets.NewLabel("Other apps watch this file and call SetLook(PreferredLook())."),
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
	hint := widgets.NewLabel("Apply writes look.json. Close without Apply discards staged changes.")
	actions := widgets.NewRow(applyBtn, hint).WithGap(12).WithPad(8)

	root := widgets.NewColumn(chrome, split, actions, status)
	root.AddFlex(split, 1)
	return root
}
