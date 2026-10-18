package soft

import (
	"math"
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// shaderFunc gives the color of a shader at a point of its own coordinate space,
// premultiplied; the shader's matrix and the transform of the canvas are already
// undone.
type shaderFunc func(q ui.Offset) rgba

// tileT applies the tile mode to a position along a gradient; ok is false where
// the shader draws nothing (decal outside 0 to 1).
func tileT(t float64, mode ui.TileMode) (float64, bool) {
	switch mode {
	case ui.TileModeRepeated:
		return t - math.Floor(t), true
	case ui.TileModeMirror:
		u := math.Mod(t, 2)
		if u < 0 {
			u += 2
		}
		if u > 1 {
			u = 2 - u
		}
		return u, true
	case ui.TileModeDecal:
		return t, t >= 0 && t <= 1
	}
	return math.Min(math.Max(t, 0), 1), true
}

// colorRamp is the colors of a gradient with their positions.
type colorRamp struct {
	pos    []float64
	colors []rgba // unpremultiplied
}

// newColorRamp builds the ramp as Skia does: evenly spaced when there are no
// stops, a color at 0 and at 1 added when the stops do not reach them, and the
// stops kept in order.
func newColorRamp(colors []ui.Color, stops []float64) colorRamp {
	var ramp colorRamp
	n := len(colors)
	for i, c := range colors {
		p := float64(i) / math.Max(float64(n-1), 1)
		if len(stops) == n {
			p = stops[i]
		}
		if len(ramp.pos) > 0 {
			p = math.Max(p, ramp.pos[len(ramp.pos)-1])
		} else if p > 0 {
			ramp.pos = append(ramp.pos, 0)
			ramp.colors = append(ramp.colors, rgba{c.R, c.G, c.B, c.A})
		}
		p = math.Min(math.Max(p, 0), 1)
		ramp.pos = append(ramp.pos, p)
		ramp.colors = append(ramp.colors, rgba{c.R, c.G, c.B, c.A})
	}
	if last := ramp.pos[len(ramp.pos)-1]; last < 1 {
		ramp.pos = append(ramp.pos, 1)
		ramp.colors = append(ramp.colors, ramp.colors[len(ramp.colors)-1])
	}
	return ramp
}

// at returns the premultiplied color at t, from 0 to 1. The colors are
// interpolated without premultiplied alpha, as Flutter's gradients are.
func (r colorRamp) at(t float64) rgba {
	i := sort.SearchFloat64s(r.pos, t) // the first position >= t
	var c rgba
	switch {
	case i == 0:
		c = r.colors[0]
	case i >= len(r.pos):
		c = r.colors[len(r.colors)-1]
	default:
		p0, p1 := r.pos[i-1], r.pos[i]
		if p1 <= p0 {
			c = r.colors[i]
			break
		}
		f := float32((t - p0) / (p1 - p0))
		a, b := r.colors[i-1], r.colors[i]
		for k := range c {
			c[k] = a[k] + (b[k]-a[k])*f
		}
	}
	for k := range c {
		c[k] = clamp01(c[k])
	}
	return premul(c)
}

// gradientFunc returns the function that colors a point of the gradient's space.
func gradientFunc(g *ui.Gradient) shaderFunc {
	ramp := newColorRamp(g.Colors(), g.ColorStops())
	tile := g.TileMode()
	last := ramp.at(1)
	paint := func(t float64) rgba {
		u, ok := tileT(t, tile)
		if !ok || math.IsNaN(u) {
			return rgba{}
		}
		return ramp.at(u)
	}
	switch g.Kind() {
	case ui.GradientKindLinear:
		from, to := g.From(), g.To()
		dx, dy := to.Dx-from.Dx, to.Dy-from.Dy
		den := dx*dx + dy*dy
		if den == 0 {
			// A degenerate gradient is the last color, or nothing for decal.
			if tile == ui.TileModeDecal {
				return func(ui.Offset) rgba { return rgba{} }
			}
			return func(ui.Offset) rgba { return last }
		}
		return func(q ui.Offset) rgba { return paint(((q.Dx-from.Dx)*dx + (q.Dy-from.Dy)*dy) / den) }
	case ui.GradientKindRadial:
		c, radius := g.Center(), g.Radius()
		return func(q ui.Offset) rgba { return paint(math.Hypot(q.Dx-c.Dx, q.Dy-c.Dy) / radius) }
	case ui.GradientKindConical:
		return conicalFunc(g, paint)
	}
	// Sweep: the angle around the center, clockwise from the positive x axis.
	c, start, end := g.Center(), g.StartAngle(), g.EndAngle()
	span := end - start
	return func(q ui.Offset) rgba {
		a := math.Atan2(q.Dy-c.Dy, q.Dx-c.Dx) - start
		a -= 2 * math.Pi * math.Floor(a/(2*math.Pi)) // into [0, 2*pi)
		return paint(a / span)
	}
}

// conicalFunc is the two-point conical gradient from the focal circle to the
// end circle: the color at a point is that of the largest t whose circle, which
// moves and grows from one to the other, goes through the point.
func conicalFunc(g *ui.Gradient, paint func(float64) rgba) shaderFunc {
	c0, r0 := g.Focal(), g.FocalRadius()
	c1, r1 := g.Center(), g.Radius()
	cdx, cdy, dr := c1.Dx-c0.Dx, c1.Dy-c0.Dy, r1-r0
	a := cdx*cdx + cdy*cdy - dr*dr
	return func(q ui.Offset) rgba {
		px, py := q.Dx-c0.Dx, q.Dy-c0.Dy
		b := px*cdx + py*cdy + r0*dr
		c := px*px + py*py - r0*r0
		var t float64
		if math.Abs(a) < 1e-12 {
			if b == 0 {
				return rgba{}
			}
			t = c / (2 * b)
			if r0+t*dr < 0 {
				return rgba{}
			}
		} else {
			disc := b*b - a*c
			if disc < 0 {
				return rgba{}
			}
			sq := math.Sqrt(disc)
			t1, t2 := (b+sq)/a, (b-sq)/a
			if t1 < t2 {
				t1, t2 = t2, t1
			}
			switch {
			case r0+t1*dr >= 0:
				t = t1
			case r0+t2*dr >= 0:
				t = t2
			default:
				return rgba{}
			}
		}
		return paint(t)
	}
}

// shaderEval returns the function that colors a point of the surface for a
// shader painted under the transform that takes the shader's local space (the
// canvas's) to the surface, and whether the result should be dithered.
func (r *renderer) shaderEval(sh ui.Shader, toDevice affine) (func(ui.Offset) rgba, bool, error) {
	nothing := func(ui.Offset) rgba { return rgba{} }
	switch s := sh.(type) {
	case *ui.Gradient:
		fn := gradientFunc(s)
		inv, ok := shaderToDevice(toDevice, s.Matrix4).invert()
		if !ok {
			return nothing, false, nil
		}
		return func(p ui.Offset) rgba { return fn(inv.apply(p.Dx, p.Dy)) }, true, nil
	case *ui.ImageShader:
		img := s.Image()
		if err := checkImage(img); err != nil {
			return nil, false, err
		}
		inv, ok := affineOf(s.Matrix4()).then(toDevice).invert()
		if !ok {
			return nothing, false, nil
		}
		return r.imageShaderFunc(s, inv), false, nil
	}
	return nil, false, errors.NewErrf("SoftNotImplemented", "The software renderer does not support the shader %T.", sh)
}

// shaderSource is shaderEval for the pixels of the surface, read at their
// centers.
func (r *renderer) shaderSource(sh ui.Shader, toDevice affine) (source, bool, error) {
	ev, dither, err := r.shaderEval(sh, toDevice)
	if err != nil {
		return nil, false, err
	}
	return func(x, y int) rgba { return ev(ui.Offset{Dx: float64(x) + 0.5, Dy: float64(y) + 0.5}) }, dither, nil
}

// shaderToDevice composes the shader's own matrix, which the gradient reports
// with a flag for whether it has one, with the transform of the canvas.
func shaderToDevice(toDevice affine, matrix func() (*ui.Matrix4, bool)) affine {
	if m, ok := matrix(); ok {
		return affineOf(m).then(toDevice)
	}
	return toDevice
}

// imageShaderFunc samples the shader's image at the pixels of the surface, which
// the inverse transform maps to the image's pixels.
func (r *renderer) imageShaderFunc(s *ui.ImageShader, inv affine) func(ui.Offset) rgba {
	img := s.Image()
	quality := s.FilterQuality()
	if quality == ui.FilterQualityUnspecified {
		quality = ui.FilterQualityLow
	}
	chain := r.levels(img)
	level := 0
	if quality == ui.FilterQualityMedium || quality == ui.FilterQualityHigh {
		// How many image pixels one surface pixel spans: the inverse transform's scale.
		if scale := math.Sqrt(math.Abs(inv.det())); scale > 1 {
			level = min(int(math.Log2(scale)+0.5), len(chain)-1)
		}
	}
	l := chain[level]
	t := tiling{x: s.TileModeX(), y: s.TileModeY()}
	w0, h0 := img.Width(), img.Height()
	return func(p ui.Offset) rgba {
		pt := inv.apply(p.Dx, p.Dy)
		return l.sample(pt.Dx, pt.Dy, w0, h0, quality, t)
	}
}
