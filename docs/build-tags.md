# What to ship: every build tag, and what it buys

Nothing in this toolkit is decided for you at run time that could be
decided at build time. An application that wants one look does not carry
thirty-four of them, and one that never shows Arabic does not carry a
shaper. The cost of that is this page: **three choices, made with a
build tag, and nobody else can make them for you.**

Everything below is measured on `cmd/uitoolkit-settings` — a real
application, not a benchmark — built as `go build` builds it on Linux
(cgo enabled) and stripped with `-ldflags "-s -w"`. There is a fourth
choice, `CGO_ENABLED=0`, in its own section at the end: it is worth
1.27 MB and costs you the GPU.

## The whole table

| build | what you gain | what you give up | size |
| --- | --- | --- | --- |
| one engine (`theme_engine_oxygen`) | the smallest product that looks like something | every other look | **12.98 MB** |
| **default** (no tags at all) | Plastik, KDE 3's default | every other look | **13.05 MB** |
| `+ font_fallback` | a rune the face lacks is borrowed from one that has it | two different CJK titles are the same row of boxes | 13.06 MB — **+0.01** |
| three engines | three looks to switch between | the other thirty-one | 13.45 MB — **+0.40** |
| `+ font_complex` | shaping, bidirectional text, Unicode line breaking | Arabic unjoined, Hebrew reversed, Japanese unwrappable | 15.21 MB — **+2.16** |
| `theme_engine_all` | all 34 engines, 132 packs | — | 21.32 MB — **+8.27** |
| `theme_engine_all,font_complex` | everything there is | — | 23.50 MB |
| *default, **not** stripped* | a usable stack trace | 5.69 MB | 18.74 MB |
| *default, `CGO_ENABLED=0`* | no shared libraries to install | the GPU path: painting is on the CPU | 11.78 MB — **−1.27** |

Read the second column downwards and the fourth across: that is the
whole decision. Two further rows, for scale:
`theme_engine_all` **unstripped** is 30.29 MB, and *one* Noto Sans CJK
family — a font, which this toolkit never embeds — is 18.6 MB on its
own.

The tags compose, and their cost does not change when you put them
together: `font_complex` is +2.16 MB on the default build and +2.18 on
`theme_engine_all`, +2.16 again with cgo off. Add them up rather than
measuring every pair.

## Three builds people actually ship

```sh
# A tool that looks like its desktop and speaks one language.
go build -ldflags "-s -w" ./cmd/yourapp                             # 13.05 MB

# The same, with text that works everywhere people type.
go build -tags font_complex -ldflags "-s -w" ./...                  # 15.21 MB

# The same again, with nothing to install beside it.
CGO_ENABLED=0 go build -tags font_complex -ldflags "-s -w" ./...    # 13.94 MB

# A theme browser, a screenshot rig, a demo of the toolkit itself.
go build -tags "theme_engine_all,font_complex" -ldflags "-s -w" ./... # 23.50 MB
```

If you are not sure, ship the first and add a tag when something is
wrong. Each of them is additive and none of them changes an API: the
same program compiles under all three.

## What each one is

### Theme engines — `theme_engine_<id>`, `theme_engine_all`

**Gain:** every look you name, and the packs that are drawn with it.
**Give up:** the looks you do not name; a pack whose engine is missing
falls back to one that is there, so nothing breaks, it just does not
look like the thing it is named after.

Naming any engine makes the default step aside, so `theme_engine_breeze`
is Breeze *instead of* Plastik and costs nothing extra. Eight megabytes
is what all of them weigh together — 39% of that binary — which is the
whole reason they are opt-in. Details, and the list of tags:
**[engines.md](engines.md)**.

### `font_fallback` — a rune the chosen face has not got

**Gain:** the rune is borrowed from the first family in
`style.FallbackFamilies()` that has it, baked to the borrowing face's
baseline.
**Give up without it:** every missing rune is the same box, so two
different CJK titles draw as two identical rows of boxes — which is how
a music player found this.

Ten kilobytes of code: the *faces* are the machine's, and that is why it
is almost free. It does not shape, order or line-break; for scripts that
need none of those — CJK, Cyrillic, Greek — it is the whole answer.
**[fonts.md](fonts.md)**.

### `font_complex` — the face's own tables

**Gain:** `GSUB` and `GPOS` (ligatures, joining forms, Indic
reordering, real kerning, mark attachment), UAX #9 bidirectional
ordering with a caret, hit test and selection to match, and UAX #14 line
breaking in labels and in the editor. Implies `font_fallback`.
**Give up without it:** Arabic and Syriac keep their isolated forms,
Hebrew comes out reversed, Indic is unreordered, and a language with no
spaces in it cannot wrap where it is allowed to.

**+2.16 MB, and no font data** — the faces are still the machine's. What
it does *not* do is mirror the interface: the text is right, the window
is still laid out left to right. **[complex-text.md](complex-text.md)**.

### Stripping — `-ldflags "-s -w"`

**Gain:** 5.69 MB, more than every theme engine but one combined, and
the single biggest thing most applications are not doing.
**Give up:** symbol names in a panic trace. Ship stripped and keep the
unstripped binary beside the tag you built it from.

### cgo — `CGO_ENABLED=0`

Not a tag, an environment variable, and the only one of these choices
that takes a feature away rather than adding one.

**Gain:** 1.27 MB, and a binary that links nothing — no `libX11`,
`libXext`, `libXrandr`, `libXi`, `libwayland-*` or `libxkbcommon` to be
present on the machine it runs on. Both window backends are pure Go
without it.
**Give up:** the GPU. paintengine2d's `GPUDevice` needs EGL and GLES2,
so `CGO_ENABLED=0` paints on the CPU always, whatever `UITK_PAINT` or
the saved renderer preference says ([platform.md](platform.md)).

macOS is the exception and has no choice: AppKit is Objective-C, so a
macOS build is `CGO_ENABLED=1` ([macos.md](macos.md)). Windows never
needs it — the Win32 backend is pure `syscall`.

### Icon packs — not a tag, a choice of files

Only a small subset of icons is embedded. The five premiere packs in
`icons/` are rendered assets your application opts into by shipping
them, so an application that uses none of them carries none of them.

## What is *not* a build tag

So that nobody goes looking for one:

- **A mirrored interface** for a right-to-left locale. Not in any build;
  `font_complex` is about text, not layout. **[contracts.md](contracts.md)**
- **Colour emoji** (COLR/CPAL, CBDT, sbix). Planned as `font_emoji`,
  not written.
- **Embedding your own face.** You install it with the application and
  name it; a registration call is a known gap. **[fonts.md](fonts.md)**
- **IME** and **printing** — platform work, not switches.
- `UITK_SYSTEM_FONTS=0` is an environment variable, not a tag: it turns
  the installed-font index off for a run, which is what makes a
  screenshot reproducible.

## Checking a claim on this page

Every number above comes from one script:

```sh
go build -tags "<the tags>" -ldflags "-s -w" -o /tmp/x ./cmd/uitoolkit-settings
ls -l /tmp/x
```

Measured on Linux/amd64 with Go 1.27 on 2026-10-13 (the toolkit itself
needs 1.22). The absolute
numbers move with the Go release and the platform; the deltas are what
the decision rests on.

and the behaviour each tag adds is under test in that configuration:
`tools/test.sh` runs the default build, `tools/test-fonts.sh` runs
`font_fallback` and `font_complex`, and `tools/engines-build.sh` builds
every engine on its own. A tag that quietly did nothing would fail the
tests written for it, and one that quietly turned itself on in the
default build would fail the tests written against it.
