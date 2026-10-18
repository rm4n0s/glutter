package ui

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
)

func TestDecodeImageFromPixels(t *testing.T) {
	rgba := []byte{
		1, 2, 3, 4, 5, 6, 7, 8,
		9, 10, 11, 12, 13, 14, 15, 16,
	}
	tests := []struct {
		name     string
		pixels   []byte
		w, h     int
		format   PixelFormat
		rowBytes int
		want     []byte
	}{
		{"rgba", rgba, 2, 2, PixelFormatRGBA8888, 0, rgba},
		{"unspecified is rgba", rgba, 2, 2, PixelFormatUnspecified, 0, rgba},
		{"bgra swaps red and blue", rgba, 2, 2, PixelFormatBGRA8888, 0, []byte{3, 2, 1, 4, 7, 6, 5, 8, 11, 10, 9, 12, 15, 14, 13, 16}},
		{"row bytes skips padding", []byte{1, 2, 3, 4, 99, 99, 5, 6, 7, 8, 99}, 1, 2, PixelFormatRGBA8888, 6, []byte{1, 2, 3, 4, 5, 6, 7, 8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixels := append([]byte(nil), tt.pixels...)
			img, err := DecodeImageFromPixels(context.Background(), pixels, tt.w, tt.h, tt.format, DecodeImageFromPixelsOpts{RowBytes: tt.rowBytes})
			if err != nil {
				t.Fatal(err)
			}
			if img.Width() != tt.w || img.Height() != tt.h {
				t.Errorf("size = %dx%d", img.Width(), img.Height())
			}
			if string(img.Pixels()) != string(tt.want) {
				t.Errorf("pixels = %v, want %v", img.Pixels(), tt.want)
			}
			pixels[0], pixels[2] = 200, 200 // the image has its own copy
			if string(img.Pixels()) != string(tt.want) {
				t.Error("pixels were not copied")
			}
		})
	}
}

func TestDecodeImageFromPixelsErrors(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name   string
		ctx    context.Context
		pixels []byte
		w, h   int
		format PixelFormat
		opts   []DecodeImageFromPixelsOpts
		tag    string
	}{
		{"zero width", context.Background(), make([]byte, 16), 0, 2, PixelFormatRGBA8888, nil, "ImageSizeInvalid"},
		{"too large", context.Background(), make([]byte, 16), 1 << 20, 2, PixelFormatRGBA8888, nil, "ImageSizeInvalid"},
		{"unknown format", context.Background(), make([]byte, 16), 2, 2, PixelFormat(9), nil, "ImageFormatUnsupported"},
		{"too few pixels", context.Background(), make([]byte, 15), 2, 2, PixelFormatRGBA8888, nil, "ImagePixelsInvalid"},
		{"row bytes too small", context.Background(), make([]byte, 16), 2, 2, PixelFormatRGBA8888, []DecodeImageFromPixelsOpts{{RowBytes: 4}}, "ImagePixelsInvalid"},
		{"canceled", canceled, make([]byte, 16), 2, 2, PixelFormatRGBA8888, nil, "ImageDecodeCanceled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := DecodeImageFromPixels(tt.ctx, tt.pixels, tt.w, tt.h, tt.format, tt.opts...)
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.tag) || img != nil {
				t.Errorf("expected %s, got %v", tt.tag, err)
			}
		})
	}
}

func TestImageHandles(t *testing.T) {
	img := NewImageFromRGBA(make([]byte, 4*3*2), 3, 2)
	clone := img.Clone()
	if !clone.IsCloneOf(img) || !img.IsCloneOf(clone) {
		t.Error("clone does not share pixels")
	}
	other := NewImageFromRGBA(make([]byte, 4), 1, 1)
	if other.IsCloneOf(img) {
		t.Error("different images are clones")
	}
	img.Dispose()
	if !img.IsDisposed() || clone.IsDisposed() {
		t.Error("disposing one handle must not dispose the other")
	}
	if clone.Pixels() == nil || clone.Width() != 3 {
		t.Error("the clone lost its pixels")
	}
	if got := clone.String(); got != "[3×2]" {
		t.Errorf("String = %q", got)
	}
}

func TestNewImageFromRGBAPanics(t *testing.T) {
	for _, tt := range []struct {
		name string
		pix  []byte
		w, h int
	}{{"wrong length", make([]byte, 7), 1, 2}, {"zero size", nil, 0, 0}} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				appErr, ok := errors.FromError(recoverError(t, recover()))
				if !ok || !appErr.HasRoute("ImagePixelsInvalid") {
					t.Errorf("expected ImagePixelsInvalid, got %v", appErr)
				}
			}()
			NewImageFromRGBA(tt.pix, tt.w, tt.h)
		})
	}
}
