package gpu

import (
	"image"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// The parts of blurs and shadows (SPEC §9.1) that the GPU renderers share: finding the paint of
// an operation, the paint that draws only a shape's coverage, the numbers of the blur styles,
// and the geometry of Canvas.drawShadow.

// PaintAccess returns the paint of a drawing operation and the function that makes
// the same operation with another paint; nil for the operations without one.
//
// Flutter: no counterpart
func PaintAccess(op ui.DisplayOp) (*ui.Paint, func(*ui.Paint) ui.DisplayOp) {
	switch v := op.(type) {
	case ui.OpDrawPath:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawRect:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawRRect:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawRSuperellipse:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawDRRect:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawOval:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawCircle:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawArc:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawLine:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawPoints:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawImage:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawImageRect:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawImageNine:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawAtlas:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	case ui.OpDrawVertices:
		return &v.Paint, func(p *ui.Paint) ui.DisplayOp { v.Paint = *p; return v }
	}
	return nil, nil
}

// WhiteFill is the paint that draws only the coverage of a shape: opaque white,
// source-over, with the geometry of p (style, stroke, anti-aliasing).
//
// Flutter: no counterpart
func WhiteFill(p *ui.Paint) *ui.Paint {
	q := *p
	q.SetMaskFilter(nil)
	q.SetImageFilter(nil)
	q.SetShader(nil)
	q.SetColorFilter(nil)
	q.SetInvertColors(false)
	q.SetColor(ui.NewColor(0xFFFFFFFF))
	q.SetBlendMode(ui.BlendModeSrcOver)
	return &q
}

// StyleCode is the number of a blur style in the shaders.
//
// Flutter: no counterpart
func StyleCode(s ui.BlurStyle) int32 {
	switch s {
	case ui.BlurStyleSolid:
		return 1
	case ui.BlurStyleOuter:
		return 2
	case ui.BlurStyleInner:
		return 3
	}
	return 0
}

// BlurRadiusToSigma is Skia's conversion of a blur radius to a deviation.
//
// Flutter: no counterpart
func BlurRadiusToSigma(radius float64) float64 {
	if radius > 0 {
		return 0.57735*radius + 0.5
	}
	return 0
}

// Skia's constants for the shadows of Flutter's physical shapes, as in the software
// renderer.
const (
	// ShadowAmbientAlpha is the alpha of the ambient shadow.
	//
	// Flutter: no counterpart
	ShadowAmbientAlpha = 0.039
	// ShadowSpotAlpha is the alpha of the spot shadow.
	//
	// Flutter: no counterpart
	ShadowSpotAlpha = 0.25
	// ShadowLightHeight is the height of the light above the surface.
	//
	// Flutter: no counterpart
	ShadowLightHeight = 600.0
	// ShadowLightRadius is the radius of the light.
	//
	// Flutter: no counterpart
	ShadowLightRadius = 800.0
	// ShadowAmbientFactor is the blur radius of the ambient shadow for each unit of elevation.
	//
	// Flutter: no counterpart
	ShadowAmbientFactor = 0.5
	// ShadowMaxAmbient is the largest blur radius of the ambient shadow.
	//
	// Flutter: no counterpart
	ShadowMaxAmbient = 300.0
)

// ShadowPlan is how Canvas.drawShadow's two blurred shadows are drawn.
//
// Flutter: no counterpart
type ShadowPlan struct {
	// AmbientSigma and SpotSigma are the deviations of the two blurs in pixels.
	AmbientSigma, SpotSigma float64
	// Spot is the transform of the path of the spot shadow: moved down and grown, away from a
	// light above the shape.
	Spot *ui.Matrix4
}

// PlanShadow plans the shadow of a path of elevation, under the transform m with the scale
// (the square root of its determinant), when the path covers devBounds of the surface. false
// says that there is nothing to draw.
//
// Flutter: no counterpart
func PlanShadow(elevation float64, m *ui.Matrix4, scale float64, devBounds image.Rectangle) (ShadowPlan, bool) {
	z := elevation * scale
	lightZ := ShadowLightHeight * scale
	if !(z > 0) || z >= lightZ || devBounds.Empty() {
		return ShadowPlan{}, false
	}
	p := ShadowPlan{
		AmbientSigma: BlurRadiusToSigma(math.Min(z*ShadowAmbientFactor, ShadowMaxAmbient)),
		SpotSigma:    BlurRadiusToSigma(ShadowLightRadius * scale * z / (lightZ - z)),
	}
	spotScale := lightZ / (lightZ - z)
	dy := z * (ShadowLightHeight*scale + float64(devBounds.Dy())/2) / (lightZ - z)
	cx, cy := float64(devBounds.Min.X+devBounds.Max.X)/2, float64(devBounds.Min.Y+devBounds.Max.Y)/2
	spot := ui.NewMatrix4TranslationValues(float32(cx*(1-spotScale)), float32(cy*(1-spotScale)+dy), 0)
	spot.Scale(float32(spotScale), float32(spotScale), 1)
	p.Spot = spot.Multiply(m)
	return p, true
}
