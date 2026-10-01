# Release notes

What changed in each release, and why. The reasons are here because a
list of names is not much use six months later: what a reader usually
wants to know is whether a change affects them, and that is a question
about the problem it solved.

---

## 0.23.0

Five samples that are meant to look like applications people use, and
what building them found — which turned out to be most of this release.

Each sample was put beside the real application and compared pixel by
pixel: Chromium, Firefox, SourceGit, Thunderbird, VS Code. Every
difference that was not the sample's own arrangement was a gap in the
toolkit, and each one is API here rather than code in an example. The
pattern repeated often enough to be worth stating: when an application
cannot say what it means, it paints around the toolkit instead, and the
paint is always slightly wrong.

New in this release, with the rest of the detail below: `style.TabShape`
and `BrowserTab.Icon`; `IconButton.Flat` and `.Badge`, with
`NewFlatIconButton` and `NewFlatMenuButton`; `widgets.FieldBox` and
`ToolItem.Grow`; `Options.Chrome` and `style.WithChrome`;
`Window.SetCaptionStyle`; `Options.AppID`; `Label.Icon`; the `diag`
package and `Application.Diagnostics`; and `style.ThemePackAvailable`.

### The tabs a browser draws, and the marks on them

The two browsers people copy chose different tab shapes and the toolkit
drew only one of them. Chrome's is merged — filled with the tool bar's
colour, concave feet running into the row below, so tab and tool bar read
as one surface. Firefox's, since Proton, is a floating card standing clear
of that row. `style.TabShape` has both; `TabsMerged` stays the default.

Writing its test found what the sample could not show: on a pack whose
title bar and tool bar are the same colour — most of them, until an
application tints one — a floating card is *invisible*, because unlike a
merged tab it has no shape to tell it from the strip. It carries a
hairline now.

`BrowserTab` had `Title`, `Tip`, `NoClose` and `Data` and no way to show a
mark, so a git client's repository tabs had bare words and the browsers
had no favicons. `BrowserTab.Icon` sits at the leading end and the label
centres in what is left. Tabs also fill the strip they are given rather
than a fixed height pinned to its bottom, and the corner radius is the
pack's to state: Chromium's tabs are round and SourceGit's nearly square,
both drawn by the same engine, so a constant could not serve both.

### A window that asked for no border gets none

`SetBorderless` zeroed the *content* insets and the frame went on painting
a border underneath. You saw it wherever nothing opaque covered it — the
tab strip, the page — and not where the tool bar did, which read as the
tool bar overstepping the window. Nothing was overstepping; the border
should not have been there.

The decision now lives in the one helper every engine calls rather than at
each of its nineteen call sites, and the compiler made each site pass the
state in. A test holds every pack to it — and writing it was worth more
than the fix, going from seventeen packs failing to none through
distinctions rather than workarounds: a skin's frame is its art; Motif and
Win31 draw sculpted bevels, not a border on a frame; the Amiga and BeOS
paint a structural frame they never declared, which is two names written
down with the reason rather than a rule that would swallow the next one.

### A mark that is a button only when you point at it

`IconButton.Flat` and `.Badge`, `NewFlatIconButton`, `NewFlatMenuButton`.
Both were helpers the samples had written out by hand — one of them
verbatim in two of them — which is how a framework gap announces itself. A
browser's lock and star, an editor's menu bar and activity rail, a mail
client's bell: all want the tool face, and the toolkit had the flat face
and the framed one and no way to ask the second for the first.

That turned up a bug the moment a *named* menu used it: `IconButton` is a
mark alone, but `MenuButton` embeds it, and File and Edit are the same
control with a word instead. The flat path drew an empty label, so an
editor's menu bar came out as four blank hit areas.

### A label was drawn in a font nobody had measured

A Thunderbird-shaped tool bar read **"Get Message", "Reply Al",
"Forwarc", "Delet"**. The labels were not too long for the bar — each
button was given exactly the width it had asked for.

`ToolButton.Measure` reserves `style.ControlFontOf(look, RoleTool)`.
The engine then draws the label in whatever face it likes and clips to
the button's box. Adwaita draws libadwaita's semibold and **declared no
control font at all**, so it was measured in the body face and drawn a
weight heavier — about a character's worth, silently cut off the end.

Two fixes, both at the seam rather than at the call site:

- `adwaitaEngine.ControlFont` now names the face it actually draws
  buttons, tool buttons and tabs in.
- A skin asked its *base pack* for the face and then painted the label
  through the *stock* chrome — `under` where the drawing uses `underFor`.
  Two different engines, one measuring for the other. Lantern, Marquee
  and Nocturne are partial skins whose base is not in the default build,
  and their tool labels were clipped in exactly that build and no other.

The regression test is not a table of which engine is bold. It renders:
a tool button is drawn twice at one width, once with the label and once
without, and the difference between the two images is the label's own
ink. Do that at the measured width and again with room to spare, and
compare how wide that ink is. A label that fits draws the same glyphs
either way; one that is clipped or elided does not. Every pack in the
build is checked, in both build configurations.

### `Label.Icon`

The toolkit's first rule is that a mark is an icon and never a character,
because the bundled faces have no arrows, no check and no chevron — a
status bar that writes `↑2 ↓0` draws two boxes. But until now the only
widget that could put a mark on screen was a **button**, so obeying the
rule in a status bar meant either a button that does nothing or a rune
that draws a box. The SourceGit sample hit it immediately and drew two
boxes.

`Label.Icon` (and `NewIconLabel(icon, text)`) is a mark before the text,
in the text's own colour and sized from its font, so it matches small
print instead of towering over it. A label with an icon and no text is
just the mark.

### Three ways to build, and two of them are opt-in

Written down at the top of [building.md](docs/building.md), because it is
the first question and it was nowhere: a **themed** application places
controls and the look draws and arranges everything around them; an
application that **states part of its own chrome** keeps the look's
controls but owns its title bar's shape and two or three surfaces; a
**skin** is art and hit areas in a file. They are not a ladder and nobody
graduates between them.

The second and third are opted into a call at a time, and that is now a
test rather than an intention: a window built with no extra line reports
the look's caption style, keeps the look's border, has no title bar of its
own, and its surfaces match the pack's exactly — and an icon button is not
flat, a tool item does not grow, a label carries no mark, a field is not
frameless. Defaults of this kind erode one convenience at a time, so the
promise is held somewhere that fails.

### The pieces a browser-shaped application needs, as toolkit pieces

The look-like-chromium sample was built beside a real Chromium and
measured against it until the two matched. Nothing it needed stayed in the
sample: every piece is API, and the shape is now five ordinary calls that
any application can make ([recipes.md](docs/recipes.md)).

