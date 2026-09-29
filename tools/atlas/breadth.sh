#!/bin/bash
# breadth.sh [DEST] — the contact sheets docs/widgets.md links from the
# rich-text editor, the MDI area and the wizard: nine looks at 1x and at
# 1.75, three to a row. DEST defaults to docs/screenshots/breadth.
#
# This exists for the reason tools/atlas/frames.sh exists. The sheets
# were composed by hand once, and by the time anything about what they
# show changed — the rich-text editor gained tables, quotes and rules —
# nobody could rebuild them without reconstructing the montage arguments
# from the pictures. A screenshot in the documentation that no script
# can rebuild is a screenshot that will be wrong and stay wrong.
#
# The stills come from the widgets' own shot tests, which write them when
# UITK_SHOTS names a directory (docs/testing.md). Everything runs under
# tools/test.sh, so it is the theme_engine_all build: the nine looks are
# from nine different engines and a default build carries one of them.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/../.." && pwd)
DEST=${1:-$REPO/docs/screenshots/breadth}
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$DEST"

echo "== taking the stills =="
cd "$REPO"
UITK_SHOTS="$WORK" tools/test.sh ./widgets/ -run 'Shot' -count=1 >/dev/null

# Each sheet is one widget in one state at one scale. The stills are
# named <widget>-<pack>@<scale>[-<state>].png, so the glob picks the
# state as well as the scale.
sheet() { # name, glob, scale-suffix
  local name=$1 glob=$2 suffix=$3
  local files=("$WORK"/$glob)
  [ -e "${files[0]}" ] || { echo "no stills for $glob" >&2; return 1; }
  # Three to a row, no labels, on the grey these sheets have always used.
  magick montage -background 'srgb(128,128,128)' -tile 3x -geometry +8+8 \
    "${files[@]}" "$WORK/raw-$name.png"
  magick "$WORK/raw-$name.png" -strip -quality 82 -define webp:method=6 "$DEST/$name$suffix.webp"
  echo "$name$suffix: ${#files[@]} looks, $(magick identify -format '%wx%h' "$DEST/$name$suffix.webp"), $(( $(stat -c %s "$DEST/$name$suffix.webp") / 1024 )) KB"
}

echo "== composing =="
for s in "1:-1x" "1.75:-175x"; do
  scale=${s%%:*} suffix=${s##*:}
  sheet richtext "richtext-*@$scale.png" "$suffix"
  sheet mdi "mdi-*@$scale.png" "$suffix"
  sheet wizard "wizard-*@$scale.png" "$suffix"
done
# The MDI area's other two states and the wizard's other layout are 1x
# only: they say what the state looks like, not how it scales.
sheet mdi-max "mdi-*@1-max.png" "-1x"
sheet mdi-tabs "mdi-*@1-tabs.png" "-1x"
sheet wizard-other "wizard-*@1-other.png" "-1x"
