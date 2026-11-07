package model

import "math"

// Wrap is how a texture is read outside of 0 to 1.
//
// Flutter: no counterpart
type Wrap uint8

const (
	// WrapRepeat repeats the texture; the zero value, and glTF's default.
	WrapRepeat Wrap = iota
	// WrapMirroredRepeat repeats it, every other time mirrored.
	WrapMirroredRepeat
	// WrapClampToEdge repeats the pixels at the edge.
	WrapClampToEdge
)

// SamplerFilter is how a texture is read between its pixels.
//
// Flutter: no counterpart
type SamplerFilter uint8

const (
	// FilterUnspecified lets the renderer choose (linear); the zero value.
	FilterUnspecified SamplerFilter = iota
	// FilterNearest takes the nearest pixel.
	FilterNearest
	// FilterLinear blends the pixels around.
	FilterLinear
)

// Sampler says how a texture is read.
//
// Flutter: no counterpart
type Sampler struct {
	// Mag is the filter when the texture is enlarged, and Min when it is reduced.
	Mag, Min SamplerFilter
	// Mipmap is how the levels of the mipmaps are blended when Min asks for them (a glTF minFilter that has "mipmap" in its name):
	// FilterUnspecified for no mipmaps, FilterNearest to take the nearest level and FilterLinear to blend two.
	Mipmap SamplerFilter
	// WrapU and WrapV are how the texture is read outside of 0 to 1.
	WrapU, WrapV Wrap
}

// Texture is an image with a sampler.
//
// Flutter: no counterpart
type Texture struct {
	// Name is the name in the file, or empty.
	Name string
	// Image is the image, or nil when the file gives it in a form that the loader does not read.
	Image *Image
	// Sampler says how the image is read.
	Sampler Sampler
}

// TextureTransform is the transform of the texture coordinates of KHR_texture_transform: the coordinates are scaled, rotated and
// then moved.
//
// Flutter: no counterpart
type TextureTransform struct {
	// Offset is added to the coordinates after the rotation.
	Offset [2]float32
	// Rotation is the angle in radians, counterclockwise in the image.
	Rotation float32
	// Scale multiplies the coordinates first.
	Scale [2]float32
}

// Apply is the transform of the coordinates (u, v).
//
// Flutter: no counterpart
func (t TextureTransform) Apply(u, v float32) (float32, float32) {
	s, c := math.Sincos(float64(t.Rotation))
	sin, cos := float32(s), float32(c)
	u, v = u*t.Scale[0], v*t.Scale[1]
	return cos*u + sin*v + t.Offset[0], -sin*u + cos*v + t.Offset[1]
}

// TextureRef is the use of a texture by a material.
//
// Flutter: no counterpart
type TextureRef struct {
	// Texture is the texture.
	Texture *Texture
	// TexCoord is the set of texture coordinates that it reads: 0 for Mesh.TexCoords and 1 for Mesh.TexCoords1.
	TexCoord int
	// Transform is the KHR_texture_transform of the use, or nil for none. A transform that gives another texture coordinate set
	// has set TexCoord.
	Transform *TextureTransform
}

// LightType is the kind of a light of KHR_lights_punctual.
//
// Flutter: no counterpart
type LightType uint8

const (
	// LightDirectional shines along the -z of its node, from infinitely far; the zero value.
	LightDirectional LightType = iota
	// LightPoint shines in all directions from its node.
	LightPoint
	// LightSpot shines in a cone along the -z of its node.
	LightSpot
)

// String is the name of the type, as glTF writes it.
//
// Flutter: no counterpart
func (t LightType) String() string {
	switch t {
	case LightDirectional:
		return "directional"
	case LightPoint:
		return "point"
	case LightSpot:
		return "spot"
	}
	return "unknown"
}

// Light is a punctual light that a node carries (KHR_lights_punctual).
//
// Flutter: no counterpart
type Light struct {
	// Name is the name in the file, or empty.
	Name string
	// Type is the kind of the light.
	Type LightType
	// Color is red, green and blue, from 0 to 1, linear; (1, 1, 1) when the file does not say.
	Color [3]float32
	// Intensity is in lux for a directional light and in candela for the others; 1 when the file does not say.
	Intensity float32
	// Range is the distance at which the light has gone, or 0 for no limit (a point or spot light only).
	Range float32
	// InnerConeAngle and OuterConeAngle are the angles in radians of a spot light where it starts to fade and where it ends:
	// 0 and a quarter of pi when the file does not say.
	InnerConeAngle, OuterConeAngle float32
}

// Camera is a camera that a node carries.
//
// Flutter: no counterpart
type Camera struct {
	// Name is the name in the file, or empty.
	Name string
	// Orthographic is true for a parallel projection and false for a perspective one.
	Orthographic bool
	// YFov is the vertical angle of view in radians and AspectRatio the width over the height, or 0 for the aspect of the view;
	// for a perspective camera.
	YFov, AspectRatio float64
	// XMag and YMag are the half width and half height of the view of an orthographic camera.
	XMag, YMag float64
	// ZNear and ZFar are the planes; ZFar is 0 for a perspective camera with an infinite far plane.
	ZNear, ZFar float64
}
