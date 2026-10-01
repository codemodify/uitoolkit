# Building an application

The path through this toolkit, in the order you meet it. Every step names
the one call you need, the thing that can quietly decide otherwise, and
where the detail lives.

Read it once, in order. The other twenty-four pages are reference and are
linked from here; none of them is a starting point.

## Which kind of application are you building

Three, and they are not a ladder — nobody graduates from one to the next.
Pick the one that fits and the other two cost you nothing.

| | the window's chrome | the controls in it | you write |
| --- | --- | --- | --- |
| **1. Themed** | the look's | the look's | nothing — this is the default |
| **2. Your own chrome** | **yours**: its shape and two or three surfaces | the look's | a call per piece you want |
| **3. Skinned** | **the art's** | **the art's**, hit areas and all | a skin file, no Go |

Read the row that fits and skip the rest of this section.

**1. A themed application.** You place controls; the look draws them and
arranges its own chrome around them. Its buttons, its frame, its caption,
its era. This is the default in the strongest sense: write none of the
calls in this page's later steps and you get it, under all 132 packs, and
your application looks like the desktop it is running on. Most
applications are this one and should stay it.

**2. An application that states part of its own chrome.** A browser's tabs
*are* its title bar; an editor's menu row *is* its caption; both tint their
own surfaces. The look still draws every control inside — the tabs, the
buttons, the field are the pack's — but the shape and the two or three
surfaces are the application's. Each piece of this is a separate opt-in
call, so you take only what your shape needs:
[the recipe](recipes.md#an-application-whose-chrome-is-its-own--a-browser-an-editor).

**3. A skin.** The art *is* the application: sprite sheets, named
sub-rects, and a table saying which rect paints which part of which
control — the answer WinAmp and VLC's skins2 gave. A skin is a pack of the
`skin` engine, selected like any other theme; the hit areas can follow the
ink rather than the box, and a window need not be a rectangle.
[skins.md](skins.md).

**Nothing in 2 or 3 happens by default.** `SetCaptionStyle` starts at
`CaptionFollowsLook`, `Options.Chrome` is empty, `IconButton.Flat` is
false, a tool item does not grow. A test in `app` holds that promise,
because it is the kind that erodes one convenient default at a time.

---

> **The one habit worth having.** Several layers of this toolkit may
> override what you state — the look's era, build tags, the person's icon
> directory, the compositor — and every one of them says so. Leave the log
> on while you build. A line that starts `uitk ` is the toolkit telling you
> it did something other than what you asked, and *what to call instead*.
> [diagnostics.md](diagnostics.md).

---

## 1. An application and a window

```go
a := uitoolkit.New(uitoolkit.Options{})
win, err := a.NewWindow(uitoolkit.WindowOptions{
    Title: "Notes", Width: 900, Height: 600,
    MinWidth: 420, MinHeight: 320,
})
if err != nil {
    log.Fatal(err)
}
win.SetContent(content())
a.Run()
```

`Options` is the application: its look, its scale, its backend.
`WindowOptions` is one window: size, **`MinWidth` / `MinHeight`** (which
exist — reach for them before writing your own floor), and `Decorations`.

`a.Run()` does not return until the last window closes. Everything after
this point happens on that one thread; `a.Post(fn)` is how another
goroutine gets work onto it.

**On Linux, say who you are.** `Options.AppID` is what a desktop files
your windows under — `xdg_toplevel.set_app_id` on Wayland, `WM_CLASS` on
X11 — and it decides which task-bar button your windows share, which icon
they wear and which `.desktop` file they match. The default is your
executable's name, which is enough to keep you separate from other
programs; set it to your `.desktop` file's name, without the suffix, if
you ship one. Windows and macOS identify a program by its executable
already and ignore the field.

*Overridden by:* nothing here. This step is yours.

---

## 2. The frame, and who draws it

Two separate questions, and conflating them is the commonest early
mistake.

**Who draws the window frame** is `WindowOptions.Decorations`:
`DecorationsServer` (the desktop draws it), `DecorationsClient` (the
toolkit does), or `DecorationsAuto`, which gives you the toolkit's frame
when you have a title bar of your own and the desktop allows it.

**What your title bar is** is `SetTitleBar`:

```go
win.SetTitleBar(widgets.NewHeaderBar(lead, centre, trail))
```

*Overridden by:* **the compositor**, which may refuse a client frame —
you get `uitk decorations:` if it does.

---

## 3. Whether your title bar *is* the caption

This is the step that catches people, so it gets its own.

A look's era decides how your title bar meets the frame. A **stacked**
frame (Windows 95 to XP, classic Mac, Motif) keeps *its own* title strip
and puts your bar in the row beneath it — correct, because that is where
those desktops put a tool bar. A **merged** frame (GTK header bars,
Windows 11, macOS) makes your bar the caption, buttons at its sides.

If your application is a *shape* rather than a citizen of an era — if
your tabs are your title bar on every desktop, the way Chromium's and VS
Code's are — say so:

```go
win.SetTitleBar(tabStrip())
win.SetCaptionStyle(style.CaptionMerged)
```

Otherwise leave it: `CaptionFollowsLook` is right for an application that
wants to belong to the desktop it is running on.

*Overridden by:* **the look's era**, unless you state a caption style.
You get `uitk caption:` at the first layout if it happened.
[decorations.md](decorations.md).

---

## 4. Content

Containers, not coordinates. Nothing in this toolkit wants a pixel
position from you.

| you want | use |
| --- | --- |
| a row or a column | `NewRow(...)`, `NewColumn(...)` — `.WithGap`, `.WithPad`, `.AddFlex(c, 1)` |
| two panes with a sash | `NewSplitter(axis, a, b)` — `MinA` / `MinB` are real floors |
| things on top of each other | `NewStack(...)` |
| more than it fits | `NewScrollView(child)` |
| rows and columns that line up | `NewGrid(...)`, `NewForm(...)` |

A widget tells its parent what it needs through `Measure`, and the parent
decides with `Arrange`. If a layout is wrong, the question is almost
always which of the two is lying. A container that must not be squeezed
past its content implements `widget.MinWidther`.

*Overridden by:* nothing — but see [recipes.md](recipes.md) for the
shapes that look like they should work and do not.

---

## 5. The look

The toolkit follows the person's own setting. An application that needs
part of it its own way states that part and leaves the rest alone:

```go
a := uitoolkit.New(uitoolkit.Options{
    Theme: uitoolkit.ThemeOverride{Pack: "adwaita"},
})
```

Their icon set, corner policy and typefaces still come from their
settings, and still follow a change to any of them without a restart.

**Theme engines are chosen at build time.** A pack whose engine is not in
your binary is not there at all, and you get the default look:

```sh
go build -tags theme_engine_adwaita ./...
go build -tags theme_engine_all ./...      # every one, +8.4 MB
```

Ask `style.ThemePackAvailable(name)` before you rely on one.
`style.Classic.Pack()` will **not** tell you — it records the name it was
given, not whether anything can draw it.

**Theming your own chrome.** A browser or an editor paints its own title
bar and tool bar rather than wearing the pack's, and `Options.Chrome`
states those surfaces by name — the same names a pack's `theme.json`
"extra" map uses, so the engine was already reading them:

```go
uitoolkit.New(uitoolkit.Options{
    Theme:  uitoolkit.ThemeOverride{Pack: "linear"},
    Chrome: map[string]paintengine2d.Color{
        "titleBar": accent,                                      // the tab strip
        "toolBar":  style.Mix(accent, paintengine2d.RGB(1,1,1), .55), // the row under it
    },
})
```

That is all a browser needs: the engine fills the *selected tab* with the
tool bar's colour, so the tab and the row below it meet and read as one
surface, which is the thing that makes a browser look like a browser. A
tint carries no ink, though — labels stay the palette's colour — so keep a
tinted surface inside the contrast the palette was built for.

*Overridden by:* **your build tags**. You get `uitk theme:` if the pack
you pinned is not in the build. [engines.md](engines.md),
[themes.md](themes.md).

---

## 6. Marks and words

Two rules, and the second is the one that is easy to get wrong.

**A mark is an icon, never a character.** The bundled face is Latin only
and there is no font fallback, so `✓ → ★` typed into a label draw as
boxes — on your machine as much as anyone's under `UITK_SYSTEM_FONTS=0`.
Use `Button.Icon`, `MenuItem.Icon`, `TableView.CellIcon`,
`TreeNode.Icon`, **`Label.Icon`** for a mark nothing clicks (a status
bar, a caption), or `style.DrawToolIcon` when painting yourself.

**Text is text, never an icon.** An icon-only control needs
`SetAccessibleName`; a screen reader cannot read a picture, and neither
can a translator.

Art ships in `icons/` and is **not** installed for you. The person's
`~/.config/uitoolkit/icons` is theirs — an application must never write
there. Your own art goes wherever you like, and you register it:

```go
style.AddSearchPath(myDataDir)
```

*Overridden by:* **the person's icon directory**. A set of theirs that
predates an id gets the toolkit's own drawn mark and a `uitk icons:`
line. [contracts.md](contracts.md), [icons/README.md](../icons/README.md).

---

## 7. Reachable

An icon-only button needs a name; a custom control needs a role. Both are
one call, and neither is optional for anything shipping.

```go
btn.SetAccessibleName("Archive")
```

[accessibility.md](accessibility.md), [keyboard.md](keyboard.md).

---

## 8. Tested

Run your tests through an environment with no display and a private bus,
the way this repository does — `tools/testenv.sh`. A test that reaches the
real desktop is a test that moves the developer's windows around.

Then assert that the toolkit is doing what you told it:

```go
for _, f := range a.Diagnostics() {
    if f.Level == uitoolkit.Warn {
        t.Errorf("%v", f)
    }
}
```

This is the one that keeps paying. Everything in this page that *can* be
silently overridden is a thing this assertion catches the day it starts
happening. [testing.md](testing.md), [diagnostics.md](diagnostics.md).

---

## Where everything else is

| | |
| --- | --- |
| every widget, and its Qt / GTK / WinForms equivalent | [widgets.md](widgets.md) |
| doing a tricky thing right | [recipes.md](recipes.md) |
| an application whose chrome is its own (a browser, an editor) | [recipes.md](recipes.md#an-application-whose-chrome-is-its-own--a-browser-an-editor) |
| window frames, captions, shapes | [decorations.md](decorations.md), [shapes.md](shapes.md) |
| theme packs, engines, skins | [themes.md](themes.md), [engines.md](engines.md), [skins.md](skins.md) |
| the status item | [tray.md](tray.md) |
| Windows and macOS | [windows.md](windows.md), [macos.md](macos.md) |
| what the text stack does not do | [contracts.md](contracts.md) |
| paint cost | [perf.md](perf.md) |
| this toolkit next to the others | [toolkits.md](toolkits.md) |
