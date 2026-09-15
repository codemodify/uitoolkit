# findings — gallery-buttons-dialogs (instance 11)

## 1. [critical][engine] GPU partial present shows STALE frames (e.g. closed About dialog reappears) after any full-frame repaint
- app: engine (seen in gallery; affects every app on the GPU/EGL path)
- repro: gallery (default GPU path, Wayland). click About… (210,373); key Esc (1); move 900 700; key Tab (15) x4, screenshot after each Tab.
  Tab #2 and #4 show the dismissed About dialog + dimming scrim again everywhere except the button rows that changed.
  shots: 11/shots/a1..a6.png, stacked crop 11/shots/z_a.png. Prior run saw the same (11/shots/z_tab.png).
- actual: every other frame after a full repaint shows the frame from 2 swaps ago outside the current damage (alternating good/stale).
- expected: only the focus ring moves.
- oracle: UITK_PAINT_FULLFRAME=1 -> clean (11/shots/f1..f6, z_f.png). partialOnly=true.
- env probe (11/probe): GPUDevice partial=true (EGL_KHR_partial_update), preserve=false, buffer age=2..3. Mesa Intel ARL.
- rootCause: paintengine2d/gpu_linux.go PresentRects: a full present (rects empty; DrawSceneDamage(nil) -> SetPresentDamage(nil) -> presentRects len 0) calls
  d.recordFrameDamage(rects) (line ~1496) which stores an EMPTY list in the age ring. The next partial present into a back buffer of age N>=2 calls
  damageForAge (line ~1453/1517) which unions the ring entries — the full frame contributes nothing, so only the current small rects are blitted into
  a buffer that still holds the pre-full-frame image. (Full frames happen on every overlay/popup open/close, SetContent, layout, resize.)
- fix: in PresentRects record the whole surface when the present was full (if !partialBlit || len(rects)==0 { record [XYWH(0,0,w,h)] }),
  or keep a 'full' flag per ring slot and make damageForAge return nil/force full blit when any frame in the window was full. Add a test: record full, record
  partial, damageForAge(cur,3) must cover the surface.
- files: pe/gpu_linux.go, pe/gpu_damage_linux_test.go

## 2. [major][platform] Wayland shm/dmabuf present has no per-buffer damage history: stale frames with UITK_WAYLAND_PRESENT=dmabuf (latent on shm)
- app: platform
- repro: UITK_PAINT=cpu UITK_WAYLAND_PRESENT=dmabuf gallery; same sequence as #1 (About… click, Esc, move away, Tab x4).
  shots 11/shots/d1..d6 (z_d.png): d4/d6 show the closed About dialog again, d5 shows a stale focus ring left on Confirm… while focus is on Window.
  With UITK_PAINT=cpu and default shm (c1..c6) KWin happens to hide it (it copies only the damaged region of shm buffers into its texture).
- rootCause: platform/wayland_linux.go Present (~1854-1891): pickSlot() (1857, first free of 4 slots via wlstate.go:56 pickPresentSlot) then
  blitDirty(slot, dirty) (1875/1931) copies ONLY this frame's dirty rects from s.img into that slot. A slot last written N frames ago still holds that old
  frame outside the rects; the buffer handed to the compositor is not a complete frame. dmabuf is sampled directly -> stale regions visible.
- fix: track per-slot pending damage (e.g. wlSlot.stale []Rect / fullStale bool): on every present, append this frame's rects (or 'full') to every other
  slot; when a slot is written, copy dirty ∪ slot.stale (full copy if fullStale or the slot was just (re)created) and clear it. Keep wl_surface.damage = dirty.
- files: platform/wayland_linux.go, platform/wlstate.go

## 3. [major][platform+app] Pointer leaving the window never clears hover: button stays hot, tooltip pops up while pointer is outside
- app: platform (all apps)
- repro (FULLFRAME, so not a paint bug): move 366 331 (Secondary hot); move 40 400 (desktop) / move 700 48 (KWin title) / move 1200 20 -> Secondary stays hot
  forever (11/shots/lv1..lv4, z_lv.png). Tooltip variant: move 166 152, move 167 153 (toolbar New), within 200ms move 40 300 (desktop), wait 1.5s ->
  "New window" tooltip appears although the pointer is outside the window (11/shots/tipout.png, z_tipout.png).
- expected: wl_pointer.leave / LeaveNotify -> hovered widget gets MouseExit, tooltip timer cancelled, cursor reset (Qt QEvent::Leave, GTK leave-notify).
- rootCause: platform/wayland_linux.go:2531-2537 uitkWlPtrLeave only sets c.ptrSurf=0 and pushes no event; platform has no leave EventKind at all
  (platform/event.go EventKind list). X11: platform/x11_linux.go:79 event_mask lacks EnterWindowMask|LeaveWindowMask, so no LeaveNotify either.
  app/window.go dispatch therefore never runs hover.MouseExit()/tipHover=nil except on FocusOut (window.go:476-485), which a pointer leave does not cause.
