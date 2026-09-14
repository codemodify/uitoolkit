# gallery-right-tabs findings (rig 13)

## BUG A: After a full-frame present (tab switch / any RequestLayout), the next partial present shows the frame from 2 swaps ago (old tab content reappears)
- app: engine (manifests in gallery tabs; every app on Wayland+GPU)
- severity: critical; partialOnly: true; confidence: high
- repro: run.sh 13 bin/gallery; in.sh 13 move 700 300 sleep 200 move 711 198 sleep 150 click sleep 400 (List tab shown OK, shot g1a/b2a) ; in.sh 13 move 900 600 sleep 400 -> shot g1b/b2b: tab strip says "List" but panel title "ScrollView", Row 01..15 of the Scroll tab, status bar "Ready." (stale), only the hover-damaged rects (tab strip, card) show new content. Next move (900 700) looks right again (alternating buffers). Reproduced 3x (g1b, b2b, prior r1b).
- oracle: UITK_PAINT_FULLFRAME=1 same sequence -> correct (f2a..f2c, prior f1b).
- rootCause: paintengine2d/gpu_linux.go:1508 recordFrameDamage(rects) records an EMPTY slot when rects is nil/empty (full-surface present: PresentRects(nil) from Window.frame when w.full, i.e. every RequestLayout -> fullInvalidate, app/window.go:363-373, TabView.Select widgets/tabs.go:322). Next frame: EGL buffer age = 2 (KWin, EGL_KHR_partial_update), damageForAge (gpu_linux.go:1517) unions this frame's rects with the empty slot, so the scissored FBO->window blit (gpu_linux.go:1445-1479) only refreshes the new frame's rects into a back buffer that still holds the pre-switch frame.
- verified fix: scratch copy of paintengine2d with recordFrameDamage appending XYWH(0,0,d.w,d.h) when len(rects)==0, gallery built with -modfile -> pA1..pA4 correct.
- proposedFix: in recordFrameDamage, record a full-surface rect (or a `full` flag per ring slot that makes damageForAge return nil => full blit) when rects is empty; also reset the ring on Resize/allocTarget.
- files: pe/gpu_linux.go
- evidence: shots/g1a.png g1b.png g1c.png b2a.png b2b.png f2a.png f2b.png pA1..pA4.png, c_g1.png c_b2.png c_f2.png c_pA.png

## BUG B: A frame painted while a wl_surface.frame callback is pending is never presented (stale hover/selection until some later unrelated repaint)
- app: platform (manifests in all gallery list/tree/table/tab views)
- severity: major; partialOnly: false (also happens with UITK_PAINT_FULLFRAME=1); confidence: high
- repro (use a build with BUG A fixed or FULLFRAME so A does not mask it): List tab; in.sh 13 move 700 392 sleep 400 ; in.sh 13 move 700 420 sleep 3 move 700 476 sleep 700 -> screen shows row 4 "style/look.go" hovered while pointer is on row 6 (bB2 / bF2). in.sh 13 move 700 504 sleep 2 move 700 560 sleep 700 -> row 7 "platform/x11_linux.go" stays hot although pointer left the list (bB3, bF3); moving 2px more (no model change) leaves it stuck (bB4). Pressing Tab (any repaint) in FULLFRAME mode shows the correct un-hovered list (bF4) => the FBO had the right frame, only the present was dropped.
- rootCause: platform/wayland_linux.go:1841-1843 `if s.framePending && !s.frameOverdue() { return nil }` drops the present of a frame the app already painted into the GPU FBO; app/window.go:908-911 then resets w.dirty; paintengine2d GPUDevice.SetPresentDamage (pe/gpu_linux.go:1382-1392) overwrites the un-presented rects with the next frame's; uitkWlFrameDone (wayland_linux.go:1923-1928) only clears framePending and schedules nothing, so the skipped frame reaches the screen only if a later present happens to cover the same pixels.
- proposedFix: remember that a present was deferred and its damage (e.g. s.deferredDamage), and when the frame callback fires push an EventExpose (or call presentGPU) so the app re-presents the union; make GPUDevice.SetPresentDamage merge rects while presentSet is still pending (never presented) instead of replacing them (and treat nil as full). Alternatively have Present return a 'deferred' result so Window.frame keeps w.dirty and the run loop repaints on the frame-done wake.
- files: platform/wayland_linux.go, app/window.go, pe/gpu_linux.go
- evidence: shots/bB1..bB4.png c_bB.png, bF2..bF4.png c_bF.png

