# The macOS backend

Linux is the shipping platform; Windows is the second; this is the
third and it is the newest. What follows is what exists, what does not,
and how to run it — the gaps are listed because a port that hides them
is worse than one that has them.

## What it is

`platform/appkit_darwin.go`, `.m` and `.h`, with the two seams in
`appkit_frame_darwin.go` and `appkit_geometry_darwin.go`: a real
`NSWindow`, an event pump, and a present that is one assignment.

**It needs cgo**, which is the first difference from Windows. The Win32
backend is pure `syscall.NewLazyDLL` and cross-compiles from Linux with
`CGO_ENABLED=0`; AppKit is Objective-C and there is no equivalent. So
the Mac is not merely the machine that runs the binary, it is the
machine that compiles it, and `tools/test-darwin.sh` mirrors the tree
there rather than shipping a binary across.

```sh
# on the Mac
CGO_ENABLED=1 go build -tags theme_engine_all ./examples/uitoolkit-sample-tour
CGO_ENABLED=1 go run ./cmd/uitk-smoke -exercise
```

## Two AppKit facts that shaped the code

**AppKit calls you from inside its own machinery.** A window delegate
is called during `setFrame:`, during a live resize, during ordering a
window in. This is the same hazard the Win32 window procedure has, so
the answer is the same one: the delegate and the view **only append to
a queue**, and `Poll` drains it. Nothing on the Objective-C side of
this backend calls into the toolkit.

**AppKit requires the process's first thread** — not a consistent
thread, that one. `platform/mainthread_darwin.go` locks it in an `init`,
as `mainthread_windows.go` does, and for the reason spelled out there:
a backend whose correctness depends on a rule enforced one layer up is
not a backend, it is a trap. `go test` runs every test on a goroutine
of its own, so `appkit_window_darwin_test.go` keeps the first thread in
`TestMain` and hands it back with `onMain`.

## The present, and the bug it hid

paintengine2d keeps premultiplied RGBA and `CGBitmapContext` takes
premultiplied RGBA, so the buffer goes up as it is — none of the BGRX
swizzling the Windows DIB needs. The present is a `CGImage` assigned
straight to `contentView.layer.contents`, inside a `CATransaction` with
actions disabled so the default implicit animation does not cross-fade
every frame. There is no `drawRect:` anywhere: the toolkit draws when
it says so, not when AppKit does.

The one trap is worth writing down, because it is invisible and it cost
a release's worth of frames before a test caught it.

The input view wanted `isFlipped` to return YES, so that mouse
coordinates would come out y-down like the rest of the toolkit. A
layer-backed **flipped** `NSView` gets `geometryFlipped` on its backing
layer, which gets `contentsAreFlipped`, which draws the layer's
`contents` image **upside down** — and here the contents image is the
whole window. Every frame was inverted, and nothing said so. The view
is unflipped now and the y flip for input is one subtraction in
`-[UitkView where:]`.

## What the capabilities say

`FrameCaps` here is move, minimize, maximize, fullscreen, keep-above,
lower, icon, client-frame, **system-shadow** and
**system-resize-band**.

The last two are the ones to read. AppKit draws the drop shadow itself,
outside the window, and a resizable `NSWindow` already has invisible
resize borders AppKit manages. A backend that reserved a margin for a
shadow here would get **two** shadows — its own inside the window and
the system's around it — and a window that lines up with nothing else
on the desktop. macOS is the only platform that sets both bits;
Windows sets the shadow one alone, because a window that owns its
non-client area answers `WM_NCHITTEST` for its own edges.

`GeometryCaps` is move, position, screen-place, visibility and
size-limits — all of them, which Wayland cannot say. macOS puts a
window where it is asked and tells you where it ended up.

## What is missing, and why

| Withheld | Why |
| --- | --- |
| `FrameResize` / `StartResize` | There is no public API to begin an interactive resize from an edge. `FrameSystemResizeBand` says the toolkit needs none. |
| `FrameMenu` / `ShowMenu` | macOS windows have no window menu. The toolkit shows its own. |
| `FrameMaximizeAxis` | Zoom is both ways or neither. |
| `FrameShade` / `SetShadedHeight` | Window shade went out with Mac OS 9. |
| `FramePalette` / `SetPalette` | The file it names is a KDE colour scheme. AppKit's frame takes its colours from the system appearance. |
| `FrameBlurBehind` | `NSVisualEffectView` is a *view*, not a window property, and the frame seam has no way to put one behind a buffer the toolkit paints. |

The **diagonal resize pointers** bend the same way. AppKit publishes no
corner-resize cursor: the one the window frame draws is private, and the
public `+[NSCursor frameResizeCursorFromPosition:inDirection:]` is too
new to rely on — it is missing from command line tools that are
otherwise current, so calling it would make the toolkit fail to build on
up-to-date Macs. The arrow stands in, and it costs little: macOS resizes
windows from its own band, so a diagonal is only ever asked for by
something the toolkit draws inside its own window, and a splitter is
horizontal or vertical.

