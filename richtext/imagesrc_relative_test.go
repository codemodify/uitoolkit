package richtext

import "testing"

// A protocol-relative src is a fetch over the network. Reading it as a
// relative path is the dangerous way to be wrong: a client told an image
// is local shows it without asking, and //host/pixel.gif is a tracking
// pixel written the one way that used to slip past.
func TestProtocolRelativeSrcIsRemote(t *testing.T) {
	for _, src := range []string{
		"//tracker.example/p.gif",
		"//tracker.example/p.gif?id=1",
		"  //tracker.example/p.gif",
		"//localhost/p.gif",
	} {
		if got := ClassifyImageSrc(src); got != ImageRemote {
			t.Errorf("ClassifyImageSrc(%q) = %v, want remote", src, got)
		}
		if !ClassifyImageSrc(src).Remote() {
			t.Errorf("%q: Remote() is false", src)
		}
	}
	// A single slash is still a path on this machine.
	for _, src := range []string{"/var/mail/logo.png", "images/logo.png", "./a.png"} {
		if got := ClassifyImageSrc(src); got != ImageLocal {
			t.Errorf("ClassifyImageSrc(%q) = %v, want local", src, got)
		}
	}
}
