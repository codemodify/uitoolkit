#!/bin/bash
# test-display.sh [go test args...] — run the suite against a real display.
#
# tools/testenv.sh hides every display on purpose, and that is right for the
# bulk of the suite: a widget test has no business touching the desktop it
# runs on. The cost is that every test which *needs* a server skips, so the
# X11 and Wayland backends — the two files where nearly every regression of
# this toolkit has been — are checked by nothing that runs by default. That
# is a silent substitution: a green suite that never compiled a window.
#
# This runs the same tests inside a nested KWin (tools/e2e/start.sh), which
# gives both a Wayland socket and an Xwayland display, so those tests run.
# Two passes, because an application has one backend and not the other:
#
#   x11       DISPLAY set, WAYLAND_DISPLAY unset
#   wayland   WAYLAND_DISPLAY set, DISPLAY unset
#
#   tools/test-display.sh ./platform/
#   tools/test-display.sh -run TestADialogKeeps ./platform/
#   UITK_DISPLAY_KINDS=x11 tools/test-display.sh ./platform/
#
# Everything else matches tools/test.sh: theme_engine_all, the bundled fonts,
# and XDG dirs of the instance's own so a test writes nothing into the
# user's config or cache. The instance's D-Bus session is used rather than
# the user's, so a test that asks for a tray or a portal reaches the nested
# desktop instead of the real one.
#
# The instance is left running (tools/e2e/stop.sh N) so a failing run can be
# looked at, and reused on the next call.
set -euo pipefail
cd "$(dirname "$0")/.."
N="${UITK_DISPLAY_RIG:-9}"
D="tools/e2e/$N"
KINDS="${UITK_DISPLAY_KINDS:-x11 wayland}"
export UITK_SYSTEM_FONTS=${UITK_SYSTEM_FONTS:-0}

# shellcheck disable=SC2086 -- the size is two words on purpose
tools/e2e/start.sh "$N" ${UITK_DISPLAY_SIZE:-1280 860} >/dev/null
disp=$(cat "$D/display")
bus=$(cat "$D/bus.addr")
[ -n "$disp" ] && [ -n "$bus" ] || { echo "test-display: instance $N has no display or bus" >&2; exit 1; }
echo "instance $N: DISPLAY=$disp WAYLAND_DISPLAY=uitk-e2e-$N"

# Built once, run twice: the compile is the slow half and the two passes
# differ only in their environment.
# Absolute, because Go refuses a relative XDG_CACHE_HOME when it has to
# locate the build cache in it.
tmp="$PWD/$D/gotmp"
mkdir -p "$tmp"
rc=0
for kind in $KINDS; do
  case $kind in
    x11)     env_args=(-u WAYLAND_DISPLAY -u WAYLAND_SOCKET "DISPLAY=$disp") ;;
    wayland) env_args=(-u DISPLAY -u WAYLAND_SOCKET "WAYLAND_DISPLAY=uitk-e2e-$N") ;;
    *) echo "test-display: unknown kind $kind" >&2; exit 2 ;;
  esac
  echo "=== $kind"
  env "${env_args[@]}" \
    DBUS_SESSION_BUS_ADDRESS="$bus" \
    XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}" \
    XDG_CONFIG_HOME="$tmp/config" XDG_CACHE_HOME="$tmp/cache" \
    XDG_DATA_HOME="$tmp/data" GOCACHE="${GOCACHE:-$(go env GOCACHE)}" \
    UITK_SYSTEM_FONTS="$UITK_SYSTEM_FONTS" \
    go test -tags theme_engine_all "$@" || rc=$?
done
exit $rc
