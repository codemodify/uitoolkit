// Package inspectorapp is the Inspector sample: a preferences table in
// the centre with an outline, a properties panel and a log docked around
// it. Every panel drags to another side, stacks as a tab, collapses,
// floats in a window of its own and closes — and the arrangement is
// remembered between runs.
//
// It is the pilot for the dock package, so it uses the parts an
// application is meant to use: [dock.Host], [dock.Host.SaveLayoutFile]
// and [dock.Host.LoadLayoutFile] for the remembered arrangement, and
// [dock.Host.ResetLayout] to put everything back.
//
// examples/inspector opens the window and calls [InspectorApp].
package inspectorapp

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/dock"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// LayoutName is the name the app saves its dock arrangement under; see
// [dock.LayoutFile].
const LayoutName = "inspector"

type prefRow struct {
	Key, Value, Scope string
}

// InspectorApp is a docking preferences inspector: the table of settings
// in the centre, an outline, a properties panel and a log docked around
// it. Every panel can be dragged to another side, stacked as a tab on
// another, collapsed, floated in a window of its own and closed — and the
// arrangement is remembered between runs.
//
// It is the pilot for the dock package, so it exercises the parts an app
// is meant to use: dock.Host, a saved layout, and a reset back to the
// default.
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
	scope := "" // the outline's filter: "", "user" or "project"

	status := widgets.NewStatusBar("Ready.", prefs[0].Key, "v"+uitoolkit.Version)

	logLines := []string{"inspector started"}
	logView := widgets.NewMonoTextView(strings.Join(logLines, "\n"), "Nothing yet")
	logView.MinRows = 3
	note := func(msg string) {
		status.Set(0, msg)
		logLines = append(logLines, time.Now().Format("15:04:05")+"  "+msg)
		if len(logLines) > 200 {
			logLines = logLines[len(logLines)-200:]
		}
		logView.SetText(strings.Join(logLines, "\n"))
	}

	// ---- the centre: the preference table --------------------------------

	var table *widgets.TableView
	shown := func() []prefRow {
		if scope == "" {
			return prefs
		}
		out := make([]prefRow, 0, len(prefs))
		for _, p := range prefs {
			if p.Scope == scope {
				out = append(out, p)
			}
		}
		return out
	}
	refresh := func() {
		rows := shown()
		table.RowCount = len(rows)
		if sel >= len(rows) {
			sel = len(rows) - 1
		}
		if sel < 0 {
			sel = 0
		}
		table.Selected = sel
		table.Invalidate()
		if sel >= 0 && sel < len(rows) {
			status.Set(1, rows[sel].Key)
		}
	}

	table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "Key", Sortable: true},
		{Title: "Value", Width: 90, Sortable: true},
		{Title: "Scope", Width: 80, Sortable: true},
	}, len(prefs), func(row, col int) string {
		rows := shown()
		if row < 0 || row >= len(rows) {
			return ""
		}
		switch col {
		case 1:
			return rows[row].Value
		case 2:
			return rows[row].Scope
		default:
			return rows[row].Key
		}
	}, func(i int) {
		rows := shown()
		if i >= 0 && i < len(rows) {
			sel = i
			status.Set(1, rows[i].Key)
			note("Inspect " + rows[i].Key)
		}
	})
	table.SetAccessibleName("Preferences")
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

	// ---- the left panel: the outline -------------------------------------

	all := widgets.NewTreeNode("All preferences")
	user := widgets.NewTreeNode("User")
	project := widgets.NewTreeNode("Project")
	outline := widgets.NewTreeView(all, user, project)
	outline.SetAccessibleName("Scopes")
	outline.Selected = all
	outline.Frameless = true
	outline.Sidebar = true
	outline.OnSelect = func(n *widgets.TreeNode) {
		switch n {
		case user:
			scope = "user"
		case project:
			scope = "project"
		default:
			scope = ""
		}
		sel = 0
		refresh()
		note("Outline: " + n.Label)
	}

	// ---- the right panel: the editors ------------------------------------

	theme := widgets.NewComboBox([]string{"Dark graphite", "Light paper", "High contrast"}, 0, func(i int) {
		names := []string{"dark", "light", "contrast"}
		if i >= 0 && i < len(names) {
			setPref(prefs, "theme", names[i])
			note("Theme: " + names[i])
			refresh()
		}
	})
	dark := widgets.NewSwitch("Dark chrome", true, func(v bool) {
		if v {
			setPref(prefs, "theme", "dark")
		} else {
			setPref(prefs, "theme", "light")
		}
		note("Dark chrome")
		refresh()
	})
	wrap := widgets.NewSwitch("Wrap editor", true, func(v bool) {
		setPref(prefs, "editor.wrap", boolStr(v))
		note("Wrap editor")
		refresh()
	})
	scaleLbl := widgets.NewLabel("UI scale  100%")
	scale := widgets.NewSlider(50, 200, 100, func(v float32) {
		scaleLbl.SetText(fmt.Sprintf("UI scale  %d%%", int(v+0.5)))
		setPref(prefs, "ui.scale", fmt.Sprintf("%.2f", float64(v)/100))
		note("UI scale")
		refresh()
	})
	scale.SetAccessibleName("Scale")
	font := widgets.NewNumberField(10, 28, 16, 1, func(v float64) {
		setPref(prefs, "font.size", fmt.Sprintf("%d", int(v)))
		note("Font size")
		refresh()
	})
	font.Tip = "UI font size"

	appearance := widgets.NewExpander("Appearance", true, widgets.NewColumn(
		widgets.NewLabel("Theme").For(theme),
		theme,
		dark,
		wrap,
		scaleLbl,
		scale,
		widgets.NewSeparator(),
		widgets.NewLabel("Font size").For(font),
		font,
	).WithGap(6))
	session := widgets.NewExpander("Session", false, widgets.NewColumn(
		widgets.NewSwitch("Autosave", true, func(v bool) {
			setPref(prefs, "autosave", boolStr(v))
			note("Autosave")
			refresh()
		}),
		widgets.NewSwitch("Restore last session", false, func(v bool) {
			setPref(prefs, "restore.session", boolStr(v))
			note("Restore session")
			refresh()
		}),
		widgets.NewSwitch("Desktop notifications", true, func(v bool) {
			setPref(prefs, "notify.desktop", boolStr(v))
			note("Notifications")
			refresh()
		}),
	).WithGap(6))
	editors := widgets.NewScrollView(widgets.NewColumn(
		widgets.NewAccordion(true, appearance, session),
	).WithGap(8).WithPad(8))

	// ---- the dock host ---------------------------------------------------

	host := dock.NewHost(table)
	outlinePanel := dock.NewPanel("outline", "Outline", outline)
	propsPanel := dock.NewPanel("properties", "Properties", editors)
	logPanel := dock.NewPanel("log", "Log", widgets.NewPad(4, logView))
	outlinePanel.SetMinSize(140, 80)
	propsPanel.SetMinSize(200, 120)
	logPanel.SetMinSize(160, 60)
	host.Dock(outlinePanel, dock.SideLeft)
	host.Dock(propsPanel, dock.SideRight)
	host.Dock(logPanel, dock.SideBottom)
	// The arrangement above is what "Reset layout" goes back to, so record
	// it before any saved one is read over the top.
	host.SetDefaultLayout()
	app.DockHost(win, host)

	// ---- the tool bar ----------------------------------------------------

	panels := []*dock.Panel{outlinePanel, propsPanel, logPanel}
	toggles := make([]*widgets.ToolItem, len(panels))
	var tools *widgets.ToolBar
	syncToggles := func() {
		for i, p := range panels {
			toggles[i].Down = !p.Closed()
		}
		if tools != nil {
			tools.Invalidate()
		}
	}
	for i, p := range panels {
		p := p
		toggles[i] = widgets.ToolToggle(p.Title(), true, func() {
			if p.Closed() {
				p.Show()
				note(p.Title() + " shown")
			} else {
				p.Close()
				note(p.Title() + " hidden")
			}
			syncToggles()
		})
		toggles[i].Tip = "Show or hide the " + p.Title() + " panel"
		p.OnShown = func(bool) { syncToggles() }
	}

	snapshot := append([]prefRow(nil), prefs...)
	defaults := append([]prefRow(nil), prefs...)

	apply := widgets.ToolText("Apply", func() {
		snapshot = append([]prefRow(nil), prefs...)
		widgets.Info(win.Content(), "Applied",
			"Preferences written for this session.", func() { note("Applied") })
	})
	apply.Tip = "Keep the current values"
	revert := widgets.ToolText("Revert", func() {
		widgets.Confirm(win.Content(), "Revert changes?",
			"Restore the last applied preference snapshot.",
			func(yes bool) {
				if !yes {
					note("Revert cancelled")
					return
				}
				prefs = append([]prefRow(nil), snapshot...)
				refresh()
				note("Reverted")
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
					note("Defaults cancelled")
					return
				}
				prefs = append([]prefRow(nil), defaults...)
				refresh()
				note("Defaults restored")
			},
		})
	})
	reset.Tip = "Factory snapshot"
	resetLayout := widgets.ToolText("Reset layout", func() {
		host.ResetLayout()
		syncToggles()
		note("Layout reset")
	})
	resetLayout.Tip = "Put every panel back where it started"

	items := []*widgets.ToolItem{apply, revert, widgets.ToolDivider(), reset, widgets.ToolDivider()}
	items = append(items, toggles...)
	items = append(items, widgets.ToolDivider(), resetLayout)
	tools = widgets.NewToolBar(items...)

	// ---- the layout an app remembers -------------------------------------

	// dock keeps the arrangement in a file of the app's own under
	// $XDG_CONFIG_HOME; see dock.LayoutFile.
	if ok, err := host.LoadLayoutFile(LayoutName); !ok && !errors.Is(err, fs.ErrNotExist) {
		// A layout from another version, or a broken file: the default
		// arrangement stands and the app says so rather than coming up
		// wrong.
		note("The saved layout could not be read; using the default")
	}
	syncToggles()
	host.OnLayoutChanged = func() {
		_ = host.SaveLayoutFile(LayoutName)
		syncToggles()
	}

	chrome := widgets.NewTitleBar("Inspector", "project preferences  ·  v"+uitoolkit.Version)
	root := widgets.NewColumn(tools, chrome, host, status).WithGap(0)
	root.AddFlex(host, 1)
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
