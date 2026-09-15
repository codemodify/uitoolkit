# settings-app findings (instance 18)

## 1. Theme switch (any full repaint) followed by a small partial frame shows the OLD frame everywhere else (GPU Wayland buffer-age ring drops full frames)
- app: settings (affects every app on the default GPU Wayland path) | severity: critical | confidence: high | partialOnly: true
- repro: `run.sh 18 uitksettings` (default UITK_PAINT=auto -> EGL); `in.sh 18 move 460 334 sleep 150 click sleep 700 move 1000 790 sleep 300`; shot.
  Reproduced 2/2 (bb_p1, bb_p2). UITK_PAINT_FULLFRAME=1: correct (bb_f1, bb_f2). UITK_PAINT=cpu: correct (bb_cpu1). UITK_PAINT=gpu: stale (bb_gpu1). X11 backend: correct.
- actual: after clicking "Motif" and then moving the pointer onto the Apply row, the presented frame shows the Motif palette ONLY in the Apply row band (y 747-798); the rest of the window (lists, preview, status bar "dark · round · classic · medium", selection on both Classic 95 Dark and Motif) is the previous Classic-95-Dark frame. Pixel (300,400) = #292929 (old) vs #aeb2c3 in full-frame.
- expected: the whole window shows the new theme.
- rootCause: paintengine2d/gpu_linux.go:1509-1514 `recordFrameDamage(rects)` pushes the frame's rect list onto the buffer-age ring; a full-surface present (Window.frame -> DrawScene -> SetPresentDamage(nil) -> PresentRects(empty)) records an EMPTY list, i.e. "nothing changed". On the next partial present PresentRects (gpu_linux.go:1440-1452) queries EGL_BUFFER_AGE (EGL_KHR_partial_update + EGL_EXT_buffer_age are advertised by Mesa on the nested KWin), gets age 2-3 and damageForAge (1516-1525) unions only the current rects with the ring entries -> the full frame contributes nothing, so only the hover/damage rect is blitted into a back buffer that still holds the frame from 2-3 swaps ago. TestPresentDamageAgeRing (gpu_damage_linux_test.go:11) never covers a full frame in the ring.
- proposedFix: in recordFrameDamage, when len(rects)==0 (full present) record `[]Rect{XYWH(0,0,float32(d.w),float32(d.h))}` (or keep a per-slot `full bool` and make damageForAge return nil => partialBlit=false when any of the age-1 frames was full). Also clear the ring on Resize/allocTarget. Add a test: full present then partial present with age 2 must blit the full surface.
- files: pe/gpu_linux.go, pe/gpu_damage_linux_test.go
- evidence: shots/kb01.png, bb_p1.png, bb_p2.png, bb_f1.png, bb_cpu1.png, bb_gpu1.png

