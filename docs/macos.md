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
- **Font enumeration.** There is no fontconfig on macOS, so
  `style/sysfont.go` finds nothing and every pack falls back to the
  bundled Titillium Web and JetBrains Mono. CoreText would enumerate
  the installed families; until it does, a Mac draws the era's
  typefaces in the toolkit's own face. This is why `tools/test.sh`
  pins `UITK_SYSTEM_FONTS=0` — see below.
- **The shortcut modifier.** Command reports as `ModSuper` and Control
  as `ModCtrl`, faithfully, so a shortcut table written as Ctrl+S does
  not fire on Cmd+S. Mapping Command to `ModCtrl` in the backend would
  make `Mods` lie to everything that reads it; the translation belongs
  above `platform`, and is not written yet.

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
