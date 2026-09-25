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
# an empty config (docs/settings.md, "Screenshot geometry").
CROP=569x782+445+10
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
