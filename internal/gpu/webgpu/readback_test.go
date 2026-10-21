package webgpu

import (
	"image/color"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

func TestUnpackRows(t *testing.T) {
	// 2 x 2 pixels, rows 12 bytes apart (the rest of a row is padding that must not be read).
	rgba := []byte{
		1, 2, 3, 4, 5, 6, 7, 8, 0xEE, 0xEE, 0xEE, 0xEE,
		9, 10, 11, 12, 13, 14, 15, 16, 0xEE, 0xEE, 0xEE, 0xEE,
	}
	gray := []byte{
		10, 20, 0xEE,
		30, 40, 0xEE,
	}
	tests := []struct {
		name   string
		raw    []byte
		pitch  int
		format explicit.TextureFormat
		want   [4]color.RGBA // row by row
	}{
		{"rgba", rgba, 12, explicit.TextureFormatRGBA8Unorm,
			[4]color.RGBA{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}, {13, 14, 15, 16}}},
		{"bgra is swapped", rgba, 12, explicit.TextureFormatBGRA8Unorm,
			[4]color.RGBA{{3, 2, 1, 4}, {7, 6, 5, 8}, {11, 10, 9, 12}, {15, 14, 13, 16}}},
		{"one channel is gray and opaque", gray, 3, explicit.TextureFormatR8Unorm,
			[4]color.RGBA{{10, 10, 10, 255}, {20, 20, 20, 255}, {30, 30, 30, 255}, {40, 40, 40, 255}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := unpackRows(tt.raw, tt.pitch, 2, 2, tt.format)
			if b := img.Bounds(); b.Dx() != 2 || b.Dy() != 2 {
				t.Fatalf("size = %v", b)
			}
			for i, want := range tt.want {
				if got := img.RGBAAt(i%2, i/2); got != want {
					t.Errorf("pixel %d = %v, want %v", i, got, want)
				}
			}
		})
	}
}
