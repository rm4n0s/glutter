package svg

import (
	"math"
	"strings"

	"github.com/rm4n0s/glutter/ui"
)

// The limits that keep a filter from taking time and memory without bound.
const (
	// maxFilterPixels is the most pixels of the buffers of one filter: a larger filter region is worked on at a
	// lower resolution and scaled up.
	maxFilterPixels = 1 << 22
	// maxFilterPrimitives is the most primitives of one filter; a filter with more is not drawn.
	maxFilterPrimitives = 100
	// maxFilterWork is the work one filter may do, in pixels times the weight of each primitive; a filter that
	// needs more is not drawn.
	maxFilterWork = 1 << 28
	// maxFilterDepth is how many filtered elements may be drawn inside the source graphic of each other.
	maxFilterDepth = 4
)

// supportedPrimitives are the filter primitives that are drawn. A filter with any other primitive is drawn as if the
// element had no filter (SPEC §26.3).
var supportedPrimitives = map[string]bool{
	"feGaussianBlur": true, "feOffset": true, "feFlood": true, "feColorMatrix": true, "feComposite": true,
	"feMerge": true, "feBlend": true, "feMorphology": true, "feDropShadow": true, "feComponentTransfer": true,
}

// filterSupported reports whether every primitive of the filter is one that is drawn.
func filterSupported(f *Filter) bool {
	for _, pr := range f.Primitives {
		if !supportedPrimitives[pr.Name] {
			return false
		}
	}
	return true
}

// effectsApply reports whether a node's mask and filters can be drawn; if not, the element is left out, as the
// specification says of a mask or filter that is in error: a box with no area where the units need one, a mask that
// refers to itself, a filter with no primitives or with too many.
//
// SVG 1.1: masking.html#EstablishingANewMask, filters.html#FilterElement
func (p *painter) effectsApply(n *Node) bool {
	if n.Mask != nil && !p.maskUsable(n.Mask, n) {
		return false
	}
	for _, f := range n.Filters {
		if !filterSupported(f) {
			continue
		}
		if len(f.Primitives) == 0 || len(f.Primitives) > maxFilterPrimitives || p.active[f] {
			return false
		}
		if f.Units == UnitsObjectBoundingBox || f.PrimitiveUnits == UnitsObjectBoundingBox {
			if _, ok := p.objectBBox(n); !ok {
				return false
			}
		}
	}
	return true
}

// drawFiltered draws the content of a node through its filters, in order: each filter takes what the one before it
// made as its source graphic. A filter that has a primitive that is not drawn is skipped.
//
// SVG 1.1: filters.html#FilterProperty
func (p *painter) drawFiltered(n *Node, filters []*Filter, a float64) {
	var fs []*Filter
	for _, f := range filters {
		if filterSupported(f) {
			fs = append(fs, f)
		}
	}
	if len(fs) == 0 {
		p.drawContent(n, a)
		return
	}
	p.runFilters(n, fs, a)
}

// runFilters draws the content of n through the filters fs, the last of which draws on the canvas.
func (p *painter) runFilters(n *Node, fs []*Filter, a float64) {
	last := fs[len(fs)-1]
	src := func(q *painter) {
		if len(fs) == 1 {
			q.drawContent(n, a)
		} else {
			q.runFilters(n, fs[:len(fs)-1], a)
		}
	}
	p.runFilter(n, last, src)
}

// filterSpace is the geometry of a filter being run: the buffer of pixels that stands for the filter region, and how
// the user space of the element maps to it.
type filterSpace struct {
	w, h int
	// m maps the user space of the element to the pixels of the buffer.
	m Affine
	// sx and sy are the scales of m along the x and y axes of the user space.
	sx, sy float64
	// bbox is the object bounding box of the element.
	bbox   ui.Rect
	hasBox bool
	// region is the filter region in the user space of the element.
	region ui.Rect
	// vw and vh are what percentages refer to.
	vw, vh float64
	// primObb is true for primitiveUnits="objectBoundingBox".
	primObb bool
	// work is the work done so far.
	work int
}

