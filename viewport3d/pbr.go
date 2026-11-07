package viewport3d

import (
	"math"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// AlphaMode is how the alpha of a PBRMaterial is used (glTF's alphaMode).
//
// Flutter: no counterpart
type AlphaMode int

const (
	// AlphaModeOpaque ignores the alpha; the zero value.
	AlphaModeOpaque AlphaMode = iota
	// AlphaModeMask draws a pixel whose alpha is at least the cutoff and none otherwise.
	AlphaModeMask
	// AlphaModeBlend blends by the alpha: the items are transparent (sorted back to front, no depth write).
	AlphaModeBlend
)

// TextureTransform is the transform of the texture coordinates of KHR_texture_transform: they are scaled, rotated and moved.
//
// Flutter: no counterpart
type TextureTransform struct {
	// Offset is added to the coordinates after the rotation.
	Offset [2]float32
	// Rotation is the angle in radians, counterclockwise in the image.
	Rotation float32
	// Scale multiplies the coordinates first; the zero value stands for (1, 1).
	Scale [2]float32
}

// matrix is the first two rows of the 3 by 3 matrix of the transform.
func (t *TextureTransform) matrix() [6]float32 {
	if t == nil {
		return [6]float32{1, 0, 0, 0, 1, 0}
	}
	sx, sy := t.Scale[0], t.Scale[1]
	if sx == 0 && sy == 0 {
		sx, sy = 1, 1
	}
	s, c := math.Sincos(float64(t.Rotation))
	sin, cos := float32(s), float32(c)
	return [6]float32{cos * sx, sin * sy, t.Offset[0], -sin * sx, cos * sy, t.Offset[1]}
}

// PBRTexture is a texture that a PBRMaterial reads, with how: its sampler, which set of texture coordinates it takes and an
// optional transform of them. The zero value is no texture.
//
// Flutter: no counterpart
type PBRTexture struct {
	// Texture is a 2D RGBA8Unorm (or RGBA16Float) texture with the usage TextureBinding; the material does not own it. The base
	// color and the emissive textures hold sRGB colors, the others hold numbers, as glTF says.
	Texture *gpu.Texture
	// Sampler says how the texture is read; nil is repeat with linear filtering.
	Sampler *gpu.Sampler
	// TexCoord is the set of texture coordinates of the mesh: 0 (MeshData.TexCoords) or 1 (MeshData.TexCoords1).
	TexCoord int
	// Transform changes the texture coordinates; nil for none.
	Transform *TextureTransform
}

// PBRSlot is the place of a texture in a PBRMaterial.
//
// Flutter: no counterpart
type PBRSlot int

// The textures of a PBRMaterial.
const (
	// PBRSlotBaseColor multiplies the color; sRGB, with the alpha.
	PBRSlotBaseColor PBRSlot = iota
	// PBRSlotMetallicRoughness has the roughness in its green channel and the metalness in its blue one.
	PBRSlotMetallicRoughness
	// PBRSlotNormal has the normals in the tangent space of the surface.
	PBRSlotNormal
	// PBRSlotOcclusion has the occlusion in its red channel.
	PBRSlotOcclusion
	// PBRSlotEmissive multiplies the emissive color; sRGB.
	PBRSlotEmissive

	pbrSlots = 5
)

// PBRMaterial draws a mesh of triangles with the physically based metallic-roughness model of glTF 2.0 (SPEC §28.5): the
// scene's directional lights and the light at the camera, image-based lighting from the scene's environment, one directional
// shadow map, and the base color, metallic-roughness, normal, occlusion and emissive textures. A scene with an item of this
// material is drawn in a float target and tone mapped (SceneStyle.ToneMapping). A mesh with Joints and Weights is skinned when its
// item has a Skin, and a mesh with morph targets follows the item's weights.
//
// Flutter: no counterpart
type PBRMaterial struct {
	materialCore
	color             ui.Color
	metallic          float64
	roughness         float64
	emissive          [3]float64
	tex               [pbrSlots]PBRTexture
	normalScale       float64
	occlusionStrength float64
	alphaMode         AlphaMode
	alphaCutoff       float64
	doubleSided       bool
	vertexColors      bool
	unlit             bool
}

// PBROpts are the optional parameters of NewPBRMaterial.
//
// Flutter: no counterpart
type PBROpts struct {
	// Color is the base color in sRGB with the alpha, as for the other materials; the zero value stands for white.
	Color ui.Color
	// Metallic is 0 for a dielectric and 1 for a metal; the default is 0.
	Metallic foundation.Opt[float64]
	// Roughness is from 0 (a mirror) to 1; the default is 0.5.
	Roughness foundation.Opt[float64]
	// Emissive is the radiance that the surface gives out, red, green and blue in linear units (it may be above 1).
	Emissive [3]float64
	// The textures. A texture of the base color or the emissive color is in sRGB.
	BaseColorTexture, MetallicRoughnessTexture, NormalTexture, OcclusionTexture, EmissiveTexture PBRTexture
	// NormalScale scales the x and y of the normals of the normal texture; the default is 1.
	NormalScale foundation.Opt[float64]
	// OcclusionStrength is how much of the occlusion texture is used, from 0 to 1; the default is 1.
	OcclusionStrength foundation.Opt[float64]
	// AlphaMode is how the alpha is used; the default ignores it.
	AlphaMode AlphaMode
	// AlphaCutoff is the alpha below which AlphaModeMask draws nothing; the default is 0.5.
	AlphaCutoff foundation.Opt[float64]
	// DoubleSided draws the back faces too, lit as seen from behind.
	DoubleSided bool
	// VertexColors multiplies the color by the colors of the mesh's vertices.
	VertexColors bool
	// Unlit shows the color as it is (glTF's KHR_materials_unlit).
	Unlit bool
}

// NewPBRMaterial creates a PBR material.
//
// Flutter: no counterpart
func NewPBRMaterial(opts ...PBROpts) *PBRMaterial {
	o := firstOpts(opts)
	m := &PBRMaterial{
		color: o.Color, metallic: o.Metallic.OrElse(0), roughness: o.Roughness.OrElse(0.5), emissive: o.Emissive,
		normalScale: o.NormalScale.OrElse(1), occlusionStrength: o.OcclusionStrength.OrElse(1),
		alphaMode: o.AlphaMode, alphaCutoff: o.AlphaCutoff.OrElse(0.5),
		doubleSided: o.DoubleSided, vertexColors: o.VertexColors, unlit: o.Unlit,
	}
	if m.color == (ui.Color{}) {
		m.color = ui.NewColor(0xFFFFFFFF)
	}
	m.tex = [pbrSlots]PBRTexture{o.BaseColorTexture, o.MetallicRoughnessTexture, o.NormalTexture, o.OcclusionTexture, o.EmissiveTexture}
	return m
}

func (m *PBRMaterial) primitive() Primitive { return PrimitiveTriangles }

// Color is the base color in sRGB.
//
// Flutter: no counterpart
func (m *PBRMaterial) Color() ui.Color { return m.color }

// SetColor changes the base color.
//
// Flutter: no counterpart
func (m *PBRMaterial) SetColor(c ui.Color) {
	if m.color != c {
		m.color = c
		m.changed()
	}
}

// Metallic is the metalness from 0 to 1.
//
// Flutter: no counterpart
func (m *PBRMaterial) Metallic() float64 { return m.metallic }

// SetMetallic changes the metalness.
//
// Flutter: no counterpart
func (m *PBRMaterial) SetMetallic(v float64) {
	if m.metallic != v {
		m.metallic = v
		m.changed()
	}
}

// Roughness is the roughness from 0 to 1.
//
// Flutter: no counterpart
func (m *PBRMaterial) Roughness() float64 { return m.roughness }

// SetRoughness changes the roughness.
//
// Flutter: no counterpart
func (m *PBRMaterial) SetRoughness(v float64) {
	if m.roughness != v {
		m.roughness = v
		m.changed()
	}
}

// Emissive is the emitted radiance in linear units.
//
// Flutter: no counterpart
func (m *PBRMaterial) Emissive() [3]float64 { return m.emissive }

// SetEmissive changes the emitted radiance.
//
// Flutter: no counterpart
func (m *PBRMaterial) SetEmissive(e [3]float64) {
	if m.emissive != e {
		m.emissive = e
		m.changed()
	}
}

// Texture is the texture of a slot (a PBRTexture with a nil Texture when there is none).
//
// Flutter: no counterpart
func (m *PBRMaterial) Texture(slot PBRSlot) PBRTexture { return m.tex[slot] }

// SetTexture changes the texture of a slot; a PBRTexture with a nil Texture removes it. The material does not own the textures.
//
// Flutter: no counterpart
func (m *PBRMaterial) SetTexture(slot PBRSlot, t PBRTexture) {
	if m.tex[slot] != t {
		m.tex[slot] = t
		m.changed()
	}
}

// AlphaMode is how the alpha is used.
//
// Flutter: no counterpart
func (m *PBRMaterial) AlphaMode() AlphaMode { return m.alphaMode }

// SetAlphaMode changes how the alpha is used.
//
// Flutter: no counterpart
func (m *PBRMaterial) SetAlphaMode(a AlphaMode) {
	if m.alphaMode != a {
		m.alphaMode = a
		m.changed()
	}
}

// DoubleSided says whether the back faces are drawn.
//
// Flutter: no counterpart
func (m *PBRMaterial) DoubleSided() bool { return m.doubleSided }

// SetDoubleSided changes whether the back faces are drawn.
//
// Flutter: no counterpart
func (m *PBRMaterial) SetDoubleSided(on bool) {
	if m.doubleSided != on {
		m.doubleSided = on
		m.changed()
	}
}

// Transparent says whether the items are drawn with blending: in AlphaModeBlend only (glTF ignores the alpha in the other modes).
//
// Flutter: no counterpart
func (m *PBRMaterial) Transparent() bool { return m.alphaMode == AlphaModeBlend }

// srgbToLinear is the sRGB transfer function, decoded.
func srgbToLinear(v float32) float32 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return float32(math.Pow(float64(v+0.055)/1.055, 2.4))
}

