# The Windows backend

Linux is the shipping platform; this is the second one, and it is new.
What follows is what exists, what does not, and how to run it — the
gaps are listed because a port that hides them is worse than one that
has them.

## What it is

`platform/win32_windows.go` and `platform/win32_frame_windows.go`: a
real top-level window, a message pump, and a DIB-section blit.

**It is pure Go.** Every call goes through `syscall.NewLazyDLL`, the way
`status_windows.go` already talks to shell32 and user32, so the backend
**cross-compiles from Linux with `CGO_ENABLED=0`** and needs no compiler
in the guest:

```sh
GOOS=windows GOARCH=amd64 go build -tags theme_engine_all \
    -o /tmp/tour.exe ./examples/uitoolkit-sample-tour
GOOS=windows go build -o /tmp/uitk-winsmoke.exe ./cmd/uitk-winsmoke
```

That is worth protecting. The Linux backends need cgo, and a Windows
backend that needed a toolchain on Windows would be far harder to keep
honest from a Linux desk.

## Two Win32 facts that shaped the code

**The window procedure is re-entrant.** `SetWindowPos` and `MoveWindow`
dispatch `WM_SIZE` synchronously, and `WM_ENTERSIZEMOVE` runs a modal
loop *inside Windows* for the whole of a resize drag. Anything the
procedure does happens while the toolkit may be in the middle of its own
call. So the procedure **only appends to a queue**, and `Poll` drains
it — never the reverse. This is the structural difference from the
Wayland and X11 backends, where events arrive on the toolkit's own terms.

**Messages belong to the thread that created the window.** They are
delivered only to that thread's queue, which is why the run loop is
pinned to the main OS thread in `app`'s `init` — a rule written down
before there was a backend that needed it.

It is now pinned in `platform/mainthread_windows.go` as well, because
`app` is not the only way in: `cmd/uitk-winsmoke` uses `platform`
directly, and without the lock Go moved the main goroutine off the
thread that had just created the window. `ShowWindow` then became a
cross-thread `SendMessage` to a thread nobody was pumping, and it never
returned. **Measured on Windows 10 19045: four runs in five wedged**,
every stack pinned on the same `ShowWindow` call. A backend whose
correctness depends on a rule enforced one layer up is not a backend,
it is a trap.

## What the capabilities say

`FrameCaps` here is move, resize, menu, minimize, maximize, fullscreen,
**keep-above**, **lower**, icon, client-frame and **system-shadow**.

The asymmetry against Wayland is the point. Keep-above and lower are
present here and absent there; a window menu is Windows' own; and
`FrameSystemShadow` says DWM draws the drop shadow itself, outside the
window, so a frame the toolkit draws reserves no margin for one. Nothing
above the boundary changed to accommodate any of it — which is what the
reshaped seam was for.

## What is missing, and why

| | |
| --- | --- |
| `MaximizeAxis` | Windows has no per-axis maximize at all. Correctly **false** |
| *(nothing)* | Every seam the Linux backends implement, this one now implements too — `seams_windows.go` asserts it and the compiler keeps it true |

**Saving preferences retries its rename**, and that is Windows-specific
and was a real bug. `style.writeJSONFile` writes a temporary file and
renames it over the target, which is atomic on Unix. On Windows a
rename over a file another process has open fails — Go opens files for
reading without `FILE_SHARE_DELETE` — so publishing `look.json` failed
exactly when another uitoolkit application happened to be reading it.
That is not a rare race: every application *watches* that file, so a
save races every other running app by design, and what a user saw was
Apply reporting that it could not write the preferences. It retries for
a few milliseconds now, which is longer than a reader holds the file and
costs nothing on Unix, where the first attempt always succeeds.

It was found by widening `tools/test-windows.sh` to the whole suite —
`TestConcurrentSavesNeverPublishHalfAFile` had never run on Windows.

**Two things used to be on this list and are not any more.**

