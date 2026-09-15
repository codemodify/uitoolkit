# findings gallery-x11-hidpi

## 1. Popup menu hover highlight leaves stale strips at both row ends (partial redraw)
- app: gallery (all apps using PopupMenu/menubar/combo/context menus); backend-independent (Wayland + X11)
- severity: minor (visible glitch on every menu hover change)
- partialOnly: true (gone with UITK_PAINT_FULLFRAME=1)
- repro: launch gallery; in.sh 15 move 165 80 sleep 100 click sleep 400 move 200 150 sleep 300 (hover "Open...") ; then move 200 200 sleep 300 (hover "About"). Zoom right edge x 280-380 y 120-240, left edge x 140-200.
- actual: a ~4-6px blue strip of the old "Open..." highlight stays at the left and right ends of the row; the new "About" highlight is ~4-7px narrower on both sides than in a full repaint (clipped by the damage rect).
- expected: old highlight fully erased; new highlight spans full width as in a full-frame paint.
- rootCause: widgets/menu.go:798-803 invalidateRow invalidates rowBounds(i).Inset(-1) = [PadL-1, PadL+innerWidth+1], but style/look.go:563-567 menuItemHighlightBounds paints the highlight from item.Min.X-PadL+2 to item.Max.X+PadR-2 (PadL=8, PadR=10 x scale: style/menu.go:58-59), i.e. 5px (left) and 7px (right) outside the invalidated rect. Partial frames clip to the damage, so those bands are never repainted.
- proposedFix: invalidate the same rect the look paints: in invalidateRow use r.Min.X-ch.PadL .. r.Max.X+ch.PadR (or a LookAndFeel.MenuItemPaintBounds(row) exposed from style so the damage matches menuItemHighlightBounds + AA margin), then Inset(-1).
- files: widgets/menu.go, style/look.go (optionally expose highlight bounds)
- evidence: rig/15/shots/zc_menu.png (pairs: wayland, x11, x11 FULLFRAME), rig/15/shots/zc_menu_left.png
- confidence: high

## 2. X11: every mouse-wheel notch scrolls twice (ButtonRelease of buttons 4-7 also emits EventScroll)
- app: all (X11 backend); gallery ScrollView/List/Table/menus etc.
- severity: major (X11 wheel scrolling is 2x too fast, 96px/notch vs intended 48; 1.6x Wayland's 60)
- partialOnly: false
- repro: UITK_BACKEND=x11 run.sh 15 bin/gallery; in.sh 15 move 870 500 sleep 200 wheel -1 sleep 400. Raw log (scratch build with X event logging): Xwayland sends type=4,5,4,5 (press/release pairs of button 5) and the app receives FOUR EventScroll{0,48}. Raw tracer (platform events only) shows the same: wheel 1 -> 4 scroll events, wheel -1 -> 2 events, always an even count. Visible: 1 notch on X11 scrolls rows 01->04 vs 01->03 on Wayland; 3 notches rows 01->12 vs 01->06.
- actual: each X wheel click (ButtonPress + ButtonRelease of button 4/5/6/7) produces two EventScroll of 48px.
- expected: one scroll step per wheel click (the core-protocol wheel emulation always sends press+release pairs; toolkits act on the press only).
- rootCause: platform/x11_linux.go:1418-1436 (x11Surface.translate): the `case C.ButtonPress, C.ButtonRelease:` branch converts buttons 4-7 into EventScroll without checking the event type, so the release half of the pair scrolls again.
- proposedFix: in that branch, `if btn >= 4 && btn <= 7 { if C.ui_event_type(xe) == C.ButtonRelease { return nil }; ...emit scroll... }`. Add a unit test feeding a synthetic press+release of button 5.
- files: platform/x11_linux.go (+ platform/x11_linux_test.go)
- evidence: rig/15/shots/cd_scroll.png (wayland 1 notch, x11 1 notch, wayland +2, x11 +2), app.log excerpts above
- confidence: high
