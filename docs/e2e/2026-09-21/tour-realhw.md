# the tour on real hardware (instance 22)

Nested KWin on the real GPU, `tools/e2e` instance 22 at 1400×900, KWin
ScreenShot2 captures, input through `org_kde_kwin_fake_input`, Wayland
and X11. Nothing was run against the user's session, and every tray item
and notification went to the instance's own D-Bus session. The instance
was stopped and `/run/user/1000/uitk-e2e-22*` removed afterwards.

Build: `examples/tour` from this branch. Shots: `tools/e2e/22/shots/*.png`
(the rig directory is gitignored).

Both backends reported the same starting point:

```
tour: backend=wayland scale=1.00 theme=metal-ocean decorations=client carries-windows=true
tour: backend=x11     scale=1.00 theme=metal-ocean decorations=client carries-windows=true
```

— so on a KDE Plasma desktop the tour gets **the toolkit's own frame** by
the default policy, and the desktop will **carry a window under the
pointer** on both.

## What was checked

| # | what | how | result |
| --- | --- | --- | --- |
| 1 | The strip is the title bar, Wayland | launch | eight tabs and "+" under the pack's own caption strip; the page said `caption  stacked — an era caption above the strip`, which metal-ocean is. `wl-tabs2.png` |
| 2 | **A tab torn out** | `move 351 123 down move 358 128 … move 520 430 up` | a second toplevel `Panels around a centre — tour` appeared **at the drop point** (339,387), holding that one page with its close button hidden; the first window was left with seven tabs. `wl-tearoff.png` |
| 3 | The drag's picture is cleaned up | shot again after the drop | the chip that follows the pointer is gone; it was only still on screen in the frame taken during the drop. `wl-tearoff-settled.png` |
| 4 | **The tab dropped back** | drag it from the second window's strip to the first's, at x 1120 | one window left; the strip read `… Access Docking`, the tab landed at the caret, selected, with its page below. `wl-merge-crop.png` |
| 5 | A panel floated by dragging it out | drag `Outline`'s title bar to 760,862 | a real toplevel `Outline` 268×236 with its own caption; the host's left area emptied and the layout JSON lost its `left`. `wl-dock-floating.png` |
| 6 | A floating panel dragged back over the host | drag its title bar to 230,450 | **the window moved and did not re-dock.** `dock.Host.dragFloatingPanel` needs `widget.DragsWindows` of the *floating* window and falls back to the desktop's interactive move; the main window carries windows but the floating one apparently does not. Reported, not worked around. `wl-dock-redock-crop.png` |
| 7 | The panel's own dock button | click the ⧉ in its title bar | the window closed and the panel docked — but into an area whose weight had collapsed, so it came back **zero pixels wide**. Reported. `wl-dock-back.png` |
| 8 | The page's own "Left" | click it | `Outline` back on the left at full width, the tree in it intact. `wl-dock-left-crop.png` |
| 9 | **The shaped pair** | Shapes → "Open the pair" | `Backdrop` 628×456 and `Shaped` 340×340 over it; the card's colours visible through the ring's hole. `wl-shapes-open.png` |
| 10 | **A click through the hole** | `move 700 450 click` ×3 | the card's counter went to 3 and the ring's stayed at 0; the compositor raised the card, which is itself the proof that the press was delivered to it. `wl-hole-clicks.png` |
| 11 | The same, with the ring kept on top | after the page raises it again | `presses: ring  0` / `presses: card  2 (the ones through the hole)`, `active  yes — the window is that path`, `input rects 250`, `not covered 167 rects`, `glass here  yes`. `wl-hole3.png` |
| 12 | **The desktop's frame** | Frames → "The desktop's frame" | KWin's Breeze caption appeared and the tab strip dropped to being the window's first row. `wl-frame-system.png` |
| 13 | **The toolkit's frame** | Frames → "The toolkit's frame" | the pack's own caption and square era buttons back, no restart, no rebuild. Both in one image: `wl-frame-both.png` |
| 14 | **A drag between two processes** | two tours side by side; drag a row from one's Drafts into the other's Filed | source: `dragging Shipping forecast out of Drafts`, `offers text/uri-list …`; target: `dropped on Filed, gap 1`, `read as text/uri-list`, `from another application`, `action copy`. `wl-cross-crop.png` |
| 15 | The same with Shift held | `key+ 42 … key- 42` | target `action move`, and the row **left** the source list. `wl-cross-move-crop.png` |
| 16 | The tray | Desktop → "Put an icon in the tray" | `tray item  sni, live host: yes` — a real StatusNotifierItem, on the instance's bus. `wl-tray-crop.png` |
| 17 | The strip on **X11** | `UITK_BACKEND=x11` | the same: client frame, eight tabs, `carries-windows=true`. `x11-tabs-crop.png` |
| 18 | **A tab torn out on X11** | as #2 | the second window was made and holds that page — but the window manager placed it at its own cascade position (199,99) rather than at the pointer. Reported. `x11-tear-crop.png` |
| 19 | **The shaped pair on X11** | as #9 | the hole is there (XShape), the card shows through, the counter plate legible. `x11-shapes.png` |
| 20 | **A click through the hole on X11** | as #10 | `backend x11`, `presses: ring  0`, `presses: card  2 (the ones through the hole)`. `x11-hole-crop.png` |

No panics and no fatals from the tour in either backend's log.

## What the rig changed in the app

Four things this pass found, all fixed on the branch:

1. **The Tabs page only knew about merged frames.** On a stacked pack —
   which metal-ocean, the default, is — it claimed the caption buttons
   were beside the tabs when the era's own caption strip was above them.
   It now tells the two apart and says which it is looking at.
2. **The readouts folded wider than the panel they sit in**, so the text
   widget wrapped them a second time and one fact read as two. The fold
   is narrower than the panel at any size the window is likely to be, and
   a value that is one long word (a MIME type) goes on its own line.
3. **A press through the hole raised the card over the ring**, so the
   hole could only be clicked once. The card now puts the shaped window
   back on top, which is what a heads-up window would do and what makes
   the gesture worth repeating.
4. **The card's press counter was unreadable** — white text in the middle
   of the card, behind the hole and over six saturated colours. It is on
   a dark plate in the bottom left now.

## What was found and not fixed

Three, all in `dock/`, `app/` or `platform/`, which this branch stays out
of:

- **A floating dock panel's title bar does not start a re-dock drag**
  (#6). The code for it is there (`dock.Host.dragFloatingPanel`,
  `dock/tearoff.go`) and the fallback to the desktop's interactive move
  is deliberate — but the fallback is what runs, on a compositor whose
  main window reports `carries-windows=true`. Worth checking whether the
  floating window's surface advertises `xdg-toplevel-drag` at all.
- **`Host.DockPanel` can restore a panel to zero width** (#7) when the
  area it belongs to was emptied while it floated: the split's weight
  goes to nothing and is not restored with the panel. The panel is
  present and in the layout JSON, and invisible.
- **On X11 a torn-off window is placed by the window manager** (#18)
  rather than under the pointer, although X11 is the case where the
  client places its own windows and `DragsWindows` is true.
