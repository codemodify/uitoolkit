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
**Preview** — the three choosers that say what the previewed window is
drawn with, which stand with the options over it —
`about` opens **Files**, and `desktop` and `colours` open **Behaviour**,
because following the desktop's colours is one of the four options in
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
between the browser and the thing it is browsing for. Everything that is
not the list of packs stands with the preview: **one folding row of
settings over it** — four check boxes and then the three choosers for the
icon set, its size and the window's corners — and the three **config
paths** under it.

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

**It is a mock application and nothing in its chrome is live.** It
carried a bar of Settings' own for a release — the icon set, the icon
size and the corner style, over the sample's menu bar, "the preview
configures itself" — and those three are settings of the page, so they
are read where the page keeps its settings. The window is what they are
about, not where they are set. What is left in here is a sample: *Send*
sends nothing, the tree lists a mailbox nobody has, the check boxes tick
themselves.

**With one exception, and it is the point of an exception.** The sample's
**File ▸ Open…** and **Save…** — and the *Open* and *Save* on its tool
bar, which are the same commands — open a real file dialog: the
desktop's own, through the XDG portal, when **OS open/save dialogs** is
ticked, and the toolkit's themed one when it is not. That option is the
one of the four whose effect is a whole window, and four words cannot
show a window. It is a preview in the strict sense: the dialog lists a
directory and nothing else, and the path it comes back with is said on
the sample's status bar and dropped — *Open ~/notes.md — a preview:
nothing was read or written*.

It follows the **staged** setting, not the applied one, and that needs
saying because it is the easy thing to get wrong.
`widgets.ShowFileDialog` opens the desktop's dialog when its `Native`
option is set **or** `style.NativeDialogs()` is, and that second one is
process-wide and still holds whatever `look.json` said when Settings
started. A preview that went through the front door would show KDE's
dialog for an unticked box on a desktop whose dialogs are applied —
a preview of the setting the user is trying to leave. So Settings asks
for the dialog by name: `ShowFileDialog` with `Native` for the desktop's,
and `widgets.NewFileDialog(...).Show(from)` for the toolkit's, which is
the published way to say *this one, whatever the process thinks*. The
answer is read at the moment the menu is used, so ticking the box and
going straight to *File ▸ Open…* shows what was ticked, with no Apply and
no rebuild.

The dialog opens **from inside the preview's theme scope**, so the
toolkit's own comes up in the pack being staged: choose Windows 95 and
untick the box, and what opens is a Windows 95 file dialog. Its title
says what it is in both — *Open — preview only* — because a file chooser
that has opened over an editor of themes is the one place a person might
reasonably expect a file to be opened.

**The sample's own tool bar**: New, Open, Save │ Cut, Copy, **Paste** │
Pen │ Send, and the bar ends in free space (`widgets.ToolStretch`), so at
the narrowest window it sheds *Send* rather than showing half a button
cut off by the window frame. Paste came back when the choosers left the
end of this bar, and nothing has been added to fill the room the bar got
back when they left the window altogether: a tool bar is not a shelf, and
eight commands in three groups is what this sample does. It is drawn in
the staged set at the staged size — **that** is the preview of the icon
chooser, and it stays the preview of it now the chooser is outside the
window, because the whole previewed window is drawn in the staged
appearance through its theme scope. The strip of fifteen loose glyphs
that used to answer the same question has not come back and does not need
to.

**The window's own furniture is flush**: menu bar, tool bar, document,
status bar, with no air between them, the way a real window is. (There
were eight pixels between each for a long time. No window has those.)

**Nothing else is allowed in the preview's own box**, and only two
things share its pane: one folding row of settings and three lines of
paths. The widget gallery used to sit under it in a
second splitter (those are the widgets the tour shows over its Controls,
Views and Documents pages, and under the preview they halved it to
answer a question the preview had already answered). A strip of fifteen
stock icons sat above it for a while when the four pages were merged,
and it read well at 1024 — but the strip keeps its own height and the
preview takes what is left, so at the 720×520 minimum the glyphs folded
onto four lines and left the preview a caption and a menu bar. The
preview is far the biggest thing in its pane at every size (about **82%**
of it at 1024×860 and **62%** at the 720×520 minimum), and the only way
to promise that is to keep what is not the preview down to one folding
row and three lines. `TestSettingsPageHoldsAtEverySize` is that promise,
in four packs at two scales at both sizes.

Both numbers went **up** when the choosers came out of the window and
onto the page, which is not what a second line of settings sounds like it
should do: the line costs the pane 30 px, and the group box the three
paths gave up under the preview was 34. They were 82% and 61% before.

