package ui

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"golang.org/x/image/bmp"
)

// testRGBA is a 4x2 image: opaque red, half transparent green, blue, white.
func testNRGBA() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	cols := []color.NRGBA{{255, 0, 0, 255}, {0, 255, 0, 128}, {0, 0, 255, 255}, {255, 255, 255, 255}}
	for y := 0; y < 2; y++ {
		for x, c := range cols {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func encodePNG(t testing.TB, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func pixel(img *Image, x, y int) [4]byte {
	i := 4 * (y*img.Width() + x)
	return [4]byte(img.Pixels()[i : i+4])
}

func pixelsNear(a, b [4]byte, tol int) bool {
	for k := range a {
		d := int(a[k]) - int(b[k])
		if d < -tol || d > tol {
			return false
		}
	}
	return true
}

func TestInstantiateImageCodecStill(t *testing.T) {
	var jpegData, bmpData bytes.Buffer
	flat := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < len(flat.Pix); i += 4 {
		flat.Pix[i], flat.Pix[i+1], flat.Pix[i+2], flat.Pix[i+3] = 200, 40, 40, 255
	}
	if err := jpeg.Encode(&jpegData, flat, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	if err := bmp.Encode(&bmpData, testNRGBA()); err != nil {
		t.Fatal(err)
	}
	webp, err := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		data []byte
		w, h int
		at   [2]int
		want [4]byte
		tol  int
	}{
		{"png red", encodePNG(t, testNRGBA()), 4, 2, [2]int{0, 0}, [4]byte{255, 0, 0, 255}, 0},
		{"png half transparent green is premultiplied", encodePNG(t, testNRGBA()), 4, 2, [2]int{1, 1}, [4]byte{0, 128, 0, 128}, 1},
		{"png blue", encodePNG(t, testNRGBA()), 4, 2, [2]int{2, 0}, [4]byte{0, 0, 255, 255}, 0},
		{"jpeg", jpegData.Bytes(), 8, 8, [2]int{4, 4}, [4]byte{200, 40, 40, 255}, 6},
		{"bmp", bmpData.Bytes(), 4, 2, [2]int{3, 0}, [4]byte{255, 255, 255, 255}, 0},
		{"webp", webp, 1, 1, [2]int{0, 0}, [4]byte{0, 0, 0, 0}, 255},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := InstantiateImageCodec(context.Background(), tt.data)
			if err != nil {
				t.Fatal(err)
			}
			if c.FrameCount() != 1 || c.RepetitionCount() != 0 {
				t.Errorf("frames = %d, repetitions = %d", c.FrameCount(), c.RepetitionCount())
			}
			f, err := c.GetNextFrame(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if f.Image.Width() != tt.w || f.Image.Height() != tt.h || f.Duration != 0 {
				t.Errorf("frame %dx%d, duration %v", f.Image.Width(), f.Image.Height(), f.Duration)
			}
			if got := pixel(f.Image, tt.at[0], tt.at[1]); !pixelsNear(got, tt.want, tt.tol) {
				t.Errorf("pixel %v = %v, want %v", tt.at, got, tt.want)
			}
			// A still image gives the same frame again, as a new handle.
			g, _ := c.GetNextFrame(context.Background())
			if g.Image == f.Image || !g.Image.IsCloneOf(f.Image) {
				t.Error("the second frame should be a clone of the first")
			}
		})
	}
}

// gifFrame is a 4x4 frame, or smaller at an offset, of one color, with index 0 transparent.
func gifFrame(bounds image.Rectangle, c color.RGBA) *image.Paletted {
	p := image.NewPaletted(bounds, color.Palette{color.RGBA{}, c})
	for i := range p.Pix {
		p.Pix[i] = 1
	}
	return p
}

func encodeGIF(t *testing.T, g *gif.GIF) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestInstantiateImageCodecGIF(t *testing.T) {
	red, blue := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}
	canvas := image.Rect(0, 0, 4, 4)
	inner := image.Rect(1, 1, 3, 3)
	tests := []struct {
		name      string
		g         *gif.GIF
		frames    int
		durations []time.Duration
		repeat    int
		// the pixel at 2,2 and at 0,0 of each frame
		center, corner [][4]byte
	}{
		{
			name: "the next frame draws over the last",
			g: &gif.GIF{
				Image: []*image.Paletted{gifFrame(canvas, red), gifFrame(inner, blue)}, Delay: []int{5, 20},
				Disposal: []byte{gif.DisposalNone, gif.DisposalNone}, LoopCount: 0, Config: image.Config{Width: 4, Height: 4},
			},
			frames: 2, durations: []time.Duration{50 * time.Millisecond, 200 * time.Millisecond}, repeat: -1,
			center: [][4]byte{{255, 0, 0, 255}, {0, 0, 255, 255}}, corner: [][4]byte{{255, 0, 0, 255}, {255, 0, 0, 255}},
		},
		{
			name: "disposal to the background clears the frame's area",
			g: &gif.GIF{
				Image: []*image.Paletted{gifFrame(canvas, red), gifFrame(inner, blue)}, Delay: []int{10, 10},
				Disposal: []byte{gif.DisposalBackground, gif.DisposalNone}, LoopCount: 3, Config: image.Config{Width: 4, Height: 4},
			},
			frames: 2, durations: []time.Duration{100 * time.Millisecond, 100 * time.Millisecond}, repeat: 3,
			center: [][4]byte{{255, 0, 0, 255}, {0, 0, 255, 255}}, corner: [][4]byte{{255, 0, 0, 255}, {0, 0, 0, 0}},
		},
		{
			name: "disposal to the previous restores the canvas",
			g: &gif.GIF{
				Image: []*image.Paletted{gifFrame(canvas, red), gifFrame(inner, blue), gifFrame(image.Rect(0, 0, 1, 1), blue)},
				Delay: []int{10, 10, 10}, Disposal: []byte{gif.DisposalNone, gif.DisposalPrevious, gif.DisposalNone},
				LoopCount: -1, Config: image.Config{Width: 4, Height: 4},
			},
			frames: 3, durations: []time.Duration{100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond}, repeat: 0,
			center: [][4]byte{{255, 0, 0, 255}, {0, 0, 255, 255}, {255, 0, 0, 255}}, corner: [][4]byte{{255, 0, 0, 255}, {255, 0, 0, 255}, {0, 0, 255, 255}},
		},
		{
			name: "a short delay lasts 100 ms",
			g: &gif.GIF{
				Image: []*image.Paletted{gifFrame(canvas, red), gifFrame(canvas, blue)}, Delay: []int{0, 1},
				Disposal: []byte{gif.DisposalNone, gif.DisposalNone}, LoopCount: 0, Config: image.Config{Width: 4, Height: 4},
			},
			frames: 2, durations: []time.Duration{100 * time.Millisecond, 100 * time.Millisecond}, repeat: -1,
			center: [][4]byte{{255, 0, 0, 255}, {0, 0, 255, 255}}, corner: [][4]byte{{255, 0, 0, 255}, {0, 0, 255, 255}},
		},
		{
			name: "a single frame does not repeat",
			g: &gif.GIF{
				Image: []*image.Paletted{gifFrame(canvas, red)}, Delay: []int{10}, Disposal: []byte{gif.DisposalNone},
				LoopCount: 0, Config: image.Config{Width: 4, Height: 4},
			},
			frames: 1, durations: []time.Duration{100 * time.Millisecond}, repeat: 0,
			center: [][4]byte{{255, 0, 0, 255}}, corner: [][4]byte{{255, 0, 0, 255}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := InstantiateImageCodec(context.Background(), encodeGIF(t, tt.g))
			if err != nil {
				t.Fatal(err)
			}
			if c.FrameCount() != tt.frames || c.RepetitionCount() != tt.repeat {
				t.Fatalf("frames = %d, repetitions = %d; want %d, %d", c.FrameCount(), c.RepetitionCount(), tt.frames, tt.repeat)
			}
			for i := 0; i < tt.frames; i++ {
				f, err := c.GetNextFrame(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if f.Duration != tt.durations[i] {
					t.Errorf("frame %d lasts %v, want %v", i, f.Duration, tt.durations[i])
				}
				if f.Image.Width() != 4 || f.Image.Height() != 4 {
					t.Errorf("frame %d is %dx%d, want the whole canvas", i, f.Image.Width(), f.Image.Height())
				}
				if got := pixel(f.Image, 2, 2); got != tt.center[i] {
					t.Errorf("frame %d center = %v, want %v", i, got, tt.center[i])
				}
				if got := pixel(f.Image, 0, 0); got != tt.corner[i] {
					t.Errorf("frame %d corner = %v, want %v", i, got, tt.corner[i])
				}
			}
			// After the last frame it starts again.
			f, _ := c.GetNextFrame(context.Background())
			if got := pixel(f.Image, 2, 2); got != tt.center[0] {
				t.Errorf("the frame after the last = %v, want the first %v", got, tt.center[0])
			}
		})
	}
}

func TestInstantiateImageCodecScaling(t *testing.T) {
	src := encodePNG(t, image.NewNRGBA(image.Rect(0, 0, 40, 20)))
	no := false
	tests := []struct {
		name string
		opts InstantiateImageCodecOpts
		w, h int
	}{
		{"none", InstantiateImageCodecOpts{}, 40, 20},
		{"both", InstantiateImageCodecOpts{TargetWidth: 10, TargetHeight: 30}, 10, 30},
		{"width keeps the aspect ratio", InstantiateImageCodecOpts{TargetWidth: 20}, 20, 10},
		{"height keeps the aspect ratio", InstantiateImageCodecOpts{TargetHeight: 40}, 80, 40},
		{"rounds", InstantiateImageCodecOpts{TargetWidth: 7}, 7, 4},
		{"never zero", InstantiateImageCodecOpts{TargetWidth: 1}, 1, 1},
		{"upscaling allowed by default", InstantiateImageCodecOpts{TargetWidth: 80, TargetHeight: 80}, 80, 80},
		{"upscaling refused, both held", InstantiateImageCodecOpts{TargetWidth: 80, TargetHeight: 80, AllowUpscaling: &no}, 40, 20},
		{"upscaling refused, one axis", InstantiateImageCodecOpts{TargetWidth: 80, TargetHeight: 10, AllowUpscaling: &no}, 40, 10},
		{"downscaling with upscaling refused", InstantiateImageCodecOpts{TargetWidth: 10, AllowUpscaling: &no}, 10, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := InstantiateImageCodec(context.Background(), src, tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			f, _ := c.GetNextFrame(context.Background())
			if f.Image.Width() != tt.w || f.Image.Height() != tt.h {
				t.Errorf("size = %dx%d, want %dx%d", f.Image.Width(), f.Image.Height(), tt.w, tt.h)
			}
		})
	}
	// A flat color stays that color when scaled, and an animation scales every frame.
	flat := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for i := 0; i < len(flat.Pix); i += 4 {
		flat.Pix[i], flat.Pix[i+1], flat.Pix[i+2], flat.Pix[i+3] = 10, 200, 90, 255
	}
	c, err := InstantiateImageCodec(context.Background(), encodePNG(t, flat), InstantiateImageCodecOpts{TargetWidth: 5, TargetHeight: 9})
	if err != nil {
		t.Fatal(err)
	}
	f, _ := c.GetNextFrame(context.Background())
	if got := pixel(f.Image, 2, 4); !pixelsNear(got, [4]byte{10, 200, 90, 255}, 1) {
		t.Errorf("scaled flat color = %v", got)
	}
	g := &gif.GIF{
		Image: []*image.Paletted{gifFrame(image.Rect(0, 0, 8, 8), color.RGBA{255, 0, 0, 255}), gifFrame(image.Rect(0, 0, 8, 8), color.RGBA{0, 0, 255, 255})},
		Delay: []int{10, 10}, Disposal: []byte{0, 0}, Config: image.Config{Width: 8, Height: 8},
	}
	ac, err := InstantiateImageCodec(context.Background(), encodeGIF(t, g), InstantiateImageCodecOpts{TargetWidth: 4})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		f, _ := ac.GetNextFrame(context.Background())
		if f.Image.Width() != 4 || f.Image.Height() != 4 {
			t.Errorf("animated frame %d is %dx%d", i, f.Image.Width(), f.Image.Height())
		}
	}
}

func TestInstantiateImageCodecErrors(t *testing.T) {
	good := encodePNG(t, testNRGBA())
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	// A PNG header that claims a huge size.
	huge := append([]byte(nil), good...)
	copy(huge[16:], []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}) // 65536 x 65536
	tests := []struct {
		name string
		ctx  context.Context
		data []byte
		opts []InstantiateImageCodecOpts
		tag  string
	}{
		{"empty", context.Background(), nil, nil, "ImageDecodeFailed"},
		{"not an image", context.Background(), []byte("hello, this is not an image"), nil, "ImageDecodeFailed"},
		{"truncated png", context.Background(), good[:len(good)/2], nil, "ImageDecodeFailed"},
		{"too large", context.Background(), huge, nil, "ImageDecodeFailed"},
		{"negative target", context.Background(), good, []InstantiateImageCodecOpts{{TargetWidth: -1}}, "ImageSizeInvalid"},
		{"canceled", canceled, good, nil, "ImageDecodeCanceled"},
		{"two options", context.Background(), good, []InstantiateImageCodecOpts{{}, {}}, "MultipleOptions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					appErr, ok := errors.FromError(recoverError(t, r))
					if !ok || !appErr.HasRoute(tt.tag) {
						t.Errorf("expected %s, got %v", tt.tag, appErr)
					}
				}
			}()
			c, err := InstantiateImageCodec(tt.ctx, tt.data, tt.opts...)
			if tt.tag == "MultipleOptions" {
				t.Fatal("expected a panic")
			}
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.tag) || c != nil {
				t.Errorf("expected %s, got %v", tt.tag, err)
			}
		})
	}
}

