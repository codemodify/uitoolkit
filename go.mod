module github.com/codemodify/uitoolkit

go 1.22.2

require (
	github.com/codemodify/paintengine2d v0.11.0
	golang.org/x/image v0.18.0
)

require golang.org/x/text v0.16.0

require github.com/godbus/dbus/v5 v5.1.0

// uitoolkit and paintengine2d are developed in lockstep and live side by side
// in the GOPATH-style tree, so the engine is built from the sibling checkout
// rather than the module proxy. Without this, a stale pin silently resolves to
// whatever version happens to be in the module cache — which is how v0.19.1
// ended up building against engine v0.9.0 while the checkout said v0.10.0.
// Drop this line once paintengine2d v0.11.0 is tagged and go.sum updated;
// it is ignored when uitoolkit is consumed as a dependency.
replace github.com/codemodify/paintengine2d => ../paintengine2d
