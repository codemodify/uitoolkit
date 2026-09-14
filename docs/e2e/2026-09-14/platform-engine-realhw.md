# Findings — platform-engine-realhw (instance 21)

## 1. TestWaylandSurfacePresent asserts buffer width == logical width (fails at any scale > 1)
- app: platform; severity: minor; confidence: high; partialOnly: false
- repro: nested KWin with isolated kwinoutputconfig.json scale 2 (or 1.25/1.75 = user's real setting):
  `WAYLAND_DISPLAY=uitk-e2e-21 DISPLAY=:N go test ./platform -run TestWaylandSurfacePresent -v`
  scale 1 PASS; scale 2 FAIL "buffer &{Width:320 Height:200 ...}"; 1.75 FAIL Width:280 Height:175; 1.25 FAIL Width:200.
- actual: platform/wayland_linux_test.go:33 `s.Buffer().Width != 160` fails; the "all zeros" Pix is just %+v of the never-painted buffer (the check runs before ctx.Clear) — nothing is "read back".
- expected: Buffer() is device pixels by contract (bufferWH = ceil(logical*scale), wayland_linux.go:1758-1782; GPU path returns gpu.Image() sized the same); test should compare with ceil(160*s.Scale()) or s.Size().
- rootCause: test bug — wayland_linux_test.go:33 hard-codes scale 1. Not a present bug.
- fix: `wantW := int(math.Ceil(float64(160 * s.Scale()))); if b := s.Buffer(); b == nil || b.Width != wantW {...}`
- files: platform/wayland_linux_test.go

## 2. Wayland surface is created at the integer wl_output scale although the fractional preferred_scale already arrived → first frame dropped, spurious EventResize; TestWaylandPresentOpaqueColor fails at fractional scale ("no present slot")
- app: platform; severity: minor; confidence: high; partialOnly: false
- repro: scale 1.25 or 1.75 (user's real 1.75): `go test ./platform -run TestWaylandPresentOpaqueColor -v` → "wayland_linux_test.go:170: no present slot". Passes at 1 and integer 2.
- actual: NewSurface (wayland_linux.go:967-972) allocates s.img = logical*int(c.outScale) — KWin advertises wl_output.scale = ceil(1.75)=2 — so a 64x48 window gets a 128x96 pixmap. During the NewSurface roundtrips (1003-1006) wp_fractional_scale_v1.preferred_scale arrives (uitkWlFracScale 2936: s.frac=1.75), so Scale()=1.75 but Buffer() is still 2x. The caller paints; first Present (1797-1817) sees bufferWH()=112x84 != 128x96, reallocates, queues EventResize, sets blank and returns without committing (imgPainted false) → the painted frame is discarded, no shm slot ever exists → test fails. In apps: first frame always thrown away + an extra resize/relayout at startup (and the app laid out its first frame against a buffer/scale mismatch: 128px/1.75 = 73 logical instead of 64).
- expected: the pixmap handed out after NewSurface matches bufferWH() at the preferred fractional scale.
- rootCause: wayland_linux.go:967-972 sizes from the integer output scale before the fractional scale is known and never re-sizes after the roundtrip; uitkWlFracScale (2936-2945) only records s.frac.
- fix: after the roundtrips in NewSurface (before tryBindGPU), `if bw, bh := s.bufferWH(); s.img.Width != bw || s.img.Height != bh { s.img = paintengine2d.NewImage(bw, bh) }`; optionally have uitkWlFracScale push EventResize when s.frac changes on a live surface so the repaint does not depend on Present's lazy realloc. Test can also Poll + repaint on EventResize.
- files: platform/wayland_linux.go, platform/wayland_linux_test.go

## 3. Pointer leaving the window is never reported: hover stays stuck, and a pending tooltip pops up after the pointer is gone (Wayland and X11)
- app: platform (all apps); severity: major; confidence: high; partialOnly: false (same with UITK_PAINT_FULLFRAME=1)
- repro (1.75, gallery Wayland): in.sh 21 move 1000 48 (KWin title bar); shot base; move 224 312 (Primary action) sleep 150; move 1000 48 sleep 700; shot → Primary action still painted hovered (lighter blue) — shots s1_base/s1_after, ff1_* (fullframe), x1/x2_* (UITK_BACKEND=x11, same result).
  Tooltip: move 241 149 (3rd toolbar tool) sleep 150; move 1000 48 sleep 1200 → "Save project" tooltip appears while the pointer is on the title bar (shots tt_base/tt_after).
- expected: like Qt/GTK/Win32, leaving the window clears hover (MouseExit), cancels the tooltip timer, hides a shown tooltip.
- rootCause: there is no leave event kind (platform/event.go:113-129). Wayland: uitkWlPtrLeave (platform/wayland_linux.go:2531-2537) only zeroes c.ptrSurf and pushes nothing. X11: the window event mask (platform/x11_linux.go:79-80) has no LeaveWindowMask/EnterWindowMask, so LeaveNotify is never received. app/window.go only clears hover on EventFocusOut (476-485), and tickTips (675-696) keeps counting on w.tipHover, so the tip is shown after the pointer left.
- fix: add platform.EventMouseLeave; Wayland: in uitkWlPtrLeave push it to wlSurfaces[c.ptrSurf] before zeroing; X11: add LeaveWindowMask|EnterWindowMask and translate LeaveNotify (mode NotifyNormal, skip NotifyGrab/Ungrab while a button is held); app.Window.dispatch: on EventMouseLeave when w.capture==nil → w.hover.MouseExit(); w.hover=nil; w.dismissTooltip(); w.tipHover=nil; w.lastTip="".
- files: platform/event.go, platform/wayland_linux.go, platform/x11_linux.go, app/window.go (+ offscreen/others to emit it)

## 4. ToolBar paints EVERY tool hot/pressed while the pointer is over any one tool (same pattern as fixed TabBar bug)
- app: gallery (any app with a ToolBar: mail, files, notes...); severity: major; confidence: high; partialOnly: false
- repro: gallery @1.75 Wayland, in.sh 21 move 241 149 sleep 250; shot → all 8 toolbar items (6 icons, About, Snap) painted with the blue hover fill (shots tb_hover.png; tb_hover_ff.png with UITK_PAINT_FULLFRAME=1 identical).
- expected: only the tool under the pointer is hot; pressing one tool presses only that tool.
- rootCause: widgets/toolbar.go:234 `st := t.State()` seeds each item with the whole bar's ControlState (StateHovered from Base.MouseEnter, StatePressed from the bar press); only StateFocused is masked (235-237). Per-item hover/press (238-243) is then irrelevant.
- fix: `st := t.State() &^ (style.StateHovered | style.StatePressed | style.StateFocused)` then add back per item (hover==i, press==i, focus==i&&keyNav) — same as commit 0719408 did for TabBar.Paint.
- files: widgets/toolbar.go
