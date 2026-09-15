# Theme packs

> Shapes per theme come from **theme engines** — see
> [theme-engines.md](theme-engines.md) for the architecture, the rules and
> how to write one. This page lists the packs and the pack schema.

`uitoolkit` skins are **token packs**, not palette-only recolors. A pack
names a bevel language, metrics, elevation, and per-state chrome colors.
`Classic` paints every control from those tokens. Corners, icon set, and
icon size stay independent `look.json` prefs.

## Built-in packs

103 packs from 29 engines, by the year their original shipped. Each one was
researched from published facts (design guides, SDK documentation, pixels
measured from screenshots of the originals); no code or pixmaps were copied.
Apps start in `metal-ocean` (`style.DefaultThemeName`) until the user picks
one; `UITK_THEME=<id>` runs any app in any pack.

| Id | Name | Year | Platform | Engine | What it reproduces |
| --- | --- | --- | --- | --- | --- |
| `system1` | System 1 | 1984 | Mac OS | `system7` | The 1984 Finder: 1-bit black on white, round-rect buttons, striped title bars and dithered greys. |
| `amiga13` | Workbench 1.3 | 1987 | Amiga | `amiga` | Workbench 1.3: blue and orange on four colours, white title bars and borders, highlights by complement. |
| `openlook` | OPEN LOOK | 1988 | Sun | `openlook` | Sun OpenWindows in 3D: obround buttons, menu marks, the elevator scrollbar and the pushpin. |
| `next` | NeXTSTEP | 1989 | NeXT | `next` | NeXTSTEP's four greys: black key-window titles, white-lit buttons, stippled scrollers, menus of raised cells. |
| `next-night` | NeXTSTEP Night | 1989 | NeXT | `next` | Not historical: NeXTSTEP never had a dark mode — its shapes and black titles on a dark grey ramp. |
| `hp-vue` | HP VUE | 1990 | Unix | `motif` | HP's Visual User Environment, CDE's parent: light-blue Motif windows, blue menus, salmon active frames. |
| `motif` | Motif | 1990 | Unix | `motif` | OSF/Motif and mwm as shipped: #c4c4c4 widgets, shadows derived by XmGetColors, diamond radios, CadetBlue frames. |
| `system7` | System 7 | 1991 | Mac OS | `system7` | System 7 in colour: the black-and-white controls in the Standard window colours, lavender, navy and grey. |
| `win-hotdog` | Hot Dog Stand | 1992 | Windows | `win31` | The infamous Windows 3.1 scheme: red windows, black captions, yellow desktop and no mercy. |
| `win31` | Windows 3.1 | 1992 | Windows | `win31` | Windows 3.1 on VGA: navy captions, white dialogs, black-edged bevelled buttons and the dotted focus. |
| `cde-alpine` | CDE Alpine | 1993 | Unix | `motif` | CDE's Alpine palette: tan windows, cool grey menus and text areas, periwinkle active frames. |
| `cde-broica` | CDE Broica | 1993 | Unix | `motif` | Broica, the palette CDE's Default copies, as four colour sets (MEDIUM_COLOR): blue-grey windows and menus. |
| `cde-charcoal` | CDE Charcoal | 1993 | Unix | `motif` | CDE's dark Charcoal palette: grey windows, slate-green menus, rose text areas, white labels. |
| `cde-crimson` | CDE Crimson | 1993 | Unix | `motif` | CDE's Crimson palette: sea-green windows, steel-blue menus, cream text areas, crimson active frames. |
| `cde` | CDE Default | 1993 | Unix | `motif` | The Common Desktop Environment's default palette: sand windows, teal menus, slate text areas, orange active frames. |
| `cde-desert` | CDE Desert | 1993 | Unix | `motif` | CDE's Desert palette: dusty blue windows, teal menus, white text areas, sand active frames. |
| `irix` | IRIX Indigo Magic | 1993 | Unix | `motif` | SGI IRIX Indigo Magic: #c1c1c1 IRIS IM widgets with SGI's graded shading, lavender fields, khaki active frames. |
| `amiga31` | Workbench 3.1 | 1994 | Amiga | `amiga` | Workbench 2.0 to 3.1: Intuition's grey 3D look, blue active borders, ridged string and cycle gadgets. |
| `openstep` | OPENSTEP | 1994 | NeXT | `next` | OPENSTEP 4: NeXTSTEP's greys with tab views, split slider knobs, pearl radio beads and a bolder close box. |
| `dark` | Classic 95 Dark | 1995 | Windows | `base` | The toolkit's own first look: Windows 95 bevels on charcoal with navy selection, and its default until Metal. |
| `light` | Classic 95 Light | 1995 | Windows | `base` | The toolkit's own first look: Windows 95 silver bevels with navy selection, and its default until Metal. |
| `win-highcontrast` | High Contrast Black | 1995 | Windows | `win95` | Accessibility scheme: black, white, cyan and green. |
| `win95` | Windows 95 | 1995 | Windows | `win95` | The four-colour bevel, navy captions, dotted focus. |
| `win95-dark` | Windows 95 Dark | 1995 | Windows | `win95` | Classic 95 shapes in a dark grey scheme. |
| `os2warp` | OS/2 Warp 4 | 1996 | IBM | `os2` | IBM's Warp 4 desktop: #CCCCCC dialogs, grooved two-pixel bevels, pastel notebook tabs, a sunken title well and halftoned disabled items. |
| `platinum` | Platinum | 1997 | Mac OS | `platinum` | Mac OS 8 and 9: grey bevels, ridged title bars, the Lavender accent. |
| `platinum-lime` | Platinum Lime | 1997 | Mac OS | `platinum` | Mac OS 8 with its Lime accent colour on menus, thumbs and progress bars. |
| `wmaker-default` | Window Maker | 1997 | Window Maker | `next` | Window Maker's Default theme: slate title gradients, grey gradient menus, dock-tile tool bars. |
| `wmaker-openstep` | Window Maker OpenStep | 1997 | Window Maker | `next` | Window Maker's OpenStep theme: midnight-blue diagonal titles over light grey menus. |
| `beos` | BeOS | 1998 | Be | `beos` | BeOS R5: the yellow window tab, soft grey bevels on #D8D8D8 and blue keyboard-focus underlines. |
| `metal-steel` | Metal (Steel) | 1998 | Java | `metal` | Swing's Java look and feel: grey-blue flush 3D borders, bumps, bold labels. |
| `wmaker-night` | Window Maker Night Sky | 1998 | Window Maker | `next` | Window Maker's Night Sky: black-to-teal title gradients, dark menus lit in warm cream. |
| `win98` | Windows 98 | 1998 | Windows | `win95` | Windows 95 with gradient captions and flat hot-tracked menus. |
| `wmaker-steelbluesilk` | Window Maker SteelBlueSilk | 1999 | Window Maker | `next` | Window Maker's SteelBlueSilk: multi-stop steel-blue silk titles, dark silk menus, violet dock tiles. |
| `win2000` | Windows 2000 | 2000 | Windows | `win95` | Warm grey #d4d0c8, blue gradient captions. |
| `aqua` | Aqua | 2001 | Mac OS | `aqua` | Mac OS X 10.0–10.4: blue gel pills, pinstripes, traffic lights. |
| `aqua-graphite` | Aqua Graphite | 2001 | Mac OS | `aqua` | The Graphite appearance: the same gel in blue-grey. |
| `aqua-night` | Aqua Graphite Night | 2001 | Mac OS | `aqua` | Not historical: Aqua never had a dark mode — graphite gel on charcoal pinstripes. |
| `luna` | Luna Blue | 2001 | Windows | `luna` | Windows XP's default: blue captions, beige 3D face, orange hot glow, Office 2003 menus. |
| `luna-olive` | Luna Olive Green | 2001 | Windows | `luna` | XP's HomeStead scheme: olive captions, green-framed buttons, copper progress. |
| `luna-silver` | Luna Silver | 2001 | Windows | `luna` | XP's Metallic scheme: silver captions with black titles, lavender-grey chrome. |
| `keramik` | Keramik | 2002 | KDE | `keramik` | KDE 3.1's default: gel buttons with a white sheen, blue gel scroll handles, rounded tabs and the bubble title bar. |
| `bluecurve` | Bluecurve | 2002 | Red Hat | `bluecurve` | Red Hat Linux 8 and Fedora Core: square bevelled greys, blue ticks and menus, diagonal grips. |
| `brushed-metal` | Brushed Metal | 2003 | Mac OS | `aqua` | Panther's textured windows: brushed aluminium and darker gels. |
| `metal-ocean` | Metal (Ocean) | 2004 | Java | `metal` | Java 5's Metal theme: soft blue gradients on buttons, scroll bars and sliders. |
| `plastik` | Plastik | 2004 | KDE | `plastik` | KDE 3.5's default: flat gradient surfaces in a soft contour, the blue mouse-over highlight, dotted grips and striped progress. |
| `luna-royale` | Royale | 2004 | Windows | `luna` | Media Center's glossy Energy Blue captions, steel buttons and Office XP menus. |
| `clearlooks` | Clearlooks | 2005 | GNOME | `clearlooks` | GNOME 2.12's default: rounded gradient buttons, the striped candy progress bar, blue tab stripes. |
| `luna-night` | Royale Noir | 2005 | Windows | `luna` | Royale's black glossy variant, as a dark scheme with Office XP-style menus. |
| `plastique` | Plastique | 2006 | Qt | `plastik` | Qt 4's port of Plastik: neutral contours, selection-tinted frames, chunked progress and right-aligned form labels. |
| `human` | Human | 2006 | Ubuntu | `clearlooks` | Ubuntu's Clearlooks-born look: warm greys, glassy buttons with an orange glow, orange progress and checks. |
| `cleanlooks` | Cleanlooks | 2007 | Qt | `clearlooks` | Qt 4's clone of Clearlooks: the same greys and blue, Qt's cut corners, dotted focus and striped progress. |
| `aero` | Aero | 2007 | Windows | `aero` | Windows Vista and 7: glass captions, two-tone glassy buttons with a blue glow, Explorer's light-blue selection boxes. |
| `nimbus` | Nimbus | 2008 | Java | `nimbus` | Java 6's vector look: glossy rounded controls, blue focus glow, orange progress. |
| `oxygen` | Oxygen | 2008 | KDE | `oxygen` | KDE 4's Oxygen: glossy slabs with soft shadows, the blue hover glow and a window-wide gradient. |
| `aero-basic` | Windows 7 Basic | 2009 | Windows | `aero` | Windows 7 without Aero Glass: the same controls under opaque light-blue frames. |
| `fusion` | Fusion | 2012 | Qt | `fusion` | Qt's cross-platform style: 2px gradient buttons, highlight focus outlines, square scroll sliders. |
| `fusion-night` | Fusion Dark | 2012 | Qt | `fusion` | Fusion in the widely used community dark palette. Not historical: Qt's 2012 Fusion shipped a light palette only. |
| `win8` | Windows 8 | 2012 | Windows | `metro` | The flat desktop of Windows 8: square 1px-bordered controls, dotted focus, solid blue selection, coloured window frames. |
| `dracula` | Dracula | 2013 | Dracula | `web` | Dracula: #282a36 panes, darker sidebars and bars, #343746 popovers, the Purple #bd93f9 accent and #815cd6 focus rings. |
| `adwaita-gtk3` | Adwaita (GTK 3) | 2014 | GNOME | `adwaita` | GNOME 3.14's Adwaita: soft gradient buttons in grey borders, blue selections, pill scroll bars. |
| `material` | Material | 2014 | Google | `material` | Material Design 2: contained buttons on 2dp shadows, outlined fields with floating labels, #6200EE. |
| `material-night` | Material Dark | 2014 | Google | `material` | Material Design 2's dark theme: #121212 surfaces lifted by elevation, primary #BB86FC. |
| `breeze` | Breeze | 2014 | KDE | `breeze` | KDE Plasma 5's flat style: 3px frames mixed from the text colour, #3daee9 focus and hover, round slider handles. |
| `breeze-night` | Breeze Dark | 2014 | KDE | `breeze` | Plasma 5's Breeze in the Breeze Dark colour scheme (Plasma 5.27 values). |
| `yosemite` | OS X Yosemite | 2014 | Mac OS | `macos` | Yosemite to Mojave: flat white buttons, the blue accent, overlay scrollers, vibrant menus. |
| `win10` | Windows 10 | 2015 | Windows | `metro` | Windows 10's flat desktop: accent-blue focus, thin scroll bars, pill toggles, rectangular slider thumbs, Explorer's light accent selection. |
| `nord-night` | Nord | 2016 | Nord | `web` | Nord's Polar Night: #2e3440 backgrounds, #3b4252 raised UI, #434c5e selections, the Frost #88c0d0 accent with dark text on it. |
| `nord` | Nord Light | 2016 | Nord | `web` | Nord's Snow Storm scheme: #eceff4 backgrounds, #d8dee9 raised UI and borders, Polar Night text, the Frost #5e81ac accent. |
| `win10-night` | Windows 10 Dark | 2018 | Windows | `metro` | Windows 10's 2018 dark mode: the same flat shapes on dark greys with the accent blue. |
| `flatlaf-darcula` | Darcula | 2019 | Java | `flatlaf` | FlatDarculaLaf: IntelliJ's Darcula — a 2px focus ring outside every border, triangle arrows. |
| `flatlaf-night` | FlatLaf Dark | 2019 | Java | `flatlaf` | FlatDarkLaf: charcoal faces, the dark blue bold default button, grey ticks. |
| `flatlaf` | FlatLaf Light | 2019 | Java | `flatlaf` | FlatLightLaf: flat white faces with an arc of 6, a 2px accent focus border, underlined tabs. |
| `adwaita` | Adwaita | 2020 | GNOME | `adwaita` | libadwaita (GNOME 42+): flat washed buttons, accent checks, pill switches, overlay scroll bars. |
| `adwaita-night` | Adwaita Dark | 2020 | GNOME | `adwaita` | libadwaita's dark style: the same flat shapes on charcoal, sky-blue accent text. |
| `bigsur` | macOS Big Sur | 2020 | Mac OS | `macos` | Big Sur: rounder controls, the unified toolbar, rounded inset selections in lists and sidebars. |
| `bigsur-night` | macOS Big Sur Dark | 2020 | Mac OS | `macos` | Big Sur's dark appearance: grey faces on charcoal, the brighter blue accent. |
| `tokyonight` | Tokyo Night | 2020 | Tokyo Night | `web` | Tokyo Night: #1a1b26 panes in #16161e chrome with near-black borders, #202330 selections, the #3d59a1 accent. |
| `tokyonight-storm` | Tokyo Night Storm | 2020 | Tokyo Night | `web` | Tokyo Night Storm: #24283b panes in #1f2335 chrome, #2c324a selections, the #3d59a1 accent. |
| `catppuccin-frappe` | Catppuccin Frappé | 2021 | Catppuccin | `web` | Catppuccin Frappé: the lightest dark flavour, #303446 Base with Surface popovers and Blue #8caaee. |
| `catppuccin-latte` | Catppuccin Latte | 2021 | Catppuccin | `web` | Catppuccin's light flavour: Base #eff1f5 panes, Mantle sidebars, Blue #1e66f5 accent, Lavender focus. |
| `catppuccin-macchiato` | Catppuccin Macchiato | 2021 | Catppuccin | `web` | Catppuccin Macchiato: #24273a Base, deeper Mantle and Crust bars, Blue #8aadf4 with Lavender focus. |
| `catppuccin-mocha` | Catppuccin Mocha | 2021 | Catppuccin | `web` | Catppuccin Mocha, the darkest flavour: #1e1e2e Base, #11111b Crust bars, Blue #89b4fa and Lavender #b4befe. |
| `material3` | Material 3 | 2021 | Google | `material` | Material You: tonal palettes from one seed colour, pill buttons, state layers, the big switch. |
| `material3-night` | Material 3 Dark | 2021 | Google | `material` | Material You's dark scheme from the same seed: tone 80 primaries on tone 10 surfaces. |
| `rosepine` | Rosé Pine | 2021 | Rosé Pine | `web` | Rosé Pine: #191724 base, #1f1d2e surfaces, #26233a overlays for hover and selection, iris #c4a7e7. |
| `rosepine-dawn` | Rosé Pine Dawn | 2021 | Rosé Pine | `web` | Rosé Pine's light variant: #faf4ed base, #fffaf3 surfaces, highlight-med selections, iris #907aa9. |
| `rosepine-moon` | Rosé Pine Moon | 2021 | Rosé Pine | `web` | Rosé Pine Moon: #232136 base, #2a273f surfaces, #393552 overlays, iris #c4a7e7. |
| `tokyonight-day` | Tokyo Night Day | 2021 | Tokyo Night | `web` | Tokyo Night Day: #e1e2e7 panes with #d0d5e3 sidebars and popups, blue #3760bf text, the #2e7de9 accent. |
| `fluent` | Fluent | 2021 | Windows | `fluent` | Windows 11's Fluent (WinUI 3): 4px and 8px corners, translucent control fills over Mica, accent check boxes, pill selection indicators, the two-colour focus visual. |
| `fluent-night` | Fluent Dark | 2021 | Windows | `fluent` | Windows 11's dark theme: Fluent's shapes on #202020 Mica with the light-blue accent and black text on it. |
| `alucard` | Alucard | 2023 | Dracula | `web` | Dracula's light variant: cream #fffbeb panes, #efeddc popovers, the Purple #644ac9 accent and the #815cd6 focus ring. |
| `primer` | Primer | 2023 | GitHub | `web` | GitHub's Primer: 6px controls in #d1d9e0 hairlines, the green primary button, the coral underline under the current tab, rows marked with an accent bar. |
| `primer-night` | Primer Dark | 2023 | GitHub | `web` | GitHub's dark theme: Primer's shapes on #0d1117 with #3d444d hairlines and #1f6feb for focus and checks. |
| `primer-dimmed` | Primer Dark Dimmed | 2023 | GitHub | `web` | GitHub's dark dimmed theme: softer #212830 greys, muted green buttons and the #316dca accent. |
| `geist` | Geist | 2023 | Vercel | `web` | Vercel's Geist: gray-1000 primary buttons, alpha-ring hairlines, a 2px blue ring beyond a 2px gap, 2px underlined tabs. |
| `geist-night` | Geist Dark | 2023 | Vercel | `web` | Geist's dark theme: #0a0a0a on black sidebars, near-white primary buttons and the #52a9ff focus ring. |
| `sourcegit` | SourceGit | 2024 | SourceGit | `web` | SourceGit's Avalonia look: square fields, bold 3px flat buttons, the 1px accent tab pipe, unfilled check boxes with an accent tick, joined sidebar selections. |
| `sourcegit-night` | SourceGit Dark | 2024 | SourceGit | `web` | SourceGit's dark theme: #252525 windows round #1c1c1c contents, the same Avalonia controls and the OS accent. |
| `shadcn` | shadcn/ui | 2025 | shadcn/ui | `web` | shadcn/ui's new-york style in neutral: near-black primary buttons, #e5e5e5 hairlines, a 3px grey focus ring outside, segmented tabs. |
| `shadcn-night` | shadcn/ui Dark | 2025 | shadcn/ui | `web` | shadcn/ui's dark neutral theme: #0a0a0a with white-alpha hairlines and near-white primary buttons. |
| `linear` | Linear | 2026 | Linear | `web` | Linear's light theme: warm #f9f9fa greys, 4px controls and 8px inputs, the indigo accent, compact rounded tabs. |
| `linear-night` | Linear Dark | 2026 | Linear | `web` | Linear's dark theme: #121213 content by a #09090a sidebar, hairlines barely lighter, the #5e6ad2 indigo. |

