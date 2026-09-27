# The Windows backend

Linux is the shipping platform; this is the second one, and it is new.
What follows is what exists, what does not, and how to run it — the
gaps are listed because a port that hides them is worse than one that
has them.

## What it is

`platform/win32_windows.go` and `platform/win32_frame_windows.go`: a
real top-level window, a message pump, and a DIB blit.

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

## Running it

`cmd/uitk-winsmoke` exists because *"a window appeared"* is not evidence.
It prints the backend, the capabilities, the scale, the window's
position, and every event it saw, so a run in a VM comes back as text
that can be read on the machine the code was written on.

```
uitk-winsmoke.exe            open a window for five seconds and report
uitk-winsmoke.exe -hold 20   keep it up longer
uitk-winsmoke.exe -quiet     capabilities only, no window
```

It paints four quadrants in known colours — red, green, blue, white —
because the failure this backend is most likely to have is the DIB
swizzle: paintengine2d keeps premultiplied RGBA and a 32-bit `BI_RGB`
DIB is BGRX, so **red and blue trading places** is what a wrong swizzle
looks like, and a photograph of the window is enough to see it.

**None of this has been run on Windows yet.** It compiles, it vets clean
for `GOOS=windows`, and every claim above about what Win32 does comes
from the API contracts rather than from having watched it work.
