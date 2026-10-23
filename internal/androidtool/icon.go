package androidtool

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/rm4n0s/errors"
)

// DefaultIcon returns Glutter's launcher icon as a PNG: a white ring with a gap and a bar,
// the shape of a "G", on the blue of the Material seed color.
//
// Flutter: no counterpart
func DefaultIcon() ([]byte, error) {
	const size = 192
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	blue := color.NRGBA{R: 0x1a, G: 0x73, B: 0xe8, A: 0xff}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	c := float64(size) / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := float64(x)+0.5-c, float64(y)+0.5-c
			d := math.Hypot(dx, dy)
			px := blue
			angle := math.Atan2(dy, dx) // 0 is to the right
			inRing := d >= 44 && d <= 72
			inGap := angle > -0.9 && angle < -0.05
			inBar := dx >= 4 && dx <= 74 && dy >= -10 && dy <= 10
			if (inRing && !inGap) || inBar {
				px = white
			}
			img.SetNRGBA(x, y, px)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, errors.NewErr(TagAndroidWriteFailed, err).SetMetadata("what", "the default icon")
	}
	return out.Bytes(), nil
}
