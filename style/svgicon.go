package style

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// A rasterizer for the shape of SVG a desktop icon theme is made of, and
// for no other shape of SVG.
//
// It exists because the icon themes people actually have are vector
// themes: Breeze and Papirus ship no PNG at all, and a "list the icon
// themes the system has" that could not open them would list, on a KDE
// machine, every theme except the one the desktop is wearing.
//
// It is not an SVG renderer and must not grow into one. What a themed
// action icon is, in Breeze, Adwaita's symbolic set, Papirus and the
// Yaru scalables, is a viewBox and a handful of filled outlines:
//
//	<svg viewBox="0 0 22 22">
//	  <path style="fill:currentColor" d="M 3 2.99 L 3 19 …"/>
//	</svg>
//
// so this reads a viewBox, walks <g>/<path>/<rect>/<circle>/<ellipse>/
// <polygon>/<polyline>/<line>, honours transform, fill, fill-rule,
// stroke and opacity, and fills paths into an offscreen pixmap. Anything
// past that — <use>, gradients, clip paths, masks, filters, patterns,
// embedded images, text — is not approximated and not ignored: the file
// is refused, the icon does not load, and the theme that needed it is
// listed as one this toolkit cannot draw (see [iconThemeDrawable]). A
// half-drawn icon is worse than a set that says it is unavailable.
//
// currentColor is the whole point of the monochrome themes and is
// rendered as black here; what makes it the look's foreground is the
// same step that tints a monochrome PNG (see [toWhiteMask] and
// [themeIconImage]), so a Breeze icon follows the palette exactly as the
// desktop's own does.

// svgUnsupported is the error a construct outside the subset raises. It
// names the construct, because the reason a theme is unavailable is
// worth being able to print.
type svgUnsupported struct{ what string }

func (e svgUnsupported) Error() string { return "svg: unsupported " + e.what }

// maxSVGBytes is the largest icon file that will be read. A themed action
// icon is one to four kilobytes; a megabyte of XML in an icon directory
// is not an icon, and parsing it on a paint path would be a stall.
const maxSVGBytes = 1 << 20

// svgPaint is the resolved painting of one element.
type svgPaint struct {
	fill        paintengine2d.Color
	hasFill     bool
	fillRule    paintengine2d.FillRule
	stroke      paintengine2d.Color
	hasStroke   bool
	strokeWidth float32
	cap         paintengine2d.Cap
	join        paintengine2d.Join
	opacity     float32
	fillOpacity float32
	strokeAlpha float32
}

func defaultSVGPaint() svgPaint {
	return svgPaint{
		fill:        paintengine2d.Black,
		hasFill:     true,
		strokeWidth: 1,
		opacity:     1,
		fillOpacity: 1,
		strokeAlpha: 1,
	}
}

// RasterizeIconSVG draws the icon file at path into a size×size pixmap.
// It returns an error for anything it will not draw, which is how a
// theme of files it cannot read is kept out of the chooser rather than
// shown as a grid of blanks.
func RasterizeIconSVG(path string, size int) (*paintengine2d.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, err := io.ReadAll(io.LimitReader(f, maxSVGBytes+1))
	if err != nil {
		return nil, err
	}
	if len(src) > maxSVGBytes {
		return nil, svgUnsupported{"file (over 1 MiB)"}
	}
	return rasterizeSVGBytes(src, size)
}

