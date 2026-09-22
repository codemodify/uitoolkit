# Frames 2 on the rig — the desktop's frame dressed, icons, X11 sync

Nested KWin 6.7.5, instance 29, real GPU; Breeze as the rig's decoration
(`start.sh` now seeds it). 1× at 1280×860, then 1.75 on a 2240×1505 output.
Shots are in `tools/e2e/29/shots/` (not committed).

| # | Check | Result | Shots |
| --- | --- | --- | --- |
| 1 | Luna under KWin's frame, Wayland | blue Breeze title bar (0,89,239) with a white title and the app's icon; with `UITK_DECORATION_PALETTE=0` Breeze's own grey. `WAYLAND_DEBUG` shows `palette_manager.create(…, wl_surface)` and one `set_palette(path)` | `cmp-luna.png` |
| 2 | Backdrop | a second window active: Luna's inactive light blue (124,155,226), light title | `keep-luna-active-backdrop.png` |
| 3 | Aqua, Tahoe under KWin's frame | Aqua's light grey (235) and Tahoe's near white (248) with dark titles, against Breeze's own | `cmp-aqua-tahoe.png` |
| 4 | Luna under KWin's frame, X11 | blue title bar and icon; `_KDE_NET_WM_COLOR_SCHEME` set, `kwin.py` reports the window's `colorScheme` as the file | `x11-luna-server-palette.png` |
| 5 | Window icon | in KWin's frame on both backends, and in KWin's own window list (debug console) | `keep-kwin-window-list-icon.png` |
| 6 | BeOS, Platinum, Window Maker at 1× and 1.75 | R5's tab gaps, Platinum's title on the bar's centre, Window Maker's middle section bevelled | `fid-1x.png`, `fid-175.png` |
| 7 | X11 live switch to the desktop's frame, 1.75 | before: no KWin title bar (frame = client geometry); after the fix: KWin's frame (708 vs 680 tall) in the look's colours, and back to the toolkit's on switching back | `x11-switch.png` (before), `x11-switch-fixed.png` |
| 8 | X11 drag-resize with and without `_NET_WM_SYNC_REQUEST` | the handshake runs (`WM_PROTOCOLS` lists it, the counter is named, the resize completes, no X errors); six mid-drag shots each show the frame and the content in step either way — no black strips, the content edge within the same 5 px of the title bar's end. The synced window trails the pointer by 7–12 px, KWin pacing it | `x11-r6-sync-*.png`, `x11-r6-nosync-*.png` |

Finding from 8: under Xwayland KWin only ever composites whole client
buffers, so the tearing the protocol prevents on a real X server cannot be
caught by a compositor screenshot here. Kept as a gap in polish.md.
