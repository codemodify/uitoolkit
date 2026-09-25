# Settings

`cmd/uitoolkit-settings` is the toolkit appearance editor and theme browser. The
command opens the window; the application is `cmd/uitoolkit-settings/settingsapp`
beside it, written against the published API like every other sample. It
is the toolkit's shop window: picking a pack draws it at once as a small
live application, frame and caption and every control, so a theme can be
judged without launching anything.

## Run

```bash
go run ./cmd/uitoolkit-settings
go run ./cmd/uitoolkit-settings -stage win98      # open with Windows 98 staged
go run ./cmd/uitoolkit-settings -page appearance  # open scrolled to the shape options
go run ./cmd/uitoolkit-settings -headless         # settings.png in cwd
go run ./cmd/uitoolkit-settings -stage aqua -screenshot out/
tools/shots/demos.sh                              # docs/screenshots/settings.webp and the other demos
```

`-page` names a **section of the one page** and opens it scrolled there:
`theme` (the default, the top), `desktop`, `shape`, `behaviour`, `files`.
The names of the four pages Settings used to have still resolve, because
they are in scripts, in the atlas tooling and in the docs of two
releases: `themes` and `packs` (and `packs & icons`, `theme packs`) open
**Theme**, `appearance` and `icons` open **Shape and weight**, `about`
opens **Files**. An unknown name is the top of the page.

## The page

There is one page and no navigation. Settings used to be four pages
behind a sidebar — Themes, Appearance, Packs, About — and they were four
answers to one question: what does this desktop look like. The window is
now a splitter: **the choices down a scrolling column on the left**, and
**the preview filling the whole of the right**, with **Apply** pinned at
the foot, outside both.

The column is about 300 logical pixels wide whatever the window and the
display scale are, and it keeps that share while the window is resized
(`Window.OnResize`). Dragging the sash replaces the split: a dragged
split stays through every resize after, and survives Apply. It was 240
while the column was the theme browser and nothing else; an option row is
a label, a control and a line of prose, and 240 wrapped every one of them
to three lines.

### The preview, on the right

The whole of the pane, at every window size: a small but fully
interactive application window (menu bar, tool bar, tabs with every
control, tree, table, dialogs, status bar) painted entirely in the staged
theme, frame, caption and all. Its caption names the pack it is really
drawing, which is where you see that *Breeze* is showing as *Breeze Dark*
because the desktop asked for dark.

It is a `widgets.ThemeScope`, so Settings itself keeps the applied look.
Staging a pack switches it where it stands — the caret stays in the
search field, the focus on the list, the column where it was scrolled.

**Nothing else is allowed in this pane.** The widget gallery used to sit
under it in a second splitter (those are the widgets the tour shows over
its Controls, Views and Documents pages, and under the preview they
halved it to answer a question the preview had already answered). The
strip of stock icons sat above it for a while when the four pages were
merged, and it read well at 1024 — but the strip keeps its own height and
the preview takes what is left, so at the 720×520 minimum the fifteen
glyphs folded onto four lines and left the preview a caption and a menu
bar. The preview is the biggest thing on this page at every size, and the
only way to promise that is to keep everything that is not the preview
out of its column. `TestSettingsPageHoldsAtEverySize` is that promise,
in four packs at two scales at both sizes.

### The column of choices, on the left

Five sections, in the order a person decides them.

**Theme** — a **search field** (a pack is found by its id, name, year,
family, engine or what its summary says — every word has to match; Return
stages the first hit, Escape empties the field), the **decade filter**
(*All decades*, one decade, or *My themes*) and the packs that pass both,
listed by year (`1995 · Windows 95`), user exports as `User · <name>`.
The list is held to a height rather than run to the foot of the window:
it is in a column that scrolls, and a view as tall as all 129 of its rows
would have made the column ten screens long and given the list no
scrollbar of its own.

Under the list, the two things that can be done to a pack:

- **Export current theme…** writes the staged look out as a theme pack
  of the user's own. It was on the Packs page, which needed a second copy
  of the browser to say which pack it meant; it is under the only browser
  there is now, and the pack it writes appears in that same list a line
  later.
- **Delete theme…** removes the staged pack from disk when it is one the
  user exported, after a Yes/No confirmation. It is **grey rather than
  absent** while a built-in pack is staged: a button that came and went
  would move the whole column under it every time a pack was picked.

**Where the colours come from** — **Follow the desktop's colours**: its
light or dark mode and its accent (see below). It is the one setting that
changes where the look comes from rather than what it is, so it sits
directly under the browser, as the first footnote to the choice above it,
and its effect is read in the preview's caption.

