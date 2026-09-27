#!/bin/bash
# engines-build.sh — check that every theme engine can be built on its own.
#
# Engines are opt-in (docs/engines.md) and an application may name any one
# of them. That only works if each engine is genuinely separable, and the
# way an engine stops being separable is quiet: someone writes a helper in
# engine_foo.go that engine_bar.go then calls, and nothing says so until a
# build that wants bar but not foo fails at a customer's desk.
#
# So: build each engine alone, and the default and the full set as well.
set -euo pipefail
cd "$(dirname "$0")/.."
fail=0
engines=$(ls style/engine_*.go | grep -v _test.go |
  sed 's|style/engine_||; s|\.go$||' |
  sed -E 's/_(frame|packs|palettes|palettes_editors|parts|systems|views|controls|ide|tone|gtk3|pixel|shape|tile)$//' |
  sort -u)
for g in $engines; do
  grep -qs "theme_engine_$g" style/engine_"$g"*.go || continue
  if ! go build -tags "theme_engine_$g" ./style/ >/dev/null 2>&1; then
    echo "FAIL: theme_engine_$g does not build on its own"
    fail=1
  fi
done
for t in "" theme_engine_all; do
  a=(); [ -n "$t" ] && a=(-tags "$t")
  if ! go build "${a[@]}" ./... >/dev/null 2>&1; then
    echo "FAIL: ${t:-default} build"
    fail=1
  fi
done
[ "$fail" = 0 ] && echo "every engine builds on its own, and so do the default and the full set"
exit $fail