func TestCodecDisposedAndCanceled(t *testing.T) {
	c, err := InstantiateImageCodec(context.Background(), encodePNG(t, testNRGBA()))
	if err != nil {
		t.Fatal(err)
	}
	f, _ := c.GetNextFrame(context.Background())
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.GetNextFrame(canceled); err == nil {
		t.Error("a canceled context should fail GetNextFrame")
	} else if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("ImageDecodeCanceled") {
		t.Errorf("error = %v", err)
	}
	c.Dispose()
	if _, err := c.GetNextFrame(context.Background()); err == nil {
		t.Error("a disposed codec should fail")
	} else if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("CodecDisposed") {
		t.Errorf("error = %v", err)
	}
	if f.Image.Width() != 4 {
		t.Error("a frame already returned must stay valid")
	}
}

func TestImageToByteData(t *testing.T) {
	c, err := InstantiateImageCodec(context.Background(), encodePNG(t, testNRGBA()))
	if err != nil {
		t.Fatal(err)
	}
	f, _ := c.GetNextFrame(context.Background())
	img := f.Image
	for _, format := range []ImageByteFormat{ImageByteFormatUnspecified, ImageByteFormatRawRGBA, ImageByteFormatRawUnmodified} {
		data, err := img.ToByteData(format)
		if err != nil || !bytes.Equal(data, img.Pixels()) {
			t.Errorf("format %v: %v, %v", format, data, err)
		}
		data[0] ^= 0xFF
		if img.Pixels()[0] == data[0] {
			t.Errorf("format %v returned the image's own pixels", format)
		}
	}
	straight, err := img.ToByteData(ImageByteFormatRawStraightRGBA)
	if err != nil {
		t.Fatal(err)
	}
	// The half transparent green, straight again: (0, 255, 0, 128).
	if got := [4]byte(straight[4:8]); !pixelsNear(got, [4]byte{0, 255, 0, 128}, 1) {
		t.Errorf("straight pixel = %v", got)
	}
	encoded, err := img.ToByteData(ImageByteFormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	back, err := InstantiateImageCodec(context.Background(), encoded)
	if err != nil {
		t.Fatal(err)
	}
	bf, _ := back.GetNextFrame(context.Background())
	if bf.Image.Width() != 4 || !bytes.Equal(bf.Image.Pixels(), img.Pixels()) {
		t.Error("a PNG of the image does not decode to the same pixels")
	}
	if _, err := img.ToByteData(ImageByteFormat(99)); err == nil {
		t.Error("an unknown format should fail")
	} else if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("ImageFormatUnsupported") {
		t.Errorf("error = %v", err)
	}
	img.Dispose()
	if _, err := img.ToByteData(ImageByteFormatRawRGBA); err == nil {
		t.Error("a disposed image should fail")
	} else if appErr, ok := errors.FromError(err); !ok || !appErr.HasRoute("ImageDisposed") {
		t.Errorf("error = %v", err)
	}
}

func TestTargetSize(t *testing.T) {
	tests := []struct {
		w, h int
		o    InstantiateImageCodecOpts
		tw   int
		th   int
	}{
		{10, 10, InstantiateImageCodecOpts{TargetWidth: 1 << 20}, 1 << 15, 1 << 11},
		{10, 10, InstantiateImageCodecOpts{TargetWidth: 1 << 14, TargetHeight: 1 << 14}, 1 << 13, 1 << 13},
	}
	for _, tt := range tests {
		tw, th := targetSize(tt.w, tt.h, tt.o)
		if tw*th > maxImagePixels || tw > maxImageSize || th > maxImageSize {
			t.Errorf("targetSize(%+v) = %dx%d is over the limits", tt.o, tw, th)
		}
	}
}

func FuzzInstantiateImageCodec(f *testing.F) {
	var b bytes.Buffer
	_ = jpeg.Encode(&b, testNRGBA(), nil)
	var bm bytes.Buffer
	_ = bmp.Encode(&bm, testNRGBA())
	webp, _ := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	f.Add(encodePNG(f, testNRGBA()))
	f.Add(b.Bytes())
	f.Add(bm.Bytes())
	f.Add(webp)
	f.Add(encodeGIFBytes(f))
	f.Add([]byte("GIF89a"))
	f.Add([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	f.Fuzz(func(t *testing.T, data []byte) {
		c, err := InstantiateImageCodec(context.Background(), data, InstantiateImageCodecOpts{TargetWidth: 7})
		if err != nil {
			if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "ImageDecodeFailed" {
				t.Fatalf("unexpected error %v", err)
			}
			return
		}
		for i := 0; i < c.FrameCount(); i++ {
			fr, err := c.GetNextFrame(context.Background())
			if err != nil || fr.Image.Width() != 7 || len(fr.Image.Pixels()) != 4*fr.Image.Width()*fr.Image.Height() {
				t.Fatalf("frame %d: %v", i, err)
			}
		}
	})
}

func encodeGIFBytes(t testing.TB) []byte {
	g := &gif.GIF{
		Image: []*image.Paletted{gifFrame(image.Rect(0, 0, 4, 4), color.RGBA{255, 0, 0, 255}), gifFrame(image.Rect(1, 1, 3, 3), color.RGBA{0, 0, 255, 255})},
		Delay: []int{10, 10}, Disposal: []byte{gif.DisposalNone, gif.DisposalPrevious}, Config: image.Config{Width: 4, Height: 4},
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
