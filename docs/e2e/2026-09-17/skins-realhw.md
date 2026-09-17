# skins on real hardware (instance 17)

Nested KWin on the real GPU, `tools/e2e` instance 17, KWin ScreenShot2 captures.
Nothing was run against the user's session. Instance stopped and
`/run/user/1000/uitk-e2e-17*` removed afterwards.

Builds: `tools/e2e/17/bin/{gallery,uitksettings,mail}` from this branch.
Shots: `tools/e2e/17/shots/*.png` (the rig directory is gitignored).

## What was checked

| # | what | how | result |
| --- | --- | --- | --- |
| 1 | Nocturne, gallery, server-side frame | `UITK_THEME=nocturne run.sh 17 gallery` | ok — `gallery-nocturne-1x.png` |
| 2 | Nocturne, gallery, the skin's own frame | `+ UITK_DECORATIONS=client` | ok — `gallery-nocturne-csd.png` |
| 3 | Cassette, gallery, the skin's own frame | `UITK_THEME=cassette` | ok — `gallery-cassette-1x.png` |
| 4 | Nocturne at 1.75 | `kscreen-doctor output.Virtual-0.scale.1.75` | ok — `gallery-nocturne-175.png` |
| 5 | Cassette at 1.75 (the pixelated path) | same | ok — `gallery-cassette-175.png` |
| 6 | A user skin as an app's own look | `look.json` → `liveprobe` under `17/cfg`, Mail | ok — `live-before.png` |
| 7 | **Live reload of an edited skin** | edit `skins/liveprobe/skin.json` under the running app | ok — `live-after.png` |
| 8 | Settings: the skin browser and preview | `uitksettings -stage nocturne` | ok — `settings-nocturne-1x.png` |
| 9 | Settings at 1.75 with a pixel skin staged | `-stage cassette` at 1.75 | ok — `settings-cassette-175.png` |

No panics, no fatals and no errors in `17/app.log` or `17/kwin.log` across the
session.

## Findings

**The frame.** With `UITK_DECORATIONS=client` a skin paints its own caption:
the band from the `caption` part, the title in a real font (bold, centred,
elided), and the caption buttons from `caption.button`. Judging the first run
on screen changed the design — the buttons were originally the skin's raised
push-button face, which reads as heavy in a title bar. They now take
`caption.button` if the skin binds it, then `tool` (flat until hovered, which
is what every frame of the last twenty years does), then `button`. Nocturne
binds a flat wash; Cassette binds raised keys, because that is what its era
did. The glyph over the button is always the toolkit's own
(`DrawCaptionGlyph`), so a caption button says what it does at every scale and
in every skin.

**1.75 is exact.** Nocturne's buttons at 1.75 on the GPU are clean rounded
corners, an even gradient and a 1px border, with no seam where the nine-slice's
middle meets its fixed edges — the clamp-to-edge padding in `skin_assets.go`
doing its job. Cassette at 1.75 is hard pixel bevels with no blur anywhere: its
2× sheet drawn nearest-neighbour at 2× and fitted, while the text beside it is
at the true 1.75. That is the `pixelated` contract working on real hardware,
and it is the reason the second demo skin exists.

**Live reload.** Mail was left running on a user-installed skin; its
`skin.json` was then edited in place to rebind `parts.button.states.normal` to
a different sprite and to change `colors.text`. Within one watcher poll the
running app repainted with amber buttons and cyan body text — no restart, no
flicker, nothing in the log. It goes through the machinery that already
reloads an edited `theme.json`: `ThemeSourceFile` answers the skin's manifest,
the look watcher stamps it, and `ReloadPreferredLook` → `InvalidateIconCache`
drops the skin's parsed manifest and its cut sprites on the same generation
counter.

**Settings.** Both skins list as `2026 · Cassette` and `2026 · Nocturne` with
their year, lineage, `engine skin` and summary, and the live preview panel is
genuinely skinned at both scales. No change to Settings was needed: a skin is
a pack and the browser already knew how to render one.

## Not a skin bug

`uitksettings -stage <pack>` paints its own window in the palette starter
(Classic 95 Dark) rather than in the look `look.json` selects. It does this
identically for `-stage aqua` and `-stage nocturne`, and not at all without
`-stage` (`settings-liveprobe-rig.png` shows the window correctly in the
installed skin). Pre-existing behaviour of the `-stage` path in
`internal/demo`, unrelated to skins; recorded here because it is visible in
`settings-nocturne-1x.png`.