func rasterizeSVGBytes(src []byte, size int) (*paintengine2d.Image, error) {
	if size < 1 {
		size = 1
	}
	if size > 512 {
		size = 512
	}
	doc, err := parseSVG(src)
	if err != nil {
		return nil, err
	}
	if len(doc.shapes) == 0 {
		return nil, svgUnsupported{"document (it draws nothing)"}
	}
	// The viewBox mapped onto a square, keeping the aspect ratio the way
	// the default preserveAspectRatio ("xMidYMid meet") does: an icon
	// authored 22 wide and 22 high in a box drawn 24 square must not
	// come out stretched, and a rare 16x22 one must not either.
	s := float32(size) / doc.w
	if t := float32(size) / doc.h; t < s {
		s = t
	}
	m := paintengine2d.Translation(
		(float32(size)-doc.w*s)/2-doc.x*s,
		(float32(size)-doc.h*s)/2-doc.y*s,
	).Mul(paintengine2d.Scaling(s, s))

	img := paintengine2d.NewImage(size, size)
	ctx := paintengine2d.NewContext(img)
	for _, sh := range doc.shapes {
		p := sh.path.Clone()
		p.Transform(m.Mul(sh.transform))
		if sh.paint.hasFill {
			ctx.DrawPath(p, paintengine2d.Paint{
				Color:     sh.paint.fill.WithAlpha(sh.paint.fill.A * sh.paint.opacity * sh.paint.fillOpacity),
				Style:     paintengine2d.StyleFill,
				FillRule:  sh.paint.fillRule,
				AntiAlias: true,
			})
		}
		if sh.paint.hasStroke && sh.paint.strokeWidth > 0 {
			w := sh.paint.strokeWidth * m.Mul(sh.transform).ApproxScale()
			if w < 0.35 {
				w = 0.35
			}
			ctx.DrawPath(p, paintengine2d.Paint{
				Color: sh.paint.stroke.WithAlpha(sh.paint.stroke.A * sh.paint.opacity * sh.paint.strokeAlpha),
				Style: paintengine2d.StyleStroke,
				Stroke: paintengine2d.Stroke{
					Width: w, Cap: sh.paint.cap, Join: sh.paint.join, MiterLimit: 4,
				},
				AntiAlias: true,
			})
		}
	}
	img.Touch()
	return img, nil
}

// ---- the document -------------------------------------------------------------

type svgShape struct {
	path      *paintengine2d.Path
	transform paintengine2d.Matrix
	paint     svgPaint
}

type svgDoc struct {
	x, y, w, h float32
	shapes     []svgShape
}

// parseSVG walks the file once, refusing at the first construct outside
// the subset.
func parseSVG(src []byte) (*svgDoc, error) {
	dec := xml.NewDecoder(strings.NewReader(string(src)))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity

	doc := &svgDoc{w: 0, h: 0}
	// The element stack's inherited state: one transform and one paint
	// per open <g> (and the <svg> itself).
	type frame struct {
		m paintengine2d.Matrix
		p svgPaint
	}
	stack := []frame{{m: paintengine2d.Identity(), p: defaultSVGPaint()}}
	// skip counts how deep inside an element whose contents paint
	// nothing (<defs>, <metadata>, <title>) the walk is.
	skip := 0
	seenSVG := false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("svg: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if skip > 0 {
				skip++
				continue
			}
			switch name {
			case "defs", "metadata", "title", "desc", "style", "sodipodi:namedview", "namedview", "rdf", "script":
				skip = 1
				continue
			// The constructs this rasterizer will not fake. A file that
			// uses one is not drawn at all — see the note at the head of
			// the file.
			case "use", "image", "text", "tspan", "mask", "clippath", "filter", "pattern", "marker",
				"switch", "foreignobject", "symbol", "textpath", "animate", "lineargradient", "radialgradient":
				return nil, svgUnsupported{"<" + name + ">"}
			}

			top := stack[len(stack)-1]
			f := frame{m: top.m, p: top.p}
			if tr := attr(t, "transform"); tr != "" {
				m, err := parseSVGTransform(tr)
				if err != nil {
					return nil, err
				}
				f.m = f.m.Mul(m)
			}
			if err := applySVGStyle(&f.p, t); err != nil {
				return nil, err
			}

			switch name {
			case "svg":
				if !seenSVG {
					seenSVG = true
					if err := readViewBox(doc, t); err != nil {
						return nil, err
					}
				}
			case "g", "a":
				// state only
			case "path", "rect", "circle", "ellipse", "polygon", "polyline", "line":
				p, err := shapePath(name, t)
				if err != nil {
					return nil, err
				}
				if p != nil && !p.Empty() {
					doc.shapes = append(doc.shapes, svgShape{path: p, transform: f.m, paint: f.p})
				}
			default:
				// An element nobody has to know about (inkscape's own,
				// a comment wrapper): its contents are not drawn, but it
				// is not a reason to refuse the file.
				skip = 1
				continue
			}
			stack = append(stack, f)
		case xml.EndElement:
			if skip > 0 {
				skip--
				continue
			}
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if !seenSVG {
		return nil, svgUnsupported{"document (no <svg>)"}
	}
	if doc.w <= 0 || doc.h <= 0 {
		return nil, svgUnsupported{"document (no viewBox or size)"}
	}
	return doc, nil
}

