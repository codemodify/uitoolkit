package richtext

import (
	"strings"

	"github.com/codemodify/paintengine2d"
)

// ImageKind is what an image's Src points at, which decides whether
// loading it is safe.
//
// A mail client cannot treat the two the same. An inline part (`cid:`)
// came with the message and costs nothing to show; a remote one is a
// fetch that tells the sender the message was opened, at the address and
// the moment it was opened, which is what a tracking pixel is for. The
// callback used to be handed every non-`data:` src with nothing to tell
// them apart, so an application either loaded both or neither, and
// re-parsed the HTML with a regular expression to find out which was
// which.
type ImageKind uint8

const (
	// ImageData is a data: URI the document carries itself. It never
	// reaches [Doc.ResolveImage] — there is nothing to resolve.
	ImageData ImageKind = iota
	// ImageInline is a part that travelled with the document: cid: in
	// mail (RFC 2392), and the mid: of a message reference.
	ImageInline
	// ImageRemote is a fetch over the network: http, https, and the
	// schemes that are one in disguise.
	ImageRemote
	// ImageLocal is a file on this machine — file:, and a bare relative
	// path, which is what a saved HTML document's images look like.
	ImageLocal
	// ImageUnknown is a scheme none of the above: neither safe to fetch
	// nor obviously local, so an application should refuse it unless it
	// knows better.
	ImageUnknown
)

func (k ImageKind) String() string {
	switch k {
	case ImageData:
		return "data"
	case ImageInline:
		return "inline"
	case ImageRemote:
		return "remote"
	case ImageLocal:
		return "local"
	default:
		return "unknown"
	}
}

// Remote reports whether showing this image means a request over the
// network — the one question a "load remote images" control asks.
func (k ImageKind) Remote() bool { return k == ImageRemote }

// ClassifyImageSrc says what src points at.
//
// It reads the scheme and nothing else: it does not resolve, fetch or
// canonicalise, because the answer is needed *before* deciding whether
// to do any of those.
func ClassifyImageSrc(src string) ImageKind {
	s := strings.TrimSpace(src)
	if s == "" {
		return ImageUnknown
	}
	scheme, rest, ok := strings.Cut(s, ":")
	if !ok || !isURLScheme(scheme) {
		// No scheme at all: a relative path beside the document.
		return ImageLocal
	}
	_ = rest
	switch strings.ToLower(scheme) {
	case "data":
		return ImageData
	case "cid", "mid":
		return ImageInline
	case "http", "https", "ftp", "ftps", "ws", "wss":
		return ImageRemote
	case "file":
		return ImageLocal
	default:
		return ImageUnknown
	}
}

// isURLScheme reports whether s is a scheme rather than the first
// segment of a relative path: a letter followed by letters, digits, +, -
// or . (RFC 3986). Without this, "images/logo.png" would be read as the
// scheme "images".
func isURLScheme(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.'):
		default:
			return false
		}
	}
	return true
}

// Kind is what this image's Src points at ([ClassifyImageSrc]).
func (im *Image) Kind() ImageKind {
	if im == nil {
		return ImageUnknown
	}
	return ClassifyImageSrc(im.Src)
}

// Images is every image in the document, in reading order.
//
// It is what a "load images" control is built from: how many there are,
// what kinds, and which are still unresolved — questions an application
// had to answer by parsing the HTML again with a regular expression,
// because the document would not say.
//
// The pointers are the document's own, so setting Pixels on one loads
// that image; call [Doc.Changed] afterwards, as with any other edit.
func (d *Doc) Images() []*Image {
	if d == nil {
		return nil
	}
	var out []*Image
	for _, b := range d.Blocks() {
		for i := range b.Spans {
			if im := b.Spans[i].Image; im != nil {
				out = append(out, im)
			}
		}
	}
	return out
}

// UnresolvedImages is the images with no pixels yet — what is left to
// load, and what the placeholders on screen are showing alt text for.
func (d *Doc) UnresolvedImages() []*Image {
	var out []*Image
	for _, im := range d.Images() {
		if im.Pixels == nil {
			out = append(out, im)
		}
	}
	return out
}

// ImageCounts is how many images of each kind the document has, for a
// control that says "3 remote images not shown".
func (d *Doc) ImageCounts() map[ImageKind]int {
	out := map[ImageKind]int{}
	for _, im := range d.Images() {
		out[im.Kind()]++
	}
	return out
}

// ImageResolver is the one function the HTML parser calls: the
// kind-aware callback where there is one, otherwise the plain one.
//
// A caller that sets both gets the kind-aware one, because the plain one
// cannot express anything the other cannot and setting both is a caller
// in the middle of moving from one to the other.
func (d *Doc) ImageResolver() func(src string) *paintengine2d.Image {
	if d == nil {
		return nil
	}
	if fn := d.ResolveImageKind; fn != nil {
		return func(src string) *paintengine2d.Image {
			return fn(src, ClassifyImageSrc(src))
		}
	}
	return d.ResolveImage
}