**Popups.** A menu, a combo list and a tooltip were rectangles painted
inside the window, clipped as soon as one was taller than the room
under it. They are `WS_POPUP` windows now, `WS_EX_NOACTIVATE` and
owned by the window they hang from, with a class of their own for
`CS_DROPSHADOW` — that is what gives a Windows menu its shadow, and a
top-level must not have it or it gets DWM's as well. Placement is
`SolvePopup` against the monitor's work area, the same function X11 and
the headless tests use.

**IME.** `WM_IME_STARTCOMPOSITION`, `WM_IME_COMPOSITION` and
`WM_IME_ENDCOMPOSITION` now reach `EventIMEPreedit` / `EventIMECommit`
/ `EventIMECancel`, with the composition string read out of the input
context with `ImmGetCompositionStringW`. Two details decided the code:
`WM_IME_SETCONTEXT` clears `ISC_SHOWUICOMPOSITIONWINDOW`, or Windows
draws the preedit in a box of its own and the same text is on screen
twice; and the caret from `GCS_CURSORPOS` counts UTF-16 units where
`Event.IMECaret` is a byte offset, which part company on exactly the
characters an input method is for. The candidate list is left to
Windows — it is the method's window, every application gets the same
one — and what the toolkit owes it is the caret, through
`ImmSetCandidateWindow` with `CFS_EXCLUDE` so a list near the foot of
the screen opens above the caret instead of over it.

**Installed fonts used to be on this list and are not any more.**
`style/sysfont.go` finds faces through `fc-list`, and Windows has no
fontconfig, so every pack fell back to the bundled Titillium Web — a
Luna that could not draw in Tahoma. `style/sysfont_scan.go` walks
`%WINDIR%\Fonts` and the per-user font directory and reads each file's
own `name` and `OS/2` tables instead. It is pure Go, so it costs the
`CGO_ENABLED=0` build nothing, and it needed no DirectWrite and no
registry. It was written for macOS, which has the same gap; see
[macos.md](macos.md) for what it does and what it was checked against.

## Why the pixels go up through a DIB section

`Present` does not hand GDI a pointer to Go memory. It keeps a
`CreateDIBSection` bitmap selected into a memory DC, swizzles into
**that**, and `BitBlt`s the damaged rectangles to the window.

The first version called `SetDIBitsToDevice` with `&s.bgra[0]`, and on
Windows 10 19045 it answered **0 scan lines set** — every frame, with
`GetLastError` reporting success, so the window came up white and said
nothing. What the probe found, against the same DC, same header and same
clip region:

| | |
| --- | --- |
| `SetDIBitsToDevice`, Go buffer, top-down | 0 |
| `SetDIBitsToDevice`, Go buffer, bottom-up | 0 |
| `StretchDIBits`, Go buffer | 0 |
| `SetDIBitsToDevice`, **one scan line** of the same Go buffer | 1 |
| `SetDIBitsToDevice`, the same buffer with **a page of slack after it** | 400 |
| `SetDIBitsToDevice`, bits GDI allocated | 400 |
| `CreateDIBSection` + `BitBlt` | ok |

So GDI reads past the end of the bits it is given. The buffer was
`make([]byte, 640*400*4)` — 1,024,000 bytes, exactly 250 pages — sitting
at the edge of Go's committed heap, and the overrun faulted into
reserved-but-uncommitted memory.

**That makes it a heap-layout bug, not a Windows-version bug**, which is
the dangerous part: the same binary that failed every time on the
Windows 10 image painted perfectly on Windows 11 24H2, because there
happened to be a committed page after the buffer. A DIB section has no
such edge — the memory is GDI's, sized by GDI — and blitting from a
memory DC is the faster path anyway, since the header is parsed once at
creation instead of once a frame.

## The clipboard, the pointer and the caption's colours

Three things that were not built and now are, and one note each on the
part that is easy to miss.

