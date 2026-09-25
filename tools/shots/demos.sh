#!/bin/bash
# demos.sh [DIR] — the README's pictures of the demos, rendered headless
# with the apps' own -shot flags: every page of the tour (a still each and
# one contact sheet) and Settings with the gallery under its preview. DIR
# defaults to docs/screenshots.
#
# The player stills (minim.webp, marquee.webp, lantern.webp) used to be made
# here too; the player moved to github.com/codemodify/media-player-music and
# takes its own now.
#
# Nothing opens on the desktop, and a clean config directory means the
# default look and no remembered skin. Needs Go and ImageMagick (`magick`),
# which writes the lossless WebPs the pages link to.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/../.." && pwd)
DEST=${1:-$REPO/docs/screenshots}
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$DEST" "$WORK/bin" "$WORK/cfg"
(cd "$REPO" && go build -o "$WORK/bin/" ./examples/uitoolkit-sample-tour ./cmd/uitksettings)
headless() { env -u WAYLAND_DISPLAY -u DISPLAY XDG_CONFIG_HOME="$WORK/cfg" UITK_ANIMATIONS=0 "$@" >/dev/null; }
webp() { # in.png out.webp
  magick "$1" -strip -define webp:lossless=true -define webp:method=6 "$2"
  echo "$(basename "$2"): $(magick identify -format '%wx%h' "$2"), $(( $(stat -c %s "$2") / 1024 )) KB"
}

# The tour: a still of every page, and all of them on one sheet.
headless "$WORK/bin/uitoolkit-sample-tour" -shot "$WORK/tour" -sheet "$WORK/tour-pages.png"
for f in "$WORK"/tour/tour-*.png; do
  webp "$f" "$DEST/$(basename "${f%.png}").webp"
done
webp "$WORK/tour-pages.png" "$DEST/tour-pages.webp"

# Settings: the theme browser, the preview and the gallery, in the default look.
headless "$WORK/bin/uitksettings" -screenshot "$WORK/settings.png"
webp "$WORK/settings.png" "$DEST/settings.webp"
