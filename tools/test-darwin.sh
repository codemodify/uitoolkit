#!/bin/bash
# test-darwin.sh — run the suite on a Mac, from here.
#
#   tools/test-darwin.sh ./...
#   tools/test-darwin.sh -run TestFoo ./platform/
#
# The macOS backend is cgo and Objective-C, so unlike the Windows one it
# cannot be cross-compiled from Linux at all: there is no darwin toolchain
# here and `go build` for darwin with CGO_ENABLED=1 needs Apple's headers
# and clang. So the tree is mirrored to a Mac, built there, and the output
# brought back — the Mac is the compiler, not just the machine that runs
# the binary.
#
# It is rsync and ssh rather than the HTTP-over-slirp contraption
# tools/test-windows.sh needs, because a Mac already has sshd and can be
# told to turn it on in one line; the Windows guests could not.
#
# UITK_MAC_HOST  user@host of the Mac; else ~/.config/uitoolkit/mac-host
# UITK_MAC_KEY   ssh key to use, if not the agent's
# UITK_MAC_DIR   where to mirror the tree (default ~/uitoolkit)
# UITK_MAC_GO    the Go toolchain there (default ~/sdk/go1.27.1)
#
# What the Mac needs, once:
#   * sshd on — System Settings ▸ General ▸ Sharing ▸ Remote Login;
#   * Xcode command line tools (xcode-select --install), for clang and the
#     macOS SDK the cgo backend compiles against;
#   * a Go toolchain matching this repository's.
#
# A note on looking at what it drew: GUI tests may open real windows here
# and that is fine, but `screencapture` run over ssh has no Screen
# Recording permission and does not say so — it returns a clean-looking
# PNG of the desktop with every window silently missing. Verify rendering
# from inside the process instead (docs/macos.md).
set -euo pipefail
cd "$(dirname "$0")/.."

# Where the Mac is, in the order that survives: what you said, what you
# wrote down once, and then its own name on the network.
#
# **Prefer the name over the address.** A laptop's address changes with
# the network it is on — this one has been 10.2.16.126, 172.20.10.9 and
# 10.60.3.244 — and every time it moves, a note with an address in it is
# wrong and the next person debugs ssh instead of the toolkit. A Mac
# announces itself over mDNS as <LocalHostName>.local, which follows it,
# and this box resolves that already (nsswitch has mdns_minimal).
HOST=${UITK_MAC_HOST:-}
HOSTFILE=${UITK_MAC_HOSTFILE:-${XDG_CONFIG_HOME:-$HOME/.config}/uitoolkit/mac-host}
if [ -z "$HOST" ] && [ -r "$HOSTFILE" ]; then
  HOST=$(grep -vE '^\s*(#|$)' "$HOSTFILE" | head -1 | tr -d '[:space:]')
fi
if [ -z "$HOST" ]; then
  cat >&2 <<'MSG'
No Mac to build on. Either:

  UITK_MAC_HOST=user@name.local tools/test-darwin.sh ./...

or write it down once, and this finds it every time after:

  mkdir -p ~/.config/uitoolkit
  echo user@name.local > ~/.config/uitoolkit/mac-host

Use the .local name rather than an address: a laptop's address changes
with the network, the name does not. `avahi-browse -tr _ssh._tcp` lists
the Macs on this network with Remote Login on.
MSG
  exit 2
fi
if ! ssh -o BatchMode=yes -o ConnectTimeout=10 "$HOST" true 2>/dev/null; then
  echo "cannot reach $HOST over ssh (key, Remote Login, or the name moved)" >&2
  echo "try: avahi-browse -tr _ssh._tcp" >&2
  exit 2
fi
DIR=${UITK_MAC_DIR:-'~/uitoolkit'}
GOROOT=${UITK_MAC_GO:-'$HOME/sdk/go1.27.1'}
SSH=(ssh -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new)
if [ -n "${UITK_MAC_KEY:-}" ]; then
  SSH+=(-i "$UITK_MAC_KEY" -o IdentitiesOnly=yes)
fi

echo "== mirroring the tree to $HOST:$DIR =="
rsync -az --delete --exclude '.git' --exclude 'tools/atlas/out' --exclude '*.test' \
  -e "${SSH[*]}" ./ "$HOST:$DIR/"

# UITK_SYSTEM_FONTS=0 for the reason tools/test.sh gives, and it matters
# more here: macOS has no fontconfig, so this machine cannot read the Mac's
# fonts and the Mac cannot read this one's. Pinned, both measure the same.
#
# There is no testenv.sh equivalent to wrap this in. That script exists to
# keep tests off the desktop's Wayland, X11 and D-Bus session, and macOS
# has none of the three: there is nothing here for a test to reach into by
# accident, and the AppKit tests want a window server on purpose.
# The arguments go through a second shell on the far side, so each one is
# quoted for it: without this a -run 'A|B' arrives as a pipeline and the
# Mac tries to run a command called TestB.
# A package to test, always. `go test` given flags alone tests the package
# in the working directory — the repository root, which holds a handful of
# files — so a run meant for the whole toolkit finished in a quarter of a
# second having tested almost nothing, and printed ok. It happened twice in
# one afternoon, both times reported as a pass. Name no package and the
# whole tree is the default.
have_pkg=0
for a in "$@"; do
  case "$a" in ./*|.|github.com/*) have_pkg=1 ;; esac
done
[ "$have_pkg" = 1 ] || set -- "$@" ./...

args=""
for a in "$@"; do args+=" $(printf %q "$a")"; done

# Resolved here and sent as a literal. Written as ${VAR:-0} inside the
# remote command it would be expanded by the *remote* shell, where the
# variable was never set — so it read 0 however it was set here, and
# "run the suite against this machine's fonts" quietly did not.
fonts=${UITK_SYSTEM_FONTS:-0}

echo "== building and running them there =="
exec "${SSH[@]}" "$HOST" \
  "export PATH=$GOROOT/bin:\$PATH CGO_ENABLED=1 UITK_SYSTEM_FONTS=$fonts; \
   cd $DIR && go test -tags theme_engine_all$args"