**The clipboard** is `OpenClipboard` / `GetClipboardData` /
`SetClipboardData` with `CF_UNICODETEXT`. `OpenClipboard` is a *lock* on
a single system-wide resource, not a handle: every path closes it,
nothing that can block happens while it is held, and failing because
another application holds it is ordinary — so it is retried over about a
quarter of a second and then given up on rather than waited for.
`SetClipboardData` **gives the memory away**; the block is only ours to
free if it refuses. There is no PRIMARY selection here, so
`ClipboardPrimaryGet` answers from the same place rather than inventing
one.

**The pointer's shape** needs two halves. `SetCursor` changes it now, and
`WM_SETCURSOR` answers for it — Windows asks the window again on every
move inside it and the default answer puts the class cursor back, which
is exactly the arrow-flickering-over-text that a backend doing only the
first half gets. Only over the client area: over the frame Windows' own
shape is right, and answering there would take the resize arrows away.
`WM_MOUSELEAVE` had to come with it, because there is no such message
without asking for it — a window that never calls `TrackMouseEvent`
believes the pointer is still over it for ever.

**`SetPalette`** turned out not to be blocked at all. Its signature names
a KDE colour-scheme file, which looked like the wrong shape for a
platform that wants three `COLORREF`s — but the file is the *source* of
the colours, not a KDE-only mechanism, and `platform` already parses that
format for the title-bar preferences. The `[WM]` group's
`activeBackground` and `activeForeground` become
`DWMWA_CAPTION_COLOR` and `DWMWA_TEXT_COLOR`. `FramePalette` is probed
for rather than assumed, because the attributes arrive in Windows 11
22000 and the capability has to say what *this machine* will do.

## Drag and drop, and the one thing it costs

Both halves are built, on OLE, in Go without cgo: a vtable of
`syscall.NewCallback` entries and objects whose first word points at it.

**Taking a drop** is `RegisterDragDrop` and an `IDropTarget`. The
mismatch it was deferred for is real and is bridged rather than solved:
`IDropTarget::DragOver` must answer *now* with the effect the window
would have, and the toolkit answers *later*, because `EventDragMotion`
goes on the queue and the application replies with `AcceptDrag` when it
gets to it. Making the synchronous question wait would mean running the
toolkit's loop inside a COM call. So the answer given is the last one the
application settled on, and the motion is queued for the next turn: a
drag that has just arrived is offered whatever that window last agreed to
take and is corrected a frame later.

**Starting a drag** is `DoDragDrop`, and **it blocks** — a modal loop of
its own until the user drops or gives up. `StartDrag` therefore records
the payload and answers at once; `Poll` runs the drag on the next turn,
so the blocking happens inside the toolkit's loop where blocking is
already expected, rather than half way down an event handler.

What that costs, plainly: **for the length of the drag the source window
does not repaint.** Windows pumps messages during its modal loop, so the
window procedure runs and everything it queues is waiting afterwards, but
nothing drains that queue until the drag ends. An application that paints
from `WM_PAINT` does not have this problem and one that paints from its
own loop does. The alternative is re-entering the toolkit from inside
`IDropSource::GiveFeedback`, which Windows calls throughout — the
re-entrancy the window procedure exists to avoid.

Files cross as a `text/uri-list` and text as UTF-8, both ways, so nothing
above the boundary learns what a path looks like on Windows.

## Building an app for Windows

Pass `-ldflags "-H windowsgui"`. Go links a Windows binary for the
console subsystem by default, so without it every uitoolkit app opens a
console window beside its own — and that console takes the keyboard
focus, which makes the app look as though it ignores typing:

```sh
GOOS=windows GOARCH=amd64 go build -tags theme_engine_all \
    -ldflags "-H windowsgui" -o settings.exe ./cmd/uitoolkit-settings
```

`cmd/uitk-winsmoke` is the exception and wants the console: it reports
what it saw as text.

## Running it

`cmd/uitk-winsmoke` exists because *"a window appeared"* is not evidence.
It prints the backend, the capabilities, the scale, the window's
position, and every event it saw, so a run in a VM comes back as text
that can be read on the machine the code was written on.

