# Theme engines are opt-in

132 theme packs are drawn by 33 engines, and an application that wants
three of them should not carry all of them. Engines are chosen at
**build time**, with build tags.

(33 is what `style.EngineIDs()` returns and what the About box in
Settings prints. There are **34 tags**, because two engines are era
variants another engine hands off to — `breeze6` for a pack whose
`plasma` parameter is 6 or more, `adwaita48` likewise — and each can be
left out on its own without losing the engine it belongs to.)

```sh
go build ./...                              # the default engine alone
go build -tags theme_engine_oxygen ./...    # Oxygen instead of it
go build -tags "theme_engine_oxygen,theme_engine_beos" ./...
go build -tags theme_engine_all ./...       # every engine there is
```

Four rules, and they compose:

1. **Name an engine and you get that engine.** `theme_engine_<id>`, as
   many as you like.
2. **Name none and you get the default**, `plastik` — KDE 3's default
   style — so a build always has a working look and a theme list that is
   not empty.
3. **Name one and the default steps aside.** It is there to be a sensible
   answer when nothing was asked for, not to be carried by everyone.
   `theme_engine_plastik` keeps it alongside the others.
4. `theme_engine_all` is every engine.

Measured on `cmd/uitoolkit-settings`, stripped (`-ldflags "-s -w"`):

| build | engines | size |
| --- | --- | --- |
| default | 1 | **12.71 MB** |
| `theme_engine_oxygen` | 1 | 12.88 MB |
| three engines | 3 | 13.29 MB |
| `theme_engine_all` | 35 | 21.15 MB |

**8.4 MB**, 40% of the binary, for an application that wants one look.

## How the tags are written

Every engine carries:

```go
//go:build theme_engine_all || theme_engine_oxygen
```

and the default engine carries the negation of all of them:

```go
//go:build theme_engine_all || theme_engine_plastik ||
//         (!theme_engine_adwaita && !theme_engine_aero && ... )
```

which is what makes rule 3 work: naming any engine makes one of those
terms false and the default drops out on its own.

