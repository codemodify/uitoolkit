package style

import (
	"bytes"
	"encoding/xml"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/codemodify/paintengine2d"
)

// svgDoc is a parsed icon: viewBox plus tintable shapes.
type svgDoc struct {
	vbX, vbY, vbW, vbH float32
	shapes             []svgShape
}

type svgShape struct {
	path    *paintengine2d.Path
	fill    bool
	stroke  bool
	strokeW float32
	cap     paintengine2d.Cap
	join    paintengine2d.Join
	opacity float32
	evenOdd bool
}

type svgStyle struct {
	fill       string
	stroke     string
	strokeW    float32
	opacity    float32
	fillOp     float32
	strokeOp   float32
	cap        paintengine2d.Cap
	join       paintengine2d.Join
	evenOdd    bool
	fillSet    bool
	strokeSet  bool
	strokeWSet bool
}

func defaultSVGStyle() svgStyle {
	return svgStyle{
		fill:     "currentColor",
		stroke:   "none",
		strokeW:  1,
		opacity:  1,
		fillOp:   1,
		strokeOp: 1,
		cap:      paintengine2d.CapButt,
		join:     paintengine2d.JoinMiter,
		fillSet:  true,
	}
}

func parseSVG(data []byte) (svgDoc, error) {
	doc := svgDoc{vbW: 24, vbH: 24}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }

	stack := []svgStyle{defaultSVGStyle()}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return svgDoc{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			st := stack[len(stack)-1]
			applySVGAttrs(&st, t.Attr)
			switch t.Name.Local {
			case "svg":
				if vb := attr(t.Attr, "viewBox"); vb != "" {
					doc.vbX, doc.vbY, doc.vbW, doc.vbH = parseViewBox(vb)
				}
				stack = append(stack, st)
			case "g":
				stack = append(stack, st)
			case "path":
				if sh, ok := shapeFromPath(attr(t.Attr, "d"), st); ok {
					doc.shapes = append(doc.shapes, sh)
				}
			case "circle":
				if sh, ok := shapeFromCircle(t.Attr, st); ok {
					doc.shapes = append(doc.shapes, sh)
				}
			case "ellipse":
				if sh, ok := shapeFromEllipse(t.Attr, st); ok {
					doc.shapes = append(doc.shapes, sh)
				}
			case "rect":
				if sh, ok := shapeFromRect(t.Attr, st); ok {
					doc.shapes = append(doc.shapes, sh)
				}
			case "line":
				if sh, ok := shapeFromLine(t.Attr, st); ok {
					doc.shapes = append(doc.shapes, sh)
				}
			case "polyline":
				if sh, ok := shapeFromPoly(attr(t.Attr, "points"), st, false); ok {
					doc.shapes = append(doc.shapes, sh)
				}
			case "polygon":
				if sh, ok := shapeFromPoly(attr(t.Attr, "points"), st, true); ok {
					doc.shapes = append(doc.shapes, sh)
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "svg", "g":
				if len(stack) > 1 {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}
	if doc.vbW <= 0 {
		doc.vbW = 24
	}
	if doc.vbH <= 0 {
		doc.vbH = 24
	}
	return doc, nil
}

func (d svgDoc) empty() bool { return len(d.shapes) == 0 }

func (d svgDoc) draw(ctx *paintengine2d.Context, dst paintengine2d.Rect, tint paintengine2d.Color) {
	if ctx == nil || d.empty() || dst.Empty() {
		return
	}
	sx := dst.Dx() / d.vbW
	sy := dst.Dy() / d.vbH
	s := sx
	if sy < sx {
		s = sy
	}
	if s <= 0 {
		return
	}
	ox := dst.Min.X + (dst.Dx()-d.vbW*s)*0.5
	oy := dst.Min.Y + (dst.Dy()-d.vbH*s)*0.5
	m := paintengine2d.Translation(ox, oy).Mul(paintengine2d.Scaling(s, s)).Mul(paintengine2d.Translation(-d.vbX, -d.vbY))
	for _, sh := range d.shapes {
		if sh.path == nil || sh.path.Empty() {
			continue
		}
		p := sh.path.Clone()
		p.Transform(m)
		col := tint.WithAlpha(tint.A * sh.opacity)
		if sh.fill {
			paint := paintengine2d.Fill(col)
			if sh.evenOdd {
				paint.FillRule = paintengine2d.FillEvenOdd
			}
			ctx.DrawPath(p, paint)
		}
		if sh.stroke && sh.strokeW > 0 {
			w := sh.strokeW * s
			paint := paintengine2d.Paint{
				Color: col,
				Style: paintengine2d.StyleStroke,
				Stroke: paintengine2d.Stroke{
					Width:      w,
					Cap:        sh.cap,
					Join:       sh.join,
					MiterLimit: 4,
				},
			}
			ctx.DrawPath(p, paint)
		}
	}
}

func applySVGAttrs(st *svgStyle, attrs []xml.Attr) {
	if s := attr(attrs, "style"); s != "" {
		for _, part := range strings.Split(s, ";") {
			k, v, ok := strings.Cut(part, ":")
			if !ok {
				continue
			}
			applySVGAttr(st, strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}
	for _, a := range attrs {
		applySVGAttr(st, a.Name.Local, strings.TrimSpace(a.Value))
	}
}

func applySVGAttr(st *svgStyle, key, val string) {
	if val == "" {
		return
	}
	switch strings.ToLower(key) {
	case "fill":
		st.fill = val
		st.fillSet = true
	case "stroke":
		st.stroke = val
		st.strokeSet = true
	case "stroke-width":
		if n, ok := parseFloat(val); ok {
			st.strokeW = n
			st.strokeWSet = true
		}
	case "opacity":
		if n, ok := parseFloat(val); ok {
			st.opacity = n
		}
	case "fill-opacity":
		if n, ok := parseFloat(val); ok {
			st.fillOp = n
		}
	case "stroke-opacity":
		if n, ok := parseFloat(val); ok {
			st.strokeOp = n
		}
	case "stroke-linecap":
		switch val {
		case "round":
			st.cap = paintengine2d.CapRound
		case "square":
			st.cap = paintengine2d.CapSquare
		default:
			st.cap = paintengine2d.CapButt
		}
	case "stroke-linejoin":
		switch val {
		case "round":
			st.join = paintengine2d.JoinRound
		case "bevel":
			st.join = paintengine2d.JoinBevel
		default:
			st.join = paintengine2d.JoinMiter
		}
	case "fill-rule":
		st.evenOdd = val == "evenodd"
	}
}

func paintFlags(st svgStyle) (fill, stroke bool, opacity float32) {
	fill = st.fillSet && !paintNone(st.fill)
	stroke = st.strokeSet && !paintNone(st.stroke)
	if !st.fillSet && !st.strokeSet {
		fill = true
	}
	opacity = st.opacity
	if opacity <= 0 {
		opacity = 1
	}
	if fill {
		opacity *= st.fillOp
	} else if stroke {
		opacity *= st.strokeOp
	}
	if opacity <= 0 {
		opacity = 1
	}
	return fill, stroke, opacity
}

func paintNone(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "none" || s == "transparent"
}

func makeShape(path *paintengine2d.Path, st svgStyle) (svgShape, bool) {
	if path == nil || path.Empty() {
		return svgShape{}, false
	}
	fill, stroke, op := paintFlags(st)
	if !fill && !stroke {
		return svgShape{}, false
	}
	w := st.strokeW
	if w <= 0 {
		w = 1
	}
	return svgShape{
		path:    path,
		fill:    fill,
		stroke:  stroke,
		strokeW: w,
		cap:     st.cap,
		join:    st.join,
		opacity: op,
		evenOdd: st.evenOdd,
	}, true
}

func shapeFromPath(d string, st svgStyle) (svgShape, bool) {
	p := parsePathData(d)
	return makeShape(p, st)
}

func shapeFromCircle(attrs []xml.Attr, st svgStyle) (svgShape, bool) {
	cx := attrF(attrs, "cx")
	cy := attrF(attrs, "cy")
	r := attrF(attrs, "r")
	if r <= 0 {
		return svgShape{}, false
	}
	p := paintengine2d.NewPath()
	p.AddCircle(paintengine2d.Pt(cx, cy), r)
	return makeShape(p, st)
}

func shapeFromEllipse(attrs []xml.Attr, st svgStyle) (svgShape, bool) {
	cx := attrF(attrs, "cx")
	cy := attrF(attrs, "cy")
	rx := attrF(attrs, "rx")
	ry := attrF(attrs, "ry")
	if rx <= 0 || ry <= 0 {
		return svgShape{}, false
	}
	p := paintengine2d.NewPath()
	p.AddEllipse(paintengine2d.Pt(cx, cy), rx, ry)
	return makeShape(p, st)
}

func shapeFromRect(attrs []xml.Attr, st svgStyle) (svgShape, bool) {
	x := attrF(attrs, "x")
	y := attrF(attrs, "y")
	w := attrF(attrs, "width")
	h := attrF(attrs, "height")
	if w <= 0 || h <= 0 {
		return svgShape{}, false
	}
	rx := attrF(attrs, "rx")
	ry := attrF(attrs, "ry")
	if ry <= 0 {
		ry = rx
	}
	if rx <= 0 {
		rx = ry
	}
	p := paintengine2d.NewPath()
	r := paintengine2d.XYWH(x, y, w, h)
	if rx > 0 || ry > 0 {
		p.AddRoundRect(r, rx, ry)
	} else {
		p.AddRect(r)
	}
	return makeShape(p, st)
}

func shapeFromLine(attrs []xml.Attr, st svgStyle) (svgShape, bool) {
	p := paintengine2d.NewPath()
	p.MoveTo(attrF(attrs, "x1"), attrF(attrs, "y1"))
	p.LineTo(attrF(attrs, "x2"), attrF(attrs, "y2"))
	line := st
	if !line.strokeSet {
		line.stroke = "currentColor"
		line.strokeSet = true
		line.fill = "none"
		line.fillSet = true
	}
	return makeShape(p, line)
}

func shapeFromPoly(points string, st svgStyle, close bool) (svgShape, bool) {
	nums := pathNumbers(points)
	if len(nums) < 4 {
		return svgShape{}, false
	}
	p := paintengine2d.NewPath()
	p.MoveTo(nums[0], nums[1])
	for i := 2; i+1 < len(nums); i += 2 {
		p.LineTo(nums[i], nums[i+1])
	}
	if close {
		p.Close()
	}
	return makeShape(p, st)
}

func attr(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func attrF(attrs []xml.Attr, name string) float32 {
	n, _ := parseFloat(attr(attrs, name))
	return n
}

func parseViewBox(s string) (x, y, w, h float32) {
	n := pathNumbers(s)
	if len(n) >= 4 {
		return n[0], n[1], n[2], n[3]
	}
	return 0, 0, 24, 24
}

func parseFloat(s string) (float32, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if i := strings.IndexAny(s, " \t,"); i >= 0 {
		s = s[:i]
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, false
	}
	return float32(v), true
}

func parsePathData(d string) *paintengine2d.Path {
	p := paintengine2d.NewPath()
	if strings.TrimSpace(d) == "" {
		return p
	}
	pen := &pathPen{p: p}
	nums := []float32{}
	cmd := byte(0)
	i := 0
	for i < len(d) {
		c := d[i]
		if c == ',' || unicode.IsSpace(rune(c)) {
			i++
			continue
		}
		if isPathCmd(c) {
			flushPathCmd(pen, cmd, nums)
			nums = nums[:0]
			cmd = c
			i++
			if cmd == 'Z' || cmd == 'z' {
				flushPathCmd(pen, cmd, nil)
				cmd = 0
			}
			continue
		}
		n, next, ok := readPathNumber(d, i)
		if !ok {
			i++
			continue
		}
		nums = append(nums, n)
		i = next
	}
	flushPathCmd(pen, cmd, nums)
	return p
}

func isPathCmd(c byte) bool {
	return strings.ContainsRune("MmLlHhVvCcSsQqTtAaZz", rune(c))
}

func readPathNumber(s string, i int) (float32, int, bool) {
	start := i
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	seenDigit := false
	seenDot := false
	seenExp := false
	for i < len(s) {
		c := s[i]
		if c >= '0' && c <= '9' {
			seenDigit = true
			i++
			continue
		}
		if c == '.' && !seenDot && !seenExp {
			seenDot = true
			i++
			continue
		}
		if (c == 'e' || c == 'E') && seenDigit && !seenExp {
			seenExp = true
			i++
			if i < len(s) && (s[i] == '+' || s[i] == '-') {
				i++
			}
			continue
		}
		break
	}
	if !seenDigit || i == start {
		return 0, start, false
	}
	v, err := strconv.ParseFloat(s[start:i], 32)
	if err != nil {
		return 0, start, false
	}
	return float32(v), i, true
}

func pathNumbers(s string) []float32 {
	var out []float32
	i := 0
	for i < len(s) {
		n, next, ok := readPathNumber(s, i)
		if !ok {
			i++
			continue
		}
		out = append(out, n)
		i = next
	}
	return out
}

type pathPen struct {
	p         *paintengine2d.Path
	x, y      float32
	sx, sy    float32
	cx, cy    float32 // last control (for S/T)
	has       bool
	hasCtrl   bool
	cubicCtrl bool
}

func (pen *pathPen) move(x, y float32) {
	pen.p.MoveTo(x, y)
	pen.x, pen.y = x, y
	pen.sx, pen.sy = x, y
	pen.has = true
	pen.hasCtrl = false
}

func (pen *pathPen) line(x, y float32) {
	if !pen.has {
		pen.move(x, y)
		return
	}
	pen.p.LineTo(x, y)
	pen.x, pen.y = x, y
	pen.hasCtrl = false
}

func (pen *pathPen) cubic(x1, y1, x2, y2, x, y float32) {
	if !pen.has {
		pen.move(x1, y1)
	}
	pen.p.CubicTo(x1, y1, x2, y2, x, y)
	pen.x, pen.y = x, y
	pen.cx, pen.cy = x2, y2
	pen.hasCtrl = true
	pen.cubicCtrl = true
}

func (pen *pathPen) quad(x1, y1, x, y float32) {
	if !pen.has {
		pen.move(x1, y1)
	}
	pen.p.QuadTo(x1, y1, x, y)
	pen.x, pen.y = x, y
	pen.cx, pen.cy = x1, y1
	pen.hasCtrl = true
	pen.cubicCtrl = false
}

func (pen *pathPen) close() {
	if !pen.has {
		return
	}
	pen.p.Close()
	pen.x, pen.y = pen.sx, pen.sy
	pen.hasCtrl = false
}

func (pen *pathPen) reflect() (float32, float32) {
	if !pen.hasCtrl {
		return pen.x, pen.y
	}
	return 2*pen.x - pen.cx, 2*pen.y - pen.cy
}

func flushPathCmd(pen *pathPen, cmd byte, n []float32) {
	if cmd == 0 || pen == nil {
		return
	}
	rel := cmd >= 'a'
	switch cmd {
	case 'M', 'm':
		for i := 0; i+1 < len(n); i += 2 {
			x, y := n[i], n[i+1]
			if rel {
				x += pen.x
				y += pen.y
			}
			if i == 0 {
				pen.move(x, y)
			} else {
				pen.line(x, y)
			}
		}
	case 'L', 'l':
		for i := 0; i+1 < len(n); i += 2 {
			x, y := n[i], n[i+1]
			if rel {
				x += pen.x
				y += pen.y
			}
			pen.line(x, y)
		}
	case 'H', 'h':
		for _, x := range n {
			if rel {
				x += pen.x
			}
			pen.line(x, pen.y)
		}
	case 'V', 'v':
		for _, y := range n {
			if rel {
				y += pen.y
			}
			pen.line(pen.x, y)
		}
	case 'C', 'c':
		for i := 0; i+5 < len(n); i += 6 {
			x1, y1, x2, y2, x, y := n[i], n[i+1], n[i+2], n[i+3], n[i+4], n[i+5]
			if rel {
				x1 += pen.x
				y1 += pen.y
				x2 += pen.x
				y2 += pen.y
				x += pen.x
				y += pen.y
			}
			pen.cubic(x1, y1, x2, y2, x, y)
		}
	case 'S', 's':
		for i := 0; i+3 < len(n); i += 4 {
			x2, y2, x, y := n[i], n[i+1], n[i+2], n[i+3]
			if rel {
				x2 += pen.x
				y2 += pen.y
				x += pen.x
				y += pen.y
			}
			x1, y1 := pen.reflect()
			if pen.hasCtrl && !pen.cubicCtrl {
				x1, y1 = pen.x, pen.y
			}
			pen.cubic(x1, y1, x2, y2, x, y)
		}
	case 'Q', 'q':
		for i := 0; i+3 < len(n); i += 4 {
			x1, y1, x, y := n[i], n[i+1], n[i+2], n[i+3]
			if rel {
				x1 += pen.x
				y1 += pen.y
				x += pen.x
				y += pen.y
			}
			pen.quad(x1, y1, x, y)
		}
	case 'T', 't':
		for i := 0; i+1 < len(n); i += 2 {
			x, y := n[i], n[i+1]
			if rel {
				x += pen.x
				y += pen.y
			}
			x1, y1 := pen.reflect()
			if pen.hasCtrl && pen.cubicCtrl {
				x1, y1 = pen.x, pen.y
			}
			pen.quad(x1, y1, x, y)
		}
	case 'A', 'a':
		for i := 0; i+6 < len(n); i += 7 {
			rx, ry, rot := n[i], n[i+1], n[i+2]
			large := n[i+3] != 0
			sweep := n[i+4] != 0
			x, y := n[i+5], n[i+6]
			if rel {
				x += pen.x
				y += pen.y
			}
			addSVGArc(pen, rx, ry, rot, large, sweep, x, y)
		}
	case 'Z', 'z':
		pen.close()
	}
}

func addSVGArc(pen *pathPen, rx, ry, phiDeg float32, large, sweep bool, x2, y2 float32) {
	x1, y1 := pen.x, pen.y
	if !pen.has {
		pen.move(x2, y2)
		return
	}
	if math.Abs(float64(x1-x2)) < 1e-6 && math.Abs(float64(y1-y2)) < 1e-6 {
		return
	}
	rx = abs32(rx)
	ry = abs32(ry)
	if rx < 1e-6 || ry < 1e-6 {
		pen.line(x2, y2)
		return
	}
	phi := float64(phiDeg) * math.Pi / 180
	cosφ, sinφ := math.Cos(phi), math.Sin(phi)
	dx := float64(x1-x2) / 2
	dy := float64(y1-y2) / 2
	x1p := cosφ*dx + sinφ*dy
	y1p := -sinφ*dx + cosφ*dy
	rx64, ry64 := float64(rx), float64(ry)
	lambda := (x1p*x1p)/(rx64*rx64) + (y1p*y1p)/(ry64*ry64)
	if lambda > 1 {
		s := math.Sqrt(lambda)
		rx64 *= s
		ry64 *= s
	}
	sq := (rx64*rx64*ry64*ry64 - rx64*rx64*y1p*y1p - ry64*ry64*x1p*x1p) /
		(rx64*rx64*y1p*y1p + ry64*ry64*x1p*x1p)
	if sq < 0 {
		sq = 0
	}
	q := math.Sqrt(sq)
	if large == sweep {
		q = -q
	}
	cxp := q * rx64 * y1p / ry64
	cyp := q * -ry64 * x1p / rx64
	cx := cosφ*cxp - sinφ*cyp + float64(x1+x2)/2
	cy := sinφ*cxp + cosφ*cyp + float64(y1+y2)/2
	θ1 := angle(1, 0, (x1p-cxp)/rx64, (y1p-cyp)/ry64)
	dθ := angle((x1p-cxp)/rx64, (y1p-cyp)/ry64, (-x1p-cxp)/rx64, (-y1p-cyp)/ry64)
	if !sweep && dθ > 0 {
		dθ -= 2 * math.Pi
	} else if sweep && dθ < 0 {
		dθ += 2 * math.Pi
	}
	seg := int(math.Ceil(math.Abs(dθ) / (math.Pi / 2)))
	if seg < 1 {
		seg = 1
	}
	if seg > 8 {
		seg = 8
	}
	for i := 0; i < seg; i++ {
		a1 := θ1 + dθ*float64(i)/float64(seg)
		a2 := θ1 + dθ*float64(i+1)/float64(seg)
		arcSegmentToCubic(pen, cx, cy, rx64, ry64, cosφ, sinφ, a1, a2)
	}
}

func angle(ux, uy, vx, vy float64) float64 {
	dot := ux*vx + uy*vy
	n := math.Hypot(ux, uy) * math.Hypot(vx, vy)
	if n == 0 {
		return 0
	}
	c := dot / n
	if c > 1 {
		c = 1
	} else if c < -1 {
		c = -1
	}
	a := math.Acos(c)
	if ux*vy-uy*vx < 0 {
		a = -a
	}
	return a
}

func arcSegmentToCubic(pen *pathPen, cx, cy, rx, ry, cosφ, sinφ, a1, a2 float64) {
	da := a2 - a1
	alpha := math.Sin(da) * (math.Sqrt(4+3*math.Tan(da/2)*math.Tan(da/2)) - 1) / 3
	e1x, e1y := ellipsePoint(cx, cy, rx, ry, cosφ, sinφ, a1)
	e2x, e2y := ellipsePoint(cx, cy, rx, ry, cosφ, sinφ, a2)
	t1x, t1y := ellipseTangent(rx, ry, cosφ, sinφ, a1)
	t2x, t2y := ellipseTangent(rx, ry, cosφ, sinφ, a2)
	pen.cubic(
		float32(e1x+alpha*t1x), float32(e1y+alpha*t1y),
		float32(e2x-alpha*t2x), float32(e2y-alpha*t2y),
		float32(e2x), float32(e2y),
	)
}

func ellipsePoint(cx, cy, rx, ry, cosφ, sinφ, a float64) (x, y float64) {
	ca, sa := math.Cos(a), math.Sin(a)
	x = cx + rx*ca*cosφ - ry*sa*sinφ
	y = cy + rx*ca*sinφ + ry*sa*cosφ
	return
}

func ellipseTangent(rx, ry, cosφ, sinφ, a float64) (x, y float64) {
	ca, sa := math.Cos(a), math.Sin(a)
	x = -rx*sa*cosφ - ry*ca*sinφ
	y = -rx*sa*sinφ + ry*ca*cosφ
	return
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