```
uitk-winsmoke.exe                open a window for five seconds and report
uitk-winsmoke.exe -hold 20s      keep it up longer (a duration, with its unit)
uitk-winsmoke.exe -quiet         capabilities only, no window
uitk-winsmoke.exe -watchdog 15s  panic after this long, to dump the stacks of
                                 a window that will not open
```

It paints four quadrants in known colours — red, green, blue, white —
because the failure this backend is most likely to have is the DIB
swizzle: paintengine2d keeps premultiplied RGBA and a 32-bit `BI_RGB`
DIB is BGRX, so **red and blue trading places** is what a wrong swizzle
looks like, and a photograph of the window is enough to see it.

It also reports what `Present` returns. That matters more than it
sounds: the first version threw the result of the blit away, which is
precisely why a window that never showed a pixel looked exactly like one
that worked.

## Testing it

Two scripts, because the two halves cost very different amounts:

```sh
tools/build-windows.sh                     # seconds, needs no Windows
UITK_WIN_VM_MON=/path/to/monitor.sock \
  tools/test-windows.sh                    # runs the tests on real Windows
```

**`tools/build-windows.sh`** cross-compiles and vets the whole tree for
`GOOS=windows`, and compiles the Windows *test* binaries too so they
cannot rot. It catches nothing that a type checker cannot see — which is
the point of saying so.

**`tools/test-windows.sh`** is the one that matters. It cross-compiles
`platform`'s tests, hands the binary to a Windows VM over QEMU's user
network (the guest reaches this host at `10.0.2.2`; nothing is exposed
outside loopback), runs them there, and brings the output back. The guest
needs two things set up once: an interactive logged-in desktop, because
the tests make real windows and a window needs a session to appear in;
and Microsoft Defender told to leave the directory alone, because it
quarantines freshly built unsigned Go binaries.

### Why this exists at all

Every file behind `//go:build windows` is invisible to `tools/test.sh`:
Go does not compile it on Linux, so the suite cannot fail on it. At the
point the backend first ran, `platform/` held **seven Windows source
files and no Windows test files**, and nothing in `tools/` built for
`GOOS=windows` at all. Three bugs went out through that gap, and all
three were the same shape — *a Win32 call whose result nobody read*:

| | what the test asserts now |
| --- | --- |
| `ShowWindow` blocked for ever without the main thread pinned (four runs in five) | windows open five times over without the test timing out |
| `SetDIBitsToDevice` answered "0 scan lines set" every frame and the window stayed white | `Present` returns an error, and the swizzled bytes are BGRX |
| `RequestDecorations` recorded the mode and did nothing to the window | the client rectangle equals the window rectangle, and `WS_CAPTION` survives |

## What has actually been run

Under QEMU, cross-compiled from Linux, driven through the QEMU monitor
(`sendkey` in, `screendump` out) with the binary and the logs moving over
the slirp gateway at `10.0.2.2`:

| | |
| --- | --- |
| Windows 10 22H2, 19045.6466 | five runs in five complete; the four quadrants land in the right corners in the right colours |
| Windows 11 24H2, 26100.9168 | five runs in five complete; same |

Keyboard, on Windows 11 24H2: Tab moves the focus ring between controls,
and typing `keram` into Settings' theme filter narrows 131 packs to
`2002 · Keramik`. So `WM_KEYDOWN` reaches the toolkit as a key and
`WM_CHAR` reaches a text field as text.

One trap worth writing down for anyone testing this way: Microsoft
Defender quarantines freshly built, unsigned Go console binaries copied
into a guest. The file vanishes, or survives and will not start —
"The system cannot execute the specified program" — which looks exactly
like a broken build. A GUI-subsystem binary of the same code was left
alone.

The quadrants come back byte-identical to what the same drawing calls
produce on Linux — `217,38,38`, `38,179,51`, `38,77,230`, `242,242,242` —
so the BGRA swizzle is right, not merely plausible.

Still unproven on real hardware: everything to do with **scale**. Both
guests run at 96 DPI, so `GetDpiForWindow` answered 96, the scale was 1,
and `WM_DPICHANGED` never fired. The per-monitor DPI declaration and the
scale path have been read, not watched.