An engine built *on* another brings it along, because that is what it
means to be built on it: `theme_engine_bluecurve` also builds Clearlooks,
`theme_engine_kde1` also builds Windows 95 (KDE 1's widgets are Qt 1's,
which are Windows 95's), `theme_engine_breeze6` brings Breeze,
`theme_engine_adwaita48` brings Adwaita, `theme_engine_macos_tahoe`
brings macOS, `theme_engine_material_expressive` brings Material.

Build tags cannot contain hyphens — `//go:build theme-engine-oxygen` is a
syntax error, `parsing //go:build line: invalid syntax at -` — so the
names use underscores throughout.

## The kits: what belongs to no engine

Engines could not be separated at first, and the reason was always the
same: a helper written inside whichever engine happened to need it first,
which every later engine then called. `snap`, a function that rounds to
the pixel grid, lived in `engine_win95.go` and had **1001 call sites
across 35 engines and the core**. While it lived there, no build could
leave Windows 95 out.

So there are kit files, with no build tag, on one rule: **anything more
than one engine needs belongs to all of them.**

| file | what |
| --- | --- |
| `enginekit.go` | `snap`, and the odds and ends |
| `enginekit_win.go` | the Windows chrome: bevels, glyphs, wells, dotted grips |
| `enginekit_color.go` | OKLab/OKLCh, the accent shifts, percentage lighten and darken |
| `enginekit_pixel.go` | the raster port System 7, the Amiga and BeOS draw through |
| `enginekit_kde3.go` | the KDE 3 vocabulary Keramik, Plastik and KDE 2 share |
| `enginekit_web.go` | the parts the web-derived design systems have in common |
| `enginekit_draw.go` | small drawing helpers |

Finding them is not a reading exercise. Static analysis said 382
declarations were shared and every engine's dependency closure was 36 of
42 — both badly wrong, inflated by method names and local variables that
happen to be called `ok`, `set`, `line`, `rect`. **The compiler is the
only reliable instrument**: tag an engine, build, and read what it says is
missing. `tools/engines-build.sh` does that for every engine, and is how
a new entanglement gets caught.

## Testing a toolkit whose parts are optional

`tools/test.sh ./...` runs the suite with `theme_engine_all`, and that is
the suite that covers what this repository ships. A plain
`go test ./...` tests the *default product build* — one engine — where a
test written about what Aqua paints has nothing to assert.

**A test like that is meant to skip, and only `style`'s do.** The
helpers are `needEngine` and `packBuilt` in
`style/engine_optin_test.go`: a test that walks a table of packs drops
the rows whose engine is not in this build, and one that is about a
single engine skips outright. `style` uses them throughout and passes on
its own.

The other packages do not have them, and **64 tests fail in a plain
`go test ./...`** because of it — 24 in `widgets`, 21 in
`cmd/uitoolkit-settings`, 18 in `app`, one in the tour. Every one is the
same thing in a different disguise: a pack named that a default build
does not carry (`unknown pack "luna"`), a look set and the fallback got
back, an assertion about what Windows 95 in particular paints, a view
that needs a long list to overflow, or a count of the registry. None of
them is a defect in the toolkit; all of them are tests written in the
all-engine build's terms.

It is worth knowing what that costs, because it is not nothing: **the
configuration a plain `go build` produces is the one with no trustworthy
test signal.** A regression there lands in an already-red run and nobody
sees it. Until the helpers reach those four packages, a change that
could plausibly affect the one-engine build has to be checked by
comparing the failure list before and after rather than by reading a
green result.

The narrowed builds that *are* covered on purpose:

- `tools/testenv.sh go test ./style/` — the default, one engine;
- the same with `-tags theme_engine_oxygen` — somebody else's pick;
- `tools/engines-build.sh` — every engine, built on its own.

## What an engine leaving takes with it

An engine registers itself and its packs from `init()`
([style.RegisterEngine], [style.RegisterPack]). Leave it out and both go:
its packs are not in `ListThemes`, and Settings does not offer them.

A pack names its engine **by string** in `theme.json`, so a name can
outlive the code that paints it. That is deliberate — a user's `look.json`
naming an engine this build does not have must not be fatal — and
`engineFor` falls back to the base engine. `TestUnbuiltEngineFallsBackToBase`
pins it.

### A pack this build does not have

`look.json` is shared with every other uitoolkit application on the
machine, so a name from somebody else's build — with other engines in it
— reaches this one routinely. Three things happen, and the third is the
one that matters:

1. **The name survives.** `Appearance.Name` keeps it, canonicalised, and
   `SaveAppearance` writes it back, so the user's theme returns the
   moment a build carries its engine. Throwing it away would be a
   preference silently lost.
2. **`Appearance.Missing()` says so**, and `style.MissingThemeNote(name)`
   is the line to log or show: it names the pack asked for and the one
   being shown. Both are answered rather than stored, so they cannot go
   stale.
3. **The look is this build's default pack**, not dark. The family used
   to be run through `ParseTheme`, which answers the two words "light"
   and "dark" and reads everything else as dark — so *every* pack a
   build was missing came up dark, and a light pack like `metal-steel`
   showed as black with nothing anywhere to say why. It is now
   `DefaultTheme()`: the look this build would have shown if the file
   had said nothing.

Why the family cannot simply be looked up: a pack's family lives in the
record its engine's `init` registers, and an engine that is not compiled
in leaves nothing behind to be asked. What is missing is the `init`
itself. A **user** pack (`themes/<name>/theme.json`) is never missing —
it carries its own palette and needs nobody's `init`.

## What is not separable yet

**No engine has to be always-built any more.** This page used to list
three — `metal`, for registering the default theme, and `aqua` and
`motif`, for the legacy era packs of the same name in core — and both
reasons are gone:

- `style.DefaultTheme()` answers `DefaultThemeName` where the build has
  it and otherwise the first pack the built engines registered, so the
  default is never a name that is not there. `go build -tags
  theme_engine_oxygen` carries `[base oxygen skin]` and nothing else.
- `eraPackIndex` offers a legacy era pack only when its engine is built
  (`style/packs.go`), so dropping `aqua` drops the name rather than
  leaving it to be painted wrong by the base engine.

What remains is entanglement between engines, which costs size rather
than correctness.

29 engine groups reach into each other: `win95` alone is
reached into by 38 others (and by core, for `snap` and `w95`), `aero` by
10 (and core, for `winSnap`), `material_tone` by 13, `web` by 9. Helpers
live in whichever engine file first needed them, so one cannot leave
without the others. Extracting those into untagged files is the bulk of
the remaining 6 MB.

The same rule applies to the tests, and the same fix: a helper more than
one engine's tests need belongs to all of them
(`style/enginetest_shared_test.go`), and a test that walks a table of
packs skips the rows whose engine is not in this build (`packBuilt`,
`needEngine`).
