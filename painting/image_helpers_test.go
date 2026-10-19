package painting

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"
)

// pngBytes encodes a w x h image filled with the color.
func pngBytes(t testing.TB, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// gifBytes encodes an animation of 4x4 frames of the colors, each shown for the
// delay in hundredths of a second.
func gifBytes(t testing.TB, delay int, loopCount int, colors ...color.Color) []byte {
	t.Helper()
	g := &gif.GIF{LoopCount: loopCount}
	for _, c := range colors {
		pal := color.Palette{color.Black, color.White, c}
		img := image.NewPaletted(image.Rect(0, 0, 4, 4), pal)
		for i := range img.Pix {
			img.Pix[i] = 2
		}
		g.Image = append(g.Image, img)
		g.Delay = append(g.Delay, delay)
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
