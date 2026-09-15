#!/bin/bash
# smoke.sh: the accessibility smoke test. A private D-Bus session gets a
# private accessibility bus and registry; the gallery runs headless with the
# AT-SPI2 bridge on, and smoke.py reads and drives it through libatspi, as a
# screen reader would. Nothing touches the desktop's session.
set -e
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
tmp=${TMPDIR:-/tmp}
(cd "$root" && go build -o "$tmp/demo" ./tools/a11y/demo)
registry=/usr/lib/at-spi2-registryd
[ -x "$registry" ] || registry=/usr/libexec/at-spi2-registryd
conf=/usr/share/defaults/at-spi2/accessibility.conf
exec env -u WAYLAND_DISPLAY -u DISPLAY -u AT_SPI_BUS_ADDRESS dbus-run-session -- bash -c "
  export AT_SPI_BUS_ADDRESS=\$(dbus-daemon --config-file=$conf --fork --print-address=1 --print-pid=3 3>'$tmp/a11ybus.pid')
  $registry & sleep 0.5
  '$tmp/demo' & sleep 1.5
  python3 '$here/smoke.py'; rc=\$?
  kill %1 %2 \$(cat '$tmp/a11ybus.pid') 2>/dev/null; exit \$rc"
