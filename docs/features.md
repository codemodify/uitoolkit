# Features, and how to switch them on

Everything optional in this toolkit is switched on in one of three
places, and which one is never arbitrary:

| | decided by | when | because |
| --- | --- | --- | --- |
| **a build tag** | the application's author | compile | it costs megabytes, and an application that does not want it should not carry it |
| **an environment variable** | whoever starts the program | run | it is an escape hatch, a test, or one machine being odd |
| **an API call** | the application's code | run | it is a choice the application makes about itself |

A feature is in the first group only when leaving it out makes the
binary meaningfully smaller. Nothing is hidden behind a tag to keep the
API tidy.

This page is the index of all three. [build-tags.md](build-tags.md)
weighs the tags against each other in megabytes;
[building.md](building.md) is the path through writing the application
itself.

## Build tags

```sh
go build                                          # the default product
go build -tags theme_engine_all                   # every look
go build -tags font_complex                       # every script
go build -tags "theme_engine_all,font_complex"    # both; tags compose
```

| tag | what it adds | cost | detail |
| --- | --- | --- | --- |
| *(none)* | Plastik, one face per string, boxes for what it lacks | 13.05 MB | |
| `theme_engine_<id>` | that look; name as many as you like, and the default steps aside | +0.1 MB each, roughly | [engines.md](engines.md) |
| `theme_engine_all` | all 34 engines, 132 packs | **+8.27 MB** | [engines.md](engines.md) |
| `font_fallback` | a rune the face lacks is borrowed from one that has it | **+0.01 MB** | [fonts.md](fonts.md) |
| `font_complex` | shaping, bidirectional text, Unicode line breaking. Implies `font_fallback` | **+2.16 MB** | [complex-text.md](complex-text.md) |
| `font_emoji` | colour emoji from the machine's emoji face. Implies `font_complex` | **+0.01 MB** | [complex-text.md](complex-text.md) |

Tag names cannot contain hyphens, so they use underscores. An unknown
tag is not an error — it simply does nothing, so a typo in
`theme_engine_breze` ships the default engine and no warning. Check
what you built with `style.EngineIDs()`, which the About box prints.

**What no tag changes:** a mirrored right-to-left *interface*, printing,
and IME on the platforms that have none. See
[contracts.md](contracts.md), which is the page to read before promising
a customer anything about text.

## The build itself

```sh
# Linux, as you would ship it.
CGO_ENABLED=0 go build -ldflags "-s -w" -o myapp ./cmd/myapp

# Windows, cross-compiled from Linux. No cgo anywhere in it.
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o myapp.exe ./cmd/myapp

# macOS, on a Mac. AppKit is Objective-C, so this one needs cgo and
# cannot be cross-compiled.
CGO_ENABLED=1 go build -ldflags "-s -w" -o myapp ./cmd/myapp
```

Three things worth knowing about those lines:

- **`-ldflags "-s -w"` is 5.69 MB**, more than every theme engine but
  one put together, and the thing most applications forget.
- **`CGO_ENABLED=0` on Linux** is another 1.27 MB and a binary that
  links no `libX11`, `libwayland-*` or `libxkbcommon` at all — both
  window backends are pure Go. What it gives up is the GPU:
  paintengine2d's device needs EGL and GLES2, so a no-cgo build paints
  on the CPU whatever `UITK_PAINT` says.
- **Windows needs no cgo ever** (the Win32 backend is `syscall`), and
  **macOS always needs it**.

[windows.md](windows.md) and [macos.md](macos.md) have the rest, and
[platform.md](platform.md) is what each backend actually does.

## Environment variables

For whoever *starts* the program — a person trying something, a test, a
container, a machine that is odd. An application should not set these
for its users; everything here has an API or a settings page behind it.

**Look and paint**

| | |
| --- | --- |
| `UITK_THEME=<pack>` | the theme pack, overriding the saved one |
| `UITK_COLOR_SCHEME=dark\|light` | the desktop's light/dark preference, for a process |
| `UITK_ACCENT=#e95420` | the desktop's accent colour, for a process |
| `UITK_SCALE=1.5` | display scale (`GDK_SCALE`, `QT_SCALE_FACTOR` and `GDK_DPI_SCALE` are read too) |
| `UITK_PAINT=gpu\|cpu\|auto` | the renderer, overriding the saved preference |
| `UITK_SCENE=off` | draw immediately instead of recording a retained scene; on by default |
| `UITK_ANIMATIONS=0\|1` | `0` makes every state change instant, `1` animates whatever else says |
| `UITK_DECORATIONS=server\|client\|none` | who draws the window frame |

