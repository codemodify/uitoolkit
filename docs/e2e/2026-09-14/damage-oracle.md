# damage-oracle findings (instance 20)

## BUG 1: Full-frame presents are recorded as "no damage" in the EGL buffer-age ring -> next partial present shows a 2-frames-old buffer (menus vanish, stale screens)
- app: engine (affects every app/window on the GPU path: gallery, mail, ...)
- severity: critical
- partialOnly: true (never with UITK_PAINT_FULLFRAME=1)
- confidence: high
- repro: rig 20: `run.sh 20 gallery; in.sh 20 move 165 80 sleep 200 click sleep 600; shot.sh` -> File title shows pressed but the File menu popup is NOT on screen (2nd run of 2 in default mode; 2/2 OK with FULLFRAME). Shots: shots/t_click_default_2.png (bad) vs t_click_full_1.png; zoom: shots/t_click_cmp.png. WAYLAND_DEBUG trace: press frame = full swap (damage_buffer 0,0,INT_MAX,INT_MAX) into new wl_buffer#35; release frame = partial swap damage_buffer(0,0,1000,31) into wl_buffer#36, which still holds the hover frame from BEFORE the menu opened (buffer age 2) -> only the menubar strip is refreshed, rest of the window is the pre-menu image.
- actual: after any full repaint (popup/menu/combo open or close, dialog open/close, relayout, tab switch...), the next partial frame is blitted into a back buffer that is 2 frames old and only its own dirty rects are refreshed -> the whole screen outside those rects reverts to the frame before the full repaint (menu disappears, closed popup reappears, etc.); alternates frame to frame = "glitching".
- expected: back buffer repaired with the full damage of the intervening full frame.
- rootCause: paintengine2d/gpu_linux.go PresentRects (1425-1505): a full present has rects==nil; line 1496 `d.recordFrameDamage(rects)` stores an EMPTY slot for it (recordFrameDamage 1508-1513 appends nothing). damageForAge (1517-1526) then unions nothing for that frame, so with age>=2 the scissored blit (1478-1479 blitDamageRects) + eglSetDamageRegionKHR (1460-1461) only cover the current rects although the intervening frame changed the whole FBO.
- proposedFix: in recordFrameDamage, when len(rects)==0 store a full-surface rect (XYWH(0,0,d.w,d.h)) or a per-slot `full` flag; in damageForAge return nil/force `partialBlit=false` when any of the age-1 previous slots is full. Also mark all ring slots full on Resize/allocTarget.
- files: pe/gpu_linux.go

## BUG 2: Wayland GPU present silently drops a frame while a frame callback is pending; its damage is discarded and nothing re-presents -> last state of a quick click/hover never shown, stale content persists
- app: platform (all apps, Wayland GPU path)
- severity: major
- partialOnly: false (FULLFRAME also shows the stale LAST frame; but only partial mode keeps it stale after later frames)
- confidence: high
- repro: gallery, `in.sh 20 move 232 331 sleep 300 down sleep 4 up sleep 500` (fast click on "Primary action") then shot; then `move 900 500 sleep 500` and shot. Repeat 3x.
  FULLFRAME: 3/3 the first shot shows the button still PRESSED and "Clicked N-1 times" (release frame dropped); heals on the next frame.
  default: 1/3 shows pressed + old count, and after moving away (a later partial frame) the label STILL says "Clicked 1 times" while the app state is 2 (permanently stale until that label is repainted again).
  evidence: shots/t_fc_both.png (left=default rows 1a,1b,2a,2b,3a,3b; right=FULLFRAME).