- **`widgets.FieldBox`** — one text-field well with controls inside it: a
  browser's address bar with its site-information mark and its bookmark
  star, a search box with a magnifier. `TextField.Clearable` does the
  trailing end only, and does it by taking the room out of the *string*,
  which cannot work at the leading end — where the text begins belongs to
  the engine, and twenty-five of them draw a field. So the box draws the
  well and the field inside it is frameless, and the well takes the focus
  ring from the field: one control to look at, not three.
- **`ToolItem.Grow` / `ToolGrow`** — the other half of `Stretch`. Both
  take a bar's spare width; `Stretch` leaves it blank and `Grow` hands it
  to a control. A widget marked `Stretch` was treated as a gap and never
  laid out, so the sample's omnibox was an invisible 950-pixel hole in its
  own tool bar.
- **`IconButton.Flat`** — the same control on the tool face: no frame
  until the pointer is over it, which is how a browser draws its lock, its
  star and its three-dot menu. The toolkit had the flat face and the
  framed one and no way to ask the second for the first, so a sample
  copying that chrome put framed buttons inside its own address bar.
  `MenuButton` inherits it.
- **`CenterButtons` now means what it says.** It fired only when the band
  was taller than the look's caption, leaving the other half of the same
  question unanswered: a button *shorter* than its band hung from the top
  with the slack beneath it however much the pack wanted it centred. Every
  pack that wanted centring had worked it out by hand into
  `ButtonPad.Top`; a pack that set the flag alone got nothing. A test now
  holds every pack that claims centring to it.
- **The web look's caption buttons and browser tabs.** It was the one
  modern pack still using the Windows idiom — a wide full-height rectangle
  — among adwaita, breeze, material and macOS, which all use a square
  button centred with a gap. Its browser tab's feet and corner radius were
  flat pixel counts, so on a short caption the whole flare compressed into
  the last pixel or two and the tab read as a rectangle with the corners
  knocked off; both are proportions of the tab now.

The root package re-exports all of it, along with `ToolStretch`,
`ToolLabel`, `ToolWidget`, `NewIconButton`, `NewMenuButton` and
`NewIconLabel`, which were reachable only by importing `widgets` directly.

### An application can paint its own chrome

The look-like-chromium sample had the right layout and still did not look
like Chromium, and measuring the two side by side said why in one line:
real Chromium's chrome is a tinted strip over a paler tool bar — `#D2E2FC`
over `#ECF2FA` — and ours was one unbroken field of white. Chromium's
whole tab metaphor rests on that contrast: the selected tab is a channel
cut through the tinted strip, continuous with the tool bar below it.

Two things were missing, and only one of them was a gap.

**The sample never painted a tool bar.** Its navigation row was a plain
`Row` on the window's background, so there was no tool bar surface for the
selected tab to merge into — and the engine had been filling that tab with
the tool bar's colour all along, correctly, into nothing. A `ToolBar` with
`ToolWidget` for the address field is the whole fix, and the tab meets it.

**An application could not tint its chrome.** `Options.Chrome` and
`style.WithChrome` state the window's surfaces by name — "titleBar",
"toolBar", "sidebar" — which is exactly how a pack states them in
`theme.json`, so every engine already reads them through `Classic.X`. The
sample now matches a real Chromium's tab strip exactly and its tool bar
within four parts in 255.

A tint carries no ink: an engine draws the text on these surfaces in the
palette's colours, and there is no "title bar text" to go with "titleBar".
So a tinted surface must stay inside the contrast its palette was built
for — which is why the Firefox sample pales the `#6094CF` it measured off
a real one rather than using it directly, and why the doc says so.

### Two programs built with this toolkit are two programs

Both Linux backends hardcoded the application's identity to the string
"uitoolkit" — Wayland in `xdg_toplevel.set_app_id`, X11 in `WM_CLASS`.
That is what a desktop files windows under, so it decides which task-bar
button they share, which icon they wear, which `.desktop` file they match
and which window rules apply. The desktop did exactly as it was told: a
mail client and a password vault, two unrelated programs, shared one
task-bar entry and one icon, and nothing either could do would separate
them. Five samples on one desktop appeared as one program.

The default is now the executable's own name, which is distinct without
anyone asking, and `Options.AppID` states it explicitly — which an
application shipping a `.desktop` file must do, since matching that file
is the only way the desktop can give it its icon.

Not a seam the other backends owe anything: Windows and macOS identify a
program by its executable already, so there was never a bug there to fix.
A Linux-shaped problem with a Linux-shaped answer.

One trap the samples found: the default being the binary's name means a
sample built as `chromium` would claim the *real* Chromium's identity —
its task-bar slot, its icon, its window rules. Each sample now states its
own, which is also the demonstration of why the field exists.

### One path through the documentation, and a test that it is true

Twenty-four reference pages and eleven thousand lines of them, and no
order to read them in: the toolkit's problem was never a shortage of
documentation. [docs/building.md](docs/building.md) is the spine — an
application from `New` to its first test, in the order you meet it, where
each step names the one call you need **and the layer that can quietly
decide otherwise**, now that each of those layers says so out loud. The
README points at it and the reference pages hang off it.

And the pages are now checked against the toolkit. `contracts.md` once
listed the places an application may put a mark — and a status bar was
not among them and could not be, because no widget could draw one unless
you clicked it. The rule the page stated was unkeepable and the page said
nothing, because nothing compared the two. Every `package.Symbol` written
in a documentation page now has to resolve, or the suite fails.

Two things it found immediately: a metric written as `style.ComboH` that
is really a field of `ChromeMetrics`, and — the interesting one — that
the comparison tables name *fyne's* `widget` and `layout` packages, which
are spelled exactly like this repository's. There is no telling them
apart, so those two are not checked inside a table and are everywhere
else. One page deliberately names a function in order to say it does not
exist; that is in a short written-down allowlist with its reason, not
filtered away by a pattern.

### The toolkit says when it did not do what it was told

Everything else in this release was found the same way: something was
stated clearly, a layer that had never been mentioned overruled it, and
nothing anywhere said so. A sample asked for a browser's shape in four
correct calls and got a caption row above its tabs. A pinned theme pack
whose engine was not in the build fell back to the default look in
silence. A tool button measured one font and was drawn in another.

These are not bugs in those layers — a Windows 95 pack *should* put a
tool bar under its title strip, and engines *should* be chosen at build
time. The bug was that being overruled was indistinguishable from being
wrong.

`diag` is a new leaf package, and `Application.Diagnostics()` returns what
it collected. Every finding has three parts and the third is the point:
what was asked, what happened instead, and **the call or build flag that
gets what was asked**.

	uitk theme: asked theme pack "adwaita", got the default look; no engine
	  in this build draws that pack — build with the engine that draws it;
	  docs/engines.md lists the tags

They are logged as they happen, for a developer, and collected, for a
test — which is the half that stops an application drifting back into the
confusion unnoticed. The caption note waits for the first layout on
purpose, so an application that settles its shape on the next line is
never told it was overruled.