- fix: add platform.EventMouseLeave; push it from uitkWlPtrLeave for the surface that had the pointer (before zeroing ptrSurf) and from X11 LeaveNotify
  (add LeaveWindowMask|EnterWindowMask; ignore NotifyInferior/grab-mode leaves); in Window.dispatch: if capture==nil { hover.MouseExit(); hover=nil };
  dismissTooltip(); tipHover=nil; lastTip="".
- files: platform/event.go, platform/wayland_linux.go, platform/x11_linux.go, app/window.go
- partialOnly: false

## 4. [major][widgets] ToolBar paints EVERY tool button hot when the pointer is over any one of them (same pattern as the fixed TabBar bug)
- app: gallery (all apps with a ToolBar: mail, notes, files...)
- repro: move 272 153 (Save icon) or move 519 153 (About text tool) -> all 8 tools get the accent-blue hot fill, indistinguishable from the checked
  'Snap' toggle; move 700 400 -> all normal. 11/shots/tb0..tb3.png, z_tb.png. Reproduced also in FULLFRAME.
- rootCause: widgets/toolbar.go:234 `st := t.State()` seeds each item from the bar's Base state; Base.State() (widget/base.go:289) sets StateHovered
  whenever the pointer is anywhere on the bar, and the loop only ORs per-item hover (238) - it never clears the inherited StateHovered (only
  StateFocused is masked at 235-236). Base.MouseEnter invalidates the whole bar, so every item repaints hot.
- fix: `st := t.State() &^ (style.StateHovered | style.StatePressed)` like widgets/menu.go:182-183 / widgets/tabs.go:83, then add per-item hover/press.
- files: widgets/toolbar.go (+ a test in widgets/toolbar_test.go)

