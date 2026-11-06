package viewport3d

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

// GridStyle is the infinite grid of a scene (SceneStyle.Grid; SPEC §27.8, "Overlays"): thin minor lines and thicker major ones on
// a plane, which fade away with the distance and where they come closer than a few pixels. The grid is drawn over the opaque
// faces and under the transparent ones, tested against their depth, and is cut by the section planes. It is not picked.
//
// Flutter: no counterpart
type GridStyle struct {
	// Show draws the grid; the zero value draws none.
	Show bool
	// Origin is a point of the grid's plane, in world coordinates in float64; the lines pass through it.
	Origin vecmath.Vec3
	// Normal is the direction that is perpendicular to the plane (it need not be a unit vector); the default is +y.
	Normal vecmath.Vec3
	// Spacing is the distance between two minor lines, in world units; the default is 1.
	Spacing float64
	// MajorEvery is how many minor lines make a major one; the default is 10.
	MajorEvery int
	// MinorColor and MajorColor are the colors of the lines; the defaults are the one of near black and near white that contrasts with
	// the background, at 18% and 40% of its opacity.
	MinorColor, MajorColor foundation.Opt[ui.Color]
	// LineWidth is the width of a minor line in logical pixels (a major line is twice as wide); the default is 1.
	LineWidth float64
	// FadeDistance is the distance from the eye at which the lines are gone, in world units; they start to fade at 40% of it. The
	// default is 200 times the spacing. An orthographic view has no fade.
	FadeDistance float64
}

const (
	defaultGridMajorEvery = 10
	defaultGridFade       = 200
	gridFadeStart         = 0.4
)

// resolvedGrid is a GridStyle with its defaults filled in.
type resolvedGrid struct {
	show         bool
	origin       vecmath.Vec3
	normal       vecmath.Vec3
	u, v         vecmath.Vec3
	spacing      float64
	majorEvery   int
	minor, major ui.Color
	width        float64
	fade         float64
}

// resolve fills in the defaults of the style, for the background of the scene.
func (g GridStyle) resolve(background ui.Color) resolvedGrid {
	r := resolvedGrid{show: g.Show, origin: g.Origin, normal: g.Normal, spacing: g.Spacing, majorEvery: g.MajorEvery, width: g.LineWidth, fade: g.FadeDistance}
	if !r.show {
		return r
	}
	if r.normal.LengthSq() == 0 || !r.normal.IsFinite() {
		r.normal = vecmath.Vec3Y
	}
	r.normal = r.normal.Normalize()
	if r.spacing <= 0 || !isFinite(r.spacing) {
		r.spacing = 1
	}
	if r.majorEvery < 1 {
		r.majorEvery = defaultGridMajorEvery
	}
	if r.width <= 0 {
		r.width = defaultLineWidth
	}
	if r.fade <= 0 || !isFinite(r.fade) {
		r.fade = defaultGridFade * r.spacing
	}
	ink := defaultEdgeColor
	if background.ComputeLuminance() < 0.4 {
		ink = lightEdgeColor
	}
	r.minor = g.MinorColor.OrElse(withOpacity(ink, 0.18))
	r.major = g.MajorColor.OrElse(withOpacity(ink, 0.4))
	r.u, r.v = gridAxes(r.normal)
	return r
}

// withOpacity is the color with its alpha multiplied by a factor.
func withOpacity(c ui.Color, factor float64) ui.Color {
	return c.WithOpacity(float64(c.A) * factor)
}

// gridAxes are the two axes of the grid's plane: the first is the projection of x on the plane (of y when the plane is x's), so
// that a grid on a plane of the world's axes has its lines along them.
func gridAxes(n vecmath.Vec3) (u, v vecmath.Vec3) {
	ref := vecmath.Vec3X
	if math.Abs(n.X) > 0.9 {
		ref = vecmath.Vec3Y
	}
	u = ref.Sub(n.Scale(ref.Dot(n))).Normalize()
	return u, n.Cross(u)
}

// period is the distance after which the pattern of the grid repeats, along each axis.
func (g *resolvedGrid) period() float64 { return g.spacing * float64(g.majorEvery) }

// putGrid writes the record of the grid (grid.wgsl) at the offset of the uniform scratch: the plane, its axes and the eye's
// coordinates on them, relative to the eye in float64 (D30).
func putGrid(b []byte, off int, g *resolvedGrid, rot *vecmath.Mat4, eye vecmath.Vec3, dpr float64) {
	n := rot.TransformDirection(g.normal)
	d := -g.normal.Dot(g.origin.Sub(eye))
	u, v := rot.TransformDirection(g.u), rot.TransformDirection(g.v)
	rel := eye.Sub(g.origin)
	p := g.period()
	eu, ev := math.Mod(rel.Dot(g.u), p), math.Mod(rel.Dot(g.v), p)
	putVec4(b, off, float32(n.X), float32(n.Y), float32(n.Z), float32(d))
	putVec4(b, off+16, float32(u.X), float32(u.Y), float32(u.Z), float32(eu))
	putVec4(b, off+32, float32(v.X), float32(v.Y), float32(v.Z), float32(ev))
	putVec4(b, off+48, float32(g.spacing), float32(g.majorEvery), float32(gridFadeStart*g.fade), float32(g.fade))
	putVec4(b, off+64, float32(g.minor.R), float32(g.minor.G), float32(g.minor.B), float32(g.minor.A))
	putVec4(b, off+80, float32(g.major.R), float32(g.major.G), float32(g.major.B), float32(g.major.A))
	putVec4(b, off+96, float32(g.width*dpr), 0, 0, 0)
}

// drawGrid draws the grid into the pass, from its record.
func (r *Renderer) drawGrid(pass *gpu.RenderPass, base pipeKey, record int) {
	pk := base
	pk.kind = pipeGrid
	pipe := r.pipeline(pk)
	if pipe == nil {
		return
	}
	pass.SetPipeline(pipe)
	pass.SetBindGroup(1, r.sectionGroup, uint32(itemBase+record*uniformAlign))
	pass.Draw(3, 1, 0, 0)
	r.stats.GridDraws++
}