// runFilter draws the node's content through one filter: it renders the source graphic into a buffer at the device
// scale, runs the primitives on the buffer, and draws the result as an image. The whole graph runs on the CPU on
// 8-bit premultiplied pixels, so that it gives the same image on every renderer.
//
// SVG 1.1: filters.html#FilterEffectsRegion
func (p *painter) runFilter(n *Node, f *Filter, src func(q *painter)) {
	if p.effectDepth >= maxFilterDepth || p.active[f] {
		return
	}
	p.active[f] = true
	defer delete(p.active, f)
	bbox, hasBox := p.objectBBox(n)
	fs := &filterSpace{bbox: bbox, hasBox: hasBox, vw: p.vw, vh: p.vh, primObb: f.PrimitiveUnits == UnitsObjectBoundingBox}
	if f.Units == UnitsObjectBoundingBox {
		if !hasBox {
			return
		}
		fs.region = ui.NewRectFromLTWH(bbox.Left+f.X.Resolve(1)*bbox.Width(), bbox.Top+f.Y.Resolve(1)*bbox.Height(),
			f.Width.Resolve(1)*bbox.Width(), f.Height.Resolve(1)*bbox.Height())
	} else {
		fs.region = ui.NewRectFromLTWH(f.X.Resolve(p.vw), f.Y.Resolve(p.vh), f.Width.Resolve(p.vw), f.Height.Resolve(p.vh))
	}
	if !(fs.region.Width() > 0 && fs.region.Height() > 0) || math.IsInf(fs.region.Width(), 0) || math.IsInf(fs.region.Height(), 0) {
		return
	}
	toDevice := Scale(p.scale, p.scale).Mul(p.ctm)
	var box bounds
	box.addTransformed(fs.region, toDevice)
	bw, bh := box.r.Width(), box.r.Height()
	if !(bw > 0 && bh > 0) || math.IsInf(bw, 0) || math.IsInf(bh, 0) || math.IsNaN(bw) || math.IsNaN(bh) {
		return
	}
	k := 1.0
	if bw*bh > maxFilterPixels {
		k = math.Sqrt(maxFilterPixels / (bw * bh))
	}
	ox, oy := math.Floor(box.r.Left*k), math.Floor(box.r.Top*k)
	fs.w = max(int(math.Ceil(box.r.Right*k-ox)), 1)
	fs.h = max(int(math.Ceil(box.r.Bottom*k-oy)), 1)
	for fs.w*fs.h > maxFilterPixels {
		// Rounding up can take the size just over the limit.
		k *= 0.99
		ox, oy = math.Floor(box.r.Left*k), math.Floor(box.r.Top*k)
		fs.w = max(int(math.Ceil(box.r.Right*k-ox)), 1)
		fs.h = max(int(math.Ceil(box.r.Bottom*k-oy)), 1)
	}
	fs.m = Translate(-ox, -oy).Mul(Scale(k, k)).Mul(toDevice)
	inv, ok := fs.m.Invert()
	if !ok {
		return
	}
	fs.sx, fs.sy = math.Hypot(fs.m.A, fs.m.B), math.Hypot(fs.m.C, fs.m.D)

	// The source graphic.
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	q := p.child(c)
	q.scale = 1
	q.effectDepth = p.effectDepth + 1
	c.Transform(fs.m.Matrix4())
	q.ctm = fs.m
	src(q)
	img, err := rec.EndRecording().ToImage(fs.w, fs.h, ui.ToImageOpts{Rasterizer: p.ras})
	if err != nil {
		return
	}
	source := newPixbuf(fs.w, fs.h)
	copy(source.pix, img.Pixels())

	out := runPrimitives(f, fs, source)
	if out == nil {
		return
	}
	res := out.inSpace(false)
	result := ui.NewImageFromRGBA(res.pix, fs.w, fs.h)
	p.save()
	defer p.restore()
	if math.Abs(fs.m.B) > 1e-9 || math.Abs(fs.m.C) > 1e-9 {
		// The buffer is the box of the region on the device: with a turned or skewed region it holds more than the
		// region, which the clip cuts off. An upright region is the buffer itself, in whole pixels.
		p.c.ClipRect(fs.region)
	}
	p.concat(inv)
	paint := ui.NewPaint()
	paint.SetFilterQuality(ui.FilterQualityLow)
	p.c.DrawImage(result, ui.Offset{}, paint)
}

