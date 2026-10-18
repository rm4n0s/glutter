package soft

import (
	"image"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// maxSurfaceSize is the largest width or height Render accepts, in pixels.
const maxSurfaceSize = 1 << 15

// flattenTolerance is how far, in pixels, a flattened curve may be from the
// curve.
const flattenTolerance = 0.05

// Render draws a scene into a new image of width x height pixels, in
// premultiplied RGBA with a transparent background.
//
// Not everything is supported yet. A layer, an operation or a paint feature that
// is not returns the error SoftNotImplemented naming it, and no image. What works
// now: container, transform, offset, clip, opacity and color filter layers and
// picture layers; every operation except text, shadows and the ones that need a
// blur; paints with colors, gradients, image shaders, all 29 blend modes, color
// filters and inverted colors; and SaveLayer with a paint. The blurs, shadows,
// image filters and the backdrop filter and shader mask layers come with step
// 2.9, text with 4.x.
//
// Flutter: no counterpart (flutter_test renders with Skia's software backend)
func Render(scene *ui.Scene, width, height int) (*image.RGBA, error) {
	if scene == nil {
		return nil, errors.New("SoftSceneNil", "Render needs a scene")
	}
	if width <= 0 || height <= 0 || width > maxSurfaceSize || height > maxSurfaceSize {
		return nil, errors.New("SoftSizeInvalid", "The surface size is not valid.", "width", width, "height", height)
	}
	r := &renderer{img: image.NewRGBA(image.Rect(0, 0, width, height)), mips: map[*ui.Image][]*mipLevel{}}
	if err := r.layer(scene.Root(), &state{m: ui.NewMatrix4Identity(), clip: newClip(r.img.Rect)}); err != nil {
		return nil, err
	}
	return r.img, nil
}

type renderer struct {
	img  *image.RGBA
	mips map[*ui.Image][]*mipLevel
}

// state is what Save and Restore keep: the transform and the clip.
type state struct {
	m    *ui.Matrix4
	clip clip
}

func (s *state) clone() *state { return &state{m: s.m.Clone(), clip: s.clip} }

// stackEntry is a Save or SaveLayer that has not been restored: the state to go
// back to and, for a layer, what to composite it with.
type stackEntry struct {
	saved state
	layer *layerRec
	comp  compositeParams
}

// picture runs the operations of a picture, starting from the state st, which it
// changes.
func (r *renderer) picture(p *ui.Picture, st *state) error {
	var stack []stackEntry
	restore := func() {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		*st = e.saved
		if e.layer != nil {
			r.endLayer(e.layer, st.clip, e.comp)
		}
	}
	for _, op := range p.DisplayList().Ops() {
		switch v := op.(type) {
		case ui.OpSave:
			stack = append(stack, stackEntry{saved: *st.clone()})
		case ui.OpSaveLayer:
			comp, err := r.compositeOf(&v.Paint)
			if err != nil {
				return err
			}
			bounds := st.clip.bounds
			if v.HasBounds {
				bounds = bounds.Intersect(deviceBounds(st.m, v.Bounds))
			}
			stack = append(stack, stackEntry{saved: *st.clone(), layer: r.beginLayer(bounds), comp: comp})
		case ui.OpRestore:
			if len(stack) > 0 {
				restore()
			}
		case ui.OpTransform:
			st.m.MultiplyBy(&v.Matrix)
		case ui.OpClipRect:
			r.clipShape(st, rectPath(v.Rect), v.AntiAlias, v.Op == ui.ClipOpDifference)
		case ui.OpClipRRect:
			r.clipShape(st, rrectPath(v.RRect), v.AntiAlias, false)
		case ui.OpClipRSuperellipse:
			r.clipShape(st, rsuperellipsePath(v.RSuperellipse), v.AntiAlias, false)
		case ui.OpClipPath:
			r.clipShape(st, v.Path, v.AntiAlias, false)
		case ui.OpDrawColor:
			r.fillAll(st, paintSetup{src: solid(v.Color), mode: v.BlendMode})
		case ui.OpDrawPaint:
			ps, err := r.setup(&v.Paint, affineOf(st.m))
			if err != nil {
				return err
			}
			r.fillAll(st, ps)
		case ui.OpDrawPicture:
			if err := r.picture(v.Picture, st.clone()); err != nil {
				return err
			}
		default:
			if err := r.draw(op, st); err != nil {
				return err
			}
		}
	}
	// A recording may end with layers still open; they are composited as if
	// restored.
	for len(stack) > 0 {
		restore()
	}
	return nil
}

// deviceBounds returns the pixels that the rectangle covers under the transform.
func deviceBounds(m *ui.Matrix4, r ui.Rect) image.Rectangle {
	b := m.TransformRect(r)
	if math.IsNaN(b.Left+b.Top+b.Right+b.Bottom) || math.IsInf(b.Left+b.Top+b.Right+b.Bottom, 0) {
		return image.Rectangle{}
	}
	clamp := func(v float64) int { return int(math.Max(-maxCoord, math.Min(maxCoord, v))) }
	return image.Rect(clamp(math.Floor(b.Left)), clamp(math.Floor(b.Top)), clamp(math.Ceil(b.Right)), clamp(math.Ceil(b.Bottom)))
}

// draw runs the drawing operations that make a shape, an image or triangles.
func (r *renderer) draw(op ui.DisplayOp, st *state) error {
	switch v := op.(type) {
	case ui.OpDrawPath:
		return r.drawPath(v.Path, &v.Paint, st, false)
	case ui.OpDrawRect:
		return r.drawPath(rectPath(v.Rect), &v.Paint, st, false)
	case ui.OpDrawRRect:
		return r.drawPath(rrectPath(v.RRect), &v.Paint, st, false)
	case ui.OpDrawRSuperellipse:
		return r.drawPath(rsuperellipsePath(v.RSuperellipse), &v.Paint, st, false)
	case ui.OpDrawDRRect:
		return r.drawPath(drrectPath(v.Outer, v.Inner), &v.Paint, st, false)
	case ui.OpDrawOval:
		return r.drawPath(ovalPath(v.Rect), &v.Paint, st, false)
	case ui.OpDrawCircle:
		return r.drawPath(circlePath(v.Center, v.Radius), &v.Paint, st, false)
	case ui.OpDrawArc:
		return r.drawPath(arcPath(v.Rect, v.StartAngle, v.SweepAngle, v.UseCenter, v.Paint.Style() != ui.PaintingStyleStroke), &v.Paint, st, false)
	case ui.OpDrawLine:
		p := ui.NewPath()
		p.MoveTo(v.P1.Dx, v.P1.Dy)
		p.LineTo(v.P2.Dx, v.P2.Dy)
		return r.drawPath(p, &v.Paint, st, true)
	case ui.OpDrawPoints:
		return r.drawPoints(v, st)
	case ui.OpDrawImage:
		w, h := float64(v.Image.Width()), float64(v.Image.Height())
		return r.drawImageRect(v.Image, ui.NewRectFromLTWH(0, 0, w, h), ui.NewRectFromLTWH(v.Offset.Dx, v.Offset.Dy, w, h), &v.Paint, st)
	case ui.OpDrawImageRect:
		return r.drawImageRect(v.Image, v.Src, v.Dst, &v.Paint, st)
	case ui.OpDrawImageNine:
		return r.drawImageNine(v, st)
	case ui.OpDrawAtlas:
		return r.drawAtlas(v, st)
	case ui.OpDrawVertices:
		return r.drawVertices(v, st)
	}
	return notImplemented("operation", op)
}

// notImplemented is the error for what a step after this one adds.
func notImplemented(what string, v any) error {
	return errors.NewErrf("SoftNotImplemented", "The software renderer does not support the %s %T yet.", what, v)
}

// paintSetup is what a drawing operation needs from a paint: the color to
// paint at each pixel, with the color filters applied, the blend mode, and
// whether to dither (gradients are).
type paintSetup struct {
	src    source
	mode   ui.BlendMode
	dither bool
}

// compositeParams is how an image or a layer is put over the surface: scaled by
// an alpha, filtered, and blended.
type compositeParams struct {
	alpha  float32
	filter filterFunc
	mode   ui.BlendMode
}

// filtersOf returns the function that applies the color filter of the paint and
// then inverts the colors if the paint says so.
func filtersOf(p *ui.Paint) filterFunc {
	var inv filterFunc
	if p.InvertColors() {
		inv = invertFunc
	}
	return chainFilters(colorFilterFunc(p.ColorFilter()), inv)
}

// checkEffects returns an error for the paint features that belong to step 2.9.
func checkEffects(p *ui.Paint) error {
	if f := p.ImageFilter(); f != nil {
		return notImplemented("paint feature", f)
	}
	if f := p.MaskFilter(); f != nil {
		return notImplemented("paint feature", f)
	}
	return nil
}

// baseSource returns the color a paint makes at each pixel, before its color
// filters: the shader's, scaled by the alpha of the paint's color, or the color.
// toDevice maps the coordinates the shapes are given in to the surface. The
// second result says that the color should be dithered.
func (r *renderer) baseSource(p *ui.Paint, toDevice affine) (source, bool, error) {
	sh := p.Shader()
	if sh == nil {
		return solid(p.Color()), false, nil
	}
	src, dither, err := r.shaderSource(sh, toDevice)
	if err != nil {
		return nil, false, err
	}
	if a := float32(p.Color().Alpha()) / 255; a != 1 {
		inner := src
		src = func(x, y int) rgba {
			c := inner(x, y)
			return rgba{c[0] * a, c[1] * a, c[2] * a, c[3] * a}
		}
	}
	return src, dither, nil
}

// setup builds the setup of a paint for shapes.
func (r *renderer) setup(p *ui.Paint, toDevice affine) (paintSetup, error) {
	if err := checkEffects(p); err != nil {
		return paintSetup{}, err
	}
	src, dither, err := r.baseSource(p, toDevice)
	if err != nil {
		return paintSetup{}, err
	}
	ps := paintSetup{src: src, mode: p.BlendMode(), dither: dither}
	if f := filtersOf(p); f != nil {
		inner := ps.src
		ps.src = func(x, y int) rgba { return f(inner(x, y)) }
	}
	return ps, nil
}

// compositeOf builds the parameters for putting an image or a layer over the
// surface with the paint: its alpha, color filter, inverted colors and blend
// mode. A shader of the paint does not apply.
func (r *renderer) compositeOf(p *ui.Paint) (compositeParams, error) {
	if err := checkEffects(p); err != nil {
		return compositeParams{}, err
	}
	return compositeParams{alpha: float32(p.Color().Alpha()) / 255, filter: filtersOf(p), mode: p.BlendMode()}, nil
}

// clipShape narrows the clip of the state to the path, in the coordinates of the
// state, or with subtract cuts the path out of it.
func (r *renderer) clipShape(st *state, path *ui.Path, antiAlias, subtract bool) {
	st.clip = shapeClip(st, path, antiAlias, subtract)
}

// shapeClip returns the clip of the state narrowed to the path.
func shapeClip(st *state, path *ui.Path, antiAlias, subtract bool) clip {
	mask := Rasterize(flatPolys(path, st.m), fillRuleOf(path), st.clip.bounds)
	if !antiAlias {
		mask = threshold(mask)
	}
	if subtract {
		return st.clip.subtract(mask)
	}
	return st.clip.intersect(mask)
}

// fillRuleOf returns the rule that fills the path.
func fillRuleOf(p *ui.Path) FillRule {
	if p.FillType() == ui.PathFillTypeEvenOdd {
		return FillEvenOdd
	}
	return FillNonZero
}

// flatPolys turns a path into polygons in device coordinates.
func flatPolys(path *ui.Path, m *ui.Matrix4) [][]ui.Offset {
	var polys [][]ui.Offset
	for _, c := range path.Flatten(m, flattenTolerance) {
		polys = append(polys, c.Points)
	}
	return polys
}

// affineScale returns how much the transform magnifies lengths on average.
func affineScale(m *ui.Matrix4) float64 {
	return math.Sqrt(math.Abs(float64(m.Entry(0, 0))*float64(m.Entry(1, 1)) - float64(m.Entry(0, 1))*float64(m.Entry(1, 0))))
}

// drawPath fills or strokes a path with the paint, under the state. With
// forceStroke the paint's style is ignored and the path is stroked.
func (r *renderer) drawPath(path *ui.Path, p *ui.Paint, st *state, forceStroke bool) error {
	ps, err := r.setup(p, affineOf(st.m))
	if err != nil {
		return err
	}
	if st.clip.empty() {
		return nil
	}
	var polys [][]ui.Offset
	rule := FillNonZero
	if forceStroke || p.Style() == ui.PaintingStyleStroke {
		polys = strokePolys(path.Flatten(nil, flattenTolerance/math.Max(affineScale(st.m), 1e-6)), p, st.m)
	} else {
		polys = flatPolys(path, st.m)
		rule = fillRuleOf(path)
	}
	r.fillPolys(polys, rule, p, ps, st)
	return nil
}

// strokePolys strokes contours given in local coordinates with the paint's
// stroke properties and returns the polygons in device coordinates.
func strokePolys(local []ui.FlatContour, p *ui.Paint, m *ui.Matrix4) [][]ui.Offset {
	scale := affineScale(m)
	if !(scale > 0) {
		return nil
	}
	width := p.StrokeWidth()
	if width <= 0 {
		width = 1 / scale // a hairline is one pixel wide
	}
	polys := Stroke(local, StrokeStyle{
		Width: width, Cap: p.StrokeCap(), Join: p.StrokeJoin(), MiterLimit: p.StrokeMiterLimit(), Scale: scale,
	})
	for _, poly := range polys {
		for i, pt := range poly {
			poly[i] = m.TransformPoint(pt)
		}
	}
	return polys
}

// fillPolys rasterizes the polygons, which are in device coordinates, and paints
// them.
func (r *renderer) fillPolys(polys [][]ui.Offset, rule FillRule, p *ui.Paint, ps paintSetup, st *state) {
	mask := Rasterize(polys, rule, st.clip.limit(r.img.Rect))
	if mask == nil {
		return
	}
	if !p.IsAntiAlias() {
		mask = threshold(mask)
	}
	r.composite(mask, st.clip, ps)
}

// source gives the color to paint at a pixel, as premultiplied red, green, blue
// and alpha from 0 to 1.
type source func(x, y int) rgba

// solid is the source of one color; the color is first rounded to 8 bits per
// channel, as Skia does.
func solid(c ui.Color) source {
	a := float32(c.Alpha()) / 255
	px := rgba{float32(c.Red()) / 255 * a, float32(c.Green()) / 255 * a, float32(c.Blue()) / 255 * a, a}
	return func(int, int) rgba { return px }
}

// fillAll paints the whole clip.
func (r *renderer) fillAll(st *state, ps paintSetup) {
	b := st.clip.limit(r.img.Rect)
	if b.Empty() {
		return
	}
	r.composite(&Mask{Rect: b, Cov: ones(b.Dx() * b.Dy())}, st.clip, ps)
}

func ones(n int) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = 1
	}
	return s
}

