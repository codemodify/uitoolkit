# End-to-end pass, 2026-09-14 (real GPU, KDE Plasma Wayland)

Twelve exploratory QA agents drove the gallery, Mail, Settings and the other
examples on real hardware through the nested-KWin rig in `tools/e2e/`, one area
each, and wrote every confirmed bug with repro, full-frame oracle result
(`partialOnly`), root cause and proposed fix. The run was stopped early (laptop
load), so coverage is partial; `notes-files-inspector` found nothing before the
stop. Screenshot paths (`<rig>/N/shots/…`) are local to that machine.

## What made the gallery "glitch out once you start clicking around"

1. **paintengine2d GPU buffer-age ring records full-frame presents as "no
   damage"** — the next partial present repairs too little and shows a
   2-frames-old buffer: closed dialogs reappear, menus vanish/reappear, theme
   switches revert. Found independently by 7 agents (`pe/gpu_linux.go`
   `recordFrameDamage` / `damageForAge` / `PresentRects`). Critical.
2. **Wayland: a frame painted while a `wl_surface.frame` callback is pending is
   dropped together with its damage** — stale hover/pressed pixels until the
   next unrelated repaint (`platform/wayland_linux.go`, `app/window.go`).
3. **Focused text field spins an IME feedback loop on KWin** (~3500 repaints/s,
   >100% CPU) — `platform/wayland_linux.go` text-input-v3 handling.
4. **No pointer-leave events** (Wayland and X11): hover sticks and tooltips pop
   up after the pointer left the window.
5. **Whole-widget state leaking into sub-items**: ToolBar paints every tool hot
   (TabBar had the same bug, fixed in 0719408).

Then: focus rings drawn outside widget bounds and clipped away, menu keyboard
navigation and accelerators, hover not following the pointer after wheel
scrolls, X11 double wheel steps, theme contrast failures (NeXT, Fusion,
Classic 95 Light, Motif/CDE rows), the Mail paperclip glyph tofu, no
multi-select in Mail, modal dialogs not modal for Alt+mnemonics.

One file per area; each `##` section is one bug.
