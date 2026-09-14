#!/bin/bash
# build.sh — build the input injector (inj) for the nested-KWin e2e rig.
# Needs wayland-scanner, a C compiler and KWin's fake-input protocol XML
# (Arch: plasma-wayland-protocols).
set -e
cd "$(dirname "$0")"
xml=""
for c in /usr/share/plasma-wayland-protocols/fake-input.xml \
         $(find "$HOME/.cargo/registry" -name fake-input.xml 2>/dev/null | head -1); do
  [ -f "$c" ] && { xml="$c"; break; }
done
[ -n "$xml" ] || { echo "fake-input.xml not found: install plasma-wayland-protocols" >&2; exit 1; }
wayland-scanner client-header "$xml" fake-input-client.h
wayland-scanner private-code "$xml" fake-input.c
cc -O1 -o inj inj.c fake-input.c $(pkg-config --cflags --libs wayland-client)
echo "built $(pwd)/inj"