Wired so far: a pinned pack this build cannot draw, a title bar an era put
in a second row, a frame the compositor refused, a theme metric clamped,
an icon set that is behind or absent, a tray menu the host would not let
the toolkit draw. Debug tracing is deliberately *not* in it — that is a
different thing. [docs/diagnostics.md](docs/diagnostics.md) has the rest,
including how to add one.

`style.ThemePackAvailable` is the question an application actually wants
to ask before pinning a pack, and is new with this.

### An application can say its title bar *is* the caption

The look-like-chromium sample asked for everything correctly — client
decorations, the tab strip as the title bar, no title text, no border —
and still came out with a caption row above its tabs. The toolkit was
doing what it was told: `DecorationSpec.Stacked` says a classic frame
keeps the era's own title strip and puts the application's bar in the row
*under* it, which is right for Windows 95's tool bars and wrong for a
browser. Nothing could say otherwise, so under a classic pack a browser
sample could never look like a browser.

`Window.SetCaptionStyle` is the application's say over the era:

```go
win.SetTitleBar(tabStrip())
win.SetCaptionStyle(style.CaptionMerged)
```

`CaptionFollowsLook` stays the default and stays right for an application
that wants to belong to the desktop it runs on. `CaptionMerged` is for one
that is a shape instead — Chromium's tabs are its title bar on every
desktop it has ever run on, and so are VS Code's menu row and SourceGit's
repository tabs. `CaptionStacked` asks for the classic strip under every
pack. The four samples whose namesakes merge now state it, and they look
like themselves under the era packs as well as the modern ones.

### A copied icon set that is one id behind is behind, not broken

Two samples drew a blank where their menu button should be. The set was
not installed wrong: `icons/README.md` says an application must never
write into `~/.config/uitoolkit/icons`, because two that do overwrite
each other silently. The consequence had not been followed through — a
copy is therefore made **once**, so the day the toolkit gains a typed id
every installed copy of every set is one stem short of it, for everybody.
This release alone added about thirty ids.

`loadFileIcon` treated that as an install to complain about and drew the
missing-icon mark, while holding a perfectly good vector for the id. Now
the two cases are told apart by `style.Drawable`: a **typed** id the set
lacks is drawn by the toolkit, with one line naming the stem and where a
fresh copy is; a **stem-only** icon, which nothing anywhere can draw,
still gets the mark, because there the box is the only honest answer.

It is the mirror of the same seam in the other direction — a stem-only
icon showing the missing-icon mark in a *drawn* set — and both are now
pinned by tests.

### The samples wear their namesake's pack, not just its shape

Getting the shape right only got half the way there: a browser laid out
correctly but painted in a classic pack's bevels still does not look like
a browser. Each sample now states the pack it wants and nothing else, so
the reader's icon set, corners and typefaces still come from their own
settings:

	uitoolkit.New(uitoolkit.Options{Theme: uitoolkit.ThemeOverride{Pack: "adwaita"}})

VS Code and SourceGit take the packs drawn after them (`vscode-night`,
`sourcegit`, both `webEngine`); the three that wear the desktop's own
chrome take `adwaita`. Theme engines are chosen at build time, so each
sample's run line now carries the tag its pack needs and each says so at
startup if it was built without it, rather than falling back silently to
the default look and appearing to be broken.

### The five samples say which sample they are

Each window's title is now its sample name, so five of them on one desktop
can be told apart in the task bar. They draw their own captions, so the
realistic title text is still there in the bar itself.

### The choices column in Settings was 198 pixels wide

`defaultChoicesRatio` works a *share* out from the window size, guessing
what the page spends on chrome before the panes ever see it. The guess
was 30 pixels and the truth is about 90, so at 760 the column asked for
216 and was given 198 — narrow enough to elide a pack's own name. A share
cannot express a floor, so the column now states one (`Splitter.MinA`)
and the ratio only decides how much more than the floor it gets.

---

## 0.22.5

The two applications kept going. secretvault's three remaining items were
all real and all the toolkit's; comms-mail made its header into real
buttons with icons and found five more.

### Three that were still open, and all three were

- **A Caps Lock hint that missed Caps Lock.** A field learned only of
  *changes*, and a window open with the lock on since before the dialog
  was shown has no change to report — so the mark appeared on the second
  passphrase, never the first, which is the one that gets refused. That
  is the entire case the feature exists for. Watchers hear the first
  event now, and a `SecretField` asks its window when it takes the focus.
  `Window.OnLockKeys` keeps its documented change-only meaning.
- **A tray that said it was shown when there was no tray.** `Shown` read
  any property error as "displayed", which was documented as deliberate:
  a bus error is not a statement. But no watcher *on the bus* is a
  statement — nothing is registering status items, so nothing can be
  showing one — and a GNOME without an AppIndicator extension is exactly
  that, which is the case the whole question exists for. A watcher that
  is there and will not answer is still read as shown.
- **A secret label that gave away the length**, on the screen and to a
  screen reader. `SecretLabel.MaskLen` is a fixed number of bullets
  measured for itself alone, its accessibility node says "concealed" and
  nothing else, and `Lines` splits at newlines so a PEM block can be
  shown at all.

### Three shapes of button with a mark on it

The toolkit had two of the three, and [docs/recipes.md](docs/recipes.md)
confidently named the wrong one.

- **`IconButton`** — the look's push-button face with the mark centred,
  square, named for the tooltip and the screen reader in one go.
  `ToolIconBtn`, which the recipe recommended, is a *tool item*: flat
  with no frame until hovered in most eras, so it does not read as a
  button. That was my advice and it did not do what the entry claimed.
- **`MenuButton`** — opens on press so a drag runs into the menu, stays
  down while open, closes on a second press, and **owns its items'
  accelerators**. An application menu moved from a `MenuBar` to a plain
  button silently loses its shortcuts; the first anyone knows is that
  Ctrl+Q stopped quitting. The matcher is shared with `MenuBar` now
  rather than written twice, which is how the two came to differ.
- **A latched push button** — `Button.Checked` and `Toggle`. Measured
  rather than assumed, as the default button was in 0.22.0: 53 of the 135
  packs draw a checked button exactly as an ordinary one, so on those it
  is drawn pressed instead — which is how Windows 3.1, Motif, CDE and
  OPEN LOOK drew a toggle anyway.

**`IconMore` and `IconMenu`**, with vectors in the drawn sets and `menu`
rendered into all five packs. Both were stem-only, which is the hazard
`IconByStem` documents — no vector in a drawn set, and every pack uses a
drawn set unless the user picks otherwise — left standing on the two
commonest button marks there are.

### Chrome that lines up with a pane

The caption is laid out **before** the content, so anything that measures
a divider and then tells the chrome is permanently one frame behind — on
a drag, a frame the user watches. The request was a header bar bound to a
pane's width; binding is the wrong shape, because it moves the
measurement and not the ordering.