**Text**

| | |
| --- | --- |
| `UITK_SYSTEM_FONTS=0` | ignore the machine's fonts: every pack reads in the two bundled faces. What makes a screenshot reproducible |
| `UITK_FONT=<family>`, `UITK_FONT_MONO=<family>` | the interface and monospace faces, overriding the saved ones |

**Platform and services**

| | |
| --- | --- |
| `UITK_BACKEND=wayland\|x11\|win32\|appkit\|offscreen` | the window backend, when the automatic choice is wrong |
| `UITK_POPUPS=surface\|layer` | popups as surfaces of their own, or drawn inside the window |
| `UITK_TRAY=fake\|stub` | a recording or no-op status item, for tests and headless runs |
| `UITK_NOTIFY=portal\|fdo\|off` | which notification mechanism |
| `UITK_OPENURI=xdg-open\|off` | skip the portal when opening links, or open nothing |
| `UITK_NATIVE_DIALOGS=1` | the desktop's own file dialogs instead of the toolkit's |
| `UITK_A11Y=1\|0` | the accessibility bridge on or off, whatever the desktop says |

Debugging and development have their own, documented where they are
used rather than here: `UITK_PERF_LOG` ([perf.md](perf.md)),
`UITK_X11_DEBUG`, `UITK_X11_SYNC`, `UITK_XDND_DEBUG`,
`UITK_PAINT_FULLFRAME`, `UITK_LAYER_SHELL`, `UITK_WAYLAND_PRESENT`
([platform.md](platform.md)), `UITK_SHOTS`, `UITK_UPDATE_GOLDEN`
([testing.md](testing.md)).

## Features the application asks for in code

The switches above are for other people. These are the ones an
application makes for itself:

| | the call | page |
| --- | --- | --- |
| a window that turns over to show its back | `Window.Flip`, `Window.FlipPanel` | [widgets.md](widgets.md) |
| animation, per window or per widget | `Window.SetMotion`, `style.SetReduceMotion` | [widgets.md](widgets.md) |
| the status item (tray) | `StatusItemAvailable`, `Application.NewStatusItem` | [tray.md](tray.md) |
| a look of your own, from art | a skin file; no Go at all | [skins.md](skins.md) |
| your own window chrome | a call per piece | [recipes.md](recipes.md#an-application-whose-chrome-is-its-own--a-browser-an-editor) |
| which families lend a missing glyph | `style.SetFallbackFamilies` | [fonts.md](fonts.md) |
| laying out text yourself | `Font.SelectionSpans`, `style.DirectionRuns` | [complex-text.md](complex-text.md) |
| naming a control for a screen reader | `SetAccessibleName` | [accessibility.md](accessibility.md) |
| exporting and loading theme packs | `ExportTheme`, `LoadTheme`, `ListThemes` | [themes.md](themes.md) |

## Checking any of it

Every claim on this page is a command away:

```sh
go build -tags "<the tags>" -ldflags "-s -w" -o /tmp/x ./cmd/uitoolkit-settings && ls -l /tmp/x
tools/test.sh                 # the default build, the whole tree
tools/test-fonts.sh           # font_fallback, font_complex, font_emoji
tools/engines-build.sh        # every engine, built on its own
tools/test-display.sh         # the X11 and Wayland backends, in a nested compositor
tools/test-windows.sh         # the suite on Windows, from here
tools/test-darwin.sh          # the suite on a Mac, from here
```

Where each build has actually been run, as of 2026-10-13:

| | Linux | Windows 11 | macOS 15 |
| --- | --- | --- | --- |
| default | ✅ whole tree | ✅ whole tree | ✅ whole tree |
| `font_fallback` | ✅ `./style/` | — | — |
| `font_complex` | ✅ whole tree | ✅ whole tree | ✅ whole tree |
| `font_emoji` | ✅ `./style/` | — | — |

A dash is "not run", not "known to fail": the text code is the same Go
everywhere, and what differs is which faces the machine has — which is
exactly why the runs that matter are the ones with a real font set
behind them.
