#!/bin/bash
# test.sh [go test args...] — run uitoolkit's own test suite.
#
# It runs through tools/testenv.sh, so nothing can reach the desktop this
# machine is running, and with -tags theme_engine_all, so every theme
# engine the toolkit ships is compiled in and tested.
#
#   tools/test.sh ./...
#   tools/test.sh -run TestFoo ./style/
#
# Why the tag: engines are opt-in at build time (docs/engines.md), and a
# plain `go test ./...` therefore tests the *default product build* — one
# engine. That is a real configuration and worth checking (see below), but
# it is not the toolkit: a test written about what Aqua paints has nothing
# to assert when Aqua was not built, and dozens of tests are like that.
# The suite that covers what this repository ships is this one.
#
# The narrowed configurations are covered too, and differently:
#
#   tools/testenv.sh go test ./style/           the default, one engine
#   tools/testenv.sh go test -tags theme_engine_oxygen ./style/
#   tools/engines-build.sh                      every engine, on its own
#
# so that "a build with only these engines works" is a claim with a test
# behind it, rather than one nobody checks.
set -euo pipefail
cd "$(dirname "$0")/.."
exec tools/testenv.sh go test -tags theme_engine_all "$@"
