# Paint performance (v0.15.0)

v0.14.0–v0.14.6 tried to go faster with dirty-rect clipping, pixel
`Context.Scroll`, `DrawSceneDamage`, and partial GPU present. That was
a bad idea: black first frames, HiDPI scale blow-up, vacated list
holes, stuck menu highlights, and bold→thin glyphs on hover.

**v0.14.7 restored the v0.13.8 paint path.** **v0.15.0** is the safer
second pass: less work per **full** paint, idle coalescing, and
alloc cuts. Dirty semantics stay the v0.13.8 model — dirty marks
which widgets to **re-record**; `DrawScene` always replays the whole
graph; `Surface.Present(nil)` always presents the full buffer.

## Current model

1. Dirty marks which widgets to **re-record** into the retained scene.
2. A full present of an unchanged tree **keeps** recorded groups.
   Layout, look, and a new content tree still `Reset` the scene cache.
3. `DrawScene` always replays the **whole** graph.
4. `Surface.Present(nil)` always presents the **full** buffer.
5. paintengine2d **v0.9.0** — no `DrawSceneDamage` / `BakeGroup` wiring.

## Headless profile (CI, `CGO_ENABLED=0`)

```bash
CGO_ENABLED=0 go test -bench='BenchmarkMailFullPaint|BenchmarkGalleryFullPaint|BenchmarkListHover|BenchmarkListScroll' -benchmem ./internal/apptest
CGO_ENABLED=0 go test -bench=BenchmarkFontAdvanceCached -benchmem ./style
CGO_ENABLED=0 go test -bench=BenchmarkLookWatchSettled -benchmem ./app
```

Wayland dogfood (`UITK_SCENE=auto` mailclientui + pprof / perf) is
still the place to confirm HiDPI; these numbers are CPU `DrawScene`
of the offscreen pixmap.

Median of 3 runs on the agent host (Xeon, `CGO_ENABLED=0`):

| Bench | v0.14.7 | v0.15.0 |
| --- | --- | --- |
| Mail full paint (1280×800) | 63.8 ms · 1305 KB · 8555 allocs · 48 reused | 62.6 ms · 36 KB · 87 allocs · 1 reused (root attach) |
| Gallery full paint (1100×720) | 18.1 ms · 106 KB · 690 allocs · 0 reused | 18.1 ms · 35 KB · 84 allocs · 1 reused |
| List hover | 778 µs · 20 KB · 142 allocs | 777 µs · 18 KB · 113 allocs |
| List scroll | 1.23 ms · 70 KB · 508 allocs | 1.22 ms · 54 KB · 305 allocs |
| `Font.Advance` warm string | 671 ns · 760 B · 6 allocs | 165 ns · 0 B · 0 allocs |
| WatchLook settled file | 2740 ns · 1000 B · 6 allocs | 629 ns · 288 B · 2 allocs |

`DrawScene` raster of the full mailbox still dominates wall time.
The win is recording: v0.14.7 dropped the scene cache on every full
invalidate (only virtualized rows survived), so Mail rebuilt ~8k
nodes per present. v0.15.0 attaches the recorded root.

## What shipped

1. **Less work per full paint** — `fullInvalidate` no longer
   `SceneCache.Reset`. `frame` drops the cache only after Measure /
   Arrange (resize, `RequestLayout`, `SetLook`, `SetContent`). Hover
   still `Invalidate`s and does **not** `RequestLayout`. `Font`
   reuses `NullShaper` runs (cap 512) for Advance / InkWidth / Draw.
   Row keep-maps and scene-cache maps are pooled / `clear`d.
2. **Idle coalescing** — `Run` / `PumpOnce` drain every surface
   (up to 64 `Poll`s) then paint each window once. WatchLook is
   Stat-first again: idle polls size+mtime and `ReadFile`s only on
   a meta change (1s settle window for coarse overlay clocks).
3. **Alloc cuts** — no paintengine2d bump. `Recorder` has no `Reset`
   in v0.9.0, so recorders are not pooled. Present stays `nil`
   (no `[]Rect` copies).

## Hard ban (until golden 1× / 2× tests exist)

Do **not** ship again:

- `Context.Scroll` blit + strip-only damage
- `DrawSceneDamage` that skips ops
- skipping `Surface.Present` on GPU
- ClearRect of a hover box that contains static text, unless a golden
  test at 1× and 2× shows identical glyphs vs a full paint
