package wintool

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/rm4n0s/errors"
)

// squarePNG is a PNG of the size with a red square in the middle half on a blue background.
func squarePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBA{B: 0xff, A: 0xff}
			if x >= w/4 && x < w*3/4 && y >= h/4 && y < h*3/4 {
				c = color.NRGBA{R: 0xff, A: 0xff}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestNewIconSizesAndFormats(t *testing.T) {
	tests := []struct {
		name string
		size int // of the source
	}{
		{"a source of 256", 256},
		{"a source of 512", 512},
		{"a source of 100", 100},
		{"a source of 16", 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon, err := NewIcon(squarePNG(t, tt.size, tt.size))
			if err != nil {
				t.Fatal(err)
			}
			if len(icon.Images) != len(IconSizes) {
				t.Fatalf("%d images, want %d", len(icon.Images), len(IconSizes))
			}
			for i, img := range icon.Images {
				want := IconSizes[i]
				if img.Size != want {
					t.Errorf("image %d is %d pixels, want %d", i, img.Size, want)
				}
				if want == 256 {
					// The 256 pixel image is a PNG.
					decoded, err := png.Decode(bytes.NewReader(img.Data))
					if err != nil || decoded.Bounds().Dx() != 256 || decoded.Bounds().Dy() != 256 {
						t.Errorf("the 256 pixel image is not a PNG of 256 pixels: %v", err)
					}
					continue
				}
				checkDIB(t, img.Data, want)
			}
			checkGroup(t, icon)
		})
	}
}

// checkDIB checks the header, the pixels and the mask of an RT_ICON bitmap.
func checkDIB(t *testing.T, data []byte, size int) {
	t.Helper()
	le := binary.LittleEndian
	maskRow := (size + 31) / 32 * 4
	if want := 40 + size*size*4 + maskRow*size; len(data) != want {
		t.Fatalf("%d pixels: %d bytes, want %d", size, len(data), want)
	}
	if le.Uint32(data[0:]) != 40 || int(le.Uint32(data[4:])) != size || int(le.Uint32(data[8:])) != 2*size ||
		le.Uint16(data[12:]) != 1 || le.Uint16(data[14:]) != 32 || le.Uint32(data[16:]) != 0 {
		t.Errorf("%d pixels: header = %v", size, data[:40])
	}
	// The pixels are BGRA from the bottom row up: the first pixel is the bottom left corner (blue),
	// the one in the middle of the image is red.
	px := func(x, y int) [4]byte { // y counted from the top
		off := 40 + ((size-1-y)*size+x)*4
		return [4]byte(data[off : off+4])
	}
	if got := px(0, 0); got != [4]byte{0xff, 0, 0, 0xff} {
		t.Errorf("%d pixels: the corner is %v, want opaque blue (BGRA)", size, got)
	}
	if got := px(size/2, size/2); got != [4]byte{0, 0, 0xff, 0xff} {
		t.Errorf("%d pixels: the middle is %v, want opaque red (BGRA)", size, got)
	}
	for _, b := range data[40+size*size*4:] {
		if b != 0 {
			t.Errorf("%d pixels: the AND mask is not all zeros", size)
			break
		}
	}
}

// checkGroup checks the GRPICONDIR against the images.
func checkGroup(t *testing.T, icon *Icon) {
	t.Helper()
	le := binary.LittleEndian
	g := icon.Group
	if len(g) != 6+14*len(icon.Images) || le.Uint16(g[0:]) != 0 || le.Uint16(g[2:]) != 1 || int(le.Uint16(g[4:])) != len(icon.Images) {
		t.Fatalf("group header = %v (%d bytes)", g[:6], len(g))
	}
	for i, img := range icon.Images {
		e := g[6+14*i:]
		w := int(e[0])
		if w == 0 {
			w = 256
		}
		h := int(e[1])
		if h == 0 {
			h = 256
		}
		if w != img.Size || h != img.Size || e[2] != 0 || e[3] != 0 || le.Uint16(e[4:]) != 1 || le.Uint16(e[6:]) != 32 ||
			int(le.Uint32(e[8:])) != len(img.Data) || int(le.Uint16(e[12:])) != i+1 {
			t.Errorf("group entry %d = %v, want %d pixels, 32 bits, %d bytes, ID %d", i, e[:14], img.Size, len(img.Data), i+1)
		}
	}
}

func TestNewIconErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		meta string
	}{
		{"not a PNG", []byte("GIF89a, really"), "what"},
		{"not square", squarePNG(t, 64, 32), "width"},
		{"under 16 pixels", squarePNG(t, 8, 8), "width"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewIcon(tt.data)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("NewIcon.WindowsIconInvalid") {
				t.Errorf("route %s, want NewIcon.WindowsIconInvalid", appErr.Route())
			}
			if _, has := appErr.Metadata[tt.meta]; !has {
				t.Errorf("metadata %v has no %q", appErr.Metadata, tt.meta)
			}
		})
	}
}

func TestDefaultIcon(t *testing.T) {
	data, err := DefaultIcon()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 256 || b.Dy() != 256 {
		t.Fatalf("the default icon is %v, want 256 by 256", b)
	}
	// Blue in the corner, white on the ring and the bar.
	if r, g, b, _ := img.At(2, 2).RGBA(); r>>8 != 0x1a || g>>8 != 0x73 || b>>8 != 0xe8 {
		t.Errorf("the corner is %x %x %x, want the blue of the seed color", r>>8, g>>8, b>>8)
	}
	if r, g, b, _ := img.At(128+40, 128).RGBA(); r>>8 != 0xff || g>>8 != 0xff || b>>8 != 0xff {
		t.Errorf("the bar is %x %x %x, want white", r>>8, g>>8, b>>8)
	}
	if _, err := NewIcon(data); err != nil {
		t.Errorf("the default icon makes no icon: %v", err)
	}
}