// readViewBox takes the user-space box from viewBox, or from width and
// height when there is none.
func readViewBox(doc *svgDoc, e xml.StartElement) error {
	if vb := attr(e, "viewBox"); vb != "" {
		n := numbers(vb)
		if len(n) != 4 || n[2] <= 0 || n[3] <= 0 {
			return svgUnsupported{"viewBox " + strconv.Quote(vb)}
		}
		doc.x, doc.y, doc.w, doc.h = n[0], n[1], n[2], n[3]
		return nil
	}
	doc.w = lengthAttr(e, "width", 0)
	doc.h = lengthAttr(e, "height", 0)
	return nil
}

// ---- shapes -------------------------------------------------------------------

func shapePath(name string, e xml.StartElement) (*paintengine2d.Path, error) {
	p := paintengine2d.NewPath()
	switch name {
	case "path":
		d := attr(e, "d")
		if strings.TrimSpace(d) == "" {
			return nil, nil
		}
		return p, parsePathData(p, d)
	case "rect":
		x, y := lengthAttr(e, "x", 0), lengthAttr(e, "y", 0)
		w, h := lengthAttr(e, "width", 0), lengthAttr(e, "height", 0)
		if w <= 0 || h <= 0 {
			return nil, nil
		}
		rx, ry := lengthAttr(e, "rx", -1), lengthAttr(e, "ry", -1)
		if rx < 0 {
			rx = ry
		}
		if ry < 0 {
			ry = rx
		}
		r := paintengine2d.XYWH(x, y, w, h)
		if rx > 0 && ry > 0 {
			p.AddRoundRect(r, min(rx, w/2), min(ry, h/2))
		} else {
			p.AddRect(r)
		}
		return p, nil
	case "circle":
		r := lengthAttr(e, "r", 0)
		if r <= 0 {
			return nil, nil
		}
		p.AddCircle(paintengine2d.Pt(lengthAttr(e, "cx", 0), lengthAttr(e, "cy", 0)), r)
		return p, nil
	case "ellipse":
		rx, ry := lengthAttr(e, "rx", 0), lengthAttr(e, "ry", 0)
		if rx <= 0 || ry <= 0 {
			return nil, nil
		}
		p.AddEllipse(paintengine2d.Pt(lengthAttr(e, "cx", 0), lengthAttr(e, "cy", 0)), rx, ry)
		return p, nil
	case "line":
		p.MoveTo(lengthAttr(e, "x1", 0), lengthAttr(e, "y1", 0))
		p.LineTo(lengthAttr(e, "x2", 0), lengthAttr(e, "y2", 0))
		return p, nil
	case "polygon", "polyline":
		n := numbers(attr(e, "points"))
		if len(n) < 4 {
			return nil, nil
		}
		p.MoveTo(n[0], n[1])
		for i := 2; i+1 < len(n); i += 2 {
			p.LineTo(n[i], n[i+1])
		}
		if name == "polygon" {
			p.Close()
		}
		return p, nil
	}
	return nil, nil
}

// ---- presentation attributes --------------------------------------------------

