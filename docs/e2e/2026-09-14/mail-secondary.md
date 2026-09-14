# mail-secondary findings (rig 17)

## BUG 1: Popup menus (M menu, context menus) vanish right after opening / reappear after closing — buffer-age history ignores full-frame presents
- app: engine (seen in mail M menu, message/folder context menus, dialogs)
- severity: critical; partialOnly: true; confidence: high
- repro: run.sh 17 bin/mail; in.sh 17 move 24 63 sleep 200 click sleep 500; shot → M title highlighted but NO drop-down on screen. Move pointer onto the (invisible) menu rows (move 30 100) → the whole menu + cascade suddenly appears. With UITK_PAINT_FULLFRAME=1 the same click shows the menu immediately.
- evidence: shots/c_a02.png (partial: no menu), shots/c_f02.png (fullframe: menu), shots/c_d02.png (top: unpatched debug build; bottom: patched engine build shows menu)
- instrumented log (scratch copy): press => "PresentRects rects=0 (full) age=0"; release 19ms later => "PresentRects rects=1 blit=1 partial=true age=2": the age-2 back buffer predates the popup, only the menubar rect is re-blitted, so the popup disappears.
- rootCause: pe/gpu_linux.go recordFrameDamage (~l.1508) appends `rects...` — for a full-surface present rects is nil, so the ring slot is EMPTY; damageForAge (~l.1517) then unions "nothing" for that frame and PresentRects does a scissored partial blit that omits everything the full frame changed. uitoolkit app/window.go SetPopup/DismissPopup/SetOverlay/SetContent/RequestLayout all use fullInvalidate() => Present(nil) => exactly this pattern (full frame then partial frame).
- fix: in recordFrameDamage, when len(rects)==0 record a full-surface rect XYWH(0,0,d.w,d.h) (or store a per-slot `full` flag and make damageForAge return nil => partialBlit=false). Verified: with that change (env-gated in scratch copy) the menu stays visible (c_d02 bottom).
- files: pe/gpu_linux.go

## BUG 2: ToolBar paints EVERY tool button hot/pressed while the pointer is anywhere on the strip (same pattern as the fixed TabBar bug)
- app: mail (main window Fetch/Write bar, compose Send/Save/Attach bar, search bar); widget-level, affects every ToolBar
- severity: major; partialOnly: false; confidence: high
- repro: UITK_PAINT_FULLFRAME=1 run.sh 17 bin/mail; in.sh 17 move 150 63 (gap between Fetch and Write) → BOTH Fetch and Write paint the blue hot fill. Partial mode: move 205 63 (over Write) → both hot (the enter repaints the whole bar); `down` on Write → both hot/pressed; after clicking Write the compose window opens and both main-window buttons stay blue.
- evidence: shots/c_t.png (rows: FF gap-hover both hot; partial hover Write both hot; pressed both hot; moved away none), shots/w01.png (after Write click: Fetch+Write both blue)
- rootCause: widgets/toolbar.go:234 `st := t.State()` copies the whole bar's StateHovered (widget.Base.hovered, widget/base.go:289) into each item; only StateFocused is masked. MenuBar.Paint (menu.go:182-183) and the fixed TabBar (tabs.go:83) mask StateHovered|StatePressed first.
- fix: `st := t.State() &^ (style.StateHovered | style.StatePressed)` in ToolBar.Paint, then OR per-item hover/press as now.
- files: widgets/toolbar.go