`Splitter.OnRatioChanged` fires on the **drag** and `HeaderBar.StartWidth`
takes a plain number, so the next layout has it in time. Nothing is tied
to anything: a footer, a status bar or a second tool bar reads the same
value, and neither widget knows the other exists. `Splitter.SetRatio`
reports too, so restoring a saved layout is not a silent change.

A `Splitter` takes its panes' own minimums as well, now that
`widget.MinWidthOf` can be asked — a sidebar cannot be dragged until its
buttons run off the edge. `MinA` / `MinB` override it, and
`AllowCollapse` lets a pane close (and really close: the old 8 % floor on
`Ratio` applied even to a pane meant to collapse).

### Buttons of your own in the title bar

- **`Window.SetCaptionButtonVisible`** turns one of the window's own
  buttons off for that window. It cannot *add* one the desktop cannot do:
  the look and the compositor decide what is possible, and this picks
  from that.
- **`Window.SetCaptionActions`** puts the application's buttons up there,
  as many as it likes, each choosing its side.

They are drawn on the era's **tool** face rather than its window-control
face, and that is a measured constraint rather than a preference. A look
draws a window control's shape and its glyph as one piece, keyed on which
control it is, so a close button's shape cannot be borrowed for a
different mark: 67 of the packs draw a glyph of their own for any value
they do not recognise. The tool face is the era's own button for a mark,
it is separable, and every pack has one.

### Fixed

- **A plain label held layouts open.** `MinWidthOf` (0.22.4) read a
  one-line label as a thing that cannot shrink, because it is no taller
  when narrowed — but a single-line label *clips*, and a layout was
  keeping a whole sentence's width for something happy to be cut short.
  `Label` answers for itself now; a wrapping one still reports its
  longest word.

### Still open