// bayer is the 8x8 ordered-dither matrix, with values from 0 to 63.
var bayer = [8][8]uint8{
	{0, 32, 8, 40, 2, 34, 10, 42}, {48, 16, 56, 24, 50, 18, 58, 26},
	{12, 44, 4, 36, 14, 46, 6, 38}, {60, 28, 52, 20, 62, 30, 54, 22},
	{3, 35, 11, 43, 1, 33, 9, 41}, {51, 19, 59, 27, 49, 17, 57, 25},
	{15, 47, 7, 39, 13, 45, 5, 37}, {63, 31, 55, 23, 61, 29, 53, 21},
}

// ditherOffset is added to a channel, in units of one level of 8 bits, before it
// is rounded, so that a smooth gradient does not show bands.
func ditherOffset(x, y int) float32 {
	return (float32(bayer[y&7][x&7])+0.5)/64 - 0.5
}

// load reads the premultiplied color of a pixel of the surface.
func load(img *image.RGBA, x, y int) rgba {
	i := img.PixOffset(x, y)
	p := img.Pix[i : i+4 : i+4]
	return rgba{float32(p[0]) / 255, float32(p[1]) / 255, float32(p[2]) / 255, float32(p[3]) / 255}
}

// store writes a premultiplied color, with 0 to 255 channels, to the surface.
func store(img *image.RGBA, x, y int, c [4]float32, dither float32) {
	i := img.PixOffset(x, y)
	px := img.Pix[i : i+4 : i+4]
	// Keep the color valid: no channel above the alpha.
	a := min(max(c[3], 0), 255)
	for k := 0; k < 3; k++ {
		c[k] = min(c[k], a)
	}
	for k := 0; k < 4; k++ {
		px[k] = quantize(c[k] + dither)
	}
}

