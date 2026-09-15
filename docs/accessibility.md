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
| platform adapter | AT-SPI2 on Linux (next), then UI Automation and NSAccessibility |

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

## Still to come
- **The AT-SPI2 bridge.** It exports the tree on the accessibility bus
  (`org.a11y.Bus`), registers the app, answers the Accessible, Component,
  Action, Value, Text and Selection interfaces, and emits focus, state and
  property change events. It stays off until `org.a11y.Status` says a screen
  reader is on.
- **Text interfaces for fields.** The caret and selection are in the model
  already; exposing words and lines is still to do.
- **Windows and macOS adapters.**
