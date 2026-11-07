package viewport3d

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// Material says how the items that use it are drawn (SPEC §27.8). It is one of the types of this package: UnlitMaterial and
// PhongMaterial for meshes of triangles, LineMaterial for meshes of lines and PointMaterial for meshes of points. A material
// is shared by its items, and changing it (its setters) draws every view of a scene that uses it again.
//
// Flutter: no counterpart
type Material interface {
	foundation.Listenable
	// Release frees what the material holds on the GPU, in every renderer that drew it; a renderer makes it again if an item
	// still uses the material.
	Release()

	core() *materialCore
	primitive() Primitive
}

// materialCore is what every material shares: its listeners, a version that the renderers compare, and the renderers that
// hold GPU resources of it.
type materialCore struct {
	foundation.ChangeNotifier
	version uint64
	// renderers are the renderers that hold GPU resources of the material, and states what they hold.
	renderers []*Renderer
	states    []*matGPU
}

func (c *materialCore) core() *materialCore { return c }

// changed bumps the version and tells the listeners.
func (c *materialCore) changed() {
	c.version++
	c.NotifyListeners()
}

// Release frees the GPU resources that renderers hold of the material.
//
// Flutter: no counterpart
func (c *materialCore) Release() {
	for len(c.renderers) > 0 {
		c.renderers[0].forgetMaterial(c)
	}
}

// Defaults of the materials.
var (
	defaultSurfaceColor = ui.NewColor(0xFFB4BCC8)
	defaultLineColor    = ui.NewColor(0xFF101418)
	defaultPointColor   = ui.NewColor(0xFFE05030)
)

const (
	defaultShininess = 32
	defaultLineWidth = 1.0
	defaultPointSize = 4.0
)

// surfaceMaterial is what UnlitMaterial and PhongMaterial share: a color, an optional texture and the faces' settings.
type surfaceMaterial struct {
	materialCore
	color        ui.Color
	texture      *gpu.Texture
	vertexColors bool
	doubleSided  bool
	transparent  bool
	lit          bool
	specular     ui.Color
	shininess    float64
}

func (m *surfaceMaterial) primitive() Primitive { return PrimitiveTriangles }

// Color is the color of the surface (before the texture, the vertex colors and the light).
//
// Flutter: no counterpart
func (m *surfaceMaterial) Color() ui.Color { return m.color }

// SetColor changes the color.
//
// Flutter: no counterpart
func (m *surfaceMaterial) SetColor(c ui.Color) {
	if m.color != c {
		m.color = c
		m.changed()
	}
}

// Texture is the texture that the color is multiplied by, or nil. It must be a 2D RGBA8Unorm texture (or any that a
// filtering sampler can read) of the device that draws, with the usage TextureBinding.
//
// Flutter: no counterpart
func (m *surfaceMaterial) Texture() *gpu.Texture { return m.texture }

// SetTexture changes the texture; nil removes it. The material does not own the texture: the app releases it.
//
// Flutter: no counterpart
func (m *surfaceMaterial) SetTexture(t *gpu.Texture) {
	if m.texture != t {
		m.texture = t
		m.changed()
	}
}

// VertexColors says whether the colors of the mesh's vertices multiply the color.
//
// Flutter: no counterpart
func (m *surfaceMaterial) VertexColors() bool { return m.vertexColors }

// SetVertexColors changes whether the colors of the mesh's vertices are used. A mesh without colors has white ones.
//
// Flutter: no counterpart
func (m *surfaceMaterial) SetVertexColors(on bool) {
	if m.vertexColors != on {
		m.vertexColors = on
		m.changed()
	}
}

// DoubleSided says whether the back faces are drawn too (lit as seen from behind). Otherwise they are culled.
//
// Flutter: no counterpart
func (m *surfaceMaterial) DoubleSided() bool { return m.doubleSided }

// SetDoubleSided changes whether the back faces are drawn.
//
// Flutter: no counterpart
func (m *surfaceMaterial) SetDoubleSided(on bool) {
	if m.doubleSided != on {
		m.doubleSided = on
		m.changed()
	}
}

// Transparent says whether the items are drawn with blending, back to front, after the opaque ones: set by the option, or
// because the color is not opaque.
//
// Flutter: no counterpart
func (m *surfaceMaterial) Transparent() bool { return m.transparent || m.color.A < 1 }

// SetTransparent changes whether the items are drawn with blending even when the color is opaque (a texture or the vertex
// colors may have alpha).
//
// Flutter: no counterpart
func (m *surfaceMaterial) SetTransparent(on bool) {
	if m.transparent != on {
		m.transparent = on
		m.changed()
	}
}

// UnlitMaterial draws a mesh of triangles with its color (or texture, or vertex colors) as it is, with no light.
//
// Flutter: no counterpart
type UnlitMaterial struct {
	surfaceMaterial
}

// UnlitOpts are the optional parameters of NewUnlitMaterial.
//
// Flutter: no counterpart
type UnlitOpts struct {
	// Color is the color; the zero value stands for the default, a light gray.
	Color ui.Color
	// Texture multiplies the color; optional.
	Texture *gpu.Texture
	// VertexColors multiplies the color by the mesh's vertex colors.
	VertexColors bool
	// DoubleSided draws the back faces too.
	DoubleSided bool
	// Transparent draws the items with blending even when the color is opaque.
	Transparent bool
}