**Shape and weight** — what the staged look is drawn in rather than what
it is: **Corners** (Theme shape / Round / Square), the **icon set**
(`classic`, `sharp`, and every folder installed under the icon
directory), and the **size** its glyphs are drawn at (Small 16, Medium
24, Large 32). One decision with three dials. The two icon choosers are
together because they are one choice: a set's glyphs are drawn at that
size, and some sets are made for one end of the range.

Under them, the **strip**: every icon the toolkit asks a set for by name,
the whole of `style.AllToolIcons`, in five tool bars of at most four
buttons — the file actions, the clipboard ones, the two histories, then
find, edit, mail and download, then the four message-box faces (info,
warning, error, question), which are the ones a set gives a colour of its
own and so the ones that say whether it can be read against a dark pack.
It is captioned `Preview — <set>` the way the theme preview is captioned
`Preview — <pack>`.

The strip is tool **bars** because a loose tool button caps its icon at
its control height, so Large would have drawn exactly like Medium; and
the bars are short because a `Wrap` folds whole bars — a bar wider than
the column is cut off at its edge rather than folded, and the icons past
the cut are simply not there. Four buttons is what fits at the largest
icon size in the narrowest column.

The strip is a `ThemeScope` carrying Settings' own theme with the staged
set and size laid over it, not the staged pack: it is the set being
previewed here, not the pack.

Last in this section, **Delete icon set…** — grey unless the staged set
is one the user copied in. The chooser above it is the only list of icon
sets on the page now, so this is where a set is named and this is where
removing one belongs.

**Behaviour** — everything `look.json` carries that is not what the
toolkit looks like but what it does: **Animations** (hover fades, the
default button's pulse, busy bars; with a note under it while the desktop
asks for reduced motion), **Use the desktop's file dialogs**, **Use
system title bar and borders**, and **Place window buttons as the theme
does**. The old Appearance page had these in three groups of one and two;
they are one group here, because the thing they have in common — none of
them is the appearance of a pack — is the only thing a reader needs to
know to skip the lot.

**Files** — the three paths Settings reads and writes: the **prefs
file**, the **user theme packs** folder, and the **icon sets** folder.
Last, because it is the only part of the page that changes nothing.

### What went

The **Packs** page is gone. Its two lists were a second theme browser and
a second icon chooser; the browser and the chooser are on this page, so
the lists were a duplicate. Its two actions were not, and both moved
next to the thing they act on: Export and Delete theme… under the theme
browser, Delete icon set… under the icon chooser.

The **sidebar** is gone with the pages, and with it the `Pages` list a
screen reader used to drive. There is nothing to drive: one Tab ring
holds the whole application.

## Staged and applied

The previews show what is **staged**. Nothing is written until **Apply**,
which saves `look.json` and switches Settings and every app that watches
the file. Apply is the only button and the only thing in the row under
the page: it sits pinned at the **right** of it, outside the column that
scrolls, so the one thing that writes anything cannot scroll away. The line that used to
lead that row — *Applied — every uitoolkit app is using this look*, or
*Staged, not applied* — is gone; Apply being enabled or greyed says the
same thing in the place you are already looking. Closing without Apply
discards the staged change.

**When something fails**, an error message box comes up: writing
`look.json`, exporting a pack and deleting one all touch the disk, and
that line beside Apply used to be where they reported themselves. A
modal is what replaced it — a failure nobody is told about looks exactly
like nothing having happened.

The window has no title bar row of its own and no status bar: the
splitter and that one row are all of it. (The preview's own status bar
belongs to the previewed application and stays.)

## Screenshot geometry

The Theme Atlas crops the preview panel out of `uitoolkit-settings -stage ID
-screenshot out.png`. In the default 1024×860 window at scale 1, with the
default look applied, the panel is the same rectangle in every pack:

```
x 317, y 10, 697 × 790        crop box (317, 10) – (1014, 800)
```

`TestSettingsPreviewPanelKeepsItsPlace` pins those numbers; a layout
change that moves them fails it, and the new ones belong here. (They were
`x 504, y 175, 510 × 387` before the gallery went under the preview,
`569 × 381` until the preview took 62% of the split instead of 55%,
`x 445, y 58, 569 × 429` until the window's title bar row and status bar
came off, `569 × 481` until the gallery came out from under the preview
and it took the whole right-hand pane, `x 445, y 10, 569 × 782` until the
icons took the head of the page, and `569 × 630` until the four pages
became one.)