`SetIcon` is the one that bends rather than refuses. macOS has no
per-window icon — a title bar shows one only for a document, and that
one is the document's file — so it sets the **application's** icon, the
Dock tile. The last window to ask wins, which for the one-icon-per-app
case every caller actually has costs nothing, and it is worth having: a
Go binary outside an `.app` bundle otherwise gets the blank generic
tile.

### Still to do

- **IME.** `NSTextInputClient` is not implemented, so there is no
  preedit and CJK input does not work. This is the same gap Windows
  has (`WM_IME_*`), and the largest one on either platform.
- **Drag and drop.** `NSDraggingSource` and `NSDraggingDestination` are
  not wired up.

## ⌘S and Ctrl+S are the same shortcut

A menu writes "Ctrl+S" on Linux and Windows and "⌘S" on a Mac, and it
is one shortcut. The toolkit writes it once — `ParseAccel` has always
read `Cmd` and `⌘` as `ModCtrl` — and the other half is turning what a
key press actually carried into what a table means.

The backend does **not** do it. Command is reported as `ModSuper` and
Control as `ModCtrl`, faithfully, because that is what the keyboard did
and `Event.Mods` is what the keyboard did; a backend handing the
toolkit a `ModCtrl` nobody pressed would make every other reader of
`Mods` wrong.

Instead `Window.dispatch` calls `platform.AccelMods` once, on the way
from the platform to the widgets. On macOS the two modifiers **swap**
rather than Command merely becoming Control, so they stay distinct: ⌘A
is Select All, and a Mac's Control+A is the emacs-ism for the start of
the line, which should not fire it. Everywhere else it is the identity.

Doing it at that one point is what makes it small. Fifty places in
`widgets` ask `e.Mods.Ctrl()` — a text field copying, a list
extending a selection — and all fifty are right on a Mac without being
touched.

## Finding the fonts without fontconfig

`style/sysfont.go` asks `fc-list`, and macOS has no fontconfig at all.
So the index came back empty, every pack fell through to the bundled
Titillium Web, and a toolkit that ships 131 packs across four decades
drew all of them in one face.

The fallback in `style/sysfont_scan.go` walks the directories macOS
keeps its fonts in and reads each file's own `name` and `OS/2` tables,
which is what `fc-list` does underneath. It is pure Go, it has no
build tag, and it fixes **Windows at the same time** — Windows has no
fontconfig either, and needed no DirectWrite or registry reading for
this.

`/System/Library/Fonts/Supplemental` is in the list and matters: that
is where Helvetica, Times, Courier, Monaco and Geneva live, which is
to say every face the older packs ask for by name. On this Mac the
scan finds **341 families**, and Helvetica, Helvetica Neue, Lucida
Grande, Geneva, Monaco, Menlo, Courier, Times New Roman, Tahoma,
Verdana, Arial and Georgia all resolve — so Aqua, Platinum and NeXT
draw in their own typefaces here rather than in the toolkit's.

Two details were not guesses:

- **Weight comes from `OS/2`, not from the name**, because a subfamily
  string is localized and a French system says "Gras". But the
  `fsSelection` BOLD *bit* beats `usWeightClass`, because they
  disagree: the toolkit's own JetBrains Mono Bold declares
  `usWeightClass` 558 — it was cut from a variable font and the axis
  value came with it — and reading the number alone files the bold
  face under medium.
- **It runs on every core.** One thread took 803 ms on this Mac, which
  is long enough for the first window to want a font before the index
  has one; the files are independent, so a worker per core takes
  115 ms. On Linux it is 30 ms, and unused, since `fc-list` answers
  first.

### One thing it turned up, still open

Running the suite with `UITK_SYSTEM_FONTS=1` on a Mac is a
configuration that had never existed, and it fails one test:
`TestPlatinumTitleCentresOnTheBar`, which compares the title the
desktop's frame draws against the same title drawn by an in-app frame.

With the bundled face the two agree exactly, at every title length.
With Geneva — which is the right font for Platinum, and which is the
point — they agree for a long title and differ for short ones: for
"Untitled" the frame's ink is 42 px wide against the in-app frame's
62, and for "Ab" the frame has no ink at all in the rows the test
samples. Geneva's metrics also make the caption band 27 px tall where
the bundled face makes it 31.

So it is a real difference between the two frames and not a flaky
measurement, but it is in Platinum's caption, not in this backend, and
it needs its own look. The suite's default is `UITK_SYSTEM_FONTS=0`,
so it is not failing anywhere today.

Checked against fontconfig on a Linux machine with 596 family keys
installed, the scan finds 590 of them. The six it does not are the
weight-suffixed spellings fontconfig invents for a variable font's
named instances — "Noto Sans Syriac Thin" — which `sfnt` cannot draw
anyway, and which the existing lookup already skips.

