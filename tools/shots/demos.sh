#!/bin/bash
# demos.sh [DIR] — the README's pictures of the demos, rendered headless
# with the apps' own -shot flags: each player in each of its skins, every
# page of the tour (a still each and one contact sheet) and Settings with
# the gallery under its preview. DIR defaults to docs/screenshots.
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
(cd "$REPO" && go build -o "$WORK/bin/" ./examples/minim ./examples/marquee ./examples/lantern \
  ./examples/tour ./cmd/uitksettings)
headless() { env -u WAYLAND_DISPLAY -u DISPLAY XDG_CONFIG_HOME="$WORK/cfg" UITK_ANIMATIONS=0 "$@" >/dev/null; }
webp() { # in.png out.webp
  magick "$1" -strip -define webp:lossless=true -define webp:method=6 "$2"
  echo "$(basename "$2"): $(magick identify -format '%wx%h' "$2"), $(( $(stat -c %s "$2") / 1024 )) KB"
}
gapx() { echo "( -size ${1}x1 xc:none )"; } # a transparent gap for +append
gapy() { echo "( -size 1x${1} xc:none )"; } # ... and for -append

# Minim: its three skins side by side, each as the snapped rack — the strip,
# the equaliser under it, the playlist under that — at 2x, where the pixel
# panels are drawn at whole multiples.
racks=()
for skin in minim minim-classic minim-silver; do
  headless "$WORK/bin/minim" -theme "$skin" -scale 2 -shot "$WORK/$skin"
  magick -background none "$WORK/$skin/minim-strip.png" "$WORK/$skin/minim-equaliser.png" \
    "$WORK/$skin/minim-playlist.png" -append "$WORK/rack-$skin.png"
  racks+=("$WORK/rack-$skin.png")
done
# shellcheck disable=SC2046
magick -background none -gravity north "${racks[0]}" $(gapx 48) "${racks[1]}" $(gapx 48) "${racks[2]}" \
  +append "$WORK/minim.png"
webp "$WORK/minim.png" "$DEST/minim.webp"

# Marquee: the cabinet, and folded into its stadium.
headless "$WORK/bin/marquee" -shot "$WORK/marquee"
# shellcheck disable=SC2046
magick -background none -gravity center "$WORK/marquee/marquee-full.png" $(gapy 32) \
  "$WORK/marquee/marquee-compact.png" -append "$WORK/marquee.png"
webp "$WORK/marquee.png" "$DEST/marquee.webp"

# Lantern: the player and its playlist in the skin, and with it dropped.
headless "$WORK/bin/lantern" -shot "$WORK/lantern"
for look in skinned themed; do
  # shellcheck disable=SC2046
  magick -background none -gravity north "$WORK/lantern/lantern-$look-main.png" $(gapx 16) \
    "$WORK/lantern/lantern-$look-playlist.png" +append "$WORK/lantern-$look.png"
done
# shellcheck disable=SC2046
magick -background none "$WORK/lantern-skinned.png" $(gapy 32) "$WORK/lantern-themed.png" -append "$WORK/lantern.png"
webp "$WORK/lantern.png" "$DEST/lantern.webp"

# The tour: a still of every page, and all of them on one sheet.
headless "$WORK/bin/tour" -shot "$WORK/tour" -sheet "$WORK/tour-pages.png"
for f in "$WORK"/tour/tour-*.png; do
  webp "$f" "$DEST/$(basename "${f%.png}").webp"
done
webp "$WORK/tour-pages.png" "$DEST/tour-pages.webp"

# Settings: the theme browser, the preview and the gallery, in the default look.
headless "$WORK/bin/uitksettings" -screenshot "$WORK/settings.png"
webp "$WORK/settings.png" "$DEST/settings.webp"
