package richtext

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// An image's source says what loading it costs. A mail client cannot
// treat the two the same: an inline part came with the message, and a
// remote one tells the sender it was opened, at the address and the
// moment it was opened.
func TestClassifyImageSrc(t *testing.T) {
	for _, c := range []struct {
		src  string
		want ImageKind
	}{
		{"data:image/png;base64,iVBOR", ImageData},
		{"cid:part1.abc@example.com", ImageInline},
		{"CID:Part1@Example", ImageInline},
		{"mid:0000@example", ImageInline},
		{"http://example.com/pixel.gif", ImageRemote},
		{"https://example.com/logo.png", ImageRemote},
		{"HTTPS://EXAMPLE.COM/x.png", ImageRemote},
		{"file:///home/ada/logo.png", ImageLocal},
		// No scheme: a path beside the document, not the scheme "images".
		{"images/logo.png", ImageLocal},
		{"./logo.png", ImageLocal},
		{"/var/tmp/logo.png", ImageLocal},
		{"javascript:alert(1)", ImageUnknown},
		{"", ImageUnknown},
		{"   ", ImageUnknown},
	} {
		if got := ClassifyImageSrc(c.src); got != c.want {
			t.Errorf("ClassifyImageSrc(%q) = %v, want %v", c.src, got, c.want)
		}
	}
	if !ImageRemote.Remote() || ImageInline.Remote() || ImageData.Remote() {
		t.Fatal("Remote() disagrees")
	}
}

// The resolver is told which it is being handed, so an application can
// load the inline parts and leave the tracking pixels.
func TestResolveImageKindIsToldTheKind(t *testing.T) {
	d := New()
	seen := map[string]ImageKind{}
	px := paintengine2d.NewImage(2, 2)
	d.ResolveImageKind = func(src string, kind ImageKind) *paintengine2d.Image {
		seen[src] = kind
		if kind.Remote() {
			return nil // a tracking pixel is not loaded
		}
		return px
	}
	const html = `<p><img src="cid:logo@x" alt="Logo">` +
		`<img src="https://tracker.example/p.gif" alt="">` +
		`<img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" alt="dot"></p>`
	if err := d.SetHTML(html); err != nil {
		t.Fatal(err)
	}
	if seen["cid:logo@x"] != ImageInline {
		t.Fatalf("the inline part was classified %v", seen["cid:logo@x"])
	}
	if seen["https://tracker.example/p.gif"] != ImageRemote {
		t.Fatalf("the remote image was classified %v", seen["https://tracker.example/p.gif"])
	}
	// A data: URI is the document's own and never reaches the callback —
	// there is nothing to fetch, decodable or not.
	for src := range seen {
		if ClassifyImageSrc(src) == ImageData {
			t.Fatalf("a data: URI was sent to the resolver: %q", src)
		}
	}

	imgs := d.Images()
	if len(imgs) != 3 {
		t.Fatalf("%d images, want 3", len(imgs))
	}
	unresolved := d.UnresolvedImages()
	if len(unresolved) != 1 || !unresolved[0].Kind().Remote() {
		t.Fatalf("unresolved %v", unresolved)
	}
	// And the placeholder still carries what it needs to be loaded later.
	if unresolved[0].Src != "https://tracker.example/p.gif" {
		t.Fatalf("the placeholder lost its src: %q", unresolved[0].Src)
	}
	counts := d.ImageCounts()
	if counts[ImageInline] != 1 || counts[ImageRemote] != 1 || counts[ImageData] != 1 {
		t.Fatalf("counts %v", counts)
	}
}

// Setting pixels on an enumerated image loads it, which is how a "load
// remote images" control works without reloading the document.
func TestLoadingAnEnumeratedImage(t *testing.T) {
	d := New()
	if err := d.SetHTML(`<p><img src="https://x/p.png" alt="P"></p>`); err != nil {
		t.Fatal(err)
	}
	left := d.UnresolvedImages()
	if len(left) != 1 {
		t.Fatalf("%d unresolved", len(left))
	}
	left[0].Pixels = paintengine2d.NewImage(4, 4)
	if len(d.UnresolvedImages()) != 0 {
		t.Fatal("still unresolved after its pixels were set")
	}
}

// The plain callback still works, and is what gets asked when it is the
// only one set.
func TestPlainResolverStillWorks(t *testing.T) {
	d := New()
	asked := 0
	d.ResolveImage = func(src string) *paintengine2d.Image {
		asked++
		return paintengine2d.NewImage(1, 1)
	}
	if err := d.SetHTML(`<p><img src="cid:a"><img src="https://b/c.png"></p>`); err != nil {
		t.Fatal(err)
	}
	if asked != 2 {
		t.Fatalf("the plain resolver was asked %d times", asked)
	}
	if len(d.UnresolvedImages()) != 0 {
		t.Fatal("something was left unresolved")
	}
}

// A data: URI whose bytes do not decode is unresolved and stays that
// way: there is nothing to fetch, so it is a broken picture rather than
// one waiting to be loaded. It is still listed, because "this document
// has a picture that will not draw" is worth being able to say.
func TestUndecodableDataURIIsUnresolvedAndNotOffered(t *testing.T) {
	d := New()
	asked := 0
	d.ResolveImage = func(string) *paintengine2d.Image { asked++; return nil }
	if err := d.SetHTML(`<p><img src="data:image/gif;base64,not-a-gif" alt="x"></p>`); err != nil {
		t.Fatal(err)
	}
	if asked != 0 {
		t.Fatalf("the resolver was asked %d times for a data: URI", asked)
	}
	left := d.UnresolvedImages()
	if len(left) != 1 || left[0].Kind() != ImageData {
		t.Fatalf("unresolved %v", left)
	}
}