The applied look sets Settings' own metrics, and the column of choices
beside the preview is drawn in it, so take atlas shots with a clean
`XDG_CONFIG_HOME`. That is why the test builds Settings with
`PreferredLook`, the way the command does, rather than with a fixture
look. Nothing above the preview shares its column any more, so the crop
no longer moves with the icon size.

## Prefs file

```
$XDG_CONFIG_HOME/uitoolkit/look.json
# fallback: ~/.config/uitoolkit/look.json
```

```json
{
  "version": 2,
  "theme": "breeze",
  "corners": "theme",
  "icons": "lucide",
  "iconSize": "medium",
  "reduceMotion": false,
  "followDesktop": true,
  "nativeDialogs": false,
  "decorations": "system",
  "captionButtons": "theme"
}
```

`theme` is the theme pack. `corners` is `theme` (the pack's own shape,
the default), `round` or `square`. `icons` is the chrome set (`classic`,
`sharp`, or an installed directory such as `lucide`). `iconSize` is
`small` (16px), `medium` (24px, default), or `large` (32px). Numeric
aliases `16` / `24` / `32` are accepted on load. `reduceMotion` turns
animations off. `followDesktop` shows the pack's light or dark sibling to
match the desktop. `nativeDialogs` shows the desktop's own file dialogs
(KDE's, GNOME's, through the XDG portal) instead of the toolkit's themed
ones. `decorations` is who draws the frame of a window with its own title
bar: `system` the desktop (**Use system title bar and borders**),
`toolkit` uitoolkit for every window, left out for the default.
`captionButtons` `theme` puts the caption buttons of a frame uitoolkit
draws where the theme's era put them (the Mac's traffic lights on the
left; **Place window buttons as the theme does**); left out, they follow
the desktop's button layout. See [decorations.md](decorations.md).

### Following the desktop's light or dark mode

With `followDesktop` on, apps ask the desktop for its light / dark
preference, the setting GTK 4, libadwaita, Qt 6 and the browsers follow
(on Linux the XDG desktop portal's `org.freedesktop.appearance
color-scheme`, which GNOME's *Style* and Plasma's colour schemes set), and
show the saved pack's sibling for it. They switch live when the desktop
does. `theme` stays the pack you chose, so a desktop that turns light
again gets it back.

Siblings pair by name: `breeze` and `breeze-night`, `win95` and
`win95-dark`; a variant takes its family's sibling (`luna-olive` shows as
`luna-night`, Royale Noir; `aqua-graphite` as `aqua-night`); every CDE
palette darkens to Charcoal and every Window Maker scheme to Night Sky.
Packs with no sibling (Amiga, OS/2 Warp, BeOS, the high-contrast scheme)
stay as they are. A user pack `mine` pairs with a user pack `mine-night`.

Following the desktop also takes its **accent colour** (Plasma's and
GNOME 47's accent, the portal's `accent-color`) in themes whose engine
recolours around one, as Windows 10 and 11, macOS, Plasma, GNOME and
Material You do: Breeze's selection, focus, hover and default button, and
everything it mixes from them. Historical looks keep their own colours.

`UITK_COLOR_SCHEME=dark` (or `light`) stands in for the desktop, to try a
theme's other side without switching the desktop; `UITK_ACCENT=#e95420`
stands in for its accent. `UITK_THEME=<pack>`
shows exactly that pack and does not follow. Headless and offscreen apps
(tests, screenshots) never ask the desktop.

The desktop's **reduced-motion** setting (GNOME's *Animations* switch,
Plasma's animation speed at *Instant*, published as the portal's
`reduced-motion`) turns every uitoolkit animation off in every app,
whatever `reduceMotion` says; Settings shows a note under its Animations
switch while it does. Apps read the portal once at start (in the
background, alongside the font index) and follow it live.

Mode `0600`. Missing or invalid files yield the default theme,
`metal-ocean` (Metal, Ocean theme), in its own corners (`theme`), with
`classic` icons at `medium` size. A file without `iconSize` migrates to
medium.

Compound v0.11–v0.12.1 theme ids migrate on load:

| Old `theme` | Becomes |
| --- | --- |
| `dark-round-classic` / `dark-round-sharp` / `dark-round` | `theme: dark`, `corners: round` |
| `light-square-sharp` / `light-square-classic` / `light-square` | `theme: light`, `corners: square` |
| `dark` / `light` (already palette-only) | same name; corners from the `corners` field or `round` |

An explicit `"corners"` field wins over corners parsed from a compound
name. `"icons"` is always the chrome set (never inferred from
`-classic` / `-sharp` suffixes).

