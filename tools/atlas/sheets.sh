#!/bin/bash
# sheets.sh [DIR] — the README's contact sheets from out/previews: one per
# decade of desktop packs, one of the web engine's packs and one of the skins,
# plus the timeline image. DIR defaults to docs/screenshots/themes.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/../.." && pwd)
OUT=${ATLAS_OUT:-$HERE/out}
DEST=${1:-$REPO/docs/screenshots/themes}
mkdir -p "$DEST"

sheet() { # name, awk filter over packs.tsv ($2 year, $4 engine)
  local name=$1 filter=$2 args=()
  while IFS=$'\t' read -r id year _ _ label _; do
    args+=(-label "$label · $year" "$OUT/previews/$id.png")
  done < <(awk -F'\t' "$filter" "$OUT/packs.tsv")
  [ ${#args[@]} -gt 0 ] || return 0
  magick montage -font Noto-Sans-Regular -pointsize 15 -fill '#23272e' -background '#eef0f2' \
    "${args[@]}" -tile 5x -geometry 320x214+10+8 "$OUT/raw-$name.png"
  magick "$OUT/raw-$name.png" -strip -quality 82 -define webp:method=6 "$DEST/themes-$name.webp"
  rm -f "$OUT/raw-$name.png"
  echo "$name: $(( ${#args[@]} / 3 )) packs, $(( $(stat -c %s "$DEST/themes-$name.webp") / 1024 )) KB"
}
for d in 1980 1990 2000 2010 2020; do
  sheet "${d}s" "\$4 != \"web\" && \$4 != \"skin\" && int(\$2 / 10) * 10 == $d"
done
sheet web '$4 == "web"'
sheet skins '$4 == "skin"'

# The timeline alone, from the atlas page build.py wrote, through headless Chromium.
python3 - "$OUT/index.html" "$OUT/timeline.html" <<'PY'
import re, sys
s = open(sys.argv[1]).read()
head = s[:s.index('<main class="wrap">')]
fig = re.search(r'<figure class="timeline">.*?</figure>', s, re.S).group(0)
open(sys.argv[2], "w").write(head + '<main class="wrap">\n' + fig + '\n</main>\n')
PY
mkdir -p "$OUT/chrome"
timeout 90 chromium --headless=new --disable-gpu --user-data-dir="$OUT/chrome" \
  --host-resolver-rules="MAP * ~NOTFOUND" --hide-scrollbars --window-size=1400,1600 \
  --screenshot="$OUT/timeline-shot.png" "file://$OUT/timeline.html" >/dev/null 2>&1
bg=$(magick "$OUT/timeline-shot.png" -format '%[pixel:p{5,5}]' info:)
magick "$OUT/timeline-shot.png" -fuzz 2% -trim +repage -bordercolor "$bg" -border 24 -colors 64 "$DEST/timeline.png"
echo "timeline: $(magick identify -format '%wx%h' "$DEST/timeline.png")"
