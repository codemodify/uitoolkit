# Accessibility

uitoolkit describes every window as a tree of accessible objects that
assistive technology can read: screen readers, magnifiers, voice control
and test tools. The model lives in package `a11y`. It follows AccessKit's
design: widgets fill in platform-neutral nodes, and a platform adapter
hands them to the system. The tree is built only when something asks for
it, so an app pays nothing while no assistive technology is running.

| Layer | What it does |
| --- | --- |
| `a11y` | the model: `Node` (role, name, description, value, state, bounds, range, position in set, level, caret, shortcut, actions, children), `Check` |
| `widget` | `Accessible` (`Describe(*a11y.Node)`), `AccessibleItems` for views whose items are not components, `AccessibleActor`, `SetAccessibleName`, `AccessibleTree` |
| `widgets` | every stock widget describes itself: buttons, check boxes, radios, switches, fields, spin buttons, sliders, progress bars, combo and date fields, lists, trees, tables (headers, rows, cells), card lists, tabs, segmented controls, tool bars, menu bars and menus, status bars, group boxes, dialogs, message boxes |
| `app` | `Window.AccessibleTree()` (content, then any dialog, menu or tooltip over it), `Window.AccessibleAction(id, action)` |
| platform adapter | AT-SPI2 on Linux (`app/atspi_linux.go`); UI Automation and NSAccessibility later |

## Naming controls

A screen reader announces a control by its name. Stock widgets name
themselves from their text. A field in a `Form` row takes the row's label,
as Qt's buddy labels do. A tool button with only an icon takes its tooltip,
or else the action its icon stands for ("Save").

Name everything else yourself: an icon-only button, a field whose label is
a separate widget, a slider next to a caption.

```go
volume := widgets.NewSlider(0, 100, 60, nil)
volume.SetAccessibleName("Volume")
more := widgets.NewButton("…", openMenu)
more.SetAccessibleName("More options")
```

Name the views too (`tree.SetAccessibleName("Folders")`), so a screen
reader says "Folders, tree" rather than just "tree".

## Checking an app

`a11y.Check(tree)` works like an accessibility linter. It reports:
- controls without a name;
- duplicate IDs;
- more than one focused node;
- ranges that run backwards;
- visible controls without a box.

Call it in your tests:

```go
for _, p := range a11y.Check(win.AccessibleTree()) {
	t.Error(p)
}
```

uitoolkit's own tests run it over every page of the gallery, over Settings
and over Mail.

## Actions

Assistive technology can act through the tree:
- `ActionDefault` presses a button, toggles a check box or switch, selects a
  list, tree, table or tab item, runs a tool, or opens a menu;
- `ActionExpand` and `ActionCollapse` open and close tree branches;
- `ActionScrollIntoView` scrolls an item into view;
- `ActionFocus` moves the keyboard focus.

Widgets implement `widget.AccessibleActor`, and
`Window.AccessibleAction(id, a)` routes an action to the right one.

## The Linux adapter (AT-SPI2)

Screen readers on Linux (Orca), accerciser and test tools read apps
through AT-SPI2. uitoolkit's bridge works like Qt's: it stays off until
`org.a11y.Status` says assistive technology is on, which the desktop sets
when a screen reader starts. Then it:
- connects to the accessibility bus and has the registry embed the app
  under the desktop;
- exports every node at `/org/a11y/atspi/accessible/<id>` with the
  Accessible, Application, Component, Action, Value, Text and EditableText
  interfaces (fields take text from assistive technology and automation
  tools such as dogtail);
- announces the active window (`window:activate`) and focus moves
  (`object:state-changed:focused`), including a view's current row, tab or
  tool;
- announces changes on the focused object: `object:state-changed:checked`
  (and `selected`, `expanded`, `pressed`, `indeterminate`, `sensitive`),
  plus name and value changes, and for fields `object:text-changed:insert`
  / `delete` (only what changed) and `object:text-caret-moved`. Only that
  object is described each frame, so this stays cheap.

Queries read a snapshot of the trees that the UI goroutine rebuilds when
something has changed since the last query. Actions (DoAction, GrabFocus,
ScrollTo) run on the UI goroutine.

`UITK_A11Y=1` turns the bridge on regardless of the desktop (a screen
reader started by hand, tests), and `UITK_A11Y=0` keeps it off.
`AT_SPI_BUS_ADDRESS` names the accessibility bus directly, as libatspi
allows.

**Testing it:** `tools/a11y/smoke.sh` starts a private D-Bus session with
its own accessibility bus and registry and runs the gallery headless with
the bridge on. `smoke.py` then reads and drives the gallery through
libatspi, as a screen reader would: roles, names, states, boxes, the
slider's value, the entry's text, toggling a check box and selecting a tab
through their actions, and the focus event when the entry takes focus.
Nothing touches the desktop's session. It needs at-spi2-core and
python-gobject.

## Still to come
- Changes on objects without the focus (a status line updating).
- Relations (a label that labels a field), the Selection and Table
  interfaces, and items built lazily for very long lists.
- Windows and macOS adapters.