## Theme packs

**Embedded era packs** (Classic 95 through FlatLaf, each with a light
and a night twin where that era had one) ship in the binary and are
**not** auto-written to disk. `dark` / `light` are the Classic 95 twins
so existing `look.json` files keep working. Until the user picks one,
apps show the default theme, Metal (Ocean) (`style.DefaultThemeName`).

Settings lists them in the theme browser, a row each, by year (`1995 ·
Windows 95`); user exports show as `User · <name>` in that same list, and
the *My themes* filter is the list of the user's own.

```
$XDG_CONFIG_HOME/uitoolkit/themes/<name>/theme.json
# fallback: ~/.config/uitoolkit/themes/<name>/theme.json
```

```json
{
  "label": "ocean",
  "palette": "dark",
  "family": "dark",
  "bevel": "classic-3d",
  "colors": {
    "background": "#3c3c3c",
    "hotFill": "#000080",
    "hotBorder": "#000040"
  }
}
```

JSON only. Pack-level `corners` / `icons` fields are ignored.
**Export current theme…** asks for a name and writes the staged
**tokens**. Corners, icons, and icon size stay in `look.json`.
A legacy `{ "palette": "dark" }` file still loads.

When a **User** theme is staged, **Delete theme…** under the browser
confirms (Yes/No) then removes `themes/<name>/` from disk; with a
built-in pack staged it is there but grey. If the deleted pack was
selected, Settings falls back to the matching builtin palette (or the
other one if that name was a shadow). If `look.json` named the deleted
pack, it is rewritten to the fallback so the selection is not left
dangling. **Delete icon set…** under the icon chooser is the same
control for a **User** icon set (not premiere or drawn classic/sharp).

`ListThemes` lists Built-in era packs (not shadowed) then user packs
(sorted by name). `LoadTheme(name)` **prefers the user pack** when both
exist — a user `dark` overrides the embedded starter and is listed once
under User. Legacy compound ids map to `dark` / `light`. Era aliases
(`classic95`, `luna-dark`, …) resolve to the shipped id.

## Icon sets (PNG files)

Chrome icons are **not** embedded. Ship-in-repo sets live at
`icons/lucide`, `icons/phosphor`, `icons/tabler`, `icons/heroicons`,
and `icons/material-symbols` (24×24 + `name@2x.png` 48×48). Each pack
covers the typed `ToolIcon` stems plus a wide chrome / Mail / UI
vocabulary (`pen` and `download` are always present). Copy them
yourself after every pull that refreshes `icons/`:

```bash
mkdir -p ~/.config/uitoolkit/icons
cp -R icons/lucide icons/phosphor icons/tabler icons/heroicons icons/material-symbols ~/.config/uitoolkit/icons/
```

See [icons/README.md](../icons/README.md) for licenses, the full stem
list, attribution, and the `@2x` convention.

Settings offers them all in one chooser, in the order
`ListBuiltinIconSets` then `ListUserIconSets`:

- **Built-in** — drawn `classic` / `sharp`, plus the five premiere
  names when those folders are present under `icons/`
- **User** — any other `icons/<name>/` folder that contains at least
  one ToolIcon PNG

The strip under the chooser draws the staged set at the staged size, in
the full vocabulary.

When a premiere or user set is selected, a missing stem logs once and
paints **`no-icon`** (pack file, or the embedded placeholder if the
folder is not copied yet). Drawn classic is used only when the Icons
pref is explicitly `classic` or `sharp`. The toolkit tints
monochrome/alpha PNGs with the Look foreground / icon color.

The v0.12.0 `filled` / `outline` / `duotone` SVG folders are removed.

`ListIconSets` / `IconsDir` / `WithIcons` are the API. Changing icons
and Apply reloads other apps the same way as a theme change.

## Live reload in other apps

Source of truth is `look.json`. After Apply, every `Application` that
watches the file stats it (poll, ~300 ms while idle; every `PumpOnce`)
and, on change, reloads prefs and calls

```go
app.SetLook(style.WithAppearance(app.Look(), style.LoadAppearance()))
```

so theme, corners, icons, and icon size update without a restart. Display
scale and density on the current look are kept. A desktop that turns light
or dark reloads the same way while the appearance follows it, and
`Application.OnLookChange` callbacks run after every change (Settings
redraws its preview there).
`Application.ApplyAppearance(ap)` applies an `Appearance` without saving
it: theme, corners, icons, motion, the file dialogs, who draws the frame,
where the caption buttons go and following the desktop.
`Application.Appearance()` is its other half — the whole appearance the
app runs in — so an app that switches packs starts from it:

