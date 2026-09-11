package demo

import (
	"fmt"
	"sort"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

type prefRow struct {
	Key, Value, Scope string
}

// InspectorApp is a small preferences inspector: table, toolbar, tabs, message box.
func InspectorApp(win *app.Window) widget.Component {
	prefs := []prefRow{
		{"theme", "dark", "user"},
		{"ui.scale", "1.00", "user"},
		{"editor.wrap", "true", "project"},
		{"notify.desktop", "true", "user"},
		{"font.size", "16", "user"},
		{"autosave", "true", "project"},
		{"restore.session", "false", "user"},
		{"editor.minimap", "false", "project"},
	}
	sel := 0

	status := widgets.NewStatusBar("Ready.", prefs[0].Key, "v"+uitoolkit.Version)
	mark := func(msg string) {
		status.Set(0, msg)
	}

	var table *widgets.TableView
	refresh := func() {
		table.RowCount = len(prefs)
		if sel >= len(prefs) {
			sel = len(prefs) - 1
		}
		if sel < 0 {
			sel = 0
		}
		table.Selected = sel
		table.Invalidate()
		if sel >= 0 && sel < len(prefs) {
			status.Set(1, prefs[sel].Key)
		}
	}

	table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "Key", Sortable: true},
		{Title: "Value", Width: 90, Sortable: true},
		{Title: "Scope", Width: 80, Sortable: true},
	}, len(prefs), func(row, col int) string {
		if row < 0 || row >= len(prefs) {
			return ""
		}
		switch col {
		case 1:
			return prefs[row].Value
		case 2:
			return prefs[row].Scope
		default:
			return prefs[row].Key
		}
	}, func(i int) {
		if i >= 0 && i < len(prefs) {
			sel = i
			status.Set(1, prefs[i].Key)
			status.Set(0, "Inspect "+prefs[i].Key)
		}
	})
	table.Selected = 0
	table.Mono = true
	table.OnSort = func(col int, asc bool) {
		sort.SliceStable(prefs, func(i, j int) bool {
			var less bool
			switch col {
			case 1:
				less = prefs[i].Value < prefs[j].Value
			case 2:
				less = prefs[i].Scope < prefs[j].Scope
			default:
				less = prefs[i].Key < prefs[j].Key
			}
			if !asc {
				return !less
			}
			return less
		})
		refresh()
	}

	theme := widgets.NewComboBox([]string{"Dark graphite", "Light paper", "High contrast"}, 0, func(i int) {
		names := []string{"dark", "light", "contrast"}
		if i >= 0 && i < len(names) {
			setPref(prefs, "theme", names[i])
			mark("Theme: " + names[i])
		}
	})
	dark := widgets.NewSwitch("Dark chrome", true, func(v bool) {
		if v {
			setPref(prefs, "theme", "dark")
		} else {
			setPref(prefs, "theme", "light")
		}
		mark("Dark chrome")
		refresh()
	})
	compact := widgets.NewSwitch("Compact metrics", false, func(bool) { mark("Compact metrics") })
	wrap := widgets.NewSwitch("Wrap editor", true, func(v bool) {
		setPref(prefs, "editor.wrap", boolStr(v))
		mark("Wrap editor")
		refresh()
	})
	scaleLbl := widgets.NewLabel("UI scale  100%")
	scale := widgets.NewSlider(50, 200, 100, func(v float32) {
		scaleLbl.SetText(fmt.Sprintf("UI scale  %d%%", int(v+0.5)))
		setPref(prefs, "ui.scale", fmt.Sprintf("%.2f", float64(v)/100))
		mark("UI scale")
		refresh()
	})
	font := widgets.NewNumberField(10, 28, 16, 1, func(v float64) {
		setPref(prefs, "font.size", fmt.Sprintf("%d", int(v)))
		mark("Font size")
		refresh()
	})
	font.Tip = "UI font size"

	appearance := widgets.NewExpander("Appearance", true, widgets.NewColumn(
		widgets.NewLabel("Theme"),
		theme,
		dark,
		compact,
		scaleLbl,
		scale,
		widgets.NewSeparator(),
		widgets.NewLabel("Font size"),
		font,
	).WithGap(6))
	session := widgets.NewExpander("Session", false, widgets.NewColumn(
		widgets.NewSwitch("Autosave", true, func(v bool) {
			setPref(prefs, "autosave", boolStr(v))
			mark("Autosave")
			refresh()
		}),
		widgets.NewSwitch("Restore last session", false, func(v bool) {
			setPref(prefs, "restore.session", boolStr(v))
			mark("Restore session")
			refresh()
		}),
		widgets.NewSwitch("Desktop notifications", true, func(v bool) {
			setPref(prefs, "notify.desktop", boolStr(v))
			mark("Notifications")
			refresh()
		}),
	).WithGap(6))
	acc := widgets.NewAccordion(true, appearance, session)

	general := widgets.NewScrollView(widgets.NewColumn(
		widgets.NewLabel("Project preferences — accordion groups, switches, sliders."),
		acc,
		widgets.NewSeparator(),
		widgets.NewSpacerSize(0, 8),
	).WithGap(10).WithPad(8))

	notes := widgets.NewTextArea(
		"// inspector — JetBrains Mono\noverride wrap = true\nfont.size = 16\n",
		"Notes",
		func(string) { mark("Notes edited") },
	)
	notes.Mono = true
	notes.MinRows = 5
	editorBody := widgets.NewColumn(
		table,
		widgets.NewSeparator(),
		widgets.NewRow(widgets.NewLabel("Editor notes"), wrap).WithGap(12),
		notes,
	).WithGap(8).WithPad(8)
	editorBody.AddFlex(table, 1)
	editorBody.AddFlex(notes, 1)
	editor := widgets.NewPad(0, editorBody)

	about := widgets.NewColumn(
		widgets.NewTitle("Inspector"),
		widgets.NewLabel("A second sample app on uitoolkit v"+uitoolkit.Version+"."),
		widgets.NewLabel("Table + ToolBar + Tabs + MessageBox, plus Switch / TextArea / Accordion."),
		widgets.NewSeparator(),
		widgets.NewButton("License…", func() {
			widgets.Info(win.Content(), "License",
				"MIT. Paints only with paintengine2d. UI: Titillium Web.", nil)
		}),
		widgets.NewSpacer(),
	).WithGap(10).WithPad(16)

	tabs := widgets.NewTabView(
		widgets.Tab{Title: "General", Content: general},
		widgets.Tab{Title: "Editor", Content: editor},
		widgets.Tab{Title: "About", Content: about},
	)
	tabs.OnChange = func(i int) {
		names := []string{"General", "Editor", "About"}
		if i >= 0 && i < len(names) {
			status.Set(0, "Tab: "+names[i])
		}
	}

	snapshot := append([]prefRow(nil), prefs...)
	defaults := append([]prefRow(nil), prefs...)

	apply := widgets.ToolText("Apply", func() {
		snapshot = append([]prefRow(nil), prefs...)
		widgets.Info(win.Content(), "Applied",
			"Preferences written for this session.", func() { status.Set(0, "Applied") })
	})
	apply.Tip = "Keep the current values"
	revert := widgets.ToolText("Revert", func() {
		widgets.Confirm(win.Content(), "Revert changes?",
			"Restore the last applied preference snapshot.",
			func(yes bool) {
				if !yes {
					status.Set(0, "Revert cancelled")
					return
				}
				prefs = append([]prefRow(nil), snapshot...)
				refresh()
				status.Set(0, "Reverted")
			})
	})
	revert.Tip = "Restore last apply"
	reset := widgets.ToolText("Defaults", func() {
		widgets.ShowMessageBox(win.Content(), widgets.MessageBoxOptions{
			Title:   "Reset to defaults?",
			Message: "Replace every preference with the factory snapshot.",
			Kind:    widgets.MessageWarning,
			Buttons: widgets.ButtonsYesNo,
			OnResult: func(r widgets.MessageResult) {
				if r != widgets.ResultYes {
					status.Set(0, "Defaults cancelled")
					return
				}
				prefs = append([]prefRow(nil), defaults...)
				refresh()
				status.Set(0, "Defaults restored")
			},
		})
	})
	reset.Tip = "Factory snapshot"
	aboutTool := widgets.ToolText("About", func() { tabs.Select(2) })
	aboutTool.Tip = "About Inspector"
	tools := widgets.NewToolBar(
		apply, revert, widgets.ToolDivider(), reset, widgets.ToolDivider(), aboutTool,
	)

	chrome := widgets.NewTitleBar("Inspector", "project preferences  ·  v"+uitoolkit.Version)
	root := widgets.NewColumn(tools, chrome, tabs, status).WithGap(0)
	root.AddFlex(tabs, 1)
	return root
}

func setPref(prefs []prefRow, key, value string) {
	for i := range prefs {
		if prefs[i].Key == key {
			prefs[i].Value = value
			return
		}
	}
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