// composite paints the source over the surface where the mask and the clip cover
// it, with the blend mode, in non-linear sRGB with premultiplied alpha. Where the
// coverage is partial the result is mixed with what was there.
func (r *renderer) composite(mask *Mask, cl clip, ps paintSetup) {
	area := cl.limit(mask.Rect.Intersect(r.img.Rect))
	over := ps.mode == ui.BlendModeSrcOver || ps.mode == ui.BlendModeUnspecified
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			cov := mask.Cov[(y-mask.Rect.Min.Y)*mask.Rect.Dx()+(x-mask.Rect.Min.X)]
			if cov == 0 {
				continue
			}
			if cl.mask != nil {
				cov *= cl.at(x, y)
				if cov == 0 {
					continue
				}
			}
			s := ps.src(x, y)
			if over && s[3] == 0 {
				continue
			}
			var dither float32
			if ps.dither {
				dither = ditherOffset(x, y)
			}
			r.mix(x, y, s, ps.mode, cov, dither)
		}
	}
}

// mix blends the color s into the pixel with the mode, by the coverage.
func (r *renderer) mix(x, y int, s rgba, mode ui.BlendMode, cov, dither float32) {
	d := load(r.img, x, y)
	var out rgba
	if mode == ui.BlendModeSrcOver || mode == ui.BlendModeUnspecified {
		inv := 1 - s[3]*cov
		for k := range out {
			out[k] = s[k]*cov + d[k]*inv
		}
	} else {
		b := blend(mode, s, d)
		for k := range out {
			out[k] = d[k] + (b[k]-d[k])*cov
		}
	}
	for k := range out {
		out[k] *= 255
	}
	store(r.img, x, y, out, dither)
}

func quantize(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}
