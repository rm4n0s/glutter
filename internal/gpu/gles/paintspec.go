package gles

import (
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// A paint's shader, color filter, inverted colors and blend mode are the uniforms
// of a draw call (shaders/paint.glsl). The builder describes them in a paintSpec,
// which is comparable and interned, so that draws with the same paint share a
// draw call.

// The paint shaders' kinds and a paint's uniforms are shared with the explicit renderer
// (gpu/paintspec.go).
const (
	shaderNone = gpu.ShaderNone
	maxStops   = gpu.MaxStops
)

// paintSpec is the shader, color filter and blend of a paint, as uniforms.
type paintSpec = gpu.PaintSpec

// intern returns the one paintSpec that equals sp.
func (b *frameBuilder) intern(sp *paintSpec) *paintSpec {
	if p, ok := b.specs[*sp]; ok {
		return p
	}
	if b.specs == nil {
		b.specs = map[paintSpec]*paintSpec{}
	}
	p := new(paintSpec)
	*p = *sp
	b.specs[*p] = p
	return p
}

// blendIndex is the index of a blend mode in shaders/blend.glsl: its place in
// the order of Flutter's BlendMode.
func blendIndex(m ui.BlendMode) (int, bool) { return gpu.BlendIndex(m) }

// shaderSpec fills the shader part of a spec for the shader sh under the
// transform that takes the shader's local space to the surface. status says what
// to do with the paint: paintNothing for a shader that draws nothing, and
// paintUnsupported with the error of unsupported for one that cannot be drawn.
func (s *shapeContext) shaderSpec(sp *paintSpec, sh ui.Shader, toDevice affine2D) (paintStatus, error) {
	switch v := sh.(type) {
	case *ui.Gradient:
		return s.status(sp.SetGradient(v, toDevice))
	case *ui.ImageShader:
		return s.imageShaderSpec(sp, v, toDevice)
	}
	return paintUnsupported, s.b.unsupported("paint shader")
}

// status turns the result of a spec into what the builder does with the paint.
func (s *shapeContext) status(r gpu.PaintResult, what string) (paintStatus, error) {
	switch r {
	case gpu.PaintNothing:
		return paintNothing, nil
	case gpu.PaintUnsupported:
		return paintUnsupported, s.b.unsupported(what)
	}
	return paintDraw, nil
}

func (s *shapeContext) imageShaderSpec(sp *paintSpec, is *ui.ImageShader, toDevice affine2D) (paintStatus, error) {
	img := is.Image()
	if err := checkImage(img); err != nil {
		return paintUnsupported, err
	}
	if s.b.res == nil {
		return paintUnsupported, s.b.unsupported("image shader")
	}
	r, err := sp.SetImageShader(is, toDevice, func(img *ui.Image) (any, int, error) {
		entry, err := s.b.res.imageTexture(img)
		if err != nil {
			return nil, 0, err
		}
		return entry.tex, entry.levels, nil
	})
	if err != nil {
		return paintUnsupported, err
	}
	return s.status(r, "")
}

// filterSpec fills the color filter and inverted colors of a spec.
func filterSpec(sp *paintSpec, f *ui.ColorFilter, invert bool) { sp.SetColorFilter(f, invert) }
