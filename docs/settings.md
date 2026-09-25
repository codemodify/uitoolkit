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
go run ./cmd/uitoolkit-settings -page appearance  # name a section of the page
go run ./cmd/uitoolkit-settings -headless         # settings.png in cwd
go run ./cmd/uitoolkit-settings -stage aqua -screenshot out/
go run ./cmd/uitoolkit-settings -version          # the toolkit version, and the only place it is stated
tools/shots/demos.sh                              # docs/screenshots/settings.webp and the other demos
```

`-page` names a **section of the one page**: `theme` (the default, the
column on the left), `behaviour`, `preview`, `files`. **Nothing on the
page scrolls out of reach**, so the flag names what is already on screen
and changes nothing: it says which part of the page you came for and the
page opens the way it always opens. The machinery that scrolled the
column to a named section (`openAt`, `scrollTo`) is **gone** — it went
with the column's scrollbar, because the column does not scroll any more
and code that cannot fire is not kept. What is kept is
`SettingsPage(name)`, which resolves a name to a section, and the flag
itself, because a flag that has stopped mattering must still not be an
error. The names of the four pages Settings used to have still
resolve, because they are in scripts, in the atlas tooling and in the
docs of two releases: `themes` and `packs` (and `packs & icons`, `theme
packs`) open **Theme**, `appearance`, `icons` and `corners` open the
**Preview**, which is where the shape and the icon set are chosen now,
`about` opens **Files**, and `desktop` and `colours` open **Behaviour**,
because following the desktop's colours is one of the five options in
that row. An unknown name is the top of the page.

## The page

There is one page and no navigation. Settings used to be four pages
behind a sidebar — Themes, Appearance, Packs, About — and they were four
answers to one question: what does this desktop look like. The window is
now a splitter: **the theme browser down a column on the left**, and
**the preview on the right**, with **Apply** pinned at the foot, outside
both. Neither side scrolls; the only thing on the page that does is the
list of packs, inside itself.

The line through the page is no longer between kinds of choice but
between the browser and the thing it is browsing for. **The preview sets
what it shows**: the icon set, the size its glyphs are drawn at and the
shape of its corners are three combo boxes on a bar of their own at the
head of the previewed window, over the sample's menu bar, each behind
the word that says what it sets. Everything else that is not the list of
packs stands with the preview: the **five on/off options in a row over
it**, and the three **config paths** under it.

The column is about 300 logical pixels wide whatever the window and the
display scale are, and it keeps that share while the window is resized
(`Window.OnResize`). Dragging the sash replaces the split: a dragged
split stays through every resize after, and survives Apply. It was 240
when the column was the browser and nothing else the first time; it is
the browser and nothing else again, and it has kept 300, because what
sets it now is a row reading `1995 · Windows 95` and the word on the
Export button — 240 elides the one and wraps the other.

### The preview, on the right

A small but fully interactive application window (menu bar, tool bar,
tabs with every control, tree, table, dialogs, status bar) painted
entirely in the staged theme, frame, caption and all. Its caption names
the pack it is really drawing, which is where you see that *Breeze* is
showing as *Breeze Dark* because the desktop asked for dark.

It is a `widgets.ThemeScope`, so Settings itself keeps the applied look.
Staging a pack switches it where it stands — the caret stays in the
search field, the focus on the list, and the list where it was scrolled.

**It sets three of the things it shows**, on a bar of its own —
`Icons [Classic ▾]  Size [24 ▾] │ Corners [Theme shape ▾]`.

- **The icon set**, **the size its glyphs are drawn at** and **the shape
  of the window's corners**, in that order, each behind a short word
  (`widgets.ToolLabel`, `widgets.ToolWidget`). The size box lists **16 /
  24 / 32**, the way a word processor's size box lists numbers; the
  corner box lists *Theme shape*, *Round*, *Square*. The sample's tool
  bar below is drawn in whatever the first two choose, and the window's
  own frame is cut to what the third chooses, so the window is the
  preview of all three.

**The bar is on top, over the menu bar**, and that is the whole of how
it says it is not part of the sample. A tool bar belongs under the menu
bar of the window it commands; a second strip *below* the sample's would
read as the same application's second row of tools, which is exactly the
mistake the last arrangement invited, when the icon choosers rode at the
free-space end of the sample's own bar and were taken for its style and
size boxes. Nothing in any application sits above its menu bar. What
does is the frame around it, in the same voice as the caption over it —
which does not name a document either but says *Preview — Windows 95*.
The seam is clean: caption and settings bar are Settings talking, and
everything from the menu bar down is the sample.

**The words are short and never disagree with what is read out.** On
screen: *Icons*, *Size*, *Corners*. To a screen reader: *Icons*, *Icon
size*, *Window corners* — each one contains the word on the screen, so
the spoken name carries the visible one and adds the context the eye
gets from where the box stands. Every chooser also has a tooltip that
says in full what it sets and that it is real.

**When the pane is too narrow** the bar sheds its words from the right,
one at a time, and keeps the three choosers whole (the tool bar rule at
`widgets.ToolStretch`: a bar drops tools and words before controls). At
1024×860 the preview pane is about 670 px and all three words show; at
the 720×520 minimum it is about 366 px, the three boxes alone need about
300 of it, and every word is dropped — the tooltips and the accessible
names are what is left. A word that is not on the bar is not in the
accessibility tree either.

**Everything else in that window is a sample**: *Send* sends nothing,
the tree lists a mailbox nobody has, the check boxes tick themselves.
The corners were three radio items in the sample's **View ▸ Window
corners** for a release — the place an application has always kept what
its window looks like — and nobody found them there, because a preview's
menus are the one part of it a reader takes for make-believe. A menu
hides; a labelled bar does not.

**The sample's own tool bar is the sample's again**: New, Open, Save │
Cut, Copy, **Paste** │ Pen │ Send. Paste is back — it was the button the
choosers cost this bar when they rode on it — and the bar ends in free
space, so at the narrowest window it sheds *Send* rather than showing
half a button cut off by the window frame.

**The window's own furniture is flush**: menu bar, tool bar, document,
status bar, with no air between them, the way a real window is. (There
were eight pixels between each for a long time. No window has those, and
they were the room the settings bar needed: the sample's document area
is 22 px shorter than before, not 64.)

**Nothing else is allowed in the preview's own box**, and only two
things share its pane: one folding row of options and three lines of
paths. The widget gallery used to sit under it in a
second splitter (those are the widgets the tour shows over its Controls,
Views and Documents pages, and under the preview they halved it to
answer a question the preview had already answered). A strip of fifteen
stock icons sat above it for a while when the four pages were merged,
and it read well at 1024 — but the strip keeps its own height and the
preview takes what is left, so at the 720×520 minimum the glyphs folded
onto four lines and left the preview a caption and a menu bar. The
preview is far the biggest thing in its pane at every size (about **82%**
of it at 1024×860 and **61%** at the 720×520 minimum), and the only way
to promise that is to keep what is not the preview down to one folding
row and three lines. `TestSettingsPageHoldsAtEverySize` is that promise,
in four packs at two scales at both sizes.

### Over and under the preview

**The five options**, in one row over the preview, each a check box
wearing a short word:

```
☑ Animations  ☐ OS open/save dialogs  ☐ OS borders  ☐ Theme buttons  ☐ OS colors
```

| On the box | To a screen reader | What it is |
| --- | --- | --- |
| **Animations** | Animations | Hover fades, the default button's pulse, busy bars (GTK's `gtk-enable-animations`). While the desktop itself asks for reduced motion it says so, because the desktop's setting wins over the preference |
| **OS open/save dialogs** | OS open/save dialogs: the desktop's own Open and Save | KDE's and GNOME's own Open and Save, through the XDG portal, instead of the themed ones |
| **OS borders** | OS borders: the desktop's title bar and borders | Chromium's switch. Windows that draw their own title bar (Mail's, with its tool bar in it) get the desktop's instead |
| **Theme buttons** | Theme buttons: the caption buttons where the theme puts them | Close, minimise and maximise in the theme's era's order (the Mac's traffic lights on the left) rather than the desktop's |
| **OS colors** | OS colors: follow the desktop's light or dark mode and its accent | The pack shows its sibling to match the desktop — a chosen *Breeze* draws as *Breeze Dark* — recoloured around the desktop's accent where the engine takes one |

**Four of them spent a release in the column**, as switches with a line
of prose each, where 300 logical pixels elided the labels themselves
(`Use the desktop's file dia…`); the fifth, the colours one, stood here
alone. They are one row now, over the window all five are about, and
they are **furniture you glance at rather than documentation**: the
sentence that was a line under each box is its tooltip and its
accessible description instead. Over the preview, every line of prose is
a line off the window the whole page is for.