// primitive weights, by name, for the work budget: a blur or a morphology costs more than a flood.
var primitiveWeight = map[string]int{
	"feGaussianBlur": 12, "feMorphology": 8, "feDropShadow": 14, "feBlend": 4, "feComposite": 3, "feColorMatrix": 3,
	"feComponentTransfer": 3, "feMerge": 2, "feOffset": 1, "feFlood": 1,
}

// runPrimitives runs the graph of primitives over the source graphic and returns the result of the last, or nil if
// the filter could not be run (it is over a limit).
func runPrimitives(f *Filter, fs *filterSpace, source *pixbuf) *pixbuf {
	sourceAlpha := source.alphaOnly()
	results := map[string]*pixbuf{}
	// lastUse is the last primitive that names each result, so that a result is dropped when nobody asks for it
	// again.
	lastUse := map[string]int{}
	for i, pr := range f.Primitives {
		for _, name := range primitiveInputs(pr) {
			lastUse[name] = i
		}
	}
	var prev *pixbuf
	for i, pr := range f.Primitives {
		fs.work += fs.w * fs.h * primitiveWeight[pr.Name]
		if fs.work > maxFilterWork {
			return nil
		}
		// input resolves an input name; std says whether it is a standard input, whose subregion is the whole
		// filter region.
		input := func(name string) (*pixbuf, bool) {
			switch name {
			case "SourceGraphic":
				return source, true
			case "SourceAlpha":
				return sourceAlpha, true
			case "BackgroundImage", "BackgroundAlpha", "FillPaint", "StrokePaint":
				return newPixbuf(fs.w, fs.h), true
			case "":
				if prev == nil {
					return source, true
				}
				return prev, false
			}
			if r := results[name]; r != nil {
				return r, false
			}
			if prev == nil {
				return source, true
			}
			return prev, false
		}
		linear := pr.Style == nil || pr.Style.ColorInterpolationFilters == ColorInterpolationLinearRGB
		var in, in2 *pixbuf
		sub := fullRect(fs)
		var inputs []*pixbuf
		var stds []bool
		get := func(name string) *pixbuf {
			b, std := input(name)
			inputs, stds = append(inputs, b), append(stds, std)
			return b.inSpace(linear)
		}
		var out *pixbuf
		switch pr.Name {
		case "feGaussianBlur":
			in = get(pr.In)
			sdx, sdy, ok := fs.stdDeviation(pr)
			if !ok {
				return nil
			}
			if sdx == 0 && sdy == 0 {
				out = in.clone()
			} else {
				out = in.blurred(sdx, sdy)
			}
		case "feOffset":
			in = get(pr.In)
			dx, dy := fs.offsetOf(pr, attrNumber(pr, "dx", 0), attrNumber(pr, "dy", 0))
			out = in.offset(int(math.Round(dx)), int(math.Round(dy)))
		case "feFlood":
			col := pr.FloodColor
			r, g, b := col.Red(), col.Green(), col.Blue()
			if linear {
				r, g, b = int(srgbToLinearLUT[r]), int(srgbToLinearLUT[g]), int(srgbToLinearLUT[b])
			}
			// The color's own alpha counts with the flood opacity.
			opacity := pr.FloodOpacity * float64(col.Alpha()) / 255
			out = flooded(fs.w, fs.h, uint8(r), uint8(g), uint8(b), opacity, 0, 0, fs.w, fs.h)
			out.linear = linear
		case "feColorMatrix":
			in = get(pr.In)
			out = in.colorMatrix(colorMatrixOf(pr))
		case "feComposite":
			in, in2 = get(pr.In), get(pr.In2)
			op := strings.TrimSpace(pr.Attrs["operator"])
			switch op {
			case "in", "out", "atop", "xor", "arithmetic":
			default:
				op = "over"
			}
			k := [4]float64{attrNumber(pr, "k1", 0), attrNumber(pr, "k2", 0), attrNumber(pr, "k3", 0), attrNumber(pr, "k4", 0)}
			out = compositeImages(in, in2, op, k)
		case "feMerge":
			var layers []*pixbuf
			for _, ch := range pr.Children {
				if ch.Name == "feMergeNode" {
					layers = append(layers, get(ch.In))
				}
			}
			if len(layers) == 0 {
				// A merge with no node is the empty image.
				get("")
			}
			out = mergeImages(fs.w, fs.h, layers)
			out.linear = linear
		case "feBlend":
			in, in2 = get(pr.In), get(pr.In2)
			out = blendImages(in, in2, strings.TrimSpace(pr.Attrs["mode"]))
		case "feMorphology":
			in = get(pr.In)
			rx, ry := fs.radiusOf(pr)
			switch {
			case rx < 0 || ry < 0:
				out = newPixbuf(fs.w, fs.h)
				out.linear = linear
			case rx == 0 && ry == 0:
				out = in.clone()
			default:
				out = in.morphology(rx, ry, strings.TrimSpace(pr.Attrs["operator"]) == "dilate")
			}
		case "feDropShadow":
			in = get(pr.In)
			sdx, sdy, ok := fs.stdDeviation(pr)
			if !ok {
				return nil
			}
			dx, dy := fs.offsetOf(pr, attrNumber(pr, "dx", 2), attrNumber(pr, "dy", 2))
			shadow := in.alphaOnly()
			if sdx > 0 || sdy > 0 {
				shadow = shadow.blurred(sdx, sdy)
			}
			shadow = shadow.offset(int(math.Round(dx)), int(math.Round(dy)))
			col := pr.FloodColor
			r, g, b := col.Red(), col.Green(), col.Blue()
			if linear {
				r, g, b = int(srgbToLinearLUT[r]), int(srgbToLinearLUT[g]), int(srgbToLinearLUT[b])
			}
			shadow = shadow.colored(uint8(r), uint8(g), uint8(b))
			if op := pr.FloodOpacity * float64(col.Alpha()) / 255; op < 1 {
				for j := 0; j < len(shadow.pix); j++ {
					shadow.pix[j] = uint8(math.Round(float64(shadow.pix[j]) * op))
				}
			}
			out = mergeImages(fs.w, fs.h, []*pixbuf{shadow, in})
			out.linear = linear
		case "feComponentTransfer":
			in = get(pr.In)
			out = in.transfer(transferFuncs(pr))
		}
		if out == nil {
			return nil
		}
		out.linear = linear
		// The default subregion is the union of the inputs' subregions, or the filter region when an input is a
		// standard one or there is none.
		if len(inputs) > 0 && !anyTrue(stds) {
			sub = unionRect(inputs)
		}
		sub = fs.subregion(pr, sub)
		out.clipTo(sub[0], sub[1], sub[2], sub[3])
		prev = out
		if pr.Result != "" {
			results[pr.Result] = out
		}
		for name, last := range lastUse {
			if last <= i {
				delete(results, name)
			}
		}
	}
	return prev
}