// NewUnlitMaterial creates a material that shows a color as it is.
//
// Flutter: no counterpart
func NewUnlitMaterial(opts ...UnlitOpts) *UnlitMaterial {
	o := firstOpts(opts)
	m := &UnlitMaterial{}
	m.color, m.texture, m.vertexColors, m.doubleSided, m.transparent = surfaceColor(o.Color), o.Texture, o.VertexColors, o.DoubleSided, o.Transparent
	return m
}

// PhongMaterial draws a mesh of triangles with Blinn-Phong shading: a light at the camera, the scene's directional lights and
// its ambient term (SPEC §27.8; the CAD default). A mesh without normals is shaded flat.
//
// Flutter: no counterpart
type PhongMaterial struct {
	surfaceMaterial
}

// PhongOpts are the optional parameters of NewPhongMaterial.
//
// Flutter: no counterpart
type PhongOpts struct {
	// Color is the color; the zero value stands for the default, a light gray.
	Color ui.Color
	// Specular is the color of the highlight; the zero value stands for a weak white.
	Specular ui.Color
	// Shininess is the exponent of the highlight; the default is 32.
	Shininess float64
	// Texture multiplies the color; optional.
	Texture *gpu.Texture
	// VertexColors multiplies the color by the mesh's vertex colors.
	VertexColors bool
	// DoubleSided draws the back faces too, lit as seen from behind.
	DoubleSided bool
	// Transparent draws the items with blending even when the color is opaque.
	Transparent bool
}

// NewPhongMaterial creates a lit material.
//
// Flutter: no counterpart
func NewPhongMaterial(opts ...PhongOpts) *PhongMaterial {
	o := firstOpts(opts)
	m := &PhongMaterial{}
	m.lit = true
	m.color, m.texture, m.vertexColors, m.doubleSided, m.transparent = surfaceColor(o.Color), o.Texture, o.VertexColors, o.DoubleSided, o.Transparent
	m.specular = o.Specular
	if m.specular == (ui.Color{}) {
		m.specular = ui.NewColor(0x55FFFFFF)
	}
	m.shininess = o.Shininess
	if m.shininess <= 0 {
		m.shininess = defaultShininess
	}
	return m
}

// surfaceColor is the color of a surface material for the option's: the default for the zero value.
func surfaceColor(c ui.Color) ui.Color {
	if c == (ui.Color{}) {
		return defaultSurfaceColor
	}
	return c
}

// Specular is the color of the highlight.
//
// Flutter: no counterpart
func (m *PhongMaterial) Specular() ui.Color { return m.specular }

// SetSpecular changes the color of the highlight.
//
// Flutter: no counterpart
func (m *PhongMaterial) SetSpecular(c ui.Color) {
	if m.specular != c {
		m.specular = c
		m.changed()
	}
}

// Shininess is the exponent of the highlight.
//
// Flutter: no counterpart
func (m *PhongMaterial) Shininess() float64 { return m.shininess }

// SetShininess changes the exponent of the highlight (above 0).
//
// Flutter: no counterpart
func (m *PhongMaterial) SetShininess(s float64) {
	if s > 0 && m.shininess != s {
		m.shininess = s
		m.changed()
	}
}

// maxStipple is how many lengths a dash pattern has: dashes and gaps in turn.
const maxStipple = 8

// LineMaterial draws a mesh of lines as anti-aliased lines of a width in logical pixels, with an optional dash pattern (SPEC
// §27.8). The lines are expanded into quads by the vertex shader, so the width does not change with the distance.
//
// Flutter: no counterpart
type LineMaterial struct {
	materialCore
	color   ui.Color
	width   float64
	stipple [maxStipple]float64
	dashes  int
	// pull is how far the lines are pulled towards the camera, relative to their depth: only the renderer's edges are.
	pull float32
}

// LineOpts are the optional parameters of NewLineMaterial.
//
// Flutter: no counterpart
type LineOpts struct {
	// Color is the color; the zero value stands for the default, near black.
	Color ui.Color
	// Width is the width in logical pixels; the default is 1.
	Width float64
	// Stipple is a dash pattern in logical pixels: a dash, a gap, a dash, a gap, ... (up to 8 lengths; an odd number repeats
	// the pattern with the roles swapped, as in SVG). The default is a solid line. The pattern runs along a polyline as the
	// segments of the mesh share ends, measured in the screen's pixels at the first end of each segment.
	Stipple []float64
}

// NewLineMaterial creates a line material.
//
// Flutter: no counterpart
func NewLineMaterial(opts ...LineOpts) *LineMaterial {
	o := firstOpts(opts)
	m := &LineMaterial{color: o.Color, width: o.Width}
	if m.color == (ui.Color{}) {
		m.color = defaultLineColor
	}
	if m.width <= 0 {
		m.width = defaultLineWidth
	}
	m.setStipple(o.Stipple)
	return m
}

