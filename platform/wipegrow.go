package platform

// appendWiping is append for bytes that must not be left behind.
//
// append reallocates when it runs out of capacity and drops the old array
// for the collector — with whatever was in it. For a passphrase read in
// pieces, a private key pasted into a secret field, that leaves a copy of
// every prefix the transfer outgrew: 4 KiB, then 8, then 16, each still
// holding the start of the secret and none of them reachable to wipe.
//
// This grows by hand and zeroes the array it outgrew before letting go.
func appendWiping(dst, add []byte) []byte {
	if len(add) == 0 {
		return dst
	}
	if cap(dst)-len(dst) >= len(add) {
		return append(dst, add...)
	}
	n := cap(dst) * 2
	if n < len(dst)+len(add) {
		n = len(dst) + len(add)
	}
	if n < 64 {
		n = 64
	}
	grown := make([]byte, len(dst)+len(add), n)
	copy(grown, dst)
	copy(grown[len(dst):], add)
	old := dst[:cap(dst)]
	for i := range old {
		old[i] = 0
	}
	return grown
}
