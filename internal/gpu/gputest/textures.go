package gputest

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// The scenes of step 30.5 for the backends that draw a registered texture (SPEC §27.4): a small texture is shown
// under a rotated, clipped, half transparent layer, and the frame must match what the software renderer draws for
// the same pixels as an image.

// TextureSize is the side of the textures of the texture tests, in texels.
const TextureSize = 8

// TextureWindow is the side of the window of the texture tests, in pixels.
const TextureWindow = 128

// TextureKind says how a texture of the tests stores its texels.
type TextureKind int

// The kinds of texture of the tests.
const (
	// TextureRGBA8 is RGBA8Unorm with sRGB-encoded values.
	TextureRGBA8 TextureKind = iota
	// TextureBGRA8 is BGRA8Unorm with sRGB-encoded values.
	TextureBGRA8
	// TextureRGBA16F is RGBA16Float with linear values.
	TextureRGBA16F
)

// TexelBytes is the size of a texel of the kind.
func (k TextureKind) TexelBytes() int {
	if k == TextureRGBA16F {
		return 8
	}
	return 4
}

// TextureCase is one texture of the tests, how it is sampled, and how far the GPU may be from the software image.
type TextureCase struct {
	Name      string
	Kind      TextureKind
	Quality   ui.FilterQuality
	Tolerance Tolerance
	// Texels is the content of the texture, in the layout of its format.
	Texels []byte
	// Reference is the same picture as sRGB-encoded RGBA8 pixels, which the software renderer draws as an image.
	Reference []byte
}

// TextureCases returns the textures of the tests.
func TextureCases() []TextureCase {
	pattern := patternRGBA()
	bgra := make([]byte, len(pattern))
	for i := 0; i < len(pattern); i += 4 {
		bgra[i], bgra[i+1], bgra[i+2], bgra[i+3] = pattern[i+2], pattern[i+1], pattern[i], pattern[i+3]
	}
	half, halfRef := gradientHalf()
	return []TextureCase{
		{"8 bit, nearest", TextureRGBA8, ui.FilterQualityNone, Tolerance{Over8Permille: 20, Over32Permille: 5}, pattern, pattern},
		{"8 bit, bilinear", TextureRGBA8, ui.FilterQualityLow, Tolerance{Over8Permille: 30, Over32Permille: 10}, pattern, pattern},
		{"BGRA, nearest", TextureBGRA8, ui.FilterQualityNone, Tolerance{Over8Permille: 20, Over32Permille: 5}, bgra, pattern},
		{"half float, linear values, nearest", TextureRGBA16F, ui.FilterQualityNone, Tolerance{Over8Permille: 20, Over32Permille: 5}, half, halfRef},
	}
}

// halfOf converts a float32 to the bits of a half float (zero and normal values only: the tests use 0..1).
func halfOf(f float32) uint16 {
	if f == 0 {
		return 0
	}
	b := math.Float32bits(f)
	exp := int((b>>23)&0xff) - 127 + 15
	return uint16(b>>16)&0x8000 | uint16(exp)<<10 | uint16((b>>13)&0x3ff)
}

// srgbOf encodes a linear value as sRGB, in 0..255.
func srgbOf(v float64) byte {
	if v <= 0.0031308 {
		return byte(math.Round(v * 12.92 * 255))
	}
	return byte(math.Round((1.055*math.Pow(v, 1/2.4) - 0.055) * 255))
}

// patternRGBA is a checker with gradients, as RGBA8 texels.
func patternRGBA() []byte {
	pix := make([]byte, TextureSize*TextureSize*4)
	for y := 0; y < TextureSize; y++ {
		for x := 0; x < TextureSize; x++ {
			i := (y*TextureSize + x) * 4
			if (x+y)%2 == 0 {
				pix[i], pix[i+1], pix[i+2] = 230, byte(30+x*25), 40
			} else {
				pix[i], pix[i+1], pix[i+2] = 30, byte(60+y*20), byte(120+x*15)
			}
			pix[i+3] = 255
		}
	}
	return pix
}

// gradientHalf is a gradient of linear values as RGBA16Float texels, and the sRGB-encoded pixels it must look like.
func gradientHalf() (texels, reference []byte) {
	texels = make([]byte, TextureSize*TextureSize*8)
	reference = make([]byte, TextureSize*TextureSize*4)
	for y := 0; y < TextureSize; y++ {
		for x := 0; x < TextureSize; x++ {
			v := float32(x) / (TextureSize - 1)
			for k, c := range [4]float32{v, 1 - v, 0.25, 1} {
				h := halfOf(c)
				i := (y*TextureSize + x) * 8
				texels[i+k*2], texels[i+k*2+1] = byte(h), byte(h>>8)
				if k < 3 {
					reference[(y*TextureSize+x)*4+k] = srgbOf(float64(c))
				} else {
					reference[(y*TextureSize+x)*4+k] = 255
				}
			}
		}
	}
	return texels, reference
}

// layeredScene is a background, and under a rotated, clipped, half transparent layer whatever add draws.
func layeredScene(add func(b *ui.SceneBuilder)) *ui.Scene {
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, PictureOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, TextureWindow, TextureWindow), FillPaint(0xFFE8E8F0))
		c.DrawRect(ui.NewRectFromLTWH(0, 40, TextureWindow, 16), FillPaint(0xFF208040))
	}))
	m := ui.NewMatrix4Identity()
	m.Translate(64, 64, 0)
	m.RotateZ(0.3)
	m.Translate(-64, -64, 0)
	b.PushTransform(m)
	b.PushClipRect(ui.NewRectFromLTWH(20, 20, 80, 70))
	b.PushOpacity(160)
	add(b)
	b.Pop()
	b.Pop()
	b.Pop()
	return b.Build()
}

// TextureScene shows the registered texture in the layers of the tests.
func TextureScene(id int64, quality ui.FilterQuality) *ui.Scene {
	return layeredScene(func(b *ui.SceneBuilder) {
		b.AddTexture(id, ui.AddTextureOpts{Offset: ui.Offset{Dx: 10, Dy: 10}, Width: 96, Height: 96, FilterQuality: quality})
	})
}

// ImageScene draws the pixels as an image in the layers of the tests: what TextureScene must look like.
func ImageScene(pix []byte, quality ui.FilterQuality) *ui.Scene {
	img := ui.NewImageFromRGBA(pix, TextureSize, TextureSize)
	return layeredScene(func(b *ui.SceneBuilder) {
		b.AddPicture(ui.Offset{}, PictureOf(func(c *ui.Canvas) {
			p := ui.NewPaint()
			p.SetFilterQuality(quality)
			c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, TextureSize, TextureSize), ui.NewRectFromLTWH(10, 10, 96, 96), p)
		}))
	})
}
