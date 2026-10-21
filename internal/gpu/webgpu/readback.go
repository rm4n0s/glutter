package webgpu

import (
	"image"
	"image/color"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// unpackRows converts the bytes of a mapped read-back buffer, w x h pixels of the format with rows
// pitch bytes apart, into an image, top row first, as the Vulkan backend does: BGRA is swapped
// and a one-channel texture becomes gray.
func unpackRows(raw []byte, pitch, w, h int, format explicit.TextureFormat) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bpp := format.BytesPerPixel()
	for y := 0; y < h; y++ {
		row := raw[y*pitch:]
		dst := img.Pix[y*img.Stride : y*img.Stride+4*w]
		for x := 0; x < w; x++ {
			var c color.RGBA
			switch format {
			case explicit.TextureFormatBGRA8Unorm:
				c = color.RGBA{row[x*bpp+2], row[x*bpp+1], row[x*bpp], row[x*bpp+3]}
			case explicit.TextureFormatR8Unorm:
				c = color.RGBA{row[x], row[x], row[x], 255}
			default:
				c = color.RGBA{row[x*bpp], row[x*bpp+1], row[x*bpp+2], row[x*bpp+3]}
			}
			dst[4*x], dst[4*x+1], dst[4*x+2], dst[4*x+3] = c.R, c.G, c.B, c.A
		}
	}
	return img
}