**The short word is inside the spoken name**, never beside it — the same
rule the preview's own settings bar follows a hand's width below, where
*Size* is read out as *Icon size*. What the eye gets from the row the box
stands in, the ear gets from the rest of the name. `Dialogs` on its own
would have been a riddle; `OS open/save dialogs` read as *OS open/save
dialogs: the desktop's own Open and Save* is one name for one control.

**Three of the five say `OS`** — *OS open/save dialogs*, *OS borders*,
*OS colors* — because those three are the ones that hand something back
to the desktop, and a reader should not have to work out that *System
frames* and *OS colors* were the same kind of thing. The other two,
*Animations* and *Theme buttons*, are the toolkit's own behaviour and
say nothing about whose.

**They are check boxes, not switches**, for two reasons that agree. The
honest one: nothing on this page takes effect when it is touched — Apply
writes `look.json` and nothing else does — and a switch is the control
that says *this is live now*, while a check box is the control that says
*this is what I am asking for*. The measured one: a switch's pill is 42
logical pixels of chrome before its word, and the right-hand pane is 392
of them at the 720×520 minimum. Five switches fold onto **three** lines
there and leave the preview 53% of the pane; five check boxes fold onto
**two** and leave it 61%, and onto **one** line at 1024×860, where the
old arrangement needed two (a switch and its line of prose).

