# Findings — rig 14, area gallery-chrome-menus

## BUG 1: After any full-frame present, the next partial present on Wayland/EGL shows a 2-frame-old buffer (menus/popups vanish, ghosts reappear)
- app: engine (visible in gallery MenuBar first)
- severity: critical
- partialOnly: true (never with UITK_PAINT_FULLFRAME=1)
- repro: rig 14 (Wayland, GPU). `run.sh 14 gallery; in.sh 14 move 165 80 sleep 300 click sleep 1000; shot` -> File title highlighted but the dropdown is NOT drawn (pixel 300,200 = 41,41,41 instead of 66,66,66). ~70% of runs (8/8 in one loop, 1 in 2 in another). FULLFRAME: dropdown always drawn.
- actual: popup frame is presented as a full blit, then the release frame (partial, rect 0,0,1000,31) gets EGL buffer age 2 and only that strip is blitted into a back buffer that predates the popup -> the popup disappears from the screen (only the menubar strip is current).
- DETERMINISTIC repro (5/5): `in.sh 14 move 519 153 sleep 300 click sleep 600 move 808 487 sleep 300 click sleep 400 move 366 331 sleep 400` (toolbar About -> dialog -> OK -> hover Secondary) => the closed About dialog + dimmer stay on screen with an undimmed hot 'Secondary' button punched through (shots/abc_1.png). FULLFRAME: dialog gone (abcF_1.png). Debug log: dismiss = `FRAME full=true -> PE present rects=0 age=2`, next hover = `rects=[170 248 283 285] partialBlit=true age=2 blit=1`.
- evidence: debug log (instrumented scratch copy) rig/14/age_2.log: `FRAME full=true popup=true -> PE present rects=0 (full)` then `FRAME rects=[0 0 1000 31] -> PE present partialBlit=true age=2 blit=1` (only the strip).
- rootCause: paintengine2d gpu_linux.go recordFrameDamage (~l.1509) pushes `rects...` into the age ring; for a full present rects is nil so the slot is EMPTY ("nothing changed"). damageForAge (~l.1517) then unions nothing for that frame, so a back buffer of age>=2 never receives the full frame's content. Every fullInvalidate() in app/window.go (SetPopup, DismissPopup, SetOverlay, RequestLayout, resize) produces such a full present.
- proposedFix: in recordFrameDamage store a full-surface rect (XYWH(0,0,d.w,d.h)) when len(rects)==0 (or keep a per-slot `full` flag and make damageForAge return nil -> partialBlit=false when any intervening slot is full). Add a regression test: full present, then partial present with age 2 must blit full.
- VERIFIED FIX: scratch copy (rig/14/src, bin/gallery_fix1) with recordFrameDamage storing XYWH(0,0,d.w,d.h) for an empty rect list: About-ghost repro 3/3 clean, File-menu repro 5/5 shows the dropdown.
- files: pe/gpu_linux.go (recordFrameDamage, damageForAge, PresentRects)
- confidence: high

## BUG 2: ToolBar paints EVERY tool button hovered while the pointer is anywhere over the bar
- app: gallery (all apps with a ToolBar: mail, notes, files)
- severity: major
- partialOnly: false (identical with UITK_PAINT_FULLFRAME=1)
- repro: `run.sh 14 gallery; in.sh 14 move 219 153 sleep 700; shot` (or move 800 153 = empty strip). All 6 icon buttons + About + Snap paint navy/hot. Evidence shots/c_tb.png (rows: idle, partial hover, FULLFRAME hover, FULLFRAME hover on empty strip).
- actual: every item hot; expected only the item under the pointer.
- rootCause: widgets/toolbar.go:234 `st := t.State()` seeds each item's ControlState from the whole bar's Base state, which carries StateHovered whenever the pointer is inside the bar (Base.MouseEnter). Only StateFocused is masked (l.235-237). Same pattern as the TabBar bug fixed in 0719408.
- proposedFix: `st := t.State() &^ (style.StateHovered | style.StatePressed)` before OR-ing per-item hover/press (mirror tabs.go fix); add a chrome_test like TestTabBarHoverOnlyUnderPointer for ToolBar.
- files: widgets/toolbar.go, widgets/chrome_test.go
- confidence: high

## BUG 3: Frames throttled by the Wayland frame callback are dropped with their damage -> stale hover/pressed pixels stay on screen
- app: platform (seen on gallery MenuBar)
- severity: major
- partialOnly: true
- repro: `run.sh 14 gallery; in.sh 14 move 165 80 sleep 200 move 209 80 sleep 5 move 258 80 sleep 5 move 400 80 sleep 5 move 400 400 sleep 600; shot` -> "Edit" (or "View") title stays highlighted navy although the pointer is at 400,400 (4 of 9 runs; pixel 200,72 = 0,0,128). Evidence shots/c_skip.png, logs rig/14/skip_2.log.
- actual (debug log): `FRAME rects=[45 0 149 32] -> PRESENT SKIP framePending`, `... SKIP`, `... SKIP`, then `FRAME DONE` and no further frame: the damage of the three skipped frames never reaches the window.
- rootCause: platform/wayland_linux.go:1841-1843 (wlSurface.Present GPU path) returns nil when `s.framePending && !s.frameOverdue()` without keeping the rects; app/window.go:908-911 then resets w.dirty as if presented; and paintengine2d GPUDevice.SetPresentDamage (pe/gpu_linux.go ~1381) overwrites the pending unpresented damage on the next frame. The skipped frame is also never pushed into the buffer-age ring. Nothing re-triggers a paint when the frame callback arrives (uitkWlFrameDone only clears the flag).
- proposedFix: keep owed damage across a skipped present: in wlSurface accumulate skipped rects (nil => full) and, in uitkWlFrameDone (or at the next Present), queue an EventExpose for the union so the app repaints+presents it; or return a sentinel error (ErrPresentDeferred) so Window.frame keeps w.dirty/w.full and retries after the callback. In GPUDevice.SetPresentDamage merge with un-presented presentRects instead of replacing (nil/full wins).
- files: platform/wayland_linux.go, app/window.go, pe/gpu_linux.go
- confidence: high