## 2. NeXT theme: every list / text field / combo text is invisible (white text on white field)
- app: settings (all apps) | severity: major | confidence: high | partialOnly: false
- repro: settings, click "NeXT" (row y=418). Also FULLFRAME (f_t05) identical.
- actual: theme list, nav list ("About"), icon list ("Sharp"), text field "Ada Lovelace", combo "Classic look" render as blank white boxes; only the selected (black) row is readable. contrast Text/Field = 1.00.
- expected: readable dark text on the white fields (NeXTSTEP: black on white) or a dark field.
- rootCause: style/packs.go:231-266 packNext sets Text #ffffff (dark-chrome family) AND Field #ffffff; the Palette has one Text color used on both chrome and fields: style/look.go:468 DrawListRow fg=palette.Text, DrawTextField font.Draw(..., p.Text) (look.go:393), DrawComboBox col := p.Text (look.go:1006). style/chrome_test.go:126 TestNeXTChromeContrast even enforces "dark chrome / light content" without checking the text on that content.
- proposedFix: either give NeXT a dark field (e.g. #2a2a2a) or add a FieldText palette token (default = Text) used by DrawListRow/DrawTextField/DrawComboBox/table/tree cells and set it to #000000 for NeXT; add a pack-wide test asserting WCAG >= 4.5 for Text on Field/Background.
- files: style/packs.go, style/palette.go, style/look.go, style/tokens.go, style/chrome_test.go
- evidence: shots/p_t05.png, f_t05.png

## 3. NeXT Night: primary buttons unreadable (white on #d0d0d0; on hover white on white)
- app: settings (all apps) | severity: major | confidence: high | partialOnly: false
- repro: FULLFRAME=1 settings, click "NeXT Night" (y=446), look at "Primary action"/"Apply"; hover "Primary action" (793,462).
- actual: label #ffffff on accent #d0d0d0 (1.54:1); hovered: fill AccentHover #ffffff -> label disappears (1.00:1). Same for Apply and the checked checkbox glyph.
- expected: dark label on the light accent (or a dark accent).
- rootCause: style/packs.go:286-289 packNextNight TextOnAccent #ffffff with Accent #d0d0d0 / AccentHover #ffffff / AccentPress #a0a0a0; style/chrome.go:61-65 and :185-189 always use TextOnAccent on Accent/AccentHover.
- proposedFix: set NeXT Night TextOnAccent to #000000 (or darken Accent to ~#5a5a5a); add a pack test TextOnAccent vs Accent/AccentHover/AccentPress >= 3:1.
- files: style/packs.go, style/theme_test.go (new contrast test)
- evidence: shots/p_t06.png, hov_nn.png, c_nn_btn.png, c_hov_nn.png

## 4. Fusion: selected list rows (and hovered menu items/titles) draw WHITE text on a pale-blue fill
- app: settings (all apps) | severity: major | confidence: high | partialOnly: false
- repro: settings, click "Fusion" (y=586). FULLFRAME identical (f_t11).
- actual: selected rows "Fusion", nav "Appearance", icons "Classic" are #ffffff on #cce0f4 (1.35:1) - practically invisible. Menu hover would be #ffffff on #e8f2fc (1.13:1).
- expected: dark text on the light Fusion selection (Qt Fusion uses a saturated #308cc6 highlight with white text, or dark text on light tint).
- rootCause: style/chrome.go:451-455 menuInvertText() returns true for EVERY BevelClassic3D pack regardless of how light the Selected/Hot fill is; style/look.go:473-475 DrawListRow then forces TextOnAccent for selected rows (also look.go:499-500 DrawMenuTitle and :592-593 DrawMenuItem). Fusion is BevelClassic3D with Selected.Fill = #4a90d9 @ 0.28 alpha and Hot.Fill #e8f2fc (style/packs.go:498-500).
- proposedFix: decide inversion from the composited fill luminance, e.g. `fg = readableOn(over(fill, p.Field), p.Text, p.TextOnAccent)` (pick the one with higher WCAG contrast) in DrawListRow/DrawMenuItem/DrawMenuTitle, and restrict menuInvertText to fills with luminance < ~0.35; or give Fusion an opaque #308cc6 selection.
- files: style/chrome.go, style/look.go, style/packs.go
- evidence: shots/p_t11.png, f_t11.png

## 5. Classic 95 Light / Motif / CDE Crimson: hovered (unselected) list rows are black text on navy/crimson
- app: settings (all apps) | severity: major | confidence: high | partialOnly: false
- repro: UITK_PAINT_FULLFRAME=1 settings; click "Classic 95 Light" (y=306); move to "CDE Charcoal" (460,362).
- actual: hovered row is #000000 on #000080 (1.31:1) - label unreadable while hovered (Motif 1.32:1, CDE Crimson #000 on #a03030 2.96:1). Selected rows are fine (inverted).
- expected: hovered row readable (white on navy, like the selected row / menu hover) or a light hover tint.
- rootCause: style/chrome.go:209-216 roleRow hover uses fill = t.Hot.Fill (navy #000080 for Classic95 Light, style/packs.go:108) but leaves fg = p.Text; style/look.go:470-475 DrawListRow only swaps to TextOnAccent when `selected`. Same pattern affects table/tree rows that use roleRow.
- proposedFix: same helper as #4: compute fg from the composited hover fill luminance (or apply `if l.menuInvertText() && (selected||hovered)` once #4's luminance-aware menuInvertText exists).
- files: style/look.go, style/chrome.go
- evidence: shots/hov_light.png (FULLFRAME), c_hov_light.png

## 6. Picking a theme below the fold (or with the keyboard) scrolls the theme list back to the top; the selected theme is no longer visible
- app: settings | severity: minor | confidence: high | partialOnly: false
- repro: settings; `in.sh 18 move 500 500 sleep 150 wheel 10 sleep 400 move 460 505 sleep 150 click sleep 700` (clicks "Fluent").
- actual: Fluent is applied but the list is back at offset 0 showing Classic 95 Dark..Breeze; no row is highlighted (fl01.png). For Breeze Night only a 1px sliver of the selection shows at the list bottom (f_t14.png). Same for the right-hand ScrollView (scroll position resets on every Corners/Icon/Export change) and keyboard focus is lost: after clicking a theme, Down/Up do nothing (kf01-kf03: staged stays "motif").
- expected: list keeps its scroll offset / keeps the selected row visible (Qt: setCurrentIndex + scrollTo).
- rootCause: internal/demo/settings.go:39-43 preview() rebuilds the whole tree with win.SetContent(buildSettings(...)) on every pick; pickerSection (settings.go:369-380) creates a fresh ListView (OffsetY=0) and only sets Selected - nothing scrolls it into view (ListView.ensureVisible is unexported, widgets/list.go:294).
- proposedFix: keep the widgets and restyle in place (a.SetLook already restyles the tree; update only labels/status/Apply enabled), or carry list OffsetY / ScrollView OffsetY / focused widget across rebuilds; at minimum export `ListView.EnsureVisible(i)` that is honoured on the first Arrange and call it in pickerSection.
- files: internal/demo/settings.go, widgets/list.go
- evidence: shots/fl01.png, f_t14.png

## 7. Theme list hover highlight sticks to the old row after wheel-scrolling (row no longer under the pointer)
- app: settings (ListView everywhere) | severity: minor | confidence: high | partialOnly: false
- repro: settings; `in.sh 18 move 500 500 sleep 150 wheel 10 sleep 500`; shot (FULLFRAME too: scf01.png).
- actual: the list scrolls 195px but the navy hover highlight stays on "Luna Night" (index 8, the row that WAS under the pointer), now at y~310, while the pointer is over "Fluent" at y=500. It stays until the mouse moves.
- expected: hover follows the row under the stationary pointer after a scroll (Qt/GTK re-evaluate hover on scroll).
- rootCause: widgets/list.go:233-245 ListView.MouseWheel changes OffsetY but never recomputes l.hovered (only MouseMove does, list.go:163-182); app/window.go:572-581 bubbleWheel does not re-run hover hit-testing (no synthetic move) after a handled wheel, so ScrollView children keep stale hover too.
- proposedFix: in ListView.MouseWheel set `h := l.indexAt(e.Pos.Y)` and invalidate old/new rows when it changes (same for Table/Tree); better, have Window.bubbleWheel call w.mouseMove(platform.Event{Pos: ev.Pos}) after a consumed wheel so hover is re-synced for every scroller.
- files: widgets/list.go, app/window.go
- evidence: shots/sc01.png, scf01.png, c_sc01.png, c_scf01.png

## 8. Hover state sticks when the pointer leaves the window (buttons/list rows stay highlighted)
- app: settings (all apps, Wayland and X11) | severity: major | confidence: high | partialOnly: false
- repro: UITK_PAINT_FULLFRAME=1 settings; `in.sh 18 move 926 462 sleep 300 move 927 463 sleep 300` (hover Secondary) then `move 1230 505 sleep 500` (desktop, outside the window). Same with a list row: hover "CDE Crimson" (461,391) then move to (1225,395).
- actual: Secondary button / CDE Crimson row remain painted hovered indefinitely after the pointer left the surface (lv02.png, lv03.png).
- expected: hover cleared on pointer leave (Qt: QEvent::Leave, GTK leave-notify).
- rootCause: platform/wayland_linux.go:2531-2537 uitkWlPtrLeave only zeroes c.ptrSurf and queues no event; platform/event.go:114-129 has no leave event kind; app/window.go:476-486 only drops w.hover on EventFocusOut (keyboard focus), which does not happen when the pointer merely leaves. X11: platform/x11_linux.go:79-80 event_mask lacks LeaveWindowMask/EnterWindowMask, so LeaveNotify never arrives either.
- proposedFix: add platform.EventMouseLeave; push it from uitkWlPtrLeave (for the surface in c.ptrSurf before zeroing) and from X11 LeaveNotify (add LeaveWindowMask); in Window.dispatch handle it: `if w.hover != nil { w.hover.MouseExit(); w.hover = nil }; w.dismissTooltip(); w.SetCursor(default)` (skip while w.capture != nil).
- files: platform/event.go, platform/wayland_linux.go, platform/x11_linux.go, app/window.go
- evidence: shots/lv01.png, lv02.png, lv03.png, c_lv.png, c_lv3.png
