package viewport3d

import (
	"log/slog"
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d/shaders"
)

// ShadowStyle is the shadow map of the PBR materials: one directional light casts the shadows of every triangle item onto the PBR
// items (SPEC §28.5).
//
// Flutter: no counterpart
type ShadowStyle struct {
	// Enabled draws the shadow of the light.
	Enabled bool
	// Light is the index in SceneStyle.Lights of the light that casts the shadow; the default is the first.
	Light int
	// Size is the side of the shadow map in texels, 256 to 8192; the default is 2048.
	Size int
	// Strength is how dark the shadow is, from 0 to 1; the default is 1.
	Strength foundation.Opt[float64]
}

const (
	defaultShadowSize = 2048
	minShadowSize     = 256
	maxShadowSize     = 8192
)

// resolvedShadow is a ShadowStyle with its defaults.
type resolvedShadow struct {
	enabled  bool
	light    int
	size     int
	strength float64
}

func (s ShadowStyle) resolve() resolvedShadow {
	r := resolvedShadow{enabled: s.Enabled, light: max(s.Light, 0), size: s.Size, strength: min(max(s.Strength.OrElse(1), 0), 1)}
	if r.size == 0 {
		r.size = defaultShadowSize
	}
	r.size = min(max(r.size, minShadowSize), maxShadowSize)
	return r
}

// shadowState is the shadow map of a renderer and the light that was fitted to the scene for the frame.
type shadowState struct {
	// color is the map (the depth of the nearest surface as a float) and depth the buffer that tests while it is drawn.
	color, depth *gpu.Texture
	size         int
	// dummy is the 1 by 1 map that the PBR shader is given when there is no shadow.
	dummy *gpu.Texture
	// group is the group of the joint matrices and the morph weights for the shadow pipelines.
	group *gpu.BindGroup
}

// shadowLight is the light of a frame fitted to the scene: lightWorld takes a position of the world to the clip space of the
// shadow map, relative to the center of the fit (float64), and the other numbers are for the shader.
type shadowLight struct {
	lightWorld vecmath.Mat4
	center     vecmath.Vec3
	rotation   vecmath.Mat4
	radius     float64
	// bias and offset are the depth bias in the units of the clip space's depth and the offset along the normal in world units.
	bias, offset float64
	index        int
}

// fitShadow fits the orthographic box of the light to the sphere of the scene's bounds. The sphere does not move with the camera,
// its radius changes by steps (a quarter of a power of two) and its center is snapped to the texels of the map, so that a shadow
// does not shimmer while the camera moves or an item moves a little. It returns false when there is nothing to shadow.
func (r *Renderer) fitShadow(style *resolvedStyle, size int) (shadowLight, bool) {
	var s shadowLight
	if !style.shadow.enabled || style.shadow.light >= len(style.lights) {
		return s, false
	}
	b := r.scene.Bounds()
	if b.IsEmpty() {
		return s, false
	}
	dir := style.lights[style.shadow.light].Direction
	if dir.LengthSq() == 0 {
		return s, false
	}
	dir = dir.Normalize()
	center := b.Min.Add(b.Max).Scale(0.5)
	radius := max(b.Max.Sub(b.Min).Length()/2, 1e-6)
	radius = math.Exp2(math.Ceil(math.Log2(radius)*4) / 4)
	up := vecmath.Vec3{Y: 1}
	if math.Abs(dir.Y) > 0.99 {
		up = vecmath.Vec3{X: 1}
	}
	rot := vecmath.LookAt(vecmath.Vec3{}, dir.Neg(), up)
	texel := 2 * radius / float64(size)
	lc := rot.TransformPoint(center)
	lc.X, lc.Y = math.Round(lc.X/texel)*texel, math.Round(lc.Y/texel)*texel
	inv, _ := rot.Inverse()
	center = inv.TransformPoint(lc)
	// The depth is 0 at the near side and 1 at the far side: vecmath.Orthographic is reversed, so its planes are swapped.
	proj := vecmath.Orthographic(-radius, radius, -radius, radius, radius, -radius)
	s.lightWorld = proj.Mul(rot).Mul(vecmath.NewMat4Translation(center.Neg()))
	s.center, s.rotation, s.radius = center, rot, radius
	s.bias = 2 * texel / (2 * radius) * 1.5
	s.offset = 1.5 * texel
	s.index = style.shadow.light + 1
	return s, true
}

// ensureShadowTargets makes the map and its depth buffer of the size.
func (r *Renderer) ensureShadowTargets(size int) bool {
	s := &r.pbr.shadow
	if s.color != nil && s.size == size {
		return true
	}
	r.releaseShadowMap()
	var err error
	ext := gpu.Extent3D{Width: size, Height: size, DepthOrArrayLayers: 1}
	if s.color, err = r.dev.NewTexture(gpu.TextureDesc{Label: "viewport3d shadow map", Size: ext, Format: gpu.TextureFormatR32Float,
		Usage: gpu.TextureUsageRenderAttachment | gpu.TextureUsageTextureBinding}); err != nil {
		slog.Error("viewport3d: the shadow map could not be made", "err", err)
		return false
	}
	if s.depth, err = r.dev.NewTexture(gpu.TextureDesc{Label: "viewport3d shadow depth", Size: ext, Format: gpu.TextureFormatDepth32Float,
		Usage: gpu.TextureUsageRenderAttachment}); err != nil {
		slog.Error("viewport3d: the shadow map could not be made", "err", err)
		r.releaseShadowMap()
		return false
	}
	s.size = size
	return true
}

// releaseShadowMap lets go of the map and its depth; the groups that name it are made again with the next frame.
func (r *Renderer) releaseShadowMap() {
	s := &r.pbr.shadow
	for _, t := range []*gpu.Texture{s.color, s.depth} {
		if t != nil {
			r.later(t)
		}
	}
	s.color, s.depth, s.size = nil, nil, 0
	r.dropPBRGroup()
}

// ensureShadowModule makes the module of the shadow pipelines.
func (r *Renderer) ensureShadowModule() error {
	if r.mods.shadow != nil {
		return nil
	}
	var err error
	r.mods.shadow, err = r.dev.NewShaderModule(shaders.Shadow)
	return err
}

// shadowCaster is an item of the shadow pass: its mesh and the matrix that takes it to the clip space of the light.
type shadowCaster struct {
	mesh  *meshGPU
	full  bool
	mv    [16]float32
	extra [4]float32
	// instance is its place in the instance buffer.
	instance int32
}