**`Button.Icon` is about 64 px wider than a leading-icon button needs**
(comms-mail #29), and that is inherent to the engine drawing the label:
every era centres and decorates it its own way, so the widget reserves a
strip at each end rather than taking the drawing over. `IconButton` is
the answer where a row is tight; a real leading-icon layout would need
either an opt-in that draws the label itself or a change to all 34
engines.

**A sidebar running beside the title bar** (#37) is not here. What landed
is what removes the frame of lag from an application doing it by hand.
The full feature needs client-side decorations by definition, and the
pattern nearly everything actually uses — a full-width title bar with the
sidebar starting under it — needs none of it.

**Font fallback for message content** stays declined; see
[contracts.md](docs/contracts.md).

---

## 0.22.4

An application can ship its own icons, themes and skins without writing
into a directory that is not its own — and a table row no longer has the
next row drawn over it.

### A block after one whose height changed was placed six pixels high

Block tops are a prefix sum: `tops[k+1] = tops[k] + heights[k]`. So
changing `heights[i]` invalidates `tops[i+1]` onward, and the last entry
still good is `tops[i]`. `setHeight` kept `tops[i+1]` — one too many — so
the block *directly after* one whose height changed was placed at the
offset the old height gave.

A block's height is estimated before it is laid out and corrected when it
is, so this fired whenever an estimate was wrong: a table row that turned
out two lines tall after being estimated at one had the next row drawn
over its second line, with no rule between them. It looked width-
dependent because at a width where nothing wrapped no estimate was wrong,
and a resize cleared it because that recomputes every top from zero —
which is why nothing an application could call fixed it. The splice path
beside it has always used the right form.

This was reported against 0.22.3 as a table bug and is not one: it moves
every kind of block whose height is not what it was guessed to be.

### An asynchronous check in a prompt

`MessageBoxInput.ValidateAsync` is for a check that cannot answer at once
— the mail server that has to be asked whether a folder name is taken.
The dialog stays up with the accepting button busy, ignoring further
presses, until `done` is called: `done(nil)` closes it with its accepting
result so `OnResult` runs, and `done(err)` puts the reason under the
field and re-enables the button with what the user typed still there.

`Validate` stays as it was, for a check the caller can make at once.

`MessageBox.Close(result)` finishes a dialog from outside a button — an
answer that arrived, a vault that locked. Dismissing the overlay by hand
takes the dialog off the screen without recording a result or running
`OnResult`, so whatever was waiting on it waited forever; that was the
only way to do it, and it is why the previous advice for an asynchronous
check did not actually work. `MessageBox.Checking` reports whether a
check is out.

### Horizontal scrolling

`ScrollView.Horizontal` and `TableView.Horizontal`, both off by default.

A scroll view scrolled up and down only, so a child with no narrower form
was clipped at the edge with the rest unreachable — and a table with more
columns than fit squeezed them to their floors and clipped what was left,
so the last columns could not be reached at all. Every file manager and
mail client on every desktop scrolls sideways there.

Off by default is deliberate. A scroll view's usual job is to give its
child the width it has and let it fold, and **a form should not scroll
sideways** — it should drop its labels above its fields or fold with
`Wrap`. Turning this on says "this content genuinely cannot be
narrower": the child is then measured at its natural width rather than
the view's.

Both give a bar, the wheel (Shift for a wheel with only one axis) and the
arrow keys. A `ScrollView` that scrolls sideways stops imposing its
child's width on anything above it, so its `MinWidth` drops to nothing.

Widgets that own their own text already scrolled sideways and are
unchanged: `TextArea` with wrap off has a real bar, and `TextField`,
`SecretField` and `SecretArea` slide their contents under the caret. This
was only ever missing from containers.

### `widget.MinWidthOf` — where the floor is

A layout that runs out of room stops at a floor rather than squeezing
children to nothing, and then runs over the edge. That is right, and Qt
and GTK do the same — but a program had no way to find out *where* the
floor was, so the only way to meet it was to hit it.

`widget.MinWidthOf(c)` is the width below which something starts leaving
the box. Size a window from it instead of from a guess, and assert it in
a test — which is what catches a button growing an icon before a user
does.

Containers answer it themselves (`widget.MinWidther`), because they have
to: a container asked to fit in one pixel reports the one pixel it was
constrained to. `Grid` sums its column floors, a row sums its children, a
column and a `Wrap` take their widest, `Pad` adds its insets, and a
`ScrollView` adds its bar — it scrolls up and down, so it makes nothing
narrower. Anything else is worked out from the **height**: what folds
gets taller when it is narrowed, what cannot keeps its height. Width
cannot answer it, because a button asked to fit in a pixel says 65.

Hand the answer to `WindowOptions.MinWidth` / `MinHeight`, which the
window system is told about on every backend — so a window cannot be
dragged below what its own content needs.

### A lever for a form in a narrow pane

`SecretField.PreferredWidth` and `SecretArea.PreferredWidth`, matching
`TokenField.PreferredWidth`. A field cannot fold, so a flexible grid
column gives it what it asks for even where there is less, and the
columns after it go off the edge — deliberately, since the alternative is
a button squeezed to nothing. Lowering what the field asks for is the
lever for a form that has to fit a narrow pane;
[docs/recipes.md](docs/recipes.md) has the rest of the answer.

### docs/recipes.md

A page for the situations where the obvious way is wrong, written for
someone who has not used this toolkit before: a mark on a button and what
`Button.Icon` costs, a row that must fold, shipping your own icons
without writing into the user's directory, a passphrase, sizing something
that wraps, a status colour as text, a check that has to ask a server,
closing a dialog from your own code, dialogs raised from inside dialogs,
and window-wide state like Caps Lock.

Each entry says what the obvious route is, why it does not work, and what
does. Every one of them is there because somebody building a real program
took the obvious route first.

### Whose directory is whose

`~/.config/uitoolkit/{icons,themes,skins}` belongs to the **person using
the machine** — it is this toolkit's `~/.icons` and `~/.themes`, where
they install a set once and every uitoolkit program picks it up. That
part was right.

What was missing was the other half. An application that wanted art of
its own had no way to offer it, so the only route was to copy into the
user's directory — and two applications doing that overwrite each other.
Last one wins, and neither can tell: a program built against a newer
toolkit installs stems an older program's copy then removes, and the
older program's icons start coming back as the missing-icon placeholder
with nothing anywhere to say why. It also meant the icons added in
0.22.2, and the reply-all fixed in 0.22.3, reached nobody who had not run
a copy command by hand.

So an application keeps its art wherever it likes and registers it:

```go
style.AddSearchPath("/opt/comms-mail/share")   // …/share/icons/<set>/*.png
```

A registered directory has the same shape as the user's (`icons/`,
`themes/`, `skins/`; any may be missing) and is private to that process.

**The user's copy comes first, file by file.** Where they have installed
a set, their version of a given icon wins; where they have nothing — a
stem their copy predates, or a set they never installed — the
application's answers. A person keeps control of how their desktop looks;
a program can rely on art it ships. Settings will not delete a theme that
came from an application: it is not Settings' to remove.

`SearchPaths`, `IconSearchDirs`, `ThemeSearchDirs`, `SkinSearchDirs` and
`IconSetDirs` report what is being looked at.

**Renamed, so the owner is in the name.** `IconsDir`, `ThemesDir`,
`SkinsDir`, `IconSetDir` and `SkinDir` are now `UserIconsDir`,
`UserThemesDir`, `UserSkinsDir`, `UserIconSetDir` and `UserSkinDir`.
They still mean the same directory — the person's — and are still where
Settings installs and removes. The old names read like "the place icons
go", which is exactly the misreading that had applications copying into
them; the new ones cannot. This breaks the old spelling on purpose.

Nothing else changes for a program that registers nothing.

---

## 0.22.3

What 0.22.2 got wrong. Both applications upgraded, checked the closed
items, and found ten things — nine of them the toolkit's own, and one of
them a feature that had never worked at all.

### The status inks were black and white

`ReadableInk` walked lightness from 0 to 1 where `flatFromHSL` takes
percent, so every candidate sat within one percent of black. On a light
background the first already cleared 4.5:1, and the ink came out all but
black; on a dark one none did, and the fallback made it white.
`plastik-night`'s danger, warning and success were all `#ffffff`, and
`plastik`'s warning ink was `#050300`. Every promise
[docs/widgets.md](docs/widgets.md) makes about keeping the pack's own
colour was void.

`#c08000` is `#8f6000` now on a light ground and `#c78500` on a dark one;
`plastik-night`'s danger is `#ff5858`.

**Both tests passed**, which is the part worth writing down. One checked
the contrast ratio — and black on a light background reads perfectly
well. The other checked hue — which survives at one percent lightness.
Neither checked the property the feature exists for. Two new ones do: a
saturated status colour comes back saturated, and nothing comes back pure
black or white where the pack asked for a colour.

### Rich-text tables

Laying cells out as paragraphs in 0.22.2 brought three problems, all of
them visible in an ordinary Markdown table.

- **A header cell was an H1.** `layoutTableRow` asked for the heading
  face with the block's level, and the HTML parser gives a `<th>` level
  1 — so header cells were title-sized and broke mid-word in a narrow
  column. A header is bold at the table's own size.
- **A wrapped row's chrome stopped after its first line.** The column
  rules and the rule under a row were drawn at the first line's height,
  so a row whose cell wrapped had its later lines outside its own cell.
  That reads as the next row overlapping this one, which is how it was
  reported.
- **Columns were squeezed past their longest word.** Shrinking every
  column toward one number was right while a cell too wide for its
  column was elided; now that cells wrap it broke words instead —
  "Price" came out "Pric", "£4.50" became "£4.5" and "0", and the rows
  grew a line each to hold the pieces. Each column has its own floor
  now, and is never *rounded* below it either, which is the chip bug of
  0.22.2 over again.

### Fixed

- **An unclosed quote holds the token split.** Reading it as no quote was
  wrong for the case that matters: while someone *types* `"Doe, Jane"
  <jane@x>`, the quote is open at the moment the comma is typed — so the
  comma split there, `"Doe` was refused, and the chip came out with the
  name's comma missing. A stray quote now holds splitting until it is
  closed or deleted, which the user can see in the editor. Comments and
  angle brackets nest too, since an address list puts commas inside both.
- **`DismissOverlay` from outside every overlay closes the top one
  again**, as it did before overlays stacked. Popping only the overlay
  the caller sits in silently broke every caller that was not inside a
  dialog: a program quitting with a recovery key on screen stopped
  wiping it, and a cancelled file chooser stayed up. An overlay that
  *was* on the stack and is no longer still closes nothing — that is the
  case reached on the way out of every dismissal, and popping the top
  there would take the dialog underneath down as well. The two are told
  apart by `widget.ContentRootHost`.
- **`SecretArea` takes the width it is offered.** Asking for 280
  whatever it was given made a grid's flexible column hand it 280 where
  there was less, and since it cannot fold, nothing squeezed it back and
  the columns after it went off the edge.
- **`uitoolkit.Version` said 0.22.1 through the whole 0.22.2 release**,
  so every application that shows the toolkit's version showed the wrong
  one. A test now compares it with the newest heading in this file.
- **heroicons' `reply-all` was the share mark** — three linked dots, a
  different action entirely. heroicons has no reply-all and
  `icons/render.sh` takes official upstream names only, so it is
  `arrow-turn-up-left`: a reply arrow, and distinct from `reply`'s own
  `arrow-uturn-left`.

### Still open

Three items from the same round are not in this release, because each is
a design change rather than a fix:

- **`Button.Icon` costs about 64 px a button.** Reserving a strip at both
  ends keeps every engine's label treatment untouched, which was the
  point, but rows that fitted narrow windows stopped fitting. A leading
  icon needs either `DrawButton` taking the label's box — a change to all
  33 engines — or an opt-in.
- **The premiere icon packs are not embedded.** They are read from
  `~/.config/uitoolkit/icons/`, refreshed by hand, so on a machine that
  has not copied them the icons added in 0.22.2 draw as the no-icon
  placeholder and heroicons' forward is still the fast-forward 0.22.2
  says was fixed. Embedding them, with the installed copy as an
  override, is the fix.
- **`MessageBoxInput.Validate` is synchronous.** A check that is a server
  round trip has no way to say "keep the dialog up" but to return an
  empty error, and no way to close the box with its result.
  `ValidateAsync` with a busy accept button is what it wants.

And font fallback for message content stays declined; see
[contracts.md](contracts.md).

---

## 0.22.2

Twenty-five items from the same two applications — a mail client and a
password vault — and this time nearly half of them were the toolkit's own
breakage: a fix to something 0.22.x shipped, or a hole left in it. One of
the twenty-five turned out not to be a gap at all.

### If you are upgrading

Four changes alter behaviour rather than adding to it.

- **`Pad`, `Spacer` and `ProgressBar` lengths follow the display scale.**
  0.22.0 made `FlexBox`'s gap and padding and `Grid`'s row and column
  gaps 1x design lengths and left these three in device pixels, so a form
  at 2x had scaled gaps between unscaled pads and a progress bar half as
  long as everything beside it. **If your application multiplied them by
  the scale itself, stop.** `RawSpacing` on `Pad` and `Spacer` is the way
  back, as it already was on `FlexBox` and `Grid`. At scale 1 nothing
  moved.
- **A flexible grid column may now be narrower than its content.** A
  wrapping label in a `Form` gets the width the form has and folds,
  instead of taking the width of its whole text on one line and running
  off the edge. Nothing that cannot fold is squeezed.
- **Overlays stack.** `ShowOverlay` pushes and `DismissOverlay` pops, so
  a dialog shown from inside another no longer closes it. If you relied
  on the second dialog replacing the first, call `Window.SetOverlay`,
  which still means "there is one dialog".
- **A square-cornered pack draws square chips.** 41 of the 135 packs were
  getting a capsule because a corner radius of zero read as "no opinion".

### Layout

- **A column that can wrap is allowed to.** A flexible track was frozen
  at its children's unbounded width when there was not room for it: right
  for a button, wrong for a label. Which one a child is cannot be told by
  width — narrowed, both come back narrow — so it is told by height: what
  folds gets taller, what cannot keeps its height.
- **`Grid` measures its rows at the widths it will use**, and `Overlay`
  settles its card's width before asking for its height. Both measured at
  one width and laid out at another, which left a form taller than what
  it drew and a dialog's buttons outside the dialog, where clicks on them
  missed. Every other container is swept by a test.

  The same rule applies to an application and cannot be enforced from
  inside the toolkit: a column measured at 600 and handed a 300-wide box
  of the height that measurement asked for still puts its last child
  outside the box. A container cannot grow a box it was given.
- **`Wrap` and `TokenField` flow at whole pixels**, because that is what
  `Arrange` has. A measure at 300.4 and a layout at 300 disagreed about
  which child fits on the line, and it showed as a line's height of empty
  space under the last row.

### Icons

- **Seventeen more**, and a way to name the rest. Fifteen typed ids for
  actions the five packs already shipped with no id to reach them by —
  trash, archive, junk, tag, folder, reply-all, settings, external-link,
  eye, user, bell, send, close, quit — plus print, which no pack carried
  and is now rendered from the same pinned upstreams as the rest.
  `IconByStem` reaches the whole shipped vocabulary; `IconStarFilled` and
  `IconDot` are drawn by the toolkit in every set, because neither is
  shipped by an outline pack and a filled star has no house style to
  match.
- **No icon draws nothing.** The drawn sets had no default arm at all, so
  anything they had no vector for simply did not appear — a button with
  an invisible mark on it. This mattered more than it looked: every
  shipped pack uses the drawn Classic set unless the user picks
  otherwise.
- **`Button.Icon`.** Only `ToolButton` and `MenuItem` had one, so a
  dialog's buttons could not carry a mark. The engine centres and draws
  the label, so the button reserves a strip at each end rather than
  pushing the text along, and every era's label treatment survives
  untouched.
- **Art**: heroicons' forward was the media fast-forward; the drawn
  paperclip read as a rounded box at the size a message list uses; the
  new cog read as a sun and the new bell as a lampshade.

### Colour

- **`Palette.Ink`, `DangerInk`, `WarningInk`, `SuccessInk` and
  `ReadableInk`.** The declared status colours are era-faithful on
  purpose — Clearlooks really did use `#c4a000` — and most of their uses
  are fills, where the chrome around them carries the contrast. Text is
  the other case, and 225 of the 405 status pairs across the shipped
  packs did not reach 4.5:1, the worst near 1.6:1. Both are kept now: the
  palette holds what the pack declared, and `Ink` is the same colour
  moved along its own lightness until it reads, hue intact. All 405 pairs
  pass as ink; the 180 that already read are returned untouched.
- **`Label.Tone`** says what a label's text *means* — danger, warning,
  success, muted, accent — and takes the ink form at paint time, so it
  follows a theme change. An application that set `Color` from the
  palette got the unreadable version and had to set it again on every
  look change.

### Secrets

- **`SecretArea`**, for a PEM block or an OpenSSH private key:
  `SecretField` over several lines, and the same widget rather than a
  second one, so the buffer, the wipes and the refusals are not written
  twice. A pasted key keeps its lines and CRLF is normalised in place.
- **`SecretField.SetBytes`** takes a generated passphrase over rather
  than copying it, and wipes what it replaces.
- **`SecretClip.OnCleared`**, for the "copied — clears in 45s"
  indicator. The alternative was polling `Cleared`, which is a timer of
  its own and a window in which the interface is wrong.
- **Caps Lock in the field itself.** `widget.LockKeysWatcher` gives every
  widget in the window the lock state, where `Window.OnLockKeys` was one
  callback that a widget had to take and hand back. A focused
  `SecretField` draws the mark with no application code.
- **`SecretField` declares itself a `widget.SecretTarget`.** Not being an
  `IMETarget` already kept the input method away; declaring it lets
  anything that asks get a straight answer rather than inferring one.

### Dialogs and windows

- **Overlays stack**, so a confirmation raised from inside a dialog comes
  back to that dialog with its fields as they were — and a dialog that
  wipes its secret fields on close no longer wipes them because something
  else opened.
- **`MessageBoxInput.Validate` and `AcceptLabel`.** A prompt can refuse a
  value without closing, showing why under the field with what was typed
  still there; and the accepting button can name its action, which is
  what every desktop's guidelines say. `SetInputError` is the same for a
  check that is a round trip; `PromptFor` is the short way to both.
- **A fitted dialog is centred for the height it ends up at.** A resize
  keeps the top-left corner, and the caller cannot sequence the two
  itself because the fit is what changes the size.
- **`StatusItem.Shown` and `SetOnShownChange`.** `Alive` says there is
  somewhere to send the item; `Shown` says someone is showing it. They
  differ on exactly the desktop where getting it wrong costs the user
  their window: a GNOME with no AppIndicator extension has a session bus
  and no tray.

### Rich text

- **A table cell keeps its bold, its code and its links, and wraps.** A
  cell was drawn as its plain text in one face fitted to one line, so a
  newsletter's table of links lost its links. Cells are laid out the way
  paragraphs are now, through the same two halves of the code, because a
  second text layout is how a widget ends up with two sets of rules.
- **The block after a table gets its space.** Rows of one table stay
  tight — they are one grid — but a quote's rule used to butt straight
  onto the bottom row.
- **A protocol-relative image src is remote.** `//host/pixel.gif` has no
  scheme and was read as a file beside the document, so a mail client
  told an image was local fetched a tracking pixel without asking.

### Fixed

- **A chip no longer cuts its own text short.** `Token.Measure` asked for
  a fractional width and `SetBounds` rounds each edge on its own, so the
  box could come back a pixel under and `Paint` elided the label it had
  just measured for.
- **A quoted comma does not split a token.** `"Lovelace, Ada" <ada@x>`
  was two recipients, both nonsense. An unclosed quote is not an error:
  the user is still typing.
- **A chip field measured without a width asks for a field's width**, not
  for every chip on one line. That measurement was what pushed a Write
  window's form past the window's edge.
- **`UITK_FONT` no longer stops a program following the desktop's dark
  mode.** The override was folded into the file read from disk, so a
  machine with no `look.json` resolved an empty file instead of the
  defaults — and an empty file says `FollowDesktop` is off.
- **A widget taken out of the tree lets go of its window.** `Add` hands
  the host down a subtree and `Remove` cleared only the parent, so
  anything holding a removed component kept the window reachable.
- **`ClickComponent` and `FocusOn` scroll to the target first**, as a
  person does. A button below a `ScrollView`'s fold was clicked where its
  box said it was, which is outside the view.

### Not done

**Font fallback for message content** was asked for again and is still
declined. One face draws everything; a rune it lacks is a box. A product
that must show other people's arbitrary scripts needs a toolkit with a
shaping engine, which [contracts.md](contracts.md) says plainly and at
length.

**`widgets.Wrap` was reported missing and is not.** It has laid controls
out in a row that folds since before 0.22, and is documented. That one
was a discoverability failure rather than a gap, and the report has been
withdrawn.

### Verified

The suite with every engine and with one; Windows and Darwin
cross-compiled. The 64 failures in a one-engine build are unchanged and
all of the same kind — a test naming a pack that build does not have.

---

## 0.22.1

The contracts 0.22.0 should have carried, and a crash found while
writing them down.

### The text contract

[docs/contracts.md](docs/contracts.md) is new: one page for what the
toolkit promises about text and what it does not. The rules existed —
scattered across a widgets subsection, a comparison page's gap list and
a README table row — and one of them was not written down anywhere.

Two rules, stated as rules:

- **A mark is an icon, not a character.** `✓ ✗ →` typed into a label are
  boxes on any machine whose selected face lacks them, and on every
  machine under `UITK_SYSTEM_FONTS=0`.
- **Text is text, not an icon.** The converse, and the one nothing draws
  a box to warn you about: an icon-only control needs
  `SetAccessibleName`, because a glyph cannot be read aloud, translated
  or searched; and prose must not be rendered as a picture to dodge the
  script limits, because it cannot be selected, copied, found or scaled.

What is not supported, in one place: **no `GSUB`** (no Arabic joining,
no Indic reordering, no ligatures of any kind), **no bidi** (right-to-
left text comes out reversed), **no right-to-left interface** — nothing
reads a locale's direction, nothing mirrors, and there is no API to ask
— and **no font fallback**, which is the one that surprises people:
installed fonts decide which single face is *selected*, not which glyph
answers for a rune.

The two bundled faces are now stated exactly, because they differ and
the difference is a trap. Titillium Web has Latin and nothing else;
JetBrains Mono carries `✓ ✗ → ← └`, Greek and Cyrillic. Both are pinned
by a test, so the table cannot drift from the fonts — and the page says
plainly not to set `Mono` on a label to win a glyph, which is rule 2
broken from the other side.

### Fixed

- **A font that cannot draw Latin no longer kills the process.**
  `BakeFamily` panicked on a script-only face — Noto Sans Arabic, Noto
  Color Emoji and about thirty others the font chooser offers on an
  ordinary desktop. The choice is saved in `look.json`, so picking one
  crashed every uitoolkit application on the machine at start, and kept
  crashing until the file was edited by hand. It falls back to the
  bundled face now, which is what an uninstalled family has always done;
  the two cases had no business differing. A test walks every installed
  family and asserts each gives back a face that can draw the interface
  — 265 on the machine this was written on, of which 30 used to be
  fatal.

---

## 0.22.0

Everything in this release came from two applications built on the
toolkit — a mail client and a password vault — writing down what they
could not do and why. Each item below closed one of those, and the two
lists are now down to a single open entry between them.

### If you are upgrading

Five changes alter behaviour rather than adding to it. None needs a code
change, and three of them will silently improve an application that did
nothing.

- **Gaps and padding follow the display scale.** `FlexBox`'s `Gap` and
  padding and `Grid`'s (and so `Form`'s) `ColGap` and `RowGap` were
  device pixels; they are 1x design lengths now, scaled by the look like
  every other length in the toolkit. **If your application multiplied
  them by the scale itself, stop** — doing both makes them square. Set
  `RawSpacing` on either to keep device pixels. At scale 1 nothing moved
  at all.