```go
ap := app.Appearance()
ap.Name = "win95"
app.ApplyAppearance(ap) // the frame, the caption buttons and motion stay
```

`style.LookAppearance(look)` reads back only what a look carries (pack,
palette, corners, icons); the preferences that are the application's come
back at their defaults, so an appearance rebuilt from it and applied would
reset them. `Application.Decorations()` and `CaptionButtons()` read the
two frame preferences on their own.

`Application.New` enables the watcher **by default when `Options.Look` is
nil** (that path already uses `PreferredLook()`). This is the least
friction for new apps:

```go
app := uitoolkit.New(uitoolkit.Options{}) // PreferredLook + look.json watch
```

Apps that pass an explicit look must opt in:

```go
look := uitoolkit.PreferredLook()
app := uitoolkit.New(uitoolkit.Options{Look: look, WatchLook: true})
```

Opt out (Settings does this so the picker stays staged until Apply):

```go
app := uitoolkit.New(uitoolkit.Options{DisableLookWatch: true})
```

`DarkLook` / `LightLook` fixtures stay static unless `WatchLook` is set,
so gallery screenshot pixels stay deterministic.

The gallery, Files, Notes and Inspector start from `PreferredLook` and
watch the file, and so should any application: an app's own chrome
preferences (a density, a layout) belong in an app file of its own and must
**not** rewrite `look.json`. The rule the mail client established is that
the only write back to `look.json` is a deliberate View → Dark / Light —
`SaveAppearance` of `WithPalette`, same corners and icons, opposite palette
starter. Gallery / screenshot fixtures keep explicit `DarkLook` /
`LightLook`.

Helpers for a live look:

```go
uitoolkit.LoadTheme("light")
uitoolkit.ListThemes()
uitoolkit.WithTheme(look, uitoolkit.ThemeLight)
uitoolkit.WithCorners(look, uitoolkit.CornersSquare)
uitoolkit.WithIcons(look, uitoolkit.IconSetLucide)
uitoolkit.WithIconSize(look, uitoolkit.IconSizeLarge)
uitoolkit.WithAppearance(look, uitoolkit.Appearance{Name: "ocean", Corners: uitoolkit.CornersSquare, IconSize: uitoolkit.IconSizeLarge})
```

`PreferredLook` is `LoadAppearance().Look()`. `LoadAppearance` reads
`theme`, `corners`, `icons`, and `iconSize` from `look.json`, resolves
the color theme with `LoadTheme` (user pack, then builtin), and applies
corners, the icon set, and icon size on top.

## API

| Symbol | Package |
| --- | --- |
| `ThemePack`, `ListThemes`, `ListBuiltinThemes`, `ListUserThemes` | `style` / `uitoolkit` |
| `LoadTheme`, `ExportTheme`, `SplitLookThemeName` | `style` / `uitoolkit` |
| `Appearance`, `ThemeName`, `CornerStyle`, `IconSetName`, `IconSize` | `style` / `uitoolkit` |
| `IconSetInfo`, `ListIconSets`, `ListBuiltinIconSets`, `ListUserIconSets` | `style` / `uitoolkit` |
| `LoadAppearance`, `SaveAppearance`, `AppearancePath` | `style` / `uitoolkit` |
| `ThemesDir`, `StarterName`, `DefaultThemeName` | `style` / `uitoolkit` |
| `PreferredLook`, `LookAppearance`, `WithIconSize`, `IconSizePixels` | `style` / `uitoolkit` |
| `Options.WatchLook`, `Options.DisableLookWatch` | `app` / `uitoolkit` |
| `Application.WatchingLook`, `Application.ReloadPreferredLook` | `app` |
| `Application.ApplyAppearance`, `Application.Appearance`, `Application.Decorations`, `Application.OnLookChange`, `Application.DesktopColorScheme`, `ColorSchemeEnv` | `app` |
| `ColorScheme`, `SchemeVariant`, `Appearance.Effective`, `SetDesktopColorScheme`, `DesktopReducesMotion` | `style` / `uitoolkit` |
| `AccentEngine`, `SetDesktopAccent`, `DesktopAccent`, `TakesAccent`, `CloneTokenMaps` | `style` |
| `DesktopPrefs`, `ReadDesktopPrefs`, `WatchDesktopPrefs` (the portal) | `platform` |
| `DrawToolIcon`, `DrawFileToolIcon` | `style` |
