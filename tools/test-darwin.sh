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
# UITK_MAC_HOST  user@host of the Mac (required)
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

HOST=${UITK_MAC_HOST:-}
if [ -z "$HOST" ]; then
  echo "set UITK_MAC_HOST to user@host of a Mac with Remote Login on" >&2
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
echo "== building and running them there =="
exec "${SSH[@]}" "$HOST" \
  "export PATH=$GOROOT/bin:\$PATH CGO_ENABLED=1 UITK_SYSTEM_FONTS=\${UITK_SYSTEM_FONTS:-0}; \
   cd $DIR && go test -tags theme_engine_all $*"