Aliases (load-only, not listed twice): `classic95` → `light`,
`classic95-dark` → `dark`, `luna-dark` → `luna-night`, and the other
`*-dark` ids → the matching `*-night` pack.

`dark` / `light` are the Classic 95 twins, the default before 2026-09-15;
`look.json` files that name them keep working.

## Token schema

`~/.config/uitoolkit/themes/<name>/theme.json`:

```json
{
  "label": "Luna",
  "era": "Luna",
  "family": "light",
  "palette": "light",
  "bevel": "luna-hottrack",
  "elevation": 0,
  "metrics": {
    "radius": 2,
    "radiusSmall": 1,
    "bevelDepth": 1,
    "gutterWidth": 10,
    "scroll": 16,
    "controlH": 32,
    "comboH": 28
  },
  "colors": {
    "background": "#ece9d8",
    "surface": "#ece9d8",
    "hotFill": "#c1d2ee",
    "hotBorder": "#316ac5"
  }
}
```

Legacy `{ "label": "ocean", "palette": "dark" }` still loads: family is
taken from `palette` / `theme`, tokens fall back to the Classic 95 pack
of that family.

`corners` and `icons` in a pack file are ignored.

### Bevel

| Value | Language |
| --- | --- |
| `none` | Flat fill + 1px border; Breeze / FlatLaf focus ring |
| `classic-3d` | Motif / Win95 highlight + shadow edges (depth 1–2) |
| `luna-hottrack` | Pale fill + thin border on hot / press / focus |
| `soft-shadow` | Raised face + drop shadow (Aqua / Material elevation) |
| `fluent-accent` | Wash + accent outline / bottom bar on focus |

### Controls painted from tokens

Button, ToolButton / ToolToggle, MenuBar titles, Menu / Popup / Context
rows, ListView / TreeView / TableView rows, Tab / TabBar, scrollbar
track / thumb / arrow wells, ComboBox field + dropdown, CheckBox /
Radio, TextField / NumberField / TextArea focus, Splitter grip, ToolBar
strip.

## API

```go
pack, _ := uitoolkit.LoadTheme("luna")
look := pack.Look()
tok := style.LookTokens(look)
_ = tok.Bevel
```