- **A program with no `look.json` follows the desktop's dark mode.**
  `DefaultAppearance` has `FollowDesktop` on. A saved `look.json` is
  still a choice and is not overridden.
- **A `TextField` with nothing to delete lets Backspace bubble**, as it
  already did for Return. A parent that handles Backspace will now see
  it.
- **`Modifiers` carries the lock keys.** Compare shortcuts with
  `Modifiers.Chord()`, which masks them out, if you compare the whole
  set rather than asking `Ctrl()` and friends.
- **`richtext` has three new block kinds.** Code that switches on
  `Block.Kind` should expect `Quote`, `Rule` and `TableRow`; the last
  keeps its text in `Cells` as well as flattened into `Spans`.

### Handling secrets

A Go string cannot be wiped: every copy lives until the collector
reaches it and then lingers in freed memory. A widget that holds a
passphrase must not be built out of one.

- **`SecretField`** keeps its contents in a `[]byte` edited in place.
  The array is zeroed when the buffer is outgrown and when a delete
  shortens it; `Bytes()` hands out a copy the caller owns and wipes
  (`WipeBytes`); `Len()` is a rune count and no content, for a strength
  meter. Copy, cut, dragging the selection out, the X11 PRIMARY
  selection and middle-click paste are all refused. A test reads the
  source to keep any `string(` conversion out.