**The row folds, it does not shed.** It is a `widgets.Wrap` (Qt's flow
layout, GTK's `FlowBox`): one line while the pane is wide, two when it is
not, and never a control cut off by the window frame the way a tool
bar's shedding would leave one — a setting nobody can reach is worse
than a second line. The order is the order they are read, and the fold
follows from it: all five fit the 697-pixel row of a 1024×860 window
with six pixels to spare, and the 453-pixel row of a 720×520 one takes
the first three and gives *Theme buttons* and *OS colors* a second line.
The frame pair (*OS borders*, *Theme buttons*) stay next to each other
across that fold, and *OS colors* is last, nearest the window whose
caption it changes.

**Files** — the three paths Settings reads and writes, one line each:
the **prefs** file, the **themes** directory and the **icons**
directory, written with `~` for the home directory and elided rather
than wrapped when the pane is narrow. It is under the preview because it
is the only part of the page that changes nothing — it says where what
the rest of the page changed ends up — and it is three lines rather than
the six it had (a line of prose and a three-row text box each) because
every line here is a line off the preview. The paths are labels now, not
text boxes: a box that can be selected from keeps three rows and grows a
scrollbar of its own as soon as a path is longer than the pane.

**Nothing on those three lines does anything.** `Delete icon set…` rode
at the end of the icons line for a release and is gone: it was the last
thing left of the old Packs page, it acted on a set chosen two inches
away on the preview's own bar, and it put a button that destroys a
directory on the one part of the page that was meant to change nothing.
Nothing replaced it — a set is a folder, `rm -r` removes it, and
`style.DeleteUserIconSet` is still there for an application that wants
the operation.

### The column, on the left

**Four bare controls**, in this order, and nothing around them:

```
[ Search themes            ]
[ All decades            ▾ ]
 1995 · Windows 95
 1995 · Windows 95 Dark
 …                          ← to the foot of the window
[  Export current theme…   ]
```

- a **search field** — a pack is found by its id, name, year, family,
  engine or what its summary says; every word has to match, Return stages
  the first hit, Escape empties the field;
- the **decade filter** — *All decades*, one decade, or *My themes*;
- the **list of packs** that pass both, by year (`1995 · Windows 95`),
  user exports as `User · <name>`;
- **Export current theme…**, which writes the staged look out as a theme
  pack of the user's own. It was on the Packs page, which needed a second
  copy of the browser to say which pack it meant; it is under the only
  browser there is now, and the pack it writes appears in that same list
  a line later.

**The list runs to the foot of the column.** It takes every pixel the
three controls around it leave, at every window size, and scrolls its 129
rows inside itself. It was held to 252 logical pixels — nine rows — inside
a column that scrolled, which was the only way to have both a list with a
scrollbar and a column with one; a view as tall as all 129 rows would
have made the column ten screens long. What that really bought was **two
scrollbars an inch apart and about 250 pixels of nothing under the
buttons**. It shows **25–27 packs at 1024×860** and **14–15 at the
720×520 minimum** now, at scale 1 and at 1.75 alike.

**The column does not scroll at all.** Nothing in it can overflow: three
controls of fixed height and a list that takes what they leave — the same
promise the preview makes on the other side of the sash. Settings owns no
`ScrollView` any more, which is why `openAt` and `scrollTo` went with it
(see `-page`, above).

**There is no group box and no heading.** The *Theme* legend around these
four, the paragraph over the search field, the **Settings** heading at the
top of the column and the **version** label under it are all gone. A
legend reading *Theme* over the only list on the page said no more than
the list says by being a list of themes; a window says what application
it is in its title bar, which the desktop draws and which already reads
*uitoolkit - Settings*; and the heading was the last of the four-page
sidebar, which needed something at the top of the column to own the pages
under it.

**What names the column instead** is on the two controls themselves:
the list is called **Themes** and the field is called **Search themes**,
which is where those names belonged all along — a group box's legend
names a *group*, not the list inside it — and `a11y.Check` fails an
unnamed list outright, whatever is written above it.
`TestSettingsIsAccessible` checks both names and that no group called
*Theme* is left claiming to be one of them.

**The window opens on the list, not on the search field.** A window with
no initial focus of its own starts on the first control a click would
focus, which is the field; a focused field shows its caret instead of its
placeholder, so the top of a captionless column was a bare empty box and
*Search themes* — the one word on the page that says what the column is —
was the one word the page would not draw. `Window.SetInitialFocus` puts
the focus on the list, which shows it, and which is the better place to
land anyway: this column is a browser, and the arrow keys walk it and
stage what they reach.

**The version** used to be the label under that heading, and it was the
only place Settings stated it. It is **`uitoolkit-settings -version`**
now. It is not in the **Files** block: that block is under the preview,
and a fourth line in it would be a line off the window the page is about
— and would move the Theme Atlas crop for a fact about the build.

That is the whole column. **Behaviour** was under it for a release —
four switches, each with a line of prose — and is the row of five
options over the preview now; see above.

### What went

The **Packs** page is gone. Its two lists were a second theme browser and
a second icon chooser; the browser and the chooser are on this page, so
the lists were a duplicate. Both its actions have now gone the same way:
Delete icon set… went round the page for a release before going
altogether, and Delete theme… followed it. **Export current theme…** is
all that is left of that page, under the browser that says what it would
write.

The **sidebar** is gone with the pages, and with it the `Pages` list a
screen reader used to drive. There is nothing to drive: one Tab ring
holds the whole application.

The section called **Shape and weight** is gone too, and nothing is left
of it in the column: corners, the icon set and the icon size are all
answered by the preview now, and the strip of fifteen glyphs under them
was a picture of a tool bar standing in for the real one two inches to
its right.

The **Behaviour** panel is gone from the column, and with it the last
thing in it that was not the theme browser: its four switches are four
check boxes in the row over the preview, with the colours one, under
short words. The column is the browser alone, which is what it was
before the four pages became one.

**`Delete icon set…` is gone**, from the column where it started and
from the icons line of **Files** where it spent a release.
`style.DeleteUserIconSet` stays — it is public toolkit API and an
application may want it — but Settings does not call it any more, and
nothing on the **Files** block does anything at all now.

**`Delete theme…` is gone**, the same call one release later. It stood
under the browser, grey for all 129 built-in packs and live for the
handful the user had exported, and what it did was destroy a directory
after a Yes/No. A pack is a folder; `rm -r ~/.config/uitoolkit/themes/<name>`
removes it. `style.DeleteUserTheme` and `style.AfterUserThemeDeleted`
stay, and so do `uitoolkit.DeleteUserTheme` and
`uitoolkit.AfterUserThemeDeleted` beside them — public toolkit API, and
an application that offers to remove a pack needs both halves, the delete
and the rewrite of an appearance that named it. Settings calls neither.
Everything the app kept for that button went with it: the `delTheme`
field, the branch of `showStaged` that greyed it, and `indexTheme`, its
only caller.

The **Theme group box** is gone, with the paragraph under its legend,
and so are the **Settings heading** and the **version label** at the top
of the column. See *The column, on the left*.

The **View ▸ Window corners** submenu is gone: the corners are the third
box on the settings bar. So is the **free-space end of the sample's tool
bar** as a place to keep a setting — the two choosers that rode there
are on the bar above, and *Paste*, which was taken off that bar to pay
for them, is back.

## Staged and applied

The preview shows what is **staged**. Nothing is written until **Apply**,
which saves `look.json` and switches Settings and every app that watches
the file. Apply is the only button and the only thing in the row under
the page: it sits pinned at the **right** of it, outside the splitter
and outside the column, so the one thing that writes anything is on
screen at every window size. The line that used to
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
x 317, y 44, 697 × 647        crop box (317, 44) – (1014, 691)
```

`TestSettingsPreviewPanelKeepsItsPlace` pins those numbers; a layout
change that moves them fails it, and the new ones belong here. (They were
`x 504, y 175, 510 × 387` before the gallery went under the preview,
`569 × 381` until the preview took 62% of the split instead of 55%,
`x 445, y 58, 569 × 429` until the window's title bar row and status bar
came off, `569 × 481` until the gallery came out from under the preview
and it took the whole right-hand pane, `x 445, y 10, 569 × 782` until the
icons took the head of the page, `569 × 630` until the four pages became
one, `x 317, y 10, 697 × 790` until the colours switch went over the
preview and the paths under it, and `x 317, y 66, 697 × 620` until the
behaviour options joined that switch in one row. Nothing that has since
happened to the column on the left has moved them.)

**Changes to the column on the left do not move them.** The splitter's
ratio is worked out from the *window's* width, not from what the column
holds, and the column's contents have never had a minimum that could push
the sash: the browser's list takes what it is given at any width. Taking
the group box, the heading, the version and Delete theme… out of the
column, and running the list to the foot of it, was re-measured in the
eight packs `TestSettingsPreviewPanelKeepsItsPlace` walks and moved
nothing — the crop is `697x647+317+44` before and after.

**What the preview carries inside itself does not move them; what stands
over it does.** The settings bar is inside the panel, and the panel takes
whatever is left over it and under it, so the crop did not move when that
bar went in — it was re-measured then, in the eight packs
`TestSettingsPreviewPanelKeepsItsPlace` walks, and had not changed. The
row of options is *outside* the panel, and it moved the crop: the single
colours switch stood over a line of prose and took 48 px, the five check
boxes fit on one line at this size and take 26, and the panel grew up
into the 22 px of difference — `y` 66 → 44, height 620 → 647, `x` and
width unchanged.

**Every atlas tile now carries the settings bar**, because it is at the
head of the window the atlas crops. With a clean `XDG_CONFIG_HOME` it
reads *Icons Classic*, *Size 24*, *Corners Theme shape* in all 129
tiles. That furniture is Settings' own and is the same in every tile;
what is the theme's is the three combo boxes it is drawn with, which is
three more views of a control the sample shows once.

**`uitoolkit-settings -plain-preview` renders the window without it**,
as the sample application alone — no icon set, no icon size and no
corner style can be chosen while it is set, so it is for pictures, not
for people. The crop is the same, so a tile can be rendered either way
by adding one flag to `CROP`'s command in `tools/atlas/render.sh`:

```sh
headless "$OUT/bin/settings" -plain-preview -stage "$id" -screenshot "$OUT/full/$id.png"
```

The atlas does **not** do this by default. The case for the bar is that
the three boxes show the pack's combo box — closed field, arrow or
stepper, the frame around the text — at the top of the tile where the
eye lands, and a pack whose signature is its combo (Aqua's blue stepper,
Win95's sunken field, Adwaita's flat pill) is better read for it. The
case against is that a tile is a picture: the bar's whole justification
is that a live control must be findable, which a PNG cannot honour, and
what it costs is 64 px of the document area and a band of identical
words in 129 tiles that the reader has to learn to skip. On top of that,
the bar is *designed* not to read as part of an application, which is
the one thing a tile is trying to be.

The applied look sets Settings' own metrics, and the column of choices
beside the preview is drawn in it, so take atlas shots with a clean
`XDG_CONFIG_HOME`. That is why the test builds Settings with
`PreferredLook`, the way the command does, rather than with a fixture
look. The row of options over the preview is drawn in the applied look
too, which is what sets the crop's `y`, and how many lines it folds onto
at 697 px is what would move it again; the icon size chosen in the
preview does not move it, because the preview's own bar grows inside the
crop rather than above it.

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
bar: `system` the desktop (**OS borders**),
`toolkit` uitoolkit for every window, left out for the default.
`captionButtons` `theme` puts the caption buttons of a frame uitoolkit
draws where the theme's era put them (the Mac's traffic lights on the
left; **Theme buttons**); left out, they follow
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

There is **no Delete on this page**, for a theme or for an icon set:
both buttons are gone, and a pack and a set are both folders that `rm -r`
removes. `style.DeleteUserTheme` is the API an application would call for
a pack, with `style.AfterUserThemeDeleted` for the other half of it: given
the appearance and the name that was deleted, it falls back to the
matching builtin palette (or the other one if that name was a shadow), so
an application can rewrite `look.json` rather than leave the selection
dangling. `style.DeleteUserIconSet` is the API for a set.

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

Settings offers them all in one chooser — on the preview's settings bar
— in the order `ListBuiltinIconSets` then `ListUserIconSets`:

- **Built-in** — drawn `classic` / `sharp`, plus the five premiere
  names when those folders are present under `icons/`
- **User** — any other `icons/<name>/` folder that contains at least
  one ToolIcon PNG

The chooser is the first box on the preview's settings bar, with the
size box beside it, and the sample's tool bar under them is drawn in
whatever they choose: the preview of a set is a real tool bar full of
it.

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
