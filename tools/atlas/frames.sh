#!/bin/bash
# frames.sh [DEST] — the README's window-frame sheet: one active frame per
# era, System 1 in 1984 to macOS Tahoe in 2025, each drawn by its own
# engine. DEST defaults to docs/screenshots/themes/frames.webp.
#
# This exists because the image did not have it. It was made by hand
# once, and by the time anything about frame drawing changed — the
# Platinum title, the fonts the suite pins — nobody could regenerate it
# without reconstructing the list of packs from the picture. Which is
# what had to be done. A screenshot in the documentation that no script
# can rebuild is a screenshot that will be wrong and stay wrong.
#
# UITK_SYSTEM_FONTS=0 for the reason tools/test.sh gives: otherwise this
# is a picture of the fonts installed on whoever rendered it last.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/../.." && pwd)
DEST=${1:-$REPO/docs/screenshots/themes/frames.webp}
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# One pack per era, in the order they are shown: by year, with the two
# 1997 window managers and System 7 kept beside their neighbours rather
# than strictly sorted, because the sheet reads as a history and a
# strict sort breaks the pairs that belong together.
PACKS=(
  system1 openlook next motif
  win31 system7 amiga31 win95
  os2warp platinum beos wmaker-default
  luna aqua keramik bluecurve
  metal-ocean aero oxygen nimbus
  win10 bigsur fluent breeze6
  adwaita48 sourcegit islands tahoe
)

cd "$REPO"
echo "== rendering ${#PACKS[@]} frame sheets =="
env -u WAYLAND_DISPLAY -u DISPLAY UITK_SYSTEM_FONTS=0 \
  go run -tags theme_engine_all ./cmd/uitk-themesheet \
  -frames -theme "$(IFS=,; echo "${PACKS[*]}")" -o "$WORK" >/dev/null

# The sheet each pack writes holds nine states; the one this wants is
# the first, "active". Its tile is at a fixed place on every sheet —
# checked against Aqua's pinstripes, System 1's 1-bit frame and Tahoe's
# rounded one, which are as far apart as the set goes.
echo "== cropping the active frame from each =="
declare -A LABEL
while IFS=$'\t' read -r id year _ _ label _; do LABEL[$id]="$label · $year"; done < <(
  env -u WAYLAND_DISPLAY -u DISPLAY UITK_SYSTEM_FONTS=0 \
    go run -tags theme_engine_all ./cmd/uitk-themesheet -list
)
args=()
for id in "${PACKS[@]}"; do
  src="$WORK/$id-frames.png"
  [ -f "$src" ] || { echo "no sheet for $id" >&2; exit 1; }
  magick "$src" -crop 380x160+10+58 +repage "$WORK/tile-$id.png"
  args+=(-label "${LABEL[$id]:-$id}" "$WORK/tile-$id.png")
done

echo "== composing =="
magick montage -font Noto-Sans-Regular -pointsize 15 -fill '#23272e' -background '#eef0f2' \
  "${args[@]}" -tile 4x -geometry +4+4 "$WORK/frames.png"
magick "$WORK/frames.png" -strip -quality 82 -define webp:method=6 "$DEST"
echo "frames: $(magick identify -format '%wx%h' "$DEST"), $(( $(stat -c %s "$DEST") / 1024 )) KB → $DEST"
