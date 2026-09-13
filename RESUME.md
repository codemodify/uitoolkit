# Resume here — code review fixes, 2026-09-13

Everything from the review pass is merged into `dev` in **both** repos. Nothing is
pushed: the sandbox that did the work could read GitHub but had no credentials, so
there is no PR for this work and `origin` still points at the pre-review `dev`.

## Where things are

| | uitoolkit | paintengine2d |
|---|---|---|
| `dev` (merged) | `c1a983d` | `a66bc6d` |
| version | 0.19.1 → **0.20.0** | 0.10.0 → **0.11.0** |
| commits | 6 + merge | 4 + merge |
| change | 130 files, +14,666 / −2,148 | 31 files, +3,301 / −362 |

- `fix/review-2026-09-13` still exists in both repos — the same commits, unmerged.
- `dev-before-review-merge` is a tag on the pre-merge `dev` in both repos.

**To undo the whole thing:** `git reset --hard dev-before-review-merge` in each repo.

## Push it / open the PR

Run these from a shell that has your GitHub credentials (the sandbox had none).
Either push `dev` straight up, or push the branch and open a PR in your usual style:

```sh
# option A — push the merged dev
cd ~/go/src/github.com/codemodify/paintengine2d && git push origin dev
cd ~/go/src/github.com/codemodify/uitoolkit   && git push origin dev

# option B — PR per repo (engine first: the toolkit depends on it)
cd ~/go/src/github.com/codemodify/paintengine2d
git push -u origin fix/review-2026-09-13
gh pr create --base dev --head fix/review-2026-09-13 \
  --title "Engine review fixes: damage replay, group clips, EGL lifetime (v0.11.0)"

cd ~/go/src/github.com/codemodify/uitoolkit
git push -u origin fix/review-2026-09-13
gh pr create --base dev --head fix/review-2026-09-13 \
  --title "Toolkit review fixes: partial redraw, focus lifecycle, mail hardening (v0.20.0)"
```

If you take option B, reset `dev` back first (`git reset --hard dev-before-review-merge`)
so the PR is not already merged into it.

## Building — read this first

`go.mod` now requires `paintengine2d v0.11.0` **and carries a relative replace** to
`../paintengine2d`. That is deliberate: the old `v0.9.0` pin only resolved because
that version happened to sit in your module cache, so the toolkit was silently
compiling against an engine two releases old. Both repos must be on the same
revision for the build to mean anything.

Drop the `replace` from `go.mod` once you tag and push `paintengine2d v0.11.0`, and
regenerate `go.sum` (`GOFLAGS=-mod=mod go get github.com/codemodify/paintengine2d@v0.11.0`).

`go.work` and `go.work.sum` are now gitignored if you prefer a workspace instead.

## Running

```sh
go run ./examples/gallery          # widget gallery
go run ./examples/mail             # mail client, in-memory demo store
go run ./cmd/uitest-driver         # scripted UI drive, no display needed
go run ./examples/gallery -screenshot /tmp/shots   # render PNGs headlessly
```

Escape hatches added in this pass:

- `UITK_PAINT_FULLFRAME=1` — restore the old full-frame repaint (bypasses the new
  partial-redraw path). Useful to A/B the damage work, or if a compositor misbehaves.
- `UITK_PAINT_MSAA=0` — turn off the engine's new multisample target.

## Verification that was run

Green in both repos, with and without the workspace: `go build`, `go vet`, `go test`,
`go test -race`, `gofmt -l`, `GOOS=darwin go vet`, `GOOS=windows go vet`,
`CGO_ENABLED=0 go build`. Ran in a Linux container with EGL/GLES/X11/Wayland headers;
there was no GPU or display, so GL-dependent tests skip rather than run.

**Not verified on real hardware:** anything needing an actual X11/Wayland server or a
GPU — the Wayland role/unmap changes, partial present against real buffer age, MSAA,
tray behaviour on a live desktop. Worth a manual pass on your box.

## What changed, in one line each

- **engine/raster** — strict rectangle detection (triangles were being filled as rects),
  NaN/overflow-guarded curve flattening (a huge control point cost 9.8 s and 1.4 GB),
  explicit opacity so alpha 0 is invisible, active-edge-list scanline.
- **engine/scene** — damage replay no longer skips the clear or double-blends; parent-space
  `GroupNode.Clip` so "scroll = new Xform" actually works; shared immutable clip masks.
- **engine/gpu** — EGL displays refcounted (closing one window used to kill every other
  window's device), buffer-age-aware partial present, id-keyed LRU texture cache, MSAA,
  surfaced GL/EGL errors, adoptable external context. `go test -race` runs at all now.
- **platform** — close is a *request* again (close-to-tray works), Wayland detaches its
  buffer before re-roling (was a protocol error that killed the client), clipboard cache
  invalidated, non-Latin keyboard layouts, 16bpp visuals, Windows tray thread.
- **style** — glyph atlas is copy-on-write (was an unguarded data race), pixel-snapped
  glyphs (crisper stems — the chrome goldens moved for this, verified), `.notdef` for
  missing runes, bounded atlas memory, theme name sanitising.
- **widgets** — dismissed popups no longer execute items, modals trap focus, TextArea
  paint 25 ms → 2.2 ms, TreeView mouse move 1.5 ms → 218 ns.
- **app** — partial redraw restored (one invalidated button 6.68 ms → 0.215 ms), scene
  cache reuse guarded by origin/size/clip, focus and hover swept on teardown, `Quit`
  race fixed, per-window DPI, `Post` never runs on the caller's goroutine.
- **mail** — path traversal that could `RemoveAll` outside the data dir, unauthenticated
  socket, plaintext credential fallbacks, Bcc leaking into headers, header injection,
  a master-key bug that made every stored OAuth token undecryptable, COPYUID/UID remap,
  deadlines, atomic store writes, I/O off the UI thread.

Full detail: `claude/review-2026-09-13.md` and `claude/fixes-landed-2026-09-13.md` in the
Claude project, plus the per-commit messages (`git log dev-before-review-merge..dev`).

## Picking it up again

- **VS Code:** open either repo folder. The Claude Code extension keeps its own session
  history — it will not see this session's conversation.
- **CLI with the conversation:** `claude --teleport session_01WQNdhvLTRmryrevTeAGdBW`
  (wants clean git state; the branch is local-only, so push it first if teleport
  insists on finding it on `origin`).
- **Just the code:** it is already on `dev`. `cd` in and run `claude`.

## Known loose ends

- The gallery header prints the *toolkit* version next to the engine's name
  ("paintengine2d · v0.20.0"). Cosmetic, not fixed.
- Deferred deliberately, with reasons, in `claude/fixes-landed-2026-09-13.md`:
  dmabuf/EGLImage import and fence export (the main gap for the compositor plan),
  paged font atlases, the macOS tray rewrite (compiles nowhere without a real Mac).
- `mail.png` in the repo root is untracked and predates this work.

## If you want the window manager next

The engine is ready enough — correct damage, parent-space clips, adoptable EGL context,
ARGB present. What is missing is buffer import and fences on the engine side, and a
`platform` backend that is server-shaped: everything there today is a client of one
toplevel. That is a new backend over DRM/KMS + GBM + libinput implementing the existing
`Backend`/`Surface`/`Event` interfaces, not an extension of `wayland_linux.go`.