func anyTrue(b []bool) bool {
	for _, v := range b {
		if v {
			return true
		}
	}
	return false
}

func fullRect(fs *filterSpace) [4]int { return [4]int{0, 0, fs.w, fs.h} }

// unionRect is the union of the subregions of the images.
func unionRect(ims []*pixbuf) [4]int {
	r := [4]int{ims[0].x0, ims[0].y0, ims[0].x1, ims[0].y1}
	for _, im := range ims[1:] {
		r[0], r[1] = min(r[0], im.x0), min(r[1], im.y0)
		r[2], r[3] = max(r[2], im.x1), max(r[3], im.y1)
	}
	return r
}

// primitiveInputs lists the names of the inputs that a primitive reads (the result names; the standard ones are
// harmless in the list).
func primitiveInputs(pr *FilterPrimitive) []string {
	names := []string{pr.In, pr.In2}
	for _, ch := range pr.Children {
		names = append(names, ch.In)
	}
	return names
}

// subregion works out the subregion of a primitive in pixels: its x, y, width and height, each of which defaults to
// the default region (the union of the inputs' or the filter region), as the specification says.
//
// SVG 1.1: filters.html#FilterPrimitiveSubRegion
func (fs *filterSpace) subregion(pr *FilterPrimitive, def [4]int) [4]int {
	if pr.X == nil && pr.Y == nil && pr.Width == nil && pr.Height == nil {
		return def
	}
	// The default in user space: the corners of the default pixel rectangle mapped back.
	inv, ok := fs.m.Invert()
	if !ok {
		return def
	}
	var b bounds
	b.addTransformed(ui.NewRectFromLTRB(float64(def[0]), float64(def[1]), float64(def[2]), float64(def[3])), inv)
	x, y := b.r.Left, b.r.Top
	w, h := b.r.Width(), b.r.Height()
	if pr.X != nil {
		if fs.primObb {
			x = fs.bbox.Left + pr.X.Resolve(1)*fs.bbox.Width()
		} else {
			x = pr.X.Resolve(fs.vw)
		}
	}
	if pr.Y != nil {
		if fs.primObb {
			y = fs.bbox.Top + pr.Y.Resolve(1)*fs.bbox.Height()
		} else {
			y = pr.Y.Resolve(fs.vh)
		}
	}
	if pr.Width != nil {
		if fs.primObb {
			w = pr.Width.Resolve(1) * fs.bbox.Width()
		} else {
			w = pr.Width.Resolve(fs.vw)
		}
	}
	if pr.Height != nil {
		if fs.primObb {
			h = pr.Height.Resolve(1) * fs.bbox.Height()
		} else {
			h = pr.Height.Resolve(fs.vh)
		}
	}
	var px bounds
	px.addTransformed(ui.NewRectFromLTWH(x, y, w, h), fs.m)
	if !px.ok || math.IsNaN(px.r.Left) || math.IsNaN(px.r.Right) {
		return def
	}
	clampI := func(v float64) int { return int(math.Max(-1, math.Min(float64(1<<20), v))) }
	return [4]int{clampI(math.Floor(px.r.Left)), clampI(math.Floor(px.r.Top)), clampI(math.Ceil(px.r.Right)), clampI(math.Ceil(px.r.Bottom))}
}

