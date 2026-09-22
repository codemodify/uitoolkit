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
# The portal flows: importer is a file dialog's stand-in, the child of the
# window a portal call names (xdg-foreign on Wayland, WM_TRANSIENT_FOR on
# X11); fakeportal owns the portal and notification names on the rig's bus.
wp=/usr/share/wayland-protocols
wayland-scanner client-header $wp/stable/xdg-shell/xdg-shell.xml xdg-shell-client-protocol.h
wayland-scanner private-code $wp/stable/xdg-shell/xdg-shell.xml xdg-shell-protocol.c
wayland-scanner client-header $wp/unstable/xdg-foreign/xdg-foreign-unstable-v2.xml xdg-foreign-unstable-v2-client-protocol.h
wayland-scanner private-code $wp/unstable/xdg-foreign/xdg-foreign-unstable-v2.xml xdg-foreign-unstable-v2-protocol.c
cc -O1 -o importer importer.c xdg-shell-protocol.c xdg-foreign-unstable-v2-protocol.c $(pkg-config --cflags --libs wayland-client x11)
echo "built $(pwd)/importer"
(cd fakeportal && go build -o ../fakeportal-bin .)
echo "built $(pwd)/fakeportal-bin"
