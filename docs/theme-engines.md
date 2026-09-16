# Theme engines

A uitoolkit theme is two things:

- a **pack** — colours, metrics and a few engine parameters (`ThemePack`,
  built in Go or loaded from `themes/<name>/theme.json`), and
- an **engine** — the Go code that decides what a button, a scrollbar or a
  tab *is* for that family of looks (`style.Engine`).

One engine paints many packs: the `win95` engine paints Windows 95, 98, 2000,
Hot Dog Stand and High Contrast; a `luna` engine paints Luna Blue, Olive and
Silver. Packs change colours and metrics; engines change shapes.

Engines so far, each a good model for its family:

| engine | looks | a good example of |
| --- | --- | --- |
| `win95` | Windows 95 / 98 / 2000 and schemes | the reference: bevels, dotted focus, overlapping tabs |
| `luna` | Windows XP Luna, Royale | gradients, memoised colours, rebar grips |
| `aqua`, `platinum` | Mac OS X, Mac OS 8 | gel materials, textures, Mac rules (inactive grey, hollow selections) |
| `motif` | Motif, CDE, IRIX, HP VUE | colour derivation, shadow thickness per pack |
| `next` | NeXTSTEP, OPENSTEP, Window Maker | textures as pack data, dithers |
| `fusion`, `oxygen`, `breeze` | Qt Fusion, KDE 4, Plasma | modern flat and glow looks, tone models |
| `web` | SourceGit, Primer, shadcn/ui, Geist, Linear, the editor palettes | a parametric engine: one idiom, every difference a param (see [The web engine](#the-web-engine)) |
| `base` | the stock looks | defaults every engine inherits |

## Where this comes from

Every serious toolkit separates *what a control does* from *how it looks*,
and the good ones let the look replace shapes, not just colours:

| Toolkit | Look is… | Takeaway for uitoolkit |
| --- | --- | --- |
| Qt | `QStyle` subclasses (Windows, Fusion, Breeze, Oxygen, Plastique…) that draw *primitives*, *controls* and *complex controls*, answer *pixel metrics*, *sub-control rects* and *style hints*, over a `QPalette` | Our `Engine`: parts + controls + metrics + scrollbar geometry; `QProxyStyle` ≈ embedding `BaseEngine` |
| WPF | Control templates in theme dictionaries — Microsoft shipped `PresentationFramework.Classic/Luna/Royale/Aero` as separate theme assemblies | One engine per era, packs per colour scheme |
| WinForms / uxtheme | `VisualStyleRenderer(part, state)` — `BP_PUSHBUTTON` × `PBS_HOT`… | `Engine.Face(role, ControlState)` |
| GTK 3/4 | CSS over a node tree (`button:hover`), rendered by GSK | Data belongs in packs (`theme.json` colours, `extra`, `params`); shapes in engines |
| Avalonia | `ControlTheme`s (Fluent, Simple) with selectors | Same split; Fluent is one engine among many |
| Swing | `LookAndFeel` + `UIDefaults`; Metal themes are palettes, Synth is XML skins | Metal/Ocean = one engine, several packs |

## Anatomy

```
style/engine.go        Engine interface, registry, scrollbar geometry, Memo
style/engine_base.go   BaseEngine: the stock painter every engine embeds
style/shapes.go        shared era idioms (Bevel4, Etched, Gloss, Bumps, …)
style/engine_<id>.go   one file per engine: engine + its packs + init()
internal/themesheet    renders every control × state for a look
cmd/uitk-themesheet    writes those sheets as PNG
```

A `Classic` look owns palette, metrics, fonts and tokens; every `Draw*` call
on it goes to its engine: `l.eng().DrawButton(l, ctx, b, st, label)`.

### Dispatch rule

`BaseEngine` never calls its own methods: every part and nested control is
re-dispatched through `l.Engine()`. So an engine that overrides only
`CheckIndicator` changes the check box *and* anything else that paints one,
while inheriting all layout and text code. Override the smallest thing that
gets the era right:

- **Parts** — `Face(role, state)` (fill + border + bevel, returns the label
  colour), `CheckIndicator`, `RadioIndicator`, `Arrow`, `Expander`,
  `MenuHighlight`, `MenuTextColor`, `FieldFocusRing`.
- **Scrollbars** — `ScrollBarStyle` (thickness, overlay vs gutter, arrow
  placement: none / both ends / grouped end / grouped start) and
  `DrawScrollBarParts`. `Transient` bars come and go over the content,
  which keeps its full width (libadwaita, WinUI, Mac OS X since Lion): the
  view shows them while it scrolls or the pointer moves over it and fades
  them out after a second of rest; the engine draws the thin idle form unless
  `ScrollState.Hovered` or a part is pressed.
- **Frames** — `GroupBoxInsets/DrawGroupBox` (titled frames),
  `WindowFrameInsets/DrawWindowFrame` (in-app dialogs: caption, close),
  `DrawWindowBackground` (pinstripes, brushed metal) and `DrawTabPane`.
  Top-level windows whose frame the toolkit draws use the optional
  `DecorationEngine` ([Window frames](#window-frames)) — every engine here
  has one — else the in-app frame adapted.
- **View frames** — `ViewFrameInsets` / `DrawViewFrame` frame lists,
  trees and tables (Qt's `PE_Frame`). By default the frame is the engine's
  own field face, as thick as the `viewFrame` metric (Win95 2px sunken,
  Luna and Aqua 1px, flat looks 0). Views work inside it; `Frameless`
  opts a view out when its pane is already framed.
- **Text and bars** — `ControlFont(role)` is the face a control is labelled
  in (Metal's bold buttons); widgets measure labels with it, so a bold look's
  text fits. `ToolBarInsets` keeps room for a tool bar grip (XP's rebar
  handle). Odd rows carry `StateAlternate` for striped lists (Aqua).
- **Spin boxes** — `SpinBoxStyle` (Qt's `CC_SpinBox`): `Inside` puts the
  step buttons inside the field's frame, which the spin box draws across the
  whole box, then paints the editor and the stepper `StateFrameless` inside
  it (Windows, KDE, GNOME and the base look); `Across` lays them side by
  side, "− +" (GTK 3 and later); neither stands a stepper beside the field
  (Mac OS, Motif, NeXT).
- **Tabs** — `TabOutset` grows the selected tab, which the tab bar paints
  last, so it overlaps its neighbours (Win95 and XP: 2px each side).
  `TabOverlap` lays every tab over its neighbour's border, so two tabs share
  one line (Aero and Metro: 1px; Qt's `PM_TabBarTabOverlap`).
  `BrowserTabEngine` draws document tabs in a title bar, browser style (one
  outlined tab whose feet flare into the tool bar, bare labels between thin
  separators, as SourceGit's repository tabs); `DrawBrowserTabOf` and
  `BrowserTabOutsetOf` fall back to the look's ordinary tab.
  `BrowserTabBarEngine` paints the strip under them where it is not in a
  title bar (`DrawBrowserTabBarOf` falls back to the tab bar);
  `widgets.BrowserTabs` uses all three.
- **Shadows** — `PopupShadow(kind)` is how far a floating layer's drop
  shadow reaches past its bounds, and `DrawPopupShadow` paints it before the
  layer (`PopupMenu` for menus and lists, `PopupTooltip`, `PopupDialog`).
  The window paints them for its popup and tooltip layers and repaints the
  reach when a layer goes away; overlays paint their dialog's. The base
  engine gives modern soft shadows (the pack param `shadow` scales them, `0`
  turns them off); eras without shadows return zero (Win95, Motif, NeXT).
  `DropShadow` is an exact nine-piece gradient shadow; `ShadowReach` is its
  extent.
- **Materials** — `ctx.BackdropBlur(r, sigma)` blurs what lies under r
  (inside the clip) on every device: paint a translucent tint over it next
  for acrylic, glass or vibrancy (Fluent's flyouts, Aero's glass frames).
  `ctx.DrawLayer` fades a group as one, `ctx.DrawCrossFade` mixes two
  drawings. `DitherTile` and `DevicePattern` tile a 50% dither (or any
  image, through `paintengine2d.ImagePattern`) anchored to the device grid,
  for the eras before true colour.
- **Controls** — the 34 `Draw*` methods, same signatures as `LookAndFeel`
  plus the look. Override when the era's layout differs (Win95 combos
  highlight their text; Aqua centres tabs).

`Role` is the uxtheme "part": `RoleButton`, `RoleTool`, `RoleField`,
`RoleCheck`, `RoleRow`, `RoleTab`, `RoleThumb`, `RoleTrack`, `RoleMenu`,
`RoleCombo`, `RoleSplitter`, `RoleBar`, `RolePanel`. `ControlState` is the
state bitset (`Hovered`, `Pressed`, `Disabled`, `Focused`, `Checked`,
`Primary` = default button, `Toggle`, `First` / `Last` in a strip or a
table row, `Inactive`, `Backdrop`, `Alternate` = an odd row, `ExpanderHot`
= the pointer is on a tree row's expander, `Frameless` = a part inside a
frame its parent drew), `Sidebar` = a view used as a sidebar and its rows
(macOS source lists, libadwaita's navigation sidebar, WinUI's navigation
pane; `ListView.Sidebar`, `TreeView.Sidebar`), `AutoRaise` = a tool button
in a tool bar, flat until hovered (Qt's `State_AutoRaise`), where a free
tool button keeps its bezel. `ViewBackground(l, st)` is what an item view's
rows sit on: the field colour, or the sidebar pane.

`StyleHint` answers behaviour questions like Qt's `styleHint`:
`HintDialogPrimaryFirst` (1: "OK Cancel", Windows and KDE; 0: "Cancel OK",
Mac and GNOME), `HintTabsCentered` (Aqua's segmented tabs) and
`HintFormLabelsRight` (Mac, NeXT, Oxygen and Breeze right-align form
labels) and `HintHoverFadeMs`: buttons, check boxes, radios, switches and
combo boxes cross-fade that long when only their hover or focus changes
(Aero 200ms, Adwaita 200ms, Windows 10, Breeze and Oxygen 150ms, Fluent
83ms; older eras switch at once, and presses never fade). The new state is
composited as one layer, so engines draw each state as usual.
`UITK_ANIMATIONS=0` turns fades off. `HintMnemonics` says when mnemonic
underlines show (Qt's `SH_UnderlineShortcut`): `MnemonicsAlways` (Windows
3.1 to 98, Motif, OS/2, KDE 3, GNOME 2, Swing), `MnemonicsOnAlt` while Alt
is held or the keyboard drives the menus (Windows 2000 onward, GNOME 3,
Plasma, FlatLaf), `MnemonicsNever` (Mac OS, NeXT, Amiga, BeOS, Material).

### Accent colours

An engine whose looks recolour around a user accent (Windows 10 and 11,
macOS, Plasma, GNOME 47, Material You) implements `AccentEngine`:

```go
func (breezeEngine) Accented(tok ThemeTokens, accent paintengine2d.Color) ThemeTokens {
    tok = CloneTokenMaps(tok) // never write into the registered pack's maps
    tok.Palette.Selection, tok.Palette.Focus = accent, accent
    tok.Extra["focus"], tok.Extra["hover"] = accent, accent
    ...
    return tok
}
```

`Accented` gets the resolved pack tokens and the desktop's accent when the
user's appearance follows the desktop (`UITK_ACCENT=#e95420` stands in for
it). Recolour what the platform recoloured, derive the rest as the platform
did (hover and pressed shades, the text on the accent), and recompute any
chrome states built from the palette. Historical looks with fixed palettes
leave it out.

### Optional hooks

An engine implements these only when its platform differs from the base:
- `ViewBackground(l, st)`: what an item view's rows sit on. A sidebar
  (`st.Sidebar()`) can have a pane of its own, such as Aqua's source list.
- `SliderTravel(l, b)`: where the slider thumb's centre travels, so tick
  marks and clicks line up with a thumb sized the engine's own way.
- `DrawSliderTicks(l, ctx, b, xs, st)`: tick marks in the platform's
  style.
- `ComboTextRect(l, b)`: where an editable combo box (`st.Editable()`) puts
  its field. Windows looks draw the editable box as a field with an arrow
  button, and must not fill it with the selection colour when focused.
  Windows 3.1 stands the button eight pixels off the edit box.
- `DrawFramelessText(l, ctx, b, st, …)`: the text of a frameless field (a
  spin box's, an editable combo's), inside a frame its parent drew. An
  engine that paints its own field text (its colour, caret, selection)
  implements it, keeping the text in `l.fieldTextBox(b)`, so those fields
  type as its plain ones do: Motif's I-beam, Windows 3.1's XOR caret.
- `Accented(tok, accent)`: see Accent colours above.
- `EngineFor(tok)` (`EraEngine`): an engine hands a later era of its
  platform to an engine type of its own, which embeds it and lives in a file
  of its own, keyed on a pack param: `macos` with `era` 2 is Tahoe's Liquid
  Glass, `breeze` with `plasma` 6 is Plasma 6, `adwaita` with `era` 48 is
  GNOME 48, `material` with `expressive` 1 (and `gen` 3) is Material 3
  Expressive. The look keeps the engine's ID and takes the era engine's
  default metrics. Go's embedding calls the embedded engine's own methods,
  so an era that changes a part overrides the controls that paint it too.
- `DrawToolGroup(l, ctx, b, n)` (`ToolGroupEngine`): a tool bar hands each
  run of its `n` buttons between separators (their union `b`) to the
  engine, which paints chrome they share under them, and the separators
  become space (Tahoe's glass capsules).
- `l.WeightFont(WeightMedium)` and `WeightSemibold`: Material's medium
  labels and Fluent's semibold headings; the installed weight, or the
  nearest drawn heavier.

### Item views

`DrawListRow`, `DrawTreeRow` and `DrawTableCell` get the item state:
`Checked` = selected, `Hovered`, `Focused` = the current row of a focused
view, `Inactive` = the view does not have keyboard focus, `Backdrop` = the
window is inactive (GTK's `:backdrop`). Follow the platform you imitate:
Windows and Mac looks grey a selection as soon as the view is `Inactive`
(Win95 to the button face, Aqua pale grey, Platinum a hollow outline), GTK
and Qt-style looks keep it until `Backdrop`, Motif and NeXT never change
it. `RoleRow` faces see the same bits, and every widget's state carries
`Backdrop` while its window is inactive (Aqua's default button turns clear).

Table cells carry `First` / `Last` (a cell with neither is a middle cell).
To draw one selection box across a row, paint the row box over
`CellSpan(b, st, reach)` clipped to the cell: the span runs past the sides
where the row goes on, so the cells join into one box (Aero's Explorer
selection, Fluent's list item and pill).

A selected list row carries `SelectedAbove` / `SelectedBelow` when the row
next to it is selected too, so an engine can join a run of selected rows
into one box: square the corners they share, keep the outer ones round
(SourceGit's sidebar lists, macOS's inset lists).

Tree rows carry their branch chain (Qt's `State_Sibling`): `st.HasNextSibling(d)`
says whether the node at depth `d` on the row's chain (its ancestors, then
the row itself) has a sibling after it. Draw a branch line through the row
only where that branch continues, and end a last child's line at its elbow
("└" rather than "├"). Rows without the chain (`StateTreeChain` unset)
answer true, so lines keep running.

List and tree painters mark a `Focused` row themselves — default
`ItemFocus` over the row, Win95 and XP a dotted rectangle around the label,
Motif the solid location cursor; tables call `DrawItemFocus` over the whole
row after its cells. Looks whose fields have a focus ring (`FieldFocusRing`)
may leave `ItemFocus` empty and let the view frame ring the focused view.

## Window frames

A window whose frame uitoolkit draws (a client-side frame,
[decorations.md](decorations.md)) asks its look for the frame. An engine
paints its era's by implementing the optional `DecorationEngine`, in a
file of its own (`style/engine_<id>_frame.go`):

```go
func (lunaEngine) Decoration(l *Classic, st DecorationState) DecorationSpec
func (lunaEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState)
func (lunaEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState)
func (lunaEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState)
```

- `DecorationState`: `Active` (the desktop's active window, not keyboard
  focus: otherwise paint the backdrop look), `Maximized` (no border, the
  maximize button restores), `Tiled` edges, `Custom` (the app's own title
  bar fills the caption: macOS makes it a 52 px unified tool bar).
- `DecorationSpec`, device pixels: `Border` (whole pixels; the toolkit
  drops it when maximized), `Caption` (a stacked strip's height, a merged
  caption's least), `Stacked` (the era's caption strip with the app's title
  bar under it, or merged: the app's title bar is the caption), `Button`
  (Y 0: the caption's full height, as Windows 10 and 11), `ButtonGap`,
  `ButtonPad` (from the caption's top and sides), `CloseButton` and
  `CloseGap` where close differs (Windows 7's wide one, Windows 95's 2 px),
  `CenterButtons` (in a caption taller than `Caption`), `CenterTitle` (the
  title centred on the window, pushed aside by the buttons rather than cut
  short where they leave no room at the centre), `Layout` (the era's
  button layout in GNOME's syntax, for `"captionButtons": "theme"`), and
  for Phase 3 `Radius` and `Shadow`.
- `DrawDecoration` paints the border inside `f.Window` and the caption band
  `f.Caption` (a stacked frame's strip; a unified look may paint `f.Bar`,
  the app's row under it), never the content. `FrameParts` lists the rects
  a frame stays inside. `DrawCaptionTitle` gets the free space between the
  button groups; `DrawCaptionButton` a button's box and its hover and
  pressed state, `st.Maximized` for the restore glyph.
- `DrawCaptionGlyph` draws the generic crisp glyphs (a cross, a bar, a
  square, two for restore, a small window) for engines without their own.
- **Shadow and corners.** `Shadow` is the invisible margin the window asks
  the compositor for; the resize handles live in it, and the look paints it
  through the optional `DrawDecorationShadow(l, ctx, b, st)` — `b` is the
  visible window, and the toolkit rasterises it once into a nine-patch per
  look, state, scale and margin. Build the shadow out of `WindowShadow`
  layers (colour, offset down, blur, spread, device px) with
  `DrawShadowLayers`, and take `Shadow` from `ShadowLayersReach` of the
  **active** window's layers whatever the state, so gaining or losing focus
  never resizes the window. An engine without the hook drops its dialog
  shadow (`PopupShadow(PopupDialog)`).
  `DecorationOf` applies the state's own rules for every look: a maximized
  window has no border, corners or shadow; a window on an uncomposited X11
  screen (`DecorationState.Solid`) is square and shadowless; a tiled edge
  loses its shadow and the corners beside it. An engine states its look's
  own frame and nothing else.
- **What each era had.** Windows 95 to XP, Windows 8, Motif, NeXT and
  Window Maker, the Mac through Mac OS 9, Intuition, BeOS, OS/2 Warp,
  Windows 3.1, OPEN LOOK and Swing's Metal are square and shadowless, as
  they were; KDE 3's Plastik and Keramik and GNOME 2's Clearlooks keep the
  rounded top corners their decorations shaped (Bluecurve, square
  everywhere by Red Hat's design, does not). Windows XP rounds its top
  corners (7px) without a shadow. The composited eras carry both: Aqua
  (5px top corners, a deep soft shadow), macOS (10px all round on Big Sur,
  4 before), Windows 7's glass (6px), Windows 10 (square, a tight shadow),
  Windows 11 (8px, DWM's shadow), Plasma's Breeze (3px top, its Large
  shadow), GNOME's Adwaita (12px, `0 3px 9px 1px` black at half alpha),
  SourceGit and the web packs (8px, a 12px ring), FlatLaf and Material
  (square, their own elevation), KDE 4's Oxygen (5px top, a wide soft
  shadow), Nimbus (6px top, its dialog shadow) and Qt's Fusion (a 4px
  chamfer, the modest shadow of its era).

Every engine in the toolkit paints its era's frame itself, so nothing
reaches the adapter today; it is the fallback a new engine gets for free
until it implements the hook. It takes the engine's in-app window frame:
the caption of `DrawWindowFrame` as a stacked strip (only its parts are
painted, never its body over the content; `WindowState.NoButtons` asks for
it without zoom or depth boxes), its borders round the window, its own
close button moved to wherever the button layout puts it (the in-app frame
is painted translated so its `WindowCloseRect` lands on the button, clipped
to it), push buttons at its size with generic glyphs for the rest, and the
title laid out by the engine over the free space. The frame stays square,
takes the shadow the engine drops under a dialog and states no button
layout of its own, so the desktop's is used. The base look, which is no
era, has the plain frame instead.

Check a frame with `go run ./cmd/uitk-themesheet -frames -theme luna -o
/tmp/f` (and `-scale 1.75`): real windows active, in the backdrop,
maximized, with the close button hot and pressed, with tabs in the title
bar, with KDE's button layout. The decoration tests run over every pack.

## The web engine

`web` (`style/engine_web*.go`) paints today's flat design systems —
SourceGit (Avalonia's Fluent theme as SourceGit restyles it), GitHub's
Primer, shadcn/ui, Vercel's Geist, Linear — the code editors' own looks
(VS Code's 2026 themes, JetBrains' Islands) and the editor palettes people
carry between apps (Catppuccin, Nord, Dracula, Tokyo Night, Rosé Pine,
Gruvbox, Solarized, Atom's One).
They share one idiom: flat faces in a 1px hairline, small radii, a keyboard
focus ring, underlined or segmented tabs, pill switches, menus of rounded
rows on a rounded popover, overlay scroll bars. So the engine is
parametric: each difference is a `params` number or an `extra` colour, and
another design system is a pack, not an engine. The top of `engine_web.go`
lists every key with its default; the ones that decide the shapes:

| params | values [default] | packs |
| --- | --- | --- |
| `radius`, `fieldRadius`, `comboRadius`, `overlayRadius`, `cardRadius`, `viewRadius`, `windowRadius`, `checkRadius`, `rowRadius`, `sideRadius`, `menuRowRadius` | corners in px [`radius` 6; the others follow it, or 8 for overlays and cards, 4 for check boxes, 6 for rows] | SourceGit 3 with square fields, Primer 6 and 12px overlays, shadcn 8, Linear 4 with 8px inputs |
| `focusStyle` | 0 a ring outside the face, 1 a band inside it [1], 2 Avalonia's dotted adorner | shadcn and Geist 0, Linear and the palettes 1, SourceGit 2 |
| `focusWidth`, `focusGap`, `focusAlpha` | the ring [2, 0, 1] | shadcn 3px at 50%, Geist 2px beyond a 2px gap |
| `hoverBorder`, `checkFocus` | the accent border under the pointer; focus as a 2px accent border on check boxes and radios [0, 0] | SourceGit |
| `primaryStyle` | the default button: 0 the accent [0], 1 the text colour, 2 the pack's `primary` | shadcn and Geist 1, Primer 2 (its green) |
| `tabStyle` | 0 underline [0], 1 segmented, 2 browser, 3 editor tabs (boxed on a strip, the active one the editor's colour with a `tabLine` along its top), 4 pills (`tabPillH`, `tabRadius`); `tabLine`, `tabLineGap`, `tabFit`, `tabAccent`, `tabDim` | SourceGit's 1px accent pipe 2px up, Primer's coral line, shadcn and Linear 1, VS Code and Atom's One 3, Islands 4 |
| `island` | px [0]: list, tree, table and sidebar frames, cards and tab views are islands, panels rounded by `viewRadius` / `cardRadius` inside a band left to what is behind (the main window between islands; nested islands merge); a group's heading goes inside, a tab view's strip is its island's top | Islands 3 |
| `checkStyle`, `radioStyle`, `radioSize`, `switchShape`, `knob` | unfilled with an accent tick or filled [1]; ring and dot or filled [1]; pill [0] or rounded switch | SourceGit unfilled, 14px radios; Primer's rounded switch |
| `menuHighlight`, `menuInset`, `menuSep` | a wash [0] or the accent under the row, its inset from the popover's sides, separators from the label column | |
| `rowInset`, `sideInset`, `rowBar`, `listSel`, `sideSel`, `sideOffWash`, … | rows boxed in from the sides or full width, the selection's alpha focused / hovered / unfocused per list, sidebar and table, an accent bar | SourceGit's joined sidebar boxes, Primer's bar |
| `scrollIdle`, `scrollInset`, `scrollArrows`, `scrollRadius` | transient bars: the idle thumb's width [2], the gap [2], arrows while expanded [0], the thumb's corner [a pill] | SourceGit 2px with arrows, Linear 6px, VS Code's square 10px sliders |
| `openRing` | 1: an open combo keeps the focus ring [0] | Islands (Swing rings whatever holds focus) |
| `captionStyle`, `captionButton` | a caption strip with a centred bold title and a close cell that turns red (0), or a web dialog's header and icon close button (1) [1] | SourceGit 0: a 28px strip, a 48px cell |
| `accentFollows` | 1: the pack takes the desktop's accent (`AccentEngine`) and derives Avalonia's shades from it | SourceGit |

A focus ring outside the face needs room: controls keep a margin as wide
as gap plus ring on every side of their face (`focusStyle` 0 packs size
their controls to include it: shadcn's 36px buttons in 42px), and the ring
goes over the face's edge where a control has none (a tool bar's buttons).
With an outside ring a pack's `fieldRing` colour gives focused fields
Geist's halo in place of the ring. The text size is the `fontSize` metric
(13px SourceGit, Linear, VS Code and Islands, 14px the rest), and the
typefaces are each system's own (Inter, Mona Sans, Geist, VS Code's system
stacks) before the engine's Inter-first stack.

## Writing an engine

1. Copy the shape of `style/engine_win95.go`:

   ```go
   type lunaEngine struct{ BaseEngine }

   func init() {
       RegisterEngine(lunaEngine{})
       for _, p := range lunaPacks() { RegisterPack(p) }
   }

   func (lunaEngine) ID() string { return "luna" }
   func (lunaEngine) DefaultMetrics() ChromeMetrics { return ChromeMetrics{…} }
   ```

2. Resolve your colours once per look with `Memo` and read pack data with
   `l.X("key", fallback)` (colours from `extra`) and `l.P("key", def)`
   (numbers from `params`):

   ```go
   type lunaKey struct{}
   func lunaColors(l *Classic) luna {
       return l.Memo(lunaKey{}, func() any { return lunaBuild(l) }).(luna)
   }
   ```

3. Register packs next to the engine — `ThemePack{Name, Label, Year, Lineage,
   Summary, Tokens: ThemeTokens{Engine: "luna", Family, Palette, Extra,
   Params}}`. Registered packs sort by `Year`; a pack with the name of a
   legacy era pack (`luna`, `aqua`, `motif`…) replaces it.

4. Look at it: `go run ./cmd/uitk-themesheet -theme luna -o /tmp/s`
   (and `-scale 2`). Read the PNG. Iterate until it matches the era. Then
   see real apps in it: `go run ./examples/gallery -screenshot /tmp/g
   -theme luna` takes every scripted gallery shot (menus, combo lists,
   tooltips, message boxes, tables) in your pack, and `UITK_THEME=luna`
   runs any uitoolkit app in it, like `GTK_THEME` or `QT_STYLE_OVERRIDE`.

5. Test it. The shared tests run over every registered pack: every control
   paints inside its bounds in every state, shadows stay within their
   reach, keyboard focus is visible, text meets the contrast checks. Add a
   test file for the engine: registration and year order, bounds at scale 1
   and 2, `WindowCloseRect` against the painting, your style hints.

## Rules

- **Paint inside the rect you are given.** Widgets paint clipped to their
  bounds; anything outside is lost (this is why focus rings used to vanish).
- **Rects arrive on whole pixels.** Layout rounds every component's bounds
  to device pixels (as WPF and Avalonia do), and tables round their column
  edges, so `snap`ped lines in your painting land on real pixels. Cells that
  meet share an edge exactly; never grow a cell to cover a seam.
- **Focus is painted inside** the control (`DrawFocusRing(ctx, b)` rings
  the inside of `b`). Check boxes and radios ring their label.
- **Scale everything** that is a design length with `l.S(v)` (display scale)
  or take it from `l.metrics` (already scaled). Snap 1px lines with `snap`.
- **Label colour comes from `Face`** — return a readable foreground for the
  fill you painted (selected rows, hot menus, default buttons).
- **Deterministic and cheap**: no randomness, no allocation-heavy work per
  paint — derive once with `Memo`; prefer rects and simple paths; gradients
  are fine (the GPU path caches their ramps).
- **Metrics**: `DefaultMetrics` is the era's geometry at the toolkit's 16px
  UI font (scale the original proportions; Windows used 11px). Set
  `Square: true` for natively square looks so the "theme" Corners pref keeps
  them square. Density shifts these values; do not fight it.
- **No hover where the era had none** only when authentic (Win95 buttons
  had no hot state) — but never lose feedback users rely on: hover on tool
  buttons and menus, pressed on everything, visible focus.
- **Facts, not copies.** uitoolkit is under The Free License, with no
  restrictions on its users. Take colours, sizes, radii,
  gradient stops and behaviour from other toolkits as facts, and write the
  drawing yourself; never transcribe or port code, path tables, pixmaps or
  data from GPL / LGPL toolkits (Qt and KDE styles, GTK engines, OpenJDK,
  Window Maker) or proprietary artwork.

## For widget authors

Widgets reach the look through `LookAndFeel`, plus optional interfaces a
look may implement; use the `…Of` helpers, which fall back when a look
does not: `ControlFontOf` (measure labels), `DrawArrowOf` (era arrows),
`DrawItemFocusOf` (current-row mark), `ViewFrameInsetsOf` /
`DrawViewFrameOf`, `PopupShadowOf` / `DrawPopupShadowOf`, `TabOutsetOf`,
`TabOverlapOf`, `BrowserTabOutsetOf` / `DrawBrowserTabOf` /
`DrawBrowserTabBarOf`, `ToolBarInsetsOf`, `DecorationOf` /
`DrawDecorationOf` / `DrawCaptionTitleOf` / `DrawCaptionButtonOf` and
`LookHint`. A scrolling view gives up `ScrollGutter` to its bar (nothing
for transient bars) and lets `scrollDrag` show, fade and hit-test it. Build
row states with `widget.ItemState` or `widget.RowItemState`, which add
`Focused`, `Inactive`, `Backdrop` and `Alternate` for you; a list with
several rows selected adds `SelectedAbove` / `SelectedBelow` itself.

## Pack schema (`theme.json`)

```json
{
  "label": "Luna Blue", "family": "light", "engine": "luna",
  "year": 2001, "lineage": "Windows", "summary": "Windows XP default.",
  "metrics": { "controlH": 28, "scroll": 17, "radius": 3, "square": false, "viewFrame": 1 },
  "colors":  { "background": "#ece9d8", "accent": "#316ac5", "selection": "#316ac5" },
  "extra":   { "btnTop": "#ffffff", "btnBottom": "#d6d0c5", "hot": "#f8b330" },
  "params":  { "glossy": 1 },
  "fonts":   { "ui": ["Tahoma", "Verdana", "DejaVu Sans"], "mono": ["Courier New"] }
}
```

`colors` are the shared palette (every engine understands them); `extra` and
`params` are engine-specific and documented at the top of each engine file.
`metrics.fontSize` is the body text size in pixels at scale 1 (unset: 16,
the toolkit's own); a pack whose design sets text at 13 or 14px sizes the
rest of its metrics in the same pixels.

`fonts` lists typefaces, most wanted first; the first one installed wins
(fontconfig), and the bundled Titillium Web / JetBrains Mono stand in when
none is. A pack that names none reads in its era's typefaces
(`style/fonts_era.go`): XP's Tahoma, Vista's Segoe UI, Aqua's Lucida Grande,
NeXT and Motif's Helvetica, GNOME's Cantarell, Plasma's Noto Sans, each
followed by open look-alikes (Nimbus Sans for Helvetica, Liberation for Arial
and Courier New, Selawik and Noto Sans for Segoe UI). A variable font's bold
is drawn by emboldening its regular outline. `UITK_SYSTEM_FONTS=0` keeps
every look on the bundled faces (reproducible screenshots).