// attrNumber reads a number attribute of a primitive, or the default.
func attrNumber(pr *FilterPrimitive, name string, def float64) float64 {
	v, ok := pr.Attrs[name]
	if !ok {
		return def
	}
	nums, ok := ParseNumberList(v)
	if !ok || len(nums) == 0 {
		return def
	}
	return nums[0]
}

// stdDeviation reads stdDeviation (one or two numbers, 2 by default for the drop shadow and 0 for the blur) in pixels.
// It reports false for a negative value, which makes the filter an error.
func (fs *filterSpace) stdDeviation(pr *FilterPrimitive) (sx, sy float64, ok bool) {
	def := 0.0
	if pr.Name == "feDropShadow" {
		def = 2
	}
	sx, sy = def, def
	if v, has := pr.Attrs["stdDeviation"]; has {
		if nums, good := ParseNumberList(v); good {
			switch len(nums) {
			case 1:
				sx, sy = nums[0], nums[0]
			case 2:
				sx, sy = nums[0], nums[1]
			}
		}
	}
	if sx < 0 || sy < 0 || math.IsNaN(sx) || math.IsNaN(sy) {
		return 0, 0, false
	}
	if fs.primObb {
		sx *= fs.bbox.Width()
		sy *= fs.bbox.Height()
	}
	return sx * fs.sx, sy * fs.sy, true
}

// offsetOf converts a vector in the user space of the element (in primitive units) to pixels.
func (fs *filterSpace) offsetOf(pr *FilterPrimitive, dx, dy float64) (float64, float64) {
	if fs.primObb {
		dx *= fs.bbox.Width()
		dy *= fs.bbox.Height()
	}
	return dx*fs.m.A + dy*fs.m.C, dx*fs.m.B + dy*fs.m.D
}

// radiusOf reads the radius of feMorphology in pixels (rounded to whole pixels); a negative radius is -1.
func (fs *filterSpace) radiusOf(pr *FilterPrimitive) (int, int) {
	rx, ry := 0.0, 0.0
	if v, has := pr.Attrs["radius"]; has {
		if nums, good := ParseNumberList(v); good {
			switch len(nums) {
			case 1:
				rx, ry = nums[0], nums[0]
			case 2:
				rx, ry = nums[0], nums[1]
			}
		}
	}
	if rx < 0 || ry < 0 || math.IsNaN(rx) || math.IsNaN(ry) {
		return -1, -1
	}
	if fs.primObb {
		rx *= fs.bbox.Width()
		ry *= fs.bbox.Height()
	}
	px := int(math.Min(math.Round(rx*fs.sx), float64(max(fs.w, fs.h))))
	py := int(math.Min(math.Round(ry*fs.sy), float64(max(fs.w, fs.h))))
	return px, py
}

