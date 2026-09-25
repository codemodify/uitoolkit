#!/bin/bash
# render.sh — renders every pack twice into out/: the Settings preview panel
# (out/previews/<id>.png) and the full widget showcase (out/gallery/<id>.png,
# taken by `uitk-shots -gallery`),
# plus out/packs.tsv, the list build.py and sheets.sh read. Headless: nothing
# opens on the desktop. Takes a few minutes for every pack.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/../.." && pwd)
OUT=${ATLAS_OUT:-$HERE/out}
mkdir -p "$OUT/bin" "$OUT/full" "$OUT/previews" "$OUT/gallery" "$OUT/cfg" "$OUT/work"
(cd "$REPO" && go build -o "$OUT/bin/settings" ./cmd/uitoolkit-settings &&
  go build -o "$OUT/bin/sheet" ./cmd/uitk-themesheet &&
  go build -o "$OUT/bin/shots" ./cmd/uitk-shots)
headless() { env -u WAYLAND_DISPLAY -u DISPLAY XDG_CONFIG_HOME="$OUT/cfg" "$@"; }
headless "$OUT/bin/sheet" -list > "$OUT/packs.tsv"

# Where the preview panel sits in a 1024x860 Settings window at scale 1, with
# an empty config (docs/settings.md, "Screenshot geometry"). What the preview
# carries inside itself does not move the crop — the panel takes whatever the
# block over it and the paths under it leave — but what stands over and under
# it does. The three choosers came off a bar inside the window and onto the
# page, which gave that block a second line and pushed the top down 30 px;
# the paths under it gave up their group box, which was 34. The column on the
# left is outside the crop and cannot move it: the splitter's ratio comes from
# the window's width, not from what the column holds. Add -plain-preview to the
# settings call below for tiles whose File menu opens nothing.
CROP=697x651+317+74
fail=0
while IFS=$'\t' read -r id _; do
  if headless "$OUT/bin/settings" -stage "$id" -screenshot "$OUT/full/$id.png" >/dev/null 2>&1; then
    magick "$OUT/full/$id.png" -crop "$CROP" +repage "$OUT/previews/$id.png"
  else
    echo "preview failed: $id"; fail=1
  fi
  if (cd "$OUT/work" && UITK_THEME="$id" timeout 60 env -u WAYLAND_DISPLAY -u DISPLAY \
      XDG_CONFIG_HOME="$OUT/cfg" "$OUT/bin/shots" -gallery "$OUT/gallery/$id.png" -tab 4 >/dev/null 2>&1); then
    :
  else
    echo "gallery failed: $id"; fail=1
  fi
done < "$OUT/packs.tsv"
echo "rendered $(ls "$OUT/previews" | wc -l) previews and $(ls "$OUT/gallery" | wc -l) galleries"
exit $fail
