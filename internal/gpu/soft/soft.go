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
// is not returns the error SoftNotImplemented naming it, and no image. What
// works now: container, transform, offset and picture layers; Save, Restore and
// Transform; the clips; every shape and image operation, with a solid color,
// source-over and no filters; DrawColor and DrawPaint with the default blend
// mode. Gradients, blend modes, color filters and layers come with step 2.8, the
// blurs, shadows and the backdrop and shader mask layers with 2.9, text with 4.x.
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
	if err := r.layer(scene.Root(), ui.NewMatrix4Identity()); err != nil {
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

// layer draws a layer and everything under it; m maps its coordinates to pixels.
func (r *renderer) layer(l ui.Layer, m *ui.Matrix4) error {
	switch v := l.(type) {
	case *ui.ContainerLayer:
	case *ui.TransformLayer:
		m = m.Multiply(&v.Matrix)
	case *ui.PictureLayer:
		pm := m.Clone()
		pm.Translate(float32(v.Offset.Dx), float32(v.Offset.Dy), 0)
		return r.picture(v.Picture, &state{m: pm, clip: newClip(r.img.Rect)})
	default:
		return notImplemented("layer", l)
	}
	for _, c := range l.Children() {
		if err := r.layer(c, m); err != nil {
			return err
		}
	}
	return nil
}

// picture runs the operations of a picture, starting from the state st, which it
// changes.
func (r *renderer) picture(p *ui.Picture, st *state) error {
	var stack []*state
	for _, op := range p.DisplayList().Ops() {
		switch v := op.(type) {
		case ui.OpSave:
			stack = append(stack, st.clone())
		case ui.OpRestore:
			if n := len(stack); n > 0 {
				*st, stack = *stack[n-1], stack[:n-1]
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
			if v.BlendMode != ui.BlendModeSrcOver && v.BlendMode != ui.BlendModeUnspecified {
				return notImplemented("blend mode", v.BlendMode)
			}
			r.fillAll(st, solid(v.Color))
		case ui.OpDrawPaint:
			if err := checkPaint(&v.Paint); err != nil {
				return err
			}
			r.fillAll(st, solid(v.Paint.Color()))
		case ui.OpDrawPicture:
			inner := st.clone()
			if err := r.picture(v.Picture, inner); err != nil {
				return err
			}
		default:
			if err := r.draw(op, st); err != nil {
				return err
			}
		}
	}
	return nil
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

// checkPaint returns an error for the paint features that are not supported yet.
func checkPaint(p *ui.Paint) error {
	switch {
	case p.Shader() != nil:
		return notImplemented("paint feature", p.Shader())
	case p.ColorFilter() != nil:
		return notImplemented("paint feature", p.ColorFilter())
	case p.ImageFilter() != nil:
		return notImplemented("paint feature", p.ImageFilter())
	case p.MaskFilter() != nil:
		return notImplemented("paint feature", p.MaskFilter())
	case p.InvertColors():
		return notImplemented("paint feature", "InvertColors")
	}
	if m := p.BlendMode(); m != ui.BlendModeSrcOver {
		return notImplemented("blend mode", m)
	}
	return nil
}

// clipShape narrows the clip of the state to the path, in the coordinates of the
// state, or with subtract cuts the path out of it.
func (r *renderer) clipShape(st *state, path *ui.Path, antiAlias, subtract bool) {
	mask := Rasterize(flatPolys(path, st.m), fillRuleOf(path), st.clip.bounds)
	if !antiAlias {
		mask = threshold(mask)
	}
	if subtract {
		st.clip = st.clip.subtract(mask)
		return
	}
	st.clip = st.clip.intersect(mask)
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
	if err := checkPaint(p); err != nil {
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
	r.fillPolys(polys, rule, p, st)
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
// them with the paint's solid color.
func (r *renderer) fillPolys(polys [][]ui.Offset, rule FillRule, p *ui.Paint, st *state) {
	mask := Rasterize(polys, rule, st.clip.limit(r.img.Rect))
	if mask == nil {
		return
	}
	if !p.IsAntiAlias() {
		mask = threshold(mask)
	}
	r.composite(mask, st.clip, solid(p.Color()))
}

// source gives the color to paint at a pixel, as premultiplied red, green, blue
// and alpha from 0 to 1.
type source func(x, y int) [4]float32

// solid is the source of one color; the color is first rounded to 8 bits per
// channel, as Skia does.
func solid(c ui.Color) source {
	a := float32(c.Alpha()) / 255
	px := [4]float32{float32(c.Red()) / 255 * a, float32(c.Green()) / 255 * a, float32(c.Blue()) / 255 * a, a}
	return func(int, int) [4]float32 { return px }
}

// fillAll paints the whole clip with the source.
func (r *renderer) fillAll(st *state, src source) {
	b := st.clip.limit(r.img.Rect)
	if b.Empty() {
		return
	}
	r.composite(&Mask{Rect: b, Cov: ones(b.Dx() * b.Dy())}, st.clip, src)
}

func ones(n int) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = 1
	}
	return s
}

// composite paints the source over the surface where the mask and the clip cover
// it, in non-linear sRGB with premultiplied alpha (Skia's source-over).
func (r *renderer) composite(mask *Mask, cl clip, src source) {
	area := cl.limit(mask.Rect.Intersect(r.img.Rect))
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
			s := src(x, y)
			if s[3] == 0 {
				continue
			}
			i := r.img.PixOffset(x, y)
			px := r.img.Pix[i : i+4 : i+4]
			inv := 1 - s[3]*cov
			for k := 0; k < 4; k++ {
				px[k] = quantize(s[k]*cov*255 + float32(px[k])*inv)
			}
		}
	}
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