// colorMatrixOf makes the 4x5 matrix of a feColorMatrix from its type and values.
//
// SVG 1.1: filters.html#feColorMatrixElement
func colorMatrixOf(pr *FilterPrimitive) [20]float64 {
	identity := [20]float64{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0}
	values, hasValues := pr.Attrs["values"]
	var nums []float64
	if hasValues {
		nums, _ = ParseNumberList(values)
	}
	switch strings.TrimSpace(pr.Attrs["type"]) {
	case "saturate":
		s := 1.0
		if len(nums) == 1 {
			s = nums[0]
		}
		return [20]float64{
			0.213 + 0.787*s, 0.715 - 0.715*s, 0.072 - 0.072*s, 0, 0,
			0.213 - 0.213*s, 0.715 + 0.285*s, 0.072 - 0.072*s, 0, 0,
			0.213 - 0.213*s, 0.715 - 0.715*s, 0.072 + 0.928*s, 0, 0,
			0, 0, 0, 1, 0,
		}
	case "hueRotate":
		a := 0.0
		if len(nums) == 1 {
			a = nums[0] * math.Pi / 180
		}
		s, c := math.Sin(a), math.Cos(a)
		return [20]float64{
			0.213 + c*0.787 - s*0.213, 0.715 - c*0.715 - s*0.715, 0.072 - c*0.072 + s*0.928, 0, 0,
			0.213 - c*0.213 + s*0.143, 0.715 + c*0.285 + s*0.140, 0.072 - c*0.072 - s*0.283, 0, 0,
			0.213 - c*0.213 - s*0.787, 0.715 - c*0.715 + s*0.715, 0.072 + c*0.928 + s*0.072, 0, 0,
			0, 0, 0, 1, 0,
		}
	case "luminanceToAlpha":
		return [20]float64{
			0, 0, 0, 0, 0,
			0, 0, 0, 0, 0,
			0, 0, 0, 0, 0,
			0.2125, 0.7154, 0.0721, 0, 0,
		}
	}
	if len(nums) == 20 {
		var m [20]float64
		copy(m[:], nums)
		return m
	}
	return identity
}

// transferFuncs makes the lookup tables of the feFuncR, feFuncG, feFuncB and feFuncA children of a
// feComponentTransfer.
//
// SVG 1.1: filters.html#feComponentTransferElement
func transferFuncs(pr *FilterPrimitive) channelFuncs {
	var out channelFuncs
	for _, ch := range pr.Children {
		var idx int
		switch ch.Name {
		case "feFuncR":
			idx = 0
		case "feFuncG":
			idx = 1
		case "feFuncB":
			idx = 2
		case "feFuncA":
			idx = 3
		default:
			continue
		}
		var table []float64
		if v, ok := ch.Attrs["tableValues"]; ok {
			table, _ = ParseNumberList(v)
		}
		var f func(float64) float64
		switch strings.TrimSpace(ch.Attrs["type"]) {
		case "table":
			if len(table) == 0 {
				continue
			}
			n := len(table) - 1
			f = func(c float64) float64 {
				if n == 0 {
					return table[0]
				}
				k := min(int(c*float64(n)), n-1)
				return table[k] + (c-float64(k)/float64(n))*float64(n)*(table[k+1]-table[k])
			}
		case "discrete":
			if len(table) == 0 {
				continue
			}
			n := len(table)
			f = func(c float64) float64 { return table[min(int(c*float64(n)), n-1)] }
		case "linear":
			slope, icpt := attrNumber(ch, "slope", 1), attrNumber(ch, "intercept", 0)
			f = func(c float64) float64 { return slope*c + icpt }
		case "gamma":
			amp, exp, off := attrNumber(ch, "amplitude", 1), attrNumber(ch, "exponent", 1), attrNumber(ch, "offset", 0)
			f = func(c float64) float64 { return amp*math.Pow(c, exp) + off }
		default:
			continue
		}
		var lut [256]uint8
		for i := range lut {
			v := f(float64(i) / 255)
			if math.IsNaN(v) {
				v = 0
			}
			lut[i] = uint8(math.Round(math.Min(1, math.Max(0, v)) * 255))
		}
		out[idx] = &lut
	}
	return out
}