- actual: UI appears to lag one step behind / not register fast clicks; partial mode leaves permanently stale regions.
- expected: every painted frame is eventually presented; damage accumulates until presented.
- rootCause: platform/wayland_linux.go:1841-1843 (wlSurface.Present GPU path) returns nil without swapping when `s.framePending && !s.frameOverdue()`. The app already replayed the frame into the GPU FBO and app/window.go:908-911 resets w.dirty/w.full unconditionally; the GPU device's pending present rects (SetPresentDamage) are then REPLACED by the next frame's rects (pe/gpu_linux.go:1382-1392), so the dropped frame's damage is never blitted to any window buffer; recordFrameDamage is never called for it either, so buffer-age repair does not include it. uitkWlFrameDone (wayland_linux.go:1924) only clears framePending and does not wake/re-present.
- proposedFix: make a skipped present keep its damage: (a) GPUDevice.SetPresentDamage should union with not-yet-presented rects when presentSet is still true (nil wins -> full); (b) wlSurface.Present should return a "deferred" status (or keep a pendingPresent flag) and uitkWlFrameDone should wake the loop (platform.WakeSurface) so Window.frame re-presents; Window.frame must not Reset dirty when the surface deferred (or the platform should re-present the accumulated damage itself on frame-done).
- files: platform/wayland_linux.go, platform/wayland_egl_linux.go, app/window.go, pe/gpu_linux.go

## BUG 3: ToolBar paints EVERY tool button hot while the pointer is anywhere over the toolbar (same pattern as the fixed TabBar bug)
- app: gallery (any app with widgets.ToolBar: gallery, mail, notes, files...)
- severity: major
- partialOnly: false (also in FULLFRAME)
- confidence: high
- repro: gallery, `in.sh 20 move 167 153` (hover the first toolbar icon) -> all 8 tool buttons (New, Open, Save, Cut, Copy, Paste, About, Snap) paint the hot/accent fill; moving off the bar clears them. evidence: tour/tb_hover2.png (rows: idle, hover New, hover Open, hover About, hover Snap, away) from tour/oracle/0[6-9]*_a.png
- actual: whole toolbar lights up; user cannot tell which button is under the pointer.
- expected: only the tool button under the pointer is hot (pressed only the pressed one).
- rootCause: widgets/toolbar.go:234 `st := t.State()` seeds every item's ControlState with the bar's own Base state; Base.State() (widget/base.go:281-293) includes StateHovered whenever the pointer is anywhere on the bar (Base.MouseEnter sets hovered). Only StateFocused is stripped (235-237), so every item gets StateHovered. (TabBar had the identical bug, fixed in 0719408; MenuBar.Paint menu.go:182-183 strips it correctly.)
- proposedFix: `st := t.State() &^ (style.StateHovered | style.StatePressed)` before adding per-item hover/press, exactly like tabs.go:83 / menu.go:183.
- files: widgets/toolbar.go

## BUG 4: Popup menu hover highlight leaves stale accent strips on both ends of the previously hovered row
- app: gallery (all apps: menubar menus, context menus - any widgets.PopupMenu)
- severity: minor (very visible though: blue slivers stack up as you move through a menu)
- partialOnly: true
- confidence: high
- repro: gallery: `in.sh 20 move 165 80 sleep 200 click sleep 400 move 190 120 sleep 50 move 195 150 sleep 400` (open File, hover New window then Open...). The old "New window" highlight keeps a 4px strip at x=146-150 and a 6px strip at x=357-363 (screen). The new highlight is also clipped narrower than in a full repaint. Deterministic: tour step 13/14 diff vs FULLFRAME oracle is exactly these two boxes (4x79 and 6x79 px) even with the engine bugs patched. evidence: shots/t_mz3.png, tour/diff_oracle_fixed/13_hov_open.png, 14_hov_menuabout.png
- actual: stale highlight fragments at the row ends; the hot row looks narrower than normal.
- expected: highlight moves cleanly between rows.
- rootCause: widgets/menu.go:798-803 invalidateRow invalidates rowBounds(i).Inset(-1) (plus 1px from app/window.go:312), but Classic.DrawMenuItem paints the hot fill in menuItemHighlightBounds (style/look.go:563-567) = row extended by PadL-2 (6px) on the left and PadR-2 (8px) on the right (PadL=8, PadR=10 in style/menu.go:58-59). Paint exceeds the invalidated box by 4px left / 6px right. MouseMove (menu.go:822-830) relies on invalidateRow.
- proposedFix: invalidate the full popup width for the row: `p.InvalidateRect(paintengine2d.XYWH(0, r.Min.Y-1, p.LocalBounds().Dx(), r.Dy()+2))` in invalidateRow (or expose the look's highlight bounds and invalidate those).
- files: widgets/menu.go (optionally style/look.go to export highlight bounds)
