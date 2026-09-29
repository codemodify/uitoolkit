# uitoolkit next to the others

Where this toolkit sits among Qt, GTK, Avalonia, Fyne, WinForms, Win32,
GPUI and Apple's own — what it does differently, and the things it does
not do that they do.

This page is for someone deciding whether to use it. It is therefore as
specific about the gaps as about the rest: a comparison that only lists
strengths is an advertisement, and you cannot plan around an
advertisement. [compare.md](compare.md) is a different page — the
widget-behaviour map, which says how a check box or a menu is expected
to act and which test fails when it stops.

## The short version

uitoolkit is a desktop toolkit written in Go that **draws every pixel
itself**. There is no Qt, no GTK, no Skia and no browser underneath it.
What it takes from the operating system is a window, an event queue and
a rectangle of pixels; everything inside that rectangle is its own.

That is not unusual — Avalonia, Fyne and Flutter all draw their own
widgets, and so does Qt in the sense that QStyle paints rather than
calls the platform. What is unusual is **what** it draws: 33 engines
that reproduce the *shapes* of four decades of desktop looks, driving
131 packs, so an application can wear Windows 95, Platinum, Aqua, Luna,
Breeze, Adwaita or Tahoe and be that thing rather than a modern widget
in a period colour scheme.

## At a glance

| | uitoolkit | Qt | GTK 4 | Avalonia | Fyne | WinForms | Win32 | GPUI | AppKit / SwiftUI |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Language | Go | C++ (+ Python, others) | C (+ bindings) | C# / .NET | Go | C# / .NET | C | Rust | Objective-C / Swift |
| Draws its own widgets | yes | yes (QStyle) | yes | yes (Skia) | yes | no — Win32 controls | no — it *is* the controls | yes | yes |
| Renders on | CPU; GPU on Linux | CPU / GPU | GPU (GSK) | GPU (Skia) | GPU (OpenGL) | GDI / GDI+ | GDI | GPU | GPU |
| Linux | first-class | yes | first-class | yes | yes | no | no | in progress | no |
| Windows | yes | yes | yes, second-class | yes | yes | first-class | first-class | in progress | no |
| macOS | yes | yes | yes, second-class | yes | yes | no | no | first-class | first-class |
| Mobile / web | no | yes | no | yes | yes | no | no | no | iOS etc. |
| Complex text (bidi, shaping) | **no** | yes (HarfBuzz) | yes (Pango) | yes | yes | yes | yes | yes | yes |
| Accessibility | Linux only | all platforms | all platforms | all platforms | partial | yes | yes | partial | yes |
| Period-accurate themes | **33 engines, 131 packs** | a few styles | CSS themes | Fluent + CSS-ish | one look | the OS look | the OS look | one look | the OS look |
| cgo / native deps | only where the OS demands | n/a | n/a | n/a | yes (OpenGL) | n/a | n/a | n/a | n/a |
| Binary | 13 MB (21 with every engine) | large, shared libs | large, shared libs | .NET runtime | ~20–40 MB | .NET runtime | tiny | large | n/a |

