# Paint performance (after v0.14.7)

v0.14.0–v0.14.6 tried to go faster with dirty-rect clipping, pixel
`Context.Scroll`, `DrawSceneDamage`, and partial GPU present. That was
a bad idea: black first frames, HiDPI scale blow-up, vacated list
holes, stuck menu highlights, and bold→thin glyphs on hover.

**v0.14.7 restores the v0.13.8 paint path.** Do not re-enable those
fast paths until they are proven on Wayland HiDPI with golden tests.

## Current model (correctness first)

1. Dirty marks which widgets to **re-record** into the retained scene.
2. `DrawScene` always replays the **whole** graph.
3. `Surface.Present(nil)` always presents the **full** buffer (sets
   Wayland `buffer_scale` / viewport; no stale tiles).
4. paintengine2d **v0.9.0** — no `DrawSceneDamage` / `BakeGroup` wiring.

## Next stab (safer)

Profile a real Mail / gallery session *before* changing paint:

```bash
CGO_ENABLED=1 go test -bench=BenchmarkListScroll -benchmem ./app
# Wayland dogfood: UITK_SCENE=auto mailclientui + pprof / perf
```

Prefer, in order:

1. **Less work per full paint** — keep row/group caches; skip
   `RequestLayout` on hover; reuse shaped text for unchanged labels.
2. **Idle coalescing** — one frame per event burst, not per widget
   invalidate; WatchLook stays Stat-first.
3. **Alloc cuts** — recorder pools, fewer `[]Rect` copies on Present.

Do **not** ship again:

- `Context.Scroll` blit + strip-only damage
- `DrawSceneDamage` that skips ops
- skipping `Surface.Present` on GPU
- ClearRect of a hover box that contains static text, unless a golden
  test at 1× and 2× shows identical glyphs vs a full paint