## 5. [minor][app+widgets] Press on a button, drag off, release: button stays "pressed" while outside and stays HOT after release
- app: gallery (any Button)
- repro (ONE inj invocation — the rig's fake input releases held buttons when inj exits): 
  in.sh 11 move 366 331 sleep 200 down sleep 1000 move 366 360 sleep 1000 move 366 440 sleep 1000 up sleep 1000, shots in parallel
  (11/shots/dp0..dp4, z_dp.png; prior run z_sec.png). FULLFRAME on.
- actual: while dragged outside (dp1/dp2) Secondary keeps the navy pressed look (releasing there will NOT click); after release outside (dp3/dp4) it
  stays hot until the pointer moves again; Confirm… under the pointer never gets hover during the drag.
- expected (Qt QAbstractButton/Win32 BUTTON): button pops up while the pointer is outside and re-presses on re-entry; after release, hover goes to the
  widget actually under the pointer.
- rootCause: app/window.go:741-757 mouseMove pins t := w.capture so hover never changes during a drag (no MouseExit to the button), and
  app/window.go:723-739 mouseUp clears w.capture but never re-hit-tests/updates w.hover, so the button keeps hovered=true
  (widgets/button.go:56-61,74-82: MouseRelease clears pressed but not hovered; there is no MouseMove to track inside/outside while pressed).
- fix: factor the hover switch out of mouseMove into w.updateHover(pos) and call it at the end of mouseUp (after capture=nil). In Button add
  MouseMove: inside := LocalBounds().Contains(e.Pos); if b.pressed && inside != b.downInside { b.downInside = inside; Invalidate() } and paint
  StatePressed only when pressed && downInside (hovered likewise only when inside).
- files: app/window.go, widgets/button.go
- partialOnly: false

## 6. [major][app] Modal dialogs are not modal for Alt+mnemonics: Alt+F opens the main menu ABOVE the dialog and steals focus out of it
- app: gallery (all apps using widgets.Info/Confirm/Warn/ShowMessageBox/Overlay + a MenuBar)
- repro: FULLFRAME. click About… (210,373); key+ 56 key 33 key- 56 (Alt+F) -> File menu drops over the dimmed modal (11/shots/m1,m2, z_m.png);
  its items (New window, Open…, About, Quit) are all live. key 1 (Esc) closes the menu; key 28 (Enter) and key 57 (Space) now do NOTHING - the
  dialog's OK is dead (m3..m5, z_m2.png); only Tab brings focus back (m6). Baseline without Alt+F: Enter closes the dialog (m7 == m6).
- expected: while a modal overlay is up, mnemonics resolve only inside it (Qt/GTK: application-modal dialog blocks the parent window's menu bar).
- rootCause: app/window.go:540-544 dispatch calls handleAlt(w.root, ev.Key) unconditionally - it walks the CONTENT tree even when w.overlay != nil
  (unlike bubbleKey/keyTarget, window.go:640-657, which respect the overlay). MenuBar.HandleAlt (widgets/menu.go:310-321) then RequestFocus()es the
  menu bar behind the overlay and opens a popup on the popup layer (painted above the overlay, window.go:924-926). When the popup closes it
  restores focus to the menu bar (menu.go:360/467), which is outside the overlay, so widget.KeyTarget (widget/focus.go:146-151) drops every key.
- fix: in dispatch use `root := w.root; if w.overlay != nil { root = w.overlay }` (or skip Alt handling while an overlay is up);
  optionally also refuse ShowPopup from components not LiveUnder the overlay.
- files: app/window.go (+ app/focus_test.go regression test)
- partialOnly: false

## 7. [minor][widgets] Message boxes are light-dismissed by any click on the dimmer - a double-click on Confirm… silently answers "No"
- app: gallery (all MessageBox users)
- repro: FULLFRAME. click Secondary (status "Secondary clicked"); move 319 373; click sleep 80 click -> the Quit dialog flashes and closes, status
  "Cancelled" (11/shots/dc3,dc4, z_dc34.png). Same for About… (dc1: dialog never visible). Single click on the dimmer at 900,200 also dismisses (co1/co2).
- expected: Qt QMessageBox / GTK / Win32 modal message boxes ignore clicks outside the dialog (at most flash/beep); a question must be answered explicitly.
- rootCause: widgets/dialog.go:86-92 Overlay.MousePress dismisses on every press outside the card, and messagebox.go:111 maps that close to
  finish(cancelResult()) (ResultNo for Yes/No). The 2nd click of a double-click on the opener lands on the freshly shown dimmer.
- fix: give Overlay a LightDismiss bool (default false for NewMessageBox/Confirm/Warn/Info; keep true only for light popovers), and when false make
  MousePress outside the card return true without dismissing (optionally flash the card).
- files: widgets/dialog.go, widgets/messagebox.go
- partialOnly: false

## 8. [minor][widgets] Confirm dialog: initial focus (and Enter) go to "No" while "Yes" is drawn as the primary/default button
- repro: click Confirm… (319,373) -> "No" gets the focus ring, "Yes" the accent fill (11/shots/z_cf.png); key Enter -> status "Cancelled" (z_cf2.png).
- expected: the emphasized default button is the one Return activates (Qt moves default emphasis to the focused autoDefault button; GTK/Win32 focus the
  default button).
- rootCause: widgets/dialog.go:37-39 Overlay.Presented -> widget.FocusFirstIn(card) (widget/focus.go:56-73) focuses the FIRST focusable, and
  messagebox.go:121-134 orders [No, Yes(primary)] / [Cancel, OK(primary)].
- fix: MessageBox should focus its primary (or an explicit Default) button on present; or draw the primary emphasis on the focused button.
- files: widgets/messagebox.go, widgets/dialog.go

## 9. [major][style+widget] Button keyboard focus ring is drawn outside the button and clipped away - Tab focus is practically invisible
- app: gallery (all Buttons; same Inset(-2) ring pattern in list/tree/table/card/scroll/checkbox rings)
- repro: FULLFRAME. Tab through the Buttons card (a3..a6/f3..f6) or dismiss a dialog so focus returns to its opener (fr2). Pixel diff g0 vs fr2 (About…
  focused): exactly ONE row changes (y=355, x164-256, gray 77->117); no left/right/bottom ring at all (11/shots/z_fr2.png 5x zoom, z_fr.png).
- expected: a clearly visible ring (dark theme = BevelClassic3D: a 1px p.Text rectangle around the button).
- rootCause: style/look.go:257-259 DrawButton calls DrawFocusRing(ctx, b.Inset(-2)); for BevelClassic3D (style/look.go:433-434) the stroke is
  b.Inset(1) = button.Inset(-1), i.e. entirely OUTSIDE the button. widget/paint.go:26-28 (paintNode) and widget/scene.go:258 (RecordTree) clip every
  widget to ctx.ClipRect(local), so the ring is clipped off (only a fractional-edge antialias sliver survives). Even unclipped, Window.Invalidate
  pads damage by only 1px (app/window.go:312), so an outside ring would leave stale pixels when focus moves.
- fix: draw the button focus ring inside the bounds (e.g. DrawFocusRing(ctx, b.Inset(2)) / Qt-Fusion style inner focus rect), or give widgets a
  paint outset that both the clip in paintNode/RecordTree and Window.Invalidate honour.
- files: style/look.go (DrawButton and other DrawFocusRing(…Inset(-2)) callers), widget/paint.go, widget/scene.go, app/window.go
- partialOnly: false
