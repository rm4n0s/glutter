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
// Transform; DrawColor with the default blend mode; and DrawPath with a solid
// color, either filled or stroked. Clips come with step 2.7, and gradients,
// blend modes, filters and layers with 2.8 and 2.9.
//
// Flutter: no counterpart (flutter_test renders with Skia's software backend)
func Render(scene *ui.Scene, width, height int) (*image.RGBA, error) {
	if scene == nil {
		return nil, errors.New("SoftSceneNil", "Render needs a scene")
	}
	if width <= 0 || height <= 0 || width > maxSurfaceSize || height > maxSurfaceSize {
		return nil, errors.New("SoftSizeInvalid", "The surface size is not valid.", "width", width, "height", height)
	}
	r := &renderer{img: image.NewRGBA(image.Rect(0, 0, width, height))}
	if err := r.layer(scene.Root(), ui.NewMatrix4Identity()); err != nil {
		return nil, err
	}
	return r.img, nil
}

type renderer struct {
	img *image.RGBA
}

// layer draws a layer and everything under it; m maps its coordinates to pixels.
func (r *renderer) layer(l ui.Layer, m *ui.Matrix4) error {
	switch v := l.(type) {
	case *ui.ContainerLayer:
	case *ui.TransformLayer:
		m = m.Multiply(&v.Matrix)
	case *ui.PictureLayer:
		pm := m.Clone()
		pm.Translate(float32(v.Offset.Dx), float32(v.Offset.Dy), 0)
		return r.picture(v.Picture, pm)
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

// picture runs the operations of a picture.
func (r *renderer) picture(p *ui.Picture, m *ui.Matrix4) error {
	stack := []*ui.Matrix4{}
	m = m.Clone()
	for _, op := range p.DisplayList().Ops() {
		switch v := op.(type) {
		case ui.OpSave:
			stack = append(stack, m.Clone())
		case ui.OpRestore:
			if n := len(stack); n > 0 {
				m, stack = stack[n-1], stack[:n-1]
			}
		case ui.OpTransform:
			m.MultiplyBy(&v.Matrix)
		case ui.OpDrawColor:
			if v.BlendMode != ui.BlendModeSrcOver && v.BlendMode != ui.BlendModeUnspecified {
				return notImplemented("blend mode", v.BlendMode)
			}
			r.fill(&Mask{Rect: r.img.Rect, Cov: ones(r.img.Rect.Dx() * r.img.Rect.Dy())}, v.Color)
		case ui.OpDrawPath:
			if err := r.drawPath(v.Path, &v.Paint, m); err != nil {
				return err
			}
		default:
			return notImplemented("operation", op)
		}
	}
	return nil
}

func ones(n int) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = 1
	}
	return s
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

// drawPath fills or strokes a path with the paint, under the matrix.
func (r *renderer) drawPath(path *ui.Path, p *ui.Paint, m *ui.Matrix4) error {
	if err := checkPaint(p); err != nil {
		return err
	}
	var polys [][]ui.Offset
	rule := FillRule(FillNonZero)
	if p.Style() == ui.PaintingStyleStroke {
		scale := math.Sqrt(math.Abs(float64(m.Entry(0, 0))*float64(m.Entry(1, 1)) - float64(m.Entry(0, 1))*float64(m.Entry(1, 0))))
		if !(scale > 0) {
			return nil
		}
		width := p.StrokeWidth()
		if width <= 0 {
			width = 1 / scale // a hairline is one pixel wide
		}
		local := path.Flatten(nil, flattenTolerance/scale)
		polys = Stroke(local, StrokeStyle{
			Width: width, Cap: p.StrokeCap(), Join: p.StrokeJoin(), MiterLimit: p.StrokeMiterLimit(), Scale: scale,
		})
		for _, poly := range polys {
			for i, pt := range poly {
				poly[i] = m.TransformPoint(pt)
			}
		}
	} else {
		for _, c := range path.Flatten(m, flattenTolerance) {
			polys = append(polys, c.Points)
		}
		if path.FillType() == ui.PathFillTypeEvenOdd {
			rule = FillEvenOdd
		}
	}
	mask := Rasterize(polys, rule, r.img.Rect)
	if mask == nil {
		return nil
	}
	if !p.IsAntiAlias() {
		for i, c := range mask.Cov {
			if c >= 0.5 {
				mask.Cov[i] = 1
			} else {
				mask.Cov[i] = 0
			}
		}
	}
	r.fill(mask, p.Color())
	return nil
}

// fill composites the color over the surface where the mask covers it, in
// non-linear sRGB with premultiplied alpha (Skia's source-over).
func (r *renderer) fill(mask *Mask, c ui.Color) {
	a := float32(c.Alpha()) / 255
	sr, sg, sb := float32(c.Red())/255*a, float32(c.Green())/255*a, float32(c.Blue())/255*a
	area := mask.Rect.Intersect(r.img.Rect)
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			cov := mask.Cov[(y-mask.Rect.Min.Y)*mask.Rect.Dx()+(x-mask.Rect.Min.X)]
			if cov == 0 {
				continue
			}
			i := r.img.PixOffset(x, y)
			px := r.img.Pix[i : i+4 : i+4]
			inv := 1 - a*cov
			px[0] = quantize(sr*cov*255 + float32(px[0])*inv)
			px[1] = quantize(sg*cov*255 + float32(px[1])*inv)
			px[2] = quantize(sb*cov*255 + float32(px[2])*inv)
			px[3] = quantize(a*cov*255 + float32(px[3])*inv)
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