The uitoolkit column is measured from this repository (see
[Numbers](#numbers)). The rest is a summary of well-known properties,
not a benchmark — check anything you plan to rely on.

## What it does that they do not

**Era engines, not themes.** A GTK theme is CSS over GTK's widgets; a Qt
style is a `QStyle` subclass; both change how a widget looks within one
era's idea of what a widget *is*. uitoolkit's engines change the shapes:
Windows 95 gets real bevels and a 2px outset border, Platinum gets a
title bar of raised ridges with the close box on the left, Aqua gets
pinstripes and a pill button. They are written from documented facts
about how each era drew, never from transcribed code, and they live
behind build tags so an application ships only the ones it wants.

**Everything is one binary.** No shared libraries to find at run time,
no theme engine to install, no GTK version to match. `go build`
produces a file you can copy.

**A very small dependency set.** paintengine2d for the drawing,
`golang.org/x/image` for font parsing, `godbus` for the Linux tray and
portals. That is the list.

**The same pixels everywhere.** Because it rasterises itself, a window
looks the same on Wayland, X11, Windows and macOS, and a headless
render in a test is the same image a user sees. That is what makes the
screenshot tests in this repository possible at all.

## What they do that it does not

**Complex text.** This is the big one. The text stack shapes runs
rune-by-rune with pair kerning from `GPOS`; there is no `GSUB`, no
bidi, no Arabic joining and no Indic reordering. **Arabic, Hebrew,
Devanagari and Thai** render as isolated glyphs in logical order, which
is to say wrongly. Qt has HarfBuzz, GTK has Pango, and if you need
those scripts you need one of them.

**And the bundled face is Latin only.** Titillium Web, which every pack
falls back to, has 456 glyphs: no Greek, no Cyrillic, no CJK, and none
of the arrows, check marks or box drawing a user interface reaches for
— measured, not assumed. Those render only when a pack resolves an
*installed* face that carries them, which it often does on a desktop
and never does under `UITK_SYSTEM_FONTS=0`. There is no font fallback:
a rune the face lacks draws as a box. Symbols are meant to come from
the icon set rather than from a font, which is a defensible line for
✓ and →, and not one for Greek or Cyrillic text.

**Accessibility beyond Linux.** The model in package `a11y` is complete
and every stock widget describes itself, but the platform bridge is
AT-SPI2 on Linux only. UI Automation on Windows and NSAccessibility on
macOS are not written. A screen reader on those platforms sees a blank
window. See [accessibility.md](accessibility.md).

**GPU rendering beyond Linux.** EGL/GLES on Wayland and X11; Windows and
macOS rasterise on the CPU and hand the buffer to the compositor. For
ordinary desktop UI that is fast enough — see [perf.md](perf.md) — but a
scene with heavy continuous animation will feel the difference.

**Printing.** There is none.

**Mobile and web.** There is no plan for either. Avalonia, Fyne and
Flutter go there; this does not.

**An ecosystem.** Qt has thirty years of libraries, designers, widgets
and answers. This has what is in this repository.

**arm64 in production.** Every arm64 target compiles — darwin, windows
and linux — and none has been *run*, because there is no ARM machine
here. If you are on Apple Silicon you are the first.

## When to choose something else

- **You need Arabic, Hebrew or an Indic script.** Qt or GTK.
- **You need a screen reader on Windows or macOS.** Qt, GTK, Avalonia,
  WinForms or the platform's own.
- **You need mobile or web from the same code.** Avalonia, Fyne or
  Flutter.
- **You want the platform's real controls**, with every convention and
  update that comes with them. Win32, WinForms or AppKit.
- **You want the largest ecosystem.** Qt.

## When this one fits

- A Go application that should be **one binary with no native
  dependencies to install**.
- Software that should **look like it belongs to an era** — a retro
  tool, an emulator front end, a demo, a period-accurate utility — or
  that should let the user choose.
- Anything where the **rendering has to be reproducible**: tests that
  assert pixels, documentation generated from the real widgets,
  screenshots that do not drift with the desktop's theme.
- A Linux-first desktop application that also has to run on Windows and
  macOS, where those two do not have to be perfect but do have to work.

## Numbers

Measured in this repository at the version in [version.go](../version.go):

| | |
| --- | --- |
| Theme engines | 33 |
| Theme packs | 131 |
| `uitoolkit-settings`, one engine | 12.98 MB stripped |
| `uitoolkit-settings`, every engine | 21.09 MB stripped |
| The tour sample, the same two ways | 14.80 / 22.92 MB |
| Direct dependencies | paintengine2d, `golang.org/x/image`, `godbus` |
| Platforms with a real backend | Wayland, X11, Win32, AppKit |
| Seams each backend implements | window, frame, geometry, popups, clipboard, drag and drop, cursors, IME — asserted by `platform/seams_*.go` |

The engine and pack counts are what `style.EngineIDs()` and
`style.ListBuiltinThemes()` return with `-tags theme_engine_all`; the
About box in Settings prints them at run time for exactly this reason,
so a number in a document cannot drift from the truth without somebody
noticing.