func (m *LineMaterial) primitive() Primitive { return PrimitiveLines }

// setStipple stores a dash pattern; a pattern that has no length or a length that is not positive is solid.
func (m *LineMaterial) setStipple(p []float64) {
	m.stipple, m.dashes = [maxStipple]float64{}, 0
	sum := 0.0
	n := len(p)
	if n > maxStipple {
		n = maxStipple
	}
	for i := 0; i < n; i++ {
		if p[i] < 0 || math.IsNaN(p[i]) || math.IsInf(p[i], 0) {
			return
		}
		sum += p[i]
	}
	if sum <= 0 {
		return
	}
	copy(m.stipple[:], p[:n])
	m.dashes = n
	if n%2 == 1 && n*2 <= maxStipple {
		// An odd pattern repeats with the roles swapped: written out it is twice as long.
		copy(m.stipple[n:], p[:n])
		m.dashes = 2 * n
	}
}

// Color is the color of the lines.
//
// Flutter: no counterpart
func (m *LineMaterial) Color() ui.Color { return m.color }

// SetColor changes the color.
//
// Flutter: no counterpart
func (m *LineMaterial) SetColor(c ui.Color) {
	if m.color != c {
		m.color = c
		m.changed()
	}
}

// Width is the width of the lines in logical pixels.
//
// Flutter: no counterpart
func (m *LineMaterial) Width() float64 { return m.width }

// SetWidth changes the width (above 0).
//
// Flutter: no counterpart
func (m *LineMaterial) SetWidth(w float64) {
	if w > 0 && m.width != w {
		m.width = w
		m.changed()
	}
}

// SetStipple changes the dash pattern; nil makes the lines solid.
//
// Flutter: no counterpart
func (m *LineMaterial) SetStipple(p []float64) {
	m.setStipple(p)
	m.changed()
}

// PointShape is the shape of a point.
//
// Flutter: no counterpart
type PointShape int

const (
	// PointShapeUnspecified is the zero value: round.
	PointShapeUnspecified PointShape = iota
	// PointShapeRound draws a disc.
	PointShapeRound
	// PointShapeSquare draws a square.
	PointShapeSquare
)

// PointMaterial draws a mesh of points as round or square marks of a size in logical pixels (SPEC §27.8).
//
// Flutter: no counterpart
type PointMaterial struct {
	materialCore
	color        ui.Color
	size         float64
	shape        PointShape
	vertexColors bool
}

// PointOpts are the optional parameters of NewPointMaterial.
//
// Flutter: no counterpart
type PointOpts struct {
	// Color is the color; the zero value stands for the default, orange.
	Color ui.Color
	// Size is the diameter or the side in logical pixels; the default is 4.
	Size float64
	// Shape is round or square; the default is round.
	Shape PointShape
	// VertexColors multiplies the color by the colors of the mesh's points (MeshData.Colors), as a scanned cloud has them; a mesh
	// without colors is drawn in the color alone.
	VertexColors bool
}

// NewPointMaterial creates a point material.
//
// Flutter: no counterpart
func NewPointMaterial(opts ...PointOpts) *PointMaterial {
	o := firstOpts(opts)
	m := &PointMaterial{color: o.Color, size: o.Size, shape: o.Shape, vertexColors: o.VertexColors}
	if m.color == (ui.Color{}) {
		m.color = defaultPointColor
	}
	if m.size <= 0 {
		m.size = defaultPointSize
	}
	if m.shape == PointShapeUnspecified {
		m.shape = PointShapeRound
	}
	return m
}

func (m *PointMaterial) primitive() Primitive { return PrimitivePoints }

// Color is the color of the points.
//
// Flutter: no counterpart
func (m *PointMaterial) Color() ui.Color { return m.color }

// SetColor changes the color.
//
// Flutter: no counterpart
func (m *PointMaterial) SetColor(c ui.Color) {
	if m.color != c {
		m.color = c
		m.changed()
	}
}

// Size is the size of the points in logical pixels.
//
// Flutter: no counterpart
func (m *PointMaterial) Size() float64 { return m.size }

// SetSize changes the size (above 0).
//
// Flutter: no counterpart
func (m *PointMaterial) SetSize(s float64) {
	if s > 0 && m.size != s {
		m.size = s
		m.changed()
	}
}

// VertexColors says whether the colors of the mesh's points multiply the color.
//
// Flutter: no counterpart
func (m *PointMaterial) VertexColors() bool { return m.vertexColors }

// SetVertexColors changes whether the colors of the mesh's points are used. A mesh without colors is drawn in the color alone.
//
// Flutter: no counterpart
func (m *PointMaterial) SetVertexColors(on bool) {
	if m.vertexColors != on {
		m.vertexColors = on
		m.changed()
	}
}

// Shape is round or square.
//
// Flutter: no counterpart
func (m *PointMaterial) Shape() PointShape { return m.shape }

// SetShape changes the shape.
//
// Flutter: no counterpart
func (m *PointMaterial) SetShape(s PointShape) {
	if s == PointShapeUnspecified {
		s = PointShapeRound
	}
	if m.shape != s {
		m.shape = s
		m.changed()
	}
}
