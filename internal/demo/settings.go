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
// Changes apply live via Application.SetLook and persist to XDG look.json.
func SettingsApp(a *app.Application, win *app.Window) widget.Component {
	return buildSettings(a, win, style.LoadAppearance(), 0)
}

func buildSettings(a *app.Application, win *app.Window, ap style.Appearance, section int) widget.Component {
	ap = ap.Normalize()
	if section < 0 || section > 1 {
		section = 0
	}

	status := widgets.NewStatusBar(ap.String(), style.AppearancePath(), "v"+uitoolkit.Version)
	chrome := widgets.NewTitleBar("Settings", "Appearance for every uitoolkit app")

	apply := func(next style.Appearance) {
		next = next.Normalize()
		if err := style.SaveAppearance(next); err != nil {
			status.Set(0, err.Error())
			return
		}
		a.SetLook(next.Look())
		win.SetContent(buildSettings(a, win, next, section))
	}

	themeSel := 0
	if ap.Theme == style.ThemeLight {
		themeSel = 1
	}
	theme := widgets.NewRadioGroup([]string{"Dark graphite", "Light paper"}, themeSel, func(i int) {
		next := ap
		if i == 1 {
			next.Theme = style.ThemeLight
		} else {
			next.Theme = style.ThemeDark
		}
		apply(next)
	})

	cornerSel := 0
	if ap.Corners == style.CornersSquare {
		cornerSel = 1
	}
	corners := widgets.NewRadioGroup([]string{"Round corners", "Square corners"}, cornerSel, func(i int) {
		next := ap
		if i == 1 {
			next.Corners = style.CornersSquare
		} else {
			next.Corners = style.CornersRound
		}
		apply(next)
	})

	iconSel := 0
	if ap.Icons == style.IconSetSharp {
		iconSel = 1
	}
	icons := widgets.NewRadioGroup([]string{"Classic (rounded stroke)", "Sharp (geometric)"}, iconSel, func(i int) {
		next := ap
		if i == 1 {
			next.Icons = style.IconSetSharp
		} else {
			next.Icons = style.IconSetClassic
		}
		apply(next)
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

	preview := widgets.NewPanel("Live preview",
		widgets.NewRow(primary, secondary, disabled).WithGap(10),
		toolbar,
		widgets.NewRow(field, combo).WithGap(10),
		widgets.NewRow(check, sw).WithGap(16),
		slider,
	)

	menu := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemAccel("&Save prefs", "Ctrl+S", func() {
				if err := style.SaveAppearance(ap); err != nil {
					status.Set(0, err.Error())
					return
				}
				status.Set(0, "Wrote "+style.AppearancePath())
			}),
			widgets.Sep(),
			widgets.ItemAccel("&Quit", "Ctrl+Q", func() { a.Quit() }),
		),
		widgets.NewMenu("&View",
			widgets.CheckItem("&Dark", ap.Theme == style.ThemeDark, func() {
				next := ap
				next.Theme = style.ThemeDark
				apply(next)
			}),
			widgets.CheckItem("&Light", ap.Theme == style.ThemeLight, func() {
				next := ap
				next.Theme = style.ThemeLight
				apply(next)
			}),
			widgets.Sep(),
			widgets.CheckItem("&Round", ap.Corners == style.CornersRound, func() {
				next := ap
				next.Corners = style.CornersRound
				apply(next)
			}),
			widgets.CheckItem("S&quare", ap.Corners == style.CornersSquare, func() {
				next := ap
				next.Corners = style.CornersSquare
				apply(next)
			}),
			widgets.Sep(),
			widgets.CheckItem("&Classic icons", ap.Icons == style.IconSetClassic, func() {
				next := ap
				next.Icons = style.IconSetClassic
				apply(next)
			}),
			widgets.CheckItem("S&harp icons", ap.Icons == style.IconSetSharp, func() {
				next := ap
				next.Icons = style.IconSetSharp
				apply(next)
			}),
		),
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
		preview,
	).WithGap(10).WithPad(4)

	aboutPath := widgets.NewMonoTextView(style.AppearancePath(), "")
	aboutPath.MinRows = 2
	aboutPath.Wrap = true
	about := widgets.NewColumn(
		widgets.NewTitle("About"),
		widgets.NewLabel("uitoolkit v"+uitoolkit.Version),
		widgets.NewLabel("LookAndFeel Classic — Titillium Web + JetBrains Mono."),
		widgets.NewLabel("Prefs file:"),
		aboutPath,
		widgets.NewLabel("Other apps apply the same skin with PreferredLook()."),
		widgets.NewButton("Open appearance", func() {
			win.SetContent(buildSettings(a, win, ap, 0))
		}),
	).WithGap(10).WithPad(4)

	sections := []string{"Appearance", "About"}
	nav := widgets.NewListView(len(sections), func(i int) string { return sections[i] }, func(i int) {
		win.SetContent(buildSettings(a, win, ap, i))
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

	root := widgets.NewColumn(chrome, menu, split, status)
	root.AddFlex(split, 1)
	return root
}