// linearToSRGB is the sRGB transfer function.
func linearToSRGB(v float32) float32 {
	if v <= 0.0031308 {
		return v * 12.92
	}
	return float32(1.055*math.Pow(float64(v), 1/2.4) - 0.055)
}

// pbrUniformSize is the size of the uniform data of a PBRMaterial: four vec4 and the ten rows of the texture transforms.
const pbrUniformSize = 4*16 + pbrSlots*32

// pbrUniforms writes the uniform data of a PBR material.
func pbrUniforms(m *PBRMaterial, dst *[matUniformMax]byte) int {
	b := dst[:]
	putVec4(b, 0, srgbToLinear(float32(m.color.R)), srgbToLinear(float32(m.color.G)), srgbToLinear(float32(m.color.B)), float32(m.color.A))
	putVec4(b, 16, float32(m.metallic), float32(m.roughness), float32(m.normalScale), float32(m.occlusionStrength))
	putVec4(b, 32, float32(m.emissive[0]), float32(m.emissive[1]), float32(m.emissive[2]), 0)
	bitsOf := 0.0
	if m.unlit {
		bitsOf += 1
	}
	if m.vertexColors {
		bitsOf += 2
	}
	putVec4(b, 48, float32(m.alphaMode), float32(m.alphaCutoff), float32(bitsOf), 0)
	for i := range m.tex {
		t := &m.tex[i]
		mt := t.Transform.matrix()
		has := float32(0)
		if t.Texture != nil {
			has = 1
		}
		set := float32(0)
		if t.TexCoord == 1 {
			set = 1
		}
		putVec4(b, 64+32*i, mt[0], mt[1], mt[2], set)
		putVec4(b, 64+32*i+16, mt[3], mt[4], mt[5], has)
	}
	return pbrUniformSize
}
