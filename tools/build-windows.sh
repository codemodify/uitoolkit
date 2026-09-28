#!/bin/bash
# build-windows.sh — the cheap gate for the Windows backend.
#
# Nothing here runs a Windows binary; it makes sure the Windows code is
# still *compiled and type-checked*, which a plain `go build ./...` on
# Linux does not do at all: every file in platform/ that Windows uses is
# behind //go:build windows, so tools/test.sh ./... never even reads them.
# Three shipped bugs went out through that gap (docs/windows.md).
#
# `go test -c` matters as much as `go build`: it compiles the Windows
# *tests* too, so they cannot rot unnoticed between runs on real Windows.
# It is thrown away — running it is tools/test-windows.sh's job.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOOS=windows GOARCH=amd64 CGO_ENABLED=0

echo "== build =="
go build ./...

echo "== vet =="
go vet ./...

echo "== test binaries compile =="
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
for p in $(go list ./... ); do
  # Only packages that actually have test files; the rest answer
  # "no test files" and are not worth the compile.
  if go test -c -o "$out/t.exe" "$p" 2>"$out/err"; then
    :
  elif grep -q 'no test files' "$out/err"; then
    :
  else
    cat "$out/err"; exit 1
  fi
done
echo "ok: the Windows build, its vet and its tests all compile"
