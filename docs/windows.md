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
| `SetPalette` | Takes a path to a KDE colour-scheme file. Windows 11 wants three `COLORREF`s through `DwmSetWindowAttribute`. The signature is an open question, so this answers **false** rather than pretending |
| `SetIcon` | `WM_SETICON` with `ICON_SMALL`/`ICON_BIG`. Not built |
| `MaximizeAxis` | Windows has no per-axis maximize at all. Correctly **false** |
| `SetShadedHeight` | Wants `WM_GETMINMAXINFO`, the analogue of `WM_NORMAL_HINTS`. Unpinning works; pinning does not |
| `SetFullscreen` | Records the state but does not yet save the placement, drop the style and cover the monitor |
| Keyboard | `WM_KEYDOWN`/`WM_CHAR` are not translated yet: the window takes mouse input only |
| Drag and drop | Not started, **on purpose**. `DoDragDrop` is a *blocking* modal call and `IDropTarget::DragOver` answers *synchronously*, where the toolkit's contract is asynchronous. That is a run-loop mismatch, not a vocabulary one, and it should be built against the real API and allowed to dictate the seam rather than designed on paper |
| The suggested DPI rectangle | `WM_DPICHANGED` carries a rectangle Windows would like the window moved to. Reading it means turning an `LPARAM` back into a pointer, which `go vet` will not have; the window is re-sized from its logical size and the new scale instead, which lands in the same place for an ordinary drag between two monitors |

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

## What has actually been run

Under QEMU, cross-compiled from Linux, driven through the QEMU monitor
(`sendkey` in, `screendump` out) with the binary and the logs moving over
the slirp gateway at `10.0.2.2`:

| | |
| --- | --- |
| Windows 10 22H2, 19045.6466 | five runs in five complete; the four quadrants land in the right corners in the right colours |
| Windows 11 24H2, 26100.9168 | five runs in five complete; same |

The quadrants come back byte-identical to what the same drawing calls
produce on Linux — `217,38,38`, `38,179,51`, `38,77,230`, `242,242,242` —
so the BGRA swizzle is right, not merely plausible.

Still unproven on real hardware: everything to do with **scale**. Both
guests run at 96 DPI, so `GetDpiForWindow` answered 96, the scale was 1,
and `WM_DPICHANGED` never fired. The per-monitor DPI declaration and the
scale path have been read, not watched.