## BUG 4: Opening a menu (click or Alt+mnemonic) steals keyboard focus from the text field and never gives it back
- app: gallery (any app with a MenuBar)
- severity: major
- partialOnly: false
- repro: FULLFRAME. click Name field (400,585), End, `x` -> "Ada Lovelacex" (works). Then click Edit (209,80), click Copy (240,121), type `x` -> nothing typed (shots/c_f6.png). Same with Alt+F, Esc, x, Esc, x (shots/c_f4.png: text unchanged).
- actual: after the menu closes the MenuBar keeps focus (no visible ring, m.focus=-1); keystrokes go nowhere. Expected (Qt/GTK/Win32): focus returns to the widget that had it before the menu was activated.
- rootCause: widgets/menu.go:252 (MousePress) and :318 (HandleAlt) call m.RequestFocus(); Open() :360 sets pop.RestoreFocusTo(m), so widget.RestoreFocus on dismissal (menu.go:466-468 -> widget/focus.go RestoreFocus) hands focus to the bar itself; nothing records the previous focus owner. OnPick (:334-341) / OnDismiss (:343-354) / Close (:374-380) also clear m.focus/m.keyNav so the bar holds invisible focus.
- proposedFix: in MenuBar remember `prev := widget.FocusOwner(m)` when the bar first takes focus (if prev != m) and restore it in OnPick/OnDismiss/Close (RestoreFocusTo(prev) for the popup when not keyboard-walking the bar); or don't RequestFocus on mouse press at all (Win32 menus never take focus).
- files: widgets/menu.go
- confidence: high

## BUG 5: Left/Right arrows do nothing while a menubar dropdown is open (cannot walk File->Edit->View by keyboard)
- app: gallery (all MenuBars)
- severity: major
- partialOnly: false
- repro: FULLFRAME. pointer at 900,600; Alt+F (key+ 56 key 33 key- 56); Down; Right -> File menu stays open (shots/c_k1.png row 3); Left -> unchanged (c_k2.png row 1).
- expected: Right opens Edit, Left opens File/View (wrap) — MenuBar.KeyPress even implements this (menu.go:282-297) and TestMenuBarArrowsWalkWhileOpen tests it by calling MenuBar.KeyPress directly.
- rootCause: while a dropdown is open, focus is the PopupMenu; app/window.go:545-556 sends keys to widget.CascadeLeaf(w.popup) and, when it returns false, returns without bubbling (`if w.popup != nil { return }`). PopupMenu.KeyPress (menu.go:928-944) only handles Right for a submenu row and Left for a cascade child; otherwise Left/Right fall through to indexForKey and return false -> swallowed. MenuBar never sees the key.
- proposedFix: give PopupMenu an OnNavigate/OnLeftRight hook (or a `bar *MenuBar` back-pointer) that MenuBar.Open sets; PopupMenu.KeyPress calls it for Left (no parentMenu) and Right (item without submenu) so the bar opens the adjacent menu (keyboard mode, first item highlighted). Add an app-level test that dispatches KeyRight through Window.dispatch.
- files: widgets/menu.go, app/window.go (optional), app/chrome_test.go
- confidence: high

## BUG 6: Keyboard-opened menu shows no highlighted item and the first Down skips the first item
- app: gallery
- severity: minor
- partialOnly: false
- repro: FULLFRAME. Alt+F -> File opens with no row highlighted (c_k1.png row 1). Down -> "Open..." (2nd row) highlighted; "New window" can only be reached by Up/Down wrap (c_k1.png row 2).
- rootCause: NewPopupMenu sets focus=firstEnabled (menu.go:418) but keyNav=false, so Paint (menu.go:780) draws nothing; the first KeyDown sets keyNav=true and moveFocus(1) (menu.go:907-917, 1010-1025) advances from the invisible focus 0 to 1. MenuBar.Open/HandleAlt never tell the popup it was opened from the keyboard.
- proposedFix: when the bar is in keyNav (HandleAlt, KeyDown/Return on the bar, arrow switching) set pop.keyNav=true after creating it so row 0 paints highlighted; or start mouse-opened popups at focus=-1 and make moveFocus(+1) from -1 land on firstEnabled.
- files: widgets/menu.go
- confidence: high

## BUG 7: Menu accelerators shown in the menus (Ctrl+N, Ctrl+O, F1, Ctrl+Q) do nothing
- app: gallery
- severity: minor
- partialOnly: false
- repro: FULLFRAME, pointer idle over the window: F1 -> no About dialog; Ctrl+O -> no file dialog (shots/sc_f1.png == sc_ctrlo.png, identical to idle); Ctrl+Q -> app still alive.
- rootCause: widgets/menu.go:33-36 ItemAccel/ItemIconAccel store Shortcut for display only; nothing parses or dispatches it. app/window.go:530-558 KeyDown handles Tab, Escape, Alt+mnemonic (handleAlt) and the focus bubble only, and with no focused widget (startup) bubbleKey drops the key entirely (keyTarget nil).
- proposedFix: add an accelerator walk next to handleAlt (e.g. MenuBar.HandleAccel(key, mods) parsing "Ctrl+Q"/"F1" once into key+mods and invoking the enabled item's OnClick), called from Window.dispatch before bubbleKey even when focus is nil; or stop rendering shortcuts that aren't wired.
- files: widgets/menu.go, app/window.go, internal/demo/gallery.go
- confidence: high