// applySVGStyle folds an element's presentation attributes and its
// style="" declarations into the inherited paint. style wins, the way
// the cascade says.
func applySVGStyle(p *svgPaint, e xml.StartElement) error {
	set := func(prop, val string) error { return applySVGProperty(p, prop, val) }
	for _, a := range e.Attr {
		switch strings.ToLower(a.Name.Local) {
		case "fill", "fill-rule", "fill-opacity", "stroke", "stroke-width",
			"stroke-opacity", "stroke-linecap", "stroke-linejoin", "opacity":
			if err := set(strings.ToLower(a.Name.Local), a.Value); err != nil {
				return err
			}
		}
	}
	for _, decl := range strings.Split(attr(e, "style"), ";") {
		k, v, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		if err := set(strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)); err != nil {
			return err
		}
	}
	return nil
}

func applySVGProperty(p *svgPaint, prop, val string) error {
	val = strings.TrimSpace(val)
	switch prop {
	case "fill":
		c, painted, err := parseSVGColor(val)
		if err != nil {
			return err
		}
		p.fill, p.hasFill = c, painted
	case "stroke":
		c, painted, err := parseSVGColor(val)
		if err != nil {
			return err
		}
		p.stroke, p.hasStroke = c, painted
	case "fill-rule":
		if strings.EqualFold(val, "evenodd") {
			p.fillRule = paintengine2d.FillEvenOdd
		} else {
			p.fillRule = paintengine2d.FillNonZero
		}
	case "fill-opacity":
		p.fillOpacity = unitFloat(val, p.fillOpacity)
	case "stroke-opacity":
		p.strokeAlpha = unitFloat(val, p.strokeAlpha)
	case "opacity":
		p.opacity = unitFloat(val, p.opacity)
	case "stroke-width":
		if v, err := strconv.ParseFloat(strings.TrimRight(val, "px "), 32); err == nil {
			p.strokeWidth = float32(v)
		}
	case "stroke-linecap":
		switch strings.ToLower(val) {
		case "round":
			p.cap = paintengine2d.CapRound
		case "square":
			p.cap = paintengine2d.CapSquare
		default:
			p.cap = paintengine2d.CapButt
		}
	case "stroke-linejoin":
		switch strings.ToLower(val) {
		case "round":
			p.join = paintengine2d.JoinRound
		case "bevel":
			p.join = paintengine2d.JoinBevel
		default:
			p.join = paintengine2d.JoinMiter
		}
	}
	return nil
}

// parseSVGColor reads a paint value. "none" paints nothing;
// "currentColor" is black here and becomes the look's foreground at draw
// time (see the note at the head of the file); url(#…) is a gradient or a
// pattern and is refused.
func parseSVGColor(v string) (paintengine2d.Color, bool, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	switch {
	case s == "" || s == "inherit":
		return paintengine2d.Black, true, nil
	case s == "none" || s == "transparent":
		return paintengine2d.Color{}, false, nil
	case s == "currentcolor":
		return paintengine2d.Black, true, nil
	case strings.HasPrefix(s, "url("):
		return paintengine2d.Color{}, false, svgUnsupported{"paint " + strconv.Quote(v)}
	case strings.HasPrefix(s, "#"):
		return parseHexColor(s)
	case strings.HasPrefix(s, "rgb("):
		n := numbers(strings.TrimSuffix(strings.TrimPrefix(s, "rgb("), ")"))
		if len(n) < 3 {
			return paintengine2d.Black, true, nil
		}
		return paintengine2d.RGB(n[0]/255, n[1]/255, n[2]/255), true, nil
	}
	if c, ok := svgNamedColors[s]; ok {
		return c, true, nil
	}
	// An unknown keyword is a colour this file does not know how to
	// read, not a reason to draw the shape in the wrong one.
	return paintengine2d.Black, true, nil
}