- **`SecretLabel`** is the read-only half — an item view's "show
  password", a TOTP code. Its value comes from a function, so a locked
  vault empties it; `Hide()` zeroes what it last drew from.
- **The input method is off** while a secret widget has the focus
  (`widget.SecretTarget`), which now covers `TextField` with `Password`
  too. An input method is another process that sees every keystroke and
  learns from it, so masking the preedit was never enough. X11's
  `SetIMEEnabled` was a no-op; it now unsets the input-context focus and
  resets it.
- **`ClipboardSetSecret`** writes CLIPBOARD only — never PRIMARY —
  hinted so clipboard managers do not record it, and clears itself after
  a timeout (45 s by default) and on quit. `ClipboardGetSecret` reads
  back as bytes.
- **`Window.SetSecureInput`** and **`SetExcludeFromCapture`**, each
  reporting what actually happened and each with an `Available` so an
  application can decide what to promise. They are not the same promise
  on every platform: macOS takes the keyboard from every other process,
  X11 stops the server delivering keys to other clients, Wayland
  inhibits the compositor's own shortcuts, and Windows has nothing an
  ordinary application may use.
- **`Modifiers.ModCapsLock`**, `Window.LockKeys()` and `OnLockKeys`. A
  right passphrase refused is nearly always Caps Lock, and a field that
  holds a secret cannot work that out for itself — the only other way is
  to read the secret.

