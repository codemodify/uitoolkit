#!/bin/bash
# test-fonts.sh [go test args...] — the suite in the font configurations
# the default run cannot cover.
#
#   tools/test-fonts.sh
#   tools/test-fonts.sh -run TestARuneTheFaceLacks
#
# Font behaviour is chosen at build time, the way theme engines are
# (docs/engines.md), so "the other build" is a different binary and not a
# different flag. tools/test.sh runs the default one — one face draws
# everything, a rune it lacks is a box — and this runs the rest:
#
#   font_fallback   a rune the face lacks is borrowed from one that has it
#
# Both halves are tested, and they are tested against each other: the
# default build has fontfallback_off_test.go pinning that nothing is
# borrowed, and this run has fontfallback_on_test.go pinning that
# something is. A fallback that quietly turned itself on in the default
# build would fail the first; one that did nothing under its own tag
# would fail the second.
#
# UITK_SYSTEM_FONTS=0 as everywhere else: the tests here lean on the two
# *bundled* faces, which differ in coverage — Titillium Web is Latin only
# and JetBrains Mono has Greek, Cyrillic and the arrows — so a borrowed
# glyph can be proved without a single installed font, on any machine and
# in a clean container. Set it to 1 to watch it work against the real
# ones instead.
set -euo pipefail
cd "$(dirname "$0")/.."
export UITK_SYSTEM_FONTS=${UITK_SYSTEM_FONTS:-0}

pkgs=(./style/)
args=("$@")
for a in "$@"; do
  case "$a" in
    ./*) pkgs=() ;;
  esac
done

echo "=== font_fallback"
exec tools/testenv.sh go test -tags "theme_engine_all,font_fallback" "${args[@]}" "${pkgs[@]}"