func parseHexColor(s string) (paintengine2d.Color, bool, error) {
	h := strings.TrimPrefix(s, "#")
	switch len(h) {
	case 3, 4:
		var v [4]float32
		v[3] = 1
		for i := 0; i < len(h); i++ {
			d, err := strconv.ParseUint(h[i:i+1], 16, 8)
			if err != nil {
				return paintengine2d.Black, true, nil
			}
			v[i] = float32(d*17) / 255
		}
		return paintengine2d.RGBA(v[0], v[1], v[2], v[3]), true, nil
	case 6, 8:
		var v [4]float32
		v[3] = 1
		for i := 0; i*2 < len(h); i++ {
			d, err := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
			if err != nil {
				return paintengine2d.Black, true, nil
			}
			v[i] = float32(d) / 255
		}
		return paintengine2d.RGBA(v[0], v[1], v[2], v[3]), true, nil
	}
	return paintengine2d.Black, true, nil
}

// svgNamedColors are the CSS keywords an icon theme actually uses.
var svgNamedColors = map[string]paintengine2d.Color{
	"black":     paintengine2d.Black,
	"white":     paintengine2d.White,
	"red":       paintengine2d.RGB(1, 0, 0),
	"green":     paintengine2d.RGB(0, 0.5, 0),
	"blue":      paintengine2d.RGB(0, 0, 1),
	"gray":      paintengine2d.RGB(0.5, 0.5, 0.5),
	"grey":      paintengine2d.RGB(0.5, 0.5, 0.5),
	"silver":    paintengine2d.RGB(0.75, 0.75, 0.75),
	"yellow":    paintengine2d.RGB(1, 1, 0),
	"orange":    paintengine2d.RGB(1, 0.647, 0),
	"darkgray":  paintengine2d.RGB(0.66, 0.66, 0.66),
	"darkgrey":  paintengine2d.RGB(0.66, 0.66, 0.66),
	"lightgray": paintengine2d.RGB(0.83, 0.83, 0.83),
	"lightgrey": paintengine2d.RGB(0.83, 0.83, 0.83),
}

// ---- transform ----------------------------------------------------------------

// parseSVGTransform reads a transform list: translate, scale, rotate,
// matrix, skewX and skewY.
func parseSVGTransform(s string) (paintengine2d.Matrix, error) {
	m := paintengine2d.Identity()
	rest := s
	for {
		rest = strings.TrimLeft(rest, " \t\r\n,")
		if rest == "" {
			return m, nil
		}
		open := strings.IndexByte(rest, '(')
		if open < 0 {
			return m, svgUnsupported{"transform " + strconv.Quote(s)}
		}
		close := strings.IndexByte(rest[open:], ')')
		if close < 0 {
			return m, svgUnsupported{"transform " + strconv.Quote(s)}
		}
		fn := strings.ToLower(strings.TrimSpace(rest[:open]))
		n := numbers(rest[open+1 : open+close])
		rest = rest[open+close+1:]
		var step paintengine2d.Matrix
		switch fn {
		case "translate":
			switch len(n) {
			case 1:
				step = paintengine2d.Translation(n[0], 0)
			case 2:
				step = paintengine2d.Translation(n[0], n[1])
			default:
				return m, svgUnsupported{"transform " + strconv.Quote(s)}
			}
		case "scale":
			switch len(n) {
			case 1:
				step = paintengine2d.Scaling(n[0], n[0])
			case 2:
				step = paintengine2d.Scaling(n[0], n[1])
			default:
				return m, svgUnsupported{"transform " + strconv.Quote(s)}
			}
		case "rotate":
			switch len(n) {
			case 1:
				step = paintengine2d.Rotation(n[0] * math.Pi / 180)
			case 3:
				step = paintengine2d.Translation(n[1], n[2]).
					Mul(paintengine2d.Rotation(n[0] * math.Pi / 180)).
					Mul(paintengine2d.Translation(-n[1], -n[2]))
			default:
				return m, svgUnsupported{"transform " + strconv.Quote(s)}
			}
		case "matrix":
			if len(n) != 6 {
				return m, svgUnsupported{"transform " + strconv.Quote(s)}
			}
			step = paintengine2d.Matrix{A: n[0], B: n[1], C: n[2], D: n[3], E: n[4], F: n[5]}
		case "skewx":
			if len(n) != 1 {
				return m, svgUnsupported{"transform " + strconv.Quote(s)}
			}
			step = paintengine2d.Matrix{A: 1, D: 1, C: float32(math.Tan(float64(n[0]) * math.Pi / 180))}
		case "skewy":
			if len(n) != 1 {
				return m, svgUnsupported{"transform " + strconv.Quote(s)}
			}
			step = paintengine2d.Matrix{A: 1, D: 1, B: float32(math.Tan(float64(n[0]) * math.Pi / 180))}
		default:
			return m, svgUnsupported{"transform " + strconv.Quote(fn)}
		}
		m = m.Mul(step)
	}
}