## A capability is not done until the state says so

`SetKeepAbove` worked from the first day: the window really did float,
and `uitk-smoke -exercise` reported `SetKeepAbove yes`. It was still
broken, and a user found it in a minute.

`WindowState` never carried `KeepAbove`. The toolkit's toggle is
`SetKeepAbove(!WindowState().KeepAbove)` and a caption's pin draws
itself from the same field, so with the state stuck at false **every
press meant "on"**: the pin never changed, and a window once pinned
could not be released. Windows had always set it in its own
`SetKeepAbove`; AppKit had not, and nothing compared the two.

The state is read back from the window's level now rather than
remembered from the last request, and `SetKeepAbove` pushes an
`EventWindowState` — AppKit has no notification for a level change to
hang it on. Three tests cover it: that the state reports it, that the
change is announced, and that a toggle goes both ways. The last one is
the user's actual gesture, and it is the one a test asserting only
"`SetKeepAbove` returned true" will never catch.

## The clipboard and the pointer

`NSPasteboard` is the easiest clipboard of the three. X11 has to own a
selection and keep answering requests for it after the window is gone;
Wayland has to hold a `wl_data_device`; NSPasteboard is a *server* the
system runs, so setting a string hands it over and there is nothing to
serve afterwards — no background drainer, and no paste that can hang
waiting for another application. There is no PRIMARY selection, exactly
as on Windows, so `ClipboardPrimaryGet` answers from the same place
rather than pretending to a second one.

The pointer is `NSCursor`, and like `WM_SETCURSOR` on Windows it takes
two halves: setting the shape now, for a widget the pointer has just
crossed into, and answering `-[NSView cursorUpdate:]` every time AppKit
asks afterwards. A backend that does only the first gets the
arrow-over-text flicker.

## Testing it

```sh
export UITK_MAC_HOST=user@the-mac
tools/test-darwin.sh ./...
tools/test-darwin.sh -run TestAppKit -v ./platform/
```

The script mirrors the tree with rsync, builds on the Mac and brings
the output back. It is ssh rather than the HTTP-over-slirp contraption
`tools/test-windows.sh` needs, because a Mac already has sshd and can
be told to turn it on in one line.

These tests open real windows on the machine that runs them. That is
allowed here in a way it is not on Linux: `tools/testenv.sh` exists to
keep the suite off a desktop somebody is using, and there is no macOS
equivalent because there is nothing to wrap — no Wayland, no X11, no
session bus.

### Looking at what it drew

**`screencapture` run over ssh does not work, and does not say so.**
It has no Screen Recording permission, and rather than failing it
returns a clean-looking PNG of the desktop and the menu bar with
**every application window silently missing** — confirmed with a
TextEdit control whose document window was absent from the capture
too. A blank-looking screenshot reads exactly like a window that never
opened, and it sent one debugging session after a backend that worked.

There is no unprivileged way around it on macOS 15:
`CGWindowListCreateImage` is obsoleted and ScreenCaptureKit wants the
same grant.

What works is reading the window's **own layer** back, which needs no
permission: `uitk_ak_readback` renders the layer tree into a bitmap
context and the test asserts known pixels. It is better than a
screenshot anyway — it is text, and text can be asserted on. That is
what caught the upside-down window above.

Synthetic input goes the same way. `[NSApp postEvent:atStart:]` puts an
event in the application's own queue and needs no Accessibility grant,
unlike driving the system pointer with `CGEvent`. The poster takes
**AppKit's** coordinates, y up, deliberately: a poster that undid what
`-[UitkView where:]` does would make a test of where a press lands a
round trip through one function, which passes however wrong that
function is. It did, and the window was upside down anyway.

### Why the suite pins the fonts

`tools/test.sh` and `tools/atlas/render.sh` both set
`UITK_SYSTEM_FONTS=0`. A look reads in the era's typeface when the
machine has it, so what a look *measures* — where a panel ends, how a
sentence wraps — depends on what is installed. Tests that assert a
pixel then pass on the machine they were written on and fail elsewhere:
three did, on macOS, and would have in any clean container. Pinning the
bundled faces makes those numbers a property of the toolkit.

## What has actually been run

On a MacBook running macOS 15.7.9, Apple silicon, at a backing scale of
2:

- `cmd/uitk-smoke -exercise`: a window, the four-quadrant present, and
  every frame and geometry capability called and reported.
- `tools/test-darwin.sh ./...`: **the whole suite, green**, with
  `-tags theme_engine_all`.
- The fifteen AppKit tests in `platform`: present, buffer size, mouse,
  keyboard, Command-is-not-text, geometry round trip, fixed-window
  caps, the refusals, maximize, and decorations off and back on.

Not run: anything with IME, the clipboard, drag and drop or a pointer
shape, because none of them exists yet.