## BUG C: Focus ring of ScrollView / ListView / TreeView / TableView / CardList is never visible (drawn outside the widget's own clip)
- app: gallery (all apps using these views); severity: major (a11y: no visible keyboard focus); partialOnly: false; confidence: high
- repro: List tab; move 1200 600 (shot lf0); move 700 364 click (focus+select LICENSE); move 1200 600 (shot lf1): diff of region 600,280-1140,540 = only the two row bands + label, list border untouched. Same with UITK_PAINT_FULLFRAME=1 (lf2/lf3) and UITK_SCENE=0 (lf4/lf5). Scroll tab: click track (focus), key Home => region pixel-identical to never-focused t0 (k1 vs t0 'identical').
- rootCause: widgets/scroll.go:169, list.go:144, tree.go:239, table.go:497, card.go:185 call lk.DrawFocusRing(ctx, b.Inset(-2)) i.e. 2px OUTSIDE LocalBounds, but widget/paint.go:27 (paintNode) and widget/scene.go:269 (recordNode) ClipRect(local) before c.Paint. Classic3D ring = DrawRect(b.Inset(1), stroke 1) => spans -1.5..-0.5 px => fully clipped (other bevels likewise: style/look.go:420-446).
- proposedFix: draw the ring inside: lk.DrawFocusRing(ctx, b) or b.Inset(1) (as fields do), or have the parent paint it / let paintNode expand the clip by the focus-ring margin for focused widgets.
- files: widgets/scroll.go widgets/list.go widgets/tree.go widgets/table.go widgets/card.go (or widget/paint.go widget/scene.go)
- evidence: shots/lf0.png lf1.png c_lf.png lf2..lf5.png, k1.png t0.png

## BUG D: CardList right-aligned meta text ("now") is cut off under the overflow scrollbar ("nov")
- app: gallery (also mail message list, same widget); severity: minor (cosmetic); partialOnly: false; confidence: high
- repro: List tab; look at Cards list right edge (x~1090, y~578): meta "now" renders as "nov" with the w under the scrollbar (c_cardnow.png from lf1.png; also g1a/c_prior). Hover/selected card fills also run under the bar.
- rootCause: widgets/card.go:159/170/178 paint every card with the full widget width b.Dx(); paintCard (card.go:211-233) right-aligns Meta at inner.Max.X-metaW with inner=b.Inset(RowPad+6), while paintOverflowBar later draws the bar on top at x in [W-bar-gap, W-gap] = [W-18, W-2] (scrollbar.go:34, bar=Metrics().Scroll=16 in Classic95 Dark). Nothing reserves the bar's width when the thumb is visible.
- proposedFix: compute rowW := b.Dx(); if thumb visible { rowW -= bar+gap } and pass it to paintCard/recordScrollingRows (and the row sig); same for ListView DrawListRow label box (list.go:108/124/132) and Tree/Table rows.
- files: widgets/card.go (widgets/list.go, widgets/tree.go, widgets/table.go for consistency)
- evidence: shots/c_cardnow.png, lf1.png

## BUG E: Row hover does not follow the pointer after wheel scrolling (ListView, CardList; check Tree/Table)
- app: gallery; severity: minor; partialOnly: false; confidence: high
- repro: List tab; move 700 392 (row 3 'widget/base.go' hot) ; wheel 1 -> highlight scrolls up with the content (now at y~333) while pointer sits on 'examples/gallery/main.go'; wheel 1 again -> no row hot at all under the pointer (wh1..wh3 clean build, wh4/wh5 real build FULLFRAME). Cards: move 700 600 (card 0 hot) wheel 1 -> card 0 highlight scrolled to top, pointer over 'go.mod' card not hot (wh6).
- rootCause: ListView.MouseWheel (widgets/list.go:233-245) and CardList.MouseWheel (widgets/card.go:383-395) change OffsetY but never recompute hovered from e.Pos; Window.bubbleWheel (app/window.go:573-582) sends no synthetic motion after a handled wheel, so hovered keeps the old row index until the pointer physically moves.
- proposedFix: after the offset changes, set hovered = indexAt(e.Pos.Y) (invalidate old/new rows) in each MouseWheel (list/card/tree/table), or generically re-dispatch w.mouseMove(ev.Pos) from bubbleWheel after a consumed wheel (also fixes ScrollView children).
- files: widgets/list.go widgets/card.go widgets/tree.go widgets/table.go (or app/window.go)
- evidence: shots/c_wh.png (wh1..wh3), c_wh2.png (wh4..wh6)