// ---- path data ----------------------------------------------------------------

// parsePathData reads the d attribute's grammar: M m L l H h V v C c S s
// Q q T t A a Z z. An arc becomes cubics (paintengine2d draws no arc
// segment of its own, and a themed icon's arcs are corners and circles,
// which cubics render exactly enough for a 24-pixel glyph).
func parsePathData(p *paintengine2d.Path, d string) error {
	sc := &numScanner{s: d}
	var cur, start paintengine2d.Point
	var lastCtrl paintengine2d.Point
	var lastCmd byte
	open := false
	for {
		sc.skipSep()
		if sc.eof() {
			break
		}
		c := sc.peek()
		var cmd byte
		if isPathCmd(c) {
			cmd = c
			sc.next()
		} else {
			if lastCmd == 0 {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			// A repeated argument list: M implies L, m implies l.
			switch lastCmd {
			case 'M':
				cmd = 'L'
			case 'm':
				cmd = 'l'
			default:
				cmd = lastCmd
			}
		}
		rel := cmd >= 'a' && cmd <= 'z'
		abs := func(x, y float32) paintengine2d.Point {
			if rel {
				return paintengine2d.Pt(cur.X+x, cur.Y+y)
			}
			return paintengine2d.Pt(x, y)
		}
		switch upper(cmd) {
		case 'M':
			n, ok := sc.nums(2)
			if !ok {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			cur = abs(n[0], n[1])
			start = cur
			p.MoveTo(cur.X, cur.Y)
			open = true
		case 'L':
			n, ok := sc.nums(2)
			if !ok {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			cur = abs(n[0], n[1])
			p.LineTo(cur.X, cur.Y)
		case 'H':
			n, ok := sc.nums(1)
			if !ok {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			if rel {
				cur.X += n[0]
			} else {
				cur.X = n[0]
			}
			p.LineTo(cur.X, cur.Y)
		case 'V':
			n, ok := sc.nums(1)
			if !ok {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			if rel {
				cur.Y += n[0]
			} else {
				cur.Y = n[0]
			}
			p.LineTo(cur.X, cur.Y)
		case 'C':
			n, ok := sc.nums(6)
			if !ok {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			c1, c2, end := abs(n[0], n[1]), abs(n[2], n[3]), abs(n[4], n[5])
			p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, end.X, end.Y)
			lastCtrl, cur = c2, end
		case 'S':
			n, ok := sc.nums(4)
			if !ok {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			c1 := cur
			if upper(lastCmd) == 'C' || upper(lastCmd) == 'S' {
				c1 = paintengine2d.Pt(2*cur.X-lastCtrl.X, 2*cur.Y-lastCtrl.Y)
			}
			c2, end := abs(n[0], n[1]), abs(n[2], n[3])
			p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, end.X, end.Y)
			lastCtrl, cur = c2, end
		case 'Q':
			n, ok := sc.nums(4)
			if !ok {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			c1, end := abs(n[0], n[1]), abs(n[2], n[3])
			p.QuadTo(c1.X, c1.Y, end.X, end.Y)
			lastCtrl, cur = c1, end
		case 'T':
			n, ok := sc.nums(2)
			if !ok {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			c1 := cur
			if upper(lastCmd) == 'Q' || upper(lastCmd) == 'T' {
				c1 = paintengine2d.Pt(2*cur.X-lastCtrl.X, 2*cur.Y-lastCtrl.Y)
			}
			end := abs(n[0], n[1])
			p.QuadTo(c1.X, c1.Y, end.X, end.Y)
			lastCtrl, cur = c1, end
		case 'A':
			n, ok := sc.nums(7)
			if !ok {
				return svgUnsupported{"path data " + strconv.Quote(shorten(d))}
			}
			end := abs(n[5], n[6])
			arcToCubics(p, cur, end, n[0], n[1], n[2], n[3] != 0, n[4] != 0)
			cur = end
		case 'Z':
			if open {
				p.Close()
			}
			cur = start
		default:
			return svgUnsupported{"path command " + strconv.Quote(string(cmd))}
		}
		if u := upper(cmd); u != 'C' && u != 'S' && u != 'Q' && u != 'T' {
			lastCtrl = cur
		}
		lastCmd = cmd
	}
	return nil
}

func isPathCmd(c byte) bool {
	switch upper(c) {
	case 'M', 'L', 'H', 'V', 'C', 'S', 'Q', 'T', 'A', 'Z':
		return true
	}
	return false
}

func upper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 32
	}
	return c
}

func shorten(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 48 {
		return s[:48] + "…"
	}
	return s
}

// arcToCubics appends the SVG elliptical arc from→to as cubic segments
// (the endpoint parameterisation of F.6.5 in the SVG specification).
func arcToCubics(p *paintengine2d.Path, from, to paintengine2d.Point, rx, ry, xrot float32, large, sweep bool) {
	if rx == 0 || ry == 0 || (from == to) {
		p.LineTo(to.X, to.Y)
		return
	}
	rx, ry = float32(math.Abs(float64(rx))), float32(math.Abs(float64(ry)))
	phi := float64(xrot) * math.Pi / 180
	sin, cos := math.Sincos(phi)
	dx, dy := float64(from.X-to.X)/2, float64(from.Y-to.Y)/2
	x1 := cos*dx + sin*dy
	y1 := -sin*dx + cos*dy
	rxf, ryf := float64(rx), float64(ry)
	// Scale the radii up when they are too small to reach (F.6.6).
	if lam := x1*x1/(rxf*rxf) + y1*y1/(ryf*ryf); lam > 1 {
		s := math.Sqrt(lam)
		rxf, ryf = rxf*s, ryf*s
	}
	num := rxf*rxf*ryf*ryf - rxf*rxf*y1*y1 - ryf*ryf*x1*x1
	den := rxf*rxf*y1*y1 + ryf*ryf*x1*x1
	if den == 0 {
		p.LineTo(to.X, to.Y)
		return
	}
	co := math.Sqrt(math.Max(0, num/den))
	if large == sweep {
		co = -co
	}
	cxp := co * rxf * y1 / ryf
	cyp := -co * ryf * x1 / rxf
	cx := cos*cxp - sin*cyp + float64(from.X+to.X)/2
	cy := sin*cxp + cos*cyp + float64(from.Y+to.Y)/2

	ang := func(ux, uy, vx, vy float64) float64 {
		d := math.Sqrt((ux*ux + uy*uy) * (vx*vx + vy*vy))
		if d == 0 {
			return 0
		}
		c := math.Max(-1, math.Min(1, (ux*vx+uy*vy)/d))
		a := math.Acos(c)
		if ux*vy-uy*vx < 0 {
			a = -a
		}
		return a
	}
	theta := ang(1, 0, (x1-cxp)/rxf, (y1-cyp)/ryf)
	delta := ang((x1-cxp)/rxf, (y1-cyp)/ryf, (-x1-cxp)/rxf, (-y1-cyp)/ryf)
	if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	} else if sweep && delta < 0 {
		delta += 2 * math.Pi
	}
	segs := int(math.Ceil(math.Abs(delta) / (math.Pi / 2)))
	if segs < 1 {
		segs = 1
	}
	step := delta / float64(segs)
	k := 4.0 / 3.0 * math.Tan(step/4)
	pt := func(a float64) (float64, float64) {
		ca, sa := math.Cos(a), math.Sin(a)
		return cx + rxf*ca*cos - ryf*sa*sin, cy + rxf*ca*sin + ryf*sa*cos
	}
	dpt := func(a float64) (float64, float64) {
		ca, sa := math.Cos(a), math.Sin(a)
		return -rxf*sa*cos - ryf*ca*sin, -rxf*sa*sin + ryf*ca*cos
	}
	a := theta
	for i := 0; i < segs; i++ {
		x0, y0 := pt(a)
		dx0, dy0 := dpt(a)
		a += step
		x3, y3 := pt(a)
		dx3, dy3 := dpt(a)
		p.CubicTo(
			float32(x0+k*dx0), float32(y0+k*dy0),
			float32(x3-k*dx3), float32(y3-k*dy3),
			float32(x3), float32(y3),
		)
	}
}

// ---- small readers ------------------------------------------------------------

func attr(e xml.StartElement, name string) string {
	for _, a := range e.Attr {
		if strings.EqualFold(a.Name.Local, name) {
			return a.Value
		}
	}
	return ""
}

func lengthAttr(e xml.StartElement, name string, def float32) float32 {
	v := strings.TrimSpace(attr(e, name))
	if v == "" {
		return def
	}
	v = strings.TrimSuffix(strings.TrimSuffix(v, "px"), "pt")
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 32)
	if err != nil {
		return def
	}
	return float32(f)
}

func unitFloat(s string, def float32) float32 {
	if strings.HasSuffix(s, "%") {
		if v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32); err == nil {
			return float32(v) / 100
		}
		return def
	}
	if v, err := strconv.ParseFloat(s, 32); err == nil {
		return float32(v)
	}
	return def
}

// numbers reads every number in s, whatever separates them.
func numbers(s string) []float32 {
	sc := &numScanner{s: s}
	var out []float32
	for {
		sc.skipSep()
		if sc.eof() {
			return out
		}
		v, ok := sc.num()
		if !ok {
			return out
		}
		out = append(out, v)
	}
}

// numScanner reads SVG's run-together number grammar: "10-3.5.25" is
// three numbers, and a comma, a space or neither may separate them.
type numScanner struct {
	s string
	i int
}

func (n *numScanner) eof() bool  { return n.i >= len(n.s) }
func (n *numScanner) peek() byte { return n.s[n.i] }
func (n *numScanner) next()      { n.i++ }

func (n *numScanner) skipSep() {
	for n.i < len(n.s) {
		switch n.s[n.i] {
		case ' ', '\t', '\r', '\n', ',':
			n.i++
		default:
			return
		}
	}
}

func (n *numScanner) num() (float32, bool) {
	start := n.i
	if n.i < len(n.s) && (n.s[n.i] == '+' || n.s[n.i] == '-') {
		n.i++
	}
	dot := false
	for n.i < len(n.s) {
		c := n.s[n.i]
		switch {
		case c >= '0' && c <= '9':
			n.i++
		case c == '.' && !dot:
			dot = true
			n.i++
		case (c == 'e' || c == 'E') && n.i > start:
			j := n.i + 1
			if j < len(n.s) && (n.s[j] == '+' || n.s[j] == '-') {
				j++
			}
			if j < len(n.s) && n.s[j] >= '0' && n.s[j] <= '9' {
				n.i = j
				continue
			}
			goto done
		default:
			goto done
		}
	}
done:
	if n.i == start {
		return 0, false
	}
	v, err := strconv.ParseFloat(n.s[start:n.i], 32)
	if err != nil {
		return 0, false
	}
	return float32(v), true
}

func (n *numScanner) nums(k int) ([]float32, bool) {
	out := make([]float32, 0, k)
	for len(out) < k {
		n.skipSep()
		v, ok := n.num()
		if !ok {
			return nil, false
		}
		out = append(out, v)
	}
	return out, true
}