### Over and under the preview

**The settings**, in one row over the preview that folds: four check
boxes, each wearing a short word, and then the three choosers that say
what a pack is drawn with.

```
☑ Animations  ☐ OS open/save dialogs  ☐ OS borders  ☐ OS colors
Icons [Classic ▾]  Size [24 ▾]  Corners [Theme shape ▾]
```

That is what it looks like at 1024×860 — and it is **one wrapping row**,
not two rows; the second line is the fold finding the same arrangement
for itself. Why it is one row is under *The fold*, below.

| On the box | To a screen reader | What it is |
| --- | --- | --- |
| **Animations** | Animations | Hover fades, the default button's pulse, busy bars (GTK's `gtk-enable-animations`). While the desktop itself asks for reduced motion it says so, because the desktop's setting wins over the preference |
| **OS open/save dialogs** | OS open/save dialogs: the desktop's own Open and Save | KDE's and GNOME's own Open and Save, through the XDG portal, instead of the themed ones |
| **OS borders** | OS borders: the desktop's title bar and borders | Chromium's switch. On, every window gets the desktop's title bar and borders, and one that draws its own title bar (Mail's, with its tool bar in it) keeps it as its first row; off, the toolkit draws every frame in the theme's style **and the theme places the caption buttons**. The two states write look.json's `"decorations"` as `system` and `toolkit` — never `auto`, which is a third thing and is not what the box says ([decorations.md](decorations.md#the-settings-switch)) — and `"captionButtons"` as `theme` and the desktop's default with them |
| **OS colors** | OS colors: follow the desktop's light or dark mode and its accent | The pack shows its sibling to match the desktop — a chosen *Breeze* draws as *Breeze Dark* — recoloured around the desktop's accent where the engine takes one |

And the three choosers after them:

| On the page | To a screen reader | What it is |
| --- | --- | --- |
| **Icons** | Icons | The icon set the chrome is drawn in: `classic` / `sharp` and any premiere or user set installed under `~/.config/uitoolkit/icons/` |
| **Size** | Icon size | The pixel size its glyphs are drawn at — **16 / 24 / 32**, the way a word processor's size box lists numbers, because a set's glyphs are drawn at it and a page that showed a fixed size would be showing something the user is not going to get |
| **Corners** | Window corners | *Theme shape* (the pack's own), *Round* or *Square*, and the previewed window's frame is cut to it |

**Where the caption buttons go is not a box any more.** It was one,
*Theme buttons*, and it asked a question about a title bar that only
exists while *OS borders* is unticked: with the desktop drawing the frame
there is no toolkit caption to put buttons on, and with the toolkit
drawing it the theme is the only thing on this page with an opinion about
where they go — the Mac's traffic lights on the left, GNOME's lone close,
KDE's window menu. So the rule is implicit: **the toolkit draws the
frame, the theme places the buttons**, and *OS borders* writes both
halves of it. `style.CaptionButtonsPref` and
`app.Application.SetCaptionButtons` are unchanged — they are public API
and an application may still want to choose; Settings is what stopped
asking — and a `look.json` that already says `"captionButtons"` keeps
saying it until that box is touched.

**The three choosers came out of the previewed window.** They spent a
release on a bar of Settings' own over the sample's menu bar, where
"what shows a setting is what sets it"; before that the set and the size
were at the free-space end of the sample's own tool bar, where they were
read as the sample's own style and size boxes, and the corners were three
radio items in its **View ▸ Window corners**, where nobody found them.
The bar solved the second problem and left the first: a strip above a
menu bar is not something a reader expects to be theirs to use, and a
window that is a picture of an application is a strange place to keep the
page's settings. They are settings of the page, so they are read where
the page keeps its settings, and the window below is a mock application
with nothing live in its chrome.

**Four of them spent a release in the column**, as switches with a line
of prose each, where 300 logical pixels elided the labels themselves
(`Use the desktop's file dia…`); the fifth, the colours one, stood here
alone. They are one row now, over the window all five are about, and
they are **furniture you glance at rather than documentation**: the
sentence that was a line under each box is its tooltip and its
accessible description instead. Over the preview, every line of prose is
a line off the window the whole page is for.

**The short word is inside the spoken name**, never beside it — the same
rule the three choosers follow at the other end of the row, where *Size*
is read out as *Icon size*. What the eye gets from the row the box
stands in, the ear gets from the rest of the name. `Dialogs` on its own
would have been a riddle; `OS open/save dialogs` read as *OS open/save
dialogs: the desktop's own Open and Save* is one name for one control.

**Three of the four say `OS`** — *OS open/save dialogs*, *OS borders*,
*OS colors* — because those three are the ones that hand something back
to the desktop, and a reader should not have to work out that *System
frames* and *OS colors* were the same kind of thing. The fourth,
*Animations*, is the toolkit's own behaviour and says nothing about
whose.

**The options are check boxes, not switches**, for two reasons that
agree. The honest one: nothing on this page takes effect when it is
touched — Apply writes `look.json` and nothing else does — and a switch
is the control that says *this is live now*, while a check box is the
control that says *this is what I am asking for*. The measured one: a
switch's pill is 42 logical pixels of chrome before its word, and the
right-hand pane is 453 of them at the 720×520 minimum.

#### The fold

**The row folds, it does not shed.** It is a `widgets.Wrap` (Qt's flow
layout, GTK's `FlowBox`): it takes another line rather than cut a control
off at the window frame the way a tool bar's shedding would — a setting
nobody can reach is worse than another line. The words in front of the
three choosers are promised for the same reason, where on the bar inside
the preview they were not: that bar shed them from the right as it
narrowed, and at the 720×520 minimum it shed all three, leaving the
tooltips and the accessible names.

Measured in the 697-pixel row of a 1024×860 window and the 453-pixel row
of a 720×520 one, at scale 1 and 1.75 alike:

| | 1024×860 | 720×520 |
| --- | --- | --- |
| the four options | 543 px — one line | 543 px — two: the first three, then *OS colors* |
| the three choosers | 497 px — the second line, to themselves | the icons and the size beside *OS colors*, the corners on a third |
| **the row** | **2 lines**, 56 px | **3 lines**, 86 px |
| the preview | **82%** of the pane | **62%** |

**One wrapping row, not two fixed ones**, and that is a measurement
rather than a preference. Two rows each fold on their own account: the
options take two lines at 453 px and the choosers take two more, which is
**four** lines where one wrapping row takes three — 78 px off a preview
that has 281, and it stops being what the pane is for. A row that folds
is also what *gives* the two-row reading wherever there is room for it:
at 1024×860 the four options fill the first line and the three choosers
fall onto the second by themselves, which is the arrangement, arrived at
by folding rather than by decree.

**What the fold must never break** is a chooser from the word in front of
it, or the icon set from the size its glyphs are drawn at. So the three
choosers go into the row as **two** children, not six: the pair that says
what is drawn, and the one that says what shape the window is — the two
groups the divider on the old bar stood between. Inside a group the
spacing does the rest: six pixels between a word and its box, fourteen
between one pair and the next, and fourteen between the groups.

The order is the order they are read. The options come first because
three of them are about the desktop and the fourth about motion, and none
of them changes what the window below is *drawn* with; *OS colors* is
last of the four, nearest the window whose caption it changes, because it
is the one that decides which pack is drawn under it (a chosen *Breeze*
shows as *Breeze Dark*).

**The three paths** Settings reads and writes, one line each: the
**prefs** file, the **themes** directory and the **icons** directory,
written with `~` for the home directory and elided rather than wrapped
when the pane is narrow. They are under the preview because they are the
only part of the page that changes nothing — they say where what the rest
of the page changed ends up — and they are three lines rather than the
six they had (a line of prose and a three-row text box each) because
every line here is a line off the preview. They are labels now, not text
boxes: a box that can be selected from keeps three rows and grows a
scrollbar of its own as soon as a path is longer than the pane.

**And no box around them.** They were a group box with *Files* on its
legend, and the legend and the frame were 34 of the 101 px the block took
— 34 px off the window the page is about, to put a word over three lines
each of which is a name and a path and so says what it is by being one.
It is the same call the *Theme* legend lost in the column on the left: a
group box's legend names a *group*, and three paths under a preview are
not a group anyone has to be told about. Those pixels are what paid for
the line the choosers cost when they came out of the preview, with four
to spare.

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
of it in the column: corners, the icon set and the icon size are three
choosers in the row over the preview, and the strip of fifteen glyphs
under them was a picture of a tool bar standing in for the real one two
inches to its right.

The **Behaviour** panel is gone from the column, and with it the last
thing in it that was not the theme browser: its four switches are check
boxes in the row over the preview, with the colours one, under short
words. The column is the browser alone, which is what it was before the
four pages became one.

The **Theme buttons** check box is gone, and nothing replaced it: where
the caption buttons of a frame the toolkit draws go follows *OS borders*
now. See the options table above. `style.CaptionButtonsPref` and
`app.Application.SetCaptionButtons` stay.

The **settings bar inside the preview** is gone — the strip over the
sample's menu bar that carried `Icons`, `Size` and `Corners` for a
release. The three choosers are on the page, after the options, and the
previewed window is a mock application again. What the bar was built out
of stays as public API: `widgets.ToolWidget`, `widgets.ToolLabel` and
`ComboBox.MinWidth` have no caller in this repository any more (the
sample's own bar still ends in a `widgets.ToolStretch`), and they are
documented, tested in `widgets/toolbar_widget_test.go`, and the right
answer for an application that wants a control on a tool bar.

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
chooser in the row over the preview. So is the **free-space end of the
sample's tool bar** as a place to keep a setting — and *Paste*, which was
taken off that bar to pay for the two choosers that rode there, is back.
Nothing was put on that bar to fill the room they left: a tool bar is not
a shelf.

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
x 317, y 74, 697 × 651        crop box (317, 74) – (1014, 725)
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
preview and the paths under it, `x 317, y 66, 697 × 620` until the
behaviour options joined that switch in one row, and `x 317, y 44,
697 × 647` until the icon and corner choosers came out of the preview and
the group box came off the paths. Nothing that has happened to the column
on the left has moved them.)

**Changes to the column on the left do not move them.** The splitter's
ratio is worked out from the *window's* width, not from what the column
holds, and the column's contents have never had a minimum that could push
the sash: the browser's list takes what it is given at any width. Taking
the group box, the heading, the version and Delete theme… out of the
column, and running the list to the foot of it, was re-measured in the
eight packs `TestSettingsPreviewPanelKeepsItsPlace` walks and moved
nothing — the crop was `697x647+317+44` before and after.

**What the preview carries inside itself does not move them; what stands
over and under it does.** The settings bar was inside the panel, and the
panel takes whatever is left over it and under it, so the crop did not
move when that bar went in — it was re-measured then, in the eight packs
`TestSettingsPreviewPanelKeepsItsPlace` walks, and had not changed. The
last change moved it both ways at once. The three choosers came off that
bar and onto the page, which gave the row over the panel a second line
and pushed the top down 30 px (`y` 44 → 74); the three paths under it
gave up their group box, which was 34 px of legend and frame. Four of
those 34 are the difference: the panel is 651 tall where it was 647, and
`x` and the width are unchanged.

**No atlas tile carries a bar of Settings' own any more.** Every tile
did, for a release, because that bar was at the head of the window the
atlas crops, and it read *Icons Classic*, *Size 24*, *Corners Theme
shape* in all 129 of them — a band of identical words the reader had to
learn to skip, and 64 px of the document area, on a strip whose whole
justification was that a live control must be findable, which a PNG
cannot honour. A tile is now the sample application and nothing else,
which is the one thing a tile is trying to be.

**`uitoolkit-settings -plain-preview` is still there**, and it now means
the one thing left that is live in that window is not: the sample's File
menu opens no dialog. Nothing in a tile is clicked, so the atlas does not
pass it and the crop is the same either way; a script that wants a window
which cannot open anything adds one flag to `CROP`'s command in
`tools/atlas/render.sh`:

```sh
headless "$OUT/bin/settings" -plain-preview -stage "$id" -screenshot "$OUT/full/$id.png"
```

The applied look sets Settings' own metrics, and the column of choices
beside the preview is drawn in it, so take atlas shots with a clean
`XDG_CONFIG_HOME`. That is why the test builds Settings with
`PreferredLook`, the way the command does, rather than with a fixture
look. The row of settings over the preview is drawn in the applied look
too, which is what sets the crop's `y`, and how many lines it folds onto
at 697 px is what would move it again; the icon size chosen in that row
does not move it, because what grows with it is the previewed window's
own tool bar, inside the crop.

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
left); left out, they follow the desktop's button layout. Settings writes
it with `decorations`, because the question only arises while the toolkit
is drawing the frame: unticking **OS borders** writes both `toolkit` and
`theme`, ticking it writes `system` and leaves `captionButtons` out. It
is read on load whatever wrote it, so a file that names a layout Settings
would not now offer keeps it until that box is touched, and
`app.Application.SetCaptionButtons` still sets it from an application.
See [decorations.md](decorations.md).

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

Settings offers them all in one chooser — in the row of settings over the
preview — in the order `ListBuiltinIconSets` then `ListUserIconSets`:

- **Built-in** — drawn `classic` / `sharp`, plus the five premiere
  names when those folders are present under `icons/`
- **User** — any other `icons/<name>/` folder that contains at least
  one ToolIcon PNG

The chooser is the first of the three that follow the options, with the
size box beside it, and the previewed window's tool bar is drawn in
whatever they choose: the preview of a set is a real tool bar full of it.

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
