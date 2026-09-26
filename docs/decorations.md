# Window decorations

A uitoolkit window's frame — its title bar, caption buttons and resize
borders — is drawn either by the desktop (KWin's Breeze frame, an X11 window
manager's) or by uitoolkit itself, the way Chromium, SourceGit, GTK 4 and
Electron apps draw theirs: the app's own title bar with tabs or a tool bar in
it, the caption buttons at the sides the desktop puts them, resize edges
around it. Either way the desktop keeps window management: every move,
resize, snap and the window menu is handed back to the compositor or window
manager from the user's button press, never emulated.

Phases 1 to 4 of [the design](#phases) are done: the toolkit-drawn frame
painted by the look (Windows 95's navy caption and bevelled buttons, XP's
blue one, Aqua's traffic lights, libadwaita's round buttons, SourceGit's
flat 48×30 cells), with tabs or a tool bar in the title bar, shadows and
rounded corners, and the [tear-off](#tear-off) drag — a tab dragged out of
its strip, or a dock panel dragged out of its host, becomes a window the
desktop carries under the pointer.

The work beside the frame is done too: under the desktop's own frame a
window [dresses it](#dressing-the-desktops-frame) in its look's colours
(KWin's server-decoration palette), every window can carry an icon
(`xdg-toplevel-icon-v1`, `_NET_WM_ICON`), and X11 resizes keep in step with
the client (`_NET_WM_SYNC_REQUEST`). What is [still open](#still-open) is
the Windows and macOS mappings.

## For apps: a title bar

Give the window a title bar with `SetTitleBar`:

```go
hb := widgets.NewHeaderBar(
	[]widget.Component{menuButton, fetchWriteToolBar}, // leading items
	nil,                                               // centre: free space that moves the window
	[]widget.Component{searchField, filterToolBar})    // trailing items
win.SetTitleBar(hb)
win.SetContent(body)
```

- **Where uitoolkit draws the frame** the header bar is the window's caption:
  the caption buttons (minimize, maximize / restore, close, the window menu)
  are added at the sides the desktop puts them, a hairline border runs around
  the window, the content sits below.
- **Under the desktop's frame** the header bar is the window's first row,
  laid out exactly as the equivalent `widgets.NewRow(…)` — no caption
  buttons, same pixels.

`NewHeaderBar(start, center, end)` lays its items out in one row (8 px apart,
centred vertically); `center` takes the free width (tabs, a search field) or,
when nil, the free space is caption. `ShowTitle` paints the window title in
that free space. `OnContextMenu` replaces the window menu on a right-click in
caption space (Chromium's tab-strip menu). Any other component passed to
`SetTitleBar` (a `TitleBar`, a row) becomes the centre of a header bar.
`SetTitleBar(nil)` goes back to no title bar.

The pilot is [the mail client](https://github.com/codemodify/comms-mail): its Thunderbird-style row
(the M menu, Fetch / Write, free space, the quick filter and its tool bar)
*is* its title bar.

A SourceGit- or Chromium-style window puts document tabs in the centre
([tabs in the title bar](#tabs-in-the-title-bar)):

```go
tabs := widgets.NewBrowserTabs("uitoolkit", "paintengine2d")
tabs.OnNew = func() { tabs.Select(tabs.AddTab(widgets.BrowserTab{Title: "New"})) }
win.SetTitleBar(widgets.NewHeaderBar([]widget.Component{menuBar}, tabs, nil))
```

How the header bar and the frame meet is the look's: a **merged** frame
makes the header bar the caption, one row with the caption buttons at its
sides (GTK's header bars, Windows 10 and 11, macOS, Chromium, SourceGit); a
**stacked** frame keeps the era's own caption — the window title and the
caption buttons in the era's strip — and puts the header bar in a row under
it, the way the tool bars of Windows 95, XP, the classic Mac and Motif sat
under their title bars. Either way the header bar's free space moves the
window, and in a merged caption a menu bar, tool bar or tab strip paints no
bar of its own: the caption band is their background.

### What moves the window

Title-bar space is caption where every component from the one under the
pointer up to the title bar says so (`widget.CaptionHitTester`), so a label
inside a button is never caption:

| Component | Caption where |
| --- | --- |
| `HeaderBar`, `FlexBox` rows / columns, `Stack`, `Pad`, `Spacer`, `Label`, `Separator`, `Picture`, `TitleBar` | everywhere (their children decide for themselves) |
| `ToolBar` | between buttons and on dividers |
| `TabBar` | after the last tab |
| `BrowserTabs` | everywhere but its tabs, their close buttons, "+" and the scroll arrows |
| `MenuBar` | after the last title |
| `WindowControls` | gaps between caption buttons |
| anything else | nowhere: it is a control |

`widgets.DragArea(c)` makes all of `c` caption (its content gets no pointer
input — a logo, a decorative title), `widgets.NoDrag(c)` none of it: the
equivalents of Electron's `app-region: drag` / `no-drag`. A custom widget
implements `CaptionAt(p paintengine2d.Point) bool` to take part.

### Gestures

| Gesture on caption space | Result |
| --- | --- |
| press and drag past the desktop's drag threshold (8 px; KDE 10) | the desktop moves the window (`xdg_toplevel.move`, `_NET_WM_MOVERESIZE`) |
| click | nothing — clicks and double-clicks keep working because the move waits for the threshold |
| double-click (within the desktop's double-click time, both on caption) | the desktop's double-click action, default toggle maximize |
| middle click | the desktop's middle-click action, default none |
| right click | the menu of the component under the pointer (`widget.CaptionMenuer`: a tab strip's), else the header bar's own (`OnContextMenu`), else the desktop's window menu, else the toolkit's (Restore / Maximize, Minimize, Close) |
| press on a resize edge or corner | the desktop resizes from there (on the press) |
| right click on a caption button | the window menu |

After the desktop takes over a move or resize, the window drops its pointer
capture and hover: the release goes to the desktop. Apps can start the same
gestures from their own press handlers: `Window.StartMove()`,
`StartResize(edges)`, `ShowWindowMenu(p)`; and `Minimize()`,
`ToggleMaximize()`, `RequestClose()` (the close button's request: an app that
hides on close keeps its window).

`Window.NonClientHit(p)` answers what a point is — client, caption, a resize
edge, a caption button — from the last layout, the way Win32's
`WM_NCHITTEST` wants it.

### Frame layout

- The look's border (`DecorationSpec.Border`, whole device pixels): a
  hairline in the plain and modern looks, Windows 95's 4 px raised frame,
  XP's five lines, Windows 7's glass, Motif's resize handles; none when
  maximized (the era's frames went off the screen).
- The caption inside it: a merged frame's header bar, at least as tall as
  the look's caption and its buttons; or a stacked frame's strip (the look's
  caption height) with the header bar's row under it when the app gave the
  header bar items of its own.
- A **fitted caption** (`DecorationSpec.CaptionFits`) is the exception: the
  band is only as wide as its own contents — the buttons at its two ends and
  the title between them (`HeaderBar.CaptionFitWidth`) — instead of as wide
  as the window. It is BeOS's tab, and it is half of a frame whose window is
  not a rectangle: the other half is the look's silhouette
  ([`WindowShapeEngine`](shapes.md#for-looks)), which leaves the rest of the
  top edge to the desktop. A band narrower than its window is a gap unless
  the window really stops there, so `DecorationOf` drops the fitted caption
  under exactly the rules that drop the silhouette — maximized and tiled —
  and the two are never out of step.
- Resize band: the border, at least 4 px, inside the left, right and bottom
  edges and at the top of the caption, with 16 px corner zones; none while
  maximized or full screen, and none on tiled edges (a quick-tiled window
  resizes only from its free sides). The band wins over the caption buttons
  of a restored window; a maximized window's buttons reach the screen's
  edge: each button's hit box runs up to the caption's top and the outermost
  one out to the window's side.
- Full screen: no frame and no caption buttons; the header bar stays the top
  row.
- A modal dialog dims the content, not the caption: the window can still be
  moved, minimized and closed, while the header bar's own controls are inert.
- A look with a drop shadow puts the window inside a larger surface: see
  [the shadow's margin](#the-shadows-margin).

Caption buttons are the look's (see [the looks](#the-looks)): sized and
spaced by the look, hover and pressed states, the restore glyph while
maximized, the backdrop look while the window is inactive. They are hidden
for actions the desktop cannot do (`wm_capabilities`,
`_NET_WM_ALLOWED_ACTIONS`), out of the Tab order, and named Minimize,
Maximize or Restore, Close and Window Menu for assistive technology.

## Tabs in the title bar

`widgets.BrowserTabs` is a strip of document tabs, browser style —
Chromium's tab strip, SourceGit's repository tabs, Dolphin's folder tabs.
As a header bar's centre it takes the caption's whole height, so its tabs
stand on the content below and the selected one joins it (SourceGit's
concave feet in the web looks, the look's own notebook tab elsewhere):

- the tabs, their close buttons (on the selected tab and the one under the
  pointer), the "+" (shown when `OnNew` is set) and the scroll arrows are
  controls; the rest of the strip is caption: a drag moves the window, a
  double click maximizes, a right click opens `OnContextMenu(-1, at)` or the
  window menu. At least 24 px at the end are always caption.
- tabs share the width, at most `MaxTabWidth` (200 px); below `MinTabWidth`
  (80 px) the strip scrolls — the wheel, two arrow buttons at its end — and
  keeps the selected tab in view. Elided titles are the tab's tool tip.
- a press selects (browsers select on the press); a drag past 8 px moves the
  tab and the others make room (`OnReorder`); a middle click closes; closing
  the selected tab selects the next one. `OnClose` asks the app, which
  removes the tab with `RemoveTab`; `NoClose` keeps a tab (a window's last).
- keys: Left, Right, Home, End with the focus (a click does not take it, as
  in browsers; Tab reaches the strip); `Shortcut(e)` from the app's own key
  handler gives Ctrl+Tab, Ctrl+Shift+Tab, Ctrl+PageUp / PageDown, Ctrl+W,
  Ctrl+F4 and Ctrl+T in one call (the window offers Ctrl+Tab to the app
  before moving the focus with it).
- assistive technology sees a page tab list of page tabs, each holding its
  pressable close button, and a New Tab button.
- under the desktop's frame the strip is simply the header bar's row.

Files is the pilot: its folders open in tabs next to the menu bar in the
title bar; the tree and the table show the selected tab's folder, and the
window title follows it.

### Tearing a tab out

A tab dragged along the strip is being reordered; the same tab pulled
clear of the strip (16 px past its edge) leaves the window for one of its
own, which the desktop carries under the pointer until the user drops it.
It is the browser gesture, and it is [one drag](#tear-off) shared with the
dock:

```go
tabs.OnTearOff = func(i int, tab widgets.BrowserTab) widget.TearOffWindow {
    w, _ := a.NewWindow(platform.WindowOptions{Title: tab.Title, Width: 1040, Height: 680})
    w.SetContent(myApp(w, tab))       // the app decides what the window holds
    return w
}
tabs.OnMergeTab = func(at int, tab widgets.BrowserTab, e widget.DropEvent) bool {
    tabs.Select(tabs.InsertTab(at, tab))  // a tab dragged in from another window
    return true
}
```

- **Out**: `OnTearOff` is asked for the window, the toolkit takes the tab
  out of the strip and hands the window to the drag. `TearOffTab(i, at)`
  is the same thing from a menu item ("Move Tab to New Window").
- **In**: dropping it over another strip merges it there, at the gap an
  accent caret marks — the strip tells that drop from a file dropped on a
  tab and marks the two differently. A tab moves rather than copies: it is
  in one window or the other.
- **The last tab** drags its whole window instead: there is nothing to
  tear off, so dropping it on another strip merges it and closes the
  window it came from, and dropping it anywhere else has moved the window.
- **Nowhere**: a drop on the desktop leaves the new window where it fell;
  Escape puts the tab back and the window goes.
- The drag offers `widgets.TabMimeType` (`application/x-uitoolkit-tab`,
  carrying the tab's title) first, so another window of the application
  knows one of its own, and whatever the app offers for the document
  behind it after that (`OnTabDrag`) — so the same drag that merges a tab
  into another window drops its folder into a file manager. A drop from
  another application has the title alone; the whole `BrowserTab` rides
  along in `DropEvent.Payload` in-process.

## Floating dock panels

A dock panel in a window of its own (`app.DockHost`, see
[widgets.md](widgets.md#dockable-panels)) is an ordinary toplevel: it goes
through the same `resolveDecorations` as every other window. Its window
hands the drags of its own caption to the dock (`Window.SetOnCaptionDrag`),
which counts as a caption of its own for the default policy, so it wears
the toolkit's frame where a window with a title bar would — the user's
"system title bar" preference and `UITK_DECORATIONS` still win. Nothing
about it is a popup or an override-redirect window — the desktop moves,
resizes, stacks and lists it like any other.

Under either frame the panel keeps the title bar it had docked, inside the
window. That is a second row of chrome under the caption, which is what Qt
Creator and Visual Studio do and is deliberate: it is the affordance that
docks the panel back, and it means one implementation of the title bar,
its buttons, its keyboard handling and its accessibility serves a panel
whether it is docked or floating.

The desktop's close button hides the panel rather than destroying the
window, through `Window.SetOnCloseRequest`, so showing the panel again
brings the same window back. `app.DockHost` also docks every panel back as
the main window closes, so no panel is left in a window of its own keeping
a finished app alive.

Dragging the panel's title bar — or the window's own caption, where the
toolkit draws it — drags the whole window, and the host it came from
lights up as it passes over: dropping it there docks the panel back where
the indicator says, and dropping it anywhere else has moved the window.
Under the desktop's frame the window's caption is the desktop's, which
moves the window without telling the client (on Wayland it cannot), so
there the panel's own title bar is the one that docks. A panel dragged the other way — out of the host, past its edge —
floats into a window that follows the pointer. Both are the same
[tear-off](#tear-off) as a tab's, so a user rearranges an app by dragging
alone and the float button is a second way rather than the only one.

Where the desktop cannot carry a window the title bar falls back to
`Window.StartMove`, the desktop's own interactive move, as a floating tool
window's title bar does everywhere, and the dock button is the way back
in.

`Window.Position()` answers where the desktop put a window, in logical
pixels like its size, for the backends that are told: X11 from `ConfigureNotify` (translated to the root
only when someone asks, so an interactive move costs no round trip), and
never on Wayland, where a toplevel has no position at all. A saved layout
therefore restores a floating panel's size exactly and its position only
on X11 — and the tear-off does not need it, because the drag protocol
carries the window instead.

## Tear-off

Dragging a tab out of its strip and dragging a dock panel out of its host
are one mechanism: a drag that carries a window. The drag itself is an
ordinary one — the same `widget.Drag`, the same drop targets, a private
type first so another window of the application knows its own — with a
window riding under the pointer, which is what lets the same gesture drop
the thing back into another window.

```go
widget.StartTearOff(c, drag, &widget.TearOff{
    Open:   func() widget.TearOffWindow { return openTheWindowFor(thing) },
    Offset: grab,                       // where in it the pointer sits
    Done:   func(res widget.TearResult, win widget.TearOffWindow) { … },
})
```

Three ways it ends, and the window's fate differs in each: **merged** — a
target took the content away (it performed a move), so the window that
carried it has nothing left in it; **kept** — the pointer was let go over
nothing, so the window stays where the desktop left it; **cancelled** —
Escape, so nothing happened and the source takes its content back. Closing
the window is the source's, since only the source knows whether the drag
made that window or merely moved one that was already there.

| | Carrying the window | Where it comes from |
| --- | --- | --- |
| Wayland with `xdg-toplevel-drag-v1` (KWin ≥ 6.0, Mutter ≥ 47) | the compositor moves it, as though the window itself were in an interactive move | `get_xdg_toplevel_drag` on the `wl_data_source` **before** `start_drag` — the protocol allows it nowhere else — then `attach(toplevel, dx, dy)` once the tear-off has a window, best while it is still unmapped |
| X11 | the toolkit moves it itself on every motion of the drag: before it maps through `WM_NORMAL_HINTS` (`USPosition`, static gravity), so the window manager maps it under the pointer rather than where it cascades new windows; once mapped with `_NET_MOVERESIZE_WINDOW` from a *user* source, because KWin keeps a window an application moves inside the screen and a torn-off window as big as its parent would otherwise be pushed back on screen, nowhere near the pointer | an X11 client places its own windows; the carried window is left out of the search for a drop target, as the Wayland protocol also requires. The drag follows the surface, not its X window, since a window taking the toolkit's frame is re-made on an ARGB visual before it first maps |
| Wayland without the protocol | it does not | the drag runs all the same, carrying its picture, and the window is made at the drop — wherever the compositor puts it. `widget.DragsWindows(c)` is what a source asks to know which it is in, because that decides whether its content leaves now or at the drop (Chromium's fallback tab dragging, for the same reason) |

`platform.ToplevelDragSurface` is the seam (`DragsToplevels`,
`AttachToplevel`); `UITK_TOPLEVEL_DRAG=0` makes the Wayland backend ignore
the protocol, which is how the fallback path is exercised on a compositor
that has it.

## The looks

A look paints the frame through `style.DecorationOf` (the measurements in
a `DecorationState`: active or backdrop, maximized, tiled, a caption with
the app's own items), `DrawDecorationOf` (border and caption band, under
the title bar), `DrawCaptionTitleOf` and `DrawCaptionButtonOf`. An engine
paints its era's frame by implementing the optional `style.DecorationEngine`
(see [theme-engines.md](theme-engines.md#window-frames)), and all 29 of them
do: every pack wears the frame its original had. An engine without the hook
— a new one — has its in-app window frame (`DrawWindowFrame`,
`WindowCloseRect`) adapted until it grows its own.

The table's corners and shadow are what the era had — the shadow's numbers
are its reach past the window at 1x, top / right / bottom / left:

| Engines | Frame | Corners | Shadow |
| --- | --- | --- | --- |
| `win95` | stacked: the 4 px raised border, the caption bar in the scheme's colour (98's and 2000's gradient), the bold title at its left, bevelled buttons with Marlett-shaped glyphs, close 2 px apart | square | none |
| `luna` | stacked: the scheme's caption gradient, the sizing frame's five lines, the bold shadowed title, XP's glossy 21 px buttons (minimize and maximize in the caption's shade, close red) | 7 px top | none (XP had none) |
| `aero` | merged: an opaque approximation of the glass all round, the black title on its glow, the joined button group hanging from the top edge (26 px frosted minimize and maximize that glow blue, the 43 px red close) | 6 px top, 3 bottom | 6/12/18/12 |
| `metro` | Windows 8 stacked: the thick flat coloured frame, the centred title, the red close hanging from the top; Windows 10 merged: 46 px flat buttons, red close | square | Windows 8 none; Windows 10 5/8/11/8 |
| `fluent` | merged: Mica, the window stroke, 46 px buttons filling the title bar, WinUI's subtle fills, `#c42b1c` close | 8 px | 7/15/23/15 |
| `aqua` | stacked: the pinstriped title bar, the centred title, the gel traffic lights | 5 px top | 7/21/35/21 |
| `macos` | merged: 28 px title (52 px unified tool bar), the centred title, 12 px traffic lights | 10 px (4 before Big Sur) | 7/25/43/25 |
| `breeze` | merged: KWin Breeze's look — glyph buttons that fill a circle under the pointer, red for close | 3 px top | 9/21/33/21 |
| `adwaita` | merged: libadwaita's header bar with 24 px round buttons (GTK 3's flat square ones) | 12 px (8 on GTK 3) | 4/7/10/7 |
| `web` | merged: SourceGit's 38 px title bar, 48×30 buttons with a black-25% wash and a pure red close | 8 px | 7/7/7/7 |
| `motif` | stacked: mwm's resize handles and raised title-bar parts | square | none |
| `flatlaf`, `material` | merged: FlatLaf's title pane; Material's surface with circular icon buttons | square | FlatLaf 5/9/13/9; Material's elevation 7/10/14/10 |
| `system7` | stacked: the title bar's six stripes broken by the 11 px close box at the left, the zoom box at the right and the bold title in its own margin; System 7's colour chrome bevels bar and boxes in lavender and navy | square | none |
| `platinum` | stacked: the black outline with the Platinum bevel inside it, a bar of raised ridges out from the title — centred on the whole bar, as Mac OS 8 put it — to the boxes, the close box left, collapse and zoom right | square | none |
| `amiga` | stacked: Intuition's borders and gadgets — 3.1's raised frame, recessed body, FILLPEN bar and close, zoom and depth gadgets; 1.3's white borders on blue, drag-bar stripes (ghosted in the backdrop) and the gadgets behind their blue lines | square | none |
| `next` | stacked: the black frame line, the title bar with miniaturize at its left and close at its right — NeXT's 15 px raised buttons, Window Maker's square tiles with the bar between them bevelled as a piece of its own — and the notched resize bar as the bottom border | square | none |
| `beos` | stacked: the yellow tab standing on the window's corner, as wide as its close box, title and zoom box with R5's gaps (19 and 17 pixels round the title), over the five-pixel border round the content; full width when maximized or tiled | square | none |
| `os2` | stacked: Warp 4's sizing border, the mini icon, the sunken #2B00AA title well and the Close, Hide and Maximize glyphs | square | none |
| `win31` | stacked: the sizing frame, the navy caption with its centred bold title, the control-menu box and the arrow buttons (both arrows while maximized) | square | none |
| `openlook` | stacked: olwm's black outline with the L-shaped resize corners, the header recessed while focused, the abbreviated menu button — the only control OPEN LOOK put on a frame | square | none |
| `kde1` | stacked: kwm's raised grey border with the title strip sunk into it, blending navy into black, the glyphs outside it on the bare grey | square | none |
| `kde2` | stacked: KWin 2's grab bar over the flat blue title bar woven with its stipple, the menu button on the bar's colour and the rest grey | square | none |
| `keramik` | stacked: the rounded gel slab, the title riding in its bubble, round gel buttons | 7 px top | none |
| `plastik` | stacked: the four-pixel border read from the outside in, the lit title bar, square buttons | 3 px top | none |
| `oxygen` | stacked: the window's own gradient running on through the caption, the embossed centred title, round slabs, the float frame's rim | 5 px top | 7/17/27/17 |
| `clearlooks` (and `bluecurve`) | stacked: Metacity's title bar in three bands lit along its top, the centred bold title over its shadow, small rounded buttons | 5 px top (Bluecurve square) | none |
| `metal` | stacked: the grooved five-line border with solid corners, primary 3 (Ocean's wash) with the bumps, flush black-edged buttons | square | none |
| `nimbus` | stacked: the rounded glossed bar, the centred bold title, round-cornered buttons, close glowing red | 6 px top | 5/9/13/9 |
| `fusion` | stacked: the MDI title bar Qt itself paints — the highlight gradient, chamfered top, centred title, bevelled boxes | 4 px top | 7/13/19/13 |
| `base` | merged: a hairline, flat buttons over the tool face, close red | square | 7/13/19/13 |
| the adapter | nothing reaches it: it is the fallback a new engine gets until it paints its own frame — its in-app caption as a stacked strip, its own close button, push buttons for the rest | square | the engine's dialog shadow |

A look whose free caption is part of its picture — Platinum's ridges,
Window Maker's middle section — implements
`style.CaptionTitleSpanEngine`: it is handed both the free space between
the button groups and the whole strip, and centres its title on the one
while dressing the other. A look that keeps its own room round the title
states it (`DecorationSpec.TitleRoom`, BeOS's 19 and 17 pixels), and a
fitted caption is measured by it. `app/frame_fidelity_test.go` holds
BeOS's tab, Platinum's title and Window Maker's seams to the looks'
in-app windows, pixel by pixel, at 1 and 1.75.

Glyphs, sizes and colours are the look's; the side and order of the
buttons are the desktop's, unless the user prefers the look's own layout:
look.json `"captionButtons": "theme"` puts the Mac's traffic lights on the
left, gives GNOME's lone close, KDE's window menu on the left. It applies
live, to every app. Settings had a **Theme buttons** box for it and does
not any more: the question only arises while the toolkit is drawing the
frame, and while it is, the theme is the only thing with an opinion — so
unticking **OS window borders** writes `"captionButtons": "theme"` with
`"decorations": "toolkit"`, and ticking it writes the desktop's layout
back. `style.CaptionButtonsPref` and `app.Application.SetCaptionButtons`
are unchanged: an application that wants to choose still can, and a
`look.json` that already names a layout keeps it.

`go run ./cmd/uitk-themesheet -frames -theme luna -o /tmp/f` renders a
pack's frames sheet: windows active, in the backdrop, maximized, with the
close button hot and pressed, maximize hot, with tabs in the title bar
(active and backdrop) and with KDE's button layout.

### Keyboard and accessibility

- The title bar's controls come first in Tab order; their mnemonics and menu
  accelerators work; keys from them reach the content root (an app's
  shortcuts).
- Alt+F4 closes a window whose frame uitoolkit draws (a compositor that owns
  the shortcut never sends it). Alt+F3, Alt+Space and Meta+arrows stay the
  desktop's.
- The accessible window's first child is the title bar (`a11y.RoleTitleBar`,
  AT-SPI `title bar`, 104) named after the window, holding the header bar's
  content and the caption buttons, which AT-SPI clients can press. Under the
  desktop's frame the header bar is a plain pane (the real title bar is the
  desktop's).

## The shadow's margin

A look that gives its windows a drop shadow (`DecorationSpec.Shadow`: the
Mac's, GNOME's, Plasma's, Windows 11's) gets a surface larger than the
window. The band around the visible window — the **margin** — holds the
shadow and nothing else, and the window system is told where the window
really is, so nothing outside the toolkit ever measures the shadow:

```
surface  = the buffer, window + margin        (Surface.Size, SurfaceSize)
window   = what the user sees and the desktop moves, snaps and tiles
           (Window.Size, Window.WindowRect; Wayland's window geometry,
            X11's _GTK_FRAME_EXTENTS)
margin   = max(the look's shadow reach, the 10 px resize band), per edge,
           as whole logical pixels; zero where there is no shadow
```

- **Resize handles live in the shadow**: a band 10 logical px outside the
  visible window, with 16 px corner zones — Chromium's `kResizeBorder`,
  GTK's 12 px handle, SourceGit's ring. The rest of the margin is not the
  window's: a press there goes to whatever is behind it (`set_input_region`
  on Wayland, an XShape input region on X11). Without a shadow the band
  stays inside, as before: the look's border, at least 4 px, and the top
  4 px of the caption.
- **The corners are cut out of the painted surface** (paintengine2d's
  dest-out operator), so everything else is still painted rectangular and
  fast, and the desktop shows through the curve with the shadow around it.
  The opaque region leaves the corners out, so a compositor never takes a
  see-through pixel for a solid one.
- **The shadow costs nothing per frame**: it is rasterised once into a
  nine-patch — four corner tiles and the one-pixel strips between them,
  keyed by look, state, scale and margin, so a resize reuses it — and
  blitted into the margin. Partial redraws are clipped to the visible
  window, so a hover, a caret or a menu never repaints the margin at all.
  A window's shadow differs between the active and the backdrop look, but
  its *reach* is always the active one's: focus never resizes a window.
- **Everything in window coordinates follows the margin.** Pointer, touch
  and drop coordinates are the surface's, and the window is laid out inside
  it, so a widget is where the user sees it; menus, combo lists and
  tooltips clamp to the window, never into the margin; the caret rectangle
  the input method gets is surface-local, as text-input wants it;
  accessible extents are relative to the window; `Capture` (and an app's
  `-screenshot`) crops to the window, while `CaptureSurface` keeps the
  shadow.

### Per state

| State | Margin | Corners | Shadow |
| --- | --- | --- | --- |
| Restored | the look's, at least the resize band | the look's | yes |
| Maximized | none | square | none |
| Full screen | no frame at all | | |
| Tiled (`xdg_toplevel` `tiled_*`, KWin quick tile) | none on the tiled edges | square where a tiled edge meets | none on those edges |
| No compositing manager (X11) | none | square | none — `WindowState.Solid`, GTK's `.solid-csd` |

A window's **silhouette** follows the same rules one step further: a
maximized, full-screen or tiled window drops its shape as it drops its
corners and its shadow, and gets it back on restore. An uncomposited X11
screen keeps the shape — XShape still cuts the window, hard-edged — but
drops its glass, since there is nothing behind it to blur. See
[docs/shapes.md](shapes.md).

A **look's** silhouette is dropped in the same states and by the same rule,
and so is the fitted caption that goes with it: the shape and the width of
the band that sits in it are one decision, made in `DecorationOf` and
`WindowShapeOf`. A full-screen window never reaches either, because it has
no frame of the toolkit's at all.

KWin sends `tiled_*` for quick tiles and screen-anchored tiles, so a
window tiled to the left keeps its shadow only on its free right edge.
EWMH has no tiled state at all, so an X11 quick tile keeps its shadow (as
GTK's windows do); `_GTK_FRAME_EXTENTS` still puts the window itself in the
right place.

### Alpha only where a frame needs it

`Frame.Alpha` is set only when there is a shadow, a rounded corner, a
silhouette or glass to paint. Without it the buffers stay opaque (`XRGB8888`, an EGL config with
no alpha, the screen's own visual on X11) and the opaque region is the
whole surface — the path every window took before Phase 3, so an opaque
window can never present see-through. With it, the content is still opaque
to its last pixel: the tests assert alpha = 1 everywhere inside the window
on the CPU and GPU paths, and the margin is the only place anything
translucent is painted.

On X11 a translucent frame needs a 32-bit visual, which a window is born
with: a window that has to change visual is re-created (invisible before
the first map, which is where a frame is normally decided). Compositing is
watched through `_NET_WM_CM_S<screen>` with XFixes, so a compositor
stopping turns every frame solid without a restart, and starting brings the
shadows back for windows created after it.

## Who draws the frame

`platform.Decorations`: `DecorationsAuto`, `DecorationsServer` (the desktop),
`DecorationsClient` (uitoolkit), `DecorationsNone` (no frame: splash screens,
kiosks). First match wins:

1. `UITK_DECORATIONS=server|client|none` (also `system`, `toolkit`, `auto`),
   for testing and as an escape hatch.
2. `WindowOptions.Decorations` when not Auto.
3. Offscreen / headless windows: none of the toolkit's (screenshots and tests
   look the same everywhere) unless 1 or 2 asks.
4. The user's preference, look.json `"decorations"`: `"system"` (the desktop
   frames every window) or `"toolkit"` (the toolkit does). These are the two
   states of Settings' **OS window borders**, ticked and unticked.
5. Auto: the toolkit's frame for a window with a title bar (`SetTitleBar`)
   where the desktop moves and resizes on request; the desktop's frame for
   every other window. So every app without a title bar looks exactly as
   before on KDE.

The choice is made before the window is first shown and again whenever an
input changes (a title bar set, the preference applied): the switch takes
effect at once, without a restart. The desktop's answer is obeyed.

### The Settings switch

Settings shows **OS window borders** — *OS window borders: the desktop's
title bar and borders* to a screen reader, Chromium's switch by another
name — last in the row of options over the preview, after **OS open/save
dialogs**. On: windows
that draw their own title bar get the desktop's title bar and borders instead,
and their title bar becomes the first row. Off: the toolkit draws the frame of
every window in the theme's style, the way it already did for a window with a
title bar. Apply writes look.json (`"decorations": "system"` or `"toolkit"`)
and every running app switches live.

The box has two states, so it writes the two definite preferences and never
`auto`. Auto is a third thing — the toolkit's frame for a window that has a
title bar of its own, the desktop's for every other — and it stays the
default of a file nobody has edited, but it is not what unticking the box
says. Unticked wrote `auto` before 0.20.0, which left every window without a
title bar of its own (Settings' own window, the sample's, most windows) with
the desktop's frame: the box promised the theme's borders and nothing
happened. A preference the user never touched is still left alone, so
applying a theme does not turn client-side frames on behind their back.

### Per platform

| Platform | Toolkit frame | Desktop frame | Notes |
| --- | --- | --- | --- |
| Wayland with `zxdg_decoration_manager_v1` (KWin, wlroots, COSMIC, Hyprland…) | `set_mode(client_side)` | `set_mode(server_side)` | A mode is always asked for (KWin reads "unset" as server) and re-sent only when it changes; the compositor's `configure(mode)` is obeyed whenever it comes. KWin answers `server_side` — meaning no frame at all — for full-screen windows and, with `BorderlessMaximizedWindows`, maximized ones: the caption buttons go, the header bar stays the top row. Hyprland always answers `server_side`. |
| Wayland without it (GNOME's Mutter, Weston) | always | — | Nobody else draws a frame: every window gets the toolkit's, a window without a title bar a default caption with its title. (Before, such windows had no title bar, buttons or resize edges at all.) |
| X11 | `_MOTIF_WM_HINTS` {flags 2, decorations 0}, set before mapping | the property removed | Auto picks the toolkit's frame only where the window manager lists `_NET_WM_MOVERESIZE` and is not a tiling manager (i3, xmonad, awesome, …). Moves and resizes are `_NET_WM_MOVERESIZE` from the press's root position after ungrabbing; the window menu is `_GTK_SHOW_WINDOW_MENU`; minimize is `XIconifyWindow`. |
| Offscreen | when asked | — | `Offscreen` records every request (`FrameCalls`) and can simulate states, capabilities and answers, for tests. |

Window state follows the desktop: `EventWindowState` (maximized, full screen,
activated, resizing, tiled edges, suspended) from `xdg_toplevel.configure` or
`_NET_WM_STATE`; `activated` (`_NET_WM_STATE_FOCUSED`) drives the active /
backdrop look; a suspended window stops blinking its caret.

## The desktop's conventions

`Application.TitleBarPrefs()` (overridable with `SetTitleBarPrefs`):

| Desktop | Button layout | Click actions | Double-click time, drag threshold |
| --- | --- | --- | --- |
| KDE Plasma | kwinrc `[org.kde.kdecoration2]` `ButtonsOnLeft` / `ButtonsOnRight` (M window menu, I minimize, A maximize, X close, `_` spacer; KWin's other buttons are skipped). Defaults `MSE` / `HIAX`: the window menu on the left, minimize, maximize, close on the right | kwinrc `[Windows] TitlebarDoubleClickCommand`, `[MouseBindings] CommandActiveTitlebar2/3` | kdeglobals `[KDE] DoubleClickInterval`, `StartDragDist` |
| GNOME and others with a GTK / GNOME portal backend | `org.gnome.desktop.wm.preferences` `button-layout` through the settings portal (GNOME's default `appmenu:close`: close only) | `action-double-click-titlebar`, `-middle-`, `-right-` | `org.gnome.desktop.peripherals.mouse` `double-click`, `drag-threshold` |
| elsewhere | GTK's `settings.ini` `gtk-decoration-layout` (GTK 4, then 3) | `gtk-titlebar-*-click` | `gtk-double-click-time`, `gtk-dnd-drag-threshold` |
| defaults | trailing minimize, maximize, close | double: toggle maximize; middle: none; right: menu | 400 ms, 8 px |

The settings portal carries none of KWin's keys on Plasma, so kwinrc is read
directly (cascading over `XDG_CONFIG_DIRS`), and re-read when a window becomes
active after a config file changed; the portal's keys arrive live. Actions
xdg-shell cannot express fall back: "maximize vertically / horizontally only"
toggle-maximizes on Wayland (X11 does it one way), "lower" works on X11 only,
"shade" and "on all desktops" do nothing.

## Dressing the desktop's frame

Where the desktop draws the frame, the window still says how to dress it
(`app/dress.go`, `platform/windowdress.go`).

- **Colours.** KWin paints each window's frame in a colour scheme of the
  window's choosing. A window hands it its look's
  (`style.KDEColorScheme`): the title bar — the scheme's Header set,
  active and inactive, and its `[WM]` group — in the look's own caption
  colours, read off its frame painted offscreen, and every other set from
  its palette. So a Luna window under KWin's Breeze frame has a blue title
  bar with a white title, lighter blue in the backdrop; Aqua's and
  Tahoe's are their light greys. The file is written once per look under
  the user's cache directory (`$XDG_CACHE_HOME/uitoolkit/colors`, named
  after its contents, so KWin never shows a stale one), never under the
  config directory where KDE keeps the user's own schemes. It is sent
  whenever the look or the frame changes, even while the toolkit draws the
  frame, so a live switch to the desktop's comes up in the right colours.
  Nothing is written or sent where the desktop does not take one;
  `UITK_DECORATION_PALETTE=0` keeps the desktop's own colours everywhere.
  Wayland: `org_kde_kwin_server_decoration_palette`, only where KWin
  advertises it; X11: `_KDE_NET_WM_COLOR_SCHEME` under KWin.
- **Icon.** `Application.SetIcon(images...)` gives every window its icon,
  square images at the sizes the app has; `Window.SetIcon` one of its own.
  The desktop shows it in the frame's title bar, the task bar and the task
  switcher, and a program run from its build tree has no desktop entry to
  take one from otherwise. The examples wear one made by
  `icons.AppIcon` — a Lucide glyph in white on a tile of the app's colour,
  16 to 128 px, 96 among them because that is the size KWin asks for.
- **Resizing on X11.** `_NET_WM_SYNC_REQUEST`: the window manager waits
  for the frame at each new size before it shows it and takes the next
  step ([platform.md](platform.md#_net_wm_sync_request)).

## Testing

- Headless: `app/frame_test.go` (hit-test table, drag threshold vs click vs
  double-click, one `StartSystemMove` per drag and none per click, capture
  dropped after it, resize edges, window menus, caption buttons, the a11y
  tree, modal dialogs, the decoration policy); `platform` tests decode the
  protocol states, capabilities, resize-edge and `_NET_WM_MOVERESIZE`
  mappings, `_MOTIF_WM_HINTS`, and parse button layouts and kwinrc / GTK
  files; `widgets/caption_test.go` checks every built-in's caption answer.
- `style`: every pack's frame at 1, 1.25, 1.5, 1.75 and 2x — borders on
  whole device pixels, buttons inside the caption, square when maximized —
  paints inside its window, title box and button boxes; every engine has a
  frame of its own and paints its caption band; corners and shadows are the
  era's and each look's own button layout is its desktop's; the adapter,
  driven directly, still takes the in-app frame's caption, borders and close
  button; glyphs stay crisp.
  `widgets`: the tab strip (layout, caption answers, select, close, new,
  drag to reorder, overflow and scrolling, keys and shortcuts, tool tips,
  accessibility). `app`: stacked frames (strip, row, hit tests, a drag from
  either, the title for AT-SPI), tabs in the title bar (a tab is a control,
  the strip caption, the strip's menu, Ctrl+Tab to the app), the theme's
  button layout following the look, the title repainting.
- The translucent frame: alpha 1 everywhere inside the window (on the CPU
  and, replaying the recorded scene, on the GPU), shadow pixels only in the
  margin, the margin and the corners per state (maximized, tiled per edge,
  uncomposited), the shadow patch reused across resizes and rebuilt when
  the state changes, partial damage never reaching the margin, and the
  window coordinates of events, drops, popups, tooltips, the caret
  rectangle, accessible extents and `Capture`. `platform` tests the margin
  rounding, the input band and the opaque region's geometry.
- `UITK_DECORATIONS=client` gives headless screenshots with the toolkit's
  frame; `uitk-themesheet -frames` renders every pack's frames, over a desk
  colour so their shadows show.
- End to end in the nested-KWin rig ([tools/e2e](../tools/e2e/README.md)):
  `UITK_DECORATIONS=client ./run.sh N <app-binary>`, drags and
  resizes with `in.sh`, window geometry and state from `./kwin.py`, protocol
  traces with `WAYLAND_DEBUG=1`; `UITK_BACKEND=x11` for the X11 path on the
  instance's Xwayland. `UITK_XDG_DECORATION=0` makes the Wayland backend
  ignore `zxdg_decoration_manager_v1`: GNOME's path, on KWin.
  `./kwin.py N` is the oracle for the margin: `bufferGeometry` minus
  `frameGeometry` is exactly it, and it goes when the window is maximized.
  `./frame-switch.sh N [BACKEND [PACK]]` drives Settings' **OS window borders** box
  both ways and asks KWin who holds the frame after each Apply
  (`clientGeometry` is inside `frameGeometry` while the desktop draws it,
  and the same rectangle once the toolkit does). Run it on both backends and
  with both a flat pack and one whose frame has a shadow: the Wayland half
  is an xdg-decoration mode change on a mapped surface, the X11 half is
  `_MOTIF_WM_HINTS` written on a mapped window, and a shadowed frame
  re-creates the X window on a 32-bit visual on the way. KWin honours all
  three; a compositor that only read the mode at map time would show up
  here as a frame that does not change until the app restarts.
  Tear-off is checked there too, on both backends: a tab dragged out of
  Files becomes a second window holding that folder, one dragged onto
  another window's strip merges into it (`kwin.py` counts the windows
  before and after), Escape puts the tab back, and the Inspector's panels
  go out and come back the same way. Note that a drag has to be one
  `in.sh` invocation — the injector's button goes up when it exits — and
  that `WAYLAND_DEBUG=1` shows `xdg_toplevel_drag_v1.attach` with the
  offset the window is carried by.
  The rig's KWin wears Breeze (`start.sh` seeds its kwinrc), so the
  palette shows as it does on a stock Plasma; `./kwin.py N` reads a
  window's `colorScheme`, KWin's debug console (`showDebugConsole` on the
  instance's bus) lists windows with their icons, and
  `./resize-shot.sh` drags a resize edge and shoots mid-drag, with
  `UITK_X11_SYNC=0` for the comparison.
  Put a second window behind one with a shadow to see the shadow composite
  and to check that a click in the outer margin reaches it, while one in
  the resize band resizes. `examples/uitoolkit-sample-shapes -mode backdrop` is the test
  card to put behind a shaped or glassy window: a patch of it seen through
  a hole says which patch it is, so moving the window shows a different one
  and a blur of it is unmistakable.

## Phases

1. **Phase 0** (done): window state, capabilities and decoration mode
   reported; press serials and root positions kept; resize and move cursors.
2. **Phase 1** (done): the opaque toolkit frame above — buffers stay opaque
   (XRGB, full opaque region, window geometry = the whole surface), the
   present path is unchanged.
3. **Phase 2** (done): themed frames — `style.DecorationEngine` with a
   native frame for every engine (the adapter is now only what a new engine
   gets before it writes one), merged and stacked title bars, a restore
   glyph per engine, `"captionButtons": "theme"`, and document tabs in the
   title bar (`widgets.BrowserTabs`, Files).
4. **Phase 3** (done): shadows and rounded corners — alpha buffers where a
   frame needs them (`wl_shm` ARGB8888, an EGL alpha config, an X11 32-bit
   visual with compositing-manager detection), `set_window_geometry`, input
   and opaque regions, `_GTK_FRAME_EXTENTS` and an XShape input region,
   resize handles in the shadow, per-edge variants for tiled windows, a
   solid frame without a compositing manager, and every window coordinate —
   events, popups, the caret rectangle, accessibility, screenshots —
   following the margin ([the shadow's margin](#the-shadows-margin)).
5. **Phase 4** (done): a drag that carries a window — Wayland's
   `xdg-toplevel-drag-v1`, X11 placing its own, and the fallback that makes
   the window at the drop — behind `widget.TearOff`, with a tab dragged out
   of its strip and a dock panel dragged out of its host as the two users
   of it, and `Window.Position()` for the backends that are told where a
   window is ([tear-off](#tear-off)).
6. **Phase 5** (done): shaped and transparent windows — `platform.Shape`,
   `Surface.SetShape` on both backends, glass through
   `ext_background_effect_v1` and `_KDE_NET_WM_BLUR_BEHIND_REGION`, a
   silhouette in the window's frame state with the maximize / tile /
   full-screen rules above, `Capture` and damage following it, a shadow
   that blurs the silhouette's mask rather than a rounded rectangle, and a
   widget-level hit shape with the bounding box as the default
   ([docs/shapes.md](shapes.md)).

Phase 3's ARGB buffers, regions and compositing-manager detection are what
phase 5 is built on: a silhouette is the same machinery told a more
interesting region.

## Still open

Nothing below blocks a window from being drawn, moved or torn off.

- ~~**KWin's server-decoration palette.**~~ [Done](#dressing-the-desktops-frame):
  the desktop's frame wears the look's colours on both backends.
- ~~**A window icon from the client**~~: `Application.SetIcon`,
  `xdg-toplevel-icon-v1` and `_NET_WM_ICON`.
- ~~**`_NET_WM_SYNC_REQUEST`.**~~ The basic handshake, on every X11
  toplevel. The extended one (`_NET_WM_FRAME_DRAWN`) is not done; in the
  nested rig, where Xwayland hands KWin only whole buffers, the two show
  no difference a screenshot can catch.
- **Windows and macOS mappings.** The decoration mode, the frame metrics
  and the caption buttons are modelled on what the Linux protocols say;
  neither `WM_NCCALCSIZE`/`DWM` nor `NSWindow`'s title-bar accessory views
  has been mapped onto them yet.