### Windows and dialogs

- **`WindowOptions.Role`, `Center` and `KeepAbove`**, applied before the
  window is shown, plus `Window.Activate`, `Center` and `SetWindowRole`.
  A prompt opened from a terminal now appears in front, focused and
  centred. `_NET_WM_WINDOW_TYPE_DIALOG` on X11, `xdg_dialog_v1` on
  Wayland, an owned dialog-framed window on Win32, a floating `NSPanel`
  on macOS.
- **`WindowOptions.FitContent`** and **`Window.FitToContent()`** make a
  window as tall as its content, measured at the window's own scale and
  through its own frame — the two things a caller measuring by hand
  could not know at the same time.

### Widgets

- **`Prompt`** — the "name this" dialog, as `MessageBoxOptions.Input`:
  the field takes the focus, Return is the default button, the initial
  value comes up selected, and `Required` greys OK out while it is
  blank.
- **`TokenField`** — chips with a cross and an editor at the end, the
  recipient field of every mail client written since Gmail. A comma, a
  semicolon or Return ends a value; leaving the field commits what is in
  the editor; a refused value stays where the user can see it.
- **`FileOpenFolder`** and **`FileDialogOptions.Name`**. A Save `Path`
  that names a file is also split into the folder to list and the name
  to suggest, which used to produce an empty listing and no error.
- **Per-tab disable and hide** on `TabView` and `TabBar`. A hidden tab
  takes no width; a disabled one is stepped over by the arrow keys; Home
  and End mean the first and last tab there *is*.
- **`HeightForRows`** on `ListView` and `TableView`, for a popover sized
  by how many rows it should show.
- **`ScrollView.ShrinkToContent`** and `MaxHeight` — a panel as tall as
  its contents up to a limit.
- **`widget.SetPopupKeysPass`** — a non-capturing popup, so a completion
  list can be open while the user goes on typing into the field under
  it.
- **Icons in views**: `TableColumn.Icon`, `TableView.CellIcon` with a
  colour, `TreeNode.Icon`, and seven stock ids — `IconAttach`,
  `IconStar`, `IconFlag`, `IconReply`, `IconForward`, `IconCheck`,
  `IconMute`. Every one resolves to a PNG the five shipped icon sets
  already carry. A message list's marks used to have to be characters,
  which meant they came out of the font.

### Rich text

- **Tables, quotes and rules.** `<table>` was read for the characters
  inside it and nothing else, `<blockquote>` as plain paragraphs, and
  `<hr>` was dropped. There are now `Quote` blocks (one accent rule per
  level of nesting, so a thread's depth reads at a glance), `Rule`, and
  `TableRow` — a table being a run of rows whose column widths are
  measured once for the run.
- **`ResolveImageKind`** tells the callback whether a src is inline,
  remote, local or a data URI. A mail client cannot treat the first two
  the same: an inline part came with the message, and a remote one tells
  the sender it was opened. `Doc.Images`, `UnresolvedImages` and
  `ImageCounts` are the enumeration a "load remote images" control is
  built from.
- An unresolved image shows its **alt text** instead of an empty box.

### Fixes

- **Text measured at its own width no longer wraps.** `Font.Wrap` added
  rune advances, which does not see pair kerning and so reports a line
  wider than it paints — safe for deciding a line is full and wrong for
  deciding it is too full. `/usr/bin/mail` is 78.05 wide and came back
  as two lines with the word broken when measured at 78.05, so every
  wrapping label in a `Form` was a line taller than the line it drew.
- **A wrapping label that flexes in a row is no longer clipped.**
  `layout.Flex` measures a child again at its final width — Qt's
  `heightForWidth`, GTK's height-for-width — and takes the taller
  answer. The pass only ever grows a child, so no existing geometry
  moved.
- **A theme this build cannot paint no longer turns the window dark.**
  The palette family was guessed from the pack's *name*, and the parser
  behind that guess knows the two words "light" and "dark" and reads
  everything else as dark — so every pack a build was missing came up
  dark. `Appearance.Missing()` and `style.MissingThemeNote()` now report
  it, and Settings prints that line.
- **Every pack marks its default button.** Which engines ignore
  `Primary` is measured rather than listed: 129 of 132 mark it
  themselves, and the two Amiga packs get a heavier border.
- **A real tray menu on macOS.** `SetMenu` stored the rows and showed
  nothing.
- **A drop carrying files** goes to the widget that takes files, not to
  a text field nested inside it that would take the paths as text.
- **`TextField.SetText`** leaves the caret at the end.

### Other

- **`plastik-night`**, a dark sibling for the default pack — 132 packs
  now. Without one, a program nobody has configured opened white on a
  dark desktop.
- **`Window.Type`, `Press`, `ClickComponent` and `FocusOn`** for driving
  a headless window from an application's own tests.
- `keyboard-shortcuts-inhibit-unstable-v1` and `xdg-dialog-v1` are
  vendored alongside the other generated Wayland protocols.

### Verified

The suite with every engine and with one; every engine built on its own;
Windows and Darwin cross-compiled; and `platform`, `app`, `widgets`,
`richtext` and `style` run on macOS 15.

---

## 0.21.0 and earlier

Before this file existed the release notes were the tag and its commit
message. `git show v0.21.0` and its neighbours have them.
